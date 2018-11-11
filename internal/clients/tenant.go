package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"errors"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util"
	log "github.com/sirupsen/logrus"
)

// NewTenant returns new Tenant client.
func NewTenant(tenantServiceURL string, authToken string) Tenant {
	logger := log.WithFields(
		log.Fields{
			"component": "tenant",
			"url":       tenantServiceURL,
		},
	)

	return Tenant{
		authToken:        authToken,
		tenantServiceURL: tenantServiceURL,
		client:           util.HTTPClient(),
		logger:           logger,
	}
}

// TenantService is contains methods that makes calls to tenant APIs
type TenantService interface {
	GetTenantInfo(tenantID string) (TenantInfo, error)
	GetNamespace(accessToken string) (Namespace, error)
}

// Tenant is a simple client for fabric8-tenant.
type Tenant struct {
	tenantServiceURL string
	authToken        string
	client           *http.Client
	logger           *log.Entry
}

// TenantInfoList is a list of tenant information.
type TenantInfoList struct {
	Data []TenantInfoData
	Meta struct {
		TotalCount int
	}
	Errors []Error `json:"errors"`
}

// TenantInfo gives imformation about tenant.
type TenantInfo struct {
	Data   TenantInfoData
	Errors []Error `json:"errors"`
}

// Error describes an HTTP error consisting of error code and its details.
type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// TenantInfoData give data about information such as attributes, id and type.
type TenantInfoData struct {
	Attributes Attributes
	ID         string
	Type       string
}

// Attributes consists of time when the tenant was created, email and list namespaces belonging to that tenant.
type Attributes struct {
	CreatedAt  time.Time `json:"created-at"`
	Email      string
	Namespaces []Namespace
}

// Namespace is a tenant space in which each username is unique.
type Namespace struct {
	ClusterURL string `json:"cluster-url"`
	Name       string
	State      string
	Type       string
}

// GetTenantInfo returns a tenant information based on tenant id.
func (t Tenant) GetTenantInfo(tenantID string) (ti TenantInfo, err error) {
	if len(tenantID) == 0 {
		err = errors.New("tenant ID cannot be empty string")
		return
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/tenants/%s", t.tenantServiceURL, tenantID), nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.authToken))

	t.logger.WithFields(log.Fields{
		"type": "id",
		"id":   tenantID,
	}).Info("Tenant by id")

	resp, err := t.client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &ti)
	if err != nil {
		return
	}

	if len(ti.Errors) > 0 {
		err = fmt.Errorf("%+v", ti.Errors)
	}

	return
}

// GetNamespaceByType searches tenant namespaces for a given type.
func GetNamespaceByType(ti TenantInfo, typ string) (r Namespace, err error) {
	for i := 0; i < len(ti.Data.Attributes.Namespaces); i++ {
		n := ti.Data.Attributes.Namespaces[i]
		if n.Type == typ {
			r = n
			return
		}
	}

	err = fmt.Errorf("could not find tenant %s Jenkins namespace", ti.Data.Attributes.Email)
	return
}

// GetNamespace gets namespace given appropriate accessToken
func (t Tenant) GetNamespace(accessToken string) (namespace Namespace, err error) {
	authClient, err := auth.DefaultClient()
	if err != nil {
		return Namespace{}, err
	}
	uid, err := authClient.UIDFromToken(accessToken)
	if err != nil {
		return Namespace{}, err
	}

	ti, err := t.GetTenantInfo(uid)
	if err != nil {
		return Namespace{}, err
	}

	namespace, err = GetNamespaceByType(ti, "jenkins")
	if err != nil {
		return Namespace{}, err
	}

	return namespace, nil
}
