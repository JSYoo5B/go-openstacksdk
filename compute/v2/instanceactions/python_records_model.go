package instanceactions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServerActionRecord preserves Source descriptors and the original page row.
// ServerID is the explicit URI parent; instance_uuid remains passive Wire data.
type ServerActionRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	ServerID       string
	Discovery      []ActionDiscoveryObservation
	row            json.RawMessage
	fromList       bool
}
type ActionDiscoveryObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

var actionRecordProfile = neutronfind.Profile{
	ServiceRole: "compute", Kind: "server actions", Singular: "instanceAction", Plural: "instanceActions", IDKey: "id", NameKey: "name", AlternateIDKey: "request_id", SourceFindPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, SourceLinkPolicy: true,
	Body: []neutronfind.Descriptor{
		{Name: "action", Wire: "action", Missing: "null"},
		{Name: "request_id", Wire: "request_id", Missing: "null"},
		{Name: "user_id", Wire: "user_id", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Missing: "null"},
		{Name: "start_time", Wire: "start_time", Missing: "null"},
		{Name: "message", Wire: "message", Missing: "null"},
		{Name: "events", Wire: "events", Missing: "null", Project: projectActionEvents, Match: matchActionEvents},
		{Name: "id", Wire: "id", Missing: "null", Alias: "request_id"},
		{Name: "name", Wire: "name", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "changes_since", Wire: "changes-since"}, {Name: "changes_before", Wire: "changes-before"}},
}

// The Source list_type is a Resource subclass of dict. JSON object elements
// construct that nested Resource; every other element becomes a plain {}.
func projectActionEvents(raw json.RawMessage) (json.RawMessage, error) {
	if raw == nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return json.RawMessage("null"), nil
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, findInvalid("events must be complete UTF-8 JSON")
	}
	value := bytes.TrimSpace(raw)
	if len(value) > 0 && value[0] != '[' {
		value = append(append([]byte{'['}, value...), ']')
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(value, &elements); err != nil {
		return nil, err
	}
	for i, item := range elements {
		actual := bytes.TrimSpace(item)
		if len(actual) == 0 || actual[0] != '{' {
			elements[i] = json.RawMessage("{}")
			continue
		}
		members, err := cloudfilter.ObjectMembers(item)
		if err != nil {
			return nil, err
		}
		body := map[string]json.RawMessage{}
		for _, key := range []string{"event", "start_time", "finish_time", "result", "traceback", "host", "host_id", "details", "id", "name", "location"} {
			body[key] = json.RawMessage("null")
		}
		for _, member := range members {
			if member.Key == "self" {
				return nil, findInvalid("event self collides with its Resource constructor")
			}
			if member.Key == "connection" {
				truth, err := cloudfilter.PythonJSONTruthy(member.Value)
				if err != nil {
					return nil, err
				}
				if truth {
					return nil, findInvalid("event connection is not an SDK-owned connection")
				}
				continue
			}
			key := member.Key
			if key == "hostId" {
				key = "host_id"
			}
			if _, known := body[key]; known && key != "location" {
				body[key] = bytes.Clone(member.Value)
			}
		}
		elements[i], err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(elements)
}

type actionPrepared struct {
	source    *cloudread.Source
	guard     func(context.Context) error
	ctx       context.Context
	serverID  string
	discovery []ActionDiscoveryObservation
}

func (a *API) captureActionRecords(ctx context.Context, serverID string) (*actionPrepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("configured server action API is required")
	}
	if strings.TrimSpace(serverID) == "" || !utf8.ValidString(serverID) || serverID == "." || serverID == ".." || strings.IndexFunc(serverID, unicode.IsControl) >= 0 {
		return nil, findInvalid("server ID must be nonblank UTF-8 literal text")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, err
	}
	var changed error
	guard := func(check context.Context) error {
		if changed == nil && a.client != selected {
			changed = findInvalid("selected server action client changed")
		}
		return errors.Join(changed, source.Guard(check), rest.CheckOperationGuard(ctx))
	}
	return &actionPrepared{source: source, guard: guard, ctx: ctx, serverID: serverID}, guard(ctx)
}
func actionVersionHeader(key string) bool {
	return strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version")
}
func (p *actionPrepared) iterate(prior *neutronfind.Engine, policy neutronfind.Options) iter.Seq2[*neutronfind.Record, error] {
	return func(yield func(*neutronfind.Record, error) bool) {
		if err := errors.Join(p.guard(p.ctx), prior.Check(p.ctx)); err != nil {
			yield(nil, err)
			return
		}
		source := *p.source
		source.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
		headers := maps.Clone(policy.Headers)
		if policy.Microversion != nil {
			for key := range source.Client.MoreHeaders {
				if actionVersionHeader(key) {
					delete(source.Client.MoreHeaders, key)
				}
			}
		}
		preflight := maps.Clone(headers)
		if policy.Microversion == nil && source.Client.Microversion == "" {
			for key := range preflight {
				if actionVersionHeader(key) {
					delete(preflight, key)
				}
			}
		}
		if err := source.WithPolicy(p.ctx, policy.Microversion, preflight); err != nil {
			yield(nil, err)
			return
		}
		selected := source.Client.Microversion
		if selected == "" && policy.Microversion == nil {
			found, err := microversions.Read(rest.WithOperationGuard(p.ctx, p.guard), &source, microversions.NovaProfile)
			for _, proof := range found.Responses {
				p.discovery = append(p.discovery, ActionDiscoveryObservation{Envelope: bytes.Clone(proof.Body), Header: proof.Header.Clone(), StatusCode: proof.StatusCode})
			}
			if err = errors.Join(err, p.guard(p.ctx), prior.Check(p.ctx)); err != nil {
				yield(nil, err)
				return
			}
			selected, err = microversions.SelectVersion(found.Maximum, found.Minimum, "2.84")
			if err != nil {
				if len(found.Responses) > 0 {
					err = found.Responses[len(found.Responses)-1].Fail(err)
				}
				yield(nil, err)
				return
			}
		}
		client := p.source.Client
		client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
		client.Microversion = selected
		for key := range client.MoreHeaders {
			if actionVersionHeader(key) {
				delete(client.MoreHeaders, key)
			}
		}
		profile := actionListProfile()
		profile.Path = "servers/" + url.PathEscape(p.serverID) + "/os-instance-actions"
		profile.VersionedPath = "/v2.1/" + profile.Path
		engine, err := neutronfind.New(p.ctx, &client, profile, func(check context.Context) error { return errors.Join(p.guard(check), prior.Check(p.ctx)) })
		if err != nil {
			yield(nil, err)
			return
		}
		policy.Microversion = &selected
		for value, err := range engine.List(p.ctx, policy) {
			if !yield(value, err) {
				return
			}
		}
	}
}

