package proxy

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"hash/fnv"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/metric"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util/logging"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

const (
	defaultRetry = 15
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
func NewProxy(
	tenant *clients.Tenant, wit clients.WIT, idler clients.IdlerService,
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
			//Check response from Jenkins and redirect if it got idled in the meantime
			if resp.StatusCode == http.StatusServiceUnavailable ||
				resp.StatusCode == http.StatusGatewayTimeout {

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

//constructRoute returns Jenkins route based on a specific pattern
func (p *Proxy) constructRoute(clusterURL string, ns string) (string, string, error) {
	appSuffix := p.clusters[clusterURL]
	if len(appSuffix) == 0 {
		return "", "", fmt.Errorf("could not find entry for cluster %s", clusterURL)
	}
	route := fmt.Sprintf("jenkins-%s.%s", ns, p.clusters[clusterURL])
	return route, "https", nil
}
