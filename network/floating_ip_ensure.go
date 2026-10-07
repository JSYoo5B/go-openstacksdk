package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"gophercloudsdk/internal/project"
	floatingipapi "gophercloudsdk/network/v2/extensions/layer3/floatingips"
	"gophercloudsdk/resource"
)

// Ensure associates a current-project floating IPv4 address with a server.
// Reuse defaults to true: already attached at this destination, then the first
// unattached candidate in response order, otherwise a new allocation. Every
// list page is consumed before mutation and constraints are checked locally.
// Known revisions guard PUT. Later failures preserve the selected/allocated IP.
func (f *FloatingIPs) Ensure(ctx context.Context, input EnsureFloatingIPRequest, options ...EnsureFloatingIPOption) (*FloatingIPAssignment, error) {
	policy, err := PrepareEnsureFloatingIPOptions(ctx, options...)
	if err != nil {
		return nil, err
	}
	if err := input.Server.Validate(); err != nil {
		return nil, fmt.Errorf("server: %w", err)
	}
	if input.Network != (resource.Ref{}) {
		if err := input.Network.Validate(); err != nil {
			return nil, fmt.Errorf("external network: %w", err)
		}
	}
	o := policy.options
	owner := o.projectID
	if o.reuse && owner == "" {
		owner, err = project.Current(ctx, f.api.RawClient())
		if err != nil {
			return nil, floatingIPWrap("resolve reuse project", err)
		}
	}
	serverID := input.Server.String()
	if input.Server.IsName() {
		if f.dependencies.Server == nil {
			return nil, floatingIPWrap("resolve server", fmt.Errorf("%w: server resolver is unavailable", resource.ErrUnsupported))
		}
		serverID, err = f.dependencies.Server(ctx, input.Server)
		if err != nil {
			return nil, floatingIPWrap("resolve server", err)
		}
	}
	if err := resource.ID(serverID).Validate(); err != nil {
		return nil, floatingIPWrap("resolve server", err)
	}
	networkID, err := f.ensureExternalNetwork(ctx, input.Network)
	if err != nil {
		return nil, floatingIPWrap("resolve external network", err)
	}
	destination, err := f.selectDestination(ctx, serverID, o.destination)
	if err != nil {
		return nil, floatingIPWrap("select destination", err)
	}
	expected := floatingipapi.CreateOpts{FloatingNetworkID: networkID, PortID: destination.portID, FixedIP: destination.address, ProjectID: owner}
	if o.reuse {
		candidate, attached, err := f.availableFloatingIP(ctx, owner, expected)
		if err != nil {
			return nil, floatingIPWrap("select available IP", err)
		}
		if candidate != nil {
			result := &FloatingIPAssignment{FloatingIP: &candidate.FloatingIP, Reused: true}
			if !attached {
				if err := ctx.Err(); err != nil {
					return result, err
				}
				value, err := f.api.Update(ctx, candidate.ID, floatingipapi.UpdateOpts{
					PortID: &destination.portID, FixedIP: destination.address, RevisionNumber: candidate.revision,
				})
				if err != nil {
					return result, floatingIPWrap("associate reused IP", err)
				}
				if err := verifyFloatingIPAssignment(value, candidate.ID, expected); err != nil {
					return result, floatingIPWrap("verify association", err)
				}
				result.FloatingIP = value
			}
			return f.finishFloatingIPAssignment(ctx, result, candidate.ID, expected, o.destination)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value, err := f.api.Create(ctx, expected)
	if err != nil {
		return nil, floatingIPWrap("allocate IP", err)
	}
	result := &FloatingIPAssignment{FloatingIP: value, Allocated: true}
	if value == nil {
		return result, floatingIPWrap("verify allocation", fmt.Errorf("Neutron returned no floating IP"))
	}
	return f.finishFloatingIPAssignment(ctx, result, value.ID, expected, o.destination)
}

func (f *FloatingIPs) ensureExternalNetwork(ctx context.Context, ref resource.Ref) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if ref != (resource.Ref{}) {
		id, err := f.externalNetworks.ResolveID(ctx, ref)
		if err != nil {
			return "", err
		}
		return id, resource.ID(id).Validate()
	}
	var first string
	for value, err := range f.externalNetworks.List(ctx) {
		if err != nil {
			return "", err
		}
		if first == "" {
			first = value.ID
			if err := resource.ID(first).Validate(); err != nil {
				return "", err
			}
		}
	}
	if first != "" {
		return first, ctx.Err()
	}
	var gateway string
	pager := floatingIPSelectionPager(routers.List(f.api.RawClient(), nil), func(r pagination.PageResult) pagination.Page {
		return routers.RouterPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
	})
	items := resource.Stream(ctx, pager, func(page pagination.Page) ([]routers.Router, error) {
		return routers.ExtractRouters(page.(floatingIPSelectionPage).Page)
	})
	for router, err := range items {
		if err != nil {
			return "", err
		}
		if gateway == "" && router.AdminStateUp && router.GatewayInfo.NetworkID != "" {
			gateway = router.GatewayInfo.NetworkID
			if err := resource.ID(gateway).Validate(); err != nil {
				return "", err
			}
		}
	}
	if gateway == "" {
		return "", &resource.NotFoundError{Resource: "external network"}
	}
	return gateway, ctx.Err()
}

// Preserve revision presence, including 0, instead of guessing from a zero int.
type availableFloatingIP struct {
	FloatingIP
	revision *int
}

func (value *availableFloatingIP) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &value.FloatingIP); err != nil {
		return err
	}
	var fields struct {
		Revision *int `json:"revision_number"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	value.revision = fields.Revision
	return nil
}

func (f *FloatingIPs) availableFloatingIP(ctx context.Context, owner string, expected floatingipapi.CreateOpts) (*availableFloatingIP, bool, error) {
	var free, attached *availableFloatingIP
	filter := floatingips.ListOpts{ProjectID: owner, FloatingNetworkID: expected.FloatingNetworkID}
	pager := floatingIPSelectionPager(floatingips.List(f.api.RawClient(), filter), func(r pagination.PageResult) pagination.Page {
		return floatingips.FloatingIPPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
	})
	items := resource.Stream(ctx, pager, func(page pagination.Page) ([]availableFloatingIP, error) {
		// ExtractIntoSlicePtr decodes anonymous fields separately, bypassing the
		// outer decoder and losing revision presence. Decode the full envelope.
		var response struct {
			Values []availableFloatingIP `json:"floatingips"`
		}
		err := page.(floatingIPSelectionPage).Page.(floatingips.FloatingIPPage).ExtractInto(&response)
		return response.Values, err
	})
	for value, err := range items {
		if err != nil {
			return nil, false, err
		}
		projectID := value.ProjectID
		if projectID == "" {
			projectID = value.TenantID
		}
		if projectID != owner || value.ProjectID != "" && value.TenantID != "" && value.ProjectID != value.TenantID || value.FloatingNetworkID != expected.FloatingNetworkID {
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
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if attached != nil {
		return attached, true, nil
	}
	return free, false, nil
}

func (f *FloatingIPs) finishFloatingIPAssignment(ctx context.Context, result *FloatingIPAssignment, id string, expected floatingipapi.CreateOpts, options createFloatingIPOptions) (*FloatingIPAssignment, error) {
	if err := resource.ID(id).Validate(); err != nil {
		return result, floatingIPWrap("verify assignment", err)
	}
	if err := verifyFloatingIPAssignment(result.FloatingIP, id, expected); err != nil {
		return result, floatingIPWrap("verify association", err)
	}
	if !options.wait {
		return result, ctx.Err()
	}
	ready, err := f.Wait(ctx, resource.ID(id), "ACTIVE", options.waitOptions...)
	if err != nil {
		return result, floatingIPWrap("assignment/wait", err)
	}
	if err := verifyFloatingIPAssignment(ready, id, expected); err != nil {
		return result, floatingIPWrap("verify association", err)
	}
	if ready.Status != floatingipapi.StatusActive {
		return result, floatingIPWrap("verify ACTIVE status", fmt.Errorf("floating IP %q status is %q", id, ready.Status))
	}
	result.FloatingIP = ready
	return result, nil
}

func verifyFloatingIPAssignment(value *FloatingIP, id string, expected floatingipapi.CreateOpts) error {
	if value == nil || value.ID != id {
		return fmt.Errorf("Neutron response does not match floating IP %q", id)
	}
	if value.FloatingNetworkID != expected.FloatingNetworkID {
		return fmt.Errorf("floating IP %q returned a different external network", id)
	}
	owner := value.ProjectID
	if owner == "" {
		owner = value.TenantID
	}
	if value.ProjectID != "" && value.TenantID != "" && value.ProjectID != value.TenantID || expected.ProjectID != "" && owner != expected.ProjectID {
		return fmt.Errorf("floating IP %q returned a different or inconsistent project", id)
	}
	address, err := netip.ParseAddr(value.FloatingIP)
	if err != nil || !address.Is4() {
		return fmt.Errorf("floating IP %q returned no valid IPv4 address", id)
	}
	return verifyFloatingIPDestination(value, expected)
}
