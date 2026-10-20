package volumes

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
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ConnectorOpts is the concrete configuration carrier for owned connector actions.
// Connector values are supplied separately and are snapshotted before options.
type ConnectorOpts struct{}
type ConnectorOption = request.Option[ConnectorOpts]

func WithConnectorHeader(key, value string) ConnectorOption {
	return request.WithHeader[ConnectorOpts](key, value)
}
func WithConnectorHeaders(values map[string]string) ConnectorOption {
	owned := maps.Clone(values)
	return func(cfg *request.Config[ConnectorOpts]) error {
		cfg.Headers = maps.Clone(owned)
		return nil
	}
}

// VolumeConnectorResponse is actual HTTP evidence, not a reconstructed Volume.
type VolumeConnectorResponse = rest.Response

// VolumeConnectorResult keeps discovery and action acknowledgements separately.
// ConnectionInfo is populated only by successful InitializeWithConnector. A
// nonnil "null" value differs from a missing field or a failed initialization.
type VolumeConnectorResult struct {
	VolumeID       string
	Microversion   string
	Discovery      []*VolumeConnectorResponse
	Response       *VolumeConnectorResponse
	ConnectionInfo json.RawMessage
}

// InitializeWithConnector forwards an arbitrary nested connector to
// os-initialize_connection and returns the raw connection_info JSON value.
func (a *API) InitializeWithConnector(ctx context.Context, id string, connector map[string]any, options ...ConnectorOption) (*VolumeConnectorResult, error) {
	return a.connectorAction(ctx, id, connector, true, options)
}

// TerminateWithConnector forwards an arbitrary nested connector to
// os-terminate_connection. Its response body is opaque acknowledgement evidence.
func (a *API) TerminateWithConnector(ctx context.Context, id string, connector map[string]any, options ...ConnectorOption) (*VolumeConnectorResult, error) {
	return a.connectorAction(ctx, id, connector, false, options)
}

func (a *API) connectorAction(ctx context.Context, id string, connector map[string]any, initialize bool, options []ConnectorOption) (*VolumeConnectorResult, error) {
	operation, action := "TerminateWithConnector", "os-terminate_connection"
	if initialize {
		operation, action = "InitializeWithConnector", "os-initialize_connection"
	}
	wrap := func(err error) error { return request.Wrap(operation, "volumes", cloudread.ContextError(ctx, err)) }
	var selected *gophercloud.ServiceClient
	if a != nil {
		selected = a.client
	}
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error {
		var replaced error
		if a == nil || a.client != selected {
			replaced = fmt.Errorf("%w: selected connector client changed", resource.ErrInvalidOption)
		}
		return errors.Join(replaced, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err := errors.Join(cinderaction.ValidateID(id), guard(ctx)); err != nil {
		return nil, wrap(err)
	}

	// Capture precedes custom MarshalJSON code. Options cannot later mutate the
	// caller's nested connector into a different request or route.
	encoded, marshalErr := json.Marshal(connector)
	if marshalErr == nil && !utf8.Valid(encoded) {
		marshalErr = fmt.Errorf("connector JSON must be UTF-8")
	}
	if marshalErr != nil {
		marshalErr = fmt.Errorf("%w: connector JSON: %w", resource.ErrInvalidOption, marshalErr)
	}
	if err := errors.Join(marshalErr, guard(ctx)); err != nil {
		return nil, wrap(err)
	}
	// Keep the JSON type: plain []byte inside an interface encodes as base64.
	snapshot := json.RawMessage(bytes.Clone(encoded))
	cfg, err := cloudread.ApplyReadOptions(ctx, ConnectorOpts{}, slices.Clone(options), cloudread.OwnReadConfig[ConnectorOpts], guard)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, true)
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return nil, wrap(err)
	}
	result := &VolumeConnectorResult{VolumeID: id}
	result.Microversion, result.Discovery, err = cinderrequest.Negotiate(ctx, source, "3.71")
	if err == nil {
		err = source.WithPolicy(ctx, &result.Microversion, cfg.Headers)
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	policy := rest.RequestHeaderPolicy{}
	if result.Microversion == "" {
		policy.Absent = []string{"Openstack-Api-Version", "X-Openstack-Volume-Api-Version"}
	} else {
		policy.Values = map[string]string{"OpenStack-API-Version": "volume " + result.Microversion, "X-OpenStack-Volume-API-Version": result.Microversion}
	}
	// Use the native spelling once in RequestOpts. Equal canonical aliases in
	// client headers would become duplicate logical retry values even though
	// net/http emits one field; caller values were already checked by policy.
	for key := range source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(source.Client.MoreHeaders, key)
		}
	}
	accepted, rejected := make([]int, 200), make([]int, 200)
	for index := range accepted {
		accepted[index], rejected[index] = 200+index, 400+index
	}
	body := map[string]any{action: map[string]any{"connector": snapshot}}
	target := source.Client.ServiceURL("volumes", url.PathEscape(id), "action")
	result.Response, err = rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodPost, target, body, policy, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, accepted...)
	if err != nil {
		return result, wrap(err)
	}
	if initialize {
		if !utf8.Valid(result.Response.Body) {
			return result, wrap(result.Response.Fail(fmt.Errorf("connector response must contain UTF-8 JSON")))
		}
		object, err := result.Response.Object("")
		if err != nil {
			return result, wrap(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(object, &fields); err != nil {
			return result, wrap(result.Response.Fail(err))
		}
		info, present := fields["connection_info"]
		if !present {
			return result, wrap(result.Response.Fail(fmt.Errorf("initialize response is missing connection_info")))
		}
		if err := guard(ctx); err != nil {
			return result, wrap(result.Response.Fail(err))
		}
		result.ConnectionInfo = bytes.Clone(info)
	}
	if err := guard(ctx); err != nil {
		return result, wrap(result.Response.Fail(err))
	}
	return result, nil
}
