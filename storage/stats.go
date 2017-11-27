package storage

import (

)

type Statistics struct {
	User string `gorm:"primary_key"` // This is the ID PK field
	LastAccessed int64
	LastBufferedRequest int64
}

func NewStatistics(user string, la int64, lbr int64) *Statistics {
	return &Statistics{
		User: user,
		LastAccessed: la,
		LastBufferedRequest: lbr,
	}
}

func (m Statistics) TableName() string {
	return "statistics"
}