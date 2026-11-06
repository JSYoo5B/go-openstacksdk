package masakari

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// WriteCalls is the shared call spec of Masakari creates and updates: an
// extension body field (request.Field) and an extension header
// (request.Header). Other shared calls are unsupported.
func WriteCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Field: calls.ConfigField[T], Header: calls.ConfigHeader[T]}
}

// ListCalls is the shared call spec of Masakari lists: an extension query
// value (request.Query with a string or []string value). Paging and other
// shared calls are unsupported; limit and marker are typed ListOpts members.
func ListCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Query: calls.ConfigQuery[T]}
}
