package proxy

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/storage"
	"github.com/patrickmn/go-cache"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	GHHeader           = "User-Agent"
	GHAgent            = "GitHub-Hookshot"
	CookieJenkinsIdled = "JenkinsIdled"
	ServiceName        = "jenkins"
	SessionCookie      = "JSESSIONID"
)

//Proxy handles requests, verifies authentication and proxies to Jenkins.
//If the request comes from Github, it bufferes it and replayes if Jenkins
//is not available.
type Proxy struct {
	//TenantCache is used as a temporary cache to optimize number of requests
	//going to tenant and wit services
	TenantCache *cache.Cache

	//ProxyCache is used as a cache for session ids passed by Jenkins in cookies
	ProxyCache       *cache.Cache
	bufferLock       *sync.Mutex
	visitLock        *sync.Mutex
	bufferCheckSleep time.Duration
	tenant           clients.Tenant
	wit              clients.WIT
	idler            clients.Idler

	//redirect is a base URL of the proxy
	redirect        string
	publicKey       *rsa.PublicKey
	authURL         string
	storageService  *storage.DBService
	indexPath       string
	maxRequestRetry int
}

type ProxyError struct {
	Errors []ProxyErrorInfo
}

type ProxyErrorInfo struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func NewProxy(t clients.Tenant, w clients.WIT, i clients.Idler, keycloakURL string, authURL string, redirect string, storageService *storage.DBService, indexPath string, maxRequestRetry int) (Proxy, error) {
	p := Proxy{
		TenantCache:      cache.New(30*time.Minute, 40*time.Minute),
		ProxyCache:       cache.New(15*time.Minute, 10*time.Minute),
		bufferLock:       &sync.Mutex{},
		visitLock:        &sync.Mutex{},
		tenant:           t,
		wit:              w,
		idler:            i,
		bufferCheckSleep: 30,
		redirect:         redirect,
		authURL:          authURL,
		storageService:   storageService,
		indexPath:        indexPath,
		maxRequestRetry:  maxRequestRetry,
	}

	//Collect and parse public key from Keycloak
	pk, err := GetPublicKey(keycloakURL)
	if err != nil {
		return p, err
	}
	p.publicKey = pk

	//Spawn a routine to process buffered requests
	go func() {
		p.ProcessBuffer()
	}()
	return p, nil
}

