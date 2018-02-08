package util

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"

	log "github.com/sirupsen/logrus"

	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

const (
	jsonToken = "token_json"
)

func CreateJWTToken(env string, username string, password string) (string, error) {
	environment, ok := environments[env]

	if !ok {
		return "", errors.New(fmt.Sprintf("%s is not a valid environment", env))
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}

	var token string
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			log.Debugf("Redirect -> %v", req.URL)
			m, _ := url.ParseQuery(req.URL.RawQuery)
			if val, ok := m[jsonToken]; ok {
				token = val[0]
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", environment.authURL+"/api/login", nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("redirect", environment.redirectURL)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	postUrl, err := extractFormPostURL(resp)
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)
	form.Add("login", "Log+in")

	authResp, err := client.PostForm(postUrl, form)
	if err != nil {
		return "", err
	}
	defer authResp.Body.Close()

	if len(token) == 0 {
		return "", errors.New("Token could not be extracted.")
	}
	return token, nil
}

func extractFormPostURL(resp *http.Response) (string, error) {
	var postUrl string
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			for _, attr := range n.Attr {
				if attr.Key == "action" {
					postUrl = attr.Val
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return postUrl, nil
}
