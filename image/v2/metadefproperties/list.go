package metadefproperties

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"

	"gophercloudsdk/internal/rest"
)

type propertyEntry struct {
	key string
	raw json.RawMessage
}

// orderedDictionary retains each key's first slot and its last value, matching
// decoded Python dictionaries while preserving the wire encounter order.
func orderedDictionary(raw json.RawMessage) ([]propertyEntry, error) {
	if _, err := object(raw); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	entries := make([]propertyEntry, 0)
	positions := make(map[string]int)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("property dictionary key must be a string")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		if index, exists := positions[key]; exists {
			entries[index].raw = value
		} else {
			positions[key] = len(entries)
			entries = append(entries, propertyEntry{key: key, raw: value})
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return entries, nil
}

// List lazily fetches one finite dictionary. MaxItems bounds local consumption;
// pagination hints remain passive and every Key retains its wire provenance.
func (s *NamespaceScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*Property, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Property, error) bool) {
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
		raw, exists := present(fields, "properties")
		if !exists {
			fail(response.Fail(fmt.Errorf("response requires a nonnull properties dictionary")))
			return
		}
		entries, err := orderedDictionary(raw)
		if err != nil {
			fail(response.Fail(err))
			return
		}
		for index, entry := range entries {
			if policy.MaxItems > 0 && index >= policy.MaxItems {
				return
			}
			if err = p.check(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
			var value Property
			if err = json.Unmarshal(entry.raw, &value); err != nil {
				fail(response.Fail(err))
				return
			}
			value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
			value.Key = copyPointer(&entry.key)
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
func (s *NamespaceScope) All(ctx context.Context, options ...ListOption) ([]*Property, error) {
	values := make([]*Property, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
