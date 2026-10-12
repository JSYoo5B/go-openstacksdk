package labels

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// MeteringLabel is the Python MeteringLabel Resource view. Typed fields are
// filled only when the JSON type matches. Resource keeps every Source
// attribute as raw JSON under its Python name (project_id falls back to
// tenant_id); Wire is the actual response object, Envelope the actual body.
type MeteringLabel struct {
	ID          string
	Name        *string
	Description *string
	ProjectID   *string
	TenantID    *string
	IsShared    *bool

	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// DeleteResult is the actual DELETE receipt; IgnoredMissing marks a clean
// 404 that the Python default ignore_missing=True turned into success.
type DeleteResult = ownedrecords.DeleteResult

func fromRecord(r *ownedrecords.Record) *MeteringLabel {
	if r == nil {
		return nil
	}
	return &MeteringLabel{ID: ownedrecords.Text(r, "id"),
		Name:        ownedrecords.Field[string](r, "name"),
		Description: ownedrecords.Field[string](r, "description"),
		ProjectID:   ownedrecords.Field[string](r, "project_id"),
		TenantID:    ownedrecords.Field[string](r, "tenant_id"),
		IsShared:    ownedrecords.Field[bool](r, "is_shared"),
		Resource:    r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *ownedrecords.Record, err error) (*MeteringLabel, error) {
	return fromRecord(r), err
}
