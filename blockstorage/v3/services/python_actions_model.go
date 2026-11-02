package services

import (
	"bytes"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"maps"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ServiceTarget is the Python Service instance an action runs on. A nil
// field sends JSON null, as Python does for a service given only as an ID
// string (that string is never sent).
type ServiceTarget struct {
	Binary *string
	Host   *string
}

// NewServiceTarget returns a ServiceTarget with the given binary and host.
func NewServiceTarget(binary, host string) ServiceTarget {
	return ServiceTarget{Binary: &binary, Host: &host}
}

// TargetOf returns the action target of a service listed by List.
func TargetOf(service *Service) ServiceTarget {
	if service == nil {
		return ServiceTarget{}
	}
	return NewServiceTarget(service.Binary, service.Host)
}

func optional(value *string) json.RawMessage {
	if value == nil {
		return json.RawMessage("null")
	}
	raw, _ := json.Marshal(*value)
	return raw
}
func (s ServiceTarget) binary() json.RawMessage { return optional(s.Binary) }
func (s ServiceTarget) host() json.RawMessage   { return optional(s.Host) }
func (s ServiceTarget) seed() map[string]json.RawMessage {
	seed := map[string]json.RawMessage{}
	if s.Binary != nil {
		seed["binary"] = s.binary()
	}
	if s.Host != nil {
		seed["host"] = s.host()
	}
	return seed
}

// ServiceRecord is the Python Service Resource after an action: the target
// attributes overlaid by the declared keys of the actual response body.
// Body keeps every declared Body key as raw JSON (null when unknown; name
// falls back to binary as the Python alias does). Wire is the actual
// response object (nil without one), Envelope the actual response body,
// Header and StatusCode the actual response, and Discovery the actual
// version discovery responses of the call.
type ServiceRecord struct {
	Binary         *string
	Host           *string
	Status         *string
	State          *string
	DisabledReason *string
	Frozen         *bool
	Cluster        *string

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	Discovery  []cinderowned.Observation
}

func serviceFromRecord(r *cinderowned.Record) *ServiceRecord {
	if r == nil {
		return nil
	}
	r = r.Clone()
	if r.View != nil && bytes.Equal(r.View["name"], []byte("null")) {
		r.View["name"] = bytes.Clone(r.View["binary"])
	}
	text := func(key string) *string { return cinderowned.Field[string](r, key) }
	return &ServiceRecord{Binary: text("binary"), Host: text("host"), Status: text("status"), State: text("state"),
		DisabledReason: text("disabled_reason"), Frozen: cinderowned.Field[bool](r, "frozen"), Cluster: text("cluster"),
		Body: r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode, Discovery: r.Discovery}
}

func serviceConvert(r *cinderowned.Record, err error) (*ServiceRecord, error) {
	return serviceFromRecord(r), err
}

// ActionReceipt keeps the actual response of an action Python returns None
// for, and the actual version discovery responses of the call.
type ActionReceipt struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	Discovery  []cinderowned.Observation
}

func receipt(p *cinderowned.Prepared, response *rest.Response, err error) *ActionReceipt {
	if p == nil {
		return nil
	}
	record := p.Evidence(response, err)
	if record == nil {
		return nil
	}
	return &ActionReceipt{Envelope: record.Envelope, Header: record.Header, StatusCode: record.StatusCode, Discovery: record.Discovery}
}

// LogLevel is one Python LogLevel: Binary, Host and Levels decoded when the
// JSON types match, and Body with the three raw members.
type LogLevel struct {
	Binary string
	Host   string
	Levels map[string]string
	Body   map[string]json.RawMessage
}

// LogLevels is the result of GetLogLevels with the actual response.
type LogLevels struct {
	Levels []LogLevel
	ActionReceipt
}

func logLevels(r *ActionReceipt) *LogLevels {
	if r == nil {
		return nil
	}
	return &LogLevels{ActionReceipt: *r}
}

// ActionOpts carries the action arguments and per-request headers.
type ActionOpts struct {
	Reason    string
	Cluster   string
	BackendID string
	Headers   map[string]string
}

func copyAction(v ActionOpts) ActionOpts { v.Headers = maps.Clone(v.Headers); return v }

// ActionOption is one option of Enable, Thaw and Freeze: request.Headers
// (replaces the headers) or request.Custom over *ActionOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type ActionOption = request.WriteCall

// DisableOption is one option of Disable: WithDisableReason, request.Headers
// (replaces the headers) or request.Custom over *ActionOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP. A nonempty reason
// selects disable-log-reason, as in Python.
type DisableOption = request.WriteCall

func WithDisableReason(reason string) DisableOption {
	return request.Custom(func(v *ActionOpts) error { v.Reason = reason; return nil })
}

// FailoverOption is one option of Failover: WithFailoverCluster,
// WithFailoverBackendID, request.Headers (replaces the headers) or
// request.Custom over *ActionOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP. Empty values are not sent.
type FailoverOption = request.WriteCall

func WithFailoverCluster(cluster string) FailoverOption {
	return request.Custom(func(v *ActionOpts) error { v.Cluster = cluster; return nil })
}
func WithFailoverBackendID(backendID string) FailoverOption {
	return request.Custom(func(v *ActionOpts) error { v.BackendID = backendID; return nil })
}

var actionCalls = calls.Spec[ActionOpts]{Headers: calls.Value(func(v *ActionOpts) *map[string]string { return &v.Headers })}

// LogOpts carries the optional log arguments. Binary nil sends "", Server
// and Prefix nil send JSON null.
type LogOpts struct {
	Binary, Server, Prefix *string
	Headers                map[string]string
}

func copyLog(v LogOpts) LogOpts {
	v.Headers = maps.Clone(v.Headers)
	for _, field := range []**string{&v.Binary, &v.Server, &v.Prefix} {
		if *field != nil {
			x := **field
			*field = &x
		}
	}
	return v
}

func (v LogOpts) body() map[string]json.RawMessage {
	binary := json.RawMessage(`""`)
	if v.Binary != nil && *v.Binary != "" {
		binary = optional(v.Binary)
	}
	return map[string]json.RawMessage{"binary": binary, "server": optional(v.Server), "prefix": optional(v.Prefix)}
}

// LogOption is one option of SetLogLevels and GetLogLevels: WithLogBinary,
// WithLogServer, WithLogPrefix, request.Headers (replaces the headers) or
// request.Custom over *LogOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type LogOption = request.GetCall

func WithLogBinary(binary string) LogOption {
	return request.Custom(func(v *LogOpts) error { x := binary; v.Binary = &x; return nil })
}
func WithLogServer(server string) LogOption {
	return request.Custom(func(v *LogOpts) error { x := server; v.Server = &x; return nil })
}
func WithLogPrefix(prefix string) LogOption {
	return request.Custom(func(v *LogOpts) error { x := prefix; v.Prefix = &x; return nil })
}

var logCalls = calls.Spec[LogOpts]{Headers: calls.Value(func(v *LogOpts) *map[string]string { return &v.Headers })}
