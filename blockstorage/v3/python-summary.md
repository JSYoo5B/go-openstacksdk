# Python Summary

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/summary`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SummaryRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummarySingleFetch`

- Source Proxy summary preserves the initial queryless requires_id=False GET followed by all_tenants=False/True aggregate fetch; stock /volumes/summary is separate from /volumes ListSummary. First fetch only declared untyped seed survives omission; explicit null overwrites. Explicit single fetch improves redundant Source request sequence.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryDefaultTwoFetchesAndAllProjects`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryRetainsFirstFetchDeclaredSeedOnly`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummarySingleFetchAndFetchSpecificVersion`.
- BlockStorageSummary fixed class ceiling is 3.36; discovery is used only without selected default, and selected high versions remain uncapped. Fetch override belongs only to second request; explicit empty override omits version headers. Stock availability is3.12+, actual low-version404 retained rather than invented local support.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummarySingleFetchAndFetchSpecificVersion`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryDiscoveryCeilingAndSelectedVersionNotClamped`.
- Untyped id/name/total_size/total_count/metadata and missing-null declarations own exact arbitrary integer/false/list values. Resource/Wire/whole Envelope/Header/status, terminal fetch Requests and independent Discovery do not alias; accepted nonJSON preserves seed/default, invalid UTF8/object shape preserves actual accepted receipt.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryDefaultTwoFetchesAndAllProjects`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryAcceptedNonJSONAndFlatEnvelopeFallback`.
- Owned JSON-domain kwargs follow Requests iterable expansion and null omission, SDK AllProjects remains authoritative. Deep query/map/raw/pointer option snapshots, nil Header-map composition and option reset are captured without changing original service. Controls and invalid Source/context/MV/header configuration reject before marshaler/discovery/HTTP. Pinned fetch headers kwarg is an ordinary Requests dictionary query and is distinct from copied typed actual HTTP Headers; query null omits and only second fetch carries query.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryInvalidControlsBeforeMarshalerOrHTTP`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryQuerySnapshotsRequestsExpansionAndOptionReset`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryHeadersKwargIsQueryAndTypedHeadersArePhysical`.
- Actual200/203/401/404/429/503 Read,Close and mixed faults retain real body/header/status plus every cause; native401 pointer ErrUnableToReauthenticate preserves ErrOriginal/ErrReauth. Willing Retry,Backoff,Reauth stay zero after physical fault; oneHTTP/oneClose. Source read-boundary mutation is sticky after Close restore, custom cancellation is terminal. Clean401 renewal and503 retry keep selected request; body/version/foreign header/omission mutations retain first rejection and stop.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryPhysicalFaultsKeepNative401PublicFieldsAndNoHooks`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryCleanReauthenticationAndRetryKeepOwnedRequest`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryReadBoundaryGuardAndCancellationCannotResend`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryRetryCannotInjectBodyOrSelectedHeaders`.

Python/Go differences:

- Go library defaults all_projects tofalse through functional options, while pinned Proxy.summary requires an explicit bool. Default retains Source two fetches; WithSummarySingleFetch(true) explicitly improves redundant first request and drops its seed contribution.
- Standalone leaf projection uses computed locationnull with unknown/location retained only inWire, a documented SDK-R1 boundary; Python Connection-derived dynamic Resource dirty/cache/session/warning behavior remains project-level parity work.
- Go Headers is a copied typed extension. Raw Source controls, unsafe route/session rewriting and duplicate all_tenants/requires_id bindings are rejected before marshaler/HTTP rather than allowing Python kwargs collisions after its firstGET. Exact envelope/header/status/fault proof and native original causes remain.
- Source accepted non-JSON bodies retain seed/default. Go additionally requires UTF8 and nonnull object representations for parsed JSON; receipt-bearing error is exposed for invalidshape rather than Python attribute-method exceptions.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.36")
    parser.add_argument("--all-projects", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    value = conn.block_storage.summary(all_projects=args.all_projects)
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.36", "selected Cinder version")
    all := flag.Bool("all-projects", false, "ask for all projects")
    single := flag.Bool("single-fetch", false, "omit pinned redundant first GET")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    value, err := service.Volumes.SummaryRecord(ctx,
        volumes.WithSummaryAllProjects(*all),
        volumes.WithSummarySingleFetch(*single))
    if err != nil { log.Fatal(err) }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}
```

## References

- [summary Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1059-L1080)
- [BlockStorageSummary](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/block_storage_summary.py)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [ListSummary](volume-summary-python.md)
- [Install](../../docs/install.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#get-volumes-summary)
- [Volumes](https://github.com/openstack/cinder/blob/master/cinder/policies/volumes.py)
- [get_volume_summary](https://github.com/openstack/cinder/blob/master/cinder/volume/api.py)
- [python_summary_contracts_test.go](volumes/python_summary_contracts_test.go)
