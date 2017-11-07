package api

import (
	"encoding/json"
	"net/http"
	"github.com/julienschmidt/httprouter"
	proxy "github.com/fabric8-services/fabric8-jenkins-proxy/proxy"
)

type ProxyAPI struct {
	Proxy *proxy.Proxy
}

func NewAPI(prx *proxy.Proxy) ProxyAPI {
	return ProxyAPI{
		Proxy: prx,
	}
}

type APIResponse struct {
	Namespace string `json:"namespace"`
	Requests int `json:"requests"`
	LastVisit string `json:"last_visit"`
	LastRequest string `json:"last_request"`
}

func (api *ProxyAPI) Info(w http.ResponseWriter, r *http.Request,  ps httprouter.Params) {
	l, t := api.Proxy.GetBufferInfo(ps.ByName("namespace"))
	resp := APIResponse{
		Namespace: ps.ByName("namespace"),
		Requests: l,
		LastRequest: t,
		LastVisit: api.Proxy.GetLastVisitString(ps.ByName("namespace")),
	}

	json.NewEncoder(w).Encode(resp)
}