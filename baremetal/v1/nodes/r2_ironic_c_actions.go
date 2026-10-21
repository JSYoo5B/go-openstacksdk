package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicwire"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Microversions the pinned Python Node class requires for the helpers below.
const (
	VIFVersion        = "1.28"
	InjectNMIVersion  = "1.29"
	TraitsVersion     = "1.37"
	BootModeVersion   = "1.76"
	FirmwareVersion   = "1.86"
	VirtualMediaMicro = "1.89"
)

func nodeInvalid(op, format string, args ...any) error {
	return request.Wrap(op, "nodes", fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...)))
}

func (a *API) nodeURL(client *gophercloud.ServiceClient, id string, segments ...string) string {
	return client.ServiceURL(append([]string{"nodes", url.PathEscape(id)}, segments...)...)
}

// SetBootMode sends PUT /nodes/{id}/states/boot_mode with {"target": target}.
// The target must be "uefi" or "bios"; any other value fails before HTTP. The
// call needs microversion 1.76: a configured client microversion that matches is
// kept, a lower one fails with resource.ErrUnsupported, and an unset one is
// replaced by 1.76 when the endpoint range contains it. Only 202 succeeds. The
// call is asynchronous and is never replayed.
func (a *API) SetBootMode(ctx context.Context, id, target string) error {
	if target != "uefi" && target != "bios" {
		return nodeInvalid("SetBootMode", "unrecognized boot mode %q; boot mode should be one of 'uefi' or 'bios'", target)
	}
	return a.putPicked(ctx, "SetBootMode", id, BootModeVersion, []string{"states", "boot_mode"}, map[string]any{"target": target}, 202)
}

// SetSecureBoot sends PUT /nodes/{id}/states/secure_boot with the explicit
// boolean target, so false is sent. It uses the same microversion rule as
// SetBootMode and accepts only 202.
func (a *API) SetSecureBoot(ctx context.Context, id string, target bool) error {
	return a.putPicked(ctx, "SetSecureBoot", id, BootModeVersion, []string{"states", "secure_boot"}, map[string]any{"target": target}, 202)
}

// SetTraits replaces every trait of the node with PUT /nodes/{id}/traits and
// {"traits": traits}. A nil slice is sent as JSON null and an empty slice as an
// empty list. Microversion 1.37 is picked like SetBootMode; only 204 succeeds.
func (a *API) SetTraits(ctx context.Context, id string, traits []string) error {
	return a.putPicked(ctx, "SetTraits", id, TraitsVersion, []string{"traits"}, map[string]any{"traits": traits}, 204)
}

func (a *API) putPicked(ctx context.Context, op, id, required string, segments []string, body any, code int) error {
	if id == "" {
		return nodeInvalid(op, "node ID or name must not be empty")
	}
	version, err := ironicwire.Pick(ctx, a.client, required)
	if err != nil {
		return request.Wrap(op, "nodes", err)
	}
	client := ironicwire.At(a.client, version)
	_, err = client.Put(ctx, a.nodeURL(client, id, segments...), body, nil, &gophercloud.RequestOpts{OkCodes: []int{code}})
	return request.Wrap(op, "nodes", err)
}

// RemoveTraitOption configures RemoveTrait.
type RemoveTraitOption func(*bool)

// WithRemoveTraitIgnoreMissing selects whether HTTP 400 (the trait is not
// assigned) is a successful no-op. The default is true, like the pinned source.
func WithRemoveTraitIgnoreMissing(ignore bool) RemoveTraitOption {
	return func(v *bool) { *v = ignore }
}

