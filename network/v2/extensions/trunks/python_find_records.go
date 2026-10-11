package trunks

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
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TrunkFindRecord owns the declared Source view and original response evidence.
// Wire contains received fields; Resource includes owned constructor/default values.
type TrunkFindRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func (v *TrunkFindRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	v.row = bytes.Clone(data)
	return json.Unmarshal(data, v.Wire)
}
func findMetadata(v *TrunkFindRecord) *resource.Metadata {
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

// findTruthy mirrors CPython JSON-domain scalar truthiness. JSON numbers
// with a fraction/exponent parse as binary64; underflow is false, overflow is
// true. Plain integers retain arbitrary precision through the shared parser.
// It changes no received or caller JSON bytes, and is not numeric equality.
func findTruthy(raw json.RawMessage) (bool, error) {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return false, findInvalid("truthiness requires complete UTF-8 JSON")
	}
	token := bytes.TrimSpace(raw)
	if (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		number, err := strconv.ParseFloat(string(token), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return false, err
		}
		return number != 0, nil
	}
	return cloudfilter.PythonTruthy(raw)
}
func findBoolean(raw json.RawMessage) (json.RawMessage, error) {
	truth, err := findTruthy(raw)
	if err != nil {
		return nil, err
	}
	if truth {
		return json.RawMessage("true"), nil
	}
	return json.RawMessage("false"), nil
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

type findDescriptor struct{ name, wire, kind, missing, alias string }

func findRejectedCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 400 + i
	}
	return codes
}

var findDescriptors = []findDescriptor{
	{"id", "id", "", "null", ""},
	{"name", "name", "", "null", ""},
	{"description", "description", "", "null", ""},
	{"project_id", "project_id", "", "null", "tenant_id"},
	{"tenant_id", "tenant_id", "", "null", ""},
	{"is_admin_state_up", "admin_state_up", "bool", "null", ""},
	{"port_id", "port_id", "", "null", ""},
	{"status", "status", "", "null", ""},
	{"sub_ports", "sub_ports", "list", "null", ""},
	{"tags", "tags", "list", "[]", ""},
}

type findQueryPair struct{ name, wire string }

var findQueryPairs = []findQueryPair{
	{"limit", "limit"},
	{"marker", "marker"},
	{"name", "name"},
	{"description", "description"},
	{"fields", "fields"},
	{"port_id", "port_id"},
	{"status", "status"},
	{"sub_ports", "sub_ports"},
	{"project_id", "project_id"},
	{"is_admin_state_up", "admin_state_up"},
	{"tags", "tags"},
	{"any_tags", "tags-any"},
	{"not_tags", "not-tags"},
	{"not_any_tags", "not-tags-any"},
}

func findBodyKnown(key string) bool {
	for _, field := range findDescriptors {
		if field.name == key {
			return true
		}
	}
	return false
}

// consumeFindBody mirrors the Source ordered canonical/wire mapping. It leaves
// unknown Neutron extensions intact, but removes the reserved response self key.
func consumeFindBody(body map[string]json.RawMessage, members []cloudfilter.JSONMember) {
	for _, member := range members {
		if member.Key == "self" {
			continue
		}
		consumed := false
		for _, field := range findDescriptors {
			if member.Key == field.name || member.Key == field.wire {
				body[field.name] = bytes.Clone(member.Value)
				consumed = true
			}
		}
		if !consumed {
			body[member.Key] = bytes.Clone(member.Value)
		}
	}
}
func findProject(value *TrunkFindRecord, seed map[string]json.RawMessage) error {
	body := findFields(seed)
	// Constructor seed was consumed in query order before entering projection.
	// Ordered response fields then overwrite it using the same Body mapping.
	delete(body, "self")
	if value.Wire != nil {
		members, err := cloudfilter.ObjectMembers(value.row)
		if err != nil {
			return err
		}
		consumeFindBody(body, members)
	}
	for _, field := range findDescriptors {
		raw, exists := body[field.name]
		if !exists && field.alias != "" {
			raw, exists = body[field.alias]
		}
		if !exists {
			body[field.name] = json.RawMessage(field.missing)
			continue
		}
		raw = bytes.Clone(raw)
		trim := bytes.TrimSpace(raw)
		if !bytes.Equal(trim, []byte("null")) {
			var err error
			switch field.kind {
			case "bool":
				raw, err = findBoolean(raw)
			case "int":
				raw, err = jsonfilter.DescriptorIntegerJSON(raw)
			case "list":
				if len(trim) > 0 && trim[0] != '[' {
					raw = append(append([]byte{'['}, raw...), ']')
				}
			case "dict":
				if len(trim) > 0 && trim[0] != '{' {
					raw = json.RawMessage("{}")
				}
			}
			if err != nil {
				return err
			}
		}
		body[field.name] = raw
	}
	// Connection-derived mutable location is intentionally an owned null unless
	// an extension returned a literal location; the original JSON is retained.
	if _, exists := body["location"]; !exists {
		body["location"] = json.RawMessage("null")
	}
	value.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	return nil
}
func findText(value *TrunkFindRecord, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Resource.Body[key], &text)
	return text
}
func findMarker(value *TrunkFindRecord) (string, error) {
	if value == nil || value.Wire == nil {
		return "", findInvalid("list marker requires a response identity")
	}
	raw := value.Wire.Body["id"]
	if len(raw) == 0 {
		return "", findInvalid("list marker requires a response identity")
	}
	values, err := cloudfilter.RequestQueryValues(raw)
	if err != nil || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", errors.Join(findInvalid("list marker must have one nonempty query value"), err)
	}
	return values[0], nil
}
func findResponse(response *rest.Response, seed map[string]json.RawMessage) (*TrunkFindRecord, error) {
	if response == nil {
		return nil, nil
	}
	value := &TrunkFindRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if !json.Valid(response.Body) {
		return value, findProject(value, seed)
	}
	if !utf8.Valid(response.Body) {
		return value, findInvalid("find response must contain UTF-8 JSON")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &envelope); err != nil {
		return value, err
	}
	raw := response.Body
	if nested, exists := envelope["trunk"]; exists {
		raw = nested
	}
	if err := value.UnmarshalJSON(raw); err != nil {
		return value, err
	}
	value.Wire.Header, value.Wire.StatusCode = response.Header.Clone(), response.StatusCode
	return value, findProject(value, seed)
}

