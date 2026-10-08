// Package rest supplies shared HTTP evidence and decoding for SDK-owned APIs.
// Service packages own routes, envelopes, success codes, schemas and defaults.
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type Response struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

func (r *Response) Fail(cause error) error {
	if r == nil {
		return cause
	}
	return &resource.ResponseError{Body: append([]byte(nil), r.Body...), Header: r.Header.Clone(), StatusCode: r.StatusCode, Cause: cause}
}

// DoJSON snapshots JSON input before HTTP and guards the exact method/URL on
// each attempt. The target must remain in the source service's origin. Accepted
// read, Close and context errors retain the actual response without a resend.
// Resource decoders enforce envelopes; empty accepted responses are valid here.
func DoJSON(ctx context.Context, source *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, codes ...int) (*Response, error) {
	return DoJSONGuarded(ctx, source, nil, method, endpoint, body, headers, codes...)
}

// DoJSONGuarded keeps a library-owned source invariant sticky across native
// callbacks, authentication and accepted read/Close failures. Its nil-guard
// form retains DoJSON's existing behavior.
func DoJSONGuarded(ctx context.Context, source *gophercloud.ServiceClient, sourceGuard func(context.Context) error, method, endpoint string, body any, headers map[string]string, codes ...int) (*Response, error) {
	return doJSONGuarded(ctx, source, sourceGuard, method, endpoint, body, headers, false, codes...)
}

// DoJSONGuardedHeaders additionally fixes operation-owned headers through native
// retries and physical attempts. Use this for invariants such as a selected
// Neutron revision, not for ordinary caller headers whose source precedence and
// retry policy are intentionally retained by DoJSONGuarded.
func DoJSONGuardedHeaders(ctx context.Context, source *gophercloud.ServiceClient, sourceGuard func(context.Context) error, method, endpoint string, body any, headers map[string]string, codes ...int) (*Response, error) {
	return doJSONGuarded(ctx, source, sourceGuard, method, endpoint, body, headers, true, codes...)
}

