package objects

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/JSYoo5B/gophercloudsdk/internal/fixedrequest"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Each exchange is one logical attempt. Native policy may start several
// physical requests; their bodies and observations are independently owned.
type objectCreatePayload struct {
	source       *objectCreateSource
	offset, size int64
}
type objectCreateOutcome struct {
	phase            *ObjectCreatePhaseResult
	retry, ambiguous bool
	err              error
}

type objectCreatePhysical struct {
	mu               sync.Mutex
	response         *ObjectCreateResponse
	err              error
	dirty, ambiguous bool
	body             *objectCreateResponseBody
}

func (r *objectCreatePhysical) add(err error, dirty bool) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = joinMetadataErrors(r.err, cloneObjectCreateError(err))
	r.dirty = r.dirty || dirty
	r.ambiguous = r.ambiguous || dirty
}
func (r *objectCreatePhysical) snapshot() (*ObjectCreateResponse, error, bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneObjectCreateResponse(r.response), cloneObjectCreateError(r.err), r.dirty, r.ambiguous
}

type objectCreateResponseBody struct {
	mu       sync.Mutex
	reader   io.ReadCloser
	physical *objectCreatePhysical
	p        *preparedCreateObject
	ctx      context.Context
	closed   bool
}

func (b *objectCreateResponseBody) Read(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return 0, io.ErrClosedPipe
	}
	b.physical.add(b.p.note(b.ctx), true)
	n, err := b.reader.Read(data)
	b.physical.mu.Lock()
	b.physical.response.Body = append(b.physical.response.Body, data[:n]...)
	b.physical.mu.Unlock()
	if err != io.EOF {
		b.physical.add(err, true)
	}
	b.physical.add(b.p.note(b.ctx), true)
	return n, err
}
func (b *objectCreateResponseBody) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.physical.add(b.p.note(b.ctx), true)
	b.closed = true
	err := b.reader.Close()
	b.physical.add(err, true)
	b.physical.add(b.p.note(b.ctx), true)
	return err
}

type objectCreateRequestBody struct {
	mu       sync.Mutex
	reader   io.Reader
	p        *preparedCreateObject
	ctx      context.Context
	physical *objectCreatePhysical
	closed   bool
}

func (b *objectCreateRequestBody) Read(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return 0, io.ErrClosedPipe
	}
	if err := b.p.guard(b.ctx); err != nil {
		b.physical.add(err, true)
		return 0, err
	}
	n, err := b.reader.Read(data)
	if err != io.EOF {
		b.physical.add(err, true)
	}
	guardErr := b.p.note(b.ctx)
	b.physical.add(guardErr, true)
	return n, joinMetadataErrors(err, guardErr)
}
func (b *objectCreateRequestBody) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	err := b.p.note(b.ctx)
	b.physical.add(err, true)
	b.closed = true
	guardErr := b.p.note(b.ctx)
	b.physical.add(guardErr, true)
	return joinMetadataErrors(err, guardErr)
}

type objectCreateExchange struct {
	p                    *preparedCreateObject
	ctx                  context.Context
	method, target, role string
	manifestHeader       string
	ownedHeaders         map[string]string
	payload              *objectCreatePayload
	codes                []int
	options              *gophercloud.RequestOpts
	owner                *bytes.Reader
	mu                   sync.Mutex
	physical             []*objectCreatePhysical
	views                []*objectCreateRequestBody
	terminal             bool
	terminalCause        error
	finished             bool
}

