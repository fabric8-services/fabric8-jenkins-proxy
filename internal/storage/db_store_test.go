package storage

import (
	"database/sql"
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/ory-am/dockertest.v3"
	"io/ioutil"
	"os"
	"testing"
	"time"
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

func Test_logging_of_db_stats(t *testing.T) {
	db, store, hook := setUp(t)
	defer db.Close()

	assert.Len(t, hook.Entries, 0, "There should not be a log message yet.")

	store.LogStats()

	assert.Len(t, hook.Entries, 1, "Unexpected log message count.")
	assert.Equal(t, "Cached requests: 0. Statistic entries count: 0", hook.LastEntry().Message, "Unexpected log message")

	request := &Request{
		ID: uuid.NewV4(),
	}
	err := store.CreateRequest(request)
	assert.NoError(t, err, "Unexpected error creating request.")

	stats := &Statistics{
		Namespace:           "foo",
		LastAccessed:        time.Now().Unix(),
		LastBufferedRequest: time.Now().Unix(),
	}
	err = store.CreateStatistics(stats)
	assert.NoError(t, err, "Unexpected error creating stats.")

	store.LogStats()
	assert.Len(t, hook.Entries, 2, "Unexpected log message count.")
	assert.Equal(t, "Cached requests: 1. Statistic entries count: 1", hook.LastEntry().Message, "Unexpected log message")
}

func setUp(t *testing.T) (*gorm.DB, Store, *test.Hook) {
	log.SetOutput(ioutil.Discard)
	testLogger, hook := test.NewNullLogger()
	dbLogger = testLogger.WithFields(log.Fields{"component": "db"})

	config, err := configuration.NewData()
	if err != nil {
		t.Fatal(err.Error())
	}

	db := Connect(config)
	store := NewDBStorage(db)

	return db, store, hook
}
