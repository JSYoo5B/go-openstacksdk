package neutronfind

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// Use adapts an option constructor over S to a calls.Spec handler, so a
// shared call is applied exactly like the constructor the package used
// before.
func Use[S, V any](build func(V) func(*S) error) func(*S, V) error {
	return func(s *S, v V) error { return build(v)(s) }
}

// KeywordPaging returns paging handlers that apply request.Limit,
// request.Marker, request.MaxItems and request.Paginated as the Python
// keyword arguments limit, marker, max_items and paginated through the
// package's own keyword constructor, with the same Go value it received
// before.
func KeywordPaging[S any](keyword func(key string, value any) func(*S) error) calls.Paging[S] {
	return calls.Paging[S]{
		Limit:     func(s *S, n int) error { return keyword("limit", n)(s) },
		Marker:    func(s *S, marker string) error { return keyword("marker", marker)(s) },
		MaxItems:  func(s *S, n int) error { return keyword("max_items", n)(s) },
		Paginated: func(s *S, paginated bool) error { return keyword("paginated", paginated)(s) },
	}
}

// UseJSON adapts a builder over one key and its JSON value (marshaled when
// the shared call was constructed, with its marshal error) to a Query or
// Field handler.
func UseJSON[S any](build func(key string, raw json.RawMessage, err error) func(*S) error) func(*S, request.CallEntry) error {
	return func(s *S, entry request.CallEntry) error { return build(entry.Key, entry.JSON, entry.JSONErr)(s) }
}
