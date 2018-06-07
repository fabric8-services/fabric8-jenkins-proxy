package logging

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// FormatHTTPRequestWithSeparator returns a formatted and readable string containing information of given HTTP request, separated by given separated.
func FormatHTTPRequestWithSeparator(r *http.Request, separator string) string {
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

	return strings.Join(request, separator)
}

// FormatHTTPRequest returns a formatted and readable string containing information of given HTTP request.
func FormatHTTPRequest(r *http.Request) string {
	return FormatHTTPRequestWithSeparator(r, "\n")
}

// RequestMethodAndURL return string containing method(GET, POST, PATCH etc) and string given an HTTP request.
func RequestMethodAndURL(r *http.Request) string {
	return fmt.Sprintf("%v %v", r.Method, r.URL)
}

// RequestHeaders returns a string containing request header for the given HTTP request.
func RequestHeaders(r *http.Request) string {
	var result []string
	for name, headers := range r.Header {
		for _, h := range headers {
			result = append(result, fmt.Sprintf("-H %v: %v", name, h))
		}
	}

	// to ensure consistency we force an order.
	sort.Strings(result)
	return strings.Join(result, " ")
}
