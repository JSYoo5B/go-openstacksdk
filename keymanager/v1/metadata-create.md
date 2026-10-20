# Metadata Create

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/create_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithCreateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithCreateRecordAttributes`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithCreateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithCreateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithCreateRecordHeader`

- Pinned _proxy.py:47-56 accepts optional **attrs and delegates Resource.new/create. CreateRecord selects the finite declared Container Body attributes, including inherited id, created_at/created, updated_at/updated, container_ref/container_id, name, type, status, secret_refs and consumers, maps aliases, discards unknown semantic values before JSON encoding and retains supplied null/empty/raw JSON values without inferred defaults. Concrete options own selected values, replace the semantic set with bulk options and apply individual values by canonical wire target; explicit extension fields and headers remain separate. Shared Container option evidence verifies canonical-wire precedence, nested/large-number snapshots, extension/header survival, callback-once preparation and concurrent reuse; each kind has its own nullable/body evidence.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`.
- Creation sends a flat POST to the captured /containers collection with no identity suffix, query, lookup, returned-reference traversal, follow-up metadata/payload GET, wait or cleanup. Empty semantic attributes send {}. Original raw JSON request values are retained before result descriptor conversion, including the tested scalar/list/dict values; Order's sole meta object stays a top-level field rather than becoming an envelope. Leaf and cached Connection fixtures exercise each kind with the selected ResourceBase, live token, source/call headers and microversion.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`.
- The returned Container Resource starts with declared input seed and overlays actual declared response attributes: omitted response values retain input while explicit null/empty replace it. Literal id takes precedence, otherwise id is the full passive alternate reference; convenience IDs are separately nullable reference suffixes. List/dict descriptors convert only the owned view. Wire, Envelope, Header and StatusCode describe only the actual POST response, and result copies are independent; they do not synthesize requested fields or executable request identities. Kind-specific seed/null and descriptor assertions cover each resource, and shared Container mutation assertions cover receipt/view/ID ownership.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`.
- The shared owned create engine accepts final 200..399 and strictly decodes a non-null UTF-8 JSON root object. Representative Container HTTP evidence covers 200/201/202/299/399 success, malformed/null/array/invalid-UTF-8/empty-204 bodies and invalid response reference, retaining accepted status/header/body in partial result and ResponseError; a valid decoded raw response remains in Wire even when view formatting fails. Accepted read/Close/source/context failures preserve original and custom causes without SDK composition resend or missing suppression. Rejected 403/404/409 preserve native method/URL/expected codes/body/header. Shared-engine evidence is representative rather than a repeated fault matrix for every kind.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordAcceptedReceiptsAndNativeFailures`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordPreflightSourceContextAndNativeReplay`.
- The shared authenticated client/provider is captured before user callbacks; options are applied once with source/context checks before and after preparation and at physical HTTP attempts. Invalid API/context/provider/service/options, known-field extension collisions, selected JSON errors, bad input reference, unsupported query/argument carriers, source mutation and pre-cancellation fail before HTTP; custom cancellation is preserved. Configured native retry can reuse the owned body and a refreshed token, source mutation stops a second attempt with native proof, and expanding retry OkCodes cannot reclassify a rejected 404 as accepted creation. Existing native Create aliases/signatures and response-only extraction remain separate: Container/Secret still require201, Order202, and Container still requires native Type.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordPreflightSourceContextAndNativeReplay`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`, `api/keymanager_create_records_test.go:TestKeyManagerNativeCreateAliasesBodiesAndStatusRemainUnchanged`.

Python/Go differences:

- CreateRecord is an additive source-shaped mapping alongside Container's unchanged native Create API. It returns an owned input-seeded Resource view and actual POST Wire/Envelope/HTTP evidence, with nullable passive convenience IDs; Python returns a mutable Resource object. Native model aliases, concrete builders and strict201/202 policies are separate APIs and reviews.
- Concrete attribute options replace Python kwargs. Canonical wire names win within one map and the last individual option for that canonical target wins across options, instead of Python dict insertion-sensitive alias collisions. Bulk options replace semantic attributes while explicit extension/header values survive; unknown semantic attributes are discarded before encoding and explicit fields cannot override declared wire names or aliases.
- JSON raw request values remain separate from descriptor-converted view values. Go preserves raw numbers and JSON null/empty values, converts list scalars to one-element view arrays and non-null non-object dict values to empty view objects, and leaves timestamp values unchanged. Non-JSON Python objects and all set/tuple conversions are outside the typed JSON mapping.
- Go accepts final200..399 but requires a non-null UTF-8 root JSON object and valid passive reference formatters. Generic Python Resource._translate_response accepts status<400 and may swallow JSON ValueError while keeping input seed. Go instead retains accepted HTTP proof, exposes processing failures and returns no successful Resource view after such failure; a valid decoded actual Wire may still be present.
- SDK composition does not resend POST for accepted read/Close/decode/formatter/source/context failures. Configured Provider native retry and reauthentication remain transport behavior; the shared physical guard permits safe live-token retry, stops source drift, and keeps terminal native rejection evidence if a retry expands OkCodes.
- Known input references are validated during source-shaped construction before POST; input and returned references are passive data and do not change the captured collection route. Creation performs no metadata/payload fetch, wait or cleanup. An accepted Order202 does not imply that its generated secret is ready.
- Computed location is null in the Go owned view. Full mutable Resource dirty/cache/session behavior and generic adapter/base_path/header/version injection remain separate original goal scope; unrelated List/Get/Delete/Find/Wait and native Create declarations are not promoted by this named-operation review.

### `python:key_manager/v1/create_order`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithCreateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithCreateRecordAttributes`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithCreateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithCreateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithCreateRecordHeader`

- Pinned _proxy.py:158-167 accepts optional **attrs and delegates Resource.new/create. CreateRecord selects the finite declared Order Body attributes, including inherited id/name, created_at/created, updated_at/updated, creator_id, meta, type, status, sub_status/sub_status_message, order_ref/order_id and secret_ref/secret_id, maps aliases, discards unknown semantic values before JSON encoding and retains supplied null/empty/raw JSON values without inferred defaults. Concrete options own selected values, replace the semantic set with bulk options and apply individual values by canonical wire target; explicit extension fields and headers remain separate. Shared Container option evidence verifies canonical-wire precedence, nested/large-number snapshots, extension/header survival, callback-once preparation and concurrent reuse; each kind has its own nullable/body evidence.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`.
- Creation sends a flat POST to the captured /orders collection with no identity suffix, query, lookup, returned-reference traversal, follow-up metadata/payload GET, wait or cleanup. Empty semantic attributes send {}. Original raw JSON request values are retained before result descriptor conversion, including the tested scalar/list/dict values; Order's sole meta object stays a top-level field rather than becoming an envelope. Leaf and cached Connection fixtures exercise each kind with the selected ResourceBase, live token, source/call headers and microversion.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`.
- The returned Order Resource starts with declared input seed and overlays actual declared response attributes: omitted response values retain input while explicit null/empty replace it. Literal id takes precedence, otherwise id is the full passive alternate reference; convenience IDs are separately nullable reference suffixes. List/dict descriptors convert only the owned view. Wire, Envelope, Header and StatusCode describe only the actual POST response, and result copies are independent; they do not synthesize requested fields or executable request identities. Kind-specific seed/null and descriptor assertions cover each resource, and shared Container mutation assertions cover receipt/view/ID ownership.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`.
- The shared owned create engine accepts final 200..399 and strictly decodes a non-null UTF-8 JSON root object. Representative Container HTTP evidence covers 200/201/202/299/399 success, malformed/null/array/invalid-UTF-8/empty-204 bodies and invalid response reference, retaining accepted status/header/body in partial result and ResponseError; a valid decoded raw response remains in Wire even when view formatting fails. Accepted read/Close/source/context failures preserve original and custom causes without SDK composition resend or missing suppression. Rejected 403/404/409 preserve native method/URL/expected codes/body/header. Shared-engine evidence is representative rather than a repeated fault matrix for every kind.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordAcceptedReceiptsAndNativeFailures`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordPreflightSourceContextAndNativeReplay`.
- The shared authenticated client/provider is captured before user callbacks; options are applied once with source/context checks before and after preparation and at physical HTTP attempts. Invalid API/context/provider/service/options, known-field extension collisions, selected JSON errors, bad input reference, unsupported query/argument carriers, source mutation and pre-cancellation fail before HTTP; custom cancellation is preserved. Configured native retry can reuse the owned body and a refreshed token, source mutation stops a second attempt with native proof, and expanding retry OkCodes cannot reclassify a rejected 404 as accepted creation. Existing native Create aliases/signatures and response-only extraction remain separate: Container/Secret still require201, Order202, and Container still requires native Type.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordPreflightSourceContextAndNativeReplay`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`, `api/keymanager_create_records_test.go:TestKeyManagerNativeCreateAliasesBodiesAndStatusRemainUnchanged`.

Python/Go differences:

- CreateRecord is an additive source-shaped mapping alongside Order's unchanged native Create API. It returns an owned input-seeded Resource view and actual POST Wire/Envelope/HTTP evidence, with nullable passive convenience IDs; Python returns a mutable Resource object. Native model aliases, concrete builders and strict201/202 policies are separate APIs and reviews.
- Concrete attribute options replace Python kwargs. Canonical wire names win within one map and the last individual option for that canonical target wins across options, instead of Python dict insertion-sensitive alias collisions. Bulk options replace semantic attributes while explicit extension/header values survive; unknown semantic attributes are discarded before encoding and explicit fields cannot override declared wire names or aliases.
- JSON raw request values remain separate from descriptor-converted view values. Go preserves raw numbers and JSON null/empty values, converts list scalars to one-element view arrays and non-null non-object dict values to empty view objects, and leaves timestamp values unchanged. Non-JSON Python objects and all set/tuple conversions are outside the typed JSON mapping.
- Go accepts final200..399 but requires a non-null UTF-8 root JSON object and valid passive reference formatters. Generic Python Resource._translate_response accepts status<400 and may swallow JSON ValueError while keeping input seed. Go instead retains accepted HTTP proof, exposes processing failures and returns no successful Resource view after such failure; a valid decoded actual Wire may still be present.
- SDK composition does not resend POST for accepted read/Close/decode/formatter/source/context failures. Configured Provider native retry and reauthentication remain transport behavior; the shared physical guard permits safe live-token retry, stops source drift, and keeps terminal native rejection evidence if a retry expands OkCodes.
- Known input references are validated during source-shaped construction before POST; input and returned references are passive data and do not change the captured collection route. Creation performs no metadata/payload fetch, wait or cleanup. An accepted Order202 does not imply that its generated secret is ready.
- Computed location is null in the Go owned view. Full mutable Resource dirty/cache/session behavior and generic adapter/base_path/header/version injection remain separate original goal scope; unrelated List/Get/Delete/Find/Wait and native Create declarations are not promoted by this named-operation review.

### `python:key_manager/v1/create_secret`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithCreateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithCreateRecordAttributes`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithCreateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithCreateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithCreateRecordHeader`

- Pinned _proxy.py:259-268 accepts optional **attrs and delegates Resource.new/create. CreateRecord selects the finite declared Secret Body attributes, including inherited id, created_at/created, updated_at/updated, expires_at/expiration, algorithm, bit_length, mode, name, secret_type, status, secret_ref/secret_id, content_types, payload and payload_content_type/payload_content_encoding, maps aliases, discards unknown semantic values before JSON encoding and retains supplied null/empty/raw JSON values without inferred defaults. Concrete options own selected values, replace the semantic set with bulk options and apply individual values by canonical wire target; explicit extension fields and headers remain separate. Shared Container option evidence verifies canonical-wire precedence, nested/large-number snapshots, extension/header survival, callback-once preparation and concurrent reuse; each kind has its own nullable/body evidence.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`.
- Creation sends a flat POST to the captured /secrets collection with no identity suffix, query, lookup, returned-reference traversal, follow-up metadata/payload GET, wait or cleanup. Empty semantic attributes send {}. Original raw JSON request values are retained before result descriptor conversion, including the tested scalar/list/dict values; Order's sole meta object stays a top-level field rather than becoming an envelope. Leaf and cached Connection fixtures exercise each kind with the selected ResourceBase, live token, source/call headers and microversion.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`.
- The returned Secret Resource starts with declared input seed and overlays actual declared response attributes: omitted response values retain input while explicit null/empty replace it. Literal id takes precedence, otherwise id is the full passive alternate reference; convenience IDs are separately nullable reference suffixes. List/dict descriptors convert only the owned view. Wire, Envelope, Header and StatusCode describe only the actual POST response, and result copies are independent; they do not synthesize requested fields or executable request identities. Kind-specific seed/null and descriptor assertions cover each resource, and shared Container mutation assertions cover receipt/view/ID ownership.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`.
- The shared owned create engine accepts final 200..399 and strictly decodes a non-null UTF-8 JSON root object. Representative Container HTTP evidence covers 200/201/202/299/399 success, malformed/null/array/invalid-UTF-8/empty-204 bodies and invalid response reference, retaining accepted status/header/body in partial result and ResponseError; a valid decoded raw response remains in Wire even when view formatting fails. Accepted read/Close/source/context failures preserve original and custom causes without SDK composition resend or missing suppression. Rejected 403/404/409 preserve native method/URL/expected codes/body/header. Shared-engine evidence is representative rather than a repeated fault matrix for every kind.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordAcceptedReceiptsAndNativeFailures`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordPreflightSourceContextAndNativeReplay`.
- The shared authenticated client/provider is captured before user callbacks; options are applied once with source/context checks before and after preparation and at physical HTTP attempts. Invalid API/context/provider/service/options, known-field extension collisions, selected JSON errors, bad input reference, unsupported query/argument carriers, source mutation and pre-cancellation fail before HTTP; custom cancellation is preserved. Configured native retry can reuse the owned body and a refreshed token, source mutation stops a second attempt with native proof, and expanding retry OkCodes cannot reclassify a rejected 404 as accepted creation. Existing native Create aliases/signatures and response-only extraction remain separate: Container/Secret still require201, Order202, and Container still requires native Type.
  Tests: `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordPreflightSourceContextAndNativeReplay`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse`, `api/keymanager_create_records_test.go:TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof`, `api/keymanager_create_records_test.go:TestKeyManagerNativeCreateAliasesBodiesAndStatusRemainUnchanged`.

