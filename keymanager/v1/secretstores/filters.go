package secretstores

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// The request argument is SDK-private: no additional public ListOpts field,
// custom builder, or Resources/CRUD capability is introduced.
const listFiltersArgument = "secret_store_filters"

type listFilterCapture struct{ options []resource.ListOption }

func capturedListFilters(arguments map[string]any) ([]resource.ListOption, error) {
	value, exists := arguments[listFiltersArgument]
	if !exists {
		return nil, nil
	}
	capture, ok := value.(listFilterCapture)
	if !ok {
		return nil, fmt.Errorf("%w: invalid SDK secret-store filter capture", resource.ErrInvalidOption)
	}
	return append([]resource.ListOption(nil), capture.options...), nil
}

func listFilterOption(option resource.ListOption) ListOption {
	return func(config *request.Config[ListOpts]) error {
		options, err := capturedListFilters(config.Arguments)
		if err != nil {
			return err
		}
		if config.Arguments == nil {
			config.Arguments = make(map[string]any)
		}
		config.Arguments[listFiltersArgument] = listFilterCapture{options: append(options, option)}
		return nil
	}
}

// WithListFilter adds a declared Python SecretStore attribute. Query attributes
// are sent to Barbican; declared non-query Body attributes match locally. Unknown
// attributes are discarded, and repeated attributes use the final value.
// The shared option snapshots its JSON value now and validates lazily before HTTP.
func WithListFilter(field string, value any) ListOption {
	return listFilterOption(resource.WithFilter(field, value))
}

// WithListFilters replaces the complete semantic filter set; nil/empty clears
// it. Typed ListOpts and raw WithListQuery remain independent. A semantic query
// conflicts with an existing typed/raw value for the same wire key, even if equal.
func WithListFilters(values map[string]any) ListOption {
	return listFilterOption(resource.WithFilters(values))
}

func listWithFilters(ctx context.Context, selected rest.CollectionSpec[SecretStore], base url.Values, control rest.ListControl, filters []resource.ListOption) iter.Seq2[*SecretStore, error] {
	if len(filters) == 0 {
		return rest.ListWithControl(ctx, selected, base, control)
	}
	collection := resource.NewCollection(resource.Adapter[SecretStore]{
		Kind: kind,
		BodyFilterFields: map[string]string{
			"id": "id", "created_at": "created_at", "updated_at": "updated_at",
			"secret_store_ref": "secret_store_ref", "secret_store_id": "secret_store_id",
		},
		BodyFilterValue: secretStoreFilterValue,
		FilterDescriptor: &resource.FilterDescriptor{
			Query: map[string]string{
				"limit": "limit", "marker": "marker", "name": "name", "status": "status",
				"global_default": "global_default", "crypto_plugin": "crypto_plugin",
				"secret_store_plugin": "secret_store_plugin", "created": "created", "updated": "updated",
			},
			Body: map[string]string{
				"id": "id", "created_at": "created_at", "updated_at": "updated_at",
				"secret_store_ref": "secret_store_ref", "secret_store_id": "secret_store_id",
			},
			Reserved: []string{"allow_unknown_params", "base_path", "list_base_path", "headers", "jmespath_filters", "max_items", "microversion", "paginated", "resource_type", "session"},
		},
		IterateControlled: func(ctx context.Context, semantic url.Values, _ resource.ListControl) iter.Seq2[*SecretStore, error] {
			query := maps.Clone(base)
			for key, values := range semantic {
				if _, exists := base[key]; exists {
					err := fmt.Errorf("%w: semantic query %q conflicts with typed/raw query", resource.ErrInvalidOption, key)
					return func(yield func(*SecretStore, error) bool) { yield(nil, err) }
				}
				query[key] = append([]string(nil), values...)
			}
			// The existing owned transport applies controls to raw rows before
			// the collection's semantic Body filter, including its limit hint.
			return rest.ListWithControl(ctx, selected, query, control)
		},
	})
	return collection.List(ctx, filters...)
}

// Project Python attributes from original row fields, never the convenience ID
// or a parsed timestamp. Response refs are passive and cannot become targets.
func secretStoreFilterValue(value *SecretStore, key string) (json.RawMessage, error) {
	if value == nil || value.Body == nil {
		return nil, fmt.Errorf("%w: secret-store response fields are required", resource.ErrInvalidOption)
	}
	field := key
	switch key {
	case "id":
		if _, exists := value.Body["id"]; !exists {
			field = "secret_store_ref"
		}
	case "created_at":
		field = "created"
	case "updated_at":
		field = "updated"
	case "secret_store_ref":
	case "secret_store_id":
		raw, err := resource.BodyRecordField(value.Body, "secret_store_ref", resource.BodyFieldJSON)
		if err != nil {
			return nil, err
		}
		projected, err := jsonfilter.ReferenceLastComponent(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: secret_store_ref formatter: %w", resource.ErrInvalidOption, err)
		}
		return projected, nil
	default:
		return nil, fmt.Errorf("%w: unsupported secret-store Body filter %q", resource.ErrInvalidOption, key)
	}
	return resource.BodyRecordField(value.Body, field, resource.BodyFieldJSON)
}
