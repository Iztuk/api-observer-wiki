package views

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type JobType string

const (
	RequestJobType  JobType = "request"
	ResponseJobType JobType = "response"
	FailureJobType  JobType = "failure"
)

type Metadata struct {
	RequestID string
	Host      string
	Method    string
	Path      string
	Query     string
	Status    int
	Timestamp time.Time
}

type RequestJob struct {
	Type    JobType
	Meta    Metadata
	Headers http.Header
	Body    []byte
}

type ResponseJob struct {
	Type    JobType
	Meta    Metadata
	Headers http.Header
	Body    []byte
}

type FailureJob struct {
	Type  JobType
	Meta  Metadata
	Error string
}

type CapturingBody struct {
	reader io.Reader
	closer io.Closer

	buf  bytes.Buffer
	once sync.Once
	done func([]byte)
}

// NOTE:
// We read the full request/response body here to capture it for auditing,
// then restore the body so the proxy/client can continue using it.
// This is synchronous and may add latency and memory overhead for large payloads.
// Future work:
//   - add a configurable max body size
//   - move body capture to the worker layer
//   - or use a streaming (io.TeeReader) approach to avoid full buffering
func NewRequestJob(r *http.Request, host string, start time.Time) *RequestJob {
	requestID := getOrCreateRequestID(r)

	return &RequestJob{
		Type: RequestJobType,
		Meta: Metadata{
			RequestID: requestID,
			Host:      host,
			Method:    r.Method,
			Path:      r.URL.Path,
			Query:     r.URL.RawQuery,
			Timestamp: start,
		},
		Headers: r.Header.Clone(),
	}
}

func NewResponseJob(r *http.Response, host string) *ResponseJob {
	requestID := getOrCreateRequestID(r.Request)

	return &ResponseJob{
		Type: ResponseJobType,
		Meta: Metadata{
			RequestID: requestID,
			Host:      host,
			Method:    r.Request.Method,
			Path:      r.Request.URL.Path,
			Query:     r.Request.URL.RawQuery,
			Status:    r.StatusCode,
			Timestamp: time.Now().UTC(),
		},
		Headers: r.Header.Clone(),
	}
}

func NewFailureJob(r *http.Request, upstream string, err error) *FailureJob {
	requestID := getOrCreateRequestID(r)

	status := http.StatusBadGateway
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		status = http.StatusGatewayTimeout
	}

	host := r.Header.Get("X-Original-Host")
	if host == "" {
		host = r.Host
	}

	return &FailureJob{
		Type: FailureJobType,
		Meta: Metadata{
			RequestID: requestID,
			Host:      host,
			Method:    r.Method,
			Path:      r.URL.Path,
			Query:     r.URL.RawQuery,
			Status:    status,
			Timestamp: time.Now().UTC(),
		},
		Error: err.Error(),
	}
}

func NewCapturingBody(body io.ReadCloser, done func([]byte)) *CapturingBody {
	cb := &CapturingBody{
		closer: body,
		done:   done,
	}

	cb.reader = io.TeeReader(body, &cb.buf)

	return cb
}

func (c *CapturingBody) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)

	if err == io.EOF {
		c.finish()
	}

	return n, err
}

func (c *CapturingBody) Close() error {
	c.finish()
	return c.closer.Close()
}

func (c *CapturingBody) finish() {
	c.once.Do(func() {
		if c.done != nil {
			bodyCopy := append([]byte(nil), c.buf.Bytes()...)
			c.done(bodyCopy)
		}
	})
}

func getOrCreateRequestID(r *http.Request) string {
	if r == nil || r.Header == nil {
		return newRequestID()
	}

	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}

	return newRequestID()
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}

	return time.Now().UTC().Format("20060102150405.000000000")
}
