package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"sort"

	"gophercloudsdk/internal/cloudlocation"
	"gophercloudsdk/internal/jsonfilter"
	"gophercloudsdk/internal/rest"
	floatingipapi "gophercloudsdk/network/v2/extensions/layer3/floatingips"
	"gophercloudsdk/resource"
)

type availabilityIP struct {
	Value FloatingIP
	resource.Metadata
}

func (ip *availabilityIP) UnmarshalJSON(data []byte) error {
	return resource.DecodeObject(data, &ip.Value, &ip.Metadata)
}

func availabilityField(fields map[string]json.RawMessage, names ...string) json.RawMessage {
	for _, name := range names {
		if raw, ok := fields[name]; ok {
			return raw
		}
	}
	return json.RawMessage("null")
}

// Available returns the first free current-project IP in the selected external
// network or allocates one. Reuse is always free-first, unlike Ensure. Server
// lookup/ports are lazy and only used for allocation. No PUT, wait or cleanup is
// sent. Known allocations and HTTP evidence survive response-processing errors.
func (f *FloatingIPs) Available(ctx context.Context, input AvailableFloatingIPRequest, options ...AvailableFloatingIPOption) (*FloatingIPAvailability, error) {
	input.Networks = append([]resource.Ref(nil), input.Networks...)
	policy, err := PrepareAvailableFloatingIPOptions(ctx, options...)
	if err != nil {
		return nil, err
	}
	for _, ref := range input.Networks {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
	}
	if input.Server != (resource.Ref{}) {
		if err := input.Server.Validate(); err != nil {
			return nil, err
		}
	}
	if policy.options.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, policy.options.timeout)
		defer cancel()
	}
	p, err := f.NewPlanner(ctx)
	if err != nil {
		return nil, err
	}
	owner, err := cloudlocation.ProjectID(f.api.RawClient().ProviderClient)
	if policy.options.projectID != nil {
		owner, err = json.Marshal(*policy.options.projectID)
	}
	if err != nil {
		return nil, err
	}
	if owner == nil {
		owner = json.RawMessage("null")
	}
	networkID, err := f.availableExternalNetwork(ctx, input.Networks, p)
	if err != nil {
		return nil, errors.Join(err, p.Check(ctx))
	}
	var selected *availabilityIP
	spec := rest.CollectionSpec[availabilityIP]{Client: f.api.RawClient(), Path: "floatingips", Kind: "floating IP",
		PluralKey: "floatingips", Validate: p.Check, SourceGuard: p.Check, ListCodes: []int{200},
		Metadata: func(ip *availabilityIP) *resource.Metadata { return &ip.Metadata }, Paging: rest.PagePolicy[availabilityIP]{HTTPLink: true}}
	for ip, err := range rest.List(ctx, spec, url.Values{}) {
		if err != nil {
			return nil, err
		}
		if ip.Value.FloatingNetworkID != networkID || !bytes.Equal(bytes.TrimSpace(availabilityField(ip.Body, "port_id")), []byte("null")) {
			continue
		}
		equal, err := jsonfilter.EqualPythonJSON(availabilityField(ip.Body, "project_id", "tenant_id"), owner)
		if err != nil {
			return nil, err
		}
		if equal && selected == nil {
			selected = ip
		}
	}
	if err := p.Check(ctx); err != nil {
		return nil, err
	}
	if selected != nil {
		return &FloatingIPAvailability{FloatingIP: &selected.Value, Metadata: selected.Metadata, Reused: true}, nil
	}
	expected := floatingipapi.CreateOpts{FloatingNetworkID: networkID}
	if input.Server != (resource.Ref{}) {
		serverID := input.Server.String()
		if input.Server.IsName() {
			if f.dependencies.Server == nil {
				return nil, floatingIPInvalid("named availability server needs a Compute resolver")
			}
			serverID, err = f.dependencies.Server(ctx, input.Server)
			if err = errors.Join(err, p.Check(ctx)); err != nil {
				return nil, err
			}
		}
		if err := resource.ID(serverID).Validate(); err != nil {
			return nil, err
		}
		selection, err := f.availableDestination(ctx, serverID, policy.options, p)
		if err != nil {
			return nil, err
		}
		expected.PortID, expected.FixedIP = selection.PortID, selection.FixedIPv4
	}
	body, err := expected.ToFloatingIPCreateMap()
	if err != nil {
		return nil, err
	}
	response, requestErr := rest.DoJSONGuarded(ctx, f.api.RawClient(), p.Check, http.MethodPost, f.api.RawClient().ServiceURL("floatingips"), body, nil, 201, 202)
	if response == nil {
		return nil, requestErr
	}
	result := &FloatingIPAvailability{Allocated: true, AllocationResponse: &FloatingIPAvailabilityResponse{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}, Envelope: bytes.Clone(response.Body)}}
	ip, decodeErr := rest.Decode[availabilityIP](response, "floatingip", func(ip *availabilityIP) *resource.Metadata { return &ip.Metadata })
	if decodeErr == nil {
		result.FloatingIP, result.Metadata = &ip.Value, ip.Metadata
		if err := verifyFloatingIPAssignment(&ip.Value, ip.Value.ID, expected); err != nil {
			decodeErr = response.Fail(err)
		}
		if err := resource.ID(ip.Value.ID).Validate(); err != nil {
			decodeErr = errors.Join(decodeErr, response.Fail(err))
		}
	}
	return result, errors.Join(requestErr, decodeErr, p.Check(ctx))
}

