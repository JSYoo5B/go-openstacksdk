package resourceproviders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateInventoryOpts is the body of POST /resource_providers/{id}/inventories.
// Total is always sent, including zero, because the pinned Python call takes it
// as a mandatory keyword; Placement validates the range. The optional fields
// are omitted when nil and keep explicit zero values.
type CreateInventoryOpts struct {
	Total           int      `json:"total"`
	Reserved        *int     `json:"reserved,omitempty"`
	MinUnit         *int     `json:"min_unit,omitempty"`
	MaxUnit         *int     `json:"max_unit,omitempty"`
	StepSize        *int     `json:"step_size,omitempty"`
	AllocationRatio *float64 `json:"allocation_ratio,omitempty"`
}

type CreateInventoryOption = request.Option[CreateInventoryOpts]

func WithCreateInventoryOptions(value CreateInventoryOpts) CreateInventoryOption {
	return request.WithOptions(value)
}

// WithCreateInventoryField adds an extension body field. resource_class and
// the concrete fields are owned by the SDK and rejected.
func WithCreateInventoryField(key string, value any) CreateInventoryOption {
	return request.WithField[CreateInventoryOpts](key, value)
}

func WithCreateInventoryHeader(key, value string) CreateInventoryOption {
	return request.WithHeader[CreateInventoryOpts](key, value)
}

// InventoryRecord is the 201 response of CreateInventory. Metadata.Body keeps
// every original field, so omitted and null values stay distinguishable.
type InventoryRecord struct {
	resource.Metadata
	ResourceProviderGeneration *int     `json:"resource_provider_generation"`
	Total                      *int     `json:"total"`
	Reserved                   *int     `json:"reserved"`
	MinUnit                    *int     `json:"min_unit"`
	MaxUnit                    *int     `json:"max_unit"`
	StepSize                   *int     `json:"step_size"`
	AllocationRatio            *float64 `json:"allocation_ratio"`
}

func (r *InventoryRecord) UnmarshalJSON(data []byte) error {
	type plain InventoryRecord
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*r = InventoryRecord(value)
	return nil
}

func inventorySegment(kind, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s must not be empty", resource.ErrInvalidOption, kind)
	}
	return nil
}

// CreateInventory POSTs {"resource_class": <class>, "total": ..., ...} to
// /resource_providers/{id}/inventories and requires 201. It adds one class to
// the provider; UpdateInventory (PUT of an existing class) and
// UpdateInventories (PUT replacing every class) are different requests.
// Identifiers are escaped as single path segments. The call is never replayed,
// so after a timeout or cancellation the class may or may not exist; read the
// provider inventories before retrying.
func (a *API) CreateInventory(ctx context.Context, resourceProviderID, resourceClass string, opts CreateInventoryOpts, options ...CreateInventoryOption) (*InventoryRecord, error) {
	const op = "CreateInventory"
	wrap := func(err error) error { return request.Wrap(op, "resourceproviders", err) }
	if err := inventorySegment("resource provider ID", resourceProviderID); err != nil {
		return nil, wrap(err)
	}
	if err := inventorySegment("resource class", resourceClass); err != nil {
		return nil, wrap(err)
	}
	config, err := request.Apply(opts, append([]CreateInventoryOption(nil), options...)...)
	if err != nil {
		return nil, wrap(err)
	}
	if err := request.ValidateCapabilities(config, true, false, true); err != nil {
		return nil, wrap(err)
	}
	if err := masakari.Headers(config.Headers); err != nil {
		return nil, wrap(err)
	}
	if _, owned := config.Fields["resource_class"]; owned {
		return nil, wrap(fmt.Errorf("%w: extension field \"resource_class\" is owned by the SDK", resource.ErrInvalidOption))
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, wrap(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, wrap(err)
	}
	body := make(map[string]any, len(fields)+1)
	for key, value := range fields {
		body[key] = value
	}
	if body, err = request.MergeFieldsFor(body, config.Fields, config.Options); err != nil {
		return nil, wrap(err)
	}
	body["resource_class"] = resourceClass
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, wrap(err)
	}
	response, err := rest.DoJSON(ctx, a.client, http.MethodPost, a.client.ServiceURL("resource_providers", url.PathEscape(resourceProviderID), "inventories"), json.RawMessage(payload), config.Headers, http.StatusCreated)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := rest.Decode(response, "", func(r *InventoryRecord) *resource.Metadata { return &r.Metadata })
	return value, wrap(err)
}
