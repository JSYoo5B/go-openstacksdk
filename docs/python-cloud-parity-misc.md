# Python Cloud Parity Misc

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_object_store/ObjectStoreCloudMixin/list_containers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListPrefix`

- ListRecords sends GET on the account root with Accept application/json, adds prefix only when WithRecordListPrefix is given, and follows the X-Account-Container-Count marker continuation with the prefix kept until an empty page.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudListContainersPrefixAndCountPaging`.

Python/Go differences:

- Python returns a list of Container resources; Go yields ContainerRecord values lazily through an iterator, so a Python list is the collected iterator.
- The deprecated full_listing argument only emits a warning in Python and has no Go counterpart.
- Python sends the Swift microversion it negotiates; Go sends the configured client microversion when one is set.

### `python:cloud/_object_store/ObjectStoreCloudMixin/search_containers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.SearchContainers`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchName`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchLocation`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchContainers`

- ListRecords sends GET on the account root with Accept application/json, adds prefix only when WithRecordListPrefix is given, and follows the X-Account-Container-Count marker continuation with the prefix kept until an empty page.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudListContainersPrefixAndCountPaging`.
- SDK collects the complete account ListRecords sequence before applying Source name/id exact-or-fnmatch identification and cloud ordered/nested mapping filters. Filters never become server prefix/query controls; duplicate rows and Source aliases remain visible.
  Tests: `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchCollectsEveryPageBeforeIdentifierAndMapping`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchNameUsesFnmatchAndDeclaredAliases`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchFalseyAndLoadedMappingFilters`.
- JMESPath returns arbitrary JSON null/scalar/object/list without inventing HTTP row association, while falsey filters, loaded float truthiness/equality, short-circuit field errors and invalid expression on empty identification follow shared SelectLoaded. Header bool getters use Source-loaded truthiness from raw SeedHeaders without changing old ListRecords policy or actual Wire, preserving null and nonempty string behavior.
  Tests: `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchExpressionsReturnArbitraryJSONWithoutRowAssociation`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchPreservesLazyMissingAndNestedMappingFailures`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchLoadedHeaderBoolViewKeepsRawWireAndNull`.
- Source is captured before callbacks; recorded/configured location is owned before HTTP and remote Wire location cannot replace it. AllRecords, selected Records, Envelope and Value are independent; late paging and guarded cancellation/source failures leave receipts without successful partial selection. Actual HTTP Content-Length faults at the second 200/203/401/404/429/503 page preserve physical body/header/status, pointer ErrOriginal/ErrReauth and zero fault retry/backoff/reauth callbacks.
  Tests: `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchLocationIsOwnedAndRemoteWireCannotReplaceIt`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchMalformedLatePagePreventsSuccessfulPartialResult`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchSourceCapturePrecedesOptionsAndKeepsCancellationCause`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchOwnsRawFiltersAndHeaderNilReset`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchPhysicalLatePageKeepsNative401AndStopsHooks`.
- Connection shares the generated Swift Cloud facade and cached service client, supplies configured location before local selection, retains explicit owned per-call location, and handles DTO reset and callbacks once. Invalid/canceled connection fails without discovery or invented response.
  Tests: `connection_swift_container_records_test.go:TestConnectionSwiftContainerSearchSharesFacadeAndAutomaticLocationBeforeSelection`, `connection_swift_container_records_test.go:TestConnectionSwiftContainerSearchPerCallLocationRemainsOwnedAndCallbackRunsOnce`, `connection_swift_container_records_test.go:TestConnectionSwiftContainerSearchPreflightDoesNotDiscoverOrFabricateResponse`.

Python/Go differences:

- Go returns ContainerSearchResult.Value plus independent correlated Records for ordinary selection and AllRecords/LastPage evidence; expressions have Expression=true and Records=nil for arbitrary JSON.
- Direct clients use recorded native token project scope; Connection supplies configured cloud/region/project names with the owned location option. Unknown facts stay null and no authentication discovery is made by the helper.
- Go exposes typed string names and raw JSON-domain filters and omits Python deprecation warnings. Mutable Resource/session/dirty/cache SDK-R1/C1/S1 and real authenticated cloud acceptance remain separately tracked.
- Go retains existing ListRecords origin/path/filter ownership, page bound and stronger cycle detection; actual pointer401 LastPage receipt is copied from native ErrOriginal without replay.

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`

- Find with resource.ID and resource.WithIgnoreMissing sends one bodyless HEAD to the container, returns name, count, bytes, custom metadata and the raw X-Container-Read header, returns nil without error on 404 and keeps other statuses such as 403 as errors.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudGetContainerHeadAndMissingNone`.

Python/Go differences:

- Python returns a Container resource or None; Go returns *ContainerResource with typed Details, Metadata keyed by the original header suffix and the cloned response Header, or nil.
- Go validates the container name before HTTP (Gophercloud CheckContainerName rejects '/' and names over 256 bytes); Python sends any name.
- Gophercloud accepts only 200 and 204 for the HEAD; Python accepts any 2xx status.
- The skip_cache argument is ignored in Python and has no Go counterpart.

### `python:cloud/_object_store/ObjectStoreCloudMixin/create_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.CreateContainer`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerPublic`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerCreateHeaders`

- The Find, CreateRecord, Find chain sends HEAD only for an existing container and HEAD, PUT, HEAD for a missing one; the PUT is bodyless and carries X-Container-Read .r:*,.rlistings only when read_ACL is passed.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudCreateContainerSequence`.
- Default public=false owns HEAD get-or-create orchestration: existing containers return the first HEAD without changing ACL; only clean404 permits a bodyless PUT, public=true fixes .r:*,.rlistings, then a second HEAD supplies the return Resource.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerCreateOwnsGetOrCreateAndPublicDefault`.
- A clean404 after successful creation returns Container nil without error and preserves both PUT and final HEAD receipts; 403/503, descriptor failure, faulty404 and retry-expanded404 remain errors without create fallback or replay.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerCreateKeepsAcknowledgementWhenSecondHeadDisappearsOrFails`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerFaultyOrExpanded404NeverEstablishesAbsence`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerDescriptorFailuresKeepActualHeadAndPreventCreate`.
- Concrete library-owned options are snapshotted before callbacks and HTTP, bulk replacement restores defaults, unknown ACL and invalid/nil source/name/context/options/header conflicts fail before requests; callback-driven route changes cannot redirect a workflow.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerOptionsOwnSnapshotsAndBulkReset`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerInvalidSourceAndInputsPrecedeCallbacks`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerOptionsCannotRerouteOrReplaceOwnedACL`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerReusableOptionsAreConcurrentAndIsolated`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerNilOptionsAndCallbackCancellationStayBeforeHTTP`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerNilHeaderCallbacksCanUseOwnedTypedHelpers`.
- Each actual initial HEAD, PUT, final HEAD, ACL POST, optional refresh HEAD and access HEAD keeps status/header/body evidence on accepted/rejected read, Close or combined faults; fault callbacks and renewal are blocked before exposure and no acknowledged mutation is replayed. Native pointer ErrUnableToReauthenticate retains ErrOriginal and ErrReauth and actual 401 receipt derives from ErrOriginal.
  Tests: `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerPhysicalFaultsKeepEveryPhaseAndBlockCallbacks`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerCancellationKeepsActualAcknowledgementWithoutReplay`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerSourceDriftIsStickyAcrossReadsAndClose`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerCleanRetryCannotChangeSelectedACLOrBodylessFraming`.
- Clean native 401 renewal remains available and sends the live renewed token without changing method, literal target, bodyless framing or selected public/empty-private ACL.
  Tests: `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerClean401RenewalKeepsMethodACLAndLiveToken`.

Python/Go differences:

- Go returns concrete workflow carriers separating actual per-phase HTTP receipts and the Source return Resource; errors remain normal Go errors with native causes instead of Python SDKException.
- Go uses bool/string options, validates literal Swift container names and headers before HTTP, snapshots maps deterministically, and rejects ambiguous header aliases and direct X-Container-Read overrides; Python accepts dynamically typed arguments and arbitrary names.
- Go retains the existing bounded Swift Resource projection policy: location is nullable and ambiguous repeated/alias response Headers become projection errors with physical proof. SDK-R1/C1/S1 mutable Resource, dirty/session/cache/location contracts remain tracked separately.
- Final HTTP 100–399 statuses follow Source raise_from_response below 400; intermediate informational and redirect processing depend on Go HTTP transport. Server authentication/Swift account ACL configuration and real cloud acceptance are not proved by fixtures.
- Source create_container returns a Container or None; Go returns ContainerCreateResult.Container plus separate Initial/Creation/Refreshed and Existing facts. Public=false sends no operation ACL, while existing ACL is never updated.

### `python:cloud/_object_store/ObjectStoreCloudMixin/delete_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.DeleteRecord`

- DeleteRecord sends one bodyless DELETE; 204 returns IgnoredMissing false (Python True), a clean 404 returns IgnoredMissing true without error (Python False) and 409 stays an HTTP error.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudDeleteContainerOutcomes`.

Python/Go differences:

- Python returns a bool; Go returns *ContainerResponse whose IgnoredMissing field is the negated Python result.
- Python converts 409 into SDKException with a 'container is not empty' message; Go returns the HTTP 409 error with its status, body and headers.
- Go validates the container name before HTTP; Python sends any name.

### `python:cloud/_object_store/ObjectStoreCloudMixin/update_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.SetMetadataAndRefresh`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataRefresh`

