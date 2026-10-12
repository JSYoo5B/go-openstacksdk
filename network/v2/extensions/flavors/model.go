package flavors

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Flavor is the Python Flavor Resource view. Typed fields are filled only
// when the JSON type matches. Resource keeps every Source attribute as raw
// JSON under its Python name (null when missing, unknown keys kept); Wire
// is the actual response object, Envelope the actual response body.
type Flavor struct {
	ID                string
	Name              *string
	Description       *string
	IsEnabled         *bool
	ServiceType       *string
	ServiceProfileIDs []string

	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// DeleteResult is the actual DELETE receipt; IgnoredMissing marks a clean
// 404 that the Python default ignore_missing=True turned into success.
type DeleteResult = ownedrecords.DeleteResult

func fromRecord(r *ownedrecords.Record) *Flavor {
	if r == nil {
		return nil
	}
	value := &Flavor{ID: ownedrecords.Text(r, "id"),
		Name:        ownedrecords.Field[string](r, "name"),
		Description: ownedrecords.Field[string](r, "description"),
		IsEnabled:   ownedrecords.Field[bool](r, "is_enabled"),
		ServiceType: ownedrecords.Field[string](r, "service_type"),
		Resource:    r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
	if ids := ownedrecords.Field[[]string](r, "service_profile_ids"); ids != nil {
		value.ServiceProfileIDs = *ids
	}
	return value
}

func convert(r *ownedrecords.Record, err error) (*Flavor, error) {
	return fromRecord(r), err
}
