# Cache

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/get_image_cache`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageCache`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeaders`

- GetImageCache sends bodyless/query-free GET at captured base plus fixed cache, requiring original actual200 only. DirectID has only requested cache endpoint; explicit Name for per-ID calls uses unchanged native all-pages exact-name resolution, with no synthetic Ref/ID for singleton routes.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- ImageCache requires canonical nonnull cached_images array of nonnull object rows and queued_images array of nonnull strings; explicit empty arrays remain nonnil, ordered and repeated. CachedImage optional canonical image_id is string; hits/size exact signedint64 integers; last_accessed/last_modified strict JSON number tokens in json.Number pointers. Missing/null optional fields nil, explicit empty/zero present; quoted epoch numbers reject, lexical fractions/exponents retained without float64. Independent typed rows/raw Body/Header preserve unknown/case-decoy/links/date/id/status data passively; Metadata links/dates stay nil and decoder failure is atomic.
  Tests: `image/cache_contracts_test.go:TestImageCacheCanonicalModelsAndRawPrecision`, `image/cache_core_test.go:TestImageCacheCoreCanonicalPrecisionAndIndependentMetadata`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Source/context/client/provider/prefix/ordinary headers/ref captured and validated before callbacks. Concrete full replacement and helper input maps/optional bool/slice/percallback configs owned before/after one application; nil/error callback/protected/conflicting header invalid before HTTP. Cache-owned target header rejected in source/options on all7. Options and captured route/header stay fixed while original provider token stays live; provider replacement/unsafe retarget and postName source changes rejected.
  Tests: `image/cache_contracts_test.go:TestImageCachePreparedOptionsAndLiveSource`, `image/cache_core_test.go:TestImageCacheCoreCompletePreflightAndOwnedTargetHeader`, `image/cache_options_test.go:TestImageCacheOptionsSnapshotReplacementAndLastWins`, `image/cache_options_test.go:TestImageCacheOptionsOwnPointersCallbacksAndSlices`, `image/cache_options_test.go:TestImageCacheOptionsProtectedHeadersAndCompleteDefaults`, `image/cache_options_test.go:TestImageCacheOptionsParallelReusableHelpers`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`.
- Typed200 read/Close/context/wholeUTF8/JSON/canonical failure returns nil typed result plus owned ResponseError retaining actual bytes/Header/status and all original read/Close/ctx.Err/custom causes. Body closes once and accepted bytes are never replayed. Other actual statuses/prebody transport/hook failures retain native error policy; expanded native OkCodes cannot broaden captured acceptance.
  Tests: `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Shared fixed request/DoJSON retain configured native prebody retry/reauth/backoff and identical-target redirects. Changed method/origin/path/query blocked before transport. Nil body cannot become JSONnull; RawBody/KeepResponseBody/JSONResponse/serialized body ownership cannot mutate before another attempt; original/hook/marshal causes retained. Native reauth ErrOriginal/ErrReauth remain in original fields. Existing native image bindings unchanged; no native cache API/inventory row fabricated.
  Tests: `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- One snapshot of selected endpoint only; no query/limit/marker/discovery/pager/cache tracking. Driver supplies numeric epoch measurements and queued processing order; data cannot select new routes or enforce future requests.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`, `image/cache_contracts_test.go:TestImageCacheCanonicalModelsAndRawPrecision`, `image/cache_core_test.go:TestImageCacheCoreCanonicalPrecisionAndIndependentMetadata`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Each cache call with default arguments sends the same bodyless request as Python: GET cache, PUT cache/{image}, DELETE cache/{image} with 404 ignored, DELETE cache without x-image-cache-clear-target for target both, GET cache/nodes/{image}, POST cache/clean and POST cache/prune, and decodes cached/queued images, node URLs and prune counts.
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageCacheDefaultRequests`.

Python/Go differences:

