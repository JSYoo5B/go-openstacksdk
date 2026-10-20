# Floating Ip Delete

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network_common/NetworkCommonCloudMixin/delete_floating_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.DeleteFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.PrepareFloatingIPDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteRetries`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteSource`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteStrict`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteDirectGet`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPDeleteTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUnlimitedFloatingIPDeleteTimeout`

- The default extra retry is one logical DELETE in addition to the initial attempt. In the Neutron fixture, accepted DELETE204 followed by public list-based Get of the same ACTIVE ID causes a second DELETE; the next empty list completes with Deleted=true/Absent=true/Down=false. The exact DELETE/GET/DELETE/GET order uses Network only, and both accepted attempts retain actual204 proof, verified flags and one query page each.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeleteDefaultRetryAndPublicLookup`.
- Retries=0 and a zero-valued bulk options replacement issue one accepted DELETE202 and skip verification, with no LastVerification, Absent or Down assertion. Negative retry -5 issues one DELETE plus public verification and completes on absence. Malformed passive DELETE body is preserved as response evidence without requiring a JSON floating-IP model. Pure preparation defaults to retry1/no SDK timeout/DirectGet=false, preserves negative retries and applies bulk replacement/unlimited timeout with last-option precedence.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeleteRetryZeroNegativeAndBulkOptions`, `compute/floating_ip_delete_options_test.go:TestFloatingIPDeleteOptionsDefaultsRetriesAndBulkReplacement`.
- A still-present logical Neutron status DOWN satisfies cloud's success policy after one accepted DELETE and one verification. Both literal DOWN and JSON-escaped equivalent are recognized; Deleted=true/Down=true while Absent=false and the actual row remains available. This is not a proof of resource disappearance or readiness.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeleteDownIsPresentAndJSONSemantic`.
- A present Neutron row with status ACTIVE, ERROR, null, false or lowercase down does not satisfy absence/DOWN. After two default logical attempts, FloatingIPDeleteVerificationError retains requested ID, attempt count2 and the last known row; Deleted/Absent/Down remain false and no third DELETE is sent.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeleteVerificationExhaustionKeepsLastRow`.
- Actual DELETE404 returns Deleted=false without an error, verification or alternate-backend fallback in tested Neutron, Nova and source None branches for retry0/1/3/-1. The attempt remains unaccepted, NotFound exposes actual404 and Failure retains the actual failed response. Only one endpoint and one physical DELETE are used.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeleteNotFoundIsFalseWithoutFallback`.
- A first accepted DELETE204 and still-present ACTIVE verification followed by a second DELETE404 returns the source's final false result, retaining both attempts and LastVerification. The first accepted response header and second missing response body remain independently available; the second404 ends without another verification.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeleteSecondMissingRetainsAcceptedHistory`.
- Nova and source None verification consume the public cloud-normalized view: a wire DOWN row with numeric ID7 is logical ACTIVE and therefore exhausts two attempts rather than satisfying DOWN. Wire retains actual DOWN. After an accepted DELETE followed by Nova list404, public lookup suppression produces absence and the delete succeeds with SuppressedNotFound retained. These tests distinguish source normalization from actual response status and do not prove IP readiness.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeleteNovaAndNoneVerificationUsesNormalizedCloudView`.
- After a Neutron DELETE204, its independent public lookup may encounter an unfiltered Network list404 and read Nova. The tested Nova row's JSON-escaped DOWN is normalized using the configured Neutron mode and satisfies the source DOWN policy. Mutation Backend remains Neutron, verification Backend is Nova, the original list404 is retained as FallbackError, and only three logical HTTP requests occur; this is query fallback, not DELETE404 mutation fallback.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeleteVerificationCanFallbackAfterNeutronAcceptance`.
- The owned delete transport preserves passive response bodies and accepts the tested source-style HTTP200/201/202/203/204/205/206/299/300/301/304/307/399 responses without a Location redirect or JSON decode requirement. With retry0 each response is one accepted mutation, retains its actual status/malformed body and returns Deleted=true without verification. These finite status fixtures do not prove every redirect/session combination or native Gophercloud default delete-code parity.
  Tests: `connection_floating_ip_delete_test.go:TestFloatingIPDeletePassiveBodyAndSuccessfulStatusPolicy`.
- Pure concrete option preparation owns Source, nested Location pointers and project raw JSON across later caller and returned-policy mutation. A custom callback is applied once, and a later callback cannot observe an externally mutated alias from the preceding callback. The separate HTTP whole-budget fixture verifies once-applied original options across two logical attempts and nested public verification; it does not cover every callback/type/backend combination.
  Tests: `compute/floating_ip_delete_options_test.go:TestFloatingIPDeleteOptionsOwnSourceAndLocation`, `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteWholeBudgetOptionsAndParentDeadline`.
