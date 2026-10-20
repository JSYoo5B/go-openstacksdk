package compute

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type supplementalAddressPort struct {
	network.Port
	MACPresent bool
}

type supplementalAddressIP struct {
	PortID, FixedIP, FloatingIP   string
	FixedPresent, FloatingPresent bool
}

func (ip *supplementalAddressIP) UnmarshalJSON(data []byte) error {
	return ip.decode(data, false)
}

func (ip *supplementalAddressIP) decode(data []byte, legacy bool) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*ip = supplementalAddressIP{}
	for _, field := range []struct {
		name, alias string
		target      *string
		present     *bool
	}{
		{"port_id", "", &ip.PortID, nil}, {"fixed_ip_address", "fixed_ip", &ip.FixedIP, &ip.FixedPresent}, {"floating_ip_address", "ip", &ip.FloatingIP, &ip.FloatingPresent},
	} {
		value, present := fields[field.name]
		if !present && legacy && field.alias != "" {
			value, present = fields[field.alias]
		}
		if present && string(value) != "null" {
			if err := json.Unmarshal(value, field.target); err != nil {
				return err
			}
			if field.present != nil {
				*field.present = true
			}
		}
	}
	return nil
}

func (p *supplementalAddressPort) UnmarshalJSON(data []byte) error {
	var base network.Port
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	mac, present := fields["mac_address"]
	*p = supplementalAddressPort{Port: base, MACPresent: present && string(mac) != "null"}
	return nil
}

// ExpandServerInterfaces supplements an ACTIVE server's address snapshot from
// Neutron, then computes public/private/default-interface access addresses.
// It performs no allocation, association or server mutation. Failures retain
// the snapshot already obtained. Existing non-IPv6 floating rows skip Neutron.
func (s *Service) ExpandServerInterfaces(ctx context.Context, server *Server, options ...ServerAddressOption) (*ServerAddressView, error) {
	if s == nil {
		return nil, invalid("compute service is required")
	}
	state, err := s.prepareAddressView(ctx, server, options, true)
	if err != nil {
		return nil, err
	}
	if err := state.servers.supplementServerAddresses(ctx, server.Status, state); err != nil {
		return state.view, wrapServerAddress("supplement", err)
	}
	view := state.view
	view.PublicIPv4, err = state.publicIPv4(ctx, state.servers.dependencies.NetworkPolicy.UseExternalNetwork())
	if err != nil {
		return view, wrapServerAddress("public IPv4", err)
	}
	if !state.options.forceIPv4 {
		view.PublicIPv6, err = state.publicIPv6(ctx)
		if err != nil {
			return view, wrapServerAddress("public IPv6", err)
		}
	}
	view.PrivateIPv4, err = state.privateIPv4(ctx, state.servers.dependencies.NetworkPolicy.UseInternalNetwork())
	if err != nil {
		return view, wrapServerAddress("private IPv4", err)
	}
	view.InterfaceIP, _, err = state.defaultIP(ctx)
	if err != nil {
		return view, wrapServerAddress("default interface", err)
	}
	if view.InterfaceIP == "" {
		switch {
		case state.options.private && view.PrivateIPv4 != "":
			view.InterfaceIP = view.PrivateIPv4
		case state.options.localIPv6 && !state.options.forceIPv4 && view.PublicIPv6 != "":
			view.InterfaceIP = view.PublicIPv6
		default:
			view.InterfaceIP = view.PublicIPv4
		}
	}
	view.AccessIPv4, view.AccessIPv6 = view.PublicIPv4, view.PublicIPv6
	if state.options.private && view.PrivateIPv4 != "" {
		view.AccessIPv4 = view.PrivateIPv4
	}
	return view, state.sourceGuard(ctx)
}

