package clients_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
)

func TestGetTenant(t *testing.T) {
	ts := MockServer(TenantData1())
	defer ts.Close()

	ct := clients.NewTenant(ts.URL, "aaa")
	ti, err := ct.GetTenantInfo("2e15e957-0366-4802-bf1e-0d6fe3f11bb6")
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%+v", ti)

	n := ct.GetNamespaceByType(ti, "jenkins")
	if !strings.HasSuffix(n.Name, "-jenkins") {
		t.Error("Could not find Jenkins namespace - ", n.Name)
	}

}