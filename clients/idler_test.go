package clients

import (
	"testing"
)


func TestIdler(t *testing.T) {
	ts := MockServer(IdlerData1())
	defer ts.Close()

	il := NewIdler(ts.URL)
	r, err := il.GetRoute("vpavlin-jenkins")
	if err != nil {
		t.Error(err)
	}

	if r != "jenkins-vpavlin-jenkins.d800.free-int.openshiftapps.com" {
		t.Error("Did not get correct route: ", r)
	}
}