// RemoveTrait sends DELETE /nodes/{id}/traits/{trait} (microversion 1.37 picked
// like SetBootMode, only 204). It reports true after a removal. With
// ignore-missing enabled, which is the default, an HTTP 400 answer returns false
// and a nil error; every other status, including 404, stays an error.
func (a *API) RemoveTrait(ctx context.Context, id, trait string, options ...RemoveTraitOption) (bool, error) {
	if id == "" || trait == "" {
		return false, nodeInvalid("RemoveTrait", "node ID and trait must not be empty")
	}
	ignore := true
	for _, option := range options {
		if option != nil {
			option(&ignore)
		}
	}
	version, err := ironicwire.Pick(ctx, a.client, TraitsVersion)
	if err != nil {
		return false, request.Wrap("RemoveTrait", "nodes", err)
	}
	client := ironicwire.At(a.client, version)
	_, err = client.Delete(ctx, a.nodeURL(client, id, "traits", url.PathEscape(trait)), &gophercloud.RequestOpts{OkCodes: []int{204}})
	if err != nil {
		if ignore && gophercloud.ResponseCodeIs(err, http.StatusBadRequest) {
			return false, nil
		}
		return false, request.Wrap("RemoveTrait", "nodes", err)
	}
	return true, nil
}

// InjectNMIChecked sends PUT /nodes/{id}/management/inject_nmi with an empty
// JSON object after asserting microversion 1.29. A configured client microversion
// below 1.29 fails with resource.ErrUnsupported before any request; an unset one
// is replaced by the highest version shared with the endpoint (at most 1.109),
// which must be at least 1.29. Only 204 succeeds and nothing is replayed.
func (a *API) InjectNMIChecked(ctx context.Context, id string) error {
	if id == "" {
		return nodeInvalid("InjectNMIChecked", "node ID or name must not be empty")
	}
	version, err := ironicwire.Assert(ctx, a.client, InjectNMIVersion, ironicwire.NodeCeiling, "")
	if err != nil {
		return request.Wrap("InjectNMIChecked", "nodes", err)
	}
	client := ironicwire.At(a.client, version)
	_, err = client.Put(ctx, a.nodeURL(client, id, "management", "inject_nmi"), map[string]any{}, nil, &gophercloud.RequestOpts{OkCodes: []int{204}})
	return request.Wrap("InjectNMIChecked", "nodes", err)
}

// ListVIFIDs returns the IDs of the VIFs attached to the node from
// GET /nodes/{id}/vifs after asserting microversion 1.28. A VIF entry without a
// string id fails the call. Only 200 succeeds.
func (a *API) ListVIFIDs(ctx context.Context, id string) ([]string, error) {
	if id == "" {
		return nil, nodeInvalid("ListVIFIDs", "node ID or name must not be empty")
	}
	version, err := ironicwire.Assert(ctx, a.client, VIFVersion, ironicwire.NodeCeiling, "")
	if err != nil {
		return nil, request.Wrap("ListVIFIDs", "nodes", err)
	}
	client := ironicwire.At(a.client, version)
	var body struct {
		VIFs []map[string]json.RawMessage `json:"vifs"`
	}
	if _, err := client.Get(ctx, a.nodeURL(client, id, "vifs"), &body, &gophercloud.RequestOpts{OkCodes: []int{200}}); err != nil {
		return nil, request.Wrap("ListVIFIDs", "nodes", err)
	}
	ids := make([]string, 0, len(body.VIFs))
	for i, vif := range body.VIFs {
		var value string
		if err := json.Unmarshal(vif["id"], &value); err != nil {
			return nil, request.Wrap("ListVIFIDs", "nodes", fmt.Errorf("vif %d has no string id: %w", i, err))
		}
		ids = append(ids, value)
	}
	return ids, nil
}

// DetachVIFOption configures DetachVIF.
type DetachVIFOption func(*bool)

// WithDetachVIFIgnoreMissing selects whether HTTP 400 (the VIF is not
// attached) is a successful no-op. The default is true.
func WithDetachVIFIgnoreMissing(ignore bool) DetachVIFOption {
	return func(v *bool) { *v = ignore }
}

