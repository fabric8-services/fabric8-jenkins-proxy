// You can edit this code!
// Click here and start typing.
package main

import (
	"net/http"
	"os"

	"github.com/fabric8-services/fabric8-jenkins-proxy/api"
	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/testutils"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

func init() {
  log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}

	if config.GetLocalDevEnv() {
		testutils.Run()
		os.Exit(0)
	}

	config.VerifyConfig()

	db := storage.Connect(config)
	defer db.Close()

	storageService := storage.NewDBService(db)

	t := clients.NewTenant(config.GetTenantURL(), config.GetAuthToken())
	w := clients.NewWIT(config.GetWitURL(), config.GetAuthToken())
	il := clients.NewIdler(config.GetIdlerURL())

	prx, err := proxy.NewProxy(t, w, il, config.GetKeycloakURL(), config.GetAuthURL(), config.GetRedirectURL(), storageService, config.GetIndexPath())
	if err != nil {
		log.Fatal(err)
	}
	api := api.NewAPI(storageService)
	proxyMux := http.NewServeMux()	

	prxRouter := httprouter.New()
	prxRouter.GET("/papi/info/:namespace", api.Info)

	go func() {
		http.ListenAndServe(":9091", prxRouter)
	}()
	
	proxyMux.HandleFunc("/", prx.Handle)

	http.ListenAndServe(":8080", proxyMux)
}

