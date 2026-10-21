package clusters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PythonMetadataProfile is an explicit compatibility profile for the pinned
// Python MetadataMixin routes of set_cluster_metadata and
// delete_cluster_metadata: POST/PUT /clusters/{id}/metadata and
// DELETE /clusters/{id}/metadata/{key}. Senlin's router registers none of
// them, so a stock deployment answers 404 (kept as the native error and never
// replaced by the GET plus PATCH of API.SetMetadata/DeleteMetadata). Creating
// the profile is an opt-in to those routes, not a server capability claim.
type PythonMetadataProfile struct{ api *API }

// PythonMetadata opts into the Python SDK's cluster metadata child routes.
func (a *API) PythonMetadata() *PythonMetadataProfile { return &PythonMetadataProfile{api: a} }

// PythonMetadataReceipt is the opaque accepted response. The body is never
// decoded, and no metadata is synthesized from the submitted values.
type PythonMetadataReceipt struct {
	StatusCode int
	Header     http.Header
	Body       json.RawMessage
}

type PythonMetadataDeletedKey struct {
	Key     string
	Receipt PythonMetadataReceipt
}

// PythonMetadataDeleteResult holds the clearing PUT or the ordered key
// deletions acknowledged so far; a failure returns the partial result.
type PythonMetadataDeleteResult struct {
	Cleared *PythonMetadataReceipt
	Deleted []PythonMetadataDeletedKey
}

func pythonMetadataCodes() []int {
	codes := make([]int, 100)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}

func (p *PythonMetadataProfile) prepare(ctx context.Context, id string, options []MetadataOption) (string, map[string]string, error) {
	if p == nil || p.api == nil {
		return "", nil, fmt.Errorf("%w: cluster API is required", resource.ErrInvalidOption)
	}
	if err := senlin.Identifier(id); err != nil {
		return "", nil, err
	}
	headers, err := metadataPreflight(ctx, p.api.RawClient(), resource.ID(id), options)
	if err != nil {
		return "", nil, err
	}
	return p.api.RawClient().ServiceURL("clusters", url.PathEscape(id), "metadata"), headers, nil
}

func (p *PythonMetadataProfile) send(ctx context.Context, method, endpoint string, body any, headers map[string]string) (*PythonMetadataReceipt, error) {
	response, err := rest.DoJSON(ctx, p.api.RawClient(), method, endpoint, body, headers, pythonMetadataCodes()...)
	if err != nil {
		return nil, err
	}
	return &PythonMetadataReceipt{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: bytes.Clone(response.Body)}, nil
}

func (p *PythonMetadataProfile) write(ctx context.Context, operation, method, id string, values map[string]any, options []MetadataOption) (*PythonMetadataReceipt, error) {
	if values == nil {
		values = map[string]any{}
	}
	endpoint, headers, err := p.prepare(ctx, id, options)
	if err != nil {
		return nil, request.Wrap(operation, "clustering.clusters", err)
	}
	receipt, err := p.send(ctx, method, endpoint, map[string]any{"metadata": values}, headers)
	return receipt, request.Wrap(operation, "clustering.clusters", err)
}

// SetMetadata POSTs {"metadata": values} (merge request); nil or empty input
// sends an explicit empty object. The cluster ID is a Senlin identifier and is
// never resolved by name.
func (p *PythonMetadataProfile) SetMetadata(ctx context.Context, id string, values map[string]any, options ...MetadataOption) (*PythonMetadataReceipt, error) {
	return p.write(ctx, "SetMetadata", http.MethodPost, id, values, options)
}

// ReplaceMetadata PUTs {"metadata": values}, the Python replace=True request.
func (p *PythonMetadataProfile) ReplaceMetadata(ctx context.Context, id string, values map[string]any, options ...MetadataOption) (*PythonMetadataReceipt, error) {
	return p.write(ctx, "ReplaceMetadata", http.MethodPut, id, values, options)
}

// DeleteMetadata clears all metadata with PUT {"metadata": {}} when keys is nil.
// A nonnil empty slice performs no HTTP. Otherwise each key is deleted in order
// by its own DELETE (duplicates repeat) and the first failure stops the
// sequence, so deletion is not atomic and earlier keys stay deleted.
func (p *PythonMetadataProfile) DeleteMetadata(ctx context.Context, id string, keys []string, options ...MetadataOption) (*PythonMetadataDeleteResult, error) {
	const op = "DeleteMetadata"
	clearAll := keys == nil
	keys = slices.Clone(keys)
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			return nil, request.Wrap(op, "clustering.clusters", fmt.Errorf("%w: metadata key must not be empty", resource.ErrInvalidOption))
		}
	}
	endpoint, headers, err := p.prepare(ctx, id, options)
	if err != nil {
		return nil, request.Wrap(op, "clustering.clusters", err)
	}
	result := &PythonMetadataDeleteResult{}
	if clearAll {
		receipt, err := p.send(ctx, http.MethodPut, endpoint, map[string]any{"metadata": map[string]any{}}, headers)
		result.Cleared = receipt
		return result, request.Wrap(op, "clustering.clusters", err)
	}
	result.Deleted = make([]PythonMetadataDeletedKey, 0, len(keys))
	for _, key := range keys {
		receipt, err := p.send(ctx, http.MethodDelete, endpoint+"/"+url.PathEscape(key), nil, headers)
		if err != nil {
			return result, request.Wrap(op, "clustering.clusters", err)
		}
		result.Deleted = append(result.Deleted, PythonMetadataDeletedKey{Key: key, Receipt: *receipt})
	}
	return result, nil
}
