package compute

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type novaIPBackend struct {
	state    *automaticIPState
	client   *gophercloud.ServiceClient
	settings network.FloatingIPPolicySettings
}

func (state *automaticIPState) novaBackend(ctx context.Context) (*novaIPBackend, error) {
	return state.novaBackendFor(ctx, true)
}

func (state *automaticIPState) novaBackendFor(ctx context.Context, attachment bool) (*novaIPBackend, error) {
	settings, err := state.policy.Settings()
	if err != nil {
		return nil, err
	}
	if settings.Port != (resource.Ref{}) || settings.NATDestination != (resource.Ref{}) || settings.ProjectID != "" {
		return nil, fmt.Errorf("%w: Nova floating IPs do not accept Neutron port, NAT-network or project overrides", resource.ErrUnsupported)
	}
	client, err := state.rawClient(ctx)
	if err != nil {
		var missing *gophercloud.ErrEndpointNotFound
		var missingValue gophercloud.ErrEndpointNotFound
		if errors.As(err, &missing) || errors.As(err, &missingValue) {
			err = errors.Join(err, fmt.Errorf("%w: legacy Compute endpoint is unavailable", resource.ErrUnsupported))
		}
		return nil, err
	}
	if version := client.Microversion; version != "" {
		parts := strings.Split(version, ".")
		if len(parts) != 2 || parts[0] != "2" {
			return nil, invalid("Nova floating IPs require a numeric Compute 2.x microversion")
		}
		minor, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil || parts[1] == "" {
			return nil, invalid("invalid Nova floating IP microversion")
		}
		// Both allocation endpoints disappear at 2.36. This workflow needs
		// them even though the action remains available until 2.44.
		if minor >= 36 {
			return nil, fmt.Errorf("%w: legacy Nova floating IP endpoints require Compute microversion below 2.36", resource.ErrUnsupported)
		}
	}
	targets := []string{client.ServiceURL("os-floating-ips")}
	if attachment {
		targets = append(targets, client.ServiceURL("servers", url.PathEscape(state.serverID), "action"))
	}
	for _, target := range targets {
		if err := rest.ValidateTarget(client, target); err != nil {
			return nil, err
		}
	}
	return &novaIPBackend{state: state, client: client, settings: settings}, state.check(ctx)
}

func (b *novaIPBackend) unavailable(err error) error {
	if cleanAddressHTTPStatus(err, 404) {
		return errors.Join(err, fmt.Errorf("%w: legacy Nova floating IP API is unavailable", resource.ErrUnsupported))
	}
	return err
}

func verifyNovaIP(ip *NovaFloatingIP) error {
	if ip == nil {
		return novaIPInvalid("no floating IP was returned")
	}
	if err := resource.ID(ip.ID).Validate(); err != nil {
		return err
	}
	address, err := netip.ParseAddr(ip.Address)
	if err != nil || !address.Is4() || address.String() != ip.Address {
		return novaIPInvalid("response address must be canonical IPv4")
	}
	if ip.Pool == "" {
		return novaIPInvalid("response lacks a pool name")
	}
	if _, present := ip.Body["instance_id"]; !present {
		return novaIPInvalid("response lacks association metadata")
	}
	if ip.FixedAddress != nil && *ip.FixedAddress != "" {
		if address, err := netip.ParseAddr(*ip.FixedAddress); err != nil || !address.Is4() {
			return novaIPInvalid("response fixed address must be IPv4")
		}
	}
	return nil
}

func (b *novaIPBackend) list(ctx context.Context) ([]*NovaFloatingIP, error) {
	var page *rest.Response
	spec := rest.CollectionSpec[NovaFloatingIP]{Client: b.client, Path: "os-floating-ips", Kind: "Nova floating IP",
		PluralKey: "floating_ips", Validate: b.state.check, SourceGuard: b.state.check,
		ListCodes: []int{200}, Metadata: func(ip *NovaFloatingIP) *resource.Metadata { return &ip.Metadata },
		ValidateResponse: func(response *rest.Response) error { page = response; return nil },
		ValidateItem:     func(ip *NovaFloatingIP) error { ip.response = page; return nil },
		Paging:           rest.PagePolicy[NovaFloatingIP]{HTTPLink: true}}
	var values []*NovaFloatingIP
	for ip, err := range rest.List(ctx, spec, url.Values{}) {
		if err != nil {
			return nil, b.unavailable(err)
		}
		values = append(values, ip)
	}
	return values, b.state.check(ctx)
}

