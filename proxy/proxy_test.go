package proxy_test

import (
	"strings"
	"fmt"
	"bytes"
	"io/ioutil"
	"net/http"
	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/proxy"
	tu "github.com/fabric8-services/fabric8-jenkins-proxy/testutils"
)

func TestGHData(t *testing.T) {
	ts := tu.MockServer(tu.TenantData1())
	defer ts.Close()
	is := tu.MockServer(tu.IdlerData2())
	defer is.Close()
	ws := tu.MockServer(tu.WITData1())

	tc := clients.NewTenant(ts.URL, "xxx")
	i := clients.NewIdler(is.URL)
	w := clients.NewWIT(ws.URL, "xxx")
	p := proxy.NewProxy(tc, w, i, "https://prod-preview.openshift.io")

	proxyMux := http.NewServeMux()	

	proxyMux.HandleFunc("/", p.Handle)
	
	go http.ListenAndServe(":8080", proxyMux)

	js := http.NewServeMux()
	js.HandleFunc("/", HandleJenkins)
	go http.ListenAndServe(":8888", js)

	b := ioutil.NopCloser(bytes.NewReader(tu.GetGHData()))
	resp, err := http.Get("http://localhost:8080/")
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != 200 {
		t.Error(resp.Status)
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/github-webhook/", b)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("User-Agent", "GitHub-Hookshot/95a0192")
	req.Header.Add("X-Hub-Signature", "sha1=07cabddd7b5094a28dab9e0a23ec8d2e8dc7ab83")
	req.Header.Add("X-GitHub-Event", "push")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%+v\n", resp)
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != 200 {
		t.Error("Could not proxy successfully: ", resp.Status)
	}
}

func HandleJenkins(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" && r.URL.Path == "/github-webhook/" {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if strings.Contains(string(body), "github.com/vpavlin") {
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}