// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/storage/store.go

// Package storage is a generated GoMock package.
package storage

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockStore is a mock of Store interface
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// CreateRequest mocks base method
func (m *MockStore) CreateRequest(r *Request) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRequest", r)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateRequest indicates an expected call of CreateRequest
func (mr *MockStoreMockRecorder) CreateRequest(r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRequest", reflect.TypeOf((*MockStore)(nil).CreateRequest), r)
}

// GetRequests mocks base method
func (m *MockStore) GetRequests(ns string) ([]Request, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRequests", ns)
	ret0, _ := ret[0].([]Request)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRequests indicates an expected call of GetRequests
func (mr *MockStoreMockRecorder) GetRequests(ns interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRequests", reflect.TypeOf((*MockStore)(nil).GetRequests), ns)
}

// IncrementRequestRetry mocks base method
func (m *MockStore) IncrementRequestRetry(r *Request) []error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IncrementRequestRetry", r)
	ret0, _ := ret[0].([]error)
	return ret0
}

// IncrementRequestRetry indicates an expected call of IncrementRequestRetry
func (mr *MockStoreMockRecorder) IncrementRequestRetry(r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IncrementRequestRetry", reflect.TypeOf((*MockStore)(nil).IncrementRequestRetry), r)
}

// GetUsers mocks base method
func (m *MockStore) GetUsers() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUsers")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUsers indicates an expected call of GetUsers
func (mr *MockStoreMockRecorder) GetUsers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUsers", reflect.TypeOf((*MockStore)(nil).GetUsers))
}

// GetRequestsCount mocks base method
func (m *MockStore) GetRequestsCount(ns string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRequestsCount", ns)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRequestsCount indicates an expected call of GetRequestsCount
func (mr *MockStoreMockRecorder) GetRequestsCount(ns interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRequestsCount", reflect.TypeOf((*MockStore)(nil).GetRequestsCount), ns)
}

// DeleteRequest mocks base method
func (m *MockStore) DeleteRequest(r *Request) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRequest", r)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRequest indicates an expected call of DeleteRequest
func (mr *MockStoreMockRecorder) DeleteRequest(r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRequest", reflect.TypeOf((*MockStore)(nil).DeleteRequest), r)
}

// CreateStatistics mocks base method
func (m *MockStore) CreateStatistics(o *Statistics) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateStatistics", o)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateStatistics indicates an expected call of CreateStatistics
func (mr *MockStoreMockRecorder) CreateStatistics(o interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateStatistics", reflect.TypeOf((*MockStore)(nil).CreateStatistics), o)
}

// UpdateStatistics mocks base method
func (m *MockStore) UpdateStatistics(o *Statistics) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatistics", o)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateStatistics indicates an expected call of UpdateStatistics
func (mr *MockStoreMockRecorder) UpdateStatistics(o interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStatistics", reflect.TypeOf((*MockStore)(nil).UpdateStatistics), o)
}

// GetStatisticsUser mocks base method
func (m *MockStore) GetStatisticsUser(ns string) (*Statistics, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStatisticsUser", ns)
	ret0, _ := ret[0].(*Statistics)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetStatisticsUser indicates an expected call of GetStatisticsUser
func (mr *MockStoreMockRecorder) GetStatisticsUser(ns interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStatisticsUser", reflect.TypeOf((*MockStore)(nil).GetStatisticsUser), ns)
}

// LogStats mocks base method
func (m *MockStore) LogStats() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "LogStats")
}

// LogStats indicates an expected call of LogStats
func (mr *MockStoreMockRecorder) LogStats() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LogStats", reflect.TypeOf((*MockStore)(nil).LogStats))
}