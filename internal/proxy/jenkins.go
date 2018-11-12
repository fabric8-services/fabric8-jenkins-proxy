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
	Login(osoToken string) (int, []*http.Cookie, error)
	State() (clients.PodState, error)
	Start() (state clients.PodState, code int, err error)
}

// Jenkins implements Jenkins interface
type Jenkins struct {
	info CacheItem

	idler  clients.IdlerService
	tenant clients.TenantService

	logger *log.Entry
}

// GetJenkins returns an intance of Jenkins struct
func GetJenkins(clusters map[string]string,
	pci *CacheItem,
	idler clients.IdlerService,
	tenant clients.TenantService,
	tokenData string,
	logger *log.Entry) (*Jenkins, string, error) {

	if pci != nil {
		return &Jenkins{
			info:   *pci,
			idler:  idler,
			tenant: tenant,
			logger: logger,
		}, "", nil
	}

	tokenJSON := &auth.TokenJSON{}
	err := json.Unmarshal([]byte(tokenData), tokenJSON)
	if err != nil {
		return &Jenkins{}, "", err
	}

	authClient, err := auth.DefaultClient()
	if err != nil {
		return &Jenkins{}, "", err
	}
	uid, err := authClient.UIDFromToken(tokenJSON.AccessToken)
	if err != nil {
		return &Jenkins{}, "", err
	}

	ti, err := tenant.GetTenantInfo(uid)
	if err != nil {
		return &Jenkins{}, "", err
	}
	osioToken := tokenJSON.AccessToken

	namespace, err := clients.GetNamespaceByType(ti, ServiceName)
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
		tenant: tenant,
		logger: logger,
	}, osioToken, nil
}

//Login to Jenkins with OSO token to get cookies
func (j *Jenkins) Login(osoToken string) (int, []*http.Cookie, error) {

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
func (j *Jenkins) State() (clients.PodState, error) {
	return j.idler.State(j.info.NS, j.info.ClusterURL)
}

// Start unidles Jenkins only if it is idled and returns the
// state of the pod, the http status of calling unidle, and error if any
func (j *Jenkins) Start() (state clients.PodState, code int, err error) {
	// Assume pods are starting and unidle only if it is in "idled" state
	code = http.StatusAccepted
	ns := j.info.NS
	clusterURL := j.info.ClusterURL
	//nsLogger := log.WithFields(log.Fields{"ns": ns, "cluster": clusterURL})

	state, err = j.idler.State(ns, clusterURL)
	if err != nil {
		return
	}
	j.logger.Infof("state : %q", state)

	if state == clients.Idled {
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

// constructRoute returns Jenkins route based on a specific pattern
func constructRoute(clusters map[string]string, clusterURL string, ns string) (string, string, error) {
	appSuffix := clusters[clusterURL]
	if len(appSuffix) == 0 {
		return "", "", fmt.Errorf("could not find entry for cluster %s", clusterURL)
	}
	route := fmt.Sprintf("jenkins-%s.%s", ns, clusters[clusterURL])
	return route, "https", nil
}
