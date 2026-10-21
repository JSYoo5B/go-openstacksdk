package shares

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ManageOpts describes an existing backend share to adopt with POST
// /shares/manage. Protocol, ExportPath and ServiceHost are required. Optional
// pointer, bool pointer and map fields are omitted only when nil, so an empty
// Name, a false IsPublic and an empty DriverOptions map are all sent.
type ManageOpts struct {
	Protocol    string
	ExportPath  string
	ServiceHost string
	// Name and Description are optional share attributes.
	Name        *string
	Description *string
	// ShareType is the share type name or ID.
	ShareType *string
	// DriverOptions are back-end specific key/value options.
	DriverOptions map[string]string
	// IsPublic sets the share visibility.
	IsPublic *bool
	// ShareServerID selects a share server (Manila 2.49 and later).
	ShareServerID *string
}

// ManageOption configures Manage.
type ManageOption = request.Option[ManageOpts]

// WithManageOptions replaces the typed Manage input.
func WithManageOptions(value ManageOpts) ManageOption { return request.WithOptions(value) }

// WithManageField adds an extension member to the share object. Core member
// names are rejected.
func WithManageField(key string, value any) ManageOption {
	return request.WithField[ManageOpts](key, value)
}

var manageCoreKeys = []string{"protocol", "export_path", "service_host", "name", "description", "share_type", "driver_options", "is_public", "share_server_id"}

func (o ManageOpts) body(fields map[string]json.RawMessage) (map[string]any, error) {
	for name, value := range map[string]string{"protocol": o.Protocol, "export_path": o.ExportPath, "service_host": o.ServiceHost} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%w: manage %s is required", resource.ErrInvalidOption, name)
		}
	}
	share := map[string]any{"protocol": o.Protocol, "export_path": o.ExportPath, "service_host": o.ServiceHost}
	if o.Name != nil {
		share["name"] = *o.Name
	}
	if o.Description != nil {
		share["description"] = *o.Description
	}
	if o.ShareType != nil {
		share["share_type"] = *o.ShareType
	}
	if o.DriverOptions != nil {
		share["driver_options"] = o.DriverOptions
	}
	if o.IsPublic != nil {
		share["is_public"] = *o.IsPublic
	}
	if o.ShareServerID != nil {
		share["share_server_id"] = *o.ShareServerID
	}
	for key, value := range fields {
		for _, core := range manageCoreKeys {
			if key == core {
				return nil, fmt.Errorf("%w: extension %q is a core input field", resource.ErrInvalidOption, key)
			}
		}
		share[key] = append(json.RawMessage(nil), value...)
	}
	return map[string]any{"share": share}, nil
}

// Manage adopts an existing backend share through POST /shares/manage and
// returns the share echoed by the service. HTTP 200 and 202 succeed; the share
// usually remains in a managing status, so wait with WaitFor before use. The
// call is never replayed and local cancellation does not undo a manage that the
// service already accepted.
func (a *API) Manage(ctx context.Context, opts ManageOpts, options ...ManageOption) (*Share, error) {
	cfg, err := request.Apply(opts, options...)
	if err != nil {
		return nil, request.Wrap("Manage", "shares", err)
	}
	if err = request.ValidateCapabilities(cfg, true, false, false); err != nil {
		return nil, request.Wrap("Manage", "shares", err)
	}
	body, err := cfg.Options.body(cfg.Fields)
	if err != nil {
		return nil, request.Wrap("Manage", "shares", err)
	}
	var out struct {
		Share *Share `json:"share"`
	}
	_, err = a.client.Post(ctx, a.client.ServiceURL("shares", "manage"), body, &out, &gophercloud.RequestOpts{OkCodes: []int{200, 202}})
	if err != nil {
		return nil, request.Wrap("Manage", "shares", err)
	}
	return out.Share, nil
}

func (a *API) shareActionNull(ctx context.Context, operation, id, action string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return request.Wrap(operation, "shares", err)
	}
	_, err := a.client.Post(ctx, a.client.ServiceURL("shares", id, "action"), map[string]any{action: nil}, nil, &gophercloud.RequestOpts{OkCodes: []int{202}})
	return request.Wrap(operation, "shares", err)
}

// SoftDelete moves a share to the recycle bin with the soft_delete action
// (Manila 2.69 and later). Only HTTP 202 succeeds and acknowledgement does not
// prove the share left its previous status.
func (a *API) SoftDelete(ctx context.Context, id string) error {
	return a.shareActionNull(ctx, "SoftDelete", id, "soft_delete")
}

// Restore returns a soft-deleted share with the restore action (Manila 2.69
// and later). Only HTTP 202 succeeds.
func (a *API) Restore(ctx context.Context, id string) error {
	return a.shareActionNull(ctx, "Restore", id, "restore")
}
