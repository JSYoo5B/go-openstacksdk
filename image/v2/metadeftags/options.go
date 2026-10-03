package metadeftags

import (
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CreateOpts configures Create.
type CreateOpts struct {
	Headers map[string]string
}
type CreateOption func(*CreateOpts) error

func copyCreate(value CreateOpts) CreateOpts {
	value.Headers = copyHeaders(value.Headers)
	return value
}

// WithCreateOpts snapshots and replaces the complete configuration.
func WithCreateOpts(value CreateOpts) CreateOption {
	owned := copyCreate(value)
	return func(config *CreateOpts) error { *config = copyCreate(owned); return nil }
}
func WithCreateHeader(key, value string) CreateOption {
	return func(config *CreateOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithCreateHeaders(value map[string]string) CreateOption {
	owned := copyHeaders(value)
	return func(config *CreateOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}

// GetOpts configures Get.
type GetOpts struct {
	Headers map[string]string
}
type GetOption func(*GetOpts) error

func copyGet(value GetOpts) GetOpts {
	value.Headers = copyHeaders(value.Headers)
	return value
}

// WithGetOpts snapshots and replaces the complete configuration.
func WithGetOpts(value GetOpts) GetOption {
	owned := copyGet(value)
	return func(config *GetOpts) error { *config = copyGet(owned); return nil }
}
func WithGetHeader(key, value string) GetOption {
	return func(config *GetOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithGetHeaders(value map[string]string) GetOption {
	owned := copyHeaders(value)
	return func(config *GetOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}

// UpdateOpts configures Update.
type UpdateOpts struct {
	Headers map[string]string
	Name    *string
}
type UpdateOption func(*UpdateOpts) error

func copyUpdate(value UpdateOpts) UpdateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Name = copyPointer(value.Name)
	return value
}

// WithUpdateOpts snapshots and replaces the complete configuration.
func WithUpdateOpts(value UpdateOpts) UpdateOption {
	owned := copyUpdate(value)
	return func(config *UpdateOpts) error { *config = copyUpdate(owned); return nil }
}
func WithUpdateHeader(key, value string) UpdateOption {
	return func(config *UpdateOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithUpdateHeaders(value map[string]string) UpdateOption {
	owned := copyHeaders(value)
	return func(config *UpdateOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithUpdateName(value string) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.Name = &owned; return nil }
}

// DeleteOpts configures Delete.
type DeleteOpts struct {
	Headers map[string]string
}
type DeleteOption func(*DeleteOpts) error

func copyDelete(value DeleteOpts) DeleteOpts {
	value.Headers = copyHeaders(value.Headers)
	return value
}

// WithDeleteOpts snapshots and replaces the complete configuration.
func WithDeleteOpts(value DeleteOpts) DeleteOption {
	owned := copyDelete(value)
	return func(config *DeleteOpts) error { *config = copyDelete(owned); return nil }
}
func WithDeleteHeader(key, value string) DeleteOption {
	return func(config *DeleteOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithDeleteHeaders(value map[string]string) DeleteOption {
	owned := copyHeaders(value)
	return func(config *DeleteOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}

// DeleteAllOpts configures DeleteAll.
type DeleteAllOpts struct {
	Headers map[string]string
}
type DeleteAllOption func(*DeleteAllOpts) error

func copyDeleteAll(value DeleteAllOpts) DeleteAllOpts {
	value.Headers = copyHeaders(value.Headers)
	return value
}

// WithDeleteAllOpts snapshots and replaces the complete configuration.
func WithDeleteAllOpts(value DeleteAllOpts) DeleteAllOption {
	owned := copyDeleteAll(value)
	return func(config *DeleteAllOpts) error { *config = copyDeleteAll(owned); return nil }
}
func WithDeleteAllHeader(key, value string) DeleteAllOption {
	return func(config *DeleteAllOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithDeleteAllHeaders(value map[string]string) DeleteAllOption {
	owned := copyHeaders(value)
	return func(config *DeleteAllOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}

// SetOpts configures Set.
type SetOpts struct {
	Headers map[string]string
	Append  *bool
}
type SetOption func(*SetOpts) error

func copySet(value SetOpts) SetOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Append = copyPointer(value.Append)
	return value
}

// WithSetOpts snapshots and replaces the complete configuration.
func WithSetOpts(value SetOpts) SetOption {
	owned := copySet(value)
	return func(config *SetOpts) error { *config = copySet(owned); return nil }
}
func WithSetHeader(key, value string) SetOption {
	return func(config *SetOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithSetHeaders(value map[string]string) SetOption {
	owned := copyHeaders(value)
	return func(config *SetOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithSetAppend(value bool) SetOption {
	return func(config *SetOpts) error { owned := value; config.Append = &owned; return nil }
}

// ListOpts configures List.
type ListOpts struct {
	Headers  map[string]string
	Limit    *int
	Marker   *string
	SortKey  *string
	SortDir  *string
	MaxItems int
}
type ListOption func(*ListOpts) error

func copyList(value ListOpts) ListOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Limit = copyPointer(value.Limit)
	value.Marker = copyPointer(value.Marker)
	value.SortKey = copyPointer(value.SortKey)
	value.SortDir = copyPointer(value.SortDir)
	return value
}

// WithListOpts snapshots and replaces the complete configuration.
func WithListOpts(value ListOpts) ListOption {
	owned := copyList(value)
	return func(config *ListOpts) error { *config = copyList(owned); return nil }
}
func WithListHeader(key, value string) ListOption {
	return func(config *ListOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithListHeaders(value map[string]string) ListOption {
	owned := copyHeaders(value)
	return func(config *ListOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithListLimit(value int) ListOption {
	return func(config *ListOpts) error { owned := value; config.Limit = &owned; return nil }
}
func WithListMarker(value string) ListOption {
	return func(config *ListOpts) error { owned := value; config.Marker = &owned; return nil }
}
func WithListSortKey(value string) ListOption {
	return func(config *ListOpts) error { owned := value; config.SortKey = &owned; return nil }
}
func WithListSortDir(value string) ListOption {
	return func(config *ListOpts) error { owned := value; config.SortDir = &owned; return nil }
}
func WithListMaxItems(value int) ListOption {
	return func(config *ListOpts) error { config.MaxItems = value; return nil }
}

func copyPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}
func copyHeaders(value map[string]string) map[string]string {
	owned := maps.Clone(value)
	if owned == nil {
		owned = make(map[string]string)
	}
	return owned
}
func setHeader(headers map[string]string, key, value string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	for old := range headers {
		if strings.EqualFold(old, key) {
			delete(headers, old)
		}
	}
	headers[http.CanonicalHeaderKey(key)] = value
	return headers
}
func apply[T any, O ~func(*T) error](options []O, copyValue func(T) T) (T, error) {
	var zero T
	config := copyValue(zero)
	for _, option := range options {
		if option == nil {
			return config, invalid("nil tag option")
		}
		working := copyValue(config)
		if err := option(&working); err != nil {
			return config, err
		}
		config = copyValue(working)
	}
	return config, nil
}
func prepareCreate(options []CreateOption) (CreateOpts, error) {
	value, err := apply(options, copyCreate)
	if err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareGet(options []GetOption) (GetOpts, error) {
	value, err := apply(options, copyGet)
	if err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareUpdate(options []UpdateOption) (UpdateOpts, error) {
	value, err := apply(options, copyUpdate)
	if err != nil {
		return value, err
	}
	if value.Name != nil {
		if err = literal(*value.Name); err != nil {
			return value, err
		}
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareDelete(options []DeleteOption) (DeleteOpts, error) {
	value, err := apply(options, copyDelete)
	if err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareDeleteAll(options []DeleteAllOption) (DeleteAllOpts, error) {
	value, err := apply(options, copyDeleteAll)
	if err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareSet(options []SetOption) (SetOpts, error) {
	value, err := apply(options, copySet)
	if err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}

func prepareList(options []ListOption) (ListOpts, url.Values, error) {
	value, err := apply(options, copyList)
	if err != nil {
		return value, nil, err
	}
	if value.MaxItems < 0 || value.Limit != nil && *value.Limit < 0 {
		return value, nil, invalid("list limits must be nonnegative")
	}
	for _, text := range []*string{value.Marker, value.SortKey} {
		if text != nil {
			if err = queryText(*text); err != nil {
				return value, nil, err
			}
		}
	}
	if value.SortDir != nil && *value.SortDir != "asc" && *value.SortDir != "desc" {
		return value, nil, invalid("sort direction must be asc or desc")
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	if err != nil {
		return value, nil, err
	}
	query := make(url.Values)
	if value.Limit != nil {
		query.Set("limit", strconv.Itoa(*value.Limit))
	}
	for _, entry := range []struct {
		key   string
		value *string
	}{{"marker", value.Marker}, {"sort_key", value.SortKey}, {"sort_dir", value.SortDir}} {
		if entry.value != nil {
			query.Set(entry.key, *entry.value)
		}
	}
	return value, query, nil
}
