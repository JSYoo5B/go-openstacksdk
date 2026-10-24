package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AttachFloatingIPByIDRequest names a floating IP only by its ID, as the pinned
// cloud layer's _neutron_attach_ip_to_server does with whatever row it holds,
// including a normalized Nova row. Server must be an ID. FixedAddress selects
// the port carrying that exact fixed IP; otherwise NATDestination (an ordinary
// name-or-ID lookup) or the shared NAT destination role narrows multiple ports
// and the newest port's first IPv4 address is used.
type AttachFloatingIPByIDRequest struct {
	Server                       resource.Ref
	FloatingIPID                 string
	FixedAddress, NATDestination string
}

// FloatingIPPortUpdate retains the selected port and the accepted PUT response.
// Sent reports an accepted (200..399) PUT response; a rejected PUT leaves it
// false and its HTTP error is returned.
type FloatingIPPortUpdate struct {
	Selection FloatingIPSelection
	Sent      bool
	Wire      *resource.RawResource
	Response  *FloatingIPAvailabilityResponse
}

// AttachByID selects the server's NAT destination port and sends one
// PUT floatingips/{id} with that port_id and fixed_ip_address. It does not
// read the floating IP first, verify ownership, wait, or retry; a 404 from the
// port list or the PUT is returned unchanged so the caller can decide whether
// to ignore it, as the pinned helper does. A server without ports, or without a
// port matching FixedAddress, fails before any mutation.
func (f *FloatingIPs) AttachByID(ctx context.Context, input AttachFloatingIPByIDRequest) (*FloatingIPPortUpdate, error) {
	if err := input.Server.Validate(); err != nil {
		return nil, err
	}
	if input.Server.IsName() {
		return nil, floatingIPInvalid("attach by ID needs a server ID")
	}
	if err := resource.ID(input.FloatingIPID).Validate(); err != nil {
		return nil, err
	}
	p, err := f.NewPlanner(ctx)
	if err != nil {
		return nil, err
	}
	serverID := input.Server.String()
	selection, err := f.allocateDestination(ctx, serverID, AllocateFloatingIPRequest{FixedAddress: input.FixedAddress, NATDestination: input.NATDestination}, p)
	if err != nil {
		return nil, err
	}
	if selection.PortID == "" {
		return &FloatingIPPortUpdate{}, errors.Join(floatingIPInvalid("unable to find a port for server %s", serverID), p.Check(ctx))
	}
	result := &FloatingIPPortUpdate{Selection: selection}
	fields := map[string]any{"port_id": selection.PortID, "fixed_ip_address": selection.FixedIPv4}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	client := f.api.RawClient()
	response, requestErr := rest.DoJSONGuarded(ctx, client, p.Check, http.MethodPut, client.ServiceURL("floatingips", url.PathEscape(input.FloatingIPID)), map[string]any{"floatingip": fields}, nil, codes...)
	if response == nil {
		return result, errors.Join(requestErr, p.Check(ctx))
	}
	result.Sent = true
	result.Response = &FloatingIPAvailabilityResponse{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}, Envelope: bytes.Clone(response.Body)}
	key := "floatingip"
	var envelope map[string]json.RawMessage
	if json.Unmarshal(response.Body, &envelope) == nil {
		if _, present := envelope[key]; !present {
			key = ""
		}
	}
	var decodeErr error
	if len(bytes.TrimSpace(response.Body)) != 0 {
		result.Wire, decodeErr = rest.Decode[resource.RawResource](response, key, func(row *resource.RawResource) *resource.Metadata { return &row.Metadata })
	}
	return result, errors.Join(requestErr, decodeErr, p.Check(ctx))
}
