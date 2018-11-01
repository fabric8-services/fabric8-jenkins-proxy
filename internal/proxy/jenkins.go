package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	log "github.com/sirupsen/logrus"
)

// JenkinsService talks to wit service, auth service, tenant service and
// various jenkins instances in order to perform task such as starting
// a jenkins, logging into one, get information for a jenkins instance
// such as its url, which namespace it belongs to, which cluster it
// belongs to etc
type JenkinsService interface {
	Info(tokenData string) (CacheItem, string, error)
	Login(pci CacheItem, osoToken string) (int, []*http.Cookie, error)
	State(ns string, clusterURL string) (clients.PodState, error)
	Start(ns string, clusterURL string) (state clients.PodState, code int, err error)
}

// Jenkins implements Jenkins interface
type Jenkins struct {
	clusters map[string]string

	idler  clients.IdlerService
	tenant clients.TenantService

	logger *log.Entry
}

// NewJenkins returns an intance of Jenkins struct
func NewJenkins(clusters map[string]string, idler clients.IdlerService, tenant clients.TenantService, logger *log.Entry) *Jenkins {
	return &Jenkins{
		clusters: clusters,
		idler:    idler,
		tenant:   tenant,
		logger:   logger,
	}
}

// Info gets you proxy cache and OSIO token for a user to which given
// token json belongs
func (jenkins *Jenkins) Info(tokenData string) (CacheItem, string, error) {
	tokenJSON := &auth.TokenJSON{}
	err := json.Unmarshal([]byte(tokenData), tokenJSON)
	if err != nil {
		return CacheItem{}, "", err
	}

	uid, err := auth.DefaultClient().UIDFromToken(tokenJSON.AccessToken)
	if err != nil {
		return CacheItem{}, "", err
	}

	ti, err := jenkins.tenant.GetTenantInfo(uid)
	if err != nil {
		return CacheItem{}, "", err
	}
	osioToken := tokenJSON.AccessToken

	namespace, err := clients.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return CacheItem{}, osioToken, err
	}

	jenkins.logger.WithField("ns", namespace.Name).Debug("Extracted information from token")
	route, scheme, err := jenkins.constructRoute(namespace.ClusterURL, namespace.Name)
	if err != nil {
		return CacheItem{}, osioToken, err
	}

	//Prepare an item for proxyCache - Jenkins info and OSO token
	pci := NewCacheItem(namespace.Name, scheme, route, namespace.ClusterURL)

	return pci, osioToken, nil
}

//Login to Jenkins with OSO token to get cookies
func (jenkins *Jenkins) Login(pci CacheItem, osoToken string) (int, []*http.Cookie, error) {

	jenkinsURL := fmt.Sprintf("%s://%s/securityRealm/commenceLogin?from=%%2F", pci.Scheme, pci.Route)

	req, _ := http.NewRequest("GET", jenkinsURL, nil)
	if len(osoToken) > 0 {
		jenkins.logger.WithField("ns", pci.NS).Infof("Jenkins login for %s", jenkinsURL)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", osoToken))
	} else {
		jenkins.logger.WithField("ns", pci.NS).Infof("Accessing Jenkins route %s", jenkinsURL)
	}
	c := http.DefaultClient
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Cookies(), err
}

// State returns state of Jenkins associated with given namespace
func (jenkins *Jenkins) State(ns string, clusterURL string) (clients.PodState, error) {
	return jenkins.idler.State(ns, clusterURL)
}

// Start unidles Jenkins only if it is idled and returns the
// state of the pod, the http status of calling unidle, and error if any
func (jenkins *Jenkins) Start(ns string, clusterURL string) (state clients.PodState, code int, err error) {
	// Assume pods are starting and unidle only if it is in "idled" state
	code = http.StatusAccepted
	//nsLogger := log.WithFields(log.Fields{"ns": ns, "cluster": clusterURL})

	state, err = jenkins.idler.State(ns, clusterURL)
	if err != nil {
		return
	}
	jenkins.logger.Infof("state : %q", state)

	if state == clients.Idled {
		// Unidle only if needed
		jenkins.logger.Infof("Unidling jenkins")
		if code, err = jenkins.idler.UnIdle(ns, clusterURL); err != nil {
			return
		}
	}
	if code == http.StatusOK {
		// XHR relies on 202 to retry and 200 to stop retrying and reload
		// since we just started jenkins pods, change the code to 202 so
		// that it retries
		// SEE: static/html/index.html
		code = http.StatusAccepted
	}
	return state, code, nil
}

// constructRoute returns Jenkins route based on a specific pattern
func (jenkins *Jenkins) constructRoute(clusterURL string, ns string) (string, string, error) {
	appSuffix := jenkins.clusters[clusterURL]
	if len(appSuffix) == 0 {
		return "", "", fmt.Errorf("could not find entry for cluster %s", clusterURL)
	}
	route := fmt.Sprintf("jenkins-%s.%s", ns, jenkins.clusters[clusterURL])
	return route, "https", nil
}
