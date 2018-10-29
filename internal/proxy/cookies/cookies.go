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

// CookieNames returns a string array with names of cookies given a cookie array
func CookieNames(cookies []*http.Cookie) []string {
	names := []string{}
	for _, cookie := range cookies {
		names = append(names, cookie.Name)
	}
	return names
}

// ExpireCookiesMatching expires all cookies that evaluates `expire` function into true
func ExpireCookiesMatching(w http.ResponseWriter, r *http.Request, expire cookieFilterFn) {
	for _, cookie := range r.Cookies() {
		if expire(cookie) {
			ExpireCookie(w, cookie)
		}
		http.SetCookie(w, cookie)
	}
}

// ExpireCookie expires a given cookie
func ExpireCookie(w http.ResponseWriter, c *http.Cookie) {
	c.Expires = time.Unix(0, 0)
	http.SetCookie(w, c)
}

// IsSessionCookie returns true if given cookie was set to manage jenkins session
func IsSessionCookie(c *http.Cookie) bool {
	return strings.HasPrefix(c.Name, SessionCookie)
}

// IsIdledCookie returns true if a given cookie was set to tell if jenkins is running or not
func IsIdledCookie(c *http.Cookie) bool {
	return c.Name == CookieJenkinsIdled
}

// IsSessionOrIdledCookie return true if given cookie was either set to manage jenkins session
// or to tell if jenkins is running or not
func IsSessionOrIdledCookie(c *http.Cookie) bool {
	return IsSessionCookie(c) || IsIdledCookie(c)
}

func anyCookie(_ *http.Cookie) bool {
	return true
}

// SetIdledCookie set a cookie to indicate that jenkins is idled for namespace stored
// in cache with cookie value as cache key
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
