package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/tenant"
	log "github.com/sirupsen/logrus"
)

// JenkinsService talks to wit service, auth service, tenant service and
// various jenkins instances in order to perform task such as starting
// a jenkins, logging into one, get information for a jenkins instance
// such as its url, which namespace it belongs to, which cluster it
// belongs to etc
type JenkinsService interface {
	Login(osoToken string) (status int, cookie []*http.Cookie, err error)
	State() (idler.PodState, error)
	Start() (state idler.PodState, code int, err error)
}

// Jenkins implements Jenkins interface
type Jenkins struct {
	info CacheItem

	idler  idler.Service
	tenant tenant.Service

	logger *log.Entry
}

// GetJenkins returns an intance of Jenkins struct
func GetJenkins(clusters map[string]string,
	pci *CacheItem,
	idler idler.Service,
	tenantClient tenant.Service,
	tokenData string,
	logger *log.Entry) (j *Jenkins, osioToken string, err error) {

	if pci != nil {
		return &Jenkins{
			info:   *pci,
			idler:  idler,
			tenant: tenantClient,
			logger: logger.WithFields(log.Fields{"ns": pci.NS, "cluster": pci.ClusterURL}),
		}, "", nil
	}

	tokenJSON := &auth.TokenJSON{}
	err = json.Unmarshal([]byte(tokenData), tokenJSON)
	if err != nil {
		return j, "", err
	}

	authClient, err := auth.DefaultClient()
	if err != nil {
		return &Jenkins{}, "", err
	}
	uid, err := authClient.UIDFromToken(tokenJSON.AccessToken)
	if err != nil {
		return &Jenkins{}, "", err
	}

	ti, err := tenantClient.GetTenantInfo(uid)
	if err != nil {
		return &Jenkins{}, "", err
	}
	osioToken = tokenJSON.AccessToken

	namespace, err := tenant.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return &Jenkins{}, osioToken, err
	}

	logger.WithField("ns", namespace.Name).Debug("Extracted information from token")
	route, scheme, err := constructRoute(clusters, namespace.ClusterURL, namespace.Name)
	if err != nil {
		return &Jenkins{}, osioToken, err
	}

	return &Jenkins{
		info:   NewCacheItem(namespace.Name, scheme, route, namespace.ClusterURL),
		idler:  idler,
		tenant: tenantClient,
		logger: logger.WithFields(log.Fields{"ns": namespace.Name, "cluster": namespace.ClusterURL}),
	}, osioToken, nil
}

//Login to Jenkins with OSO token to get cookies
func (j *Jenkins) Login(osoToken string) (status int, cookie []*http.Cookie, err error) {

	jenkinsURL := fmt.Sprintf("%s://%s/securityRealm/commenceLogin?from=%%2F", j.info.Scheme, j.info.Route)

	req, _ := http.NewRequest("GET", jenkinsURL, nil)
	if len(osoToken) > 0 {
		j.logger.WithField("ns", j.info.NS).Infof("Jenkins login for %s", jenkinsURL)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", osoToken))
	} else {
		j.logger.WithField("ns", j.info.NS).Infof("Accessing Jenkins route %s", jenkinsURL)
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
func (j *Jenkins) State() (idler.PodState, error) {
	return j.idler.State(j.info.NS, j.info.ClusterURL)
}

// Start unidles Jenkins only if it is idled and returns the
// state of the pod, the http status of calling unidle, and error if any
func (j *Jenkins) Start() (state idler.PodState, code int, err error) {
	// Assume pods are starting and unidle only if it is in "idled" state
	code = http.StatusAccepted
	ns := j.info.NS
	clusterURL := j.info.ClusterURL

	state, err = j.idler.State(ns, clusterURL)
	if err != nil {
		return
	}
	j.logger.Infof("state : %q", state)

	if state == idler.Idled {
		// Unidle only if needed
		j.logger.Infof("Unidling jenkins")
		if code, err = j.idler.UnIdle(ns, clusterURL); err != nil {
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
