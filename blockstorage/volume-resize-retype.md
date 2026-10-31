# Volume Resize Retype

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/extend_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ExtendVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ExtendVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ExtendVolume`

- ExtendVolume requires a machine integer and sends os-extend.new_size exactly, including zero and negative values. The typed value is encoded without the native float64 request-builder roundtrip; the 64-bit-only independent case retains9007199254740993. No local positivity or existing-size comparison is imposed.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Direct calls capture context and Cinder role/provider/route/header policy before validating one safe explicit volume ID and making queryless POST volumes/{ID}/action. No name/Type lookup, Resource conversion, location work or wait occurs; unsafe ID, nil context/client, wrong role and reserved auth fail locally with the actual operation and no fabricated response proof.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectPreflightValidatesOnlySelectedSourceAndUTF8BodyText`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses`.
- Connection uses only cached Cinder v3 and ignores invalid recorded cloud location. Context/Connection/provider/ID preflight and getter failures retain operation/resource identity and custom cancellation causes before any HTTP.
  Tests: `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses`.
- Selected3.1,3.80 or latest is forwarded literally without discovery, ceiling or minimum gate. Empty selection derives an owned Volume3.71 ceiling, respects lower maximum3.70 and omits version when minimum3.72 exceeds the ceiling. Discovery proof is separate from Applied and selection remains per-call without changing cached client version.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeSelectedLiteralAndDiscoveryKeepActionPayloadAndOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeNegotiationKeepsAcceptedStagesSeparate`.
- Owned HTTP100..399 admission retains opaque action bytes without parsing. Current extend-specific cases use200/203 and302 in the64-bit precision case, and populate independently owned Applied plus literal input VolumeID. Completed is local helper acknowledgement and adds no server size/status confirmation or follow-up request.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Native400 and one503retry with expanded native400 acceptance retain actual native status/body/header while original SDK admission rejects400: no Applied, ResponseError or Completed is fabricated.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeOriginalHTTPPolicyKeepsRejectedNativeEvidence`.
- Accepted Read and Close faults joined with observed Source drift and custom cancellation retain current Applied plus independently owned ResponseError and every original cause. Body closes once, Completed is false, and accepted faults invoke no native retry or SDK replay.
  Tests: `blockstorage/volume_resize_transport_test.go:TestVolumeResizeAcceptedFaultsKeepActualOpaqueProofAndEveryCause`.
- New v3 service ExtendVolume uses the required typed size and opaque Source acknowledgement while existing ExtendSize retains its current numeric-zero builder rule and202-only error contract. Nil API produces the actual ExtendVolume label without HTTP or response proof.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`.

Python/Go differences:

- Source size annotations are not runtime conversion/validation: arbitrary-precision integers, bool, null, strings and containers can be forwarded dynamically. Go deliberately narrows to the platform machine int and includes zero/negative; the precision case is guarded for64-bit platforms.
- Native numeric-required zero exemption and its request-builder float conversion are observed current local/cache snapshot behavior, not a claim about a fresh upstream distribution. Existing ExtendSize is unchanged and unpromoted.
- New extend cases exercise accepted joined faults, original-policy rejection and native expanded503retry. Dedicated extend401 reauth, changed-body/version and307 matrices remain explicit shared Cinder policy reuse rather than additional extend-specific test cases.
- Go requires one explicit safe UTF-8 single-segment volume ID. Source accepts Volume/VolumeDetail/Resource/dict/Munch/None through local _get_resource with constructor, descriptor, location and mutable identity effects; Go skips those paths and custom urljoin coercion without classifying their separate declarations.
- Source returns None and does not refresh Volume fields. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement; Completed never confirms size/type/server status. No lookup, CurrentLocation, model conversion, poll, wait or automatic undo is added.
- Selected nonempty microversion is literal and has no additional API-minimum gate. Empty selection reuses bounded operation-owned Volume3.71 discovery, maximum before optional minimum, clean concrete native404/405 fallback and current accepted proof. Full Keystoneauth discovery/cache/link/envelope/status ordering and session canonicalization are outside this mapping; official normalization reference is not a verified pinned dependency runtime.
- Finite HTTP100..399 is the owned mapping of Source status<400. Action bytes remain opaque; native>=400 or expanded OkCodes cannot become an SDK acknowledgement. Native discarded rejected POST body IO remains a limitation.
- Captured source and exact method/route/body/framing/version ownership constrain mutable native retry and redirect hooks. Mutations restored wholly inside one callback are not detectable. Auth/retry/redirect dependency behavior and authenticated OpenStack/Python runtime parity are not established by these offline contracts.
- Only this actual pinned v3 Proxy declaration is a verified Go mapping. Native/resource/inherited/v2/cloud targets and shared dependencies remain independently classified; no full SDK completion or automatic support promotion is inferred.

