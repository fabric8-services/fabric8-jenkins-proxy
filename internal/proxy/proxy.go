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

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	jsonerrors "github.com/fabric8-services/fabric8-jenkins-proxy/internal/util/errors"
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
)

var proxyLogger = log.WithFields(log.Fields{"component": "proxy"})

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
	wit              *clients.WIT
	idler            clients.IdlerService

	//redirect is a base URL of the proxy
	redirect        string
	publicKey       *rsa.PublicKey
	authURL         string
	storageService  storage.Store
	indexPath       string
	maxRequestRetry int
	clusters        map[string]string
	loginInstance   interfaceOflogin
}

// NewProxy creates an instance of Proxy client
func NewProxy(tenant *clients.Tenant, wit *clients.WIT, idler clients.IdlerService, storageService storage.Store, config configuration.Configuration, clusters map[string]string) (Proxy, error) {
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
		loginInstance:    &login{},
	}

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
	isGH := p.isGitHubRequest(r)
	var requestType string
	if isGH {
		requestType = "GitHub"
	} else {
		requestType = "Jenkins UI"
	}

	requestURL := logging.RequestMethodAndURL(r)
	requestHeaders := logging.RequestHeaders(r)
	requestHash := createRequestHash(requestURL, requestHeaders)
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

func (p *Proxy) handleJenkinsUIRequest(responseWriter http.ResponseWriter, request *http.Request, requestLogEntry *log.Entry) (cacheKey string, namespace string, noProxy bool) {

	needsAuth := true
	noProxy = true

	// Checks if token_json is present in the query
	if tokenJSON, ok := request.URL.Query()["token_json"]; ok {
		namespace = p.processAuthenticatedRequest(responseWriter, request, requestLogEntry, tokenJSON, &noProxy)
	}

	if len(request.Cookies()) > 0 {
		cacheKey, namespace, noProxy, needsAuth = p.checkCookies(responseWriter, request, requestLogEntry)
	}
	//Check if we need to redirect to auth service
	if needsAuth {
		p.redirectToAuth(responseWriter, request, requestLogEntry)
	}

	return cacheKey, namespace, noProxy
}

func (p *Proxy) redirectToAuth(responseWriter http.ResponseWriter, request *http.Request, requestLogEntry *log.Entry) {
	// redirectURL is used for auth service as a target of successful auth redirect
	redirectURL, err := url.ParseRequestURI(fmt.Sprintf("%s%s", strings.TrimRight(p.redirect, "/"), request.URL.Path))
	if err != nil {
		jsonerrors.JSONError(responseWriter, err, requestLogEntry)
		return
	}

	redirAuth := GetAuthURI(p.authURL, redirectURL.String())
	requestLogEntry.Infof("Redirecting to auth: %s", redirAuth)
	http.Redirect(responseWriter, request, redirAuth, 301)
}

