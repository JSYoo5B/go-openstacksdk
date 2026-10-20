package request

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// NeutronFieldsGet is the request lane used by generated Neutron GetWithFields
// methods. BodyNames contains canonical Python Body descriptors, independently
// of list-query aliases. Empty fields omit the query; unknown fields and
// repeated fields retain their spelling and order. Inputs are copied before
// native request callbacks. No service or provider client setting is changed.
// The caller uses its pinned native GetResult for body extraction; successful
// HTTP statuses 200 through 399 follow the dedicated Python read profile.
func NeutronFieldsGet(ctx context.Context, client *gophercloud.ServiceClient, segments, fields []string, bodyNames map[string]string, body any) (http.Header, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: Neutron field selection requires a context", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return nil, neutronDeleteContextError(ctx, nil)
	}
	if client == nil || client.ProviderClient == nil {
		return nil, fmt.Errorf("%w: Neutron field selection requires a service and provider client", resource.ErrInvalidOption)
	}
	parts := append([]string(nil), segments...)
	selected := append([]string(nil), fields...)
	names := maps.Clone(bodyNames)
	target := client.ServiceURL(parts...)
	if len(selected) != 0 {
		parsed, err := url.Parse(target)
		if err != nil {
			return nil, err
		}
		query := parsed.Query()
		for _, field := range selected {
			if wire, declared := names[field]; declared {
				field = wire
			}
			query.Add("fields", field)
		}
		parsed.RawQuery = query.Encode()
		target = parsed.String()
	}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	response, err := client.Get(ctx, target, body, &gophercloud.RequestOpts{OkCodes: codes})
	_, header, err := gophercloud.ParseResponse(response, err)
	if ctx.Err() != nil {
		err = neutronDeleteContextError(ctx, err)
	}
	return header, err
}
