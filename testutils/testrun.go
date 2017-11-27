package testutils

import (
	"fmt"
	"net/http"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/jinzhu/gorm"
	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/storage"
	log "github.com/sirupsen/logrus"
)

func Run() {
	js := MockJenkins()
	defer js.Close()
	os := MockOpenShift(js.URL)
	defer os.Close()
	ts := MockServer(TenantData1(os.URL))
	defer ts.Close()
	is := MockServer(IdlerData1(js.URL))
	defer is.Close()
	ws := MockServer(WITData1())
	defer ws.Close()
	as := MockRedirect(AuthData1())
	defer as.Close()

	log.Warn(fmt.Sprintf("JS: %s, OS: %s, TS: %s, IS: %s, WS: %s, AS: %s", js.URL, os.URL, ts.URL, is.URL, ws.URL, as.URL))


	tc := clients.NewTenant(ts.URL, "xxx")
	i := clients.NewIdler(is.URL)
	w := clients.NewWIT(ws.URL, "xxx")

	db, err := gorm.Open("sqlite3", "/tmp/proxy_test.db")
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	db.CreateTable(&storage.Request{})
	db.CreateTable(&storage.Statistics{})

	storageService := storage.NewDBService(db)

	p, err := proxy.NewProxy(tc, w, i, "https://sso.prod-preview.openshift.io", as.URL, "https://localhost:8443/", storageService, "static/html/index.html")
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