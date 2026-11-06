# Design

## Layers and ownership

The SDK provides Connection, service, resource, and workflow layers using explicit Go types and errors. Gophercloud supplies authentication, HTTP execution, native serializers, and response models. Connection selects configuration, TLS/HTTP policy, region/interface, endpoints, microversion policies, and cached successful service instances. Services expose APIs and collections. Collections implement references, exact names, duplicates, missing policies, iteration, and waits. Workflows resolve dependencies across services, order mutations and waits, and preserve partial success.

See the [upstream SDK layout](https://docs.openstack.org/openstacksdk/latest/contributor/layout.html). Similar API names do not establish equivalent behavior.

## Inputs and defaults

Required inputs use concrete request structs. Optional behavior uses typed options; controls an operation cannot honor do not compile. Defaults are constructed per call. Invalid or nil options fail before mutation where the operation can validate them locally.

Generated native operations and hand-written service operations take shared per-call controls. Each operation keeps its variadic signature, and its option type is an alias of a sealed call category of package `request`, named for the combination of controls the operation supports (the package documentation lists each combination). A control outside the operation's category does not compile; a control the category admits but the operation never supported fails with `request.UnsupportedCall` (`resource.ErrInvalidOption`) before HTTP. One call can mix shared values built by `request.Field`, `request.Query`, `request.Header` and, for SDK-owned categories, `request.Headers`, `request.Microversion`, `request.IgnoreMissing`, `request.BasePath` and the paging constructors, with the package's operation-specific options and `request.Custom` over the operation's state. Operation-specific options built for another operation, or a `request.Custom` over another state type, fail with `resource.ErrInvalidOption` before HTTP. Calls only snapshot their inputs, and each operation keeps its own validation and ordering rules.

Generated native operations use the `request.Native*Call` categories derived from their native builder's extension mechanisms and build their configuration with `request.ApplyCalls`, which keeps the validation and error text of the generic `request.WithField`, `request.WithQuery`, `request.WithQueryValues` and `request.WithHeader` options. They keep `With<Op>Options` and `With<Op><Argument>`; `request.Custom` there takes a `func(*request.Config[<Op>Opts]) error`. Native query serialization carries text only, so `request.Query` accepts a string or a `[]string` (a repeated key) and rejects other values before HTTP.

Hand-written operations use the SDK-owned categories `request.GetCall`, `request.GetQueryCall`, `request.FindCall`, `request.DeleteCall`, `request.ListCall`, `request.ListBasePathCall` and `request.WriteCall`; an option type shared by several operations uses the narrowest category that covers all of them. They keep their `With*Opts` snapshots and accept `request.Custom` over the exported `...Opts` struct; `internal/calls` translates shared calls into each operation's state in caller order. Each method's doc comment or package guide lists its supported calls. Operations that take JSON keyword values send a `request.Query` string as a JSON string and a `[]string` as a JSON array; `request.Query` and `request.Field` marshal their value when constructed, so they also evaluate a kwarg that the operation then discards. A package keeps its own option where the shared constructor would change behavior: header options that validate and merge into the collected headers (`request.Headers` replaces them and is rejected there), field and query options that check the key before marshaling the value, and controls that no category admits together with the operation's other controls, such as find paging or a query on a write.

`request.Optional[T]` distinguishes omitted, null, and present values: zero omits a field; `request.Null[T]()` sends null; `request.Present(value)` sends false, zero, or empty values explicitly. Operations validate their own field rules. Callers use concrete options rather than implementing serialization builders.

Extension fields use JSON snapshots and reject protected-field collisions. This does not make arbitrary extension schemas statically safe. Per-operation capability checks remain distinct from connection-level microversion selection.

## References, scopes, and pagination

`resource.ID` and `resource.Name` make caller intent explicit, including names that resemble IDs. Ordinary exact-name lookup checks all relevant pages for duplicates. Automatic identity-finding APIs have their own documented GET/fallback policies; do not infer those policies from ordinary Find.

Parent scopes fix routing identities once. Heat requires a stack name+ID pair, unlike ordinary ID-only scopes. Iterators fetch pages lazily and stop on consumer break. Continuation policies preserve origin, collection paths, and filters; cycles return `ErrPaginationCycle`. Whole-page decoding can fail before a local consumption cap reaches an otherwise valid row.

## Concurrency and failures

Successful service construction is cached under synchronization. Services share the live ProviderClient and token rather than cloning authentication state for endpoint overrides. Prepare HTTP and raw-client configuration before concurrent requests; concurrent mutation of RawClient settings is unsupported.

Returned values are not an automatic persistence cache. Owned metadata preserves raw fields, null/omission, exact numbers, headers, and status as specified by the operation. Accepted response read/decode errors retain partial data and causes in `resource.ResponseError`. An ambiguous mutation result must not trigger automatic replay or rollback. A create can return a resource and a later wait error together; cancellation does not undo remote execution.

## SDK-owned APIs

Senlin, Masakari, and other additions use internal transport helpers while declaring their own routes, envelopes, statuses, identifiers, and microversion gates. A common transport implementation does not establish completion of each service operation. Track support in the [ledger](sdk-support-ledger.md) and operation-specific guides.
