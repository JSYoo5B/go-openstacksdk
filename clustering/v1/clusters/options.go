package clusters

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// CreateOpts leaves omitted sizes to Senlin's defaults. A present zero size or
// MaxSize=-1 remains explicit. ProfileID may be a profile name, UUID or short ID.
type CreateOpts struct {
	Name            string                `json:"name"`
	ProfileID       string                `json:"profile_id"`
	DesiredCapacity *int                  `json:"desired_capacity,omitempty"`
	MinSize         *int                  `json:"min_size,omitempty"`
	MaxSize         *int                  `json:"max_size,omitempty"`
	Timeout         request.Optional[int] `json:"timeout,omitzero"`
	Config          json.RawMessage       `json:"config,omitempty"`
	Metadata        json.RawMessage       `json:"metadata,omitempty"`
}

// UpdateOpts exposes mutable fields only. Optional scalar fields distinguish
// absence, null and a value. Any explicit ProfileOnly requires microversion 1.6.
type UpdateOpts struct {
	Name            request.Optional[string] `json:"name,omitzero"`
	ProfileID       request.Optional[string] `json:"profile_id,omitzero"`
	Timeout         request.Optional[int]    `json:"timeout,omitzero"`
	Config          json.RawMessage          `json:"config,omitempty"`
	Metadata        json.RawMessage          `json:"metadata,omitempty"`
	ProfileOnly     *bool                    `json:"profile_only,omitempty"`
	profileOnlyNull bool
}

// MarshalJSON preserves an explicit profile_only null without changing the
// existing pointer input used for bool values and omission.
func (value UpdateOpts) MarshalJSON() ([]byte, error) {
	type plain UpdateOpts
	encoded, err := json.Marshal(plain(value))
	if err != nil || !value.profileOnlyNull || value.ProfileOnly != nil {
		return encoded, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	fields["profile_only"] = json.RawMessage("null")
	return json.Marshal(fields)
}

// UnmarshalJSON also keeps null presence when WithUpdateOptions snapshots a
// previously decoded input. JSON bool values still use the public pointer.
func (value *UpdateOpts) UnmarshalJSON(encoded []byte) error {
	type plain UpdateOpts
	var decoded plain
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return err
	}
	*value = UpdateOpts(decoded)
	for key := range fields {
		if strings.EqualFold(key, "profile_only") {
			// The plain decoder determines the last value across JSON aliases.
			// A nil pointer with a present field therefore means explicit null.
			value.profileOnlyNull = decoded.ProfileOnly == nil
			break
		}
	}
	return nil
}

// DeleteOpts defaults to ignoring absence for ordinary deletion and requiring
// existence for force deletion. IgnoreMissing explicitly overrides either default.
type DeleteOpts struct {
	Force         bool  `json:"force"`
	IgnoreMissing *bool `json:"ignore_missing,omitempty"`
}

type ListOpts struct {
	// MaxItems counts wire rows before local filtering; zero is unlimited.
	MaxItems int
	// Paginated nil uses all pages; an explicit false returns one page.
	Paginated     *bool
	Limit         int
	Marker        string
	Name          string
	Status        string
	Sort          string
	GlobalProject *bool
}

type CreateOption = request.Option[CreateOpts]
type UpdateOption = request.Option[UpdateOpts]
type DeleteOption = request.Option[DeleteOpts]
type ListOption = request.Option[ListOpts]

func WithCreateOptions(value CreateOpts) CreateOption { return senlin.Snapshot(value) }
func WithUpdateOptions(value UpdateOpts) UpdateOption { return senlin.Snapshot(value) }
func WithDeleteOptions(value DeleteOpts) DeleteOption { return senlin.Snapshot(value) }
func WithListOptions(value ListOpts) ListOption       { return senlin.Snapshot(value) }

func withJSON[T any](value any, set func(*T, json.RawMessage)) request.Option[T] {
	encoded, err := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: cluster JSON input: %v", resource.ErrInvalidOption, err)
		}
		set(&config.Options, append(json.RawMessage(nil), encoded...))
		return nil
	}
}

func WithCreateConfig(value any) CreateOption {
	return withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Config = raw })
}
func WithCreateMetadata(value any) CreateOption {
	return withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Metadata = raw })
}
func WithUpdateConfig(value any) UpdateOption {
	return withJSON(value, func(opts *UpdateOpts, raw json.RawMessage) { opts.Config = raw })
}
func WithUpdateMetadata(value any) UpdateOption {
	return withJSON(value, func(opts *UpdateOpts, raw json.RawMessage) { opts.Metadata = raw })
}
func WithUpdateProfileOnly(value bool) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		copy := value
		config.Options.ProfileOnly = &copy
		config.Options.profileOnlyNull = false
		return nil
	}
}

// WithUpdateProfileOnlyNull assigns an explicit null, including when the cached
// field is absent. Sending it requires microversion 1.6 like a bool value.
func WithUpdateProfileOnlyNull() UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.ProfileOnly = nil
		config.Options.profileOnlyNull = true
		return nil
	}
}
func WithDeleteForce(value bool) DeleteOption {
	return func(config *request.Config[DeleteOpts]) error { config.Options.Force = value; return nil }
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(config *request.Config[DeleteOpts]) error {
		copy := value
		config.Options.IgnoreMissing = &copy
		return nil
	}
}
func WithListGlobalProject(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
		return nil
	}
}
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithUpdateHeader(key, value string) UpdateOption {
	return request.WithHeader[UpdateOpts](key, value)
}
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
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
	for key, field := range map[string]string{"name": value.Name, "status": value.Status, "sort": value.Sort} {
		if field != "" {
			query.Set(key, field)
		}
	}
	if value.GlobalProject != nil {
		query.Set("global_project", strconv.FormatBool(*value.GlobalProject))
	}
	for key, values := range config.Query {
		if key == "limit" || key == "marker" || key == "max_items" || key == "paginated" || key == "sort" || key == "global_project" {
			return nil, fmt.Errorf("%w: query %q is a concrete cluster option", resource.ErrInvalidOption, key)
		}
		if _, err := filterField(key); err == nil {
			return nil, fmt.Errorf("%w: query %q is a concrete option or local cluster filter", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, nil
}
