# Metadef Property

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/get_metadef_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithGetHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithGetResourceType`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetAttributes`

- Zero-HTTP InNamespace validates required literal namespace and captures API/client/original provider/type/endpoint/base/microversion for scope lifetime. Parent and child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected and colon/Unicode/space escaped once. No Ref/find/parentGET or response-driven routing. Each call copies latest ordinary source headers, uses live original authentication and rechecks scope/source/context before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Canonical Property requires a whole validUTF8 nonnullJSONobject. Exact-case optional nullable name/type/title/description/self/schema/literal dates remain pointers; no ID/namespace/name/links seed or time parsing. Unknown keywords including numbers/default remain independent RawMessages in Body; canonical type failures are atomic and retain actual response evidence. Key is separate list-only wire-map-key provenance, nil on CRUD, and never rewrites passive Name or raw Body.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesCanonicalModelsAndRawOwnership`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- One fixed child GET200; ResourceType nil omits query, explicit empty sends resource_type= and nonempty is a literal validUTF8 control-free query. Actual server association prefix stripping is not reproduced locally. Passive returned Name/Key/self/schema cannot retarget scope; no cache reuse, selector, fallback, discovery or follow-up request.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFixedScopedRoutesAndPayloads`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`.
- Named strict getter boundary audit distinguishes Python **query as _get controls or Resource seed attributes from an HTTP query. Existing strict200 fresh DTO retains actual descriptor keywords/default omission/aliases raw, rejects304 without cache reuse and independently decodes later success. WithGetResourceType remains a deliberate Go server-query extension; this raw getter itself does not implement Resource seeding/default/coercion/fetch translation. The additive owned GetRecord contracts below close that formerly unresolved named getter gap.
  Tests: `image/v2/metadefproperties/get_mapping_test.go:TestMetadefPropertyGetMappingDescriptorKeywordsRemainRaw`, `image/v2/metadefproperties/get_mapping_test.go:TestMetadefPropertyGetMappingDoesNotReuseCachedOrSeededResults`.
- GetRecord snapshots concrete ID-or-Resource input before dependency/options callbacks, retains recognized initial attributes, applies option attrs before selecting final ID or alternate name and never mutates the input. String+id attribute is a preflight collision; Resource id attr may retarget before HTTP. Required fixed literal parent/child and zero query/body GET reject unsafe or null/nonstring identity; response fields cannot retarget. Known seed descriptor errors and reserved route/cache/session/class/version controls fail preflight, while unknown ordinary attrs and superseded alias encoding failures are ignored.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordSeedAttributesAndResponseOverlay`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordInputAndAttributesPreflight`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordSupersededAliasDoesNotExposeEncodingFailure`.
- Owned Resource projects eighteen declared Body descriptors plus inherited id, current Connection location and fixed Go namespace_name provenance. Missing min_length/min_items default0 and require_unique_items false; explicit null is retained. Raw untyped JSON, list singleton wrapping and dict object-or-empty fallback, shared integer descriptor and Python truthiness are applied after seed/response overlay. Omitted response fields preserve seeds. Maps prefer canonical presence including null; parsed response aliases follow dictionary member order with duplicate first slot/last value.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordFixedRouteDefaultsAndOwnedReceipts`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordSeedAttributesAndResponseOverlay`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordDescriptorsAliasesAndExplicitNull`.
- Actual200..399 successful fetch always reads/closes once and verifies whole UTF8. Parsed nonnull object overlays declared Body only and retains complete independent Wire; empty/opaque/syntax-invalid JSON returns seeded/default Resource with nil Wire and actual Envelope/Header/status. Valid parsed null/array/scalar and descriptor/read/Close/context/source failures are atomic accepted ResponseErrors preserving receipt and causes. Native400+ errors retain native ownership; no accepted-body replay.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordStatusParsingAndMissingPolicy`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordAcceptedBodyErrorsPreserveAllCauses`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordDescriptorsAliasesAndExplicitNull`.
- RecordGet FullOpts replace configuration, headers and recognized semantic attribute capture with factory/callback ownership before and after. Bulk attrs replace and singular attrs overlay; callbacks run once, latest source ordinary headers are captured and live Provider auth is preserved. Connection location is captured before options and fixed namespace URI means seed/response computed location never replaces it. Resource/Wire/Envelope/headers own independent bytes and only actual response metadata is attached.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordOptionFactoriesAndRetainedStorage`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordPreflightOrderAndCallbackCauses`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordFixedRouteDefaultsAndOwnedReceipts`.
- Getter reuses REST/fixedrequest native prebody retry/reauth/backoff policies with fixed method/target/body/framing/representation/status ownership. Source/namespace/outer/context guards cover callbacks, physical attempts, read/Close and final projection; observed failure stays sticky after target restoration. Retry mutation cannot expand actual accepted range or replay an accepted-body failure.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordNativeRetryPoliciesAndOwnership`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordSourceAndOuterGuardAtPhysicalBoundaries`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordPreflightOrderAndCallbackCauses`.
- Both high-level Connection.Image API and versioned Connection.ImageV2 inject CurrentLocation and share the original service client/provider. Owned configured facts survive caller mutation and option-updated authentication is live; view preserves seeds/default conversion with actual203 receipt and passive response fields. Original typed Get retains strict200 raw Body and additive resource_type query independently.
  Tests: `connection_image_metadef_property_record_test.go:TestConnectionImageMetadefPropertyRecordUsesOwnedLocationAndLiveAuth`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordLegacyGetRemainsRawAndStrict`.

Python/Go differences:

- Go mapping uses an immutable owned concrete RecordRequest and fresh GET; Python mutates existing Resource/cache/dirty/session state and accepts broader dict/Munch/None parent inputs. Go requires fixed literal namespace and selected string child, explicitly rejects route/cache/class/session/version overrides, and does not add finder/discovery or conditional cached returns.
- Resource view has21 fields: pinned Python default to_dict20 plus fixed namespace_name provenance. Unknown/self/schema/date fields remain only in actual Wire/Envelope; Source Body descriptor names, null/defaults, list/dict/bool/int projection and response overlay are owned. Response identity is passive, and canonical map aliases have deterministic precedence while Python kwargs use dictionary order.
- Shared JSON integer projection preserves arbitrary integer JSON and bool, truncates IEEE754 float values, uses Unicode16 digit properties and omits CPython configurable digit-string limits. Bool numeric truthiness compares exact decimal zero, so underflow/JSON backend and nonfinite runtime behavior may differ. Arbitrary Go attributes use encoding/json snapshots, with deferred failures only for finally selected declared attrs.
- Always-on fixed query-free owned fetch accepts actual200..399 and tolerates successful syntax-invalid JSON, but requires wholeUTF8 and parsed nonnull objects. Original strict200 raw typed Get remains a separate API, with WithGetResourceType as a deliberate Go server-query extension. Python public resource_type/value/namespace_name kwargs cause duplicate-binding TypeError beforeHTTP.
- Connection current location including explicitly configured Zone is preserved; pinned Python current_location normally has zone=None. Direct New has null location, while NewWithDependencies supplies a concrete callback. Fixed namespace URI causes seed/response location to remain passive in Wire.
- Local transport fixtures and compile/install builds establish the named getter Go contract; server schema/policy/DB defaults, authenticated side effects and live Python execution are not proven. This mapping does not claim whole Python Resource/session lifecycle parity or change the remaining metadata CRUD/list inventory.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
value = conn.image.get_metadef_property("hw_cpu_policy", "OS::Compute::Libvirt")
print(value.to_dict())
```

