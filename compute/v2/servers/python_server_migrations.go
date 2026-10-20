package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ServerMigrationOpts owns explicit Go transport policy. IgnoreMissing defaults
// true: abort suppresses only a clean 404. Source GET discards this constructor
// attribute and remains strict. LIST and force-complete have no such argument.
type ServerMigrationOpts struct {
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
}
type ServerMigrationOption func(*ServerMigrationOpts) error

func copyServerMigrationOpts(v ServerMigrationOpts) ServerMigrationOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithServerMigrationOpts(v ServerMigrationOpts) ServerMigrationOption {
	owned := copyServerMigrationOpts(v)
	return func(c *ServerMigrationOpts) error { *c = copyServerMigrationOpts(owned); return nil }
}
func WithServerMigrationHeaders(v map[string]string) ServerMigrationOption {
	owned := maps.Clone(v)
	return func(c *ServerMigrationOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithServerMigrationMicroversion(v string) ServerMigrationOption {
	return func(c *ServerMigrationOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithServerMigrationIgnoreMissing(v bool) ServerMigrationOption {
	return func(c *ServerMigrationOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

// ServerMigrationRecord separates the declared Source view from actual wire
// fields and physical response evidence. Discovery holds only observed version
// responses; its absence never invents an HTTP exchange.
type ServerMigrationRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      microversions.Discovery
}

func cloneServerMigrationDiscovery(value microversions.Discovery) microversions.Discovery {
	value.Responses = slices.Clone(value.Responses)
	for index, response := range value.Responses {
		if response == nil {
			continue
		}
		owned := *response
		owned.Body = bytes.Clone(response.Body)
		owned.Header = response.Header.Clone()
		value.Responses[index] = &owned
	}
	return value
}

func serverMigrationRecord(value *neutronfind.Record, discovery microversions.Discovery) *ServerMigrationRecord {
	if value == nil {
		return nil
	}
	return &ServerMigrationRecord{Resource: value.Resource, Wire: value.Wire, Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode, Discovery: cloneServerMigrationDiscovery(discovery)}
}
func serverMigrationReceipt(err error, discovery microversions.Discovery) *ServerMigrationRecord {
	receipt := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		receipt = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(receipt, &native) {
		return &ServerMigrationRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual, Discovery: cloneServerMigrationDiscovery(discovery)}
	}
	var physical *resource.ResponseError
	if errors.As(err, &physical) {
		return &ServerMigrationRecord{Envelope: bytes.Clone(physical.Body), Header: physical.Header.Clone(), StatusCode: physical.StatusCode, Discovery: cloneServerMigrationDiscovery(discovery)}
	}
	if len(discovery.Responses) != 0 {
		return &ServerMigrationRecord{Discovery: cloneServerMigrationDiscovery(discovery)}
	}
	return nil
}
func serverMigrationProfile(serverID string) neutronfind.Profile {
	profile := neutronfind.Profile{ServiceRole: "compute", Kind: "server_migrations", Path: "servers/" + url.PathEscape(serverID) + "/migrations", Singular: "migration", Plural: "migrations", IDKey: "id", NameKey: "name", AlternateIDKey: "uuid", SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true}
	for _, name := range []string{"created_at", "dest_host", "dest_compute", "dest_node", "disk_processed_bytes", "disk_remaining_bytes", "disk_total_bytes", "memory_processed_bytes", "memory_remaining_bytes", "memory_total_bytes", "project_id", "server_uuid", "source_compute", "source_node", "status", "updated_at", "user_id", "uuid", "id", "name"} {
		field := neutronfind.Descriptor{Name: name, Wire: name, Missing: "null"}
		if name == "id" {
			field.Alias = "uuid"
		}
		profile.Body = append(profile.Body, field)
	}
	profile.Query = []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}
	return profile
}
func serverMigrationID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return userActionInvalid("migration ID must be nonblank literal text without controls or dot segments")
	}
	return nil
}

type preparedServerMigration struct {
	action    *pythonUserAction
	engine    *neutronfind.Engine
	policy    ServerMigrationOpts
	version   string
	discovery microversions.Discovery
}

func (a *API) prepareServerMigration(ctx context.Context, serverID string, options []ServerMigrationOption) (*preparedServerMigration, error) {
	action, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return nil, err
	}
	engine, err := neutronfind.New(ctx, a.client, serverMigrationProfile(serverID), action.guard)
	if err != nil {
		return nil, err
	}
	p := &preparedServerMigration{action: action, engine: engine}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return p, err
		}
		if apply == nil {
			return p, userActionInvalid("nil server migration option")
		}
		next := copyServerMigrationOpts(p.policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return p, err
		}
		p.policy = copyServerMigrationOpts(next)
	}
	if err := action.source.WithPolicy(ctx, p.policy.Microversion, p.policy.Headers); err != nil {
		return p, err
	}
	p.version = action.source.Client.Microversion
	if p.version == "" {
		profile := microversions.NovaProfile
		profile.StopFaultRetryCallbacks = true
		p.discovery, err = microversions.Read(ctx, action.source, profile)
		if err == nil {
			p.version, err = microversions.SelectVersion(p.discovery.Maximum, p.discovery.Minimum, "2.80")
		}
	}
	return p, errors.Join(err, engine.Check(ctx))
}
func (p *preparedServerMigration) transport() neutronfind.Options {
	return neutronfind.Options{Headers: p.policy.Headers, Microversion: &p.version}
}
func (p *preparedServerMigration) record(value *neutronfind.Record, err error) (*ServerMigrationRecord, error) {
	row := serverMigrationRecord(value, p.discovery)
	if row == nil {
		row = serverMigrationReceipt(err, p.discovery)
	}
	err = errors.Join(err, p.engine.Check(p.action.ctx))
	if err != nil && row != nil {
		row.Resource = nil
	}
	return row, err
}

