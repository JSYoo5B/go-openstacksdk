package neutronfind

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

// SourceState is the pinned Python Resource._body.attributes view of one
// looked-up resource: a wire-keyed map of raw JSON values before any Body
// type cast or missing default. It follows Resource._consume_attrs exactly:
// a member is consumed when its key is a declared wire key or a declared
// attribute name (exact match), and it is then stored under every declared
// wire key whose wire or attribute name equals the member key ignoring case.
// Response objects behave like CPython json.loads (a duplicate key keeps its
// first position and its last value), so a later canonical alias overrides an
// earlier wire member and vice versa. Unknown members are kept only when the
// profile preserves unknown Body members.
//
// Member responses use Resource._translate_response (header-named members
// remain unknown Body members). List rows and constructor seeds use
// Resource._collect_attrs (declared Header names are consumed as headers).
type SourceState struct {
	body    map[string]json.RawMessage
	mapping sourceMapping
}

type sourceWire struct{ wire, attr string }

type sourceMapping struct {
	body, headers []sourceWire
	unknown       bool
}

// Body returns an owned copy of the wire-keyed Python body attributes. It is
// nil when no resource view was produced.
func (s SourceState) Body() map[string]json.RawMessage {
	if s.body == nil {
		return nil
	}
	return findFields(s.body)
}

// Translate applies Resource._translate_response body consumption to one
// already selected response object (resource key and "self" handling are the
// caller's). It returns the wire-keyed attributes the response contributes.
func (s SourceState) Translate(object json.RawMessage) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(object)
	if err != nil {
		return nil, err
	}
	return s.mapping.translate(members), nil
}

func (p Profile) sourceMapping() sourceMapping {
	return sourceMapping{body: sourceWires(p.Body), headers: sourceWires(p.Headers), unknown: p.PreserveUnknownBody}
}

// sourceWires mirrors Resource._get_mapping: the result maps wire to attribute
// name, so a later attribute declared for the same wire replaces the earlier
// attribute while keeping the first position.
func sourceWires(fields []Descriptor) []sourceWire {
	out := make([]sourceWire, 0, len(fields))
	for _, field := range fields {
		replaced := false
		for i := range out {
			if out[i].wire == field.Wire {
				out[i].attr, replaced = field.Name, true
			}
		}
		if !replaced {
			out = append(out, sourceWire{wire: field.Wire, attr: field.Name})
		}
	}
	return out
}

// pythonDict reproduces CPython dict construction from ordered JSON members.
func pythonDict(members []cloudfilter.JSONMember) []cloudfilter.JSONMember {
	out := make([]cloudfilter.JSONMember, 0, len(members))
	index := make(map[string]int, len(members))
	for _, member := range members {
		if at, exists := index[member.Key]; exists {
			out[at].Value = bytes.Clone(member.Value)
			continue
		}
		index[member.Key] = len(out)
		out = append(out, cloudfilter.JSONMember{Key: member.Key, Value: bytes.Clone(member.Value)})
	}
	return out
}

// consume implements Resource._consume_attrs. foldWire selects the
// CaseInsensitiveDict lookup used by Header mappings.
func consume(mapping []sourceWire, members []cloudfilter.JSONMember, foldWire bool, into map[string]json.RawMessage) []cloudfilter.JSONMember {
	rest := make([]cloudfilter.JSONMember, 0, len(members))
	for _, member := range members {
		known := false
		for _, entry := range mapping {
			if member.Key == entry.attr || member.Key == entry.wire || (foldWire && strings.EqualFold(member.Key, entry.wire)) {
				known = true
				break
			}
		}
		if !known {
			rest = append(rest, member)
			continue
		}
		lower := strings.ToLower(member.Key)
		for _, entry := range mapping {
			if lower == strings.ToLower(entry.wire) || lower == strings.ToLower(entry.attr) {
				into[entry.wire] = bytes.Clone(member.Value)
			}
		}
	}
	return rest
}

func (m sourceMapping) translate(members []cloudfilter.JSONMember) map[string]json.RawMessage {
	members = slices.DeleteFunc(pythonDict(members), func(member cloudfilter.JSONMember) bool { return member.Key == "self" })
	out := map[string]json.RawMessage{}
	rest := consume(m.body, members, false, out)
	if m.unknown {
		for _, member := range rest {
			out[member.Key] = bytes.Clone(member.Value)
		}
	}
	return out
}

func (m sourceMapping) construct(members []cloudfilter.JSONMember) map[string]json.RawMessage {
	members = pythonDict(members)
	out := map[string]json.RawMessage{}
	rest := consume(m.body, members, false, out)
	rest = consume(m.headers, rest, true, map[string]json.RawMessage{})
	if m.unknown {
		for _, member := range rest {
			out[member.Key] = bytes.Clone(member.Value)
		}
	}
	return out
}

// sourceSeed converts the engine's constructor seed back to wire keys. Seeds
// are keyed by declared attribute names for SourceFindPolicy profiles and by
// raw kwargs otherwise; declared Header names never reach the Body.
func (m sourceMapping) seed(seed map[string]json.RawMessage) map[string]json.RawMessage {
	members := make([]cloudfilter.JSONMember, 0, len(seed))
	for _, key := range slices.Sorted(maps.Keys(seed)) {
		members = append(members, cloudfilter.JSONMember{Key: key, Value: seed[key]})
	}
	return m.construct(members)
}

// sourceState records the Python body attributes for one projected record.
func (e *Engine) sourceState(value *Record, seed map[string]json.RawMessage) {
	mapping := e.profile.sourceMapping()
	state := SourceState{mapping: mapping}
	if value.fromList {
		members, err := cloudfilter.ObjectMembers(value.row)
		if err == nil {
			members = slices.DeleteFunc(members, func(member cloudfilter.JSONMember) bool { return member.Key == "self" })
			state.body = mapping.construct(members)
		}
		value.source = state
		return
	}
	state.body = mapping.seed(seed)
	if value.Wire != nil {
		if members, err := cloudfilter.ObjectMembers(value.row); err == nil {
			maps.Copy(state.body, mapping.translate(members))
		}
	}
	value.source = state
}

// Source returns the Python body-attribute state captured with this record.
func (v *Record) Source() SourceState {
	if v == nil {
		return SourceState{}
	}
	if v.source.body == nil {
		return SourceState{mapping: v.source.mapping}
	}
	return SourceState{body: findFields(v.source.body), mapping: v.source.mapping}
}
