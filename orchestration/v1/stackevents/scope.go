package stackevents

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// StackEventScope reads events for one complete Heat stack identity.
// A single event also requires its resource name; ForResource fixes that part.
type StackEventScope struct {
	api      *API
	identity stacks.StackIdentity
}

// ResourceEventScope reads events for one logical resource of a fixed stack.
type ResourceEventScope struct {
	stack        *StackEventScope
	resourceName string
}

func (a *API) validateClient() error {
	if a == nil || a.client == nil || a.client.ProviderClient == nil {
		return fmt.Errorf("%w: stack events require a service client", resource.ErrInvalidOption)
	}
	return nil
}

// ForStack binds a known canonical identity without making an HTTP request.
func (a *API) ForStack(identity stacks.StackIdentity) (*StackEventScope, error) {
	if err := a.validateClient(); err != nil {
		return nil, err
	}
	bound, err := stacks.New(a.client).ForStack(identity)
	if err != nil {
		return nil, err
	}
	return &StackEventScope{api: a, identity: bound.Identity()}, nil
}

// InStack reuses the stack SDK's exact name or canonical ID lookup once.
func (a *API) InStack(ctx context.Context, ref resource.Ref) (*StackEventScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("InStack", "stack events", err)
	}
	if err := a.validateClient(); err != nil {
		return nil, err
	}
	bound, err := stacks.New(a.client).InStack(ctx, ref)
	if err != nil {
		return nil, err
	}
	return a.ForStack(bound.Identity())
}

func (s *StackEventScope) Identity() stacks.StackIdentity { return s.identity }

// ForResource binds the resource's logical name, without looking it up or
// treating its physical resource ID as a name.
func (s *StackEventScope) ForResource(name string) (*ResourceEventScope, error) {
	if err := validateEventSegment(name); err != nil {
		return nil, err
	}
	return &ResourceEventScope{stack: s, resourceName: name}, nil
}

func (s *ResourceEventScope) Identity() stacks.StackIdentity { return s.stack.Identity() }
func (s *ResourceEventScope) ResourceName() string           { return s.resourceName }

func validateEventSegment(value string) error {
	if err := resource.ID(value).Validate(); err != nil {
		return err
	}
	if strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%w: event paths require a non-empty unescaped segment", resource.ErrInvalidOption)
	}
	return nil
}

type GetOpts struct{}
type GetOption = request.Option[GetOpts]

func WithGetHeader(key, value string) GetOption { return request.WithHeader[GetOpts](key, value) }
func WithListHeader(key, value string) ListOption {
	return request.WithHeader[ListOpts](key, value)
}
func WithListResourceEventsHeader(key, value string) ListResourceEventsOption {
	return request.WithHeader[ListResourceEventsOpts](key, value)
}

func eventHeaders[T any](cfg request.Config[T]) (map[string]string, error) {
	headers, err := request.MergeHeadersFor(map[string]string{
		"Accept": "application/json", "X-Auth-Token": "", "OpenStack-API-Version": "",
	}, cfg.Headers, cfg.Options)
	delete(headers, "X-Auth-Token")
	delete(headers, "OpenStack-API-Version")
	return headers, err
}

// Get uses the resource-scoped endpoint; Heat has no stack/eventID GET route.
func (s *StackEventScope) Get(ctx context.Context, resourceName, eventID string, options ...GetOption) (*EventResource, error) {
	bound, err := s.ForResource(resourceName)
	if err != nil {
		return nil, err
	}
	return bound.Get(ctx, eventID, options...)
}

func (s *ResourceEventScope) Get(ctx context.Context, eventID string, options ...GetOption) (*EventResource, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("Get", "stack events", err)
	}
	if err := validateEventSegment(eventID); err != nil {
		return nil, err
	}
	cfg, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, true)
	}
	if err != nil {
		return nil, err
	}
	headers, err := eventHeaders(cfg)
	if err != nil {
		return nil, err
	}
	var wire struct {
		Event json.RawMessage `json:"event"`
	}
	response, err := s.stack.api.client.Get(ctx, s.stack.endpoint(s.resourceName)+"/"+url.PathEscape(eventID), &wire,
		&gophercloud.RequestOpts{OkCodes: []int{200}, MoreHeaders: headers})
	if err != nil {
		if gophercloud.ResponseCodeIs(err, 404) {
			err = &resource.NotFoundError{Resource: "stack events", Reference: eventID, Cause: err}
		}
		return nil, request.Wrap("Get", "stack events", err)
	}
	value, err := decodeEvent(wire.Event, response.Header, s.stack.identity, s.resourceName, eventID, true)
	return value, request.Wrap("Get", "stack events", err)
}

