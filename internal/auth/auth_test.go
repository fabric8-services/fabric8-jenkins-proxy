package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// created from curl https://auth.prod-preview.openshift.io/api/token/keys?format=pem
var goodKeys = `
{
  "keys": [
    {
      "key": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAmCUKKr6LbO9eYKPyJB3kirzlSV8GpgmUcpTYoiDziLAxc2svPotyyUThWafEmCQ6OR46z60hZImaJcxDXyJJRKFjIA5LGA3ERbLdUGwLmJOE1ZZtcF5+c/pWqREsi5+E5VG/WN2I2ZNCqlQ+cfCqTwuQg+7Pr1jJvxm6Xf268r25kKRKD8uc3bRFaZNSmaK+G3KcZSiFyCAPgQcQU8ZN/9bLjGC8i5DQgCU5SYjvs1SmAUpcEkyVA5IQEuks0Wg0ZCogX50Bpx7FMV+Pbh4wE8tgOr/c8CHQAIGeyzRA2bi1FOcnUeA2OEhGKVv6KxCGsGkxrGpNxDXMYTaiVK2yzwIDAQAB",
      "kid": "zD-57oBFIMUZsAWqUnIsVu_x71VIjd1irGkGUOiTsL8"
    },
    {
      "key": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0eiOo86RtUEil7vbUQWW2rXDSNnGqXjXxxSAeCoBv5ecWGH7ZagmoFPfjSfjxT2qdQcTlRMkQzwsrztPbl3nKYkj/SCgxk0BPVQKWKJObe2bYIzxK8a+s2yvcS7LNrKV5+oWrOG6X46KQJbNh3SyIGy1o0lbBiOTNkQl984JgL9iuxJByz5azjU5W36IP6Hi2eJEYZxmve+I6jGF7PGx6sa7vNHcRDdxBU578IFw5CaMq+2zfza2AQwEH4LKb2bnTmbotgYPjhANgSl6oiqWehzFqeyDuZGJZAzkj31NLguNhmd6MbKCx5QmGJB7907cN1slvYcRA8UTKImfaK3+wQIDAQAB",
      "kid": "PE6-BEECZZpPZIVxLR6NinbthOHJcGqYrfl8v7v6BVA"
    }
  ]
}
`

var badKeys = `
{
  "keys": [
    {
      "key": "a bad key",
      "kid": "zD-bad-keyid"
    },
    {
      "key": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0eiOo86RtUEil7vbUQWW2rXDSNnGqXjXxxSAeCoBv5ecWGH7ZagmoFPfjSfjxT2qdQcTlRMkQzwsrztPbl3nKYkj/SCgxk0BPVQKWKJObe2bYIzxK8a+s2yvcS7LNrKV5+oWrOG6X46KQJbNh3SyIGy1o0lbBiOTNkQl984JgL9iuxJByz5azjU5W36IP6Hi2eJEYZxmve+I6jGF7PGx6sa7vNHcRDdxBU578IFw5CaMq+2zfza2AQwEH4LKb2bnTmbotgYPjhANgSl6oiqWehzFqeyDuZGJZAzkj31NLguNhmd6MbKCx5QmGJB7907cN1slvYcRA8UTKImfaK3+wQIDAQAB",
      "kid": "PE6-BEECZZpPZIVxLR6NinbthOHJcGqYrfl8v7v6BVA"
    }
  ]
}
`

type mockAuthService struct {
	calls     int
	Responses []string
	done      chan bool // closed when all responses are sent
	called    chan bool //
}

func (f *mockAuthService) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/vnd.publickeys+json")
	w.WriteHeader(200)
	fmt.Fprintln(w, f.Responses[f.calls])
	f.calls++

	// let the test know that all reponse is sent
	f.called <- true

	// when all responses are sent, cleanup and close channels
	// signal end/done
	if f.calls == len(f.Responses) {
		f.cleanup()
	}
}

func (f *mockAuthService) emptyCalled() {
	for {
		select {
		case <-f.called:
		default:
			return
		}
	}
}

func (f *mockAuthService) cleanup() {
	f.emptyCalled()
	close(f.called)
	close(f.done)
}

func newMockAuthService(responses []string) *mockAuthService {
	return &mockAuthService{
		Responses: responses,
		done:      make(chan bool),
		called:    make(chan bool, len(responses)),
	}

}

func TestClient_default_is_nil(t *testing.T) {
	authClient, err := DefaultClient()
	assert.NotNil(t, err, "auth default client is nil")
	assert.Nil(t, authClient, "auth default client is nil")
}

func TestClient_no_update_if_success(t *testing.T) {
	mockAuth := newMockAuthService([]string{goodKeys, goodKeys})
	ts := httptest.NewServer(mockAuth)
	defer ts.Close()
	defer mockAuth.cleanup() // not all responses will be sent so force shutdown

	assert.Equal(t, mockAuth.calls, 0, "setup validation")
	c := NewClient(ts.URL)
	<-mockAuth.called

	assert.Equal(t, mockAuth.calls, 1, "client failed to update keys")

	// must not make any more calls since the previous on succeeded
	c.updatePublicKeysOnce()
	assert.Equal(t, mockAuth.calls, 1, "client updated keys when it should not")
}

func TestClient_auto_populates_keys(t *testing.T) {
	mockAuth := newMockAuthService([]string{goodKeys})
	ts := httptest.NewServer(mockAuth)
	defer ts.Close()

	assert.Equal(t, mockAuth.calls, 0, "setup validation")

	NewClient(ts.URL)
	<-mockAuth.done
	assert.Equal(t, mockAuth.calls, 1, "client failed to update keys")
}

func TestClient_can_update_if_failed(t *testing.T) {
	mockAuth := newMockAuthService([]string{badKeys, badKeys, goodKeys})
	ts := httptest.NewServer(mockAuth)
	defer ts.Close()

	assert.Equal(t, mockAuth.calls, 0, "setup validation")
	c := NewClient(ts.URL)
	<-mockAuth.called
	assert.Equal(t, mockAuth.calls, 1, "client failed to update keys")

	err := c.updatePublicKeysOnce()
	<-mockAuth.called
	assert.NotNil(t, err, "auth did not return bad keys")

	err = c.updatePublicKeysOnce()
	assert.Nil(t, err, "auth did not return good keys")
	assert.Equal(t, mockAuth.calls, 3, "client failed to update keys")
}

func TestNewClient_multiple_updates(t *testing.T) {
	mockAuth := newMockAuthService([]string{badKeys, goodKeys})
	ts := httptest.NewServer(mockAuth)
	defer ts.Close()

	assert.Equal(t, mockAuth.calls, 0, "setup validation")
	c := NewClient(ts.URL)

	// first one will fail due to badKeys, and then one of the
	// following updates must get the good keys
	for i := 0; i < 10; i++ {
		go c.updatePublicKeysOnce()
	}

	<-mockAuth.done
	assert.Equal(t, mockAuth.calls, 2, "client didn't make expected calls")
}
