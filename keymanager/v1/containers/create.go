package containers

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
	return keymanagerread.WithCreateRecordOptions("containers", value)
}
func WithCreateRecordAttributes(value map[string]any) CreateRecordOption {
	return keymanagerread.WithCreateRecordAttributes("containers", value)
}
func WithCreateRecordAttribute(key string, value any) CreateRecordOption {
	return keymanagerread.WithCreateRecordAttribute("containers", key, value)
}
func WithCreateRecordField(key string, value any) CreateRecordOption {
	return keymanagerread.WithCreateRecordField("containers", key, value)
}
func WithCreateRecordHeader(key, value string) CreateRecordOption {
	return keymanagerread.WithCreateRecordHeader(key, value)
}

// CreatedContainer keeps seeded attributes separate from actual POST evidence.
// Convenience IDs and full references are passive values, not request identities.
type CreatedContainer struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	ContainerID    *string
}

// CreateRecord is the owned mapping of Python create_container. Native
// Create retains its existing concrete options, aliases and success-code policy.
func (a *API) CreateRecord(ctx context.Context, options ...CreateRecordOption) (*CreatedContainer, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	record, err := keymanagerread.CreateRecord(ctx, client, "containers", options...)
	if record == nil {
		return nil, request.Wrap("CreateRecord", "containers", err)
	}
	value := &CreatedContainer{Resource: record.Resource, Wire: record.Wire, Envelope: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
	if record.Resource != nil {
		value.ContainerID = keymanagerread.NullableString(record.Resource.Body["container_id"])
	}
	return value, request.Wrap("CreateRecord", "containers", err)
}
