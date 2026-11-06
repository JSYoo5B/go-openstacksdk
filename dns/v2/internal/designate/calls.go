package designate

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// FieldCalls is the shared call spec of Designate creates and updates that
// take extension body members (request.Field) only.
func FieldCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Field: calls.ConfigField[T]}
}

// FieldHeaderCalls is the shared call spec of Designate writes that also take
// an extension header (request.Header).
func FieldHeaderCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Field: calls.ConfigField[T], Header: calls.ConfigHeader[T]}
}

// ListCalls is the shared call spec of Designate lists: request.Query with a
// string or []string value. Paging calls are unsupported; limit and marker
// are typed ListOpts members.
func ListCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Query: calls.ConfigQuery[T]}
}
