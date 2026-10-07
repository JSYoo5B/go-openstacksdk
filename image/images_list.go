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
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ListImages lazily follows guarded canonical next links, preserving the full
// owned filter query when Glance collapses repeated values in advertised links.
// Local caps and early breaks skip unused rows and continuation data.
func (s *Service) ListImages(ctx context.Context, options ...ListImagesOption) iter.Seq2[*ImageInfo, error] {
	owned := append([]ListImagesOption(nil), options...)
	return func(yield func(*ImageInfo, error) bool) {
		fail := func(err error) { yield(nil, wrapImageMutationError(ctx, "ListImages", err)) }
		p, err := s.captureTaskSource(ctx)
		if err != nil {
			fail(err)
			return
		}
		policy, initial, err := parseListImagesOptions(owned)
		if err = p.finish(ctx, policy.Headers, err); err != nil {
			fail(err)
			return
		}
		base, err := url.Parse(p.base + "images")
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
			raw, exists := schemaField(fields, "images")
			if !exists || bytes.TrimSpace(raw)[0] != '[' {
				fail(response.Fail(uploadInvalid("image list requires a nonnull images array")))
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
				var value ImageInfo
				if err = json.Unmarshal(row, &value); err != nil {
					fail(response.Fail(err))
					return
				}
				value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
				if err = p.check(ctx); err != nil {
					fail(response.Fail(err))
					return
				}
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
			if policy.SinglePage {
				return
			}
			next, err := imageNextPage(fields, base, current, initial)
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

// AllImages returns a nonnil empty slice on success and discards partial rows if
// any consumed row, accepted response or later page fails.
func (s *Service) AllImages(ctx context.Context, options ...ListImagesOption) ([]*ImageInfo, error) {
	values := make([]*ImageInfo, 0)
	for value, err := range s.ListImages(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func imageNextPage(fields map[string]json.RawMessage, base, current *url.URL, initial url.Values) (*url.URL, error) {
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
		return nil, uploadInvalid("invalid image continuation: %v", err)
	}
	if parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" || strings.Contains(link, "#") || strings.HasPrefix(link, "//") {
		return nil, uploadInvalid("image continuation changes URL authority")
	}
	for _, part := range strings.Split(parsed.Path, "/") {
		if part == "." || part == ".." {
			return nil, uploadInvalid("image continuation contains a dot segment")
		}
	}
	if parsed.RawPath != "" && parsed.EscapedPath() != base.EscapedPath() {
		return nil, uploadInvalid("image continuation uses an escaped path alias")
	}
	if parsed.Scheme == "" && parsed.Host == "" && parsed.Path == "/v2/images" {
		parsed.Path, parsed.RawPath = base.Path, base.RawPath
	}
	next := current.ResolveReference(parsed)
	if next.User != nil || next.Opaque != "" || next.Fragment != "" || !strings.EqualFold(next.Scheme, base.Scheme) || !strings.EqualFold(next.Host, base.Host) || next.EscapedPath() != base.EscapedPath() {
		return nil, uploadInvalid("image continuation changes collection origin or path")
	}
	query, err := url.ParseQuery(next.RawQuery)
	if err != nil {
		return nil, uploadInvalid("invalid image continuation query: %v", err)
	}
	for key, values := range query {
		if key == "marker" {
			continue
		}
		original, exists := initial[key]
		if !exists || !imageContinuationValues(values, original) {
			return nil, uploadInvalid("image continuation changes filter %q", key)
		}
	}
	for key, values := range initial {
		if key != "marker" && !imageContinuationValues(query[key], values) {
			return nil, uploadInvalid("image continuation drops filter %q", key)
		}
	}
	markers := query["marker"]
	if len(markers) != 1 || markers[0] == "" {
		return nil, uploadInvalid("image continuation requires one nonempty marker")
	}
	if !utf8.ValidString(markers[0]) {
		return nil, uploadInvalid("image continuation marker must be valid UTF-8")
	}
	restored := copyImageQuery(initial)
	restored.Set("marker", markers[0])
	next.RawQuery = restored.Encode()
	return next, nil
}

// Glance's dict(request.params) may retain one value of a repeated key. Only
// that bounded projection is accepted; the request restores all original values.
func imageContinuationValues(incoming, original []string) bool {
	if reflect.DeepEqual(incoming, original) {
		return true
	}
	if len(original) > 1 && len(incoming) == 1 {
		for _, value := range original {
			if incoming[0] == value {
				return true
			}
		}
	}
	return false
}
