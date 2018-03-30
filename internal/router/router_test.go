package router

import (
	"bytes"
	"net/http"
	"testing"

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

func Test_all_routes_are_setup(t *testing.T) {
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
