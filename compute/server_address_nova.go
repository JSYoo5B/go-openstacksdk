package compute

import (
	"context"
	"iter"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// The legacy Nova response can carry Neutron-style fields as extensions.
// A present canonical null wins over its legacy alias, as in the pinned loader.
type novaAddressIP struct {
	supplementalAddressIP
}

func (ip *novaAddressIP) UnmarshalJSON(data []byte) error {
	return ip.supplementalAddressIP.decode(data, true)
}

func (s *Servers) supplementalNovaIPs(ctx context.Context, client *gophercloud.ServiceClient, portID string, guard func(context.Context) error, validate func(*supplementalAddressIP) error) iter.Seq2[*supplementalAddressIP, error] {
	return func(yield func(*supplementalAddressIP, error) bool) {
		spec := rest.CollectionSpec[novaAddressIP]{Client: client, Path: "os-floating-ips", Kind: "Nova floating IP",
			SingleKey: "floating_ip", PluralKey: "floating_ips", Validate: guard, SourceGuard: guard,
			ListCodes: []int{http.StatusOK, http.StatusNoContent},
			Metadata:  func(*novaAddressIP) *resource.Metadata { return &resource.Metadata{} }}
		spec.ValidateItem = func(ip *novaAddressIP) error { return validate(&ip.supplementalAddressIP) }
		for value, err := range rest.List(ctx, spec, url.Values{}) {
			if err != nil {
				if cleanAddressHTTPStatus(err, 404) {
					return
				}
				yield(nil, err)
				return
			}
			if value.PortID == portID {
				if !yield(&value.supplementalAddressIP, nil) {
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
