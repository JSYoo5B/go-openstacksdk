# Support reference validation

Run from the SDK root:

```sh
go run ./internal/cmd/paritycheck
go run ./internal/cmd/paritycheck -sync
```

The default command validates pinned inventories, the complete catalog, and durable reviews. `-sync` adds new catalog operations without changing review decisions. Missing decisions remain unresolved. Source drift, deleted operations, invalid input, or failed writes preserve the existing catalog; writes use a temporary file followed by rename. `-root` selects another repository checkout.

Operation IDs combine source and declaration, for example `gophercloud:compute/v2/servers.GetPassword`. Fingerprints bind the source pin and declaration metadata, excluding candidate names and support decisions. They do not independently parse function bodies or establish semantic equivalence. Review the source before changing a fingerprint.

Supported/go_mapping decisions require public Go API references, behavior contracts, valid test selectors, documentation, and no remaining work. Go mappings must explain observable differences. Unresolved/unsupported entries require concrete remaining boundaries. Partial verified contracts can remain attached to an unresolved operation.

Validation rejects unknown fields, duplicate JSON keys/IDs, missing catalog operations, source drift, and missing APIs/tests/docs. It checks public Go declarations and test signatures through ASTs, but does not infer promoted methods or prove test semantics/execution. Internal packages cannot be cited as public SDK APIs.

See [inventories](../../../api/README.md), [review JSON](../../../api/sdk_reviews.json), and [support definitions](../../../docs/sdk-support-ledger.md). Direct declarations are distinct from inherited, descriptor, Resource, session, and cache coverage.