- Python queue/delete/clear/clean proxy calls return None; get returns Cache Resource and nodes/prune return typing.cast JSON without runtime schema validation. Go returns actual acknowledgements or strict canonical owned typed/raw data, preserving response causes and requiring captured exact statuses.
- One snapshot of selected endpoint only; no query/limit/marker/discovery/pager/cache tracking. Driver supplies numeric epoch measurements and queued processing order; data cannot select new routes or enforce future requests.
- Go accepts only the documented success status of each cache route; Python accepts any status below 400.
- Python Cache caps the negotiated microversion at 2.15; Go sends the configured Image microversion and adds no capability discovery.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:image/v2/cache_delete_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.CacheDeleteImage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheDeleteIgnoreMissing`

- CacheDeleteImage sends bodyless/query-free DELETE at captured base plus fixed cache/{image_id}, requiring actual204 acknowledgement only, with separately owned404 acceptance for the narrow nil/true missing policy. DirectID has only requested cache endpoint; explicit Name for per-ID calls uses unchanged native all-pages exact-name resolution, with no synthetic Ref/ID for singleton routes.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`, `image/cache_contracts_test.go:TestImageCacheExactNameResolutionAndDirectIdentity`.
- Actual204 acknowledgement records fixed requested ImageID, nil Target and independent opaque Body/Header/status. Nil IgnoreMissing defaults true; only fully read/closed/context-successful physically owned DELETE404 yields nil,nil. Accepted404 Read/Close/context failure returns nil acknowledgement plus ResponseError404. Every Name/transport/hook/reauth error remains an error.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`, `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`.
- Source/context/client/provider/prefix/ordinary headers/ref captured and validated before callbacks. Concrete full replacement and helper input maps/optional bool/slice/percallback configs owned before/after one application; nil/error callback/protected/conflicting header invalid before HTTP. Cache-owned target header rejected in source/options on all7. Options and captured route/header stay fixed while original provider token stays live; provider replacement/unsafe retarget and postName source changes rejected.
  Tests: `image/cache_contracts_test.go:TestImageCachePreparedOptionsAndLiveSource`, `image/cache_core_test.go:TestImageCacheCoreCompletePreflightAndOwnedTargetHeader`, `image/cache_options_test.go:TestImageCacheOptionsSnapshotReplacementAndLastWins`, `image/cache_options_test.go:TestImageCacheOptionsOwnPointersCallbacksAndSlices`, `image/cache_options_test.go:TestImageCacheOptionsProtectedHeadersAndCompleteDefaults`, `image/cache_options_test.go:TestImageCacheOptionsParallelReusableHelpers`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`.
- Accepted expected-status acknowledgement Read/Close/context failure retains result plus independent ResponseError bytes/Header/status and original causes; default owned404 failure has nil acknowledgement as described. Body closes once and accepted bytes are never replayed. Other actual statuses/prebody transport/hook failures retain native error policy; expanded native OkCodes cannot broaden captured acceptance.
  Tests: `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`.
- Shared fixed request/DoJSON retain configured native prebody retry/reauth/backoff and identical-target redirects. Changed method/origin/path/query blocked before transport. Nil body cannot become JSONnull; RawBody/KeepResponseBody/JSONResponse/serialized body ownership cannot mutate before another attempt; original/hook/marshal causes retained. Native reauth ErrOriginal/ErrReauth remain in original fields. Existing native image bindings unchanged; no native cache API/inventory row fabricated.
  Tests: `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Private owned404 acceptance bypasses native404 RetryFunc only with nil/true IgnoreMissing. Explicitfalse preserves native strict204/404 retry ownership. Native Name errors including logical missing never quiet. Ordinary server404 alone cannot distinguish missing image from disabled cache, so either can be quiet under this narrow default.
  Tests: `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`, `image/cache_contracts_test.go:TestImageCacheExactNameResolutionAndDirectIdentity`.
- Each cache call with default arguments sends the same bodyless request as Python: GET cache, PUT cache/{image}, DELETE cache/{image} with 404 ignored, DELETE cache without x-image-cache-clear-target for target both, GET cache/nodes/{image}, POST cache/clean and POST cache/prune, and decodes cached/queued images, node URLs and prune counts.
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageCacheDefaultRequests`.

Python/Go differences:

