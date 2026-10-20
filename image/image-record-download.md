# Image Record Download

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/download_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.DownloadTo`
- `github.com/JSYoo5B/go-openstacksdk/image.WithDownloadImageOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithDownloadChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithDownloadStorePreferences`
- `github.com/JSYoo5B/go-openstacksdk/image.WithDownloadChecksumVerification`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.DownloadImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DownloadImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDownloadOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDownloadStream`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDownloadChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDownloadStorePreferences`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDownloadChecksumVerification`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDownloadHashFactory`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDownloadHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDownloadHeaders`

- Legacy typed writer-only DownloadTo profile only: DownloadTo defaults to metadata GET200 then full binary GET200, using a bounded 1MiB copy chunk and available checksum verification. It retains independent actual native Image, raw metadata body/header/status, actual accepted binary header/status and exact BytesWritten without keeping the full payload or fabricating completion.
  Tests: `image/download_contracts_test.go:TestDownloadToDefaultSequenceAndEvidence`, `image/download_core_test.go:TestDownloadCoreFixedRouteBoundedCopyAndOwnedEvidence`.
- Legacy typed writer-only DownloadTo profile only: Service/source/provider/context/ref/writer and concrete options are validated before any HTTP or Write. Nil and typed-nil writers, invalid route/source/header/Range/If-Range controls, nil callbacks, nonpositive or over64MiB chunks and malformed store preferences stop locally. Nil chunk selects1MiB and nil verification selects true; explicit false remains distinct.
  Tests: `image/download_contracts_test.go:TestDownloadToCompletePreflight`, `image/download_options_test.go:TestDownloadOptionsDefaultsAndBoundedPreflight`.
- Legacy typed writer-only DownloadTo profile only: Whole-policy and leaf helpers snapshot pointers and preferences for last replacement, independent reuse and concurrent application. Custom callbacks run once before requests and their retained configuration cannot alter transfer options. Preferences preserve order and duplicates in one URL-encoded comma-separated prefer value; nil/empty omit it.
  Tests: `image/download_contracts_test.go:TestDownloadToPreparedSnapshotsAndPreferences`, `image/download_options_test.go:TestDownloadOptionsOwnFullReplacementAndConcurrentReuse`, `image/download_options_test.go:TestDownloadOptionsCustomCallbacksOnceAndOwnedPointers`.
- Legacy typed writer-only DownloadTo profile only: Explicit ID chooses one fresh metadata GET; Name uses existing exact-name full-page resolution followed by fresh GET. The exact lowercase canonical metadata id must equal the fixed chosen ID. Incidental typed case aliases and response self/file/schema/Location cannot choose the binary route. Actual native projection remains separate from original response evidence; Name pager continuation keeps existing native semantics.
  Tests: `image/download_contracts_test.go:TestDownloadToCanonicalIdentityAndIntegrity`, `image/download_core_test.go:TestDownloadCoreRetainsAcceptedMetadataAndStopsInvalidIdentity`, `image/download_core_test.go:TestDownloadCoreFixedRouteBoundedCopyAndOwnedEvidence`.
- Legacy typed writer-only DownloadTo profile only: Canonical complete nonempty os_hash_algo/os_hash_value wins over legacy metadata checksum, which wins over literal binary Content-MD5. Incomplete pairs fall back; disabled verification skips canonical hash-field validation while retaining native Image decode. The bounded algorithms md5/sha1/sha224/sha256/sha384/sha512/sha512_224/sha512_256 are implemented. Unsupported primary algorithms and malformed canonical hash types fail before binary GET. Expected strings are not hex-normalized or base64-decoded.
  Tests: `image/download_contracts_test.go:TestDownloadToCanonicalIdentityAndIntegrity`, `image/download_core_test.go:TestDownloadCoreCanonicalHashSelectionAndAlgorithms`, `image/download_core_test.go:TestDownloadCoreNoDataRejectedPartialAndLiteralHashMismatch`.
- Legacy typed writer-only DownloadTo profile only: Only actual binary200/204 are accepted. Actual204 performs no Write, clears checksum evidence and retains header/status. Unsolicited206 or HTTP errors retain metadata plus original native rejection without an accepted binary status claim. Metadata-selected algorithm/expected survive binary open/rejection failures. Nonempty Content-Range on actual200 is rejected before Write with accepted200/header and selected metadata proof retained.
  Tests: `image/download_contracts_test.go:TestDownloadToNoDataAndNativeCompatibility`, `image/download_core_test.go:TestDownloadCoreNoDataRejectedPartialAndLiteralHashMismatch`.
- Legacy typed writer-only DownloadTo profile only: Before accepted metadata200 the result is nil. Accepted metadata read/decode/canonical/native-model failures retain raw200 body/header/status and original error, stop before binary GET and do not Write. Later file/open/read/write/short-write failures retain prior metadata, any actual accepted binary response and exact writer-reported successful bytes without rollback, body replay or recovery requests.
  Tests: `image/download_contracts_test.go:TestDownloadToPartialTransferFailures`, `image/download_core_test.go:TestDownloadCoreRetainsAcceptedMetadataAndStopsInvalidIdentity`.
