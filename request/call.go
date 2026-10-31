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
// example, a paging limit to a Get does not compile.
type Call interface {
	applyCall(*CallConfig) error
}

// ReadCall is accepted by Get and Find operations.
type ReadCall interface {
	Call
	readCall()
}

// ListCall is accepted by List, All and Search operations.
type ListCall interface {
	Call
	listCall()
}

// MutationCall is accepted by Create, Update, Delete and action operations.
type MutationCall interface {
	Call
	mutationCall()
}

// AnyCall is accepted by every operation category.
type AnyCall interface {
	ReadCall
	ListCall
	MutationCall
}

// ReadListCall is accepted by read and list operations.
type ReadListCall interface {
	ReadCall
	ListCall
}

// ReadMutationCall is accepted by read and mutation operations.
type ReadMutationCall interface {
	ReadCall
	MutationCall
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

func (anyCall) readCall()     {}
func (anyCall) listCall()     {}
func (anyCall) mutationCall() {}

type readListCall struct{ entryCall }

func (readListCall) readCall() {}
func (readListCall) listCall() {}

type readMutationCall struct{ entryCall }

func (readMutationCall) readCall()     {}
func (readMutationCall) mutationCall() {}

type listCall struct{ entryCall }

func (listCall) listCall() {}

type mutationCall struct{ entryCall }

func (mutationCall) mutationCall() {}

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
func Query(key string, value any) ReadListCall {
	return readListCall{entryCall{jsonEntry(CallQuery, key, value)}}
}

// BasePath overrides an operation's service-relative collection route.
func BasePath(path string) ReadListCall {
	return readListCall{entryCall{CallEntry{Kind: CallBasePath, Text: path}}}
}

// IgnoreMissing selects whether Find or Delete treat a missing resource as a
// non-error result. Each operation documents its own default.
func IgnoreMissing(ignore bool) ReadMutationCall {
	return readMutationCall{entryCall{CallEntry{Kind: CallIgnoreMissing, Bool: ignore}}}
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

// Field sets one additional JSON body attribute of a mutation. The value is
// marshaled when the call is constructed; operations reject collisions with
// fields they own.
func Field(key string, value any) MutationCall {
	return mutationCall{entryCall{jsonEntry(CallField, key, value)}}
}
