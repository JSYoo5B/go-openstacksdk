package hosts

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type CreateOpts struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	ControlAttributes string `json:"control_attributes"`
	OnMaintenance     *bool  `json:"on_maintenance,omitempty"`
	Reserved          *bool  `json:"reserved,omitempty"`
}

// Control attributes may be submitted through an explicit extension until the
// controller's update schema is verified; Create uses its documented string.
type UpdateOpts struct {
	Name          *string `json:"name,omitempty"`
	Type          *string `json:"type,omitempty"`
	OnMaintenance *bool   `json:"on_maintenance,omitempty"`
	Reserved      *bool   `json:"reserved,omitempty"`
}

type ListOpts struct {
	Limit             *int    `json:"limit,omitempty"`
	Marker            *string `json:"marker,omitempty"`
	SortKey           *string `json:"sort_key,omitempty"`
	SortDir           *string `json:"sort_dir,omitempty"`
	FailoverSegmentID *string `json:"failover_segment_id,omitempty"`
	Type              *string `json:"type,omitempty"`
	OnMaintenance     *bool   `json:"on_maintenance,omitempty"`
	Reserved          *bool   `json:"reserved,omitempty"`
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

var listFields = []string{"limit", "marker", "sort_key", "sort_dir", "failover_segment_id", "type", "on_maintenance", "reserved", "segment_id", "segment"}

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
	for key, value := range map[string]*string{"marker": o.Marker, "sort_key": o.SortKey, "sort_dir": o.SortDir, "failover_segment_id": o.FailoverSegmentID, "type": o.Type} {
		if value != nil {
			values.Set(key, *value)
		}
	}
	for key, value := range map[string]*bool{"on_maintenance": o.OnMaintenance, "reserved": o.Reserved} {
		if value != nil {
			values.Set(key, strconv.FormatBool(*value))
		}
	}
	return masakari.Query(config, values, listFields...)
}
