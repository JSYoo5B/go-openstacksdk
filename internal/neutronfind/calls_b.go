package neutronfind

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// OrderedJSON returns a keyword handler that records one JSON value in the
// ordered map that target points at: a new key is appended to the order and
// a later value replaces an earlier one. A marshal error is joined with
// fail(key) and nothing is recorded.
func OrderedJSON[S any](target func(*S) (*map[string]json.RawMessage, *[]string), fail func(key string) error) calls.KeywordFunc[S] {
	return func(s *S, key string, raw json.RawMessage, err error) error {
		if err != nil {
			return errors.Join(fail(key), err)
		}
		values, order := target(s)
		if *values == nil {
			*values = make(map[string]json.RawMessage)
		}
		if !slices.Contains(*order, key) {
			*order = append(*order, key)
		}
		(*values)[key] = bytes.Clone(raw)
		return nil
	}
}

// FieldJSON returns a request.Field handler that passes the call's key and
// JSON value to kw.
func FieldJSON[S any](kw calls.KeywordFunc[S]) func(*S, request.CallEntry) error {
	return func(s *S, entry request.CallEntry) error { return kw(s, entry.Key, entry.JSON, entry.JSONErr) }
}

// KeywordCall returns a request.Custom over S that applies kw to key with the
// value marshaled now; a marshal error surfaces when the call is applied.
func KeywordCall[S any](kw calls.KeywordFunc[S], key string, value any) request.AnyCall {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(s *S) error { return kw(s, key, bytes.Clone(owned), err) })
}
