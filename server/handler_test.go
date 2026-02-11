package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	. "github.com/onsi/gomega"
)

func TestStatusError_Status(t *testing.T) {
	RegisterTestingT(t)

	e := NewStatusError(500, "some error")
	Ω(e.Error()).Should(Equal("some error"))
	Ω(e.Code).Should(Equal(500))
}

func TestHandler(t *testing.T) {
	RegisterTestingT(t)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	mux := chi.NewMux()

	mux.Handle("/error", Handler{func(w http.ResponseWriter, r *http.Request) error {
		return NewStatusError(http.StatusInternalServerError, "Horrible error")
	}})

	mux.Handle("/ok", Handler{func(w http.ResponseWriter, r *http.Request) error {
		_, err := w.Write([]byte(`{}`))
		return err
	}})

	req, _ := http.NewRequest("GET", "/error", nil)
	mux.ServeHTTP(rr, req)

	Expect(rr.Code).To(Equal(http.StatusInternalServerError))
	Expect(rr.Header().Get("content-type")).To(Equal("application/json; charset=utf-8"))
	Expect(strings.TrimSpace(rr.Body.String())).To(Equal(`{"error":"Horrible error"}`))

	req, _ = http.NewRequest("GET", "/ok", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req)

	Expect(rr2.Code).To(Equal(http.StatusOK))
	Expect(strings.TrimSpace(rr2.Body.String())).To(Equal(`{}`))
}
