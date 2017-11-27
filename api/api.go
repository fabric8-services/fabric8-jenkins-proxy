package api

import (
	"time"
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
	LastVisit time.Time `json:"last_visit"`
	LastRequest time.Time `json:"last_request"`
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
		LastRequest: time.Unix(s.LastBufferedRequest, 0),
		LastVisit: time.Unix(s.LastAccessed, 0),
	}

	json.NewEncoder(w).Encode(resp)
}