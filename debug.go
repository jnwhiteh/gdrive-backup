package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// A RoundTripper that logs the details of requests and their responses
type LoggingRoundTripper struct {
	w  io.Writer
	rt http.RoundTripper
}

func NewLoggingRoundTripper(w io.Writer, rt http.RoundTripper) http.RoundTripper {
	return &LoggingRoundTripper{w, rt}
}

func (t *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	buf := bytes.NewBufferString("\n[request]\n")
	var body bytes.Buffer

	// Capture the request body if present
	if req.Body != nil {
		req.Body = ioutil.NopCloser(&CopyingReader{src: req.Body, dst: &body})
	}
	req.Write(buf)
	if req.Body != nil {
		req.Body = ioutil.NopCloser(&body)
	}
	io.WriteString(buf, "\n[/request]\n")

	// Perform the request
	res, err := t.rt.RoundTrip(req)

	// Capture the response
	io.WriteString(buf, "[response]\n")
	if err != nil {
		fmt.Fprintf(buf, "ERROR: %v", err)
	} else {
		body := res.Body
		res.Body = nil
		res.Write(buf)
		if body != nil {
			res.Body = ioutil.NopCloser(&CopyingReader{src: body, dst: buf})
		}
	}
	io.WriteString(buf, "[/response]\n")

	// Actually write the response
	t.w.Write(buf.Bytes())

	return res, err
}

// An io.Reader that copies everything it reads to another destination
type CopyingReader struct {
	src io.Reader
	dst io.Writer
}

func (r *CopyingReader) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	if n > 0 {
		r.dst.Write(p[:n])
	}
	return n, err
}
