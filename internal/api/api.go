package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

//ProxyAPI is an API to serve user statistics
type ProxyAPI interface {
	Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
	Clear(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

type proxy struct {
	storageService storage.Store
	tenant         clients.TenantService
}

// NewAPI creates an instance of ProxyAPI on taking a storage/database service as input.
func NewAPI(storageService storage.Store, tenant clients.TenantService) ProxyAPI {
	return &proxy{
		storageService: storageService,
		tenant:         tenant,
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

// Clear deletes statistics and requests for a given namespace
func (api *proxy) Clear(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Get the namespace from authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		err := errors.New("Could not find Bearer token in Authorization Header")
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	accessToken := strings.Split(authHeader, " ")[1]
	namespace, err := api.tenant.GetNamespace(accessToken)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Infof("Found a valid token info in the authorization header. Namespace is %s", namespace.Name)

	// TODO: Check if this there are statistics and requests with this namespace

	err = api.storageService.DeleteStatisticsUser(namespace.Name)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = api.storageService.DeleteRequestsUser(namespace.Name)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