- SetMetadataAndRefresh with WithMetadataRefresh(false) sends one bodyless POST whose headers translate suffix keys to X-Container-Meta-*, keep prefixed keys and map system aliases such as read_ACL and write_ACL to X-Container-Read and X-Container-Write, including an empty X-Container-Read value; a 404 stays an error.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudUpdateContainerAndAccessPost`, `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudUpdateMissingContainerIsError`.

Python/Go differences:

- Python returns None; Go returns the POST acknowledgement with Refreshed nil because refresh is disabled.
- Python headers are keyword arguments; Go takes a map[string]any converted with Python string spellings, and colliding aliases fail before HTTP instead of last-assignment wins.

### `python:cloud/_object_store/ObjectStoreCloudMixin/set_container_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.SetMetadataAndRefresh`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataRefresh`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.SetContainerAccess`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerAccessRefresh`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerAccessOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerAccessHeaders`

- SetMetadataAndRefresh with WithMetadataRefresh(false) sends one bodyless POST whose headers translate suffix keys to X-Container-Meta-*, keep prefixed keys and map system aliases such as read_ACL and write_ACL to X-Container-Read and X-Container-Write, including an empty X-Container-Read value; a 404 stays an error.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudUpdateContainerAndAccessPost`, `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudUpdateMissingContainerIsError`.
- Only literal public/private values are accepted before callbacks/HTTP, public translates to .r:*,.rlistings and private to an explicitly present empty X-Container-Read value; cloud refresh defaults false and true explicitly performs POST then HEAD. A failed final HEAD, including clean404, remains strict and preserves the successful POST without a replay.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerSetAccessOwnsTranslationAndRefreshDefault`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerSetAccessRejectsUnknownBeforeCallbacksOrHTTP`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerSetAccessRefreshFailurePreservesPostAndStrictMissing`.
- POST response projection follows Source has_body=False and does not seed the requested ACL as observed state; requested policy, actual POST acknowledgement and optional observed HEAD remain separate.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerSetAccessOwnsTranslationAndRefreshDefault`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerHeadAndPostIgnoreJSONBodiesButRetainPhysicalEnvelope`.
- Concrete library-owned options are snapshotted before callbacks and HTTP, bulk replacement restores defaults, unknown ACL and invalid/nil source/name/context/options/header conflicts fail before requests; callback-driven route changes cannot redirect a workflow.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerOptionsOwnSnapshotsAndBulkReset`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerInvalidSourceAndInputsPrecedeCallbacks`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerOptionsCannotRerouteOrReplaceOwnedACL`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerReusableOptionsAreConcurrentAndIsolated`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerNilOptionsAndCallbackCancellationStayBeforeHTTP`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerNilHeaderCallbacksCanUseOwnedTypedHelpers`.
- Each actual initial HEAD, PUT, final HEAD, ACL POST, optional refresh HEAD and access HEAD keeps status/header/body evidence on accepted/rejected read, Close or combined faults; fault callbacks and renewal are blocked before exposure and no acknowledged mutation is replayed. Native pointer ErrUnableToReauthenticate retains ErrOriginal and ErrReauth and actual 401 receipt derives from ErrOriginal.
  Tests: `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerPhysicalFaultsKeepEveryPhaseAndBlockCallbacks`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerCancellationKeepsActualAcknowledgementWithoutReplay`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerSourceDriftIsStickyAcrossReadsAndClose`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerCleanRetryCannotChangeSelectedACLOrBodylessFraming`.
- Clean native 401 renewal remains available and sends the live renewed token without changing method, literal target, bodyless framing or selected public/empty-private ACL.
  Tests: `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerClean401RenewalKeepsMethodACLAndLiveToken`.

Python/Go differences:

- Go returns concrete workflow carriers separating actual per-phase HTTP receipts and the Source return Resource; errors remain normal Go errors with native causes instead of Python SDKException.
- Go uses bool/string options, validates literal Swift container names and headers before HTTP, snapshots maps deterministically, and rejects ambiguous header aliases and direct X-Container-Read overrides; Python accepts dynamically typed arguments and arbitrary names.
- Go retains the existing bounded Swift Resource projection policy: location is nullable and ambiguous repeated/alias response Headers become projection errors with physical proof. SDK-R1/C1/S1 mutable Resource, dirty/session/cache/location contracts remain tracked separately.
- Final HTTP 100–399 statuses follow Source raise_from_response below 400; intermediate informational and redirect processing depend on Go HTTP transport. Server authentication/Swift account ACL configuration and real cloud acceptance are not proved by fixtures.
- Source set_container_access returns a mutable Resource; Go exposes Requested separately from actual Acknowledgement/Refreshed/Container receipts. A POST missing read_ACL does not assert that the server now has the requested ACL.

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_container_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.GetContainerAccess`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerReadHeaders`

- Find with resource.ID and resource.WithIgnoreMissing sends one bodyless HEAD to the container, returns name, count, bytes, custom metadata and the raw X-Container-Read header, returns nil without error on 404 and keeps other statuses such as 403 as errors.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudGetContainerHeadAndMissingNone`.
- One bodyless HEAD classifies missing/empty ACL as private and exact .r:*,.rlistings as public; all other ACL strings fail with actual HEAD evidence and no value fabrication.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerGetAccessClassifiesOnlyExactObservedACL`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerHeadAndPostIgnoreJSONBodiesButRetainPhysicalEnvelope`.
- Clean404 is a container-not-found error preserving native404 and actual HEAD evidence; other statuses retain native errors, and physical/cancellation/source failures carrying404 never become absence.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerGetAccessMissingAndOtherStatusesKeepEvidence`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerPhysicalFaultsKeepEveryPhaseAndBlockCallbacks`.
- Concrete library-owned options are snapshotted before callbacks and HTTP, bulk replacement restores defaults, unknown ACL and invalid/nil source/name/context/options/header conflicts fail before requests; callback-driven route changes cannot redirect a workflow.
  Tests: `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerOptionsOwnSnapshotsAndBulkReset`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerInvalidSourceAndInputsPrecedeCallbacks`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerOptionsCannotRerouteOrReplaceOwnedACL`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerReusableOptionsAreConcurrentAndIsolated`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerNilOptionsAndCallbackCancellationStayBeforeHTTP`, `objectstorage/cloud_containers_test.go:TestSwiftCloudContainerNilHeaderCallbacksCanUseOwnedTypedHelpers`.
- Each actual initial HEAD, PUT, final HEAD, ACL POST, optional refresh HEAD and access HEAD keeps status/header/body evidence on accepted/rejected read, Close or combined faults; fault callbacks and renewal are blocked before exposure and no acknowledged mutation is replayed. Native pointer ErrUnableToReauthenticate retains ErrOriginal and ErrReauth and actual 401 receipt derives from ErrOriginal.
  Tests: `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerPhysicalFaultsKeepEveryPhaseAndBlockCallbacks`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerCancellationKeepsActualAcknowledgementWithoutReplay`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerSourceDriftIsStickyAcrossReadsAndClose`, `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerCleanRetryCannotChangeSelectedACLOrBodylessFraming`.
- Clean native 401 renewal remains available and sends the live renewed token without changing method, literal target, bodyless framing or selected public/empty-private ACL.
  Tests: `objectstorage/v1/containers/cloud_records_test.go:TestSwiftCloudContainerClean401RenewalKeepsMethodACLAndLiveToken`.

Python/Go differences:

- Go returns concrete workflow carriers separating actual per-phase HTTP receipts and the Source return Resource; errors remain normal Go errors with native causes instead of Python SDKException.
- Go uses bool/string options, validates literal Swift container names and headers before HTTP, snapshots maps deterministically, and rejects ambiguous header aliases and direct X-Container-Read overrides; Python accepts dynamically typed arguments and arbitrary names.
- Go retains the existing bounded Swift Resource projection policy: location is nullable and ambiguous repeated/alias response Headers become projection errors with physical proof. SDK-R1/C1/S1 mutable Resource, dirty/session/cache/location contracts remain tracked separately.
- Final HTTP 100–399 statuses follow Source raise_from_response below 400; intermediate informational and redirect processing depend on Go HTTP transport. Server authentication/Swift account ACL configuration and real cloud acceptance are not proved by fixtures.
- Source returns an access string and SDKException for missing or unknown ACL; Go returns ContainerAccessResult.Access plus the actual HEAD record, with ErrNotFound on clean missing and native HTTP causes retained.

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_object_capabilities`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.Service.GetInfo`

- GetInfo sends the same GET /info as the delegated Python get_info and returns the swift and slo sections.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftInfoAndSegmentSize`.

Python/Go differences:

- The helper only delegates to Proxy.get_info; the get_info differences apply.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_object_segment_size`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.Service.GetObjectSegmentSize`

