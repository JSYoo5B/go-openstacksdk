package rules

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// MeteringLabelRule is the Python MeteringLabelRule Resource view. Typed
// fields are filled only when the JSON type matches. Resource keeps every
// Source attribute as raw JSON under its Python name (project_id falls back
// to tenant_id); Wire is the actual response object, Envelope the actual body.
type MeteringLabelRule struct {
	ID                  string
	Name                *string
	Direction           *string
	IsExcluded          *bool
	MeteringLabelID     *string
	ProjectID           *string
	TenantID            *string
	RemoteIPPrefix      *string
	SourceIPPrefix      *string
	DestinationIPPrefix *string

	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// DeleteResult is the actual DELETE receipt; IgnoredMissing marks a clean
// 404 that the Python default ignore_missing=True turned into success.
type DeleteResult = ownedrecords.DeleteResult

func fromRecord(r *ownedrecords.Record) *MeteringLabelRule {
	if r == nil {
		return nil
	}
	return &MeteringLabelRule{ID: ownedrecords.Text(r, "id"),
		Name:                ownedrecords.Field[string](r, "name"),
		Direction:           ownedrecords.Field[string](r, "direction"),
		IsExcluded:          ownedrecords.Field[bool](r, "is_excluded"),
		MeteringLabelID:     ownedrecords.Field[string](r, "metering_label_id"),
		ProjectID:           ownedrecords.Field[string](r, "project_id"),
		TenantID:            ownedrecords.Field[string](r, "tenant_id"),
		RemoteIPPrefix:      ownedrecords.Field[string](r, "remote_ip_prefix"),
		SourceIPPrefix:      ownedrecords.Field[string](r, "source_ip_prefix"),
		DestinationIPPrefix: ownedrecords.Field[string](r, "destination_ip_prefix"),
		Resource:            r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *ownedrecords.Record, err error) (*MeteringLabelRule, error) {
	return fromRecord(r), err
}
