package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

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

// PodState represents status of the pod which could be running|starting|idled|
type PodState string

const (
	// UnknownState used when the state isn't known, along with err
	UnknownState PodState = ""
	// Running represents a running pod
	Running = "running"
	// Starting represents pod that is starting
	Starting = "starting"
	// Idled represents pod that is idled
	Idled = "idled"
)

// clusterView is a view of the cluster topology which only includes the OpenShift API URL and the application DNS for this
// cluster.
type clusterView struct {
	APIURL string
	AppDNS string
}

// ErrorCode is an integer that clients to can use to compare errors
type ErrorCode uint32

// ResponseError contains and err description along with http status code
type ResponseError struct {
	Code        ErrorCode `json:"code"`
	Description string    `json:"description"`
}

// JenkinsInfo contains Jenkins pod state
type JenkinsInfo struct {
	State PodState `json:"state"`
}

// StatusResponse provides jenkins status and errors that occurred while making the status API call to idler
type StatusResponse struct {
	Data   *JenkinsInfo    `json:"data,omitempty"`
	Errors []ResponseError `json:"errors,omitempty"`
}

// IdlerService provides methods to talk to the idler client
type IdlerService interface {
	UnIdle(tenant string, openShiftAPIURL string) (int, error)
	State(tenant string, openShiftAPIURL string) (PodState, error)
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

// State returns the state of Jenkins instance for the specified tenant
func (i *idler) State(tenant string, openShiftAPIURL string) (PodState, error) {
	namespace := tenant
	if !strings.HasSuffix(tenant, namespaceSuffix) {
		namespace = tenant + namespaceSuffix
		log.WithField("ns", tenant).Debugf("Adding namespace suffix - resulting namespace: %s", namespace)
	}
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/idler/status/%s", i.idlerAPI, namespace), nil)
	if err != nil {
		return UnknownState, err
	}

	q := req.URL.Query()
	q.Add(OpenShiftAPIParam, util.EnsureSuffix(openShiftAPIURL, "/"))
	req.URL.RawQuery = q.Encode()

	logger := log.WithFields(log.Fields{
		"request": logging.FormatHTTPRequestWithSeparator(req, " "),
		"type":    "state",
	})

	logger.Debug("Calling Idler API")

	client := httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return UnknownState, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return UnknownState, err
	}

	sr := &StatusResponse{}
	err = json.Unmarshal(body, sr)
	if err != nil {
		return UnknownState, err
	}

	if len(sr.Errors) != 0 {
		return UnknownState, sr.Errors[0]

	}

	state := sr.Data.State
	logger.Debugf("Jenkins pod on %q is in %q state", namespace, state)

	return state, nil
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

	client := httpClient()
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

func (e ResponseError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Description)
}

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
	}
}
