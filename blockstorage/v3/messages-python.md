# Messages Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/delete_message`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordDeleteHeaders`

- DeleteRecord makes a bodyless query-free DELETE /messages/{literal escaped id}, accepts Source200..399 without JSON-body decoding and defaults to ignoring only a clean native404. Strict false preserves404;403 and joined read/close/cancel/source/retry causes on404 are never suppressed. No predelete Get/List or status/version capability gate is invented.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageDeleteDefaultStrictAndOriginalStatusPolicy`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRejectedStatusFamiliesPreservePhysicalCauses`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageAcceptedPhysicalFaultsAndStickySource`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryPreservesCleanMissingAndFixedVersionHeaders`.
- Owned capture/concrete options retain Source route/provider/type/version across callbacks, physical Read/Close and retries; partial accepted receipts retain real status/body/header, rejected400..599 retain native HTTP evidence plus read/close causes with oneClose. Sticky changes remain failures after restoration. Custom cancel/callback causes, concurrent option reuse and live token/reauth are preserved without mutating the selected shared client.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRejectedStatusFamiliesPreservePhysicalCauses`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageAcceptedPhysicalFaultsAndStickySource`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessagePreflightCancellationAndRetainedOptionOwnership`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageConcurrentOwnedOptionsAndLiveReauthentication`.
- Message Source has no max microversion: absent selected default remains versionless with no invented3.3 negotiation/gate, while any configured/per-operation default is retained. Both selected Cinder version headers or their absence are guarded through physical attempts/retry hooks. Pinned generic raise_from_response accepts valid HTTP100..399; deployment4xx and named delete ignore-missing behavior remain actual server observations.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageGetStrictSourceEnvelopeAndSeedShapes`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageDeleteDefaultStrictAndOriginalStatusPolicy`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryPreservesCleanMissingAndFixedVersionHeaders`.
- Local required-absence HeaderPolicy retains cloudread exclusions through native retry hooks and physical attempts: Authorization, X-Service-Token, Host, Cookie, Content-Length, Transfer-Encoding, Connection, Trailer, TE, Upgrade and foreign X-OpenStack-Nova-API-Version cannot be injected. Selected Cinder version headers cannot be removed via OmitHeaders; absent selected versions remain absent. Clean initial503 evidence and callback cause survive with ErrInvalidOption, exactly one request and one Close. Ordinary retry headers and provider live X-Auth-Token remain available. Transport-exposed199 follows pinned generic valid100..399 success policy and keeps actual response receipts.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryReservedHeaderAbsenceAndSelectedOmission`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryOrdinaryHeadersAndLiveTokenRemainAvailable`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageTransportExposedInterimStatus`.

Python/Go differences:

- Python named delete_message returnsNone; Go returns an error-only result. Accepted delete body/Close/cancellation faults retain actual resource.ResponseError receipts rather than fabricate success, and native rejected evidence remains in the error chain.
- Go owns literal validated UTF8 IDs and independent record/receipt/raw JSON/options; Python accepts mutable Message/Resource/Munch values. Location is an owned null without Python Connection synthesis. Resource/session/subclass/dirty-state/cache/legacy JMESPath and general adapter behavior remain global SDK-R1/C1/S1, not newly promoted named-operation coverage.
- Stock UserMessages require3.3 while pinned Message has no max microversion; the SDK retains absent/default/per-operation selection rather than silently negotiating a new version. Examples select3.37 and actual unsupported-version HTTP errors remain visible, except a clean DELETE404 still follows the named proxy default ignore_missing policy.
- Source/member success200..399 is retained; strict UTF8 and fixed operation origin/path/filter/protocol guards are explicit Go protections. PythonString uses the existing audited shared JSON-domain representation including exact integer values and bounded float/Unicode/runtime spelling differences. Authenticated stock Cinder RBAC/deployment integration is separate from local fixtures.
- Explicit Go request ownership retains initially forbidden auth/framing/foreign-service version header absence across native retry hooks; reserved injection and selected-version OmitHeaders cannot replay. Ordinary native retry headers and provider live authentication remain available; this is not whole Python custom session/adapter parity.

