package testutils

import (
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/proxy"
	log "github.com/sirupsen/logrus"
)

func Run() {
	ts := MockServer(TenantData1())
	defer ts.Close()
	is := MockServer(IdlerData1())
	defer is.Close()
	ws := MockServer(WITData1())

	tc := clients.NewTenant(ts.URL, "xxx")
	i := clients.NewIdler(is.URL)
	w := clients.NewWIT(ws.URL, "xxx")
	p, err := proxy.NewProxy(tc, w, i, "https://sso.prod-preview.openshift.io",  "https://auth.prod-preview.openshift.io", "https://localhost:8443/")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Starting test proxy..")

	proxyMux := http.NewServeMux()	

	proxyMux.HandleFunc("/", p.Handle)
	err = http.ListenAndServeTLS(":8443", "server.crt", "server.key", proxyMux)
	if err != nil {
		log.Error(err)
	}
	log.Info("Proxy finished..")
}