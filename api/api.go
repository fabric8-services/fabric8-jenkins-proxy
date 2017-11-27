package api

import (
	"github.com/fabric8-services/fabric8-jenkins-proxy/storage"
	"encoding/json"
	"net/http"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

type ProxyAPI struct {
	storageService *storage.DBService
}

func NewAPI(storageService *storage.DBService) ProxyAPI {
	return ProxyAPI{
		storageService: storageService,
	}
}

type APIResponse struct {
	Namespace string `json:"namespace"`
	Requests int `json:"requests"`
	LastVisit int64 `json:"last_visit"`
	LastRequest int64 `json:"last_request"`
}

func (api *ProxyAPI) Info(w http.ResponseWriter, r *http.Request,  ps httprouter.Params) {
	ns := ps.ByName("namespace")
	s, err := api.storageService.GetStatisticsUser(ns)
	if err != nil {
		log.Error(err) //FIXME
		return
	}
	c, err := api.storageService.GetRequestsCount(ns)
	if err != nil {
		log.Error(err) //FIXME
		return
	}
	resp := APIResponse{
		Namespace: ns,
		Requests: c,
		LastRequest: s.LastBufferedRequest,
		LastVisit: s.LastAccessed,
	}

	json.NewEncoder(w).Encode(resp)
}