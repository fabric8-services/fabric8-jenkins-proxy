package storage

import (
	"github.com/jinzhu/gorm"
)

func NewDBService(db *gorm.DB) *DBService {
	return &DBService{db: db}
}

type DBService struct {
	db *gorm.DB
}

func (s *DBService) CreateRequest(r *Request) (error) {
	return s.db.Save(r).Error
}

func (s *DBService) GetRequests(ns string) (result []Request, err error) {
	var r Request
	err = s.db.Table(r.TableName()).Where("namespace = ?", ns).Find(&result).Error
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

func (s *DBService) CreateStatistics(o *Statistics) (error) {
	return s.db.Save(o).Error
}

func (s *DBService) GetStatisticsUser(user string) (o Statistics, err error) {
	err = s.db.Table(o.TableName()).Where("user = ?", user).Find(&o).Error
	return o, err
}