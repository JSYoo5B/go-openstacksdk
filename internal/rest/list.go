package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// PagePolicy selects continuation representations supported by a service.
// Nil LinkKeys uses links and <plural>_links; empty NextKey uses next.
// HTTPLink and marker fallback are opt-in. Fallback activates only with an
// explicit limit and derives the wire marker with Marker, never a generic
// model identifier. Unbounded requests rely on server continuation links.
type PagePolicy[T any] struct {
	LinkKeys       []string
	NextKey        string
	HTTPLink       bool
	MarkerFallback bool
	Marker         func(*T) (string, error)
	// MarkerOnShortPage follows every nonempty page when an explicit limit is
	// present. Python Resource uses this rule; other APIs may stop when their
	// returned page is shorter than the limit. An empty page always stops.
	MarkerOnShortPage bool
	// AllowFirstLimitReduction permits only the first continuation to reduce
	// the requested limit. That reduced limit is fixed for all later pages.
	AllowFirstLimitReduction bool
	// MaxItemsLimitHint allows a controlled Collection iteration to supply its
	// max-items cap as a wire limit when the caller did not specify one.
	MaxItemsLimitHint bool
	// StopOnEmptyPage ignores continuations on an empty page. The default keeps
	// supporting services which advertise a next link from an empty page.
	StopOnEmptyPage bool
	// OffsetPagination permits advertised, strictly increasing offset cursors
	// instead of markers. An omitted initial offset means zero. The first next
	// link may introduce a positive server limit when none was requested; that
	// limit then stays fixed. Marker input and marker fallback are unsupported.
	OffsetPagination bool
}

type continuationRules struct {
	reduceLimit, offsetPagination, firstPage bool
}

// ListControl limits raw, successfully decoded and validated rows before any
// caller-side local filtering. MaxItems zero is unbounded. LimitHint is opt-in
// because many collections do not support a limit query at all.
type ListControl struct {
	MaxItems   int
	SinglePage bool
	LimitHint  bool
}

