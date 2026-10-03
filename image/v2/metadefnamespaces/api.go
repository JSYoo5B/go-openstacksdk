// Package metadefnamespaces provides literal-name Glance metadata namespaces.
// Scalar updates use the server's replacement semantics without an implicit GET.
package metadefnamespaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
)

const kind = "image.metadef_namespace"

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// Create posts a literal namespace and only the supplied scalar fields. Nested
// definitions are outside this operation; omission delegates server defaults.
func (a *API) Create(ctx context.Context, namespace string, options ...CreateOption) (*Namespace, error) {
	p, err := a.capture(ctx)
	if err == nil {
		err = literal(namespace)
	}
	if err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	policy, err := prepareCreate(append([]CreateOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Create", err)
	}
	body := scalarBody(namespace, policy.DisplayName, policy.Description, policy.Visibility, policy.Owner, policy.Protected)
	response, err := rest.DoJSON(ctx, p.client, http.MethodPost, p.url(nil), body, nil, http.StatusCreated)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Create", err)
}

// Get fetches one literal namespace. ResourceType is a server-side projection;
// returned self, schema and namespace fields never become request targets.
func (a *API) Get(ctx context.Context, namespace string, options ...GetOption) (*Namespace, error) {
	p, err := a.capture(ctx)
	if err == nil {
		err = literal(namespace)
	}
	if err != nil {
		return nil, wrap(ctx, "Get", err)
	}
	policy, err := prepareGet(append([]GetOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Get", err)
	}
	endpoint := p.url(&namespace)
	if policy.ResourceType != nil {
		query := url.Values{"resource_type": {*policy.ResourceType}}
		endpoint += "?" + query.Encode()
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodGet, endpoint, nil, nil, http.StatusOK)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Get", err)
}

// Update sends one PUT to the fixed current namespace. Namespace defaults to
// current and may explicitly rename it. Omitted scalar fields are cleared or
// reset by Glance; this operation does not fetch or merge the existing object.
func (a *API) Update(ctx context.Context, current string, options ...UpdateOption) (*Namespace, error) {
	p, err := a.capture(ctx)
	if err == nil {
		err = literal(current)
	}
	if err != nil {
		return nil, wrap(ctx, "Update", err)
	}
	policy, err := prepareUpdate(append([]UpdateOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrap(ctx, "Update", err)
	}
	namespace := current
	if policy.Namespace != nil {
		namespace = *policy.Namespace
	}
	body := scalarBody(namespace, policy.DisplayName, policy.Description, policy.Visibility, policy.Owner, policy.Protected)
	response, err := rest.DoJSON(ctx, p.client, http.MethodPut, p.url(&current), body, nil, http.StatusOK)
	value, err := decodeResponse(ctx, p, response, err)
	return value, wrap(ctx, "Update", err)
}

// Delete quietly handles only a fully owned actual 404 by default. That private
// status bypasses the configured RetryFunc; IgnoreMissing false retains native
// error and retry handling. A masked 404 does not prove the namespace is absent.
func (a *API) Delete(ctx context.Context, namespace string, options ...DeleteOption) (*Acknowledgement, error) {
	p, err := a.capture(ctx)
	if err == nil {
		err = literal(namespace)
	}
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
	response, err := rest.DoJSON(ctx, p.client, http.MethodDelete, p.url(&namespace), nil, nil, codes...)
	err = checkResponse(ctx, p, response, err)
	return acknowledgement(namespace, response), wrap(ctx, "Delete", err)
}

func checkResponse(ctx context.Context, p *preparedSource, response *rest.Response, err error) error {
	if response != nil {
		if checkErr := p.check(ctx); checkErr != nil {
			err = joinErrors(err, response.Fail(checkErr))
		}
	}
	return err
}

func decodeResponse(ctx context.Context, p *preparedSource, response *rest.Response, err error) (*Namespace, error) {
	if err = checkResponse(ctx, p, response, err); err != nil {
		return nil, err
	}
	var value Namespace
	if err = json.Unmarshal(response.Body, &value); err != nil {
		return nil, response.Fail(err)
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	if err = p.check(ctx); err != nil {
		return nil, response.Fail(err)
	}
	return &value, nil
}
