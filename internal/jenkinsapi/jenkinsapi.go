package jenkinsapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	log "github.com/sirupsen/logrus"
)

//JenkinsAPI contains API to check whether Jenkins for the current user is idle|running|starting
type JenkinsAPI interface {
	Start(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
	Status(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

// JenkinsAPIImpl implements JenkinsAPI
type jenkinsAPIImpl struct {
	tenant clients.TenantService
	idler  clients.IdlerService
}

// NewJenkinsAPI creates a new instance of JenkinsAPI
func NewJenkinsAPI(tenant clients.TenantService, idler clients.IdlerService) JenkinsAPI {
	return &jenkinsAPIImpl{
		tenant: tenant,
		idler:  idler,
	}
}

// Returns namespace from request headers
func lookupNamespace(r *http.Request, t clients.TenantService) (clients.Namespace, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		err := errors.New("Could not find Bearer token in Authorization Header")
		return clients.Namespace{}, err
	}
	accessToken := strings.Split(authHeader, " ")[1]
	namespace, err := t.GetNamespace(accessToken)
	log.Infof("Found token info in the query. Namespace is %s and clusterURL is %s", namespace.Name, namespace.ClusterURL)
	return namespace, err
}

// Returns the Jenkins pods status for current user
func (api *jenkinsAPIImpl) Status(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := clients.StatusResponse{}

	namespace, err := lookupNamespace(r, api.tenant)
	if err != nil {
		HandleError(w, resp, err, http.StatusUnauthorized)
		return
	}
	status, err := api.idler.State(namespace.Name, namespace.ClusterURL)
	if err != nil {
		HandleError(w, resp, err, http.StatusInternalServerError)
		return
	}
	resp.Data = &clients.JenkinsInfo{
		State: status,
	}
	json.NewEncoder(w).Encode(resp)
}

// Start returns the Jenkins status for the current user
func (api *jenkinsAPIImpl) Start(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := clients.StatusResponse{}

	namespace, err := lookupNamespace(r, api.tenant)
	if err != nil {
		HandleError(w, resp, err, http.StatusUnauthorized)
		return
	}

	status, err := api.idler.State(namespace.Name, namespace.ClusterURL)
	if err != nil {
		HandleError(w, resp, err, http.StatusInternalServerError)
		return
	}
	resp.Data = &clients.JenkinsInfo{
		State: status,
	}

	if status != clients.Running {
		httpCode, err := api.idler.UnIdle(namespace.Name, namespace.ClusterURL)
		if err != nil {
			HandleError(w, resp, err, httpCode)
			return
		}
		w.WriteHeader(httpCode)
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleError logs the error and encodes it in the response
func HandleError(w http.ResponseWriter, resp clients.StatusResponse, err error, httpCode int) {
	log.Error(err)
	w.WriteHeader(httpCode)
	respErr := clients.ResponseError{
		Code:        clients.ErrorCode(httpCode),
		Description: err.Error(),
	}
	resp.Errors = append(resp.Errors, respErr)
	json.NewEncoder(w).Encode(resp)
}