// ListServerMigrationRecords lazily walks the pinned Source server-scoped list.
// URI server_id is never exposed as a Body field; unknown extensions stay Wire.
func (a *API) ListServerMigrationRecords(ctx context.Context, serverID string, options ...ServerMigrationOption) iter.Seq2[*ServerMigrationRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ServerMigrationRecord, error) bool) {
		p, err := a.prepareServerMigration(ctx, serverID, owned)
		if err != nil {
			var row *ServerMigrationRecord
			if p != nil {
				row = serverMigrationReceipt(err, p.discovery)
			}
			yield(row, userActionWrap(ctx, "ListServerMigrationRecords", err))
			return
		}
		for value, err := range p.engine.List(ctx, p.transport()) {
			row, err := p.record(value, err)
			if !yield(row, userActionWrap(ctx, "ListServerMigrationRecords", err)) {
				return
			}
		}
	}
}
func (a *API) AllServerMigrationRecords(ctx context.Context, serverID string, options ...ServerMigrationOption) ([]*ServerMigrationRecord, error) {
	rows := make([]*ServerMigrationRecord, 0)
	for row, err := range a.ListServerMigrationRecords(ctx, serverID, options...) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// GetServerMigrationRecord makes one member GET. Source ignore_missing is
// discarded by this class constructor; it neither travels as a query nor
// suppresses a 404 or triggers a list fallback.
func (a *API) GetServerMigrationRecord(ctx context.Context, serverID, migrationID string, options ...ServerMigrationOption) (*ServerMigrationRecord, error) {
	const op = "GetServerMigrationRecord"
	if err := serverMigrationID(migrationID); err != nil {
		return nil, userActionWrap(ctx, op, err)
	}
	p, err := a.prepareServerMigration(ctx, serverID, options)
	if err != nil {
		var row *ServerMigrationRecord
		if p != nil {
			row = serverMigrationReceipt(err, p.discovery)
		}
		return row, userActionWrap(ctx, op, err)
	}
	value, err := p.engine.Get(ctx, migrationID, p.transport())
	row, err := p.record(value, err)
	return row, userActionWrap(ctx, op, err)
}

// ServerMigrationActionResult preserves actual action receipts. Completed means
// an accepted physical response finished; false does not prove nonexecution.
type ServerMigrationActionResult struct {
	Completed      bool
	IgnoredMissing bool
	Response       *ServerMigrationRecord
}

// AbortServerMigration DELETEs the selected member and ignores only clean 404
// by default. Transport, read, Close and context faults remain terminal errors.
func (a *API) AbortServerMigration(ctx context.Context, serverID, migrationID string, options ...ServerMigrationOption) (*ServerMigrationActionResult, error) {
	const op = "AbortServerMigration"
	if err := serverMigrationID(migrationID); err != nil {
		return nil, userActionWrap(ctx, op, err)
	}
	p, err := a.prepareServerMigration(ctx, serverID, options)
	if err != nil {
		if p != nil {
			if receipt := serverMigrationReceipt(err, p.discovery); receipt != nil {
				return &ServerMigrationActionResult{Response: receipt}, userActionWrap(ctx, op, err)
			}
		}
		return nil, userActionWrap(ctx, op, err)
	}
	value, err := p.engine.Mutate(ctx, http.MethodDelete, migrationID, nil, nil, p.transport())
	row, err := p.record(value, err)
	result := &ServerMigrationActionResult{Completed: err == nil, Response: row}
	if p.policy.IgnoreMissing == nil || *p.policy.IgnoreMissing {
		ignored := resource.IgnoreMissing(err)
		result.IgnoredMissing = err != nil && ignored == nil
		err = ignored
	}
	return result, userActionWrap(ctx, op, err)
}

// ForceCompleteServerMigration sends the Source force_complete:null action and
// preserves its receipt without decoding the response or applying feature gates.
func (a *API) ForceCompleteServerMigration(ctx context.Context, serverID, migrationID string, options ...ServerMigrationOption) (*ServerMigrationActionResult, error) {
	const op = "ForceCompleteServerMigration"
	if err := serverMigrationID(migrationID); err != nil {
		return nil, userActionWrap(ctx, op, err)
	}
	p, err := a.prepareServerMigration(ctx, serverID, options)
	if err != nil {
		if p != nil {
			if receipt := serverMigrationReceipt(err, p.discovery); receipt != nil {
				return &ServerMigrationActionResult{Response: receipt}, userActionWrap(ctx, op, err)
			}
		}
		return nil, userActionWrap(ctx, op, err)
	}
	if err := p.action.source.WithPolicy(ctx, &p.version, p.policy.Headers); err != nil {
		return nil, userActionWrap(ctx, op, err)
	}
	accepted := make([]int, 300)
	for i := range accepted {
		accepted[i] = 100 + i
	}
	rejected := make([]int, 200)
	for i := range rejected {
		rejected[i] = 400 + i
	}
	response, err := rest.DoJSONGuardedRejectionsHeaders(ctx, &p.action.source.Client, p.action.guard, http.MethodPost, p.action.source.Client.ServiceURL("servers", url.PathEscape(serverID), "migrations", url.PathEscape(migrationID), "action"), map[string]any{"force_complete": nil}, nil, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
	var row *ServerMigrationRecord
	if response != nil {
		row = &ServerMigrationRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, Discovery: cloneServerMigrationDiscovery(p.discovery)}
	} else {
		row = serverMigrationReceipt(err, p.discovery)
	}
	err = errors.Join(err, p.engine.Check(ctx))
	return &ServerMigrationActionResult{Completed: err == nil, Response: row}, userActionWrap(ctx, op, err)
}
