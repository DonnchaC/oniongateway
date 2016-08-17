package util

import (
	"net/http"
	"net/url"
)

type redirectError struct {
}

func (e *redirectError) Error() string {
	return "Do not follow redirect"
}

// IgnoreRedirect always returns special error type.
// Provide CheckRedirect=IgnoreRedirect to http.Client to prevent it
// from following redirects. Use IsRedirectError to detect such
// condition.
func IgnoreRedirect(req *http.Request, via []*http.Request) error {
	return &redirectError{}
}

// IsRedirectError returns if error of client.Get is the ignored redirect
func IsRedirectError(err error) bool {
	urlError, ok := err.(*url.Error)
	if ok {
		_, ok := urlError.Err.(*redirectError)
		if ok {
			return true
		}
	}
	return false
}
