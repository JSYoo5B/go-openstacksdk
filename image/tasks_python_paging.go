package image

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

// taskBodyFilterFields maps each pinned Python openstack/image/v2/task.py Body
// attribute that Resource.list treats as a client-side filter (a Body
// attribute that is not a query parameter) to its response field. type and
// status are query parameters there and remain WithListTasksType/Status.
var taskBodyFilterFields = map[string]string{
	"id": "id", "name": "name", "created_at": "created_at", "updated_at": "updated_at",
	"expires_at": "expires_at", "input": "input", "message": "message",
	"owner_id": "owner", "result": "result", "schema": "schema",
}

// WithListTasksBodyFilter adds a local filter like the client-side Body
// filters of pinned Python Resource.list. name is a Python Task attribute name
// (id, name, created_at, updated_at, expires_at, input, message, owner_id,
// result or schema; owner_id compares the response owner field). value is
// encoded to JSON when the option is created; encoding errors and unknown
// names surface when the option is applied, before HTTP. A row is yielded
// only when every filter matches: object filters match a subset of a nonempty
// response object, other values compare exactly and a missing field compares
// as null (internal/jsonfilter.MatchFilters, the shared Body-filter policy).
// Filters never reach the wire. Repeating a name replaces its value.
//
// As in Python, WithListTasksMaxItems counts raw rows before filtering, and
// rows excluded by a filter still advance marker fallback.
func WithListTasksBodyFilter(name string, value any) ListTasksOption {
	encoded, err := json.Marshal(value)
	return func(config *ListTasksOpts) error {
		if err != nil {
			return uploadInvalid("task body filter %q: %v", name, err)
		}
		if _, known := taskBodyFilterFields[name]; !known {
			return uploadInvalid("unknown task body filter %q", name)
		}
		if config.BodyFilters == nil {
			config.BodyFilters = make(map[string]json.RawMessage)
		}
		config.BodyFilters[name] = append(json.RawMessage(nil), encoded...)
		return nil
	}
}

// WithListTasksMarkerFallback enables pinned Python Resource.list's final
// continuation rule: when a nonempty page has no usable next link and an
// explicit positive WithListTasksLimit was given, request the same collection
// again with the initial filters, the same limit and marker set to the last
// raw row's id. Iteration ends at the first empty page, so a fully consumed
// collection costs one extra request, exactly as in Python. A repeated marker
// fails as a pagination cycle; a last row without a string id fails. Disabled
// by default, and ignored without a positive limit, with SinglePage, or after
// a MaxItems cap or caller break.
func WithListTasksMarkerFallback(enabled bool) ListTasksOption {
	return func(config *ListTasksOpts) error { config.MarkerFallback = enabled; return nil }
}

// WithListTasksLinkHeader lets an RFC 8288 HTTP Link header with rel="next"
// continue the listing when the response body has no next field, as pinned
// Python Resource.list does. The header target passes the same guard as a
// body next link (same origin and collection path, unchanged filters and one
// nonempty marker). A body next link always takes precedence. Disabled by
// default; malformed Link headers fail only when the option is enabled.
func WithListTasksLinkHeader(enabled bool) ListTasksOption {
	return func(config *ListTasksOpts) error { config.LinkHeader = enabled; return nil }
}

func validateTaskBodyFilters(filters map[string]json.RawMessage) error {
	for name, raw := range filters {
		if _, known := taskBodyFilterFields[name]; !known {
			return uploadInvalid("unknown task body filter %q", name)
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return uploadInvalid("task body filter %q must be one complete JSON value", name)
		}
	}
	return nil
}

// taskWireFilters translates Python attribute names to response fields.
func taskWireFilters(filters map[string]json.RawMessage) map[string]json.RawMessage {
	if len(filters) == 0 {
		return nil
	}
	wire := make(map[string]json.RawMessage, len(filters))
	for name, raw := range filters {
		wire[taskBodyFilterFields[name]] = raw
	}
	return wire
}

func matchTaskBodyFilters(value *TaskInfo, filters map[string]json.RawMessage) (bool, error) {
	if len(filters) == 0 {
		return true, nil
	}
	return jsonfilter.MatchFilters(value.Body, filters)
}

// taskLinkHeaderNext returns the first rel="next" target of the Link headers.
func taskLinkHeaderNext(header http.Header) (string, error) {
	for _, line := range header.Values("Link") {
		for _, part := range splitTaskLinkHeader(line) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			end := strings.IndexByte(part, '>')
			if !strings.HasPrefix(part, "<") || end < 1 {
				return "", uploadInvalid("invalid task Link header")
			}
			_, params, err := mime.ParseMediaType("application/links" + part[end+1:])
			if err != nil {
				return "", uploadInvalid("invalid task Link header: %v", err)
			}
			for _, rel := range strings.Fields(params["rel"]) {
				if strings.EqualFold(rel, "next") {
					return part[1:end], nil
				}
			}
		}
	}
	return "", nil
}

func splitTaskLinkHeader(header string) []string {
	var parts []string
	start := 0
	quoted, angle, escaped := false, false, false
	for i, c := range header {
		switch {
		case escaped:
			escaped = false
		case quoted && c == '\\':
			escaped = true
		case c == '"':
			quoted = !quoted
		case !quoted && c == '<':
			angle = true
		case !quoted && c == '>':
			angle = false
		case !quoted && !angle && c == ',':
			parts = append(parts, header[start:i])
			start = i + 1
		}
	}
	return append(parts, header[start:])
}

// taskFallbackPage builds Python's limit/marker retry against the fixed base
// collection: initial filters and limit are kept and marker is replaced.
func taskFallbackPage(base *url.URL, initial url.Values, lastID *string) (*url.URL, error) {
	if lastID == nil || *lastID == "" {
		return nil, uploadInvalid("task marker fallback requires the last row to have a nonempty string id")
	}
	if err := taskQueryText(*lastID); err != nil {
		return nil, err
	}
	query := make(url.Values, len(initial)+1)
	for key, values := range initial {
		query[key] = append([]string(nil), values...)
	}
	query.Set("marker", *lastID)
	next := *base
	next.RawQuery = query.Encode()
	return &next, nil
}

func taskPositiveLimit(initial url.Values) bool {
	limit := initial.Get("limit")
	return limit != "" && limit != "0"
}
