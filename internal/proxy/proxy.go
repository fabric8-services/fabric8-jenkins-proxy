package proxy

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"hash/fnv"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/metric"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy/cookieutil"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy/reverseproxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util/logging"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/wit"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

const (
	defaultRetry = 15

	// ServiceName is name of service that we are trying to idle or unidle
	ServiceName = "jenkins"
)

var proxyLogger = log.WithFields(log.Fields{"component": "proxy"})

// Recorder to capture events
var Recorder = metric.PrometheusRecorder{}

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
	tenant           tenant.Service
	wit              wit.Service
	idler            idler.Service
	//redirect is a base URL of the proxy
	redirect        string
	responseTimeout time.Duration
	authURL         string
	storageService  storage.Store
	indexPath       string
	maxRequestRetry int
	clusters        map[string]string
}

// New creates an instance of Proxy client
func New(
	idler idler.Service,
	tenant tenant.Service,
	wit wit.Service,
	storageService storage.Store,
	config configuration.Configuration,
	clusters map[string]string) (Proxy, error) {

	p := Proxy{
		TenantCache:      cache.New(30*time.Minute, 40*time.Minute),
		ProxyCache:       cache.New(15*time.Minute, 10*time.Minute),
		visitLock:        &sync.Mutex{},
		tenant:           tenant,
		wit:              wit,
		idler:            idler,
		bufferCheckSleep: 30 * time.Second,
		redirect:         config.GetRedirectURL(),
		responseTimeout:  config.GetGatewayTimeout(),
		authURL:          config.GetAuthURL(),
		storageService:   storageService,
		indexPath:        config.GetIndexPath(),
		maxRequestRetry:  config.GetMaxRequestRetry(),
		clusters:         clusters,
	}

	//Initialize metrics
	Recorder.Initialize()

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

	Recorder.RecordReqByTypeTotal(requestType)

	// store copy of the actual url so that it can be passed to reverse-proxy
	// to force refreshing by redirecting to the actual url
	actualURL := *r.URL

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
	var okToForward bool

	// NOTE: Response payload and status codes (including errors) are written
	// to the ResponseWriter (w) in the called methods

	if isGH {
		ns, okToForward = p.handleGitHubRequest(w, r, logEntryWithHash)
	} else {
		// If this is no GitHub traffic (e.g. user accessing UI)
		cacheKey, ns, okToForward = p.handleJenkinsUIRequest(w, r, logEntryWithHash)
		logEntryWithHash.Infof("returned: |key: %q |ns: %q |fwd: %v|", cacheKey, ns, okToForward)
	}

	if !okToForward {
		return
	}

	//Write usage stats to DB, run in a goroutine to not slow down the proxy
	go func() {
		p.recordStatistics(ns, time.Now().Unix(), 0)
	}()

	var onError func(http.ResponseWriter, *http.Request, int) error
	if isGH {
		onError = func(rw http.ResponseWriter, req *http.Request, code int) error {
			return nil
		}
	} else {
		onError = p.OnErrorUIRequest
	}
	// at this point we know jenkins is up and running let the reverse-proxy
	// forward request to actual jenkins
	rp := reverseproxy.NewReverseProxy(
		actualURL,
		p.responseTimeout,
		onError,
		logEntryWithHash,
	)
	rp.ServeHTTP(w, r)
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

func (p *Proxy) invalidateSession(w http.ResponseWriter, sessionCookies []*http.Cookie) {
	for _, cookie := range sessionCookies {
		var pci CacheItem

		cacheKey := cookie.Value
		cacheVal, ok := p.ProxyCache.Get(cacheKey)
		if ok {
			pci = cacheVal.(CacheItem)
			p.ProxyCache.Delete(cacheKey)

			proxyLogger.Infof("clearing cache for namespace: %s, cache_key: %s", pci.NS, cacheKey)
		}

		cookieutil.ExpireCookie(w, cookie)
		proxyLogger.Infof("cookie is OLD; expiring the cookie, cookie_name: %s, namespace: %s", cookie.Name, pci.NS)

	}
}

func checkSessionValidity(req *http.Request, sessionCookie *http.Cookie, timeout time.Duration) (bool, error) {

	requestURL := req.URL
	jenkinsURL := fmt.Sprintf("%s://%s", requestURL.Scheme, requestURL.Host)

	ctx, cancel := context.WithTimeout(req.Context(), timeout)
	defer cancel()

	r, _ := http.NewRequest("GET", jenkinsURL, nil)
	r.AddCookie(sessionCookie)
	r = r.WithContext(ctx)

	c := &http.Client{Timeout: timeout}

	resp, err := c.Do(r)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, err

	case http.StatusForbidden:
		return false, err
	}
	err = fmt.Errorf("received unexpected status code: %s", resp.Status)
	return false, err
}

// OnErrorUIRequest handles when there is an error while reverse proxy
func (p *Proxy) OnErrorUIRequest(rw http.ResponseWriter, req *http.Request, code int) error {
	if code != http.StatusForbidden {
		return nil
	}

	sessionCookies := cookieutil.Filter(req.Cookies(), cookieutil.IsSessionCookie)

	if len(sessionCookies) > 1 {
		p.invalidateSession(rw, sessionCookies)
		return nil
	}

	valid, err := checkSessionValidity(req, sessionCookies[0], p.responseTimeout)
	if err != nil {
		return err
	}

	if !valid {
		p.invalidateSession(rw, sessionCookies)
	}
	return nil
}