func (f *FloatingIPs) availableExternalNetwork(ctx context.Context, refs []resource.Ref, p *FloatingIPPlanner) (string, error) {
	if len(refs) == 0 {
		return f.planExternalNetwork(ctx, resource.Ref{}, p.Check, p.NetworkRoles)
	}
	roles, err := p.NetworkRoles(ctx)
	if err != nil {
		return "", err
	}
	for _, ref := range refs {
		for _, n := range roles.ExternalIPv4Floating {
			if ref.IsName() && n.Name == ref.String() || !ref.IsName() && n.ID == ref.String() {
				return n.ID, errors.Join(resource.ID(n.ID).Validate(), p.Check(ctx))
			}
		}
	}
	return "", &resource.NotFoundError{Resource: "external floating network"}
}

type availabilityPort struct {
	Value Port
	resource.Metadata
}

func (p *availabilityPort) UnmarshalJSON(data []byte) error {
	return resource.DecodeObject(data, &p.Value, &p.Metadata)
}

func (f *FloatingIPs) availableDestination(ctx context.Context, serverID string, o availableFloatingIPOptions, p *FloatingIPPlanner) (FloatingIPSelection, error) {
	var zero FloatingIPSelection
	spec := rest.CollectionSpec[availabilityPort]{Client: f.api.RawClient(), Path: "ports", Kind: "port", PluralKey: "ports",
		Validate: p.Check, SourceGuard: p.Check, ListCodes: []int{200}, Metadata: func(port *availabilityPort) *resource.Metadata { return &port.Metadata }, Paging: rest.PagePolicy[availabilityPort]{HTTPLink: true}}
	var ports []*availabilityPort
	for port, err := range rest.List(ctx, spec, url.Values{"device_id": {serverID}}) {
		if err != nil {
			return zero, err
		}
		if port.Value.DeviceID == serverID {
			ports = append(ports, port)
		}
	}
	if len(ports) == 0 {
		return zero, p.Check(ctx)
	}
	if o.fixed == "" && len(ports) > 1 {
		var networkID string
		if o.nat != (resource.Ref{}) {
			collection := rest.Collection(rest.CollectionSpec[Network]{Client: f.api.RawClient(), Path: "networks", Kind: "network", PluralKey: "networks", Get: true, SingleKey: "network", Validate: p.Check, SourceGuard: p.Check, Metadata: planMetadata[Network], ID: func(n *Network) string { return n.ID }, Name: func(n *Network) string { return n.Name }, NameQuery: func(s string) string { return s }, ListCodes: []int{200}, Paging: rest.PagePolicy[Network]{HTTPLink: true}})
			var err error
			networkID, err = collection.ResolveID(ctx, o.nat)
			if err != nil {
				return zero, err
			}
		} else {
			roles, err := p.NetworkRoles(ctx)
			if err != nil {
				return zero, err
			}
			if roles.NATDestination == nil {
				return zero, &resource.AmbiguousError{Resource: "floating IP NAT destination", Name: serverID}
			}
			networkID = roles.NATDestination.ID
		}
		var matching []*availabilityPort
		for _, port := range ports {
			if port.Value.NetworkID == networkID {
				matching = append(matching, port)
			}
		}
		if len(matching) == 0 {
			return zero, floatingIPInvalid("no server port matches the availability NAT destination")
		}
		ports = matching
	}
	if o.fixed == "" {
		sort.SliceStable(ports, func(i, j int) bool {
			var a, b string
			_ = json.Unmarshal(availabilityField(ports[i].Body, "created_at"), &a)
			_ = json.Unmarshal(availabilityField(ports[j].Body, "created_at"), &b)
			return a > b
		})
	}
	for _, port := range ports {
		for _, ip := range port.Value.FixedIPs {
			if o.fixed != "" && ip.IPAddress != o.fixed {
				continue
			}
			address, err := netip.ParseAddr(ip.IPAddress)
			if err != nil || !address.Is4() {
				continue
			}
			if err := resource.ID(port.Value.ID).Validate(); err != nil {
				return zero, err
			}
			return FloatingIPSelection{ServerID: serverID, PortID: port.Value.ID, PortNetworkID: port.Value.NetworkID, FixedIPv4: ip.IPAddress}, p.Check(ctx)
		}
	}
	if o.fixed != "" {
		return zero, p.Check(ctx)
	}
	return zero, floatingIPInvalid("no fixed IPv4 address on availability server ports")
}