```go
service, err := conn.Image(ctx)
if err != nil { return err }
scope, err := service.API.MetadefProperties.InNamespace(ctx, "OS::Compute::Libvirt")
if err != nil { return err }
value, err := scope.Get(ctx, "hw_cpu_policy")
if err != nil { return err }
```

```python
# TypeError before HTTP: Proxy._get already binds resource_type.
value = conn.image.get_metadef_property(
    "hw_cpu_policy", "OS::Compute::Libvirt", resource_type="OS::Nova::Server"
)
```

```go
value, err := scope.Get(ctx, "hw_cpu_policy",
    metadefproperties.WithGetResourceType("OS::Nova::Server"))
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    name := flag.String("name", "hw_cpu_policy", "literal property name")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *name); err != nil { log.Fatal(err) }
}

func printFailure(err error) error {
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    var receipt map[string]any
    switch {
    case errors.As(err, &accepted):
        receipt = map[string]any{"status_code": accepted.StatusCode,
            "header": accepted.Header, "body": string(accepted.Body)}
    case errors.As(err, &unexpected):
        receipt = map[string]any{"status_code": unexpected.Actual,
            "header": unexpected.ResponseHeader, "body": string(unexpected.Body)}
    default:
        return nil
    }
    data, printErr := json.MarshalIndent(receipt, "", "  ")
    if printErr != nil { return printErr }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, name string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    value, err := scope.Get(ctx, name)
    if err != nil { return errors.Join(err, printFailure(err)) }
    data, err := json.MarshalIndent(value.Body, "", "  ")
    if err != nil { return err }
    fmt.Printf("namespace=%q requested_name=%q status_code=%d\n%s\n",
        scope.NamespaceName(), name, value.StatusCode, data)
    return nil
}
```

## References

- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Property Records](metadef-property-records.md)
- [core](v2/metadefproperties/core_test.go)
- [Contracts Test](v2/metadefproperties/contracts_test.go)
- [Options Test](v2/metadefproperties/options_test.go)
- [Connection](../connection_image_metadef_properties_test.go)
- [Get Mapping Test](v2/metadefproperties/get_mapping_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [public getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1900-L1931)
- [MetadefProperty](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L21-L79)
- [Proxy._get_resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L605)
- [Proxy._get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L822-L865)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1849)
- [response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398)