- Legacy typed writer-only DownloadTo profile only: The borrowed writer is never closed, sought or truncated. The owned chunk loop does not use ReaderFrom/WriterTo shortcuts, enforces short writes and counts reported written bytes. Simultaneous read/write errors, accepted-body Close errors and context/custom causes remain inspectable. SDK accepted bodies close once and native Request owns rejected bodies. Actual/Complete require full EOF and successful writes; Verified requires a matching digest. Typed DownloadChecksumMismatchError unwraps ErrChecksumMismatch and carries algorithm/expected/actual. Close-only failure after valid full transfer preserves Complete/Verified proof; partial failures do not claim them.
  Tests: `image/download_contracts_test.go:TestDownloadToPartialTransferFailures`, `image/download_contracts_test.go:TestDownloadToBorrowedWriterCancellationAndClose`, `image/download_core_test.go:TestDownloadCoreJoinsReadWriteCloseAndPreservesFullDigestProof`, `image/download_core_test.go:TestDownloadCoreNoDataRejectedPartialAndLiteralHashMismatch`.
- Legacy typed writer-only DownloadTo profile only: Captured prefix/headers/client selection use the original provider live token and configured pre-body reauthentication/retry. Original source/provider changes are checked before subsequent requests. Binary redirects are disabled and no accepted-body read/write failure restarts download. Parent deadlines, HTTPClient timeout, cancellation and custom cause remain in returned failures; tests release borrowed blocking writers and close all handles. Name pagination retains the existing pager contract rather than adding a new continuation policy.
  Tests: `image/download_contracts_test.go:TestDownloadToProviderSourceAndRetryBoundaries`, `image/download_contracts_test.go:TestDownloadToBorrowedWriterCancellationAndClose`.
- Legacy typed writer-only DownloadTo profile only: Existing generated ImageData.Download raw-body API stays independently callable with native defaultGET200 behavior and without workflow metadata or integrity. Generator checks the native Download context/client/id/result signature and Image canonical ID/checksum string fields and rejects type/tag/arity drift. Upper writer documentation does not invent a native operation, model or resource collection; native declarations and inventory remain unchanged.
  Tests: `image/download_contracts_test.go:TestDownloadToNoDataAndNativeCompatibility`, `internal/cmd/sdkgen/glance_download_test.go:TestGlanceDownloadKeepsNativeStreamAndDocumentsWriterWorkflow`, `internal/cmd/sdkgen/glance_download_test.go:TestGlanceDownloadRejectsNativeSignatureIdentityAndChecksumDrift`.
