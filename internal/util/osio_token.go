package util

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
)

// User represents an user of OSIO.
type User struct {
	Data Data `json:"data"`
}

// Data contains user attributes.
type Data struct {
	Attributes Attributes `json:"attributes"`
}

// Attributes consists of user attributes such as username, email and fullname.
type Attributes struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
}

// CreateOSIOToken creates OSIO token given inputs such as user identity, environment, etc.
func CreateOSIOToken(env string, uuid string, key string, keyID string, valid int64, session string) (string, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(key))
	if err != nil {
		return "", fmt.Errorf("unable to parse private key file: %s", err)
	}

	user, err := loadUser(env, uuid)
	if err != nil {
		return "", fmt.Errorf("unable to GET user for uuid '%s': %s", uuid, err)
	}

	token, err := generateToken(env, privateKey, keyID, user, uuid, valid, session)
	if err != nil {
		return "", fmt.Errorf("unable to generate token for uuid '%s': %s", uuid, err)
	}

	return token, nil
}

// generateToken generates the JWT token using the specified private key.
// See also https://self-issued.info/docs/draft-ietf-oauth-json-web-token.html.
func generateToken(env string, key *rsa.PrivateKey, keyID string, user User, userID string, valid int64, session string) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)

	kid := ""
	if len(keyID) > 0 {
		kid = keyID
	} else {
		kid = environments[env].privateKeyID
	}
	token.Header["kid"] = kid

	// JWT ID, needs to be unique
	token.Claims.(jwt.MapClaims)["jti"] = uuid.NewV4().String()

	// Deal with various time related fields
	now := time.Now().Unix()
	token.Claims.(jwt.MapClaims)["iat"] = now            // Token issue time
	token.Claims.(jwt.MapClaims)["exp"] = now + 60*valid // Token expiry time

	token.Claims.(jwt.MapClaims)["iss"] = "https://sso.openshift.io/auth/realms/fabric8"
	token.Claims.(jwt.MapClaims)["aud"] = "fabric8-online-platform"
	token.Claims.(jwt.MapClaims)["typ"] = "Bearer"
	token.Claims.(jwt.MapClaims)["azp"] = "fabric8-online-platform"
	token.Claims.(jwt.MapClaims)["sub"] = userID

	token.Claims.(jwt.MapClaims)["acr"] = "1"
	token.Claims.(jwt.MapClaims)["approved"] = "true"
	token.Claims.(jwt.MapClaims)["company"] = ""
	token.Claims.(jwt.MapClaims)["name"] = user.Data.Attributes.FullName
	token.Claims.(jwt.MapClaims)["preferred_username"] = user.Data.Attributes.Username
	token.Claims.(jwt.MapClaims)["given_name"] = ""
	token.Claims.(jwt.MapClaims)["family_name"] = ""
	token.Claims.(jwt.MapClaims)["email"] = user.Data.Attributes.Email

	if len(session) > 0 {
		token.Claims.(jwt.MapClaims)["session_state"] = session
	}

	token.Claims.(jwt.MapClaims)["allowed-origins"] = []string{
		environments[env].osioURL,
		environments[env].authURL,
		environments[env].apiURL,
	}

	realmAccess := make(map[string]interface{})
	realmAccess["roles"] = []string{"uma_authorization"}
	token.Claims.(jwt.MapClaims)["realm_access"] = realmAccess

	resourceAccess := make(map[string]interface{})
	broker := make(map[string]interface{})
	broker["roles"] = []string{"read-token"}
	resourceAccess["broker"] = broker

	account := make(map[string]interface{})
	account["roles"] = []string{"manage-account", "manage-account-links", "view-profile"}
	resourceAccess["account"] = account

	token.Claims.(jwt.MapClaims)["resource_access"] = resourceAccess

	tokenStr, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

func loadUser(env string, uuid string) (User, error) {
	var user User
	req, err := http.NewRequest("GET", environments[env].authURL+"/api/users/"+uuid, nil)
	if err != nil {
		return user, err
	}
	res, err := HTTPClient().Do(req)
	if err != nil {
		return user, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return user, err
	}
	if res.StatusCode != http.StatusOK {
		return user, errors.New("Status is not 200 OK: " + res.Status)
	}
	err = json.Unmarshal(body, &user)

	return user, err
}