- GetObjectSegmentSize sends the same GET /info as the delegated Python method and returns the same selected size or fallback value.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftInfoAndSegmentSize`.

Python/Go differences:

- The helper only delegates to Proxy.get_object_segment_size; Python returns the float 2684354561.0 on fallback where Go returns the integer 2684354561.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:cloud/_object_store/ObjectStoreCloudMixin/is_object_stale`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.IsObjectStale`

- IsObjectStale sends the same object HEAD as the delegated Python method and reaches the same stale decision.
  Tests: `objectstorage/v1/objects/python_rejudge_test.go:TestPythonRejudgeObjectStaleRequests`.

Python/Go differences:

- The helper only delegates to Proxy.is_object_stale; Go returns a nullable Stale with HEAD evidence instead of a bool.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:cloud/_object_store/ObjectStoreCloudMixin/create_object`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.CreateObject`

Unresolved boundaries:

- This helper only delegates to Proxy.create_object, which stays unresolved: Python kwargs headers, mutable Object and container precedence, segment cleanup and exception classes are not reproduced wholesale by the Go upload workflow.

### `python:cloud/_object_store/ObjectStoreCloudMixin/update_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.SetMetadataFromFresh`

- SetMetadataFromFresh sends a query-free HEAD and then one bodyless POST that resends the observed custom metadata, adds the new suffix keys, maps content_type to Content-Type and omits falsey values, with no final HEAD.
  Tests: `objectstorage/v1/objects/python_parity_test.go:TestPythonObjectCloudUpdateObjectMergesFreshMetadata`.

Python/Go differences:

- Python merges the metadata dict and keyword headers itself; the Go caller passes one map[string]any holding both.
- Python returns None; Go returns the HEAD observation and the POST acknowledgement.
- Go escapes '/' in object names as %2F while Python joins the raw name; Swift decodes both to the same object path.

### `python:cloud/_object_store/ObjectStoreCloudMixin/list_objects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.InContainer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListPrefix`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMaxPages`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListSinglePage`

- Pinned Object MRO projection owns six Body, 28 Header, updated_at aka and null SDK location; ordered canonical/wire aliases, alternate name ID, untyped RawMessage metadata, container URI from call, unknowns and subdir retained in Wire, actual page receipt cloned independently. No object HEAD or synthetic metadata observation.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsDefaultRequestAndProjection`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsOwnedOptionsNilHeaderResetAndImmutableProjection`.
- GET escaped fixed container with format=json and operation Accept JSON; Object query mapping client alias wins wire alias, unknown queries discarded before Marshaler/DTO JSON validation, declared canonical Body filters remain local, no transport-wrapper counting. CPython loaded integer/float/bool recursive equality and float truthiness are explicit shared policy opt-ins. Empty dictionary filter matches every truthy scalar/list/mapping; falsey actual is excluded and nonempty dictionary against truthy nonmapping remains a real page error, including recursive empty filters.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsQueryMappingHeadersAndLocalFilters`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsReservedControlsRejectBeforeMarshalAndUnknownDTOValuesAreDiscarded`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsLocalFiltersUseLoadedFloatsAndDeclaredHeaderTruthiness`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsEmptyDictionaryFilterMatchesEveryTruthyType`.
- Default full lazy pagination uses raw consumed count against X-Container-Object-Count, last raw row ID/name marker, truthy-limit continuation, falsey limit sent once, source body next and RFC Link interoperability, fixed origin/container/filters and SDK cycle/page bounds. Consumer changes cannot alter raw marker; raw cap includes rejected rows; empty page ends.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsCountAndMarkerContinuation`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsMaxItemsSinglePageAndPageBound`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectCloudListObjectsPrefixAndCountPaging`.
- Named cloud list_objects delegates same owned Source-shaped listing; prefix absent/explicit-empty/selected value remain distinct, full_listing is ignored by pinned Python and does not disable paging. Go iterator is collected by caller.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectCloudListObjectsPrefixAndCountPaging`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsQueryMappingHeadersAndLocalFilters`.
- Actual 100..399 admitted only after page JSON/row projection; malformed accepted singleton/scalar/constructor and every 400..599 (including 404) remain terminal with actual status/header/body and OperationError, no ignore-missing.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsStatusesAndOperationContext`.
- Mixed physical Read/Close/sticky restored source/custom cancellation faults on 200/203/401/404/429/498/503 preserve whole actual receipt, one HTTP and one Close and zero willing caller retry/reauth/backoff callbacks. Native pointer ErrUnableToReauthenticate ErrOriginal and ErrReauth are retained separately. Clean native retry/reauth/backoff preserve fixed route/headers/conditions/selected version and live token.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsPhysicalFaultsPreserveReceiptNative401AndStopHooks`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsCleanRetriesPreserveRouteHeadersAndLiveToken`.
- Owned concrete options clone raw bytes/maps/pointers and callback resets; invalid/reserved/unsupported Source controls fail before Marshaler/HTTP, nil Headers reset composes safely, repeated iterator consumption is independent, consumer break avoids continuation and custom cancellation during yield preserves page proof.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsReservedControlsRejectBeforeMarshalAndUnknownDTOValuesAreDiscarded`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsOwnedOptionsNilHeaderResetAndImmutableProjection`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsIteratorReuseConsumerBreakAndCancellation`.

Python/Go differences:

- Go accepts literal string container names, escapes one path segment and preflights empty/slash/dot/control names; pinned Python also accepts a Container resource/name conversion and interpolates its raw name, including str(None) for an empty Resource name. No mutable Python Resource object identity/cache is exposed.
- Go exposes concrete DTO/functional options and iter.Seq2; callers collect to a slice for named cloud list_objects. Pinned full_listing is ignored. Unknown queries are discarded before serialization; per-call headers/microversion/max_items have dedicated controls. Unsafe cls/self/resource_type/session and __conflicting_attrs, arbitrary base_path and deprecated JMESPath are explicitly rejected before HTTP rather than silently claiming those callable/route branches.
- Go SDK-owned location is null and each row metadata map owns raw JSON values. Pinned Connection location calculation and BaseResource class-level shared metadata/dirty state remain common SDK-R1/C1/S1 contracts; actual remote fields remain in Wire and full Envelope.
- Go max_items stops immediately at the raw cap while pinned Python may request the next page before stopping on its first row. Go follows interoperable RFC Link rel=next URLs while pinned _get_next_link indexes requests response.links next uri, a missing key in normal Requests output; Go rejects foreign/changed/cyclic continuations and defaults to 10000-page bound.
- Go sends captured or explicit selected Swift microversion without Python server-max negotiation, and sorts query keys. Requests query/str conversion retains the reviewed shared JSON-domain representation policy and its documented primitive/runtime limits; custom Python objects are outside the Go JSON-domain API.
- Fixture tests and guide builds do not prove a real Swift account/container ACL, deployment middleware/backend, auth renewal or replace-free install; actual-cloud acceptance and release gates remain pending.

### `python:cloud/_object_store/ObjectStoreCloudMixin/search_objects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.InContainer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchObjects`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.SearchObjects`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchName`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchLocation`

- SDK collects the full literal-container Objects.ListRecords sequence with format=json and account-independent actual page receipts before cloud name/id exact-or-fnmatch and ordered/nested mapping selection; filters never become server query/prefix and duplicate rows remain.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchCollectsEveryPageBeforeIdentifierAndMapping`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchNameUsesFnmatchAndDeclaredAliases`.
- Source falsey filters and loaded numeric equality, mapping short-circuit missing fields and header descriptor/Body alias getters are applied to declared Resource views; URI container and plain metadata remain separate and cannot become invented cloud-filter fields.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchFalseyAndLoadedMappingFilters`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchPreservesLazyMissingAndNestedMappingFailures`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchLoadedHeaderBoolViewKeepsRawWireAndNull`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchUsesDefaultNullAndKeepsURIAndMetadataOutsideCloudView`.
- Full JMESPath returns null/scalar/object/list as independent Value with Expression=true and no invented Records correlation; malformed expressions on empty name selection preserve errors.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchExpressionsReturnArbitraryJSONWithoutRowAssociation`.
- Direct Source Object location remains null while Connection automatically supplies configured CurrentLocation before selection; explicit overrides and whole DTO reset own all pointers/bytes and cannot accept forged remote location.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchLocationIsOwnedAndRemoteWireCannotReplaceIt`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchUsesDefaultNullAndKeepsURIAndMetadataOutsideCloudView`, `connection_swift_object_search_test.go:TestConnectionSwiftObjectSearchSharesFacadeAndAutomaticLocationBeforeSelection`, `connection_swift_object_search_test.go:TestConnectionSwiftObjectSearchPerCallLocationRemainsOwnedAndCallbackRunsOnce`.
- Late page or physical handling/source/cancellation failures retain partial AllRecords and actual LastPage without a successful Value/Records. Real HTTP Content-Length faults at 200/203/401/404/429/503 preserve native pointer401 ErrOriginal/ErrReauth and block fault hooks. Source is captured before callbacks and preflight errors perform no discovery or fabricated response.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchMalformedLatePagePreventsSuccessfulPartialResult`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchSourceCapturePrecedesOptionsAndKeepsCancellationCause`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchOwnsRawFiltersAndHeaderNilReset`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchPhysicalLatePageKeepsNative401AndStopsHooks`, `connection_swift_object_search_test.go:TestConnectionSwiftObjectSearchPreflightDoesNotDiscoverOrFabricateResponse`.

Python/Go differences:

- Go returns ObjectSearchResult.Value and correlated Records only for ordinary selection, plus independent AllRecords/LastPage receipts; expression output is arbitrary JSON and never fabricates HTTP records.
- Direct Source-style clients preserve location null; Connection injects its owned CurrentLocation snapshot. Explicit raw location bytes are validated without discovery, and forged remote location remains only in Wire.
- Go exposes typed name/container strings and raw JSON-domain filters, owns per-instance metadata instead of Source class-level shared mutable state, and omits JMESPath deprecation warnings.
- Existing object listing descriptor/query/pagination contracts and documented RFC Link repair, same-origin/container/filter ownership, page bound and stronger cycle checks remain in force. SDK-R1/C1/S1 mutable Resource/dirty/session/cache and authenticated real-cloud acceptance remain tracked separately.

### `python:cloud/_object_store/ObjectStoreCloudMixin/delete_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.DeleteObject`

- DeleteObject sends HEAD and then DELETE, adding multipart-manifest=delete for a static large object; 404 at HEAD or DELETE returns IgnoredMissing true without error (Python False) and 409 stays an HTTP error.
  Tests: `objectstorage/v1/objects/python_parity_test.go:TestPythonObjectCloudDeleteObjectOutcomes`.

Python/Go differences:

- Python returns False for every SDKException, including 409 and 5xx; Go returns those failures as errors and reports only a clean 404 through IgnoredMissing.
- The meta argument is unused in Python and has no Go counterpart.
- Go escapes '/' in object names as %2F while Python joins the raw name; Swift decodes both to the same object path.

### `python:cloud/_object_store/ObjectStoreCloudMixin/delete_autocreated_image_objects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.DeleteAutocreatedImageObjects`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithAutocreatedImageObjectsOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithAutocreatedImageTasks`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithAutocreatedImageContainer`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithAutocreatedImagePrefix`

- Task policy defaults false with no Swift requests; enabled cleanup defaults nil container to images and forwards optional prefix through lazy list records. It reads fresh metadata and deletes only exact x-sdk-autocreated=true, interleaving each object before requesting the next page.
  Tests: `objectstorage/cloud_autocreated_image_objects_test.go:TestSwiftAutocreatedImagesDefaultDisabledAndOwnedOptions`, `objectstorage/cloud_autocreated_image_objects_test.go:TestSwiftAutocreatedImagesExactMarkerLazyDeleteAndPageOrdering`.
