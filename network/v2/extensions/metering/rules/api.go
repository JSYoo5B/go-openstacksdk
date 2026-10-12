// Package rules implements the pinned openstacksdk network v2 metering
// label rule proxy calls over Neutron /v2.0/metering/metering-label-rules, a
// route Gophercloud v2.15.0 does not bind.
package rules

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/gophercloud/gophercloud/v2"
)

// spec is the pinned MeteringLabelRule Resource: it inherits Resource
// directly (id and name), and project_id falls back to tenant_id. name is
// not a declared query parameter, so find lists without a name hint.
var spec = &ownedrecords.Spec{Resource: "meteringlabelrules", Bound: "metering_label_rule", Profile: neutronfind.Profile{
	Kind: "meteringlabelrules", Path: "metering/metering-label-rules", Singular: "metering_label_rule", Plural: "metering_label_rules",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: false,
	VersionedPath: "/v2.0/metering/metering-label-rules", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "direction", Wire: "direction", Missing: "null"},
		{Name: "is_excluded", Wire: "excluded", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "metering_label_id", Wire: "metering_label_id", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
		{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "remote_ip_prefix", Wire: "remote_ip_prefix", Missing: "null"},
		{Name: "source_ip_prefix", Wire: "source_ip_prefix", Missing: "null"},
		{Name: "destination_ip_prefix", Wire: "destination_ip_prefix", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "direction", Wire: "direction"}, {Name: "fields", Wire: "fields"},
		{Name: "metering_label_id", Wire: "metering_label_id"}, {Name: "remote_ip_prefix", Wire: "remote_ip_prefix"},
		{Name: "source_ip_prefix", Wire: "source_ip_prefix"}, {Name: "destination_ip_prefix", Wire: "destination_ip_prefix"},
		{Name: "project_id", Wire: "project_id"},
		{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
	},
}}

// API shares the selected authenticated network client without owning it.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *API) target() ownedrecords.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return ownedrecords.Target{Spec: spec, Client: client}
}

// Create implements create_metering_label_rule(**attrs): POST
// metering/metering-label-rules with {"metering_label_rule": attrs}.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*MeteringLabelRule, error) {
	return convert(ownedrecords.Create(ctx, a.target(), options))
}

// Get implements get_metering_label_rule(metering_label_rule, fields=...).
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*MeteringLabelRule, error) {
	return convert(ownedrecords.Get(ctx, a.target(), id, options))
}

// Update implements update_metering_label_rule(metering_label_rule,
// **attrs): PUT metering/metering-label-rules/{id} with only the given
// attributes.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*MeteringLabelRule, error) {
	return convert(ownedrecords.Update(ctx, a.target(), id, options))
}

// Delete implements delete_metering_label_rule(metering_label_rule,
// ignore_missing=True).
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) (*DeleteResult, error) {
	return ownedrecords.Delete(ctx, a.target(), id, options)
}

// List implements metering_label_rules(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*MeteringLabelRule, error] {
	source := ownedrecords.List(ctx, a.target(), options)
	return func(yield func(*MeteringLabelRule, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List and discards partial rows on failure.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*MeteringLabelRule, error) {
	var values []*MeteringLabelRule
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_metering_label_rule(name_or_id, ignore_missing=True,
// **query).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*MeteringLabelRule, error) {
	return convert(ownedrecords.Find(ctx, a.target(), nameOrID, options))
}
