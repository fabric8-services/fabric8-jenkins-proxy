package proxy

import (
	"crypto/rsa"
	"time"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/satori/go.uuid"
	"github.com/patrickmn/go-cache"
	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/storage"
	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	log "github.com/sirupsen/logrus"
)

const (
	GHHeader = "User-Agent"
	GHAgent = "GitHub-Hookshot"
	CookieJenkinsIdled = "JenkinsIdled"
	ServiceName = "jenkins"
	SessionCookie = "JSESSIONID"
)

type Proxy struct {
	TenantCache *cache.Cache
	ProxyCache *cache.Cache
	bufferLock *sync.Mutex
	visitLock *sync.Mutex
	service string
	bufferCheckSleep time.Duration
	tenant clients.Tenant
	wit clients.WIT
	idler clients.Idler
	redirect string
	publicKey *rsa.PublicKey
	authURL string
	storageService *storage.DBService
	indexPath string
	maxRequestRetry int
}

type BufferedReuqests struct {
	LastRequest time.Time
	Requests []BufferedReuqest
	Targets []string
}

type BufferedReuqest struct {
	Request *http.Request
	Body []byte
}

type TokenJSON struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType string `json:"token_type"`
	ExpiresIn int `json:"expires_in"`
	RefreshExpiresIn int `json:"refresh_expires_in"`
	Errors []ProxyErrorInfo
}

type ProxyError struct {
	Errors []ProxyErrorInfo
}

type ProxyErrorInfo struct{
	Code string `json:"code"`
	Detail string `json:"detail"`
}

