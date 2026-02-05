package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const contentTypeHeader string = "Content-Type"

var (
	jsonContentTypeValue = []string{"application/json; charset=utf-8"}
	jsContentTypeValue   = []string{"application/javascript; charset=utf-8"}
)

// WriteJSON serializes body to provided writer
func WriteJSON(status int, body interface{}, w http.ResponseWriter) error {
	header := w.Header()
	if val := header[contentTypeHeader]; len(val) == 0 {
		header[contentTypeHeader] = jsonContentTypeValue
	}
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}

// WriteJSONP serializes body as JSONP
func WriteJSONP(status int, body interface{}, callback string, w http.ResponseWriter) error {
	header := w.Header()
	if val := header[contentTypeHeader]; len(val) == 0 {
		header[contentTypeHeader] = jsContentTypeValue
	}
	jsonArr, err := json.Marshal(body)
	if nil != err {
		return err
	}

	w.WriteHeader(status)
	_, err = w.Write([]byte(fmt.Sprintf("%s(%s);", callback, jsonArr)))
	return err
}

// ReadJSON reads
func ReadJSON(rq *http.Request, val interface{}) error {
	defer rq.Body.Close()

	rqBody, err := io.ReadAll(rq.Body)
	if err != nil {
		return fmt.Errorf("cannot read request body: %w", err)
	}

	err = json.Unmarshal(rqBody, val)
	if err != nil {
		return fmt.Errorf("cannot unmarshal request: %w", err)
	}
	return err
}
