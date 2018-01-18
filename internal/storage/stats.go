package storage

type Statistics struct {
	Namespace           string `gorm:"primary_key"` // This is the ID PK field
	LastAccessed        int64
	LastBufferedRequest int64
}

func NewStatistics(ns string, la int64, lbr int64) *Statistics {
	return &Statistics{
		Namespace:           ns,
		LastAccessed:        la,
		LastBufferedRequest: lbr,
	}
}

func (m Statistics) TableName() string {
	return "statistics"
}