func queryCopy(query url.Values) url.Values {
	copy := make(url.Values, len(query))
	for key, values := range query {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}

func pageKey(u *url.URL) string {
	copy := *u
	copy.Scheme, copy.Host = strings.ToLower(copy.Scheme), strings.ToLower(copy.Host)
	copy.RawQuery = copy.Query().Encode()
	return copy.String()
}

func paginationInput(query url.Values) (int, error) {
	if values, exists := query["marker"]; exists && (len(values) != 1 || strings.TrimSpace(values[0]) == "") {
		return 0, fmt.Errorf("%w: pagination requires one nonempty marker", resource.ErrInvalidOption)
	}
	values, exists := query["limit"]
	if !exists {
		return 0, nil
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("%w: pagination requires one limit", resource.ErrInvalidOption)
	}
	limit, err := strconv.Atoi(values[0])
	if err != nil || limit < 1 {
		return 0, fmt.Errorf("%w: pagination limit must be positive", resource.ErrInvalidOption)
	}
	return limit, nil
}

func paginationOffset(query url.Values) (uint64, error) {
	if _, exists := query["marker"]; exists {
		return 0, fmt.Errorf("%w: offset pagination does not accept a marker", resource.ErrInvalidOption)
	}
	values, exists := query["offset"]
	if !exists {
		return 0, nil
	}
	if len(values) != 1 || values[0] == "" {
		return 0, fmt.Errorf("%w: pagination requires one nonnegative offset", resource.ErrInvalidOption)
	}
	for _, ch := range values[0] {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("%w: pagination offset must be a nonnegative decimal integer", resource.ErrInvalidOption)
		}
	}
	offset, err := strconv.ParseUint(values[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: pagination offset is out of range", resource.ErrInvalidOption)
	}
	return offset, nil
}

// List streams raw-decoded objects lazily. Continuations stay on the collection
// path, inherit omitted initial filters, and cannot replace or add filters.
// Every page revalidates the source and uses an independently guarded request.
func List[T any](ctx context.Context, spec CollectionSpec[T], query url.Values) iter.Seq2[*T, error] {
	return ListWithControl(ctx, spec, query, ListControl{})
}

// ListWithControl stops before decoding unconsumed rows or interpreting next
// links after its cap. SinglePage still validates every consumed first-page row.
// Each iteration owns its count, query snapshot and continuation guards.
func ListWithControl[T any](ctx context.Context, spec CollectionSpec[T], query url.Values, control ListControl) iter.Seq2[*T, error] {
	initial := queryCopy(query)
	if control.LimitHint && control.MaxItems > 0 && !initial.Has("limit") {
		initial.Set("limit", strconv.Itoa(control.MaxItems))
	}
	return func(yield func(*T, error) bool) {
		fail := func(err error) { yield(nil, err) }
		if control.MaxItems < 0 {
			fail(fmt.Errorf("%w: max items must be non-negative", resource.ErrInvalidOption))
			return
		}
		if err := spec.validate(ctx); err != nil {
			fail(err)
			return
		}
		if spec.ValidateInitialQuery != nil {
			if err := spec.ValidateInitialQuery(ctx, queryCopy(initial)); err != nil {
				fail(err)
				return
			}
		}
		if spec.PluralKey == "" || spec.Metadata == nil {
			fail(fmt.Errorf("%w: list envelope and metadata are required", resource.ErrInvalidOption))
			return
		}
		limit, err := paginationInput(initial)
		if err != nil {
			fail(err)
			return
		}
		if spec.Paging.MarkerFallback && spec.Paging.Marker == nil {
			fail(fmt.Errorf("%w: marker fallback requires a wire marker callback", resource.ErrInvalidOption))
			return
		}
		if spec.Paging.OffsetPagination {
			if spec.Paging.MarkerFallback {
				fail(fmt.Errorf("%w: offset pagination cannot use marker fallback", resource.ErrInvalidOption))
				return
			}
			if _, err := paginationOffset(initial); err != nil {
				fail(err)
				return
			}
		}
		base, err := url.Parse(spec.Client.ServiceURL(spec.Path))
		if err != nil {
			fail(err)
			return
		}
		base.RawQuery = initial.Encode()
		if err := ValidateTarget(spec.Client, base.String()); err != nil {
			fail(err)
			return
		}
		current := base
		visited := map[string]bool{pageKey(base): true}
		markers := map[string]bool{}
		if marker := initial.Get("marker"); marker != "" {
			markers[marker] = true
		}
		consumed := 0
		for pageNumber := 0; ; pageNumber++ {
			if pageNumber > 0 {
				if err := spec.validate(ctx); err != nil {
					fail(err)
					return
				}
			}
			if spec.ValidateQuery != nil {
				if err := spec.ValidateQuery(ctx, queryCopy(current.Query())); err != nil {
					fail(err)
					return
				}
			}
			response, err := DoJSONGuarded(ctx, spec.Client, spec.SourceGuard, http.MethodGet, current.String(), nil, nil, successCodes(spec.ListCodes, http.StatusOK)...)
			if err != nil {
				fail(err)
				return
			}
			if spec.ValidateResponse != nil {
				if err := spec.ValidateResponse(response); err != nil {
					fail(response.Fail(err))
					return
				}
			}
			// An explicitly accepted 204 has no collection representation.
			// Read/Close/source failures are already retained by DoJSONGuarded.
			if response.StatusCode == http.StatusNoContent {
				if err := ctx.Err(); err != nil {
					fail(response.Fail(err))
				}
				return
			}
			fields, items, err := pageItems(response, spec.PluralKey)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			for _, item := range items {
				if err := ctx.Err(); err != nil {
					fail(response.Fail(err))
					return
				}
				value, err := decodeListItem(item, response, spec.Metadata)
				if err != nil {
					fail(response.Fail(err))
					return
				}
				if spec.ValidateItem != nil {
					if err := spec.ValidateItem(value); err != nil {
						fail(response.Fail(err))
						return
					}
				}
				consumed++
				if !yield(value, nil) {
					return
				}
				if control.MaxItems > 0 && consumed >= control.MaxItems {
					if err := ctx.Err(); err != nil {
						fail(response.Fail(err))
					}
					return
				}
			}
			if err := ctx.Err(); err != nil {
				fail(response.Fail(err))
				return
			}
			if control.SinglePage || (spec.Paging.StopOnEmptyPage && len(items) == 0) {
				return
			}
			rules := continuationRules{reduceLimit: pageNumber == 0 && spec.Paging.AllowFirstLimitReduction,
				offsetPagination: spec.Paging.OffsetPagination, firstPage: pageNumber == 0}
			next, err := continuation(fields, response.Header, spec.PluralKey, spec.Paging, base, current, rules)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			if next == nil && spec.Paging.MarkerFallback && limit > 0 && len(items) > 0 && (spec.Paging.MarkerOnShortPage || len(items) >= limit) {
				// Derive the marker from retained wire data. The consumer owns
				// yielded models and may have changed their fields or metadata.
				last, err := decodeListItem(items[len(items)-1], response, spec.Metadata)
				if err != nil {
					fail(response.Fail(err))
					return
				}
				marker, err := spec.Paging.Marker(last)
				if err != nil {
					fail(response.Fail(err))
					return
				}
				if strings.TrimSpace(marker) == "" {
					fail(response.Fail(fmt.Errorf("%w: empty wire pagination marker", resource.ErrInvalidOption)))
					return
				}
				copy := *current
				query := current.Query()
				query.Set("marker", marker)
				copy.RawQuery = query.Encode()
				next = &copy
			}
			if next == nil {
				return
			}
			if err := lockContinuation(base, current, next, rules); err != nil {
				fail(response.Fail(err))
				return
			}
			key, marker := pageKey(next), next.Query().Get("marker")
			if visited[key] || (marker != "" && markers[marker]) {
				fail(response.Fail(&resource.PaginationCycleError{URL: next.String()}))
				return
			}
			visited[key] = true
			if marker != "" {
				markers[marker] = true
			}
			limit, _ = paginationInput(next.Query())
			current = next
		}
	}
}

func decodeListItem[T any](item json.RawMessage, response *Response, metadata func(*T) *resource.Metadata) (*T, error) {
	var value T
	if err := resource.DecodeObject(item, &value, metadata(&value)); err != nil {
		return nil, err
	}
	meta := metadata(&value)
	if meta == nil {
		return nil, fmt.Errorf("%w: model metadata is required", resource.ErrInvalidOption)
	}
	meta.Header, meta.StatusCode = response.Header.Clone(), response.StatusCode
	return &value, nil
}

func pageItems(response *Response, plural string) (map[string]json.RawMessage, []json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, nil, err
	}
	if fields == nil {
		return nil, nil, fmt.Errorf("list response must be a JSON object")
	}
	raw := bytes.TrimSpace(fields[plural])
	if len(raw) == 0 || raw[0] != '[' {
		return nil, nil, fmt.Errorf("list response requires %q array", plural)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, nil, err
	}
	return fields, items, nil
}

