package servergroups

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
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServerGroupRecord separates declared Source descriptors from the actual row.
// Raw untyped values and unknown response fields stay available in Wire.
type ServerGroupRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
}

func (v *ServerGroupRecord) UnmarshalJSON(data []byte) error {
	v.Wire = &resource.RawResource{}
	if err := json.Unmarshal(data, v.Wire); err != nil {
		return err
	}
	v.row = bytes.Clone(data)
	return nil
}

var groupRecordFields = []string{"id", "name", "policies", "policy", "member_ids", "metadata", "project_id", "rules", "user_id"}

func groupRecordCanonical(key string) (string, bool) {
	if key == "members" {
		return "member_ids", true
	}
	for _, field := range groupRecordFields {
		if key == field {
			return key, true
		}
	}
	return "", false
}
func groupRecordInvalid(format string, values ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, values...))
}
func groupRecordID(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) {
		return groupRecordInvalid("identifier must be nonblank UTF-8 literal text")
	}
	for _, c := range value {
		if unicode.IsControl(c) {
			return groupRecordInvalid("identifier has controls")
		}
	}
	return nil
}
func groupRecordNormalized(data json.RawMessage) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(data)
	if err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{}
	for _, member := range members {
		if key, ok := groupRecordCanonical(member.Key); ok {
			fields[key] = bytes.Clone(member.Value)
		}
	}
	return fields, nil
}
func groupRecordProject(v *ServerGroupRecord, fields map[string]json.RawMessage) error {
	body := map[string]json.RawMessage{}
	for _, key := range groupRecordFields {
		raw, err := resource.BodyRecordField(fields, key, resource.BodyFieldJSON)
		if err != nil {
			return err
		}
		// Source fields._convert_type(dict) returns {} for nonmapping, nonnull values.
		if key == "rules" {
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) != 0 && trimmed[0] != '{' && !bytes.Equal(trimmed, []byte("null")) {
				raw = json.RawMessage(`{}`)
			}
		}
		body[key] = bytes.Clone(raw)
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func groupRecordMetadata(v *ServerGroupRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func groupRecordReceipt(r *rest.Response) *ServerGroupRecord {
	if r == nil {
		return nil
	}
	return &ServerGroupRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
}
func groupRecordResponse(r *rest.Response, id string) (*ServerGroupRecord, error) {
	v := groupRecordReceipt(r)
	if v == nil {
		return nil, nil
	}
	rawID, _ := json.Marshal(id)
	fields := map[string]json.RawMessage{"id": rawID}
	if json.Valid(r.Body) {
		if !utf8.Valid(r.Body) {
			return v, groupRecordInvalid("response must be UTF-8 JSON")
		}
		var root resource.RawResource
		if err := json.Unmarshal(r.Body, &root); err != nil {
			return v, err
		}
		raw := r.Body
		if nested, ok := root.Body["server_group"]; ok {
			raw = nested
		}
		var wire resource.RawResource
		if err := json.Unmarshal(raw, &wire); err != nil {
			return v, err
		}
		wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
		v.Wire = wire.Clone()
		observed, err := groupRecordNormalized(raw)
		if err != nil {
			return v, err
		}
		for key, value := range observed {
			fields[key] = value
		}
	}
	return v, groupRecordProject(v, fields)
}
func groupRecordText(v *ServerGroupRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var value string
	_ = json.Unmarshal(v.Resource.Body[key], &value)
	return value
}
func groupRecordMarker(v *ServerGroupRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", groupRecordInvalid("group marker missing")
	}
	var value string
	if err := json.Unmarshal(v.Wire.Body["id"], &value); err != nil || value == "" {
		return "", groupRecordInvalid("group marker must be a nonempty string")
	}
	return value, nil
}

type preparedGroupRecord struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

func groupRecordCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func groupRecordWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "servergroups", cloudread.ContextError(ctx, err))
}
func (a *API) captureGroupRecord(ctx context.Context) (*preparedGroupRecord, error) {
	if a == nil {
		return nil, groupRecordInvalid("server group API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, err
	}
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && a.client != selected {
			changed = groupRecordInvalid("selected group client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &preparedGroupRecord{ctx: ctx, source: source, guard: guard}, guard(ctx)
}
func (p *preparedGroupRecord) policy(headers map[string]string) error {
	if p.source.Client.Microversion == "" {
		discovery, err := microversions.Read(p.ctx, p.source, microversions.NovaProfile)
		if err != nil {
			return err
		}
		version, err := microversions.SelectVersion(discovery.Maximum, discovery.Minimum, "2.64")
		if err != nil {
			return err
		}
		if version != "" {
			if err := p.source.WithPolicy(p.ctx, &version, nil); err != nil {
				return err
			}
		}
	}
	return errors.Join(p.source.WithPolicy(p.ctx, nil, headers), p.guard(p.ctx))
}
func (p *preparedGroupRecord) get(id string) (*ServerGroupRecord, error) {
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodGet, p.source.Client.ServiceURL("os-server-groups", url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: []int{400, 403, 404}, PreserveCleanRetry: true}, groupRecordCodes()...)
	if err != nil {
		return groupRecordReceipt(response), err
	}
	value, err := groupRecordResponse(response, id)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}
func (p *preparedGroupRecord) list(query url.Values) iter.Seq2[*ServerGroupRecord, error] {
	return func(yield func(*ServerGroupRecord, error) bool) {
		var origin *rest.Response
		spec := rest.CollectionSpec[ServerGroupRecord]{Client: &p.source.Client, Path: "os-server-groups", Kind: "servergroups", PluralKey: "server_groups", Metadata: groupRecordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: groupRecordCodes(), ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *ServerGroupRecord) error {
			v.Envelope = bytes.Clone(origin.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			fields, err := groupRecordNormalized(v.row)
			if err != nil {
				return err
			}
			return errors.Join(groupRecordProject(v, fields), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[ServerGroupRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: groupRecordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for value, err := range rest.ListWithControl(p.ctx, spec, query, rest.ListControl{}) {
			if !yield(value, err) {
				return
			}
		}
	}
}

// GroupFindOpts mirrors find_server_group's defaults. AllProjects is included
// only on the fallback list, never the direct member GET.
type GroupFindOpts struct {
	IgnoreMissing *bool
	AllProjects   bool
	Headers       map[string]string
}
type GroupFindOption func(*GroupFindOpts) error

func copyGroupFind(v GroupFindOpts) GroupFindOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithGroupFindOpts(value GroupFindOpts) GroupFindOption {
	owned := copyGroupFind(value)
	return func(v *GroupFindOpts) error { *v = copyGroupFind(owned); return nil }
}
func WithGroupFindIgnoreMissing(value bool) GroupFindOption {
	return func(v *GroupFindOpts) error { x := value; v.IgnoreMissing = &x; return nil }
}
func WithGroupFindAllProjects(value bool) GroupFindOption {
	return func(v *GroupFindOpts) error { v.AllProjects = value; return nil }
}
func WithGroupFindHeaders(value map[string]string) GroupFindOption {
	owned := maps.Clone(value)
	return func(v *GroupFindOpts) error { v.Headers = maps.Clone(owned); return nil }
}

// FindRecord GETs the literal identity first; a fault-free 400/403/404 permits
// a complete exact ID-or-name search. Missing defaults to nil,nil; duplicate
// matches, late page failures and physical response errors are never ignored.
func (a *API) FindRecord(ctx context.Context, nameOrID string, options ...GroupFindOption) (*ServerGroupRecord, error) {
	const op = "FindRecord"
	p, err := a.captureGroupRecord(ctx)
	if err == nil {
		err = groupRecordID(nameOrID)
	}
	if err != nil {
		return nil, groupRecordWrap(ctx, op, err)
	}
	opts := GroupFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err = p.guard(ctx); err != nil {
			return nil, groupRecordWrap(ctx, op, err)
		}
		if apply == nil {
			return nil, groupRecordWrap(ctx, op, groupRecordInvalid("nil find option"))
		}
		next := copyGroupFind(opts)
		if err = errors.Join(apply(&next), p.guard(ctx)); err != nil {
			return nil, groupRecordWrap(ctx, op, err)
		}
		opts = copyGroupFind(next)
	}
	if err = p.policy(opts.Headers); err != nil {
		return nil, groupRecordWrap(ctx, op, err)
	}
	collection := resource.NewCollection(resource.Adapter[ServerGroupRecord]{Kind: "servergroups", IdentityFind: true, ValidateID: groupRecordID, IdentityDirectGet: groupRecordID, IdentityResponseID: func(v *ServerGroupRecord) (string, error) {
		if v == nil {
			return "", groupRecordInvalid("find response missing")
		}
		return groupRecordText(v, "id"), nil
	}, Name: func(v *ServerGroupRecord) string { return groupRecordText(v, "name") }, Get: func(_ context.Context, id string) (*ServerGroupRecord, error) { return p.get(id) }, Iterate: func(_ context.Context, _ url.Values) iter.Seq2[*ServerGroupRecord, error] {
		query := url.Values{}
		if opts.AllProjects {
			query.Set("all_projects", "True")
		}
		if _, ok := p.source.Client.MoreHeaders["Accept"]; !ok {
			p.source.Client.MoreHeaders["Accept"] = "application/json"
		}
		return p.list(query)
	}})
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(opts.IgnoreMissing == nil || *opts.IgnoreMissing))
	return value, groupRecordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
}
