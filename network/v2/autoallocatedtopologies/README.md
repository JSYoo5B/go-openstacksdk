# Autoallocatedtopologies

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/get_auto_allocated_topology`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithOptions`

- Literal or recorded auth project is bound once without name lookup; actual response id may differ from request project.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyGetBindsProjectSeedAndActualNetworkIdentity`.
- Default project requires recorded scoped auth and never guesses from token/endpoint.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyRecordedCurrentProjectAndMissingAuthNeverGuessRoute`.
- Source alias fallback respects explicit null, unknown fields and uncached fresh requests.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyResponseAliasNullPrecedenceAndNoCache`.
- All three topology operations retain physical Read/Close causes and native401 ErrOriginal actual Method/URL/body/header with one HTTP/Close and zero faulty retry/reauth hooks.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyAllThreeOperationsFaultsPreserve401NativeFieldsAndZeroHooks`.
- Pre-canceled contexts retain caller cause before auth lookup or HTTP.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyCancellationRetainsCauseBeforeAnyAuthLookup`.

Python/Go differences:

- Go returns detached Source Resource/Wire with actual response metadata and uses context plus owned callback guards; optional Python Adapter cache/session and object/dirty lifecycle remain tracked whole-project SDK-R1/C1/S1.
- Common Neutron/REST engine owns strict UTF8, continuation origin and request/header/version policies; no live-cloud support is claimed.
- Nil project uses recorded Keystone auth only, without an extra auth refresh; manual/unscoped tokens require explicit literal project instead of inferring it.

### `python:network/v2/delete_auto_allocated_topology`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithIgnoreMissing`

- Pinned ignore_missing default is false; only explicit true suppresses clean actual404 and retains literal receipt.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyDeleteDefaultStrict404AndExplicitIgnoreOwnActualProof`.
- All three topology operations retain physical Read/Close causes and native401 ErrOriginal actual Method/URL/body/header with one HTTP/Close and zero faulty retry/reauth hooks.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyAllThreeOperationsFaultsPreserve401NativeFieldsAndZeroHooks`.
- Pre-canceled contexts retain caller cause before auth lookup or HTTP.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyCancellationRetainsCauseBeforeAnyAuthLookup`.

Python/Go differences:

- Go returns detached Source Resource/Wire with actual response metadata and uses context plus owned callback guards; optional Python Adapter cache/session and object/dirty lifecycle remain tracked whole-project SDK-R1/C1/S1.
- Common Neutron/REST engine owns strict UTF8, continuation origin and request/header/version policies; no live-cloud support is claimed.
- Nil project uses recorded Keystone auth only, without an extra auth refresh; manual/unscoped tokens require explicit literal project instead of inferring it.
- Go owned DELETE retains the reviewed mutation success policy HTTP200..399; synthetic informational100..199 is an error. Source raise_from_response only rejects400 and above. The new SourceStatusPolicy opt-in governs record reads and bulk responses, not existing mutation acceptance.

### `python:network/v2/validate_auto_allocated_topology`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.API.ValidateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithOptions`

- Fixed fields=dry-run GET has URI project but no Resource.id seed, extra calls, client-side allocation or fabricated dry_run boolean.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyValidateFixedDryRunHasNoProjectIDSeedOrCreateCalls`.
- Headers/selected version are detached and fixed through Source callback boundaries without changing shared client.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyOptionsKeepHeadersVersionAndCapturedSource`.
- Malformed accepted response retains actual proof and passive receipt without Resource success.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyInvalidOptionsAndAcceptedDecodeFailuresAreAtomic`.
- Clean retry/renewal preserve dry-run query and captured route/version.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyCleanRetryAndReauthKeepDryRunAndDeletionMissingPolicy`.
- All three topology operations retain physical Read/Close causes and native401 ErrOriginal actual Method/URL/body/header with one HTTP/Close and zero faulty retry/reauth hooks.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyAllThreeOperationsFaultsPreserve401NativeFieldsAndZeroHooks`.
- Pre-canceled contexts retain caller cause before auth lookup or HTTP.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyCancellationRetainsCauseBeforeAnyAuthLookup`.

Python/Go differences:

- Go returns detached Source Resource/Wire with actual response metadata and uses context plus owned callback guards; optional Python Adapter cache/session and object/dirty lifecycle remain tracked whole-project SDK-R1/C1/S1.
- Common Neutron/REST engine owns strict UTF8, continuation origin and request/header/version policies; no live-cloud support is claimed.
- Nil project uses recorded Keystone auth only, without an extra auth refresh; manual/unscoped tokens require explicit literal project instead of inferring it.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
validation = conn.network.validate_auto_allocated_topology(project="project-id")
print(validation.id, validation.project_id, validation.dry_run)
topology = conn.network.get_auto_allocated_topology(project="project-id")
print(topology.id)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := autoallocatedtopologies.New(service.RawClient())
    project := autoallocatedtopologies.WithProjectID("project-id")
    validation, err := api.ValidateRecord(ctx, project)
    if err != nil { log.Fatal(err) }
    fmt.Println(string(validation.Resource.Body["id"]), string(validation.Resource.Body["dry_run"]))
    topology, err := api.GetRecord(ctx, project)
    if err != nil { log.Fatal(err) }
    fmt.Println(topology.ProjectID, string(topology.Resource.Body["id"]))
}
```

## References

- [network proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Neutron27 backend](https://github.com/openstack/neutron/blob/27.0.0/neutron/services/auto_allocate/db.py)
- [Auto Allocated Topology](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/auto_allocated_topology.py)
- [Records Test](records_test.go)