Python/Go differences:

- CreateRecord is an additive source-shaped mapping alongside Secret's unchanged native Create API. It returns an owned input-seeded Resource view and actual POST Wire/Envelope/HTTP evidence, with nullable passive convenience IDs; Python returns a mutable Resource object. Native model aliases, concrete builders and strict201/202 policies are separate APIs and reviews.
- Concrete attribute options replace Python kwargs. Canonical wire names win within one map and the last individual option for that canonical target wins across options, instead of Python dict insertion-sensitive alias collisions. Bulk options replace semantic attributes while explicit extension/header values survive; unknown semantic attributes are discarded before encoding and explicit fields cannot override declared wire names or aliases.
- JSON raw request values remain separate from descriptor-converted view values. Go preserves raw numbers and JSON null/empty values, converts list scalars to one-element view arrays and non-null non-object dict values to empty view objects, and leaves timestamp values unchanged. Non-JSON Python objects and all set/tuple conversions are outside the typed JSON mapping.
- Go accepts final200..399 but requires a non-null UTF-8 root JSON object and valid passive reference formatters. Generic Python Resource._translate_response accepts status<400 and may swallow JSON ValueError while keeping input seed. Go instead retains accepted HTTP proof, exposes processing failures and returns no successful Resource view after such failure; a valid decoded actual Wire may still be present.
- SDK composition does not resend POST for accepted read/Close/decode/formatter/source/context failures. Configured Provider native retry and reauthentication remain transport behavior; the shared physical guard permits safe live-token retry, stops source drift, and keeps terminal native rejection evidence if a retry expands OkCodes.
- Known input references are validated during source-shaped construction before POST; input and returned references are passive data and do not change the captured collection route. Creation performs no metadata/payload fetch, wait or cleanup. An accepted Order202 does not imply that its generated secret is ready.
- Computed location is null in the Go owned view. Full mutable Resource dirty/cache/session behavior and generic adapter/base_path/header/version injection remain separate original goal scope; unrelated List/Get/Delete/Find/Wait and native Create declarations are not promoted by this named-operation review.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
)

