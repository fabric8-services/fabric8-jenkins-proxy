package util

// Error represents list of error informations.
type Error struct {
	Errors []ErrorInfo
}

// ErrorInfo describes an HTTP error, consisting of HTTP status code and error detail.
type ErrorInfo struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}
