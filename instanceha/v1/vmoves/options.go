package vmoves

import (
	"fmt"
	"net/url"
	"strconv"

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

type ListOption = request.Option[ListOpts]

func WithListOptions(opts ListOpts) ListOption   { return masakari.Snapshot(opts) }
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

var listFields = []string{"limit", "marker", "sort_key", "sort_dir", "type", "status", "notification_id", "notification_uuid", "notification"}

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
	for key, value := range map[string]*string{"marker": o.Marker, "sort_key": o.SortKey, "sort_dir": o.SortDir, "type": o.Type, "status": o.Status} {
		if value != nil {
			values.Set(key, *value)
		}
	}
	return masakari.Query(config, values, listFields...)
}
