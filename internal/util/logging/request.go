package logging

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

func FormatHttpRequest(r *http.Request) string {
	var request []string

	url := fmt.Sprintf("%v %v", r.Method, r.URL)
	request = append(request, url)
	request = append(request, fmt.Sprintf("Host: %v", r.Host))

	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, " ")
		request = append(request, r.Form.Encode())
	}

	return strings.Join(request, "\n")
}

func RequestMethodAndURL(r *http.Request) string {
	return fmt.Sprintf("%v %v", r.Method, r.URL)
}

func RequestHeaders(r *http.Request) string {
	var result []string
	for name, headers := range r.Header {
		for _, h := range headers {
			result = append(result, fmt.Sprintf("-H %v: %v", name, h))
		}
	}

	// to ensure consistency we force an order
	sort.Strings(result)
	return strings.Join(result, " ")
}