### `python:block_storage/v3/retype_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.RetypeVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.RetypeVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.RetypeVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeRetypeMigrationPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeRetypeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeRetypeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeRetypeMigrationPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeRetypeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeRetypeOptions`

- RetypeVolume always includes literal new_type. Required UTF-8 text may be empty or contain whitespace, slash, query/percent/fragment characters or controls; it is neither route-validated nor resolved by Type GET. Nil MigrationPolicy defaults to Proxy never, explicit empty omits migration_policy, and arbitrary valid policy is accepted without enum validation.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues`.
- Complete replacement and later options determine final policy. Factories copy caller pointers when created and every invocation/result owns a fresh pointer; callbacks execute a copied original slice in order with copied intermediate configs, so replacement of a later caller callback or mutation of retained prior pointers cannot alter the request. Captured ordinary headers stay original while provider authentication is live.
  Tests: `blockstorage/volume_resize_options_test.go:TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeOriginalsOwnOrderCallbackSliceAndRetainedPointersThroughHTTP`.
- Service-free Prepare runs originals once without selecting a service or doing HTTP. Nil context/options and callback/context failures stop later originals, retain all callback and custom cancellation causes, and return zero config. Migration policy UTF-8 is checked only after originals, permitting an invalid earlier value to be replaced by a valid final value before any action.
  Tests: `blockstorage/volume_resize_options_test.go:TestVolumeResizePreparePreservesCausesAndValidatesOnlyFinalRetypePolicy`.
- Direct calls capture Cinder Source, validate safe volume ID, then validate newType UTF-8 before originals. Source/context is guarded around originals; an observed route/source fault remains terminal and prevents later repair callbacks or HTTP while keeping joined callback and cancellation causes.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectPreflightValidatesOnlySelectedSourceAndUTF8BodyText`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeDirectPreflightAndStickySourcesStopOriginalsAndHTTP`.
- Connection preflight precedes one original Prepare; final migration policy and newType are checked before ID admission and only cached Cinder v3 selection. The getter cannot mutate a retained original config into the action; only an owned full-policy factory reaches core. Location is ignored and preflight/getter errors preserve actual operation and cancellation causes.
  Tests: `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeOriginalsRunOnceBeforeGetterWithOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses`.
- Selected literal3.1/3.80/latest bypasses discovery and class ceiling without a new minimum gate. Otherwise finite per-call selection uses Volume3.71, lower maximum or no version for minimum above ceiling; cached source is unchanged and accepted discovery pages remain separate from action Applied, including terminal malformed discovery evidence.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeSelectedLiteralAndDiscoveryKeepActionPayloadAndOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeNegotiationKeepsAcceptedStagesSeparate`.
- Opaque200/203/302/399 replies own raw Applied and local Completed independently across calls. Native400 and503retry expanded400 stay native rejected evidence with no Applied or success proof; no response JSON or refreshed Volume type/status model is created.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_test.go:TestVolumeResizeOriginalHTTPPolicyKeepsRejectedNativeEvidence`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Accepted Read/Close/source/custom cancellation retains current opaque Applied, independent ResponseError, every joined cause, close-once and Completed=false without replay or caller retry.
  Tests: `blockstorage/volume_resize_transport_test.go:TestVolumeResizeAcceptedFaultsKeepActualOpaqueProofAndEveryCause`.
- One native401 reauth and one bounded503retry use live token with original captured ordinary headers and exact owned never policy despite mutation of retained option pointer. A retry changing migration_policy is stopped before third transport and retains original503method/URL/body/header; originals are never rerun.
  Tests: `blockstorage/volume_resize_transport_test.go:TestVolumeRetypeReauthRetryKeepsOwnedPolicyAndRejectsChangedBody`.
- New service RetypeVolume admits empty literal newType, defaultnever and explicit empty omission, preserving actual labels on nil API. Existing ChangeType still requires nonempty NewType, omits empty migration policy and accepts202 only; those native builders and targets are unchanged.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`.

