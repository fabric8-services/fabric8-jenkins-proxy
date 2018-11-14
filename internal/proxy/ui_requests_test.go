package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"

	uuid "github.com/satori/go.uuid"
	gock "gopkg.in/h2non/gock.v1"

	"github.com/stretchr/testify/assert"
)

const (
	testTokenJSON = "%7B%22access_token%22%3A%22test_access_token%22%2C%22expires_in%22%3A2592000%2C%22not-before-policy%22%3Anull%2C%22refresh_expires_in%22%3A2592000%2C%22refresh_token%22%3A%22test_refresh_token%22%2C%22token_type%22%3A%22bearer%22%7D"
)

func TestBadToken(t *testing.T) {
	p := NewMockProxy("")
	req := httptest.NewRequest("GET", "http://proxy?token_json=BADTOKEN", nil)
	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNoTokenRedirectstoAuthService(t *testing.T) {

	p := NewMockProxy("")

	req := httptest.NewRequest("GET", "http://proxy", nil)
	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, w.Header().Get("Location"), "http://authURL/api/login?redirect=http:%2F%2Fredirect")
}

func TestCorrectTokenWithJenkinsIdled(t *testing.T) {

	p := NewMockProxy(clients.Idled)

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
	for _, a := range setCookieHeaders {
		fmt.Println(a)
	}
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
		assert.True(t, ok, "item object is of type cache item")
		assert.Equal(t, info, cacheItem)
	}
}

func TestWithTokenJenkinsRunningButLoginFailed(t *testing.T) {
	defer gock.Off()

	gock.New("https://jenkins-namespace-jenkins.test_route/securityRealm/commenceLogin?from=%2F").
		Get("").
		Reply(401)

	p := NewMockProxy(clients.Running)

	req := httptest.NewRequest("GET", "http://proxy?token_json="+testTokenJSON, nil)

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, w.Header().Get("Location"), "http://redirect")
}

func TestWithTokenJenkinsRunningAndLoginSuccessful(t *testing.T) {
	defer gock.Off()

	gock.New("https://jenkins-namespace-jenkins.test_route/securityRealm/commenceLogin?from=%2F").
		Get("").
		Reply(200).
		SetHeaders(map[string]string{
			"Set-Cookie": "JSESSIONID." + uuid.NewV4().String() + "=" + uuid.NewV4().String(),
		})

	p := NewMockProxy(clients.Running)

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
		assert.True(t, ok, "item object is of type cache item")
		assert.Equal(t, info, cacheItem)
	}
}

func TestExpireCookieIfNotInCache(t *testing.T) {
	p := NewMockProxy("")

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

func TestIdledCookiePresentWhenJenkinsIdled(t *testing.T) {
	p := NewMockProxy(clients.Idled)
	cookieVal := uuid.NewV4().String()
	cacheItem := CacheItem{
		ClusterURL: "Valid_OpenShift_API_URL",
		NS:         "namespace-jenkins",
		Scheme:     "https",
		Route:      "jenkins-namespace-jenkins.test_route",
	}
	p.ProxyCache.SetDefault(cookieVal, cacheItem)
	req := httptest.NewRequest("GET", "http://proxy", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JenkinsIdled",
		Value: cookieVal,
	})

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestIdledCookiePresentWhenJenkinsRunning(t *testing.T) {
	defer gock.Off()

	gock.New("https://jenkins-namespace-jenkins.test_route/securityRealm/commenceLogin?from=%2F").
		Get("").
		Reply(403)

	p := NewMockProxy(clients.Running)
	cookieVal := uuid.NewV4().String()
	cacheItem := CacheItem{
		ClusterURL: "Valid_OpenShift_API_URL",
		NS:         "namespace-jenkins",
		Scheme:     "https",
		Route:      "jenkins-namespace-jenkins.test_route",
	}
	p.ProxyCache.SetDefault(cookieVal, cacheItem)
	req := httptest.NewRequest("GET", "http://proxy", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JenkinsIdled",
		Value: cookieVal,
	})

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusOK, w.Code) // ?
	setCookieHeaders := w.Header()["Set-Cookie"]
	assert.Equal(t, 1, len(setCookieHeaders))

	assert.Contains(t, setCookieHeaders[0], "JenkinsIdled")
	assert.Contains(t, setCookieHeaders[0], "Expires")
	c, ok := p.ProxyCache.Get(cookieVal)
	assert.False(t, ok, "key not found")
	assert.Nil(t, c)

}

func TestSessionCookiePresentWhenJenkinsIdled(t *testing.T) {
	p := NewMockProxy(clients.Idled)
	cookieVal := uuid.NewV4().String()
	cacheItem := CacheItem{
		ClusterURL: "Valid_OpenShift_API_URL",
		NS:         "namespace-jenkins",
		Scheme:     "https",
		Route:      "jenkins-namespace-jenkins.test_route",
	}
	p.ProxyCache.SetDefault(cookieVal, cacheItem)
	req := httptest.NewRequest("GET", "http://proxy", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID." + uuid.NewV4().String(),
		Value: cookieVal,
	})

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	setCookieHeaders := w.Header()["Set-Cookie"]
	assert.Equal(t, 2, len(setCookieHeaders)) // Should be changed to one once we fix multiple cookie expiration

	assert.Contains(t, setCookieHeaders[0], "JSESSIONID")
	assert.Contains(t, setCookieHeaders[0], "Expires")

	c, ok := p.ProxyCache.Get(cookieVal)
	assert.False(t, ok, "key not found")
	assert.Nil(t, c)
}

func TestSessionCookiePresentWhenJenkinsRunning(t *testing.T) {
	p := NewMockProxy(clients.Running)
	cookieVal := uuid.NewV4().String()
	cacheItem := CacheItem{
		ClusterURL: "Valid_OpenShift_API_URL",
		NS:         "namespace-jenkins",
		Scheme:     "https",
		Route:      "jenkins-namespace-jenkins.test_route",
	}
	p.ProxyCache.SetDefault(cookieVal, cacheItem)
	req := httptest.NewRequest("GET", "http://proxy", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID." + uuid.NewV4().String(),
		Value: cookieVal,
	})

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	// Cause we have not set the code here, it should show 200
	assert.Equal(t, http.StatusOK, w.Code)

	setCookieHeaders, ok := w.Header()["Set-Cookie"]
	assert.False(t, ok, "No Set-Cookie header found")
	assert.Nil(t, setCookieHeaders)
}
