package network

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// CreateNetworkRequest keeps the literal name, including an explicit empty name.
// Creation defaults admin_state_up to true and omits shared/external false.
type CreateNetworkRequest struct{ Name string }

// ProviderNetwork describes known provider fields without an application builder.
// Optional distinguishes omitted, null, zero and empty values.
type ProviderNetwork struct {
	NetworkType     request.Optional[string]
	PhysicalNetwork request.Optional[string]
	SegmentationID  request.Optional[any]
}

type networkMutationOptions struct {
	fields   map[string]json.RawMessage
	hints    bool
	project  bool
	revision *int
	create   bool
}

// NetworkOption applies an SDK-owned network field. Options can be reused;
// the last value wins, and an invalid earlier option is never hidden.
type NetworkOption func(*networkMutationOptions) error

func networkMutationInvalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}

func networkField(key string, value any) NetworkOption {
	encoded, encodeErr := json.Marshal(value)
	return func(o *networkMutationOptions) error {
		if encodeErr != nil {
			return networkMutationInvalid("network field is not valid JSON: " + encodeErr.Error())
		}
		o.fields[key] = append(json.RawMessage(nil), encoded...)
		return nil
	}
}

func WithNetworkName(name string) NetworkOption { return networkField("name", name) }
func WithNetworkDescription(description string) NetworkOption {
	return networkField("description", description)
}
func WithNetworkAdminStateUp(enabled bool) NetworkOption {
	return networkField("admin_state_up", enabled)
}
func WithNetworkShared(enabled bool) NetworkOption { return networkField("shared", enabled) }
func WithNetworkExternal(enabled bool) NetworkOption {
	return networkField("router:external", enabled)
}
func WithNetworkPortSecurity(enabled bool) NetworkOption {
	return networkField("port_security_enabled", enabled)
}
func WithNetworkDNSDomain(domain string) NetworkOption { return networkField("dns_domain", domain) }

func optionalNetworkField[T any](key string, value request.Optional[T]) NetworkOption {
	if !value.IsSet() {
		return func(*networkMutationOptions) error { return nil }
	}
	return networkField(key, value)
}

// Value options additionally preserve a supplied JSON null. An unset Optional
// leaves the field unchanged; use ordinary With functions for concrete values.
func WithNetworkNameValue(value request.Optional[string]) NetworkOption {
	return optionalNetworkField("name", value)
}
func WithNetworkAdminStateValue(value request.Optional[bool]) NetworkOption {
	return optionalNetworkField("admin_state_up", value)
}
func WithNetworkSharedValue(value request.Optional[bool]) NetworkOption {
	return optionalNetworkField("shared", value)
}
func WithNetworkExternalValue(value request.Optional[bool]) NetworkOption {
	return optionalNetworkField("router:external", value)
}
func WithNetworkDNSDomainValue(value request.Optional[string]) NetworkOption {
	return optionalNetworkField("dns_domain", value)
}

// WithNetworkMTU accepts zero only for Create (which omits it). Nonzero MTU
// must be at least 68; backend and IPv6-subnet constraints belong to Neutron.
func WithNetworkMTU(size int) NetworkOption {
	set := networkField("mtu", size)
	return func(o *networkMutationOptions) error {
		if size < 0 || size > 0 && size < 68 || !o.create && size == 0 {
			return networkMutationInvalid("network MTU must be zero or at least 68")
		}
		return set(o)
	}
}

// WithNetworkProjectID is create-only. Neutron checks permission to select
// another project's owner; the SDK does not infer authorization from the ID.
func WithNetworkProjectID(id string) NetworkOption {
	set := networkField("project_id", id)
	return func(o *networkMutationOptions) error {
		if !o.create {
			return networkMutationInvalid("project owner is create-only")
		}
		o.project = true
		return set(o)
	}
}