func (x *objectCreateExchange) latest() *objectCreatePhysical {
	x.mu.Lock()
	defer x.mu.Unlock()
	if len(x.physical) == 0 {
		return nil
	}
	return x.physical[len(x.physical)-1]
}
func (x *objectCreateExchange) ownership() error {
	o := x.options
	if o == nil || o.RawBody != x.expectedRawBody() || o.JSONBody != nil || o.JSONResponse != nil || !o.KeepResponseBody || !slices.Equal(o.OkCodes, x.codes) {
		return metadataInvalid("native policy changes object body or success-code ownership")
	}
	position, err := x.owner.Seek(0, io.SeekCurrent)
	if err != nil || x.owner.Size() != 0 || position != 0 {
		return metadataInvalid("native policy changes owned object replay carrier")
	}
	return x.headers(o.MoreHeaders, true)
}
func (x *objectCreateExchange) forcedHeader(name string) bool {
	return (name == "etag" && (x.role == "segment" || x.role == "slo" || x.role == "dlo")) || (name == "if-none-match" && x.role == "segment") || (name == "accept" && x.role == "slo")
}
func (x *objectCreateExchange) ownedHeader(name string) bool {
	return strings.HasPrefix(name, "x-object-meta-") || name == "x-object-manifest" || (name == "content-type" && x.role == "directory-marker")
}
func (x *objectCreateExchange) headers(values map[string]string, requireOwned bool) error {
	seen := make(map[string]string, len(values))
	for key, value := range values {
		name := strings.ToLower(key)
		if x.forcedHeader(name) {
			continue
		}
		canonical := http.CanonicalHeaderKey(key)
		if _, exists := seen[canonical]; exists {
			return metadataInvalid("native object header aliases %q", key)
		}
		seen[canonical] = value
		if x.ownedHeader(name) {
			if expected, exists := x.ownedHeaders[canonical]; !exists || expected != value {
				return metadataInvalid("native policy changes owned object header %q", key)
			}
			continue
		}
		if _, err := validateCreateObjectHeaders(map[string]string{key: value}); err != nil {
			return err
		}
	}
	if requireOwned {
		for key, value := range x.ownedHeaders {
			if x.ownedHeader(strings.ToLower(key)) {
				if actual, exists := seen[key]; !exists || actual != value {
					return metadataInvalid("native policy removes owned object header %q", key)
				}
			}
		}
	}
	if x.p.headerPolicy != nil {
		return x.p.headerPolicy(values)
	}
	return nil
}
func (x *objectCreateExchange) wireHeaders(headers http.Header) error {
	values := make(map[string]string, len(headers))
	for key, entries := range headers {
		name := strings.ToLower(key)
		if x.forcedHeader(name) {
			continue
		}
		if name == "x-auth-token" {
			if key != http.CanonicalHeaderKey(key) || len(entries) != 1 || !metadataFieldValue(entries[0]) {
				return metadataInvalid("native policy changes authenticated object header")
			}
			continue
		}
		// Native raw ordinary headers may have several values. Validate each,
		// while aliases and SDK-owned fields still retain their strict rules.
		if len(entries) == 0 {
			return metadataInvalid("empty native object header %q", key)
		}
		for _, value := range entries {
			if err := x.headers(map[string]string{key: value}, false); err != nil {
				return err
			}
		}
		values[key] = entries[0]
		if len(entries) != 1 && x.ownedHeader(name) {
			return metadataInvalid("multiple owned object header %q values", key)
		}
	}
	return x.headers(values, true)
}
func (x *objectCreateExchange) expectedRawBody() io.Reader {
	if x.payload == nil {
		return nil
	}
	return x.owner
}
func (x *objectCreateExchange) stop(original, callback error) error {
	err := joinMetadataErrors(original, callback, x.ownership(), x.p.guard(x.ctx))
	if last := x.latest(); last != nil {
		_, prior, dirty, _ := last.snapshot()
		if dirty {
			err = joinMetadataErrors(err, prior)
		}
		if err != nil {
			last.add(err, false)
		}
	}
	if err != nil {
		x.mu.Lock()
		x.terminal = true
		x.terminalCause = joinMetadataErrors(x.terminalCause, cloneObjectCreateError(err))
		x.mu.Unlock()
	}
	return err
}
func (x *objectCreateExchange) beforeHook(original error) error {
	x.mu.Lock()
	terminal, terminalCause := x.terminal, x.terminalCause
	x.mu.Unlock()
	if terminal {
		return joinMetadataErrors(original, cloneObjectCreateError(terminalCause))
	}
	if last := x.latest(); last != nil {
		last.add(original, false)
		_, prior, dirty, _ := last.snapshot()
		if dirty {
			return x.stop(original, prior)
		}
	}
	if err := joinMetadataErrors(x.ownership(), x.p.guard(x.ctx)); err != nil {
		return x.stop(original, err)
	}
	return nil
}
func (x *objectCreateExchange) body(ctx context.Context, record *objectCreatePhysical) io.ReadCloser {
	if x.payload == nil || x.payload.size == 0 {
		return http.NoBody
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.finished {
		return &objectCreateRequestBody{p: x.p, ctx: ctx, physical: record, closed: true}
	}
	view := &objectCreateRequestBody{reader: x.payload.source.section(x.payload.offset, x.payload.size), p: x.p, ctx: ctx, physical: record}
	x.views = append(x.views, view)
	return view
}
func removeObjectCreateHeader(headers http.Header, name string) {
	for key := range headers {
		if strings.EqualFold(key, name) {
			delete(headers, key)
		}
	}
}
func (x *objectCreateExchange) force(req *http.Request) {
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	if x.payload != nil {
		removeObjectCreateHeader(req.Header, "Content-Length")
		removeObjectCreateHeader(req.Header, "Transfer-Encoding")
		removeObjectCreateHeader(req.Header, "Trailer")
		req.ContentLength, req.TransferEncoding, req.Trailer = x.payload.size, nil, nil
	}
	if x.role == "segment" || x.role == "slo" || x.role == "dlo" {
		removeObjectCreateHeader(req.Header, "ETag")
	}
	if x.role == "segment" {
		removeObjectCreateHeader(req.Header, "If-None-Match")
		req.Header.Set("If-None-Match", "*")
	}
	if x.role == "slo" {
		removeObjectCreateHeader(req.Header, "Accept")
		req.Header.Set("Accept", "application/json")
	}
	if x.role == "dlo" {
		removeObjectCreateHeader(req.Header, "X-Object-Manifest")
		req.Header.Set("X-Object-Manifest", x.manifestHeader)
	}
}
func (x *objectCreateExchange) roundTrip(transport http.RoundTripper, req *http.Request) (*http.Response, error) {
	if err := joinMetadataErrors(x.ownership(), x.wireHeaders(req.Header), x.p.guard(req.Context())); err != nil {
		return nil, x.stop(nil, err)
	}
	if x.payload == nil && ((req.Body != nil && req.Body != http.NoBody) || req.ContentLength != 0 || len(req.TransferEncoding) != 0 || len(req.Trailer) != 0) {
		return nil, x.stop(nil, metadataInvalid("native policy changes bodyless object request"))
	}
	// Scope is checked here before recording a started physical request, and
	// again by fixedrequest immediately before authentication and transport.
	want, _ := url.Parse(x.target)
	if req.Method != x.method || req.URL == nil || req.URL.Scheme != want.Scheme || req.URL.Host != want.Host || req.URL.EscapedPath() != want.EscapedPath() || req.URL.RawQuery != want.RawQuery || req.URL.User != nil || req.URL.Opaque != "" || (req.Host != "" && req.Host != want.Host) {
		return nil, x.stop(nil, metadataInvalid("object request changes fixed method or target"))
	}
	record := &objectCreatePhysical{}
	x.mu.Lock()
	x.physical = append(x.physical, record)
	x.mu.Unlock()
	if x.payload != nil {
		// The original SDK request must also be replayable: http.Client makes
		// its 307/308 decision before invoking CheckRedirect. Its initial
		// bytes.Reader carrier already supplies GetBody even if send forks it.
		x.force(req)
		req.GetBody = func() (io.ReadCloser, error) { return x.body(req.Context(), record), nil }
	}
	copy := req.Clone(req.Context())
	x.force(copy)
	if x.payload != nil {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		copy.Body = x.body(req.Context(), record)
		copy.GetBody = func() (io.ReadCloser, error) { return x.body(req.Context(), record), nil }
	}
	wire, err := transport.RoundTrip(copy)
	if wire != nil {
		record.mu.Lock()
		record.response = &ObjectCreateResponse{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
		record.ambiguous = record.ambiguous || wire.StatusCode == 408 || wire.StatusCode >= 500 || (x.role == "segment" && wire.StatusCode == 202) || ((x.role == "slo" || x.role == "dlo") && (wire.StatusCode == 201 || wire.StatusCode == 202))
		record.mu.Unlock()
		if wire.Body == nil {
			wire.Body = http.NoBody
		}
		body := &objectCreateResponseBody{reader: wire.Body, physical: record, p: x.p, ctx: req.Context()}
		record.body = body
		wire.Body = body
	}
	if err != nil {
		record.add(err, false)
		record.mu.Lock()
		record.ambiguous = true
		record.mu.Unlock()
	}
	record.add(x.p.note(req.Context()), true)
	return wire, err
}

func (p *preparedCreateObject) exchange(ctx context.Context, method, target, role string, payload *objectCreatePayload, headers map[string]string, logical int, codes ...int) objectCreateOutcome {
	out := objectCreateOutcome{phase: &ObjectCreatePhaseResult{}}
	if err := joinMetadataErrors(p.guard(ctx), rest.ValidateTarget(p.metadata.client, target)); err != nil {
		out.err = err
		return out
	}
	client, err := fixedrequest.New(p.metadata.client, method, target)
	if err != nil {
		out.err = err
		return out
	}
	// Service headers were captured before callbacks. Phase overlays are
	// request-local; a native hook may replace permitted ordinary headers.
	client.MoreHeaders = cloneMetadataHeaders(p.metadata.headers)
	for key, value := range headers {
		client.MoreHeaders[key] = value
	}
	x := &objectCreateExchange{p: p, ctx: ctx, method: method, target: target, role: role, payload: payload, codes: append([]int(nil), codes...), owner: bytes.NewReader(nil)}
	x.ownedHeaders = cloneMetadataHeaders(client.MoreHeaders)
	for key, value := range headers {
		if strings.EqualFold(key, "X-Object-Manifest") {
			x.manifestHeader = value
		}
	}
	x.options = &gophercloud.RequestOpts{RawBody: x.expectedRawBody(), KeepResponseBody: true, OkCodes: append([]int(nil), codes...)}
	provider := client.ProviderClient
	transport := provider.HTTPClient.Transport
	provider.HTTPClient.Transport = objectDeleteTransport(func(req *http.Request) (*http.Response, error) { return x.roundTrip(transport, req) })
	if retry := provider.RetryFunc; retry != nil {
		provider.RetryFunc = func(callbackCtx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			if err := x.beforeHook(original); err != nil {
				return err
			}
			callbackErr := retry(callbackCtx, method, target, opts, original, count)
			if err := joinMetadataErrors(callbackErr, x.ownership(), p.guard(callbackCtx)); err != nil {
				return x.stop(original, err)
			}
			return nil
		}
	}
	if backoff := provider.RetryBackoffFunc; backoff != nil {
		provider.RetryBackoffFunc = func(callbackCtx context.Context, response *gophercloud.ErrUnexpectedResponseCode, original error, count uint) error {
			if err := x.beforeHook(joinMetadataErrors(response, original)); err != nil {
				return err
			}
			callbackErr := backoff(callbackCtx, response, original, count)
			if err := joinMetadataErrors(callbackErr, x.ownership(), p.guard(callbackCtx)); err != nil {
				return x.stop(joinMetadataErrors(response, original), err)
			}
			return nil
		}
	}
	if reauth := provider.ReauthFunc; reauth != nil {
		provider.ReauthFunc = func(callbackCtx context.Context) error {
			if err := x.beforeHook(nil); err != nil {
				return err
			}
			callbackErr := reauth(callbackCtx)
			if err := joinMetadataErrors(callbackErr, x.ownership(), p.guard(callbackCtx)); err != nil {
				return x.stop(nil, err)
			}
			return nil
		}
	}
	redirect := provider.HTTPClient.CheckRedirect
	provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if err := x.beforeHook(nil); err != nil {
			return err
		}
		if x.payload != nil {
			if next.Body != nil {
				_ = next.Body.Close()
			}
			next.Body = x.body(next.Context(), x.latest())
			x.force(next)
			next.GetBody = func() (io.ReadCloser, error) { return x.body(next.Context(), x.latest()), nil }
		}
		expectedBody, expectedGetBody := next.Body, next.GetBody
		auth := append([]string(nil), next.Header.Values("X-Auth-Token")...)
		callbackErr := redirect(next, via)
		var ownershipErr error
		length := int64(0)
		if payload != nil {
			length = payload.size
		}
		if next.Body != expectedBody || next.ContentLength != length || len(next.TransferEncoding) != 0 || len(next.Trailer) != 0 || !slices.Equal(auth, next.Header.Values("X-Auth-Token")) {
			ownershipErr = metadataInvalid("redirect changes owned object body or framing")
		}
		if (next.GetBody == nil) != (expectedGetBody == nil) || (next.GetBody != nil && reflect.ValueOf(next.GetBody).Pointer() != reflect.ValueOf(expectedGetBody).Pointer()) {
			ownershipErr = joinMetadataErrors(ownershipErr, metadataInvalid("redirect changes owned object replay factory"))
		}
		if err := joinMetadataErrors(callbackErr, ownershipErr, x.ownership(), x.wireHeaders(next.Header), p.guard(next.Context())); err != nil {
			return x.stop(nil, err)
		}
		x.force(next)
		// http.Client requires GetBody to follow a 307/308 with content. The
		// transport supplies a further independent view for the actual send.
		if x.payload != nil {
			next.GetBody = func() (io.ReadCloser, error) { return x.body(next.Context(), x.latest()), nil }
		}
		return nil
	}
	wire, nativeErr := client.Request(ctx, method, target, x.options)
	accepted := wire != nil && nativeErr == nil && slices.Contains(codes, wire.StatusCode)
	if accepted {
		_, readErr := io.Copy(io.Discard, wire.Body)
		nativeErr = joinMetadataErrors(readErr, wire.Body.Close())
	}
	// Rejected native reads/Close deliberately lose their errors. Our owned
	// observer retains them, and closes any response abandoned by transport.
	x.mu.Lock()
	x.finished = true
	physicals := append([]*objectCreatePhysical(nil), x.physical...)
	views := append([]*objectCreateRequestBody(nil), x.views...)
	x.mu.Unlock()
	for _, physical := range physicals {
		if physical.body != nil {
			_ = physical.body.Close()
		}
	}
	for _, view := range views {
		_ = view.Close()
	}
	guardErr := p.guard(ctx)
	last := x.latest()
	if last != nil {
		last.add(joinMetadataErrors(nativeErr, guardErr), guardErr != nil)
	}
	var finalResponse *ObjectCreateResponse
	var finalDirty bool
	for index, physical := range physicals {
		response, cause, dirty, ambiguous := physical.snapshot()
		if response != nil && !slices.Contains(codes, response.StatusCode) && cause == nil && !(response.StatusCode >= 300 && response.StatusCode < 400) {
			cause = gophercloud.ErrUnexpectedResponseCode{Method: method, URL: target, Expected: append([]int(nil), codes...), Actual: response.StatusCode, Body: append([]byte(nil), response.Body...), ResponseHeader: response.Header.Clone()}
		}
		out.phase.Attempts = append(out.phase.Attempts, ObjectCreateAttempt{LogicalAttempt: logical, PhysicalAttempt: index + 1, Response: response, Error: cloneObjectCreateError(cause)})
		out.ambiguous = out.ambiguous || ambiguous
		if physical == last {
			finalResponse, finalDirty = response, dirty
			nativeErr = joinMetadataErrors(nativeErr, cause)
		}
	}
	if accepted && finalResponse != nil && slices.Contains(codes, finalResponse.StatusCode) {
		out.phase.Acknowledgement = cloneObjectCreateResponse(finalResponse)
		out.err = objectCreateResponseError(finalResponse, joinMetadataErrors(nativeErr, guardErr))
		return out
	}
	out.err = joinMetadataErrors(nativeErr, guardErr)
	x.mu.Lock()
	terminal := x.terminal
	x.mu.Unlock()
	out.retry = out.err != nil && !terminal && !finalDirty && guardErr == nil && (x.role != "segment" || finalResponse == nil || finalResponse.StatusCode != http.StatusPreconditionFailed)
	return out
}

// Native errors may contain response maps or byte slices also exposed to a
// retry callback. Clone those carriers while retaining opaque native shapes.
func cloneObjectCreateError(err error) error {
	if err == nil {
		return nil
	}
	switch value := err.(type) {
	case *resource.ResponseError:
		copy := *value
		copy.Body = append([]byte(nil), value.Body...)
		copy.Header = value.Header.Clone()
		copy.Cause = cloneObjectCreateError(value.Cause)
		return &copy
	case *ObjectCreateUnconfirmedSegmentError:
		copy := *value
		return &copy
	case gophercloud.ErrUnexpectedResponseCode:
		value.Body = append([]byte(nil), value.Body...)
		value.ResponseHeader = value.ResponseHeader.Clone()
		value.Expected = append([]int(nil), value.Expected...)
		return value
	case *gophercloud.ErrUnexpectedResponseCode:
		copy := cloneObjectCreateError(*value).(gophercloud.ErrUnexpectedResponseCode)
		return &copy
	case *gophercloud.ErrUnableToReauthenticate:
		copy := *value
		copy.ErrOriginal, copy.ErrReauth = cloneObjectCreateError(value.ErrOriginal), cloneObjectCreateError(value.ErrReauth)
		return &copy
	case *gophercloud.ErrErrorAfterReauthentication:
		copy := *value
		copy.ErrOriginal = cloneObjectCreateError(value.ErrOriginal)
		return &copy
	case *url.Error:
		copy := *value
		copy.Err = cloneObjectCreateError(value.Err)
		return &copy
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		copies := make([]error, len(causes))
		for i, cause := range causes {
			copies[i] = cloneObjectCreateError(cause)
		}
		return errors.Join(copies...)
	}
	return err
}
