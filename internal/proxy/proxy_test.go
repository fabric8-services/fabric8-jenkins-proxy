package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"errors"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy/reverseproxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils/mock"
	cache "github.com/patrickmn/go-cache"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	gock "gopkg.in/h2non/gock.v1"
)

type mockWit struct {
	testCounter int // we will use this to count how many times this we get there
}

func (mw *mockWit) SearchCodebase(repo string) (*clients.WITInfo, error) {
	mw.testCounter++
	return &clients.WITInfo{
		OwnedBy: "Faker",
	}, errors.New("Reterror")
}

type fakeProxy struct {
	Proxy
}

func TestGetUserWithRetry(t *testing.T) {
	wit := &mockWit{testCounter: 0}

	p := &fakeProxy{}
	p.wit = wit
	numberofretry := 1

	logEntry := log.WithFields(log.Fields{"component": "proxy"})
	cache := cache.New(2*time.Millisecond, 1*time.Millisecond)
	p.TenantCache = cache

	_, err := p.getUserWithRetry("http://test", logEntry, numberofretry)
	assert.Error(t, err, "Faker")
	assert.Equal(t, numberofretry, wit.testCounter)
}

func TestValidateSession(t *testing.T) {
	// Reverse proxy checks the session validity on recieving status Forbidden
	// from jenkins
	defer gock.Off()

	gock.New("https://jenkinsHost").
		Get("/path").
		Reply(http.StatusForbidden)

	gock.New("https://jenkinsHost").
		Get("").
		Reply(http.StatusForbidden)

	config := mock.NewConfig()

	p := NewMockProxy(clients.Running)
	cookieVal := uuid.NewV4().String()
	info := CacheItem{
		ClusterURL: "Valid_OpenShift_API_URL",
		NS:         "namespace-jenkins",
		Scheme:     "https",
		Route:      "https://jenkinsHost",
	}
	p.ProxyCache.SetDefault(cookieVal, info)

	req := httptest.NewRequest("GET", "https://jenkinsHost/path", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID." + uuid.NewV4().String(),
		Value: cookieVal,
	})

	w := httptest.NewRecorder()

	rp := reverseproxy.NewReverseProxy(
		//*req.URL,
		url.URL{
			Scheme: "https",
			Host:   "proxy",
			Path:   "/path",
		},
		config.GetGatewayTimeout(),
		p.OnError,

		log.WithFields(log.Fields{"component": "testing"}),
	)

	// Session cookie exists and there a cache item in proxy cache
	// with key as the cookie value, but session has expired from jenkins
	// side. Proxy doesn't know about this just yet.
	// Jenkins would return status forbidden on this, we expire cookie
	// and delete the cache

	_, ok := p.ProxyCache.Get(cookieVal)
	assert.True(t, ok, "CacheItem exists")
	rp.ServeHTTP(w, req)
	_, ok = p.ProxyCache.Get(cookieVal)
	assert.False(t, ok, "CacheItem has been deleted")

	setCookieHeaders := w.Header()["Set-Cookie"]
	assert.Equal(t, 1, len(setCookieHeaders))
	assert.Contains(t, setCookieHeaders[0], "JSESSIONID")
	assert.Contains(t, setCookieHeaders[0], "Expires")

	// Redirects to the redirect url
	assert.Equal(t, w.Header().Get("Location"), "https://proxy/path")
}
