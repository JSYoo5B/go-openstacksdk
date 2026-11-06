package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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

// CreateOption is one option of Create: WithCreateOptions, WithCreatePayload,
// request.Field (an extension body member that cannot replace a concrete
// one), request.Header or request.Custom over *request.Config[CreateOpts].
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

// ListOption is one option of List, All and the marker-fallback lists:
// WithListOptions, request.Query with a string or []string value (it cannot
// replace a concrete filter) or request.Custom over *request.Config[ListOpts].
// Limit and marker are ListOpts members; request.Limit, request.Marker and
// other shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var (
	createCalls = masakari.WriteCalls[CreateOpts]()
	listCalls   = masakari.ListCalls[ListOpts]()
)

func WithCreateOptions(opts CreateOpts) CreateOption {
	return request.Custom(masakari.Snapshot(opts))
}
func WithListOptions(opts ListOpts) ListOption { return request.Custom(masakari.Snapshot(opts)) }

// WithCreatePayload snapshots an object at option construction. The encoded
// bytes preserve integers without decoding through map[string]float64.
func WithCreatePayload(payload map[string]any) CreateOption {
	encoded, err := json.Marshal(payload)
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: notification payload: %v", resource.ErrInvalidOption, err)
		}
		config.Options.Payload = append(json.RawMessage(nil), encoded...)
		return nil
	})
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
	for key, value := range map[string]*string{"marker": o.Marker, "sort_key": o.SortKey, "sort_dir": o.SortDir, "source_host_uuid": o.SourceHostUUID, "type": o.Type, "status": o.Status, "generated-since": o.GeneratedSince} {
		if value != nil {
			values.Set(key, *value)
		}
	}
	return masakari.Query(config, values, listFields...)
}
