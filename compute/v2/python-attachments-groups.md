# Python Attachments Groups

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/find_server_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithGroupFindAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithGroupFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithGroupFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithGroupFindHeaders`

- GET-first exact identity lookup falls back only for fault-free 400/403/404, matches either ID or name, and adds all_projects=True only to the fallback list. Untyped descriptors, Source dict rules, nullable fields, aliases and raw unknown response values are retained.
  Tests: `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordUsesGETThenExactIDOrNameAndAllProjects`.
- Missing defaults to nil,nil, strict missing is ErrNotFound, and multiple exact ID-or-name matches are AmbiguousError; later advertised pages are read before returning a candidate and late HTTP failure remains observable.
  Tests: `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordMissingDefaultStrictAndDuplicate`, `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordReadsLaterPagesBeforeReturningCandidate`.
- Physical rejected body Read/Close/cancel/transient source changes retain actual native status/body/header and never cause fallback, while a clean native retry callback returning the original rejection still permits fallback.
  Tests: `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordRejectedPhysicalErrorsPreventFallback`, `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordCleanRetryErrorAllowsFallback`.
- Options are snapshotted, nil options/invalid identity/protected headers fail before HTTP, joined cancellation causes survive, and accepted body Read/Close errors preserve actual response proof.
  Tests: `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordSnapshotsOptionsAndRejectsBeforeHTTP`, `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordSuccessfulReadClosePreservesProof`.
- An absent default version discovers the Source 2.64 ceiling; a configured lower version remains unchanged without discovery.
  Tests: `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordDiscoversResourceCeilingAndHonorsSelectedVersion`.

Python/Go differences:

- The SDK returns owned Resource and Wire record views with actual envelope/header/status rather than a mutable Python Resource; the record location is null and the shared complete cloud-location contract remains separately tracked.
- Explicit string identifiers, source/header guards, retained callback snapshots and same-origin pagination validation reject invalid or changing routes before physical requests.
- A configured microversion is honored unchanged; when absent, shared Nova discovery applies the Source Resource ceiling without mutating the original service client.
- Group lookup requires string ID/name; JSON bool/number identity fields are retained in the record but are not coerced into Go identity strings.

### `python:compute/v2/volume_attachments`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.VolumeAttachmentScope.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentLimit`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentOffset`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentPaginated`

- The lazy fixed-server list sends limit and offset, retains offset on Source marker fallback even after short pages, falls back from absent id to the alternate volumeId, and stops on the next empty page without using consumer-mutated rows.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordsListIsLazyAndPagesLimitOffset`.
- max_items counts received rows before canonical Body filters; zero limit, null offset omission, unknown query discard, paginated=False and early break retain Source behavior and stop further HTTP.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordsControlCountsRawRowsBeforeFilters`.
- Plural singleton objects and advertised next links preserve last response alias order for bdm_uuid/bdm_id and volumeId/volume_id and exact raw unknown numbers.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordsLinksAndLocalAliasProjection`.
- Negative/invalid/nil pagination inputs and protected headers fail before HTTP; accepted 204 without a collection representation is an error carrying the actual response.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordsRejectInvalidInputsBeforeHTTP`, `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordUpdateSnapshotsAndScope`.
- An absent selected version discovers min(2.89, advertised maximum), while an explicit lower version is honored with no discovery or promotion.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordsDiscoverCeilingWithoutChangingClient`.

Python/Go differences:

- The SDK returns owned Resource and Wire record views with actual envelope/header/status rather than a mutable Python Resource; the record location is null and the shared complete cloud-location contract remains separately tracked.
- Explicit string identifiers, source/header guards, retained callback snapshots and same-origin pagination validation reject invalid or changing routes before physical requests.
- A configured microversion is honored unchanged; when absent, shared Nova discovery applies the Source Resource ceiling without mutating the original service client.
- Go local Body filters deliberately distinguish JSON bool from number and use exact decimal equality; unlike Python malformed object filtering, a nonobject actual value is a mismatch rather than an attribute exception. Typed pagination requires one nonnegative integer limit/offset; dictionary links are an explicit Go compatibility extension.

### `python:compute/v2/update_volume_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.VolumeAttachmentScope.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentSwapTo`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentAttribute`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithAttachmentUpdateHeaders`

- One PUT uses the current volume ID in the route and defaults volumeId to that ID; swap changes only the body. False, zero, explicit null and empty values persist, unknown attrs are discarded before marshaling, and named volume_id=None differs from a wire volumeId=null attr.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordUpdateDefaultsSwapAndExplicitValues`.
- Input/option snapshots remain independent of caller mutations; scoped explicit parent ID needs no lookup. Bodyless accepted success retains declared seeds and actual 204 proof without inventing a received row.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordUpdateSnapshotsAndScope`.
- Invalid fixed target attrs, identifiers, nil options and protected headers fail before HTTP; successful response Read/Close errors, option cancellation and source mutation retain all physical/callback/context causes without resend.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordsRejectInvalidInputsBeforeHTTP`, `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordsPreservePhysicalErrorsAndCanceledOptions`.
- Source Resource ceiling 2.89 is selected only when no default microversion exists; explicit 2.1 remains 2.1 and the service client is unchanged.
  Tests: `compute/v2/volumeattach/python_records_contracts_test.go:TestPythonAttachmentRecordsDiscoverCeilingWithoutChangingClient`.

