package subnets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// subnetListProfile reuses the pinned Subnet Body/Query declarations of the owned
// find record. Source list adds no name hint and classifies kwargs exactly.
func subnetListProfile() neutronfind.Profile {
	profile := recordFindProfile
	profile.Body = slices.Clone(recordFindProfile.Body)
	profile.Headers = slices.Clone(recordFindProfile.Headers)
	profile.Query = slices.Clone(recordFindProfile.Query)
	profile.NameQueryHint = false
	profile.SourceFindPolicy, profile.StopFaultRetryCallbacks = true, true
	profile.PythonLoadedEquality = true
	profile.SourceStatusPolicy, profile.SourceLinkPolicy = true, true
	return profile
}

// RecordListOpts owns JSON Source list kwargs and transport conveniences.
// Query retains omission/null/false/zero. Microversion applies to every page.
// Default pagination is true; max_items 0 is unbounded and limit is omitted.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	BasePath     *string
	expression   json.RawMessage
	order        []string
}
type RecordListOption func(*RecordListOpts) error

func recordListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
}
func copyRecordListOpts(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(recordListOptions(v))
	v.Query, v.Headers, v.Microversion, v.BasePath, v.order = owned.Query, owned.Headers, owned.Microversion, owned.BasePath, owned.Order
	v.expression = bytes.Clone(v.expression)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRecordListOpts(v)
	return func(c *RecordListOpts) error { *c = copyRecordListOpts(owned); return nil }
}

// recordListIgnores reports kwargs that Source Resource.list discards before
// value evaluation: neither a declared query name/wire nor a Body attribute.
func recordListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return false
	}
	switch key {
	case "cls", "paginated", "max_items", "base_path", "resource_type", "session", "microversion", "headers", "self", "__conflicting_attrs", "jmespath_filters":
		return false
	}
	for _, pair := range recordFindProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range recordFindProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

func recordListControlError(key string) error {
	switch key {
	case "cls", "resource_type", "session", "self":
		return findInvalid("%q is a Resource/Proxy control; use the dedicated owned options for headers and microversion", key)
	}
	return nil
}

// WithRecordListQuery snapshots known kwargs in insertion order; unknowns are
// discarded before encoding their transport value, matching Source list.
// Declared query names and wire names go to the server; other declared Body
// attributes become local filters on each received row. Source runtime controls
// fail by presence before invoking an application JSON marshaler.
func WithRecordListQuery(key string, value any) RecordListOption {
	if recordListIgnores(key) {
		return func(*RecordListOpts) error { return nil }
	}
	if controlErr := recordListControlError(key); controlErr != nil {
		return func(*RecordListOpts) error { return controlErr }
	}
	var raw json.RawMessage
	var err error
	if key == "jmespath_filters" {
		switch v := value.(type) {
		case string:
			raw, err = json.Marshal(v)
		case json.RawMessage:
			raw = bytes.Clone(v)
		default:
			raw = json.RawMessage("null") // Source ignores every non-string, even truthy values.
		}
	} else {
		raw, err = json.Marshal(value)
	}
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}

		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error {
		c.Headers = maps.Clone(owned)
		delete(c.Query, "headers")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "headers" })
		return nil
	}
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error {
		x := v
		c.Microversion = &x
		delete(c.Query, "microversion")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "microversion" })
		return nil
	}
}

// WithRecordListBasePath uses a service-relative collection route while
// retaining source, origin and continuation guards. An explicit empty path yields no HTTP pages; null selects the default.
func WithRecordListBasePath(path string) RecordListOption {
	return func(c *RecordListOpts) error {
		value := path
		c.BasePath = &value
		delete(c.Query, "base_path")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "base_path" })
		return nil
	}
}

// WithRecordListConflictingAttrs restores Source kwargs once, preserving JSON
// object order. Duplicate bound paginated/base_path arguments remain errors.
func WithRecordListConflictingAttrs(value json.RawMessage) RecordListOption {
	return WithRecordListQuery("__conflicting_attrs", value)
}

func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMarker(v string) RecordListOption  { return WithRecordListQuery("marker", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }

// WithRecordListJMESPath evaluates the deprecated Proxy expression through
// ProxyListRecords. Empty expressions leave the lazy resource generator intact.
func WithRecordListJMESPath(expression string) RecordListOption {
	return WithRecordListQuery("jmespath_filters", expression)
}

func (a *API) subnetListEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("configured subnets API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, subnetListProfile(), func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected subnets API client changed")
		}
		return nil
	})
}
func subnetListRecord(value *neutronfind.Record) *SubnetRecord {
	if value == nil {
		return nil
	}
	return &SubnetRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
}
func applyRecordListOptions(ctx context.Context, engine *neutronfind.Engine, options []RecordListOption) (RecordListOpts, error) {
	policy := RecordListOpts{}
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, findInvalid("nil subnets list option")
		}
		next := copyRecordListOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = copyRecordListOpts(next)
	}
	normalized, expression, err := subnetProxyArguments(recordListOptions(policy))
	if err != nil {
		return policy, err
	}
	policy.Query, policy.Headers, policy.Microversion, policy.BasePath, policy.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.BasePath, normalized.Order
	policy.expression = bytes.Clone(expression)
	return copyRecordListOpts(policy), nil
}

