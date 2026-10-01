// Package rest supplies shared HTTP evidence and decoding for SDK-owned APIs.
// Service packages own routes, envelopes, success codes, schemas and defaults.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/resource"
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
// each attempt. The target must remain in the source service's origin. Empty
// accepted responses are valid here; resource decoders enforce their envelopes.
func DoJSON(ctx context.Context, source *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, codes ...int) (*Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateTarget(source, endpoint); err != nil {
		return nil, err
	}
	if len(codes) == 0 {
		return nil, fmt.Errorf("%w: explicit success codes are required", resource.ErrInvalidOption)
	}
	options := &gophercloud.RequestOpts{OkCodes: append([]int(nil), codes...), KeepResponseBody: true, MoreHeaders: maps.Clone(headers)}
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		options.JSONBody = json.RawMessage(encoded)
	}
	client, err := fixedrequest.New(source, method, endpoint)
	if err != nil {
		return nil, err
	}
	wire, err := client.Request(ctx, method, endpoint, options)
	if err != nil {
		// Native errors already retain status/body/header and their original
		// cause. Do not mask them with a successful-response decode error.
		return nil, err
	}
	result := &Response{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	defer wire.Body.Close()
	result.Body, err = io.ReadAll(wire.Body)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		return result, result.Fail(err)
	}
	return result, nil
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
