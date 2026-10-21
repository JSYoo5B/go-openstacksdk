package masakari

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WireIDMarker derives the pagination marker from the original wire "id" of a
// row, as Resource.list does with value.id. Integer and string ids are
// supported; absence, null and any other JSON type are rejected rather than
// silently producing a repeated page.
func WireIDMarker(metadata *resource.Metadata) (string, error) {
	if metadata == nil {
		return "", fmt.Errorf("%w: missing row metadata for the wire marker", resource.ErrInvalidOption)
	}
	raw, exists := metadata.Body["id"]
	raw = bytes.TrimSpace(raw)
	if !exists || len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", fmt.Errorf("%w: row has no wire id to use as a pagination marker", resource.ErrInvalidOption)
	}
	switch raw[0] {
	case '"':
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return "", err
		}
		return text, nil
	case '{', '[', 't', 'f':
		return "", fmt.Errorf("%w: wire id must be a JSON string or integer", resource.ErrInvalidOption)
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return "", err
	}
	return number.String(), nil
}

// PythonPaging mirrors the pinned Resource.list continuation order: body
// links (links, <plural>_links, then a truthy top-level next), Link header,
// then the wire id of the last row as marker when an explicit limit was sent
// and the page was not empty.
func PythonPaging[T any](metadata func(*T) *resource.Metadata) rest.PagePolicy[T] {
	return rest.PagePolicy[T]{
		HTTPLink: true, SourceLinkPolicy: true, IgnoreFalseyNext: true,
		MarkerFallback: true, MarkerOnShortPage: true, StopOnEmptyPage: true,
		Marker: func(value *T) (string, error) {
			if value == nil {
				return "", fmt.Errorf("%w: missing row for the wire marker", resource.ErrInvalidOption)
			}
			return WireIDMarker(metadata(value))
		},
	}
}

// ListMarkerFallback lists with PythonPaging. The caller supplies a prepared
// query and the operation labels used in errors.
func ListMarkerFallback[T any](ctx context.Context, spec rest.CollectionSpec[T], query url.Values, operation, kind string) iter.Seq2[*T, error] {
	spec.Paging = PythonPaging(spec.Metadata)
	return func(yield func(*T, error) bool) {
		for value, err := range rest.List(ctx, spec, query) {
			if err != nil {
				yield(nil, request.Wrap(operation, kind, err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}
