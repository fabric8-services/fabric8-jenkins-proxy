package util

import (
	"net/http"
	"time"
)

// HTTPClient returns a client with a timeout
func HTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 15,
	}
}
