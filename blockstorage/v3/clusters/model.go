package clusters

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
)

// Cluster is the Python Cluster Resource view. Name is the alternate id.
// Typed fields are filled only when the JSON type matches; Body keeps every
// declared Body key as raw JSON (null when unknown). Wire is the actual
// unwrapped response object (nil without one), Envelope the actual response
// body, Header and StatusCode the actual response, and Discovery the actual
// version discovery responses of the call.
type Cluster struct {
	Name              string
	Binary            *string
	State             *string
	Status            *string
	DisabledReason    *string
	Frozen            *bool
	NumHosts          *int64
	NumDownHosts      *int64
	ReplicationStatus *string
	ActiveBackendID   *string
	LastHeartbeat     *string
	CreatedAt         *string
	UpdatedAt         *string

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	Discovery  []cinderowned.Observation
}

// Target is the Python Cluster instance an action runs on. A nil Name sends
// JSON null, as Python does for a cluster given only as a string (that
// string becomes the id, not the name). Binary is sent only when nonempty.
type Target struct {
	Name   *string
	Binary *string
}

// NewTarget returns a Target with the given name and, when nonempty, binary.
func NewTarget(name, binary string) Target {
	target := Target{Name: &name}
	if binary != "" {
		target.Binary = &binary
	}
	return target
}

// Target returns the action target of a cluster read from Cinder.
func (c *Cluster) Target() Target {
	if c == nil {
		return Target{}
	}
	return Target{Name: cinderowned.Decode[string](c.Body["name"]), Binary: cinderowned.Decode[string](c.Body["binary"])}
}

func fromRecord(r *cinderowned.Record) *Cluster {
	if r == nil {
		return nil
	}
	r = r.Clone()
	text := func(key string) *string { return cinderowned.Field[string](r, key) }
	return &Cluster{Name: r.Text("name"), Binary: text("binary"), State: text("state"), Status: text("status"),
		DisabledReason: text("disabled_reason"), Frozen: cinderowned.Field[bool](r, "frozen"),
		NumHosts: cinderowned.Field[int64](r, "num_hosts"), NumDownHosts: cinderowned.Field[int64](r, "num_down_hosts"),
		ReplicationStatus: text("replication_status"), ActiveBackendID: text("active_backend_id"),
		LastHeartbeat: text("last_heartbeat"), CreatedAt: text("created_at"), UpdatedAt: text("updated_at"),
		Body: r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode, Discovery: r.Discovery}
}

func convert(r *cinderowned.Record, err error) (*Cluster, error) {
	return fromRecord(r), err
}
