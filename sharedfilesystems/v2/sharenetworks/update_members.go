package sharenetworks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateMembersOpts is the Python-compatible update body. Unlike UpdateOpts a
// zero Optional is omitted, request.Present sends an empty string, and
// request.Null sends an explicit JSON null for every member.
type UpdateMembersOpts struct {
	Name            request.Optional[string] `json:"name,omitzero"`
	Description     request.Optional[string] `json:"description,omitzero"`
	NeutronNetID    request.Optional[string] `json:"neutron_net_id,omitzero"`
	NeutronSubnetID request.Optional[string] `json:"neutron_subnet_id,omitzero"`
	NovaNetID       request.Optional[string] `json:"nova_net_id,omitzero"`
}

// UpdateMembersOption is one option of UpdateMembers: request.Field (an
// extension member; declared members are protected) or request.Custom over
// *request.Config[UpdateMembersOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type UpdateMembersOption = request.WriteCall

// UpdateMembers PUTs /share-networks/{id} with exactly the members that are
// set. It returns the complete response object, preserving unknown members,
// and accepts HTTP 200 only. The request is never replayed after an uncertain
// failure; an update that sets no member fails before HTTP.
func (a *API) UpdateMembers(ctx context.Context, id string, opts UpdateMembersOpts, options ...UpdateMembersOption) (*resource.RawResource, error) {
	const op, kind = "UpdateMembers", "sharenetworks"
	if ctx == nil {
		return nil, request.Wrap(op, kind, fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	config, err := calls.ApplyConfig(opts, manilaown.FieldCalls[UpdateMembersOpts](), options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	var encoded []byte
	if err == nil {
		encoded, err = json.Marshal(config.Options)
	}
	var members map[string]json.RawMessage
	if err == nil {
		err = json.Unmarshal(encoded, &members)
	}
	if err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	inner := make(map[string]any, len(members))
	for key, value := range members {
		inner[key] = value
	}
	body, err := request.MergeFieldsFor(map[string]any{"share_network": inner}, config.Fields, config.Options)
	if err == nil && len(body["share_network"].(map[string]any)) == 0 {
		err = fmt.Errorf("%w: update requires at least one member", resource.ErrInvalidOption)
	}
	if err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	client := a.RawClient()
	response, err := rest.DoJSON(ctx, client, "PUT", client.ServiceURL("share-networks", url.PathEscape(id)), body, nil, 200)
	if err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	value, err := rest.Decode(response, "share_network", func(r *resource.RawResource) *resource.Metadata { return &r.Metadata })
	return value, request.Wrap(op, kind, err)
}