- Pure preparation rejects nil option/context, invalid source, zero duration in the positive-timeout helper and negative bulk timeout as ErrInvalidOption. A pre-cancelled context preserves its custom cancellation cause and prevents option callbacks. Negative retry remains a supported source value rather than a negative-timeout error.
  Tests: `compute/floating_ip_delete_options_test.go:TestFloatingIPDeleteOptionsPreflightAndCancellationCause`.
- A negative retry -2 clamps to one logical DELETE with one verification; explicit additional retries3 allow four logical DELETE/verification pairs. A still-present ACTIVE row exhausts the exact max(0,retry)+1 limit with retained attempt count and Deleted=false.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteNegativeAndAdditionalRetryLimits`.
- Verification consumes the advertised second marker page before deciding the first accepted deletion: a late exact-ID DOWN match completes with two query pages, but late403, wrong-shaped200 collection or duplicate exact-ID matches fail without marking Verified/Deleted or issuing a second logical DELETE. Duplicate selection exposes ErrAmbiguous and actual late403 is retained in Failure.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteVerificationConsumesPagesAndRejectsIncompleteSelection`.
- Default UUID verification and DirectGet=true with a non-UUID ID still use public list selection; empty lists complete as absent. Explicit direct UUID verification sends a member GET: actual member404 remains an error after accepted deletion, not a mutation NotFound or successful absence. Following source getter trust, a member containing a different ID with DOWN can satisfy Down=true/Absent=false; the actual other ID stays in Wire and this result does not prove disappearance of the requested ID.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteDirectVerificationAndMemberNotFound`.
- An accepted DELETE202 followed by Close failure, custom cancellation cause or cached Network API replacement retains actual accepted response/header/passive body and ResponseError202. Each mode ends after one physical DELETE with Accepted=true, Verified=false, no LastVerification and Deleted=false; Close/cancel preserve the custom cause.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteAcceptedFailureRetainsReceiptAndStops`.
- After accepted DELETE204, an accepted GET200 DOWN response whose Close fails, cancels with a custom cause or changes the cached Network endpoint remains a verification error. The prior DELETE proof and LastVerification survive, GET200 ResponseError/Failure is retained, Verified/Deleted remain false and no second DELETE occurs. Close/cancel retain the custom cause.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteVerificationAcceptedFailureKeepsPriorDelete`.
- Direct DELETE 403/503 failures retain original status/body without becoming NotFound or triggering logical retry. A native callback expanding OkCodes with404 yields one callback/two physical DELETEs but one logical attempt and a terminal SDK failure, not successful/missing deletion. A 503 callback that changes the cached Network endpoint yields one callback/one physical DELETE, retaining the original503 and blocking a second request.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteHTTPFailureAndRetryTerminalAreNotMissing`.
- Across the two-attempt DELETE/GET/DELETE/GET fixture, original delete option callbacks apply once and the result retains the entry ID despite an outside input edit. All four request deadlines are identical: default and explicit unlimited have no SDK cap, a positive timeout is nonzero, and a ten-second parent remains the exact earlier deadline. The dedicated unlimited helper clears the preceding positive timeout.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteWholeBudgetOptionsAndParentDeadline`.
- A real 60 ms SDK timeout expires while the verification request waits for its context. DeadlineExceeded is preserved after the accepted DELETE204, with exactly one DELETE and one GET, the accepted attempt still observable and Deleted=false; no further logical attempt is sent.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteRealDeadlineStopsAfterAcceptedDelete`.
- Connection nil/context-nil/pre-cancelled/unsafe-path-ID/nil-option preflight stops with no HTTP and no service locator, preserving a custom cancellation cause. An explicitly acquired Service whose Servers pointer is cleared in an option returns ErrInvalidOption before HTTP instead of recapturing the changed source. Legacy Nova microversion2.36 returns ErrUnsupported before mutation; its earlier explicit Compute discovery is not claimed to have locator0.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeletePreflightCapturedServiceAndLegacyVersion`.
- An accepted DELETE 202 whose response read returns a custom failure retains that cause, ResponseError status 202 and the partial passive body in both the accepted attempt receipt and error proof. It sends one physical DELETE, returns Deleted=false, preserves the accepted logical attempt and performs no public verification. This is bounded mutation read-error evidence, not every read/Close/backend combination.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteAcceptedReadErrorPreservesBodyAndCause`.
- An explicitly acquired Compute Service uses its captured Connection dependencies to select Neutron for a successful retry0 deletion. The first direct Service call locates Compute then Network and sends one DELETE; a subsequent Connection retry0 call succeeds with one more DELETE and no additional locator. This proves the tested cached dependency path without asserting shared role-cache mutation, verification or every receiver/backend policy.
  Tests: `connection_floating_ip_delete_boundaries_test.go:TestFloatingIPDeleteServiceUsesCachedConnectionDependencies`.

