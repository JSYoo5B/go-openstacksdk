// Package keymanagerread implements the SDK-owned metadata read for Barbican.
// Service leaves retain their public concrete types and operation names.
package keymanagerread

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/internal/jsonfilter"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Record keeps the actual read apart from its source-shaped attribute view.
type Record struct {
	RequestID      string
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// Fetch accepts only an explicit executable ID. Returned HREFs never route HTTP.
func Fetch[T any](ctx context.Context, client *gophercloud.ServiceClient, kind string, ref resource.Ref, options ...request.Option[T]) (*Record, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if kind != "containers" && kind != "orders" {
		return nil, invalid("unsupported key-manager read resource")
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if ref.IsName() {
		return nil, fmt.Errorf("%w: metadata fetch requires an explicit ID", resource.ErrUnsupported)
	}
	id := ref.String()
	if !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return nil, invalid("metadata request ID must be valid text without controls")
	}
	source, err := cloudread.Capture(ctx, client, "key-manager")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	owned := slices.Clone(options)
	guarded := make([]request.Option[T], len(owned))
	for i, option := range owned {
		apply := option
		guarded[i] = func(config *request.Config[T]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil fetch option")
			}
			config.Headers = maps.Clone(config.Headers)
			if config.Headers == nil {
				config.Headers = make(map[string]string)
			}
			applyErr := apply(config)
			config.Headers = maps.Clone(config.Headers)
			return errors.Join(applyErr, guard(ctx))
		}
	}
	var base T
	config, err := request.Apply(base, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err == nil {
		err = guard(ctx)
	}
	if err != nil {
		return nil, cloudread.ContextError(ctx, err)
	}
	// Resource requires a single fixed metadata request, not list/name lookup.
	// Python's translate_response accepts status<400. Parsing failures are an
	// explicit Go difference, retaining the full physical response receipt.
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
	}
	target := source.Client.ServiceURL(kind, url.PathEscape(id))
	response, prior := rest.DoJSONGuarded(ctx, &source.Client, guard, http.MethodGet, target, nil, nil, codes...)
	if response == nil {
		return nil, cloudread.ContextError(ctx, prior)
	}
	result := &Record{RequestID: id, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	wire, decodeErr := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
	if wire != nil {
		result.Wire = wire.Clone()
	}
	if err := errors.Join(prior, decodeErr, guard(ctx)); err != nil {
		return result, cloudread.ContextError(ctx, err)
	}
	view, err := project(wire, kind, id)
	if err != nil {
		return result, response.Fail(err)
	}
	if err := guard(ctx); err != nil {
		return result, response.Fail(cloudread.ContextError(ctx, err))
	}
	result.Resource = view
	return result, nil
}

func invalid(message string) error { return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message) }

// project has only the concrete getter's audited Body descriptors. Wire owns
// all original extension fields; the Resource view excludes unknown/self keys.
func project(wire *resource.RawResource, kind, requestID string) (*resource.RawResource, error) {
	view := wire.Clone()
	view.Body = make(map[string]json.RawMessage)
	// Known client attribute names are accepted when the canonical wire name
	// is absent. Present canonical values (including null) win deterministically
	// in Go; Python mixed-alias collisions instead follow body insertion order.
	effective := wire.Clone()
	aliases := map[string]string{"created_at": "created", "updated_at": "updated"}
	if kind == "containers" {
		aliases["container_id"] = "container_ref"
	} else {
		aliases["order_id"], aliases["secret_id"] = "order_ref", "secret_ref"
	}
	for attr, field := range aliases {
		if _, exists := effective.Body[field]; !exists {
			if raw, present := effective.Body[attr]; present {
				effective.Body[field] = bytes.Clone(raw)
			}
		}
	}
	fields := map[string]string{"name": "name", "status": "status", "type": "type", "created_at": "created", "updated_at": "updated"}
	if kind == "containers" {
		fields["container_ref"], fields["secret_refs"], fields["consumers"] = "container_ref", "secret_refs", "consumers"
	} else {
		for _, key := range []string{"creator_id", "meta", "order_ref", "secret_ref", "sub_status", "sub_status_message"} {
			fields[key] = key
		}
	}
	for dest, source := range fields {
		raw, err := resource.BodyRecordField(effective.Body, source, resource.BodyFieldJSON)
		if err != nil {
			return nil, err
		}
		view.Body[dest] = raw
	}
	// Python fetch starts with the caller's explicit Body id. Missing response
	// id preserves that seed; a present null still overrides it.
	if raw, present := wire.Body["id"]; present {
		view.Body["id"] = bytes.Clone(raw)
	} else {
		view.Body["id"], _ = json.Marshal(requestID)
	}
	view.Body["location"] = json.RawMessage("null")
	format := func(dest, source string) error {
		raw, err := resource.BodyRecordField(effective.Body, source, resource.BodyFieldJSON)
		if err != nil {
			return err
		}
		value, err := jsonfilter.ReferenceLastComponent(raw)
		if err != nil {
			return fmt.Errorf("%w: %s formatter: %w", resource.ErrInvalidOption, source, err)
		}
		view.Body[dest] = value
		return nil
	}
	if kind == "containers" {
		if err := format("container_id", "container_ref"); err != nil {
			return nil, err
		}
		for _, key := range []string{"secret_refs", "consumers"} {
			raw := bytes.TrimSpace(view.Body[key])
			if !bytes.Equal(raw, []byte("null")) && len(raw) > 0 && raw[0] != '[' {
				view.Body[key] = append(append(json.RawMessage("["), raw...), ']')
			}
		}
	} else {
		if err := format("order_id", "order_ref"); err != nil {
			return nil, err
		}
		if err := format("secret_id", "secret_ref"); err != nil {
			return nil, err
		}
		raw := bytes.TrimSpace(view.Body["meta"])
		if !bytes.Equal(raw, []byte("null")) && len(raw) > 0 && raw[0] != '{' {
			view.Body["meta"] = json.RawMessage("{}")
		}
	}
	return view, nil
}

// NullableString only consumes the formatter's known string-or-null result.
func NullableString(raw json.RawMessage) *string {
	var value *string
	_ = json.Unmarshal(raw, &value)
	return value
}
