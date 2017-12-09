package storage

import (
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-proxy/configuration"
	"github.com/jinzhu/gorm"
	"time"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

func Connect(config *configuration.Data) *gorm.DB {
	var err error
	var db *gorm.DB
	for {
		db, err = gorm.Open("postgres", config.GetPostgresConfigString())
		if err != nil {
			log.Errorf("ERROR: Unable to open connection to database %v", err)
			log.Infof("Retrying to connect in %v...", config.GetPostgresConnectionRetrySleep())
			time.Sleep(config.GetPostgresConnectionRetrySleep())
		} else {
			log.Info("Successfully connected to database")
			break
		}
	}

	if config.GetPostgresConnectionMaxIdle() > 0 {
		log.Infof("Configured connection pool max idle %v", config.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(config.GetPostgresConnectionMaxIdle())
	}
	if config.GetPostgresConnectionMaxOpen() > 0 {
		log.Infof("Configured connection pool max open %v", config.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(config.GetPostgresConnectionMaxOpen())
	}

	db.CreateTable(&Request{})
	db.CreateTable(&Statistics{})

	return db
}

func NewDBService(db *gorm.DB) *DBService {
	return &DBService{db: db}
}

type DBService struct {
	db *gorm.DB
}

func (s *DBService) CreateOrUpdateRequest(r *Request) error {
	return s.db.Save(r).Error
}

func (s *DBService) GetRequests(ns string) (result []Request, err error) {
	var r Request
	err = s.db.Table(r.TableName()).Where("namespace = ?", ns).Find(&result).Error
	return
}

func (s *DBService) IncRequestRetry(r *Request) (errs []error) {
	r.Retries++
	err := s.CreateOrUpdateRequest(r)
	if err != nil {
		errs = append(errs, fmt.Errorf("Could not update request for %s (%s) - deleting: %s", r.ID, r.Namespace, err))
		err = s.DeleteRequest(r)
		if err != nil {
			errs = append(errs, fmt.Errorf(ErrorFailedDelete, r.ID, r.Namespace, err))
		}
	}

	return
}

func (s *DBService) GetUsers() (result []string, err error) {
	var r Request

	err = s.db.Table(r.TableName()).Pluck("DISTINCT namespace", &result).Error
	return
}

func (s *DBService) GetRequestsCount(ns string) (result int, err error) {
	var r Request
	err = s.db.Table(r.TableName()).Where("namespace = ?", ns).Count(&result).Error
	return
}

func (s *DBService) DeleteRequest(r *Request) error {
	return s.db.Delete(r).Error
}

func (s *DBService) CreateStatistics(o *Statistics) error {
	return s.db.Save(o).Error
}

func (s *DBService) GetStatisticsUser(ns string) (o *Statistics, notFound bool, err error) {
	o = &Statistics{}
	d := s.db.Table(o.TableName()).Find(
		o, "namespace = ?", ns)
	err = d.Error
	notFound = d.RecordNotFound()
	return
}
