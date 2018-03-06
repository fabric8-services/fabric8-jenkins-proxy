package router

import (
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/julienschmidt/httprouter"
)

func CreateAPIRouter(api api.ProxyAPI) *httprouter.Router {
	// Create router for API
	proxyRouter := httprouter.New()
	proxyRouter.GET("/api/info/:namespace", api.Info)
	return proxyRouter
}
