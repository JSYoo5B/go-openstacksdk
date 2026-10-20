# Design

## Layers and ownership

The SDK provides Connection, service, resource, and workflow layers using explicit Go types and errors. Gophercloud supplies authentication, HTTP execution, native serializers, and response models. Connection selects configuration, TLS/HTTP policy, region/interface, endpoints, microversion policies, and cached successful service instances. Services expose APIs and collections. Collections implement references, exact names, duplicates, missing policies, iteration, and waits. Workflows resolve dependencies across services, order mutations and waits, and preserve partial success.

See the [upstream SDK layout](https://docs.openstack.org/openstacksdk/latest/contributor/layout.html). Similar API names do not establish equivalent behavior.

## Inputs and defaults

Required inputs use concrete request structs. Optional behavior uses operation-specific options, preventing unrelated option types from mixing. Defaults are constructed per call. Invalid or nil options fail before mutation where the operation can validate them locally.

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
