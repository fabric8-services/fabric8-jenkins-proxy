package router

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
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

	var routes = []struct {
		route  string
		target string
	}{
		{"/api/info/:namespace", "Info"},
	}

	for _, testRoute := range routes {
		w := new(mockResponseWriter)

		req, _ := http.NewRequest("GET", testRoute.route, nil)
		mockedRouter.ServeHTTP(w, req)

		assert.Equal(t, testRoute.target, w.GetBody(), fmt.Sprintf("Routing failed for %s", testRoute.route))
	}
}
