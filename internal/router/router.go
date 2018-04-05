package router

import (
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
)

// CreateAPIRouter is creating a router for the REST API of the Proxy.
func CreateAPIRouter(api api.ProxyAPI) *httprouter.Router {
	// Create router for API
	proxyRouter := httprouter.New()
	proxyRouter.GET("/api/info/:namespace", api.Info)
	proxyRouter.Handler("GET", "/metrics", prometheus.Handler())
	return proxyRouter
}
