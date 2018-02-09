package logging

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"net/url"
	"testing"
)

const (
	getRequestExpectation = `GET http://example.com/foo?bar=baz
Host: example.com
authentication: Bearer 123`

	postRequestExpectation = `POST http://example.com
Host: example.com
content-type: application/x-www-form-urlencoded
 
foo=bar`
)

func Test_format_get_request(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/foo?bar=baz", nil)
	req.Header.Set("Authentication", "Bearer 123")

	requestAsString := FormatHttpRequest(req)
	assert.Equal(t, getRequestExpectation, requestAsString, "Format does not match.")

	requestMethodAndURL := RequestMethodAndURL(req)
	assert.Equal(t, "GET http://example.com/foo?bar=baz", requestMethodAndURL, "Format does not match.")
}

func Test_format_post_request(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.com", nil)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	form := url.Values{}
	form.Add("foo", "bar")
	req.PostForm = form

	requestAsString := FormatHttpRequest(req)
	assert.Equal(t, postRequestExpectation, requestAsString, "Format does not match.")

	requestMethodAndURL := RequestMethodAndURL(req)
	assert.Equal(t, "POST http://example.com", requestMethodAndURL, "Format does not match.")
}

func Test_request_headers(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/foo?bar=baz", nil)
	req.Header.Set("Foo", "Bar")
	req.Header.Set("Authentication", "Bearer 123")

	requestHeaders := RequestHeaders(req)
	assert.Equal(t, "-H Authentication: Bearer 123 -H Foo: Bar", requestHeaders, "Format does not match.")
}
