// Package attributes manages UUID-addressed Cyborg deployable attributes.
package attributes

import (
	"context"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/common"
	"github.com/JSYoo5B/gophercloudsdk/internal/cyborg"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct {
	*resource.Collection[Attribute]
	Resources *resource.Collection[Attribute]
	client    *gophercloud.ServiceClient
}

func New(client *gophercloud.ServiceClient) *API {
	// Attribute keys can repeat across deployables and are not resource names.
	collection := cyborg.Collection(client, "attributes", "attribute", "attributes",
		func(v *Attribute) string { return v.UUID }, nil, nil,
		func(v *Attribute) *common.Metadata { return &v.Metadata },
		func(id string) string { return client.ServiceURL("attributes", url.PathEscape(id)) }, validateUUID)
	return &API{Collection: collection, Resources: collection, client: client}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create sends a flat object. A singleton plural response is accepted for
// deployments matching the API reference; multiple results are rejected.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Attribute, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cyborg.RequireMicroversion(a.client, 0); err != nil {
		return nil, request.Wrap("Create", "attributes", err)
	}
	body, headers, err := prepareCreate(opts, options...)
	if err == nil {
		headers, err = cyborg.Headers(headers)
	}
	if err != nil {
		return nil, request.Wrap("Create", "attributes", err)
	}
	value, err := cyborg.DecodeSingleMutation(ctx, a.client, "POST", a.client.ServiceURL("attributes"), body, headers,
		"attribute", "attributes", func(v *Attribute) *common.Metadata { return &v.Metadata }, 201)
	return value, request.Wrap("Create", "attributes", err)
}
