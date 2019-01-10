package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/wit"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestWebhookRequestHasGHHeader(t *testing.T) {
	p := NewMock(idler.Idled, wit.DefaultMockOwner)
	req := httptest.NewRequest("GET", "http://proxy", nil)
	req.Header.Add("User-Agent", "GitHub-Hookshot"+uuid.NewV4().String())
	assert.True(t, p.isGitHubRequest(req), "p.isGitHubRequest(req) should evaluate to true")
}
func TestGHWebHookRequestJenkinsIdled(t *testing.T) {
	testGHWebHook(t, idler.Idled)
}

func TestGHWebHookRequestJenkinsRunning(t *testing.T) {
	testGHWebHook(t, idler.Running)
}

func testGHWebHook(t *testing.T, jenkinsState idler.PodState) {
	p := NewMock(jenkinsState, wit.DefaultMockOwner)

	w, req := getGHWebHookRecorderAndRequest()
	ns, okToForward := p.handleGitHubRequest(w, req, proxyLogger)

	assert.Equal(t, ns, "namespace-jenkins")

	_, ok := p.TenantCache.Get("https://github.com/test-username/test-repo.git")
	assert.True(t, ok, "An entry should have been created in tenant cache with repo url as key")

	if jenkinsState == idler.Running {
		assert.True(t, okToForward, "It should be ok to forward, because state of jenkins is running")
		// writes status http.StatusOK when request is ok to forward
		assert.Equal(t, http.StatusOK, w.Code)
	} else if jenkinsState == idler.Idled {
		assert.False(t, okToForward, "It should not be ok to forward, because state of jenkins is idled")
		// writes status http.StatusAccepted on accepting and storing the request
		assert.Equal(t, http.StatusAccepted, w.Code)
	}
}

func TestGHWebHookUnableToGetUser(t *testing.T) {
	p := NewMock(idler.Idled, "")

	w, req := getGHWebHookRecorderAndRequest()
	ns, okToForward := p.handleGitHubRequest(w, req, proxyLogger)

	// Since we failed to get user because wit.OwnedBy being empty,
	// we should have go an empty namespace and it should not be
	// ok to forward
	assert.Equal(t, ns, "")

	_, ok := p.TenantCache.Get("https://github.com/test-username/test-repo.git")
	assert.False(t, ok, `An entry should not have been created in tenant cache with repo url as key,
		since we failed to get the namespace associated with this repo url`)

	assert.False(t, okToForward, "It should not be ok to forward, because state of jenkins is idled")

	assert.Equal(t, http.StatusInternalServerError, w.Code)

}

func getGHWebHookRecorderAndRequest() (*httptest.ResponseRecorder, *http.Request) {
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

	return w, req
}