- Python queue/delete/clear/clean proxy calls return None; get returns Cache Resource and nodes/prune return typing.cast JSON without runtime schema validation. Go returns actual acknowledgements or strict canonical owned typed/raw data, preserving response causes and requiring captured exact statuses.
- Private owned404 acceptance bypasses native404 RetryFunc only with nil/true IgnoreMissing. Explicitfalse preserves native strict204/404 retry ownership. Native Name errors including logical missing never quiet. Ordinary server404 alone cannot distinguish missing image from disabled cache, so either can be quiet under this narrow default.
- Go accepts only the documented success status of each cache route; Python accepts any status below 400.
- Python Cache caps the negotiated microversion at 2.15; Go sends the configured Image microversion and adds no capability discovery.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:image/v2/queue_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.QueueImage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeaders`

- QueueImage sends bodyless/query-free PUT at captured base plus fixed cache/{image_id}, requiring original actual202 only. DirectID has only requested cache endpoint; explicit Name for per-ID calls uses unchanged native all-pages exact-name resolution, with no synthetic Ref/ID for singleton routes.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`, `image/cache_contracts_test.go:TestImageCacheExactNameResolutionAndDirectIdentity`.
- Acknowledgement records fixed requested ImageID, nil Target, and actual opaque Body/Header/StatusCode202. Empty/nonJSON/nonUTF8 body bytes are not translated into cached-image state or a mutable seeded Resource. Accepted body failure retains nonnil acknowledgement plus ResponseError.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`.
- Source/context/client/provider/prefix/ordinary headers/ref captured and validated before callbacks. Concrete full replacement and helper input maps/optional bool/slice/percallback configs owned before/after one application; nil/error callback/protected/conflicting header invalid before HTTP. Cache-owned target header rejected in source/options on all7. Options and captured route/header stay fixed while original provider token stays live; provider replacement/unsafe retarget and postName source changes rejected.
  Tests: `image/cache_contracts_test.go:TestImageCachePreparedOptionsAndLiveSource`, `image/cache_core_test.go:TestImageCacheCoreCompletePreflightAndOwnedTargetHeader`, `image/cache_options_test.go:TestImageCacheOptionsSnapshotReplacementAndLastWins`, `image/cache_options_test.go:TestImageCacheOptionsOwnPointersCallbacksAndSlices`, `image/cache_options_test.go:TestImageCacheOptionsProtectedHeadersAndCompleteDefaults`, `image/cache_options_test.go:TestImageCacheOptionsParallelReusableHelpers`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`.
- Accepted expected-status acknowledgement Read/Close/context failure retains result plus independent ResponseError bytes/Header/status and original causes; default owned404 failure has nil acknowledgement as described. Body closes once and accepted bytes are never replayed. Other actual statuses/prebody transport/hook failures retain native error policy; expanded native OkCodes cannot broaden captured acceptance.
  Tests: `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`.
- Shared fixed request/DoJSON retain configured native prebody retry/reauth/backoff and identical-target redirects. Changed method/origin/path/query blocked before transport. Nil body cannot become JSONnull; RawBody/KeepResponseBody/JSONResponse/serialized body ownership cannot mutate before another attempt; original/hook/marshal causes retained. Native reauth ErrOriginal/ErrReauth remain in original fields. Existing native image bindings unchanged; no native cache API/inventory row fabricated.
  Tests: `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Actual routed server checks registry image existence and active status before cache.queue_image and worker submission despite stale controller docstring. SDK adds no metadata fetch/active gate/wait/cache completion claim;202 is asynchronous acknowledgement only.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Each cache call with default arguments sends the same bodyless request as Python: GET cache, PUT cache/{image}, DELETE cache/{image} with 404 ignored, DELETE cache without x-image-cache-clear-target for target both, GET cache/nodes/{image}, POST cache/clean and POST cache/prune, and decodes cached/queued images, node URLs and prune counts.
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageCacheDefaultRequests`.

Python/Go differences:

- Python queue/delete/clear/clean proxy calls return None; get returns Cache Resource and nodes/prune return typing.cast JSON without runtime schema validation. Go returns actual acknowledgements or strict canonical owned typed/raw data, preserving response causes and requiring captured exact statuses.
- Actual routed server checks registry image existence and active status before cache.queue_image and worker submission despite stale controller docstring. SDK adds no metadata fetch/active gate/wait/cache completion claim;202 is asynchronous acknowledgement only.
- Go accepts only the documented success status of each cache route; Python accepts any status below 400.
- Python Cache caps the negotiated microversion at 2.15; Go sends the configured Image microversion and adds no capability discovery.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:image/v2/clear_cache`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.ClearCache`
- `github.com/JSYoo5B/go-openstacksdk/image.WithClearCacheOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithClearCacheHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithClearCacheHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithClearCacheTarget`

