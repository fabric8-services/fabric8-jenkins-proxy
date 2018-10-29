package cookiesutil

import (
	"net/http"
	"strings"
	"time"

	"github.com/satori/go.uuid"
)

const (
	// CookieJenkinsIdled stores name of the cookie through which we can check if service is idled
	CookieJenkinsIdled = "JenkinsIdled"

	// SessionCookie stores name of the session cookie of the service in question
	SessionCookie = "JSESSIONID"
)

type cookieFilterFn func(cookie *http.Cookie) bool

func CookieNames(cookies []*http.Cookie) []string {
	names := []string{}
	for _, cookie := range cookies {
		names = append(names, cookie.Name)
	}
	return names
}

func ExpireCookiesMatching(w http.ResponseWriter, r *http.Request, expire cookieFilterFn) {
	for _, cookie := range r.Cookies() {
		if expire(cookie) {
			ExpireCookie(w, cookie)
		}
		http.SetCookie(w, cookie)
	}
}

func ExpireCookie(w http.ResponseWriter, c *http.Cookie) {
	c.Expires = time.Unix(0, 0)
	http.SetCookie(w, c)
}

func IsSessionCookie(c *http.Cookie) bool {
	return strings.HasPrefix(c.Name, SessionCookie)
}

func IsIdledCookie(c *http.Cookie) bool {
	return c.Name == CookieJenkinsIdled
}

func IsSessionOrIdledCookie(c *http.Cookie) bool {
	return IsSessionCookie(c) || IsIdledCookie(c)
}

func anyCookie(_ *http.Cookie) bool {
	return true
}

func SetIdledCookie(w http.ResponseWriter) string {
	c := &http.Cookie{}
	c.Name = CookieJenkinsIdled
	c.Value = uuid.NewV4().String()
	http.SetCookie(w, c)
	return c.Value
}

// SetJenkinsCookies sets all cookies to w and returns the session cookie if it exists
func SetJenkinsCookies(w http.ResponseWriter, cookies []*http.Cookie) *http.Cookie {
	var jsessionCookie *http.Cookie

	for _, cookie := range cookies {
		http.SetCookie(w, cookie)
		if IsSessionCookie(cookie) {
			jsessionCookie = cookie
		}
	}
	return jsessionCookie
}
