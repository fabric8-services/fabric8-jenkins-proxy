package router

import (
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/jenkinsapi"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/goadesign/goa"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// CustomMuxHandle is adding routes for services like prometheus
func CustomMuxHandle(service *goa.Service) {
	mux := func() goa.MuxHandler {
		return func(rw http.ResponseWriter, req *http.Request, v url.Values) {
			promhttp.Handler().ServeHTTP(rw, req)
		}
	}()
	service.Mux.Handle("GET", "/metrics", mux)
}

// CreateJenkinsAPIRouter is creating a router for the REST API of the Proxy.
func CreateJenkinsAPIRouter(jenkinsAPI jenkinsapi.JenkinsAPI) *httprouter.Router {
	// Create router for API
	jenkinsAPIRouter := httprouter.New()
	jenkinsAPIRouter.POST("/api/jenkins/start", jenkinsAPI.Start)
	jenkinsAPIRouter.GET("/api/jenkins/status", jenkinsAPI.Status)
	return jenkinsAPIRouter
}

// CreateProxyRouter is the HTTP server handler which handles the incoming webhook and UI requests.
func CreateProxyRouter(proxy *proxy.Proxy) *http.ServeMux {
	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/", proxy.Handle)

	return proxyMux
}
