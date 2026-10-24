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

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PagePolicy selects continuation representations supported by a service.
// Nil LinkKeys uses links and <plural>_links; empty NextKey uses next.
// HTTPLink and marker fallback are opt-in. Fallback activates only with an
// explicit limit and derives the wire marker with Marker, never a generic
// model identifier. Unbounded requests rely on server continuation links.
type PagePolicy[T any] struct {
	LinkKeys []string
	NextKey  string
	// IgnoreFalseyNext ignores falsey JSON values in the top-level next field
	// before decoding a URL. Truthy nonstrings still fail. This opt-in matches
	// Resource pagination while retaining strict defaults for other services.
	IgnoreFalseyNext bool
	// NextTruthy optionally supplies a library-owned JSON truthiness policy.
	// It is used only with IgnoreFalseyNext and receives an independent copy.
	NextTruthy func(json.RawMessage) (bool, error)
	// VersionedPath accepts one exact escaped relative version-prefixed path
	// as an alias of the fixed collection, including behind reverse proxies.
	// Other origins and paths retain the normal continuation checks.
	VersionedPath  string
	HTTPLink       bool
	MarkerFallback bool
	Marker         func(*T) (string, error)
	// DictionaryLinks also accepts Keystone's links:{next:...} shape. This is
	// an explicit service compatibility option; other link formats stay strict.
	DictionaryLinks bool
	// SourceLinkPolicy selects Resource._get_next_link JSON semantics instead
	// of dictionary interoperability: present root links suppresses plural
	// links, dictionaries are ignored, and the first literal next/href item
	// wins before the top-level next fallback. HTTPLink keeps its own policy.
	// The zero value preserves all existing formats and conflict checks.
	SourceLinkPolicy bool
	// MarkerOnShortPage follows every nonempty page when an explicit limit is
	// present. Python Resource uses this rule; other APIs may stop when their
	// returned page is shorter than the limit. An empty page always stops.
	MarkerOnShortPage bool
	// AllowFirstLimitReduction permits only the first continuation to reduce
	// the requested limit. That reduced limit is fixed for all later pages.
	AllowFirstLimitReduction bool
	// AllowFirstServerLimit permits the first continuation to introduce one
	// positive limit when the initial query had none. Later pages retain that
	// limit, and a server-only limit does not enable marker fallback.
	AllowFirstServerLimit bool
	// AllowZeroLimit accepts an explicit zero wire limit. Zero does not enable
	// marker fallback; a positive MaxItems hint may replace this falsey limit.
	// Services retain positive-only limits unless they opt in.
	AllowZeroLimit bool
	// MaxItemsLimitHint allows a controlled Collection iteration to supply its
	// max-items cap as a wire limit when the caller did not specify one.
	MaxItemsLimitHint bool
	// StopOnEmptyPage ignores continuations on an empty page. The default keeps
	// supporting services which advertise a next link from an empty page.
	StopOnEmptyPage bool
	// SingletonObject permits a plural envelope containing one object instead
	// of an array. Scalars and null still fail with the original page receipt.
	SingletonObject bool
	// DecodeNoContent requires a representation even for an accepted 204. The
	// default preserves collections which treat 204 as an empty result.
	DecodeNoContent bool
	// OffsetPagination permits advertised, strictly increasing offset cursors
	// instead of markers. An omitted initial offset means zero. The first next
	// link may introduce a positive server limit when none was requested; that
	// limit then stays fixed. Marker input and marker fallback are unsupported.
	OffsetPagination bool
	// MarkerOrOffsetPagination permits one cursor family per request, including
	// marker fallback after advertised offset pages. Cursor switches discard the
	// previous family; offsets must exceed every earlier offset in the stream.
	// It is independent of and cannot be combined with OffsetPagination.
	MarkerOrOffsetPagination bool
}

