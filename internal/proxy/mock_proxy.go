package proxy

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils/mock"
	cache "github.com/patrickmn/go-cache"
)

// NewMockProxy returns an instance of proxy object that uses mocked dependency services
func NewMockProxy(jenkinsState clients.PodState) *Proxy {

	if jenkinsState == "" {
		jenkinsState = clients.Idled
	}
	auth.SetDefaultClient(auth.NewMockAuth("http://authURL"))

	_, b, _, _ := runtime.Caller(0)

	return &Proxy{
		tenant: &mock.Tenant{},
		idler:  mock.NewMockIdler("", jenkinsState, false),
		wit:    &clients.MockWit{},
		clusters: map[string]string{
			"Valid_OpenShift_API_URL": "test_route",
		},
		ProxyCache:     cache.New(15*time.Minute, 10*time.Minute),
		TenantCache:    cache.New(30*time.Minute, 40*time.Minute),
		redirect:       "http://redirect",
		indexPath:      strings.TrimSuffix(filepath.Dir(b), "/internal/proxy") + "/static/html/index.html",
		storageService: &storage.MockStore{},
		visitLock:      &sync.Mutex{},
	}
}