func continuation[T any](fields map[string]json.RawMessage, headers http.Header, plural string, policy PagePolicy[T], base, current *url.URL, rules continuationRules) (*url.URL, error) {
	keys := policy.LinkKeys
	if keys == nil {
		keys = []string{"links", plural + "_links"}
	}
	var links []string
	for _, key := range keys {
		if raw, exists := fields[key]; exists {
			var values []resource.Link
			if err := json.Unmarshal(raw, &values); err != nil {
				return nil, err
			}
			for _, link := range values {
				if strings.EqualFold(link.Rel, "next") && link.Href != "" {
					links = append(links, link.Href)
				}
			}
		}
	}
	key := policy.NextKey
	if key == "" {
		key = "next"
	}
	if raw, exists := fields[key]; exists {
		var next string
		if err := json.Unmarshal(raw, &next); err != nil {
			return nil, err
		}
		if next != "" {
			links = append(links, next)
		}
	}
	if policy.HTTPLink {
		for _, header := range headers.Values("Link") {
			values, err := headerNextLinks(header)
			if err != nil {
				return nil, err
			}
			links = append(links, values...)
		}
	}
	var result *url.URL
	for _, link := range links {
		parsed, err := url.Parse(link)
		if err != nil {
			return nil, err
		}
		next := current.ResolveReference(parsed)
		if err := lockContinuation(base, current, next, rules); err != nil {
			return nil, err
		}
		if result != nil && pageKey(result) != pageKey(next) {
			return nil, fmt.Errorf("%w: conflicting pagination links", resource.ErrInvalidOption)
		}
		result = next
	}
	return result, nil
}