Python/Go differences:

- Go exposes a typed request and concrete options with a result separating the source boolean Deleted from actual absence, present DOWN, accepted logical attempts, last public-Get verification and failed-response evidence. An accepted DELETE is not itself verified disappearance; retry0 intentionally has no verification. This does not reproduce Python's scalar return or mutable Resource/Munch/session surface.
- The delete's verification deliberately reuses ordinary cloud Get semantics rather than always using GetFloatingIPByID. Mutation backend, query fallback backend and normalization source remain distinct. Nova's source-synthetic logical ACTIVE is separated from raw Wire; a DOWN success can describe a still-existing row and never asserts an actual ACTIVE wait or deletion convergence.
- Typed safe UTF-8/path IDs, explicit source/location/strict/direct-get policy and optional one shared operation timeout are Go API choices. Defaults add no SDK deadline; a bulk zero-valued options replacement sets retry0, while dedicated default preparation retains retry1. The mapping does not establish every Python dynamic retry argument, configuration or has_service/session policy.
- Owned mutation transport follows the cloud-style passive HTTP200–399 response policy rather than promoting Gophercloud's native DELETE status contract. Actual response/body/header proof is retained separately from model parsing and success verification. The tested Neutron accepted Close/cancel/source and terminal retry failures preserve partial evidence, while full redirect/session/retry/default/backend combinations remain unresolved.
- Direct UUID verification intentionally preserves source getter trust: a different response ID with DOWN can satisfy the source boolean, with the actual row retained and Absent=false. The result is a source success policy rather than identity-checked requested-resource disappearance. Dedicated deletion options do not add filters or expression parameters that are absent from the source deletion signature.
- Selected Compute microversion2.36 or higher is rejected as ErrUnsupported before legacy Nova DELETE. The pinned cloud helper issues its legacy request and can translate an actual404 into false; Go's local version preflight is a deliberate difference and does not establish automatic endpoint/API-version negotiation or downgrade.
- The declared ID/retry deletion contract is mapped. Connection/profile use_direct_get configuration is expressed as an explicit Go option with default false; typed safe IDs and int retries, exact missing-catalog fallback, guarded context/accepted-error behavior, and the local legacy version preflight are documented Go choices. Inherited mutable Resource/session/cache APIs remain separately tracked in the SDK scope rather than missing arguments of this declaration.
- The fixed success policy accepts final HTTP200 through399. Unlike the source's literal status<400 comparison, final informational1xx responses are rejected; normal net/http informational handling and source session redirects are not exact Python transport parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
result, err := conn.DeleteFloatingIP(ctx,
	compute.DeleteFloatingIPRequest{ID: ipID},
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
	id := flag.String("id", "", "floating IP ID to delete")
	retries := flag.Int("retry", 1, "additional deletes; zero skips verification")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	direct := flag.Bool("direct", false, "use direct GET for a UUID verification ID")
	flag.Parse()
	if *id == "" {
		return errors.New("-id is required")
	}

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
		compute.WithFloatingIPDeleteRetries(*retries),
		compute.WithFloatingIPDeleteDirectGet(*direct),
		compute.WithFloatingIPDeleteTimeout(time.Minute),
	}
	if *source != "" {
		opts = append(opts, compute.WithFloatingIPDeleteSource(compute.FloatingIPSource(*source)))
	}
	result, deleteErr := conn.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: *id}, opts...)
	var summary map[string]any
	if result != nil {
		accepted := 0
		for _, attempt := range result.Attempts {
			if attempt.Accepted {
				accepted++
			}
		}
		summary = map[string]any{
			"id": result.ID, "backend": result.Backend,
			"deleted": result.Deleted, "absent": result.Absent, "down": result.Down,
			"attempts": len(result.Attempts), "accepted_attempts": accepted,
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(deleteErr, encoder.Encode(summary))
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
deleted = conn.delete_floating_ip("IP_ID")
accepted = conn.delete_floating_ip("IP_ID", retry=0)
```

## References

- [Floating Ip Queries](floating-ip-queries.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [public retry/verification](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L977-L1017)
- [Neutron/Nova delete helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1019-L1046)
- [ Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L148-L201)
- [Available](floating-ip-available.md)
- [IP helper](server-ip-helpers.md)
- [CreateFloatingIP](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
