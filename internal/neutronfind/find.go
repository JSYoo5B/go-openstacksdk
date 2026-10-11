// Package neutronfind owns Source-shaped Neutron lookup mechanics. Profiles
// are concrete library data; SDK users never implement a builder or decoder.
package neutronfind

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
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// DescriptorKind names the casts used by the pinned resource declarations.
type DescriptorKind string

const (
	Untyped    DescriptorKind = ""
	Boolean    DescriptorKind = "bool"
	Integer    DescriptorKind = "int"
	List       DescriptorKind = "list"
	Dictionary DescriptorKind = "dict"
)

// Descriptor preserves the ordered canonical/wire declaration, explicit null,
// missing JSON default and an optional fallback alias from the Source MRO.
type Descriptor struct {
	Name, Wire     string
	Kind           DescriptorKind
	Missing, Alias string
}

type QueryPair struct{ Name, Wire string }

// Profile belongs to the SDK leaf binding. Headers and tag Body defaults must
// be declared from that class's actual MRO, never assumed for every Neutron type.
type Profile struct {
	Kind, Path, Singular, Plural string
	IDKey, NameKey               string
	Body, Headers                []Descriptor
	Query                        []QueryPair
	PreserveUnknownBody          bool
	NameQueryHint                bool
	VersionedPath                string
}

// Engine captures selected service facts before leaf option callbacks run.
// selectedGuard is supplied only by the private SDK leaf binding.
type Engine struct {
	profile       Profile
	source        *cloudread.Source
	selectedGuard func(context.Context) error
	observed      error
}

func New(ctx context.Context, client *gophercloud.ServiceClient, profile Profile, selectedGuard func(context.Context) error) (*Engine, error) {
	source, err := cloudread.Capture(ctx, client, "network")
	if err != nil {
		return nil, err
	}
	profile.Body = slices.Clone(profile.Body)
	profile.Headers = slices.Clone(profile.Headers)
	profile.Query = slices.Clone(profile.Query)
	engine := &Engine{profile: profile, source: source, selectedGuard: selectedGuard}
	if err := engine.Check(ctx); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) Check(ctx context.Context) error {
	if e == nil || e.source == nil {
		return findInvalid("Neutron find engine is required")
	}
	if e.observed == nil && e.selectedGuard != nil {
		e.observed = e.selectedGuard(ctx)
	}
	return errors.Join(cloudread.ContextError(ctx, e.observed), e.source.Guard(ctx), rest.CheckOperationGuard(ctx))
}

// Record owns the declared Source view and original response evidence.
type Record struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func (v *Record) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	v.row = bytes.Clone(data)
	return json.Unmarshal(data, v.Wire)
}

// Row and FromList preserve private leaf decoder state without sharing raw bytes.
func (v *Record) Row() json.RawMessage { return bytes.Clone(v.row) }
func (v *Record) FromList() bool       { return v.fromList }

func findMetadata(v *Record) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func findFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		out[key] = bytes.Clone(raw)
	}
	return out
}
func findSegment(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) {
		return findInvalid("nonblank UTF-8 identity segment is required")
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return findInvalid("identity contains controls")
		}
	}
	return nil
}
func findCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}

func findRejectedCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 400 + i
	}
	return codes
}

func (e *Engine) findBodyKnown(key string) bool {
	for _, field := range e.profile.Body {
		if field.Name == key {
			return true
		}
	}
	return false
}