func (s *Servers) supplementServerAddresses(ctx context.Context, status string, state *serverAddressState) error {
	fixedNetworks := make(map[string]string)
	for _, key := range state.view.NetworkOrder {
		for _, row := range state.view.Addresses[key] {
			if row.Version == 6 {
				continue
			}
			if row.Type == "floating" {
				return state.sourceGuard(ctx)
			}
			fixedNetworks[row.Address] = key
		}
	}
	if state.options.source == FloatingIPNone || status != "ACTIVE" || s.dependencies.AddressNetworks == nil {
		return state.sourceGuard(ctx)
	}
	if err := resource.ID(state.view.ServerID).Validate(); err != nil {
		return err
	}
	service, err := s.dependencies.AddressNetworks(ctx)
	if err != nil {
		return state.supplementFailure(ctx, err)
	}
	if service == nil {
		return state.sourceGuard(ctx)
	}
	client := service.RawClient()
	if client == nil || client.ProviderClient == nil {
		return invalid("authenticated address network service is required")
	}
	provider, endpoint, base := client.ProviderClient, client.Endpoint, client.ResourceBase
	api, ports, ips, roles := service.API, service.Ports, service.FloatingIPs, service.Roles
	computeClient := s.client
	var computeProvider *gophercloud.ProviderClient
	var computeEndpoint, computeBase string
	if state.options.source == FloatingIPNova {
		if computeClient == nil && s.dependencies.AddressCompute != nil {
			computeClient, err = s.dependencies.AddressCompute(ctx)
			if err != nil {
				return state.supplementFailure(ctx, err)
			}
		}
		if computeClient == nil || computeClient.ProviderClient == nil {
			return invalid("authenticated compute client is required for Nova address supplementation")
		}
		computeProvider, computeEndpoint, computeBase = computeClient.ProviderClient, computeClient.Endpoint, computeClient.ResourceBase
	}
	var sourceError error
	var mu sync.Mutex
	guard := func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if err := state.sourceGuard(ctx); err != nil {
			return err
		}
		if sourceError == nil && (service.RawClient() != client || client.ProviderClient != provider || client.Endpoint != endpoint || client.ResourceBase != base ||
			service.API != api || service.Ports != ports || service.FloatingIPs != ips || service.Roles != roles ||
			(state.options.source == FloatingIPNova && ((s.client != nil && s.client != computeClient) || computeClient.ProviderClient != computeProvider || computeClient.Endpoint != computeEndpoint || computeClient.ResourceBase != computeBase))) {
			sourceError = invalid("address network service source changed during supplementation")
		}
		return sourceError
	}
	portSpec := rest.CollectionSpec[supplementalAddressPort]{
		Client: client, Path: "ports", Kind: "port", PluralKey: "ports", SingleKey: "port", Validate: guard, SourceGuard: guard,
		ListCodes: []int{http.StatusOK, http.StatusNoContent},
		ID:        func(p *supplementalAddressPort) string { return p.ID }, Metadata: func(*supplementalAddressPort) *resource.Metadata { return &resource.Metadata{} },
		Paging: rest.PagePolicy[supplementalAddressPort]{HTTPLink: true},
		ValidateItem: func(p *supplementalAddressPort) error {
			if p.DeviceID == state.view.ServerID {
				return resource.ID(p.ID).Validate()
			}
			return nil
		},
	}
	for port, err := range rest.List(ctx, portSpec, url.Values{"device_id": {state.view.ServerID}}) {
		if err != nil {
			return state.supplementFailure(ctx, err)
		}
		if port.DeviceID != state.view.ServerID {
			continue
		}
		validateIP := func(ip *supplementalAddressIP) error {
			_, matches := fixedNetworks[ip.FixedIP]
			if ip.PortID == port.ID && ip.FixedPresent && matches && !ip.FloatingPresent {
				return invalid("supplemental floating address requires a string value")
			}
			return nil
		}
		ipSpec := rest.CollectionSpec[supplementalAddressIP]{
			Client: client, Path: "floatingips", Kind: "floating IP", PluralKey: "floatingips", SingleKey: "floatingip", Validate: guard, SourceGuard: guard,
			ListCodes: []int{http.StatusOK, http.StatusNoContent},
			Metadata:  func(*supplementalAddressIP) *resource.Metadata { return &resource.Metadata{} },
			Paging:    rest.PagePolicy[supplementalAddressIP]{HTTPLink: true}, ValidateItem: validateIP,
		}
		rows := rest.List(ctx, ipSpec, url.Values{"port_id": {port.ID}})
		if state.options.source == FloatingIPNova {
			rows = s.supplementalNovaIPs(ctx, computeClient, port.ID, guard, validateIP)
		}
		for ip, err := range rows {
			if err != nil {
				return state.supplementFailure(ctx, err)
			}
			if ip.PortID != port.ID || !ip.FixedPresent {
				continue
			}
			key, present := fixedNetworks[ip.FixedIP]
			if !present {
				continue
			}
			fields := make(map[string]json.RawMessage)
			var mac any
			if port.MACPresent {
				mac = port.MACAddress
			}
			for field, value := range map[string]any{"version": 4, "addr": ip.FloatingIP, "OS-EXT-IPS:type": "floating", "OS-EXT-IPS-MAC:mac_addr": mac} {
				fields[field], _ = json.Marshal(value)
			}
			state.view.Addresses[key] = append(state.view.Addresses[key], ServerAddress{Version: 4, Address: ip.FloatingIP,
				Type: "floating", MACAddress: port.MACAddress, MACPresent: port.MACPresent, Supplemental: true, Fields: fields})
		}
	}
	return guard(ctx)
}

func (state *serverAddressState) supplementFailure(ctx context.Context, err error) error {
	if contextError := ctx.Err(); contextError != nil {
		return errors.Join(err, contextError, context.Cause(ctx))
	}
	// Only clean HTTP failures are optional lookup failures. Callback joins,
	// accepted response decoding/read/Close and source-policy errors are fatal.
	switch err.(type) {
	case gophercloud.ErrUnexpectedResponseCode, *gophercloud.ErrUnexpectedResponseCode:
		state.view.SupplementalError = err
		return nil
	case *url.Error:
		var joined interface{ Unwrap() []error }
		var networkError net.Error
		if !errors.As(err, &joined) && errors.As(err.(*url.Error).Err, &networkError) {
			state.view.SupplementalError = err
			return nil
		}
		return err
	default:
		return err
	}
}