- Capture Service/API/Images/raw Provider/Endpoint/ResourceBase/type/microversion/ordinary headers and sticky parent guard before caller metadata marshal, options or current-location callbacks. Own raw maps, byte slices, preferences and option pointers; concrete full/leaf helpers and custom callbacks run once and Service/Connection use one prepared source through fetch, transfer and deferred stream consumption.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadCapturesConstructorOptionsLocationAndHeadersBeforeCallbacks`, `image/image_record_download_test.go:TestImageRecordDownloadOptionMergeReplacementReuseAndStickyCallbackGuards`, `image/image_record_download_test.go:TestImageRecordDownloadDeferredStreamReadAndClosePreserveCausesWithoutInventedBodyProof`, `image/image_record_download_test.go:TestImageRecordDownloadBorrowedWriterCallbacksKeepStickyGuardsAndPartialCount`, `image/image_record_download_test.go:TestImageRecordDownloadDeferredBodyKeepsCapturedContextAndHTTPTimeoutAlive`, `image/image_record_download_test.go:TestImageRecordDownloadCustomHashWriteAndSumCannotHideFaultsOrChangeCompletedProof`, `image/image_record_download_test.go:TestImageRecordDownloadNativeBackoffRetainsOriginalRejectionAcrossCallbackGuards`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadRejectsInvalidInvocation`.
- Pinned Proxy._get_resource accepts literal ID, existing Image, and an actual dict/Munch branch despite the narrower public annotation. Literal construction starts unsynchronized; dict/Munch starts synchronized then _update({}); existing Image keeps its private current/original/dirty state and plain data while _update({}) resets ImportMethods/microversion and rematerializes descriptors/location. Go must cover whole owned Record plus raw JSON constructor seed and literal ID without deriving raw state from an externally edited view. Name resolution belongs only to the separate cloud wrapper.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadRawSynchronizedConstructorProjectsAllFieldsWithoutUploadHooks`, `image/image_record_download_test.go:TestImageRecordDownloadPrivatePendingRecordIgnoresPublicEditsAndFetchControlsBaseline`, `image/image_record_download_test.go:TestImageRecordDownloadPreflightRejectsAmbiguousSelectorsAndUnsafeLocalInputs`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Project every one of 64 raw inherited/declared Body fields plus computed location before HTTP, preserving aliases, unknown-property packing and raw defaults/conversions separately from components. Source dict constructor collisions self/connection/_synchronized and mutable microversion/class behavior need explicit fixed-control bounds. There is no private _create hook in download: __conflicting_attrs, base_path, resource_type and filename are ordinary unknown metadata keys here, not upload controls or a local output filename.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadRawSynchronizedConstructorProjectsAllFieldsWithoutUploadHooks`, `image/image_record_download_test.go:TestImageRecordDownloadPreflightRejectsAmbiguousSelectorsAndUnsafeLocalInputs`, `image/image_record_download_test.go:TestImageRecordDownloadCapturesConstructorOptionsLocationAndHeadersBeforeCallbacks`.
- DownloadMixin performs self.fetch(session) in every output, memory and stream mode, before reading checksum descriptors, building prefer or issuing binary GET. Fetch negotiates Resource microversion, prepares /images/{id}, uses normal metadata retry/auth and translates the full response. No conditional skip because a Record already has hashes/status, no queued/active gate, no public Get facade recapture and no final GET after binary transfer.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadPrivatePendingRecordIgnoresPublicEditsAndFetchControlsBaseline`, `image/image_record_download_test.go:TestImageRecordDownloadMetadataFailuresKeepActualEvidenceAndNeverOpenOutput`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Accepted valid JSON object metadata strips self, normalizes all raw aliases/unknown properties, overlays the supplied private constructor/current state and cleans the full new baseline. Sparse response can retain prior checksum/id/format/property fields. Metadata response replaces ImportMethods from its actual header, rematerializes all descriptors/location and retains borrowed data. Metadata envelope/header/status/Wire remain independent of current/original/dirty channels and binary response.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadPrivatePendingRecordIgnoresPublicEditsAndFetchControlsBaseline`, `image/image_record_download_test.go:TestImageRecordDownloadCapturesConstructorOptionsLocationAndHeadersBeforeCallbacks`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Source tolerates JSON ValueError without overlay or clean but still consumes headers/location and rematerializes the descriptor view. Valid nonobject JSON or projected descriptor errors stop before file GET. Go strict UTF-8/Unicode/object/atomic partial-record bounds must be documented. Accepted metadata read/Close/context/source/decode failures retain actual metadata bytes/header/status and the appropriate seed; native rejection cannot start binary IO or output opening.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadPrivatePendingRecordIgnoresPublicEditsAndFetchControlsBaseline`, `image/image_record_download_test.go:TestImageRecordDownloadMetadataFailuresKeepActualEvidenceAndNeverOpenOutput`.
- Build the binary file target from the private fetched raw id, including an id changed by metadata response overlay or retained through sparse/invalid JSON. Do not require equality with the initial requested id as legacy DownloadTo does. Response self/file/schema/Location and editable public Resource views cannot redirect. Source urljoin stringifies falsey/nonstring ids and strips slashes; Go nonblank valid Unicode fixed one-escaped literal identity is an explicit transport-safety bound.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadFetchedIdentityIsOnceEscapedAndNeverUsesResponseLinks`, `image/image_record_download_test.go:TestImageRecordDownloadPreflightRejectsAmbiguousSelectorsAndUnsafeLocalInputs`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Truthy Source store_preferences uses a literal comma join in supplied order, preserving repeats and arbitrary string elements including empty strings, commas and whitespace; falsey input omits prefer. Join failure occurs after metadata fetch but before binary GET. Go typed ordered string slices and URL-encoded query are explicit bounds; do not silently reuse the legacy nonempty/no-comma identifier validator for this whole profile. Server store ordering/availability/cache effects remain server-owned.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadStorePreferencesPreserveRawOrderAndSourceJoinSemantics`, `image/image_record_download_test.go:TestImageRecordDownloadCapturesConstructorOptionsLocationAndHeadersBeforeCallbacks`, `image/image_record_download_test.go:TestImageRecordDownloadNativeRetryAndReauthenticationPreserveFixedGETOwnership`.
- Source session.get(fileURL, stream=stream) occurs after fetch and before hasher construction, without _translate_response or explicit status raise. Go must document native HTTP rejection versus Source quiet error-body consumption, retain actual accepted binary header/status separately, preserve native rejection/fault causes and choose a declared status/range policy. Metadata evidence survives all file-open/HTTP/transport/guard failures; no implicit binary-to-Record translation.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadRawHashSelectionOccursAfterGETAndKeepsLiteralExpected`, `image/image_record_download_test.go:TestImageRecordDownloadCustomPrimaryFactoryRunsOnceAfterBinaryAndKeepsFallbackOwned`, `image/image_record_download_test.go:TestImageRecordDownloadNativeRejectionsKeepPhaseEvidenceAndNeverConsumeOutput`, `image/image_record_download_test.go:TestImageRecordDownloadNativeRetryAndReauthenticationPreserveFixedGETOwnership`, `image/image_record_download_test.go:TestImageRecordDownloadAcceptedBinaryBoundaryFailuresKeepHeadersAndReleaseUnreadBody`, `image/image_record_download_test.go:TestImageRecordDownloadNativeBackoffRetainsOriginalRejectionAcrossCallbackGuards`, `image/image_record_download_test.go:TestImageRecordDownloadRejectedBinaryReadAndCloseFaultsPreventCleanReplay`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- After accepted binary GET, read the fetched descriptor raw JSON values hash_value/hash_algo/checksum. A truthy algorithm/value pair tries hashlib.new(str(raw algorithm)); expected value remains raw. Use PythonTruthy/PythonString for JSON values, not eager string-only decoding. Truthy nonstring expected values cause a later literal mismatch; falsey values skip that selection. No checksum selection/unsupported-algorithm rejection before file GET in the Source default profile.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadRawHashSelectionOccursAfterGETAndKeepsLiteralExpected`, `image/image_record_download_test.go:TestImageRecordDownloadCustomPrimaryFactoryRunsOnceAfterBinaryAndKeepsFallbackOwned`.
- hashlib.new may return any runtime-supported algorithm. Source catches only ValueError: when the algorithm string appears in that error text it falls through to MD5/unverified; when absent it raises SDKException, while other exception types propagate. Record this branch and cause order explicitly. A typed Go finite registry/factory policy must state its domain and must not describe the legacy eight algorithms as the whole Python registry. The agreed finite SDK registry includes MD5/SHA1/SHA2/truncations/SHA3/BLAKE2 and case/hyphen aliases; default SHAKE succeeds at selection but consumed completion returns the typed missing-length error. Optional primary-only function factory uses ErrImageRecordDownloadHashUnsupported for fallback and other errors for the Source fatal branch; MD5 fallback always uses SDK md5.New. Dynamic OpenSSL registry remains explicit R1/S1.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadSupportedHashRegistryUsesIndependentDigestVectors`, `image/image_record_download_test.go:TestImageRecordDownloadCustomPrimaryFactoryRunsOnceAfterBinaryAndKeepsFallbackOwned`, `image/image_record_download_test.go:TestImageRecordDownloadSHAKESelectsStreamButConsumedModeRequiresDigestLength`, `image/image_record_download_test.go:TestImageRecordDownloadCustomHashWriteAndSumCannotHideFaultsOrChangeCompletedProof`.
- If no hasher was selected, truthy metadata checksum wins over binary Content-MD5; md5 is constructed with usedforsecurity=False. _verify_checksum checks only truthy expected values, compares hasher.hexdigest() exactly against the raw expected value and raises InvalidResponse on mismatch. No uppercase/hex normalization, trimming, base64 decoding or comparison against a generated native string model. Preserve raw expected evidence for nonstrings and selected algorithm separately.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadRawHashSelectionOccursAfterGETAndKeepsLiteralExpected`, `image/image_record_download_test.go:TestImageRecordDownloadStreamMD5CompatibilityKeepsActualHeaderAndRawExpectedSeparate`, `image/image_record_download_test.go:TestImageRecordDownloadAcceptedStatusRangeAndEmptyBodyUseWholeChecksumProfile`.
- When no hasher exists Source logs one warning after binary GET and proceeds unverified. Expose unavailable verification honestly rather than failing for absent hash or fabricating Verified. Any Go explicit verification-disable option is additive; its false policy may skip hash factory/validation while still fetching and projecting the whole metadata Record. Source default checksum logic remains enabled in consumed output/memory modes.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadRawHashSelectionOccursAfterGETAndKeepsLiteralExpected`, `image/image_record_download_test.go:TestImageRecordDownloadCustomPrimaryFactoryRunsOnceAfterBinaryAndKeepsFallbackOwned`.
- Truthy output selects the output branch before stream-only handling; stream=True plus writer/path still consumes and verifies output and returns the Response after completion. Source passes the original stream flag into requests even for output; stream=False can preload whole response before iteration. A Go bounded always-streamed writer/path path is a documented memory improvement, while the independent stream selector still determines the no-output branch. Typed-nil and simultaneous writer/path policy must be concrete and predictable.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadPreflightRejectsAmbiguousSelectorsAndUnsafeLocalInputs`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Source output io.IOBase is borrowed: iterate resp.iter_content(chunk_size), optionally hash each chunk before yielding it, call write(chunk), and never seek/truncate/flush/close the borrowed output. Source ignores write counts; Go short-write checks, exact successful byte counts, Read-only/Write-only views and bounded 1MiB default copy loop are explicit improvements. Chunk size is only used in output iteration in Source; Go 1..64MiB preflight policy is a declared bound, not a Python validator.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadPreflightRejectsAmbiguousSelectorsAndUnsafeLocalInputs`, `image/image_record_download_test.go:TestImageRecordDownloadMemoryWriterFaultsRetainPartialProgressAndCompletedCloseProof`, `image/image_record_download_test.go:TestImageRecordDownloadBorrowedWriterCallbacksKeepStickyGuardsAndPartialCount`.
- Non-IOBase truthy output opens path with wb only after metadata GET, binary GET and hash selection. It creates/truncates immediately, owns the output handle close on success/failure and leaves any partial or mismatching file in place. Do not pre-open the path before Source phases, reinterpret a metadata filename, synthesize directory creation, atomic rename, rollback or deletion. Local open/write/Close/context errors retain metadata and accepted binary evidence; SDK HTTP body cleanup must still occur.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadMetadataFailuresKeepActualEvidenceAndNeverOpenOutput`, `image/image_record_download_test.go:TestImageRecordDownloadOwnedOutputPathOpensAfterHashSelectionAndLeavesPartialFiles`.
- _integrity_iter updates the hash before yielding each chunk and verifies only after complete exhaustion; output can therefore be fully written before checksum mismatch is raised. A Source writer failure prevents final verification even though an attempted chunk was hashed. Go Complete/Actual/Verified proof must describe its chosen successfully-written versus consumed-byte scope and remain false for partial read/write/no-progress/context failures. A later Close-only failure does not erase valid complete-digest proof.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadRawHashSelectionOccursAfterGETAndKeepsLiteralExpected`, `image/image_record_download_test.go:TestImageRecordDownloadAcceptedStatusRangeAndEmptyBodyUseWholeChecksumProfile`, `image/image_record_download_test.go:TestImageRecordDownloadMemoryWriterFaultsRetainPartialProgressAndCompletedCloseProof`, `image/image_record_download_test.go:TestImageRecordDownloadCustomHashWriteAndSumCannotHideFaultsOrChangeCompletedProof`.
- No truthy output and stream=False is the default whole-memory Response.content branch. Source returns Response, not bytes, despite the docstring; it hashes the entire content then verifies if available. Go should return owned bytes plus actual Response metadata and Record/metadata evidence; return partial buffered bytes on read failures. Do not implement a reader-only or writer-only substitute. Agreed Go adds no memory cap; default remains whole memory.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadMemoryWriterFaultsRetainPartialProgressAndCompletedCloseProof`.
- No truthy output and stream=True returns the unconsumed Response immediately after hash selection without wrapping _integrity_iter or verifying downloaded bytes. The caller owns reading and must close an unconsumed Response. Go should return io.ReadCloser with captured route/context/source checks, no buffering/replay/automatic checksum verification, idempotent owned Close and deferred read/Close errors. Data consumed by caller cannot mutate or claim full checksum proof in an immutable returned result unless an explicit stream-proof API is introduced.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadSHAKESelectsStreamButConsumedModeRequiresDigestLength`, `image/image_record_download_test.go:TestImageRecordDownloadDeferredStreamReadAndClosePreserveCausesWithoutInventedBodyProof`, `image/image_record_download_test.go:TestImageRecordDownloadDeferredBodyKeepsCapturedContextAndHTTPTimeoutAlive`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Only stream-only mode rewrites resp.headers[content-md5] to selected expected raw value when algorithm is md5 and expected truthy, including metadata checksum taking precedence over the actual header. Keep actual binary headers independent from this compatibility view; all modes retain raw expected values. Source case-insensitive response header container can store nonstring values; Agreed Go keeps actual Header independent, raw ContentMD5 json.RawMessage and cloned CompatibilityHeader with PythonString projection rather than rejecting the raw metadata domain.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadStreamMD5CompatibilityKeepsActualHeaderAndRawExpectedSeparate`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Source has no binary status/range gate: quiet 4xx/5xx/unsolicited206 may be consumed or returned, and 204 still participates in checksum comparison against an empty payload rather than clearing the selected checksum. Range is not a public download parameter but configured Adapter headers can affect the wire. Agreed Go accepts actual200..399 including206/204 without Content-Range rejection and returns native400..599 errors as an explicit difference. The whole Source default checksum branch for empty accepted data cannot silently inherit legacy checksum=nil on204.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAcceptedStatusRangeAndEmptyBodyUseWholeChecksumProfile`, `image/image_record_download_test.go:TestImageRecordDownloadNativeRejectionsKeepPhaseEvidenceAndNeverConsumeOutput`.
- The pinned SDK never explicitly calls resp.close in DownloadMixin; requests consumption and the path context manager own their dependency-specific cleanup. Go should close its accepted HTTP body exactly once in memory/writer/path and transfer that ownership in stream-only mode, retaining read/write/Close/checksum/context/custom cancellation/source guard causes with typed errors. Source wraps every Exception in the output branch as SDKException text, while memory mismatch and stream read failures have different timing; Go errors.Join is a documented improvement.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadMetadataFailuresKeepActualEvidenceAndNeverOpenOutput`, `image/image_record_download_test.go:TestImageRecordDownloadMemoryWriterFaultsRetainPartialProgressAndCompletedCloseProof`, `image/image_record_download_test.go:TestImageRecordDownloadOwnedOutputPathOpensAfterHashSelectionAndLeavesPartialFiles`, `image/image_record_download_test.go:TestImageRecordDownloadDeferredStreamReadAndClosePreserveCausesWithoutInventedBodyProof`, `image/image_record_download_test.go:TestImageRecordDownloadBorrowedWriterCallbacksKeepStickyGuardsAndPartialCount`, `image/image_record_download_test.go:TestImageRecordDownloadAcceptedBinaryBoundaryFailuresKeepHeadersAndReleaseUnreadBody`, `image/image_record_download_test.go:TestImageRecordDownloadRejectedBinaryReadAndCloseFaultsPreventCleanReplay`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Preserve captured method/path/query/provider/bindings and ordinary headers through every physical metadata/file attempt, auth wait/reauth/retry callback and borrowed IO callback. Safe binary GET pre-body auth/retry remains usable; never restart after accepted-body reading or writing. Binary redirects must not send the token outside the fixed target. Deferred stream Read/Close needs guards and original context/HTTPClient timeout lifetime; a user blocking writer remains caller-owned and cannot be forcibly released by the SDK.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadOptionMergeReplacementReuseAndStickyCallbackGuards`, `image/image_record_download_test.go:TestImageRecordDownloadNativeRetryAndReauthenticationPreserveFixedGETOwnership`, `image/image_record_download_test.go:TestImageRecordDownloadDeferredStreamReadAndClosePreserveCausesWithoutInventedBodyProof`, `image/image_record_download_test.go:TestImageRecordDownloadBorrowedWriterCallbacksKeepStickyGuardsAndPartialCount`, `image/image_record_download_test.go:TestImageRecordDownloadDeferredBodyKeepsCapturedContextAndHTTPTimeoutAlive`, `image/image_record_download_test.go:TestImageRecordDownloadCustomHashWriteAndSumCannotHideFaultsOrChangeCompletedProof`, `image/image_record_download_test.go:TestImageRecordDownloadAcceptedBinaryBoundaryFailuresKeepHeadersAndReleaseUnreadBody`, `image/image_record_download_test.go:TestImageRecordDownloadNativeBackoffRetainsOriginalRejectionAcrossCallbackGuards`, `image/image_record_download_test.go:TestImageRecordDownloadRejectedBinaryReadAndCloseFaultsPreventCleanReplay`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Pinned GET /images/{id} enforces get_image, then GET /images/{id}/file enforces download_image; deactivated data is forbidden for non-admin and allowed to pass this check for admin. Truthy prefer adds download_from_store, validates enabled backend/configured identifiers, and stable-partitions matching existing image locations before others with fallback. Server cache can bypass store preference deserialization. Forward private/shared/community/public/owner/admin metadata without client RBAC or storage capability guesses; actual deployed policy/storage remains unproved.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadStorePreferencesPreserveRawOrderAndSourceJoinSemantics`, `image/image_record_download_test.go:TestImageRecordDownloadAcceptedStatusRangeAndEmptyBodyUseWholeChecksumProfile`, `image/image_record_download_test.go:TestImageRecordDownloadNativeRejectionsKeepPhaseEvidenceAndNeverConsumeOutput`.
- The separate ImageCloudMixin.download_image requires exactly one non-None output_path/output_file, resolves find_image(name_or_id, ignore_missing=False) and delegates with output_file or output_path. This wrapper is not a dependency of lower Proxy.download_image and must remain separately scoped. Modern public create_image full20-file12-contract graph, private metadata POST/create engine, Upload/Stage/Import/task/Swift/wait/checksum-upload/rollback are not download dependencies.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadPrivatePendingRecordIgnoresPublicEditsAndFetchControlsBaseline`, `image/image_record_download_test.go:TestImageRecordDownloadFetchedIdentityIsOnceEscapedAndNeverUsesResponseLinks`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.
- Preserve all10 legacy DownloadTo contracts, 18 real test refs and generated native raw Download profile. Add the complete owned DownloadImageRecord public graph with all output modes, full raw constructor/Record/fetch semantics and checksum branches. Only selected python:image/v2/download_image may be promoted after real focused/full45/actual generator zero drift/guide/install local+remote proofs and exact Source/contract/test declarations. Runtime class extensions R1, API cache C1 and full Adapter/session S1 remain shared unresolved scope; this static audit executes no tests or promotion.
  Tests: `image/image_record_download_test.go:TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream`, `image/image_record_download_test.go:TestImageRecordDownloadOptionMergeReplacementReuseAndStickyCallbackGuards`, `image/image_record_download_test.go:TestImageRecordDownloadNativeRejectionsKeepPhaseEvidenceAndNeverConsumeOutput`, `image/image_record_download_test.go:TestImageRecordDownloadNativeRetryAndReauthenticationPreserveFixedGETOwnership`, `image/image_record_download_test.go:TestImageRecordDownloadDeferredStreamReadAndClosePreserveCausesWithoutInventedBodyProof`, `image_record_download_integration_test.go:TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage`.

