package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type MockProxyAPI interface {
	Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

type MockProxyAPIImpl struct{}

func (m MockProxyAPIImpl) Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	resp := api.Response{}
	json.NewEncoder(w).Encode(resp)
}

func TestAPIServerCORSHeaders(t *testing.T) {
	apiServer := newAPIServer(MockProxyAPIImpl{})
	reader, _ := http.NewRequest("GET", "/", nil)
	randomOrigin := uuid.NewV4().String()
	reader.Header.Set("origin", randomOrigin) // allowing everything for now.

	writer := httptest.NewRecorder()
	apiServer.Handler.ServeHTTP(writer, reader)

	assert.Equal(t, randomOrigin, writer.Header().Get("access-control-allow-origin"))

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
