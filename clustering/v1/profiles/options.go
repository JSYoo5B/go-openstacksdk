package profiles

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type CreateOpts struct {
	Name     string          `json:"name"`
	Spec     json.RawMessage `json:"spec"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// UpdateOpts only exposes mutable fields. Metadata nil omits it; raw JSON null
// and an empty object remain explicit values. The server validates null support.
type UpdateOpts struct {
	Name     *string         `json:"name,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type ValidateOpts struct {
	Spec json.RawMessage `json:"spec"`
}

type ListOpts struct {
	// MaxItems counts wire rows before local filtering; zero is unlimited.
	MaxItems int
	// Paginated nil uses all pages; an explicit false returns one page.
	Paginated     *bool
	Limit         int
	Marker        string
	Name          string
	Type          string
	Sort          string
	GlobalProject *bool
}

type CreateOption = request.Option[CreateOpts]
type UpdateOption = request.Option[UpdateOpts]
type ValidateOption = request.Option[ValidateOpts]
type ListOption = request.Option[ListOpts]

func WithCreateOptions(value CreateOpts) CreateOption       { return senlin.Snapshot(value) }
func WithUpdateOptions(value UpdateOpts) UpdateOption       { return senlin.Snapshot(value) }
func WithValidateOptions(value ValidateOpts) ValidateOption { return senlin.Snapshot(value) }
func WithListOptions(value ListOpts) ListOption             { return senlin.Snapshot(value) }

func withJSON[T any](value any, set func(*T, json.RawMessage)) request.Option[T] {
	encoded, err := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: profile JSON input: %v", resource.ErrInvalidOption, err)
		}
		set(&config.Options, append(json.RawMessage(nil), encoded...))
		return nil
	}
}

func WithCreateSpec(value any) CreateOption {
	return withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Spec = raw })
}
func WithCreateMetadata(value any) CreateOption {
	return withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Metadata = raw })
}
func WithUpdateMetadata(value any) UpdateOption {
	return withJSON(value, func(opts *UpdateOpts, raw json.RawMessage) { opts.Metadata = raw })
}
func WithValidateSpec(value any) ValidateOption {
	return withJSON(value, func(opts *ValidateOpts, raw json.RawMessage) { opts.Spec = raw })
}
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}
func WithValidateField(key string, value any) ValidateOption {
	return request.WithField[ValidateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithUpdateHeader(key, value string) UpdateOption {
	return request.WithHeader[UpdateOpts](key, value)
}
func WithValidateHeader(key, value string) ValidateOption {
	return request.WithHeader[ValidateOpts](key, value)
}

// WithListMaxItems limits wire rows before local filtering. Zero is unlimited.
func WithListMaxItems(value int) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.MaxItems = value; return nil }
}

// WithListPaginated controls continuation without changing server query fields.
func WithListPaginated(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.Paginated = &copy
		return nil
	}
}

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }
func WithListGlobalProject(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
		return nil
	}
}

func listQuery(config request.Config[ListOpts]) (url.Values, error) {
	if err := senlin.ValidateListCapabilities(config, localFiltersKey); err != nil {
		return nil, err
	}
	if err := senlin.RejectListControlQuery(config.Query); err != nil {
		return nil, err
	}
	value := config.Options
	if value.MaxItems < 0 {
		return nil, fmt.Errorf("%w: max items must be non-negative", resource.ErrInvalidOption)
	}
	if err := senlin.SortGrammar(value.Sort); err != nil {
		return nil, err
	}
	query, err := senlin.Query(request.Config[senlin.ListOpts]{Options: senlin.ListOpts{Limit: value.Limit, Marker: value.Marker}})
	if err != nil {
		return nil, err
	}
	for key, field := range map[string]string{"name": value.Name, "type": value.Type, "sort": value.Sort} {
		if field != "" {
			query.Set(key, field)
		}
	}
	if value.GlobalProject != nil {
		query.Set("global_project", strconv.FormatBool(*value.GlobalProject))
	}
	for key, values := range config.Query {
		switch key {
		case "limit", "marker", "max_items", "paginated", "name", "type", "sort", "global_project", "metadata", "spec", "id", "project", "project_id", "domain", "domain_id", "user", "user_id", "created_at", "updated_at":
			return nil, fmt.Errorf("%w: query %q is a concrete option or local profile filter", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, nil
}