### `python:block_storage/v3/get_message`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.API.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordGetSourceEnvelope`

- GetRecord makes one bodyless/query-free GET /messages/{literal escaped id} with no list fallback and no missing suppression. Actual accepted partial responses preserve ID seeds only in Source view; explicit null/nonstring IDs override them. Blank/malformed JSON keeps seeds, valid wrong shapes or badUTF8 return actual ACK errors.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageGetSourceStockEnvelopesAndStringDescriptors`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageGetStrictSourceEnvelopeAndSeedShapes`.
- Pinned Message.resource_key is plural messages even for member GET; stock Cinder documentation returns singular message. Default Go decoder supports Source messages, flat objects and actual stock message with Source-key presence precedence. WithRecordGetSourceEnvelope(true) reproduces pinned Source seed-only behavior for a stock message wrapper without losing literal Wire/envelope evidence.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageGetSourceStockEnvelopesAndStringDescriptors`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageGetStrictSourceEnvelopeAndSeedShapes`.
- MessageRecord.Resource exposes inherited id/name, eight Source type=str Body descriptors and nullable location; absent/explicitnull are preserved, bool/large integer/container string conversions use shared PythonString. Unknown keys/original types/links are retained separately in Wire, and Resource/Wire/wholeEnvelope/Header own independent copies.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageGetSourceStockEnvelopesAndStringDescriptors`.
- Owned capture/concrete options retain Source route/provider/type/version across callbacks, physical Read/Close and retries; partial accepted receipts retain real status/body/header, rejected400..599 retain native HTTP evidence plus read/close causes with oneClose. Sticky changes remain failures after restoration. Custom cancel/callback causes, concurrent option reuse and live token/reauth are preserved without mutating the selected shared client.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRejectedStatusFamiliesPreservePhysicalCauses`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageAcceptedPhysicalFaultsAndStickySource`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessagePreflightCancellationAndRetainedOptionOwnership`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageConcurrentOwnedOptionsAndLiveReauthentication`.
- Message Source has no max microversion: absent selected default remains versionless with no invented3.3 negotiation/gate, while any configured/per-operation default is retained. Both selected Cinder version headers or their absence are guarded through physical attempts/retry hooks. Pinned generic raise_from_response accepts valid HTTP100..399; deployment4xx and named delete ignore-missing behavior remain actual server observations.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageGetStrictSourceEnvelopeAndSeedShapes`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageDeleteDefaultStrictAndOriginalStatusPolicy`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryPreservesCleanMissingAndFixedVersionHeaders`.
- Local required-absence HeaderPolicy retains cloudread exclusions through native retry hooks and physical attempts: Authorization, X-Service-Token, Host, Cookie, Content-Length, Transfer-Encoding, Connection, Trailer, TE, Upgrade and foreign X-OpenStack-Nova-API-Version cannot be injected. Selected Cinder version headers cannot be removed via OmitHeaders; absent selected versions remain absent. Clean initial503 evidence and callback cause survive with ErrInvalidOption, exactly one request and one Close. Ordinary retry headers and provider live X-Auth-Token remain available. Transport-exposed199 follows pinned generic valid100..399 success policy and keeps actual response receipts.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryReservedHeaderAbsenceAndSelectedOmission`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryOrdinaryHeadersAndLiveTokenRemainAvailable`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageTransportExposedInterimStatus`.

Python/Go differences:

- Default GetRecord intentionally fixes the pinned Source plural-key/stock singular-key mismatch by accepting actual Cinder message envelope. Strict WithRecordGetSourceEnvelope(true) provides the exact pinned Source interpretation. This improvement is explicit and fixture-proven; Source and stock are not silently claimed identical.
- Go owns literal validated UTF8 IDs and independent record/receipt/raw JSON/options; Python accepts mutable Message/Resource/Munch values. Location is an owned null without Python Connection synthesis. Resource/session/subclass/dirty-state/cache/legacy JMESPath and general adapter behavior remain global SDK-R1/C1/S1, not newly promoted named-operation coverage.
- Stock UserMessages require3.3 while pinned Message has no max microversion; the SDK retains absent/default/per-operation selection rather than silently negotiating a new version. Examples select3.37 and actual unsupported-version HTTP errors remain visible, except a clean DELETE404 still follows the named proxy default ignore_missing policy.
- Source/member success200..399 is retained; strict UTF8 and fixed operation origin/path/filter/protocol guards are explicit Go protections. PythonString uses the existing audited shared JSON-domain representation including exact integer values and bounded float/Unicode/runtime spelling differences. Authenticated stock Cinder RBAC/deployment integration is separate from local fixtures.
- Explicit Go request ownership retains initially forbidden auth/framing/foreign-service version header absence across native retry hooks; reserved injection and selected-version OmitHeaders cannot replay. Ordinary native retry headers and provider live authentication remain available; this is not whole Python custom session/adapter parity.

### `python:block_storage/v3/messages`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.WithRecordListMaxItems`