Python/Go differences:

- Source accepts a Type Resource through Resource._get_id, or forwards arbitrary non-Resource values including null/bool/container domains. Go newType and MigrationPolicy are typed UTF-8 strings, deliberately omit constructor/Resource identity and preserve valid literal body text without Type lookup.
- Proxy defaultnever differs from Resource.retype defaultNone. Go nil pointer means Proxy defaultnever; explicit empty string models omitted field, rather than reproducing explicit Python None/False/0 dynamic inputs.
- Factories, options, service-free preparation, copied callback slices/configs, zero-on-error and direct versus Connection ordering are SDK-owned contracts absent from positional Python Source.
- Current retype-specific native401/503 case covers retained policy mutation, live auth and changed body rejection. Exhaustive version/framing/307 matrices remain explicit shared Cinder transport reuse, not new retype-specific cases.
- Native ChangeType required nonempty string and202-only policy stay independent of the new Source mapping; enum constants are not runtime validation and no native target is promoted.
- Go requires one explicit safe UTF-8 single-segment volume ID. Source accepts Volume/VolumeDetail/Resource/dict/Munch/None through local _get_resource with constructor, descriptor, location and mutable identity effects; Go skips those paths and custom urljoin coercion without classifying their separate declarations.
- Source returns None and does not refresh Volume fields. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement; Completed never confirms size/type/server status. No lookup, CurrentLocation, model conversion, poll, wait or automatic undo is added.
- Selected nonempty microversion is literal and has no additional API-minimum gate. Empty selection reuses bounded operation-owned Volume3.71 discovery, maximum before optional minimum, clean concrete native404/405 fallback and current accepted proof. Full Keystoneauth discovery/cache/link/envelope/status ordering and session canonicalization are outside this mapping; official normalization reference is not a verified pinned dependency runtime.
- Finite HTTP100..399 is the owned mapping of Source status<400. Action bytes remain opaque; native>=400 or expanded OkCodes cannot become an SDK acknowledgement. Native discarded rejected POST body IO remains a limitation.
- Captured source and exact method/route/body/framing/version ownership constrain mutable native retry and redirect hooks. Mutations restored wholly inside one callback are not detectable. Auth/retry/redirect dependency behavior and authenticated OpenStack/Python runtime parity are not established by these offline contracts.
- Only this actual pinned v3 Proxy declaration is a verified Go mapping. Native/resource/inherited/v2/cloud targets and shared dependencies remain independently classified; no full SDK completion or automatic support promotion is inferred.

### `python:block_storage/v3/complete_volume_extend`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.CompleteVolumeExtend`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CompleteVolumeExtend`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.CompleteVolumeExtend`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeExtendCompletionError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeExtendCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeExtendCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeExtendCompletionError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeExtendCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeExtendCompletionOptions`

- CompleteVolumeExtend sends os-extend_volume_completion.error on every action. Omitted options/nil Error default to false; explicit false and true retain the supplied bool, and full empty replacement restores false. Error=true informs Cinder of an extend error; SDK performs no rollback or backend completion check.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues`, `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Reusable full-policy factories copy caller input and own each returned bool. Later options win, original callback slice is snapshotted and intermediate/retained config pointers cannot alter later callbacks or the HTTP body; captured ordinary headers remain original while token is live.
  Tests: `blockstorage/volume_resize_options_test.go:TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeOriginalsOwnOrderCallbackSliceAndRetainedPointersThroughHTTP`.
- Service-free Prepare validates context and nil callbacks, retains callback/custom cancellation causes, stops later originals on failure and returns zero config. Defaults and returned pointers are owned, and no client selection or HTTP is performed.
  Tests: `blockstorage/volume_resize_options_test.go:TestVolumeResizePreparePreservesCausesAndValidatesOnlyFinalRetypePolicy`.
