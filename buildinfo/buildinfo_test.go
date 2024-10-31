package buildinfo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildInfo(t *testing.T) {
	buildInfo := GetBuildInfo()
	buildInfo.Name = "test"
	rr := httptest.NewRecorder()
	rr.WriteHeader(http.StatusOK)
	e := json.NewEncoder(rr).Encode(buildInfo)
	// Check the status code is what we expect.
	if nil != e {
		t.Error("Something went wrong with serialization")
	}

	expected := `{"name":"test"}`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("incorrect build format response: got %v want %v",
			rr.Body.String(), expected)
	}
}
