package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	log "github.com/sirupsen/logrus"
)

// JSONError creates a JSON response with a given error and writes it to ResponseWriter
func JSONError(w http.ResponseWriter, err error, requestLogEntry *log.Entry) {
	// log the error
	var location string
	if err != nil {
		programCounter, fileName, lineNumber, ok := runtime.Caller(1)
		if ok == false {
			log.Errorf("It was not possible to recover the information from runtime.Caller()")
		}

		location = fmt.Sprintf(" %s[%s:%d]", runtime.FuncForPC(programCounter).Name(), fileName, lineNumber)
	}

	requestLogEntry.WithFields(
		log.Fields{
			"location": location,
			"error":    err,
		}).Error("Error Handling proxy request.")

	// create error response
	w.WriteHeader(http.StatusInternalServerError)

	e := Error{
		Errors: make([]ErrorInfo, 1),
	}

	e.Errors[0] = ErrorInfo{
		Code:   fmt.Sprintf("%d", http.StatusInternalServerError),
		Detail: err.Error(),
	}

	errorBody, err := json.Marshal(e)
	if err != nil {
		requestLogEntry.Error(err)
	}
	w.Write(errorBody)

}

// Error represents list of error informations.
type Error struct {
	Errors []ErrorInfo
}

// ErrorInfo describes an HTTP error, consisting of HTTP status code and error detail.
type ErrorInfo struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}
