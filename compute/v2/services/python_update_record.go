package services

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
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateRecordOpts carries the Python update_service attributes. An omitted
// field is not sent, while Present keeps false and empty strings and Null
// sends JSON null. Other declared Service attributes use WithUpdateRecordField
// with their server keys; the server validates every value.
type UpdateRecordOpts struct {
	Status         request.Optional[string] `json:"status,omitzero"`
	DisabledReason request.Optional[string] `json:"disabled_reason,omitzero"`
	ForcedDown     request.Optional[bool]   `json:"forced_down,omitzero"`
}

type UpdateRecordOption = request.Option[UpdateRecordOpts]

func WithUpdateRecordOptions(value UpdateRecordOpts) UpdateRecordOption {
	return request.WithOptions(value)
}
func WithUpdateRecordStatus(value request.Optional[string]) UpdateRecordOption {
	return func(config *request.Config[UpdateRecordOpts]) error { config.Options.Status = value; return nil }
}
func WithUpdateRecordDisabledReason(value request.Optional[string]) UpdateRecordOption {
	return func(config *request.Config[UpdateRecordOpts]) error {
		config.Options.DisabledReason = value
		return nil
	}
}
func WithUpdateRecordForcedDown(value request.Optional[bool]) UpdateRecordOption {
	return func(config *request.Config[UpdateRecordOpts]) error { config.Options.ForcedDown = value; return nil }
}

// WithUpdateRecordField adds another attribute to the unwrapped body. Keys
// owned by UpdateRecordOpts and the URI identity id are rejected.
func WithUpdateRecordField(key string, value any) UpdateRecordOption {
	return request.WithField[UpdateRecordOpts](key, value)
}
func WithUpdateRecordHeader(key, value string) UpdateRecordOption {
	return request.WithHeader[UpdateRecordOpts](key, value)
}

// ServiceRecord is the committed Python Service view. Service is the native
// typed decode of the response object and Fields keeps that object unchanged.
// Legacy force-down responses carry no id, so Service stays nil for them.
// Sent is false when no attribute was dirty and nothing was sent.
type ServiceRecord struct {
	Service    *Service
	Fields     map[string]json.RawMessage
	Header     http.Header
	StatusCode int
	Sent       bool
}

// UpdateRecord implements update_service: below microversion 2.53 it fails
// before HTTP like Python's SDKException. Otherwise Service.commit sends PUT
// os-services/{id} with only the supplied attributes in an unwrapped body
// (prepend_key False) and accepts every status below 400. With no attribute
// it sends nothing.
func (a *API) UpdateRecord(ctx context.Context, serviceID string, options ...UpdateRecordOption) (*ServiceRecord, error) {
	const op = "UpdateRecord"
	fail := func(value *ServiceRecord, err error) (*ServiceRecord, error) {
		return value, request.Wrap(op, "services", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: service API is required", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, a.client, "compute")
	if err != nil {
		return fail(nil, err)
	}
	if err := pythonServiceRequire(source.Client.Microversion, "2.53"); err != nil {
		return fail(nil, err)
	}
	if strings.TrimSpace(serviceID) == "" {
		return fail(nil, fmt.Errorf("%w: service id is required", resource.ErrInvalidOption))
	}
	config, err := cloudread.ApplyReadOptions(ctx, UpdateRecordOpts{}, options, cloudread.OwnReadConfig[UpdateRecordOpts], source.Guard)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	var attrs map[string]any
	if err == nil {
		attrs, err = pythonServiceAttrs(config)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err != nil {
		return fail(nil, err)
	}
	if len(attrs) == 0 {
		return &ServiceRecord{}, nil
	}
	return pythonServicePut(ctx, source, op, serviceID, attrs)
}

// pythonServicePut sends the committed body to os-services/{id} and decodes
// the typed Service view.
func pythonServicePut(ctx context.Context, source *cloudread.Source, op, serviceID string, attrs map[string]any) (*ServiceRecord, error) {
	target := source.Client.ServiceURL("os-services", url.PathEscape(serviceID))
	return pythonServiceSend(ctx, source, op, target, attrs, true)
}

func pythonServiceSend(ctx context.Context, source *cloudread.Source, op, target string, body map[string]any, typed bool) (*ServiceRecord, error) {
	fail := func(value *ServiceRecord, err error) (*ServiceRecord, error) {
		return value, request.Wrap(op, "services", cloudread.ContextError(ctx, err))
	}
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPut, target, body, nil, pythonServiceCodes()...)
	if err != nil {
		return fail(nil, err)
	}
	value, err := pythonServiceRecord(response, typed)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return fail(value, response.Fail(err))
	}
	return value, nil
}

// pythonServiceSupports follows supports_microversion for the client's one
// selected microversion. An unset microversion is Nova's 2.1 default and
// latest is not a finite-major match, as in keystoneauth version_match.
func pythonServiceSupports(selected, required string) (bool, error) {
	if selected == "" {
		return false, nil
	}
	ok, err := microversions.Matches(selected, required)
	if err != nil {
		return false, fmt.Errorf("%w: %v", resource.ErrInvalidOption, err)
	}
	return ok, nil
}

func pythonServiceRequire(selected, required string) error {
	ok, err := pythonServiceSupports(selected, required)
	if err == nil && !ok {
		err = fmt.Errorf("%w: method requires at least microversion %s (client uses %q)", resource.ErrUnsupported, required, selected)
	}
	return err
}

// pythonServiceAttrs merges the typed attributes and extension fields.
// Extensions cannot replace a typed key even when that key is omitted.
func pythonServiceAttrs(config request.Config[UpdateRecordOpts]) (map[string]any, error) {
	if _, exists := config.Fields["id"]; exists {
		return nil, fmt.Errorf("%w: service id is the URI identity", resource.ErrInvalidOption)
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

// pythonServiceCodes is raise_from_response: every status below 400.
func pythonServiceCodes() []int {
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	return codes
}

// pythonServiceRecord follows _translate_response: the service envelope is
// unwrapped when present and an empty or non-JSON body leaves no fields.
func pythonServiceRecord(response *rest.Response, typed bool) (*ServiceRecord, error) {
	value := &ServiceRecord{Header: response.Header.Clone(), StatusCode: response.StatusCode, Sent: true}
	trimmed := bytes.TrimSpace(response.Body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return value, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return value, fmt.Errorf("%w: service response must be a JSON object", resource.ErrInvalidOption)
	}
	object := trimmed
	if nested, exists := fields["service"]; exists {
		fields = nil
		if err := json.Unmarshal(nested, &fields); err != nil || fields == nil {
			return value, fmt.Errorf("%w: service envelope must be a JSON object", resource.ErrInvalidOption)
		}
		object = nested
	}
	value.Fields = fields
	if typed {
		service := new(Service)
		if err := json.Unmarshal(object, service); err != nil {
			return value, err
		}
		value.Service = service
	}
	return value, nil
}
