package tenant

import (
	"errors"
)

// Mock is a mock client for fabric8-tenant.
type Mock struct {
}

// GetTenantInfo returns a tenant information based on tenant id.
func (t Mock) GetTenantInfo(tenantID string) (ti Info, err error) {
	return Info{
		Data: InfoData{
			Attributes: Attributes{
				Namespaces: []Namespace{
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
func (t Mock) GetNamespace(accessToken string) (namespace Namespace, err error) {
	if accessToken == "ValidToken" {
		return Namespace{
			ClusterURL: "Valid_OpenShift_API_URL",
			Name:       "namespace-jenkins",
			State:      "",
			Type:       "jenkins",
		}, nil

	}

	return Namespace{}, errors.New("Invalid Token")
}
