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

	"errors"

	"hash/fnv"
	"runtime"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/metric"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util/logging"
	"github.com/patrickmn/go-cache"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	// GHHeader is how we determine whether we are dealing with a GitHub webhook
	GHHeader = "User-Agent"
	// GHAgent is the prefix of the GHHeader
	GHAgent = "GitHub-Hookshot"
	// CookieJenkinsIdled stores name of the cookie through which we can check if service is idled
	CookieJenkinsIdled = "JenkinsIdled"
	// ServiceName is name of service that we are trying to idle or unidle
	ServiceName = "jenkins"
	// SessionCookie stores name of the session cookie of the service in question
	SessionCookie = "JSESSIONID"

	defaultRetry = 15
)

var proxyLogger = log.WithFields(log.Fields{"component": "proxy"})

// Recorder to capture events
var Recorder = metric.PrometheusRecorder{}

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

//Proxy handles requests, verifies authentication and proxies to Jenkins.
//If the request comes from Github, it buffers it and replays if Jenkins
//is not available.
type Proxy struct {
	//TenantCache is used as a temporary cache to optimize number of requests
	//going to tenant and wit services
	TenantCache *cache.Cache

	//ProxyCache is used as a cache for session ids passed by Jenkins in cookies
	ProxyCache       *cache.Cache
	visitLock        *sync.Mutex
	bufferCheckSleep time.Duration
	tenant           *clients.Tenant
	wit              clients.WIT
	idler            clients.IdlerService

	//redirect is a base URL of the proxy
	redirect        string
	publicKey       *rsa.PublicKey
	authURL         string
	storageService  storage.Store
	indexPath       string
	maxRequestRetry int
	clusters        map[string]string
}

// Error represents list of error informations.
type Error struct {
	Errors []ErrorInfo
}

// ErrorInfo describes an HTTP error, consisting of HTTP status code and error detail.
type ErrorInfo struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// NewProxy creates an instance of Proxy client
func NewProxy(tenant *clients.Tenant, wit clients.WIT, idler clients.IdlerService, storageService storage.Store, config configuration.Configuration, clusters map[string]string) (Proxy, error) {
	p := Proxy{
		TenantCache:      cache.New(30*time.Minute, 40*time.Minute),
		ProxyCache:       cache.New(15*time.Minute, 10*time.Minute),
		visitLock:        &sync.Mutex{},
		tenant:           tenant,
		wit:              wit,
		idler:            idler,
		bufferCheckSleep: 30,
		redirect:         config.GetRedirectURL(),
		authURL:          config.GetAuthURL(),
		storageService:   storageService,
		indexPath:        config.GetIndexPath(),
		maxRequestRetry:  config.GetMaxRequestRetry(),
		clusters:         clusters,
	}

	//Initialize metrics
	Recorder.Initialize()

	//Collect and parse public key from Keycloak
	pk, err := GetPublicKey(config.GetKeycloakURL())
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
	// Setting header here to all the subsequent calls have this set
	w.Header().Set("Access-Control-Allow-Origin", "*")

	isGH := p.isGitHubRequest(r)
	var requestType string
	if isGH {
		requestType = "GitHub"
	} else {
		requestType = "Jenkins UI"
	}

	Recorder.RecordReqByTypeTotal(requestType)

	requestURL := logging.RequestMethodAndURL(r)
	requestHeaders := logging.RequestHeaders(r)
	requestHash := p.createRequestHash(requestURL, requestHeaders)
	logEntryWithHash := proxyLogger.WithField("request-hash", requestHash)

	logEntryWithHash.WithFields(
		log.Fields{
			"request": requestURL,
			"header":  requestHeaders,
			"type":    requestType,
		}).Info("Handling incoming proxy request.")

	var ns string
	var cacheKey string
	var noProxy bool

	// NOTE: Response payload and status codes (including errors) are written
	// to the ResponseWriter (w) in the called methods
	if isGH {
		ns, noProxy = p.handleGitHubRequest(w, r, logEntryWithHash)
		if noProxy {
			return
		}
	} else { //If this is no GitHub traffic (e.g. user accessing UI)
		cacheKey, ns, noProxy = p.handleJenkinsUIRequest(w, r, logEntryWithHash)
		if noProxy {
			return
		}
	}

	//Write usage stats to DB, run in a goroutine to not slow down the proxy
	go func() {
		p.recordStatistics(ns, time.Now().Unix(), 0)
	}()

	reqURL := *r.URL

	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			log.WithField("ns", ns).WithField("url", reqURL.String()).Info("Proxying")
		},
		ModifyResponse: func(resp *http.Response) error {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			//Check response from Jenkins and redirect if it got idled in the meantime
			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout {
				if len(cacheKey) > 0 { //Delete cache entry to force new check whether Jenkins is idled
					log.Infof("Deleting cache key: %s", cacheKey)
					p.ProxyCache.Delete(cacheKey)
				}
				if len(reqURL.String()) > 0 { //Block proxying to 503, redirect to self
					log.Infof("Redirecting to %s, because %d", reqURL.String(), resp.StatusCode)
					http.Redirect(w, r, reqURL.String(), http.StatusFound)
				}
			}
			return nil
		},
	}).ServeHTTP(w, r)
}

