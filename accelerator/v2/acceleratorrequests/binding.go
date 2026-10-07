package acceleratorrequests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/common"
	"github.com/JSYoo5B/gophercloudsdk/internal/cyborg"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type BindingOperation struct {
	Op    string  `json:"op"`
	Path  string  `json:"path"`
	Value *string `json:"value,omitempty"`
}
type BindOpts struct {
	Hostname     string
	DeviceRPUUID string
	InstanceUUID string
	ProjectID    *string
}
type BindingOption = request.Option[struct{}]

func WithBindingHeader(key, value string) BindingOption {
	return request.WithHeader[struct{}](key, value)
}

func (a *API) Bind(ctx context.Context, ref resource.Ref, opts BindOpts, options ...BindingOption) (*common.Metadata, error) {
	patch := []BindingOperation{{Op: "add", Path: "/hostname", Value: &opts.Hostname}, {Op: "add", Path: "/device_rp_uuid", Value: &opts.DeviceRPUUID}, {Op: "add", Path: "/instance_uuid", Value: &opts.InstanceUUID}}
	if opts.ProjectID != nil {
		patch = append(patch, BindingOperation{Op: "add", Path: "/project_id", Value: opts.ProjectID})
	}
	return a.Patch(ctx, ref, patch, options...)
}

func (a *API) Unbind(ctx context.Context, ref resource.Ref, options ...BindingOption) (*common.Metadata, error) {
	return a.Patch(ctx, ref, []BindingOperation{{Op: "remove", Path: "/hostname"}, {Op: "remove", Path: "/device_rp_uuid"}, {Op: "remove", Path: "/instance_uuid"}}, options...)
}

// Patch binds or unbinds one request; it uses the collection endpoint with a
// UUID-keyed document, rather than appending the UUID to that endpoint.
func (a *API) Patch(ctx context.Context, ref resource.Ref, patch []BindingOperation, options ...BindingOption) (*common.Metadata, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if ref.IsName() {
		return nil, fmt.Errorf("%w: accelerator requests have no unique name", resource.ErrUnsupported)
	}
	return a.PatchMany(ctx, map[string][]BindingOperation{ref.String(): patch}, options...)
}

// PatchMany accepts the service's batch binding document. Operations in a
// batch share one add/remove mode. The caller supplies any required service
// token through WithBindingHeader; authorization errors remain server errors.
func (a *API) PatchMany(ctx context.Context, patches map[string][]BindingOperation, options ...BindingOption) (*common.Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cyborg.RequireMicroversion(a.client, 0); err != nil {
		return nil, err
	}
	if len(patches) == 0 {
		return nil, fmt.Errorf("%w: binding patch must not be empty", resource.ErrInvalidOption)
	}
	mode := ""
	for id, patch := range patches {
		if err := validateID(id); err != nil {
			return nil, err
		}
		if len(patch) == 0 {
			return nil, fmt.Errorf("%w: request %s has an empty patch", resource.ErrInvalidOption, id)
		}
		seen := map[string]bool{}
		for _, op := range patch {
			if op.Op != "add" && op.Op != "remove" {
				return nil, fmt.Errorf("%w: only add/remove binding operations are supported", resource.ErrInvalidOption)
			}
			if mode == "" {
				mode = op.Op
			}
			if mode != op.Op {
				return nil, fmt.Errorf("%w: binding patch cannot mix add and remove", resource.ErrInvalidOption)
			}
			if seen[op.Path] {
				return nil, fmt.Errorf("%w: duplicate binding field %s", resource.ErrInvalidOption, op.Path)
			}
			seen[op.Path] = true
			switch op.Path {
			case "/hostname", "/device_rp_uuid", "/instance_uuid":
			case "/project_id":
				if err := cyborg.RequireMicroversion(a.client, 1); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("%w: unknown binding field %s", resource.ErrInvalidOption, op.Path)
			}
			if op.Op == "add" {
				if op.Value == nil || strings.TrimSpace(*op.Value) == "" {
					return nil, fmt.Errorf("%w: binding value is required for %s", resource.ErrInvalidOption, op.Path)
				}
			} else if op.Value != nil {
				return nil, fmt.Errorf("%w: remove operation must omit value", resource.ErrInvalidOption)
			}
		}
		if mode == "add" && (!seen["/hostname"] || !seen["/device_rp_uuid"] || !seen["/instance_uuid"]) {
			return nil, fmt.Errorf("%w: binding requires hostname, device_rp_uuid and instance_uuid", resource.ErrInvalidOption)
		}
	}
	c, err := request.Apply(struct{}{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(c, false, false, true)
	}
	if err != nil {
		return nil, err
	}
	headers, err := cyborg.Headers(c.Headers)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(patches)
	if err != nil {
		return nil, err
	}
	meta, err := cyborg.Mutate(ctx, a.client, "PATCH", a.client.ServiceURL("accelerator_requests"), json.RawMessage(body), headers, 202)
	return meta, request.Wrap("patch", "arqs", err)
}
