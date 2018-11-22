package proxy

import (
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/tenant"
	cache "github.com/patrickmn/go-cache"
)

// NewMock returns an instance of proxy object that uses mocked dependency services
func NewMock(jenkinsState idler.PodState) *Proxy {

	if jenkinsState == "" {
		jenkinsState = idler.Idled
	}
	auth.SetDefaultClient(auth.NewMockAuth("http://authURL"))

	return &Proxy{
		tenant: &tenant.Mock{},
		idler:  idler.NewMock("", jenkinsState, false),
		clusters: map[string]string{
			"Valid_OpenShift_API_URL": "test_route",
		},
		ProxyCache: cache.New(15*time.Minute, 10*time.Minute),
		redirect:   "http://redirect",
	}
}
