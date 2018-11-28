package main

import (
	"database/sql"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"io/ioutil"
	"strconv"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/wit"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/ory-am/dockertest.v3"
)

var (
	mockConfig = configuration.NewMock()
)

func TestMain(m *testing.M) {
	var db *sql.DB
	var err error
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "9.6", []string{"POSTGRES_PASSWORD=" + mockConfig.GetPostgresPassword(), "POSTGRES_DB=" + mockConfig.GetPostgresDatabase()})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if err = pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", mockConfig.GetPostgresUser(), mockConfig.GetPostgresPassword(), resource.GetPort("5432/tcp"), mockConfig.GetPostgresDatabase()))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// we need to make sure that the test can use the port exposed on the host
	port, _ := strconv.Atoi(resource.GetPort("5432/tcp"))
	mockConfig.PostgresPort = port
	code := m.Run()

	// When you're done, kill and remove the container
	err = pool.Purge(resource)

	os.Exit(code)
}

func TestProxy(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	// register a global log hook
	hook := test.NewGlobal()

	js := testutils.MockJenkins()
	defer js.Close()
	openShift := testutils.MockOpenShift(js.URL)
	defer openShift.Close()
	ts := testutils.MockServer(testutils.TenantData1(openShift.URL))
	defer ts.Close()
	is := testutils.MockServer(testutils.IdlerData1(js.URL))
	defer is.Close()
	ws := testutils.MockServer(testutils.WITData1())
	defer ws.Close()
	as := testutils.MockRedirect(testutils.AuthData1())
	defer as.Close()

	log.Info(fmt.Sprintf("JS: %s, OS: %s, TS: %s, IS: %s, WS: %s, AS: %s", js.URL, openShift.URL, ts.URL, is.URL, ws.URL, as.URL))

	tenant := tenant.New(ts.URL, "xxx")
	idler := idler.New(is.URL)
	wit := wit.New(ws.URL, "xxx")

	mockConfig.AuthURL = as.URL
	mockConfig.TenantURL = ts.URL
	mockConfig.IdlerURL = is.URL
	mockConfig.WitURL = ws.URL

	db, err := storage.Connect(&mockConfig)
	log.Info(storage.PostgresConfigString(&mockConfig))
	assert.NoError(t, err)
	defer db.Close()

	store := storage.NewDBStorage(db)

	go func() {
		// Send SIGTERM after max 10 seconds
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			logMessages := testutils.ExtractLogMessages(hook.Entries)
			if contains(logMessages, "Starting API router on port :9091") {
				syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			}
		}
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	clusters := make(map[string]string)
	clusters["https://api.free-stg.openshift.com/"] = "1b7d.free-stg.openshiftapps.com"
	start(&mockConfig, idler, &tenant, wit, store, clusters)

	// TODO - Test an actual workflow by triggering some of the MockURLs

	logMessages := testutils.ExtractLogMessages(hook.Entries)
	assert.Contains(t, logMessages, "Shutting down proxy on port :8080", "Proxy should shut down gracefully")
	assert.Contains(t, logMessages, "Shutting down API router on port :9091", "API router should shutdown gracefully")
}

func contains(list []string, s string) bool {
	for _, elem := range list {
		if elem == s {
			return true
		}
	}
	return false
}
