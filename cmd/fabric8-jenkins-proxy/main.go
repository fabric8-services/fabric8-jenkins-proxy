package main

import (
	"net/http"
	"os"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/router"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"

	"github.com/rs/cors"

	"context"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/version"
	log "github.com/sirupsen/logrus"

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
	mainLogger.Info("Starting  proxy..")
	mainLogger.Infof("Proxy version: %s", version.GetVersion())

	// Init configuration
	config, err := configuration.NewConfiguration()
	if err != nil {
		log.Fatal(err)
	}

	mainLogger.Infof("Proxy config: %s", config.String())

	// Connect to DB
	db, err := storage.Connect(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := storage.NewDBStorage(db)

	// Create tenant client
	tenant := clients.NewTenant(config.GetTenantURL(), config.GetAuthToken())

	// Create WorkItemTracker client
	wit := clients.NewWIT(config.GetWitURL(), config.GetAuthToken())

	// Create Idler client
	idler := clients.NewIdler(config.GetIdlerURL())

	// Get the cluster view from the Idler
	clusters, err := idler.Clusters()
	if err != nil {
		mainLogger.WithField("error", err).Fatalf("Failure to retrieve cluster view")
	}
	mainLogger.WithField("clusters", clusters).Info("Retrieved cluster view")

	start(config, &tenant, wit, idler, store, clusters)
}

func start(config configuration.Configuration, tenant *clients.Tenant, wit clients.WIT, idler clients.IdlerService, store storage.Store, clusters map[string]string) {
	proxy, err := proxy.NewProxy(tenant, wit, idler, store, config, clusters)
	if err != nil {
		log.Fatal(err)
	}

	// Start the various Go routines
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startWorkers(ctx, &wg, cancel, store, &proxy, defaultStatsLoggingInterval, config.GetDebugMode())
	setupSignalChannel(cancel)
	wg.Wait()
}

func startWorkers(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, store storage.Store, proxy *proxy.Proxy, interval time.Duration, addProfiler bool) {
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

	api := api.NewAPI(store)
	wg.Add(1)
	go func() {
		defer wg.Done()
		srv := newAPIServer(api)

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
		srv := newProxyServer(proxy)
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

func newAPIServer(api api.ProxyAPI) *http.Server {
	c := cors.New(cors.Options{
		AllowCredentials: true,
	})
	srv := &http.Server{
		Addr:    apiRouterPort,
		Handler: c.Handler(router.CreateAPIRouter(api)),
	}
	return srv
}

func newProxyServer(p *proxy.Proxy) *http.Server {
	c := cors.New(cors.Options{
		AllowCredentials: true,
	})
	srv := &http.Server{
		Addr:    proxyPort,
		Handler: c.Handler(router.CreateProxyRouter(p)),
	}
	return srv
}
