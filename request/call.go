package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Call is one cross-cutting per-call control, such as a header, microversion,
// query value or paging limit. Calls only collect snapshots of their inputs;
// each operation interprets them with its own rules and rejects calls it does
// not support with resource.ErrInvalidOption before HTTP.
//
// The interface is sealed: only constructors in this package implement it.
// Operations accept one of the category interfaces below, so passing, for
// example, a paging limit to a Get or a body field to a Delete does not
// compile. Categories follow what operations of that shape support:
//
//   - GetCall: Header, Headers, Microversion
//   - FindCall: Header, Headers, Microversion, Query, IgnoreMissing
//   - ListCall: Header, Headers, Microversion, Query, BasePath, Limit, Marker,
//     MaxItems, Paginated
//   - DeleteCall: Header, Headers, Microversion, IgnoreMissing
//   - WriteCall (create, update and actions): Header, Headers, Microversion,
//     Field
//
// Custom implements every category. An operation may still reject a call of
// its category that it does not implement; it documents those exceptions.
type Call interface {
	applyCall(*CallConfig) error
}

// GetCall is accepted by Get operations.
type GetCall interface {
	Call
	getCall()
}

// FindCall is accepted by Find operations.
type FindCall interface {
	Call
	findCall()
}

// ListCall is accepted by List, All, Search and Proxy list operations.
type ListCall interface {
	Call
	listCall()
}

// DeleteCall is accepted by Delete operations.
type DeleteCall interface {
	Call
	deleteCall()
}

// WriteCall is accepted by Create, Update and action operations.
type WriteCall interface {
	Call
	writeCall()
}

// AnyCall is accepted by every operation category.
type AnyCall interface {
	GetCall
	FindCall
	ListCall
	DeleteCall
	WriteCall
}

// FindListCall is accepted by Find and List operations.
type FindListCall interface {
	FindCall
	ListCall
}

// FindDeleteCall is accepted by Find and Delete operations.
type FindDeleteCall interface {
	FindCall
	DeleteCall
}

// CallKind identifies the control carried by a CallEntry.
type CallKind uint8

const (
	CallHeader CallKind = iota + 1
	CallHeaders
	CallMicroversion
	CallQuery
	CallBasePath
	CallIgnoreMissing
	CallLimit
	CallMarker
	CallMaxItems
	CallPaginated
	CallField
	CallCustom
)

func (k CallKind) String() string {
	switch k {
	case CallHeader:
		return "Header"
	case CallHeaders:
		return "Headers"
	case CallMicroversion:
		return "Microversion"
	case CallQuery:
		return "Query"
	case CallBasePath:
		return "BasePath"
	case CallIgnoreMissing:
		return "IgnoreMissing"
	case CallLimit:
		return "Limit"
	case CallMarker:
		return "Marker"
	case CallMaxItems:
		return "MaxItems"
	case CallPaginated:
		return "Paginated"
	case CallField:
		return "Field"
	case CallCustom:
		return "Custom"
	}
	return fmt.Sprintf("CallKind(%d)", uint8(k))
}

// CallEntry is the snapshot one Call contributes. Only the fields named for
// its Kind are meaningful:
//
//   - Header: Key and Text
//   - Headers: Headers (nil when the caller passed nil)
//   - Microversion, BasePath, Marker: Text
//   - IgnoreMissing, Paginated: Bool
//   - Limit, MaxItems: Int
//   - Query, Field: Key, JSON and JSONErr
//   - Custom: Func, read with CustomFunc
//
// JSONErr retains a marshal failure. The operation decides whether and how to
// report it, so an operation that discards a key can also discard its error.
type CallEntry struct {
	Kind    CallKind
	Key     string
	Text    string
	Headers map[string]string
	Bool    bool
	Int     int
	JSON    json.RawMessage
	JSONErr error
	Func    any
}

func (e CallEntry) clone() CallEntry {
	e.Headers = maps.Clone(e.Headers)
	if e.JSON != nil {
		e.JSON = bytes.Clone(e.JSON)
	}
	return e
}

// CallConfig is the shared carrier that calls write into, in caller order.
type CallConfig struct {
	Entries []CallEntry
}

// Entry returns an owned copy of the snapshot of one call. A nil call is an
// invalid option; operations normally reject nil first with their own error.
func Entry(call Call) (CallEntry, error) {
	if call == nil {
		return CallEntry{}, fmt.Errorf("%w: nil call option", resource.ErrInvalidOption)
	}
	var c CallConfig
	if err := call.applyCall(&c); err != nil {
		return CallEntry{}, err
	}
	if len(c.Entries) != 1 {
		return CallEntry{}, fmt.Errorf("%w: malformed call option", resource.ErrInvalidOption)
	}
	return c.Entries[0].clone(), nil
}