func (p *actionPrepared) discoveryCopy() []ActionDiscoveryObservation {
	result := make([]ActionDiscoveryObservation, len(p.discovery))
	for i, proof := range p.discovery {
		result[i] = ActionDiscoveryObservation{Envelope: bytes.Clone(proof.Envelope), Header: proof.Header.Clone(), StatusCode: proof.StatusCode}
	}
	return result
}

// A JSON object decoded through list_type=ServerActionEvent is a Resource,
// whose Source __eq__ rejects a plain dict. JSON filters cannot instantiate
// Python Resource operands; non-object source elements remain plain {}.
func matchActionEvents(projected, wire, filter json.RawMessage) (bool, bool, error) {
	actual, wanted := bytes.TrimSpace(projected), bytes.TrimSpace(filter)
	if len(actual) == 0 || actual[0] != '[' || len(wanted) == 0 || wanted[0] != '[' {
		return false, false, nil
	}
	var expected []json.RawMessage
	if err := json.Unmarshal(filter, &expected); err != nil {
		return false, true, err
	}
	raw := bytes.TrimSpace(wire)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return false, false, nil
	}
	if raw[0] != '[' {
		raw = append(append([]byte{'['}, raw...), ']')
	}
	var original []json.RawMessage
	if err := json.Unmarshal(raw, &original); err != nil {
		return false, true, err
	}
	if len(original) != len(expected) {
		return false, true, nil
	}
	for _, item := range original {
		trim := bytes.TrimSpace(item)
		if len(trim) > 0 && trim[0] == '{' {
			return false, true, nil
		}
	}
	return false, false, nil
}
