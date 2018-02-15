package main

import (
	"net/http"
	"os"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"

	"context"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/version"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/openshift"
	_ "net/http/pprof"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	// defaultStatsLoggingInterval determines the default Duration for logging the store stats.
	defaultStatsLoggingInterval = 5 * time.Minute
	shutdownTimeout             = 5
	apiRouterPort               = ":9091"
	proxyPort                   = ":8080"
	profilerPort                = ":6060"
)

var mainLogger = log.WithFields(log.Fields{"component": "main"})

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	level := log.InfoLevel
	switch levelStr, _ := os.LookupEnv("JC_LOG_LEVEL"); levelStr {
	case "info":
		level = log.InfoLevel
	case "debug":
		level = log.DebugLevel
	case "warning":
		level = log.WarnLevel
	case "error":
		level = log.ErrorLevel
	default:
		level = log.InfoLevel
	}
	log.SetLevel(level)
}

func main() {
	mainLogger.Infof("Proxy version: %s", version.GetVersion())

	//Init configuration
	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}

	//Check if we have all we need
	config.VerifyConfig()

	//Connect to db
	db := storage.Connect(config)
	defer db.Close()

	store := storage.NewDBStorage(db)

	//Create tenant client
	tenant := clients.NewTenant(config.GetTenantURL(), config.GetAuthToken())

	//Create WorkItemTracker client
	wit := clients.NewWIT(config.GetWitURL(), config.GetAuthToken())

	//Create Idler client
	idler := clients.NewIdler(config.GetIdlerURL())

	//Create OpenShift client
	oc := openshift.NewClient(config.GetOpenShiftApiURL(), config.GetOpenShiftApiToken())

	start(config, &tenant, &wit, &idler, oc, store)
}

func start(config *configuration.Data, tenant *clients.Tenant, wit *clients.WIT, idler *clients.Idler, oc openshift.Client, store storage.Store) {
	proxy, err := proxy.NewProxy(tenant, wit, idler, oc, store, config)
	if err != nil {
		log.Fatal(err)
	}

	// Start the various Go routines
	// TODO - Eventually all goroutines should be started and controlled from the method below
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startWorkers(&wg, ctx, cancel, store, &proxy, defaultStatsLoggingInterval, config.GetDebugMode())
	setupSignalChannel(cancel)
	wg.Wait()
}

func startWorkers(wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc, store storage.Store, proxy *proxy.Proxy, interval time.Duration, addProfiler bool) {
	mainLogger.Info("Starting  all workers")
	wg.Add(1)
	go func() {
		defer wg.Done()
		mainLogger.Info("Starting stats logger")
		if err := storage.LogStorageStats(ctx, store, interval); err != nil {
			cancel()
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		srv := &http.Server{
			Addr:    apiRouterPort,
			Handler: createAPIRouter(store),
		}

		go func() {
			mainLogger.Infof("Starting API router on port %s", apiRouterPort)
			if err := srv.ListenAndServe(); err != nil {
				cancel()
				return
			}
		}()

		for {
			select {
			case <-ctx.Done():
				mainLogger.Infof("Shutting down API router on port %s", apiRouterPort)
				ctx, cancel := context.WithTimeout(ctx, shutdownTimeout*time.Second)
				srv.Shutdown(ctx)
				cancel()
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		srv := &http.Server{
			Addr:    proxyPort,
			Handler: createProxyRouter(proxy),
		}

		go func() {
			mainLogger.Infof("Starting proxy on port %s", proxyPort)
			if err := srv.ListenAndServe(); err != nil {
				cancel()
				return
			}
		}()

		for {
			select {
			case <-ctx.Done():
				mainLogger.Infof("Shutting down proxy on port %s", proxyPort)
				ctx, cancel := context.WithTimeout(ctx, shutdownTimeout*time.Second)
				srv.Shutdown(ctx)
				cancel()
				return
			}
		}
	}()

	if addProfiler {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv := &http.Server{
				Addr: profilerPort,
			}

			go func() {
				mainLogger.Infof("Starting profiler on port %s", profilerPort)
				if err := srv.ListenAndServe(); err != nil {
					cancel()
					return
				}
			}()

			for {
				select {
				case <-ctx.Done():
					mainLogger.Infof("Shutting down profiler on port %s", profilerPort)
					ctx, cancel := context.WithTimeout(ctx, shutdownTimeout*time.Second)
					srv.Shutdown(ctx)
					cancel()
					return
				}
			}
		}()
	}
}

func createAPIRouter(store storage.Store) *httprouter.Router {
	// Create Proxy API
	api := api.NewAPI(store)

	// Create router for API
	proxyRouter := httprouter.New()
	proxyRouter.GET("/papi/info/:namespace", api.Info)
	return proxyRouter
}

func createProxyRouter(proxy *proxy.Proxy) *http.ServeMux {
	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/", proxy.Handle)

	return proxyMux
}

// setupSignalChannel registers a listener for Unix signals for a ordered shutdown
func setupSignalChannel(cancel context.CancelFunc) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM)

	go func() {
		<-sigchan
		mainLogger.Info("Received SIGTERM signal. Initiating shutdown.")
		cancel()
	}()
}
