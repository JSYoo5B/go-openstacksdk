package metadeftags

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// List is lazy and reusable. A positive wire limit follows full pages using
// the raw last tag name. Nil limit and explicit zero each make one GET.
// MaxItems caps local consumption; links and next remain passive JSON.
func (s *NamespaceScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*Tag, error] {
	owned := append([]ListOption(nil), options...)
	return func(yield func(*Tag, error) bool) {
		fail := func(err error) { yield(nil, wrap(ctx, "List", err)) }
		p, err := s.capture(ctx, nil)
		if err != nil {
			fail(err)
			return
		}
		policy, query, err := prepareList(owned)
		if err = p.finish(ctx, policy.Headers, err); err != nil {
			fail(err)
			return
		}
		seen := make(map[string]bool)
		if policy.Marker != nil {
			seen[*policy.Marker] = true
		}
		consumed := 0
		for {
			if err = p.check(ctx); err != nil {
				fail(err)
				return
			}
			endpoint := p.url(nil)
			if encoded := query.Encode(); encoded != "" {
				endpoint += "?" + encoded
			}
			response, err := rest.DoJSON(ctx, p.client, http.MethodGet, endpoint, nil, nil, http.StatusOK)
			if err = checkResponse(ctx, p, response, err); err != nil {
				fail(err)
				return
			}
			fields, err := object(response.Body)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			rows, err := tagRows(fields)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			var lastName *string
			for index, raw := range rows {
				if policy.MaxItems > 0 && consumed >= policy.MaxItems {
					return
				}
				if err = p.check(ctx); err != nil {
					fail(response.Fail(err))
					return
				}
				var value Tag
				if err = json.Unmarshal(raw, &value); err != nil {
					fail(response.Fail(err))
					return
				}
				// Own the wire marker before any caller can mutate the yielded model.
				if index == len(rows)-1 {
					lastName = copyPointer(value.Name)
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
			if policy.Limit == nil || *policy.Limit == 0 || len(rows) < *policy.Limit {
				return
			}
			if lastName == nil || *lastName == "" {
				fail(response.Fail(invalid("full tag page requires a nonempty last raw name")))
				return
			}
			if err = queryText(*lastName); err != nil {
				fail(response.Fail(err))
				return
			}
			query.Set("marker", *lastName)
			if seen[*lastName] {
				fail(response.Fail(&resource.PaginationCycleError{URL: p.url(nil) + "?" + query.Encode()}))
				return
			}
			seen[*lastName] = true
			// Capture latest ordinary source headers for each subsequent request.
			p, err = s.capture(ctx, nil)
			if err == nil {
				err = p.finish(ctx, policy.Headers, nil)
			}
			if err != nil {
				fail(response.Fail(err))
				return
			}
		}
	}
}

// All returns a nonnil empty slice on success and discards rows on any error.
func (s *NamespaceScope) All(ctx context.Context, options ...ListOption) ([]*Tag, error) {
	values := make([]*Tag, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
