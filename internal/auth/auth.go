package auth

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util"
	"github.com/matryer/resync"
	log "github.com/sirupsen/logrus"
)

// AuthService talks to fabric8-auth for authentication and authorication
type AuthService interface {
	UIDFromToken(accessToken string) (sub string, err error)
	OSOTokenForCluster(clusterURL, accessToken string) (osoToken string, err error)
	CreateRedirectURL(to string) string
}

// Client used to access auth service
type Client struct {
	URL          string
	log          *log.Entry
	publicKeys   sync.Map
	updateWait   time.Duration // How long to wait after keys are fetched once
	singleUpdate resync.Once
}

// TokenJSON represents a JSON Web Token
type TokenJSON struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	Errors           []util.ErrorInfo
}

type Key struct {
	KID string `json:"kid"`
	Key string `json:"key"`
}

type KeyList struct {
	Keys []Key `json:"keys"`
}

var defaultClient AuthService

// SetDefaultClient set auth client that will be returned by auth.DefaultClient
func SetDefaultClient(c AuthService) {
	defaultClient = c
}

// DefaultClient returns default auth client
func DefaultClient() (AuthService, error) {
	if defaultClient != nil {
		return defaultClient, nil
	}

	return nil, fmt.Errorf("auth default client is nil")
}

// NewClient create a new auth client
func NewClient(authURL string) *Client {
	c := &Client{
		URL:        authURL,
		log:        log.WithField("component", "auth"),
		updateWait: 5 * time.Minute,
	}
	go c.updatePublicKeysOnce()
	return c
}

// UIDFromToken returns user identity given a raw jwt token
func (c *Client) UIDFromToken(accessToken string) (sub string, err error) {
	t, err := jwt.Parse(accessToken, c.publicKeyForToken)
	if err != nil {
		return
	}

	// to dump the token; uncomment below
	// c.log.Debug("==================================================")
	// c.log.Debugf("token: %s", spew.Sdump(t))
	// c.log.Debug("==================================================")

	claims, ok := t.Claims.(jwt.MapClaims)
	if ok && t.Valid {
		if claims["sub"].(string) == "" {
			err = fmt.Errorf("Could not find user id in token")
			return
		}
		sub = claims["sub"].(string)
	}
	return
}

// OSOTokenForCluster returns Openshift online token given the clusterURL and raw JWT token
func (c *Client) OSOTokenForCluster(clusterURL, accessToken string) (osoToken string, err error) {
	url := fmt.Sprintf("%s/api/token?for=%s", strings.TrimRight(c.URL, "/"), clusterURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := util.HTTPClient().Do(req)
	if err != nil {
		return
	}

	tj := &TokenJSON{}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, tj)
	if err != nil {
		return
	}

	if len(tj.Errors) > 0 {
		err = fmt.Errorf(tj.Errors[0].Detail)
		return
	}

	if len(tj.AccessToken) > 0 {
		osoToken = tj.AccessToken
	} else {
		err = fmt.Errorf("OSO access token empty for %s", c.URL)
	}
	return
}

// CreateRedirectURL gets us the URI which we are supposed to use to logging in
// with fabric8-auth Client on giving auth Client URL and redirectURL as input.
func (c *Client) CreateRedirectURL(to string) string {
	return fmt.Sprintf(
		"%s/api/login?redirect=%s",
		strings.TrimRight(c.URL, "/"), url.PathEscape(to))
}

func (c *Client) rsaPublicKeyForID(kid string) (*rsa.PublicKey, error) {
	if val, ok := c.publicKeys.Load(kid); ok {
		return val.(*rsa.PublicKey), nil
	}

	if err := c.updatePublicKeysOnce(); err != nil {
		return nil, fmt.Errorf("no public key for key-id: %q; err: %s", kid, err)
	}

	if val, ok := c.publicKeys.Load(kid); ok {
		return val.(*rsa.PublicKey), nil
	}

	return nil, fmt.Errorf("no public key for key-id: %q", kid)
}

func (c *Client) publicKeyForToken(token *jwt.Token) (interface{}, error) {

	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	kid, ok := token.Header["kid"]
	if !ok {
		return nil, fmt.Errorf("missing mandatory kid header: %v", token.Header)
	}

	if pk, err := c.rsaPublicKeyForID(kid.(string)); err == nil {
		return pk, nil
	}

	return nil, fmt.Errorf("no public key found for kid: %q", kid)
}

// updatePublicKeysOnce ensures that keys would be fetched only once
// even if the func is called by several go routines at once. Keys can
// however be fetched again after the waiting period expires or if
// updatePublicKeys generates an error
func (c *Client) updatePublicKeysOnce() error {
	var err error
	c.singleUpdate.Do(func() {
		err = c.updatePublicKeys()
	})

	if err != nil {
		c.log.Debug("An error occurred when fetching keys - reset now ")
		c.singleUpdate.Reset()
	} else {
		c.log.Debugf("Will reset fetching keys after %v", c.updateWait)
		time.AfterFunc(c.updateWait, c.singleUpdate.Reset)
	}
	return err
}

func (c *Client) updatePublicKeys() error {
	tokenURL := strings.TrimRight(c.URL, "/") + "/api/token/keys?format=pem"

	c.log.Infof("Fetching public keys from %s", tokenURL)
	resp, err := util.HTTPClient().Get(tokenURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Got status code %s (%d) from %s",
			resp.Status, resp.StatusCode, c.URL)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	keys := &KeyList{}

	if err = json.Unmarshal(body, keys); err != nil {
		return err
	}

	for _, key := range keys.Keys {
		rsaPubKey, e := jwt.ParseRSAPublicKeyFromPEM(makePEM(key.Key))
		if e != nil {
			err = e
			c.log.WithFields(log.Fields{"kid": key.KID, "key": key.Key}).
				Warnf("failed to parse key; error: %v", err)
			continue
		}
		c.publicKeys.Store(key.KID, rsaPubKey)
	}

	return err
}

func makePEM(key string) []byte {
	return []byte(fmt.Sprintf(`
-----BEGIN PUBLIC KEY-----
%s
-----END PUBLIC KEY-----
`, key))
}