// DetachVIF sends DELETE /nodes/{id}/vifs/{vif} after asserting microversion
// 1.28, and reports whether a VIF was detached. With ignore-missing enabled,
// which is the default, HTTP 400 returns false and a nil error. Only 204 is a
// detach; any other status is an error and nothing is replayed.
func (a *API) DetachVIF(ctx context.Context, id, vifID string, options ...DetachVIFOption) (bool, error) {
	if id == "" || vifID == "" {
		return false, nodeInvalid("DetachVIF", "node ID and VIF ID must not be empty")
	}
	ignore := true
	for _, option := range options {
		if option != nil {
			option(&ignore)
		}
	}
	version, err := ironicwire.Assert(ctx, a.client, VIFVersion, ironicwire.NodeCeiling, "")
	if err != nil {
		return false, request.Wrap("DetachVIF", "nodes", err)
	}
	client := ironicwire.At(a.client, version)
	_, err = client.Delete(ctx, a.nodeURL(client, id, "vifs", url.PathEscape(vifID)), &gophercloud.RequestOpts{OkCodes: []int{204}})
	if err != nil {
		if ignore && gophercloud.ResponseCodeIs(err, http.StatusBadRequest) {
			return false, nil
		}
		return false, request.Wrap("DetachVIF", "nodes", err)
	}
	return true, nil
}

// DetachVirtualMediaJSON detaches virtual media with the wire form of the
// pinned Python client: DELETE /nodes/{id}/vmedia whose JSON body is
// {"device_types": "a,b"} (a comma-joined string) when deviceTypes is not empty
// and which has no body otherwise. It asserts microversion 1.89 first and
// accepts only 204. DetachVirtualMedia sends the device types as a query
// parameter instead; this server-side equivalence is not verified here.
func (a *API) DetachVirtualMediaJSON(ctx context.Context, id string, deviceTypes []string) error {
	if id == "" {
		return nodeInvalid("DetachVirtualMediaJSON", "node ID or name must not be empty")
	}
	version, err := ironicwire.Assert(ctx, a.client, VirtualMediaMicro, ironicwire.NodeCeiling, "")
	if err != nil {
		return request.Wrap("DetachVirtualMediaJSON", "nodes", err)
	}
	client := ironicwire.At(a.client, version)
	opts := &gophercloud.RequestOpts{OkCodes: []int{204}}
	if len(deviceTypes) > 0 {
		opts.JSONBody = map[string]any{"device_types": strings.Join(deviceTypes, ",")}
	}
	_, err = client.Delete(ctx, a.nodeURL(client, id, "vmedia"), opts)
	return request.Wrap("DetachVirtualMediaJSON", "nodes", err)
}

// ListFirmwareRaw returns the JSON document of GET /nodes/{id}/firmware
// unchanged, after asserting microversion 1.86. Only 200 succeeds.
func (a *API) ListFirmwareRaw(ctx context.Context, id string) (json.RawMessage, error) {
	if id == "" {
		return nil, nodeInvalid("ListFirmwareRaw", "node ID or name must not be empty")
	}
	version, err := ironicwire.Assert(ctx, a.client, FirmwareVersion, ironicwire.NodeCeiling, "")
	if err != nil {
		return nil, request.Wrap("ListFirmwareRaw", "nodes", err)
	}
	client := ironicwire.At(a.client, version)
	var raw json.RawMessage
	if _, err := client.Get(ctx, a.nodeURL(client, id, "firmware"), &raw, &gophercloud.RequestOpts{OkCodes: []int{200}}); err != nil {
		return nil, request.Wrap("ListFirmwareRaw", "nodes", err)
	}
	return slices.Clone(raw), nil
}

// ListVendorPassthruRaw returns GET /nodes/{id}/vendor_passthru/methods as a
// map from method name to its unchanged JSON document, so unknown per-method
// fields are kept. Only 200 succeeds, and the body must be a JSON object.
func (a *API) ListVendorPassthruRaw(ctx context.Context, id string) (map[string]json.RawMessage, error) {
	if id == "" {
		return nil, nodeInvalid("ListVendorPassthruRaw", "node ID or name must not be empty")
	}
	var methods map[string]json.RawMessage
	if _, err := a.client.Get(ctx, a.nodeURL(a.client, id, "vendor_passthru", "methods"), &methods, &gophercloud.RequestOpts{OkCodes: []int{200}}); err != nil {
		return nil, request.Wrap("ListVendorPassthruRaw", "nodes", err)
	}
	if methods == nil {
		methods = map[string]json.RawMessage{}
	}
	return methods, nil
}
