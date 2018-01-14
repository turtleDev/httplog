package httplog

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
)

// Middleware represents a http middleware
type Middleware func(http.Handler) http.Handler

// Reporter interface reports (logs) the http request
type Reporter interface {
	Report(res *http.Response, req *http.Request)
}

// LoggingMiddleware is the adapter that implements logging
type LoggingMiddleware struct {
	h http.Handler
	r Reporter
}

func (l LoggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	defer panicGuard()

	raw, err := httputil.DumpRequest(r, true)
	if err != nil {
		// what to do if this fails?
	}

	// create a mirror of the request
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		// what if THIS fails?
	}

	rw := httptest.NewRecorder()

	// call the underlying http.Handler
	l.h.ServeHTTP(rw, r)

	// forward the values to upstream
	header := w.Header()
	for key, val := range rw.HeaderMap {
		header[key] = val
	}
	w.WriteHeader(rw.Code)
	w.Write(rw.Body.Bytes())

	// pass the result to reporter
	l.r.Report(rw.Result(), req)
}

// NewMiddleware is a factory that creates a LogMiddleware
func NewMiddleware(r Reporter) Middleware {
	return func(h http.Handler) http.Handler {
		return LoggingMiddleware{h, r}
	}
}

func panicGuard() {
	if e := recover(); e != nil {
		// report error
	}
}
