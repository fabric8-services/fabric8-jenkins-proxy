package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

//ProxyAPI is an API to serve user statistics
type ProxyAPI interface {
	Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

type proxy struct {
	storageService storage.Store
}

// NewAPI creates an instance of ProxyAPI on taking a storage/database service as input.
func NewAPI(storageService storage.Store) ProxyAPI {
	return &proxy{
		storageService: storageService,
	}
}

// Response contains Proxy usage statistics.
type Response struct {
	Namespace   string `json:"namespace"`
	Requests    int    `json:"requests"`
	LastVisit   int64  `json:"last_visit"`
	LastRequest int64  `json:"last_request"`
}

// Info returns JSON including information about Proxy usage statistics for a given namespace.
func (api *proxy) Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	resp := Response{
		Namespace:   ns,
		Requests:    c,
		LastRequest: s.LastBufferedRequest,
		LastVisit:   s.LastAccessed,
	}

	json.NewEncoder(w).Encode(resp)
}