func main() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	secret, err := service.Secrets.CreateRecord(ctx, secrets.WithCreateRecordAttributes(map[string]any{
		"name": "stored-demo", "secret_type": secrets.OpaqueSecret,
		"payload": "demo", "payload_content_type": "text/plain",
	}))
	if err != nil {
		log.Fatal(err)
	}
	var secretRef string
	if err := json.Unmarshal(secret.Resource.Body["secret_ref"], &secretRef); err != nil || secretRef == "" {
		log.Fatalf("created secret has no usable reference: %v", err)
	}
	container, err := service.Containers.CreateRecord(ctx,
		containers.WithCreateRecordAttribute("type", containers.GenericContainer),
		containers.WithCreateRecordAttribute("name", "container-demo"),
		containers.WithCreateRecordAttribute("secret_refs", []map[string]any{{"name": "stored", "secret_ref": secretRef}}),
	)
	if err != nil {
		log.Fatal(err)
	}
	order, err := service.Orders.CreateRecord(ctx, orders.WithCreateRecordAttributes(map[string]any{
		"type": orders.KeyOrder, "meta": map[string]any{"algorithm": "aes", "bit_length": 256, "mode": "cbc"},
	}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("secret=%s container=%s order=%s\n", secret.Resource.Body["id"], container.Resource.Body["id"], order.Resource.Body["id"])
}
```

## References

- [Testing](../../docs/testing.md)
- [Implementation Plan](../../docs/implementation-plan.md)
- [SDK Support Ledger](../../docs/sdk-support-ledger.md)
