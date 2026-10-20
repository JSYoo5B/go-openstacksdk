# Cloud Containers Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
container = conn.create_container("photos", public=True)
if container is not None:
    print(container.name, container.read_ACL)

updated = conn.set_container_access("photos", "private")
observed = conn.set_container_access("photos", "public", refresh=True)
print(conn.get_container_access("photos"))
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorage(ctx)
    if err != nil { log.Fatal(err) }

    created, err := swift.Cloud.CreateContainer(ctx, "photos",
        objectstorage.WithContainerPublic(true))
    if err != nil { log.Fatal(err) }
    if created.Container != nil {
        fmt.Printf("existing=%t name=%s ACL=%s\n", created.Existing,
            created.Container.Resource.Body["name"],
            created.Container.Resource.Body["read_ACL"])
    }

    updated, err := swift.Cloud.SetContainerAccess(ctx, "photos", "private")
    if err != nil { log.Fatal(err) }
    fmt.Printf("POST=%d requested=%s\n", updated.Acknowledgement.StatusCode, updated.Requested)

    observed, err := swift.Cloud.SetContainerAccess(ctx, "photos", "public",
        objectstorage.WithContainerAccessRefresh(true))
    if err != nil { log.Fatal(err) }
    fmt.Printf("POST=%d HEAD=%d observed ACL=%s\n",
        observed.Acknowledgement.StatusCode, observed.Refreshed.StatusCode,
        observed.Container.Resource.Body["read_ACL"])

    access, err := swift.Cloud.GetContainerAccess(ctx, "photos")
    if err != nil { log.Fatal(err) }
    fmt.Printf("access=%s HEAD=%d\n", access.Access, access.Container.StatusCode)
}
```

## References

- [Cloud Containers Test](cloud_containers_test.go)
- [Cloud Records Test](v1/containers/cloud_records_test.go)