func (p *Proxy) handleJenkinsUIRequest(w http.ResponseWriter, r *http.Request, requestLogEntry *log.Entry) (cacheKey string, ns string, noProxy bool) {
	//redirect determines if we need to redirect to auth service
	needsAuth := true
	noProxy = true

	//redirectURL is used for auth service as a target of successful auth redirect
	redirectURL, err := url.ParseRequestURI(fmt.Sprintf("%s%s", strings.TrimRight(p.redirect, "/"), r.URL.Path))
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}

	//If the user provides OSO token, we can directly proxy
	if _, ok := r.Header["Authorization"]; ok { //FIXME Do we need this?
		needsAuth = false
	}

	if tj, ok := r.URL.Query()["token_json"]; ok { //If there is token_json in query, process it, find user info and login to Jenkins
		if len(tj) < 1 {
			p.HandleError(w, errors.New("could not read JWT token from URL"), requestLogEntry)
			return
		}

		pci, osioToken, err := p.processToken([]byte(tj[0]), requestLogEntry)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
			return
		}
		ns = pci.NS
		clusterURL := pci.ClusterURL
		requestLogEntry.WithFields(log.Fields{"ns": ns, "cluster": clusterURL}).Debug("Found token info in query")

		isIdle, err := p.idler.IsIdle(ns, clusterURL)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
			return
		}

		//Break the process if the Jenkins is idled, set a cookie and redirect to self
		if isIdle {
			_, err := p.idler.UnIdle(ns, clusterURL)
			if err != nil {
				p.HandleError(w, err, requestLogEntry)
				return
			}

			p.setIdledCookie(w, pci)
			requestLogEntry.WithField("ns", ns).Info("Redirecting to remove token from URL")
			http.Redirect(w, r, redirectURL.String(), http.StatusFound) //Redirect to get rid of token in URL
			return
		}

		osoToken, err := GetOSOToken(p.authURL, pci.ClusterURL, osioToken)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
			return
		}
		requestLogEntry.WithField("ns", ns).Debug("Loaded OSO token")

		statusCode, cookies, err := p.loginJenkins(pci, osoToken, requestLogEntry)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
			return
		}
		if statusCode == http.StatusOK {
			for _, cookie := range cookies {
				if cookie.Name == CookieJenkinsIdled {
					continue
				}
				http.SetCookie(w, cookie)
				if strings.HasPrefix(cookie.Name, SessionCookie) { //Find session cookie and use it's value as a key for cache
					p.ProxyCache.SetDefault(cookie.Value, pci)
					requestLogEntry.WithField("ns", ns).Infof("Cached Jenkins route %s in %s", pci.Route, cookie.Value)
					requestLogEntry.WithField("ns", ns).Infof("Redirecting to %s", redirectURL.String())
					//If all good, redirect to self to remove token from url
					http.Redirect(w, r, redirectURL.String(), http.StatusFound)
					return
				}

				//If we got here, the cookie was not found - report error
				p.HandleError(w, fmt.Errorf("could not find cookie %s for %s", SessionCookie, pci.NS), requestLogEntry)
			}
		} else {
			p.HandleError(w, fmt.Errorf("could not login to Jenkins in %s namespace", ns), requestLogEntry)
		}
	}

	if len(r.Cookies()) > 0 { //Check cookies and proxy cache to find user info
		for _, cookie := range r.Cookies() {
			cacheVal, ok := p.ProxyCache.Get(cookie.Value)
			if !ok {
				continue
			}
			if strings.HasPrefix(cookie.Name, SessionCookie) { //We found a session cookie in cache
				cacheKey = cookie.Value
				pci := cacheVal.(CacheItem)
				r.Host = pci.Route //Configure proxy upstream
				r.URL.Host = pci.Route
				r.URL.Scheme = pci.Scheme
				ns = pci.NS
				needsAuth = false //user is probably logged in, do not redirect
				noProxy = false
				break
			} else if cookie.Name == CookieJenkinsIdled { //Found a cookie saying Jenkins is idled, verify and act accordingly
				cacheKey = cookie.Value
				needsAuth = false
				pci := cacheVal.(CacheItem)
				ns = pci.NS
				clusterURL := pci.ClusterURL
				isIdle, err := p.idler.IsIdle(ns, clusterURL)
				if err != nil {
					p.HandleError(w, err, requestLogEntry)
					return
				}
				if isIdle { //If jenkins is idled, return loading page and status 202
					code, err := p.idler.UnIdle(ns, clusterURL)
					if err != nil {
						p.HandleError(w, err, requestLogEntry)
						return
					}

					if code == http.StatusOK {
						code = http.StatusAccepted
					} else if code != http.StatusServiceUnavailable {
						p.HandleError(w, fmt.Errorf("Failed to send unidle request using fabric8-jenkins-idler"), requestLogEntry)
					}

					w.WriteHeader(code)
					err = p.processTemplate(w, ns, requestLogEntry)
					if err != nil {
						p.HandleError(w, err, requestLogEntry)
						return
					}
					p.recordStatistics(pci.NS, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?
				} else { //If Jenkins is running, remove the cookie
					//OpenShift can take up to couple tens of second to update HAProxy configuration for new route
					//so even if the pod is up, route might still return 500 - i.e. we need to check the route
					//before claiming Jenkins is up
					var statusCode int
					statusCode, _, err = p.loginJenkins(pci, "", requestLogEntry)
					if err != nil {
						p.HandleError(w, err, requestLogEntry)
						return
					}
					if statusCode == 200 || statusCode == 403 {
						cookie.Expires = time.Unix(0, 0)
						http.SetCookie(w, cookie)
					} else {
						w.WriteHeader(http.StatusAccepted)
						err = p.processTemplate(w, ns, requestLogEntry)
						if err != nil {
							p.HandleError(w, err, requestLogEntry)
							return
						}
					}
				}

				if err != nil {
					p.HandleError(w, err, requestLogEntry)
				}
				break
			}
		}
		if len(cacheKey) == 0 { //If we do not have user's info cached, run through login process to get it
			requestLogEntry.WithField("ns", ns).Info("Could not find cache, redirecting to re-login")
		} else {
			requestLogEntry.WithField("ns", ns).Infof("Found cookie %s", cacheKey)
		}
	}

	//Check if we need to redirect to auth service
	if needsAuth {
		redirAuth := GetAuthURI(p.authURL, redirectURL.String())
		requestLogEntry.Infof("Redirecting to auth: %s", redirAuth)
		http.Redirect(w, r, redirAuth, http.StatusTemporaryRedirect)
	}
	return
}