- Fresh or retained listed SLO flag selects manifest delete with Python string truthiness; absent flag adds another HEAD and latest custom metadata is forwarded on DELETE. Clean missing deletion/discovery counts true; first metadata HEAD missing propagates. Earlier completed observations remain on later failure.
  Tests: `objectstorage/cloud_autocreated_image_objects_test.go:TestSwiftAutocreatedImagesSLOTruthinessAndSecondHead`, `objectstorage/cloud_autocreated_image_objects_test.go:TestSwiftAutocreatedImagesErrorsStopAfterCompletedPhases`.
- Owned options preserve defaults and pointer inputs; nil/context/route mutation checks fail before requests and cancellation keeps its cause. Each list/HEAD/discovery/DELETE observation is separate from requested input.
  Tests: `objectstorage/cloud_autocreated_image_objects_test.go:TestSwiftAutocreatedImagesPreflightAndCancellation`, `objectstorage/cloud_autocreated_image_objects_test.go:TestSwiftAutocreatedImagesDefaultDisabledAndOwnedOptions`.

Python/Go differences:

- Python connection image_api_use_tasks and None/default container become explicit owned concrete Go options with default false/images. Return value carries the Source Deleted bool plus partial ordered receipts rather than only bool.
- The Go profile reuses finite JSON listing records, literal string name validation and existing paging safety bounds. Metadata HEAD uses the owned helper 200..399 policy; final deletion accepts 100..399 or ignored clean404. Mutable Python Resource subclasses and arbitrary connection reconfiguration during iteration are not offered.
- Deletion forwards observed custom metadata and preserves raw acknowledgement without adopting the independent DeleteObject bulk-report validation policy. SDK live Swift/auth/task-upload release validation is not claimed.

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_object_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.HeadMetadata`

- HeadMetadata sends one bodyless, query-free HEAD and returns custom metadata keyed by the lower-cased X-Object-Meta suffix; 404 stays an error as in Python.
  Tests: `objectstorage/v1/objects/python_parity_test.go:TestPythonObjectCloudGetObjectMetadataHead`.

Python/Go differences:

- Python returns the metadata dict; Go returns GetMetadataResult whose Metadata.Values holds the same keys, together with typed system headers and the response evidence.
- Go escapes '/' in object names as %2F while Python joins the raw name; Swift decodes both to the same object path.

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_object_raw`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetObjectRaw`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRawObjectOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRawObjectQueryString`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRawObjectStream`

- Default stream=False and query_string=None map to library-owned omitted-query buffered GET; functional concrete options allow explicit encoded query and stream=True. Literal escaped container/object, empty obj selects container, raw repeated query order stays intact, no HEAD/metadata/content decoding. Complete opaque bytes include NUL/invalid UTF8, physical body closes once, accepted HTTP100..999 alone never raises status errors.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectCloudGetObjectRawReturnsEveryStatus`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawBufferedDefaultOwnsOpaqueBytesAndAllowsContainerEndpoint`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectCloudGetObjectRawRejectsInputsBeforeHTTP`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawOptionBoundaryCapturesSourceBeforeCallbacks`.
- Buffered physical Read/Close/sticky restored source/custom cancellation faults retain actual status/header/partial payload without replays, on 200/206/401/404/429/498/503; 401 pointer ErrUnableToReauthenticate public ErrOriginal and ErrReauth retained; willing retry/reauth/backoff hooks remain zero after physical faults. HTTP status clean versus mixed callback errors are classified separately.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawPhysicalFaultsPreserveActualPayloadAndNative401WithoutHooks`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawCleanNativeRenewalAndMixedCallbackErrorsStayDistinct`.
- Full named stream=True branch returns accepted body before any Read, uses shared owned bodyless unread exchange, opaque bytes flow through caller-owned guarded reader, clean EOF and Close are stable and one physical Close, valid extension HTTP777 remains raw. Original admitted policy/body/header/route cannot be broadened by callbacks.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawStreamingIsUnreadAndGuardsOpaqueLifecycle`.
- Caller stream Read/Close faults and restored source/custom cancel preserve only the current partial chunk with immutable actual status/header and all causes, despite caller mutating exported result. SDK operation wrapper retains exact EOF; no reauth/retry/backoff or resend after exposure. Early Close succeeds/idempotent and later Read is ErrClosedPipe; cancel after headers closes without a physical Read.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawStreamingFaultsRetainImmutableWireEvidenceAndNoReplay`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawStreamingCloseBeforeEOFAndCanceledUnreadBody`.
- Clean native401 reauth and429/498 backoff renew live token while fixing URL/query/Accept/conditions/selected version. Final clean native statuses are rebuilt only from actual native bytes already read/closed. Faulty rejected Close/native pointer401 preserve evidence and stop hooks before streaming body is returned.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawCleanNativeRenewalAndMixedCallbackErrorsStayDistinct`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawStreamingCleanAndFaultyNative401AndBackoff`.

Python/Go differences:

- Go returns RawObjectResponse(StatusCode,Header,Body), rather than mutable Requests Response, and configures omitted query + buffered default via concrete optional functions. The public stream branch preserves actual wire facts and close ownership rather than exposing mutable HTTP implementation state.
- Go preserves one escaped opaque object name, including slash versus literal percent-escape distinction; Python urllib.parse.quote leaves object slashes unescaped. Empty obj selects the container in both. Go validates literal container/object names and already-encoded QueryString UTF8/control/space/fragment before HTTP; Requests automatic URI requoting and custom Python objects are not reproduced.
- Go provider-owned clean native401 renewal and429/498 backoff occur before the final raw result. Native rejected bodies have already been consumed, so final clean statuses and failed native response evidence are rebuilt from their actual bytes, even in stream=True. Mixed callback/handling/source/context failures remain errors and never become clean raw status results.
- Go streamed terminal failure proof retains current partial chunk and immutable physical status/header without buffering prior successful chunks. Buffered failures may return nonnil raw result and error together. Python transport/cache/session/Resource object ownership remain common SDK-R1/C1/S1 contracts, not a full SDK parity claim.
- Fixture/guide validation does not establish live Swift ACL/backend/middleware/auth/cache or release acceptance; those project gates remain tracked.

