package storage

import (
	"fmt"

	"github.com/jinzhu/gorm"
	// Importing postgres driver to connect to the database
	// The underscore import is used for the side-effect of a package.
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var dbLogger = log.WithFields(log.Fields{"component": "db"})

// NewDBStorage creates an instance of database client.
func NewDBStorage(db *gorm.DB) Store {
	return &DBStore{db: db}
}

// DBStore describes a database client.
type DBStore struct {
	db *gorm.DB
}

// CreateRequest creates an entry of request in the database.
func (s *DBStore) CreateRequest(r *Request) error {
	return s.db.Create(r).Error
}

// GetRequests gets one or more requests from the database given a namespace as input.
func (s *DBStore) GetRequests(ns string) (result []Request, err error) {
	var r Request
	err = s.db.Table(r.TableName()).Where("namespace = ?", ns).Find(&result).Error
	return
}

// IncrementRequestRetry increases retries for a given request in the database.
func (s *DBStore) IncrementRequestRetry(r *Request) (errs []error) {
	r.Retries++
	err := s.updateRequest(r)
	if err != nil {
		errs = append(errs, fmt.Errorf("could not update request for %s (%s) - deleting: %s", r.ID, r.Namespace, err))
		err = s.DeleteRequest(r)
		if err != nil {
			errs = append(errs, fmt.Errorf(ErrorFailedDelete, r.ID, r.Namespace, err))
		}
	}

	return
}

// GetUsers gets namespaces from the database.
func (s *DBStore) GetUsers() (result []string, err error) {
	var r Request

	err = s.db.Table(r.TableName()).Pluck("DISTINCT namespace", &result).Error
	return
}

// GetRequestsCount gets requests count given a namespace.
func (s *DBStore) GetRequestsCount(ns string) (result int, err error) {
	var r Request
	err = s.db.Table(r.TableName()).Where("namespace = ?", ns).Count(&result).Error
	return
}

// DeleteRequest deletes a request from the database.
func (s *DBStore) DeleteRequest(r *Request) error {
	return s.db.Delete(r).Error
}

// DeleteRequestsUser deletes requests of a namespace from the database.
func (s *DBStore) DeleteRequestsUser(ns string) error {
	return s.db.Unscoped().Table("requests").Where("namespace = ?", ns).Delete(&Request{}).Error
}

// CreateStatistics creates an entry of Statistics in the database.
func (s *DBStore) CreateStatistics(o *Statistics) error {
	return s.db.Create(o).Error
}

// UpdateStatistics updates Statistics in the database.
func (s *DBStore) UpdateStatistics(o *Statistics) error {
	return s.db.Save(o).Error
}

// GetStatisticsUser gets Statistics of a namespace from the database.
func (s *DBStore) GetStatisticsUser(ns string) (o *Statistics, notFound bool, err error) {
	o = &Statistics{}
	d := s.db.Table(o.TableName()).Find(
		o, "namespace = ?", ns)
	err = d.Error
	notFound = d.RecordNotFound()
	return
}

// DeleteStatisticsUser deletes Statistics of a namespace from the database.
func (s *DBStore) DeleteStatisticsUser(ns string) error {
	return s.db.Unscoped().Table("statistics").Where("namespace = ?", ns).Delete(&Statistics{}).Error
}

// LogStats logs number of cached number of cached requests and statistics entries count.
func (s *DBStore) LogStats() {
	var requestCount, statisticCount int
	s.db.Table("requests").Count(&requestCount)
	s.db.Table("statistics").Count(&statisticCount)
	dbLogger.Infof("Cached requests: %d. Statistic entries count: %d", requestCount, statisticCount)
}

func (s *DBStore) updateRequest(r *Request) error {
	return s.db.Save(r).Error
}