- Direct source capture and safe-ID admission precede originals. Source/context guards around originals make observed source drift and joined callback/cancellation failure terminal before later repair callbacks or HTTP, with no fabricated response proof.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectPreflightValidatesOnlySelectedSourceAndUTF8BodyText`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeDirectPreflightAndStickySourcesStopOriginalsAndHTTP`.
- Connection preflights context/Connection/provider, runs originals once before safe-ID/getter selection, and passes only copied complete policy to core. Getter mutation of retained config cannot change true body; cached Cinder v3 ignores location, and preflight/getter failures retain operation and custom cancellation identities.
  Tests: `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeOriginalsRunOnceBeforeGetterWithOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses`.
- Selected literal3.1/3.80/latest bypasses discovery/ceiling and no minimum gate is added. Empty selection uses finite per-call Volume3.71/lower maximum/noMV for high minimum; admitted discovery evidence and action Applied remain distinct, and cached service version stays unchanged.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeSelectedLiteralAndDiscoveryKeepActionPayloadAndOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeNegotiationKeepsAcceptedStagesSeparate`.
- Accepted200/203/302/399 opaque bytes populate owned Applied and local helper Completed without parsing or refreshing Volume state. Native400 and one503 expanded400 retry retain rejected native body/header and cannot become Applied or Completed.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_test.go:TestVolumeResizeOriginalHTTPPolicyKeepsRejectedNativeEvidence`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Accepted Read/Close faults, observed Source drift and custom cancellation retain every cause and current opaque Applied plus independent ResponseError. Close executes once, Completed=false, and no accepted-fault caller retry or replay occurs.
  Tests: `blockstorage/volume_resize_transport_test.go:TestVolumeResizeAcceptedFaultsKeepActualOpaqueProofAndEveryCause`.
