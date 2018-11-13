package reverseproxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	cookiesutil "github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy/cookies"

	log "github.com/sirupsen/logrus"
)

// ReverseProxy is an HTTP Handler that takes an incoming request and
// sends it to another server (jenkins), proxying the response back to
// the client if it is received within a timeout (responseTimeout).
// In case the handler does not recieve any response from the server
// within the timeout, a 302 to the same URL is sent back to the client.
type ReverseProxy struct {
	RedirectURL     url.URL
	ResponseTimeout time.Duration
	IsValidSession  bool

	Logger *log.Entry
}

// NewReverseProxy returns an instance of reverse proxy on passing redirect url,
// response timeout, session validity flag and a logger object
func NewReverseProxy(redirectURL url.URL, responseTimeout time.Duration, isSessionValid bool, logger *log.Entry) *ReverseProxy {
	return &ReverseProxy{
		RedirectURL:     redirectURL,
		ResponseTimeout: responseTimeout,
		IsValidSession:  isSessionValid,

		Logger: logger,
	}
}

func director(req *http.Request) {
	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
}

func (rp *ReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	logger := rp.Logger.WithFields(log.Fields{
		"redirect_url":     rp.RedirectURL.String(),
		"request_url":      req.URL.String(),
		"response_timeout": rp.ResponseTimeout,
		"is_valid_session": rp.IsValidSession,
	})

	ctx, cancel := context.WithTimeout(req.Context(), rp.ResponseTimeout)
	defer cancel()

	outreq := req.WithContext(ctx)

	rr := newResponseRecorder(rw, logger)
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(rr, outreq)

	if rr.err != nil {
		logger.Warnf("Error %q - code: %d", rr.err, rr.statusCode)

		if rr.statusCode == http.StatusForbidden {
			fmt.Println("YES HERE")
			var err error
			rp.IsValidSession, err = checkSessionValidity(req, rp)
			if err != nil {
				logger.Error(err)
			}
		}
		http.Redirect(rw, req, rp.RedirectURL.String(), http.StatusFound)
	}
}

func checkSessionValidity(req *http.Request, rp *ReverseProxy) (bool, error) {

	requestURL := req.URL
	cookies := req.Cookies()

	jenkinsURL := fmt.Sprintf("%s://%s", requestURL.Scheme, requestURL.Host)

	for _, cookie := range cookies {
		if cookiesutil.IsSessionCookie(cookie) {

			checkSessionValidityWithCookie := func(cookie *http.Cookie, rp *ReverseProxy) (bool, error) {
				ctx, cancel := context.WithTimeout(req.Context(), rp.ResponseTimeout)
				defer cancel()

				r, _ := http.NewRequest("GET", jenkinsURL, nil)
				r.AddCookie(cookie)
				r = r.WithContext(ctx)

				c := &http.Client{
					Timeout: rp.ResponseTimeout,
				}

				resp, err := c.Do(r)
				if err != nil {
					return false, err
				}
				defer resp.Body.Close()
				switch resp.StatusCode {
				case http.StatusOK:
					return true, err
				case http.StatusForbidden:
					return false, err
				default:
					err = fmt.Errorf("received unexpected error, code: %s", resp.Status)
					return false, err
				}
			}

			return checkSessionValidityWithCookie(cookie, rp)
		}
	}

	return false, nil
}
