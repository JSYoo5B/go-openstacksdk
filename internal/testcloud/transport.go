package testcloud

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

// RoundTripperFunc adapts a function to http.RoundTripper. RoundTrip returns
// exactly what the function returns; use LinkRequest when the fake response
// should carry its request as a real transport's does.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f(r).
func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// LinkRequest wraps fn so that a non-nil response whose Request is nil gets
// the outgoing request, as net/http.Transport sets Response.Request. A
// response that already names a request keeps it.
func LinkRequest(fn func(*http.Request) (*http.Response, error)) RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		v, err := fn(r)
		if v != nil && v.Request == nil {
			v.Request = r
		}
		return v, err
	}
}

// SetTransport replaces the provider's HTTP transport, so requests reach rt
// instead of the fixture server. Client endpoints still name the server URL.
func (c *Cloud) SetTransport(rt http.RoundTripper) {
	c.Provider.HTTPClient.Transport = rt
}

// Response builds a fake response with only StatusCode, Header and Body set.
// The header is used as given; nil leaves Response.Header nil.
func Response(status int, body string, header http.Header) *http.Response {
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}

// StreamResponse is Response with a caller-supplied body, such as a *Body.
func StreamResponse(status int, body io.ReadCloser, header http.Header) *http.Response {
	return &http.Response{StatusCode: status, Header: header, Body: body}
}

// Call is one request seen by a Recorder.
type Call struct {
	Method      string      // r.Method
	Path        string      // r.URL.Path
	EscapedPath string      // r.URL.EscapedPath()
	RawQuery    string      // r.URL.RawQuery
	Body        string      // the full request body; "" when r.Body is nil
	Header      http.Header // r.Header.Clone()
}

// Recorder is an http.RoundTripper that records every request and answers it
// with Reply. It is safe for concurrent use.
type Recorder struct {
	reply RoundTripperFunc
	mu    sync.Mutex
	calls []Call
}

// NewRecorder returns a Recorder that answers through LinkRequest(reply).
func NewRecorder(reply func(*http.Request) (*http.Response, error)) *Recorder {
	return &Recorder{reply: LinkRequest(reply)}
}

// RoundTrip records r and then calls the reply function. The request body is
// read to its end before the reply runs; r.Body is then replaced with a fresh
// reader over the same bytes, so the reply can still inspect it. A request
// body read error is not reported; Call.Body holds the bytes read before it.
func (rec *Recorder) RoundTrip(r *http.Request) (*http.Response, error) {
	body := ""
	if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	call := Call{Method: r.Method, Path: r.URL.Path, EscapedPath: r.URL.EscapedPath(), RawQuery: r.URL.RawQuery, Body: body, Header: r.Header.Clone()}
	rec.mu.Lock()
	rec.calls = append(rec.calls, call)
	rec.mu.Unlock()
	return rec.reply(r)
}

// Calls returns a copy of the recorded calls in arrival order.
func (rec *Recorder) Calls() []Call {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return append([]Call(nil), rec.calls...)
}

// Len returns the number of recorded calls.
func (rec *Recorder) Len() int {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return len(rec.calls)
}

// Reset discards the recorded calls, so later Calls and Len see only the
// requests that arrive afterwards. The reply function is unchanged.
func (rec *Recorder) Reset() {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.calls = nil
}

// Body is a scripted response body for read, close and truncation faults.
// Reads yield Data; the read that exhausts Data returns ReadErr (io.EOF when
// ReadErr is nil) together with the last bytes, and later reads return 0 and
// the same error. A truncated body is Data plus io.ErrUnexpectedEOF. OnRead
// runs once, after the first Read has copied its bytes and before it returns.
// Close runs OnClose on every call and returns CloseErr. Fields must be set
// before the first Read or Close.
type Body struct {
	Data     string
	ReadErr  error
	CloseErr error
	OnRead   func()
	OnClose  func()

	mu     sync.Mutex
	offset int
	onRead sync.Once
	reads  atomic.Int32
	closes atomic.Int32
}

// Read implements io.Reader.
func (b *Body) Read(p []byte) (int, error) {
	b.reads.Add(1)
	b.mu.Lock()
	n := copy(p, b.Data[b.offset:])
	b.offset += n
	done := b.offset == len(b.Data)
	b.mu.Unlock()
	if b.OnRead != nil {
		b.onRead.Do(b.OnRead)
	}
	if !done {
		return n, nil
	}
	if b.ReadErr != nil {
		return n, b.ReadErr
	}
	return n, io.EOF
}

// Close implements io.Closer.
func (b *Body) Close() error {
	b.closes.Add(1)
	if b.OnClose != nil {
		b.OnClose()
	}
	return b.CloseErr
}

// Reads returns how many times Read was called.
func (b *Body) Reads() int { return int(b.reads.Load()) }

// Closes returns how many times Close was called.
func (b *Body) Closes() int { return int(b.closes.Load()) }
