package proxy

import (
	"testing"
	"time"

	"errors"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	cache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockWit struct {
	testCounter int // we will use this to count how many times this we get there
}

func (mw *mockWit) SearchCodebase(repo string) (*clients.WITInfo, error) {
	mw.testCounter++
	return &clients.WITInfo{
		OwnedBy: "Faker",
	}, errors.New("Reterror")
}

type fakeProxy struct {
	Proxy
}

func TestGetUserWithRetry(t *testing.T) {
	wit := &mockWit{testCounter: 0}

	p := &fakeProxy{}
	p.wit = wit
	numberofretry := 1

	logEntry := log.WithFields(log.Fields{"component": "proxy"})
	cache := cache.New(2*time.Millisecond, 1*time.Millisecond)
	p.TenantCache = cache

	_, err := p.getUserWithRetry("http://test", logEntry, numberofretry)
	assert.Error(t, err, "Faker")
	assert.Equal(t, numberofretry+1, wit.testCounter)
}