- ClearCache sends bodyless/query-free DELETE at captured base plus fixed cache, requiring original actual204 only. DirectID has only requested cache endpoint; explicit Name for per-ID calls uses unchanged native all-pages exact-name resolution, with no synthetic Ref/ID for singleton routes.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Acknowledgement records empty ImageID and independently owned Target pointer including default CacheBoth, plus actual opaque Body/Header/status204. Controller internal cache_deleted/queue_deleted counts are discarded by serializer and never synthesized. CacheBoth omits owned target header; CacheOnly/QueueOnly emit exact cache/queue. Only uint8 enum0/1/2 accepted.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`.
- Source/context/client/provider/prefix/ordinary headers/ref captured and validated before callbacks. Concrete full replacement and helper input maps/optional bool/slice/percallback configs owned before/after one application; nil/error callback/protected/conflicting header invalid before HTTP. Cache-owned target header rejected in source/options on all7. Options and captured route/header stay fixed while original provider token stays live; provider replacement/unsafe retarget and postName source changes rejected.
  Tests: `image/cache_contracts_test.go:TestImageCachePreparedOptionsAndLiveSource`, `image/cache_core_test.go:TestImageCacheCoreCompletePreflightAndOwnedTargetHeader`, `image/cache_options_test.go:TestImageCacheOptionsSnapshotReplacementAndLastWins`, `image/cache_options_test.go:TestImageCacheOptionsOwnPointersCallbacksAndSlices`, `image/cache_options_test.go:TestImageCacheOptionsProtectedHeadersAndCompleteDefaults`, `image/cache_options_test.go:TestImageCacheOptionsParallelReusableHelpers`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`.
- Accepted expected-status acknowledgement Read/Close/context failure retains result plus independent ResponseError bytes/Header/status and original causes; default owned404 failure has nil acknowledgement as described. Body closes once and accepted bytes are never replayed. Other actual statuses/prebody transport/hook failures retain native error policy; expanded native OkCodes cannot broaden captured acceptance.
  Tests: `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`.
- Shared fixed request/DoJSON retain configured native prebody retry/reauth/backoff and identical-target redirects. Changed method/origin/path/query blocked before transport. Nil body cannot become JSONnull; RawBody/KeepResponseBody/JSONResponse/serialized body ownership cannot mutate before another attempt; original/hook/marshal causes retained. Native reauth ErrOriginal/ErrReauth remain in original fields. Existing native image bindings unchanged; no native cache API/inventory row fabricated.
  Tests: `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Python target both also omits header. Literal both header is rejected by pinned controller although admin prose lists it; absent/empty actual wire means both. Source/option x-image-cache-clear-target aliases are reserved across all seven methods and may not bypass typed target policy.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Each cache call with default arguments sends the same bodyless request as Python: GET cache, PUT cache/{image}, DELETE cache/{image} with 404 ignored, DELETE cache without x-image-cache-clear-target for target both, GET cache/nodes/{image}, POST cache/clean and POST cache/prune, and decodes cached/queued images, node URLs and prune counts.
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageCacheDefaultRequests`.

Python/Go differences:

- Python queue/delete/clear/clean proxy calls return None; get returns Cache Resource and nodes/prune return typing.cast JSON without runtime schema validation. Go returns actual acknowledgements or strict canonical owned typed/raw data, preserving response causes and requiring captured exact statuses.
- Python target both also omits header. Literal both header is rejected by pinned controller although admin prose lists it; absent/empty actual wire means both. Source/option x-image-cache-clear-target aliases are reserved across all seven methods and may not bypass typed target policy.
- Go accepts only the documented success status of each cache route; Python accepts any status below 400.
- Python Cache caps the negotiated microversion at 2.15; Go sends the configured Image microversion and adds no capability discovery.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:image/v2/cached_image_nodes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.CachedImageNodes`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeaders`

