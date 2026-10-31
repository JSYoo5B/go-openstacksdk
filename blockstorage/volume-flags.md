# Volume Flags

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/set_volume_bootable_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SetVolumeBootableStatus`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SetVolumeBootableStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetVolumeBootableStatus`

- SetVolumeBootableStatus requires a bool and sends exactly {"os-set_bootable":{"bootable":value}}, including false, through the direct package, Connection and v3 service APIs without a default.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`.
- Use one safe explicit ID and fixed volumes/action route without name resolution.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsPreflightAndGetterErrorsKeepCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectPreflightPrecedesReadonlyOriginalsAndGuardsCapturedSource`.
- Selected version wins without cap; empty selection negotiates operation-owned Volume ceiling3.71.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsKeepNegotiationAndAppliedStagesSeparate`, `blockstorage/volume_flags_test.go:TestVolumeFlagsSelectedLiteralAndDiscoveryUseOwnedVolumeMicroversionPolicy`.
- Admitted response is opaque acknowledgement; no model refresh or server state assertion.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsKeepNegotiationAndAppliedStagesSeparate`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`, `blockstorage/volume_flags_test.go:TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements`.
- Accepted Read/Close/source/cancellation failures retain current Applied and every cause, without replay.
  Tests: `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`.
- Connection uses cached Cinder v3 and preserves typed preflight/getter causes without location work.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsPreflightAndGetterErrorsKeepCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.
- Exact body and version ownership reuse common scoped POST policy.
  Tests: `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements`.
- New required-bool API remains independent of native200-only API and existing cloud defaultTrue lookup helper.
  Tests: `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.

Python/Go differences:

- Proxy requires bootable; Resource defaultTrue does not create a Proxy/Go default.
- Go bool narrows dynamic Python value domain; Source forwards values without descriptor BoolStr coercion.
- Source accepts ID/Resource/dict/Munch/None through local _get_resource; safe UTF-8 single-segment ID Go input excludes those dynamic constructor paths and unsafe IDs.
- No connected/disconnected Volume conversion, location computation, descriptor normalization or constructor warnings are claimed.
- Finite discovery/first usable v3/max-before-min handling is a bounded SDK policy, not complete Keystoneauth endpoint discovery/cache/links/status-priority parity.
- Missing maximum yields noMV; malformed accepted discovery is terminal evidence; clean direct404/405 fallback only.
- Current official Keystoneauth normalization reference is not a verified pinned dependency runtime.
- Source methods return None; Go deliberately returns owned Result and local completion evidence.
- Go explicit100..399 admission is a bounded HTTP mapping of Source status<400.
- SDK publishes response/error proof and local Completedfalse instead of Source None/raised error alone.
- Current new flag case covers joined Read+Close+source+customcancel for bootableFalse with Close1 and native retry0.
- Connection and typed service wrappers are SDK surfaces; no new Python cloud method or v2 promotion is inferred.
- Underlying Python Adapter auth/retry/redirect behavior is dependency-owned; SDK imposes stronger sticky fixed scope/body/framing/header ownership.
- New bootableFalse native503 retry with expanded400 rejection has an actual public assertion; dedicated bootable changed-body/version/307 matrix is not a new flag-specific case. Shared state transport remains explicit common-policy reuse only.
- Native BootableOpts zero valueFalse serializes; nativeSetBootable accepts200 only; new acknowledgement mapping accepts wider Source sub400 codes.
- New service test independently exercises native200/error-only, new Source203 acknowledgement, nilAPI labels, and existing cloud lookup/defaultTrue. It does not promote the native or cloud targets.
- Only this actual pinned v3 Proxy declaration is mapped. Resource/inherited/v2/native targets, authenticated OpenStack/Python SDK runtime and full SDK completion are not classified by these offline Go contracts. Rejected POST body IO discarded by native Gophercloud remains a limitation.

### `python:block_storage/v3/set_volume_readonly`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SetVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SetVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadonlyOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeReadonlyOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeReadonlyOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeReadonlyOptions`

- SetVolumeReadonly sends exactly {"os-update_readonly_flag":{"readonly":value}}. Omitted options or a nil Readonly field default to true; WithVolumeReadonly(false) and a nonnil false pointer retain explicit false.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyOptionsDefaultsFalseReplacementAndFactoryCopies`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`.
- Complete replacement, order and reusable factories own caller and returned bool pointers.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyOptionsDefaultsFalseReplacementAndFactoryCopies`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`.
- Original callbacks run in captured order and cannot replace callback slice or retained prior config.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyOriginalsOwnCallbackSliceAndRetainedPointersThroughHTTP`.
- Service-free Prepare preserves context/callback causes and stops later originals on failure.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyPrepareStopsNilContextCancellationNilCallbackAndErrors`, `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.
- Direct source capture and safe-ID validation precede originals; observed source violations are sticky.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyDirectPreflightAndStickySourceErrorsStopLaterOriginals`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectPreflightPrecedesReadonlyOriginalsAndGuardsCapturedSource`.
- Connection originals run once before cached getter, then only owned config factory reaches core.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeReadonlyPreparesOriginalsOnceBeforeGetter`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsPreflightAndGetterErrorsKeepCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.
- Shared Volume3.71 negotiation and opaque evidence keep Discovery separate from Applied/Completed.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsKeepNegotiationAndAppliedStagesSeparate`, `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`, `blockstorage/volume_flags_test.go:TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements`, `blockstorage/volume_flags_test.go:TestVolumeFlagsSelectedLiteralAndDiscoveryUseOwnedVolumeMicroversionPolicy`.
- Owned false request survives bounded native retry and rejects changed bool before resend while preserving native evidence.
  Tests: `blockstorage/volume_flag_transport_test.go:TestVolumeReadonlyRetryPreservesFalseAndRejectsChangedFlag`, `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements`, `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.

