package clusters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// MetadataView is an observed GET result. Values is independent of Cluster's
// raw fields. Present and Null distinguish omitted metadata from explicit null.
type MetadataView struct {
	Values  map[string]json.RawMessage
	Present bool
	Null    bool
	Cluster *Cluster
}

// MetadataResult separates observed and requested metadata from the accepted
// cluster response. Changed means a replacement PATCH was accepted, not applied.
// An empty-key Delete has no observation and leaves ResponseCluster nil.
type MetadataResult struct {
	Previous        map[string]json.RawMessage
	Requested       map[string]json.RawMessage
	PreviousPresent bool
	PreviousNull    bool
	ResponseCluster *Cluster
	Operation       *actions.Submission
	Changed         bool
}

// MetadataOpts has no body or query fields; metadata helpers accept headers only.
type MetadataOpts struct{}
type MetadataOption = request.Option[MetadataOpts]

func WithMetadataHeader(key, value string) MetadataOption {
	return request.WithHeader[MetadataOpts](key, value)
}

func cloneMetadata(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	copy := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		copy[key] = bytes.Clone(value)
	}
	return copy
}

func metadataValues(value *Cluster) (map[string]json.RawMessage, bool, bool, error) {
	raw, present := value.Body["metadata"]
	if !present {
		return nil, false, false, nil
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, true, true, nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil, true, false, fmt.Errorf("%w: cluster metadata must be an object or null", resource.ErrInvalidOption)
	}
	return values, true, false, nil
}

func metadataPreflight(ctx context.Context, client *gophercloud.ServiceClient, ref resource.Ref, options []MetadataOption) (map[string]string, error) {
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, err
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	config, err := request.Apply(MetadataOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = senlin.Headers(config.Headers)
	}
	if err != nil {
		return nil, err
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, err
	}
	return maps.Clone(config.Headers), nil
}

func metadataParentSpec(client *gophercloud.ServiceClient) rest.CollectionSpec[Cluster] {
	parent := spec(client)
	parent.ValidateItem = func(value *Cluster) error {
		var id string
		if err := json.Unmarshal(value.Body["id"], &id); err != nil {
			return fmt.Errorf("%w: metadata Name lookup requires a canonical id string", resource.ErrInvalidOption)
		}
		if err := senlin.Identifier(id); err != nil {
			return err
		}
		value.ID, value.Name = id, ""
		if raw, present := value.Body["name"]; present {
			if err := json.Unmarshal(raw, &value.Name); err != nil {
				return fmt.Errorf("%w: cluster name must be a string or null", resource.ErrInvalidOption)
			}
		}
		return nil
	}
	return parent
}

// metadataRead keeps an explicit ID route independent of the response body.
// Names resolve once; only the selected canonical list ID determines the route.
func (a *API) metadataRead(ctx context.Context, ref resource.Ref, headers map[string]string) (*MetadataView, string, error) {
	client := a.RawClient()
	id, err := rest.Collection(metadataParentSpec(client)).ResolveID(ctx, ref)
	if err != nil {
		return nil, "", err
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, "", err
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, client.ServiceURL("clusters", url.PathEscape(id)), nil, headers, http.StatusOK)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			err = &resource.NotFoundError{Resource: "clustering.clusters", Reference: ref.String(), Cause: err}
		}
		return nil, "", err
	}
	value, err := rest.Decode(response, "cluster", func(value *Cluster) *resource.Metadata { return &value.Metadata })
	if err != nil {
		return nil, "", err
	}
	values, present, null, err := metadataValues(value)
	if err != nil {
		return nil, "", response.Fail(err)
	}
	value.UserMetadata = cloneMetadata(values)
	return &MetadataView{Values: cloneMetadata(values), Present: present, Null: null, Cluster: value}, id, nil
}

// FetchMetadata reads the supported cluster GET route, without inventing a
// metadata subresource endpoint. It does not cache or follow an action Location.
func (a *API) FetchMetadata(ctx context.Context, ref resource.Ref) (*MetadataView, error) {
	_, err := metadataPreflight(ctx, a.RawClient(), ref, nil)
	if err != nil {
		return nil, request.Wrap("FetchMetadata", "clustering.clusters", err)
	}
	value, _, err := a.metadataRead(ctx, ref, nil)
	return value, request.Wrap("FetchMetadata", "clustering.clusters", err)
}

