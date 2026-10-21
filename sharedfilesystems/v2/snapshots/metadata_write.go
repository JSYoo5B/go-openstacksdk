package snapshots

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (a *API) writeMetadata(ctx context.Context, op, method, id string, metadata map[string]string) (map[string]string, error) {
	wrap := func(err error) error { return request.Wrap(op, "snapshots", err) }
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap(err)
	}
	copy := make(map[string]string, len(metadata))
	for k, v := range metadata {
		copy[k] = v
	}
	resp, err := manilaown.Call{Client: a.client, Method: method, URL: a.client.ServiceURL("snapshots", id, "metadata"), Body: map[string]any{"metadata": copy}, OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	out, err := manilaown.MetadataMap(resp, "metadata")
	return out, wrap(err)
}

// SetMetadata merges keys with POST /snapshots/{id}/metadata
// (set_share_snapshot_metadata with replace=False). A nil map sends
// {"metadata":{}}. Only HTTP 200 succeeds and the service's resulting metadata
// is returned. The call is never replayed.
func (a *API) SetMetadata(ctx context.Context, id string, metadata map[string]string) (map[string]string, error) {
	return a.writeMetadata(ctx, "SetMetadata", http.MethodPost, id, metadata)
}

// ReplaceMetadata replaces all metadata with PUT (replace=True). A nil or
// empty map clears the metadata. Only HTTP 200 succeeds.
func (a *API) ReplaceMetadata(ctx context.Context, id string, metadata map[string]string) (map[string]string, error) {
	return a.writeMetadata(ctx, "ReplaceMetadata", http.MethodPut, id, metadata)
}
