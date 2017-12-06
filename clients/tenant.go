package clients

import (
	"errors"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
	"net/http"
)

func NewTenant(tenantServiceURL string, authToken string) Tenant {
	return Tenant{
		authToken: authToken,
		tenantServiceURL: tenantServiceURL,
	}
}

type TenantInfoList struct {
	Data []TenantInfoData
	Meta struct{
		TotalCount int
	}
	Errors []Error `json:"errors"`
}
type TenantInfo struct {
	Data TenantInfoData
	Errors []Error `json:"errors"`
}

type Error struct {
	Code string `json:"code"`
	Detail string `json:"detail"`
}

type TenantInfoData struct {
	Attributes Attributes
	Id string
	Type string
}

type Attributes struct {
	CreatedAt time.Time `json:"created-at"`
	Email string
	Namespaces []Namespace
}

type Namespace struct {
	ClusterURL string `json:"cluster-url"`
	Name string
	State string
	Type string
}

//Tenant is a simple client for fabric8-tenant
type Tenant struct {
	tenantServiceURL string
	authToken string
}

//GetTenantInfo returns a tenant information based on tenant id
func (t Tenant) GetTenantInfo(tenantId string) (ti TenantInfo, err error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/tenants/%s", t.tenantServiceURL, tenantId), nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.authToken))

	client := &http.Client{}
	resp, err := client.Do(req)
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
		err = errors.New(fmt.Sprintf("%+v",ti.Errors))
	}

	return
}

//GetNamespaceByType searches tenant namespaces for a given type
func (t Tenant) GetNamespaceByType(ti TenantInfo, typ string) (r *Namespace, err error) {
	for i:=0;i<len(ti.Data.Attributes.Namespaces);i++ {
		n := ti.Data.Attributes.Namespaces[i]
		if n.Type == typ {
			r = &n
			return
		}
	}

	err = errors.New(fmt.Sprintf("Could not find tenant %s Jenkins namespace.", ti.Data.Attributes.Email))
	return
}

func (t Tenant) GetTenantInfoByNamespace(api string, ns string) (ti TenantInfoList, err error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/tenants", t.tenantServiceURL), nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.authToken))

	q := req.URL.Query()
	q.Add("master_url", api)
	q.Add("namespace", ns)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &ti)

	return
}