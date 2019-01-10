package storage

// Mock is a mock implementation of Store struct.
// This implementation is meant to be used for testing
type Mock struct {
}

// CreateRequest creates an entry of request in the database.
func (s *Mock) CreateRequest(r *Request) error {
	return nil
}

// GetRequests gets one or more requests from the database given a namespace as input.
func (s *Mock) GetRequests(ns string) (result []Request, err error) {
	return
}

// IncrementRequestRetry increases retries for a given request in the database.
func (s *Mock) IncrementRequestRetry(r *Request) (errs []error) {
	return
}

// GetUsers gets namespaces from the database.
func (s *Mock) GetUsers() (result []string, err error) {
	return
}

// GetRequestsCount gets requests count given a namespace.
func (s *Mock) GetRequestsCount(ns string) (result int, err error) {
	return
}

// DeleteRequest deletes a request from the database.
func (s *Mock) DeleteRequest(r *Request) error {
	return nil
}

// CreateStatistics creates an entry of Statistics in the database.
func (s *Mock) CreateStatistics(o *Statistics) error {
	return nil
}

// UpdateStatistics updates Statistics in the database.
func (s *Mock) UpdateStatistics(o *Statistics) error {
	return nil
}

// GetStatisticsUser gets Statistics of a namespace from the database.
func (s *Mock) GetStatisticsUser(ns string) (o *Statistics, notFound bool, err error) {
	return &Statistics{}, false, nil
}

// LogStats logs number of cached number of cached requests and statistics entries count.
func (s *Mock) LogStats() {
	dbLogger.Info("mock db stats")
}
