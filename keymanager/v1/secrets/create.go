package secrets

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
	return keymanagerread.WithCreateRecordOptions("secrets", value)
}
func WithCreateRecordAttributes(value map[string]any) CreateRecordOption {
	return keymanagerread.WithCreateRecordAttributes("secrets", value)
}
func WithCreateRecordAttribute(key string, value any) CreateRecordOption {
	return keymanagerread.WithCreateRecordAttribute("secrets", key, value)
}
func WithCreateRecordField(key string, value any) CreateRecordOption {
	return keymanagerread.WithCreateRecordField("secrets", key, value)
}
func WithCreateRecordHeader(key, value string) CreateRecordOption {
	return keymanagerread.WithCreateRecordHeader(key, value)
}

// CreatedSecret keeps seeded attributes separate from actual POST evidence.
// Convenience IDs and full references are passive values, not request identities.
type CreatedSecret struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	SecretID       *string
}

// CreateRecord is the owned mapping of Python create_secret. Native
// Create retains its existing concrete options, aliases and success-code policy.
func (a *API) CreateRecord(ctx context.Context, options ...CreateRecordOption) (*CreatedSecret, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	record, err := keymanagerread.CreateRecord(ctx, client, "secrets", options...)
	if record == nil {
		return nil, request.Wrap("CreateRecord", "secrets", err)
	}
	value := &CreatedSecret{Resource: record.Resource, Wire: record.Wire, Envelope: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
	if record.Resource != nil {
		value.SecretID = keymanagerread.NullableString(record.Resource.Body["secret_id"])
	}
	return value, request.Wrap("CreateRecord", "secrets", err)
}
