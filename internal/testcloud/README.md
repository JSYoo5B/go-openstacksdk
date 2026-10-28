# testcloud

Package `testcloud` is the shared test fixture for this module. It never
talks to a live cloud: every request reaches either an `httptest` server or a
fake `http.RoundTripper` installed by the test.

## Fixture server

```go
cloud := testcloud.New(t)               // server, mux and token-authenticated provider
cloud.Mux.HandleFunc("GET /v2.1/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
	testcloud.JSON(w, 200, `{"server":{"id":"s"}}`)
})
client := cloud.Client("compute", "/v2.1/") // endpoint under the fixture server
```

`New` registers `t.Cleanup` for the server. The provider token is
`test-token`; call `cloud.Provider.SetToken` to use another one.
`NewServer` starts a stand-alone server with the same idle-connection policy,
and `AdvertiseNova` serves a Nova version document.

HTTP handlers may run off the test goroutine, so they report failures with
`t.Errorf` and then `return`; they never call `t.Fatal`.

## Fake transports

Use a fake transport when a test must control the response object itself
(status, headers, body faults, `Response.Request`) or must observe requests
without a server.

| API | Behavior |
| --- | --- |
| `type RoundTripperFunc func(*http.Request) (*http.Response, error)` | Adapter; `RoundTrip` returns exactly what the function returns. |
| `func LinkRequest(fn func(*http.Request) (*http.Response, error)) RoundTripperFunc` | Calls `fn`; a non-nil response whose `Request` is nil gets the outgoing request. An existing `Request` is kept. |
| `func (c *Cloud) SetTransport(rt http.RoundTripper)` | Sets `c.Provider.HTTPClient.Transport`. Client endpoints still name the fixture server URL. |
| `func Response(status int, body string, header http.Header) *http.Response` | Sets only `StatusCode`, `Header` (as given, nil stays nil) and a string `Body`. |
| `func StreamResponse(status int, body io.ReadCloser, header http.Header) *http.Response` | Same, with a caller-supplied body such as `*Body`. |
| `func NewRecorder(reply func(*http.Request) (*http.Response, error)) *Recorder` | Transport that records each request, then answers through `LinkRequest(reply)`. Safe for concurrent use. |
| `func (rec *Recorder) Calls() []Call`, `Len() int` | Copy of the recorded calls in arrival order; number of calls. |
| `type Call struct{ Method, Path, EscapedPath, RawQuery, Body string; Header http.Header }` | `r.Method`, `r.URL.Path`, `r.URL.EscapedPath()`, `r.URL.RawQuery`, the full request body (`""` for a nil body) and `r.Header.Clone()`. |
| `type Body struct{ Data string; ReadErr, CloseErr error; OnRead, OnClose func() }` | Scripted response body, see below. `Reads()` and `Closes()` count calls. |

`Body` yields `Data`; the read that exhausts it returns `ReadErr` (or
`io.EOF` when `ReadErr` is nil) together with the last bytes, and later reads
return `0` and the same error. A truncated body is `Data` plus
`io.ErrUnexpectedEOF`. `OnRead` runs once, after the first read has copied
its bytes. `Close` runs `OnClose` on every call and returns `CloseErr`.

```go
calls := testcloud.NewRecorder(func(r *http.Request) (*http.Response, error) {
	return testcloud.Response(200, `{"group":{}}`, http.Header{"Content-Type": {"application/json"}}), nil
})
cloud.SetTransport(calls)
// ...
if calls.Len() != 1 || calls.Calls()[0].EscapedPath != "/cinder/v3/project/groups/a%2Fb" {
	t.Fatal(calls.Calls())
}
```

Worked examples: `blockstorage/v3/groups` (recorder, response builder and
fault body) and `network/v2/extensions/vpnaas/services` (recorder with a
projection onto a local call struct, and `LinkRequest` with a caller-owned
call slice).

## Migrating an ad-hoc transport

Convert one package at a time and keep every assertion's meaning. Run
`go vet` and `go test -race` for the package afterwards.

1. **Adapter type.** Delete the local `type xTransport func(*http.Request)
   (*http.Response, error)` and its `RoundTrip` method.
   - A method that sets `Request` only when the response has none becomes
     `testcloud.LinkRequest(fn)`.
   - A method that is just `return f(r)` becomes `testcloud.RoundTripperFunc(fn)`.
     Do not switch it to `LinkRequest`: setting `Response.Request` can change
     what the code under test observes.
   - A method that always overwrites `Request` is equivalent to `LinkRequest`
     only when the function never returns a response that already names a
     request (for example one from `http.DefaultTransport`). Check before
     converting.
2. **Installation.** `cloud.Provider.HTTPClient.Transport = x` becomes
   `cloud.SetTransport(x)`.
3. **Response builders.** A builder that returns
   `&http.Response{StatusCode: code, Header: h, Body: io.NopCloser(strings.NewReader(body))}`
   becomes `testcloud.Response(code, body, h)`; one with an `io.ReadCloser`
   becomes `testcloud.StreamResponse`. Builders that also set `Status`,
   `ContentLength`, `Request`, `Proto` or trailers stay local, or keep the
   extra assignments after the call.
4. **Call recorders.** A closure that reads the request body and appends
   method, path, escaped path, raw query, body and cloned headers becomes
   `testcloud.NewRecorder(reply)`. Map fields as `method→Method`,
   `path→Path`, `escaped→EscapedPath`, `query→RawQuery`, `body→Body`,
   `header(s)→Header`; `len(*calls)` becomes `calls.Len()` and
   `(*calls)[i]` becomes `calls.Calls()[i]`.
   - When a test compares whole call slices with `reflect.DeepEqual` against
     a local struct, keep that struct and add a projection from
     `rec.Calls()` (see `pythonVPNServiceCalls`). Comparing `[]testcloud.Call`
     directly would also compare headers and escaped paths.
   - The recorder records before the reply runs and replaces `r.Body` with a
     fresh reader over the same bytes. A reply that relied on finding the body
     already drained, or a recorder that records after the reply, header
     values instead of a clone, `url.Values` instead of the raw query, or
     decoded JSON, stays local or needs a projection.
   - A request body read error is ignored by the recorder, as the existing
     closures do; a test that asserts on it stays local.
5. **Fault bodies.** A body that returns its data and then a read error,
   counts closes, runs a hook and returns a close error maps to
   `testcloud.Body`: `text/data→Data`, `readErr→ReadErr`,
   `closeErr→CloseErr`, `afterRead/onRead→OnRead` (once),
   `afterClose/onClose→OnClose` (every close), `closed/closes→Closes()`.
   Check the exact read sequence first. Differences that only show after the
   first error (a body that reports its error once and then `io.EOF`, as the
   former `groupFaultBody` did) or between returning the error with the last
   bytes and on the next read are invisible to `io.ReadAll` and JSON
   decoding; convert such a body only when the code under test stops at the
   first read error. Keep a local body when it wraps a real response body
   (`ReadCloser` embedding); when it blocks, waits on a context or channel, or
   cancels; when it serves at most one buffer of data per read on purpose; or
   when it implements `Seek` or other interfaces.
6. **Delays and cancellation.** Transports that sleep, wait for
   `r.Context().Done()` or release on a channel stay as ordinary functions
   passed to `RoundTripperFunc` or `LinkRequest`; the helper has no
   scheduling API.
7. **Wrapping transports.** Struct types that call `base.RoundTrip` and then
   decorate the response stay local; only their installation changes to
   `SetTransport`.
