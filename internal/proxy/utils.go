package proxy

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"runtime"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/util"
	log "github.com/sirupsen/logrus"
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

	pei := util.ErrorInfo{
		Code:   fmt.Sprintf("%d", http.StatusInternalServerError),
		Detail: err.Error(),
	}
	e := util.Error{
		Errors: make([]util.ErrorInfo, 1),
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
	// Jenkins takes around 4 mins to start so start with
	// 2 min, 1 min, ...  until 15 sec
	// see index.html that implements the exponential backoff retry
	data := struct {
		RetryMaxInterval int
		RetryMinInterval int
	}{45, 15}

	requestLogEntry.WithField("ns", ns).Debug("Templating index.html")
	err = tmplt.Execute(w, data)
	return
}

// constructRoute returns Jenkins route based on a specific pattern
func constructRoute(clusters map[string]string, clusterURL string, ns string) (string, string, error) {
	appSuffix := clusters[clusterURL]
	if len(appSuffix) == 0 {
		return "", "", fmt.Errorf("could not find entry for cluster %s", clusterURL)
	}
	route := fmt.Sprintf("jenkins-%s.%s", ns, clusters[clusterURL])
	return route, "https", nil
}
