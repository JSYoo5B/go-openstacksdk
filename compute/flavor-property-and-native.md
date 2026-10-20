# Flavor Property And Native

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/get_flavor_extra_specs_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.GetExtraSpecsProperty`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsHeader`

- GetExtraSpecsProperty reads only the selected flavor/property member route with an empty body and no query, flavor GET, list or whole-specs GET. The concrete ID/native Flavor/RawResource input and selected source are captured before option callbacks; inline specs never skip this read, explicit ID determines the route, and the supplied existing extra_specs remain unchanged.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- The selected Value remains exact owned JSON rather than a string/dict conversion: string, precise large number, array, object and explicit null are retained. Missing properties return Value null and Present false, while explicit null is Present true. Wire contains only the actual flat response, and Value/Envelope/Header do not alias Wire. Actual final200/201/202/203/300 payloads retain their body/header/status receipt.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`.
- The property uses the same concrete read-options/source/version engine as FetchExtraSpecs: selected2.100 is retained, an unselected client performs one finite discovery and chooses2.61, and explicit empty is versionless without changing the caller client. Options execute once before discovery/member GET; input identity is already snapshotted when a callback changes caller models. Empty/dot/control property selections fail before callbacks and HTTP.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`, `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyPreflightAndBindingTerminalReceipts`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts`.
- Actual HTTP404 stays a strict native failure with original method/URL/body/header and no missing normalization. Accepted read/Close causes, source drift and nonobject response roots retain the actual partial receipt with nil Value/Wire and a ResponseError; nested404 cannot become ignored missing. Accepted processing errors make no SDK resend or native RetryFunc callback, while native provider handling of an actual HTTP failure is preserved.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyPreflightAndBindingTerminalReceipts`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts`.

Python/Go differences:

- Python accepts an ID or connected mutable Flavor; Go uses the concrete FlavorExtraSpecsRequest input and independent record, snapshots existing values, and exposes Present to distinguish missing from explicit null. The scalar selection does not update extra_specs or synthesize a Flavor Resource view.
- Python's cast(str | None, response.json().get(prop)) performs no runtime conversion. Go likewise retains arbitrary actual member JSON in Value, including exact large integers, rather than using native GetExtraSpec's string map.
- Go validates a nonempty valid property without controls/dot segments and escapes one literal path component, including slash/percent/space/reserved characters. Python utils.urljoin strips slash components; Go also requires a safe string route identity instead of allowing Python's broader dynamic identity values.
- Owned member reads accept final200..399 and require a valid UTF-8 nonnull JSON object; accepted failures include physical receipt and cause. WithFlavorExtraSpecsOptions/Microversion/Header are Go extensions; source's public named getter has no query/filter kwargs. Selected or finite2.61/versionless policy is shared with the separately reviewed FetchExtraSpecs.
- Full generic mutable Resource/session/cache and other flavor list/find declarations remain separate SDK goal work rather than additional required branches of this fixed single-property getter.

### `gophercloud:compute/v2/flavors.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Get`

- Get binds one bodyless queryless native GET to the selected ResourceBase flavors/{id} member using the live token/source headers and both selected Nova version headers. Native Flavor fields, unknown-field omission, numeric/string/blank swap conversion, missing public/specs zero defaults and nullable fields are preserved. Missing/null flavor wrappers return nil,nil; known-field and swap decode failures retain the native allocated/partially assigned Flavor together with their typed/numeric cause.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetNativeBindingDTOAndOriginalHTTPErrors`.
- Only native200 succeeds. Actual203/204/403 remain native failures with nil result, expected200, original method/URL/body/header and selected client unchanged; no list or specs lookup is introduced.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetNativeBindingDTOAndOriginalHTTPErrors`.

Python/Go differences:

- Go owns the exported concrete facade and wraps upstream errors while preserving native Get/Extract behavior and partial values. This native method does not add Cloud GetFlavorByID safe-result validation, ignored missing, Find fallback or optional specs enrichment.
- The returned Flavor is the native typed DTO; it does not synthesize Python descriptor defaults, unknown/location fields or connected mutable Resource state.

### `gophercloud:compute/v2/flavors.GetExtraSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.GetExtraSpec`

- GetExtraSpec makes one bodyless queryless native GET to flavors/{id}/os-extra_specs/{key}, retaining the selected token and both Nova version headers. Its result is the complete flat map[string]string, including unrelated string keys and a null member as an empty string; a missing requested property does not reduce the returned map. Null root retains nil map and empty object retains nonnil empty map.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecNativeBindingReturnsWholeStringMap`.
- Numeric members retain the native partial map plus UnmarshalTypeError and an array root returns the native type failure. Only200 succeeds; actual203/204/404 preserve nil result and the original expected-code/method/URL/body/header proof without flavor lookup or ignored missing.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecNativeBindingReturnsWholeStringMap`.

Python/Go differences:

- The library-owned facade preserves native route/result/status ABI and wraps errors; this whole string map is deliberately distinct from the owned raw scalar GetExtraSpecsProperty API.
- Native null string values become empty strings and nonstring JSON values produce native partial decode errors. The caller must check error even when a map is returned.

### `gophercloud:compute/v2/flavors.ListDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithListDetailOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithListDetailQuery`

- ListDetail owns a concrete request.Options builder while preserving the fixed flavors/detail native route. Default typed options omit is_public and all other query values. All eight native ListOpts fields reach their original wire spelling, raw limit/is_public extensions override typed values, and vendor query is retained. Options execute once at sequence construction while HTTP remains lazy; bodyless requests use the selected token and both version headers.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`.
- Native Flavor/Swap/inline-specs DTOs and flavors_links continuation are preserved across two pages. Native200 empty and204 return no rows,300 page is accepted, and consumer break never fetches the next page. An empty native page stops even when it advertises a next link; no Python summary route, is_public default or item enrichment is invented.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsOrdinaryCollectionsRetainQueryWithoutIdentityPolicies`.
- Late403 retains already yielded rows and yields one terminal error with original method/URL/body/header and expected200/204/300. First203, malformed JSON and typed malformed Flavor rows yield one terminal native error. A failing option produces no HTTP and retains the original callback cause.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`.