func (p *Proxy) loginJenkins(pci CacheItem, osoToken string, requestLogEntry *log.Entry) (int, []*http.Cookie, error) {
	//Login to Jenkins with OSO token to get cookies
	jenkinsURL := fmt.Sprintf("%s://%s/securityRealm/commenceLogin?from=%%2F", pci.Scheme, pci.Route)
	req, _ := http.NewRequest("GET", jenkinsURL, nil)
	if len(osoToken) > 0 {
		requestLogEntry.WithField("ns", pci.NS).Infof("Jenkins login for %s", jenkinsURL)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", osoToken))
	} else {
		requestLogEntry.WithField("ns", pci.NS).Infof("Accessing Jenkins route %s", jenkinsURL)
	}
	c := http.DefaultClient
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Cookies(), err
}

func (p *Proxy) setIdledCookie(w http.ResponseWriter, pci CacheItem) {
	c := &http.Cookie{}
	u1 := uuid.NewV4().String()
	c.Name = CookieJenkinsIdled
	c.Value = u1
	p.ProxyCache.SetDefault(u1, pci)
	http.SetCookie(w, c)
	return
}

func (p *Proxy) handleGitHubRequest(w http.ResponseWriter, r *http.Request, requestLogEntry *log.Entry) (ns string, noProxy bool) {
	noProxy = true
	//Load request body if it's GH webhook
	gh := GHHookStruct{}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}

	err = json.Unmarshal(body, &gh)
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}

	requestLogEntry.WithField("json", gh).Debug("Processing GitHub JSON payload")

	namespace, err := p.getUserWithRetry(gh.Repository.CloneURL, requestLogEntry, defaultRetry)
	ns = namespace.Name
	clusterURL := namespace.ClusterURL
	requestLogEntry.WithFields(log.Fields{"ns": ns, "cluster": clusterURL, "repository": gh.Repository.CloneURL}).Info("Processing GitHub request ")
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}
	route, scheme, err := p.constructRoute(namespace.ClusterURL, namespace.Name)
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}

	r.URL.Scheme = scheme
	r.URL.Host = route
	r.Host = route

	isIdle, err := p.idler.IsIdle(ns, clusterURL)
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}

	//If Jenkins is idle, we need to cache the request and return success
	if isIdle {
		p.storeGHRequest(w, r, ns, body, requestLogEntry)
		_, err = p.idler.UnIdle(ns, clusterURL)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
		}
		return
	}

	noProxy = false
	r.Body = ioutil.NopCloser(bytes.NewReader(body))
	//If Jenkins is up, we can simply proxy through
	requestLogEntry.WithField("ns", ns).Infof(fmt.Sprintf("Passing through %s", r.URL.String()))
	return
}

