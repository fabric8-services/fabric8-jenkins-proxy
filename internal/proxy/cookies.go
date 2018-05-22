package proxy

import (
	"github.com/satori/go.uuid"
	"net/http"
	"strings"
	"time"
)

const (
	// CookieJenkinsIdled stores name of the cookie through which we can check if service is idled
	CookieJenkinsIdled = "JenkinsIdled"

	// SessionCookie stores name of the session cookie of the service in question
	SessionCookie = "JSESSIONID"
)

type cookieFilterFn func(cookie *http.Cookie) bool

func cookieNames(cookies []*http.Cookie) []string {
	names := []string{}
	for _, cookie := range cookies {
		names = append(names, cookie.Name)
	}
	return names
}

func expireCookiesMatching(w http.ResponseWriter, r *http.Request, expire cookieFilterFn) {
	for _, cookie := range r.Cookies() {
		if expire(cookie) {
			expireCookie(w, cookie)
		}
		http.SetCookie(w, cookie)
	}
}

func expireCookie(w http.ResponseWriter, c *http.Cookie) {
	c.Expires = time.Unix(0, 0)
	http.SetCookie(w, c)
}

func isSessionCookie(c *http.Cookie) bool {
	return strings.HasPrefix(c.Name, SessionCookie)
}

func isIdledCookie(c *http.Cookie) bool {
	return c.Name == CookieJenkinsIdled
}

func isSessionOrIdledCookie(c *http.Cookie) bool {
	return isSessionCookie(c) || isIdledCookie(c)
}

func anyCookie(_ *http.Cookie) bool {
	return true
}

func setIdledCookie(w http.ResponseWriter) string {
	c := &http.Cookie{}
	c.Name = CookieJenkinsIdled
	c.Value = uuid.NewV4().String()
	http.SetCookie(w, c)
	return c.Value
}

// set all cookies to w and returns the session cookie if it exists
func setJenkinsCookies(w http.ResponseWriter, cookies []*http.Cookie) *http.Cookie {
	var jsessionCookie *http.Cookie

	for _, cookie := range cookies {
		http.SetCookie(w, cookie)
		if isSessionCookie(cookie) {
			jsessionCookie = cookie
		}
	}
	return jsessionCookie
}