Python/Go differences:

- Go exposes iter.Seq2[*Flavor,error] over the upstream native pager rather than returning a Pager and requiring caller extraction. Native FlavorPage/status/query behavior is preserved.
- Native fixed detailed-list paging does not add Python flavors(details=False), per-item get_extra_specs, semantic alias/Body filters, HTTP Link or marker fallback. Those separate public Python list/find declarations retain their own completion work.
- PrivateAccess/AllAccess behavior remains server-policy dependent; this mapping does not imply those requests are authorized for every core-user deployment.

### `python:cloud/_compute/ComputeCloudMixin/get_flavor_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetFlavorByID`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorByIDOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorByIDExtraSpecs`

- GetFlavorByID owns Cloud's default-false get_extra flag and strict GET-only behavior: one native flavor GET, no query/body/list/automatic discovery, and400/403/404 stay original HTTP errors rather than name fallback or nil success. It returns canonical native flavor fields and keeps the selected client2.55/live token.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsMissingPoliciesAndUnsafeNames`.
- Omitted/false enrichment makes no specs request. True fetches canonical returned ID once for missing/null/empty inline specs and keeps a nonempty inline map without another GET. Unrelated typed flavor fields are retained, caller query never leaks to specs, and actual specs404 is terminal. The existing native enrichment engine also preserves its missing/null-map and typed/transport/read/cancel error policy.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsEnrichmentFailuresAreTerminal`.
- Typed FlavorByID options are reusable captured values and execute once per call. Nil service/context/option, unsafe ID and unsupported header/query/field/argument configurations fail before HTTP; source drift or option/cancellation error stops later callbacks and retains the original custom option/context causes.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDTypedOptionsAndPreflight`.
- The Cloud entry delegates to the already reviewed strict FindFallbackNever/IgnoreMissing(false) proxy-get mapping and checks the captured service/client/component binding around options and completion. The shared GET/enrichment implementation preserves live provider/version/context and rejects unsafe or null successful identity models before returning a Cloud result.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDTypedOptionsAndPreflight`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`.

