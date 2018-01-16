package api

import (
	"encoding/json"
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
)

//ProxyAPI is an API to serve user statistics
type ProxyAPI struct {
	storageService *storage.DBService
}

func NewAPI(storageService *storage.DBService) ProxyAPI {
	return ProxyAPI{
		storageService: storageService,
	}
}

type APIResponse struct {
	Namespace   string `json:"namespace"`
	Requests    int    `json:"requests"`
	LastVisit   int64  `json:"last_visit"`
	LastRequest int64  `json:"last_request"`
}

//Info returns JSON including information about Proxy usage statistics for a given namespace
func (api *ProxyAPI) Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ns := ps.ByName("namespace")
	s, notFound, err := api.storageService.GetStatisticsUser(ns)
	if err != nil {
		if notFound {
			log.Debugf("Did not find data for %s", ns)
		} else {
			log.Error(err) //FIXME
			fmt.Fprintf(w, "{'error': '%s'}", err)
			return
		}
	}
	c, err := api.storageService.GetRequestsCount(ns)
	if err != nil {
		log.Error(err) //FIXME
		fmt.Fprintf(w, "{'error': '%s'}", err)
		return
	}
	resp := APIResponse{
		Namespace:   ns,
		Requests:    c,
		LastRequest: s.LastBufferedRequest,
		LastVisit:   s.LastAccessed,
	}

	json.NewEncoder(w).Encode(resp)
}
