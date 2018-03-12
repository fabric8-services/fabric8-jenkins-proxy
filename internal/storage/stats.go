package storage

import (
	"fmt"
	"time"
)

// Statistics consists of namespace, last time time was accessed and last buffered request to that namespace.
type Statistics struct {
	Namespace           string `gorm:"primary_key"` // This is the ID PK field
	LastAccessed        int64
	LastBufferedRequest int64
}

// NewStatistics returns an instance of statistics on giving namespace, last accessed time and last buffered request as an input.
func NewStatistics(ns string, la int64, lbr int64) *Statistics {
	return &Statistics{
		Namespace:           ns,
		LastAccessed:        la,
		LastBufferedRequest: lbr,
	}
}

// TableName returns table name for the given statistics.
func (m Statistics) TableName() string {
	return "statistics"
}

func (m Statistics) String() string {
	return fmt.Sprintf("Statistics[ns: %s, lastAccessed: %s, lastBufferedRequest: %s]",
		m.Namespace, time.Unix(m.LastAccessed, 0), time.Unix(m.LastBufferedRequest, 0))
}
