package routers

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/responsejson"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RemoveCloudInterfaceRecord is the closed Source cloud dependency. It reuses
// owned reference selection/JSON decoding and opts into valid100..399 plus
// physical-fault callback suppression without changing RemoveInterfaceRecord.
// The cloud layer alone owns the both-missing validation and None return.
func (a *API) RemoveCloudInterfaceRecord(ctx context.Context, id string, options ...RouterInterfaceOption) (*RouterJSONResult, error) {
	wrap := func(err error) error {
		return request.Wrap("RemoveCloudInterfaceRecord", "routers", cloudread.ContextError(ctx, err))
	}
	source, guard, err := a.captureRouterJSONAction(ctx, id)
	if err != nil {
		return nil, wrap(err)
	}
	cfg := RouterInterfaceOpts{}
	for _, apply := range append([]RouterInterfaceOption(nil), options...) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(routerActionInvalid("nil cloud interface option"))
		}
		next := copyRouterInterfaceOpts(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyRouterInterfaceOpts(next)
	}
	body, err := routerInterfaceBody(cfg, false)
	if err != nil {
		return nil, wrap(err)
	}
	if err := source.WithPolicy(ctx, nil, cfg.Headers); err != nil {
		return nil, wrap(err)
	}
	target := source.Client.ServiceURL("routers", url.PathEscape(id), "remove_router_interface")
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodPut, target, body, routerJSONHeaderPolicy(source), rest.RejectionPolicy{Codes: routerJSONCodes(400, 600), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, routerJSONCodes(100, 400)...)
	if err != nil {
		return routerJSONObservation(response), wrap(err)
	}
	value := routerJSONObservation(response)
	normalized, err := responsejson.Normalize(response.Body, response.Header)
	if err == nil {
		decodedResponse := *response
		decodedResponse.Body = normalized
		decoded, decodeErr := routerJSONFromResponse(&decodedResponse)
		err = decodeErr
		if err == nil {
			// Only parsed views use normalized JSON. Keep the original HTTP
			// envelope and use response.Fail below for actual wire proof.
			value.Value, value.Wire = decoded.Value, decoded.Wire
		}
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		if value != nil {
			value.Value = nil
		}
		return value, wrap(response.Fail(err))
	}
	return value, nil
}