func doJSONGuarded(ctx context.Context, source *gophercloud.ServiceClient, sourceGuard func(context.Context) error, method, endpoint string, body any, headers map[string]string, fixedHeaders bool, codes ...int) (*Response, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return nil, responseContextError(ctx, err)
	}
	checkSource := func() error {
		if sourceGuard != nil {
			return sourceGuard(ctx)
		}
		return nil
	}
	if err := checkSource(); err != nil {
		return nil, responseContextError(ctx, err)
	}
	if err := ValidateTarget(source, endpoint); err != nil {
		return nil, responseContextError(ctx, err)
	}
	if len(codes) == 0 {
		return nil, fmt.Errorf("%w: explicit success codes are required", resource.ErrInvalidOption)
	}
	expectedCodes := append([]int(nil), codes...)
	var expectedHeaders map[string]string
	if fixedHeaders {
		expectedHeaders = maps.Clone(headers)
	}
	options := &gophercloud.RequestOpts{OkCodes: append([]int(nil), expectedCodes...), KeepResponseBody: true, MoreHeaders: maps.Clone(headers)}
	var expectedBody []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, responseContextError(ctx, err)
		}
		// The retry hook can mutate RawMessage in place. Its request encoding
		// must never alias the original bytes used to validate ownership.
		expectedBody = append([]byte(nil), encoded...)
		options.JSONBody = json.RawMessage(append([]byte(nil), encoded...))
	}
	client, err := fixedrequest.NewGuarded(source, method, endpoint, sourceGuard)
	if err != nil {
		return nil, responseContextError(ctx, err)
	}
	if len(expectedHeaders) != 0 {
		client.HTTPClient.Transport = ownedHeaderTransport{base: client.HTTPClient.Transport, expected: maps.Clone(expectedHeaders)}
	}
	if retry := client.ProviderClient.RetryFunc; retry != nil {
		client.ProviderClient.RetryFunc = func(ctx context.Context, method, endpoint string, options *gophercloud.RequestOpts, original error, count uint) error {
			callbackErr := retry(ctx, method, endpoint, options, original, count)
			ownershipErr := responseRequestOwnership(options, expectedBody, expectedHeaders)
			sourceErr := checkSource()
			if ownershipErr != nil || sourceErr != nil {
				return responseContextError(ctx, joinResponseErrors(original, callbackErr, ownershipErr, sourceErr))
			}
			if callbackErr != nil || ctx.Err() != nil {
				// Returning the native input error stops retrying without
				// adding another cause. Duplicate copies must not make a
				// clean final HTTP rejection look like a handling failure.
				if native, ok := original.(gophercloud.ErrUnexpectedResponseCode); ok {
					if returned, ok := callbackErr.(gophercloud.ErrUnexpectedResponseCode); ok && reflect.DeepEqual(native, returned) {
						return responseContextError(ctx, callbackErr)
					}
				}
				return responseContextError(ctx, joinResponseErrors(original, callbackErr))
			}
			return nil
		}
	}
	wire, err := client.Request(ctx, method, endpoint, options)
	if err != nil {
		// Native errors already retain status/body/header and their original
		// cause. Do not mask them with a successful-response decode error.
		return nil, responseContextError(ctx, joinResponseErrors(err, checkSource()))
	}
	result := &Response{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	var readErr error
	result.Body, readErr = io.ReadAll(wire.Body)
	// Observe the read boundary before Close can restore a changed source.
	readSourceErr := checkSource()
	bodyErr := responseContextError(ctx, joinResponseErrors(readErr, readSourceErr, wire.Body.Close(), checkSource()))
	if !slices.Contains(expectedCodes, result.StatusCode) {
		// Native hooks can expand OkCodes. Actual response evidence still
		// cannot establish success outside the SDK's original policy.
		native := gophercloud.ErrUnexpectedResponseCode{
			Method: method, URL: endpoint, Expected: append([]int(nil), expectedCodes...), Actual: result.StatusCode,
			Body: append([]byte(nil), result.Body...), ResponseHeader: result.Header.Clone(),
		}
		rejection := joinResponseErrors(native, bodyErr)
		if sourceGuard != nil {
			// This is a completed native attempt rejected by our original
			// policy, not a clean native error eligible for lookup fallback.
			return nil, &statusPolicyError{cause: rejection}
		}
		return nil, rejection
	}
	if bodyErr != nil {
		return result, result.Fail(bodyErr)
	}
	return result, nil
}

// statusPolicyError preserves native status evidence through errors.As while
// distinguishing original-policy rejection from a direct native HTTP error.
type statusPolicyError struct{ cause error }

func (e *statusPolicyError) Error() string            { return e.cause.Error() }
func (e *statusPolicyError) Unwrap() error            { return e.cause }
func (e *statusPolicyError) TerminalSDKFailure() bool { return true }

// responseRequestOwnership checks serialized bytes, not interface identity.
// A nil expected body means no request body; an explicit JSON null has bytes.
func responseRequestOwnership(options *gophercloud.RequestOpts, expected []byte, headers map[string]string) error {
	invalid := func() error {
		return fmt.Errorf("%w: retry changes SDK request headers, JSON request or response body ownership", resource.ErrInvalidOption)
	}
	if options == nil || !ownedRequestHeaders(options.MoreHeaders, headers) {
		return invalid()
	}
	changed := !options.KeepResponseBody || options.JSONResponse != nil || options.RawBody != nil
	if options.JSONBody == nil {
		if expected == nil && !changed {
			return nil
		}
		return invalid()
	}
	encoded, err := json.Marshal(options.JSONBody)
	if err != nil {
		return joinResponseErrors(invalid(), err)
	}
	if changed || expected == nil || !bytes.Equal(encoded, expected) {
		return invalid()
	}
	// Do not invoke a replacement marshaler again during native recursion:
	// it may return different bytes on its next call. The owned snapshot is
	// the only JSON body that can reach the next HTTP request.
	options.JSONBody = json.RawMessage(append([]byte(nil), expected...))
	return nil
}