// GetMetadata is the compatibility alias of FetchMetadata.
func (a *API) GetMetadata(ctx context.Context, ref resource.Ref) (*MetadataView, error) {
	return a.FetchMetadata(ctx, ref)
}

func metadataChanged(previous, requested map[string]json.RawMessage) bool {
	if len(previous) != len(requested) {
		return true
	}
	for key, value := range requested {
		if current, present := previous[key]; !present || !senlin.EqualJSON(current, value) {
			return true
		}
	}
	return false
}

func (a *API) metadataReplace(ctx context.Context, view *MetadataView, id string, requested map[string]json.RawMessage, headers map[string]string) (*MetadataResult, error) {
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, err
	}
	result := &MetadataResult{Previous: cloneMetadata(view.Values), Requested: cloneMetadata(requested),
		PreviousPresent: view.Present, PreviousNull: view.Null, ResponseCluster: view.Cluster}
	if !metadataChanged(view.Values, requested) {
		return result, nil
	}
	body, err := json.Marshal(map[string]any{"cluster": map[string]any{"metadata": requested}})
	if err != nil {
		return nil, fmt.Errorf("%w: metadata replacement: %v", resource.ErrInvalidOption, err)
	}
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, err
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPatch, client.ServiceURL("clusters", url.PathEscape(id)), json.RawMessage(body), headers, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	value, err := decodeMutation(client, response, true)
	if err != nil {
		return nil, err
	}
	values, _, _, err := metadataValues(value)
	if err != nil {
		return nil, response.Fail(err)
	}
	value.UserMetadata = cloneMetadata(values)
	result.ResponseCluster, result.Changed = value, true
	operation := *value.Operation
	operation.Header, operation.Body = operation.Header.Clone(), bytes.Clone(operation.Body)
	result.Operation = &operation
	return result, nil
}

// SetMetadata shallow-merges an owned input snapshot into the observed object,
// then submits one full replacement PATCH. There is no server-side CAS or retry.
func (a *API) SetMetadata(ctx context.Context, ref resource.Ref, values map[string]any, options ...MetadataOption) (*MetadataResult, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, request.Wrap("SetMetadata", "clustering.clusters", fmt.Errorf("%w: metadata input: %v", resource.ErrInvalidOption, err))
	}
	var updates map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &updates); err != nil {
		return nil, request.Wrap("SetMetadata", "clustering.clusters", err)
	}
	headers, err := metadataPreflight(ctx, a.RawClient(), ref, options)
	if err != nil {
		return nil, request.Wrap("SetMetadata", "clustering.clusters", err)
	}
	view, id, err := a.metadataRead(ctx, ref, headers)
	if err != nil {
		return nil, request.Wrap("SetMetadata", "clustering.clusters", err)
	}
	requested := cloneMetadata(view.Values)
	if requested == nil {
		requested = make(map[string]json.RawMessage)
	}
	for key, raw := range updates {
		requested[key] = bytes.Clone(raw)
	}
	result, err := a.metadataReplace(ctx, view, id, requested, headers)
	return result, request.Wrap("SetMetadata", "clustering.clusters", err)
}

// DeleteMetadata uses nil keys to clear all metadata. A nonnil empty slice is a
// zero-HTTP no-op; selected keys are removed together in one replacement PATCH.
func (a *API) DeleteMetadata(ctx context.Context, ref resource.Ref, keys []string, options ...MetadataOption) (*MetadataResult, error) {
	clearAll := keys == nil
	keys = slices.Clone(keys)
	headers, err := metadataPreflight(ctx, a.RawClient(), ref, options)
	if err != nil {
		return nil, request.Wrap("DeleteMetadata", "clustering.clusters", err)
	}
	if !clearAll && len(keys) == 0 {
		return &MetadataResult{}, nil
	}
	view, id, err := a.metadataRead(ctx, ref, headers)
	if err != nil {
		return nil, request.Wrap("DeleteMetadata", "clustering.clusters", err)
	}
	requested := make(map[string]json.RawMessage)
	if !clearAll {
		requested = cloneMetadata(view.Values)
		if requested == nil {
			requested = make(map[string]json.RawMessage)
		}
		for _, key := range keys {
			delete(requested, key)
		}
	}
	result, err := a.metadataReplace(ctx, view, id, requested, headers)
	return result, request.Wrap("DeleteMetadata", "clustering.clusters", err)
}
