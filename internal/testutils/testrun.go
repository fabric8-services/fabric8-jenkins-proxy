package testutils

import (
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/julienschmidt/httprouter"
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

	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}

	log.Info(config.GetPostgresConfigString())

	db := storage.Connect(config)
	defer db.Close()

	storageService := storage.NewDBService(db)
	// db, err := gorm.Open("sqlite3", "/tmp/proxy_test.db")
	// if err != nil {
	// 	log.Panic(err)
	// }
	// defer db.Close()

	// db.CreateTable(&storage.Request{})
	// db.CreateTable(&storage.Statistics{})

	// storageService := storage.NewDBService(db)

	p, err := proxy.NewProxy(tc, w, i, "https://sso.prod-preview.openshift.io", as.URL, "https://localhost:8443/", storageService, "static/html/index.html", config.GetMaxRequestretry())
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Starting test proxy..")

	proxyMux := http.NewServeMux()
	prxRouter := httprouter.New()

	api := api.NewAPI(storageService)
	prxRouter.GET("/papi/info/:namespace", api.Info)

	go func() {
		http.ListenAndServe(":9091", prxRouter)
	}()

	proxyMux.HandleFunc("/", p.Handle)
	err = http.ListenAndServeTLS(":8443", "server.crt", "server.key", proxyMux)
	if err != nil {
		log.Error(err)
	}

	log.Info("Proxy finished..")
}
