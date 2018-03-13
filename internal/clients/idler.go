package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util"
)

const (
	namespaceSuffix = "-jenkins"

	// OpenShiftAPIParam is the parameter name under which the OpenShift cluster API URL is passed using
	// Idle, UnIdle and IsIdle.
	OpenShiftAPIParam = "openshift_api_url"
)

type status struct {
	IsIdle bool `json:"is_idle"`
}

type IdlerService interface {
	IsIdle(tenant string, openShiftAPIURL string) (bool, error)
	UnIdle(tenant string, openShiftAPIURL string) error
}

//Idler is a simple client for Idler
type idler struct {
	idlerApi string
}

func NewIdler(url string) IdlerService {
	return &idler{
		idlerApi: url,
	}
}

// IsIdle returns true if the Jenkins instance for the specified tenant is idled. False otherwise.
func (i idler) IsIdle(tenant string, openShiftAPIURL string) (bool, error) {
	namespace := tenant
	if !strings.HasSuffix(tenant, namespaceSuffix) {
		namespace = tenant + namespaceSuffix
		log.WithField("ns", tenant).Debugf("Adding namespace suffix - resulting namespace: %s", namespace)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/idler/isidle/%s", i.idlerApi, namespace), nil)
	if err != nil {
		return false, err
	}

	q := req.URL.Query()
	q.Add(OpenShiftAPIParam, util.EnsureSuffix(openShiftAPIURL, "/"))
	req.URL.RawQuery = q.Encode()

	log.WithField("request", req).Debug("Calling Idler API")

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

// Initiates un-idling of the Jenkins instance for the specified tenant.
func (i idler) UnIdle(tenant string, openShiftAPIURL string) error {
	namespace := tenant
	if !strings.HasSuffix(tenant, namespaceSuffix) {
		namespace = tenant + namespaceSuffix
	}
	resp, err := http.Get(fmt.Sprintf("%s/api/idler/unidle/%s", i.idlerApi, namespace))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	} else {
		return errors.New(fmt.Sprintf("unexpected status code '%d' as response to unidle call.", resp.StatusCode))
	}
}