// recordListExpression removes the Proxy argument before Resource.list sees
// kwargs. Source evaluates only a truthy Python string; non-strings are ignored.
func recordListExpression(policy *RecordListOpts) (string, error) {
	raw := bytes.Clone(policy.expression)
	if raw == nil {
		return "", nil
	}
	policy.expression = nil
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return "", errors.Join(findInvalid("jmespath_filters must contain complete UTF-8 JSON"), err)
	}
	if !truthy {
		return "", nil
	}
	var expression string
	if err := json.Unmarshal(raw, &expression); err != nil {
		return "", nil
	}
	return expression, nil
}

// ListRecords is the lazy Source-shaped subnets list (Proxy.subnets). It has no
// implicit name hint or member fetch. Declared QueryParameters go to the
// server, other declared Body attributes filter received rows locally, and
// subnets_links, top-level next, HTTP Link and limit/marker continuation follow
// the pinned Resource.list paging.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*SubnetRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*SubnetRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListRecords", "subnets", cloudread.ContextError(ctx, err))
		}
		engine, err := a.subnetListEngine(ctx)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		policy, err := applyRecordListOptions(ctx, engine, snapshot)
		if err == nil {
			var expression string
			expression, err = recordListExpression(&policy)
			if err == nil && expression != "" {
				err = findInvalid("JMESPath may return arbitrary values; use ProxyListRecords")
			}
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		for value, err := range subnetListIterator(ctx, engine, policy) {
			if !yield(subnetListRecord(value), wrap(err)) {
				return
			}
		}
	}
}

// AllRecords discards partial collected values on error and returns a nonnil
// empty slice on success. Iterator consumers keep already yielded records.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*SubnetRecord, error) {
	values := make([]*SubnetRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func subnetRemoveArgument(v *neutronfind.Options, key string) {
	delete(v.Query, key)
	v.Order = slices.DeleteFunc(v.Order, func(k string) bool { return k == key })
}

// subnetProxyArguments performs only the eager Proxy binding/restoration pass.
// Resource.list is a generator, so its controls remain raw until iteration.
func subnetProxyArguments(v neutronfind.Options) (neutronfind.Options, json.RawMessage, error) {
	v = neutronfind.CloneOptions(v)
	for _, key := range []string{"self", "resource_type", "cls", "session"} {
		if _, ok := v.Query[key]; ok {
			return v, nil, findInvalid("%s collides with a bound Proxy/Resource list argument", key)
		}
	}
	expression := bytes.Clone(v.Query["jmespath_filters"])
	subnetRemoveArgument(&v, "jmespath_filters")
	if raw, ok := v.Query["__conflicting_attrs"]; ok {
		truthy, err := cloudfilter.PythonJSONTruthy(raw)
		if err != nil {
			return v, expression, err
		}
		if truthy {
			members, err := cloudfilter.ObjectMembers(raw)
			if err != nil {
				return v, expression, findInvalid("truthy __conflicting_attrs must be an object")
			}
			if v.Query == nil {
				v.Query = make(map[string]json.RawMessage)
			}
			for _, member := range members {
				if member.Key == "paginated" || member.Key == "base_path" || member.Key == "cls" || member.Key == "session" {
					return v, expression, findInvalid("restored %s duplicates a bound Resource.list argument", member.Key)
				}
				if !slices.Contains(v.Order, member.Key) {
					v.Order = append(v.Order, member.Key)
				}
				v.Query[member.Key] = bytes.Clone(member.Value)
			}
		}
		subnetRemoveArgument(&v, "__conflicting_attrs")
	}
	// Restored self/resource_type and expression are unknown Resource kwargs.
	for _, key := range []string{"self", "resource_type", "jmespath_filters"} {
		subnetRemoveArgument(&v, key)
	}
	return v, expression, nil
}

// subnetVersion preserves KSA's JSON-domain call override normalization. The
// resource Body never grows a list microversion field. None selects the captured
// default, other falsey values disable it, and truthy numeric/iterable versions
// normalize through the pinned discover.normalize_version_number convention.
func subnetVersion(raw json.RawMessage) (*string, error) {
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	text := ""
	if !truthy {
		return &text, nil
	}
	token := bytes.TrimSpace(raw)
	switch token[0] {
	case '"':
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
		text = strings.TrimLeft(text, "v")
	case 't':
		text = "1.0"
	case '[':
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		parts := make([]string, 0, len(values))
		for _, item := range values {
			part, err := cloudfilter.PythonString(item)
			if err != nil {
				return nil, err
			}
			parts = append(parts, part)
		}
		text = strings.Join(parts, ".")
	case '{':
		members, err := cloudfilter.ObjectMembers(raw)
		if err != nil {
			return nil, err
		}
		parts := make([]string, 0, len(members))
		for _, item := range members {
			parts = append(parts, item.Key)
		}
		text = strings.Join(parts, ".")
	default:
		floating, err := jsonfilter.DescriptorFloatJSON(raw)
		if err != nil {
			return nil, err
		}
		if !bytes.ContainsAny(floating, ".eE") {
			floating = append(floating, []byte(".0")...)
		}
		text, err = cloudfilter.PythonString(floating)
		if err != nil {
			return nil, err
		}
		if !strings.ContainsAny(text, ".eE") {
			text += ".0"
		}
	}
	parts := strings.Split(text, ".")
	if len(parts) == 1 {
		if parts[0] == "latest" {
			parts = append(parts, "latest")
		} else {
			parts = append(parts, "0")
		}
	}
	for i, part := range parts {
		if part == "latest" {
			continue
		}
		integer, err := jsonfilter.PythonIntegerString(part)
		if err != nil {
			return nil, findInvalid("invalid microversion component %q", part)
		}
		parts[i] = integer.String()
	}
	if parts[0] != "latest" && slices.Contains(parts[1:], "latest") {
		return nil, findInvalid("major.latest microversion is not allowed")
	}
	if slices.IndexFunc(parts, func(p string) bool { return p != "latest" }) < 0 {
		text = "latest"
	} else {
		text = strings.Join(parts, ".")
	}
	return &text, nil
}
func subnetResourceArguments(v neutronfind.Options) (neutronfind.Options, error) {
	v = neutronfind.CloneOptions(v)
	if raw, ok := v.Query["microversion"]; ok {
		version, err := subnetVersion(raw)
		if err != nil {
			return v, err
		}
		v.Microversion = version
		subnetRemoveArgument(&v, "microversion")
	} else if v.Microversion != nil {
		raw, _ := json.Marshal(*v.Microversion)
		version, err := subnetVersion(raw)
		if err != nil {
			return v, err
		}
		v.Microversion = version
	}
	normalized, _, err := neutronfind.NormalizeProxyListOptions(v)
	return normalized, err
}
func subnetListIterator(ctx context.Context, engine *neutronfind.Engine, policy RecordListOpts) iter.Seq2[*neutronfind.Record, error] {
	return func(yield func(*neutronfind.Record, error) bool) {
		if err := engine.Check(ctx); err != nil {
			yield(nil, err)
			return
		}
		prepared, err := subnetResourceArguments(recordListOptions(policy))
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			yield(nil, err)
			return
		}
		if prepared.BasePath != nil && *prepared.BasePath == "" {
			return
		}
		for value, err := range engine.ListWithSourceControls(ctx, prepared) {
			if !yield(value, err) {
				return
			}
		}
	}
}

