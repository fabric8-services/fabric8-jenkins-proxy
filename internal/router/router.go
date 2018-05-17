package router

import (
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/jenkinsapi"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// CreateAPIRouter is creating a router for the REST API of the Proxy.
func CreateAPIRouter(api api.ProxyAPI) *httprouter.Router {
	// Create router for API
	proxyRouter := httprouter.New()
	proxyRouter.GET("/api/info/:namespace", api.Info)
	proxyRouter.Handler("GET", "/metrics", promhttp.Handler())
	return proxyRouter
}

// CreateJenkinsAPIRouter is creating a router for the REST API of the Proxy.
func CreateJenkinsAPIRouter(jenkinsAPI jenkinsapi.JenkinsAPI) *httprouter.Router {
	// Create router for API
	jenkinsAPIRouter := httprouter.New()
	jenkinsAPIRouter.GET("/api/jenkins/start", jenkinsAPI.Start)
	return jenkinsAPIRouter
}

// CreateProxyRouter is the HTTP server handler which handles the incoming webhook and UI requests.
func CreateProxyRouter(proxy *proxy.Proxy) *http.ServeMux {
	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/", proxy.Handle)

	return proxyMux
}
