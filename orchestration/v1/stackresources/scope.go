package stackresources

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/orchestration/v1/stackresources"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// StackScope fixes both components of the owning stack identity.
type StackScope struct {
	api      *API
	identity stacks.StackIdentity
	values   *resource.Collection[ResourceView]
}

func (a *API) ForStack(identity stacks.StackIdentity) (*StackScope, error) {
	parent, err := stacks.New(a.client).ForStack(identity)
	if err != nil {
		return nil, err
	}
	return a.bindStack(parent.Identity()), nil
}

func (a *API) InStack(ctx context.Context, ref resource.Ref) (*StackScope, error) {
	parent, err := stacks.New(a.client).InStack(ctx, ref)
	if err != nil {
		return nil, err
	}
	return a.bindStack(parent.Identity()), nil
}

func (a *API) bindStack(identity stacks.StackIdentity) *StackScope {
	scope := &StackScope{api: a, identity: identity}
	scope.values = resource.NewCollection(resource.Adapter[ResourceView]{
		Kind: "stackresources", ValidateID: validateName,
		ID:     func(value *ResourceView) string { return value.Name },
		Name:   func(value *ResourceView) string { return value.Name },
		Status: func(value *ResourceView) string { return value.Status },
		Failed: failedStatus, Get: scope.Get,
		List: func(query url.Values) pagination.Pager {
			if err := validateListQuery(query); err != nil {
				return pagination.Pager{Err: err}
			}
			// Heat stores action/status in uppercase. Keep the shared local
			// case-insensitive matcher without narrowing lowercase requests to
			// an empty server result. Clone before changing caller options.
			query = cloneQuery(query)
			for index, status := range query["status"] {
				query["status"][index] = strings.ToUpper(status)
			}
			endpoint := scope.endpoint()
			if len(query) > 0 {
				endpoint += "?" + query.Encode()
			}
			return pagination.NewPager(a.client, endpoint, func(result pagination.PageResult) pagination.Page {
				return resourcePage{SinglePageBase: pagination.SinglePageBase(result)}
			})
		},
		Extract: func(page pagination.Page) ([]ResourceView, error) {
			native := page.(resourcePage)
			items, err := native.entries()
			if err != nil {
				return nil, err
			}
			detailed, _ := strconv.ParseBool(native.URL.Query().Get("with_detail"))
			values := make([]ResourceView, len(items))
			for index, raw := range items {
				view, err := decodeResource(raw, native.Header, native.StatusCode, detailed)
				if err != nil {
					return nil, err
				}
				values[index] = *view
			}
			return values, nil
		},
	})
	return scope
}

// The server returns a single page. Validate the envelope before treating a
// missing/null resources member as an empty list.
type resourcePage struct{ pagination.SinglePageBase }

func (p resourcePage) entries() ([]json.RawMessage, error) {
	var body struct {
		Resources []json.RawMessage `json:"resources"`
	}
	if err := p.ExtractInto(&body); err != nil {
		return nil, err
	}
	if body.Resources == nil {
		return nil, fmt.Errorf("resources must be a JSON array")
	}
	return body.Resources, nil
}

func (p resourcePage) IsEmpty() (bool, error) {
	if p.StatusCode == http.StatusNoContent {
		return true, nil
	}
	items, err := p.entries()
	return len(items) == 0, err
}

func (s *StackScope) Identity() stacks.StackIdentity { return s.identity }

func (s *StackScope) endpoint(segments ...string) string {
	path := []string{"stacks", url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID), "resources"}
	for _, segment := range segments {
		path = append(path, url.PathEscape(segment))
	}
	return s.api.client.ServiceURL(path...)
}

// Get addresses resource_name within the fixed stack, not a physical UUID.
func (s *StackScope) Get(ctx context.Context, name string) (*ResourceView, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("get", "stackresources", err)
	}
	if err := validateName(name); err != nil {
		return nil, err
	}
	var body struct {
		Resource json.RawMessage `json:"resource"`
	}
	response, err := s.api.client.Get(ctx, s.endpoint(name), &body, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, s.wrapHTTP("get", name, err)
	}
	view, err := decodeResource(body.Resource, response.Header, response.StatusCode, true)
	if err == nil && view.Name != name {
		err = fmt.Errorf("response resource_name %q does not match requested %q", view.Name, name)
	}
	if err == nil {
		err = s.checkOwner(view, false)
	}
	if err != nil {
		return nil, request.Wrap("get", "stackresources", err)
	}
	return view, nil
}

func (s *StackScope) List(ctx context.Context, options ...resource.ListOption) iter.Seq2[*ResourceView, error] {
	return s.values.List(ctx, options...)
}

func (s *StackScope) All(ctx context.Context, options ...resource.ListOption) ([]*ResourceView, error) {
	return s.values.All(ctx, options...)
}

func (s *StackScope) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*ResourceView, error) {
	value, err := s.values.Find(ctx, ref, options...)
	if err == nil && value != nil {
		err = s.checkOwner(value, false)
	}
	if err != nil {
		return nil, request.Wrap("find", "stackresources", err)
	}
	return value, nil
}

