// Package metadefproperties provides Glance metadata properties in a literal namespace.
// Updates replace definitions; lists consume one finite ordered dictionary.
package metadefproperties

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "image.metadef_property"

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// NamespaceScope fixes the parent and service target for its lifetime.
type NamespaceScope struct {
	api                                                  *API
	source                                               *gophercloud.ServiceClient
	provider                                             *gophercloud.ProviderClient
	namespace, serviceType, endpoint, base, microversion string
}

// InNamespace validates a literal parent without making an HTTP request.
func (a *API) InNamespace(ctx context.Context, namespace string) (*NamespaceScope, error) {
	source := a.RawClient()
	_, err := validateSource(ctx, source)
	if err == nil {
		err = literal(namespace)
	}
	if err != nil {
		return nil, wrap(ctx, "InNamespace", err)
	}
	return &NamespaceScope{api: a, source: source, provider: source.ProviderClient, namespace: namespace, serviceType: source.Type, endpoint: source.Endpoint, base: source.ServiceURL(), microversion: source.Microversion}, nil
}
func (s *NamespaceScope) NamespaceName() string {
	if s == nil {
		return ""
	}
	return s.namespace
}
func (s *NamespaceScope) RawClient() *gophercloud.ServiceClient {
	if s == nil {
		return nil
	}
	return s.source
}

// Create posts a required name, type and title with a flat definition.
func (s *NamespaceScope) Create(ctx context.Context, name string, options ...CreateOption) (*Property, error) {
	p, err := s.capture(ctx, &name)
	if err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	policy, err := prepareCreate(append([]CreateOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodPost, p.url(nil), propertyBody(name, policy.Type, policy.Title, policy.Description, policy.Attributes), nil, http.StatusCreated)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Create", err)
}

// Get fetches one fixed literal child; response names and URLs are passive.
func (s *NamespaceScope) Get(ctx context.Context, name string, options ...GetOption) (*Property, error) {
	p, err := s.capture(ctx, &name)
	if err != nil {
		return nil, wrap(ctx, "Get", err)
	}
	policy, err := prepareGet(append([]GetOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Get", err)
	}
	endpoint := p.url(&name)
	if policy.ResourceType != nil {
		endpoint += "?" + (url.Values{"resource_type": {*policy.ResourceType}}).Encode()
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodGet, endpoint, nil, nil, http.StatusOK)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Get", err)
}

// Update sends one PUT to current, with name defaulting to current or explicitly
// renamed. Omitted optional fields reset on the server; no fetch or merge occurs.
func (s *NamespaceScope) Update(ctx context.Context, current string, options ...UpdateOption) (*Property, error) {
	p, err := s.capture(ctx, &current)
	if err != nil {
		return nil, wrap(ctx, "Update", err)
	}
	policy, err := prepareUpdate(append([]UpdateOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Update", err)
	}
	name := current
	if policy.Name != nil {
		name = *policy.Name
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodPut, p.url(&current), propertyBody(name, policy.Type, policy.Title, policy.Description, policy.Attributes), nil, http.StatusOK)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Update", err)
}

// Delete defaults to handling only a clean owned physical 404. This private
// status bypasses RetryFunc; IgnoreMissing false retains native error handling.
func (s *NamespaceScope) Delete(ctx context.Context, name string, options ...DeleteOption) (*Acknowledgement, error) {
	p, err := s.capture(ctx, &name)
	if err != nil {
		return nil, wrap(ctx, "Delete", err)
	}
	policy, err := prepareDelete(append([]DeleteOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Delete", err)
	}
	codes := []int{http.StatusNoContent}
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		codes = append(codes, http.StatusNotFound)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodDelete, p.url(&name), nil, nil, codes...)
	err = checkResponse(ctx, p, response, err)
	return acknowledgement(s.namespace, &name, response), wrap(ctx, "Delete", err)
}

// DeleteAll sends one collection DELETE. Missing is always an error; the opaque
// acknowledgement does not report a count or enumerate individual children.
func (s *NamespaceScope) DeleteAll(ctx context.Context, options ...DeleteAllOption) (*Acknowledgement, error) {
	p, err := s.capture(ctx, nil)
	if err != nil {
		return nil, wrap(ctx, "DeleteAll", err)
	}
	policy, err := prepareDeleteAll(append([]DeleteAllOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "DeleteAll", err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodDelete, p.url(nil), nil, nil, http.StatusNoContent)
	err = checkResponse(ctx, p, response, err)
	return acknowledgement(s.namespace, nil, response), wrap(ctx, "DeleteAll", err)
}

func checkResponse(ctx context.Context, p *preparedSource, response *rest.Response, err error) error {
	if response != nil {
		if checkErr := p.check(ctx); checkErr != nil {
			err = joinErrors(err, response.Fail(checkErr))
		}
	}
	return err
}
func decodeResponse(ctx context.Context, p *preparedSource, response *rest.Response, err error) (*Property, error) {
	if err = checkResponse(ctx, p, response, err); err != nil {
		return nil, err
	}
	var value Property
	if err = json.Unmarshal(response.Body, &value); err != nil {
		return nil, response.Fail(err)
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	if err = p.check(ctx); err != nil {
		return nil, response.Fail(err)
	}
	return &value, nil
}
