package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

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

	return s.IsIdle, nil
}

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
		Route string
		TLS bool
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

	return
}