- New service CompleteVolumeExtend sends defaultfalse or explicittrue and admits opaque acknowledgement; nil API preserves actual completion operation label without originals, request or proof. This helper does not wait for volume extension or invent a native completion API.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`.

Python/Go differences:

- Source positional error annotation is not runtime bool conversion. Explicit dynamic None/nonbool values can be forwarded as JSON; Go optional *bool deliberately owns defaultFalse and narrows the domain to bool.
- Options, reusable full replacement, copied callback/config ownership, service-free preparation and Connection originals-before-getter ordering are SDK additions to the Source positional defaultFalse contract.
- The completion action is an acknowledgement, not a poll or server-state postcondition; error=true is Cinder input rather than SDK rollback. No native completion builder/API is invented.
- New completion cases cover accepted joined faults and original-policy rejection including expanded503retry. Dedicated completion401/changed-body/version/307 matrices remain explicit shared scoped Cinder policy reuse rather than new completion-specific cases.
- Go requires one explicit safe UTF-8 single-segment volume ID. Source accepts Volume/VolumeDetail/Resource/dict/Munch/None through local _get_resource with constructor, descriptor, location and mutable identity effects; Go skips those paths and custom urljoin coercion without classifying their separate declarations.
- Source returns None and does not refresh Volume fields. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement; Completed never confirms size/type/server status. No lookup, CurrentLocation, model conversion, poll, wait or automatic undo is added.
- Selected nonempty microversion is literal and has no additional API-minimum gate. Empty selection reuses bounded operation-owned Volume3.71 discovery, maximum before optional minimum, clean concrete native404/405 fallback and current accepted proof. Full Keystoneauth discovery/cache/link/envelope/status ordering and session canonicalization are outside this mapping; official normalization reference is not a verified pinned dependency runtime.
- Finite HTTP100..399 is the owned mapping of Source status<400. Action bytes remain opaque; native>=400 or expanded OkCodes cannot become an SDK acknowledgement. Native discarded rejected POST body IO remains a limitation.
- Captured source and exact method/route/body/framing/version ownership constrain mutable native retry and redirect hooks. Mutations restored wholly inside one callback are not detectable. Auth/retry/redirect dependency behavior and authenticated OpenStack/Python runtime parity are not established by these offline contracts.
- Only this actual pinned v3 Proxy declaration is a verified Go mapping. Native/resource/inherited/v2/cloud targets and shared dependencies remain independently classified; no full SDK completion or automatic support promotion is inferred.

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
    action := os.Getenv("RESIZE_ACTION")
    if action != "extend" && action != "retype" && action != "complete-extend" {
        return fmt.Errorf("RESIZE_ACTION must be extend, retype or complete-extend")
    }
    path := os.Getenv("RESIZE_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown RESIZE_ACTION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var size int
    var newType string
    var retype blockstorage.VolumeRetypeOpts
    var completion blockstorage.VolumeExtendCompletionOpts
    switch action {
    case "extend":
        text, supplied := os.LookupEnv("EXTEND_SIZE")
        if !supplied {
            return fmt.Errorf("EXTEND_SIZE is required")
        }
        parsed, err := strconv.Atoi(text)
        if err != nil {
            return fmt.Errorf("EXTEND_SIZE: %w", err)
        }
        size = parsed
    case "retype":
        text, supplied := os.LookupEnv("NEW_VOLUME_TYPE")
        if !supplied {
            return fmt.Errorf("NEW_VOLUME_TYPE is required; an explicit empty string is allowed")
        }
        newType = text
        var originals []blockstorage.VolumeRetypeOption
        if policy, supplied := os.LookupEnv("MIGRATION_POLICY"); supplied {
            originals = append(originals, blockstorage.WithVolumeRetypeMigrationPolicy(policy))
        }
        prepared, err := blockstorage.PrepareVolumeRetypeOptions(ctx, originals...)
        if err != nil {
            return err
        }
        retype = prepared
    case "complete-extend":
        var originals []blockstorage.VolumeExtendCompletionOption
        if text, supplied := os.LookupEnv("COMPLETION_ERROR"); supplied {
            flag, err := strconv.ParseBool(text)
            if err != nil {
                return fmt.Errorf("COMPLETION_ERROR: %w", err)
            }
            originals = append(originals, blockstorage.WithVolumeExtendCompletionError(flag))
        }
        prepared, err := blockstorage.PrepareVolumeExtendCompletionOptions(ctx, originals...)
        if err != nil {
            return err
        }
        completion = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input, size, newType, retype, completion)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest, size int, newType string, retype blockstorage.VolumeRetypeOpts, completion blockstorage.VolumeExtendCompletionOpts) (*blockstorage.VolumeActionResult, error) {
    r := blockstorage.WithVolumeRetypeOptions(retype)
    c := blockstorage.WithVolumeExtendCompletionOptions(completion)
    if path == "connection" {
        switch action {
        case "extend": return conn.ExtendVolume(ctx, input, size)
        case "retype": return conn.RetypeVolume(ctx, input, newType, r)
        default: return conn.CompleteVolumeExtend(ctx, input, c)
        }
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        switch action {
        case "extend": return blockstorage.ExtendVolume(ctx, cinder.RawClient(), input, size)
        case "retype": return blockstorage.RetypeVolume(ctx, cinder.RawClient(), input, newType, r)
        default: return blockstorage.CompleteVolumeExtend(ctx, cinder.RawClient(), input, c)
        }
    }
    switch action {
    case "extend": return cinder.Volumes.ExtendVolume(ctx, input.VolumeID, size)
    case "retype": return cinder.Volumes.RetypeVolume(ctx, input.VolumeID, newType, r)
    default: return cinder.Volumes.CompleteVolumeExtend(ctx, input.VolumeID, c)
    }
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
            fmt.Printf("opaque body: %q\n", result.Applied.Body)
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
conn.block_storage.extend_volume(volume_id, 20)
conn.block_storage.retype_volume(volume_id, "actual-type-id", migration_policy="never")
conn.block_storage.complete_volume_extend(volume_id, error=False)
```

## References

- [extend/completion Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1084)
- [retype Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1126)
- [Volume.extend/complete_extend](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L165)
- [Volume.retype](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L317)
- [Resource._get_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1016)
- [Volume Flags](volume-flags.md)
- [Volume Actions](volume-actions.md)
