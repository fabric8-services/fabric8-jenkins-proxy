package proxy

import (
	"sync"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/wit"

	cache "github.com/patrickmn/go-cache"
)

// NewMock returns an instance of proxy object that uses mocked dependency services
func NewMock(jenkinsState idler.PodState, ownedBy string) *Proxy {

	if jenkinsState == "" {
		jenkinsState = idler.Idled
	}
	auth.SetDefaultClient(auth.NewMockAuth("http://authURL"))

	return &Proxy{
		tenant: &tenant.Mock{},
		idler:  idler.NewMock("", jenkinsState, false),
		wit: &wit.Mock{
			OwnedBy: ownedBy,
		},
		clusters: map[string]string{
			"Valid_OpenShift_API_URL": "test_route",
		},
		ProxyCache:     cache.New(15*time.Minute, 10*time.Minute),
		TenantCache:    cache.New(30*time.Minute, 40*time.Minute),
		redirect:       "http://redirect",
		storageService: &storage.Mock{},
		visitLock:      &sync.Mutex{},
	}
}