// UnsupportedCall reports a call whose category the operation accepts but
// whose control the operation does not implement.
func UnsupportedCall(entry CallEntry) error {
	return fmt.Errorf("%w: %s call option is unsupported by this operation", resource.ErrInvalidOption, entry.Kind)
}

type entryCall struct{ entry CallEntry }

func (c entryCall) applyCall(cfg *CallConfig) error {
	cfg.Entries = append(cfg.Entries, c.entry.clone())
	return nil
}

type anyCall struct{ entryCall }

func (anyCall) getCall()    {}
func (anyCall) findCall()   {}
func (anyCall) listCall()   {}
func (anyCall) deleteCall() {}
func (anyCall) writeCall()  {}

type findListCall struct{ entryCall }

func (findListCall) findCall() {}
func (findListCall) listCall() {}

type findDeleteCall struct{ entryCall }

func (findDeleteCall) findCall()   {}
func (findDeleteCall) deleteCall() {}

type listCall struct{ entryCall }

func (listCall) listCall() {}

type writeCall struct{ entryCall }

func (writeCall) writeCall() {}

func jsonEntry(kind CallKind, key string, value any) CallEntry {
	raw, err := json.Marshal(value)
	return CallEntry{Kind: kind, Key: key, JSON: bytes.Clone(raw), JSONErr: err}
}

// Header sets one additional request header. Operations validate the name and
// reject collisions with headers they own.
func Header(key, value string) AnyCall {
	return anyCall{entryCall{CallEntry{Kind: CallHeader, Key: key, Text: value}}}
}

// Headers replaces the whole set of additional request headers collected so
// far. The map is copied when the call is constructed.
func Headers(values map[string]string) AnyCall {
	return anyCall{entryCall{CallEntry{Kind: CallHeaders, Headers: maps.Clone(values)}}}
}

// Microversion selects a per-call API microversion. The operation validates it
// against its own capability rules.
func Microversion(version string) AnyCall {
	return anyCall{entryCall{CallEntry{Kind: CallMicroversion, Text: version}}}
}

// Query sets one query or keyword value. The value is marshaled to JSON when
// the call is constructed; a marshal failure is kept for the operation.
func Query(key string, value any) FindListCall {
	return findListCall{entryCall{jsonEntry(CallQuery, key, value)}}
}

// BasePath overrides an operation's service-relative collection route.
func BasePath(path string) ListCall {
	return listCall{entryCall{CallEntry{Kind: CallBasePath, Text: path}}}
}

// IgnoreMissing selects whether Find or Delete treat a missing resource as a
// non-error result. Each operation documents its own default.
func IgnoreMissing(ignore bool) FindDeleteCall {
	return findDeleteCall{entryCall{CallEntry{Kind: CallIgnoreMissing, Bool: ignore}}}
}

// Limit sets the server page size.
func Limit(n int) ListCall {
	return listCall{entryCall{CallEntry{Kind: CallLimit, Int: n}}}
}

// Marker sets the server pagination marker.
func Marker(marker string) ListCall {
	return listCall{entryCall{CallEntry{Kind: CallMarker, Text: marker}}}
}

// MaxItems caps the number of yielded items across pages.
func MaxItems(n int) ListCall {
	return listCall{entryCall{CallEntry{Kind: CallMaxItems, Int: n}}}
}

// Paginated selects whether a list follows later pages.
func Paginated(paginated bool) ListCall {
	return listCall{entryCall{CallEntry{Kind: CallPaginated, Bool: paginated}}}
}

// Field sets one additional JSON body attribute of a create, update or action.
// The value is marshaled when the call is constructed; operations reject
// collisions with fields they own.
func Field(key string, value any) WriteCall {
	return writeCall{entryCall{jsonEntry(CallField, key, value)}}
}

// Custom wraps a function over an operation's exported options struct T, for
// example a package's ...Opts type. It implements every category; an
// operation whose options type is not T rejects it with
// resource.ErrInvalidOption before HTTP. The function runs at its position
// among the other calls, and the operation snapshots its state around it. A
// nil function is reported by the operation as a nil option.
func Custom[T any](fn func(*T) error) AnyCall {
	return anyCall{entryCall{CallEntry{Kind: CallCustom, Func: fn}}}
}

// CustomFunc returns the function of a Custom entry built for T. ok is false
// when the entry is not a Custom call over T; fn is nil when Custom was given
// a nil function.
func CustomFunc[T any](entry CallEntry) (fn func(*T) error, ok bool) {
	if entry.Kind != CallCustom {
		return nil, false
	}
	fn, ok = entry.Func.(func(*T) error)
	return fn, ok
}
