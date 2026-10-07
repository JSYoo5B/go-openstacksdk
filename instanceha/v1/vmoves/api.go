package vmoves

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/masakari"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API     { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

type NotificationScope struct {
	*resource.Collection[VMove]
	Resources      *resource.Collection[VMove]
	client         *gophercloud.ServiceClient
	notificationID string
	spec           rest.CollectionSpec[VMove]
}

// InNotification requires selected >=1.3 before resolving the UUID parent.
// Notifications have no names, so binding an explicit ID requires no GET.
func (a *API) InNotification(ctx context.Context, ref resource.Ref) (*NotificationScope, error) {
	if err := masakari.Require(ctx, a.client, 3); err != nil {
		return nil, request.Wrap("InNotification", "vmoves", err)
	}
	if err := ref.Validate(); err != nil {
		return nil, request.Wrap("InNotification", "vmoves", err)
	}
	if ref.IsName() {
		return nil, request.Wrap("InNotification", "vmoves", fmt.Errorf("%w: notifications have no name identity", resource.ErrUnsupported))
	}
	id := ref.String()
	if err := masakari.UUID(id); err != nil {
		return nil, request.Wrap("InNotification", "vmoves", err)
	}
	scope := &NotificationScope{client: a.client, notificationID: id}
	spec := rest.CollectionSpec[VMove]{
		Client: a.client, Path: "notifications/" + url.PathEscape(id) + "/vmoves", Kind: "vmoves", SingleKey: "vmove", PluralKey: "vmoves",
		ID: func(v *VMove) string { return v.UUID }, Status: func(v *VMove) string { return v.Status },
		Metadata: func(v *VMove) *resource.Metadata { v.NotificationID = id; return &v.Metadata },
		Validate: scope.validate, ValidateQuery: scope.validateQuery, ValidateID: masakari.UUID, Get: true,
		Failed: func(status string) bool { return strings.EqualFold(status, "failed") }, Paging: rest.PagePolicy[VMove]{HTTPLink: true},
	}
	scope.spec = spec
	scope.Resources = rest.Collection(spec)
	scope.Collection = scope.Resources
	return scope, nil
}

func (s *NotificationScope) NotificationID() string                { return s.notificationID }
func (s *NotificationScope) RawClient() *gophercloud.ServiceClient { return s.client }
func (s *NotificationScope) validate(ctx context.Context) error {
	if err := masakari.Require(ctx, s.client, 3); err != nil {
		return err
	}
	return masakari.UUID(s.notificationID)
}
func (s *NotificationScope) validateQuery(ctx context.Context, q url.Values) error {
	if err := s.validate(ctx); err != nil {
		return err
	}
	for _, key := range []string{"notification_id", "notification_uuid", "notification"} {
		if q.Has(key) {
			return fmt.Errorf("%w: %s is the fixed URI parent", resource.ErrInvalidOption, key)
		}
	}
	return nil
}

func (s *NotificationScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*VMove, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*VMove, error) bool) {
		query, err := prepareList(options...)
		if err != nil {
			yield(nil, request.Wrap("List", "vmoves", err))
			return
		}
		for value, err := range rest.List(ctx, s.spec, query) {
			if err != nil {
				yield(nil, request.Wrap("List", "vmoves", err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

func (s *NotificationScope) All(ctx context.Context, options ...ListOption) ([]*VMove, error) {
	values := make([]*VMove, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
