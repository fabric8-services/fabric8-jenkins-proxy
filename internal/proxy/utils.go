package proxy

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"runtime"
)

//HandleError creates a JSON response with a given error and writes it to ResponseWriter
func (p *Proxy) HandleError(w http.ResponseWriter, err error, requestLogEntry *log.Entry) {
	// log the error
	location := ""
	if err != nil {
		pc, fn, line, _ := runtime.Caller(1)

		location = fmt.Sprintf(" %s[%s:%d]", runtime.FuncForPC(pc).Name(), fn, line)
	}

	requestLogEntry.WithFields(
		log.Fields{
			"location": location,
			"error":    err,
		}).Error("Error Handling proxy request request.")

	// create error response
	w.WriteHeader(http.StatusInternalServerError)

	pei := ErrorInfo{
		Code:   fmt.Sprintf("%d", http.StatusInternalServerError),
		Detail: err.Error(),
	}
	e := Error{
		Errors: make([]ErrorInfo, 1),
	}
	e.Errors[0] = pei

	eb, err := json.Marshal(e)
	if err != nil {
		requestLogEntry.Error(err)
	}
	w.Write(eb)
}

func (p *Proxy) processTemplate(w http.ResponseWriter, ns string, requestLogEntry *log.Entry) (err error) {
	tmplt, err := template.ParseFiles(p.indexPath)
	if err != nil {
		return
	}
	data := struct{ Retry int }{Retry: 15}
	requestLogEntry.WithField("ns", ns).Debug("Templating index.html")
	err = tmplt.Execute(w, data)

	return
}
