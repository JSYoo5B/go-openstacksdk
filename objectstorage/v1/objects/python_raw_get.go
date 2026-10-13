package objects

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// RawObjectResponse retains the actual unchecked status and opaque response
// body. HTTP status alone is not an error. The caller must close Body; the
// default buffers and closes the physical body before returning this reader.
type RawObjectResponse struct {
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
}

// RawObjectOpts configures the named cloud raw GET. Stream=false is the
// pinned default; Stream=true transfers a closeable guarded reader.
type RawObjectOpts struct {
	QueryString string
	Stream      bool
}
type RawObjectOption func(*RawObjectOpts) error

func WithRawObjectOpts(value RawObjectOpts) RawObjectOption {
	return func(cfg *RawObjectOpts) error { *cfg = value; return nil }
}
func WithRawObjectQueryString(value string) RawObjectOption {
	return func(cfg *RawObjectOpts) error { cfg.QueryString = value; return nil }
}

func WithRawObjectStream(value bool) RawObjectOption {
	return func(cfg *RawObjectOpts) error { cfg.Stream = value; return nil }
}

// GetObjectRaw implements get_object_raw(container, obj, query_string=None,
// stream=False). An empty obj selects the container itself like Source.
// QueryString is already encoded and appended verbatim; the route remains
// fixed through native authentication and retry. Payload is never decoded.
func (a *API) GetObjectRaw(ctx context.Context, container, object string, options ...RawObjectOption) (*RawObjectResponse, error) {
	const op = "GetObjectRaw"
	p, err := a.captureObjectList(ctx, container)
	if err != nil {
		return nil, request.Wrap(op, "objects", err)
	}
	if object != "" {
		if err := validateMetadataObject(object); err != nil {
			return nil, request.Wrap(op, "objects", metadataContextError(ctx, err))
		}
	}
	cfg := RawObjectOpts{}
	for _, option := range slices.Clone(options) {
		if err := p.check(ctx); err != nil {
			return nil, request.Wrap(op, "objects", err)
		}
		if option == nil {
			return nil, request.Wrap(op, "objects", metadataInvalid("nil raw object option"))
		}
		next := cfg
		if err := errors.Join(option(&next), p.check(ctx)); err != nil {
			return nil, request.Wrap(op, "objects", metadataContextError(ctx, err))
		}
		cfg = next
	}
	if err := validateRawObjectQuery(cfg.QueryString); err != nil {
		return nil, request.Wrap(op, "objects", metadataContextError(ctx, err))
	}
	target := p.target
	if object != "" {
		target += "/" + url.PathEscape(object)
	}
	if cfg.QueryString != "" {
		target += "?" + cfg.QueryString
	}
	if _, exists := p.client.MoreHeaders["Accept"]; !exists {
		p.client.MoreHeaders["Accept"] = "*/*"
	}
	policy := rawObjectHeaderPolicy(p)
	if cfg.Stream {
		wire, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(ctx, &p.client, p.check, http.MethodGet, target, policy, rest.RejectionPolicy{Codes: []int{401, 429, 498}, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, rawObjectCodes()...)
		var result *RawObjectResponse
		if wire != nil {
			result = &RawObjectResponse{StatusCode: wire.StatusCode, Header: wire.Header.Clone(), Body: rawObjectStreamBody{wire.Body}}
		} else if native, ok := rawObjectNativeReceipt(err); ok {
			// Native rejected responses have already been read and closed.
			// Their actual bytes are returned without reading the body again.
			result = &RawObjectResponse{StatusCode: native.Actual, Header: native.ResponseHeader.Clone(), Body: io.NopCloser(bytes.NewReader(bytes.Clone(native.Body)))}
		}
		if err != nil && rawObjectCleanStatus(err) {
			err = nil
		}
		return result, request.Wrap(op, "objects", metadataContextError(ctx, err))
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &p.client, p.check, http.MethodGet, target, nil, policy, rest.RejectionPolicy{Codes: []int{401, 429, 498}, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, rawObjectCodes()...)
	if response == nil && err != nil {
		if native, ok := rawObjectNativeReceipt(err); ok {
			response = &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	// Only an exact clean native status can become an unchecked raw response.
	// Mixed callback/source/Read/Close/cancellation/reauth failures stay errors.
	if err != nil && rawObjectCleanStatus(err) {
		err = nil
	}
	var result *RawObjectResponse
	if response != nil {
		result = &RawObjectResponse{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: io.NopCloser(bytes.NewReader(bytes.Clone(response.Body)))}
	}
	return result, request.Wrap(op, "objects", metadataContextError(ctx, err))
}

func rawObjectHeaderPolicy(p *objectListSource) rest.RequestHeaderPolicy {
	policy := rest.RequestHeaderPolicy{Values: map[string]string{"Accept": p.client.MoreHeaders["Accept"]}, Absent: slices.Clone(objectListAbsent)}
	if p.client.Microversion != "" {
		policy.Values["OpenStack-API-Version"] = "object-store " + p.client.Microversion
	} else {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version")
	}
	for _, key := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
		if value, exists := p.client.MoreHeaders[key]; exists {
			policy.Values[key] = value
		} else {
			policy.Absent = append(policy.Absent, key)
		}
	}
	return policy
}

func rawObjectNativeReceipt(err error) (gophercloud.ErrUnexpectedResponseCode, bool) {
	receiptErr := err
	var reauth *gophercloud.ErrUnableToReauthenticate
	if errors.As(receiptErr, &reauth) {
		receiptErr = reauth.ErrOriginal
	}
	var after *gophercloud.ErrErrorAfterReauthentication
	if errors.As(receiptErr, &after) {
		receiptErr = after.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	ok := errors.As(receiptErr, &native)
	return native, ok
}

func rawObjectCleanStatus(err error) bool {
	switch err := err.(type) {
	case gophercloud.ErrUnexpectedResponseCode:
		return true
	case *gophercloud.ErrErrorAfterReauthentication:
		return rawObjectCleanStatus(err.ErrOriginal)
	case interface{ Unwrap() []error }:
		causes := err.Unwrap()
		return len(causes) == 1 && rawObjectCleanStatus(causes[0])
	}
	return false
}

func validateRawObjectQuery(query string) error {
	if !utf8.ValidString(query) || strings.ContainsAny(query, "# ") {
		return metadataInvalid("raw object query must be encoded UTF-8 without spaces or fragments")
	}
	for _, b := range []byte(query) {
		if b < 32 || b == 127 {
			return metadataInvalid("invalid raw object query control")
		}
	}
	return nil
}

func rawObjectCodes() []int {
	codes := make([]int, 0, 897)
	for code := 100; code < 1000; code++ {
		if code != 401 && code != 429 && code != 498 {
			codes = append(codes, code)
		}
	}
	return codes
}

// rawObjectStreamBody attaches the public operation while preserving exact
// EOF for io.Copy/ReadAll. The shared unread body owns proof and physical Close.
type rawObjectStreamBody struct{ io.ReadCloser }

func (body rawObjectStreamBody) Read(data []byte) (int, error) {
	n, err := body.ReadCloser.Read(data)
	if err != nil && err != io.EOF {
		err = request.Wrap("GetObjectRaw", "objects", err)
	}
	return n, err
}
func (body rawObjectStreamBody) Close() error {
	return request.Wrap("GetObjectRaw", "objects", body.ReadCloser.Close())
}
