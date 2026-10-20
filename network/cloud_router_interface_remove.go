package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// RouterInterfaceRemoveOpts preserves None/false/zero and SDK Resource identity.
// There are no cloud defaults beyond absent subnet_id/port_id. Headers are a Go
// transport convenience; the selected client's version is retained.
type RouterInterfaceRemoveOpts struct {
	SubnetID, PortID *routers.RouterInterfaceRef
	Headers          map[string]string
}
type RouterInterfaceRemoveOption func(*RouterInterfaceRemoveOpts) error

func copyRouterInterfaceRemoveRef(v *routers.RouterInterfaceRef) *routers.RouterInterfaceRef {
	if v == nil {
		return nil
	}
	return &routers.RouterInterfaceRef{Value: bytes.Clone(v.Value), Resource: v.Resource}
}
func copyRouterInterfaceRemove(v RouterInterfaceRemoveOpts) RouterInterfaceRemoveOpts {
	v.SubnetID = copyRouterInterfaceRemoveRef(v.SubnetID)
	v.PortID = copyRouterInterfaceRemoveRef(v.PortID)
	v.Headers = maps.Clone(v.Headers)
	return v
}
func WithRouterInterfaceRemoveOpts(v RouterInterfaceRemoveOpts) RouterInterfaceRemoveOption {
	owned := copyRouterInterfaceRemove(v)
	return func(c *RouterInterfaceRemoveOpts) error { *c = copyRouterInterfaceRemove(owned); return nil }
}
func routerInterfaceRemoveRef(v any) (*routers.RouterInterfaceRef, error) {
	var cached *resource.RawResource
	switch value := v.(type) {
	case *resource.RawResource:
		cached = value
	case resource.RawResource:
		cached = &value
	case *ports.PortRecord:
		if value != nil {
			if value.Resource == nil {
				return nil, qosReadInvalid("port evidence has no Resource")
			}
			cached = value.Resource
		}
	case *subnets.SubnetRecord:
		if value != nil {
			if value.Resource == nil {
				return nil, qosReadInvalid("subnet evidence has no Resource")
			}
			cached = value.Resource
		}
	default:
		raw, err := json.Marshal(v)
		return &routers.RouterInterfaceRef{Value: bytes.Clone(raw)}, err
	}
	if cached == nil {
		return &routers.RouterInterfaceRef{Value: json.RawMessage("null")}, nil
	}
	raw := bytes.Clone(cached.Body["id"])
	if raw == nil {
		raw = json.RawMessage("null")
	}
	return &routers.RouterInterfaceRef{Value: raw, Resource: true}, nil
}
func routerInterfaceRemoveOption(v any, port bool) RouterInterfaceRemoveOption {
	owned, err := routerInterfaceRemoveRef(v)
	return func(c *RouterInterfaceRemoveOpts) error {
		if err != nil {
			return err
		}
		if port {
			c.PortID = copyRouterInterfaceRemoveRef(owned)
		} else {
			c.SubnetID = copyRouterInterfaceRemoveRef(owned)
		}
		return nil
	}
}
func WithRouterInterfaceRemoveSubnetID(v any) RouterInterfaceRemoveOption {
	return routerInterfaceRemoveOption(v, false)
}
func WithRouterInterfaceRemovePortID(v any) RouterInterfaceRemoveOption {
	return routerInterfaceRemoveOption(v, true)
}
func WithRouterInterfaceRemoveHeaders(v map[string]string) RouterInterfaceRemoveOption {
	owned := maps.Clone(v)
	return func(c *RouterInterfaceRemoveOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func routerInterfaceRemoveTruthy(v *routers.RouterInterfaceRef) (bool, error) {
	if v == nil {
		return false, nil
	}
	if v.Resource {
		return true, nil
	}
	return cloudfilter.PythonJSONTruthy(v.Value)
}

// RouterInterfaceRemoveResult separates Source's None return from the decoded
// response.json() it discards. Value is JSON null only on success; all other
// fields are independent actual HTTP observations, never a Router Resource.
// A successful acknowledgement does not establish backend convergence.
type RouterInterfaceRemoveResult struct {
	Value, ResponseJSON json.RawMessage
	Wire                *resource.RawResource
	Envelope            json.RawMessage
	Header              http.Header
	StatusCode          int
}

func routerInterfaceRemoveReceipt(v *routers.RouterJSONResult, err error) *RouterInterfaceRemoveResult {
	if v != nil {
		return &RouterInterfaceRemoveResult{ResponseJSON: bytes.Clone(v.Value), Wire: v.Wire.Clone(), Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
	}
	var response *resource.ResponseError
	if errors.As(err, &response) && response != nil {
		return &RouterInterfaceRemoveResult{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) {
		return &RouterInterfaceRemoveResult{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	return nil
}

// RemoveRouterInterface checks both references before validating a service or
// router ID, then issues a single literal-router PUT. It never resolves a router
// name, fetches it, or invents an interface Resource from the acknowledgement.
func (s *Service) RemoveRouterInterface(ctx context.Context, routerID string, options ...RouterInterfaceRemoveOption) (*RouterInterfaceRemoveResult, error) {
	wrap := func(err error) error {
		return request.Wrap("RemoveRouterInterface", "router interfaces", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	// Capture before callbacks, but defer configuration errors until after the
	// cloud's both-falsey check. Capturing is local and never performs discovery.
	var source *cloudread.Source
	var captureErr error
	var selectedAPI *networkv2.Service
	var selectedRouters *routers.API
	var selectedClient *gophercloud.ServiceClient
	if s == nil || s.API == nil || s.client == nil || s.API.Routers == nil || s.API.RawClient() != s.client || s.API.Routers.RawClient() != s.client {
		captureErr = qosReadInvalid("configured router network service required")
	} else {
		selectedAPI, selectedRouters, selectedClient = s.API, s.API.Routers, s.client
		source, captureErr = cloudread.Capture(ctx, s.client, "network")
	}
	var changed error
	guard := func(ctx context.Context) error {
		if err := cloudread.Context(ctx); err != nil {
			return err
		}
		if captureErr != nil {
			return captureErr
		}
		if changed == nil && (s.API != selectedAPI || s.client != selectedClient || s.API.Routers != selectedRouters || selectedAPI.RawClient() != selectedClient || selectedRouters.RawClient() != selectedClient) {
			changed = qosReadInvalid("selected router interface service/API changed")
		}
		return errors.Join(changed, source.Guard(ctx))
	}
	cfg := RouterInterfaceRemoveOpts{}
	for _, apply := range slices.Clone(options) {
		if err := cloudread.Context(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(qosReadInvalid("nil router interface removal option"))
		}
		if captureErr == nil {
			if err := guard(ctx); err != nil {
				return nil, wrap(err)
			}
		}
		next := copyRouterInterfaceRemove(cfg)
		callbackErr := apply(&next)
		var sourceErr error
		if captureErr == nil {
			sourceErr = guard(ctx)
		}
		if err := errors.Join(callbackErr, sourceErr); err != nil {
			return nil, wrap(err)
		}
		cfg = copyRouterInterfaceRemove(next)
	}
	subnet, err := routerInterfaceRemoveTruthy(cfg.SubnetID)
	if err != nil {
		return nil, wrap(err)
	}
	if !subnet {
		port, err := routerInterfaceRemoveTruthy(cfg.PortID)
		if err != nil {
			return nil, wrap(err)
		}
		if !port {
			return nil, wrap(qosReadInvalid("at least one of subnet_id or port_id must be supplied"))
		}
	}
	if err := errors.Join(guard(ctx), rest.CheckOperationGuard(ctx)); err != nil {
		return nil, wrap(err)
	}
	api := networkv2.New(&source.Client)
	leafCtx := rest.WithOperationGuard(ctx, guard)
	value, err := api.Routers.RemoveCloudInterfaceRecord(leafCtx, routerID, routers.WithRouterInterfaceOpts(routers.RouterInterfaceOpts{Subnet: copyRouterInterfaceRemoveRef(cfg.SubnetID), Port: copyRouterInterfaceRemoveRef(cfg.PortID), Headers: maps.Clone(cfg.Headers)}))
	err = errors.Join(err, guard(ctx), rest.CheckOperationGuard(ctx))
	result := routerInterfaceRemoveReceipt(value, err)
	if err != nil {
		if result != nil {
			result.ResponseJSON = nil
		}
		return result, wrap(err)
	}
	if result == nil {
		return nil, wrap(qosReadInvalid("missing actual router removal response"))
	}
	result.Value = json.RawMessage("null")
	return result, nil
}
