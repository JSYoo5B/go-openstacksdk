package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Call is one shared per-call control, such as a header, microversion, query
// value or paging limit. Calls only collect snapshots of their inputs; each
// operation interprets them with its own rules and rejects calls it does not
// support with resource.ErrInvalidOption before HTTP.
//
// The interface is sealed: only constructors in this package implement it.
// Operations accept one of the category interfaces listed in the package
// documentation, so a control the operation cannot honor does not compile.
type Call interface {
	applyCall(*CallConfig) error
}

// GetCall is accepted by SDK-owned member reads without query input.
type GetCall interface {
	Call
	getCall()
}

// GetQueryCall is accepted by SDK-owned member reads with query input.
type GetQueryCall interface {
	Call
	getQueryCall()
}

// FindCall is accepted by SDK-owned Find operations.
type FindCall interface {
	Call
	findCall()
}

// DeleteCall is accepted by SDK-owned Delete operations.
type DeleteCall interface {
	Call
	deleteCall()
}

// ListCall is accepted by SDK-owned List and All operations with a fixed
// collection route.
type ListCall interface {
	Call
	listCall()
}

// ListBasePathCall is accepted by SDK-owned List, All and Proxy operations
// whose collection route can be overridden with BasePath.
type ListBasePathCall interface {
	Call
	listBasePathCall()
}

// WriteCall is accepted by SDK-owned Create, Update and action operations.
type WriteCall interface {
	Call
	writeCall()
}

// NativeCall is accepted by generated native operations that support no
// extension field, query value or header.
type NativeCall interface {
	Call
	nativeCall()
}

// NativeHeaderCall is accepted by generated native operations that support
// extension headers only.
type NativeHeaderCall interface {
	Call
	nativeHeaderCall()
}

// NativeQueryCall is accepted by generated native operations that support
// extension query values only.
type NativeQueryCall interface {
	Call
	nativeQueryCall()
}

// NativeQueryHeaderCall is accepted by generated native operations that
// support extension query values and headers.
type NativeQueryHeaderCall interface {
	Call
	nativeQueryHeaderCall()
}

// NativeFieldCall is accepted by generated native operations that support
// extension JSON body fields only.
type NativeFieldCall interface {
	Call
	nativeFieldCall()
}

// NativeFieldHeaderCall is accepted by generated native operations that
// support extension JSON body fields and headers.
type NativeFieldHeaderCall interface {
	Call
	nativeFieldHeaderCall()
}

// NativeFieldQueryCall is accepted by generated native operations that
// support extension JSON body fields and query values.
type NativeFieldQueryCall interface {
	Call
	nativeFieldQueryCall()
}