- ListRecords lazily reads /messages plural array/singleton with Source project_id/sort/limit/offset/marker server query classification. Declared nonserver Body attributes including id/name filter locally after string descriptor projection; unknown parameters are ignored. Requests query boolean/iterable/null encoding and Source Python equality/dictionary predicates are reused with actual-page errors for invalid mapping get.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageListQueryClassificationOwnedLocalFiltersAndRawCap`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageListDictionarySemanticsAfterDescriptorConversion`.
- Raw MaxItems counts before local filtering and supplies an initial limit hint when missing/zero. Cap/SinglePage/consumer break stops before unconsumed row/next decoding. Default full iteration supports messages_links, top-level next, HTTP Link, nonempty short-page marker fallback with exact retained wire ID and empty-page termination. Explicit offset-only queries use guarded increasing offsets.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageListLinkedMarkerOffsetAndSinglePage`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageListQueryClassificationOwnedLocalFiltersAndRawCap`.
- List constructor connection/microversion/_synchronized conflicts, null rows, missing/null plural envelopes, decode204 and changed continuation filters fail with actual whole-page status/header/body proof rather than pretending successful list absence. List errors do not suppress404 or manufacture decoded records.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageListEnvelopeConstructorAndContinuationErrors`.
- Owned capture/concrete options retain Source route/provider/type/version across callbacks, physical Read/Close and retries; partial accepted receipts retain real status/body/header, rejected400..599 retain native HTTP evidence plus read/close causes with oneClose. Sticky changes remain failures after restoration. Custom cancel/callback causes, concurrent option reuse and live token/reauth are preserved without mutating the selected shared client.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRejectedStatusFamiliesPreservePhysicalCauses`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageAcceptedPhysicalFaultsAndStickySource`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessagePreflightCancellationAndRetainedOptionOwnership`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageConcurrentOwnedOptionsAndLiveReauthentication`.
- Message Source has no max microversion: absent selected default remains versionless with no invented3.3 negotiation/gate, while any configured/per-operation default is retained. Both selected Cinder version headers or their absence are guarded through physical attempts/retry hooks. Pinned generic raise_from_response accepts valid HTTP100..399; deployment4xx and named delete ignore-missing behavior remain actual server observations.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageGetStrictSourceEnvelopeAndSeedShapes`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageDeleteDefaultStrictAndOriginalStatusPolicy`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryPreservesCleanMissingAndFixedVersionHeaders`.
- Local required-absence HeaderPolicy retains cloudread exclusions through native retry hooks and physical attempts: Authorization, X-Service-Token, Host, Cookie, Content-Length, Transfer-Encoding, Connection, Trailer, TE, Upgrade and foreign X-OpenStack-Nova-API-Version cannot be injected. Selected Cinder version headers cannot be removed via OmitHeaders; absent selected versions remain absent. Clean initial503 evidence and callback cause survive with ErrInvalidOption, exactly one request and one Close. Ordinary retry headers and provider live X-Auth-Token remain available. Transport-exposed199 follows pinned generic valid100..399 success policy and keeps actual response receipts.
  Tests: `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryReservedHeaderAbsenceAndSelectedOmission`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageRetryOrdinaryHeadersAndLiveTokenRemainAvailable`, `blockstorage/v3/messages/python_records_contracts_test.go:TestPythonCinderMessageTransportExposedInterimStatus`.

Python/Go differences:

- Go exposes owned list controls through concrete WithRecordList functions and Query through known server/Body attributes, rather than Python arbitrary control-key kwargs, session/base_path mutation or deprecated JMESPath. Multiple local filters are evaluated in sorted key order and stop at first false because Go maps do not retain kwargs insertion order.
- Go rejects negative typed limits/caps, multiple/invalid pagination controls and unsafe continuation before subsequent HTTP. Dictionary links are an explicit compatibility extension; offset-only pagination follows a strictly increasing offset continuation rather than Source automatic marker fallback. All observations still preserve real received page evidence.
- Go owns literal validated UTF8 IDs and independent record/receipt/raw JSON/options; Python accepts mutable Message/Resource/Munch values. Location is an owned null without Python Connection synthesis. Resource/session/subclass/dirty-state/cache/legacy JMESPath and general adapter behavior remain global SDK-R1/C1/S1, not newly promoted named-operation coverage.
- Stock UserMessages require3.3 while pinned Message has no max microversion; the SDK retains absent/default/per-operation selection rather than silently negotiating a new version. Examples select3.37 and actual unsupported-version HTTP errors remain visible, except a clean DELETE404 still follows the named proxy default ignore_missing policy.
- Source/member success200..399 is retained; strict UTF8 and fixed operation origin/path/filter/protocol guards are explicit Go protections. PythonString uses the existing audited shared JSON-domain representation including exact integer values and bounded float/Unicode/runtime spelling differences. Authenticated stock Cinder RBAC/deployment integration is separate from local fixtures.
- Explicit Go request ownership retains initially forbidden auth/framing/foreign-service version header absence across native retry hooks; reserved injection and selected-version OmitHeaders cannot replay. Ordinary native retry headers and provider live authentication remain available; this is not whole Python custom session/adapter parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.37")
    parser.add_argument("--message", default="")
    parser.add_argument("--project", default="")
    parser.add_argument("--delete", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    if args.message:
        if args.delete:
            conn.block_storage.delete_message(args.message, ignore_missing=False)
        else:
            value = conn.block_storage.get_message(args.message)
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
    else:
        query = {"project_id": args.project} if args.project else {}
        for value in conn.block_storage.messages(**query):
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printMessage(value *resource.RawResource) {
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.37", "Cinder selected version")
    id := flag.String("message", "", "optional message ID")
    project := flag.String("project", "", "optional server project filter")
    remove := flag.Bool("delete", false, "strictly delete the selected message")
    flag.Parse()
    if *remove && *id == "" { log.Fatal("-delete requires -message") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Messages
    if *id != "" {
        if *remove {
            if err := api.DeleteRecord(ctx, *id,
                messages.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
            return
        }
        value, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printMessage(value.Resource)
        return
    }
    options := []messages.RecordListOption{}
    if *project != "" { options = append(options, messages.WithRecordListQuery("project_id", *project)) }
    for value, err := range api.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        printMessage(value.Resource)
    }
}
```

## References

- [Message](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/message.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3029-L3064)
- [Install](../../docs/install.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#show-message-details)
- [Policy Personas](https://docs.openstack.org/cinder/2026.1/configuration/block-storage/policy-personas.html#user-messages-microversion-3-3)
- [Policy](https://docs.openstack.org/cinder/2026.1/configuration/block-storage/policy.html)
- [Python Records Contracts Test](messages/python_records_contracts_test.go)
