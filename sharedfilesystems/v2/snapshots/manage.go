package snapshots

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ManageOpts describes an existing backend snapshot to adopt with POST
// /snapshots/manage. ShareID and ProviderLocation are required. Optional
// pointer and map fields are omitted only when nil, so an empty Name and an
// empty DriverOptions map are sent.
type ManageOpts struct {
	ShareID          string
	ProviderLocation string
	Name             *string
	Description      *string
	// DriverOptions are back-end specific key/value options.
	DriverOptions map[string]string
}

// ManageOption configures Manage.
type ManageOption = request.Option[ManageOpts]

// WithManageOptions replaces the typed Manage input.
func WithManageOptions(value ManageOpts) ManageOption { return request.WithOptions(value) }

// WithManageField adds an extension member to the snapshot object. Core member
// names are rejected.
func WithManageField(key string, value any) ManageOption {
	return request.WithField[ManageOpts](key, value)
}

var manageCoreKeys = []string{"share_id", "provider_location", "name", "description", "driver_options"}

func (o ManageOpts) body(fields map[string]json.RawMessage) (map[string]any, error) {
	if strings.TrimSpace(o.ShareID) == "" || strings.TrimSpace(o.ProviderLocation) == "" {
		return nil, fmt.Errorf("%w: manage share ID and provider location are required", resource.ErrInvalidOption)
	}
	snapshot := map[string]any{"share_id": o.ShareID, "provider_location": o.ProviderLocation}
	if o.Name != nil {
		snapshot["name"] = *o.Name
	}
	if o.Description != nil {
		snapshot["description"] = *o.Description
	}
	if o.DriverOptions != nil {
		snapshot["driver_options"] = o.DriverOptions
	}
	for key, value := range fields {
		for _, core := range manageCoreKeys {
			if key == core {
				return nil, fmt.Errorf("%w: extension %q is a core input field", resource.ErrInvalidOption, key)
			}
		}
		snapshot[key] = append(json.RawMessage(nil), value...)
	}
	return map[string]any{"snapshot": snapshot}, nil
}

// Manage adopts an existing backend snapshot through POST /snapshots/manage
// and returns the snapshot echoed by the service. HTTP 200 and 202 succeed; the
// snapshot usually remains in a managing status, so wait with WaitFor. The call
// is never replayed and local cancellation does not undo accepted work.
func (a *API) Manage(ctx context.Context, opts ManageOpts, options ...ManageOption) (*Snapshot, error) {
	cfg, err := request.Apply(opts, options...)
	if err != nil {
		return nil, request.Wrap("Manage", "snapshots", err)
	}
	if err = request.ValidateCapabilities(cfg, true, false, false); err != nil {
		return nil, request.Wrap("Manage", "snapshots", err)
	}
	body, err := cfg.Options.body(cfg.Fields)
	if err != nil {
		return nil, request.Wrap("Manage", "snapshots", err)
	}
	var out struct {
		Snapshot *Snapshot `json:"snapshot"`
	}
	_, err = a.client.Post(ctx, a.client.ServiceURL("snapshots", "manage"), body, &out, &gophercloud.RequestOpts{OkCodes: []int{200, 202}})
	if err != nil {
		return nil, request.Wrap("Manage", "snapshots", err)
	}
	return out.Snapshot, nil
}

// Unmanage removes a snapshot from Manila management without deleting backend
// data, using POST /snapshots/{id}/action with {"unmanage":null}. Only HTTP 202
// succeeds and acknowledgement does not prove the snapshot is gone.
func (a *API) Unmanage(ctx context.Context, id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return request.Wrap("Unmanage", "snapshots", err)
	}
	_, err := a.client.Post(ctx, a.client.ServiceURL("snapshots", id, "action"), map[string]any{"unmanage": nil}, nil, &gophercloud.RequestOpts{OkCodes: []int{202}})
	return request.Wrap("Unmanage", "snapshots", err)
}
