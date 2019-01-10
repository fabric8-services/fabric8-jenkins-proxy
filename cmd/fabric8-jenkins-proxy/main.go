package main

import (
	"net/http"
	"os"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/api/app"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/jenkinsapi"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/router"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/wit"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"

	"github.com/rs/cors"

	"context"

	_ "net/http/pprof"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fabric8-services/fabric8-common/log"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/version"
	goalogrus "github.com/goadesign/goa/logging/logrus"
	"github.com/sirupsen/logrus"
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

var mainLogger = logrus.WithFields(logrus.Fields{"component": "main"})

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	var level logrus.Level
	switch levelStr, _ := os.LookupEnv("JC_LOGRUS_LEVEL"); levelStr {
	case "info":
		level = logrus.InfoLevel
	case "debug":
		level = logrus.DebugLevel
	case "warning":
		level = logrus.WarnLevel
	case "error":
		level = logrus.ErrorLevel
	default:
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
}

func main() {
	mainLogger.Info("Starting  proxy..")
	mainLogger.Infof("Proxy version: %s", version.GetVersion())

	// Init configuration
	config, err := configuration.NewConfiguration()
	if err != nil {
		logrus.Fatal(err)
	}

	mainLogger.Infof("Proxy config: %s", config.String())

	// Connect to DB
	db, err := storage.Connect(config)
	if err != nil {
		logrus.Fatal(err)
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
		logrus.Fatal(err)
	}

	// Start the various Go routines
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startWorkers(ctx, &wg, cancel, store, &proxy, defaultStatsLoggingInterval, config)
	setupSignalChannel(cancel)
	wg.Wait()
}

func serviceListennAndServe(service *goa.Service, cancel context.CancelFunc,
	enableHTTPS bool, port string) {
	if enableHTTPS {
		if err := service.ListenAndServeTLS(port, "server.crt", "server.key"); err != nil {
			logrus.Error(err)
			cancel()
			return
		}
	} else {
		if err := service.ListenAndServe(port); err != nil {
			logrus.Error(err)
			cancel()
			return
		}
	}

	if err := service.ListenAndServe(":8080"); err != nil {
		service.LogError("startup", "err", err)
	}

}

func listenAndServe(srv *http.Server, cancel context.CancelFunc, enableHTTPS bool) {
	if enableHTTPS {
		if err := srv.ListenAndServeTLS("server.crt", "server.key"); err != nil {
			logrus.Error(err)
			cancel()
			return
		}
	} else {
		if err := srv.ListenAndServe(); err != nil {
			logrus.Error(err)
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

	apiService := goa.New("fabric8-jenkins-proxy-api")
	wg.Add(1)
	go func() {
		defer wg.Done()
		prepareAPIService(apiService)

		go func() {
			// Mount "stats" controller
			c := api.NewStatsController(apiService, store)
			app.MountStatsController(apiService, c)
			router.CustomMuxHandle(apiService)

			mainLogger.Infof("Starting API router on port %s", apiRouterPort)
			serviceListennAndServe(apiService, cancel, config.GetHTTPSEnabled(), apiRouterPort)
		}()

		for {
			select {
			case <-ctx.Done():
				mainLogger.Infof("Shutting down API router on port %s", apiRouterPort)
				apiService.CancelAll()
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

func prepareAPIService(service *goa.Service) {
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(true))
	service.Use(middleware.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	service.WithLogger(goalogrus.New(log.Logger()))
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
