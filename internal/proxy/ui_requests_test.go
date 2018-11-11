package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	cache "github.com/patrickmn/go-cache"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils/mock"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/stretchr/testify/assert"
)

const (
	testTokenJSON = "%7B%22access_token%22%3A%22test_access_token%22%2C%22expires_in%22%3A2592000%2C%22not-before-policy%22%3Anull%2C%22refresh_expires_in%22%3A2592000%2C%22refresh_token%22%3A%22test_refresh_token%22%2C%22token_type%22%3A%22bearer%22%7D"
)

func TestBadToken(t *testing.T) {
	p := Proxy{}
	p.redirect = "http://redirect"
	req := httptest.NewRequest("GET", "http://proxy/v1?token_json=BADTOKEN", nil)
	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNoTokenRedirectstoAuthService(t *testing.T) {

	p := Proxy{}
	auth.SetDefaultClient(auth.NewClient("http://authURL"))
	p.redirect = "http://redirect"

	req := httptest.NewRequest("GET", "http://proxy/v1", nil)
	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, w.Header().Get("Location"), "http://authURL/api/login?redirect=http:%2F%2Fredirect%2Fv1")
}

func TestCorrectTokenWithJenkinsIdled(t *testing.T) {
	m := make(map[string]string)
	m["Valid_OpenShift_API_URL"] = "test_route"
	p := Proxy{
		tenant:     &mock.Tenant{},
		idler:      mock.NewMockIdler("", clients.Idled, false),
		clusters:   m,
		ProxyCache: cache.New(15*time.Minute, 10*time.Minute),
	}
	p.redirect = "http://redirect"

	auth.SetDefaultClient(auth.NewMockAuth("http://authURL"))

	req := httptest.NewRequest("GET", "http://proxy/v1?token_json="+testTokenJSON, nil)
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID." + uuid.NewV4().String(),
		Value: uuid.NewV4().String(),
	})

	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, http.StatusFound, w.Code)
	setCookieHeaders := w.Header()["Set-Cookie"]

	assert.Equal(t, 2, len(setCookieHeaders))
	for _, a := range setCookieHeaders {
		fmt.Println(a)
	}
	assert.Contains(t, setCookieHeaders[0], "JSESSIONID")
	assert.Contains(t, setCookieHeaders[1], "JenkinsIdled=")
}
