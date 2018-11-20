package reverseproxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

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
	OnError         func(http.ResponseWriter, *http.Request, int) error
	Logger          *log.Entry
}

// NewReverseProxy returns an instance of reverse proxy on passing redirect url,
// response timeout, session validity flag and a logger object
func NewReverseProxy(redirectURL url.URL, responseTimeout time.Duration, onError func(http.ResponseWriter, *http.Request, int) error, logger *log.Entry) *ReverseProxy {
	return &ReverseProxy{
		RedirectURL:     redirectURL,
		ResponseTimeout: responseTimeout,
		OnError:         onError,

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
	})

	ctx, cancel := context.WithTimeout(req.Context(), rp.ResponseTimeout)
	defer cancel()

	outreq := req.WithContext(ctx)

	rr := newResponseRecorder(rw, logger)
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(rr, outreq)

	if rr.err != nil {
		logger.Warnf("Error %q - code: %d", rr.err, rr.statusCode)

		err := rp.OnError(rw, req, rr.statusCode)
		if err != nil {
			logger.Error(err)
		}

		http.Redirect(rw, req, rp.RedirectURL.String(), http.StatusFound)

	}
}
