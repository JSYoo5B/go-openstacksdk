package network

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	floatingipapi "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AttachPrepared rechecks the selected port and existing IP, and preserves the
// original revision for a conditional PUT. Its result keeps the last verified
// IP on a later failure; Allocated is always false. No failure triggers cleanup.
func (f *FloatingIPs) AttachPrepared(ctx context.Context, plan FloatingIPAttachPlan) (*FloatingIPAssignment, error) {
	if ctx == nil {
		return nil, floatingIPInvalid("context is required")
	}
	if f == nil || plan.service != f || plan.guard == nil || !plan.policy.prepared {
		return nil, floatingIPInvalid("an attachment plan prepared by this floating IP service is required")
	}
	initial := cloneAttachIP(plan.ip)
	result := &FloatingIPAssignment{FloatingIP: &initial.FloatingIP, Reused: true}
	if err := plan.guard(ctx); err != nil {
		return result, err
	}
	ports := f.planPortSpec(plan.guard)
	ports.ValidateItem = func(port *Port) error { return verifyFloatingIPPlanPort(port, plan.selection.FloatingIPSelection) }
	if _, err := rest.Collection(ports).Get(ctx, plan.selection.PortID); err != nil {
		return result, floatingIPWrap("recheck attachment destination", err)
	}
	spec := f.attachIPSpec(plan.guard)
	spec.ValidateItem = func(ip *availableFloatingIP) error {
		if err := verifyAttachIPIdentity(ip); err != nil {
			return err
		}
		if ip.ID != plan.selection.IPID || ip.FloatingIP.FloatingIP != plan.selection.Address || ip.FloatingNetworkID != plan.selection.NetworkID {
			return floatingIPInvalid("selected floating IP identity, address or external network changed")
		}
		if owner := attachIPOwner(&plan.ip.FloatingIP); owner != "" && attachIPOwner(&ip.FloatingIP) != owner {
			return floatingIPInvalid("selected floating IP project changed")
		}
		original := ip.PortID == plan.ip.PortID && ip.FixedIP == plan.ip.FixedIP
		selected := ip.PortID == plan.selection.PortID && ip.FixedIP == plan.selection.FixedIPv4
		if !original && !selected {
			return floatingIPInvalid("selected floating IP association changed")
		}
		return nil
	}
	current, err := rest.Collection(spec).Get(ctx, plan.selection.IPID)
	if err != nil {
		return result, floatingIPWrap("recheck existing IP", err)
	}
	copy := cloneAttachIP(*current)
	result.FloatingIP = &copy.FloatingIP
	verify := func(ip *FloatingIP) error { return verifyAttachedIP(ip, plan) }
	if current.PortID != plan.selection.PortID || current.FixedIP != plan.selection.FixedIPv4 {
		body := map[string]any{"floatingip": map[string]any{"port_id": plan.selection.PortID, "fixed_ip_address": plan.selection.FixedIPv4}}
		headers := map[string]string{}
		if plan.ip.revision != nil {
			headers["If-Match"] = fmt.Sprintf("revision_number=%d", *plan.ip.revision)
		}
		value, _, err := f.planWrite(ctx, plan.guard, http.MethodPut, plan.selection.IPID, body, headers, verify, 200)
		if value != nil && verify(value) == nil {
			result.FloatingIP = value
		}
		if err != nil {
			return result, floatingIPWrap("attach existing IP", err)
		}
	}
	if err := verify(result.FloatingIP); err != nil {
		return result, floatingIPWrap("verify existing IP attachment", err)
	}
	if !plan.policy.options.destination.wait {
		return result, plan.guard(ctx)
	}
	waitSpec := rest.CollectionSpec[FloatingIP]{Client: f.api.RawClient(), Path: "floatingips", Kind: "floating IP",
		SingleKey: "floatingip", Get: true, SourceGuard: plan.guard, Validate: plan.guard, Metadata: planMetadata[FloatingIP],
		ID: func(ip *FloatingIP) string { return ip.ID }, Status: func(ip *FloatingIP) string { return ip.Status },
		Failed: func(status string) bool { return strings.EqualFold(status, floatingipapi.StatusError) }, ValidateItem: verify}
	ready, err := rest.Collection(waitSpec).Wait(ctx, resource.ID(plan.selection.IPID), floatingipapi.StatusActive, plan.policy.options.destination.waitOptions...)
	if err != nil {
		return result, floatingIPWrap("existing IP attachment/wait", errors.Join(err, plan.guard(ctx)))
	}
	if ready.Status != floatingipapi.StatusActive {
		return result, floatingIPInvalid("floating IP %q status is %q", plan.selection.IPID, ready.Status)
	}
	result.FloatingIP = ready
	return result, plan.guard(ctx)
}

func attachIPOwner(ip *FloatingIP) string {
	if ip.ProjectID != "" {
		return ip.ProjectID
	}
	return ip.TenantID
}

func verifyAttachedIP(ip *FloatingIP, plan FloatingIPAttachPlan) error {
	expected := floatingipapi.CreateOpts{FloatingNetworkID: plan.selection.NetworkID, PortID: plan.selection.PortID,
		FixedIP: plan.selection.FixedIPv4, ProjectID: attachIPOwner(&plan.ip.FloatingIP)}
	if err := verifyFloatingIPAssignment(ip, plan.selection.IPID, expected); err != nil {
		return err
	}
	if ip.FloatingIP != plan.selection.Address {
		return floatingIPInvalid("selected floating IP address changed")
	}
	return nil
}
