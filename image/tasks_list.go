package image

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Tasks lazily follows guarded canonical next links. Each iteration owns its
// options, source and cycle state; local caps and early breaks skip unused rows
// and continuation data. Stub detail fields remain absent without extra GETs.
func (s *Service) Tasks(ctx context.Context, options ...ListTasksOption) iter.Seq2[*TaskInfo, error] {
	owned := append([]ListTasksOption(nil), options...)
	return func(yield func(*TaskInfo, error) bool) {
		fail := func(err error) { yield(nil, wrapImageMutationError(ctx, "Tasks", err)) }
		p, err := s.captureTaskSource(ctx)
		if err != nil {
			fail(err)
			return
		}
		policy, initial, err := parseListTasksOptions(owned)
		if err = p.finish(ctx, policy.Headers, err); err != nil {
			fail(err)
			return
		}
		base, err := url.Parse(p.base + "tasks")
		if err != nil {
			fail(err)
			return
		}
		base.RawQuery = initial.Encode()
		current := base
		visited := map[string]bool{taskPageKey(base): true}
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
			if err = checkTaskResponse(ctx, p, response, err); err != nil {
				fail(err)
				return
			}
			fields, err := schemaObject(response.Body)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			raw, exists := schemaField(fields, "tasks")
			if !exists || bytes.TrimSpace(raw)[0] != '[' {
				fail(response.Fail(uploadInvalid("task list requires a nonnull tasks array")))
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
				var value TaskInfo
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
			next, err := taskNextPage(fields, base, current, initial)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			if next == nil {
				return
			}
			key, marker := taskPageKey(next), next.Query().Get("marker")
			if visited[key] || markers[marker] {
				fail(response.Fail(&resource.PaginationCycleError{URL: next.String()}))
				return
			}
			visited[key], markers[marker] = true, true
			if err = p.refresh(ctx, policy.Headers); err != nil {
				fail(response.Fail(err))
				return
			}
			current = next
		}
	}
}

// AllTasks returns a nonnil empty slice on success and discards partial rows if
// any consumed row, accepted response or later page fails.
func (s *Service) AllTasks(ctx context.Context, options ...ListTasksOption) ([]*TaskInfo, error) {
	values := make([]*TaskInfo, 0)
	for value, err := range s.Tasks(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func taskPageKey(value *url.URL) string {
	owned := *value
	owned.Scheme, owned.Host = strings.ToLower(owned.Scheme), strings.ToLower(owned.Host)
	owned.RawQuery = owned.Query().Encode()
	return owned.String()
}
func taskNextPage(fields map[string]json.RawMessage, base, current *url.URL, initial url.Values) (*url.URL, error) {
	raw, exists := schemaField(fields, "next")
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
	if err := taskQueryText(link); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, uploadInvalid("invalid task continuation: %v", err)
	}
	if parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" || strings.Contains(link, "#") || strings.HasPrefix(link, "//") {
		return nil, uploadInvalid("task continuation changes URL authority")
	}
	for _, part := range strings.Split(parsed.Path, "/") {
		if part == "." || part == ".." {
			return nil, uploadInvalid("task continuation contains a dot segment")
		}
	}
	if parsed.RawPath != "" && parsed.EscapedPath() != base.EscapedPath() {
		return nil, uploadInvalid("task continuation uses an escaped path alias")
	}
	if parsed.Scheme == "" && parsed.Host == "" && parsed.Path == "/v2/tasks" {
		parsed.Path, parsed.RawPath = base.Path, base.RawPath
	}
	next := current.ResolveReference(parsed)
	if next.User != nil || next.Opaque != "" || next.Fragment != "" || !strings.EqualFold(next.Scheme, base.Scheme) || !strings.EqualFold(next.Host, base.Host) || next.EscapedPath() != base.EscapedPath() {
		return nil, uploadInvalid("task continuation changes collection origin or path")
	}
	query, err := url.ParseQuery(next.RawQuery)
	if err != nil {
		return nil, uploadInvalid("invalid task continuation query: %v", err)
	}
	for key, values := range query {
		if key == "marker" {
			continue
		}
		original, exists := initial[key]
		if !exists || !reflect.DeepEqual(values, original) {
			return nil, uploadInvalid("task continuation changes filter %q", key)
		}
	}
	for key, values := range initial {
		if key != "marker" && !reflect.DeepEqual(query[key], values) {
			return nil, uploadInvalid("task continuation drops filter %q", key)
		}
	}
	markers := query["marker"]
	if len(markers) != 1 || markers[0] == "" {
		return nil, uploadInvalid("task continuation requires one nonempty marker")
	}
	if err := taskQueryText(markers[0]); err != nil {
		return nil, err
	}
	next.RawQuery = query.Encode()
	return next, nil
}
