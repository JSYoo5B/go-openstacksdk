package keypairs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// KeypairListOpts selects server queries and local iteration controls. Nil
// Microversion retains a selected version or discovers at most 2.10. An
// explicit empty microversion performs a versionless request.
type KeypairListOpts struct {
	UserID       string
	Limit        int
	Marker       string
	MaxItems     int
	Paginated    *bool
	Microversion *string
}

type KeypairListOption = request.Option[KeypairListOpts]

func copyKeypairListOptions(value KeypairListOpts) KeypairListOpts {
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	if value.Microversion != nil {
		owned := *value.Microversion
		value.Microversion = &owned
	}
	return value
}

// WithKeypairListOptions replaces typed options while retaining headers, raw
// query and semantic filters. Mutable pointer values are owned now.
func WithKeypairListOptions(value KeypairListOpts) KeypairListOption {
	owned := copyKeypairListOptions(value)
	return func(config *request.Config[KeypairListOpts]) error {
		config.Options = copyKeypairListOptions(owned)
		return nil
	}
}
func WithKeypairListUserID(value string) KeypairListOption {
	return func(config *request.Config[KeypairListOpts]) error { config.Options.UserID = value; return nil }
}
func WithKeypairListLimit(value int) KeypairListOption {
	return func(config *request.Config[KeypairListOpts]) error { config.Options.Limit = value; return nil }
}
func WithKeypairListMarker(value string) KeypairListOption {
	return func(config *request.Config[KeypairListOpts]) error { config.Options.Marker = value; return nil }
}

// WithKeypairListMaxItems caps consumed raw rows before local Body filtering.
func WithKeypairListMaxItems(value int) KeypairListOption {
	return func(config *request.Config[KeypairListOpts]) error { config.Options.MaxItems = value; return nil }
}
func WithKeypairListPaginated(value bool) KeypairListOption {
	return func(config *request.Config[KeypairListOpts]) error {
		owned := value
		config.Options.Paginated = &owned
		return nil
	}
}
func WithKeypairListMicroversion(value string) KeypairListOption {
	return func(config *request.Config[KeypairListOpts]) error {
		owned := value
		config.Options.Microversion = &owned
		return nil
	}
}
func WithKeypairListHeader(key, value string) KeypairListOption {
	return request.WithHeader[KeypairListOpts](key, value)
}

// WithKeypairListQuery supplies a raw wire query, overriding the same typed
// query. Operation controls require their dedicated helpers.
func WithKeypairListQuery(key, value string) KeypairListOption {
	return request.WithQuery[KeypairListOpts](key, value)
}

const keypairListFiltersArgument = "keypair_record_filters"

type keypairListFilterCapture struct{ options []resource.ListOption }

func capturedKeypairListFilters(arguments map[string]any) ([]resource.ListOption, error) {
	value, exists := arguments[keypairListFiltersArgument]
	if !exists {
		return nil, nil
	}
	capture, ok := value.(keypairListFilterCapture)
	if !ok {
		return nil, fmt.Errorf("%w: invalid SDK keypair filter capture", resource.ErrInvalidOption)
	}
	return append([]resource.ListOption(nil), capture.options...), nil
}
func keypairListFilterOption(option resource.ListOption) KeypairListOption {
	return func(config *request.Config[KeypairListOpts]) error {
		options, err := capturedKeypairListFilters(config.Arguments)
		if err != nil {
			return err
		}
		if config.Arguments == nil {
			config.Arguments = make(map[string]any)
		}
		config.Arguments[keypairListFiltersArgument] = keypairListFilterCapture{options: append(options, option)}
		return nil
	}
}

// WithKeypairListFilter classifies a declared server query or local Body field.
// Unknown attributes are ignored; repeated attributes use the final value.
func WithKeypairListFilter(field string, value any) KeypairListOption {
	return keypairListFilterOption(resource.WithFilter(field, value))
}