// RecordListGenerator retains Source Resource.list as an opaque lazy object.
type RecordListGenerator struct {
	Records iter.Seq2[*SubnetRecord, error]
}

// JMESPathIdentity exposes the same opaque object to the shared evaluator.
func (g *RecordListGenerator) JMESPathIdentity() any { return g }

// MarshalJSON rejects the opaque generator without consuming it. Python's
// to_string(@) also fails when asked to JSON-serialize Resource.list.
func (*RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, errors.New("a lazy subnet record generator is not JSON")
}

// RecordProxyListResult keeps the actual Proxy result. Value is either the
// opaque generator, arbitrary JSON, or a JSON-shaped value containing that
// same generator. JSON is populated only for completely serializable results.
// A null expression result has Value nil and JSON "null".
type RecordProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

func subnetProxyHasGenerator(value any) bool {
	switch v := value.(type) {
	case *RecordListGenerator:
		return true
	case []any:
		for _, item := range v {
			if subnetProxyHasGenerator(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range v {
			if subnetProxyHasGenerator(item) {
				return true
			}
		}
	}
	return false
}

// ProxyListRecords preserves Proxy.subnets(**query), including the deprecated
// JMESPath branch. Resource.list is a generator, not a collected JSON array:
// projection/index/type expressions can return null without HTTP; @ returns
// the generator; length(@) can fail before the first request. Use ListRecords
// or AllRecords for the ordinary resource-only Go convenience.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ProxyListRecords", "subnets", cloudread.ContextError(ctx, err))
	}
	engine, err := a.subnetListEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := applyRecordListOptions(ctx, engine, slices.Clone(options))
	if err != nil {
		return nil, wrap(err)
	}
	expression, err := recordListExpression(&policy)
	if err != nil {
		return nil, wrap(err)
	}
	generator := &RecordListGenerator{Records: func(yield func(*SubnetRecord, error) bool) {
		for value, err := range subnetListIterator(ctx, engine, policy) {
			if !yield(subnetListRecord(value), wrap(err)) {
				return
			}
		}
	}}
	result := &RecordProxyListResult{Value: generator}
	if expression != "" {
		result.Value, err = jmespath.Search(expression, generator)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	if !subnetProxyHasGenerator(result.Value) {
		result.JSON, err = json.Marshal(result.Value)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	return result, wrap(engine.Check(ctx))
}
