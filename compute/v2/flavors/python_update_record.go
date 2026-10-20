package flavors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateRecordOpts carries the Python update_flavor description. An omitted
// field is not sent, Null sends JSON null and Present keeps an empty string,
// both of which Nova accepts to clear the description. Other declared Flavor
// attributes use WithUpdateRecordField with their server keys; the server
// validates every value.
type UpdateRecordOpts struct {
	Description request.Optional[string] `json:"description,omitzero"`
}

type UpdateRecordOption = request.Option[UpdateRecordOpts]

func WithUpdateRecordOptions(value UpdateRecordOpts) UpdateRecordOption {
	return request.WithOptions(value)
}
func WithUpdateRecordDescription(value request.Optional[string]) UpdateRecordOption {
	return func(config *request.Config[UpdateRecordOpts]) error { config.Options.Description = value; return nil }
}

// WithUpdateRecordField adds another attribute inside the flavor envelope.
// The typed description key and the URI identity id are rejected.
func WithUpdateRecordField(key string, value any) UpdateRecordOption {
	return request.WithField[UpdateRecordOpts](key, value)
}
func WithUpdateRecordHeader(key, value string) UpdateRecordOption {
	return request.WithHeader[UpdateRecordOpts](key, value)
}

// FlavorUpdateRecord is the committed Python Flavor view. Flavor is the
// native typed decode of the response object and Fields keeps that object
// unchanged. Sent is false when no attribute was dirty and nothing was sent.
type FlavorUpdateRecord struct {
	Flavor     *Flavor
	Fields     map[string]json.RawMessage
	Header     http.Header
	StatusCode int
	Sent       bool
}

// UpdateRecord implements update_flavor: Resource.commit sends PUT
// flavors/{id} with only the supplied attributes in {"flavor": ...} and
// accepts every status below 400. With no attribute it sends nothing.
func (a *API) UpdateRecord(ctx context.Context, flavorID string, options ...UpdateRecordOption) (*FlavorUpdateRecord, error) {
	const op = "UpdateRecord"
	fail := func(value *FlavorUpdateRecord, err error) (*FlavorUpdateRecord, error) {
		return value, request.Wrap(op, "flavors", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: flavor API is required", resource.ErrInvalidOption))
	}
	if strings.TrimSpace(flavorID) == "" {
		return fail(nil, fmt.Errorf("%w: flavor id is required", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, a.client, "compute")
	if err != nil {
		return fail(nil, err)
	}
	config, err := cloudread.ApplyReadOptions(ctx, UpdateRecordOpts{}, options, cloudread.OwnReadConfig[UpdateRecordOpts], source.Guard)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	var attrs map[string]any
	if err == nil {
		attrs, err = pythonFlavorUpdateAttrs(config)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err != nil {
		return fail(nil, err)
	}
	if len(attrs) == 0 {
		return &FlavorUpdateRecord{}, nil
	}
	target := source.Client.ServiceURL("flavors", url.PathEscape(flavorID))
	body := map[string]any{"flavor": attrs}
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPut, target, body, nil, pythonFlavorUpdateCodes()...)
	if err != nil {
		return fail(nil, err)
	}
	value, err := pythonFlavorUpdateRecord(response)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return fail(value, response.Fail(err))
	}
	return value, nil
}

// pythonFlavorUpdateAttrs merges the typed attributes and extension fields.
// Extensions cannot replace a typed key even when that key is omitted.
func pythonFlavorUpdateAttrs(config request.Config[UpdateRecordOpts]) (map[string]any, error) {
	if _, exists := config.Fields["id"]; exists {
		return nil, fmt.Errorf("%w: flavor id is the URI identity", resource.ErrInvalidOption)
	}
	attrs, err := request.MergeFieldsFor(map[string]any{}, config.Fields, config.Options)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, err
	}
	var typed map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &typed); err != nil {
		return nil, err
	}
	for key, value := range typed {
		attrs[key] = value
	}
	return attrs, nil
}

// pythonFlavorUpdateCodes is raise_from_response: every status below 400.
func pythonFlavorUpdateCodes() []int {
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	return codes
}

// pythonFlavorUpdateRecord follows _translate_response: the flavor envelope
// is unwrapped when present and an empty or non-JSON body leaves no fields.
func pythonFlavorUpdateRecord(response *rest.Response) (*FlavorUpdateRecord, error) {
	value := &FlavorUpdateRecord{Header: response.Header.Clone(), StatusCode: response.StatusCode, Sent: true}
	trimmed := bytes.TrimSpace(response.Body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return value, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return value, fmt.Errorf("%w: flavor response must be a JSON object", resource.ErrInvalidOption)
	}
	object := trimmed
	if nested, exists := fields["flavor"]; exists {
		fields = nil
		if err := json.Unmarshal(nested, &fields); err != nil || fields == nil {
			return value, fmt.Errorf("%w: flavor envelope must be a JSON object", resource.ErrInvalidOption)
		}
		object = nested
	}
	typed := new(Flavor)
	if err := json.Unmarshal(object, typed); err != nil {
		return value, err
	}
	value.Flavor, value.Fields = typed, fields
	return value, nil
}
