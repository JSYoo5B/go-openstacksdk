package swcommon

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// WriteCalls is the shared call spec of Heat software creates and updates:
// an extension body member (request.Field) and an extension header
// (request.Header). Other shared calls are unsupported.
func WriteCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Field: calls.ConfigField[T], Header: calls.ConfigHeader[T]}
}

// ListCalls is the shared call spec of Heat software lists: request.Query
// with a string or []string value, and request.Limit and request.Marker,
// which set the typed limit and marker members that limit and marker point
// at. Other shared calls are unsupported.
func ListCalls[T any](limit func(*T) **int, marker func(*T) **string) calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{
		Query: calls.ConfigQuery[T],
		Paging: calls.Paging[request.Config[T]]{
			Limit:  func(c *request.Config[T], n int) error { *limit(&c.Options) = &n; return nil },
			Marker: func(c *request.Config[T], m string) error { *marker(&c.Options) = &m; return nil },
		},
	}
}
