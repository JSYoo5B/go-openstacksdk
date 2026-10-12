package siteconnections

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

// UpdateRecordOpts carries the Python update_vpn_ipsec_site_connection
// attributes that the generated UpdateOpts drops when empty. An omitted field
// is not sent, Null sends JSON null and Present keeps empty strings, zero and
// empty lists. Other declared VpnIPSecSiteConnection attributes such as
// auth_mode, route_mode, ikepolicy_id or vpnservice_id use
// WithUpdateRecordField; the server validates every value.
type UpdateRecordOpts struct {
	Name           request.Optional[string]         `json:"name,omitzero"`
	Description    request.Optional[string]         `json:"description,omitzero"`
	LocalID        request.Optional[string]         `json:"local_id,omitzero"`
	PeerAddress    request.Optional[string]         `json:"peer_address,omitzero"`
	PeerID         request.Optional[string]         `json:"peer_id,omitzero"`
	PeerCIDRs      request.Optional[[]string]       `json:"peer_cidrs,omitzero"`
	LocalEPGroupID request.Optional[string]         `json:"local_ep_group_id,omitzero"`
	PeerEPGroupID  request.Optional[string]         `json:"peer_ep_group_id,omitzero"`
	MTU            request.Optional[int]            `json:"mtu,omitzero"`
	Initiator      request.Optional[string]         `json:"initiator,omitzero"`
	PSK            request.Optional[string]         `json:"psk,omitzero"`
	DPD            request.Optional[map[string]any] `json:"dpd,omitzero"`
	AdminStateUp   request.Optional[bool]           `json:"admin_state_up,omitzero"`
}

type UpdateRecordOption = request.Option[UpdateRecordOpts]

func WithUpdateRecordOptions(value UpdateRecordOpts) UpdateRecordOption {
	return request.WithOptions(value)
}

// WithUpdateRecordField adds another attribute inside the
// ipsec_site_connection envelope.
// Keys owned by UpdateRecordOpts and the URI identity id are rejected.
func WithUpdateRecordField(key string, value any) UpdateRecordOption {
	return request.WithField[UpdateRecordOpts](key, value)
}
func WithUpdateRecordHeader(key, value string) UpdateRecordOption {
	return request.WithHeader[UpdateRecordOpts](key, value)
}

// ConnectionRecord is the committed Python VpnIPSecSiteConnection view.
// Connection is the native typed decode of the response object and Fields
// keeps that object unchanged. Sent is false when no attribute was dirty and
// nothing was sent.
type ConnectionRecord struct {
	Connection *Connection
	Fields     map[string]json.RawMessage
	Header     http.Header
	StatusCode int
	Sent       bool
}

// UpdateRecord implements update_vpn_ipsec_site_connection: Resource.commit
// sends PUT vpn/ipsec-site-connections/{id} with only the supplied attributes
// in {"ipsec_site_connection": ...} and accepts every status below 400. With
// no attribute it sends nothing.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...UpdateRecordOption) (*ConnectionRecord, error) {
	const op = "UpdateRecord"
	fail := func(value *ConnectionRecord, err error) (*ConnectionRecord, error) {
		return value, request.Wrap(op, "siteconnections", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: IPsec site connection API is required", resource.ErrInvalidOption))
	}
	if id := strings.TrimSpace(id); id == "" || id == "." || id == ".." {
		return fail(nil, fmt.Errorf("%w: IPsec site connection ID is required", resource.ErrInvalidOption))
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
		attrs, err = pythonSiteConnectionAttrs(config)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err != nil {
		return fail(nil, err)
	}
	if len(attrs) == 0 {
		return &ConnectionRecord{}, nil
	}
	target := source.Client.ServiceURL("vpn", "ipsec-site-connections", url.PathEscape(id))
	body := map[string]any{"ipsec_site_connection": attrs}
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPut, target, body, nil, pythonSiteConnectionCodes()...)
	if err != nil {
		return fail(nil, err)
	}
	value, err := pythonSiteConnectionRecord(response)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return fail(value, response.Fail(err))
	}
	return value, nil
}

// pythonSiteConnectionAttrs merges the typed attributes and extension fields.
// Extensions cannot replace a typed key even when that key is omitted.
func pythonSiteConnectionAttrs(config request.Config[UpdateRecordOpts]) (map[string]any, error) {
	if _, exists := config.Fields["id"]; exists {
		return nil, fmt.Errorf("%w: IPsec site connection id is the URI identity", resource.ErrInvalidOption)
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

// pythonSiteConnectionCodes is raise_from_response: every status below 400.
func pythonSiteConnectionCodes() []int {
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	return codes
}

// pythonSiteConnectionRecord follows _translate_response: the ipsec_site_connection envelope is
// unwrapped when present and an empty or non-JSON body leaves no fields.
func pythonSiteConnectionRecord(response *rest.Response) (*ConnectionRecord, error) {
	value := &ConnectionRecord{Header: response.Header.Clone(), StatusCode: response.StatusCode, Sent: true}
	trimmed := bytes.TrimSpace(response.Body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return value, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return value, fmt.Errorf("%w: IPsec site connection response must be a JSON object", resource.ErrInvalidOption)
	}
	object := trimmed
	if nested, exists := fields["ipsec_site_connection"]; exists {
		fields = nil
		if err := json.Unmarshal(nested, &fields); err != nil || fields == nil {
			return value, fmt.Errorf("%w: IPsec site connection envelope must be a JSON object", resource.ErrInvalidOption)
		}
		object = nested
	}
	typed := new(Connection)
	if err := json.Unmarshal(object, typed); err != nil {
		return value, err
	}
	value.Connection, value.Fields = typed, fields
	return value, nil
}