// consumeFindBody mirrors the Source ordered canonical/wire mapping. It leaves
// unknown Neutron extensions intact, but removes the reserved response self key.
func (e *Engine) consumeFindBody(body map[string]json.RawMessage, members []cloudfilter.JSONMember) {
	for _, member := range members {
		if member.Key == "self" {
			continue
		}
		consumed := false
		for _, field := range e.profile.Body {
			if member.Key == field.Name || member.Key == field.Wire {
				body[field.Name] = bytes.Clone(member.Value)
				consumed = true
			}
		}
		if !consumed && e.profile.PreserveUnknownBody {
			body[member.Key] = bytes.Clone(member.Value)
		}
	}
}
func (e *Engine) findProject(value *Record, seed map[string]json.RawMessage) error {
	body := findFields(seed)
	headerSeed := make(map[string]json.RawMessage, len(e.profile.Headers))
	for _, field := range e.profile.Headers {
		raw, supplied := body[field.Name]
		for key, value := range body {
			if strings.EqualFold(key, field.Wire) {
				if !supplied {
					raw, supplied = value, true
				}
				delete(body, key)
			}
		}
		delete(body, field.Name)
		if supplied {
			headerSeed[field.Name] = bytes.Clone(raw)
		}
	}
	// Bulk options have no map ordering contract. Canonical constructor values
	// win over wire aliases; ordered response members remain last-value-wins.
	for _, field := range e.profile.Body {
		if raw, exists := body[field.Wire]; exists && field.Name != field.Wire {
			if _, canonical := body[field.Name]; !canonical {
				body[field.Name] = bytes.Clone(raw)
			}
			delete(body, field.Wire)
		}
	}
	delete(body, "self")
	if value.Wire != nil {
		members, err := cloudfilter.ObjectMembers(value.row)
		if err != nil {
			return err
		}
		if value.fromList {
			// Resource.existing consumes row Header attributes; page response
			// headers do not become each row's Header descriptor values.
			rows := make([]cloudfilter.JSONMember, 0, len(members))
			for _, member := range members {
				isHeader := false
				for _, field := range e.profile.Headers {
					if member.Key == field.Name || strings.EqualFold(member.Key, field.Wire) {
						headerSeed[field.Name] = bytes.Clone(member.Value)
						isHeader = true
					}
				}
				if !isHeader {
					rows = append(rows, member)
				}
			}
			members = rows
		}
		e.consumeFindBody(body, members)
	}
	for _, field := range e.profile.Body {
		raw, exists := body[field.Name]
		if !exists && field.Alias != "" {
			raw, exists = body[field.Alias]
		}
		if !exists {
			body[field.Name] = json.RawMessage(field.Missing)
			continue
		}
		converted, err := descriptorJSON(raw, field.Kind)
		if err != nil {
			return err
		}
		body[field.Name] = converted
	}
	// A Body extension using a declared Header name cannot shadow it. Member
	// HTTP headers update Header state; list constructors use row Header attrs.
	for _, field := range e.profile.Headers {
		raw, exists := headerSeed[field.Name]
		if !exists {
			raw = json.RawMessage(field.Missing)
		}
		if !value.fromList {
			for key, values := range value.Header {
				if strings.EqualFold(key, field.Wire) {
					header := ""
					if len(values) > 0 {
						header = values[0]
					}
					raw, _ = json.Marshal(header)
					break
				}
			}
		}
		converted, err := descriptorJSON(raw, field.Kind)
		if err != nil {
			return err
		}
		body[field.Name] = converted
	}
	// Connection-derived mutable location stays an owned null unless supplied.
	if _, exists := body["location"]; !exists {
		body["location"] = json.RawMessage("null")
	}
	value.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	return nil
}

func descriptorJSON(raw json.RawMessage, kind DescriptorKind) (json.RawMessage, error) {
	raw = bytes.Clone(raw)
	trim := bytes.TrimSpace(raw)
	if bytes.Equal(trim, []byte("null")) {
		return raw, nil
	}
	switch kind {
	case Boolean:
		return jsonfilter.BooleanJSON(raw)
	case Integer:
		return jsonfilter.DescriptorIntegerJSON(raw)
	case List:
		if len(trim) > 0 && trim[0] != '[' {
			raw = append(append([]byte{'['}, raw...), ']')
		}
	case Dictionary:
		if len(trim) > 0 && trim[0] != '{' {
			raw = json.RawMessage("{}")
		}
	}
	return raw, nil
}

func findText(value *Record, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Resource.Body[key], &text)
	return text
}
func (e *Engine) findMarker(value *Record) (string, error) {
	if value == nil || value.Wire == nil {
		return "", findInvalid("list marker requires a response identity")
	}
	raw := value.Wire.Body[e.profile.IDKey]
	if len(raw) == 0 {
		return "", findInvalid("list marker requires a response identity")
	}
	values, err := cloudfilter.RequestQueryValues(raw)
	if err != nil || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", errors.Join(findInvalid("list marker must have one nonempty query value"), err)
	}
	return values[0], nil
}
func (e *Engine) findResponse(response *rest.Response, seed map[string]json.RawMessage) (*Record, error) {
	if response == nil {
		return nil, nil
	}
	value := &Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if !json.Valid(response.Body) {
		return value, e.findProject(value, seed)
	}
	if !utf8.Valid(response.Body) {
		return value, findInvalid("find response must contain UTF-8 JSON")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &envelope); err != nil {
		return value, err
	}
	raw := response.Body
	if nested, exists := envelope[e.profile.Singular]; exists {
		raw = nested
	}
	if err := value.UnmarshalJSON(raw); err != nil {
		return value, err
	}
	value.Wire.Header, value.Wire.StatusCode = response.Header.Clone(), response.StatusCode
	return value, e.findProject(value, seed)
}

// Options owns Source kwargs. Omitted IgnoreMissing defaults to true.
// Query preserves JSON values, including arrays and null; unknown kwargs go to
// the member GET but are ignored by the list, following Source Resource.find.
type Options struct {
	IgnoreMissing *bool
	Microversion  *string
	Order         []string
	Query         map[string]json.RawMessage
	Headers       map[string]string
}

