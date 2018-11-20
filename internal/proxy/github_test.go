package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestWebhookRequestHasGHHeader(t *testing.T) {

	p := NewMockProxy("")

	req := httptest.NewRequest("GET", "http://proxy", nil)

	req.Header.Add("User-Agent", "GitHub-Hookshot"+uuid.NewV4().String())
	assert.True(t, p.isGitHubRequest(req), "p.isGitHubRequest(req) should evaluate to true")
}

func TestGHWebHookRequest(t *testing.T) {

	p := NewMockProxy("")

	gh := []byte(`{
		"repository": {
			"name": "test-repo",
			"full_name": "test-username/test-repo",
			"git_url": "git://github.com/test-username/test-repo.git",
			"clone_url": "https://github.com/test-username/test-repo.git"
		}
	}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://proxy", bytes.NewBuffer(gh))
	req.Header.Add("User-Agent", "GitHub-Hookshot"+uuid.NewV4().String())

	p.handleGitHubRequest(w, req, proxyLogger)
	_, ok := p.TenantCache.Get("https://github.com/test-username/test-repo.git")
	assert.True(t, ok, "An entry should have been created in tenant cache with repo url as key")
	// writes status http.StatusAccepted on accepting and storing the request
	assert.Equal(t, w.Code, http.StatusAccepted)
}
