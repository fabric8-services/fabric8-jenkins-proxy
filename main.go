// You can edit this code!
// Click here and start typing.
package main

import (
	"net/http"
	"os"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/api"
	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/testutils"

	_ "github.com/lib/pq"

	"github.com/jinzhu/gorm"
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

	db := connect(config)
	defer db.Close()

	storageService := storage.NewDBService(nil)

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

func connect(config *configuration.Data) *gorm.DB {
	var err error
	var db *gorm.DB
	for {
		db, err = gorm.Open("postgres", config.GetPostgresConfigString())
		if err != nil {
			log.Errorf("ERROR: Unable to open connection to database %v", err)
			log.Infof("Retrying to connect in %v...", config.GetPostgresConnectionRetrySleep())
			time.Sleep(config.GetPostgresConnectionRetrySleep())
		} else {
			break
		}
	}

	if config.GetPostgresConnectionMaxIdle() > 0 {
		log.Infof("Configured connection pool max idle %v", config.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(config.GetPostgresConnectionMaxIdle())
	}
	if config.GetPostgresConnectionMaxOpen() > 0 {
		log.Infof("Configured connection pool max open %v", config.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(config.GetPostgresConnectionMaxOpen())
	}

	db.CreateTable(&storage.Request{})
	db.CreateTable(&storage.Statistics{})

	return db
}