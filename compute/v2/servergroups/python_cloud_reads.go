package servergroups

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// This is the pinned ServerGroup declaration, not NetworkResource. Only rules
// has a type cast; members is the wire name of the untyped member_ids field.
// Unknown attrs stay only in Wire because Source disallows unknown Body attrs.
func cloudGroupProfile() neutronfind.Profile {
	p := neutronfind.Profile{Kind: "servergroups", ServiceRole: "compute", Path: "os-server-groups", Singular: "server_group", Plural: "server_groups", VersionedPath: "/v2.1/os-server-groups", IDKey: "id", NameKey: "name", SourceFindPolicy: true, SourceStatusPolicy: true, SourceCollectionGET: true, SourceLinkPolicy: true, PythonLoadedEquality: true, StopFaultRetryCallbacks: true}
	for _, name := range groupRecordFields {
		wire := name
		kind := neutronfind.Untyped
		if name == "member_ids" {
			wire = "members"
		}
		if name == "rules" {
			kind = neutronfind.Dictionary
		}
		p.Body = append(p.Body, neutronfind.Descriptor{Name: name, Wire: wire, Kind: kind, Missing: "null"})
	}
	p.Query = []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "all_projects", Wire: "all_projects"}}
	return p
}
func cloudGroupRecord(v *neutronfind.Record) *ServerGroupRecord {
	if v == nil {
		return nil
	}
	return &ServerGroupRecord{Resource: v.Resource.Clone(), Wire: v.Wire.Clone(), Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
}
func (a *API) cloudGroupEngine(ctx context.Context, identity *string) (*neutronfind.Engine, error) {
	p, err := a.captureGroupRecord(ctx)
	if err == nil && identity != nil && *identity != "" {
		err = groupRecordID(*identity)
	}
	if err != nil {
		return nil, err
	}
	// Keep the existing selected-version-first/2.64 ceiling policy. This copied
	// discovery opts into physical-fault hook fences, preserving old FindRecord.
	if p.source.Client.Microversion == "" {
		profile := microversions.NovaProfile
		profile.StopFaultRetryCallbacks = true
		discovered, cause := microversions.Read(ctx, p.source, profile)
		if cause != nil {
			return nil, cause
		}
		version, cause := microversions.SelectVersion(discovered.Maximum, discovered.Minimum, "2.64")
		if cause != nil {
			return nil, cause
		}
		if version != "" {
			if cause = p.source.WithPolicy(ctx, &version, nil); cause != nil {
				return nil, cause
			}
		}
	}
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	return neutronfind.New(ctx, &p.source.Client, cloudGroupProfile(), p.guard)
}

// ListCloudRecords is the narrow Source server_groups() read used by cloud
// search. It has no filter/query pushdown or all-projects privilege default.
// Existing native APIs and FindRecord retain their reviewed policies.
func (a *API) ListCloudRecords(ctx context.Context) iter.Seq2[*ServerGroupRecord, error] {
	return func(yield func(*ServerGroupRecord, error) bool) {
		e, err := a.cloudGroupEngine(ctx, nil)
		if err != nil {
			yield(nil, groupRecordWrap(ctx, "ListCloudRecords", err))
			return
		}
		for v, err := range e.List(ctx, neutronfind.Options{}) {
			if !yield(cloudGroupRecord(v), groupRecordWrap(ctx, "ListCloudRecords", err)) {
				return
			}
		}
	}
}

// CloudGroupFindResult separates the member observation from consumed fallback
// rows. A second exact match stops before any later page, as Resource.find does.
type CloudGroupFindResult struct {
	Group, Observed *ServerGroupRecord
	Inventory       []*ServerGroupRecord
}

func cloudGroupCleanFallback(err error) bool {
	if err == nil {
		return false
	}
	if terminal, ok := err.(interface{ TerminalSDKFailure() bool }); ok && terminal.TerminalSDKFailure() {
		return false
	}
	switch v := err.(type) {
	case gophercloud.ErrUnexpectedResponseCode:
		return v.Actual == 400 || v.Actual == 403 || v.Actual == 404
	case interface{ Unwrap() []error }:
		causes := v.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !cloudGroupCleanFallback(cause) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		return cloudGroupCleanFallback(v.Unwrap())
	}
	return false
}
func cloudGroupRejectedObservation(err error) *ServerGroupRecord {
	actual := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		actual = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(actual, &native) {
		return &ServerGroupRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	var proof *resource.ResponseError
	if errors.As(err, &proof) {
		return &ServerGroupRecord{Envelope: bytes.Clone(proof.Body), Header: proof.Header.Clone(), StatusCode: proof.StatusCode}
	}
	return nil
}

// FindCloudRecord follows only the default cloud find path: GET first, clean
// 400/403/404 fallback, exact equality without globs, ignore missing true, and
// second-match immediate ambiguity. Empty string uses Source's trailing-slash
// collection GET; the caller's None identity is not an empty string.
func (a *API) FindCloudRecord(ctx context.Context, identity string) (*CloudGroupFindResult, error) {
	e, err := a.cloudGroupEngine(ctx, &identity)
	if err != nil {
		return nil, groupRecordWrap(ctx, "FindCloudRecord", err)
	}
	result := &CloudGroupFindResult{}
	value, err := e.Get(ctx, identity, neutronfind.Options{})
	result.Observed = cloudGroupRecord(value)
	if result.Observed == nil {
		result.Observed = cloudGroupRejectedObservation(err)
	}
	if err == nil {
		result.Group = cloudGroupRecord(value)
		return result, nil
	}
	if !cloudGroupCleanFallback(err) || e.Check(ctx) != nil {
		return result, groupRecordWrap(ctx, "FindCloudRecord", errors.Join(err, e.Check(ctx)))
	}
	for value, cause := range e.List(ctx, neutronfind.Options{}) {
		row := cloudGroupRecord(value)
		if row != nil {
			result.Inventory = append(result.Inventory, row)
		}
		if cause = errors.Join(cause, e.Check(ctx)); cause != nil {
			result.Group = nil
			return result, groupRecordWrap(ctx, "FindCloudRecord", cause)
		}
		if row == nil || row.Resource == nil {
			result.Group = nil
			return result, groupRecordWrap(ctx, "FindCloudRecord", groupRecordInvalid("fallback yielded no resource"))
		}
		// Source _get_one_match uses == directly; it does not stringify bool/null
		// IDs or turn a default-find identity into a wildcard pattern.
		matches := false
		for _, key := range []string{"id", "name"} {
			raw := bytes.TrimSpace(row.Resource.Body[key])
			if len(raw) == 0 || raw[0] != '"' {
				continue
			}
			var text string
			if json.Unmarshal(raw, &text) == nil && text == identity {
				matches = true
			}
		}
		if !matches {
			continue
		}
		if result.Group != nil {
			result.Group = nil
			err := (&rest.Response{Body: bytes.Clone(row.Envelope), Header: row.Header.Clone(), StatusCode: row.StatusCode}).Fail(&resource.AmbiguousError{Resource: "servergroup", Name: identity})
			return result, groupRecordWrap(ctx, "FindCloudRecord", err)
		}
		result.Group = cloudGroupRecord(value)
	}
	if err = e.Check(ctx); err != nil {
		result.Group = nil
	}
	return result, groupRecordWrap(ctx, "FindCloudRecord", err)
}
