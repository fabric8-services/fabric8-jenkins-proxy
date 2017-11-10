package clients_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	tu "github.com/fabric8-services/fabric8-jenkins-proxy/testutils"
)


func TestIdler(t *testing.T) {
	ts := tu.MockServer(tu.IdlerData1())
	defer ts.Close()

	il := clients.NewIdler(ts.URL)
	s, r, err := il.GetRoute("vpavlin-jenkins")
	if err != nil {
		t.Error(err)
	}

	if r != "jenkins-vpavlin-jenkins.d800.free-int.openshiftapps.com" {
		t.Error("Did not get correct route: ", r)
	}

	if s != "https" {
		t.Error("Did not get correct scheme: ", s)
	}
}
