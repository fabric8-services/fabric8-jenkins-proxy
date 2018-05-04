package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util/logging"
	log "github.com/sirupsen/logrus"
)

const (
	namespaceSuffix = "-jenkins"

	// OpenShiftAPIParam is the parameter name under which the OpenShift cluster API URL is passed using
	// Idle, UnIdle and IsIdle.
	OpenShiftAPIParam = "openshift_api_url"
)

// status represent status of the service true if idle, false if unidle.
type status struct {
	IsIdle bool `json:"is_idle"`
}

// clusterView is a view of the cluster topology which only includes the OpenShift API URL and the application DNS for this
// cluster.
type clusterView struct {
	APIURL string
	AppDNS string
}

// IdlerService provides methods to talk to the idler client
type IdlerService interface {
	IsIdle(tenant string, openShiftAPIURL string) (bool, error)
	UnIdle(tenant string, openShiftAPIURL string) (int, error)
	Clusters() (map[string]string, error)
}

// idler is a hand-rolled Idler client using plain HTTP requests.
type idler struct {
	idlerAPI string
}

// NewIdler returns an instance of idler client on taking URL of idler service as an input.
func NewIdler(url string) IdlerService {
	return &idler{
		idlerAPI: url,
	}
}

// IsIdle returns true if the Jenkins instance for the specified tenant is idled, false otherwise.
func (i *idler) IsIdle(tenant string, openShiftAPIURL string) (bool, error) {
	namespace := tenant
	if !strings.HasSuffix(tenant, namespaceSuffix) {
		namespace = tenant + namespaceSuffix
		log.WithField("ns", tenant).Debugf("Adding namespace suffix - resulting namespace: %s", namespace)
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/idler/isidle/%s", i.idlerAPI, namespace), nil)
	if err != nil {
		return false, err
	}

	q := req.URL.Query()
	q.Add(OpenShiftAPIParam, util.EnsureSuffix(openShiftAPIURL, "/"))
	req.URL.RawQuery = q.Encode()

	log.WithFields(log.Fields{"request": logging.FormatHTTPRequestWithSeparator(req, " "), "type": "isidle"}).Debug("Calling Idler API")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	s := &status{}
	err = json.Unmarshal(body, s)
	if err != nil {
		return false, err
	}

	log.Debugf("Jenkins is idle (%t) in %s", s.IsIdle, namespace)

	return s.IsIdle, nil
}

// UnIdle initiates un-idling of the Jenkins instance for the specified tenant.
func (i *idler) UnIdle(tenant string, openShiftAPIURL string) (int, error) {
	namespace := tenant
	if !strings.HasSuffix(tenant, namespaceSuffix) {
		namespace = tenant + namespaceSuffix
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/idler/unidle/%s", i.idlerAPI, namespace), nil)
	if err != nil {
		return 0, err
	}

	q := req.URL.Query()
	q.Add(OpenShiftAPIParam, util.EnsureSuffix(openShiftAPIURL, "/"))
	req.URL.RawQuery = q.Encode()

	log.WithFields(log.Fields{"request": logging.FormatHTTPRequestWithSeparator(req, " "), "type": "unidle"}).Debug("Calling Idler API")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// return err only for unexpected responses from idler
	if resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusServiceUnavailable {
		return resp.StatusCode, nil
	}

	return 0, fmt.Errorf("unexpected status code '%d' as response to unidle call", resp.StatusCode)
}

// Clusters returns a map which maps the OpenShift API URL to the application DNS for this cluster. An empty map together with
// an error is returned if an error occurs.
func (i *idler) Clusters() (map[string]string, error) {
	var clusters = make(map[string]string)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/idler/cluster", i.idlerAPI), nil)
	if err != nil {
		return clusters, err
	}

	log.WithFields(log.Fields{"request": logging.FormatHTTPRequestWithSeparator(req, " "), "type": "cluster"}).Debug("Calling Idler API")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return clusters, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return clusters, err
	}

	var clusterViews []clusterView
	err = json.Unmarshal(body, &clusterViews)
	if err != nil {
		return clusters, err
	}

	for _, clusterView := range clusterViews {
		clusters[clusterView.APIURL] = clusterView.AppDNS
	}

	return clusters, nil
}
