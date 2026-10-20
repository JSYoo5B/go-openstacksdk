package network

import (
	"context"
	"net/netip"
	"net/url"
	"sort"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (f *FloatingIPs) attachIPSpec(guard func(context.Context) error) rest.CollectionSpec[availableFloatingIP] {
	return rest.CollectionSpec[availableFloatingIP]{Client: f.api.RawClient(), Path: "floatingips", Kind: "floating IP",
		SingleKey: "floatingip", PluralKey: "floatingips", Get: true, SourceGuard: guard, Validate: guard,
		ID: func(ip *availableFloatingIP) string { return ip.ID }, Metadata: planMetadata[availableFloatingIP],
		ListCodes: []int{200, 204}, Paging: rest.PagePolicy[availableFloatingIP]{HTTPLink: true}}
}

func (f *FloatingIPs) planExistingIP(ctx context.Context, ref resource.Ref, guard func(context.Context) error) (*availableFloatingIP, error) {
	spec := f.attachIPSpec(guard)
	if !ref.IsName() {
		spec.ValidateItem = func(ip *availableFloatingIP) error {
			if ip.ID != ref.String() {
				return floatingIPInvalid("Neutron returned a different requested floating IP")
			}
			return verifyAttachIPIdentity(ip)
		}
		return rest.Collection(spec).Get(ctx, ref.String())
	}
	spec.ValidateItem = func(ip *availableFloatingIP) error {
		if ip.FloatingIP.FloatingIP == ref.String() {
			return verifyAttachIPIdentity(ip)
		}
		return nil
	}
	var selected *availableFloatingIP
	var ids []string
	for ip, err := range rest.List(ctx, spec, url.Values{"floating_ip_address": {ref.String()}}) {
		if err != nil {
			return nil, err
		}
		if ip.FloatingIP.FloatingIP != ref.String() {
			continue
		}
		ids = append(ids, ip.ID)
		if selected == nil {
			copy := cloneAttachIP(*ip)
			selected = &copy
		}
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, &resource.NotFoundError{Resource: "floating IP", Reference: ref.String()}
	}
	if len(ids) > 1 {
		sort.Strings(ids)
		return nil, &resource.AmbiguousError{Resource: "floating IP", Name: ref.String(), IDs: ids}
	}
	return selected, nil
}

func verifyAttachIPIdentity(ip *availableFloatingIP) error {
	if ip == nil {
		return floatingIPInvalid("Neutron returned no existing floating IP")
	}
	for _, id := range []string{ip.ID, ip.FloatingNetworkID} {
		if err := resource.ID(id).Validate(); err != nil {
			return err
		}
	}
	address, err := netip.ParseAddr(ip.FloatingIP.FloatingIP)
	if err != nil || !address.Is4() {
		return floatingIPInvalid("existing floating IP has no valid IPv4 address")
	}
	if ip.ProjectID != "" && ip.TenantID != "" && ip.ProjectID != ip.TenantID {
		return floatingIPInvalid("existing floating IP has inconsistent project aliases")
	}
	if ip.revision != nil && *ip.revision < 0 {
		return floatingIPInvalid("floating IP revision must be non-negative")
	}
	return nil
}
