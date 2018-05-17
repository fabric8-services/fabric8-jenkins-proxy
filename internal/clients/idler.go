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

type podState string

const (
	// UnknownState used when the state isn't known, along with err
	UnknownState podState = ""
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

// Jenkins pod state response structs
// ErrorCode is an integer that clients to can use to compare errors
type errorCode uint32

type responseError struct {
	Code        errorCode `json:"code"`
	Description string    `json:"description"`
}

type jenkinsInfo struct {
	State podState `json:"state"`
}

type statusResponse struct {
	Data   *jenkinsInfo    `json:"data,omitempty"`
	Errors []responseError `json:"errors,omitempty"`
}

// IdlerService provides methods to talk to the idler client
type IdlerService interface {
	UnIdle(tenant string, openShiftAPIURL string) (int, error)
	State(tenant string, openShiftAPIURL string) (podState, error)
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
func (i *idler) State(tenant string, openShiftAPIURL string) (podState, error) {
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

	sr := &statusResponse{}
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

func (e responseError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Description)
}

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
	}
}
