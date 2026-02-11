package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestNoHandlerFound(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/health-check", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	notFoundHandler := func(w http.ResponseWriter, rq *http.Request) {
		w.WriteHeader(404)
		if _, err := w.Write([]byte("not found")); err != nil {
			t.Fatal(err)
		}
	}

	mux := chi.NewMux()
	mux.NotFound(notFoundHandler)
	mux.Get("/fake", func(w http.ResponseWriter, rq *http.Request) {
		if err := WriteJSON(http.StatusOK, map[string]string{"status": "OK"}, w); err != nil {
			t.Fatal(err)
		}
	})
	mux.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	// Check the response body is what we expect.
	expected := `not found`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestHandlerFound(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/health-check", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	notFoundHandler := func(w http.ResponseWriter, rq *http.Request) {
		w.WriteHeader(404)
		if _, err := w.Write([]byte("not found")); err != nil {
			t.Fatal(err)
		}
	}

	mux := chi.NewMux()
	mux.NotFound(notFoundHandler)
	mux.HandleFunc("/health-check", func(w http.ResponseWriter, rq *http.Request) {
		w.WriteHeader(400)
		if _, err := w.Write([]byte("some error")); err != nil {
			t.Fatal(err)
		}
	})
	mux.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	expected := `some error`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
