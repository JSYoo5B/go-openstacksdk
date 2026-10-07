package network

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"gophercloudsdk/internal/rest"
	floatingipapi "gophercloudsdk/network/v2/extensions/layer3/floatingips"
	"gophercloudsdk/resource"
)

func (f *FloatingIPs) executeFloatingIPPlan(ctx context.Context, selection FloatingIPSelection, policy EnsureFloatingIPPolicy, guard func(context.Context) error) (*FloatingIPAssignment, error) {
	o := policy.options
	expected := floatingipapi.CreateOpts{FloatingNetworkID: selection.NetworkID, PortID: selection.PortID, FixedIP: selection.FixedIPv4, ProjectID: o.projectID}
	var result *FloatingIPAssignment
	if o.reuse {
		candidate, attached, err := f.planAvailableIP(ctx, expected, guard)
		if err != nil {
			return nil, floatingIPWrap("select available IP", err)
		}
		if candidate != nil {
			result = &FloatingIPAssignment{FloatingIP: &candidate.FloatingIP, Reused: true}
			if !attached {
				body := map[string]any{"floatingip": map[string]any{"port_id": selection.PortID, "fixed_ip_address": selection.FixedIPv4}}
				headers := map[string]string{}
				if candidate.revision != nil {
					headers["If-Match"] = fmt.Sprintf("revision_number=%d", *candidate.revision)
				}
				value, _, err := f.planWrite(ctx, guard, http.MethodPut, candidate.ID, body, headers,
					func(ip *FloatingIP) error { return verifyFloatingIPAssignment(ip, candidate.ID, expected) }, 200)
				// A matching accepted body is current evidence even when its read,
				// Close or source check failed. Never adopt an invalid response.
				if value != nil && verifyFloatingIPAssignment(value, candidate.ID, expected) == nil {
					result.FloatingIP = value
				}
				if err != nil {
					return result, floatingIPWrap("associate reused IP", err)
				}
				result.FloatingIP = value
			}
		}
	}
	if result == nil {
		body, err := expected.ToFloatingIPCreateMap()
		if err != nil {
			return nil, err
		}
		value, accepted, err := f.planWrite(ctx, guard, http.MethodPost, "", body, nil,
			func(ip *FloatingIP) error {
				if err := resource.ID(ip.ID).Validate(); err != nil {
					return err
				}
				return verifyFloatingIPAssignment(ip, ip.ID, expected)
			}, 201, 202)
		if accepted {
			result = &FloatingIPAssignment{FloatingIP: value, Allocated: true}
		}
		if err != nil {
			return result, floatingIPWrap("allocate IP", err)
		}
	}
	if result.FloatingIP == nil {
		return result, floatingIPInvalid("Neutron returned no floating IP")
	}
	id := result.FloatingIP.ID
	if err := resource.ID(id).Validate(); err != nil {
		return result, floatingIPWrap("verify assignment", err)
	}
	if err := verifyFloatingIPAssignment(result.FloatingIP, id, expected); err != nil {
		return result, floatingIPWrap("verify assignment", err)
	}
	if !o.destination.wait {
		return result, guard(ctx)
	}
	spec := rest.CollectionSpec[FloatingIP]{Client: f.api.RawClient(), Path: "floatingips", Kind: "floating IP",
		SingleKey: "floatingip", Get: true, SourceGuard: guard, Validate: guard, Metadata: planMetadata[FloatingIP],
		ID: func(ip *FloatingIP) string { return ip.ID }, Status: func(ip *FloatingIP) string { return ip.Status },
		Failed:       func(status string) bool { return strings.EqualFold(status, floatingipapi.StatusError) },
		ValidateItem: func(ip *FloatingIP) error { return verifyFloatingIPAssignment(ip, id, expected) }}
	ready, err := rest.Collection(spec).Wait(ctx, resource.ID(id), floatingipapi.StatusActive, o.destination.waitOptions...)
	if err != nil {
		return result, floatingIPWrap("assignment/wait", errors.Join(err, guard(ctx)))
	}
	if ready.Status != floatingipapi.StatusActive {
		return result, floatingIPInvalid("floating IP %q status is %q", id, ready.Status)
	}
	result.FloatingIP = ready
	return result, guard(ctx)
}

func (f *FloatingIPs) planWrite(ctx context.Context, guard func(context.Context) error, method, id string, body any, headers map[string]string, verify func(*FloatingIP) error, codes ...int) (*FloatingIP, bool, error) {
	client := f.api.RawClient()
	endpoint := client.ServiceURL("floatingips")
	if id != "" {
		endpoint = client.ServiceURL("floatingips", url.PathEscape(id))
	}
	response, requestErr := rest.DoJSONGuardedHeaders(ctx, client, guard, method, endpoint, body, headers, codes...)
	accepted := false
	if response != nil {
		for _, code := range codes {
			accepted = accepted || response.StatusCode == code
		}
	}
	if response == nil || !accepted {
		return nil, accepted, requestErr
	}
	value, err := rest.Decode(response, "floatingip", planMetadata[FloatingIP])
	if err == nil && verify != nil {
		if validation := verify(value); validation != nil {
			err = response.Fail(validation)
		}
	}
	return value, accepted, errors.Join(requestErr, err)
}

func (f *FloatingIPs) planAvailableIP(ctx context.Context, expected floatingipapi.CreateOpts, guard func(context.Context) error) (*availableFloatingIP, bool, error) {
	var attached, free *availableFloatingIP
	spec := rest.CollectionSpec[availableFloatingIP]{Client: f.api.RawClient(), Path: "floatingips", Kind: "floating IP",
		PluralKey: "floatingips", SourceGuard: guard, Validate: guard, Metadata: planMetadata[availableFloatingIP], ListCodes: []int{200, 204}, Paging: rest.PagePolicy[availableFloatingIP]{HTTPLink: true}}
	for value, err := range rest.List(ctx, spec, url.Values{"project_id": {expected.ProjectID}, "floating_network_id": {expected.FloatingNetworkID}}) {
		if err != nil {
			return nil, false, err
		}
		owner := value.ProjectID
		if owner == "" {
			owner = value.TenantID
		}
		if owner != expected.ProjectID || value.ProjectID != "" && value.TenantID != "" && value.ProjectID != value.TenantID || value.FloatingNetworkID != expected.FloatingNetworkID {
			continue
		}
		address, err := netip.ParseAddr(value.FloatingIP.FloatingIP)
		if err != nil || !address.Is4() || value.Status == floatingipapi.StatusError {
			continue
		}
		if value.PortID != "" && (value.PortID != expected.PortID || value.FixedIP != expected.FixedIP) {
			continue
		}
		if err := resource.ID(value.ID).Validate(); err != nil {
			return nil, false, err
		}
		if value.revision != nil && *value.revision < 0 {
			return nil, false, floatingIPInvalid("floating IP revision must be non-negative")
		}
		if value.PortID != "" {
			if attached == nil {
				attached = value
			}
		} else if free == nil {
			free = value
		}
	}
	if err := guard(ctx); err != nil {
		return nil, false, err
	}
	if attached != nil {
		return attached, true, nil
	}
	return free, false, nil
}