Python/Go differences:

- The pinned Cloud declaration delegates get_extra=False to compute.get_flavor and introduces no independent query/filter/list behavior. Go's concrete bool options own the strict Never/IgnoreMissing(false) combination instead of requiring callers to assemble it.
- Go returns the native Flavor DTO and requires a safe explicit input/returned ID. Native missing/null ExtraSpecs stays nil and missing IsPublic stays false, with no seeded mutable Python Resource identity/location/default reconstruction.
- The existing native GET/specs calls retain the configured microversion and native strict200 policy without automatic source Resource version discovery. Cloud binding checks do not replace the native provider's physical retry/reauth machine with owned MemberGet; the single-property API has its separate owned transport.
- Other Cloud/proxy flavor list/find/search and generic Resource/session/cache APIs remain in the original SDK goal independently of this ID-only alias.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
flavor = conn.get_flavor_by_id("7", get_extra=False)
value = conn.compute.get_flavor_extra_specs_property(flavor, "hw:cpu_policy")
print(flavor.id, value)
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
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    id := flag.String("flavor-id", "", "flavor ID")
    property := flag.String("property", "hw:cpu_policy", "single extra-spec property")
    extra := flag.Bool("extra", false, "conditionally fetch missing inline flavor specs")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *id, *property, *extra); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, id, property string, extra bool) error {
    if id == "" {
        return fmt.Errorf("-flavor-id is required")
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    cloudFlavor, err := service.GetFlavorByID(ctx, id,
        compute.WithFlavorByIDExtraSpecs(extra))
    if err != nil {
        return fmt.Errorf("cloud flavor by ID: %w", err)
    }
    api := service.API.Flavors
    record, propertyErr := api.GetExtraSpecsProperty(ctx,
        flavors.FlavorExtraSpecsRequest{ID: id}, property)
    if record != nil {
        data, printErr := json.MarshalIndent(map[string]any{
            "value": record.Value,
            "present": record.Present,
            "wire": record.Wire,
            "response_body": string(record.Envelope),
            "header": record.Header,
            "status_code": record.StatusCode,
        }, "", "  ")
        if printErr != nil {
            return errors.Join(propertyErr, printErr)
        }
        fmt.Println(string(data))
    }
    if propertyErr != nil {
        return fmt.Errorf("single property: %w", propertyErr)
    }
    nativeFlavor, err := api.Get(ctx, id)
    if err != nil {
        return fmt.Errorf("native flavor: %w", err)
    }
    nativeProperty, err := api.GetExtraSpec(ctx, id, property)
    if err != nil {
        return fmt.Errorf("native property string map: %w", err)
    }
    listed := make([]*flavors.Flavor, 0, 20)
    for flavor, listErr := range api.ListDetail(ctx,
        flavors.WithListDetailOptions(flavors.ListOpts{Limit: 20})) {
        if listErr != nil {
            return fmt.Errorf("native detailed list: %w", listErr)
        }
        listed = append(listed, flavor)
        if len(listed) == 20 {
            break
        }
    }
    data, err := json.MarshalIndent(map[string]any{
        "cloud_flavor": cloudFlavor,
        "native_flavor": nativeFlavor,
        "native_property_map": nativeProperty,
        "native_detailed_flavors": listed,
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}
```

## References

- [Policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Flavor Extra Specs](flavor-extra-specs.md)
- [User Read APIs](user-read-apis.md)
- [Finding Identities](../docs/finding-identities.md)
