package metadefobjects

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"iter"
	"net/http"
)

// List lazily fetches one finite collection. MaxItems bounds local consumption;
// links, next and other pagination hints remain passive JSON.
func (s *NamespaceScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*Object, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Object, error) bool) {
		fail := func(err error) { yield(nil, wrap(ctx, "List", err)) }
		p, err := s.capture(ctx, nil)
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
		raw, exists := present(fields, "objects")
		if !exists {
			fail(response.Fail(fmt.Errorf("response requires a nonnull objects array")))
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
			var value Object
			if err = json.Unmarshal(raw, &value); err != nil {
				fail(response.Fail(err))
				return
			}
			value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
			if !yield(&value, nil) {
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

// All collects List and discards partial rows on error. Empty success is nonnil.
func (s *NamespaceScope) All(ctx context.Context, options ...ListOption) ([]*Object, error) {
	values := make([]*Object, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
