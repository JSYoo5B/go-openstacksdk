package notifications

import (
	"context"
	"iter"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/masakari"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct {
	*resource.Collection[Notification]
	Resources *resource.Collection[Notification]
	client    *gophercloud.ServiceClient
	spec      rest.CollectionSpec[Notification]
}

// New exposes only the actual list/get/create contract. Notifications have no
// name lookup and no update/delete operation.
func New(client *gophercloud.ServiceClient) *API {
	spec := rest.CollectionSpec[Notification]{
		Client: client, Path: "notifications", Kind: "notifications", SingleKey: "notification", PluralKey: "notifications",
		ID: func(n *Notification) string { return n.UUID }, Status: func(n *Notification) string { return n.Status },
		Metadata: func(n *Notification) *resource.Metadata { return &n.Metadata },
		Validate: func(ctx context.Context) error { return masakari.Require(ctx, client, 0) }, ValidateID: masakari.UUID, Get: true,
		Failed: func(status string) bool {
			return strings.EqualFold(status, "error") || strings.EqualFold(status, "failed")
		},
		Paging: rest.PagePolicy[Notification]{HTTPLink: true},
	}
	collection := rest.Collection(spec)
	return &API{Collection: collection, Resources: collection, client: client, spec: spec}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Notification, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Notification, error) bool) {
		query, err := prepareList(options...)
		if err != nil {
			yield(nil, request.Wrap("List", "notifications", err))
			return
		}
		for value, err := range rest.List(ctx, a.spec, query) {
			if err != nil {
				yield(nil, request.Wrap("List", "notifications", err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Notification, error) {
	values := make([]*Notification, 0)
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Notification, error) {
	if err := masakari.Require(ctx, a.client, 0); err != nil {
		return nil, request.Wrap("Create", "notifications", err)
	}
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, request.Wrap("Create", "notifications", err)
	}
	if err := masakari.Required(config.Options.Type, config.Options.Hostname, config.Options.GeneratedTime); err != nil {
		return nil, request.Wrap("Create", "notifications", err)
	}
	if err := validatePayload(config.Options.Payload); err != nil {
		return nil, request.Wrap("Create", "notifications", err)
	}
	body, err := masakari.Body(config, "notification", "id", "uuid", "notification_uuid", "status", "source_host_uuid", "recovery_workflow_details", "created_at", "updated_at", "deleted", "deleted_at")
	if err != nil {
		return nil, request.Wrap("Create", "notifications", err)
	}
	response, err := rest.DoJSON(ctx, a.client, "POST", a.client.ServiceURL("notifications"), body, config.Headers, 202)
	if err != nil {
		return nil, request.Wrap("Create", "notifications", err)
	}
	value, err := rest.Decode(response, "notification", a.spec.Metadata)
	return value, request.Wrap("Create", "notifications", err)
}