//Handle handles requests coming to the proxy and performs action based on
//the type of request and state of Jenkins.
func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	isGH := false
	//Is the request coming from Github Webhook? FIXME - should we only care about push?
	if ua, exist := r.Header[GHHeader]; exist {
		isGH = strings.HasPrefix(ua[0], GHAgent)
	}

	var body []byte
	var err error
	var ns string
	var cacheKey string
	var reqURL url.URL
	if isGH {
		//Load request body if it's GH webhok
		gh := GHHookStruct{}
		defer r.Body.Close()
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			p.HandleError(w, err)
			return
		}
		err = json.Unmarshal(body, &gh)
		if err != nil {
			p.HandleError(w, err)
			return
		}

		if err != nil {
			p.HandleError(w, fmt.Errorf("Could not parse GH payload: %s", err))
			return
		}
		log.Info("Processing request from %s", gh.Repository.CloneURL)
		ns, err = p.GetUser(gh)
		if err != nil {
			p.HandleError(w, err)
			return
		}
		scheme, route, err := p.idler.GetRoute(ns)
		if err != nil {
			p.HandleError(w, err)
			return
		}
		r.URL.Scheme = scheme
		r.URL.Host = route
		r.Host = route

		isIdle, err := p.idler.IsIdle(ns)
		if err != nil {
			p.HandleError(w, err)
			return
		}

		//If Jenkins is idle, we need to cache the request and return success
		if isIdle {
			w.Header().Set("Server", "Webhook-Proxy")
			sr, err := storage.NewRequest(r, ns, body)
			if err != nil {
				p.HandleError(w, err)
				return
			}
			err = p.storageService.CreateRequest(sr)
			if err != nil {
				p.HandleError(w, err)
				return
			}
			err = p.RecordStatistics(ns, 0, time.Now().Unix())
			if err != nil {
				p.HandleError(w, err)
				return
			}

			log.Info("Webhook request buffered for ", ns)
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(""))
			return
		} else { //If Jenkins is up, we can simply proxy through
			log.Info(fmt.Sprintf("Passing through %s", r.URL.String()))
		}
	} else { //If this is no Github traffic (e.g. user accessing UI)
		//redirect determines if we need to redirect to auth service
		redirect := true

		//redirectURL is used for auth service as a target of successful auth redirect
		redirectURL, err := url.ParseRequestURI(fmt.Sprintf("%s%s", strings.TrimRight(p.redirect, "/"), r.URL.Path))
		if err != nil {
			p.HandleError(w, err)
			return
		}

		//If the user provides OSO token, we can directly proxy
		if _, ok := r.Header["Authorization"]; ok { //FIXME Do we need this?
			redirect = false
		}

		reqURL = *r.URL

		if len(r.Cookies()) > 0 { //Check cookies and proxy cache to find user info
			for _, cookie := range r.Cookies() {
				if strings.HasPrefix(cookie.Name, SessionCookie) {
					if cacheVal, ok := p.ProxyCache.Get(cookie.Value); ok { //We found a session cookie in cache
						pci := cacheVal.(ProxyCacheItem)
						r.Host = pci.Route
						r.URL.Host = pci.Route
						r.URL.Scheme = "https"
						if !pci.TLS {
							r.URL.Scheme = "http"
						}
						ns = pci.NS
						redirect = false //user is probably logged in, do not redirect
						cacheKey = cookie.Value
						break
					}
				} else if cookie.Name == CookieJenkinsIdled { //Found a cookie saying Jenkins is idled, verify and act accordingly
					if c, ok := p.ProxyCache.Get(cookie.Value); ok {
						pci := c.(ProxyCacheItem)
						oc := ic.NewOpenShiftWithClient(http.DefaultClient, pci.ClusterURL, pci.OsoToken)
						isIdle, err := oc.IsIdle(pci.NS, ServiceName)
						if err != nil {
							p.HandleError(w, err)
							return
						}
						if isIdle != ic.JenkinsRunning { //If jenkins is not running, return loading page and status 202
							w.WriteHeader(http.StatusAccepted)
							tmplt, err := template.ParseFiles(p.indexPath)
							if err != nil {
								p.HandleError(w, err)
								return
							}
							data := struct {
								Message string
								Retry   int
							}{
								Message: "Jenkins has been idled. It is starting now, please wait...",
								Retry:   10,
							}
							log.Info("Templating index.html")
							err = tmplt.Execute(w, data)
							if err != nil {
								p.HandleError(w, err)
								return
							}
							p.RecordStatistics(pci.NS, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?
						} else { //If Jenkins is running, remove the cookie
							cookie.Expires = time.Unix(0, 0)
							http.SetCookie(w, cookie)
						}
						return //Do not proceed furter - everything is written to ResponseWriter
					}
				}
			}
			if len(cacheKey) == 0 { //If we do not have user's info cached, run through login process to get it
				log.Info("Could not find cache, redirecting to re-login")
			}
		}
		if tj, ok := r.URL.Query()["token_json"]; ok { //If there is token_json in query, process it, find user info and login to Jenkins
			log.Info("Found token info in query")
			tokenJSON := &TokenJSON{}
			if len(tj) < 1 {
				p.HandleError(w, fmt.Errorf("Could not read JWT token from URL"))
				return
			}
			err = json.Unmarshal([]byte(tj[0]), tokenJSON)
			if err != nil {
				p.HandleError(w, err)
				return
			}
			log.Info("Extracted JWT token")

			uid, err := GetTokenUID(tokenJSON.AccessToken, p.publicKey)
			if err != nil {
				p.HandleError(w, err)
				return
			}

			log.Info("Extracted UID from JWT token")

			ti, err := p.tenant.GetTenantInfo(uid)
			if err != nil {
				p.HandleError(w, err)
				return
			}

			namespace, err := p.tenant.GetNamespaceByType(ti, ServiceName)
			if err != nil {
				p.HandleError(w, err)
				return
			}

			ns = namespace.Name
			log.Info(fmt.Sprintf("Extracted Tenant Info: %s", ns))

			osoToken, err := GetOSOToken(p.authURL, namespace.ClusterURL, tokenJSON.AccessToken)
			if err != nil {
				p.HandleError(w, err)
				return
			}

			oc := ic.NewOpenShiftWithClient(http.DefaultClient, namespace.ClusterURL, osoToken)
			route, tls, err := oc.GetRoute(ns, ServiceName)
			if err != nil {
				p.HandleError(w, err)
				return
			}
			scheme := oc.GetScheme(tls)

			isIdle, err := oc.IsIdle(ns, ServiceName)
			if err != nil {
				p.HandleError(w, err)
				return
			}

			//Prepare an item for proxyCache - Jenkins info and OSO token
			pci := NewProxyCacheItem(namespace.Name, tls, route, namespace.ClusterURL, osoToken)
			//Break the process if the Jenkins is idled, set a cookie and redirect to self
			if isIdle != ic.JenkinsRunning {
				c := &http.Cookie{}
				u1 := uuid.NewV4().String()
				c.Name = CookieJenkinsIdled
				c.Value = u1
				p.ProxyCache.SetDefault(u1, pci)
				http.SetCookie(w, c)
				log.Info("Redirecting to remove token from URL")
				http.Redirect(w, r, redirectURL.String(), http.StatusFound) //Redirect to get rid of token in URL
				return
			}

			log.Info("Loaded OSO token")

			//Login to Jenkins with OSO token to get cookies
			jenkinsURL := fmt.Sprintf("%s://%s/", scheme, route)
			log.Info(fmt.Sprintf("Logging in %s", jenkinsURL))
			nr, _ := http.NewRequest("GET", jenkinsURL, nil)
			nr.Header.Set("Authorization", fmt.Sprintf("Bearer %s", osoToken))
			c := http.DefaultClient
			nresp, err := c.Do(nr)
			if err != nil {
				p.HandleError(w, err)
				return
			}
			if nresp.StatusCode == http.StatusOK {
				cached := false
				for _, cookie := range nresp.Cookies() {
					if cookie.Name == CookieJenkinsIdled {
						continue
					}
					http.SetCookie(w, cookie)
					if strings.HasPrefix(cookie.Name, SessionCookie) { //Find session cookie and use it's value as a key for cache
						p.ProxyCache.SetDefault(cookie.Value, pci)
						log.Info(fmt.Sprintf("Cached Jenkins route %s in %s", route, cookie.Value))
						cached = true
					}
				}
				//If all good, redirect to self to remove token from url
				if cached {
					log.Info(fmt.Sprintf("Redirecting to %s", redirectURL.String()))
					http.Redirect(w, r, redirectURL.String(), http.StatusFound)
				} else {
					p.HandleError(w, fmt.Errorf("Could not find cookie %s for %s", SessionCookie, namespace.Name))
				}
				return
			} else {
				p.HandleError(w, fmt.Errorf("Could not login to Jenkins in %s namespace on behalf of the user %s", ns, ti.Data.Attributes.Email))
				return
			}
		}

		//Check if we need to redirec tto auth service
		if redirect {
			redirAuth := GetAuthURI(p.authURL, redirectURL.String())
			log.Info(fmt.Sprintf("Redirecting to %s", redirAuth))
			http.Redirect(w, r, redirAuth, 301)
			return
		}
	}
	//Write usage stats to DB, run in a goroutine to not slow down the proxy
	go func() {
		p.RecordStatistics(ns, time.Now().Unix(), 0)
	}()

	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			//Add body back if it has been read (for GH)
			if len(body) > 0 {
				log.Info("Adding body back to request.")
				req.Body = ioutil.NopCloser(bytes.NewReader(body))
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			//Check response from Jenkins and redirect if it got idled in the meantime
			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout {
				if len(cacheKey) > 0 { //Delete cache entry to force new check whethe Jenkins is idled
					log.Info(fmt.Sprintf("Deleting cache Key: %s", cacheKey))
					p.ProxyCache.Delete(cacheKey)
				}
				if len(reqURL.String()) > 0 { //Block proxying to 503, redirect to self
					log.Info(fmt.Sprintf("Redirecting to %s, because %d", reqURL.String(), resp.StatusCode))
					http.Redirect(w, r, reqURL.String(), http.StatusFound)
				}
			}
			return nil
		},
	}).ServeHTTP(w, r)
}