func orderedFindKeys(query map[string]json.RawMessage, order []string) []string {
	keys := make([]string, 0, len(query))
	for _, key := range order {
		if _, exists := query[key]; exists && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	missing := make([]string, 0)
	for key := range query {
		if !slices.Contains(keys, key) {
			missing = append(missing, key)
		}
	}
	slices.Sort(missing)
	return append(keys, missing...)
}
func CloneOptions(value Options) Options {
	value.Query = findFields(value.Query)
	value.Order = orderedFindKeys(value.Query, value.Order)
	value.Headers = maps.Clone(value.Headers)
	if value.Microversion != nil {
		x := *value.Microversion
		value.Microversion = &x
	}
	if value.IgnoreMissing != nil {
		x := *value.IgnoreMissing
		value.IgnoreMissing = &x
	}
	return value
}
func findKey(key string) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return findInvalid("query key must be nonblank UTF-8")
	}
	for _, character := range key {
		if unicode.IsControl(character) {
			return findInvalid("query key has controls")
		}
	}
	switch key {
	case "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key":
		return findInvalid("%q is not an owned find query", key)
	case "id":
		return findInvalid("id kwargs collide with the find identity")
	}
	return nil
}
func (e *Engine) prepareFindQuery(policy Options, identity string) (url.Values, url.Values, map[string]json.RawMessage, map[string]json.RawMessage, rest.ListControl, error) {
	getQuery, listQuery := make(url.Values), make(url.Values)
	filters, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	control := rest.ListControl{LimitHint: true}
	for _, key := range policy.Order {
		raw := policy.Query[key]
		if err := findKey(key); err != nil {
			return nil, nil, nil, nil, control, err
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, nil, nil, control, findInvalid("query %q must contain complete UTF-8 JSON", key)
		}
		if key == "all_projects" {
			continue
		} // Source find separates it from fetch; Neutron list discards it.
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		if len(values) != 0 {
			getQuery[key] = values
		}
		if e.profile.PreserveUnknownBody || e.findBodyKnown(key) {
			seed[key] = bytes.Clone(raw)
		}
		for _, field := range e.profile.Headers {
			if key == field.Name || strings.EqualFold(key, field.Wire) {
				seed[key] = bytes.Clone(raw)
			}
		}
		switch key {
		case "max_items":
			var amount int
			if err := json.Unmarshal(raw, &amount); err != nil || amount < 0 {
				return nil, nil, nil, nil, control, findInvalid("max_items must be a nonnegative Go integer")
			}
			control.MaxItems = amount
		case "paginated":
			enabled, err := cloudfilter.PythonTruthy(raw)
			if err != nil {
				return nil, nil, nil, nil, control, err
			}
			control.SinglePage = !enabled
		default:
			declared := false
			for _, pair := range e.profile.Query {
				if pair.Name == key {
					declared = true
				}
			}
			if e.findBodyKnown(key) && !declared {
				filters[key] = bytes.Clone(raw)
			}
		}
	}
	// QueryParameters iterates its declarations. The canonical name wins over
	// its own wire alias, and a later declaration for the same wire key wins.
	for _, pair := range e.profile.Query {
		raw, exists := policy.Query[pair.Name]
		if !exists {
			raw, exists = policy.Query[pair.Wire]
		}
		if !exists {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		delete(listQuery, pair.Wire)
		if len(values) != 0 {
			listQuery[pair.Wire] = slices.Clone(values)
		}
	}
	if e.profile.NameQueryHint {
		if _, supplied := policy.Query[e.profile.NameKey]; !supplied {
			listQuery.Set(e.profile.NameKey, identity)
		}
	}
	return getQuery, listQuery, filters, seed, control, nil
}

func findLocalMatch(actual, filter json.RawMessage) (bool, error) {
	if len(actual) == 0 {
		actual = json.RawMessage("null")
	}
	trim := bytes.TrimSpace(filter)
	if len(trim) > 0 && trim[0] == '{' {
		wanted, err := cloudfilter.ObjectMembers(trim)
		if err != nil {
			return false, err
		}
		truthy, err := cloudfilter.PythonTruthy(actual)
		if err != nil || !truthy {
			return false, err
		}
		// Source reaches d.get only while iterating a nonempty filter.
		if len(wanted) == 0 {
			return true, nil
		}
		var observed map[string]json.RawMessage
		if err := json.Unmarshal(actual, &observed); err != nil || observed == nil {
			return false, fmt.Errorf("local dictionary filter requires an actual mapping")
		}
		for _, member := range wanted {
			matched, err := findLocalMatch(observed[member.Key], member.Value)
			if err != nil || !matched {
				return matched, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}

// Find matches the pinned Source GET-first, complete ID-or-name search.
// It suppresses only clean GET 400/403/404. A physical body/Close/source/cancel
// failure never becomes a fallback or missing-resource success.
func (e *Engine) Find(ctx context.Context, identity string, policy Options) (*Record, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := e.Check(ctx); err != nil {
		return nil, err
	}
	guard := e.Check
	source := e.source
	policy = CloneOptions(policy)
	getQuery, listQuery, filters, seed, control, err := e.prepareFindQuery(policy, identity)
	if err == nil {
		err = errors.Join(source.WithPolicy(ctx, policy.Microversion, policy.Headers), guard(ctx))
	}
	if err != nil {
		return nil, err
	}
	getHeaders := maps.Clone(source.Client.MoreHeaders)
	if getHeaders == nil {
		getHeaders = make(map[string]string)
	}
	var versionAbsent []string
	if source.Client.Microversion != "" {
		getHeaders["OpenStack-API-Version"] = "network " + source.Client.Microversion
	} else {
		versionAbsent = []string{"OpenStack-API-Version"}
	}
	listHeaders := maps.Clone(getHeaders)
	listHeaders["Accept"] = "application/json"
	for key, value := range policy.Headers {
		if strings.EqualFold(key, "Accept") {
			listHeaders["Accept"] = value
		}
	}
	listClient := source.Client
	listClient.MoreHeaders = maps.Clone(listHeaders)
	get := func(_ context.Context, id string) (*Record, error) {
		target := source.Client.ServiceURL(e.profile.Path, url.PathEscape(id))
		if len(getQuery) != 0 {
			target += "?" + getQuery.Encode()
		}
		response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodGet, target, nil, rest.RequestHeaderPolicy{Values: getHeaders, Absent: versionAbsent}, rest.RejectionPolicy{Codes: findRejectedCodes(), PreserveCleanRetry: true}, findCodes()...)
		if err != nil {
			return nil, err
		}
		owned := findFields(seed)
		owned[e.profile.IDKey], _ = json.Marshal(id)
		value, err := e.findResponse(response, owned)
		if err = errors.Join(err, guard(ctx)); err != nil {
			err = response.Fail(err)
		}
		return value, err
	}
	list := func(context.Context, url.Values) iter.Seq2[*Record, error] {
		return func(yield func(*Record, error) bool) {
			var origin *rest.Response
			spec := rest.CollectionSpec[Record]{Client: &listClient, Path: e.profile.Path, Kind: e.profile.Kind, PluralKey: e.profile.Plural, Metadata: findMetadata, Validate: guard, SourceGuard: guard, ListCodes: findCodes(), ReadPage: func(pageCtx context.Context, target string, codes ...int) (*rest.Response, error) {
				return rest.DoJSONGuardedRejectionsHeaderPolicy(pageCtx, &listClient, guard, http.MethodGet, target, nil, rest.RequestHeaderPolicy{Values: listHeaders, Absent: versionAbsent}, rest.RejectionPolicy{Codes: findRejectedCodes(), PreserveCleanRetry: true}, codes...)
			}, ValidateResponse: func(response *rest.Response) error { origin = response; return guard(ctx) }, ValidateItem: func(value *Record) error {
				value.fromList = true
				for _, key := range []string{"connection", "microversion", "_synchronized"} {
					if _, exists := value.Wire.Body[key]; exists {
						return fmt.Errorf("list row %q collides with Resource.existing constructor", key)
					}
				}
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				return errors.Join(e.findProject(value, nil), guard(ctx))
			}, Paging: rest.PagePolicy[Record]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: e.findMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, VersionedPath: e.profile.VersionedPath}}
			for value, err := range rest.ListWithControl(ctx, spec, listQuery, control) {
				if err != nil {
					yield(nil, err)
					return
				}
				matched := true
				for _, key := range policy.Order {
					filter, exists := filters[key]
					if !exists {
						continue
					}
					equal, cause := findLocalMatch(value.Resource.Body[key], filter)
					if cause = errors.Join(cause, guard(ctx)); cause != nil {
						yield(nil, origin.Fail(cause))
						return
					}
					if !equal {
						matched = false
						break
					}
				}
				if matched && !yield(value, nil) {
					return
				}
			}
		}
	}
	collection := resource.NewCollection(resource.Adapter[Record]{Kind: e.profile.Kind, IdentityFind: true, ValidateID: findSegment, IdentityDirectGet: findSegment, IdentityResponseID: func(value *Record) (string, error) {
		return findText(value, e.profile.IDKey), nil
	}, Name: func(value *Record) string { return findText(value, e.profile.NameKey) }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	return value, errors.Join(err, guard(ctx))
}