Python/Go differences:

- Complete finite owned Proxy.download_image graph: literal/private Record/synchronized raw constructor, whole64 raw Body/65 descriptor view, mandatory seeded metadata fetch with sparse overlay/invalid-JSON tolerance and post-fetch identity, after-GET raw checksum selection/fallback and all borrowed-writer/owned-path/default-memory/caller-stream branches. This go_mapping is not full SDK supported parity.
- Legacy typed writer-only DownloadTo profile only: DownloadTo is an explicit bounded writer workflow with a fresh metadata request and optional available-checksum validation. Python accepts mutable Image or ID, output path/IOBase and stream/no-output modes, with a default whole-memory response branch. Existing raw native Download remains independently available.
- Legacy typed writer-only DownloadTo profile only: Go fixes the selected ID and requires matching lowercase canonical metadata id, using raw decision fields alongside independent native/raw evidence. Python metadata fetch mutates its supplied Resource and later uses self.id; complete cache/descriptors/session/adapter identity semantics are broader.
- Legacy typed writer-only DownloadTo profile only: Go concrete options freeze caller pointers/slices and callback results before requests, bound chunk size to1..64MiB, preserve ordered repeated preferences and use URL encoding. Python defaults to1MiB but does not impose this bound or Go snapshot policy; both nil/empty preferences omit prefer.
- Legacy typed writer-only DownloadTo profile only: Go supports a bounded standard-library hash registry and rejects an unsupported complete primary pair before binary GET. Python performs the binary GET first and commonly falls through to MD5 or unverified transfer when the caught unsupported-hash error includes the algorithm name. Go disabled verification also skips canonical hash-field checks.
- Legacy typed writer-only DownloadTo profile only: Go exposes selected algorithm/expected and full-EOF Actual/Complete/Verified without changing raw headers, retaining selected metadata hashes on binary failure and full digest proof on Close-only error. Python verifies consumed output after iteration, may rewrite Content-MD5 in raw-stream mode and logs missing hashes instead of returning explicit verification availability.
- Legacy typed writer-only DownloadTo profile only: Go accepts full200/no-data204, rejects unsolicited206 and other statuses, and rejects Content-Range on accepted200 before writing. Pinned Python binary DownloadMixin does not translate response errors, while Proxy.request defaults raise_exc=False, so returned error bodies may be consumed. These are intentionally stronger Go accepted-status and whole-image policies.
- Legacy typed writer-only DownloadTo profile only: Go enforces short-write counts, retains exact partial bytes and joins read/write/Close/context causes including a typed checksum mismatch. Python output ignores write counts and wraps output errors as text; both can leave already-written partial output. Go closes its HTTP body and borrows the writer without path opening or output rollback.
- Go owns immutable JSON-domain Records/options/raw phase receipts and deferred stream guards. Mutable Python class/constructor extensions R1, optional API cache C1 and full Adapter/discovery/session S1 remain unresolved globally. The separate cloud exact-one-output/name wrapper is not closed by this lower operation.
- Go accepts actual200..399 including204/206 without Content-Range rejection and reports native>=400 errors. Pinned binary DownloadMixin has no status translation and may consume/return error bodies. Accepted204 still verifies its empty payload; binary never translates Record or performs a final metadata fetch.
- Go default registry covers MD5/SHA1/SHA2/truncations/SHA3/BLAKE2/SHAKE and a finite case/hyphen alias policy, not every installed OpenSSL hash. Optional primary-only function factory runs after accepted GET; Unsupported sentinel selects separate SDK md5.New fallback, other errors stop. Raw truthy nonstring expectations mismatch after complete output; selected SHAKE consumed mode reports missing output length after EOF.
- Go retains actual binary Header separately from raw ContentMD5 and PythonString CompatibilityHeader. Only stream-only selected md5 compatibility mutates that clone, preserving Source exact algorithm-name comparison; caller stream is unconsumed and never automatically verified. Typed mismatch, successful byte counts and joined read/write/Close/context/source causes are explicit improvements.
- Go writer/path modes always use a bounded1MiB default copy loop rather than Source requests default stream=False buffering; chunk1..64MiB, fixed literal Unicode identity and source/control guards are explicit bounds. Whole-memory default has no cap. Filename opens/truncates only after metadata/file GET/hash selection, owns close and leaves partial data; borrowed output never closes/seeks/flushes.
- Ordered UTF-8 store preference strings preserve repeats/empty/comma/control values and encode one prefer query instead of Source literal comma query. Backend validation/stable location fallback/cache effects and admin deactivated-image/download policy remain server-owned; static pinned evidence does not prove deployed policy or storage.
- Legacy typed DownloadTo and generated native Download remain separate preserved profiles. Modern public create_image12 and Source dynamic R1/C1/S1 remain unresolved; Upload30/Stage28/Import28 reviewed objects are unchanged.

