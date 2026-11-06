package hosts

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
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

// CreateOption is one option of Create: WithCreateOptions, request.Field (an
// extension body member that cannot replace a concrete one), request.Header
// or request.Custom over *request.Config[CreateOpts]. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

// UpdateOption is one option of Update: WithUpdateOptions, request.Field,
// request.Header or request.Custom over *request.Config[UpdateOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

// ListOption is one option of List, All and the marker-fallback lists:
// WithListOptions, request.Query with a string or []string value (it cannot
// replace a concrete filter) or request.Custom over *request.Config[ListOpts].
// Limit and marker are ListOpts members; request.Limit, request.Marker and
// other shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var (
	createCalls = masakari.WriteCalls[CreateOpts]()
	updateCalls = masakari.WriteCalls[UpdateOpts]()
	listCalls   = masakari.ListCalls[ListOpts]()
)

func WithCreateOptions(opts CreateOpts) CreateOption {
	return request.Custom(masakari.Snapshot(opts))
}
func WithUpdateOptions(opts UpdateOpts) UpdateOption {
	return request.Custom(masakari.Snapshot(opts))
}
func WithListOptions(opts ListOpts) ListOption { return request.Custom(masakari.Snapshot(opts)) }

var listFields = []string{"limit", "marker", "sort_key", "sort_dir", "failover_segment_id", "type", "on_maintenance", "reserved", "segment_id", "segment"}

func prepareList(options ...ListOption) (url.Values, error) {
	config, err := calls.ApplyConfig(ListOpts{}, listCalls, options...)
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
