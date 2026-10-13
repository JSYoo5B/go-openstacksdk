// Package waits supplies SDK-owned concrete Barbican resource waiters.
package waits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/keymanagerread"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// API owns finite Container, Order and Secret routing. SecretStore, ACL, quota
// and arbitrary Python Resource subclasses are not covered here.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

// Record separates accumulated Source attributes from the last actual metadata
// response. Payload retains its own proof. On polling failure Resource is nil.
// A clean metadata 404 returns the original owned Resource with actual 404 proof
// in Envelope/Header/StatusCode/Wire.
type Record struct {
	Kind, RequestID string
	Resource, Wire  *resource.RawResource
	Envelope        json.RawMessage
	Header          http.Header
	StatusCode      int
	Payload         *secrets.FetchedPayload
}

func (r *Record) Clone() *Record {
	if r == nil {
		return nil
	}
	c := *r
	c.Resource = r.Resource.Clone()
	c.Wire = r.Wire.Clone()
	c.Envelope = bytes.Clone(r.Envelope)
	c.Header = r.Header.Clone()
	if r.Payload != nil {
		p := *r.Payload
		p.Body = bytes.Clone(p.Body)
		p.Header = p.Header.Clone()
		if p.Text != nil {
			v := *p.Text
			p.Text = &v
		}
		c.Payload = &p
	}
	return &c
}

// Target cannot be supplied by a developer adapter. Concrete constructors own
// its immutable route and snapshot all attribute and response channels.
type Target struct {
	kind, id string
	seed     *Record
	err      error
}

func literal(kind, id string) Target {
	view, err := keymanagerread.ProjectRecord(&resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{}}}, kind, &id)
	return Target{kind: kind, id: id, seed: &Record{Kind: kind, RequestID: id, Resource: view}, err: err}
}
func Container(id string) Target { return literal("containers", id) }
func Order(id string) Target     { return literal("orders", id) }
func Secret(id string) Target    { return literal("secrets", id) }
func fromRecord(kind, id string, r *Record) Target {
	if r == nil || r.Resource == nil {
		return Target{err: invalid("a successfully fetched resource is required")}
	}
	r = r.Clone()
	r.Kind = kind
	r.RequestID = id
	return Target{kind: kind, id: id, seed: r}
}
func ForContainer(r *containers.FetchedContainer) Target {
	if r == nil {
		return Target{err: invalid("container is required")}
	}
	return fromRecord("containers", r.RequestID, &Record{Resource: r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode})
}
func ForOrder(r *orders.FetchedOrder) Target {
	if r == nil {
		return Target{err: invalid("order is required")}
	}
	return fromRecord("orders", r.RequestID, &Record{Resource: r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode})
}
func ForSecret(r *secrets.FoundSecretRecord) Target {
	if r == nil {
		return Target{err: invalid("secret is required")}
	}
	return fromRecord("secrets", r.RequestID, &Record{Resource: r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode, Payload: r.Payload})
}

// Options distinguishes omission from zero. Nil Interval defaults to two
// seconds; explicit zero follows Source's short pause. Nil Timeout defaults to
// unlimited status wait or 120-second deletion wait. Failures nil means ERROR,
// while an explicit empty slice disables failure-state detection.
type Options struct {
	Interval, Timeout *time.Duration
	Unlimited         bool
	Failures          []string
	Attribute         *string
	Callback          func(int) error
	Headers           map[string]string
	Microversion      *string
}
type Option func(*Options) error

func ownOptions(o Options) Options {
	if o.Interval != nil {
		v := *o.Interval
		o.Interval = &v
	}
	if o.Timeout != nil {
		v := *o.Timeout
		o.Timeout = &v
	}
	if o.Attribute != nil {
		v := *o.Attribute
		o.Attribute = &v
	}
	if o.Microversion != nil {
		v := *o.Microversion
		o.Microversion = &v
	}
	if o.Failures != nil {
		o.Failures = append([]string{}, o.Failures...)
	}
	o.Headers = maps.Clone(o.Headers)
	return o
}
func WithOptions(o Options) Option {
	o = ownOptions(o)
	return func(c *Options) error { *c = ownOptions(o); return nil }
}
func WithInterval(v time.Duration) Option {
	return func(c *Options) error { c.Interval = &v; return nil }
}
func WithTimeout(v time.Duration) Option {
	return func(c *Options) error { c.Timeout = &v; c.Unlimited = false; return nil }
}
func WithUnlimitedWait() Option {
	return func(c *Options) error { c.Unlimited = true; c.Timeout = nil; return nil }
}
func WithFailures(v ...string) Option {
	owned := append([]string{}, v...)
	return func(c *Options) error { c.Failures = append([]string{}, owned...); return nil }
}
func WithAttribute(v string) Option { return func(c *Options) error { c.Attribute = &v; return nil } }
func WithProgressCallback(v func(int) error) Option {
	return func(c *Options) error {
		if v == nil {
			return invalid("callback is required")
		}
		c.Callback = v
		return nil
	}
}
func WithHeader(k, v string) Option {
	return func(c *Options) error {
		if c.Headers == nil {
			c.Headers = map[string]string{}
		}
		c.Headers[k] = v
		return nil
	}
}
func WithMicroversion(v string) Option {
	return func(c *Options) error { c.Microversion = &v; return nil }
}
func invalid(v string) error { return fmt.Errorf("%w: %s", resource.ErrInvalidOption, v) }

// Finite Body aliases merge synchronized seed attributes with each incoming
// response before reusing the existing owned projector.
func knownFields(kind string) map[string]string {
	m := map[string]string{"id": "id", "name": "name", "status": "status", "created_at": "created", "updated_at": "updated"}
	switch kind {
	case "containers":
		for _, k := range []string{"type", "container_ref", "secret_refs", "consumers"} {
			m[k] = k
		}
	case "orders":
		for _, k := range []string{"type", "creator_id", "meta", "order_ref", "secret_ref", "sub_status", "sub_status_message"} {
			m[k] = k
		}
	case "secrets":
		for _, k := range []string{"algorithm", "bit_length", "content_types", "mode", "secret_ref", "secret_type", "payload", "payload_content_type", "payload_content_encoding"} {
			m[k] = k
		}
		m["expires_at"] = "expiration"
	}
	return m
}
func canonicalFields(kind string, body map[string]json.RawMessage) map[string]json.RawMessage {
	fields := knownFields(kind)
	result := map[string]json.RawMessage{}
	for name, wire := range fields {
		if raw, ok := body[name]; ok {
			result[wire] = bytes.Clone(raw)
		}
	}
	// Existing projector policy gives canonical wire keys priority over aliases.
	for _, wire := range fields {
		if raw, ok := body[wire]; ok {
			result[wire] = bytes.Clone(raw)
		}
	}
	return result
}
func validAttribute(kind, attribute string) bool {
	if kind == secretACLKind {
		return secretACLAttribute(attribute)
	}
	if kind == quotaKind {
		return quotaAttribute(attribute)
	}
	if kind == "secret-stores" {
		return secretStoreAttribute(attribute)
	}
	if _, ok := knownFields(kind)[attribute]; ok {
		return true
	}
	switch kind {
	case "containers":
		return attribute == "container_id" || attribute == "location"
	case "orders":
		return attribute == "order_id" || attribute == "secret_id" || attribute == "location"
	case "secrets":
		return attribute == "secret_id" || attribute == "location"
	}
	return false
}
func text(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil, nil
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, invalid("wait attribute must be a string or null")
	}
	return &v, nil
}
