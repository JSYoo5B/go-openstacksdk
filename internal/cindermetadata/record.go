package cindermetadata

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/metadata"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// ExchangeRecord shares the fixed metadata transport with owned Resource views.
// It leaves body projection to the caller and retains every physical receipt.
func (s *Scope) ExchangeRecord(ctx context.Context, method string, key *string, body any, sourceGuard func(context.Context) error, options ...metadata.Option) (*rest.Response, error) {
	if err := s.validate(ctx); err != nil {
		return nil, err
	}
	if sourceGuard == nil {
		return nil, invalid("a captured metadata source guard is required")
	}
	check := func(ctx context.Context) error {
		return errors.Join(s.validate(ctx), sourceGuard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err := check(ctx); err != nil {
		return nil, err
	}
	target := s.endpoint
	switch method {
	case http.MethodGet:
		if key != nil || body != nil {
			return nil, invalid("metadata GET is bodyless and collection scoped")
		}
	case http.MethodPost, http.MethodPut:
		if key != nil {
			return nil, invalid("metadata writes are collection scoped")
		}
	case http.MethodDelete:
		if key == nil || body != nil {
			return nil, invalid("metadata DELETE requires one key and no body")
		}
		if err := metadataKey(*key); err != nil {
			return nil, err
		}
		target += "/" + url.PathEscape(*key)
	default:
		return nil, invalid("unsupported metadata record method %q", method)
	}
	copied := slices.Clone(options)
	guarded := make([]metadata.Option, len(copied))
	for index, option := range copied {
		apply := option
		guarded[index] = func(config *request.Config[metadata.Opts]) error {
			if err := check(ctx); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil metadata option")
			}
			return errors.Join(apply(config), check(ctx))
		}
	}
	headers, err := s.prepare(ctx, guarded)
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, err
	}
	canonical, err := canonicalHeaders(s.client.MoreHeaders, true, s.client.Microversion)
	if err != nil {
		return nil, err
	}
	client := *s.client
	client.MoreHeaders = maps.Clone(canonical)
	for name := range client.MoreHeaders {
		if strings.EqualFold(name, "OpenStack-API-Version") || strings.EqualFold(name, "X-OpenStack-Volume-API-Version") {
			delete(client.MoreHeaders, name)
		}
	}
	values := maps.Clone(canonical)
	if values == nil {
		values = make(map[string]string)
	}
	maps.Copy(values, headers)
	for name := range values {
		if strings.EqualFold(name, "OpenStack-API-Version") || strings.EqualFold(name, "X-OpenStack-Volume-API-Version") {
			delete(values, name)
		}
	}
	var absent []string
	if client.Microversion != "" {
		values["OpenStack-API-Version"] = "volume " + client.Microversion
		values["X-OpenStack-Volume-API-Version"] = client.Microversion
	} else {
		absent = append(absent, "OpenStack-API-Version", "X-OpenStack-Volume-API-Version")
	}
	absent = append(absent, "X-OpenStack-Nova-API-Version")
	accepted, rejected := make([]int, 300), make([]int, 200)
	for i := range accepted {
		accepted[i] = 100 + i
	}
	for i := range rejected {
		rejected[i] = 400 + i
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &client, check, method, target, body, rest.RequestHeaderPolicy{Values: values, Absent: absent}, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, accepted...)
	if response == nil && err != nil {
		var physical gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &physical) {
			response = &rest.Response{Body: bytes.Clone(physical.Body), Header: physical.ResponseHeader.Clone(), StatusCode: physical.Actual}
		}
	}
	return response, err
}
