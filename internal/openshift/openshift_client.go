package openshift

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

var logger = log.WithFields(log.Fields{"component": "openshift-client"})

// OpenShift is a client for OpenShift API
type Client interface {
	GetRoute(n string, s string) (r string, tls bool, err error)
}

// OpenShift is a client for OpenShift API
type OpenShift struct {
	token  string
	apiURL string
	client *http.Client
}

// NewOpenShift creates new OpenShift client with new http client.
func NewClient(apiURL string, token string) Client {
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(10) * time.Second,
	}

	return NewClientWithHttp(c, apiURL, token)
}

// NewOpenShiftWithClient create new OpenShift client with given HTTP client.
func NewClientWithHttp(client *http.Client, apiURL string, token string) Client {
	if !strings.HasPrefix(apiURL, "http") {
		apiURL = fmt.Sprintf("https://%s", strings.TrimRight(apiURL, "/"))
	}
	return &OpenShift{
		apiURL: apiURL,
		token:  token,
		client: client,
	}
}

// GetRoute returns the OpenShift route for a given namespace and service and whether the route uses TLS.
func (o *OpenShift) GetRoute(n string, s string) (string, bool, error) {
	req, err := o.reqOAPI("GET", n, fmt.Sprintf("routes/%s", s), nil)
	if err != nil {
		return "", false, err
	}
	resp, err := o.do(req)
	if err != nil {
		return "", false, err
	}

	type route struct {
		Spec struct {
			Host string
			TLS  struct {
				Termination string
			} `json:"tls"`
		}
	}
	defer bodyClose(resp)

	rt := route{}
	err = json.NewDecoder(resp.Body).Decode(&rt)
	if err != nil {
		return "", false, err
	}

	tls := len(rt.Spec.TLS.Termination) > 0
	return rt.Spec.Host, tls, nil
}

// req constructs a HTTP request for OpenShift/Kubernetes API.
func (o *OpenShift) req(method string, oapi bool, namespace string, command string, body io.Reader, watch bool) (req *http.Request, err error) {
	api := "api"
	if oapi {
		api = "oapi"
	}

	url := fmt.Sprintf("%s/%s/v1", o.apiURL, api)
	if len(namespace) > 0 {
		url = fmt.Sprintf("%s/%s/%s", url, "namespaces", namespace)
	}

	url = fmt.Sprintf("%s/%s", url, command)

	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	req.Header.Add("Authorization", "Bearer "+o.token)
	if watch {
		v := req.URL.Query()
		v.Add("watch", "true")
		req.URL.RawQuery = v.Encode()
	}

	return
}

// reqOAPI is a helper to construct a request for OpenShift API
func (o *OpenShift) reqOAPI(method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(method, true, namespace, command, body, false)
}

// do uses client.Do function to perform request and return response
func (o *OpenShift) do(req *http.Request) (resp *http.Response, err error) {
	requestDump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		logger.Errorf("Unable to dump HTTP request: %s", err.Error())
	} else {
		logger.WithField("request", string(requestDump[:])).Debug("OpenShift API request.")
	}

	resp, err = o.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("got status %s (%d) from %s", resp.Status, resp.StatusCode, req.URL)
	}

	return
}

func bodyClose(resp *http.Response) {
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}
