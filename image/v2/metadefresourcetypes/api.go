// Package metadefresourcetypes provides Glance metadata resource types and
// their associations with literal namespaces. Both lists are finite responses.
package metadefresourcetypes

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "image.metadef_resource_type"

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// NamespaceScope fixes the literal parent and service target for its lifetime.
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

// Create posts a name and supplied optional fields. The server can create a
// missing global resource type; no lookup or separate creation is performed.
func (s *NamespaceScope) Create(ctx context.Context, name string, options ...CreateOption) (*Association, error) {
	p, err := s.capture(ctx, &name)
	if err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	policy, err := prepareCreate(append([]CreateOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	body := map[string]any{"name": name}
	if policy.Prefix != nil {
		body["prefix"] = *policy.Prefix
	}
	if policy.PropertiesTarget != nil {
		body["properties_target"] = *policy.PropertiesTarget
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodPost, p.url(nil), body, nil, http.StatusCreated)
	if err = checkResponse(ctx, p, response, err); err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	var value Association
	if err = json.Unmarshal(response.Body, &value); err != nil {
		return nil, wrap(ctx, "Create", response.Fail(err))
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	if err = p.check(ctx); err != nil {
		return nil, wrap(ctx, "Create", response.Fail(err))
	}
	return &value, nil
}

// Delete removes the association. IgnoreMissing defaults to a clean physically
// owned 404; false retains native rejection and retry handling for that status.
// The global resource type is not deleted.
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
	return acknowledgement(s.namespace, name, response), wrap(ctx, "Delete", err)
}
func checkResponse(ctx context.Context, p *preparedSource, response *rest.Response, err error) error {
	if response != nil {
		if checkErr := p.check(ctx); checkErr != nil {
			err = joinErrors(err, response.Fail(checkErr))
		}
	}
	return err
}