// AnyCall is accepted by every operation category. Custom and the
// operation-specific options of generated packages return it.
type AnyCall interface {
	GetCall
	GetQueryCall
	FindCall
	DeleteCall
	ListCall
	ListBasePathCall
	WriteCall
	NativeCall
	NativeHeaderCall
	NativeQueryCall
	NativeQueryHeaderCall
	NativeFieldCall
	NativeFieldHeaderCall
	NativeFieldQueryCall
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

// QueryForm records which value type a Query call was built from.
type QueryForm uint8

const (
	// QueryJSON is any value other than string or []string. Only operations
	// that take JSON keyword arguments accept it.
	QueryJSON QueryForm = iota
	// QueryText is a string value, carried in Text.
	QueryText
	// QueryList is a []string value, carried in Values; the key repeats.
	QueryList
)

// CallEntry is the snapshot one Call contributes. Only the fields named for
// its Kind are meaningful:
//
//   - Header: Key and Text
//   - Headers: Headers (nil when the caller passed nil)
//   - Microversion, BasePath, Marker: Text
//   - IgnoreMissing, Paginated: Bool
//   - Limit, MaxItems: Int
//   - Query: Key, Form, JSON and JSONErr, plus Text (QueryText) or Values
//     (QueryList)
//   - Field: Key, JSON and JSONErr
//   - Custom: Func, read with CustomFunc
//
// JSON is the value marshaled when the call was constructed, for every Query
// form. JSONErr retains a marshal failure. The operation decides whether and
// how to report it, so an operation that discards a key can also discard its
// error.
type CallEntry struct {
	Kind    CallKind
	Key     string
	Text    string
	Values  []string
	Form    QueryForm
	Headers map[string]string
	Bool    bool
	Int     int
	JSON    json.RawMessage
	JSONErr error
	Func    any
}

func (e CallEntry) clone() CallEntry {
	e.Headers = maps.Clone(e.Headers)
	e.Values = slices.Clone(e.Values)
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

// Category marks. Each constructor's result embeds exactly the marks of the
// categories whose combination allows its control.
type (
	getMark               struct{}
	getQueryMark          struct{}
	findMark              struct{}
	deleteMark            struct{}
	listMark              struct{}
	listBasePathMark      struct{}
	writeMark             struct{}
	nativeMark            struct{}
	nativeHeaderMark      struct{}
	nativeQueryMark       struct{}
	nativeQueryHeaderMark struct{}
	nativeFieldMark       struct{}
	nativeFieldHeaderMark struct{}
	nativeFieldQueryMark  struct{}
	sdkControlMarks       struct {
		getMark
		getQueryMark
		findMark
		deleteMark
		listMark
		listBasePathMark
		writeMark
	}
)

func (getMark) getCall()                             {}
func (getQueryMark) getQueryCall()                   {}
func (findMark) findCall()                           {}
func (deleteMark) deleteCall()                       {}
func (listMark) listCall()                           {}
func (listBasePathMark) listBasePathCall()           {}
func (writeMark) writeCall()                         {}
func (nativeMark) nativeCall()                       {}
func (nativeHeaderMark) nativeHeaderCall()           {}
func (nativeQueryMark) nativeQueryCall()             {}
func (nativeQueryHeaderMark) nativeQueryHeaderCall() {}
func (nativeFieldMark) nativeFieldCall()             {}
func (nativeFieldHeaderMark) nativeFieldHeaderCall() {}
func (nativeFieldQueryMark) nativeFieldQueryCall()   {}

// headerCall is accepted by every category that takes extension headers.
type headerCall struct {
	entryCall
	sdkControlMarks
	nativeHeaderMark
	nativeQueryHeaderMark
	nativeFieldHeaderMark
}

// controlCall carries a control only SDK-owned operations implement.
type controlCall struct {
	entryCall
	sdkControlMarks
}

type queryCall struct {
	entryCall
	getQueryMark
	findMark
	listMark
	listBasePathMark
	nativeQueryMark
	nativeQueryHeaderMark
	nativeFieldQueryMark
}

type ignoreMissingCall struct {
	entryCall
	findMark
	deleteMark
}

type pagingCall struct {
	entryCall
	listMark
	listBasePathMark
}

type basePathCall struct {
	entryCall
	listBasePathMark
}

type fieldCall struct {
	entryCall
	writeMark
	nativeFieldMark
	nativeFieldHeaderMark
	nativeFieldQueryMark
}

type anyCall struct {
	entryCall
	sdkControlMarks
	nativeMark
	nativeHeaderMark
	nativeQueryMark
	nativeQueryHeaderMark
	nativeFieldMark
	nativeFieldHeaderMark
	nativeFieldQueryMark
}

func jsonEntry(kind CallKind, key string, value any) CallEntry {
	raw, err := json.Marshal(value)
	return CallEntry{Kind: kind, Key: key, JSON: bytes.Clone(raw), JSONErr: err}
}

// Header sets one additional request header. Every category that takes
// extension headers accepts it. Operations validate the name and reject
// collisions with headers they own.
func Header(key, value string) headerCall {
	return headerCall{entryCall: entryCall{CallEntry{Kind: CallHeader, Key: key, Text: value}}}
}

// Headers replaces the whole set of additional request headers collected so
// far. The map is copied when the call is constructed. Only SDK-owned
// categories accept it.
func Headers(values map[string]string) controlCall {
	return controlCall{entryCall: entryCall{CallEntry{Kind: CallHeaders, Headers: maps.Clone(values)}}}
}

// Microversion selects a per-call API microversion. Only SDK-owned categories
// accept it; the operation validates it against its own capability rules.
func Microversion(version string) controlCall {
	return controlCall{entryCall: entryCall{CallEntry{Kind: CallMicroversion, Text: version}}}
}

// Query sets one query or keyword value. Its meaning depends on the value's
// type:
//
//   - string: one raw text value for the key.
//   - []string: the key repeats once per value, in order, like a Python list
//     passed to requests params. No values remove an earlier value for the
//     key.
//   - any other type: a Python-style JSON keyword value. Only operations that
//     take JSON keyword arguments accept it; generated native operations
//     reject it with resource.ErrInvalidOption before HTTP. Convert named
//     string types with string(v) to send raw text.
//
// The value is also marshaled to JSON when the call is constructed, so
// operations that take JSON keyword arguments see a string as a JSON string
// and a []string as a JSON array. A marshal failure is kept for the
// operation.
func Query(key string, value any) queryCall {
	entry := jsonEntry(CallQuery, key, value)
	switch v := value.(type) {
	case string:
		entry.Form, entry.Text = QueryText, v
	case []string:
		entry.Form, entry.Values = QueryList, slices.Clone(v)
	default:
		entry.Form = QueryJSON
	}
	return queryCall{entryCall: entryCall{entry}}
}

// BasePath overrides an operation's service-relative collection route.
func BasePath(path string) basePathCall {
	return basePathCall{entryCall: entryCall{CallEntry{Kind: CallBasePath, Text: path}}}
}

// IgnoreMissing selects whether Find or Delete treat a missing resource as a
// non-error result. Each operation documents its own default.
func IgnoreMissing(ignore bool) ignoreMissingCall {
	return ignoreMissingCall{entryCall: entryCall{CallEntry{Kind: CallIgnoreMissing, Bool: ignore}}}
}

// Limit sets the server page size.
func Limit(n int) pagingCall {
	return pagingCall{entryCall: entryCall{CallEntry{Kind: CallLimit, Int: n}}}
}

// Marker sets the server pagination marker.
func Marker(marker string) pagingCall {
	return pagingCall{entryCall: entryCall{CallEntry{Kind: CallMarker, Text: marker}}}
}

// MaxItems caps the number of yielded items across pages.
func MaxItems(n int) pagingCall {
	return pagingCall{entryCall: entryCall{CallEntry{Kind: CallMaxItems, Int: n}}}
}

// Paginated selects whether a list follows later pages.
func Paginated(paginated bool) pagingCall {
	return pagingCall{entryCall: entryCall{CallEntry{Kind: CallPaginated, Bool: paginated}}}
}

// Field sets one additional JSON body attribute of a create, update or action.
// The value is marshaled when the call is constructed; operations reject
// collisions with fields they own.
func Field(key string, value any) fieldCall {
	return fieldCall{entryCall: entryCall{jsonEntry(CallField, key, value)}}
}

// Custom wraps a function over an operation's state struct T. For SDK-owned
// operations T is the operation's exported ...Opts struct; for generated
// native operations T is request.Config of the operation's options type, so
// an existing func(*request.Config[X]) error or request.Option[X] value can
// be wrapped unchanged. Custom implements every category; an operation whose
// state type is not T rejects it with resource.ErrInvalidOption before HTTP.
// The function runs at its position among the other calls. A nil function is
// reported by the operation as a nil option.
func Custom[T any](fn func(*T) error) AnyCall {
	return anyCall{entryCall: entryCall{CallEntry{Kind: CallCustom, Func: fn}}}
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