func (p *Proxy) storeGHRequest(w http.ResponseWriter, r *http.Request, ns string, body []byte, requestLogEntry *log.Entry) {
	w.Header().Set("Server", "Webhook-Proxy")
	sr, err := storage.NewRequest(r, ns, body)
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}
	err = p.storageService.CreateRequest(sr)
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}
	err = p.recordStatistics(ns, 0, time.Now().Unix())
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}

	requestLogEntry.WithField("ns", ns).Info("Webhook request buffered")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(""))
	return
}

func (p *Proxy) processTemplate(w http.ResponseWriter, ns string, requestLogEntry *log.Entry) (err error) {
	tmplt, err := template.ParseFiles(p.indexPath)
	if err != nil {
		return
	}
	data := struct{ Retry int }{Retry: 15}
	requestLogEntry.WithField("ns", ns).Debug("Templating index.html")
	err = tmplt.Execute(w, data)

	return
}

func (p *Proxy) processToken(tokenData []byte, requestLogEntry *log.Entry) (pci CacheItem, osioToken string, err error) {
	tokenJSON := &TokenJSON{}
	err = json.Unmarshal(tokenData, tokenJSON)
	if err != nil {
		return
	}

	uid, err := GetTokenUID(tokenJSON.AccessToken, p.publicKey)
	if err != nil {
		return
	}

	ti, err := p.tenant.GetTenantInfo(uid)
	if err != nil {
		return
	}
	osioToken = tokenJSON.AccessToken

	namespace, err := p.tenant.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return
	}

	requestLogEntry.WithField("ns", namespace.Name).Debug("Extracted information from token")
	route, scheme, err := p.constructRoute(namespace.ClusterURL, namespace.Name)
	if err != nil {
		return
	}

	//Prepare an item for proxyCache - Jenkins info and OSO token
	pci = NewCacheItem(namespace.Name, scheme, route, namespace.ClusterURL)

	return
}

//HandleError creates a JSON response with a given error and writes it to ResponseWriter
func (p *Proxy) HandleError(w http.ResponseWriter, err error, requestLogEntry *log.Entry) {
	// log the error
	location := ""
	if err != nil {
		pc, fn, line, _ := runtime.Caller(1)

		location = fmt.Sprintf(" %s[%s:%d]", runtime.FuncForPC(pc).Name(), fn, line)
	}

	requestLogEntry.WithFields(
		log.Fields{
			"location": location,
			"error":    err,
		}).Error("Error Handling proxy request request.")

	// create error response
	w.WriteHeader(http.StatusInternalServerError)

	pei := ErrorInfo{
		Code:   fmt.Sprintf("%d", http.StatusInternalServerError),
		Detail: err.Error(),
	}
	e := Error{
		Errors: make([]ErrorInfo, 1),
	}
	e.Errors[0] = pei

	eb, err := json.Marshal(e)
	if err != nil {
		requestLogEntry.Error(err)
	}
	w.Write(eb)
}

func (p *Proxy) isGitHubRequest(r *http.Request) bool {
	isGH := false
	// Is the request coming from Github Webhook?
	// FIXME - should we only care about push?
	if ua, exist := r.Header[GHHeader]; exist {
		isGH = strings.HasPrefix(ua[0], GHAgent)
	}
	return isGH
}

func (p *Proxy) createRequestHash(url string, headers string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(url + headers + fmt.Sprint(time.Now())))
	return h.Sum32()
}

func (p *Proxy) getUserWithRetry(repositoryCloneURL string, logEntry *log.Entry, retry int) (clients.Namespace, error) {

	for i := 0; i < retry; i++ {
		namespace, err := p.getUser(repositoryCloneURL, logEntry)

		if err == nil {
			return namespace, err
		}

		time.Sleep(1000 * time.Millisecond)

	}

	// Last chance
	return p.getUser(repositoryCloneURL, logEntry)

}

