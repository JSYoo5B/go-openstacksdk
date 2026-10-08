package orders

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/keymanagerread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// CreateRecordOpts supplies declared source attributes, including explicit nulls.
type CreateRecordOpts = keymanagerread.CreateRecordOpts
type CreateRecordOption = keymanagerread.CreateRecordOption

func WithCreateRecordOptions(value CreateRecordOpts) CreateRecordOption {
	return keymanagerread.WithCreateRecordOptions("orders", value)
}
func WithCreateRecordAttributes(value map[string]any) CreateRecordOption {
	return keymanagerread.WithCreateRecordAttributes("orders", value)
}
func WithCreateRecordAttribute(key string, value any) CreateRecordOption {
	return keymanagerread.WithCreateRecordAttribute("orders", key, value)
}
func WithCreateRecordField(key string, value any) CreateRecordOption {
	return keymanagerread.WithCreateRecordField("orders", key, value)
}
func WithCreateRecordHeader(key, value string) CreateRecordOption {
	return keymanagerread.WithCreateRecordHeader(key, value)
}

// CreatedOrder keeps seeded attributes separate from actual POST evidence.
// Convenience IDs and full references are passive values, not request identities.
type CreatedOrder struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	OrderID        *string
	SecretID       *string
}

// CreateRecord is the owned mapping of Python create_order. Native
// Create retains its existing concrete options, aliases and success-code policy.
func (a *API) CreateRecord(ctx context.Context, options ...CreateRecordOption) (*CreatedOrder, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	record, err := keymanagerread.CreateRecord(ctx, client, "orders", options...)
	if record == nil {
		return nil, request.Wrap("CreateRecord", "orders", err)
	}
	value := &CreatedOrder{Resource: record.Resource, Wire: record.Wire, Envelope: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
	if record.Resource != nil {
		value.OrderID = keymanagerread.NullableString(record.Resource.Body["order_id"])
		value.SecretID = keymanagerread.NullableString(record.Resource.Body["secret_id"])
	}
	return value, request.Wrap("CreateRecord", "orders", err)
}
