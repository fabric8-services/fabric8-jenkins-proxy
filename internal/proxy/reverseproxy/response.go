package reverseproxy

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	err         error
	wroteHeader bool
	log         *log.Entry
}

func newResponseRecorder(w http.ResponseWriter, logger *log.Entry) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		log:            logger.WithField("response_recorder", "true"),
		statusCode:     http.StatusOK,
		err:            nil,
		wroteHeader:    false,
	}
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.log.Infof("Write Header called with %d", code)

	if rr.wroteHeader {
		rr.log.Warnf("Multiple Write Header %d called; SKIPPED", code)
		return
	}

	rr.statusCode = code
	rr.wroteHeader = true

	// refer httputils/reverseproxy.go ServerHTTP which sets the
	// StatusBadGateway when transport.RoundTrip returns an error
	if code == http.StatusBadGateway ||
		code == http.StatusGatewayTimeout ||
		code == http.StatusServiceUnavailable {

		// set err for bad gateway
		rr.err = fmt.Errorf("bad gateway error: %d", code)
		rr.log.Warnf("Error BadGateway WriteHeader %v - SKIPPED", code)
		return
	}

	if code == http.StatusForbidden {
		rr.err = fmt.Errorf("forbidden error: %d", code)
		rr.log.Warnf("Error Forbidden WriteHeader %v - SKIPPED", code)
		return
	}

	rr.log.Infof("WriteHeader %d - OK", code)
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(stuff []byte) (int, error) {
	if rr.err == nil {
		rr.log.Infof("Write stuff len: %d - OK", len(stuff))
		return rr.ResponseWriter.Write(stuff)
	}

	// fake that it wrote it
	rr.log.Warnf("Faking write len: %d - SKIPPED as err is set", len(stuff))
	return len(stuff), nil
}
