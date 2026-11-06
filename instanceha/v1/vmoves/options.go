package vmoves

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type ListOpts struct {
	Limit   *int    `json:"limit,omitempty"`
	Marker  *string `json:"marker,omitempty"`
	SortKey *string `json:"sort_key,omitempty"`
	SortDir *string `json:"sort_dir,omitempty"`
	Type    *string `json:"type,omitempty"`
	Status  *string `json:"status,omitempty"`
}

// ListOption is one option of List, All and the marker-fallback lists:
// WithListOptions, request.Query with a string or []string value (it cannot
// replace a concrete filter) or request.Custom over *request.Config[ListOpts].
// Limit and marker are ListOpts members; request.Limit, request.Marker and
// other shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = masakari.ListCalls[ListOpts]()

func WithListOptions(opts ListOpts) ListOption { return request.Custom(masakari.Snapshot(opts)) }

var listFields = []string{"limit", "marker", "sort_key", "sort_dir", "type", "status", "notification_id", "notification_uuid", "notification"}

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
	for key, value := range map[string]*string{"marker": o.Marker, "sort_key": o.SortKey, "sort_dir": o.SortDir, "type": o.Type, "status": o.Status} {
		if value != nil {
			values.Set(key, *value)
		}
	}
	return masakari.Query(config, values, listFields...)
}