func (s *StackEventScope) endpoint(resourceName string) string {
	parts := []string{"stacks", url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID)}
	if resourceName != "" {
		parts = append(parts, "resources", url.PathEscape(resourceName))
	}
	parts = append(parts, "events")
	return s.api.client.ServiceURL(parts...)
}

func (s *StackEventScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*EventResource, error] {
	return func(yield func(*EventResource, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(nil, err)
			return
		}
		cfg, err := request.Apply(ListOpts{}, options...)
		if err == nil {
			err = request.ValidateCapabilities(cfg, false, true, true)
		}
		var query string
		if err == nil {
			query, err = cfg.Options.ToStackEventListQuery()
		}
		if err == nil {
			query, err = request.ExtendQuery(query, cfg.Query)
		}
		headers, headerErr := eventHeaders(cfg)
		if err == nil {
			err = headerErr
		}
		if err != nil {
			yield(nil, err)
			return
		}
		s.list(ctx, "", query, headers)(yield)
	}
}

func (s *ResourceEventScope) List(ctx context.Context, options ...ListResourceEventsOption) iter.Seq2[*EventResource, error] {
	return func(yield func(*EventResource, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(nil, err)
			return
		}
		cfg, err := request.Apply(ListResourceEventsOpts{}, options...)
		if err == nil {
			err = request.ValidateCapabilities(cfg, false, true, true)
		}
		var query string
		if err == nil {
			query, err = cfg.Options.ToResourceEventListQuery()
		}
		if err == nil {
			query, err = request.ExtendQuery(query, cfg.Query)
		}
		headers, headerErr := eventHeaders(cfg)
		if err == nil {
			err = headerErr
		}
		if err != nil {
			yield(nil, err)
			return
		}
		s.stack.list(ctx, s.resourceName, query, headers)(yield)
	}
}

func (s *StackEventScope) All(ctx context.Context, options ...ListOption) ([]*EventResource, error) {
	values := make([]*EventResource, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *ResourceEventScope) All(ctx context.Context, options ...ListResourceEventsOption) ([]*EventResource, error) {
	values := make([]*EventResource, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// eventPage follows Heat's native marker protocol, not links nested in events.
// Raw entries are decoded by the iterator, so break avoids later item errors.
type eventPage struct{ pagination.MarkerPageBase }

func (p eventPage) entries() ([]json.RawMessage, error) {
	var wire struct {
		Events json.RawMessage `json:"events"`
	}
	if err := p.ExtractInto(&wire); err != nil {
		return nil, err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(wire.Events, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		return nil, fmt.Errorf("events must be a JSON array")
	}
	return entries, nil
}

func (p eventPage) IsEmpty() (bool, error) {
	if p.StatusCode == http.StatusNoContent {
		return true, nil
	}
	entries, err := p.entries()
	return len(entries) == 0, err
}

func (p eventPage) LastMarker() (string, error) {
	entries, err := p.entries()
	if err != nil || len(entries) == 0 {
		return "", err
	}
	var last struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(entries[len(entries)-1], &last); err != nil {
		return "", err
	}
	if err := validateEventSegment(last.ID); err != nil {
		return "", err
	}
	return last.ID, nil
}

func (s *StackEventScope) list(ctx context.Context, resourceName, query string, headers map[string]string) iter.Seq2[*EventResource, error] {
	return func(yield func(*EventResource, error) bool) {
		pager := pagination.NewPager(s.api.client, s.endpoint(resourceName)+query, func(result pagination.PageResult) pagination.Page {
			page := eventPage{pagination.MarkerPageBase{PageResult: result}}
			page.Owner = page
			return page
		})
		pager.Headers = headers
		for rawPage, err := range resource.Pages(ctx, pager) {
			if err != nil {
				yield(nil, request.Wrap("List", "stack events", err))
				return
			}
			page := rawPage.(eventPage)
			entries, err := page.entries()
			if err != nil {
				yield(nil, request.Wrap("List", "stack events", err))
				return
			}
			for _, raw := range entries {
				if err := ctx.Err(); err != nil {
					yield(nil, request.Wrap("List", "stack events", err))
					return
				}
				value, err := decodeEvent(raw, page.Header, s.identity, resourceName, "", false)
				if err != nil {
					yield(nil, request.Wrap("List", "stack events", err))
					return
				}
				if !yield(value, nil) {
					return
				}
			}
		}
	}
}
