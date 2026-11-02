package images

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TaskRecordsOpts owns the inherited list conveniences. The pinned image_tasks
// proxy itself accepts only its image selector; these are explicit Go options.
type TaskRecordsOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
	Limit        *int
	Marker       string
	MaxItems     int
	Paginated    *bool
	order        []string
}

// TaskRecordsOption is one option of TaskRecords and AllTaskRecords:
// WithTaskRecordsOpts, WithTaskRecordsLocation, request.Query (the value is
// kept as JSON in caller order), request.Headers (replaces the headers),
// request.Microversion, request.Limit, request.Marker, request.MaxItems,
// request.Paginated or request.Custom over *TaskRecordsOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type TaskRecordsOption = request.ListCall

var taskRecordsCalls = calls.Spec[TaskRecordsOpts]{
	Headers:      calls.Value(func(c *TaskRecordsOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *TaskRecordsOpts) **string { return &c.Microversion }),
	Query: func(c *TaskRecordsOpts, entry request.CallEntry) error {
		if entry.JSONErr != nil {
			return entry.JSONErr
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if _, ok := c.Query[entry.Key]; !ok {
			c.order = append(c.order, entry.Key)
		}
		c.Query[entry.Key] = bytes.Clone(entry.JSON)
		return nil
	},
	Paging: calls.Paging[TaskRecordsOpts]{
		Limit:     calls.Pointer(func(c *TaskRecordsOpts) **int { return &c.Limit }),
		Marker:    calls.Value(func(c *TaskRecordsOpts) *string { return &c.Marker }),
		MaxItems:  calls.Value(func(c *TaskRecordsOpts) *int { return &c.MaxItems }),
		Paginated: calls.Pointer(func(c *TaskRecordsOpts) **bool { return &c.Paginated }),
	},
}

func taskRecordCopy(v TaskRecordsOpts) TaskRecordsOpts {
	v.Query = taskRecordClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	v.order = slices.Clone(v.order)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	return v
}
func WithTaskRecordsOpts(v TaskRecordsOpts) TaskRecordsOption {
	owned := taskRecordCopy(v)
	return request.Custom(func(c *TaskRecordsOpts) error { *c = taskRecordCopy(owned); return nil })
}
func WithTaskRecordsLocation(v resource.CloudLocation) TaskRecordsOption {
	owned := v.Clone()
	return request.Custom(func(c *TaskRecordsOpts) error { x := owned.Clone(); c.Location = &x; return nil })
}
func taskRecordKeys(v TaskRecordsOpts) []string {
	keys := make([]string, 0, len(v.Query))
	seen := make(map[string]bool)
	for _, key := range v.order {
		if _, ok := v.Query[key]; ok && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	var missing []string
	for key := range v.Query {
		if !seen[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return append(keys, missing...)
}
func taskRecordText(value string) bool {
	return utf8.ValidString(value) && strings.IndexFunc(value, unicode.IsControl) < 0
}
func taskRecordIdentity(value string) error {
	if !taskRecordText(value) || strings.TrimSpace(value) == "" || value == "." || value == ".." {
		return taskRecordInvalid("image selector must be a nonblank literal UTF8 string without controls")
	}
	return nil
}
func taskRecordQueryKey(value string) error {
	if !taskRecordText(value) || strings.TrimSpace(value) == "" {
		return taskRecordInvalid("query key must be nonblank UTF8 without controls")
	}
	switch value {
	case "image_id", "cls", "session", "self", "connection", "_synchronized", "base_path", "allow_unknown_params", "paginated", "microversion", "headers", "max_items", "location", "resource_type", "__conflicting_attrs":
		return taskRecordInvalid("query %q is a fixed parent/Source constructor/control; use owned options", value)
	}
	return nil
}
func taskRecordPrepare(v TaskRecordsOpts, imageID string) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query, filters := make(url.Values), make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, control, taskRecordInvalid("list limit/cap must be nonnegative")
	}
	filters["image_id"], _ = json.Marshal(imageID)
	for _, key := range taskRecordKeys(v) {
		raw := v.Query[key]
		if err := taskRecordQueryKey(key); err != nil {
			return nil, nil, control, err
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, control, taskRecordInvalid("query %q requires UTF8 JSON", key)
		}
		switch key {
		case "limit", "marker":
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, control, err
			}
			if len(values) > 0 {
				query[key] = slices.Clone(values)
			}
		default:
			if taskRecordField(key) {
				filters[key] = bytes.Clone(raw)
			}
		}
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		query.Set("marker", v.Marker)
	}
	if query.Has("limit") {
		values := query["limit"]
		if len(values) != 1 {
			return nil, nil, control, taskRecordInvalid("limit requires one nonnegative integer")
		}
		number, err := strconv.Atoi(values[0])
		if err != nil || number < 0 {
			return nil, nil, control, taskRecordInvalid("limit requires one nonnegative integer")
		}
	}
	if query.Has("marker") {
		values := query["marker"]
		if len(values) != 1 || !taskRecordText(values[0]) || strings.TrimSpace(values[0]) == "" {
			return nil, nil, control, taskRecordInvalid("marker requires one nonblank UTF8 value")
		}
	}
	return query, filters, control, nil
}