func lockContinuation(base, current, next *url.URL, rules continuationRules) error {
	if next.User != nil || next.Opaque != "" || next.Fragment != "" || !strings.EqualFold(next.Scheme, base.Scheme) || !strings.EqualFold(next.Host, base.Host) || next.EscapedPath() != base.EscapedPath() {
		return fmt.Errorf("%w: pagination changes collection origin or path", resource.ErrInvalidOption)
	}
	query, err := url.ParseQuery(next.RawQuery)
	if err != nil {
		return fmt.Errorf("%w: invalid pagination query: %v", resource.ErrInvalidOption, err)
	}
	previous := current.Query()
	for key, values := range query {
		if key == "marker" && !rules.offsetPagination || key == "offset" && rules.offsetPagination {
			continue
		}
		old, exists := previous[key]
		if key == "limit" && rules.offsetPagination && rules.firstPage && !exists {
			if _, err := paginationInput(url.Values{"limit": values}); err == nil {
				continue
			}
		}
		if key == "limit" && rules.reduceLimit && exists {
			newLimit, err := paginationInput(url.Values{"limit": values})
			oldLimit, oldErr := paginationInput(url.Values{"limit": old})
			if err == nil && oldErr == nil && newLimit <= oldLimit {
				continue
			}
		}
		if !exists || !reflect.DeepEqual(values, old) {
			return fmt.Errorf("%w: pagination changes filter %q", resource.ErrInvalidOption, key)
		}
	}
	for key, values := range previous {
		if key != "marker" && !query.Has(key) {
			query[key] = append([]string(nil), values...)
		}
	}
	// A link which omits the marker retains the previous marker. It cannot
	// silently restart at the first page; the cycle guard rejects repetition.
	if !query.Has("marker") && previous.Has("marker") {
		query["marker"] = append([]string(nil), previous["marker"]...)
	}
	if _, err := paginationInput(query); err != nil {
		return err
	}
	if rules.offsetPagination {
		oldOffset, err := paginationOffset(previous)
		if err != nil {
			return err
		}
		newOffset, err := paginationOffset(query)
		if err != nil {
			return err
		}
		if newOffset == oldOffset {
			return &resource.PaginationCycleError{URL: next.String()}
		}
		if newOffset < oldOffset {
			return fmt.Errorf("%w: pagination offset moves backwards", resource.ErrInvalidOption)
		}
	}
	next.RawQuery = query.Encode()
	return nil
}

func headerNextLinks(header string) ([]string, error) {
	var parts []string
	start, quoted, escaped, angle := 0, false, false, false
	for i, ch := range header {
		if escaped {
			escaped = false
			continue
		}
		if quoted && ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			quoted = !quoted
		}
		if !quoted {
			if ch == '<' {
				angle = true
			}
			if ch == '>' {
				angle = false
			}
			if ch == ',' && !angle {
				parts = append(parts, header[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, header[start:])
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		end := strings.IndexByte(part, '>')
		if !strings.HasPrefix(part, "<") || end < 1 {
			return nil, fmt.Errorf("invalid HTTP pagination Link header")
		}
		_, params, err := mime.ParseMediaType("application/link" + part[end+1:])
		if err != nil {
			return nil, err
		}
		for _, rel := range strings.Fields(params["rel"]) {
			if strings.EqualFold(rel, "next") {
				result = append(result, part[1:end])
				break
			}
		}
	}
	return result, nil
}
