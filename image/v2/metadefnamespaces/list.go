package metadefnamespaces

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// List lazily follows only guarded advertised next links. Each iteration owns
// its options, source snapshot and cycle guards. MaxItems is a local cap and
// never supplies a wire limit or an invented marker.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Namespace, error] {
	owned := append([]ListOption(nil), options...)
	return func(yield func(*Namespace, error) bool) {
		fail := func(err error) { yield(nil, wrap(ctx, "List", err)) }
		p, err := a.capture(ctx)
		if err != nil {
			fail(err)
			return
		}
		policy, initial, err := prepareList(owned)
		if err = p.finish(ctx, policy.Headers, err); err != nil {
			fail(err)
			return
		}
		base, err := url.Parse(p.url(nil))
		if err != nil {
			fail(err)
			return
		}
		base.RawQuery = initial.Encode()
		current := base
		visited := map[string]bool{pageKey(base): true}
		markers := make(map[string]bool)
		if marker := initial.Get("marker"); marker != "" {
			markers[marker] = true
		}
		consumed := 0
		for {
			if err = p.check(ctx); err != nil {
				fail(err)
				return
			}
			response, err := rest.DoJSON(ctx, p.client, http.MethodGet, current.String(), nil, nil, http.StatusOK)
			if err = checkResponse(ctx, p, response, err); err != nil {
				fail(err)
				return
			}
			fields, err := object(response.Body)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			raw, exists := present(fields, "namespaces")
			if !exists || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '[' {
				fail(response.Fail(fmt.Errorf("namespace list requires a nonnull namespaces array")))
				return
			}
			var rows []json.RawMessage
			if err = json.Unmarshal(raw, &rows); err != nil {
				fail(response.Fail(err))
				return
			}
			for _, row := range rows {
				if policy.MaxItems > 0 && consumed >= policy.MaxItems {
					return
				}
				if err = p.check(ctx); err != nil {
					fail(response.Fail(err))
					return
				}
				var value Namespace
				if err = json.Unmarshal(row, &value); err != nil {
					fail(response.Fail(err))
					return
				}
				value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
				consumed++
				if !yield(&value, nil) {
					return
				}
				if err = p.check(ctx); err != nil {
					fail(response.Fail(err))
					return
				}
				if policy.MaxItems > 0 && consumed >= policy.MaxItems {
					return
				}
			}
			if err = p.check(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
			if len(rows) == 0 || policy.SinglePage {
				return
			}
			next, err := nextPage(fields, base, current, initial)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			if next == nil {
				return
			}
			key, marker := pageKey(next), next.Query().Get("marker")
			if visited[key] || markers[marker] {
				fail(response.Fail(&resource.PaginationCycleError{URL: next.String()}))
				return
			}
			visited[key], markers[marker] = true, true
			current = next
		}
	}
}

// All returns a nonnil empty slice on success and discards partial rows if any
// consumed row or later page fails.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*Namespace, error) {
	values := make([]*Namespace, 0)
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func pageKey(value *url.URL) string {
	owned := *value
	owned.Scheme, owned.Host = strings.ToLower(owned.Scheme), strings.ToLower(owned.Host)
	owned.RawQuery = owned.Query().Encode()
	return owned.String()
}

func nextPage(fields map[string]json.RawMessage, base, current *url.URL, initial url.Values) (*url.URL, error) {
	raw, exists := present(fields, "next")
	if !exists {
		return nil, nil
	}
	var link string
	if err := json.Unmarshal(raw, &link); err != nil {
		return nil, err
	}
	if link == "" {
		return nil, nil
	}
	if err := queryText(link); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, invalid("invalid namespace next link: %v", err)
	}
	// Inspect before ResolveReference can erase dot segments or URL authority.
	if parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" || strings.Contains(link, "#") || strings.HasPrefix(link, "//") {
		return nil, invalid("namespace continuation changes URL authority")
	}
	for _, part := range strings.Split(parsed.Path, "/") {
		if part == "." || part == ".." {
			return nil, invalid("namespace continuation contains a dot segment")
		}
	}
	if parsed.RawPath != "" && parsed.EscapedPath() != base.EscapedPath() {
		return nil, invalid("namespace continuation uses an escaped path alias")
	}
	if parsed.Scheme == "" && parsed.Host == "" && parsed.Path == "/v2/metadefs/namespaces" {
		parsed.Path, parsed.RawPath = base.Path, base.RawPath
	}
	next := current.ResolveReference(parsed)
	if next.User != nil || next.Opaque != "" || next.Fragment != "" || !strings.EqualFold(next.Scheme, base.Scheme) || !strings.EqualFold(next.Host, base.Host) || next.EscapedPath() != base.EscapedPath() {
		return nil, invalid("namespace continuation changes collection origin or path")
	}
	query, err := url.ParseQuery(next.RawQuery)
	if err != nil {
		return nil, invalid("invalid namespace continuation query: %v", err)
	}
	for key, values := range query {
		if key == "marker" {
			continue
		}
		original, exists := initial[key]
		if !exists || !reflect.DeepEqual(values, original) {
			return nil, invalid("namespace continuation changes filter %q", key)
		}
	}
	for key, values := range initial {
		if key != "marker" && !reflect.DeepEqual(query[key], values) {
			return nil, invalid("namespace continuation drops filter %q", key)
		}
	}
	markers := query["marker"]
	if len(markers) != 1 {
		return nil, invalid("namespace continuation requires one marker")
	}
	if err := literal(markers[0]); err != nil {
		return nil, err
	}
	next.RawQuery = query.Encode()
	return next, nil
}
