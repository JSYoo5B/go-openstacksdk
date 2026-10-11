package consistencygroups

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// RecordGetOpts owns the explicit Go request policy for literal member GET.
type RecordGetOpts struct {
	Microversion *string
	Headers      map[string]string
}
type RecordGetOption func(*RecordGetOpts) error

func copyGet(v RecordGetOpts) RecordGetOpts {
	v.Headers = recordHeaders(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordGetOpts(v RecordGetOpts) RecordGetOption {
	owned := copyGet(v)
	return func(c *RecordGetOpts) error { *c = copyGet(owned); return nil }
}
func WithRecordGetMicroversion(v string) RecordGetOption {
	return func(c *RecordGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordGetHeaders(v map[string]string) RecordGetOption {
	owned := recordHeaders(v)
	return func(c *RecordGetOpts) error { c.Headers = recordHeaders(owned); return nil }
}

func recordHeaders(v map[string]string) map[string]string {
	owned := maps.Clone(v)
	if owned == nil {
		owned = make(map[string]string)
	}
	return owned
}

// RecordCreateOpts uses known Source attributes; unknown attrs are discarded.
// Null, zero, false and empty values are distinct from absent map members.
type RecordCreateOpts struct {
	Fields       map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
}
type RecordCreateOption func(*RecordCreateOpts) error

func copyCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Fields = recordClone(v.Fields)
	v.Headers = recordHeaders(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := copyCreate(v)
	return func(c *RecordCreateOpts) error { *c = copyCreate(owned); return nil }
}
func WithRecordCreateField(key string, value any) RecordCreateOption {
	if recordReserved(key) {
		return func(*RecordCreateOpts) error { return recordInvalid("Resource control %q is SDK owned", key) }
	}
	if !recordKnown(key) {
		return func(*RecordCreateOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordCreateOpts) error {
		if err != nil {
			return err
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordCreateMicroversion(v string) RecordCreateOption {
	return func(c *RecordCreateOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordCreateHeaders(v map[string]string) RecordCreateOption {
	owned := recordHeaders(v)
	return func(c *RecordCreateOpts) error { c.Headers = recordHeaders(owned); return nil }
}

// RecordUpdateOpts is an owned partial update. Empty, unknown-only and ID-only
// updates return a local seed without discovery or mutation HTTP.
type RecordUpdateOpts struct {
	Fields       map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
}
type RecordUpdateOption func(*RecordUpdateOpts) error

func copyUpdate(v RecordUpdateOpts) RecordUpdateOpts {
	v.Fields = recordClone(v.Fields)
	v.Headers = recordHeaders(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordUpdateOpts(v RecordUpdateOpts) RecordUpdateOption {
	owned := copyUpdate(v)
	return func(c *RecordUpdateOpts) error { *c = copyUpdate(owned); return nil }
}
func WithRecordUpdateField(key string, value any) RecordUpdateOption {
	if recordReserved(key) {
		return func(*RecordUpdateOpts) error { return recordInvalid("Resource control %q is SDK owned", key) }
	}
	if !recordKnown(key) {
		return func(*RecordUpdateOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordUpdateOpts) error {
		if err != nil {
			return err
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordUpdateMicroversion(v string) RecordUpdateOption {
	return func(c *RecordUpdateOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordUpdateHeaders(v map[string]string) RecordUpdateOption {
	owned := recordHeaders(v)
	return func(c *RecordUpdateOpts) error { c.Headers = recordHeaders(owned); return nil }
}

// RecordSourceOpts defaults all four create-from-source fields to null.
// Source identifiers are literal IDs; empty IDs follow Source's falsey→null.
type RecordSourceOpts struct {
	ConsistencyGroupSnapshotID, ConsistencyGroupID, Name, Description *string
	Microversion                                                      *string
	Headers                                                           map[string]string
}
type RecordSourceOption func(*RecordSourceOpts) error

func recordString(v *string) *string {
	if v == nil {
		return nil
	}
	x := *v
	return &x
}
func copySource(v RecordSourceOpts) RecordSourceOpts {
	v.ConsistencyGroupSnapshotID = recordString(v.ConsistencyGroupSnapshotID)
	v.ConsistencyGroupID = recordString(v.ConsistencyGroupID)
	v.Name = recordString(v.Name)
	v.Description = recordString(v.Description)
	v.Microversion = recordString(v.Microversion)
	v.Headers = recordHeaders(v.Headers)
	return v
}
func WithRecordSourceOpts(v RecordSourceOpts) RecordSourceOption {
	owned := copySource(v)
	return func(c *RecordSourceOpts) error { *c = copySource(owned); return nil }
}
func WithRecordSourceConsistencyGroupSnapshotID(v string) RecordSourceOption {
	return func(c *RecordSourceOpts) error { x := v; c.ConsistencyGroupSnapshotID = &x; return nil }
}
func WithRecordSourceConsistencyGroupID(v string) RecordSourceOption {
	return func(c *RecordSourceOpts) error { x := v; c.ConsistencyGroupID = &x; return nil }
}
func WithRecordSourceName(v string) RecordSourceOption {
	return func(c *RecordSourceOpts) error { x := v; c.Name = &x; return nil }
}
func WithRecordSourceDescription(v string) RecordSourceOption {
	return func(c *RecordSourceOpts) error { x := v; c.Description = &x; return nil }
}
func WithRecordSourceMicroversion(v string) RecordSourceOption {
	return func(c *RecordSourceOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordSourceHeaders(v map[string]string) RecordSourceOption {
	owned := recordHeaders(v)
	return func(c *RecordSourceOpts) error { c.Headers = recordHeaders(owned); return nil }
}

// RecordDeleteOpts sends POST /{id}/delete with force:false by default.
// PinnedSourceWire replays the known Source spelling rejected by stock Cinder.
type RecordDeleteOpts struct {
	Force            *bool
	PinnedSourceWire bool
	Microversion     *string
	Headers          map[string]string
}
type RecordDeleteOption func(*RecordDeleteOpts) error

func copyDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = recordHeaders(v.Headers)
	v.Microversion = recordString(v.Microversion)
	if v.Force != nil {
		x := *v.Force
		v.Force = &x
	}
	return v
}
func WithRecordDeleteOpts(v RecordDeleteOpts) RecordDeleteOption {
	owned := copyDelete(v)
	return func(c *RecordDeleteOpts) error { *c = copyDelete(owned); return nil }
}
func WithRecordDeleteForce(v bool) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { x := v; c.Force = &x; return nil }
}
func WithRecordDeletePinnedSourceWire(v bool) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { c.PinnedSourceWire = v; return nil }
}
func WithRecordDeleteMicroversion(v string) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordDeleteHeaders(v map[string]string) RecordDeleteOption {
	owned := recordHeaders(v)
	return func(c *RecordDeleteOpts) error { c.Headers = recordHeaders(owned); return nil }
}

// RecordListOpts keeps details=True, declared server aliases, local Body filters
// and raw-row paging limits. Unknown query kwargs do not become wire extensions.
type RecordListOpts struct {
	Details      *bool
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Paginated    *bool
	Limit        *int
	Marker       string
	MaxItems     int
}
type RecordListOption func(*RecordListOpts) error

func copyList(v RecordListOpts) RecordListOpts {
	v.Query = recordClone(v.Query)
	v.Headers = recordHeaders(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
	}
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyList(v)
	return func(c *RecordListOpts) error { *c = copyList(owned); return nil }
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := recordHeaders(v)
	return func(c *RecordListOpts) error { c.Headers = recordHeaders(owned); return nil }
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordListDetails(v bool) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Details = &x; return nil }
}
func WithRecordListPaginated(v bool) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Paginated = &x; return nil }
}
func WithRecordListLimit(v int) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Limit = &x; return nil }
}
func WithRecordListMarker(v string) RecordListOption {
	return func(c *RecordListOpts) error { c.Marker = v; return nil }
}
func WithRecordListMaxItems(v int) RecordListOption {
	return func(c *RecordListOpts) error { c.MaxItems = v; return nil }
}
func recordServer(key string) bool {
	switch key {
	case "limit", "marker", "offset", "sort_dir", "sort_key", "sort":
		return true
	}
	return false
}

// Raw list kwargs cannot override named Source/Proxy controls. Use the owned
// typed options instead; legacy JMESPath/dynamic conflict injection is SDK-J1.
func recordListReserved(key string) bool {
	if recordReserved(key) {
		return true
	}
	switch key {
	case "session", "headers", "paginated", "max_items", "allow_unknown_params", "details", "jmespath_filters", "__conflicting_attrs", "list_base_path":
		return true
	}
	return false
}
func WithRecordListQuery(key string, value any) RecordListOption {
	if recordListReserved(key) {
		return func(*RecordListOpts) error { return recordInvalid("Resource control %q is SDK owned", key) }
	}
	if !recordServer(key) && !recordKnown(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
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
func prepareList(v RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query, filters := make(url.Values), make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, control, recordInvalid("negative limit/cap")
	}
	for key, raw := range v.Query {
		if recordListReserved(key) {
			return nil, nil, control, recordInvalid("Resource control %q is SDK owned", key)
		}
		if !recordServer(key) && !recordKnown(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, control, recordInvalid("query %q must be UTF-8 JSON", key)
		}
		if recordKnown(key) {
			filters[key] = bytes.Clone(raw)
		}
	}
	for _, key := range []string{"limit", "marker", "offset", "sort_dir", "sort_key", "sort"} {
		raw, supplied := v.Query[key]
		wire := key
		if !supplied {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, control, err
		}
		if len(values) > 0 {
			query[wire] = slices.Clone(values)
		}
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, control, recordInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	if query.Has("limit") {
		values := query["limit"]
		n, err := strconv.Atoi(query.Get("limit"))
		if len(values) != 1 || err != nil || n < 0 {
			return nil, nil, control, recordInvalid("limit must be one nonnegative integer")
		}
	}
	if query.Has("marker") && (len(query["marker"]) != 1 || query.Get("marker") == "") {
		return nil, nil, control, recordInvalid("marker must be one nonempty value")
	}
	if query.Has("offset") && !query.Has("marker") {
		_, err := strconv.ParseUint(query.Get("offset"), 10, 64)
		if len(query["offset"]) != 1 || err != nil {
			return nil, nil, control, recordInvalid("offset must be one nonnegative integer")
		}
	}
	return query, filters, control, nil
}
func recordMatch(actual, filter json.RawMessage) (bool, error) {
	if len(actual) == 0 {
		actual = json.RawMessage("null")
	}
	if bytes.HasPrefix(bytes.TrimSpace(filter), []byte("{")) {
		truthy, err := cloudfilter.PythonTruthy(actual)
		if err != nil || !truthy {
			return false, err
		}
		wanted, err := cloudfilter.ObjectMembers(filter)
		if err != nil {
			return false, err
		}
		if len(wanted) == 0 {
			return true, nil
		}
		observed, err := cloudfilter.ObjectMembers(actual)
		if err != nil {
			return false, recordInvalid("truthy consistency group filter value has no mapping get")
		}
		fields := make(map[string]json.RawMessage, len(observed))
		for _, member := range observed {
			fields[member.Key] = member.Value
		}
		for _, member := range wanted {
			observed := fields[member.Key]
			if len(observed) == 0 {
				observed = json.RawMessage("null")
			}
			match, err := jsonfilter.EqualPythonJSON(observed, member.Value)
			if err != nil || !match {
				return match, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}
