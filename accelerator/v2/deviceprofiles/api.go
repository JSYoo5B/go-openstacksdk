// Package deviceprofiles manages Cyborg device profiles.
package deviceprofiles

import (
	"context"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/common"
	"github.com/JSYoo5B/go-openstacksdk/internal/cyborg"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct {
	*resource.Collection[DeviceProfile]
	Resources *resource.Collection[DeviceProfile]
	client    *gophercloud.ServiceClient
}

// New uses UUIDs for endpoint identifiers. Name references are resolved by an
// exact list lookup first; this avoids the controller's batch name deletion.
func New(client *gophercloud.ServiceClient) *API {
	collection := cyborg.Collection(client, "device_profiles", "device_profile", "device_profiles",
		func(v *DeviceProfile) string { return v.UUID }, func(v *DeviceProfile) string { return v.Name }, nil,
		func(v *DeviceProfile) *common.Metadata { return &v.Metadata },
		func(id string) string { return client.ServiceURL("device_profiles", url.PathEscape(id)) }, validateUUID)
	return &API{Collection: collection, Resources: collection, client: client}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create sends an array containing exactly one profile, matching the Cyborg
// controller rather than inventing a batch creation contract.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*DeviceProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cyborg.RequireMicroversion(a.client, 0); err != nil {
		return nil, request.Wrap("Create", "device_profiles", err)
	}
	body, headers, err := prepareCreate(opts, options...)
	if err == nil {
		headers, err = cyborg.Headers(headers)
	}
	if err != nil {
		return nil, request.Wrap("Create", "device_profiles", err)
	}
	value, err := cyborg.DecodeSingleMutation(ctx, a.client, "POST", a.client.ServiceURL("device_profiles"), body, headers,
		"device_profile", "device_profiles", func(v *DeviceProfile) *common.Metadata { return &v.Metadata }, 201)
	return value, request.Wrap("Create", "device_profiles", err)
}
