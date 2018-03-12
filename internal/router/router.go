package router

import (
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/julienschmidt/httprouter"
)

// CreateAPIRouter is creating a router for the REST API of the Proxy.
func CreateAPIRouter(api api.ProxyAPI) *httprouter.Router {
	// Create router for API
	proxyRouter := httprouter.New()
	proxyRouter.GET("/api/info/:namespace", api.Info)
	return proxyRouter
}