- CachedImageNodes sends bodyless/query-free GET at captured base plus fixed cache/nodes/{image_id}, requiring original actual200 only. DirectID has only requested cache endpoint; explicit Name for per-ID calls uses unchanged native all-pages exact-name resolution, with no synthetic Ref/ID for singleton routes.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`, `image/cache_contracts_test.go:TestImageCacheExactNameResolutionAndDirectIdentity`.
- Finite bare nonnull array of nonnull strings, including empty/foreign/literal URL values, returned in order with repeats. Explicit [] remains nonnil. Result preserves fixed requested ImageID plus independent entire Body/Header/StatusCode; no envelope/singleton coercion, URL validation/following, paging or node fanout.
  Tests: `image/cache_contracts_test.go:TestImageCacheCanonicalModelsAndRawPrecision`, `image/cache_core_test.go:TestImageCacheCoreCanonicalPrecisionAndIndependentMetadata`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Source/context/client/provider/prefix/ordinary headers/ref captured and validated before callbacks. Concrete full replacement and helper input maps/optional bool/slice/percallback configs owned before/after one application; nil/error callback/protected/conflicting header invalid before HTTP. Cache-owned target header rejected in source/options on all7. Options and captured route/header stay fixed while original provider token stays live; provider replacement/unsafe retarget and postName source changes rejected.
  Tests: `image/cache_contracts_test.go:TestImageCachePreparedOptionsAndLiveSource`, `image/cache_core_test.go:TestImageCacheCoreCompletePreflightAndOwnedTargetHeader`, `image/cache_options_test.go:TestImageCacheOptionsSnapshotReplacementAndLastWins`, `image/cache_options_test.go:TestImageCacheOptionsOwnPointersCallbacksAndSlices`, `image/cache_options_test.go:TestImageCacheOptionsProtectedHeadersAndCompleteDefaults`, `image/cache_options_test.go:TestImageCacheOptionsParallelReusableHelpers`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`.
- Typed200 read/Close/context/wholeUTF8/JSON/canonical failure returns nil typed result plus owned ResponseError retaining actual bytes/Header/status and all original read/Close/ctx.Err/custom causes. Body closes once and accepted bytes are never replayed. Other actual statuses/prebody transport/hook failures retain native error policy; expanded native OkCodes cannot broaden captured acceptance.
  Tests: `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Shared fixed request/DoJSON retain configured native prebody retry/reauth/backoff and identical-target redirects. Changed method/origin/path/query blocked before transport. Nil body cannot become JSONnull; RawBody/KeepResponseBody/JSONResponse/serialized body ownership cannot mutate before another attempt; original/hook/marshal causes retained. Native reauth ErrOriginal/ErrReauth remain in original fields. Existing native image bindings unchanged; no native cache API/inventory row fabricated.
  Tests: `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Pinned server requires configured cache, centralized_db driver, authorized list_cached_nodes policy and registry image existence. Node URLs are passive references; SDK does not validate hosts/follow URLs/discover drivers/fanout or apply a local capability gate.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`, `image/cache_contracts_test.go:TestImageCacheCanonicalModelsAndRawPrecision`, `image/cache_core_test.go:TestImageCacheCoreCanonicalPrecisionAndIndependentMetadata`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Each cache call with default arguments sends the same bodyless request as Python: GET cache, PUT cache/{image}, DELETE cache/{image} with 404 ignored, DELETE cache without x-image-cache-clear-target for target both, GET cache/nodes/{image}, POST cache/clean and POST cache/prune, and decodes cached/queued images, node URLs and prune counts.
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageCacheDefaultRequests`.

Python/Go differences:

