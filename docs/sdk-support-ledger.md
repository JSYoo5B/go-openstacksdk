# SDK support ledger

Coverage is evaluated against Gophercloud **v2.15.0** and Python openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**. Generated functions, test counts, and behavior-contract counts are distinct from completed APIs.

The [operation review JSON](../api/sdk_reviews.json) records public Go APIs, source fingerprints, behavior contracts, test selectors, documentation, differences, and unresolved boundaries. The [coverage tables](implementation-plan.md) are generated from these decisions. `make parity` validates references; `make progress` refreshes counts and `make progress-check` checks freshness.

## Status definitions

| Status | Meaning |
|---|---|
| `supported` | Pinned source behavior is implemented with concrete API, contract tests, and usage documentation. |
| `go_mapping` | An intentional Go API with explicit differences in types, defaults, results, or errors provides the functionality. |
| `unsupported` | Behavior is explicitly excluded, with a concrete reason. It is not supported functionality. |
| `unresolved` | Behavior or evidence remains incomplete, including declarations without reviews. Partial verified contracts can coexist with this status. |

A completed review cannot retain unresolved behavior. Go mappings must explain observable differences rather than use type differences as a reason to avoid required functionality. Removing a method from an implementation target does not make the overall SDK complete.

## Deliberate runtime differences

The SDK does not reproduce the following Python runtime behaviors. An operation whose only remaining gaps fall in these categories is recorded as `go_mapping`, with the difference stated in its review.

- **Immutable results.** Operations return owned, immutable snapshots with actual wire and HTTP evidence. Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1), and reuse of that object as later input, is not reproduced.
- **Typed inputs.** Inputs are typed Go values. Arbitrary Python objects, iterables and numeric types accepted by dynamic kwargs have no equivalent.
- **Integer paging controls.** `limit` and `max_items` accept only nonnegative integers; other values fail before HTTP. Python forwards booleans, negatives, fractions and strings, which Neutron and Nova reject with HTTP 400 (both require an integer of 0 or more) or which fail in a client-side comparison.
- **Finite JSON numbers.** Only standard finite JSON numbers are accepted. `NaN`, `Infinity` and overflowing literals such as `1e400` return `ErrUnsupported` with the original response, where Python's `json` loads them as floats. A latest microversion must be written as `"latest"`, not as an overflowing number.
- **No charset guessing.** Text is decoded only from UTF-8, a byte-order mark, or a supported declared charset (UTF-16/UTF-32 with a BOM or explicit endianness, ISO-8859-1). Undeclared non-UTF-8 bytes and other codecs return `ErrUnsupported` with the original bytes, instead of Requests' statistical detection (charset_normalizer) or the full CPython codec registry.
- **Plain JSON rows in JMESPath filters.** JMESPath string filters, which the pinned Python SDK deprecates with `RemovedInSDK60Warning`, evaluate rows as plain JSON objects. Expressions that depend on the Resource class (`keys`, `values`, `length` or `type` of a row), Resource equality with a literal object, or CPython float arithmetic and serialization can differ. Dictionary filters behave the same.

- **No remote configuration.** A clouds.yaml profile named by URL, including the nested URLs of the built-in ovh and vexxhost profiles, is a configuration error. Python fetches such a profile from the vendor's well-known document at connect time, before authentication, so the settings that decide endpoints would come from a remote document. Write the vendor settings in clouds.yaml instead.
- **Polling errors return immediately.** A lookup error while waiting is returned with its cause, following the Gophercloud `WaitFor` convention, and one context deadline bounds the whole wait. Python's cloud `wait_for_server` ignores every exception and keeps polling until its timeout, and passes `timeout - int(elapsed)` to later stages. Retries for transient HTTP failures belong to the `ProviderClient` retry settings.

Caveat: charset guessing matters mainly for Swift object text, whose content is arbitrary user data. An object stored as, for example, undeclared EUC-KR text is returned as an error with its bytes, where Python would guess an encoding. If this becomes a practical problem, an encoding detector can be added as an explicit opt-in; its results would still not match charset_normalizer exactly.

## Server guarantees outside the SDK

Some properties belong to the server, and neither the Python SDK nor this SDK can establish them from a client. Reviews state them as limitations, not as unresolved behavior.

- **Swift consistency.** An accepted upload, a matching status observation or a `HEAD` 404 is an observation of one request at one moment. Replica races, eventual consistency, proxy caching, middleware permissions and backend quota decide when other readers see the change. Segmented uploads cannot be published atomically or rolled back; the SDK keeps unique segment prefixes and cleanup evidence so callers can see what was written.
- **Go I/O conventions.** Streams are `io.ReadCloser` values that the caller must close and read from one goroutine at a time. File output goes to a caller-supplied writer or to a path the SDK opens and closes itself. Payload integrity checks are explicit options, not implied by a read.

## Shared unresolved boundaries

- Full unknown/partial/empty/malformed success-response merging.
- Accepted-status profiles that have not been compared per operation.
- Fetch/constructor/header/microversion controls on lookup bindings that have not adopted the corresponding owned entry points.
- Authenticated deployment acceptance, policy overrides, and release-specific interoperability.

These shared limitations do not erase narrower completed operation contracts. Read each guide's exact input/result/error profile. [Priority planning](implementation-plan.md), [policy branches](core-policy-branches.md), and [release criteria](release-milestones.md) distinguish implementation targets from verification.

## Completion criteria

A support decision needs concrete public inputs, omission/null/false/zero rules, library-owned serialization, defaults, result ownership, and actual error semantics. Where applicable, verify exact/duplicate names, missing policies, paging, cancellation, deletion, waits, versions, and parent identities. Composite operations need ordered phase evidence, partial-success behavior, and documented retry/cleanup policy.

Keep inherited/descriptor/Resource surfaces visible beyond the directly declared catalog. Source inspection, bounded Python execution, Go HTTP fixtures, example builds, authenticated cloud acceptance, and CI are separate evidence. A source fingerprint validates the comparison identity, not test success or complete SDK equivalence.