// WithKeypairListFilters replaces only the semantic filter set. Nil clears it.
func WithKeypairListFilters(values map[string]any) KeypairListOption {
	return keypairListFilterOption(resource.WithFilters(values))
}

// Own mutable carriers around each callback without reapplying its options.
func copyKeypairReadConfig[T any](config *request.Config[T]) {
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	config.Query = maps.Clone(config.Query)
	if config.Query == nil {
		config.Query = make(url.Values)
	}
	for key, values := range config.Query {
		config.Query[key] = append([]string(nil), values...)
	}
	config.Fields = maps.Clone(config.Fields)
	if config.Fields == nil {
		config.Fields = make(map[string]json.RawMessage)
	}
	for key, raw := range config.Fields {
		config.Fields[key] = append(json.RawMessage(nil), raw...)
	}
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
	if value, exists := config.Arguments[keypairListFiltersArgument]; exists {
		if capture, ok := value.(keypairListFilterCapture); ok {
			config.Arguments[keypairListFiltersArgument] = keypairListFilterCapture{options: append([]resource.ListOption(nil), capture.options...)}
		}
	}
}

type keypairListParameters struct {
	query        url.Values
	control      rest.ListControl
	filters      []resource.ListOption
	microversion *string
	headers      map[string]string
}

func prepareKeypairList(ctx context.Context, guard func(context.Context) error, options []KeypairListOption) (keypairListParameters, error) {
	guarded := make([]KeypairListOption, len(options))
	for index, option := range options {
		apply := option
		guarded[index] = func(config *request.Config[KeypairListOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil keypair-list option", resource.ErrInvalidOption)
			}
			config.Options = copyKeypairListOptions(config.Options)
			copyKeypairReadConfig(config)
			applyErr := apply(config)
			config.Options = copyKeypairListOptions(config.Options)
			copyKeypairReadConfig(config)
			return errors.Join(applyErr, guard(ctx))
		}
	}
	config, err := request.Apply(KeypairListOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, true, keypairListFiltersArgument)
	}
	if err != nil {
		return keypairListParameters{}, err
	}
	filters, err := capturedKeypairListFilters(config.Arguments)
	if err != nil {
		return keypairListParameters{}, err
	}
	value := copyKeypairListOptions(config.Options)
	if value.Limit < 0 || value.MaxItems < 0 {
		return keypairListParameters{}, fmt.Errorf("%w: limit and max items must be non-negative", resource.ErrInvalidOption)
	}
	query := make(url.Values)
	if value.UserID != "" {
		query.Set("user_id", value.UserID)
	}
	if value.Limit != 0 {
		query.Set("limit", strconv.Itoa(value.Limit))
	}
	if value.Marker != "" {
		query.Set("marker", value.Marker)
	}
	for key, values := range config.Query {
		if strings.TrimSpace(key) == "" {
			return keypairListParameters{}, fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
		}
		if keypairRecordControl(key) {
			return keypairListParameters{}, fmt.Errorf("%w: list control %q requires a dedicated SDK option", resource.ErrUnsupported, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return keypairListParameters{query: query, control: rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated, LimitHint: true}, filters: filters, microversion: value.Microversion, headers: maps.Clone(config.Headers)}, nil
}

func keypairRecordControl(key string) bool {
	switch strings.ToLower(key) {
	case "max_items", "paginated", "session", "resource_type", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "jmespath_filters":
		return true
	}
	return false
}

func keypairRecordFilterDescriptor() *resource.FilterDescriptor {
	return &resource.FilterDescriptor{
		Query:    map[string]string{"user_id": "user_id", "limit": "limit", "marker": "marker"},
		Body:     map[string]string{"id": "id", "created_at": "created_at", "is_deleted": "is_deleted", "fingerprint": "fingerprint", "name": "name", "private_key": "private_key", "public_key": "public_key", "type": "type"},
		Reserved: []string{"max_items", "paginated", "session", "resource_type", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "jmespath_filters"},
	}
}
