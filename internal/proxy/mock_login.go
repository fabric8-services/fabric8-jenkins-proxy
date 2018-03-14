package proxy

import (
	"net/http"

	"errors"

	log "github.com/sirupsen/logrus"
)

type mockLogin struct {
	isLoggedIn   bool
	isTokenValid bool
	giveOSOToken bool
}

// loginJenkins always logs in
func (l *mockLogin) loginJenkins(pci CacheItem, osoToken string, requestLogEntry *log.Entry) (int, []*http.Cookie, error) {
	c := &http.Cookie{
		Name:  "JSESSIONID",
		Value: "Some Session ID",
	}
	return 200, []*http.Cookie{c}, nil
}

func (l *mockLogin) processToken(tokenData []byte, requestLogEntry *log.Entry, p *Proxy) (pci *CacheItem, osioToken string, err error) {

	if !l.isTokenValid {
		return nil, "", errors.New("Could not Process Token Properly")
	}
	pci = &CacheItem{
		ClusterURL: "https://api.free-stg.openshift.com/",
		NS:         "someNameSpace",
		Route:      "1b7d.free-stg.openshiftapps.com",
		Scheme:     "",
	}

	return pci, "OSIO_TOKEN", nil
}

func (l *mockLogin) GetOSOToken(authURL string, clusterURL string, token string) (osoToken string, err error) {
	if l.giveOSOToken {
		return "valid OSO Token", nil
	}

	return "", errors.New("Could not get valid OSO Token")
}
