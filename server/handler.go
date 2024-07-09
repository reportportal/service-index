package server

import (
	errs "errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// stackTracer is an error containing stack trace
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// HTTPError represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type HTTPError interface {
	error
	Status() int
}

// StatusError represents an error with an associated HTTP status code.
type StatusError struct {
	Code int
	Err  error
}

// NewStatusError creates new StatusError
func NewStatusError(code int, err string) StatusError {
	return StatusError{code, errs.New(err)}
}

// ToStatusError creates new StatusError
func ToStatusError(code int, err error) StatusError {
	return StatusError{code, err}
}

// Error allows StatusError to satisfy the error interface.
func (se StatusError) Error() string {
	return se.Err.Error()
}

// Status returns our HTTP status code.
func (se StatusError) Status() int {
	return se.Code
}

// StackTrace returns stacktrace of child error or nil
func (se StatusError) StackTrace() errors.StackTrace {
	if se, ok := se.Err.(stackTracer); ok {
		return se.StackTrace()
	}
	return nil
}

// The Handler struct that takes a configured Env and a function matching
// our useful signature.
type Handler struct {
	H func(w http.ResponseWriter, r *http.Request) error
}

// ServeHTTP allows our Handler type to satisfy http.Handler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.H(w, r)
	if err != nil {
		if err, ok := err.(stackTracer); ok {

			stackTrace := make([]string, len(err.StackTrace()))
			for i, f := range err.StackTrace() {
				stackTrace[i] = fmt.Sprintf("%+s", f)
			}
			fmt.Println(strings.Join(stackTrace, "\n"))
		}

		var httpErr HTTPError
		switch {
		case errors.As(errors.Cause(err), &httpErr):
			// We can retrieve the status here and write out a specific
			// HTTP status code.
			log.Printf("HTTP %d - %s\n", httpErr.Status(), httpErr)
			if err := WriteJSON(httpErr.Status(), map[string]string{"error": httpErr.Error()}, w); err != nil {
				log.Error(err)
			}
		default:
			// Any error types we don't specifically look out for default
			// to serving a HTTP 500
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}
	}
}
