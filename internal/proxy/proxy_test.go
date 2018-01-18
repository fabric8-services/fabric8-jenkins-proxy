package proxy_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	tu "github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils"
)

// TODO - Update or delete test. See https://github.com/fabric8-services/fabric8-jenkins-proxy/issues/72 (HF)
//func TestGHData(t *testing.T) {
//	ts := tu.MockServer(tu.TenantData1(""))
//	defer ts.Close()
//	is := tu.MockServer(tu.IdlerData1(""))
//	defer is.Close()
//	ws := tu.MockServer(tu.WITData1())
//
//	tc := clients.NewTenant(ts.URL, "xxx")
//	i := clients.NewIdler(is.URL)
//	w := clients.NewWIT(ws.URL, "xxx")
//	p, err := proxy.NewProxy(tc, w, i, "https://sso.prod-preview.openshift.io", "https://auth.prod-preview.openshift.io", "https://localhost:8443/")
//	if err != nil {
//		t.Error(err)
//	}
//
//	proxyMux := http.NewServeMux()
//
//	proxyMux.HandleFunc("/", p.Handle)
//	http.ListenAndServeTLS(":8443", "../server.crt", "../server.key", proxyMux)
//	//http.ListenAndServe(":8080", proxyMux)
//
//	js := http.NewServeMux()
//	js.HandleFunc("/", HandleJenkins)
//	http.ListenAndServe(":8888", js)
//
//	resp, err := http.Get("http://localhost:8080/")
//	if err != nil {
//		t.Error(err)
//	}
//	if resp.StatusCode != 200 {
//		t.Error(resp.Status)
//		defer resp.Body.Close()
//		b, err := ioutil.ReadAll(resp.Body)
//		if err != nil {
//			t.Error(err)
//		}
//		t.Error(string(b))
//	}
//
//	client := &http.Client{}
//	WebhookHit(t, client)
//	WebhookHit(t, client)
//}

func WebhookHit(t *testing.T, client *http.Client) {
	b := ioutil.NopCloser(bytes.NewReader(tu.GetGHData()))
	req, err := http.NewRequest("POST", "http://localhost:8080/github-webhook/", b)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("User-Agent", "GitHub-Hookshot/95a0192")
	req.Header.Add("X-Hub-Signature", "sha1=07cabddd7b5094a28dab9e0a23ec8d2e8dc7ab83")
	req.Header.Add("X-GitHub-Event", "push")

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
	}

	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != 202 {
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
