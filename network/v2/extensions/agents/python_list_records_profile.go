package agents

import (
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
)

// ListRecord preserves the declared Source view and independently owned HTTP evidence.
type ListRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func listRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func listRecordProfile() neutronfind.Profile {
	return neutronfind.Profile{
		ServiceRole: "network", Kind: "agents", Path: "agents", Singular: "agent", Plural: "agents", IDKey: "id", NameKey: "name", PreserveUnknownBody: true,
		VersionedPath: "/v2.0/agents", SourceFindPolicy: true, SourceLinkPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
		Body: []neutronfind.Descriptor{{Name: "agent_type", Wire: "agent_type", Missing: "null"},
			{Name: "availability_zone", Wire: "availability_zone", Missing: "null"},
			{Name: "binary", Wire: "binary", Missing: "null"},
			{Name: "configuration", Wire: "configurations", Missing: "null"},
			{Name: "created_at", Wire: "created_at", Missing: "null"},
			{Name: "description", Wire: "description", Missing: "null"},
			{Name: "last_heartbeat_at", Wire: "heartbeat_timestamp", Missing: "null"},
			{Name: "host", Wire: "host", Missing: "null"},
			{Name: "is_admin_state_up", Wire: "admin_state_up", Missing: "null", Kind: neutronfind.Boolean},
			{Name: "is_alive", Wire: "alive", Missing: "null", Kind: neutronfind.Boolean},
			{Name: "resources_synced", Wire: "resources_synced", Missing: "null", Kind: neutronfind.Boolean},
			{Name: "started_at", Wire: "started_at", Missing: "null"},
			{Name: "topic", Wire: "topic", Missing: "null"},
			{Name: "ha_state", Wire: "ha_state", Missing: "null"},
			{Name: "ha_chassis_priority", Wire: "ha_chassis_priority", Missing: "null", Kind: neutronfind.Integer},
			{Name: "id", Wire: "id", Missing: "null"},
			{Name: "name", Wire: "name", Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "agent_type", Wire: "agent_type"}, {Name: "availability_zone", Wire: "availability_zone"}, {Name: "binary", Wire: "binary"}, {Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"}, {Name: "host", Wire: "host"}, {Name: "topic", Wire: "topic"}, {Name: "is_admin_state_up", Wire: "admin_state_up"}, {Name: "is_alive", Wire: "alive"}, {Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}}
}
