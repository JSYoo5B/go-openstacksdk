package rest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// DoUnreadGuardedRejectionsHeaderPolicy submits one owned bodyless GET/HEAD
// workflow. Accepted bodies remain unread and caller-owned. Rejected bodies
// retain Gophercloud's native status/body/header error and retry policy, while
// observed handling/source faults cannot establish a clean rejection or replay.
func DoUnreadGuardedRejectionsHeaderPolicy(ctx context.Context, source *gophercloud.ServiceClient, sourceGuard func(context.Context) error, method, endpoint string, headers RequestHeaderPolicy, policy RejectionPolicy, codes ...int) (*http.Response, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := responseContextError(ctx, ctx.Err()); err != nil {
		return nil, err
	}
	var faults RejectedResponseFaults
	checkOriginal := func(check context.Context) error {
		var cause error
		if sourceGuard != nil {
			cause = sourceGuard(check)
		}
		return responseContextError(check, joinResponseErrors(cause, CheckOperationGuard(check)))
	}
	guard := func(check context.Context) error {
		return responseContextError(check, joinResponseErrors(checkOriginal(check), faults.Err()))
	}
	checkSource := func() error { return guard(ctx) }
	if err := checkSource(); err != nil {
		return nil, err
	}
	if method != http.MethodGet && method != http.MethodHead {
		return nil, fmt.Errorf("%w: unread requests must be GET or HEAD", resource.ErrInvalidOption)
	}
	if err := ValidateTarget(source, endpoint); err != nil {
		return nil, responseContextError(ctx, err)
	}
	if len(codes) == 0 {
		return nil, fmt.Errorf("%w: explicit success codes are required", resource.ErrInvalidOption)
	}
	accepted := slices.Clone(codes)
	selected := slices.Clone(policy.Codes)
	expectedHeaders, expectedAbsent := maps.Clone(headers.Values), slices.Clone(headers.Absent)
	options := &gophercloud.RequestOpts{OkCodes: slices.Clone(accepted), KeepResponseBody: true, MoreHeaders: maps.Clone(expectedHeaders)}
	client, err := fixedrequest.NewGuarded(source, method, endpoint, guard)
	if err != nil {
		return nil, responseContextError(ctx, err)
	}
	parent := client.HTTPClient.Transport
	// Returning an independent response fixes physical status/header facts
	// before a custom body or native callback can mutate the transport's value.
	client.HTTPClient.Transport = rejectedResponseTransport(func(req *http.Request) (*http.Response, error) {
		observed, cause := parent.RoundTrip(req)
		if observed == nil {
			return nil, cause
		}
		private := *observed
		private.Header = observed.Header.Clone()
		if private.Body != nil && slices.Contains(selected, private.StatusCode) {
			private.Body = &rejectedResponseBody{ReadCloser: private.Body, faults: &faults, ctx: req.Context(), guard: checkOriginal}
		}
		return &private, cause
	})
	if len(expectedHeaders) != 0 || len(expectedAbsent) != 0 {
		client.HTTPClient.Transport = ownedHeaderTransport{base: client.HTTPClient.Transport, expected: maps.Clone(expectedHeaders), absent: slices.Clone(expectedAbsent)}
	}
	if retry := client.RetryFunc; retry != nil {
		client.RetryFunc = func(callbackCtx context.Context, method, endpoint string, requestOptions *gophercloud.RequestOpts, original error, count uint) error {
			receipt := unreadPinnedError(original)
			if policy.StopFaultRetryCallbacks {
				if cause := checkSource(); cause != nil || callbackCtx.Err() != nil {
					return responseContextError(callbackCtx, joinResponseErrors(receipt, cause))
				}
			}
			callbackErr := retry(callbackCtx, method, endpoint, requestOptions, original, count)
			ownershipErr := joinResponseErrors(ValidateUnreadResponseRequest(requestOptions), responseRequestOwnership(requestOptions, nil, expectedHeaders, expectedAbsent))
			sourceErr := checkSource()
			if ownershipErr != nil || sourceErr != nil {
				return responseContextError(callbackCtx, joinResponseErrors(receipt, callbackErr, ownershipErr, sourceErr))
			}
			if callbackErr != nil || callbackCtx.Err() != nil {
				if native, ok := receipt.(gophercloud.ErrUnexpectedResponseCode); ok && policy.PreserveCleanRetry && slices.Contains(selected, native.Actual) {
					if returned, ok := callbackErr.(gophercloud.ErrUnexpectedResponseCode); ok && reflect.DeepEqual(native, returned) {
						return responseContextError(callbackCtx, receipt)
					}
				}
				return responseContextError(callbackCtx, joinResponseErrors(receipt, callbackErr))
			}
			return nil
		}
	}
	installGuardedBackoffCallback(client, options, checkSource, nil, expectedHeaders, expectedAbsent, policy.StopFaultRetryCallbacks)
	wire, err := client.Request(ctx, method, endpoint, options)
	if err != nil {
		return nil, responseContextError(ctx, joinResponseErrors(err, checkSource()))
	}
	physicalStatus, physicalHeader := wire.StatusCode, wire.Header.Clone()
	if !slices.Contains(accepted, physicalStatus) {
		// A hook may expand OkCodes, but it cannot expand the SDK success
		// policy. This is a completed attempt, not an accepted caller stream.
		data, readErr := io.ReadAll(wire.Body)
		readSourceErr := checkSource()
		bodyErr := responseContextError(ctx, joinResponseErrors(readErr, readSourceErr, wire.Body.Close(), checkSource()))
		native := gophercloud.ErrUnexpectedResponseCode{Method: method, URL: endpoint, Expected: slices.Clone(accepted), Actual: physicalStatus, Body: bytes.Clone(data), ResponseHeader: physicalHeader.Clone()}
		return nil, &statusPolicyError{cause: joinResponseErrors(native, bodyErr)}
	}
	if wire.Body == nil {
		wire.Body = http.NoBody
	}
	body := &unreadResponseBody{raw: wire.Body, ctx: ctx, guard: guard, status: physicalStatus, header: physicalHeader.Clone()}
	wire.Header = physicalHeader.Clone()
	wire.Body = body
	if cause := checkSource(); cause != nil {
		closeErr := body.Close()
		return wire, body.proof(nil, joinResponseErrors(cause, closeErr))
	}
	return wire, nil
}

