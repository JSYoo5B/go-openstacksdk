package devices

import (
	"context"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/common"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cyborg"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ActionOption is one option of Enable and Disable: request.Header or
// request.Custom over *request.Config[struct{}]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ActionOption = request.WriteCall

// Enable sets the device to enabled. The response contains no Device object;
// use Wait with the same UUID when a refreshed status is needed.
func (a *API) Enable(ctx context.Context, device resource.Ref, options ...ActionOption) (*common.Metadata, error) {
	return a.action(ctx, device, "enable", options...)
}

// Disable sets the device to maintaining. Unlike status fetch/wait (2.3+),
// the controller does not microversion-gate these actions.
func (a *API) Disable(ctx context.Context, device resource.Ref, options ...ActionOption) (*common.Metadata, error) {
	return a.action(ctx, device, "disable", options...)
}

func (a *API) action(ctx context.Context, device resource.Ref, action string, options ...ActionOption) (*common.Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cyborg.RequireMicroversion(a.client, 0); err != nil {
		return nil, request.Wrap(action, "devices", err)
	}
	c, err := calls.ApplyConfig(struct{}{}, cyborg.HeaderCalls, options...)
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
	id, err := a.ResolveID(ctx, device)
	if err != nil {
		return nil, err
	}
	meta, err := cyborg.Mutate(ctx, a.client, "POST", a.client.ServiceURL("devices", url.PathEscape(id), action), nil, headers, 200)
	return meta, request.Wrap(action, "devices", err)
}
