package main

import (
	"net/http"
	"os"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/jenkinsapi"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/router"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/wit"

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
	jenkinsAPIRouterPort        = ":9092"
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

	// Create auth client and set it as default; can be accessed by
	// auth.DefaultClient() in other packages
	auth.SetDefaultClient(auth.NewClient(config.GetAuthURL()))

	// Create tenant client
	tenant := tenant.New(config.GetTenantURL(), config.GetAuthToken())

	// Create WorkItemTracker client
	wit := wit.New(config.GetWitURL(), config.GetAuthToken())

	// Create Idler client
	idler := idler.New(config.GetIdlerURL())

	// Get the cluster view from the Idler
	clusters, err := idler.Clusters()
	if err != nil {
		mainLogger.WithField("error", err).Fatalf("Failure to retrieve cluster view")
	}
	mainLogger.WithField("clusters", clusters).Info("Retrieved cluster view")

	start(config, idler, &tenant, wit, store, clusters)
}

func start(config configuration.Configuration, idler idler.Service, tenant tenant.Service, wit wit.Service, store storage.Store, clusters map[string]string) {
	proxy, err := proxy.New(idler, tenant, wit, store, config, clusters)
	if err != nil {
		log.Fatal(err)
	}

	// Start the various Go routines
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startWorkers(ctx, &wg, cancel, store, &proxy, defaultStatsLoggingInterval, config)
	setupSignalChannel(cancel)
	wg.Wait()
}

func listenAndServe(srv *http.Server, cancel context.CancelFunc, enableHTTPS bool) {
	if enableHTTPS {
		if err := srv.ListenAndServeTLS("server.crt", "server.key"); err != nil {
			log.Error(err)
			cancel()
			return
		}
	} else {
		if err := srv.ListenAndServe(); err != nil {
			log.Error(err)
			cancel()
			return
		}
	}
}

func startWorkers(
	ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc,
	store storage.Store, proxy *proxy.Proxy, interval time.Duration,
	config configuration.Configuration) {

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
			listenAndServe(srv, cancel, config.GetHTTPSEnabled())
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

	tenant := tenant.New(config.GetTenantURL(), config.GetAuthToken())
	idler := idler.New(config.GetIdlerURL())
	jenkinsAPI := jenkinsapi.NewJenkinsAPI(&tenant, idler)
	wg.Add(1)
	go func() {
		defer wg.Done()
		srv := newJenkinsAPIServer(jenkinsAPI, config)

		go func() {
			mainLogger.Infof("Starting Jenkins Status API router on port %s", jenkinsAPIRouterPort)
			listenAndServe(srv, cancel, config.GetHTTPSEnabled())
		}()

		for {
			select {
			case <-ctx.Done():
				mainLogger.Infof("Shutting down Jenkins Status API router on port %s", jenkinsAPIRouterPort)
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
			listenAndServe(srv, cancel, config.GetHTTPSEnabled())
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

	// add profile if debug mode is enabled
	if config.GetDebugMode() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv := &http.Server{
				Addr: profilerPort,
			}

			go func() {
				mainLogger.Infof("Starting profiler on port %s", profilerPort)
				listenAndServe(srv, cancel, config.GetHTTPSEnabled())
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
	return &http.Server{
		Addr:    apiRouterPort,
		Handler: router.CreateAPIRouter(api),
	}
}

func newJenkinsAPIServer(jenkinsAPI jenkinsapi.JenkinsAPI, config configuration.Configuration) *http.Server {
	c := cors.New(cors.Options{
		AllowedOrigins: config.GetAllowedOrigins(),
		AllowedMethods: []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders: []string{"*"},
		Debug:          config.GetDebugMode(),
	})
	srv := &http.Server{
		Addr:    jenkinsAPIRouterPort,
		Handler: c.Handler(router.CreateJenkinsAPIRouter(jenkinsAPI)),
	}
	return srv
}

func newProxyServer(p *proxy.Proxy) *http.Server {
	srv := &http.Server{
		Addr:    proxyPort,
		Handler: router.CreateProxyRouter(p),
	}
	return srv
}
