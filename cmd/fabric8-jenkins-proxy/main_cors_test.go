package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type MockJenkinsAPI interface {
	Start(w http.ResponseWriter, r *http.Request, _ httprouter.Params)
}

type MockJenkinsAPIImpl struct{}

func (api *MockJenkinsAPIImpl) Start(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := clients.StatusResponse{}
	json.NewEncoder(w).Encode(resp)
}

func TestAPIServerCORSHeaders(t *testing.T) {
	apiServer := newJenkinsAPIServer(&MockJenkinsAPIImpl{})
	reader, _ := http.NewRequest("POST", "/jenkins/start", nil)
	// Check for origin "https://*.openshift.io"
	randomOrigin := uuid.NewV4().String()
	reader.Header.Set("origin", "https://"+randomOrigin+".openshift.io")
	writer := httptest.NewRecorder()
	apiServer.Handler.ServeHTTP(writer, reader)
	assert.Equal(t, "https://"+randomOrigin+".openshift.io", writer.Header().Get("access-control-allow-origin"))

	// Check for origin "https://openshift.io"
	reader.Header.Set("origin", "https://.openshift.io")
	writer = httptest.NewRecorder()
	apiServer.Handler.ServeHTTP(writer, reader)
	assert.Equal(t, "https://.openshift.io", writer.Header().Get("access-control-allow-origin"))
}

func TestProxyCORSHeaders(t *testing.T) {
	apiServer := newProxyServer(&proxy.Proxy{})
	reader, _ := http.NewRequest("GET", "/", nil)
	randomOrigin := uuid.NewV4().String()
	reader.Header.Set("origin", randomOrigin) // allowing everything for now.

	writer := httptest.NewRecorder()
	apiServer.Handler.ServeHTTP(writer, reader)

	assert.Equal(t, "", writer.Header().Get("access-control-allow-origin"))

}