// RecordFindOpts owns Source kwargs. Omitted IgnoreMissing defaults to true.
// Query preserves JSON values, including arrays and null; unknown kwargs go to
// the member GET but are ignored by the list, following Source Resource.find.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	queryOrder    []string
	Query         map[string]json.RawMessage
	Headers       map[string]string
}
type RecordFindOption func(*RecordFindOpts) error

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
func copyFindOpts(value RecordFindOpts) RecordFindOpts {
	value.Query = findFields(value.Query)
	value.queryOrder = orderedFindKeys(value.Query, value.queryOrder)
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
func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyFindOpts(value)
	return func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil }
}
func WithRecordFindIgnoreMissing(value bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := value; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.queryOrder, key) {
			c.queryOrder = append(c.queryOrder, key)
		}
		return nil
	}
}
func WithRecordFindMicroversion(value string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := value; c.Microversion = &x; return nil }
}
func WithRecordFindHeaders(value map[string]string) RecordFindOption {
	owned := maps.Clone(value)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
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
	case "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return findInvalid("%q is not an owned find query", key)
	case "id":
		return findInvalid("id kwargs collide with the find identity")
	}
	return nil
}
func prepareFindQuery(policy RecordFindOpts, identity string) (url.Values, url.Values, map[string]json.RawMessage, map[string]json.RawMessage, rest.ListControl, error) {
	getQuery, listQuery := make(url.Values), make(url.Values)
	filters, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	control := rest.ListControl{LimitHint: true}
	for _, key := range policy.queryOrder {
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
		// Resource.existing consumes kwargs in insertion order. This is
		// distinct from the canonical-name preference of LIST query mapping.
		consumeFindBody(seed, []cloudfilter.JSONMember{{Key: key, Value: raw}})
		switch key {
		case "max_items":
			var amount int
			if err := json.Unmarshal(raw, &amount); err != nil || amount < 0 {
				return nil, nil, nil, nil, control, findInvalid("max_items must be a nonnegative Go integer")
			}
			control.MaxItems = amount
		case "paginated":
			enabled, err := findTruthy(raw)
			if err != nil {
				return nil, nil, nil, nil, control, err
			}
			control.SinglePage = !enabled
		default:
			declared := false
			for _, pair := range findQueryPairs {
				if pair.name == key {
					declared = true
				}
			}
			if findBodyKnown(key) && !declared {
				filters[key] = bytes.Clone(raw)
			}
		}
	}
	// QueryParameters iterates its declarations. The canonical name wins over
	// its own wire alias, and a later declaration for the same wire key wins.
	for _, pair := range findQueryPairs {
		raw, exists := policy.Query[pair.name]
		if !exists {
			raw, exists = policy.Query[pair.wire]
		}
		if !exists {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		delete(listQuery, pair.wire)
		if len(values) != 0 {
			listQuery[pair.wire] = slices.Clone(values)
		}
	}
	if _, supplied := policy.Query["name"]; !supplied {
		listQuery.Set("name", identity)
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
		truthy, err := findTruthy(actual)
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

// FindRecord matches the pinned Source GET-first, complete ID-or-name search.
// It suppresses only clean GET 400/403/404. A physical body/Close/source/cancel
// failure never becomes a fallback or missing-resource success.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*TrunkFindRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "trunks", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("trunks API is required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, wrap(err)
	}
	listSource := *source
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && a.client != selected {
			changed = findInvalid("selected trunks client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	policy := RecordFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil find option"))
		}
		next := copyFindOpts(policy)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	getQuery, listQuery, filters, seed, control, err := prepareFindQuery(policy, identity)
	if err == nil {
		// Source Resource.find forwards its microversion only to fetch. Its
		// list uses the captured service default; each phase fixes that choice.
		if policy.Microversion != nil {
			source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
			for key := range source.Client.MoreHeaders {
				if strings.EqualFold(key, "OpenStack-API-Version") {
					delete(source.Client.MoreHeaders, key)
				}
			}
		}
		err = errors.Join(source.WithPolicy(ctx, policy.Microversion, policy.Headers), listSource.WithPolicy(ctx, nil, policy.Headers), guard(ctx))
	}
	if err != nil {
		return nil, wrap(err)
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
	listHeaders := maps.Clone(listSource.Client.MoreHeaders)
	if listHeaders == nil {
		listHeaders = make(map[string]string)
	}
	var listVersionAbsent []string
	if listSource.Client.Microversion != "" {
		listHeaders["OpenStack-API-Version"] = "network " + listSource.Client.Microversion
	} else {
		listVersionAbsent = []string{"OpenStack-API-Version"}
	}
	listHeaders["Accept"] = "application/json"
	for key, value := range policy.Headers {
		if strings.EqualFold(key, "Accept") {
			listHeaders["Accept"] = value
		}
	}
	listClient := listSource.Client
	listClient.MoreHeaders = maps.Clone(listHeaders)
	get := func(_ context.Context, id string) (*TrunkFindRecord, error) {
		target := source.Client.ServiceURL("trunks", url.PathEscape(id))
		if len(getQuery) != 0 {
			target += "?" + getQuery.Encode()
		}
		response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodGet, target, nil, rest.RequestHeaderPolicy{Values: getHeaders, Absent: versionAbsent}, rest.RejectionPolicy{Codes: findRejectedCodes(), PreserveCleanRetry: true}, findCodes()...)
		if err != nil {
			return nil, err
		}
		owned := findFields(seed)
		owned["id"], _ = json.Marshal(id)
		value, err := findResponse(response, owned)
		if err = errors.Join(err, guard(ctx)); err != nil {
			err = response.Fail(err)
		}
		return value, err
	}
	list := func(context.Context, url.Values) iter.Seq2[*TrunkFindRecord, error] {
		return func(yield func(*TrunkFindRecord, error) bool) {
			var origin *rest.Response
			spec := rest.CollectionSpec[TrunkFindRecord]{Client: &listClient, Path: "trunks", Kind: "trunks", PluralKey: "trunks", Metadata: findMetadata, Validate: guard, SourceGuard: guard, ListCodes: findCodes(), ReadPage: func(pageCtx context.Context, target string, codes ...int) (*rest.Response, error) {
				return rest.DoJSONGuardedRejectionsHeaderPolicy(pageCtx, &listClient, guard, http.MethodGet, target, nil, rest.RequestHeaderPolicy{Values: listHeaders, Absent: listVersionAbsent}, rest.RejectionPolicy{Codes: findRejectedCodes(), PreserveCleanRetry: true}, codes...)
			}, ValidateResponse: func(response *rest.Response) error { origin = response; return guard(ctx) }, ValidateItem: func(value *TrunkFindRecord) error {
				value.fromList = true
				for _, key := range []string{"connection", "microversion", "_synchronized"} {
					if _, exists := value.Wire.Body[key]; exists {
						return fmt.Errorf("list row %q collides with Resource.existing constructor", key)
					}
				}
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				return errors.Join(findProject(value, nil), guard(ctx))
			}, Paging: rest.PagePolicy[TrunkFindRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: findMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, VersionedPath: "/v2.0/trunks"}}
			for value, err := range rest.ListWithControl(ctx, spec, listQuery, control) {
				if err != nil {
					yield(nil, err)
					return
				}
				matched := true
				for _, key := range policy.queryOrder {
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
	collection := resource.NewCollection(resource.Adapter[TrunkFindRecord]{Kind: "trunks", IdentityFind: true, ValidateID: findSegment, IdentityDirectGet: findSegment, IdentityResponseID: func(value *TrunkFindRecord) (string, error) {
		return findText(value, "id"), nil
	}, Name: func(value *TrunkFindRecord) string { return findText(value, "name") }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	return value, wrap(errors.Join(err, guard(ctx)))
}
