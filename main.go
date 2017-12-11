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
	//Init configuration
	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}

	if config.GetDebugMode() {
		log.SetLevel(log.DebugLevel)
	}

	//Run mock services if this is local dev env
	if config.GetLocalDevEnv() {
		testutils.Run()
		os.Exit(0)
	}

	//Check if we have all we need
	config.VerifyConfig()

	//Connect to db
	db := storage.Connect(config)
	defer db.Close()

	storageService := storage.NewDBService(db)

	//Create tenant client
	t := clients.NewTenant(config.GetTenantURL(), config.GetAuthToken())
	//Create WorkItemTracker client
	w := clients.NewWIT(config.GetWitURL(), config.GetAuthToken())
	//Create Idler client
	il := clients.NewIdler(config.GetIdlerURL())

	prx, err := proxy.NewProxy(t, w, il, config.GetKeycloakURL(), config.GetAuthURL(), config.GetRedirectURL(),
															storageService, config.GetIndexPath(), config.GetMaxRequestretry())
	if err != nil {
		log.Fatal(err)
	}
	//Create Proxy API
	api := api.NewAPI(storageService)
	proxyMux := http.NewServeMux()	

	//Creare router for API
	prxRouter := httprouter.New()
	prxRouter.GET("/papi/info/:namespace", api.Info)

	//Listen for API
	go func() {
		http.ListenAndServe(":9091", prxRouter)
	}()
	
	proxyMux.HandleFunc("/", prx.Handle)

	//Listen for Proxy
	http.ListenAndServe(":8080", proxyMux)
}