func unreadPinnedError(err error) error {
	if native, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
		native.Expected = slices.Clone(native.Expected)
		native.Body = bytes.Clone(native.Body)
		native.ResponseHeader = native.ResponseHeader.Clone()
		return native
	}
	return err
}

// unreadResponseBody retains only terminal proof, never accumulated payload.
// In particular a failure's Body is the failing Read's current partial chunk;
// prior successful chunks belong only to the caller.
type unreadResponseBody struct {
	raw        io.ReadCloser
	ctx        context.Context
	guard      func(context.Context) error
	status     int
	header     http.Header
	mu         sync.Mutex
	terminal   error
	closed     bool
	exhausted  bool
	emptyReads int
	closeOnce  sync.Once
	closeCause error
}

func (b *unreadResponseBody) proof(chunk []byte, cause error) error {
	if cause == nil {
		return nil
	}
	return &resource.ResponseError{Body: bytes.Clone(chunk), Header: b.header.Clone(), StatusCode: b.status, Cause: cause}
}
func (b *unreadResponseBody) check() error { return responseContextError(b.ctx, b.guard(b.ctx)) }
func (b *unreadResponseBody) state() (error, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return unreadPinnedProof(b.terminal), b.closed
}
func (b *unreadResponseBody) retain(cause error) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.terminal == nil {
		b.terminal = unreadPinnedProof(cause)
	}
	return unreadPinnedProof(b.terminal)
}
func (b *unreadResponseBody) closePhysical() error {
	b.closeOnce.Do(func() {
		before := b.check()
		var cause error
		if b.raw != nil {
			cause = b.raw.Close()
		}
		after := b.check()
		b.mu.Lock()
		b.closed = true
		b.closeCause = joinResponseErrors(before, cause, after)
		b.mu.Unlock()
	})
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closeCause
}
func (b *unreadResponseBody) Read(data []byte) (int, error) {
	if failure, closed := b.state(); failure != nil {
		return 0, failure
	} else if closed {
		b.mu.Lock()
		exhausted := b.exhausted
		b.mu.Unlock()
		if exhausted {
			return 0, io.EOF
		}
		return 0, io.ErrClosedPipe
	}
	if cause := b.check(); cause != nil {
		return 0, b.retain(b.proof(nil, joinResponseErrors(cause, b.closePhysical())))
	}
	if len(data) == 0 {
		return 0, nil
	}
	n, cause := b.raw.Read(data)
	after := b.check()
	if n < 0 || n > len(data) {
		if cause == io.EOF {
			cause = nil
		}
		invalid := fmt.Errorf("%w: unread body returned an invalid byte count", resource.ErrInvalidOption)
		return 0, b.retain(b.proof(nil, joinResponseErrors(invalid, cause, after, b.closePhysical())))
	}
	b.mu.Lock()
	if n == 0 && cause == nil {
		b.emptyReads++
	} else {
		b.emptyReads = 0
	}
	if b.emptyReads >= 100 {
		cause = io.ErrNoProgress
	}
	b.mu.Unlock()
	if cause == io.EOF && after == nil {
		b.mu.Lock()
		b.exhausted = true
		b.mu.Unlock()
		if closeErr := b.closePhysical(); closeErr != nil {
			return n, b.retain(b.proof(data[:n], closeErr))
		}
		return n, io.EOF
	}
	if cause != nil || after != nil {
		if cause == io.EOF {
			cause = nil
		}
		combined := joinResponseErrors(cause, after, b.closePhysical())
		return n, b.retain(b.proof(data[:n], combined))
	}
	return n, nil
}
func (b *unreadResponseBody) Close() error {
	closeErr := b.closePhysical()
	if terminal, _ := b.state(); terminal != nil {
		return terminal
	}
	if closeErr != nil {
		return b.retain(b.proof(nil, closeErr))
	}
	return nil
}

func unreadPinnedProof(err error) error {
	if proof, ok := err.(*resource.ResponseError); ok {
		return &resource.ResponseError{Body: bytes.Clone(proof.Body), Header: proof.Header.Clone(), StatusCode: proof.StatusCode, Cause: proof.Cause}
	}
	return err
}
