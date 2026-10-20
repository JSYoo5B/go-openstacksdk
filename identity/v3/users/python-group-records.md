# Python Group Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/group_users`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListInGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListInGroupOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListInGroupQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListUsersInGroupRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.AllUsersInGroupRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ProxyUsersInGroupRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithGroupRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.UserRecord.Clone`

- ListInGroup sends GET groups/{id}/users with the User query mapping such as password_expires_at like Python group_users and sends a limit without a marker follow-up.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserGroupUsersRequests`.
- group_users uses complete User profile, not reduced subclass: declared query incl federated IDs/expiration/enabled alias and local email/Body preserve raw dates/defaults/casts/unknown discard with literal escaped fixed Group collection and actual private proof.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsFullUserProfileAndFixedGroupScope`.
- Immutable group route persists across short-limit raw cursor pages, filtering/cap/break and caller view mutations; Source root dictionary/null and cross-group continuation guard keep original actual page proof.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsMarkerRawCapAndBreakKeepGroupRoute`, `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsSourceLinksAndCrossGroupContinuationProof`.
- Source-bound base_path/null/raw group collision and restored duplicate controls fail before HTTP; malformed literal ID and bool limit use documented safe Go validation without fabricated response.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsBasePathBindingAndBooleanLimitPreflight`.
- Opaque group Proxy uses original User generator model without HTTP for projection/type errors and preserves lazy identity; group-specific profile copy never changes subsequent standalone User list path.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsOpaqueProxyAndStandaloneProfileIsolation`.
- Loaded nested User Body subset/equality and explicit immutable User location retain configured scope despite default_project_id/unknown project_id/location. Full User schema/header/version formal restore is SDK-owned.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsLoadedLocalFilterAndExplicitLocation`, `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsFormalRestorationUsesFullUserSchema`.
- Option map/raw/pointer snapshots and sticky before/after selected source/custom cancel guard prevent restoration callbacks and physical requests on invalid preflight.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsOptionsCloneStickySourceAndCancel`.
- 28 physical accepted/rejected Read/Close/restored Source/cancel lanes preserve actual body/header/status and independent value ErrOriginal/native pointer401 ErrReauth, Close once and no fault retry; clean401/503/429/498 uses live token and frozen group request.
  Tests: `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsPhysicalFaultsKeepActualProofAndStopRetry`, `identity/v3/users/python_group_records_test.go:TestPythonGroupUserRecordsCleanRetryKeepsGroupScopeAndLiveAuth`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Library-owned concrete functional options, raw nullable Source view and separate actual receipt preserve existing native APIs. No user builder interface or new transport framework.
- Go caps/limit require nonnegative decimal integers. Source raw limit True/False is sent as True/False and truthy True activates fallback; Go rejects bool before HTTP and does not substitute1/0. Safe fixed origin/path/query/header/cursor/cycle bounds are explicit.
- Source rootlink JSON precedence/dictionary-ignore/null-error is preserved. RFC HTTP Link uses explicit Go interoperability for ordinary Requests next[url] versus Source next[uri] bug.
- Unconsumed Proxy generator is opaque and no fake JSON/HTTP association is created. Go iter.Seq2 reuse reissues captured requests instead of sharing Python one-shot state.
- Standalone default location is null. Explicit cloned recorded facts are a Go convenience; Region has no zone descriptor and normalizes zone null, while GroupUser reuses existing User explicit-zone convenience. Unknown response scope fields are not trusted.
- Already selected Keystone v3 native API required. Mutable Resource object/group subclass routing, automatic Connection location/cache, generic synchronized lifecycle and session remain SDK-R1/C1/S1.
- Group identity is a required literal escaped segment; Source Group object base_path changes, slash path composition and percent URI formatting failures are bounded rather than copied. New Group-to-User names avoid existing inverse User-to-Group ListGroupRecords. Source base_path keyword presence incl null is rejected as already bound.
- Generic SDK-R1/C1/S1 response cache, arbitrary synchronized Resource/object lifecycle and automatic Connection/session bridge are not completed. These two finite mappings have supported delta0; root Go/race/main gates must pass before catalog promotion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--group-id", required=True)
    args = parser.parse_args()
    conn = openstack.connect()
    for row in conn.identity.group_users(
        args.group_id, email="member@example.com", limit=100, max_items=100
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))


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

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
)

func main() {
    groupID := flag.String("group-id", "", "required group UUID")
    flag.Parse()
    if *groupID == "" { log.Fatal("-group-id is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Users.ListUsersInGroupRecords(ctx, *groupID,
        users.WithGroupRecordListQuery("email", "member@example.com"),
        users.WithGroupRecordListLimit(100),
        users.WithGroupRecordListMaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s\n", data)
    }
}
```

## References

- [Group](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/group.py)
- [controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/groups.py)
- [Python Records](python_records.md)
