// You can edit this code!
// Click here and start typing.
package main

import (
	"github.com/fabric8-services/fabric8-jenkins-proxy/api"
	"strings"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-jenkins-proxy/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"

	"github.com/julienschmidt/httprouter"
	viper "github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
)

func init() {
  log.SetFormatter(&log.JSONFormatter{})
}

func main() {

	v := viper.New()
	v.SetEnvPrefix("JC")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetTypeByDefaultValue(true)

	missingParam := false
	apiURL := v.GetString("idler.api.url")
	_, err := url.ParseRequestURI(apiURL)
	if len(apiURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide URL to Idler API endpoint in JC_IDLER_API_URL environment variable")
	}

	authToken := v.GetString("auth.token")
	if len(authToken) == 0 {
		missingParam = true
		log.Error("You need to provide fabric8-auth token")
	}
	tenantApiURL := v.GetString("f8tenant.api.url")
	_, err = url.ParseRequestURI(tenantApiURL)
	if len(tenantApiURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide fabric8-tenant service URL")
	}

	witApiURL := v.GetString("wit.api.url")
	_, err = url.ParseRequestURI(witApiURL)
	if len(witApiURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide WIT API service URL")
	}

	redirURL := v.GetString("redirect.url")
	_, err = url.ParseRequestURI(redirURL)
	if len(redirURL) == 0 || err != nil {
		missingParam = true
		log.Error("You need to provide redirect URL")
	}

	if missingParam {
		log.Fatal("A value for envinronment variable(s) is missing")
	}

	t := clients.NewTenant(tenantApiURL, authToken)
	w := clients.NewWIT(witApiURL, authToken)
	il := clients.NewIdler(apiURL)

	prx := proxy.NewProxy(t, w, il, redirURL)
	api := api.NewAPI(&prx)
	proxyMux := http.NewServeMux()	

	prxRouter := httprouter.New()
	prxRouter.GET("/papi/info/:namespace", api.Info)

	go func() {
		http.ListenAndServe(":9091", prxRouter)
	}()
	
	proxyMux.HandleFunc("/", prx.Handle)

	http.ListenAndServe(":8080", proxyMux)
}