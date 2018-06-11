package reverseproxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// ReverseProxy is a HTTP Handler that takes an incoming request and
// sends it to another server (jenkins), proxying the response back to
// the client if it is received within a timeout (responseTimeout).
// In case the handler does not recieve any response from the server
// within the timeout, a 302 to the same URL is sent back to the client.
type ReverseProxy struct {
	RedirectURL     url.URL
	Logger          *log.Entry
	ResponseTimeout time.Duration
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
		http.Redirect(rw, req, rp.RedirectURL.String(), http.StatusFound)
	}
}
