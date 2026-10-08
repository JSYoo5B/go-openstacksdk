package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ConsoleAuthTokenRecord separates the input-seeded Resource from the actual
// response object. Returned IDs, hosts and ports are passive JSON values and
// never change the fixed token lookup route or provider authentication token.
type ConsoleAuthTokenRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// ValidateConsoleAuthToken fetches connection information for a literal console
// token. A selected microversion is retained; otherwise discovery selects at
// most 2.99 on a private client. No server lookup or console creation occurs.
func (s *Service) ValidateConsoleAuthToken(ctx context.Context, consoleToken string) (*ConsoleAuthTokenRecord, error) {
	fail := func(value *ConsoleAuthTokenRecord, err error) (*ConsoleAuthTokenRecord, error) {
		return value, request.Wrap("ValidateConsoleAuthToken", "compute", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if err := resource.ID(consoleToken).Validate(); err != nil {
		return fail(nil, err)
	}
	if !utf8.ValidString(consoleToken) || strings.IndexFunc(consoleToken, unicode.IsControl) >= 0 {
		return fail(nil, fmt.Errorf("%w: console token must be valid text without controls", resource.ErrInvalidOption))
	}
	if s == nil || s.API == nil || s.Servers == nil || s.API.Servers == nil {
		return fail(nil, fmt.Errorf("%w: compute console token service is required", resource.ErrInvalidOption))
	}
	original, api, collection := s.client, s.API, s.Servers
	serverAPI, resources := api.Servers, collection.Collection
	if api.RawClient() != original || serverAPI.RawClient() != original || collection.client != original {
		return fail(nil, fmt.Errorf("%w: compute console token components must share their selected client", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return fail(nil, err)
	}
	locationReader := collection.dependencies.CloudLocation
	var changed error
	// Only source facts live in this guard. The operation context composes it
	// with any caller guard, avoiding recursion through the discovery reader.
	guard := func(checkCtx context.Context) error {
		if changed == nil && (s.client != original || s.API != api || s.Servers != collection || api.Servers != serverAPI || api.RawClient() != original || serverAPI.RawClient() != original || collection.client != original || collection.Collection != resources) {
			changed = fmt.Errorf("%w: compute console token source changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, source.Guard(checkCtx))
	}
	check := func(checkCtx context.Context) error {
		return errors.Join(guard(checkCtx), rest.CheckOperationGuard(checkCtx))
	}
	if err := check(ctx); err != nil {
		return fail(nil, err)
	}
	// Python constructs the seeded Resource before its fetch selects a
	// microversion. Own the computed location at the same point in the flow.
	location := json.RawMessage("null")
	if locationReader != nil {
		value, locationErr := locationReader()
		if locationErr == nil {
			location, locationErr = value.ForResource(nil, nil)
		}
		if locationErr = errors.Join(locationErr, check(ctx)); locationErr != nil {
			return fail(nil, locationErr)
		}
	}
	operationCtx := rest.WithOperationGuard(ctx, guard)
	chosen := source.Client.Microversion
	if chosen == "" {
		advertised, discoveryErr := microversions.Read(operationCtx, source, microversions.NovaProfile)
		if discoveryErr != nil {
			return fail(nil, discoveryErr)
		}
		chosen, err = microversions.SelectVersion(advertised.Maximum, advertised.Minimum, "2.99")
		if err != nil {
			err = fmt.Errorf("%w: console token version selection: %w", resource.ErrInvalidOption, err)
			if len(advertised.Responses) != 0 {
				err = advertised.Responses[len(advertised.Responses)-1].Fail(cloudread.ContextError(ctx, err))
			}
			return fail(nil, err)
		}
	}
	if err := check(ctx); err != nil {
		return fail(nil, err)
	}
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = index + 200
	}
	target := source.Client.ServiceURL("os-console-auth-tokens", url.PathEscape(consoleToken))
	response, prior := microversions.MemberGet(operationCtx, source, target, chosen, microversions.NovaProfile, codes...)
	if response == nil {
		return fail(nil, prior)
	}
	result := &ConsoleAuthTokenRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err := errors.Join(prior, check(ctx)); err != nil {
		return fail(result, response.Fail(cloudread.ContextError(ctx, err)))
	}
	seedID, _ := json.Marshal(consoleToken)
	seed := map[string]json.RawMessage{"id": seedID}
	known := []string{"instance_uuid", "host", "port", "tls_port", "internal_access_path", "id", "name"}
	// Resource.fetch tolerates response.json ValueError. An accepted read,
	// Close, source or cancellation failure never reaches this parse policy.
	if utf8.Valid(response.Body) && json.Valid(response.Body) {
		root, decodeErr := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
		if decodeErr != nil {
			return fail(result, decodeErr)
		}
		wire := root
		if _, present := root.Body["console"]; present {
			wire, decodeErr = rest.Decode(response, "console", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
			if decodeErr != nil {
				return fail(result, decodeErr)
			}
		}
		result.Wire = wire.Clone()
		for _, key := range known {
			if raw, present := wire.Body[key]; present {
				seed[key] = bytes.Clone(raw)
			}
		}
	}
	view := &resource.RawResource{Metadata: resource.Metadata{Body: make(map[string]json.RawMessage, 8), Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	for _, key := range known {
		raw, fieldErr := resource.BodyRecordField(seed, key, resource.BodyFieldJSON)
		if fieldErr != nil {
			return fail(result, response.Fail(cloudread.ContextError(ctx, fieldErr)))
		}
		view.Body[key] = raw
	}
	view.Body["location"] = bytes.Clone(location)
	if err := check(ctx); err != nil {
		return fail(result, response.Fail(cloudread.ContextError(ctx, err)))
	}
	result.Resource = view
	return result, nil
}
