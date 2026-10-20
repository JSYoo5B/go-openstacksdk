package metadefresourcetypes

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// List lazily consumes one finite global response. MaxItems is local; query,
// links, next and sorting are not interpreted.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ResourceType, error] {
	return finite(ctx, options, func() (*preparedSource, error) { return a.capture(ctx) }, "resource_types", func(raw []byte, response *rest.Response) (*ResourceType, error) {
		var value ResourceType
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
		return &value, nil
	})
}

// All collects List, discards partial rows on error and returns a nonnil empty
// slice on empty success.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*ResourceType, error) {
	values := make([]*ResourceType, 0)
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// List lazily consumes one finite association response using the same local
// options as the global list.
func (s *NamespaceScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*Association, error] {
	return finite(ctx, options, func() (*preparedSource, error) { return s.capture(ctx, nil) }, "resource_type_associations", func(raw []byte, response *rest.Response) (*Association, error) {
		var value Association
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
		return &value, nil
	})
}

// All collects association List without returning partial rows on error.
func (s *NamespaceScope) All(ctx context.Context, options ...ListOption) ([]*Association, error) {
	values := make([]*Association, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
func finite[T any](ctx context.Context, options []ListOption, capture func() (*preparedSource, error), key string, decode func([]byte, *rest.Response) (*T, error)) iter.Seq2[*T, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*T, error) bool) {
		fail := func(err error) { yield(nil, wrap(ctx, "List", err)) }
		p, err := capture()
		if err != nil {
			fail(err)
			return
		}
		policy, err := prepareList(options)
		if err = p.finish(ctx, policy.Headers, err); err != nil {
			fail(err)
			return
		}
		response, err := rest.DoJSON(ctx, p.client, http.MethodGet, p.url(nil), nil, nil, http.StatusOK)
		if err = checkResponse(ctx, p, response, err); err != nil {
			fail(err)
			return
		}
		fields, err := object(response.Body)
		if err != nil {
			fail(response.Fail(err))
			return
		}
		raw, exists := present(fields, key)
		if !exists {
			fail(response.Fail(fmt.Errorf("response requires a nonnull %s array", key)))
			return
		}
		var rows []json.RawMessage
		if err = json.Unmarshal(raw, &rows); err != nil {
			fail(response.Fail(err))
			return
		}
		for index, raw := range rows {
			if policy.MaxItems > 0 && index >= policy.MaxItems {
				return
			}
			if err = p.check(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
			value, err := decode(raw, response)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			if !yield(value, nil) {
				return
			}
			if err = p.check(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
		}
		if err = p.check(ctx); err != nil {
			fail(response.Fail(err))
		}
	}
}