type continuationRules struct {
	reduceLimit, offsetPagination, firstPage bool
	allowFirstServerLimit, allowZeroLimit    bool
	sourceRawLimit                           bool
	markerOrOffsetPagination                 bool
	offsetFloor                              uint64
}

// ListControl limits raw, successfully decoded and validated rows before any
// caller-side local filtering. MaxItems zero is unbounded. LimitHint is opt-in
// because many collections do not support a limit query at all.
type ListControl struct {
	MaxItems   int
	SinglePage bool
	LimitHint  bool
	// BeforeItem is an opt-in raw-row gate, before decoding or projection.
	BeforeItem func(context.Context, int) (bool, error)
	// SourceRawLimit keeps finite Source limit hints on the wire.
	SourceRawLimit bool
	// SourceLimitFalsey, with SourceRawLimit, records that the caller found
	// the raw Source limit Python-falsey (for example false or 0.0). Python
	// keeps that original limit for the whole loop and then never performs
	// the limit-driven marker GET, whatever spelling the wire carries.
	SourceLimitFalsey bool
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

func paginationInput(query url.Values, allowZero ...bool) (int, error) {
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
	if err != nil || limit < 0 || (limit == 0 && (len(allowZero) == 0 || !allowZero[0])) {
		return 0, fmt.Errorf("%w: invalid pagination limit", resource.ErrInvalidOption)
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
	zeroLimit := spec.Paging.AllowZeroLimit && len(initial["limit"]) == 1 && initial.Get("limit") == "0"
	if control.LimitHint && control.MaxItems > 0 && (!initial.Has("limit") || zeroLimit) {
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
		limit, err := controlledPaginationInput(initial, spec.Paging.AllowZeroLimit, control.SourceRawLimit)
		if err != nil {
			fail(err)
			return
		}
		if control.SourceRawLimit && control.SourceLimitFalsey {
			limit = 0
		}
		markerFallback := spec.Paging.MarkerFallback && (!spec.Paging.AllowFirstServerLimit || limit > 0)
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
		var offsetFloor uint64
		if spec.Paging.MarkerOrOffsetPagination {
			if spec.Paging.OffsetPagination {
				fail(fmt.Errorf("%w: select one offset pagination policy", resource.ErrInvalidOption))
				return
			}
			if initial.Has("offset") {
				offsetFloor, err = paginationOffset(initial)
				if err != nil {
					fail(err)
					return
				}
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
			codes := successCodes(spec.ListCodes, http.StatusOK)
			var response *Response
			var err error
			if spec.ReadPage != nil {
				response, err = spec.ReadPage(ctx, current.String(), codes...)
			} else {
				response, err = DoJSONGuarded(ctx, spec.Client, spec.SourceGuard, http.MethodGet, current.String(), nil, nil, codes...)
			}
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
			if response.StatusCode == http.StatusNoContent && !spec.Paging.DecodeNoContent {
				if err := ctx.Err(); err != nil {
					fail(response.Fail(err))
				}
				return
			}
			fields, items, err := pageItems(response, spec.PluralKey, spec.Paging.SingletonObject)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			for _, item := range items {
				if err := ctx.Err(); err != nil {
					fail(response.Fail(err))
					return
				}
				if control.BeforeItem != nil {
					stop, err := control.BeforeItem(ctx, consumed)
					if err != nil {
						fail(response.Fail(err))
						return
					}
					if stop {
						return
					}
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
				offsetPagination: spec.Paging.OffsetPagination, firstPage: pageNumber == 0,
				allowFirstServerLimit: spec.Paging.AllowFirstServerLimit, allowZeroLimit: spec.Paging.AllowZeroLimit, sourceRawLimit: control.SourceRawLimit,
				markerOrOffsetPagination: spec.Paging.MarkerOrOffsetPagination, offsetFloor: offsetFloor}
			next, err := continuation(fields, response.Header, spec.PluralKey, spec.Paging, base, current, rules)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			if next == nil && markerFallback && limit > 0 && len(items) > 0 && (spec.Paging.MarkerOnShortPage || len(items) >= limit) {
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
				if spec.Paging.MarkerOrOffsetPagination {
					query.Del("offset")
				}
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
			if spec.Paging.MarkerOrOffsetPagination && next.Query().Has("offset") {
				offsetFloor, _ = paginationOffset(next.Query())
			}
			limit, _ = controlledPaginationInput(next.Query(), spec.Paging.AllowZeroLimit, control.SourceRawLimit)
			if control.SourceRawLimit && control.SourceLimitFalsey {
				limit = 0
			}
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

func pageItems(response *Response, plural string, singleton bool) (map[string]json.RawMessage, []json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, nil, err
	}
	if fields == nil {
		return nil, nil, fmt.Errorf("list response must be a JSON object")
	}
	raw := bytes.TrimSpace(fields[plural])
	if singleton && len(raw) != 0 && raw[0] == '{' {
		return fields, []json.RawMessage{bytes.Clone(raw)}, nil
	}
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
	if policy.SourceLinkPolicy {
		link, err := sourceJSONContinuation(fields, keys, policy.NextKey, policy.NextTruthy)
		if err != nil {
			return nil, err
		}
		if link != "" {
			links = append(links, link)
		}
	} else {
		for _, key := range keys {
			if raw, exists := fields[key]; exists {
				if policy.DictionaryLinks && len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '{' {
					var values map[string]json.RawMessage
					if err := json.Unmarshal(raw, &values); err != nil {
						return nil, err
					}
					if value, exists := values["next"]; exists {
						var next string
						if err := json.Unmarshal(value, &next); err != nil {
							return nil, err
						}
						if next != "" {
							links = append(links, next)
						}
					}
					continue
				}
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
			decode := true
			if policy.IgnoreFalseyNext {
				truthy := cloudfilter.PythonTruthy
				if policy.NextTruthy != nil {
					truthy = policy.NextTruthy
				}
				keep, err := truthy(bytes.Clone(raw))
				if err != nil {
					return nil, err
				}
				decode = keep
			}
			if decode {
				var next string
				if err := json.Unmarshal(raw, &next); err != nil {
					return nil, err
				}
				if next != "" {
					links = append(links, next)
				}
			}
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
		if policy.VersionedPath != "" && parsed.Scheme == "" && parsed.Host == "" && parsed.EscapedPath() == policy.VersionedPath {
			parsed.Path, parsed.RawPath = base.Path, base.RawPath
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

// sourceJSONContinuation interprets finite JSON exactly as Resource's body
// link selection. Scope/version/header interoperability stays with continuation.
func sourceJSONContinuation(fields map[string]json.RawMessage, keys []string, nextKey string, truthy func(json.RawMessage) (bool, error)) (string, error) {
	if truthy == nil {
		truthy = cloudfilter.PythonJSONTruthy
	}
	var selected json.RawMessage
	for _, key := range keys {
		if raw, present := fields[key]; present {
			selected = raw
			break
		}
	}
	var next json.RawMessage
	if selected != nil {
		raw := bytes.TrimSpace(selected)
		if len(raw) == 0 || !json.Valid(raw) {
			return "", fmt.Errorf("Source links require complete JSON")
		}
		switch raw[0] {
		case '{':
			// Source yields {key:value} independently; no such item can contain both
			// rel and href, so even {next:url} is ignored without decoding its value.
		case '[':
			var items []json.RawMessage
			if err := json.Unmarshal(raw, &items); err != nil {
				return "", err
			}
			for _, item := range items {
				item = bytes.TrimSpace(item)
				if len(item) == 0 || item[0] != '{' {
					return "", fmt.Errorf("Source link item has no mapping get")
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(item, &fields); err != nil {
					return "", err
				}
				var rel string
				if err := json.Unmarshal(fields["rel"], &rel); err != nil || rel != "next" {
					continue
				}
				if href, present := fields["href"]; present {
					next = bytes.Clone(href)
					break
				}
			}
		case '"':
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return "", err
			}
			if text != "" {
				return "", fmt.Errorf("Source string link item has no mapping get")
			}
		default:
			return "", fmt.Errorf("Source links are not iterable")
		}
	}
	keep, err := truthy(bytes.Clone(next))
	if err != nil {
		return "", err
	}
	if !keep {
		if nextKey == "" {
			nextKey = "next"
		}
		next = bytes.Clone(fields[nextKey])
		keep, err = truthy(bytes.Clone(next))
		if err != nil {
			return "", err
		}
	}
	if !keep {
		return "", nil
	}
	var text string
	if err := json.Unmarshal(next, &text); err != nil {
		return "", fmt.Errorf("Source next link is truthy but has no string startswith: %w", err)
	}
	return text, nil
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
		if key == "marker" && (!rules.offsetPagination || rules.markerOrOffsetPagination) || key == "offset" && (rules.offsetPagination || rules.markerOrOffsetPagination) {
			continue
		}
		old, exists := previous[key]
		if key == "limit" && rules.firstPage && !exists && (rules.offsetPagination || rules.markerOrOffsetPagination || rules.allowFirstServerLimit) {
			if _, err := controlledPaginationInput(url.Values{"limit": values}, rules.allowZeroLimit, rules.sourceRawLimit); err == nil {
				continue
			}
		}
		if key == "limit" && rules.reduceLimit && exists {
			newLimit, err := controlledPaginationInput(url.Values{"limit": values}, rules.allowZeroLimit, rules.sourceRawLimit)
			oldLimit, oldErr := controlledPaginationInput(url.Values{"limit": old}, rules.allowZeroLimit, rules.sourceRawLimit)
			if err == nil && oldErr == nil && newLimit <= oldLimit {
				continue
			}
		}
		if !exists || !reflect.DeepEqual(values, old) {
			return fmt.Errorf("%w: pagination changes filter %q", resource.ErrInvalidOption, key)
		}
	}
	for key, values := range previous {
		if key != "marker" && !(rules.markerOrOffsetPagination && key == "offset") && !query.Has(key) {
			query[key] = append([]string(nil), values...)
		}
	}
	if rules.markerOrOffsetPagination && !query.Has("marker") && !query.Has("offset") && previous.Has("offset") {
		query["offset"] = append([]string(nil), previous["offset"]...)
	}
	// A link which omits the marker retains the previous marker. It cannot
	// silently restart at the first page; the cycle guard rejects repetition.
	if !query.Has("marker") && previous.Has("marker") && !(rules.markerOrOffsetPagination && query.Has("offset")) {
		query["marker"] = append([]string(nil), previous["marker"]...)
	}
	if _, err := controlledPaginationInput(query, rules.allowZeroLimit, rules.sourceRawLimit); err != nil {
		return err
	}
	if rules.markerOrOffsetPagination && query.Has("offset") {
		nextOffset, err := paginationOffset(query)
		if err != nil {
			return err
		}
		if nextOffset == rules.offsetFloor {
			return &resource.PaginationCycleError{URL: next.String()}
		}
		if nextOffset < rules.offsetFloor {
			return fmt.Errorf("%w: pagination offset moves backwards", resource.ErrInvalidOption)
		}
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

// Source uses a truthy raw limit only to request another marker page. Typed
// collections retain the existing positive-integer validation.
func controlledPaginationInput(query url.Values, allowZero, source bool) (int, error) {
	if !source {
		return paginationInput(query, allowZero)
	}
	clone := queryCopy(query)
	delete(clone, "limit")
	if _, err := paginationInput(clone, allowZero); err != nil {
		return 0, err
	}
	if values, ok := query["limit"]; ok && len(values) > 0 && values[0] != "" && values[0] != "0" {
		return 1, nil
	}
	return 0, nil
}