//GetUser returns a namespace name based on GitHub repository URL
func (p *Proxy) getUser(repositoryCloneURL string, logEntry *log.Entry) (clients.Namespace, error) {
	if n, found := p.TenantCache.Get(repositoryCloneURL); found {
		namespace := n.(clients.Namespace)
		logEntry.WithFields(
			log.Fields{
				"ns": namespace,
			}).Infof("Cache hit for repository %s", repositoryCloneURL)
		return namespace, nil
	}

	logEntry.Infof("Cache miss for repository %s", repositoryCloneURL)
	wi, err := p.wit.SearchCodebase(repositoryCloneURL)
	if err != nil {
		return clients.Namespace{}, err
	}

	if len(strings.TrimSpace(wi.OwnedBy)) == 0 {
		return clients.Namespace{}, fmt.Errorf("unable to determine tenant id for repository %s", repositoryCloneURL)
	}

	logEntry.Infof("Found id %s for repo %s", wi.OwnedBy, repositoryCloneURL)
	ti, err := p.tenant.GetTenantInfo(wi.OwnedBy)
	if err != nil {
		return clients.Namespace{}, err
	}

	n, err := p.tenant.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return clients.Namespace{}, err
	}

	p.TenantCache.SetDefault(repositoryCloneURL, n)
	return n, nil
}

//RecordStatistics writes usage statistics to a database
func (p *Proxy) recordStatistics(ns string, la int64, lbf int64) (err error) {
	log.WithField("ns", ns).Debug("Recording stats")
	s, notFound, err := p.storageService.GetStatisticsUser(ns)
	if err != nil {
		log.WithFields(
			log.Fields{
				"ns": ns,
			}).Warningf("Could not load statistics: %s", err)
		if !notFound {
			return
		}
	}

	if notFound {
		log.WithField("ns", ns).Infof("New user %s", ns)
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
		log.WithField("ns", ns).Errorf("Could not update statistics for %s: %s", ns, err)
	}

	return
}

//constructRoute returns Jenkins route based on a specific pattern
func (p *Proxy) constructRoute(clusterURL string, ns string) (string, string, error) {
	appSuffix := p.clusters[clusterURL]
	if len(appSuffix) == 0 {
		return "", "", fmt.Errorf("could not find entry for cluster %s", clusterURL)
	}
	route := fmt.Sprintf("jenkins-%s.%s", ns, p.clusters[clusterURL])
	return route, "https", nil
}

//ProcessBuffer is a loop running through buffered webhook requests trying to replay them
func (p *Proxy) ProcessBuffer() {
	for {
		namespaces, err := p.storageService.GetUsers()
		if err != nil {
			log.Error(err)
		} else {
			for _, ns := range namespaces {
				requests, err := p.storageService.GetRequests(ns)
				if err != nil {
					log.Error(err)
					continue
				}
				for _, r := range requests {
					gh := GHHookStruct{}
					err = json.Unmarshal(r.Payload, &gh)
					if err != nil {
						log.Error(err)
						break
					}

					log.WithFields(log.Fields{"ns": ns, "repository": gh.Repository.CloneURL}).Info("Retrying request")
					namespace, err := p.getUserWithRetry(gh.Repository.CloneURL, proxyLogger, defaultRetry)
					clusterURL := namespace.ClusterURL

					isIdle, err := p.idler.IsIdle(ns, clusterURL)
					if err != nil {
						log.Error(err)
						break
					}
					err = p.recordStatistics(ns, 0, time.Now().Unix())
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
								errs := p.storageService.IncrementRequestRetry(&r)
								if len(errs) > 0 {
									for _, e := range errs {
										log.Error(e)
									}
								}
								break
							}

							if resp.StatusCode != 200 && resp.StatusCode != 404 { //Retry later if the response is not 200
								log.Error(fmt.Sprintf("Got status %s after retrying request on %s", resp.Status, req.URL))
								errs := p.storageService.IncrementRequestRetry(&r)
								if len(errs) > 0 {
									for _, e := range errs {
										log.Error(e)
									}
								}
								break
							} else if resp.StatusCode == 404 || resp.StatusCode == 400 { //400 - missing payload
								log.Warn(fmt.Sprintf("Got status %s after retrying request on %s, throwing away the request", resp.Status, req.URL))
							}

							log.WithField("ns", ns).Infof(fmt.Sprintf("Request for %s to %s forwarded.", ns, req.Host))
						}

						//Delete request if we tried too many times or the replay was successful
						err = p.storageService.DeleteRequest(&r)
						if err != nil {
							log.Errorf(storage.ErrorFailedDelete, r.ID, r.Namespace, err)
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