// processAuthenticatedRequest processes token_json if present in query and find user info and login to Jenkins and redirects the target page
func (p *Proxy) processAuthenticatedRequest(responseWriter http.ResponseWriter, request *http.Request, requestLogEntry *log.Entry, tokenJSON []string, noProxy *bool) (namespace string) {

	// redirectURL is used for auth service as a target of successful auth redirect
	redirectURL, err := url.ParseRequestURI(fmt.Sprintf("%s%s", strings.TrimRight(p.redirect, "/"), request.URL.Path))
	if err != nil {
		jsonerrors.JSONError(responseWriter, err, requestLogEntry)
		return namespace
	}

	if len(tokenJSON) < 1 {
		jsonerrors.JSONError(responseWriter, errors.New("could not read JWT token from URL"), requestLogEntry)
		return namespace
	}

	proxyCacheItem, osioToken, err := p.loginInstance.processToken([]byte(tokenJSON[0]), requestLogEntry, p)
	if err != nil {
		jsonerrors.JSONError(responseWriter, err, requestLogEntry)
		return namespace
	}

	namespace = proxyCacheItem.NS
	clusterURL := proxyCacheItem.ClusterURL
	requestLogEntry.WithField("ns", namespace).Debug("Found token info in query")

	// Check if jenkins is idled
	isIdle, err := p.idler.IsIdle(namespace, clusterURL)
	if err != nil {
		jsonerrors.JSONError(responseWriter, err, requestLogEntry)
		return namespace

	}

	// Break the process if the Jenkins is idled, set a cookie and redirect to self
	if isIdle {
		// Initiates unidling of the Jenkins instance
		err = p.idler.UnIdle(namespace, clusterURL)
		if err != nil {
			jsonerrors.JSONError(responseWriter, err, requestLogEntry)
			return namespace
		}

		// sets a cookie, in which cookie value is id of proxyCacheItem in cache
		p.setIdledCookie(responseWriter, *proxyCacheItem)
		requestLogEntry.WithField("ns", namespace).Info("Redirecting to remove token from URL")
		// Redirect to get rid of token in URL
		http.Redirect(responseWriter, request, redirectURL.String(), http.StatusFound)
		return namespace
	}
	// The part below here is executed only if Jenkins is not idle

	// gets openshift online token
	osoToken, err := p.loginInstance.GetOSOToken(p.authURL, clusterURL, osioToken)
	if err != nil {
		jsonerrors.JSONError(responseWriter, err, requestLogEntry)
		return namespace
	}
	requestLogEntry.WithField("ns", namespace).Debug("Loaded OSO token")

	// login to Jenkins and gets cookies
	statusCode, cookies, err := p.loginInstance.loginJenkins(*proxyCacheItem, osoToken, requestLogEntry)
	if err != nil {
		jsonerrors.JSONError(responseWriter, err, requestLogEntry)
		return namespace
	}

	if statusCode == http.StatusOK {
		// sets all cookies that we got from loginJenkins method
		for _, cookie := range cookies {
			// No need to set a cookie for whether jenkins if idled, because we have already done that
			if cookie.Name == CookieJenkinsIdled {
				continue
			}
			http.SetCookie(responseWriter, cookie)
			if strings.HasPrefix(cookie.Name, SessionCookie) {
				// Find session cookie and use it's value as a key for cache
				p.ProxyCache.SetDefault(cookie.Value, proxyCacheItem)
				requestLogEntry.WithField("ns", namespace).Infof("Cached Jenkins route %s in %s", proxyCacheItem.Route, cookie.Value)
				requestLogEntry.WithField("ns", namespace).Infof("Redirecting to %s", redirectURL.String())
				//If all good, redirect to self to remove token from url
				http.Redirect(responseWriter, request, redirectURL.String(), http.StatusFound)
				return namespace
			}

			//If we got here, the cookie was not found - report error
			jsonerrors.JSONError(responseWriter, fmt.Errorf("could not find cookie %s for %s", SessionCookie, proxyCacheItem.NS), requestLogEntry)
		}
	} else {
		jsonerrors.JSONError(responseWriter, fmt.Errorf("could not login to Jenkins in %s namespace", namespace), requestLogEntry)
	}
	return namespace
}

