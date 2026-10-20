package compute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServerNetworkInterface selects one Nova NIC. Network and Port may both be
// supplied; Nova checks their compatibility. Zero references are omitted.
// FixedIP is sent literally. Tag zero omits the key; request.Present and
// request.Null preserve a supplied string or null. Tag requires Compute 2.42
// or later whenever it is set, including an empty string or null.
type ServerNetworkInterface struct {
	Network resource.Ref
	Port    resource.Ref
	FixedIP string
	Tag     request.Optional[string]
}

// WithNetworks replaces earlier NIC/mode selections with network-only NICs.
// Explicit IDs bypass Neutron lookup; names resolve through the Connection.
func WithNetworks(refs ...resource.Ref) CreateServerOption {
	refs = append([]resource.Ref(nil), refs...)
	return func(o *createServerOptions) error {
		if len(refs) == 0 {
			return invalid("network selection must not be empty")
		}
		nics := make([]ServerNetworkInterface, len(refs))
		for i, ref := range refs {
			if err := ref.Validate(); err != nil {
				return err
			}
			nics[i].Network = ref
		}
		o.networkInterfaces, o.networkMode = nics, ""
		return nil
	}
}

// WithNetworkInterfaces replaces earlier NIC/mode selections. The SDK owns
// the slice and optional tag values, and preserves NIC order. Empty NIC objects
// and fixed-IP/tag-only entries are allowed; Nova validates their meaning.
func WithNetworkInterfaces(nics ...ServerNetworkInterface) CreateServerOption {
	nics = cloneServerNetworkInterfaces(nics)
	return func(o *createServerOptions) error {
		if len(nics) == 0 {
			return invalid("network interface selection must not be empty")
		}
		for i, nic := range nics {
			for _, ref := range []resource.Ref{nic.Network, nic.Port} {
				if ref != (resource.Ref{}) {
					if err := ref.Validate(); err != nil {
						return fmt.Errorf("network interface %d: %w", i, err)
					}
				}
			}
		}
		o.networkInterfaces, o.networkMode = cloneServerNetworkInterfaces(nics), ""
		return nil
	}
}

// WithNetworkMode replaces earlier NIC selections with Nova's auto or none
// mode. Both modes require a selected Compute microversion of at least 2.37.
// It does not change the shared client version or perform extra discovery.
func WithNetworkMode(mode string) CreateServerOption {
	return func(o *createServerOptions) error {
		if mode != "auto" && mode != "none" {
			return invalid("network mode must be auto or none")
		}
		o.networkInterfaces, o.networkMode = nil, mode
		return nil
	}
}

func cloneServerNetworkInterfaces(nics []ServerNetworkInterface) []ServerNetworkInterface {
	return append([]ServerNetworkInterface(nil), nics...)
}

func networkMicroversionAtLeast(version string, minor uint64) bool {
	return strings.HasPrefix(version, "2.") && microversionAtLeast(version, 2, minor)
}

func (o *createServerOptions) validateNetworkVersion(version string) error {
	if o.networkMode != "" && !networkMicroversionAtLeast(version, 37) {
		return fmt.Errorf("%w: network mode requires Compute microversion 2.37 or later (client uses %q)", resource.ErrUnsupported, version)
	}
	for _, nic := range o.networkInterfaces {
		if nic.Tag.IsSet() && !networkMicroversionAtLeast(version, 42) {
			return fmt.Errorf("%w: network interface tag requires Compute microversion 2.42 or later (client uses %q)", resource.ErrUnsupported, version)
		}
	}
	return nil
}

func (s *Servers) prepareServerNetworks(ctx context.Context, o *createServerOptions) error {
	if len(o.networkInterfaces) == 0 && o.networkMode == "" && s.dependencies.DefaultNetwork != nil {
		if err := checkServerCreation(ctx); err != nil {
			return err
		}
		ref, err := s.dependencies.DefaultNetwork(ctx)
		err = errors.Join(err, checkServerCreation(ctx))
		if err != nil {
			return s.wrap("select default network", err)
		}
		if err := checkServerCreation(ctx); err != nil {
			return err
		}
		if ref != (resource.Ref{}) {
			if err := ref.Validate(); err != nil {
				return s.wrap("select default network", err)
			}
			o.networkInterfaces = []ServerNetworkInterface{{Network: ref}}
		}
	}
	if len(o.networkInterfaces) == 0 {
		if o.networkMode != "" {
			o.base.Networks = o.networkMode
		} else if networkMicroversionAtLeast(s.client.Microversion, 37) {
			// Nova requires an explicit selection at 2.37 and later.
			o.base.Networks = "auto"
		}
		return nil
	}
	rows := make([]map[string]any, len(o.networkInterfaces))
	for i, nic := range o.networkInterfaces {
		row := make(map[string]any)
		for _, selection := range []struct {
			key, kind string
			ref       resource.Ref
			resolve   func(context.Context, resource.Ref) (string, error)
		}{{"uuid", "network", nic.Network, s.dependencies.Network}, {"port", "port", nic.Port, s.dependencies.Port}} {
			if selection.ref == (resource.Ref{}) {
				continue
			}
			if err := checkServerCreation(ctx); err != nil {
				return err
			}
			id := selection.ref.String()
			if selection.ref.IsName() {
				if selection.resolve == nil {
					return fmt.Errorf("%w: %s resolver is unavailable", resource.ErrUnsupported, selection.kind)
				}
				var err error
				id, err = selection.resolve(ctx, selection.ref)
				err = errors.Join(err, checkServerCreation(ctx))
				if err != nil {
					return s.wrap("resolve "+selection.kind, err)
				}
				if err := resource.ID(id).Validate(); err != nil {
					return s.wrap("resolve "+selection.kind, err)
				}
			}
			row[selection.key] = id
		}
		if nic.FixedIP != "" {
			row["fixed_ip"] = nic.FixedIP
		}
		if nic.Tag.IsSet() {
			row["tag"] = nic.Tag
		}
		rows[i] = row
	}
	// The owned overlay preserves empty tags that the native Network string
	// field would omit, while other server fields keep native serialization.
	o.fields["networks"] = rows
	return checkServerCreation(ctx)
}