// WithNetworkAvailabilityZoneHints is create-only. Even an empty list checks
// the network_availability_zone extension and sends an explicit [] field.
func WithNetworkAvailabilityZoneHints(hints ...string) NetworkOption {
	set := networkField("availability_zone_hints", append([]string{}, hints...))
	return func(o *networkMutationOptions) error {
		if !o.create {
			return networkMutationInvalid("availability-zone hints are create-only")
		}
		o.hints = true
		return set(o)
	}
}

// WithNetworkProvider replaces the previous provider selection as a whole.
// Unknown provider attributes are supplied through WithNetworkField.
func WithNetworkProvider(provider ProviderNetwork) NetworkOption {
	var fields []NetworkOption
	for _, field := range []struct {
		key   string
		set   bool
		value any
	}{
		{"provider:network_type", provider.NetworkType.IsSet(), provider.NetworkType},
		{"provider:physical_network", provider.PhysicalNetwork.IsSet(), provider.PhysicalNetwork},
		{"provider:segmentation_id", provider.SegmentationID.IsSet(), provider.SegmentationID},
	} {
		if field.set {
			fields = append(fields, networkField(field.key, field.value))
		}
	}
	return func(o *networkMutationOptions) error {
		for _, key := range []string{"provider:network_type", "provider:physical_network", "provider:segmentation_id"} {
			delete(o.fields, key)
		}
		for _, apply := range fields {
			if err := apply(o); err != nil {
				return err
			}
		}
		return nil
	}
}

// WithNetworkRevision is update-only and sends If-Match: revision_number=N.
func WithNetworkRevision(revision int) NetworkOption {
	return func(o *networkMutationOptions) error {
		if o.create {
			return networkMutationInvalid("network revision is update-only")
		}
		if revision < 0 {
			return networkMutationInvalid("network revision must be non-negative")
		}
		value := revision
		o.revision = &value
		return nil
	}
}

var networkMutationCoreFields = map[string]bool{
	"name": true, "description": true, "admin_state_up": true, "shared": true,
	"router:external": true, "project_id": true, "tenant_id": true,
	"availability_zone_hints": true, "port_security_enabled": true, "mtu": true,
	"dns_domain": true, "provider:network_type": true, "provider:physical_network": true,
	"provider:segmentation_id": true, "revision_number": true,
}

// WithNetworkField snapshots an additional JSON extension, including null.
// Known fields must use their concrete options and cannot be overwritten here.
func WithNetworkField(key string, value any) NetworkOption {
	set := networkField(key, value)
	return func(o *networkMutationOptions) error {
		if strings.TrimSpace(key) == "" || networkMutationCoreFields[key] {
			return networkMutationInvalid("empty or reserved network extension field: " + key)
		}
		return set(o)
	}
}

func prepareNetworkMutation(create bool, name string, options []NetworkOption) (networkMutationOptions, error) {
	o := networkMutationOptions{fields: map[string]json.RawMessage{}, create: create}
	if create {
		_ = networkField("name", name)(&o)
		_ = WithNetworkAdminStateUp(true)(&o)
	}
	for _, apply := range options {
		if apply == nil {
			return o, networkMutationInvalid("nil network option")
		}
		if err := apply(&o); err != nil {
			return o, err
		}
	}
	if create {
		if o.revision != nil {
			return o, networkMutationInvalid("network revision is update-only")
		}
		// Pinned cloud creation omits these false/zero/empty default values.
		for key, omitted := range map[string]string{"shared": "false", "router:external": "false", "mtu": "0", "dns_domain": `""`} {
			if string(o.fields[key]) == omitted {
				delete(o.fields, key)
			}
		}
		for _, key := range []string{"shared", "router:external", "dns_domain"} {
			if string(o.fields[key]) == "null" {
				delete(o.fields, key)
			}
		}
	} else {
		if o.hints || o.project {
			return o, networkMutationInvalid("project owner and availability-zone hints are create-only")
		}
		if string(o.fields["mtu"]) == "0" {
			return o, networkMutationInvalid("updated network MTU must be at least 68")
		}
	}
	return o, nil
}
