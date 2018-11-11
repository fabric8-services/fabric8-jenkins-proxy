package mock

import (
	"errors"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
)

// Tenant is a mock client for fabric8-tenant.
type Tenant struct {
}

// GetTenantInfo returns a tenant information based on tenant id.
func (t Tenant) GetTenantInfo(tenantID string) (ti clients.TenantInfo, err error) {
	return clients.TenantInfo{
		Data: clients.TenantInfoData{
			Attributes: clients.Attributes{
				Namespaces: []clients.Namespace{
					{
						ClusterURL: "Valid_OpenShift_API_URL",
						Type:       "jenkins",
						Name:       "namespace-jenkins",
					},
				},
			},
			ID: "",
		},
	}, nil
}

// GetNamespace mock gets namespace
func (t Tenant) GetNamespace(accessToken string) (namespace clients.Namespace, err error) {
	if accessToken == "ValidToken" {
		return clients.Namespace{
			ClusterURL: "Valid_OpenShift_API_URL",
			Name:       "namespace-jenkins",
			State:      "",
			Type:       "jenkins",
		}, nil

	}

	return clients.Namespace{}, errors.New("Invalid Token")
}
