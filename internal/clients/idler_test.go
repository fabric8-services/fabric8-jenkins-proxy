package clients

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	req, _ := newRequest("http://blah")
	assert.NotEmpty(t, req.Header.Get("X-Request-ID"), "X-Request-ID wasn't generated")

	_, err := newRequest("blah://foo\\/\\/bar")
	assert.Error(t, err, "Should have generated a URL error")
}
