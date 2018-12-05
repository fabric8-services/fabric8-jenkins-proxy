package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy/reverseproxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/wit"
	log "github.com/sirupsen/logrus"

	uuid "github.com/satori/go.uuid"
	gock "gopkg.in/h2non/gock.v1"

	"github.com/stretchr/testify/assert"
)

const (
	testTokenJSON = "%7B%22access_token%22%3A%22test_access_token%22%2C%22expires_in%22%3A2592000%2C%22not-before-policy%22%3Anull%2C%22refresh_expires_in%22%3A2592000%2C%22refresh_token%22%3A%22test_refresh_token%22%2C%22token_type%22%3A%22bearer%22%7D"
)

func TestBadToken(t *testing.T) {
	p := NewMock("", wit.DefaultMockOwner)
	req := httptest.NewRequest("GET", "http://proxy?token_json=BADTOKEN", nil)
	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNoTokenRedirectstoAuthService(t *testing.T) {

	p := NewMock("", wit.DefaultMockOwner)

	req := httptest.NewRequest("GET", "http://proxy", nil)
	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, w.Header().Get("Location"), "http://authURL/api/login?redirect=http:%2F%2Fredirect")
}

func TestCorrectTokenWithJenkinsIdled(t *testing.T) {

	p := NewMock(idler.Idled, wit.DefaultMockOwner)

	req := httptest.NewRequest("GET", "http://proxy?token_json="+testTokenJSON, nil)
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID." + uuid.NewV4().String(),
		Value: uuid.NewV4().String(),
	})

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, w.Header().Get("Location"), "http://redirect")

	setCookieHeaders := w.Header()["Set-Cookie"]
	assert.Equal(t, 2, len(setCookieHeaders))

	assert.Contains(t, setCookieHeaders[0], "JSESSIONID")
	assert.Contains(t, setCookieHeaders[0], "Expires")
	assert.Contains(t, setCookieHeaders[1], "JenkinsIdled=")
	assert.NotContains(t, setCookieHeaders[1], "Expires")

	// A proxy cacheitem is added to proxycache
	items := p.ProxyCache.Items()
	assert.Equal(t, len(items), 1)
	for _, item := range items {
		info, ok := item.Object.(CacheItem)
		cacheItem := CacheItem{
			ClusterURL: "Valid_OpenShift_API_URL",
			NS:         "namespace-jenkins",
			Scheme:     "https",
			Route:      "jenkins-namespace-jenkins.test_route",
		}
		assert.True(t, ok, "item object should be of type cache item")
		assert.Equal(t, info, cacheItem)
	}
}

func TestWithTokenJenkinsRunningButLoginFailed(t *testing.T) {
	gock.New("https://jenkins-namespace-jenkins.test_route/securityRealm/commenceLogin?from=%2F").
		Get("").
		Reply(401)

	p := NewMock(idler.Running, wit.DefaultMockOwner)

	req := httptest.NewRequest("GET", "http://proxy?token_json="+testTokenJSON, nil)

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, w.Header().Get("Location"), "http://redirect")
}

func TestWithTokenJenkinsRunningAndLoginSuccessful(t *testing.T) {
	gock.New("https://jenkins-namespace-jenkins.test_route/securityRealm/commenceLogin?from=%2F").
		Get("").
		Reply(200).
		SetHeaders(map[string]string{
			"Set-Cookie": "JSESSIONID." + uuid.NewV4().String() + "=" + uuid.NewV4().String(),
		})

	p := NewMock(idler.Running, wit.DefaultMockOwner)

	req := httptest.NewRequest("GET", "http://proxy?token_json="+testTokenJSON, nil)
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID." + uuid.NewV4().String(),
		Value: uuid.NewV4().String(),
	})
	req.AddCookie(&http.Cookie{
		Name:  "JenkinsIdled",
		Value: uuid.NewV4().String(),
	})

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, w.Header().Get("Location"), "http://redirect")

	setCookieHeaders := w.Header()["Set-Cookie"]
	assert.Equal(t, 3, len(setCookieHeaders))

	assert.Contains(t, setCookieHeaders[0], "JSESSIONID")
	assert.Contains(t, setCookieHeaders[0], "Expires")
	assert.Contains(t, setCookieHeaders[1], "JenkinsIdled=")
	assert.Contains(t, setCookieHeaders[1], "Expires")
	assert.Contains(t, setCookieHeaders[2], "JSESSIONID")
	assert.NotContains(t, setCookieHeaders[2], "Expires")

	items := p.ProxyCache.Items()
	assert.Equal(t, len(items), 1)
	for _, item := range items {
		info, ok := item.Object.(CacheItem)
		cacheItem := CacheItem{
			ClusterURL: "Valid_OpenShift_API_URL",
			NS:         "namespace-jenkins",
			Scheme:     "https",
			Route:      "jenkins-namespace-jenkins.test_route",
		}
		assert.True(t, ok, "item object should be of type cache item")
		assert.Equal(t, info, cacheItem)
	}
}

func TestExpireCookieIfNotInCache(t *testing.T) {
	p := NewMock("", wit.DefaultMockOwner)

	req := httptest.NewRequest("GET", "http://proxy", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID." + uuid.NewV4().String(),
		Value: uuid.NewV4().String(),
	})

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	setCookieHeaders := w.Header()["Set-Cookie"]
	assert.Equal(t, 2, len(setCookieHeaders))

	assert.Contains(t, setCookieHeaders[0], "JSESSIONID")
	assert.Contains(t, setCookieHeaders[0], "Expires")
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

	config := configuration.NewMock()

	p := NewMock(idler.Running, wit.DefaultMockOwner)
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
		p.OnErrorUIRequest,

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