- Python queue/delete/clear/clean proxy calls return None; get returns Cache Resource and nodes/prune return typing.cast JSON without runtime schema validation. Go returns actual acknowledgements or strict canonical owned typed/raw data, preserving response causes and requiring captured exact statuses.
- Pinned server requires configured cache, centralized_db driver, authorized list_cached_nodes policy and registry image existence. Node URLs are passive references; SDK does not validate hosts/follow URLs/discover drivers/fanout or apply a local capability gate.
- Go accepts only the documented success status of each cache route; Python accepts any status below 400.
- Python Cache caps the negotiated microversion at 2.15; Go sends the configured Image microversion and adds no capability discovery.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:image/v2/clean_cache`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.CleanCache`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeaders`

- CleanCache sends bodyless/query-free POST at captured base plus fixed cache/clean, requiring original actual200 only. DirectID has only requested cache endpoint; explicit Name for per-ID calls uses unchanged native all-pages exact-name resolution, with no synthetic Ref/ID for singleton routes.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Acknowledgement records empty ImageID, nil Target, actual opaque Body/Header/status200. Serializer does not emit JSON counts. Empty/nonJSON/nonUTF8 accepted bytes remain evidence without parsing or inferring cleaning outcome; accepted body failure retains nonnil acknowledgement plus ResponseError.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`.
- Source/context/client/provider/prefix/ordinary headers/ref captured and validated before callbacks. Concrete full replacement and helper input maps/optional bool/slice/percallback configs owned before/after one application; nil/error callback/protected/conflicting header invalid before HTTP. Cache-owned target header rejected in source/options on all7. Options and captured route/header stay fixed while original provider token stays live; provider replacement/unsafe retarget and postName source changes rejected.
  Tests: `image/cache_contracts_test.go:TestImageCachePreparedOptionsAndLiveSource`, `image/cache_core_test.go:TestImageCacheCoreCompletePreflightAndOwnedTargetHeader`, `image/cache_options_test.go:TestImageCacheOptionsSnapshotReplacementAndLastWins`, `image/cache_options_test.go:TestImageCacheOptionsOwnPointersCallbacksAndSlices`, `image/cache_options_test.go:TestImageCacheOptionsProtectedHeadersAndCompleteDefaults`, `image/cache_options_test.go:TestImageCacheOptionsParallelReusableHelpers`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`.
- Accepted expected-status acknowledgement Read/Close/context failure retains result plus independent ResponseError bytes/Header/status and original causes; default owned404 failure has nil acknowledgement as described. Body closes once and accepted bytes are never replayed. Other actual statuses/prebody transport/hook failures retain native error policy; expanded native OkCodes cannot broaden captured acceptance.
  Tests: `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy`.
- Shared fixed request/DoJSON retain configured native prebody retry/reauth/backoff and identical-target redirects. Changed method/origin/path/query blocked before transport. Nil body cannot become JSONnull; RawBody/KeepResponseBody/JSONResponse/serialized body ownership cannot mutate before another attempt; original/hook/marshal causes retained. Native reauth ErrOriginal/ErrReauth remain in original fields. Existing native image bindings unchanged; no native cache API/inventory row fabricated.
  Tests: `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Official API history states2.18/cache-enabled for clean/prune; Python Cache max_microversion remains2.15. Go preserves supplied source version policy and adds no discovery/minimum-version gate, cap mirroring, background scheduler, automatic cleanup or backend-state proof.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Each cache call with default arguments sends the same bodyless request as Python: GET cache, PUT cache/{image}, DELETE cache/{image} with 404 ignored, DELETE cache without x-image-cache-clear-target for target both, GET cache/nodes/{image}, POST cache/clean and POST cache/prune, and decodes cached/queued images, node URLs and prune counts.
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageCacheDefaultRequests`.

Python/Go differences:

- Python queue/delete/clear/clean proxy calls return None; get returns Cache Resource and nodes/prune return typing.cast JSON without runtime schema validation. Go returns actual acknowledgements or strict canonical owned typed/raw data, preserving response causes and requiring captured exact statuses.
- Official API history states2.18/cache-enabled for clean/prune; Python Cache max_microversion remains2.15. Go preserves supplied source version policy and adds no discovery/minimum-version gate, cap mirroring, background scheduler, automatic cleanup or backend-state proof.
- Go accepts only the documented success status of each cache route; Python accepts any status below 400.
- Python Cache caps the negotiated microversion at 2.15; Go sends the configured Image microversion and adds no capability discovery.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:image/v2/prune_cache`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.PruneCache`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCacheHeaders`

