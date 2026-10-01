package segments

import (
	"fmt"
	"net/url"
	"strconv"

	"gophercloudsdk/internal/masakari"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type CreateOpts struct {
	Name           string  `json:"name"`
	RecoveryMethod string  `json:"recovery_method"`
	ServiceType    string  `json:"service_type"`
	Description    *string `json:"description,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

type UpdateOpts struct {
	Name           *string `json:"name,omitempty"`
	RecoveryMethod *string `json:"recovery_method,omitempty"`
	ServiceType    *string `json:"service_type,omitempty"`
	Description    *string `json:"description,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

type ListOpts struct {
	Limit          *int    `json:"limit,omitempty"`
	Marker         *string `json:"marker,omitempty"`
	SortKey        *string `json:"sort_key,omitempty"`
	SortDir        *string `json:"sort_dir,omitempty"`
	RecoveryMethod *string `json:"recovery_method,omitempty"`
	ServiceType    *string `json:"service_type,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

type CreateOption = request.Option[CreateOpts]
type UpdateOption = request.Option[UpdateOpts]
type ListOption = request.Option[ListOpts]

func WithCreateOptions(opts CreateOpts) CreateOption { return masakari.Snapshot(opts) }
func WithUpdateOptions(opts UpdateOpts) UpdateOption { return masakari.Snapshot(opts) }
func WithListOptions(opts ListOpts) ListOption       { return masakari.Snapshot(opts) }
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
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

var listFields = []string{"limit", "marker", "sort_key", "sort_dir", "recovery_method", "service_type", "enabled"}

func prepareList(options ...ListOption) (url.Values, error) {
	config, err := request.Apply(ListOpts{}, options...)
	if err != nil {
		return nil, err
	}
	o, values := config.Options, url.Values{}
	if o.Limit != nil {
		if *o.Limit < 1 {
			return nil, fmt.Errorf("%w: Masakari list limit must be positive", resource.ErrInvalidOption)
		}
		values.Set("limit", strconv.Itoa(*o.Limit))
	}
	for key, value := range map[string]*string{"marker": o.Marker, "sort_key": o.SortKey, "sort_dir": o.SortDir, "recovery_method": o.RecoveryMethod, "service_type": o.ServiceType} {
		if value != nil {
			values.Set(key, *value)
		}
	}
	if o.Enabled != nil {
		values.Set("enabled", strconv.FormatBool(*o.Enabled))
	}
	return masakari.Query(config, values, listFields...)
}