//HandleError creates a JSON response with a given error and writes it to ResponseWriter
func (p *Proxy) HandleError(w http.ResponseWriter, err error) {
	log.Error(err)
	w.WriteHeader(http.StatusInternalServerError)

	pei := ProxyErrorInfo{
		Code:   fmt.Sprintf("%d", http.StatusInternalServerError),
		Detail: err.Error(),
	}
	e := ProxyError{
		Errors: make([]ProxyErrorInfo, 1),
	}
	e.Errors[0] = pei

	eb, err := json.Marshal(e)
	if err != nil {
		log.Error(err)
	}
	w.Write(eb)
}

//GHHookStruct a simplified structure to get info from
//a webhook request
type GHHookStruct struct {
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		GitURL   string `json:"git_url"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

//GetUser returns a namespace name based on Github repository URL
func (p *Proxy) GetUser(pl GHHookStruct) (namespace string, err error) {
	if n, found := p.TenantCache.Get(pl.Repository.CloneURL); found {
		log.Info("Cache hit")
		namespace = n.(string)
		return
	}
	log.Info("Cache miss")
	wi, err := p.wit.SearchCodebase(pl.Repository.CloneURL)
	if err != nil {
		return
	}

	log.Infof("Found id %s for repo %s", wi.OwnedBy, pl.Repository.CloneURL)
	ti, err := p.tenant.GetTenantInfo(wi.OwnedBy)
	if err != nil {
		return
	}

	n, err := p.tenant.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return
	}
	namespace = n.Name
	p.TenantCache.SetDefault(pl.Repository.CloneURL, namespace)
	return
}

//RecordStatistics writes usage statistics to a database
func (p *Proxy) RecordStatistics(ns string, la int64, lbf int64) (err error) {
	log.Infof("Recording stats for %s", ns)
	s, notFound, err := p.storageService.GetStatisticsUser(ns)
	if err != nil {
		log.Warningf("Could not load statistics for %s: %s (%v)", ns, err, s)
		if !notFound {
			return
		}
	}
	if notFound {
		log.Infof("New user %s", ns)
		s = storage.NewStatistics(ns, la, lbf)
		err = p.storageService.CreateStatistics(s)
		if err != nil {
			log.Errorf("Could not create statistics for %s: %s", ns, err)
		}
		return
	}
	if la != 0 {
		s.LastAccessed = la
	}
	if lbf != 0 {
		s.LastBufferedRequest = lbf
	}
	p.visitLock.Lock()
	err = p.storageService.UpdateStatistics(s)
	p.visitLock.Unlock()
	if err != nil {
		log.Errorf("Could not update statistics for %s: %s", ns, err)
	}

	return
}

//ProcessBuffer is a loop running through buffered webhook requests trying to replay them
func (p *Proxy) ProcessBuffer() {
	for {
		namespaces, err := p.storageService.GetUsers()
		if err != nil {
			log.Error(err)
		} else {
			for _, ns := range namespaces {
				reqs, err := p.storageService.GetRequests(ns)
				if err != nil {
					log.Error(err)
					continue
				}
				for _, r := range reqs {
					log.Info("Retrying request for ", ns)
					isIdle, err := p.idler.IsIdle(ns)
					if err != nil {
						log.Error(err)
						break
					}
					err = p.RecordStatistics(ns, 0, time.Now().Unix())
					if err != nil {
						log.Error(err)
					}
					if !isIdle {
						req, err := r.GetHTTPRequest()
						if err != nil {
							log.Errorf("Could not format request %s (%s): %s - deleting", r.ID, r.Namespace, err)
							err = p.storageService.DeleteRequest(&r)
							if err != nil {
								log.Errorf(storage.ErrorFailedDelete, r.ID, r.Namespace, err)
							}
							break
						}
						client := http.DefaultClient
						if r.Retries < p.maxRequestRetry { //Check how many times we retired (since the Jenkins started)
							resp, err := client.Do(req)
							if err != nil {
								log.Error("Error: ", err)
								errs := p.storageService.IncRequestRetry(&r)
								if len(errs) > 0 {
									for _, e := range errs {
										log.Error(e)
									}
								}
								break
							}

							if resp.StatusCode != 200 && resp.StatusCode != 404 { //Retry later if the response is not 200
								log.Error(fmt.Sprintf("Got status %s after retrying request on %s", resp.Status, req.URL))
								errs := p.storageService.IncRequestRetry(&r)
								if len(errs) > 0 {
									for _, e := range errs {
										log.Error(e)
									}
								}
								break
							} else if resp.StatusCode == 404 || resp.StatusCode == 400 { //400 - missing payload
								log.Warn(fmt.Sprintf("Got status %s after retrying request on %s, throwing away the request", resp.Status, req.URL))
							}

							log.Info(fmt.Sprintf("Request for %s to %s forwarded.", ns, req.Host))
						}

						//Delete request if we tried too many times or the replay was successful
						err = p.storageService.DeleteRequest(&r)
						if err != nil {
							log.Error(storage.ErrorFailedDelete, r.ID, r.Namespace, err)
						}
					} else {
						//Do not try other requests for user if Jenkins is not running
						break
					}
				}
			}
		}
		time.Sleep(p.bufferCheckSleep * time.Second)
	}
}
