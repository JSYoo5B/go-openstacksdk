// Package ironicrest holds the shared request engine of the SDK-owned Ironic
// and ironic-inspector collection APIs (deploy templates, runbooks, inspection
// rules and introspection rules). It never retries or replays a request.
package ironicrest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Response is one accepted HTTP response. Body is nil for bodyless statuses.
type Response struct {
	Body   []byte
	Header http.Header
	Status int
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Preflight rejects a nil context, client or canceled context before any HTTP.
func Preflight(ctx context.Context, c *gophercloud.ServiceClient) error {
	if ctx == nil {
		return invalid("context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil {
		return invalid("service client must not be nil")
	}
	return nil
}

// ValidateID rejects identifiers that cannot be one safe path segment.
func ValidateID(kind, id string) error {
	switch {
	case id == "":
		return invalid("%s ID or name must not be empty", kind)
	case id == "." || id == "..":
		return invalid("%s ID or name %q is not a valid path segment", kind, id)
	case strings.TrimSpace(id) != id:
		return invalid("%s ID or name must not have surrounding whitespace", kind)
	}
	return nil
}

// Send performs exactly one request and accepts only the listed statuses. The
// response body is read completely; a read failure is reported with the
// accepted status and headers so a possibly applied mutation stays visible.
func Send(ctx context.Context, c *gophercloud.ServiceClient, method, target string, body any, headers map[string]string, ok ...int) (*Response, error) {
	if err := Preflight(ctx, c); err != nil {
		return nil, err
	}
	opts := &gophercloud.RequestOpts{OkCodes: ok, KeepResponseBody: true, MoreHeaders: map[string]string{}}
	for k, v := range headers {
		opts.MoreHeaders[k] = v
	}
	if body != nil {
		opts.JSONBody = body
	}
	resp, err := c.Request(ctx, method, target, opts)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, err
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(resp.Body)
	out := &Response{Body: raw, Header: resp.Header.Clone(), Status: resp.StatusCode}
	if readErr != nil {
		return out, &resource.ResponseError{Body: raw, Header: out.Header, StatusCode: out.Status, Cause: readErr}
	}
	return out, nil
}

// IsNotFound reports a native HTTP 404.
func IsNotFound(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusNotFound) }

// Decode turns an accepted response into a model. Any failure keeps the
// accepted evidence in a resource.ResponseError.
func Decode[T any](resp *Response, decode func(json.RawMessage, *Response) (*T, error)) (*T, error) {
	value, err := decode(resp.Body, resp)
	if err != nil {
		return nil, &resource.ResponseError{Body: resp.Body, Header: resp.Header, StatusCode: resp.Status, Cause: err}
	}
	return value, nil
}

// Fields returns the CSV value of a fields query, rejecting empty names.
func Fields(fields []string) (string, error) {
	for _, f := range fields {
		if f == "" || strings.ContainsAny(f, ", \t\r\n") {
			return "", invalid("field name %q must be non-empty without commas or spaces", f)
		}
	}
	return strings.Join(fields, ","), nil
}

// ListConfig carries the server-side query and client-side paging controls.
type ListConfig struct {
	Query     url.Values
	Headers   map[string]string
	Paginated bool
	MaxItems  int
	// NoLimitHint keeps MaxItems purely client-side: no limit query is sent.
	NoLimitHint bool
}

// Row is one list element with the evidence of the page that carried it.
type Row struct {
	Raw  json.RawMessage
	Page *Response
}

// List walks a collection. It follows a next link from the body when one is
// published (only its query is used, against the same collection route), and
// otherwise, when an explicit limit was given, continues with marker=<last
// uuid> until an empty page, the way the Python list generator does.
func List(ctx context.Context, c *gophercloud.ServiceClient, collection, listKey string, cfg ListConfig) iter.Seq2[Row, error] {
	return func(yield func(Row, error) bool) {
		if err := Preflight(ctx, c); err != nil {
			yield(Row{}, err)
			return
		}
		if cfg.MaxItems < 0 {
			yield(Row{}, invalid("max items must not be negative"))
			return
		}
		base := c.ServiceURL(collection)
		query := url.Values{}
		for k, v := range cfg.Query {
			query[k] = append([]string(nil), v...)
		}
		if cfg.MaxItems > 0 && !cfg.NoLimitHint && query.Get("limit") == "" {
			query.Set("limit", strconv.Itoa(cfg.MaxItems))
		}
		limit := query.Get("limit")
		yielded := 0
		for {
			target := base
			if len(query) > 0 {
				target += "?" + query.Encode()
			}
			resp, err := Send(ctx, c, http.MethodGet, target, nil, cfg.Headers, http.StatusOK)
			if err != nil {
				yield(Row{}, err)
				return
			}
			rows, next, err := parsePage(resp.Body, listKey)
			if err != nil {
				yield(Row{}, &resource.ResponseError{Body: resp.Body, Header: resp.Header, StatusCode: resp.Status, Cause: err})
				return
			}
			lastMarker := query.Get("marker")
			query.Del("marker")
			query.Del("limit")
			marker := ""
			for _, raw := range rows {
				if !yield(Row{Raw: raw, Page: resp}, nil) {
					return
				}
				yielded++
				marker = rowUUID(raw)
				if cfg.MaxItems > 0 && yielded >= cfg.MaxItems {
					return
				}
			}
			if len(rows) == 0 || !cfg.Paginated {
				return
			}
			var params url.Values
			if next != "" {
				params, err = nextQuery(next, collection)
				if err != nil {
					yield(Row{}, &resource.ResponseError{Body: resp.Body, Header: resp.Header, StatusCode: resp.Status, Cause: err})
					return
				}
			} else if limit != "" {
				params = url.Values{"marker": {marker}, "limit": {limit}}
			} else {
				return
			}
			if m, has := params["marker"]; has && (len(m) == 0 || m[0] == "" || m[0] == lastMarker) {
				yield(Row{}, &resource.PaginationCycleError{URL: target})
				return
			}
			for k, v := range params {
				query[k] = v
			}
		}
	}
}

func rowUUID(raw json.RawMessage) string {
	var row struct {
		UUID string `json:"uuid"`
	}
	_ = json.Unmarshal(raw, &row)
	return row.UUID
}

func parsePage(body []byte, listKey string) ([]json.RawMessage, string, error) {
	var page map[string]json.RawMessage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, "", fmt.Errorf("list response must be a JSON object: %w", err)
	}
	rawRows, ok := page[listKey]
	if !ok {
		return nil, "", fmt.Errorf("list response has no %q array", listKey)
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(rawRows, &rows); err != nil || rows == nil {
		return nil, "", fmt.Errorf("list response %q must be an array", listKey)
	}
	for _, row := range rows {
		if t := strings.TrimSpace(string(row)); !strings.HasPrefix(t, "{") {
			return nil, "", fmt.Errorf("list response %q elements must be objects", listKey)
		}
	}
	next := ""
	if raw, ok := page["next"]; ok {
		var s *string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, "", fmt.Errorf("list response next link must be a string")
		}
		if s != nil {
			next = *s
		}
	}
	if next == "" {
		if raw, ok := page["links"]; ok {
			var links []struct {
				Rel  string `json:"rel"`
				Href string `json:"href"`
			}
			if json.Unmarshal(raw, &links) == nil {
				for _, l := range links {
					if l.Rel == "next" && l.Href != "" {
						next = l.Href
						break
					}
				}
			}
		}
	}
	return rows, next, nil
}

// nextQuery accepts only a continuation of the same collection and returns its
// query. A host in the link is never contacted.
func nextQuery(link, collection string) (url.Values, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("next link is not a URL: %w", err)
	}
	if !strings.HasSuffix(strings.TrimRight(u.Path, "/"), "/"+collection) {
		return nil, errors.New("next link does not continue the same collection")
	}
	return u.Query(), nil
}
