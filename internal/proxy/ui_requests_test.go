package proxy

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadToken(t *testing.T) {
	p := Proxy{}
	p.redirect = "http://redirect"
	req := httptest.NewRequest("GET", "http://proxy/v1?token_json=BADTOKEN", nil)
	w := httptest.NewRecorder()

	p.handleJenkinsUIRequest(w, req, proxyLogger)
	assert.Equal(t, 500, w.Code)
}
