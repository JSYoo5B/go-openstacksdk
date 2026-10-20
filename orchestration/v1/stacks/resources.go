package stacks

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/orchestration/v1/stacks"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// StackResources owns compound-identity lookup and the shared resource policies.
type StackResources struct {
	api       *API
	values    *resource.Collection[StackResource]
	deletions *resource.Collection[StackResource]
}

// Resources exposes SDK policies alongside the unchanged native API methods.
func (a *API) Resources() *StackResources {
	values := &StackResources{api: a}
	binding := resource.Adapter[StackResource]{
		Kind:       "stacks",
		ValidateID: validateSegment,
		ID:         func(value *StackResource) string { return value.ID },
		Name:       func(value *StackResource) string { return value.Name },
		Get:        func(ctx context.Context, id string) (*StackResource, error) { return a.findID(ctx, id) },
		Delete: func(ctx context.Context, id string) error {
			value, err := a.findID(ctx, id)
			if err != nil {
				return err
			}
			identity, err := value.Identity()
			if err != nil {
				return err
			}
			return a.Delete(ctx, url.PathEscape(identity.Name), url.PathEscape(identity.ID))
		},
		List:    func(query url.Values) pagination.Pager { return resourcePager(a.client, query) },
		Extract: extractResources,
		Status:  func(value *StackResource) string { return value.Status },
		Failed:  failedStatus,
	}
	values.values = resource.NewCollection(binding)
	binding.Get = func(ctx context.Context, id string) (*StackResource, error) {
		value, err := a.findID(ctx, id)
		return deletionState(value, err)
	}
	values.deletions = resource.NewCollection(binding)
	return values
}

func (s *StackResources) Get(ctx context.Context, id string, options ...GetOption) (*StackResource, error) {
	return s.api.findID(ctx, id, options...)
}

func (s *StackResources) List(ctx context.Context, options ...resource.ListOption) iter.Seq2[*StackResource, error] {
	return s.values.List(ctx, options...)
}

func (s *StackResources) All(ctx context.Context, options ...resource.ListOption) ([]*StackResource, error) {
	return s.values.All(ctx, options...)
}

// Find returns a list summary for Name, and a detailed GET result for ID.
func (s *StackResources) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*StackResource, error) {
	return s.values.Find(ctx, ref, options...)
}

// ResolveIdentity preserves both name and UUID. Explicit IDs use Heat's native
// identity GET; names use a complete list with exact local duplicate detection.
func (s *StackResources) ResolveIdentity(ctx context.Context, ref resource.Ref) (StackIdentity, error) {
	value, err := s.Find(ctx, ref)
	if err != nil {
		return StackIdentity{}, err
	}
	return value.Identity()
}

func (s *StackResources) Delete(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return s.values.Delete(ctx, ref, options...)
}

func (s *StackResources) Wait(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*StackResource, error) {
	return s.values.Wait(ctx, ref, status, options...)
}

func (s *StackResources) WaitDeleted(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return s.deletions.WaitDeleted(ctx, ref, options...)
}

type getOptions struct{ resolveOutputs bool }

// GetOption controls library-owned stack retrieval policies.
type GetOption func(*getOptions) error

// WithResolveOutputs matches Python's GET option. The default is true; false
// appends resolve_outputs=False directly to the URL to avoid redirect duplication.
func WithResolveOutputs(value bool) GetOption {
	return func(options *getOptions) error { options.resolveOutputs = value; return nil }
}

func retrievalURL(endpoint string, options []GetOption) (string, error) {
	config := getOptions{resolveOutputs: true}
	for _, apply := range options {
		if apply == nil {
			return "", fmt.Errorf("%w: nil get option", resource.ErrInvalidOption)
		}
		if err := apply(&config); err != nil {
			return "", err
		}
	}
	if !config.resolveOutputs {
		endpoint += "?resolve_outputs=False"
	}
	return endpoint, nil
}

func (a *API) findID(ctx context.Context, id string, options ...GetOption) (*StackResource, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("get", "stacks", err)
	}
	if err := validateSegment(id); err != nil {
		return nil, err
	}
	endpoint, err := retrievalURL(a.client.ServiceURL("stacks", url.PathEscape(id)), options)
	if err != nil {
		return nil, err
	}
	value, err := fetchResource(ctx, a.client, endpoint)
	if err != nil {
		return nil, err
	}
	if value.ID != id {
		return nil, request.Wrap("get", "stacks", fmt.Errorf("stack identity response changed requested ID %q to %q", id, value.ID))
	}
	return value, nil
}

func fetchResource(ctx context.Context, client *gophercloud.ServiceClient, endpoint string) (*StackResource, error) {
	var result upstream.GetResult
	response, err := client.Get(ctx, endpoint, &result.Body, nil)
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	value, err := result.Extract()
	if err != nil {
		if gophercloud.ResponseCodeIs(err, 404) {
			return nil, &resource.NotFoundError{Resource: "stacks", Reference: endpoint, Cause: err}
		}
		return nil, request.Wrap("get", "stacks", err)
	}
	if value == nil {
		return nil, request.Wrap("get", "stacks", fmt.Errorf("stack response is missing its stack object"))
	}
	resource := &StackResource{RetrievedStack: *value, Detailed: true}
	if _, err := resource.Identity(); err != nil {
		return nil, request.Wrap("get", "stacks", err)
	}
	return resource, nil
}

func failedStatus(status string) bool {
	return strings.HasSuffix(strings.ToUpper(status), "_FAILED") || strings.EqualFold(status, "ERROR")
}

func deletionState(value *StackResource, err error) (*StackResource, error) {
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(value.Status, "DELETE_COMPLETE") {
		return nil, &resource.NotFoundError{Resource: "stacks", Reference: value.ID}
	}
	if failedStatus(value.Status) {
		return nil, &resource.FailedStateError{Resource: "stacks", ID: value.ID, Status: value.Status}
	}
	return value, nil
}