### `python:cloud/_image/ImageCloudMixin/download_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.DownloadCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCloudDownloadStream`

- Missing or duplicate destinations, typed nil writers, invalid filenames, blank identities, nil options and invalid headers fail before HTTP.
  Tests: `image/image_record_cloud_download_test.go:TestCloudImageRecordDownloadStrictFindAndPreflight`.
- DownloadCloudImageRecord finds the image with the owned ignore_missing false Find graph, returning ErrNotFound without download requests when absent, then passes the found record to DownloadImageRecord: mandatory metadata GET of the found identity, binary GET of the fetched identity and default checksum verification, writing to the borrowed writer or SDK-created file. Stream is forwarded but the required destination wins; chunk size and headers reach the lower phases.
  Tests: `image/image_record_cloud_download_test.go:TestCloudImageRecordDownloadFindsThenDownloadsToRequiredOutput`, `image/image_record_cloud_download_test.go:TestCloudImageRecordDownloadStrictFindAndPreflight`.
- A lower download failure returns Found plus the partial download result and the native cause under the DownloadCloudImageRecord operation without nesting the delegated operation.
  Tests: `image/image_record_cloud_download_test.go:TestCloudImageRecordDownloadFindsThenDownloadsToRequiredOutput`.

Python/Go differences:

- Python returns the lower download value or raises SDKException/ResourceNotFound; Go returns ImageRecordCloudDownloadResult separating the strict Find record from the lower download phases and wraps typed errors.
- Python treats an empty output_path as present and then falls back to memory mode; Go treats an empty Filename as absent and rejects it before HTTP. Headers are a Go extension for both phases, and Proxy-only download options keep their defaults because the Cloud signature has no such arguments.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
response = conn.image.download_image(
    "literal-image-id",
    output="ubuntu.qcow2",
    store_preferences=["fast", "archive"],
)
print(response.status_code)
```

```python
with open("ubuntu.qcow2", "wb") as output:
    response = conn.image.download_image("literal-image-id", output=output)

