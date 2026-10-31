// Package request implements SDK-owned request configuration and extension
// serialization for generated APIs. Callers supply concrete options, never
// builders.
//
// # Shared call options
//
// Operations that take shared call options declare their option type as an
// alias of one sealed category interface, for example
//
//	type CreateOption = request.NativeFieldCall
//
// A category is the combination of controls an operation supports. Each
// shared constructor implements exactly the categories whose combination
// allows its control, so passing a control the operation cannot honor does
// not compile. Custom, and the operation-specific options of a package (its
// With<Op>Options and With<Op><Argument> functions), implement every
// category; an operation rejects one built for another operation's state with
// resource.ErrInvalidOption before HTTP.
//
// SDK-owned operations interpret their calls with their own rules and list the
// calls they implement in their doc comments:
//
//	Category          Shared calls besides Custom
//	GetCall           Header, Headers, Microversion
//	GetQueryCall      Header, Headers, Microversion, Query
//	FindCall          Header, Headers, Microversion, Query, IgnoreMissing
//	DeleteCall        Header, Headers, Microversion, IgnoreMissing
//	ListCall          Header, Headers, Microversion, Query, Limit, Marker,
//	                  MaxItems, Paginated
//	ListBasePathCall  ListCall plus BasePath
//	WriteCall         Header, Headers, Microversion, Field
//
// Generated native operations use the category that matches the extension
// mechanisms of their native builder and apply calls with ApplyCalls:
//
//	Category               Shared calls besides Custom
//	NativeCall             (none)
//	NativeHeaderCall       Header
//	NativeQueryCall        Query
//	NativeQueryHeaderCall  Query, Header
//	NativeFieldCall        Field
//	NativeFieldHeaderCall  Field, Header
//	NativeFieldQueryCall   Field, Query
//
// Field, Header and Query keep the validation and error text of WithField,
// WithHeader, WithQuery and WithQueryValues. Native operations accept Query
// only with a string or []string value; any other value is a JSON keyword
// value for SDK-owned operations and fails on a native operation with
// resource.ErrInvalidOption before HTTP. Custom over Config of the
// operation's options type runs like an Option of that type.
package request
