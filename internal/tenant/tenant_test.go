package tenant

import (
	"strings"
	"testing"

	tu "github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils"
)

func TestGetTenant(t *testing.T) {
	ts := tu.MockServer(tu.TenantData1(""))
	defer ts.Close()

	ct := New(ts.URL, "aaa")
	ti, err := ct.GetTenantInfo("2e15e957-0366-4802-bf1e-0d6fe3f11bb6")
	if err != nil {
		t.Error(err)
	}

	n, err := GetNamespaceByType(ti, "jenkins")
	if err != nil {
		t.Error(err)
	}
	if !strings.HasSuffix(n.Name, "-jenkins") {
		t.Error("Could not find Jenkins namespace - ", n.Name)
	}

}

func TestGetError(t *testing.T) {
	ts := tu.MockServer(tu.TenantData2())
	defer ts.Close()

	ct := New(ts.URL, "aaa")
	ti, err := ct.GetTenantInfo("2e15e957-0366-4802-bf1e-0d6fe3f11bb6")
	if err == nil {
		t.Error("Expected Errors to be populated in output")
	}

	if len(ti.Errors) != 1 {
		t.Error(ti.Errors)
	}
}
