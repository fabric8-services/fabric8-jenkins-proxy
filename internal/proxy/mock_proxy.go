package proxy

import (
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils/mock"
	cache "github.com/patrickmn/go-cache"
)

// NewMockProxy returns an instance of proxy object that uses mocked dependency services
func NewMockProxy(jenkinsState clients.PodState) *Proxy {

	if jenkinsState == "" {
		jenkinsState = clients.Idled
	}
	auth.SetDefaultClient(auth.NewMockAuth("http://authURL"))

	return &Proxy{
		tenant: &mock.Tenant{},
		idler:  mock.NewMockIdler("", jenkinsState, false),
		clusters: map[string]string{
			"Valid_OpenShift_API_URL": "test_route",
		},
		ProxyCache: cache.New(15*time.Minute, 10*time.Minute),
		redirect:   "http://redirect",
	}
}
