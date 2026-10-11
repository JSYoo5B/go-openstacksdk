package routers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func (a *API) captureRouterJSONAction(ctx context.Context, id string) (*cloudread.Source, func(context.Context) error, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, nil, err
	}
	var selected *gophercloud.ServiceClient
	if a != nil {
		selected = a.client
	}
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, nil, err
	}
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && a.client != selected {
			observed = routerActionInvalid("selected Neutron client changed")
		}
		return errors.Join(observed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err := errors.Join(routerActionID(id), guard(ctx)); err != nil {
		return nil, nil, err
	}
	return source, guard, nil
}
func routerInterfaceJSON(raw json.RawMessage) error {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return routerActionInvalid("interface/gateway value must be complete UTF-8 JSON")
	}
	return nil
}
func routerInterfaceTruthy(raw json.RawMessage) (bool, error) {
	if err := routerInterfaceJSON(raw); err != nil {
		return false, err
	}
	token := bytes.TrimSpace(raw)
	if (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		number, err := strconv.ParseFloat(string(token), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return false, err
		}
		return number != 0, nil
	}
	return cloudfilter.PythonTruthy(raw)
}
func routerInterfaceBody(value RouterInterfaceOpts, add bool) (map[string]json.RawMessage, error) {
	port, subnet := value.Port, value.Subnet
	if value.PortID != nil {
		port = value.PortID
	}
	if value.SubnetID != nil {
		subnet = value.SubnetID
	}
	if port != nil {
		if err := routerInterfaceJSON(port.Value); err != nil {
			return nil, err
		}
	}
	body := map[string]json.RawMessage{"subnet_id": json.RawMessage("null")}
	truth := false
	var err error
	if port != nil {
		truth = port.Resource
		if !truth {
			truth, err = routerInterfaceTruthy(port.Value)
		}
		if err != nil {
			return nil, err
		}
	}
	if truth {
		body = map[string]json.RawMessage{"port_id": bytes.Clone(port.Value)}
	} else if subnet != nil {
		if err := routerInterfaceJSON(subnet.Value); err != nil {
			return nil, err
		}
		body["subnet_id"] = bytes.Clone(subnet.Value)
	}
	if value.AdvertiseHost != nil {
		if !add {
			return nil, routerActionInvalid("advertise_host is an add_interface_to_router argument only")
		}
		truth, err := routerInterfaceTruthy(value.AdvertiseHost)
		if err != nil {
			return nil, err
		}
		if truth {
			body["advertise_host"] = bytes.Clone(value.AdvertiseHost)
		}
	}
	return body, nil
}
func routerJSONCodes(first, last int) []int {
	codes := make([]int, last-first)
	for index := range codes {
		codes[index] = first + index
	}
	return codes
}

// routerJSONHeaderPolicy fixes the selected protocol and JSON framing through
// native retry hooks. Ordinary retry headers and the live authentication token
// continue to use the shared REST/native request policy.
func routerJSONHeaderPolicy(source *cloudread.Source) rest.RequestHeaderPolicy {
	contentType := "application/json"
	for key, value := range source.Client.MoreHeaders {
		if strings.EqualFold(key, "Content-Type") {
			contentType = value
		}
	}
	policy := rest.RequestHeaderPolicy{
		Values: map[string]string{"Content-Type": contentType},
		Absent: []string{
			"Authorization", "X-Service-Token", "Cookie", "Host",
			"Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade",
			"X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version",
		},
	}
	if source.Client.Microversion == "" {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version")
	} else {
		policy.Values["OpenStack-API-Version"] = "network " + source.Client.Microversion
	}
	return policy
}
func routerJSONActionRequest(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, target string, body any) (*RouterJSONResult, error) {
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodPut, target, body, routerJSONHeaderPolicy(source),
		rest.RejectionPolicy{Codes: routerJSONCodes(400, 600), PreserveCleanRetry: true}, routerJSONCodes(200, 400)...)
	if err != nil {
		return routerJSONObservation(response), err
	}
	value, err := routerJSONFromResponse(response)
	if err = errors.Join(err, guard(ctx)); err != nil {
		if value != nil {
			value.Value = nil
		}
		return value, response.Fail(err)
	}
	return value, nil
}
func (a *API) routerInterface(ctx context.Context, operation, route, id string, add bool, options []RouterInterfaceOption) (*RouterJSONResult, error) {
	wrap := func(err error) error { return request.Wrap(operation, "routers", cloudread.ContextError(ctx, err)) }
	source, guard, err := a.captureRouterJSONAction(ctx, id)
	if err != nil {
		return nil, wrap(err)
	}
	config := RouterInterfaceOpts{}
	for _, option := range append([]RouterInterfaceOption(nil), options...) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if option == nil {
			return nil, wrap(routerActionInvalid("nil interface option"))
		}
		next := copyRouterInterfaceOpts(config)
		callbackErr := option(&next)
		if err := errors.Join(callbackErr, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		config = copyRouterInterfaceOpts(next)
	}
	body, err := routerInterfaceBody(config, add)
	if err != nil {
		return nil, wrap(err)
	}
	if err := source.WithPolicy(ctx, nil, config.Headers); err != nil {
		return nil, wrap(err)
	}
	target := source.Client.ServiceURL("routers", url.PathEscape(id), route)
	value, err := routerJSONActionRequest(ctx, source, guard, target, body)
	return value, wrap(err)
}

