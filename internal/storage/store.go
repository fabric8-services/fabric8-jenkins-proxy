package storage

import (
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/configuration"
	"github.com/jinzhu/gorm"
	"time"

	"context"
	"fmt"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var storeLogger = log.WithFields(log.Fields{"component": "store"})

type Store interface {
	CreateRequest(r *Request) error
	GetRequests(ns string) (result []Request, err error)
	IncRequestRetry(r *Request) (errs []error)
	GetUsers() (result []string, err error)
	GetRequestsCount(ns string) (result int, err error)
	DeleteRequest(r *Request) error

	CreateStatistics(o *Statistics) error
	UpdateStatistics(o *Statistics) error
	GetStatisticsUser(ns string) (o *Statistics, notFound bool, err error)

	LogStats()
}

func LogStorageStats(ctx context.Context, store Store, interval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			storeLogger.Info("Stopping to log store statistics.")
			return ctx.Err()
		case <-time.After(interval):
			store.LogStats()
		}
	}
}

func Connect(config configuration.Configuration) (*gorm.DB, error) {
	var err error
	var db *gorm.DB
	for {
		db, err = gorm.Open("postgres", PostgresConfigString(config))
		if err != nil {
			return nil, err
		} else {
			storeLogger.Info("Successfully connected to database")
			break
		}
	}

	if config.GetPostgresConnectionMaxIdle() > 0 {
		storeLogger.Infof("Configured connection pool max idle %v", config.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(config.GetPostgresConnectionMaxIdle())
	}
	if config.GetPostgresConnectionMaxOpen() > 0 {
		storeLogger.Infof("Configured connection pool max open %v", config.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(config.GetPostgresConnectionMaxOpen())
	}

	if config.GetDebugMode() {
		db = db.Debug()
	}

	request := &Request{}
	if !db.HasTable(request) {
		db.CreateTable(request)
	}

	stats := &Statistics{}
	if !db.HasTable(stats) {
		db.CreateTable(stats)
	}

	return db, nil
}

// GetPostgresConfigString returns a ready to use string for usage in sql.Open()
func PostgresConfigString(config configuration.Configuration) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		config.GetPostgresHost(),
		config.GetPostgresPort(),
		config.GetPostgresUser(),
		config.GetPostgresPassword(),
		config.GetPostgresDatabase(),
		config.GetPostgresSSLMode(),
		config.GetPostgresConnectionTimeout(),
	)
}