func novaEvidence(response *rest.Response) *NovaFloatingIPResponse {
	if response == nil {
		return nil
	}
	return &NovaFloatingIPResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

func (b *novaIPBackend) read(ctx context.Context, id string) (*NovaFloatingIP, error) {
	response, requestErr := rest.DoJSONGuarded(ctx, b.client, b.state.check, http.MethodGet, b.client.ServiceURL("os-floating-ips", url.PathEscape(id)), nil, nil, 200)
	if response == nil {
		return nil, b.unavailable(requestErr)
	}
	ip, err := rest.Decode[NovaFloatingIP](response, "floating_ip", func(ip *NovaFloatingIP) *resource.Metadata { return &ip.Metadata })
	if err == nil {
		ip.response = response
		err = verifyNovaIP(ip)
		if err == nil && ip.ID != id {
			err = novaIPInvalid("GET returned a different floating IP ID")
		}
		if err != nil {
			err = response.Fail(err)
			ip = nil
		}
	}
	return ip, errors.Join(requestErr, err, b.state.check(ctx))
}

func (b *novaIPBackend) pool(ctx context.Context, ref resource.Ref) (string, error) {
	if ref != (resource.Ref{}) {
		if err := ref.Validate(); err != nil {
			return "", err
		}
		// Nova pools have names only. Both Ref forms are explicit literals;
		// they are never resolved as Neutron network identities.
		return ref.String(), b.state.check(ctx)
	}
	type pool struct {
		resource.Metadata
		Name string `json:"name"`
	}
	spec := rest.CollectionSpec[pool]{Client: b.client, Path: "os-floating-ip-pools", Kind: "Nova floating IP pool", PluralKey: "floating_ip_pools",
		ListCodes: []int{200}, Validate: b.state.check, SourceGuard: b.state.check, Metadata: func(p *pool) *resource.Metadata { return &p.Metadata },
		ValidateItem: func(p *pool) error {
			if p.Name == "" {
				return novaIPInvalid("pool response lacks a name")
			}
			return nil
		}}
	for p, err := range rest.List(ctx, spec, url.Values{}) {
		if err != nil {
			return "", b.unavailable(err)
		}
		return p.Name, b.state.check(ctx)
	}
	return "", fmt.Errorf("%w: no Nova floating IP pool", resource.ErrNotFound)
}

func (b *novaIPBackend) selectAddress(ctx context.Context, address string) (*NovaFloatingIP, error) {
	values, err := b.list(ctx)
	if err != nil {
		return nil, err
	}
	var selected *NovaFloatingIP
	for _, ip := range values {
		if ip.Address != address {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("%w: multiple Nova floating IPs have address %q", resource.ErrAmbiguous, address)
		}
		selected = ip
	}
	if selected == nil {
		return nil, fmt.Errorf("%w: Nova floating IP %q", resource.ErrNotFound, address)
	}
	if err := verifyNovaIP(selected); err != nil {
		return nil, selected.fail(err)
	}
	return selected, b.state.check(ctx)
}

func (b *novaIPBackend) selectPool(ctx context.Context, pool string) (*NovaFloatingIP, error) {
	if !b.settings.Reuse {
		return nil, b.state.check(ctx)
	}
	values, err := b.list(ctx)
	if err != nil {
		return nil, err
	}
	for _, ip := range values {
		// Match source's raw instance_id=None predicate, including its
		// distinction from the empty string, before allocation.
		if ip.Pool != pool {
			continue
		}
		if _, present := ip.Body["instance_id"]; !present {
			return nil, ip.fail(novaIPInvalid("pool candidate lacks association metadata"))
		}
		if ip.InstanceID == nil {
			if err := verifyNovaIP(ip); err != nil {
				return nil, ip.fail(err)
			}
			return ip, b.state.check(ctx)
		}
	}
	return nil, b.state.check(ctx)
}

func (b *novaIPBackend) create(ctx context.Context, pool string) (*NovaFloatingIPAssignment, error) {
	result := &NovaFloatingIPAssignment{}
	response, requestErr := rest.DoJSONGuarded(ctx, b.client, b.state.check, http.MethodPost, b.client.ServiceURL("os-floating-ips"), map[string]any{"pool": pool}, nil, 200)
	if response == nil {
		return result, b.unavailable(requestErr)
	}
	result.Allocated, result.AllocationResponse = true, novaEvidence(response)
	ip, decodeErr := rest.Decode[NovaFloatingIP](response, "floating_ip", func(ip *NovaFloatingIP) *resource.Metadata { return &ip.Metadata })
	if decodeErr == nil {
		ip.response = response
		// Keep the actual decoded allocation even if its fields contradict
		// the request. It is known partial state, never a valid next target.
		result.FloatingIP = ip
		if err := verifyNovaIP(ip); err != nil {
			decodeErr = response.Fail(err)
		} else if ip.Pool != pool || ip.InstanceID != nil {
			decodeErr = response.Fail(novaIPInvalid("allocation response does not match the free requested pool"))
		}
	}
	if err := errors.Join(requestErr, decodeErr, b.state.check(ctx)); err != nil {
		return result, err
	}
	current, err := b.read(ctx, ip.ID)
	if current != nil && current.Address == ip.Address && current.Pool == pool && current.InstanceID == nil {
		result.FloatingIP = current
	} else if current != nil {
		err = errors.Join(err, current.fail(novaIPInvalid("allocated floating IP changed before attachment")))
	}
	return result, errors.Join(err, b.state.check(ctx))
}

func (b *novaIPBackend) attach(ctx context.Context, result *NovaFloatingIPAssignment) error {
	selected := result.FloatingIP
	current, err := b.read(ctx, selected.ID)
	if current != nil {
		sameInstance := current.InstanceID == nil && selected.InstanceID == nil
		if current.InstanceID != nil && selected.InstanceID != nil {
			sameInstance = *current.InstanceID == *selected.InstanceID
		}
		atServer := current.InstanceID != nil && *current.InstanceID == b.state.serverID
		if current.Address != selected.Address || current.Pool != selected.Pool || (!sameInstance && !atServer) {
			return errors.Join(err, current.fail(novaIPInvalid("selected floating IP changed before attachment")))
		}
		result.FloatingIP = current
	}
	if err != nil {
		return err
	}
	if current.InstanceID != nil && *current.InstanceID == b.state.serverID && (b.settings.FixedAddress == "" || current.FixedAddress != nil && *current.FixedAddress == b.settings.FixedAddress) {
		result.AlreadyAttached = true
		return b.state.check(ctx)
	}
	body := map[string]any{"address": current.Address}
	if b.settings.FixedAddress != "" {
		body["fixed_address"] = b.settings.FixedAddress
	}
	response, err := rest.DoJSONGuarded(ctx, b.client, b.state.check, http.MethodPost, b.client.ServiceURL("servers", url.PathEscape(b.state.serverID), "action"), map[string]any{"addFloatingIp": body}, nil, 202)
	result.ActionResponse = novaEvidence(response)
	result.ActionAccepted = response != nil
	return errors.Join(err, b.state.check(ctx))
}

func (state *automaticIPState) novaAssignment(ctx context.Context, address string, poolRef resource.Ref) (*NovaFloatingIPAssignment, error) {
	b, err := state.novaBackend(ctx)
	if err != nil {
		return nil, err
	}
	var selected *NovaFloatingIP
	var pool string
	if address != "" {
		selected, err = b.selectAddress(ctx, address)
	} else {
		pool, err = b.pool(ctx, poolRef)
		if err == nil {
			selected, err = b.selectPool(ctx, pool)
		}
	}
	if err != nil {
		return nil, err
	}
	result := &NovaFloatingIPAssignment{FloatingIP: selected, Reused: selected != nil}
	if selected == nil {
		result, err = b.create(ctx, pool)
		if err != nil {
			return result, err
		}
	}
	state.decision.NovaSelections = append(state.decision.NovaSelections, NovaFloatingIPSelection{ID: result.FloatingIP.ID, Address: result.FloatingIP.Address, Pool: result.FloatingIP.Pool})
	err = b.attach(ctx, result)
	return result, errors.Join(err, state.check(ctx))
}
