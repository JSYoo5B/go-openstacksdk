package pybodyfilter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// MarkerFromBody returns the wire marker of a row as pinned Resource.list
// derives it from value.id: the first present, non-null string or integer
// among keys (typically "id", then the alternate identity such as "uuid").
func MarkerFromBody(metadata *resource.Metadata, keys ...string) (string, error) {
	if metadata == nil {
		return "", fmt.Errorf("%w: missing row metadata for the wire marker", resource.ErrInvalidOption)
	}
	for _, key := range keys {
		raw := bytes.TrimSpace(metadata.Body[key])
		if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
			continue
		}
		switch raw[0] {
		case '"':
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return "", err
			}
			return text, nil
		case '{', '[', 't', 'f':
			return "", fmt.Errorf("%w: wire %s must be a JSON string or integer", resource.ErrInvalidOption, key)
		}
		var number json.Number
		if err := json.Unmarshal(raw, &number); err != nil {
			return "", err
		}
		return number.String(), nil
	}
	return "", fmt.Errorf("%w: row has no wire identity to use as a pagination marker", resource.ErrInvalidOption)
}

// PythonPaging mirrors Resource.list continuation: body links (links,
// <plural>_links, then a truthy next), Link header, then the wire identity of
// the last row as marker when an explicit limit was sent and the page was not
// empty. An empty page ends the list.
func PythonPaging[T any](metadata func(*T) *resource.Metadata, markerKeys ...string) rest.PagePolicy[T] {
	return rest.PagePolicy[T]{
		HTTPLink: true, SourceLinkPolicy: true, IgnoreFalseyNext: true,
		MarkerFallback: true, MarkerOnShortPage: true, StopOnEmptyPage: true, MaxItemsLimitHint: true,
		Marker: func(value *T) (string, error) {
			if value == nil {
				return "", fmt.Errorf("%w: missing row for the wire marker", resource.ErrInvalidOption)
			}
			return MarkerFromBody(metadata(value), markerKeys...)
		},
	}
}

// REST wraps an SDK-owned rest collection spec with Body filters evaluated on
// each decoded row's retained Metadata.Body. Kinds selects an audited
// response descriptor (integer or boolean) per canonical field; other fields
// compare as raw JSON. Server query extensions are accepted exactly as the
// spec's ValidateQuery allows.
func REST[T any](spec rest.CollectionSpec[T], aliases map[string]string, kinds map[string]resource.BodyFieldType) *resource.Collection[T] {
	adapter := resource.Adapter[T]{Kind: spec.Kind, ID: spec.ID, Name: spec.Name, Status: spec.Status, Failed: spec.Failed, ValidateID: spec.ValidateID}
	adapter.Iterate = func(ctx context.Context, query url.Values) iter.Seq2[*T, error] {
		return rest.List(ctx, spec, query)
	}
	adapter.IterateControlled = func(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*T, error] {
		return rest.ListWithControl(ctx, spec, query, rest.ListControl{MaxItems: control.MaxItems, SinglePage: control.SinglePage, LimitHint: spec.Paging.MaxItemsLimitHint})
	}
	adapter.BodyFilterFields = aliases
	adapter.BodyFilterValue = func(value *T, field string) (json.RawMessage, error) {
		if value == nil {
			return nil, fmt.Errorf("%w: list row is required", resource.ErrInvalidOption)
		}
		metadata := spec.Metadata(value)
		if metadata == nil || metadata.Body == nil {
			return nil, fmt.Errorf("%w: list row is not an object", resource.ErrInvalidOption)
		}
		return resource.BodyRecordField(metadata.Body, field, kinds[field])
	}
	return resource.NewCollection(adapter)
}