// AddInterfaceRecord follows Source port/subnet selection and advertise_host
// defaults. It returns the whole parsed response JSON without Router descriptor
// defaults, envelope extraction or an ID/cache seed. The server validates IDs.
func (a *API) AddInterfaceRecord(ctx context.Context, id string, options ...RouterInterfaceOption) (*RouterJSONResult, error) {
	return a.routerInterface(ctx, "AddInterfaceRecord", "add_router_interface", id, true, options)
}

// RemoveInterfaceRecord follows Source's port-first selection and explicit
// subnet null default. A clean 404 is an error and advertise_host is unsupported.
func (a *API) RemoveInterfaceRecord(ctx context.Context, id string, options ...RouterInterfaceOption) (*RouterJSONResult, error) {
	return a.routerInterface(ctx, "RemoveInterfaceRecord", "remove_router_interface", id, false, options)
}

func routerGatewayBody(value RouterGatewayOpts, add bool) (any, error) {
	fields := copyRouterGatewayOpts(value).Fields
	for key, raw := range fields {
		if !utf8.ValidString(key) {
			return nil, routerActionInvalid("gateway field must be UTF-8")
		}
		if key == "router" || key == "session" || key == "self" {
			return nil, routerActionInvalid("%q conflicts with a pinned Source method argument", key)
		}
		if err := routerInterfaceJSON(raw); err != nil {
			return nil, err
		}
	}
	if value.LegacySourceRoutes {
		return fields, nil
	}
	if !add {
		if len(fields) != 0 {
			return nil, routerActionInvalid("stock gateway removal does not accept ignored attributes; use explicit legacy route compatibility for Source body attributes")
		}
		return map[string]any{"router": map[string]any{"external_gateway_info": nil}}, nil
	}
	var network string
	if err := json.Unmarshal(fields["network_id"], &network); err != nil || strings.TrimSpace(network) == "" {
		return nil, routerActionInvalid("stock AddGatewayRecord requires a nonblank string network_id to prevent an empty gateway body from clearing an existing gateway")
	}
	return map[string]any{"router": map[string]any{"external_gateway_info": fields}}, nil
}
func (a *API) routerGateway(ctx context.Context, operation, id string, add bool, options []RouterGatewayOption) (*RouterJSONResult, error) {
	wrap := func(err error) error { return request.Wrap(operation, "routers", cloudread.ContextError(ctx, err)) }
	source, guard, err := a.captureRouterJSONAction(ctx, id)
	if err != nil {
		return nil, wrap(err)
	}
	config := RouterGatewayOpts{}
	for _, option := range append([]RouterGatewayOption(nil), options...) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if option == nil {
			return nil, wrap(routerActionInvalid("nil gateway option"))
		}
		next := copyRouterGatewayOpts(config)
		callbackErr := option(&next)
		if err := errors.Join(callbackErr, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		config = copyRouterGatewayOpts(next)
	}
	body, err := routerGatewayBody(config, add)
	if err != nil {
		return nil, wrap(err)
	}
	if err := source.WithPolicy(ctx, nil, config.Headers); err != nil {
		return nil, wrap(err)
	}
	target := source.Client.ServiceURL("routers", url.PathEscape(id))
	if config.LegacySourceRoutes {
		route := "remove_gateway_router"
		if add {
			route = "add_gateway_router"
		}
		target = source.Client.ServiceURL("routers", url.PathEscape(id), route)
	}
	value, err := routerJSONActionRequest(ctx, source, guard, target, body)
	return value, wrap(err)
}

// AddGatewayRecord configures a single external gateway with stock Neutron's
// PUT /routers/{id} external_gateway_info API. This intentionally repairs the
// absent legacy Source action route. Requires network_id; explicit Source route
// compatibility is optional and retains strict native HTTP error handling.
func (a *API) AddGatewayRecord(ctx context.Context, id string, options ...RouterGatewayOption) (*RouterJSONResult, error) {
	return a.routerGateway(ctx, "AddGatewayRecord", id, true, options)
}

// RemoveGatewayRecord clears the single external gateway with an explicit null
// external_gateway_info on stock Neutron. Unknown removal fields are never
// silently discarded; they require an explicitly supported legacy action route.
func (a *API) RemoveGatewayRecord(ctx context.Context, id string, options ...RouterGatewayOption) (*RouterJSONResult, error) {
	return a.routerGateway(ctx, "RemoveGatewayRecord", id, false, options)
}