- PruneCache sends bodyless/query-free POST at captured base plus fixed cache/prune, requiring original actual200 only. DirectID has only requested cache endpoint; explicit Name for per-ID calls uses unchanged native all-pages exact-name resolution, with no synthetic Ref/ID for singleton routes.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Nonnull canonical root object requires total_files_pruned and total_bytes_pruned, each exact signedint64 integer lexeme; absent/null/string/fraction/exponent/overflow fail. Explicit zeros retained. Embedded Metadata owns unknown/raw/case-decoy values, independent actual200 Header, with links/dates passive and no inferred capacity/completion.
  Tests: `image/cache_contracts_test.go:TestImageCacheCanonicalModelsAndRawPrecision`, `image/cache_core_test.go:TestImageCacheCoreCanonicalPrecisionAndIndependentMetadata`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Source/context/client/provider/prefix/ordinary headers/ref captured and validated before callbacks. Concrete full replacement and helper input maps/optional bool/slice/percallback configs owned before/after one application; nil/error callback/protected/conflicting header invalid before HTTP. Cache-owned target header rejected in source/options on all7. Options and captured route/header stay fixed while original provider token stays live; provider replacement/unsafe retarget and postName source changes rejected.
  Tests: `image/cache_contracts_test.go:TestImageCachePreparedOptionsAndLiveSource`, `image/cache_core_test.go:TestImageCacheCoreCompletePreflightAndOwnedTargetHeader`, `image/cache_options_test.go:TestImageCacheOptionsSnapshotReplacementAndLastWins`, `image/cache_options_test.go:TestImageCacheOptionsOwnPointersCallbacksAndSlices`, `image/cache_options_test.go:TestImageCacheOptionsProtectedHeadersAndCompleteDefaults`, `image/cache_options_test.go:TestImageCacheOptionsParallelReusableHelpers`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`.
- Typed200 read/Close/context/wholeUTF8/JSON/canonical failure returns nil typed result plus owned ResponseError retaining actual bytes/Header/status and all original read/Close/ctx.Err/custom causes. Body closes once and accepted bytes are never replayed. Other actual statuses/prebody transport/hook failures retain native error policy; expanded native OkCodes cannot broaden captured acceptance.
  Tests: `image/cache_contracts_test.go:TestImageCacheAcceptedEvidenceAndFailureRetention`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Shared fixed request/DoJSON retain configured native prebody retry/reauth/backoff and identical-target redirects. Changed method/origin/path/query blocked before transport. Nil body cannot become JSONnull; RawBody/KeepResponseBody/JSONResponse/serialized body ownership cannot mutate before another attempt; original/hook/marshal causes retained. Native reauth ErrOriginal/ErrReauth remain in original fields. Existing native image bindings unchanged; no native cache API/inventory row fabricated.
  Tests: `image/cache_contracts_test.go:TestImageCacheProviderHooksAndCauseSafeMissing`, `image/cache_core_test.go:TestImageCacheCoreCapturedSourceAndPrebodyGuards`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`.
- Only explicit actual200 files/bytes counts are returned, including zero values. Server owns configured capacity/pruning/policy/node-local behavior. Python max2.15 versus documented2.18 does not become a new Go discovery/minimum-version gate.
  Tests: `image/cache_contracts_test.go:TestImageCacheFixedRoutesAndClearTargets`, `image/cache_core_test.go:TestImageCacheCoreFixedRoutesAndActualEvidence`, `connection_image_cache_test.go:TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements`, `internal/cmd/sdkgen/glance_cache_test.go:TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults`, `image/cache_contracts_test.go:TestImageCacheCanonicalModelsAndRawPrecision`, `image/cache_core_test.go:TestImageCacheCoreCanonicalPrecisionAndIndependentMetadata`, `image/cache_core_test.go:TestImageCacheCoreMalformedModelsRetainAcceptedProof`.