// checkCookies checks cookies and proxy cache to find user info
func (p *Proxy) checkCookies(responseWriter http.ResponseWriter, request *http.Request, requestLogEntry *log.Entry) (cacheKey string, namespace string, noProxy bool, needsAuth bool) {
	needsAuth = true
	noProxy = true

	for _, cookie := range request.Cookies() {
		cacheVal, ok := p.ProxyCache.Get(cookie.Value)
		if !ok {
			continue
		}
		if strings.HasPrefix(cookie.Name, SessionCookie) { // We found a session cookie in cache
			cacheKey = cookie.Value
			proxyCacheItem := cacheVal.(CacheItem)
			request.Host = proxyCacheItem.Route // Configure proxy upstream
			request.URL.Host = proxyCacheItem.Route
			request.URL.Scheme = proxyCacheItem.Scheme
			namespace = proxyCacheItem.NS
			needsAuth = false // user is logged in, do not redirect
			noProxy = false
			break
		} else if cookie.Name == CookieJenkinsIdled { // Found a cookie saying Jenkins is idled, verify and act accordingly
			cacheKey = cookie.Value
			needsAuth = false
			proxyCacheItem := cacheVal.(CacheItem)
			namespace = proxyCacheItem.NS
			clusterURL := proxyCacheItem.ClusterURL
			isIdle, err := p.idler.IsIdle(proxyCacheItem.NS, clusterURL)
			if err != nil {
				jsonerrors.JSONError(responseWriter, err, requestLogEntry)
				return cacheKey, namespace, noProxy, needsAuth
			}

			if isIdle { //If jenkins is idled, return loading page and status 202
				err = p.idler.UnIdle(namespace, clusterURL)
				if err != nil {
					jsonerrors.JSONError(responseWriter, err, requestLogEntry)
					return cacheKey, namespace, noProxy, needsAuth
				}

				err = p.processTemplate(responseWriter, namespace, requestLogEntry)
				if err != nil {
					jsonerrors.JSONError(responseWriter, err, requestLogEntry)
				}
				p.recordStatistics(proxyCacheItem.NS, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?
			} else { //If Jenkins is running, remove the cookie
				//OpenShift can take up to couple tens of second to update HAProxy configuration for new route
				//so even if the pod is up, route might still return 500 - i.e. we need to check the route
				//before claiming Jenkins is up
				var statusCode int
				statusCode, _, err = p.loginInstance.loginJenkins(proxyCacheItem, "", requestLogEntry)
				if err != nil {
					jsonerrors.JSONError(responseWriter, err, requestLogEntry)
					return cacheKey, namespace, noProxy, needsAuth
				}
				if statusCode == 200 || statusCode == 403 {
					cookie.Expires = time.Unix(0, 0)
					http.SetCookie(responseWriter, cookie)
				} else {
					err = p.processTemplate(responseWriter, namespace, requestLogEntry)
					if err != nil {
						jsonerrors.JSONError(responseWriter, err, requestLogEntry)
					}
				}
			}
			break
		}
	}

	if len(cacheKey) == 0 { //If we do not have user's info cached, run through login process to get it
		requestLogEntry.WithField("ns", namespace).WithField("needsAuth", needsAuth).Info("Could not find cache, redirecting to re-login")
	} else {
		requestLogEntry.WithField("ns", namespace).Infof("Found cookie %s", cacheKey)
	}

	return cacheKey, namespace, noProxy, needsAuth
}

func (p *Proxy) setIdledCookie(w http.ResponseWriter, proxyCacheItem CacheItem) {
	c := &http.Cookie{}
	id := uuid.NewV4().String()
	c.Name = CookieJenkinsIdled
	c.Value = id
	// Store proxyCacheItem at id in cache
	p.ProxyCache.SetDefault(id, proxyCacheItem)
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
		jsonerrors.JSONError(w, err, requestLogEntry)
		return
	}

	err = json.Unmarshal(body, &gh)
	if err != nil {
		jsonerrors.JSONError(w, err, requestLogEntry)
		return
	}

	requestLogEntry.WithField("json", gh).Debug("Processing GitHub JSON payload")

	namespace, err := p.getUser(gh.Repository.CloneURL, requestLogEntry)
	ns = namespace.Name
	clusterURL := namespace.ClusterURL
	requestLogEntry.WithFields(log.Fields{"ns": ns, "cluster": clusterURL, "repository": gh.Repository.CloneURL}).Info("Processing GitHub request ")
	if err != nil {
		jsonerrors.JSONError(w, err, requestLogEntry)
		return
	}
	route, scheme, err := p.constructRoute(namespace.ClusterURL, namespace.Name)
	if err != nil {
		jsonerrors.JSONError(w, err, requestLogEntry)
		return
	}

	r.URL.Scheme = scheme
	r.URL.Host = route
	r.Host = route

	isIdle, err := p.idler.IsIdle(ns, clusterURL)
	if err != nil {
		jsonerrors.JSONError(w, err, requestLogEntry)
		return
	}

	//If Jenkins is idle, we need to cache the request and return success
	if isIdle {
		p.storeGHRequest(w, r, ns, body, requestLogEntry)
		err = p.idler.UnIdle(ns, clusterURL)
		if err != nil {
			jsonerrors.JSONError(w, err, requestLogEntry)
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
		jsonerrors.JSONError(w, err, requestLogEntry)
		return
	}
	err = p.storageService.CreateRequest(sr)
	if err != nil {
		jsonerrors.JSONError(w, err, requestLogEntry)
		return
	}
	err = p.recordStatistics(ns, 0, time.Now().Unix())
	if err != nil {
		jsonerrors.JSONError(w, err, requestLogEntry)
		return
	}

	requestLogEntry.WithField("ns", ns).Info("Webhook request buffered")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(""))
	return
}

