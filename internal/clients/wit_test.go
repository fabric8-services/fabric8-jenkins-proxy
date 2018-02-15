package clients

import (
	"testing"

	tu "github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils"
)

func TestWIT(t *testing.T) {
	ts := tu.MockServer(tu.WITData1())

	defer ts.Close()

	wit := NewWIT(ts.URL, "xxx")
	wi, err := wit.SearchCodebase("github.com/vpavlin/vpavlin-prod-prev-test.git")
	if err != nil {
		t.Error(err)
	}

	if wi.OwnedBy != "2e15e957-0366-4802-bf1e-0d6fe3f11bb6" {
		t.Error("Could not find tenant id: ", wi.OwnedBy)
	}

}
