package storage

import (
	"context"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// // TODO turn this into a general in memory store service (HF)
// type mockStore struct {
// 	buffer bytes.Buffer
// }

// func (m *mockStore) CreateRequest(r *Request) error {
// 	return nil
// }

// func (m *mockStore) GetRequests(ns string) (result []Request, err error) {
// 	return nil, nil
// }

// func (m *mockStore) IncrementRequestRetry(r *Request) (errs []error) {
// 	return nil
// }

// func (m *mockStore) GetUsers() (result []string, err error) {
// 	return nil, nil
// }

// func (m *mockStore) GetRequestsCount(ns string) (result int, err error) {
// 	return 0, nil
// }

// func (m *mockStore) DeleteRequest(r *Request) error {
// 	return nil
// }

// func (m *mockStore) CreateStatistics(o *Statistics) error {
// 	return nil
// }

// func (m *mockStore) UpdateStatistics(o *Statistics) error {
// 	return nil
// }

// func (m *mockStore) GetStatisticsUser(ns string) (o *Statistics, notFound bool, err error) {
// 	return nil, false, nil
// }

// func (m *mockStore) LogStats() {
// 	dbLogger.Info("mock db stats")
// }

func Test_logging_of_store_stats(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	testLogger, hook := test.NewNullLogger()
	dbLogger = testLogger.WithFields(log.Fields{"component": "db"})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := LogStorageStats(ctx, &Mock{}, 100*time.Millisecond); err != nil {
			cancel()
		}
	}()

	wg.Wait()

	assert.Len(t, hook.Entries, 9, "Unexpected log message count.")
	for _, entry := range hook.Entries {
		assert.Equal(t, "mock db stats", entry.Message, "Unexpected log message")
	}
}
