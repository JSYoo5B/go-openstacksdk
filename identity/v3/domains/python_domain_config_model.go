package domains

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DomainConfigRecord owns the declared configuration view and the literal
// response separately. The domain URI parameter never becomes a Body field.
// Nested configuration views do not share storage with their received JSON.
type DomainConfigRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (value *DomainConfigRecord) Clone() *DomainConfigRecord {
	if value == nil {
		return nil
	}
	out := *value
	out.Resource = value.Resource.Clone()
	out.Wire = value.Wire.Clone()
	out.Envelope = bytes.Clone(value.Envelope)
	out.Header = value.Header.Clone()
	return &out
}

func domainConfigRecord(value *neutronfind.Record) *DomainConfigRecord {
	if value == nil {
		return nil
	}
	return &DomainConfigRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}

// domainConfigChild projects the pinned Resource-subclass descriptor. Source
// constructs the child from a mapping; a nonmapping nonnull value becomes a
// plain empty dict rather than a child populated with descriptor defaults.
func domainConfigChild(raw json.RawMessage, names ...string) (json.RawMessage, error) {
	trim := bytes.TrimSpace(raw)
	if bytes.Equal(trim, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	if len(trim) == 0 || trim[0] != '{' {
		return json.RawMessage("{}"), nil
	}
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage, len(names)+3)
	for _, name := range append(names, "id", "name", "location") {
		fields[name] = json.RawMessage("null")
	}
	// A duplicate self is rejected during constructor argument binding, before
	// even a connection or other constructor member can be evaluated.
	for _, member := range members {
		if member.Key == "self" {
			return nil, recordInvalid("nested configuration self duplicates the Resource constructor argument")
		}
	}
	bodyPresent := false
	for _, member := range members {
		if member.Key == "connection" {
			truthy, cause := cloudfilter.PythonJSONTruthy(member.Value)
			if cause != nil {
				return nil, cause
			}
			if truthy {
				// A JSON value cannot supply Connection methods. Source raises
				// AttributeError inside the child getter; parent to_dict's
				// getattr(..., None) consumes it and exposes the child as null.
				return json.RawMessage("null"), nil
			}
			continue
		}
		if _, declared := fields[member.Key]; declared {
			fields[member.Key] = bytes.Clone(member.Value)
			bodyPresent = bodyPresent || member.Key != "location"
		}
	}
	if bodyPresent {
		// _collect_attrs recomputes leftover attributes when it collects any
		// Body value. With no child Connection, supplied location is dropped.
		fields["location"] = json.RawMessage("null")
	}
	return json.Marshal(fields)
}

// domainConfigProfile is scoped to a single caller-selected Domain route.
// Identity/LDAP are nested Resource descriptors, not arbitrary dictionaries.
var domainConfigProfile = neutronfind.Profile{
	ServiceRole: "identity", Kind: "domain_configs", Singular: "config",
	IDKey: "id", NameKey: "name", SourceFindPolicy: true,
	SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true,
	PythonLoadedEquality: true, SourceCollectionGET: true, SourceSingletonMutation: true,
	Body: []neutronfind.Descriptor{
		{Name: "identity", Wire: "identity", Missing: "null", Project: func(raw json.RawMessage) (json.RawMessage, error) {
			return domainConfigChild(raw, "driver")
		}},
		{Name: "ldap", Wire: "ldap", Missing: "null", Project: func(raw json.RawMessage) (json.RawMessage, error) {
			return domainConfigChild(raw, "user_tree_dn", "url")
		}},
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
	},
}

func (a *API) domainConfigEngine(ctx context.Context, domainID string) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("configured domains API is required")
	}
	if err := domainMutationIdentity(domainID); err != nil {
		return nil, err
	}
	selected := a.client
	profile := domainConfigProfile
	profile.Path = "domains/" + url.PathEscape(domainID) + "/config"
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return recordInvalid("selected domain configuration client changed")
		}
		return nil
	})
}