Python/Go differences:

- The SDK returns owned Resource and Wire record views with actual envelope/header/status rather than a mutable Python Resource; the record location is null and the shared complete cloud-location contract remains separately tracked.
- Explicit string identifiers, source/header guards, retained callback snapshots and same-origin pagination validation reject invalid or changing routes before physical requests.
- A configured microversion is honored unchanged; when absent, shared Nova discovery applies the Source Resource ceiling without mutating the original service client.
- The API returns an AttachmentRecord although the Python proxy docstring says None; pinned implementation actually returns the updated VolumeAttachment Resource.
- Named swap and attrs are concrete options; ID/parent overrides are rejected rather than exposing duplicate-keyword Python TypeError.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
for attachment in conn.compute.volume_attachments(
    server_id, limit=20, offset=0, max_items=100,
):
    print(attachment.to_dict())

conn.compute.update_volume_attachment(
    server_id, current_volume_id, delete_on_termination=False,
)
conn.compute.update_volume_attachment(
    server_id, current_volume_id, volume_id=replacement_volume_id,
)
group = conn.compute.find_server_group(group_name, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    serverID := flag.String("server", "", "server ID to list attachments")
    groupName := flag.String("group", "", "server group exact ID or name")
    allProjects := flag.Bool("all-projects", false, "search server groups across projects")
    updateVolume := flag.String("update-volume", "", "current volume ID to update")
    replacement := flag.String("swap-to", "", "replacement volume ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.Compute, "2.89"))
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    if *serverID != "" {
        for value, err := range service.VolumeAttachments.ListRecords(ctx, *serverID,
            volumeattach.WithAttachmentLimit(20),
            volumeattach.WithAttachmentOffset(0),
            volumeattach.WithAttachmentMaxItems(100)) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    }
    if *groupName != "" {
        value, err := service.ServerGroups.FindRecord(ctx, *groupName,
            servergroups.WithGroupFindIgnoreMissing(false),
            servergroups.WithGroupFindAllProjects(*allProjects))
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
    }
    if *updateVolume != "" {
        if *serverID == "" { log.Fatal("-server is required for -update-volume") }
        options := []volumeattach.AttachmentUpdateOption{
            volumeattach.WithAttachmentAttribute("delete_on_termination", false),
        }
        if *replacement != "" {
            options = append(options, volumeattach.WithAttachmentSwapTo(*replacement))
        }
        value, err := service.VolumeAttachments.UpdateRecord(ctx,
            *serverID, *updateVolume, options...)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
        fmt.Printf("HTTP %d request-id=%s\n", value.StatusCode, value.Header.Get("X-Openstack-Request-Id"))
    }
}
```

## References

- [Install](../../docs/install.md)
