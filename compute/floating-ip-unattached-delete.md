# Floating Ip Unattached Delete

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network_common/NetworkCommonCloudMixin/delete_unattached_floating_ips`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteUnattachedFloatingIPs`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.DeleteUnattachedFloatingIPs`
- `github.com/JSYoo5B/go-openstacksdk/compute.PrepareFloatingIPDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteRetries`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteSource`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteStrict`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteDirectGet`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUnlimitedFloatingIPDeleteTimeout`

- The Neutron cleanup finishes a two-page unfiltered public inventory before any DELETE, preserves source order, skips an attached row with an unusable ID and sends no If-Match revision header. Its exact sequence is two inventory GETs, DELETE a/public-Get absence verification, then DELETE b/public-Get verification; only Network is located. Count2/AllDeleted=true and the owned two-page inventory, accepted204/verified attempts and absence flags are retained.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedEagerPagedInventoryAndConfiguredDeleteOrder`.
- Configured Nova/None returns a successful ineligible zero without any locator or HTTP. Clean Network endpoint absence locates only Network and skips inventory/Compute/delete. A mixed absence plus terminal catalog cause remains an error with that cause and no aggregate success. Empty or attached-only eligible inventory returns Count0/AllDeleted=true after one Network list and no DELETE; the whitespace port remains attached.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedSourceAndServiceGateDistinguishesSuccessfulZero`.
- A first clean DELETE404 records an unaccepted false and still processes later candidates; a second logical DELETE404 after accepted deletion/ACTIVE verification similarly preserves both attempts and continues. Both completed false-containing runs keep all three items, Count2 and AllDeleted=false with no error. Actual DELETE403 or verification403 stops at the second item, retains Count1 and the failed item proof, and never deletes the third candidate.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedFalseContinuesAndErrorStopsWithPartialHistory`.
- The original options execute once across two candidate items and observe default retry1. Retry1 adds one logical DELETE for the first still-ACTIVE IP, retry3 adds three, retry0 makes one passive accepted DELETE per item with no verification, and negative retry-2 makes one DELETE with verification. Nonzero verification finishes on logical DOWN and the exact request/attempt totals are asserted; Count2/AllDeleted=true reflects the mapped per-item boolean rather than actual disappearance.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedPreparedRetryPolicyIsAppliedOnceAcrossItems`.
- Candidate selection consumes public Resource port_id with JSON truthiness. Tested missing/null/empty-string/false/zero/empty-array/empty-object produce seven ordered deletions, while whitespace/string-zero/nonzero-number/nonempty-array/nonempty-object rows with unsafe IDs are skipped. Missing Neutron port is logical null and remains absent from actual Wire, including Strict=true.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedResourcePortTruthinessAndStrictFallback`.
- A clean Neutron inventory404 can fall back to numeric-ID7 Nova inventory normalized under configured Neutron. Nonstrict view supplies the null port_id alias and the subsequent DELETE still targets captured Neutron /floatingips/7, not the inventory Nova backend. Strict fallback instead lacks port_id and returns ErrInvalidOption before DELETE, preserving Eligible, inventory/fallback proof, Count0 and AllDeleted=false.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedResourcePortTruthinessAndStrictFallback`.
- Incomplete public inventory never deletes an eligible prefix: tested early403, first-row descriptor failure, later403, wrong-shaped later200, continuation cycle, accepted later Close error, cached endpoint drift and caller cancellation all leave Items empty, Count0, AllDeleted=false and no DELETE. Consumed page proof remains; physical403 or accepted200 ResponseError/Failure and custom Close/cancellation/source causes are distinguished.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedInventoryErrorsNeverDeleteEligiblePrefix`.
- Executable IDs are consumed sequentially only after port eligibility. A later unsafe or missing ID errors before that DELETE while preserving the earlier completed/accepted item and Count1, plus the bad selected Resource with Deletion=nil and Item.Error. Duplicate a,a,c remains three ordered calls: the second a404 is a false item and processing continues to c, returning Count2/AllDeleted=false. Attached unsafe IDs are never consumed for deletion.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedSequentialInvalidIDAndDuplicateHistory`.
- After candidate a completes, candidate b accepted202 with Close failure, cached Network API replacement, custom caller cancellation or a real25-ms operation expiry retains its passive body/accepted response and202 ResponseError but is not counted as deleted. Count1 and the two items remain, no candidate c is deleted, and Close/cancel/source/DeadlineExceeded causes remain observable. No verification is performed under the prepared retry0 policy.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedAcceptedFailuresAndOneOperationBudget`.
- One absolute deadline spans the inventory and all candidate deletions rather than resetting per item. The fixtures assert no deadline for default/unlimited, a nonzero shared deadline for explicit timeout, exact earlier one-minute parent deadline under a two-minute SDK cap, and unlimited clearing a previous cap. Successful retry0 runs complete three items; the real25-ms boundary preserves partial progress instead of detaching later work.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedAcceptedFailuresAndOneOperationBudget`.
- Bulk Source/Location factories retain the original Neutron/source project facts despite later caller pointer/raw-byte mutation. Callback count is one per aggregate call. Each selected duplicate row independently owns Resource/Wire; mutating one item or its verification model leaves inventory, the other item, its Wire and independent response headers unchanged. Reusing the same factory after output mutation restores original scope/name and succeeds again.
  Tests: `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedOwnedSnapshotsAndReusablePreparedOptions`.
- Nil Connection/Service and nil context/option, invalid source, dedicated zero timeout, negative bulk timeout or pre-cancelled context fail before HTTP/locator and preserve typed/cancellation causes, even for a source-None skip. A directly acquired Service whose original option clears the captured Servers pointer fails before HTTP rather than recapturing the replacement; its earlier explicit Compute discovery is recorded rather than claimed to have locator0.
  Tests: `connection_floating_ip_unattached_delete_boundaries_test.go:TestFloatingIPUnattachedPreflightAndCapturedSource`, `connection_floating_ip_unattached_delete_test.go:TestFloatingIPUnattachedOwnedSnapshotsAndReusablePreparedOptions`.
- A completed lower Delete=true stays in aggregate Count even when an injected later aggregate operation guard fails. The finite final-guard injection scan requires reaching at least one completed item, keeps Count1 and AllDeleted=false, preserves the guard cause, and never sends more than one DELETE. Accepted-but-failed lower deletion remains a separate uncounted case.
  Tests: `connection_floating_ip_unattached_delete_boundaries_test.go:TestFloatingIPUnattachedCompletedDeleteSurvivesAggregateGuardFailure`.
- A successfully acquired Compute Service and a later Connection cleanup reuse the same cached Connection dependencies. Both calls override configured source None with per-call Neutron, complete two default-retry items with public absence verification, execute original options once per call and retain exact Compute-then-Network locator count without rediscovery. Each five-request inventory/delete/verification sequence shares a nonzero minute cap; item errors remain nil and actual Neutron verification pages are retained.
  Tests: `connection_floating_ip_unattached_delete_boundaries_test.go:TestFloatingIPUnattachedCachedServiceAndConnectionShareDependencies`.

Python/Go differences:

- The named cleanup declaration has only retry=1. Go reuses concrete Delete options and returns an owned result separating eligibility, complete inventory proof, ordered item decisions/deletion results, a count of completed Deleted=true items, aggregate AllDeleted and an optional item Error. Python returns int Count if all candidate booleans succeed, False after any clean false, or raises and discards earlier booleans on an exception.
- Empty or ineligible cleanup maps the actual source all([]) integer0 to Count=0/AllDeleted=true, despite the source docstring saying False if none. Go additionally retains a partial successful Count on a false or error, including a lower Delete already completed before a later aggregate guard error; accepted lower requests whose processing/verification failed are not counted.
- Eligibility uses configured Neutron and a captured lazy Network binding. Pure endpoint absence is an empty skip; mixed source/context/catalog failures propagate instead of reproducing broad Python has_service exception swallowing. Explicit Nova/None never makes this function clean up Nova floating IPs. Generic has_service/session/config negotiation remains separately tracked.
- Candidates consume the public Resource port_id with ordinary JSON truthiness, rather than Wire-only fields or canonical port/attached aliases. Strict Nova fallback lacks the compatibility port_id and raises a typed structural error, preserving the source attribute-access failure. Configured Neutron remains the DELETE backend even if the public inventory falls back to Nova.
- IDs are consumed sequentially only for unattached rows from public Resource data. Owned passive nullable/malformed IDs remain in Inventory/item proof, while unsafe/null/container executable identities return local typed errors before that DELETE. Arbitrary Python object coercions and Resource-like executable dictionaries are not exposed.
- All caller options are prepared once and one optional whole-operation deadline covers backend selection, complete inventory, every per-IP retry and verification. Source/context guards and independently cloned selected rows/actual HTTP proof are explicit Go semantics; no caller cancellation is bypassed for detached cleanup.
- Per-IP deletion retains the mapped standalone Delete contract: exact retry0 skips verification, negative retry performs one DELETE plus verification, default1 adds one logical attempt, no sleep, DELETE404 false continues the aggregate loop, and public Get absence/DOWN succeeds. DOWN or retry0 true is a source boolean policy rather than identity-checked physical disappearance.
- The named workflow is a snapshot inventory followed by ordered deletions, not an atomic association revalidation or revision-aware purge. The pinned two-step create/attach race note still applies. Independent Create/availability/attach/detach/native/Proxy/mutable Resource/session APIs are separate SDK scope and are not missing declared arguments of this cleanup.
- Existing typed query envelope/status/continuation and legacy Compute microversion checks remain documented Go policies. Selected Nova microversion>=2.36 is unsupported without hidden downgrade if a fallback query is actually required; final informational1xx/session redirect combinations and malformed Python runtime corners are not exact transport parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
result, err := conn.DeleteUnattachedFloatingIPs(ctx,
	compute.WithFloatingIPDeleteRetries(1),
	compute.WithFloatingIPDeleteTimeout(time.Minute),
)
```

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	retry := flag.Int("retry", 1, "additional deletes per IP; zero skips verification")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	strict := flag.Bool("strict", false, "strict Nova fallback view")
	direct := flag.Bool("direct", false, "direct UUID verification GET")
	timeout := flag.Duration("timeout", time.Minute, "whole cleanup cap; zero removes SDK cap")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.Compute, "2.35"),
	)
	if err != nil {
		return err
	}
	opts := []compute.FloatingIPDeleteOption{
		compute.WithFloatingIPDeleteRetries(*retry),
		compute.WithFloatingIPDeleteStrict(*strict),
		compute.WithFloatingIPDeleteDirectGet(*direct),
	}
	if *source != "" {
		opts = append(opts, compute.WithFloatingIPDeleteSource(compute.FloatingIPSource(*source)))
	}
	if *timeout == 0 {
		opts = append(opts, compute.WithUnlimitedFloatingIPDeleteTimeout())
	} else {
		opts = append(opts, compute.WithFloatingIPDeleteTimeout(*timeout))
	}
	result, cleanupErr := conn.DeleteUnattachedFloatingIPs(ctx, opts...)
	var summary map[string]any
	if result != nil {
		summary = map[string]any{
			"eligible": result.Eligible, "count": result.Count,
			"all_deleted": result.AllDeleted, "items": len(result.Items),
		}
		if result.Inventory != nil {
			summary["inventory_backend"] = result.Inventory.Backend
			summary["inventory_pages"] = len(result.Inventory.Pages)
			summary["inventory_rows"] = len(result.Inventory.FloatingIPs)
		}
		items := make([]map[string]any, 0, len(result.Items))
		for _, item := range result.Items {
			entry := map[string]any{"id": item.ID}
			if item.Error != nil {
				entry["error"] = item.Error.Error()
			}
			if deletion := item.Deletion; deletion != nil {
				entry["backend"] = deletion.Backend
				entry["deleted"] = deletion.Deleted
				entry["absent"] = deletion.Absent
				entry["down"] = deletion.Down
				entry["attempts"] = len(deletion.Attempts)
			}
			items = append(items, entry)
		}
		summary["item_results"] = items
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(cleanupErr, encoder.Encode(summary))
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
deleted = conn.delete_unattached_floating_ips(retry=1)
if deleted is False:
    print("Some candidate deletions returned False")
else:
    print("Completed:", deleted)  # 0 is a successful empty result
```

## References

- [Floating Ip Queries](floating-ip-queries.md)
- [DeleteFloatingIP](floating-ip-delete.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1048-L1078)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L511-L566)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L977-L1046)
