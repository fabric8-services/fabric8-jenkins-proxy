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
