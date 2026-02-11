package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	e := WriteJSON(http.StatusOK, map[string]string{"hello": "world"}, rr)
	// Check the status code is what we expect.
	if nil != e {
		t.Error("Something went wrong with serialization")
	}

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the status code is what we expect.
	if contentType := rr.Header().Get("content-type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, "application/json; charset=utf-8")
	}

	// Check the response body is what we expect.
	expected := `{"hello":"world"}`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestWriteJSONP(t *testing.T) {
	rr := httptest.NewRecorder()
	e := WriteJSONP(http.StatusOK, map[string]string{"hello": "world"}, "jsonp", rr)
	// Check the status code is what we expect.
	if nil != e {
		t.Error("Something went wrong with serialization")
	}

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the status code is what we expect.
	if contentType := rr.Header().Get("content-type"); contentType != "application/javascript; charset=utf-8" {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, "application/javascript; charset=utf-8")
	}

	// Check the response body is what we expect.
	expected := `jsonp({"hello":"world"});`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
