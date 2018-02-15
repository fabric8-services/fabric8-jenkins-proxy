package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/storage"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/testutils"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/ory-am/dockertest.v3"
)

const (
	database = "tenant"
	user     = "postgres"
	password = "mysecretpassword"
)

func TestMain(m *testing.M) {
	var db *sql.DB
	var err error
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "9.6", []string{"POSTGRES_PASSWORD=" + password, "POSTGRES_DB=" + database})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if err = pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", user, password, resource.GetPort("5432/tcp"), database))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// we needto make sure that the test can use the port exposed on the host
	os.Setenv("JC_POSTGRES_PORT", resource.GetPort("5432/tcp"))
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

	tenant := clients.NewTenant(ts.URL, "xxx")
	idler := clients.NewIdler(is.URL)
	wit := clients.NewWIT(ws.URL, "xxx")

	// Make sure the right config values are picked up from the environment for this test
	// TODO - Once Configuration is an interface we can provide a mock config (HF)
	os.Setenv("JC_KEYCLOAK_URL", "https://sso.prod-preview.openshift.io")
	os.Setenv("JC_AUTH_URL", as.URL)
	os.Setenv("JC_REDIRECTREDIRECT_URL", "https://localhost:8443/")
	os.Setenv("JC_INDEX_PATH", "static/html/index.html")
	//os.Setenv("JC_OSO_CLUSTERS", testutils.OSOClusters())
	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}

	log.Info(config.GetPostgresConfigString())

	db := storage.Connect(config)
	defer db.Close()

	store := storage.NewDBStorage(db)

	go func() {
		// Send SIGTERM after two seconds
		time.Sleep(2 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	start(config, &tenant, &wit, &idler, store)

	// TODO - Test an actual workflow by triggering some of the MockURLs

	logMessages := testutils.ExtractLogMessages(hook.Entries)
	assert.Contains(t, logMessages, "Shutting down proxy on port :8080", "Proxy should shut down gracefully")
	assert.Contains(t, logMessages, "Shutting down API router on port :9091", "API router should shutdown gracefully")
}
