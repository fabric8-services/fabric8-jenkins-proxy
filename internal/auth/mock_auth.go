package auth

import (
	"fmt"
	"net/url"
	"strings"
)

type MockAuth struct {
	URL string
}

func NewMockAuth(authURL string) *MockAuth {
	return &MockAuth{
		URL: authURL,
	}
}

// UIDFromToken returns user identity given a raw jwt token
func (c *MockAuth) UIDFromToken(accessToken string) (sub string, err error) {
	return "test_subject", nil
}

// OSOTokenForCluster returns Openshift online token given the clusterURL and raw JWT token
func (c *MockAuth) OSOTokenForCluster(clusterURL, accessToken string) (osoToken string, err error) {
	return "test_oso_token", nil
}

// CreateRedirectURL gets us the URI which we are supposed to use to logging in
// with fabric8-auth Client on giving auth Client URL and redirectURL as input.
func (c *MockAuth) CreateRedirectURL(to string) string {
	return fmt.Sprintf(
		"%s/api/login?redirect=%s",
		strings.TrimRight(c.URL, "/"), url.PathEscape(to))
}