// ResolveID returns the routing resource_name. A name lookup must establish the
// owner before its result can be converted into an ID for subsequent requests.
func (s *StackScope) ResolveID(ctx context.Context, ref resource.Ref) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", request.Wrap("resolve", "stackresources", err)
	}
	if !ref.IsName() {
		if err := validateName(ref.String()); err != nil {
			return "", err
		}
		return ref.String(), nil
	}
	value, err := s.Find(ctx, ref)
	if err != nil {
		return "", err
	}
	if err := s.checkOwner(value, true); err != nil {
		return "", request.Wrap("resolve", "stackresources", err)
	}
	return value.Name, nil
}

// Metadata preserves Heat's JSON object values, including nested objects and
// arrays. The generated API's map[string]string remains available separately.
func (s *StackScope) Metadata(ctx context.Context, ref resource.Ref) (map[string]any, error) {
	name, err := s.ResolveID(ctx, ref)
	if err != nil {
		return nil, err
	}
	result := upstream.Metadata(ctx, s.api.client, url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID), url.PathEscape(name))
	var body struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := result.ExtractInto(&body); err != nil {
		return nil, s.wrapHTTP("metadata", name, err)
	}
	return body.Metadata, nil
}

// MarkUnhealthy verifies the addressed resource and its actual owning stack
// before PATCH. This also avoids Heat's physical-ID fallback for health changes.
func (s *StackScope) MarkUnhealthy(ctx context.Context, ref resource.Ref, unhealthy bool, options ...HealthOption) error {
	config, err := healthConfig(options)
	if err != nil {
		return err
	}
	name, err := s.ResolveID(ctx, ref)
	if err != nil {
		return err
	}
	value, err := s.Get(ctx, name)
	if err != nil {
		return err
	}
	if err := s.checkOwner(value, true); err != nil {
		return request.Wrap("mark unhealthy", "stackresources", err)
	}
	if err := ctx.Err(); err != nil {
		return request.Wrap("mark unhealthy", "stackresources", err)
	}
	return s.api.MarkUnhealthy(ctx, url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID), url.PathEscape(name), MarkUnhealthyOpts{MarkUnhealthy: unhealthy, ResourceStatusReason: config.reason})
}

func cloneQuery(query url.Values) url.Values {
	copy := make(url.Values, len(query))
	for key, values := range query {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}

func (s *StackScope) Wait(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*ResourceView, error) {
	if strings.TrimSpace(status) == "" {
		return nil, fmt.Errorf("%w: target status must not be empty", resource.ErrInvalidOption)
	}
	if err := resource.ValidateWaitOptionsFor[ResourceView](options...); err != nil {
		return nil, err
	}
	name, err := s.ResolveID(ctx, ref)
	if err != nil {
		return nil, err
	}
	return s.values.Wait(ctx, resource.ID(name), status, options...)
}

func (s *StackScope) checkOwner(value *ResourceView, required bool) error {
	if !value.OwnerKnown {
		if required {
			return fmt.Errorf("%w: resource owner is unknown; use a response with a stack/self identity link", resource.ErrUnsupported)
		}
		return nil
	}
	if value.Owner != s.identity {
		return fmt.Errorf("%w: resource belongs to stack %q/%q, not fixed stack %q/%q; bind its ResourceIdentity", resource.ErrInvalidOption, value.Owner.Name, value.Owner.ID, s.identity.Name, s.identity.ID)
	}
	return nil
}

func (s *StackScope) wrapHTTP(operation, name string, err error) error {
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		err = &resource.NotFoundError{Resource: "stackresources", Reference: name, Cause: err}
	}
	return request.Wrap(operation, "stackresources", err)
}

// ResourceScope fixes resource_name and its actual owning stack together.
type ResourceScope struct {
	parent   *StackScope
	identity ResourceIdentity
}

func (a *API) ForResource(identity ResourceIdentity) (*ResourceScope, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	parent, err := a.ForStack(identity.Stack)
	if err != nil {
		return nil, err
	}
	return &ResourceScope{parent: parent, identity: identity}, nil
}

func (s *StackScope) InResource(ctx context.Context, ref resource.Ref) (*ResourceScope, error) {
	name, err := s.ResolveID(ctx, ref)
	if err != nil {
		return nil, err
	}
	return s.api.ForResource(ResourceIdentity{Stack: s.identity, Name: name})
}

func (s *ResourceScope) Identity() ResourceIdentity { return s.identity }
func (s *ResourceScope) Get(ctx context.Context) (*ResourceView, error) {
	return s.parent.Get(ctx, s.identity.Name)
}
func (s *ResourceScope) Metadata(ctx context.Context) (map[string]any, error) {
	return s.parent.Metadata(ctx, resource.ID(s.identity.Name))
}
func (s *ResourceScope) MarkUnhealthy(ctx context.Context, unhealthy bool, options ...HealthOption) error {
	return s.parent.MarkUnhealthy(ctx, resource.ID(s.identity.Name), unhealthy, options...)
}
func (s *ResourceScope) Wait(ctx context.Context, status string, options ...resource.WaitOption) (*ResourceView, error) {
	return s.parent.Wait(ctx, resource.ID(s.identity.Name), status, options...)
}
