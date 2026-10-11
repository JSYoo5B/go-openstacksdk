package volumetypes

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
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// VolumeTypeRecord owns the declared Source view and original response evidence.
type VolumeTypeRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (v *VolumeTypeRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return json.Unmarshal(data, v.Wire)
}
func findMetadata(v *VolumeTypeRecord) *resource.Metadata {
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

var findBodyFields = []string{"id", "name", "description", "extra_specs", "is_public"}

func findBodyKnown(key string) bool { return slices.Contains(findBodyFields, key) }
func findProject(value *VolumeTypeRecord, seed map[string]json.RawMessage) error {
	body := findFields(seed)
	if value.Wire != nil {
		for _, key := range findBodyFields {
			wireKey := key
			if key == "is_public" {
				wireKey = "os-volume-type-access:is_public"
			}
			if raw, exists := value.Wire.Body[wireKey]; exists {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	for _, key := range findBodyFields {
		if _, exists := body[key]; !exists {
			body[key] = json.RawMessage("null")
		}
	}
	if raw, exists := body["is_public"]; exists {
		normalized, err := jsonfilter.BooleanJSON(raw)
		if err != nil {
			return err
		}
		body["is_public"] = normalized
	}
	if raw := bytes.TrimSpace(body["extra_specs"]); len(raw) > 0 && !bytes.Equal(raw, []byte("null")) && raw[0] != '{' {
		body["extra_specs"] = json.RawMessage("{}")
	}
	body["location"] = json.RawMessage("null")
	value.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	return nil
}
func findText(value *VolumeTypeRecord, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Resource.Body[key], &text)
	return text
}
func findMarker(value *VolumeTypeRecord) (string, error) {
	text := findText(value, "id")
	if text == "" {
		return "", findInvalid("list marker must be a nonempty string")
	}
	return text, nil
}
func findResponse(response *rest.Response, seed map[string]json.RawMessage) (*VolumeTypeRecord, error) {
	if response == nil {
		return nil, nil
	}
	value := &VolumeTypeRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
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
	if nested, exists := envelope["volume_type"]; exists {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return value, err
	}
	wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
	value.Wire = wire.Clone()
	return value, findProject(value, seed)
}

// RecordFindOpts owns Source kwargs. Omitted IgnoreMissing defaults to true.
// Query preserves JSON values, including arrays and null; unknown kwargs go to
// the member GET but are ignored by the list, following Source Resource.find.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Query         map[string]json.RawMessage
	Headers       map[string]string
}
type RecordFindOption func(*RecordFindOpts) error

func copyFindOpts(value RecordFindOpts) RecordFindOpts {
	value.Query = findFields(value.Query)
	value.Headers = maps.Clone(value.Headers)
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
		return nil
	}
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
	if key == "id" {
		return findInvalid("id kwargs collide with the find identity")
	}
	return nil
}
func prepareFindQuery(policy RecordFindOpts) (url.Values, url.Values, map[string]json.RawMessage, map[string]json.RawMessage, rest.ListControl, error) {
	getQuery, listQuery := make(url.Values), make(url.Values)
	filters, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	control := rest.ListControl{LimitHint: true}
	for key, raw := range policy.Query {
		if err := findKey(key); err != nil {
			return nil, nil, nil, nil, control, err
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, nil, nil, control, findInvalid("query %q must contain complete UTF-8 JSON", key)
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		if len(values) != 0 {
			getQuery[key] = values
		}
		if findBodyKnown(key) {
			seed[key] = bytes.Clone(raw)
		}
		switch key {
		case "is_public", "limit", "marker", "offset", "sort_dir", "sort_key", "sort", "all_projects":
			if len(values) != 0 {
				wireKey := key
				if key == "all_projects" {
					wireKey = "all_tenants"
				}
				listQuery[wireKey] = slices.Clone(values)
			}
		case "max_items":
			var amount int
			if err := json.Unmarshal(raw, &amount); err != nil || amount < 0 {
				return nil, nil, nil, nil, control, findInvalid("max_items must be a nonnegative Go integer")
			}
			control.MaxItems = amount
		case "paginated":
			var enabled bool
			if err := json.Unmarshal(raw, &enabled); err != nil {
				return nil, nil, nil, nil, control, findInvalid("paginated must be a boolean")
			}
			control.SinglePage = !enabled
		default:
			if findBodyKnown(key) {
				filters[key] = bytes.Clone(raw)
			}
		}
	}
	getQuery.Set("is_public", "none")
	listQuery.Set("is_public", "none")
	return getQuery, listQuery, filters, seed, control, nil
}
func findLocalMatch(actual, filter json.RawMessage) (bool, error) {
	if len(actual) == 0 {
		actual = json.RawMessage("null")
	}
	trim := bytes.TrimSpace(filter)
	if len(trim) > 0 && trim[0] == '{' {
		var wanted, observed map[string]json.RawMessage
		if err := json.Unmarshal(trim, &wanted); err != nil {
			return false, err
		}
		if err := json.Unmarshal(actual, &observed); err != nil {
			return false, nil
		}
		if len(observed) == 0 {
			return false, nil
		}
		for key, value := range wanted {
			matched, err := findLocalMatch(observed[key], value)
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
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*VolumeTypeRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "volumetypes", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("volumetypes API is required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = findInvalid("selected volumetypes client changed")
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
	getQuery, listQuery, filters, seed, control, err := prepareFindQuery(policy)
	if err == nil {
		err = errors.Join(source.WithPolicy(ctx, nil, policy.Headers), guard(ctx))
	}
	if err != nil {
		return nil, wrap(err)
	}
	get := func(_ context.Context, id string) (*VolumeTypeRecord, error) {
		target := source.Client.ServiceURL("types", url.PathEscape(id))
		if len(getQuery) != 0 {
			target += "?" + getQuery.Encode()
		}
		response, err := rest.DoJSONGuardedRejections(ctx, &source.Client, guard, http.MethodGet, target, nil, nil, rest.RejectionPolicy{Codes: []int{400, 403, 404}, PreserveCleanRetry: true}, findCodes()...)
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
	list := func(context.Context, url.Values) iter.Seq2[*VolumeTypeRecord, error] {
		return func(yield func(*VolumeTypeRecord, error) bool) {
			var origin *rest.Response
			spec := rest.CollectionSpec[VolumeTypeRecord]{Client: &source.Client, Path: "types", Kind: "volumetypes", PluralKey: "volume_types", Metadata: findMetadata, Validate: guard, SourceGuard: guard, ListCodes: findCodes(), ValidateResponse: func(response *rest.Response) error { origin = response; return guard(ctx) }, ValidateItem: func(value *VolumeTypeRecord) error {
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				return errors.Join(findProject(value, nil), guard(ctx))
			}, Paging: rest.PagePolicy[VolumeTypeRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: findMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
			for value, err := range rest.ListWithControl(ctx, spec, listQuery, control) {
				if err != nil {
					yield(nil, err)
					return
				}
				matched := true
				for key, filter := range filters {
					equal, cause := findLocalMatch(value.Resource.Body[key], filter)
					if cause = errors.Join(cause, guard(ctx)); cause != nil {
						yield(nil, origin.Fail(cause))
						return
					}
					if !equal {
						matched = false
					}
				}
				if matched && !yield(value, nil) {
					return
				}
			}
		}
	}
	collection := resource.NewCollection(resource.Adapter[VolumeTypeRecord]{Kind: "volumetypes", IdentityFind: true, ValidateID: findSegment, IdentityDirectGet: findSegment, IdentityResponseID: func(value *VolumeTypeRecord) (string, error) {
		id := findText(value, "id")
		if id == "" {
			return "", findInvalid("find response identity must be a nonempty string")
		}
		return id, nil
	}, Name: func(value *VolumeTypeRecord) string { return findText(value, "name") }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	return value, wrap(errors.Join(err, guard(ctx)))
}
