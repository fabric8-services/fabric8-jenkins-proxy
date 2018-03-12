package proxy

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
)

// TokenJSON represents a JSON Web Token
type TokenJSON struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	Errors           []ErrorInfo
}

// GetTokenUID gets user identity on giving raw JWT token and public key of auth service as input.
func GetTokenUID(token string, pk *rsa.PublicKey) (sub string, err error) {
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return pk, nil
	})
	if err != nil {
		return
	}

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

// GetOSOToken returns Openshift online token on giving raw JWT token, cluster URL and auth service url as input.
func GetOSOToken(authURL string, clusterURL string, token string) (osoToken string, err error) {
	url := fmt.Sprintf("%s/api/token?for=%s", strings.TrimRight(authURL, "/"), clusterURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	c := http.DefaultClient

	resp, err := c.Do(req)
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
		err = fmt.Errorf("OSO access token empty for %s", authURL)
	}
	return
}

// GetPublicKey gets public key of keycloak realm which Proxy service is using.
func GetPublicKey(kcURL string) (pk *rsa.PublicKey, err error) {
	resp, err := http.Get(fmt.Sprintf("%s/auth/realms/fabric8/", strings.TrimRight(kcURL, "/")))
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Got status code %s (%d) from %s", resp.Status, resp.StatusCode, kcURL)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	type realmInfo struct {
		PublicKey string `json:"public_key"`
	}

	ri := &realmInfo{}
	err = json.Unmarshal(body, ri)
	if err != nil {
		return
	}

	pk, err = jwt.ParseRSAPublicKeyFromPEM([]byte(fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----", ri.PublicKey)))
	if err != nil {
		return
	}

	return
}

// GetAuthURI gets us the URI which we are supposed to use to logging in with fabric8-auth service on
// giveing auth service URL and redirectURL as input.
func GetAuthURI(authURL string, redirectURL string) string {
	return fmt.Sprintf("%s/api/login?redirect=%s", strings.TrimRight(authURL, "/"), url.PathEscape(redirectURL))
}
