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

// Start returns the Jenkins status for the current user
func (api *jenkinsAPIImpl) Start(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := clients.StatusResponse{}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		HandleError(w, resp, errors.New("Could not find Bearer Token in Authorization Header"), http.StatusUnauthorized)
		return
	}

	accessToken := strings.Split(r.Header.Get("Authorization"), " ")[1]
	namespace, err := api.tenant.GetNamespace(accessToken)
	if err != nil {
		HandleError(w, resp, err, http.StatusUnauthorized)
		return
	}
	log.Infof("Found token info in the query. Namespace is %s and clusterURL is %s", namespace.Name, namespace.ClusterURL)

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

// HandleError logs the error and encodes it in the reponse
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
