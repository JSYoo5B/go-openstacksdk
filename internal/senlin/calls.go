package senlin

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// The handlers below translate the shared request calls of package request
// for the Senlin operations of the clustering packages. Each operation keeps
// its request.Config apply loop, so calls.ApplyConfig or calls.Options turn
// the calls into the same options the removed per-operation With* functions
// built. Controls without a handler fail with request.UnsupportedCall.

// ListCalls handles a Senlin list over its options T: request.Header (an
// additional header, validated when the list is prepared), request.Microversion
// (a numeric 1.N for this list only; explicit empty selects the server
// default), request.Query (a string or []string vendor query) and
// request.MaxItems and request.Paginated through the given fields. Senlin
// lists never took Limit or Marker calls or replacing Headers.
func ListCalls[T any](maxItems func(*T) *int, paginated func(*T) **bool) calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{
		Header:       func(c *request.Config[T], key, value string) error { return WithListHeader[T](key, value)(c) },
		Microversion: func(c *request.Config[T], version string) error { return WithListMicroversion[T](version)(c) },
		Query:        calls.ConfigQuery[T],
		Paging: calls.Paging[request.Config[T]]{
			MaxItems:  calls.Value(func(c *request.Config[T]) *int { return maxItems(&c.Options) }),
			Paginated: calls.Pointer(func(c *request.Config[T]) **bool { return paginated(&c.Options) }),
		},
	}
}

// SharedListCalls handles the lists whose options are ListOpts. MaxItems and
// Paginated apply WithMaxItems and WithPaginated.
var SharedListCalls = func() calls.Spec[request.Config[ListOpts]] {
	spec := ListCalls(func(o *ListOpts) *int { return &o.MaxItems }, func(o *ListOpts) **bool { return &o.Paginated })
	spec.MaxItems = func(c *request.Config[ListOpts], n int) error { return WithMaxItems(n)(c) }
	spec.Paginated = func(c *request.Config[ListOpts], paginated bool) error { return WithPaginated(paginated)(c) }
	return spec
}()

// FindCalls handles FindIdentity: request.Header, request.Microversion (both
// phases; explicit empty selects the server default) and
// request.IgnoreMissing. FindIdentity never took a query.
var FindCalls = calls.Spec[request.Config[FindOpts]]{
	Header:        func(c *request.Config[FindOpts], key, value string) error { return WithFindHeader(key, value)(c) },
	Microversion:  func(c *request.Config[FindOpts], version string) error { return WithFindMicroversion(version)(c) },
	IgnoreMissing: func(c *request.Config[FindOpts], ignore bool) error { return WithFindIgnoreMissing(ignore)(c) },
}

// WriteCalls handles create, update, validate and action commands:
// request.Header and request.Field (one extension body attribute).
func WriteCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Header: calls.ConfigHeader[T], Field: calls.ConfigField[T]}
}

// HeaderCalls handles operations that take additional headers only.
func HeaderCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Header: calls.ConfigHeader[T]}
}

// GetQueryCalls handles member reads with request.Header and request.Query
// (a string or []string vendor query).
func GetQueryCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Header: calls.ConfigHeader[T], Query: calls.ConfigQuery[T]}
}

// DeleteCalls handles deletes with request.Header and request.IgnoreMissing,
// stored through the given field.
func DeleteCalls[T any](ignoreMissing func(*T) **bool) calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{
		Header:        calls.ConfigHeader[T],
		IgnoreMissing: calls.Pointer(func(c *request.Config[T]) **bool { return ignoreMissing(&c.Options) }),
	}
}