Python/Go differences:

- Proxy defaultsTrue; Resource requires readonly.
- Go bool pointer omits/defaults nil; Source explicit dynamic None would forward JSONnull rather than become defaultTrue.
- Options, full replacement and pointer ownership are SDK additions to a positional Source bool.
- Source has no option callback layer; owned ordinary headers plus live auth are SDK policy.
- Service-free preparation and nil callback validation are SDK additions; no HTTP/catalog selection occurs in Prepare.
- Source Resource/dict/Munch/None constructor/model/location paths are excluded by typed safe-ID input.
- SDK observes source/context around callback boundaries; it does not claim to detect a mutation restored entirely inside a single callback.
- SDK Connection sequencing is distinct from direct Source capture sequencing; Prepare runs before selecting service.
- No CurrentLocation, model normalization, lookup or v2 promotion is introduced.
- Source None becomes SDK owned Result; no refreshed Volume/read-only/status model is fabricated.
- Finite discovery and explicit100..399 admission remain bounded mappings; actual public tests bind per-target3.71/belowcap/minabovecap/missingmax and native/expanded400 rejection.
- Current readonly-specific retry covers one503retry unchangedFalse and changedTrue rejection with original503method/URL/body/header via errors.As.
- Exhaustive framing/version/307/auth variants are shared policy reuse, not additional new readonly flag cases.
- Native AttachMode ReadOnly ro is unrelated; no native readonly action was invented.
- Only this actual pinned v3 Proxy declaration is mapped. Resource/inherited/v2/native targets, authenticated OpenStack/Python SDK runtime and full SDK completion are not classified by these offline Go contracts. Rejected POST body IO discarded by native Gophercloud remains a limitation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "strconv"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
    if cloudName == "" || volumeID == "" {
        return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
    }
    action := os.Getenv("FLAG_ACTION")
    if action != "bootable" && action != "readonly" {
        return fmt.Errorf("FLAG_ACTION must be bootable or readonly")
    }
    path := os.Getenv("FLAG_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown FLAG_ACTION_PATH %q", path)
    }
    text, supplied := os.LookupEnv("FLAG_VALUE")
    if action == "bootable" && !supplied {
        return fmt.Errorf("FLAG_VALUE is required for bootable")
    }
    flag := true
    if supplied {
        parsed, err := strconv.ParseBool(text)
        if err != nil {
            return fmt.Errorf("FLAG_VALUE: %w", err)
        }
        flag = parsed
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var policy blockstorage.VolumeReadonlyOpts
    if action == "readonly" {
        var originals []blockstorage.VolumeReadonlyOption
        if supplied {
            originals = append(originals, blockstorage.WithVolumeReadonly(flag))
        }
        prepared, err := blockstorage.PrepareVolumeReadonlyOptions(ctx, originals...)
        if err != nil {
            return err
        }
        policy = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input, flag, policy)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest, flag bool, policy blockstorage.VolumeReadonlyOpts) (*blockstorage.VolumeActionResult, error) {
    options := []blockstorage.VolumeReadonlyOption{blockstorage.WithVolumeReadonlyOptions(policy)}
    if path == "connection" {
        if action == "bootable" {
            return conn.SetVolumeBootableStatus(ctx, input, flag)
        }
        return conn.SetVolumeReadonly(ctx, input, options...)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        if action == "bootable" {
            return blockstorage.SetVolumeBootableStatus(ctx, cinder.RawClient(), input, flag)
        }
        return blockstorage.SetVolumeReadonly(ctx, cinder.RawClient(), input, options...)
    }
    if action == "bootable" {
        return cinder.Volumes.SetVolumeBootableStatus(ctx, input.VolumeID, flag)
    }
    return cinder.Volumes.SetVolumeReadonly(ctx, input.VolumeID, options...)
}

func inspect(result *blockstorage.VolumeActionResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for i, page := range result.Discovery {
            if page != nil {
                fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
            }
        }
        if result.Applied != nil {
            fmt.Println("admitted action response:", result.Applied.StatusCode)
            fmt.Printf("opaque action body: %q\n", result.Applied.Body)
        }
        fmt.Println("helper acknowledgement completed:", result.Completed)
    }
    if err != nil {
        fmt.Println("action error:", err)
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
        }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.set_volume_bootable_status(volume_id, False)
conn.block_storage.set_volume_readonly(volume_id)       # default True
conn.block_storage.set_volume_readonly(volume_id, False)
```

## References

- [bootable Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1149)
- [readonly Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1111)
- [Volume methods](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L177)
- [Volume class maximum 3.71](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L146)
- [Volume Actions](volume-actions.md)
- [cloud `SetVolumeBootable`](volume-mutations.md)
- [native `SetBootable`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go#L644)
