package router

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/jenkinsapi"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"
)

type mockResponseWriter struct {
	buffer bytes.Buffer
}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	m.buffer.Write(p)
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	m.buffer.WriteString(s)
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

func (m *mockResponseWriter) GetBody() string {
	return m.buffer.String()
}

type mockProxyAPI struct {
	storageService storage.Store
}

func (i *mockProxyAPI) Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("Info"))
	w.WriteHeader(http.StatusOK)
}

func (i *mockProxyAPI) Clear(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

type mockJenkinsAPI struct{}

// Start mock returns the Jenkins status for the current user
func (api *mockJenkinsAPI) Start(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := clients.StatusResponse{}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		jenkinsapi.HandleError(w, resp, errors.New("Could not find Bearer Token in Authorization Header"), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	resp.Data = &clients.JenkinsInfo{
		State: clients.UnknownState,
	}
	json.NewEncoder(w).Encode(resp)
}

// Status mock returns the Jenkins status for current user
func (api *mockJenkinsAPI) Status(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := clients.StatusResponse{}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		jenkinsapi.HandleError(w, resp, errors.New("Could not find Bearer Token in Authorization Header"), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	resp.Data = &clients.JenkinsInfo{
		State: clients.UnknownState,
	}
	json.NewEncoder(w).Encode(resp)
}

func Test_API_routes_are_setup(t *testing.T) {
	mockedProxyAPI := &mockProxyAPI{}
	mockedRouter := CreateAPIRouter(mockedProxyAPI)
	req, _ := http.NewRequest("GET", "/api/info/:namespace", nil)
	w := new(mockResponseWriter)
	mockedRouter.ServeHTTP(w, req)
	require.Equal(t, "Info", w.GetBody(), "Routing failed for /api/info/:namespace")

	req, _ = http.NewRequest("GET", "/metrics", nil)
	w = new(mockResponseWriter)
	mockedRouter.ServeHTTP(w, req)
	require.Contains(t, w.GetBody(), "go_gc_duration_seconds", "Routing failed for /metrics")
}

func Test_JenkinsAPI_routes_are_setup(t *testing.T) {
	mockedJenkinsAPI := &mockJenkinsAPI{}
	mockedRouter := CreateJenkinsAPIRouter(mockedJenkinsAPI)
	req, _ := http.NewRequest("POST", "/api/jenkins/start", nil)
	req.Header.Add("Authorization", "Bearer Doesn't Matter")
	w := new(mockResponseWriter)
	mockedRouter.ServeHTTP(w, req)
	require.Equal(t, "{\"data\":{\"state\":\"\"}}\n", w.GetBody(), "Routing failed for /api/jenkins/start")
}