func NewProxy(t clients.Tenant, w clients.WIT, i clients.Idler, keycloakURL string, authURL string, redirect string, storageService *storage.DBService, indexPath string, maxRequestRetry int) (Proxy, error) {
	p := Proxy{
		TenantCache: cache.New(30*time.Minute, 40*time.Minute),
		ProxyCache: cache.New(15*time.Minute, 10*time.Minute),
		bufferLock: &sync.Mutex{},
		visitLock: &sync.Mutex{},
		tenant: t,
		wit: w,
		idler: i,
		bufferCheckSleep: 30,
		redirect: redirect,
		authURL: authURL,
		storageService: storageService,
		indexPath: indexPath,
		maxRequestRetry: maxRequestRetry,
	}
	
	pk, err := GetPublicKey(keycloakURL)
	if err != nil {
		return p, err
	}

	p.publicKey = pk

	go func() {
		p.ProcessBuffer()
	}()
	return p, nil
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	isGH := false
	if ua, exist := r.Header[GHHeader]; exist {
		isGH = strings.HasPrefix(ua[0], GHAgent)
	}

	var body []byte
	var err error
	var ns string
	var cacheKey string
	var reqURL url.URL
	if isGH {
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

		if isIdle {
			w.Header().Set("Server", "Webhook-Proxy")
			sr, err := storage.NewRequest(r, ns, body)
			if err != nil {
				p.HandleError(w, err)
				return
			}
			err = p.storageService.CreateOrUpdateRequest(sr)
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
		} else {
			log.Info(fmt.Sprintf("Passing through %s", r.URL.String()))
		}
	} else {
		redirect := true

		redirectURL, err := url.ParseRequestURI(fmt.Sprintf("%s%s", strings.TrimRight(p.redirect, "/"), r.URL.Path))
		if err != nil {
			p.HandleError(w, err)
			return
		}

		if _, ok := r.Header["Authorization"]; ok { //FIXME Do we need this?
			redirect = false
		}

		reqURL = *r.URL

		if len(r.Cookies()) > 0 { //Check cookies and proxy cache to find user info
			for _, cookie := range r.Cookies() {
				if strings.HasPrefix(cookie.Name, SessionCookie) {
					if cacheVal, ok := p.ProxyCache.Get(cookie.Value); ok {
						pci := cacheVal.(ProxyCacheItem)
						r.Host = pci.Route
						r.URL.Host = pci.Route
						r.URL.Scheme = "https"
						if !pci.TLS {
							r.URL.Scheme = "http"
						}
						ns = pci.NS
						redirect = false
						cacheKey = cookie.Value
						break
					}
				} else if cookie.Name == CookieJenkinsIdled { //Found a cookie saying Jenkins is idled
					if c, ok := p.ProxyCache.Get(cookie.Value); ok {
						pci := c.(ProxyCacheItem)
						oc := ic.NewOpenShiftWithClient(http.DefaultClient, pci.ClusterURL, pci.OsoToken)
						isIdle, err := oc.IsIdle(pci.NS, ServiceName) 
						if err != nil {
							p.HandleError(w, err)
							return
						}
						if isIdle != ic.JenkinsRunning {
							w.WriteHeader(http.StatusAccepted)
							tmplt, err := template.ParseFiles(p.indexPath)
							if err != nil {
								p.HandleError(w, err)
								return
							}
							data := struct {
								Message string
								Retry int
							}{
								Message: "Jenkins has been idled. It is starting now, please wait...",
								Retry: 10,
							}
							log.Info("Templating index.html")
							err = tmplt.Execute(w, data)
							if err != nil {
								p.HandleError(w, err)
								return
							}
							p.RecordStatistics(pci.NS, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?
						} else {
							cookie.Expires = time.Unix(0, 0)
							http.SetCookie(w, cookie)
						}
						return
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

			pci := NewProxyCacheItem(namespace.Name, tls, route, namespace.ClusterURL, osoToken)
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

		if redirect {
			redirAuth := GetAuthURI(p.authURL, redirectURL.String())
			log.Info(fmt.Sprintf("Redirecting to %s", redirAuth))
			http.Redirect(w, r, redirAuth, 301)
			return
		}
	}
	//log.Info("Updating visit stats for ", ns)
	go func() {
		p.RecordStatistics(ns, time.Now().Unix(), 0)
	}()

	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			if len(body) > 0 {
				log.Info("Adding body back to request.")
				req.Body = ioutil.NopCloser(bytes.NewReader(body))
			}
		},
		ModifyResponse: func(resp *http.Response) (error) {
			//log.Info("Modifying response")
			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout  {
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

func (p *Proxy) HandleError(w http.ResponseWriter, err error) {
	log.Error(err)
	w.WriteHeader(http.StatusInternalServerError)

	pei := ProxyErrorInfo{
		Code: fmt.Sprintf("%d", http.StatusInternalServerError),
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

type GHHookStruct struct {
	Repository struct {
		Name string `json:"name"`
		FullName string `json:"full_name"`
		GitURL string `json:"git_url"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

func (p *Proxy) GetUser(pl GHHookStruct) (res string, err error) {
	if n, found := p.TenantCache.Get(pl.Repository.CloneURL); found {
		log.Info("Cache hit")
		res = n.(string)
		return
	}
	log.Info("Cache miss")
	wi, err := p.wit.SearchCodebase(pl.Repository.CloneURL)
	if err != nil {
		return
	}

	ti, err := p.tenant.GetTenantInfo(wi.OwnedBy)
	if err != nil {
		return
	}

	n, err := p.tenant.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return
	}
	res = n.Name
	p.TenantCache.SetDefault(pl.Repository.CloneURL, res)
	return
}

func (p *Proxy) RecordStatistics(ns string, la int64, lbf int64) (err error) {
	//Needs to go database
	s, notFound, err := p.storageService.GetStatisticsUser(ns)
	if err != nil {
		if !notFound {
			log.Errorf("Could not load statistics for %s: %s", ns, err)
			return
		}
	}
	if la != 0 {
		s.LastAccessed = la
	}
	if lbf != 0 {
		s.LastBufferedRequest = lbf
	}
	p.visitLock.Lock()
	err = p.storageService.CreateStatistics(&s)
	p.visitLock.Unlock()
	if err != nil {
		log.Errorf("Could not record statistics for %s: %s", ns, err)
	}

	return
}

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
						if r.Retries < p.maxRequestRetry { //Delete request if we tried too many times
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

							if resp.StatusCode != 200 && resp.StatusCode != 404 {
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
		time.Sleep(p.bufferCheckSleep*time.Second)
	}
}