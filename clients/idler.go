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

//GetRoute returns sheme and route for a given namespace
func (i Idler) GetRoute(n string) (scheme string, rt string, err error) {
	resp, err := http.Get(fmt.Sprintf("%s/iapi/idler/route/%s", i.idlerApi, n))
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	type route struct {
		Service string
		Route   string
		TLS     bool
	}
	r := route{}

	err = json.Unmarshal(body, &r)
	if err != nil {
		return
	}

	if r.TLS {
		scheme = "https"
	} else {
		scheme = "http"
	}

	rt = r.Route
	fmt.Printf(rt)

	return
}
