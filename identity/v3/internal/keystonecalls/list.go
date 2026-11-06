// Package keystonecalls holds the shared-call handlers that several SDK-owned
// identity record packages use for their Source Resource.list keyword
// arguments, so that each package declares only its own key rules.
package keystonecalls

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ListState points at the Source list fields of one option state.
type ListState struct {
	Query        *map[string]json.RawMessage
	Headers      *map[string]string
	Microversion **string
	BasePath     **string
	Order        *[]string
}

// formal stores a headers, microversion or base_path keyword through the
// Proxy normalization and removes it from Query.
func (st ListState) formal(key string, raw json.RawMessage) error {
	normalized, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{key: bytes.Clone(raw)}})
	if err != nil {
		return err
	}
	switch key {
	case "headers":
		*st.Headers = normalized.Headers
	case "microversion":
		*st.Microversion = normalized.Microversion
	case "base_path":
		*st.BasePath = normalized.BasePath
	}
	delete(*st.Query, key)
	return nil
}

func (st ListState) set(key string, raw json.RawMessage) {
	if *st.Query == nil {
		*st.Query = map[string]json.RawMessage{}
	}
	(*st.Query)[key] = bytes.Clone(raw)
	if !slices.Contains(*st.Order, key) {
		*st.Order = append(*st.Order, key)
	}
}

func isFormal(key string) bool {
	return key == "headers" || key == "microversion" || key == "base_path"
}

// KnownList handles lists that validate every key, then keep only the keys
// the Resource class declares. Formal headers, microversion and base_path
// replace the matching raw keyword without touching the keyword order.
type KnownList[S any] struct {
	State func(*S) ListState
	// Key validates a keyword name before anything else.
	Key func(key string) error
	// Known reports declared keywords; others are dropped with their marshal
	// error.
	Known func(key string) bool
}

// Apply applies one keyword argument given as JSON.
func (l KnownList[S]) Apply(s *S, key string, raw json.RawMessage, err error) error {
	if keyErr := l.Key(key); keyErr != nil {
		return keyErr
	}
	if !l.Known(key) {
		return nil
	}
	if err != nil {
		return err
	}
	st := l.State(s)
	if *st.Query == nil {
		*st.Query = map[string]json.RawMessage{}
	}
	if isFormal(key) {
		return st.formal(key, raw)
	}
	st.set(key, raw)
	return nil
}

// Query returns the package's own query option: the key is validated and an
// unknown key dropped before the value is marshaled, which request.Query
// (marshaled at construction) cannot do.
func (l KnownList[S]) Query(key string, value any) request.AnyCall {
	if err := l.Key(key); err != nil {
		return request.Custom(func(*S) error { return err })
	}
	if !l.Known(key) {
		return request.Custom(func(*S) error { return nil })
	}
	return Keyword(l.Apply, key, value)
}

// Spec handles request.Headers (replaces), request.Microversion,
// request.BasePath and the paging calls, which are Source keyword arguments.
// request.Query is unsupported: Query checks keys before marshaling.
func (l KnownList[S]) Spec() calls.Spec[S] {
	formal := func(s *S, key string) { delete(*l.State(s).Query, key) }
	return calls.Spec[S]{
		Headers: func(s *S, values map[string]string) error {
			*l.State(s).Headers = values
			formal(s, "headers")
			return nil
		},
		Microversion: func(s *S, version string) error {
			*l.State(s).Microversion = &version
			formal(s, "microversion")
			return nil
		},
		BasePath: func(s *S, path string) error {
			*l.State(s).BasePath = &path
			formal(s, "base_path")
			return nil
		},
		Paging: calls.JSONPaging(l.Apply),
	}
}

// SourceList handles lists that reject Source runtime controls by presence,
// discard keywords Resource.list ignores before evaluating them, and wrap
// encoding failures. Formal headers, microversion and base_path also leave
// the keyword order.
type SourceList[S any] struct {
	State func(*S) ListState
	// Control rejects runtime controls and malformed names.
	Control func(key string) error
	// Ignores reports keywords Resource.list discards.
	Ignores func(key string) bool
	// Invalid builds the package's invalid-option error.
	Invalid func(format string, args ...any) error
}

func (l SourceList[S]) drop(s *S, key string) {
	st := l.State(s)
	delete(*st.Query, key)
	*st.Order = slices.DeleteFunc(*st.Order, func(k string) bool { return k == key })
}

// Apply applies one keyword argument given as JSON.
func (l SourceList[S]) Apply(s *S, key string, raw json.RawMessage, err error) error {
	if controlErr := l.Control(key); controlErr != nil {
		return controlErr
	}
	if l.Ignores(key) {
		return nil
	}
	if err != nil {
		return errors.Join(l.Invalid("query encoding"), err)
	}
	if isFormal(key) {
		if err := l.State(s).formal(key, raw); err != nil {
			return err
		}
		l.drop(s, key)
		return nil
	}
	l.State(s).set(key, raw)
	return nil
}

// Query returns the package's own query option: Source runtime controls fail
// and ignored keys are dropped before the value is marshaled.
func (l SourceList[S]) Query(key string, value any) request.AnyCall {
	if err := l.Control(key); err != nil {
		return request.Custom(func(*S) error { return err })
	}
	if l.Ignores(key) {
		return request.Custom(func(*S) error { return nil })
	}
	return Keyword(l.Apply, key, value)
}

// ControlSpec is Spec without request.Query, for lists whose callers rely on
// Query checking keys before marshaling.
func (l SourceList[S]) ControlSpec() calls.Spec[S] {
	spec := l.Spec()
	spec.Query = nil
	return spec
}

// Spec handles request.Query, request.Headers (replaces), request.Microversion,
// request.BasePath and the paging calls, which are Source keyword arguments.
func (l SourceList[S]) Spec() calls.Spec[S] {
	return calls.Spec[S]{
		Query: calls.JSONQuery(l.Apply),
		Headers: func(s *S, values map[string]string) error {
			*l.State(s).Headers = values
			l.drop(s, "headers")
			return nil
		},
		Microversion: func(s *S, version string) error {
			*l.State(s).Microversion = &version
			l.drop(s, "microversion")
			return nil
		},
		BasePath: func(s *S, path string) error {
			*l.State(s).BasePath = &path
			l.drop(s, "base_path")
			return nil
		},
		Paging: calls.JSONPaging(l.Apply),
	}
}

// Keyword returns an operation-specific option that applies one keyword
// argument through apply, marshaled now, such as a package's JMESPath option.
func Keyword[S any](apply calls.KeywordFunc[S], key string, value any) request.AnyCall {
	raw, err := json.Marshal(value)
	return request.Custom(func(s *S) error { return apply(s, key, raw, err) })
}
