package routers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func routerActionID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return routerActionInvalid("router ID must be a nonblank UTF-8 segment")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return routerActionInvalid("router ID contains controls")
		}
	}
	return nil
}

func (a *API) routerAction(ctx context.Context, operation, action, id string, body any, options []RouterActionOption) (*RouterActionRecord, error) {
	wrap := func(err error) error { return request.Wrap(operation, "routers", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	var selected *gophercloud.ServiceClient
	if a != nil {
		selected = a.client
	}
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, wrap(err)
	}
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && a.client != selected {
			observed = routerActionInvalid("selected Neutron client changed")
		}
		return errors.Join(observed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err := errors.Join(routerActionID(id), guard(ctx)); err != nil {
		return nil, wrap(err)
	}
	// Source passes the positional body directly as session.put(json=body).
	// Snapshot before option callbacks can mutate caller-owned JSON values.
	var ownedBody any
	if body != nil {
		raw, marshalErr := json.Marshal(body)
		if marshalErr == nil && (!utf8.Valid(raw) || !json.Valid(raw)) {
			marshalErr = routerActionInvalid("action body must be UTF-8 JSON")
		}
		if err := errors.Join(marshalErr, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		ownedBody = json.RawMessage(append([]byte(nil), raw...))
	}
	config := RouterActionOpts{}
	for _, option := range append([]RouterActionOption(nil), options...) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if option == nil {
			return nil, wrap(routerActionInvalid("nil action option"))
		}
		next := copyRouterActionOpts(config)
		callbackErr := option(&next)
		if err := errors.Join(callbackErr, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		config = copyRouterActionOpts(next)
	}
	seed, seededMatch, err := routerActionSeed(id, config.Seed)
	if err != nil {
		return nil, wrap(err)
	}
	if err := source.WithPolicy(ctx, nil, config.Headers); err != nil {
		return nil, wrap(err)
	}
	if err := guard(ctx); err != nil {
		return nil, wrap(err)
	}
	target := source.Client.ServiceURL("routers", url.PathEscape(id), action)
	// raise_from_response accepts all non-error statuses. Observe every native
	// rejection's physical Read/Close failures without converting it to success.
	accepted, rejected := make([]int, 200), make([]int, 200)
	for i := range accepted {
		accepted[i], rejected[i] = 200+i, 400+i
	}
	response, err := rest.DoJSONGuardedRejections(ctx, &source.Client, guard, http.MethodPut, target, ownedBody, nil,
		rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, accepted...)
	if err != nil {
		return routerActionObservation(response), wrap(err)
	}
	value, err := routerActionFromResponse(response, seed, seededMatch)
	err = errors.Join(err, guard(ctx))
	if err != nil {
		return value, wrap(response.Fail(err))
	}
	return value, nil
}

// AddExtraRoutesRecord implements the pinned add_extra_routes_to_router proxy.
// Body is the complete caller-supplied JSON, including any router envelope;
// this method never fetches the router or synthesizes a body from descriptors.
func (a *API) AddExtraRoutesRecord(ctx context.Context, id string, body any, options ...RouterActionOption) (*RouterActionRecord, error) {
	return a.routerAction(ctx, "AddExtraRoutesRecord", "add_extraroutes", id, body, options)
}

// RemoveExtraRoutesRecord removes the explicitly supplied extra routes.
// The response descriptor view contains actual response fields and an ID seed.
func (a *API) RemoveExtraRoutesRecord(ctx context.Context, id string, body any, options ...RouterActionOption) (*RouterActionRecord, error) {
	return a.routerAction(ctx, "RemoveExtraRoutesRecord", "remove_extraroutes", id, body, options)
}

// AddExternalGatewaysRecord implements the Source multi-gateway action.
// Stock Neutron permits project members to select network_id, while supplied
// enable_snat and external_fixed_ips require administrator policy privileges.
func (a *API) AddExternalGatewaysRecord(ctx context.Context, id string, body any, options ...RouterActionOption) (*RouterActionRecord, error) {
	return a.routerAction(ctx, "AddExternalGatewaysRecord", "add_external_gateways", id, body, options)
}

// UpdateExternalGatewaysRecord submits the complete multi-gateway action body.
// Unknown extensions and explicit null, false, zero and empty collections stay
// in the body. The server remains responsible for authorization and validation.
func (a *API) UpdateExternalGatewaysRecord(ctx context.Context, id string, body any, options ...RouterActionOption) (*RouterActionRecord, error) {
	return a.routerAction(ctx, "UpdateExternalGatewaysRecord", "update_external_gateways", id, body, options)
}

// RemoveExternalGatewaysRecord implements the Source multi-gateway removal.
// A 404 remains an error: these actions do not have an ignore_missing default.
func (a *API) RemoveExternalGatewaysRecord(ctx context.Context, id string, body any, options ...RouterActionOption) (*RouterActionRecord, error) {
	return a.routerAction(ctx, "RemoveExternalGatewaysRecord", "remove_external_gateways", id, body, options)
}
