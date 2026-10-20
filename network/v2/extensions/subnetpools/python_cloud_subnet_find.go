package subnetpools

import (
	"bytes"
	"context"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FindSubnetSourceRecord supplies create_subnet's default subnetpool lookup.
// This Source opt-in keeps the old typed/list dictionary-link policy unchanged.
func (a *API) FindSubnetSourceRecord(ctx context.Context, identity string) (*SubnetPoolRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("FindSubnetSourceRecord", "subnetpools", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(poolRecordInvalid("configured subnetpools API required"))
	}
	selected := a.client
	profile := neutronfind.Profile{Kind: "subnetpools", Path: "subnetpools", Singular: "subnetpool", Plural: "subnetpools", IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true, VersionedPath: "/v2.0/subnetpools", SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, PythonLoadedEquality: true, StopFaultRetryCallbacks: true,
		Body: []neutronfind.Descriptor{
			{Name: "id", Wire: "id", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "tags", Wire: "tags", Kind: neutronfind.List, Missing: "[]"},
			{Name: "address_scope_id", Wire: "address_scope_id", Kind: neutronfind.Untyped, Missing: "null"},
			{Name: "created_at", Wire: "created_at", Kind: neutronfind.Untyped, Missing: "null"},
			{Name: "default_prefix_length", Wire: "default_prefixlen", Kind: neutronfind.Integer, Missing: "null"},
			{Name: "default_quota", Wire: "default_quota", Kind: neutronfind.Integer, Missing: "null"},
			{Name: "description", Wire: "description", Kind: neutronfind.Untyped, Missing: "null"},
			{Name: "ip_version", Wire: "ip_version", Kind: neutronfind.Integer, Missing: "null"},
			{Name: "is_default", Wire: "is_default", Kind: neutronfind.Boolean, Missing: "null"},
			{Name: "is_shared", Wire: "shared", Kind: neutronfind.Boolean, Missing: "null"},
			{Name: "maximum_prefix_length", Wire: "max_prefixlen", Kind: neutronfind.Integer, Missing: "null"},
			{Name: "minimum_prefix_length", Wire: "min_prefixlen", Kind: neutronfind.Integer, Missing: "null"},
			{Name: "name", Wire: "name", Kind: neutronfind.Untyped, Missing: "null"},
			{Name: "prefixes", Wire: "prefixes", Kind: neutronfind.List, Missing: "null"},
			{Name: "project_id", Wire: "project_id", Kind: neutronfind.Untyped, Missing: "null", Alias: "tenant_id"},
			{Name: "tenant_id", Wire: "tenant_id", Kind: neutronfind.Untyped, Missing: "null"},
			{Name: "revision_number", Wire: "revision_number", Kind: neutronfind.Integer, Missing: "null"},
			{Name: "updated_at", Wire: "updated_at", Kind: neutronfind.Untyped, Missing: "null"},
		}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "name", Wire: "name"}}}
	engine, err := neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return poolRecordInvalid("selected subnetpools client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	value, err := engine.Find(ctx, identity, neutronfind.Options{})
	if value != nil {
		return &SubnetPoolRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}, wrap(err)
	}
	var physical *resource.ResponseError
	if errors.As(err, &physical) && physical != nil {
		return &SubnetPoolRecord{Envelope: bytes.Clone(physical.Body), Header: physical.Header.Clone(), StatusCode: physical.StatusCode}, wrap(err)
	}
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) {
		return &SubnetPoolRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}, wrap(err)
	}
	return nil, wrap(err)
}
