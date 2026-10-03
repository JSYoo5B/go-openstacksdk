package image

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
)

func (s *Service) prepareSchema(ctx context.Context, options []GetSchemaOption) (*preparedImageMutation, error) {
	var source *gophercloud.ServiceClient
	if s != nil {
		source = s.client
	}
	headers, err := validateImageMutationSource(ctx, source)
	if err != nil {
		return nil, err
	}
	client := *source
	client.MoreHeaders = headers
	// Only the unchanged source capture/check fields are needed here. Schema
	// discovery has no Ref, lookup or synthetic image identity.
	prepared := &preparedImageMutation{service: s, source: source, client: &client, base: source.ServiceURL(), provider: source.ProviderClient}
	policy, err := parseGetSchemaOptions(append([]GetSchemaOption(nil), options...))
	if err == nil {
		err = prepared.check(ctx)
	}
	if err != nil {
		return nil, err
	}
	for key, value := range policy.Headers {
		client.MoreHeaders[key] = value
	}
	return prepared, nil
}