// Only headers supplied by the SDK operation are fixed. Native service/version
// headers and the caller's unrelated retry headers retain their existing policy.
func ownedRequestHeaders(actual, expected map[string]string) bool {
	headers := make(http.Header, len(actual))
	for key, value := range actual {
		headers.Add(key, value)
	}
	return ownedWireHeaders(headers, expected)
}

func ownedWireHeaders(actual http.Header, expected map[string]string) bool {
	for key, value := range expected {
		values := actual.Values(key)
		if len(values) != 1 || values[0] != value {
			return false
		}
	}
	return true
}

type ownedHeaderTransport struct {
	base     http.RoundTripper
	expected map[string]string
}

func (t ownedHeaderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if !ownedWireHeaders(request.Header, t.expected) {
		return nil, fmt.Errorf("%w: request changes SDK-owned headers", resource.ErrInvalidOption)
	}
	return t.base.RoundTrip(request)
}

func joinResponseErrors(causes ...error) error {
	var present []error
	for _, cause := range causes {
		if cause != nil {
			present = append(present, cause)
		}
	}
	if len(present) == 1 {
		return present[0]
	}
	return errors.Join(present...)
}

func responseContextError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = joinResponseErrors(err, cause)
			}
		}
	}
	return err
}

// ValidateTarget prevents server-provided continuation/location URLs from
// receiving the shared token at a different origin or URL authority.
func ValidateTarget(source *gophercloud.ServiceClient, endpoint string) error {
	if source == nil || source.ProviderClient == nil {
		return fmt.Errorf("%w: service client is required", resource.ErrInvalidOption)
	}
	origin, err := url.Parse(source.Endpoint)
	if err != nil || origin.Host == "" || origin.User != nil || origin.Opaque != "" || origin.Fragment != "" || origin.RawQuery != "" || (origin.Scheme != "http" && origin.Scheme != "https") {
		return fmt.Errorf("%w: invalid service endpoint", resource.ErrInvalidOption)
	}
	target, err := url.Parse(endpoint)
	if err != nil || target.Host == "" || target.User != nil || target.Opaque != "" || target.Fragment != "" || !strings.EqualFold(target.Scheme, origin.Scheme) || !strings.EqualFold(target.Host, origin.Host) {
		return fmt.Errorf("%w: request target changes service origin or URL authority", resource.ErrInvalidOption)
	}
	return nil
}

// Object requires a flat object or a named singular envelope. A service must
// deliberately choose an envelope; arbitrary plural arrays are not singletons.
func (r *Response) Object(key string) (json.RawMessage, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: HTTP response is required", resource.ErrInvalidOption)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(r.Body, &fields); err != nil {
		return nil, r.Fail(err)
	}
	if fields == nil {
		return nil, r.Fail(fmt.Errorf("response must be a JSON object"))
	}
	if key == "" {
		return append(json.RawMessage(nil), r.Body...), nil
	}
	body, exists := fields[key]
	var nested map[string]json.RawMessage
	if !exists {
		return nil, r.Fail(fmt.Errorf("response lacks %q object", key))
	}
	if err := json.Unmarshal(body, &nested); err != nil {
		return nil, r.Fail(err)
	}
	if nested == nil {
		return nil, r.Fail(fmt.Errorf("response %q must be a JSON object", key))
	}
	return append(json.RawMessage(nil), body...), nil
}

func Decode[T any](response *Response, key string, metadata func(*T) *resource.Metadata) (*T, error) {
	body, err := response.Object(key)
	if err != nil {
		return nil, err
	}
	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, response.Fail(err)
	}
	if metadata == nil {
		return nil, response.Fail(fmt.Errorf("%w: model metadata is required", resource.ErrInvalidOption))
	}
	meta := metadata(&value)
	if meta == nil {
		return nil, response.Fail(fmt.Errorf("%w: model metadata is required", resource.ErrInvalidOption))
	}
	meta.Header, meta.StatusCode = response.Header.Clone(), response.StatusCode
	return &value, nil
}
