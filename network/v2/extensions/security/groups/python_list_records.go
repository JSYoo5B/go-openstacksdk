package groups

import (
	"context"
	"errors"
	"iter"
	"slices"
	"sort"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// The Source MRO Body inventory is shared with the existing owned Find record.
// The private binding selects descriptor/default policy without user builders.
func securityGroupListProfile() neutronfind.Profile {
	profile := neutronfind.Profile{Kind: "groups", Path: "security-groups", Singular: "security_group", Plural: "security_groups", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/security-groups", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true}
	keys := make([]string, 0, len(neutronBodyFieldNames))
	for key := range neutronBodyFieldNames {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		field := neutronfind.Descriptor{Name: key, Wire: neutronBodyFieldNames[key], Missing: "null"}
		switch key {
		case "revision_number":
			field.Kind = neutronfind.Integer
		case "tags":
			field.Kind, field.Missing = neutronfind.List, "[]"
		case "security_group_rules":
			field.Kind = neutronfind.List
		case "is_shared":
			field.Kind = neutronfind.Boolean
		case "project_id":
			field.Alias = "tenant_id"
		}
		profile.Body = append(profile.Body, field)
	}
	profile.Headers = []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Kind: neutronfind.List, Missing: "null"}}
	for _, key := range []string{"limit", "marker", "description", "fields", "id", "name", "stateful", "project_id", "tenant_id", "revision_number", "sort_dir", "sort_key", "tags"} {
		profile.Query = append(profile.Query, neutronfind.QueryPair{Name: key, Wire: key})
	}
	for _, key := range []string{"is_shared", "any_tags", "not_tags", "not_any_tags"} {
		profile.Query = append(profile.Query, neutronfind.QueryPair{Name: key, Wire: findQueryAliases[key]})
	}
	return profile
}
func (a *API) securityGroupListEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("configured Security Groups API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, securityGroupListProfile(), func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected Security Groups API client changed")
		}
		return nil
	})
}
func securityGroupListRecord(value *neutronfind.Record) *SecurityGroupRecord {
	if value == nil {
		return nil
	}
	return &SecurityGroupRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}

// ListRecords is the lazy Source-shaped security_groups list. It has no
// implicit name hint/member fetch; server/query and local Body filters are
// classified by the exact pinned SecurityGroup declaration and inherited MRO.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*SecurityGroupRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*SecurityGroupRecord, error) bool) {
		wrap := func(err error) error { return request.Wrap("ListRecords", "groups", cloudread.ContextError(ctx, err)) }
		engine, err := a.securityGroupListEngine(ctx)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		policy := RecordListOpts{}
		for _, apply := range snapshot {
			if err := engine.Check(ctx); err != nil {
				yield(nil, wrap(err))
				return
			}
			if apply == nil {
				yield(nil, wrap(findInvalid("nil Security Groups list option")))
				return
			}
			next := copyRecordListOpts(policy)
			if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
				yield(nil, wrap(err))
				return
			}
			policy = copyRecordListOpts(next)
		}
		if _, present := policy.Query["jmespath_filters"]; present {
			yield(nil, wrap(findInvalid("deprecated jmespath_filters is outside the owned JSON list profile")))
			return
		}
		for value, err := range engine.List(ctx, recordListOptions(policy)) {
			if !yield(securityGroupListRecord(value), wrap(err)) {
				return
			}
		}
	}
}

// AllRecords discards partial collected values on error and returns a nonnil
// empty slice on success. Iterator consumers keep already yielded records.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*SecurityGroupRecord, error) {
	values := make([]*SecurityGroupRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