func (p *Proxy) processTemplate(w http.ResponseWriter, ns string, requestLogEntry *log.Entry) (err error) {
	w.WriteHeader(http.StatusAccepted)
	tmplt, err := template.ParseFiles(p.indexPath)
	if err != nil {
		return
	}
	data := struct {
		Message string
		Retry   int
	}{
		Message: "Jenkins has been idled. It is starting now, please wait...",
		Retry:   15,
	}
	requestLogEntry.WithField("ns", ns).Debug("Templating index.html")
	err = tmplt.Execute(w, data)

	return
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

func createRequestHash(url string, headers string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(url + headers + fmt.Sprint(time.Now())))
	return h.Sum32()
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
func (p *Proxy) recordStatistics(namespace string, lastAccessed int64, lastBufferedRequest int64) (err error) {
	log.WithField("namespace", namespace).Debug("Recording stats")
	s, notFound, err := p.storageService.GetStatisticsUser(namespace)
	if err != nil {
		log.WithFields(
			log.Fields{
				"namespace": namespace,
			}).Warningf("Could not load statistics: %s", err)
		if !notFound {
			return
		}
	}

	if notFound {
		log.WithField("namespace", namespace).Infof("New user %s", namespace)
		s = storage.NewStatistics(namespace, lastAccessed, lastBufferedRequest)
		err = p.storageService.CreateStatistics(s)
		if err != nil {
			log.Errorf("Could not create statistics for %s: %s", namespace, err)
		}
		return
	}

	if lastAccessed != 0 {
		s.LastAccessed = lastAccessed
	}

	if lastBufferedRequest != 0 {
		s.LastBufferedRequest = lastBufferedRequest
	}

	p.visitLock.Lock()
	err = p.storageService.UpdateStatistics(s)
	p.visitLock.Unlock()
	if err != nil {
		log.WithField("namespace", namespace).Errorf("Could not update statistics for %s: %s", namespace, err)
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
					namespace, err := p.getUser(gh.Repository.CloneURL, proxyLogger)
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

type interfaceOflogin interface {
	loginJenkins(pci CacheItem, osoToken string, requestLogEntry *log.Entry) (int, []*http.Cookie, error)
	processToken(tokenData []byte, requestLogEntry *log.Entry, p *Proxy) (pci *CacheItem, osioToken string, err error)
	GetOSOToken(authURL string, clusterURL string, token string) (osoToken string, err error)
}

type login struct {
}

func (l *login) loginJenkins(pci CacheItem, osoToken string, requestLogEntry *log.Entry) (int, []*http.Cookie, error) {
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

// Extract openshift.io token (access_token from token_json)
// and get proxy cache item using openshift.io token
func (l *login) processToken(tokenData []byte, requestLogEntry *log.Entry, p *Proxy) (proxyCacheItem *CacheItem, osioToken string, err error) {
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
	pci := NewCacheItem(namespace.Name, scheme, route, namespace.ClusterURL)
	proxyCacheItem = &pci

	return
}

// GetOSOToken returns Openshift online token on giving raw JWT token, cluster URL and auth service url as input.
func (l *login) GetOSOToken(authURL string, clusterURL string, token string) (osoToken string, err error) {
	url := fmt.Sprintf("%s/api/token?for=%s", strings.TrimRight(authURL, "/"), clusterURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	c := http.DefaultClient

	resp, err := c.Do(req)
	if err != nil {
		return
	}

	tj := &TokenJSON{}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, tj)
	if err != nil {
		return
	}

	if len(tj.Errors) > 0 {
		err = fmt.Errorf(tj.Errors[0].Detail)
		return
	}

	if len(tj.AccessToken) > 0 {
		osoToken = tj.AccessToken
	} else {
		err = fmt.Errorf("OSO access token empty for %s", authURL)
	}
	return
}
