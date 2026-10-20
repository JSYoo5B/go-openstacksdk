package bgpvpns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateRecordOpts carries the declared Python BgpVpn attributes for
// create_bgpvpn. An omitted field is not sent, Null sends JSON null and
// Present keeps zero, empty strings and empty lists, so local_pref=0 and
// vni=0 reach the server. BgpVpn allows unknown body attributes, so other keys
// such as tenant_id use WithCreateRecordField; the server validates every value.
type CreateRecordOpts struct {
	Name                request.Optional[string]   `json:"name,omitzero"`
	ProjectID           request.Optional[string]   `json:"project_id,omitzero"`
	Type                request.Optional[string]   `json:"type,omitzero"`
	RouteDistinguishers request.Optional[[]string] `json:"route_distinguishers,omitzero"`
	RouteTargets        request.Optional[[]string] `json:"route_targets,omitzero"`
	ImportTargets       request.Optional[[]string] `json:"import_targets,omitzero"`
	ExportTargets       request.Optional[[]string] `json:"export_targets,omitzero"`
	LocalPref           request.Optional[int]      `json:"local_pref,omitzero"`
	VNI                 request.Optional[int]      `json:"vni,omitzero"`
}

type CreateRecordOption = request.Option[CreateRecordOpts]

func WithCreateRecordOptions(value CreateRecordOpts) CreateRecordOption {
	return request.WithOptions(value)
}
func WithCreateRecordLocalPref(value request.Optional[int]) CreateRecordOption {
	return func(config *request.Config[CreateRecordOpts]) error { config.Options.LocalPref = value; return nil }
}

// WithCreateRecordField adds another attribute inside the bgpvpn envelope.
// Keys owned by CreateRecordOpts are rejected.
func WithCreateRecordField(key string, value any) CreateRecordOption {
	return request.WithField[CreateRecordOpts](key, value)
}
func WithCreateRecordHeader(key, value string) CreateRecordOption {
	return request.WithHeader[CreateRecordOpts](key, value)
}

// BGPVPNRecord is the created Python BgpVpn view. BGPVPN is the native typed
// decode of the response object and Fields keeps that object unchanged.
type BGPVPNRecord struct {
	BGPVPN     *BGPVPN
	Fields     map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// CreateRecord implements create_bgpvpn: Resource.create sends POST
// bgpvpn/bgpvpns with the supplied attributes in {"bgpvpn": ...}, even when
// none are supplied, and accepts every status below 400.
func (a *API) CreateRecord(ctx context.Context, options ...CreateRecordOption) (*BGPVPNRecord, error) {
	const op = "CreateRecord"
	fail := func(value *BGPVPNRecord, err error) (*BGPVPNRecord, error) {
		return value, request.Wrap(op, "bgpvpns", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: BGP VPN API is required", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, a.client, "network")
	if err != nil {
		return fail(nil, err)
	}
	config, err := cloudread.ApplyReadOptions(ctx, CreateRecordOpts{}, options, cloudread.OwnReadConfig[CreateRecordOpts], source.Guard)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	var attrs map[string]any
	if err == nil {
		attrs, err = pythonBGPVPNAttrs(config)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err != nil {
		return fail(nil, err)
	}
	target := source.Client.ServiceURL("bgpvpn", "bgpvpns")
	body := map[string]any{"bgpvpn": attrs}
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPost, target, body, nil, pythonBGPVPNCodes()...)
	if err != nil {
		return fail(nil, err)
	}
	value, err := pythonBGPVPNRecord(response)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return fail(value, response.Fail(err))
	}
	return value, nil
}

// pythonBGPVPNAttrs merges the typed attributes and extension fields.
// Extensions cannot replace a typed key even when that key is omitted.
func pythonBGPVPNAttrs(config request.Config[CreateRecordOpts]) (map[string]any, error) {
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

// pythonBGPVPNCodes is raise_from_response: every status below 400.
func pythonBGPVPNCodes() []int {
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	return codes
}

// pythonBGPVPNRecord follows _translate_response: the bgpvpn envelope is
// unwrapped when present and an empty or non-JSON body leaves no fields.
func pythonBGPVPNRecord(response *rest.Response) (*BGPVPNRecord, error) {
	value := &BGPVPNRecord{Header: response.Header.Clone(), StatusCode: response.StatusCode}
	trimmed := bytes.TrimSpace(response.Body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return value, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return value, fmt.Errorf("%w: BGP VPN response must be a JSON object", resource.ErrInvalidOption)
	}
	object := trimmed
	if nested, exists := fields["bgpvpn"]; exists {
		fields = nil
		if err := json.Unmarshal(nested, &fields); err != nil || fields == nil {
			return value, fmt.Errorf("%w: BGP VPN envelope must be a JSON object", resource.ErrInvalidOption)
		}
		object = nested
	}
	typed := new(BGPVPN)
	if err := json.Unmarshal(object, typed); err != nil {
		return value, err
	}
	value.BGPVPN, value.Fields = typed, fields
	return value, nil
}
