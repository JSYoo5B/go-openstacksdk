// Package metadeftags provides Glance namespace tag operations.
// Bulk Set and paged List preserve actual server response evidence.
package metadeftags

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "image.metadef_tag"

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

// Create sends one bodyless POST to a fixed literal tag route.
func (s *NamespaceScope) Create(ctx context.Context, name string, options ...CreateOption) (*Tag, error) {
	p, err := s.capture(ctx, &name)
	if err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	policy, err := prepareCreate(append([]CreateOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodPost, p.url(&name), nil, nil, http.StatusCreated)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Create", err)
}

// Get fetches one fixed literal child; response names and URLs are passive.
func (s *NamespaceScope) Get(ctx context.Context, name string, options ...GetOption) (*Tag, error) {
	p, err := s.capture(ctx, &name)
	if err != nil {
		return nil, wrap(ctx, "Get", err)
	}
	policy, err := prepareGet(append([]GetOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Get", err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodGet, p.url(&name), nil, nil, http.StatusOK)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Get", err)
}

// Update renames at the fixed current route, defaulting the body name to current.
func (s *NamespaceScope) Update(ctx context.Context, current string, options ...UpdateOption) (*Tag, error) {
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
	response, err := rest.DoJSON(ctx, p.client, http.MethodPut, p.url(&current), map[string]string{"name": name}, nil, http.StatusOK)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Update", err)
}

// Delete is strict: only an actual 204 acknowledges deletion; 404 is an error.
func (s *NamespaceScope) Delete(ctx context.Context, name string, options ...DeleteOption) (*Acknowledgement, error) {
	p, err := s.capture(ctx, &name)
	if err != nil {
		return nil, wrap(ctx, "Delete", err)
	}
	policy, err := prepareDelete(append([]DeleteOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Delete", err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodDelete, p.url(&name), nil, nil, http.StatusNoContent)
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

// Set posts submitted names with an owned append header. An empty request does
// not clear stored tags; DeleteAll is the explicit clear operation.
func (s *NamespaceScope) Set(ctx context.Context, names []string, options ...SetOption) (*SetResult, error) {
	p, err := s.capture(ctx, nil)
	if err != nil {
		return nil, wrap(ctx, "Set", err)
	}
	names = slices.Clone(names)
	for _, name := range names {
		if !utf8.ValidString(name) {
			return nil, wrap(ctx, "Set", invalid("tag entries must be valid UTF-8"))
		}
	}
	policy, err := prepareSet(append([]SetOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Set", err)
	}
	appendValue := "False"
	if policy.Append != nil && *policy.Append {
		appendValue = "True"
	}
	p.client.MoreHeaders["X-Openstack-Append"] = appendValue
	tags := make([]map[string]string, 0, len(names))
	for _, name := range names {
		tags = append(tags, map[string]string{"name": name})
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodPost, p.url(nil), map[string]any{"tags": tags}, nil, http.StatusCreated)
	if err = checkResponse(ctx, p, response, err); err != nil {
		return nil, wrap(ctx, "Set", err)
	}
	value := SetResult{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	if err = json.Unmarshal(response.Body, &value); err != nil {
		return nil, wrap(ctx, "Set", response.Fail(err))
	}
	if err = p.check(ctx); err != nil {
		return nil, wrap(ctx, "Set", response.Fail(err))
	}
	return &value, nil
}

func checkResponse(ctx context.Context, p *preparedSource, response *rest.Response, err error) error {
	if response != nil {
		if checkErr := p.check(ctx); checkErr != nil {
			err = joinErrors(err, response.Fail(checkErr))
		}
	}
	return err
}
func decodeResponse(ctx context.Context, p *preparedSource, response *rest.Response, err error) (*Tag, error) {
	if err = checkResponse(ctx, p, response, err); err != nil {
		return nil, err
	}
	var value Tag
	if err = json.Unmarshal(response.Body, &value); err != nil {
		return nil, response.Fail(err)
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	if err = p.check(ctx); err != nil {
		return nil, response.Fail(err)
	}
	return &value, nil
}
