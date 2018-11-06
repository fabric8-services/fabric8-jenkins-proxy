package reverseproxy

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy/cookies"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils/mock"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestReverseProxy(t *testing.T) {
	const backendResponse = "I am the backend"
	const backendStatus = 404
	retries := 0
	jenkins := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// simulate connection termination by closing the connection
		// if the "mode" is "hangup"
		if r.Method == "GET" && r.FormValue("mode") == "hangup" {
			if retries < 5 {
				// NOTE Hijack results in calling the handlerfunc again for the
				// first time so for the first time retry will be incremented to 2
				retries++
				c, _, _ := w.(http.Hijacker).Hijack()
				// must not have any affect since the connection is going to be closed
				defer c.Close()
			} else {
				w.WriteHeader(200)
				w.Write([]byte("all good"))
			}
			return
		}

		assert.Len(t, r.TransferEncoding, 0, "backend got unexpected TransferEncoding")
		assert.NotEmpty(t, r.Header.Get("X-Forwarded-For"), "didn't get X-Forwarded-For header")
		assert.Empty(t, r.Header.Get("Connection"), "handler got Connection header value")
		assert.Empty(t, r.Header.Get("Upgrade"), "handler got Upgrade header value")
		assert.Empty(t, r.Header.Get("Proxy-Connection"), "handler got Proxy-Connection header value")

		w.Header().Set("Trailers", "not a special header field name")
		w.Header().Set("Trailer", "X-Trailer")
		w.Header().Set("X-Foo", "bar")
		w.Header().Set("Upgrade", "foo")
		w.Header().Add("X-Multi-Value", "foo")
		w.Header().Add("X-Multi-Value", "bar")
		http.SetCookie(w, &http.Cookie{Name: "flavor", Value: "chocolateChip"})
		w.WriteHeader(backendStatus)
		w.Write([]byte(backendResponse))
		w.Header().Set("X-Trailer", "trailer_value")
		w.Header().Set(http.TrailerPrefix+"X-Unannounced-Trailer", "unannounced_trailer_value")
	}))
	defer jenkins.Close()

	jenkinsURL, err := url.Parse(jenkins.URL)
	assert.NoError(t, err, "url parse error")

	// this fakes the jenkins proxy that which sets the request URL
	fakeJenkinsProxy := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestURL := *r.URL

			r.Host = jenkinsURL.Host
			r.URL.Host = jenkinsURL.Host
			r.URL.Scheme = jenkinsURL.Scheme

			// logger := log.WithField("test_mode", "true")
			logger, _ := test.NewNullLogger()
			//
			proxyHandler := &ReverseProxy{
				RedirectURL:     requestURL,
				ResponseTimeout: 3 * time.Second,
				Logger:          logger.WithField("mode", "testing"),
			}
			proxyHandler.ServeHTTP(w, r)
		}))
	defer fakeJenkinsProxy.Close()

	frontendClient := fakeJenkinsProxy.Client()
	getReq, _ := http.NewRequest("GET", fakeJenkinsProxy.URL+"?foo=bar", nil)
	getReq.Header.Set("Connection", "close")
	getReq.Header.Set("Proxy-Connection", "should be deleted")
	getReq.Header.Set("Upgrade", "foo")
	getReq.Close = true
	res, err := frontendClient.Do(getReq)
	assert.NoError(t, err, "creation of frontend failed")
	assert.Equal(t, res.Header.Get("X-Foo"), "bar", "got unexpected X-Foo")

	assert.Equal(t,
		res.Header.Get("Trailers"), "not a special header field name",
		"got unexpected header Trailers",
	)
	assert.Len(t, res.Header["X-Multi-Value"], 2, "unexpected X-Multi-Value header values")
	assert.Len(t, res.Header["Set-Cookie"], 1, "got unexpected Set-Cookie")

	if g, e := res.Trailer, (http.Header{"X-Trailer": nil}); !reflect.DeepEqual(g, e) {
		t.Errorf("before reading body, Trailer = %#v; want %#v", g, e)
	}

	assert.Equal(t, res.Cookies()[0].Name, "flavor", "Cookie setting failed")

	bodyBytes, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, backendResponse, string(bodyBytes), "unexpected response body")

	assert.Equal(t, res.Trailer.Get("X-Trailer"), "trailer_value", "Trailer(X-Trailer)")
	assert.Equal(t, res.Trailer.Get("X-Unannounced-Trailer"),
		"unannounced_trailer_value", "Trailer(X-Unannounced-Trailer)")

	// Test that a backend failing to be reached or one which doesn't return
	// a response results in a StatusBadGateway.
	noRedirectClient := fakeJenkinsProxy.Client()

	noRedirectClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // do not follow redirect
	}

	getReq, _ = http.NewRequest("GET", fakeJenkinsProxy.URL+"/?mode=hangup", nil)
	getReq.Close = true
	assert.Equal(t, retries, 0)
	res, err = noRedirectClient.Do(getReq)
	assert.NoError(t, err, "")
	assert.Equal(t, res.StatusCode, http.StatusFound)
	assert.Equal(t, retries, 2)

	// retry and should still get 302 redirect
	res, err = noRedirectClient.Do(getReq)
	assert.Equal(t, res.StatusCode, http.StatusFound)
	assert.Equal(t, retries, 3)

	// retry and should still get 302 redirect
	res, err = noRedirectClient.Do(getReq)
	res, err = noRedirectClient.Do(getReq)
	assert.Equal(t, retries, 5) // magic number to make the call pass

	res, err = noRedirectClient.Do(getReq)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	body, err := ioutil.ReadAll(res.Body)
	assert.Equal(t, string(body), "all good")

}

