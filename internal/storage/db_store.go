package storage

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var dbLogger = log.WithFields(log.Fields{"component": "db"})

func NewDBStorage(db *gorm.DB) Store {
	return &DBStore{db: db}
}

type DBStore struct {
	db *gorm.DB
}

func (s *DBStore) CreateRequest(r *Request) error {
	return s.db.Create(r).Error
}

func (s *DBStore) GetRequests(ns string) (result []Request, err error) {
	var r Request
	err = s.db.Table(r.TableName()).Where("namespace = ?", ns).Find(&result).Error
	return
}

func (s *DBStore) IncRequestRetry(r *Request) (errs []error) {
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

func (s *DBStore) GetUsers() (result []string, err error) {
	var r Request

	err = s.db.Table(r.TableName()).Pluck("DISTINCT namespace", &result).Error
	return
}

func (s *DBStore) GetRequestsCount(ns string) (result int, err error) {
	var r Request
	err = s.db.Table(r.TableName()).Where("namespace = ?", ns).Count(&result).Error
	return
}

func (s *DBStore) DeleteRequest(r *Request) error {
	return s.db.Delete(r).Error
}

func (s *DBStore) CreateStatistics(o *Statistics) error {
	return s.db.Create(o).Error
}

func (s *DBStore) UpdateStatistics(o *Statistics) error {
	return s.db.Save(o).Error
}

func (s *DBStore) GetStatisticsUser(ns string) (o *Statistics, notFound bool, err error) {
	o = &Statistics{}
	d := s.db.Table(o.TableName()).Find(
		o, "namespace = ?", ns)
	err = d.Error
	notFound = d.RecordNotFound()
	return
}

func (s *DBStore) LogStats() {
	var requestCount, statisticCount int
	s.db.Table("requests").Count(&requestCount)
	s.db.Table("statistics").Count(&statisticCount)
	dbLogger.Infof("Cached requests: %d. Statistic entries count: %d", requestCount, statisticCount)
}

func (s *DBStore) updateRequest(r *Request) error {
	return s.db.Save(r).Error
}
