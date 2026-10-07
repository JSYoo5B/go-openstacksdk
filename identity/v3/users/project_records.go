package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

const userProjectRecordKind = "identity.user_projects"

// UserProjectRecord keeps the actual response separate from the Python-style
// view bound to the requested user. Both resources own their fields and metadata.
type UserProjectRecord struct {
	Resource *resource.RawResource
	Wire     *resource.RawResource
}

// UnmarshalJSON retains arbitrary project fields without native model coercion.
// Keep the Wire pointer used by the shared decoder's metadata hook stable.
func (value *UserProjectRecord) UnmarshalJSON(data []byte) error {
	if value == nil {
		return fmt.Errorf("%w: user-project receiver is required", resource.ErrInvalidOption)
	}
	if err := unmarshalMembershipRecord(data, &value.Wire); err != nil {
		return err
	}
	value.Resource = nil
	return nil
}

func userProjectMetadata(value *UserProjectRecord) *resource.Metadata {
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}

func prepareUserProjectRecord(value *UserProjectRecord, userID string) error {
	if value == nil || value.Wire == nil || value.Wire.Body == nil {
		return fmt.Errorf("%w: user-project response fields are required", resource.ErrInvalidOption)
	}
	view, err := cloneMembershipRecordView(value.Wire, userID)
	if err != nil {
		return err
	}
	if raw, exists := view.Body["options"]; exists {
		trimmed := bytes.TrimSpace(raw)
		if !bytes.Equal(trimmed, []byte("null")) && (len(trimmed) == 0 || trimmed[0] != '{') {
			view.Body["options"] = json.RawMessage("{}")
		}
	}
	value.Resource = view
	return nil
}

func userProjectMarker(value *UserProjectRecord) (string, error) {
	if value == nil || value.Wire == nil {
		return "", fmt.Errorf("%w: user-project wire marker is required", resource.ErrInvalidOption)
	}
	var marker string
	if err := json.Unmarshal(value.Wire.Body["id"], &marker); err != nil {
		return "", fmt.Errorf("%w: user-project wire ID must be a pagination string: %w", resource.ErrInvalidOption, err)
	}
	if strings.TrimSpace(marker) == "" {
		return "", fmt.Errorf("%w: user-project wire ID must be nonempty for pagination", resource.ErrInvalidOption)
	}
	return marker, nil
}

func userProjectFilterValue(value *UserProjectRecord, key string) (json.RawMessage, error) {
	if value == nil || value.Resource == nil || value.Resource.Body == nil {
		return nil, fmt.Errorf("%w: user-project Resource fields are required", resource.ErrInvalidOption)
	}
	return resource.BodyRecordField(value.Resource.Body, key, resource.BodyFieldJSON)
}

func userProjectsWithFilters(ctx context.Context, selected rest.CollectionSpec[UserProjectRecord], base url.Values, control rest.ListControl, filters []resource.ListOption) iter.Seq2[*UserProjectRecord, error] {
	if len(filters) == 0 {
		return rest.ListWithControl(ctx, selected, base, control)
	}
	descriptor := projectRecordFilterDescriptor()
	collection := resource.NewCollection(resource.Adapter[UserProjectRecord]{
		Kind: userProjectRecordKind, FilterDescriptor: descriptor,
		BodyFilterFields: descriptor.Body, BodyFilterValue: userProjectFilterValue,
		IterateControlled: func(ctx context.Context, semantic url.Values, _ resource.ListControl) iter.Seq2[*UserProjectRecord, error] {
			query := maps.Clone(base)
			for key, values := range semantic {
				if _, exists := base[key]; exists {
					err := fmt.Errorf("%w: semantic query %q conflicts with typed/raw query", resource.ErrInvalidOption, key)
					return func(yield func(*UserProjectRecord, error) bool) { yield(nil, err) }
				}
				query[key] = append([]string(nil), values...)
			}
			// Rest counts consumed raw rows before semantic Body filtering.
			return rest.ListWithControl(ctx, selected, query, control)
		},
	})
	return collection.List(ctx, filters...)
}

// ListProjectRecords lazily lists projects available to the explicit user ID.
// It preserves response JSON, applies declared semantic filters and follows
// advertised continuations or a raw-ID marker when an explicit limit is set.
// Break stops requests; each iteration applies options once to a fresh snapshot.
// The native ListProjects method and its native Project results remain available.
func (a *API) ListProjectRecords(ctx context.Context, userID string, options ...ListProjectRecordsOption) iter.Seq2[*UserProjectRecord, error] {
	owned := append([]ListProjectRecordsOption(nil), options...)
	return a.listMembershipRecords(ctx, userID, membershipRecordRead{
		operation: "ListProjectRecords", kind: userProjectRecordKind, plural: "projects",
		prepare: func(ctx context.Context, guard func(context.Context) error) (projectListParameters, error) {
			return prepareProjectList(ctx, guard, owned)
		},
		project: prepareUserProjectRecord, marker: userProjectMarker,
		iterate: func(ctx context.Context, selected rest.CollectionSpec[UserProjectRecord], parameters projectListParameters) iter.Seq2[*UserProjectRecord, error] {
			return userProjectsWithFilters(ctx, selected, parameters.query, parameters.control, parameters.filters)
		},
	})
}
