package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"

	"github.com/julienschmidt/httprouter"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type MockJenkinsAPIImpl struct{}

func (api *MockJenkinsAPIImpl) Start(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := idler.StatusResponse{}
	json.NewEncoder(w).Encode(resp)
}

func (api *MockJenkinsAPIImpl) Status(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := idler.StatusResponse{}
	json.NewEncoder(w).Encode(resp)
}

func TestAPIServerCORSHeaders(t *testing.T) {
	config := configuration.NewMock()
	apiServer := newJenkinsAPIServer(&MockJenkinsAPIImpl{}, &config)

	reader, _ := http.NewRequest("POST", "/doesntmatter", nil)

	randomOrigin := uuid.NewV4().String()

	corsTests := []struct {
		name     string
		given    string
		expected string
	}{
		{name: "sub domain prefix", given: "https://" + randomOrigin + ".openshift.io", expected: "https://" + randomOrigin + ".openshift.io"},
		{name: "only domain", given: "https://openshift.io", expected: "https://openshift.io"},
		{name: "domain suffix", given: "https://localhost:" + randomOrigin, expected: "https://localhost:" + randomOrigin},
		{name: "domain suffix non SSL", given: "http://localhost:" + randomOrigin, expected: "http://localhost:" + randomOrigin},
		{name: "block random domain", given: "https://" + randomOrigin + ".io", expected: ""},
	}

	for _, test := range corsTests {
		t.Run(test.name, func(t *testing.T) {
			assertCorsHeaders(reader, test.given, test.expected, apiServer, t)
		})
	}
}

func assertCorsHeaders(r *http.Request, given string, expected string, apiServer *http.Server, t *testing.T) {
	// GIVEN
	r.Header.Set("Origin", given)

	// WHEN
	writer := httptest.NewRecorder()
	apiServer.Handler.ServeHTTP(writer, r)

	// THEN
	got := writer.Header().Get("access-control-allow-origin")
	assert.Equal(t, expected, got)
}
