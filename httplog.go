package httplog

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
)

// Middleware represents a http middleware
type Middleware func(http.Handler) http.Handler

// Reporter interface reports (logs) the http request
type Reporter interface {
	Report(res *http.Response, req *http.Request)
}

// ReporterFunc is a helper for defining a Reporter
// in the same spirit as http.HandlerFunc
type ReporterFunc func(*http.Response, *http.Request)

// Report statisfies the reporter interface
func (r ReporterFunc) Report(res *http.Response, req *http.Request) {
	r(res, req)
}

// LoggingMiddleware is the adapter that implements logging
type LoggingMiddleware struct {
	h    http.Handler
	r    Reporter
	body bool
}

// inserts content-length header to raw request dumped from httputil.DumpRequest
func insertContentLength(raw []byte) []byte {
	data := strings.Split(string(raw), "\r\n\r\n")
	header, body := data[0], data[1]
	if len(body) == 0 {
		// the body was empty
		return raw
	}
	contentLength := len(body)
	final := fmt.Sprintf("%s\r\nContent-Length: %d\r\n\r\n%s", header, contentLength, body)
	return []byte(final)
}

// dump an incoming http request, adding in Content-Length header where required
func dumpRequest(req *http.Request, body bool) ([]byte, error) {
	raw, err := httputil.DumpRequest(req, body)
	if err != nil {
		return nil, err
	}
	return insertContentLength(raw), nil
}

func (l LoggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// request and response objects passed to the reporter
	// the reporter is not called if something goes wrong while reading or parsing
	// the request
	var (
		req *http.Request
		rw  *httptest.ResponseRecorder
	)

	// get the raw representation of the request.
	// note that it changes certain aspects of the request (for example, case of the headers)
	raw, err := dumpRequest(r, l.body)
	if err != nil {
		log.Printf("httplog: %v\n", err)
		l.h.ServeHTTP(w, r)
		return
	}

	// create a mirror of the request
	req, err = http.ReadRequest(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		log.Printf("httplog: %v\n", err)
		l.h.ServeHTTP(w, r)
		return
	}

	rw = httptest.NewRecorder()

	// call the underlying http.Handler
	l.h.ServeHTTP(rw, r)

	// forward the values to upstream
	header := w.Header()
	for key, val := range rw.HeaderMap {
		header[key] = val
	}
	w.WriteHeader(rw.Code)
	w.Write(rw.Body.Bytes())

	l.r.Report(rw.Result(), req)
}

// New is a factory that creates a LogMiddleware
func New(r Reporter, body bool) Middleware {
	return func(h http.Handler) http.Handler {
		return LoggingMiddleware{h, r, body}
	}
}
