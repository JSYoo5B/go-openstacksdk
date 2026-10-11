package images

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (p *preparedImage) metadataTarget(id string) string {
	return p.source.Client.ServiceURL("images", url.PathEscape(id), "metadata")
}
func metadataFields(id string, raw json.RawMessage) map[string]json.RawMessage {
	rawID, _ := json.Marshal(id)
	fields := map[string]json.RawMessage{"id": rawID}
	if raw != nil {
		fields["metadata"] = bytes.Clone(raw)
	}
	return fields
}
func (a *API) fetchMetadata(ctx context.Context, op, id string, options []ReadOption) (*ImageRecord, error) {
	p, err := a.capture(ctx, "2.38")
	if err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	opts, err := applyOptions(ctx, p.guard, ReadOpts{}, copyRead, slices.Clone(options))
	if err == nil {
		err = imageID(id)
	}
	if err == nil {
		err = p.headers(opts.Headers)
	}
	if err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	r, err := rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodGet, p.metadataTarget(id), nil, nil, imageCodes()...)
	if err != nil {
		return imageReceipt(r), imageWrap(ctx, op, err)
	}
	v := imageReceipt(r)
	var wire resource.RawResource
	if !utf8.Valid(r.Body) {
		err = imageInvalid("metadata response must be UTF-8 JSON")
	} else {
		err = json.Unmarshal(r.Body, &wire)
	}
	if err == nil {
		wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
		v.Wire = wire.Clone()
		err = projectImage(v, metadataFields(id, wire.Body["metadata"]))
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = r.Fail(err)
	}
	return v, imageWrap(ctx, op, err)
}

// FetchMetadata requires a JSON object. Unlike GetRecord, syntax errors fail.
func (a *API) FetchMetadata(ctx context.Context, id string, options ...ReadOption) (*ImageRecord, error) {
	return a.fetchMetadata(ctx, "FetchMetadata", id, options)
}

// GetMetadata is the pinned Source's deprecated alias of FetchMetadata.
func (a *API) GetMetadata(ctx context.Context, id string, options ...ReadOption) (*ImageRecord, error) {
	return a.fetchMetadata(ctx, "GetMetadata", id, options)
}

// SetMetadata POSTs the submitted map. The returned view uses that map and
// does not consume the HTTP response body as a replacement resource.
func (a *API) SetMetadata(ctx context.Context, id string, metadata map[string]any, options ...ReadOption) (*ImageRecord, error) {
	const op = "SetMetadata"
	p, err := a.capture(ctx, "2.38")
	if err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	owned, err := json.Marshal(metadata)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	if !utf8.Valid(owned) {
		return nil, imageWrap(ctx, op, imageInvalid("metadata must be UTF-8 JSON"))
	}
	opts, err := applyOptions(ctx, p.guard, ReadOpts{}, copyRead, slices.Clone(options))
	if err == nil {
		err = imageID(id)
	}
	if err == nil {
		err = p.headers(opts.Headers)
	}
	if err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	r, err := rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodPost, p.metadataTarget(id), map[string]json.RawMessage{"metadata": bytes.Clone(owned)}, nil, imageCodes()...)
	if err != nil {
		return imageReceipt(r), imageWrap(ctx, op, err)
	}
	v := imageReceipt(r)
	if err = errors.Join(projectImage(v, metadataFields(id, owned)), p.guard(ctx)); err != nil {
		err = r.Fail(err)
	}
	return v, imageWrap(ctx, op, err)
}

// DeleteMetadata defaults to PUT {"metadata":{}}. WithDeleteMetadataKeys()
// selects a zero-request no-op; keys are deduplicated in first-seen order.
func (a *API) DeleteMetadata(ctx context.Context, id string, options ...DeleteMetadataOption) error {
	const op = "DeleteMetadata"
	p, err := a.capture(ctx, "2.38")
	if err != nil {
		return imageWrap(ctx, op, err)
	}
	opts, err := applyOptions(ctx, p.guard, DeleteMetadataOpts{}, copyDeleteMetadata, slices.Clone(options))
	if err == nil {
		err = imageID(id)
	}
	if err == nil {
		err = p.headers(opts.Headers)
	}
	if err != nil {
		return imageWrap(ctx, op, err)
	}
	if opts.Keys == nil {
		_, err = rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodPut, p.metadataTarget(id), map[string]any{"metadata": map[string]any{}}, nil, imageCodes()...)
		return imageWrap(ctx, op, err)
	}
	keys := make([]string, 0, len(*opts.Keys))
	seen := make(map[string]bool)
	for _, key := range *opts.Keys {
		if err := imageID(key); err != nil {
			return imageWrap(ctx, op, err)
		}
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		if err = p.guard(ctx); err != nil {
			return imageWrap(ctx, op, err)
		}
		_, err = rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodDelete, p.source.Client.ServiceURL("images", url.PathEscape(id), "metadata", url.PathEscape(key)), nil, nil, imageCodes()...)
		if err != nil {
			return imageWrap(ctx, op, err)
		}
	}
	return imageWrap(ctx, op, p.guard(ctx))
}
