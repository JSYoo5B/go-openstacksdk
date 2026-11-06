package keystonecalls

import (
	"encoding/json"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
)

// FindState points at the Source find fields of one option state.
type FindState struct {
	Query         *map[string]json.RawMessage
	Order         *[]string
	Headers       *map[string]string
	Microversion  **string
	IgnoreMissing **bool
}

// FindSpec handles request.Query (a Source find keyword argument kept in
// insertion order; a marshal error fails), request.Headers (replaces the
// headers), request.Microversion and request.IgnoreMissing.
func FindSpec[S any](state func(*S) FindState) calls.Spec[S] {
	return calls.Spec[S]{
		Query: calls.JSONQuery(func(s *S, key string, raw json.RawMessage, err error) error {
			if err != nil {
				return err
			}
			st := state(s)
			if *st.Query == nil {
				*st.Query = make(map[string]json.RawMessage)
			}
			(*st.Query)[key] = raw
			if !slices.Contains(*st.Order, key) {
				*st.Order = append(*st.Order, key)
			}
			return nil
		}),
		Headers:       func(s *S, values map[string]string) error { *state(s).Headers = values; return nil },
		Microversion:  func(s *S, version string) error { *state(s).Microversion = &version; return nil },
		IgnoreMissing: func(s *S, ignore bool) error { *state(s).IgnoreMissing = &ignore; return nil },
	}
}