func TestServeHTTP(t *testing.T) {

	config := mock.NewConfig()

	rp := NewReverseProxy(
		url.URL{
			Scheme: "https",
			Host:   "jenkinsHost",
			Path:   "/path",
		},
		config.GetGatewayTimeout(),
		false,

		log.WithFields(log.Fields{"component": "reverseproxy"}),
	)

	// On recieving status BadGateway, GatewayTimeout, ServiceUnavailable
	// or Forbidden from jenkins, reverse proxy should redirect that request to redirect
	// url, thus writing status to Found (includes all)
	testProxyStatus(t, rp, http.StatusBadGateway, http.StatusFound)
	testProxyStatus(t, rp, http.StatusGatewayTimeout, http.StatusFound)
	testProxyStatus(t, rp, http.StatusServiceUnavailable, http.StatusFound)
	testProxyStatus(t, rp, http.StatusForbidden, http.StatusFound)

	// On receiving any other status from jenkins, reverse proxy should also write the
	// same status (does not include all cases)
	testProxyStatus(t, rp, http.StatusOK, http.StatusOK)
	testProxyStatus(t, rp, http.StatusAccepted, http.StatusAccepted)
	testProxyStatus(t, rp, http.StatusUnauthorized, http.StatusUnauthorized)
	testProxyStatus(t, rp, http.StatusFound, http.StatusFound)

	// Reverse proxy checks the session validity on recieving status Forbidden
	// from jenkins
	testSessionCheckWithoutCookie(t, rp)
	testSessionCheckWithCookie(t, rp, cookiesutil.SessionCookie+uuid.NewV4().String())
	testSessionCheckWithCookie(t, rp, "DoesntstartWithJSESSIONID")
}

func testProxyStatus(t *testing.T, rp *ReverseProxy, jenkinsStatusCode int, proxyStatusCode int) {
	defer gock.Off()

	gock.New("https://jenkinsHost").
		Get("/path").
		Reply(jenkinsStatusCode)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "https://jenkinsHost/path", nil)

	rp.ServeHTTP(rr, req)
	assert.Equal(t, proxyStatusCode, rr.Code)
}

func testSessionCheckWithoutCookie(t *testing.T, rp *ReverseProxy) {
	defer gock.Off()

	gock.New("https://jenkinsHost").
		Get("/path").
		Reply(403)

	rr1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "https://jenkinsHost/path", nil)

	rp.IsValidSession = true
	assert.True(t, rp.IsValidSession)
	// On finding Forbidden from jenkins, ServeHTTP would check for session
	// by looking for session cookie. If it finds no session it means invalid
	// If invalid it would set rep.IsValidSession to false
	rp.ServeHTTP(rr1, req1)
	assert.False(t, rp.IsValidSession)
}

func testSessionCheckWithCookie(t *testing.T, rp *ReverseProxy, cookieName string) {
	defer gock.Off()

	gock.New("https://jenkinsHost").
		Get("/path").
		Reply(403)

	gock.New("https://jenkinsHost").
		Get("").
		Reply(200)

	rr1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "https://jenkinsHost/path", nil)

	cookie := http.Cookie{
		Name:    cookieName,
		Value:   uuid.NewV4().String(),
		Expires: time.Now().Local().Add(time.Hour),
	}
	req1.AddCookie(&cookie)

	if cookiesutil.IsSessionCookie(&cookie) {
		rp.IsValidSession = false
		assert.False(t, rp.IsValidSession)
		rp.ServeHTTP(rr1, req1)
		assert.True(t, rp.IsValidSession)
	} else {
		rp.IsValidSession = true
		assert.True(t, rp.IsValidSession)
		rp.ServeHTTP(rr1, req1)
		assert.False(t, rp.IsValidSession)
	}

}
