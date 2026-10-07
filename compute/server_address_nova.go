package compute

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

// The legacy Nova response can carry Neutron-style fields as extensions.
// A present canonical null wins over its legacy alias, as in the pinned loader.
type novaAddressIP struct {
	PortID, FixedIP, FloatingIP string
}

func (ip *novaAddressIP) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, field := range []struct {
		name, alias string
		target      *string
	}{{"port_id", "", &ip.PortID}, {"fixed_ip_address", "fixed_ip", &ip.FixedIP}, {"floating_ip_address", "ip", &ip.FloatingIP}} {
		value, present := fields[field.name]
		if !present && field.alias != "" {
			value, present = fields[field.alias]
		}
		if present {
			if err := json.Unmarshal(value, field.target); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Servers) supplementalNovaIPs(ctx context.Context, portID string, guard func(context.Context) error) iter.Seq2[*network.FloatingIP, error] {
	return func(yield func(*network.FloatingIP, error) bool) {
		spec := rest.CollectionSpec[novaAddressIP]{Client: s.client, Path: "os-floating-ips", Kind: "Nova floating IP",
			SingleKey: "floating_ip", PluralKey: "floating_ips", Validate: guard, SourceGuard: guard,
			ListCodes: []int{http.StatusOK, http.StatusNoContent},
			Metadata:  func(*novaAddressIP) *resource.Metadata { return &resource.Metadata{} }}
		for value, err := range rest.List(ctx, spec, url.Values{}) {
			if err != nil {
				if cleanAddressHTTPStatus(err, 404) {
					return
				}
				yield(nil, err)
				return
			}
			if value.PortID == portID {
				if !yield(&network.FloatingIP{PortID: value.PortID, FixedIP: value.FixedIP, FloatingIP: value.FloatingIP}, nil) {
					return
				}
			}
		}
	}
}

func cleanAddressHTTPStatus(err error, status int) bool {
	switch value := err.(type) {
	case gophercloud.ErrUnexpectedResponseCode:
		return value.Actual == status
	case *gophercloud.ErrUnexpectedResponseCode:
		return value != nil && value.Actual == status
	default:
		return false
	}
}