response = conn.image.download_image("literal-image-id")
data = response.content

response = conn.image.download_image("literal-image-id", stream=True)
try:
    with open("ubuntu.qcow2", "wb") as output:
        for chunk in response.iter_content(chunk_size=1048576):
            output.write(chunk)
finally:
    response.close()
```

```sh
go run . -cloud dev -id literal-image-id -mode file -output ubuntu.qcow2
go run . -cloud dev -id literal-image-id -mode writer -output ubuntu.qcow2 -service
go run . -cloud dev -id literal-image-id -mode buffer -prefer fast -prefer archive
go run . -cloud dev -id literal-image-id -mode stream -output ubuntu.qcow2 -timeout 10m
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
)

type preferences []string

func (value *preferences) String() string { return fmt.Sprint([]string(*value)) }
func (value *preferences) Set(store string) error {
    *value = append(*value, store)
    return nil
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    id := flag.String("id", "", "required literal image ID")
    mode := flag.String("mode", "file", "file, writer, buffer or stream")
    output := flag.String("output", "", "output path for file, writer or stream")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    chunk := flag.Int("chunk-size", 1<<20, "SDK consumed-mode copy chunk in bytes, up to 64 MiB")
    verify := flag.Bool("verify-checksum", true, "verify available hashes in SDK consumed modes")
    timeout := flag.Duration("timeout", 2*time.Minute, "authentication, GETs and consumption deadline")
    var stores preferences
    flag.Var(&stores, "prefer", "store preference; repeat to preserve order and duplicates")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required by this example") }
    if *mode != "file" && *mode != "writer" && *mode != "buffer" && *mode != "stream" {
        log.Fatal("-mode must be file, writer, buffer or stream")
    }
    if *mode != "buffer" && *output == "" { log.Fatal("this mode requires -output") }
    if *timeout <= 0 { log.Fatal("-timeout must be positive") }
    options := []image.ImageRecordDownloadOption{
        image.WithImageRecordDownloadOpts(image.ImageRecordDownloadOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-download-example"},
        }),
        image.WithImageRecordDownloadStream(*mode == "stream"),
        image.WithImageRecordDownloadChunkSize(*chunk),
        image.WithImageRecordDownloadChecksumVerification(*verify),
        image.WithImageRecordDownloadStorePreferences(stores...),
    }
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *id, *mode, *output, *viaService, options); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, id, mode, output string, viaService bool, options []image.ImageRecordDownloadOption) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordDownloadRequest{ID: id}
    var writer *os.File
    if mode == "file" {
        input.Filename = output
    } else if mode == "writer" {
        // Caller creates/truncates this file before the SDK metadata GET.
        writer, err = os.Create(output)
        if err != nil { return err }
        input.Output = writer
    }
    var result *image.ImageRecordDownloadResult
    var operationErr error
    if viaService {
        service, serviceErr := conn.Image(ctx)
        if serviceErr != nil { operationErr = serviceErr } else {
            result, operationErr = service.DownloadImageRecord(ctx, input, options...)
        }
    } else {
        result, operationErr = conn.DownloadImageRecord(ctx, input, options...)
    }
    if writer != nil { operationErr = errors.Join(operationErr, writer.Close()) }
    var callerBytes int64
    if result != nil && result.Downloaded != nil && result.Downloaded.Stream != nil {
        // Consumption and Close happen while the original context is alive.
        // These caller bytes do not update result.BytesWritten or Checksum.
        var consumeErr error
        callerBytes, consumeErr = saveStream(result.Downloaded.Stream, output)
        operationErr = errors.Join(operationErr, consumeErr)
    }
    return printDownload(result, operationErr, callerBytes)
}

