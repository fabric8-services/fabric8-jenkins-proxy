package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

//Idler is a simple client for Idler
type Idler struct {
	idlerApi string
}

func NewIdler(url string) Idler {
	return Idler{
		idlerApi: url,
	}
}

type Status struct {
	IsIdle bool `json:"is_idle"`
}

//IsIdle returns true if Jenkins is idled for a given namespace
func (i Idler) IsIdle(namespace string) (bool, error) {
	resp, err := http.Get(fmt.Sprintf("%s/iapi/idler/isidle/%s", i.idlerApi, namespace))
	if err != nil {
		return true, err
	}
	defer resp.Body.Close()

	// This is a temporary workaround for multi-cluster. ATM, the Idler is only aware of a single OpenShift cluster.
	// If a IsIdle request is made for a namespace in a different cluster, the Idler will return 404.
	// For now we don't treat this as an error and just return false, assuming that Idling is only working on
	// a single cluster for now. See https://github.com/fabric8-services/fabric8-jenkins-proxy/issues/150
	// and https://github.com/fabric8-services/fabric8-jenkins-proxy/issues/151
	if resp.StatusCode == 404 {
		return false, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	s := &Status{}
	err = json.Unmarshal(body, s)
	if err != nil {
		return false, err
	}

	log.Debugf("Jenkins is idle (%t) in %s", s.IsIdle, namespace)

	return s.IsIdle, nil
}