- Each cache call with default arguments sends the same bodyless request as Python: GET cache, PUT cache/{image}, DELETE cache/{image} with 404 ignored, DELETE cache without x-image-cache-clear-target for target both, GET cache/nodes/{image}, POST cache/clean and POST cache/prune, and decodes cached/queued images, node URLs and prune counts.
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageCacheDefaultRequests`.

Python/Go differences:

- Python queue/delete/clear/clean proxy calls return None; get returns Cache Resource and nodes/prune return typing.cast JSON without runtime schema validation. Go returns actual acknowledgements or strict canonical owned typed/raw data, preserving response causes and requiring captured exact statuses.
- Only explicit actual200 files/bytes counts are returned, including zero values. Server owns configured capacity/pruning/policy/node-local behavior. Python max2.15 versus documented2.18 does not become a new Go discovery/minimum-version gate.
- Go accepts only the documented success status of each cache route; Python accepts any status below 400.
- Python Cache caps the negotiated microversion at 2.15; Go sends the configured Image microversion and adds no capability discovery.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

type CacheEvidence struct {
    State *image.ImageCache
    Queued *image.ImageCacheAcknowledgement
    Nodes *image.CachedImageNodesResult
    Deleted *image.ImageCacheAcknowledgement
    Cleared *image.ImageCacheAcknowledgement
    Cleaned *image.ImageCacheAcknowledgement
    Pruned *image.CachePruneResult
}

func inspectAndManageCache(ctx context.Context, service *image.Service, ref resource.Ref, target image.CacheTarget) (out CacheEvidence, err error) {
    options := []image.CacheOption{
        image.WithCacheOpts(image.CacheOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithCacheHeader("X-Cache-Request", "manage"),
        image.WithCacheHeaders(map[string]string{"X-Cache-Trace": "sample"}),
    }
    out.State, err = service.GetImageCache(ctx, options...)
    if err != nil { return out, err }
    out.Queued, err = service.QueueImage(ctx, ref, options...)
    if err != nil { return out, err }
    out.Nodes, err = service.CachedImageNodes(ctx, ref, options...)
    if err != nil { return out, err }
    out.Deleted, err = service.CacheDeleteImage(ctx, ref,
        image.WithCacheDeleteOpts(image.CacheDeleteOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithCacheDeleteHeader("X-Cache-Request", "delete"),
        image.WithCacheDeleteHeaders(map[string]string{"X-Cache-Trace": "sample"}),
        image.WithCacheDeleteIgnoreMissing(false))
    if err != nil { return out, err }
    out.Cleared, err = service.ClearCache(ctx,
        image.WithClearCacheOpts(image.ClearCacheOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithClearCacheHeader("X-Cache-Request", "clear"),
        image.WithClearCacheHeaders(map[string]string{"X-Cache-Trace": "sample"}),
        image.WithClearCacheTarget(target))
    if err != nil { return out, err }
    out.Cleaned, err = service.CleanCache(ctx, options...)
    if err != nil { return out, err }
    out.Pruned, err = service.PruneCache(ctx, options...)
    return out, err
}

func cacheTargets() [3]image.CacheTarget {
    return [3]image.CacheTarget{image.CacheBoth, image.CacheOnly, image.QueueOnly}
}
```

```python
def inspect_and_manage_cache(conn, image_id, target="both"):
    state = conn.image.get_image_cache()
    conn.image.queue_image(image_id)
    nodes = conn.image.cached_image_nodes(image_id)
    conn.image.cache_delete_image(image_id, ignore_missing=False)
    conn.image.clear_cache(target=target)
    conn.image.clean_cache()
    pruned = conn.image.prune_cache()
    return state, nodes, pruned
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L97-L155)
- [Cache/CachedImage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/cache.py#L24-L152)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#cache-manage)
- [clear controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L219-L240)
- [Cache](https://docs.openstack.org/glance/latest/admin/cache.html#controlling-image-cache-using-v2-api)
- [204 serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L349-L366)
- [xattr](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/image_cache/drivers/xattr.py)
- [sqlite](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/image_cache/drivers/sqlite.py)
- [centralized DB](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L2306-L2318)
- [nodes controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L189-L217)
- [queue controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L252-L276)
- [Cache Contracts Test](cache_contracts_test.go)
- [core tests](cache_core_test.go)
- [option tests](cache_options_test.go)
- [Connection test](../connection_image_cache_test.go)
- [generator test](../internal/cmd/sdkgen/glance_cache_test.go)