func saveStream(stream io.ReadCloser, filename string) (count int64, err error) {
    defer func() { err = errors.Join(err, stream.Close()) }()
    output, err := os.Create(filename)
    if err != nil { return 0, err }
    defer func() { err = errors.Join(err, output.Close()) }()
    return io.Copy(output, stream)
}

func printDownload(result *image.ImageRecordDownloadResult, operationErr error, callerBytes int64) error {
    value := map[string]any{
        "partial": result != nil && operationErr != nil,
        "caller_stream_bytes_written": callerBytes,
    }
    if result != nil {
        value["bytes_written"] = result.BytesWritten
        if result.Metadata != nil {
            value["metadata"] = map[string]any{
                "body": result.Metadata.Body, "header": result.Metadata.Header,
                "status_code": result.Metadata.StatusCode,
            }
        }
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource, "wire": result.Record.Wire,
                "status_code": result.Record.StatusCode,
                "import_methods": result.Record.ImportMethods,
            }
        }
        if result.Downloaded != nil {
            value["downloaded"] = map[string]any{
                "buffered_bytes": len(result.Downloaded.Body),
                "status_code": result.Downloaded.StatusCode,
                "header": result.Downloaded.Header,
                "compatibility_header": result.Downloaded.CompatibilityHeader,
                "content_md5": result.Downloaded.ContentMD5,
            }
        }
        if result.Checksum != nil {
            value["checksum"] = map[string]any{
                "algorithm": result.Checksum.Algorithm,
                "expected": result.Checksum.Expected, "actual": result.Checksum.Actual,
                "complete": result.Checksum.Complete, "verified": result.Checksum.Verified,
            }
        }
    }
    if operationErr != nil {
        value["error"] = operationErr.Error()
        value["checksum_mismatch"] = errors.Is(operationErr, image.ErrChecksumMismatch)
        value["hash_length_required"] = errors.Is(operationErr, image.ErrImageRecordDownloadHashLengthRequired)
    }
    body, err := json.MarshalIndent(value, "", "  ")
    if err == nil { _, err = fmt.Fprintln(os.Stdout, string(body)) }
    return errors.Join(operationErr, err)
}
```

```go
var seed resource.RawResource
if err := json.Unmarshal([]byte(`{
  "id": "literal-image-id",
  "os_hash_algo": "sha256",
  "os_hash_value": "expected-hex",
  "properties": {"purpose": "download-example"}
}`), &seed); err != nil {
    return err
}
result, err := service.DownloadImageRecord(ctx, image.ImageRecordDownloadRequest{
    Resource: &seed,
    Filename: "ubuntu.qcow2",
})
```

## References

- [Image Records](image-records.md)
- [Install](../docs/install.md)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [Licensing](../docs/licensing.md)
- [`DownloadTo`](download.md)
- [Implementation Plan](../docs/implementation-plan.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
