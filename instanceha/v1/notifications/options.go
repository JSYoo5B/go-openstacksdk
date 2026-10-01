package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"gophercloudsdk/internal/masakari"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type CreateOpts struct {
	Type          string          `json:"type"`
	Hostname      string          `json:"hostname"`
	GeneratedTime string          `json:"generated_time"`
	Payload       json.RawMessage `json:"payload"`
}

type ListOpts struct {
	Limit          *int    `json:"limit,omitempty"`
	Marker         *string `json:"marker,omitempty"`
	SortKey        *string `json:"sort_key,omitempty"`
	SortDir        *string `json:"sort_dir,omitempty"`
	SourceHostUUID *string `json:"source_host_uuid,omitempty"`
	Type           *string `json:"type,omitempty"`
	Status         *string `json:"status,omitempty"`
	GeneratedSince *string `json:"generated-since,omitempty"`
}

type CreateOption = request.Option[CreateOpts]
type ListOption = request.Option[ListOpts]

func WithCreateOptions(opts CreateOpts) CreateOption { return masakari.Snapshot(opts) }
func WithListOptions(opts ListOpts) ListOption       { return masakari.Snapshot(opts) }
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

// WithCreatePayload snapshots an object at option construction. The encoded
// bytes preserve integers without decoding through map[string]float64.
func WithCreatePayload(payload map[string]any) CreateOption {
	encoded, err := json.Marshal(payload)
	return func(config *request.Config[CreateOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: notification payload: %v", resource.ErrInvalidOption, err)
		}
		config.Options.Payload = append(json.RawMessage(nil), encoded...)
		return nil
	}
}

func validatePayload(data json.RawMessage) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' || !json.Valid(data) {
		return fmt.Errorf("%w: notification payload must be a JSON object", resource.ErrInvalidOption)
	}
	return nil
}

var listFields = []string{"limit", "marker", "sort_key", "sort_dir", "source_host_uuid", "type", "status", "generated-since", "generated_since"}

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
	for key, value := range map[string]*string{"marker": o.Marker, "sort_key": o.SortKey, "sort_dir": o.SortDir, "source_host_uuid": o.SourceHostUUID, "type": o.Type, "status": o.Status, "generated-since": o.GeneratedSince} {
		if value != nil {
			values.Set(key, *value)
		}
	}
	return masakari.Query(config, values, listFields...)
}
