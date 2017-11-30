package storage

import (
	"net/url"
	"bytes"
	"io"
	"encoding/json"
	"net/http"
	"io/ioutil"
	uuid "github.com/satori/go.uuid"
)

const (
	ErrorFailedDelete = "Failed to delete request for %s (%s): %s"
)

type Request struct {
	ID uuid.UUID `sql:"type:uuid" gorm:"primary_key"` // This is the ID PK field
	Method string
	Headers []byte
	Payload []byte
	Host string
	Scheme string
	Path string
	Namespace string
	Retries int
}

func NewRequest(r *http.Request, ns string, body []byte) (*Request, error) {
	h, err := json.Marshal(r.Header)
	if err != nil {
		return nil, err
	}

	return &Request{
		ID: uuid.NewV4(),
		Method: r.Method,
		Headers: h,
		Payload: body,
		Host: r.Host,
		Scheme: r.URL.Scheme,
		Path: r.URL.Path,
		Namespace: ns,
		Retries: 0,
	}, nil
}

func (m Request) TableName() string {
	return "requests"
}

func (m Request) GetHeaders() (result map[string][]string, err error) {
	result = make(map[string][]string)
	err = json.Unmarshal(m.Headers, &result) 
	return
}

func (m Request) GetPayloadReader() (io.ReadCloser) {
	return ioutil.NopCloser(bytes.NewReader(m.Payload))
}

func (m Request) GetHTTPRequest() (r *http.Request, err error) {
	u := url.URL{}
	u.Host = m.Host
	u.Scheme = m.Scheme
	u.Path = m.Path
	r, err = http.NewRequest(m.Method, u.String(), m.GetPayloadReader())
	if err != nil {
		return
	}

	h, err := m.GetHeaders()
	if err != nil {
		return
	}
	for k, lv := range h {
		for _, v := range lv {
			r.Header.Add(k, v)
		}
	}
	
	return
}