package rules

import (
	"context"
	"errors"
	"iter"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// recordListProfile is the pinned SecurityGroupRule MRO, including the existing
// canonical Body mapping and the NetworkResource/TagMixin descriptors. The
// public record is shared with FindRecord; the shared engine owns its transport.
func recordListProfile() neutronfind.Profile {
	profile := neutronfind.Profile{Kind: "rules", Path: "security-group-rules", VersionedPath: "/v2.0/security-group-rules", Singular: "security_group_rule", Plural: "security_group_rules", IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true}
	for _, name := range []string{"created_at", "description", "direction", "ether_type", "port_range_max", "port_range_min", "project_id", "protocol", "remote_group_id", "remote_address_group_id", "remote_ip_prefix", "security_group_id", "tenant_id", "updated_at", "revision_number", "id", "name", "tags"} {
		field := neutronfind.Descriptor{Name: name, Wire: neutronBodyFieldNames[name], Missing: "null"}
		switch name {
		case "port_range_min", "port_range_max", "revision_number":
			field.Kind = neutronfind.Integer
		case "tags":
			field.Kind = neutronfind.List
			field.Missing = "[]"
		case "project_id":
			field.Alias = "tenant_id"
		}
		profile.Body = append(profile.Body, field)
	}
	profile.Headers = []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Kind: neutronfind.List, Missing: "null"}}
	for _, name := range []string{"limit", "marker", "description", "direction", "fields", "id", "protocol", "remote_group_id", "security_group_id", "remote_address_group_id", "port_range_max", "port_range_min", "remote_ip_prefix", "revision_number", "project_id", "tenant_id", "sort_dir", "sort_key", "ether_type", "tags", "any_tags", "not_tags", "not_any_tags"} {
		wire, _ := findQueryWire(name)
		profile.Query = append(profile.Query, neutronfind.QueryPair{Name: name, Wire: wire})
	}
	return profile
}
func recordListWrap(ctx context.Context, err error) error {
	return request.Wrap("ListRecords", "rules", cloudread.ContextError(ctx, err))
}
func recordListValue(value *neutronfind.Record) *SecurityGroupRuleRecord {
	if value == nil {
		return nil
	}
	return &SecurityGroupRuleRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}

// ListRecords lazily lists Source-shaped SecurityGroupRule records. Server
// aliases, declared local Body filters and unknown kwargs belong to the SDK;
// caller options and physical response carriers are independently owned.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*SecurityGroupRuleRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*SecurityGroupRuleRecord, error) bool) {
		fail := func(err error) { yield(nil, recordListWrap(ctx, err)) }
		if err := cloudread.Context(ctx); err != nil {
			fail(err)
			return
		}
		if a == nil {
			fail(findInvalid("configured security rule API required"))
			return
		}
		selected := a.client
		engine, err := neutronfind.New(ctx, selected, recordListProfile(), func(context.Context) error {
			if a.client != selected {
				return findInvalid("selected security rule API client changed")
			}
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		policy := RecordListOpts{}
		for _, apply := range snapshot {
			if err := engine.Check(ctx); err != nil {
				fail(err)
				return
			}
			if apply == nil {
				fail(findInvalid("nil security rule list option"))
				return
			}
			next := copyRecordList(policy)
			if err := errors.Join(apply(&next), recordListCheck(next), engine.Check(ctx)); err != nil {
				fail(err)
				return
			}
			policy = copyRecordList(next)
		}
		if err := engine.Check(ctx); err != nil {
			fail(err)
			return
		}
		for value, err := range engine.List(ctx, recordListOptions(policy)) {
			if !yield(recordListValue(value), recordListWrap(ctx, err)) {
				return
			}
		}
	}
}

// AllRecords collects a complete list; a later error discards the partial
// slice. Successful empty lists are nonnil, and iterator break stops work.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*SecurityGroupRuleRecord, error) {
	result := make([]*SecurityGroupRuleRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
