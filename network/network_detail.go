package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// NetworkDetail is the SDK-owned network model returned by CreateNetworkDetail
// and UpdateNetworkDetail. Network holds the native known fields. The typed
// extension fields expose the pinned Python Network Body attributes that the
// native model omits. Metadata.Body keeps every member of the network object,
// including unknown members, and Metadata.Header/StatusCode are the response's.
//
// RouterExternal and PortSecurityEnabled follow the Python Body defaults: an
// absent member reads as false, while an explicit null stays nil. The other
// pointers are nil for an absent or null member. ProviderSegmentationID and
// Segments are untyped in Python and stay raw JSON (nil when absent).
type NetworkDetail struct {
	Network
	RouterExternal          *bool
	PortSecurityEnabled     *bool
	ProviderNetworkType     *string
	ProviderPhysicalNetwork *string
	ProviderSegmentationID  json.RawMessage
	MTU                     *int
	DNSDomain               *string
	IsDefault               *bool
	QoSPolicyID             *string
	IPv4AddressScope        *string
	IPv6AddressScope        *string
	AvailabilityZones       []string
	Segments                json.RawMessage
	VLANTransparent         *bool
	VLANQinQ                *bool
	PVLAN                   *bool
	Metadata                resource.Metadata
}

// UnmarshalJSON decodes one network object. A known extension member with a
// JSON type that cannot hold its Python Body type is an error.
func (d *NetworkDetail) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("%w: network must be a JSON object", resource.ErrInvalidOption)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return err
	}
	var base Network
	if err := json.Unmarshal(trimmed, &base); err != nil {
		return err
	}
	var extension struct {
		RouterExternal          *bool    `json:"router:external"`
		PortSecurityEnabled     *bool    `json:"port_security_enabled"`
		ProviderNetworkType     *string  `json:"provider:network_type"`
		ProviderPhysicalNetwork *string  `json:"provider:physical_network"`
		MTU                     *int     `json:"mtu"`
		DNSDomain               *string  `json:"dns_domain"`
		IsDefault               *bool    `json:"is_default"`
		QoSPolicyID             *string  `json:"qos_policy_id"`
		IPv4AddressScope        *string  `json:"ipv4_address_scope"`
		IPv6AddressScope        *string  `json:"ipv6_address_scope"`
		AvailabilityZones       []string `json:"availability_zones"`
		VLANTransparent         *bool    `json:"vlan_transparent"`
		VLANQinQ                *bool    `json:"vlan_qinq"`
		PVLAN                   *bool    `json:"pvlan"`
	}
	if err := json.Unmarshal(trimmed, &extension); err != nil {
		return err
	}
	pythonDefaultFalse := func(key string, value *bool) *bool {
		if _, present := fields[key]; !present {
			return new(bool)
		}
		return value
	}
	raw := func(key string) json.RawMessage {
		if value, present := fields[key]; present {
			return bytes.Clone(value)
		}
		return nil
	}
	*d = NetworkDetail{
		Network:                 base,
		RouterExternal:          pythonDefaultFalse("router:external", extension.RouterExternal),
		PortSecurityEnabled:     pythonDefaultFalse("port_security_enabled", extension.PortSecurityEnabled),
		ProviderNetworkType:     extension.ProviderNetworkType,
		ProviderPhysicalNetwork: extension.ProviderPhysicalNetwork,
		ProviderSegmentationID:  raw("provider:segmentation_id"),
		MTU:                     extension.MTU,
		DNSDomain:               extension.DNSDomain,
		IsDefault:               extension.IsDefault,
		QoSPolicyID:             extension.QoSPolicyID,
		IPv4AddressScope:        extension.IPv4AddressScope,
		IPv6AddressScope:        extension.IPv6AddressScope,
		AvailabilityZones:       extension.AvailabilityZones,
		Segments:                raw("segments"),
		VLANTransparent:         extension.VLANTransparent,
		VLANQinQ:                extension.VLANQinQ,
		PVLAN:                   extension.PVLAN,
		Metadata:                resource.Metadata{Body: fields},
	}
	return nil
}

func networkDetailFromResponse(value *Network, response *rest.Response) (*NetworkDetail, error) {
	if value == nil || response == nil {
		return nil, nil
	}
	body, err := response.Object("network")
	if err != nil {
		return nil, err
	}
	detail := &NetworkDetail{}
	if err := json.Unmarshal(body, detail); err != nil {
		return nil, response.Fail(err)
	}
	detail.Metadata.Header, detail.Metadata.StatusCode = response.Header.Clone(), response.StatusCode
	return detail, nil
}

// CreateNetworkDetail is CreateNetwork with the owned NetworkDetail result. It
// sends the same single request with the same defaults, extension gate and role
// reset. A detail decode failure after an accepted create is an error, and the
// native verification error keeps the partially decoded detail.
func (s *Service) CreateNetworkDetail(ctx context.Context, input CreateNetworkRequest, options ...NetworkOption) (*NetworkDetail, error) {
	value, response, err := s.createNetwork(ctx, input, options)
	detail, decodeErr := networkDetailFromResponse(value, response)
	if decodeErr != nil {
		return nil, errors.Join(err, wrapNetworkMutation("create/decode", decodeErr))
	}
	return detail, err
}

// UpdateNetworkDetail is UpdateNetwork with the owned NetworkDetail result. The
// optionless branch returns the current GET/list row (with that response's
// metadata) after the same lookup and role reset; a mutation projects the
// single accepted PUT response.
func (s *Service) UpdateNetworkDetail(ctx context.Context, ref resource.Ref, options ...NetworkOption) (*NetworkDetail, error) {
	var current *NetworkDetail
	lookup := func(ctx context.Context, guard func(context.Context) error) (*Network, error) {
		found, err := s.networkDetailLookup(guard).Find(ctx, ref)
		if found == nil {
			return nil, err
		}
		current = found
		return &found.Network, err
	}
	value, response, err := s.updateNetwork(ctx, ref, options, lookup)
	if response == nil {
		if value == nil {
			return nil, err
		}
		return current, err
	}
	detail, decodeErr := networkDetailFromResponse(value, response)
	if decodeErr != nil {
		return nil, errors.Join(err, wrapNetworkMutation("update/decode", decodeErr))
	}
	return detail, err
}

func (s *Service) networkDetailLookup(guard func(context.Context) error) *resource.Collection[NetworkDetail] {
	return rest.Collection(rest.CollectionSpec[NetworkDetail]{
		Client: s.client, Path: "networks", Kind: "network", SingleKey: "network", PluralKey: "networks",
		ID: func(value *NetworkDetail) string { return value.ID }, Name: func(value *NetworkDetail) string { return value.Name },
		NameQuery: func(name string) string { return name },
		Metadata:  func(value *NetworkDetail) *resource.Metadata { return &value.Metadata },
		Validate:  guard, SourceGuard: guard, Get: true, GetCodes: []int{http.StatusOK},
		ListCodes: []int{http.StatusOK, http.StatusNoContent}, Paging: rest.PagePolicy[NetworkDetail]{HTTPLink: true},
	})
}