### `python:cloud/_object_store/ObjectStoreCloudMixin/stream_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.StreamObject`
- `github.com/JSYoo5B/go-openstacksdk.Connection.StreamObjectChunks`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.StreamObjectChunks`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectStreamOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectStreamChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectStreamQueryString`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.StreamCloudObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudStreamOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudStreamChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudStreamQueryString`

- Library-owned factory returns a lazy, single-use closeable binary chunk iterator; default1024 and query_string ignored even when malformed. No discovery/options/GET before iteration; connection supplies the cached Swift Cloud facade. Receipt is actual status/header and each returned chunk is independently owned.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamLazyDefaultBinaryChunksAndSingleUse`, `objectstorage/cloud_object_stream_test.go:TestSwiftCloudObjectStreamLazyAliasIgnoredQueryAndCleanMissing`, `connection_swift_object_stream_test.go:TestConnectionSwiftChunkStreamDiscoversLazilyAndSharesCachedFacade`.
- Typed concrete ChunkSize/QueryString options and pointer snapshots require no application interface builder. Short physical reads are assembled into bounded owned chunks, nil default and explicit sizes are copied before mutation; nil/cancelled context/client/options and unsafe literal names fail without HTTP.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamChunkSizeSnapshotShortReadsAndValidation`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamNilCancelledAndOptionSourceMutationBeforeHTTP`, `objectstorage/cloud_object_stream_test.go:TestSwiftCloudObjectStreamNilCancelledAndClosedHandleStayLazy`, `connection_swift_object_stream_test.go:TestConnectionSwiftChunkStreamPreflightAndClosedHandleNeverDiscover`.
- Source-valid physical100..399 status range includes199/203/204/206 and provided304 body; native400+ errors retain actual status/header/body. Only a clean404 after successful physicalRead/Close and encoded-body validation becomes empty success, retaining receipt404; mixed404 physical/decode failures remain errors.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamSourceSuccessStatusAnd304Body`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamClean404OnlyEmptyAndOtherNativeStatuses`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamMissingReadCloseFaultsAreNotSuppressed`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamCompressedRejectionsValidateBeforeIgnoring404`, `connection_swift_object_stream_test.go:TestConnectionSwiftChunkStreamMissingReturnsActualReceipt`.
- EOF/error/break/consumerClose cleans physical body once; callbackClose prevents GET; Close during GET adopts and closes late unread body plus actual receipt. Check Err/Close for early-break cleanup failure; consumerClose prevents next Read; private receipt and unary/native error snapshots resist caller header/body mutation.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamBreakClosesOnceAndRetainsCleanupFailure`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamCloseFromOptionsAndConsumerStopsBeforeAnotherRead`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamConcurrentCloseDuringGETOwnsLateBody`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamPartialReadCloseAndImmutableCurrentChunkProof`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamNativePolicyErrorCopiesUnaryActualReceipt`, `connection_swift_object_stream_test.go:TestConnectionSwiftChunkStreamCleanupErrorsKeepPrivateProof`.
- Bodyless GET pins selected source/provider/account route/version/header facts through options, read, Close, auth/retry. Restored source plus custom cancellation/read/Close retain actual physical partial proof; native401 physical fault keeps ErrOriginal+ErrReauth and suppresses all retry/reauth/backoff hooks. Clean401 renewal and429/498 backoff reuse fixed request.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamStickyRestoredSourceAndCancellationAfterActualRead`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamNative401PhysicalFaultDoesNotInvokeHooks`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamCleanNativeRenewalAndBackoffReuseFixedRoute`.
- Requests baseline Accept:*/* and explicit Accept-Encoding:gzip, deflate, zstd fix request defaults without X-Newest. Owned caller override is snapshotted and native retry cannot change it. Real default Go transport preserves actual compressed Content-Encoding/Content-Length while bytes stream decoded, with no accidental auto-gzip double decoding.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamEncodingOverrideIsOwnedThroughCallbacksAndNativeRetry`, `objectstorage/cloud_object_stream_test.go:TestSwiftCloudObjectStreamDefaultTransportGzipAndZstandard`.
- Incremental gzip/x-gzip, zlib/rawdeflate including raw zlib-prefix fallback, zstd and reverse ordered chains up to5 match installed Requests decoder selection; unrecognized single/br remains opaque. Empty bodies yield nothing. Clean gzip truncatedtrailer/partialheader and post-complete tail normalize after actualEOF; badmagic/CRC and physical faults never become successfulEOF, including encoded404.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamRequestsCodecsAndReverseEncodingChain`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamEmptyBodiesAndMalformedCodecKeepReceipt`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamDeflateFallsBackAfterValidZlibPrefix`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamGzipCleanEOFTrailerTailHeaderAndCompressed404`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamGzipEOFNormalizationNeverHidesPhysicalFaults`.

Python/Go differences:

- This closes the named cloud fixed string container/object branch only. Python dynamically accepted mutable Object/Container, dirtyattrs, inherited metadata and arbitrary sessions are excluded typed Go semantics; Proxy.stream_object and native Objects.StreamObject review judgments are unchanged. SDK-R1/C1/S1 constructor/resource/session/cache foundations remain unsupported and supported=0.
- Go ChunkSize is typed positive int1..16777216; nil selects SDK1024. Python None/negative/bool/zero/float branches and HTTP-versus-chunk-validation precedence are deliberate Go defaults/safety differences; invalidtyped options are rejected beforeGET.
- Go assembles short Reads/framing pieces into owned bounded chunks, whereas Requests physicalchunked response can yield shorter wire-boundary segments; binary content and maximum requested chunk bound are preserved, not identical transportsegmentation.
- CloudObjectStream/Chunks/Err/Close/Receipt are SDK-owned implementations, single-use like Source generator. SDK automatic EOF/error/breakcleanup and partialbytes+error improve Go ownership. Close may stop pendingGET and capture latebody; serialiteration/Close only, blockingRead interruption uses context rather than concurrentClose as with shared guarded reader.
- Go validates safe UTF8 literalSwift names and PathEscapes complete object segment; Source utils.urljoin strips outer slashes and preserves inner object slash as path. Name/route/framing/auth/version guards and malformedtyped preflight are deliberate Go protocol policy.
- Receipt exposes actual status/header snapshot plus errorchain proof without buffering successful object payload. physical Read errors retain only that current compressed/raw Read partialbytes; decoded partial bytes are yielded separately. Rejected physical bytes are already owned by nativeGophercloud and codec validation never rereads/recloses that response.
- Decoder comparison uses CPython3.14.8, Requests2.34.2 andurllib3.2.8.0 with gzip/xgzip/deflate/zstd installed; zstd implementation is pinned klauspost/compressv1.18.0. Single optionalbr/unknown stays opaque as in this runtime, and optionalPython decoder installation/session customization is not genericResource/transport parity.
- NativeGophercloud clean401/429/498 callbacks remain provider-owned with SDK guard; ErrUnableToReauthenticate does notUnwrap its original/reauth branches. Accepted status policy is Source100..399 and does not expand existing nativeStreamObject200/206/304 semantics. Source exception rendering/KSA authentication/cache behavior remain foundationdifferences.
- Actual pinned Source execution has41 observations using real cloud/Proxy/Object/Requests/urllib3 and only memoryHTTP adapter/Connectionstub. Finalroot focused3packages passed28newgroups(leaf21/cloud3/Connection4); PythonmainAST and byteexactPython/Go guide fences passed. These are not livecloud authentication, SwiftACL or releaseacceptance. Review file is a candidate pending canonicalmerge; fullsupported count remains0.

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.DownloadObject`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetObjectText`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.GetObjectText`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetCloudObjectText`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetQueryString`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetStream`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetOutfile`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetWriter`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithoutObjectGetContent`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithoutObjectGetOutfile`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetQueryString`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetStream`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetOutfile`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetWriter`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutCloudGetContent`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutCloudGetOutfile`

- Eager closed string-name cloud get_object ignores declaredquery_string/stream, default response.text returns lowercase tupleheaders plus optionaltext/actualreceipt. Memory ignores resp_chunk_size, byte/text empty and unknown/repeated headers retain presence; clean404 Missing retains actualreceipt404 without header/body tuple.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetDefaultTextLowerHeadersEagerAndIgnoredControls`, `objectstorage/cloud_object_get_test.go:TestSwiftCloudObjectGetEagerLowerHeadersTextAndMissingReceipt`.
- Concrete owned options snapshot pointerChunkSize/Outfile and distinguish defaultNone(memory) from falseynonNone(emptypath/OmitContent headers-only): falsebranch does no acceptedbodyread/text/write/decoder/chunkvalidation and closesbody. Output reset returns memory; nil/client/context/cancel/options/unsafe names/conflictingborrowedtargets fail withoutHTTP.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetOptionsSnapshotsPreflightAndDestinationReset`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetFalseyNonNoneOutfileSkipsReadWriteTextAndChunkValidation`, `objectstorage/cloud_object_get_test.go:TestSwiftCloudObjectGetNilCancelledAndStatisticalGapStayExplicit`.
- Nonempty path opens afteracceptedGET/status/headercopy, writes opaque decodedbinary and closes itsownedhandle; clean404 never opens/truncates path. Borrowedwriter success flushes onlyifFlush supported, nevercloses; noFlush io.Writer works withoutdeveloperbuilder. Destination bytes count actualwrites and binarypath bypasses textdetector.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetPathOutputBinaryOwnedCloseAndMissingNoTruncation`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetBorrowedWriterFlushOnlyAndNoStatisticalTextRead`, `objectstorage/cloud_object_get_test.go:TestSwiftCloudObjectGetFileWriterAndFalseyOutputOptions`.
- Accepted decoderconstruction follows output/pathopen; missingdirectory wins overbadgzip or6encodingchain ctorerror, existingpath truncatesbeforectorfailure, falseypath neverdecoderinit. Source raw integerheader bad is preserved withoutnew eagervalidation; actualreadfault keeps itsphysicalproof.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetPathOpenFailurePrecedesBodyCodecAndAlwaysClosesResponse`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetDecoderConstructorsFollowPathOpenAndSkipFalseyOutfile`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetTranslatedIntegerHeaderKeepsRawValueAndReadFailuresRemainFirst`.
- Write/shortwrite/flush failures retain actualHTTP receipt, currentdecodedchunk/acceptedBytesWritten and independentcleanup causes; failedRead nevercallsdestination, writefailure skipsFlush, borrowedClose neverruns. Postwrite sourcechange/customcancel remainssticky whenphysicalClose restoresendpoint and errorchain retains read/Close/actual401original+reauth withouthooks/replay.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetWriterFailureShortWriteFlushFailurePreserveStageAndCleanup`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetAcceptedReadFaultNeverCallsDestination`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetWriterSourceCancellationIsStickyAcrossRestoringClose`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetBufferedPhysicalFaultsAnd401OriginalRemainActual`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetOptionMutationGuardsSourceBeforeDestinationOrHTTP`.
- Subset textsemantics matchRequests headerpermissivesplit/case-sensitive defaulttextLatin1/defaultJSONUTF8/lastcharsetwins, explicitASCII/Latin1/cp1252/UTF8SIG andUTF16/32 includingshortreplacement/highsurrogate+oddbyte, rawIANA ShiftJIS alias andnormalBOM. Defaultstatistical unmarkednonUTF8/NUL/malformedBOM andknownunsupportedPythoncodecs return explicitErrUnsupported with actualfullreceived-body evidence ratherthansilentdeclaredcompletion.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetRequestsHeaderCharsetDefaultsReplacementAndAliases`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetUnmarkedBOMAndExplicitUTF16UTF32`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetStatisticalCharsetGapIsExplicitWithOwnedActualEvidence`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetDetectorGapIncludesUTF8NULMalformedBOMAndKnownPythonCodecs`.
- ReuseSourceaccepted100..399 guardedrawGET, pinnedheaderdefaults/encoding policy andincrementalgzip/xgzip/zlib/rawdeflate/zstd decoderhelpers; originalcompressedHeader remainsdistinct fromdecodedtext/filebytes. Cleancompressed404 requiresdecoder success, corruptcompressed404 stayserror with actualraw nativeproof andnoMissing.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetSharedCodecsAndCompressedMissingFaults`.
- Eager Connection text/output handling shares the cached Swift facade and retained actual receipt, preserving cancellation and invalid-call failures before discovery.
  Tests: `connection_cycle7_records_test.go:TestConnectionObjectGetTextSharesCachedSwiftAndRetainsReceipt`, `connection_cycle7_records_test.go:TestConnectionCycle7PreflightPreservesCancellationBeforeDiscovery`.

Python/Go differences:

- GetObjectText is the closed cloud text API; general Proxy.get_object mutable Object/Container state and the existing native GetObject/DownloadObject reviews are unchanged.
- GoObjectGetResult separates lowercaseHeaders, optionalBody *string, actualReceipt, Missing andBytesWritten. SourceNone onclean404 becomes Missingwith actual404receipt, nofakeheaders/text; errorresults may retainreceivedheader/status/partialproof evenwhenSourcewouldraisebeforetuple.
- Go supports limited explicitheadercharset/BOM/validUTF8 policies with golang.org/x/textv0.41.0 andsimplePythonaliasnormalization; these do notreplaceRequests2.34.2 charset_normalizer3.5.2 apparent_encoding orfullCPython textcodec lookup/errorsemantics. Unresolved detector scopeincludesallunmarkedbytes, notmerelynonUTF8; knownNUL/malformedBOM/unicode_escape/cp1006/palmos negatives retainUnsupported+actualbytes.
- Sourceget_object outfileNone/falseynonNone truthiness is represented by typednilcontrols, explicitemptyOutfile/OmitContent, concretepathorborrowedio.Writer. Pythonarbitrary filelikebool/callbacks/Resources are nottransferredtoGo. QueryString/Streamignored formalcontrols staylibraryowned, noapplicationinterface builderrequired.
- NonemptyGooutput chunk is typedpositive1..16MiB validated beforeHTTP/fileopen, whileSource mayopenpaththenvalidate dynamicchunk type; memory/discardignoreChunkSize. Go detects io.ErrShortWrite whereSource ignoreswrite returncount, optionalFlush supportsstandardunbuffered Go writers, andownedpath/HTTPbody cleanup occurs onall errors instead ofSourcefinally absence.
- Accepted outputorder is rawGET/status→lowerheadercopy→pathopen→decoderinit→reads/writes→Flush/Close. SourceObject rawheadercopy is retained withoutdemanding eagerdescriptorcasts. CallerPathError concreteerrors.As type remainsavailable; no reusedcachedstream-error snapshot canerase it.
- Names followSDK safeUTF8 literalSwiftvalidation andwholeobject PathEscape, source/provider/accountendpoint/version/header facts are capturedbeforeoptions andremainsticky throughnativecallbacks/Read/Write/Flush/Close. Late writer/reader resource.ErrNotFound is aphasefailure with actualreceivedproof, not cleanHTTP404 suppression.
- Decoder behavior andnative clean401/429/498 policyreuse existingownedSwift streamhelpers, accepted100..399 doesnotexpandnativeStreamObject200/206/304. Error bodyproof maybeactual currentcompressedphysicalbytes orcurrentdecodedwriterchunk asstageappropriate; noentireobjectbuffer iskeptforoutputmodes.
- ActualpinnedSource cloud/Proxy/Object/Requests/urllib3 execution has65observations withmemoryHTTPadapter/eagerRequests-like buffering andonlypathopen recordingstub. Sourcefield/runtimefiles arehash-recorded; oracle/mains/fixtures donotprove liveauth/filesystemACL/release. Originalpins areGophercloudv2.15.0 andopenstacksdkef55d7d1666099f50bf1e1c40b59d7e7b72a51fe; fullsupported count remains0.
- Text is decoded only from UTF-8, a byte-order mark or a supported declared charset (UTF-16/UTF-32 with a BOM or explicit endianness, ISO-8859-1); undeclared non-UTF-8 bytes and other codecs return ErrUnsupported with the original bytes instead of Requests' statistical detection or the full CPython codec registry; see Deliberate runtime differences in docs/sdk-support-ledger.md.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/current_location`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CurrentLocation`

- CurrentLocation returns cloud, region_name, null zone and the project id from the recorded token scope with project name and domain from configuration, encoded with the Python key names, without any request beyond the initial authentication.
  Tests: `python_parity_test.go:TestPythonConnectionCurrentLocationProjectAndName`.

Python/Go differences:

- Python returns a Munch; Go returns resource.CloudLocation with nullable pointer and raw JSON fields.
- For a connection built from OS_* environment variables Python names the cloud envvars (or OS_CLOUD_NAME), while Go uses the identity endpoint host; FromProvider leaves the cloud null unless WithCloudLocation supplies it.
- Python reports region_name from configuration (an empty string when unset); Go leaves it null when no region is configured.
- Python reads the project id through session.get_project_id(), which can authenticate; Go reads the scope recorded at Connect without I/O.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/current_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CurrentLocation`

- CurrentLocation returns cloud, region_name, null zone and the project id from the recorded token scope with project name and domain from configuration, encoded with the Python key names, without any request beyond the initial authentication.
  Tests: `python_parity_test.go:TestPythonConnectionCurrentLocationProjectAndName`.

Python/Go differences:

- Python returns a Munch with id, name, domain_id and domain_name; Go exposes the same four fields as CurrentLocation().Project.
- Python reads the project id through session.get_project_id(), which can authenticate; Go reads the scope recorded at Connect without I/O.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/current_project_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CurrentLocation`

- CurrentLocation returns cloud, region_name, null zone and the project id from the recorded token scope with project name and domain from configuration, encoded with the Python key names, without any request beyond the initial authentication.
  Tests: `python_parity_test.go:TestPythonConnectionCurrentLocationProjectAndName`.
- An unscoped token yields a nil CurrentLocation().Project.ID, matching Python None.
  Tests: `python_parity_test.go:TestPythonConnectionCurrentProjectIDUnscopedIsNull`.

Python/Go differences:

- Python returns a string or None; Go returns CurrentLocation().Project.ID as raw JSON or nil.
- Python reads the project id through session.get_project_id(), which can authenticate; Go reads the scope recorded at Connect without I/O.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/get_name`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CurrentLocation`

- CurrentLocation returns cloud, region_name, null zone and the project id from the recorded token scope with project name and domain from configuration, encoded with the Python key names, without any request beyond the initial authentication.
  Tests: `python_parity_test.go:TestPythonConnectionCurrentLocationProjectAndName`.

Python/Go differences:

- Python returns the configured cloud name string; Go returns CurrentLocation().Cloud as a nullable pointer.
- For a connection built from OS_* environment variables Python names the cloud envvars (or OS_CLOUD_NAME), while Go uses the identity endpoint host; FromProvider leaves the cloud null unless WithCloudLocation supplies it.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/auth_token`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.AuthToken`

- Returns the token obtained at Connect without further HTTP requests, and an empty string when none is held.
  Tests: `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionRecordsTokenUserAndCatalogWithoutRequests`, `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionWithoutAuthResultReportsIt`.

Python/Go differences:

- Go reads the token held by the shared provider and never renews a still-held expired token; keystoneauth get_token re-validates and renews. Rejected requests trigger the native reauthentication instead.
- An absent token is an empty string instead of None.

### `python:connection/authorize`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.Authorize`

- Returns the held token without a request; with no token it reauthenticates once through the provider function, and without one fails with an invalid-option error.
  Tests: `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionRecordsTokenUserAndCatalogWithoutRequests`, `api/py_core_misc_session_test.go:TestPyCoreMiscAuthorizeReauthenticatesOnlyWhenTokenMissing`, `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionWithoutAuthResultReportsIt`.
- A canceled context fails before any call and keeps context.Canceled observable.
  Tests: `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionWithoutAuthResultReportsIt`.

Python/Go differences:

- Go reads the token held by the shared provider and never renews a still-held expired token; keystoneauth get_token re-validates and renews. Rejected requests trigger the native reauthentication instead.
- Failures wrap the native cause rather than SDKException.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/current_user_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CurrentUserID`

- Returns the user id recorded in the Keystone token response without a request; a connection without a recorded response returns ErrNoAuthResult.
  Tests: `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionRecordsTokenUserAndCatalogWithoutRequests`, `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionWithoutAuthResultReportsIt`.

Python/Go differences:

- Python re-fetches access info through the session and raises ConfigException for non-keystone auth; Go reads the recorded response and returns ErrNoAuthResult.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/service_catalog`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ServiceCatalog`

- Returns a copy of the recorded catalog (id, name, type and endpoint id, interface, region, region id, url) without a request; mutation of the copy does not alter the recorded response.
  Tests: `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionRecordsTokenUserAndCatalogWithoutRequests`.
- A connection without a recorded token response returns ErrNoAuthResult.
  Tests: `api/py_core_misc_session_test.go:TestPyCoreMiscConnectionWithoutAuthResultReportsIt`.

Python/Go differences:

- Entries are typed values instead of raw dictionaries and unknown endpoint fields are dropped; Python reads the live access object and may refresh it.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/session`

Status: `unsupported`.

Unresolved boundaries:

- Exposes the keystoneauth Session object; Go shares a Gophercloud ProviderClient and deliberately does not expose a session abstraction.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/close`

Status: `unsupported`.

Unresolved boundaries:

- Releases the requests session and thread pool Python creates; the Go Connection owns no such resources because the caller owns the http.Client.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/set_global_request_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GlobalRequest`

- The request ID is applied to every request of the derived Connection and never to the receiver.
  Tests: `connection_r2_misc_derive_test.go:TestR2MiscGlobalRequestHeaderOnDerivedConnectionOnly`.
- Invalid IDs are rejected.
  Tests: `connection_r2_misc_derive_test.go:TestR2MiscGlobalRequestRejectsUnsafeIDs`.

Python/Go differences:

- Deliberate design choice: Python mutates the existing Connection so later requests change; Go Connections are immutable after construction because requests run concurrently, so the ID is set only by deriving a Connection with GlobalRequest.
- The per-request global_request_id argument of Proxy.request has no counterpart.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/global_request`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GlobalRequest`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CustomServiceClient`

- The derived Connection shares the provider token and sends the request ID header on compute and network requests, while the parent and already obtained services send none; a derived Connection can be re-derived.
  Tests: `connection_r2_misc_derive_test.go:TestR2MiscGlobalRequestHeaderOnDerivedConnectionOnly`.
- Empty, spaced, control, non-ASCII and nil-receiver inputs are rejected before anything is built.
  Tests: `connection_r2_misc_derive_test.go:TestR2MiscGlobalRequestRejectsUnsafeIDs`.
- Custom service clients of a derived Connection carry the header with their own client.
  Tests: `connection_r2_misc_endpoint_test.go:TestR2MiscAddServiceBuildsLazyClientOncePerConnection`.

Python/Go differences:

- Python mutates nothing on the receiver either, but also copies configuration through cloud_region.from_session; Go copies the Connection options, shares the provider and starts with a fresh service cache and microversion negotiation.
- The ID is validated (non-empty printable ASCII without spaces); Python accepts any string.
- Python can also use the context-manager form; Go callers keep the returned Connection in a variable scope.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/pprint`

Status: `unsupported`.

Unresolved boundaries:

- Python-only wrapper around the standard pprint module; Go callers use fmt or encoding/json.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/pformat`

Status: `unsupported`.

Unresolved boundaries:

- Python-only wrapper around pprint.pformat; Go callers use fmt or encoding/json.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/range_search`

Status: `unsupported`.

Unresolved boundaries:

- Operates on untyped dictionaries with string range expressions; the Go API exposes typed collections and concrete filter options, so no equivalent entry point exists by design.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/search_resources`

Status: `unsupported`.

Unresolved boundaries:

- Resolves a service.resource string through the dynamic Python proxy registry; the Go API is statically typed and has no string-keyed resource registry by design.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/project_cleanup`

Status: `unresolved`.

Unresolved boundaries:

- Go has no project cleanup workflow with service dependency ordering, dry run, filters, evaluation callback and status queue.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/has_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.HasService`
- `github.com/JSYoo5B/go-openstacksdk.Connection.HasServiceVersion`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ServiceEnabled`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SessionEndpoint`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SessionEndpointContext`
- `github.com/JSYoo5B/go-openstacksdk.WithServiceEnabled`

- HasService reports catalog or override presence without a request and returns errors for lookup failures.
  Tests: `connection_r2_misc_endpoint_test.go:TestR2MiscEndpointForCatalogSelectionAndOverride`, `connection_r2_misc_endpoint_test.go:TestR2MiscEndpointForInvalidAndUnrecordedCatalog`.
- HasServiceVersion asks SessionEndpointContext for min_version=version and max_version='<major>.latest', so a catalog URL of that major needs no request, other majors consult the version document, and an absent version is false.
  Tests: `connection_endpoint_discovery_test.go:TestHasServiceVersionAndServiceFlags`, `connection_endpoint_discovery_test.go:TestSessionEndpointContextProjectPeekWithoutTokenProject`.
- has_<service> flags from clouds.yaml or WithServiceEnabled (explicit options win) disable a service before any lookup; unset flags default to official os-service-types types, so aliases such as volumev3 are false unless enabled; non-boolean flags are rejected.
  Tests: `connection_endpoint_discovery_test.go:TestHasServiceVersionAndServiceFlags`, `connection_endpoint_discovery_test.go:TestCloudsYAMLServiceFlagsAndSecurityGroupSource`, `connection_r2_misc_endpoint_test.go:TestR2MiscEndpointForInvalidAndUnrecordedCatalog`.

Python/Go differences:

- Go returns lookup failures and invalid versions as errors where Python returns False.
- HasService performs no version discovery; HasServiceVersion with an empty version is has_service without a version (including the implied version of volumev2/volumev3/sharev2/workflowv2).
- Catalog types match exactly; os-service-types aliases are not resolved during catalog lookup, as for EndpointFor.
- has_<service> values must be booleans and are validated when the Connection is created instead of asserting at call time; region-specific clouds.yaml values are not read.
- Version discovery is not cached between calls and the debug log of disabled services is not emitted.

### `python:cloud/openstackcloud/_OpenStackCloudMixin/get_session_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.SessionEndpoint`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SessionEndpointContext`
- `github.com/JSYoo5B/go-openstacksdk.Connection.EndpointFor`
- `github.com/JSYoo5B/go-openstacksdk.WithEndpointMinVersion`
- `github.com/JSYoo5B/go-openstacksdk.WithEndpointMaxVersion`

- Override or catalog endpoint with region, interface and service name; false when absent.
  Tests: `connection_r2_misc_endpoint_test.go:TestR2MiscEndpointForCatalogSelectionAndOverride`.
- SessionEndpointContext follows keystoneauth discovery: an override or a catalog URL already within the bounds and major is returned without a request; otherwise unauthenticated GETs (a 401 retried with the token) of the unversioned URL, matched versioned URL, identity hack and catalog URL select the newest CURRENT/SUPPORTED/STABLE/DEPRECATED version in range by its self link (relative and foreign-host links repaired, project suffix restored); no match is not found and no document falls back to the catalog URL.
  Tests: `connection_endpoint_discovery_test.go:TestSessionEndpointContextVersionDiscovery`, `connection_endpoint_discovery_test.go:TestSessionEndpointContextProjectPeekWithoutTokenProject`.
- Invalid, contradictory or alias-mismatched versions are errors before HTTP, overrides skip discovery, version bounds are rejected by the context-free SessionEndpoint, and cancellation stops discovery.
  Tests: `connection_endpoint_discovery_test.go:TestSessionEndpointContextOverrideInvalidAndCanceled`.

Python/Go differences:

- Go returns (url, found, error); invalid versions, malformed version documents and cancellation are errors where Python raises SDKException.
- Version discovery runs only in SessionEndpointContext; SessionEndpoint never makes a request.
- Catalog types match exactly; os-service-types aliases are not resolved during catalog lookup, as for EndpointFor.
- Discovery documents are not cached between calls; the discovery GET uses the provider HTTP client and the current token without reauthentication.

### `python:connection/endpoint_for`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.EndpointFor`
- `github.com/JSYoo5B/go-openstacksdk.WithEndpointInterface`
- `github.com/JSYoo5B/go-openstacksdk.WithEndpointRegion`
- `github.com/JSYoo5B/go-openstacksdk.WithEndpointServiceName`

- A WithEndpoint override of the type wins; otherwise the first matching catalog endpoint by interface (URL suffix ignored) and region is returned, the bool is false when absent, and no request is made.
  Tests: `connection_r2_misc_endpoint_test.go:TestR2MiscEndpointForCatalogSelectionAndOverride`.
- Without a recorded token catalog the provider endpoint locator is used with the type, region, interface and name, absence maps to not found, and invalid arguments or other failures are errors.
  Tests: `connection_r2_misc_endpoint_test.go:TestR2MiscEndpointForInvalidAndUnrecordedCatalog`.

Python/Go differences:

- Go returns (url, found, error) instead of None; lookup failures are errors.
- Catalog types match exactly; os-service-types aliases are not resolved.
- Overrides are the Connection WithEndpoint options per Service; per-service endpoint/endpoint_override configuration keys are not read.

### `python:connection/connect_as`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ConnectAs`
- `github.com/JSYoo5B/go-openstacksdk.AsUsername`
- `github.com/JSYoo5B/go-openstacksdk.AsUserID`
- `github.com/JSYoo5B/go-openstacksdk.AsPassword`
- `github.com/JSYoo5B/go-openstacksdk.AsUserDomainID`
- `github.com/JSYoo5B/go-openstacksdk.AsUserDomainName`
- `github.com/JSYoo5B/go-openstacksdk.AsProjectName`
- `github.com/JSYoo5B/go-openstacksdk.AsProjectID`
- `github.com/JSYoo5B/go-openstacksdk.AsProjectDomainID`
- `github.com/JSYoo5B/go-openstacksdk.AsProjectDomainName`
- `github.com/JSYoo5B/go-openstacksdk.AsAuthOptions`

- The derived Connection authenticates once with the recorded options overridden by typed settings, applies the user/project discard rules and project domain inheritance, keeps endpoint/region/location facts, and uses its own token.
  Tests: `connection_r2_misc_derive_test.go:TestR2MiscConnectAsOverridesAuthAndSharesNothing`.
- Invalid overrides, nil options, canceled contexts and FromProvider connections make no token request; authentication failures wrap the native cause; a derived Connection does not inherit the global request ID.
  Tests: `connection_r2_misc_derive_test.go:TestR2MiscConnectAsInvalidInputsMakeNoRequest`, `connection_r2_misc_derive_test.go:TestR2MiscConnectAsDoesNotInheritGlobalRequestID`.

Python/Go differences:

- Overrides are typed options instead of **kwargs; AsAuthOptions covers other auth settings.
- No discovery cache is shared because Go keeps none; microversion selection is negotiated again.
- With a user ID the inherited user domain is dropped because Identity v3 rejects both.
- Connections created by FromProvider record no credentials and return ErrNoRecordedAuth.
- Failures wrap the native cause rather than SDKException; context cancellation stays observable and nothing is replayed automatically.

### `python:connection/connect_as_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ConnectAsProject`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ConnectAs`

- A project by name and project domain ID is scoped with the inherited project fields discarded; a project ID selects the ID scope.
  Tests: `connection_r2_misc_derive_test.go:TestR2MiscConnectAsOverridesAuthAndSharesNothing`.
- An empty project reference is rejected without a request.
  Tests: `connection_r2_misc_derive_test.go:TestR2MiscConnectAsInvalidInputsMakeNoRequest`.

Python/Go differences:

- The dict form is ProjectRef{ID, Name, DomainID}; a plain string is ProjectRef{Name: name}.
- Same ownership as ConnectAs.
- Failures wrap the native cause rather than SDKException; context cancellation stays observable and nothing is replayed automatically.

### `python:connection/add_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.AddService`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CustomServiceClient`

- Registration performs no request; the client is built lazily once from the shared provider, endpoint override or locator, and cached; duplicates, built-in names and invalid input are rejected; canceled contexts fail.
  Tests: `connection_r2_misc_endpoint_test.go:TestR2MiscAddServiceBuildsLazyClientOncePerConnection`.
- Factory errors and nil clients are reported and a locator is required without an endpoint.
  Tests: `connection_r2_misc_endpoint_test.go:TestR2MiscAddServiceWithoutEndpointUsesLocatorAndReportsFactoryErrors`.

Python/Go differences:

- Python attaches a ServiceDescription proxy to the Connection class (process-wide) and enables the type; Go registers a Gophercloud client factory on one Connection and returns a ServiceClient, not a typed proxy.
- Registrations cannot be replaced.

### `python:cloud/_utils/FileSegment/read`

Status: `unsupported`.

Unresolved boundaries:

- FileSegment is an internal Python file-like upload helper with no public Go counterpart by design; Go callers can use io.SectionReader and segment uploads use internal readers.

### `python:cloud/_utils/FileSegment/reset`

Status: `unsupported`.

Unresolved boundaries:

- FileSegment is an internal Python file-like upload helper with no public Go counterpart by design; Go callers can use io.SectionReader and segment uploads use internal readers.

### `python:cloud/_utils/FileSegment/seek`

Status: `unsupported`.

Unresolved boundaries:

- FileSegment is an internal Python file-like upload helper with no public Go counterpart by design; Go callers can use io.SectionReader and segment uploads use internal readers.

### `python:cloud/_utils/FileSegment/tell`

Status: `unsupported`.

Unresolved boundaries:

- FileSegment is an internal Python file-like upload helper with no public Go counterpart by design; Go callers can use io.SectionReader and segment uploads use internal readers.

### `python:cloud/inventory/OpenStackInventory/list_hosts`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Inventory.ListHosts`
- `github.com/JSYoo5B/go-openstacksdk.Inventory.SearchHosts`
- `github.com/JSYoo5B/go-openstacksdk.Inventory.GetHost`
- `github.com/JSYoo5B/go-openstacksdk.NewInventory`
- `github.com/JSYoo5B/go-openstacksdk.NewInventoryFromCloudFiles`
- `github.com/JSYoo5B/go-openstacksdk.CloudNames`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryAllProjects`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryContinueOnCloudError`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryFilters`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryExpand`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryBare`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryPrivate`
- `github.com/JSYoo5B/go-openstacksdk.InventoryHostView`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListCloudRecords`

- Servers of every cloud are listed in order with all_tenants on request; a failing cloud contributes nothing; the default aborts and keeps earlier hosts; continue mode records failures; cancellation aborts without requests.
  Tests: `connection_r2_misc_inventory_test.go:TestR2MiscInventoryListsEveryCloudInOrderWithAllProjectsQuery`, `connection_r2_misc_inventory_test.go:TestR2MiscInventoryFailurePolicyKeepsEarlierHostsAndNeverRetries`, `connection_r2_misc_inventory_test.go:TestR2MiscInventoryCancellationAlwaysAborts`.
- Clouds are enumerated from the first clouds.yaml in file order and connected lazily.
  Tests: `connection_r2_misc_inventory_test.go:TestR2MiscInventoryValidationAndCloudNames`.
- ListHosts expands every host into hostvars by default, keeps only the interface expansion with WithInventoryExpand(false), skips expansion with WithInventoryBare, maps private to the private interface address, and fails a cloud whose expansion lookup fails while recording swallowed security group failures.
  Tests: `connection_cloud_inventory_hostvars_test.go:TestInventoryHostsDefaultToHostvarsExpansion`, `connection_cloud_inventory_hostvars_test.go:TestInventoryExpandFalseBareAndPrivate`, `connection_cloud_inventory_hostvars_test.go:TestInventoryHostvarsSwallowSecurityGroupFailureButFailCloudOnLookupError`.
- Hosts are listed through the cloud list_servers path (servers.API.ListCloudRecords): GET /servers/detail with the Python Accept header, all_tenants=True only for all projects, continuation links followed, and each InventoryHost keeps the Source Record row.
  Tests: `connection_cloud_inventory_proxy_listing_test.go:TestInventoryListsThroughCloudListServersProxyPath`, `connection_r2_misc_inventory_test.go:TestR2MiscInventoryListsEveryCloudInOrderWithAllProjectsQuery`.

Python/Go differences:

- Go returns typed compute servers with explicit failures instead of swallowing SDKException.
- Python's use_direct_get is accepted but unused, and refresh only invalidates the Python cache; Go has no cache layer, so neither has a counterpart. The private flag is a per-call WithInventoryPrivate option rather than a constructor argument and maps to compute.WithPrivateCloud(true).
- WithInventoryBare is a Go extension that skips every expansion. An expansion failure fails that cloud under the same failure policy as a listing failure; the security group and volume steps record their failures in the hostvars as get_hostvars_from_server swallows them.
- The default expansion reuses ExpandServerInterfaces; its own get_server_public_ip review stays unresolved only for the separate cloud Create/Get/Wait lifecycle, while the expansion this operation returns is covered by the HTTP fixtures listed here.
- Rows come from the python:compute/v2/servers listing and inherit its recorded differences (immutable snapshots, integer-only limit and max_items).

### `python:cloud/inventory/OpenStackInventory/search_hosts`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Inventory.ListHosts`
- `github.com/JSYoo5B/go-openstacksdk.Inventory.SearchHosts`
- `github.com/JSYoo5B/go-openstacksdk.Inventory.GetHost`
- `github.com/JSYoo5B/go-openstacksdk.NewInventory`
- `github.com/JSYoo5B/go-openstacksdk.NewInventoryFromCloudFiles`
- `github.com/JSYoo5B/go-openstacksdk.CloudNames`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryAllProjects`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryContinueOnCloudError`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryFilters`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryExpand`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryBare`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryPrivate`
- `github.com/JSYoo5B/go-openstacksdk.InventoryHostView`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListCloudRecords`

- Glob identifier selection, nested filters and JMESPath string filters over server records.
  Tests: `connection_r2_misc_inventory_test.go:TestR2MiscInventorySearchAndGetHost`.
- SearchHosts filters InventoryHostView rows, so filters see hostvars keys such as public_v4, interface_ip and flavor.name, and the private flag changes interface_ip and access_ipv4.
  Tests: `connection_cloud_inventory_hostvars_test.go:TestInventoryHostsDefaultToHostvarsExpansion`, `connection_cloud_inventory_hostvars_test.go:TestInventoryExpandFalseBareAndPrivate`.
- Hosts are listed through the cloud list_servers path (servers.API.ListCloudRecords): GET /servers/detail with the Python Accept header, all_tenants=True only for all projects, continuation links followed, and each InventoryHost keeps the Source Record row.
  Tests: `connection_cloud_inventory_proxy_listing_test.go:TestInventoryListsThroughCloudListServersProxyPath`, `connection_r2_misc_inventory_test.go:TestR2MiscInventoryListsEveryCloudInOrderWithAllProjectsQuery`.

Python/Go differences:

- Filter keys are the JSON field names of the Go compute server record.
- Python's use_direct_get is accepted but unused, and refresh only invalidates the Python cache; Go has no cache layer, so neither has a counterpart. The private flag is a per-call WithInventoryPrivate option rather than a constructor argument and maps to compute.WithPrivateCloud(true).
- WithInventoryBare is a Go extension that skips every expansion. An expansion failure fails that cloud under the same failure policy as a listing failure; the security group and volume steps record their failures in the hostvars as get_hostvars_from_server swallows them.
- The default expansion reuses ExpandServerInterfaces; its own get_server_public_ip review stays unresolved only for the separate cloud Create/Get/Wait lifecycle, while the expansion this operation returns is covered by the HTTP fixtures listed here.
- Rows come from the python:compute/v2/servers listing and inherit its recorded differences (immutable snapshots, integer-only limit and max_items).

### `python:cloud/inventory/OpenStackInventory/get_host`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Inventory.ListHosts`
- `github.com/JSYoo5B/go-openstacksdk.Inventory.SearchHosts`
- `github.com/JSYoo5B/go-openstacksdk.Inventory.GetHost`
- `github.com/JSYoo5B/go-openstacksdk.NewInventory`
- `github.com/JSYoo5B/go-openstacksdk.NewInventoryFromCloudFiles`
- `github.com/JSYoo5B/go-openstacksdk.CloudNames`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryAllProjects`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryContinueOnCloudError`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryFilters`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryExpand`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryBare`
- `github.com/JSYoo5B/go-openstacksdk.WithInventoryPrivate`
- `github.com/JSYoo5B/go-openstacksdk.InventoryHostView`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListCloudRecords`

- Single match, nil for none, AmbiguousError for several, invalid expression results.
  Tests: `connection_r2_misc_inventory_test.go:TestR2MiscInventorySearchAndGetHost`.
- GetHost selects one expanded host by identifier and hostvars filters and rejects a JMESPath value.
  Tests: `connection_cloud_inventory_hostvars_test.go:TestInventoryHostsDefaultToHostvarsExpansion`.
- Hosts are listed through the cloud list_servers path (servers.API.ListCloudRecords): GET /servers/detail with the Python Accept header, all_tenants=True only for all projects, continuation links followed, and each InventoryHost keeps the Source Record row.
  Tests: `connection_cloud_inventory_proxy_listing_test.go:TestInventoryListsThroughCloudListServersProxyPath`, `connection_r2_misc_inventory_test.go:TestR2MiscInventoryListsEveryCloudInOrderWithAllProjectsQuery`.

Python/Go differences:

- Go returns resource.AmbiguousError instead of SDKException.
- Python's use_direct_get is accepted but unused, and refresh only invalidates the Python cache; Go has no cache layer, so neither has a counterpart. The private flag is a per-call WithInventoryPrivate option rather than a constructor argument and maps to compute.WithPrivateCloud(true).
- WithInventoryBare is a Go extension that skips every expansion. An expansion failure fails that cloud under the same failure policy as a listing failure; the security group and volume steps record their failures in the hostvars as get_hostvars_from_server swallows them.
- The default expansion reuses ExpandServerInterfaces; its own get_server_public_ip review stays unresolved only for the separate cloud Create/Get/Wait lifecycle, while the expansion this operation returns is covered by the HTTP fixtures listed here.
- Rows come from the python:compute/v2/servers listing and inherit its recorded differences (immutable snapshots, integer-only limit and max_items).

## References

- [openstack/cloud/_object_store.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py)
- [openstack/cloud/openstackcloud.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/openstackcloud.py)
- [openstack/cloud/_utils.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)
- [openstack/cloud/inventory.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/inventory.py)
- [openstack/connection.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/connection.py)
- [Python Record List](../objectstorage/v1/objects/python-record-list.md)
- [Python Raw Get](../objectstorage/v1/objects/python-raw-get.md)
- [Cloud Object Get Python](../objectstorage/cloud-object-get-python.md)
- [Cloud Object Stream Python](../objectstorage/cloud-object-stream-python.md)
- [Cloud Containers Python](../objectstorage/cloud-containers-python.md)
- [Cloud Container Search Python](../objectstorage/cloud-container-search-python.md)
- [Cloud Object Search Python](../objectstorage/cloud-object-search-python.md)
- [Cloud Object Search Test](../objectstorage/cloud_object_search_test.go)
- [Connection Swift Object Search Test](../connection_swift_object_search_test.go)
- [objectstorage/v1/containers/python_parity_test.go](../objectstorage/v1/containers/python_parity_test.go)
- [objectstorage/v1/objects/python_parity_test.go](../objectstorage/v1/objects/python_parity_test.go)
- [python_parity_test.go](../python_parity_test.go)
