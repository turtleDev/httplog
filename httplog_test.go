package httplog

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func outputPrefix(s string) string {
	return "Received:" + s
}

// echoHandler simply writes back whatever it gets in it's body
func echoHandler(w http.ResponseWriter, r *http.Request) {
	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprint(w, "error reading body", err)
		w.WriteHeader(500)
		return
	}
	fmt.Fprint(w, outputPrefix(string(raw)))
}

func TestRequest(t *testing.T) {

	// reported is the incoming payload reported by the reporter
	var reported string
	var err error

	// incoming payload
	payload := "incoming request"

	// reporter func
	reporter := func(res *http.Response, req *http.Request) {
		var raw []byte
		raw, err = ioutil.ReadAll(req.Body)
		if err == nil {
			reported = string(raw)
		}
	}

	// create a new middleware
	middleware := New(ReporterFunc(reporter), true)

	// wrap the handler in the middleware
	h := middleware(http.HandlerFunc(echoHandler))

	r := httptest.NewRequest("POST", "/", strings.NewReader(payload))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("echo handler returned error: %d %s", w.Code, w.Body.String())
	}

	if reported != payload {
		t.Errorf("invalid request payload, expected '%s', got '%s'", payload, reported)
	}

	if err != nil {
		t.Error("error in reporter", err)
	}
}

func TestResponse(t *testing.T) {

	// response reported by the reporter
	var reported string
	var err error

	// response from the handler (echoed back)
	payload := "outgoing request"

	// reporter that logs the outgoing response
	reporter := func(res *http.Response, req *http.Request) {
		var raw []byte
		raw, err = ioutil.ReadAll(res.Body)
		if err == nil {
			reported = string(raw)
		}
	}

	middleware := New(ReporterFunc(reporter), true)
	h := middleware(http.HandlerFunc(echoHandler))

	r := httptest.NewRequest("POST", "/", strings.NewReader(payload))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("echo handler returned error: %d %s", w.Code, w.Body.String())
	}

	if reported != outputPrefix(payload) {
		t.Errorf("invalid request payload, expected '%s', got '%s'", payload, reported)
	}

	if err != nil {
		t.Error("error in reporter", err)
	}
}
