package speakers

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

// UpdateRecordOpts carries the Python update_bgp_speaker attributes. An omitted
// field is not sent, so a name-only update leaves both advertise flags alone.
// Null sends JSON null and Present keeps false and empty strings. BgpSpeaker
// allows unknown body attributes, so other keys use WithUpdateRecordField;
// the server validates every value.
type UpdateRecordOpts struct {
	Name                          request.Optional[string] `json:"name,omitzero"`
	AdvertiseFloatingIPHostRoutes request.Optional[bool]   `json:"advertise_floating_ip_host_routes,omitzero"`
	AdvertiseTenantNetworks       request.Optional[bool]   `json:"advertise_tenant_networks,omitzero"`
}

type UpdateRecordOption = request.Option[UpdateRecordOpts]

func WithUpdateRecordOptions(value UpdateRecordOpts) UpdateRecordOption {
	return request.WithOptions(value)
}
func WithUpdateRecordName(value request.Optional[string]) UpdateRecordOption {
	return func(config *request.Config[UpdateRecordOpts]) error { config.Options.Name = value; return nil }
}
func WithUpdateRecordAdvertiseFloatingIPHostRoutes(value request.Optional[bool]) UpdateRecordOption {
	return func(config *request.Config[UpdateRecordOpts]) error {
		config.Options.AdvertiseFloatingIPHostRoutes = value
		return nil
	}
}
func WithUpdateRecordAdvertiseTenantNetworks(value request.Optional[bool]) UpdateRecordOption {
	return func(config *request.Config[UpdateRecordOpts]) error {
		config.Options.AdvertiseTenantNetworks = value
		return nil
	}
}

// WithUpdateRecordField adds another attribute inside the bgp_speaker envelope.
// Keys owned by UpdateRecordOpts and the URI identity id are rejected.
func WithUpdateRecordField(key string, value any) UpdateRecordOption {
	return request.WithField[UpdateRecordOpts](key, value)
}
func WithUpdateRecordHeader(key, value string) UpdateRecordOption {
	return request.WithHeader[UpdateRecordOpts](key, value)
}

// BGPSpeakerRecord is the committed Python BgpSpeaker view. BGPSpeaker is the
// native typed decode of the response object and Fields keeps that object
// unchanged. Sent is false when no attribute was dirty and nothing was sent.
type BGPSpeakerRecord struct {
	BGPSpeaker *BGPSpeaker
	Fields     map[string]json.RawMessage
	Header     http.Header
	StatusCode int
	Sent       bool
}

// UpdateRecord implements update_bgp_speaker: Resource.commit sends PUT
// bgp-speakers/{id} with only the supplied attributes in {"bgp_speaker": ...} and
// accepts every status below 400. With no attribute it sends nothing.
func (a *API) UpdateRecord(ctx context.Context, speakerID string, options ...UpdateRecordOption) (*BGPSpeakerRecord, error) {
	const op = "UpdateRecord"
	fail := func(value *BGPSpeakerRecord, err error) (*BGPSpeakerRecord, error) {
		return value, request.Wrap(op, "speakers", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: BGP speaker API is required", resource.ErrInvalidOption))
	}
	if id := strings.TrimSpace(speakerID); id == "" || id == "." || id == ".." {
		return fail(nil, fmt.Errorf("%w: BGP speaker ID is required", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, a.client, "network")
	if err != nil {
		return fail(nil, err)
	}
	config, err := cloudread.ApplyReadOptions(ctx, UpdateRecordOpts{}, options, cloudread.OwnReadConfig[UpdateRecordOpts], source.Guard)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	var attrs map[string]any
	if err == nil {
		attrs, err = pythonBGPSpeakerAttrs(config)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err != nil {
		return fail(nil, err)
	}
	if len(attrs) == 0 {
		return &BGPSpeakerRecord{}, nil
	}
	target := source.Client.ServiceURL("bgp-speakers", url.PathEscape(speakerID))
	body := map[string]any{"bgp_speaker": attrs}
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPut, target, body, nil, pythonBGPSpeakerCodes()...)
	if err != nil {
		return fail(nil, err)
	}
	value, err := pythonBGPSpeakerRecord(response)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return fail(value, response.Fail(err))
	}
	return value, nil
}

// pythonBGPSpeakerAttrs merges the typed attributes and extension fields.
// Extensions cannot replace a typed key even when that key is omitted.
func pythonBGPSpeakerAttrs(config request.Config[UpdateRecordOpts]) (map[string]any, error) {
	if _, exists := config.Fields["id"]; exists {
		return nil, fmt.Errorf("%w: BGP speaker id is the URI identity", resource.ErrInvalidOption)
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

// pythonBGPSpeakerCodes is raise_from_response: every status below 400.
func pythonBGPSpeakerCodes() []int {
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	return codes
}

// pythonBGPSpeakerRecord follows _translate_response: the bgp_speaker envelope is
// unwrapped when present and an empty or non-JSON body leaves no fields.
func pythonBGPSpeakerRecord(response *rest.Response) (*BGPSpeakerRecord, error) {
	value := &BGPSpeakerRecord{Header: response.Header.Clone(), StatusCode: response.StatusCode, Sent: true}
	trimmed := bytes.TrimSpace(response.Body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return value, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return value, fmt.Errorf("%w: BGP speaker response must be a JSON object", resource.ErrInvalidOption)
	}
	object := trimmed
	if nested, exists := fields["bgp_speaker"]; exists {
		fields = nil
		if err := json.Unmarshal(nested, &fields); err != nil || fields == nil {
			return value, fmt.Errorf("%w: BGP speaker envelope must be a JSON object", resource.ErrInvalidOption)
		}
		object = nested
	}
	typed := new(BGPSpeaker)
	if err := json.Unmarshal(object, typed); err != nil {
		return value, err
	}
	value.BGPSpeaker, value.Fields = typed, fields
	return value, nil
}
