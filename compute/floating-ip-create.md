# Floating Ip Create

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network_common/NetworkCommonCloudMixin/create_floating_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.Allocate`
- `github.com/JSYoo5B/go-openstacksdk/compute.PrepareFloatingIPCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreatePort`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateFixedAddress`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateNATDestination`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateWait`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateWaitTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateWaitInterval`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateSource`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateStrict`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateDirectGet`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPCreateTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUnlimitedFloatingIPCreateTimeout`

- Concrete preparation defaults to Wait=false, a60-second post-allocation WaitTimeout, automatic wait interval and no SDK whole-operation cap. Last-option precedence permits a zero-valued bulk replacement to clear budgets; negative wait timeout is retained for a consumed immediate timeout rather than globally rejecting ignored destination values.
  Tests: `compute/floating_ip_create_options_test.go:TestFloatingIPCreateOptionsDefaultsBulkAndIgnoredSelectors`.
- Fresh Neutron creation resolves an ordinary network by member then name, without requiring router:external=true, and posts only floating_network_id plus consumed port/fixed fields. The finite lower fixtures accept200/201/202/299/399 and retain actual status/header plus an exact large-number unknown field; they do not inspect or reuse a free-IP inventory.
  Tests: `network/floating_ip_allocate_test.go:TestFloatingIPAllocateOrdinaryNetworkLookupAndFreshBody`.
- A nonempty explicit Neutron port bypasses even an invalid Server ref, FixedAddress and NATDestination, and invokes no named-server resolver. Its tested flat-object response is admitted. Cloud logical request seeds supply only missing network/port values while actual Wire stays unseeded, and Allocation/FloatingIP logical data do not alias.
  Tests: `network/floating_ip_allocate_test.go:TestFloatingIPAllocateExplicitPortIgnoresOtherDestinations`, `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreateNeutronFreshAndRequestSeedOwnership`.
- When a Server destination is consumed, no ports or an unmatched explicit fixed literal produce an unattached allocation. A matching IPv6 literal is sent unchanged. Without an explicit fixed address, multiple ports are narrowed by NAT and the latest created_at port supplies the first parseable IPv4; malformed automatic fixed-IP items are skipped before a valid address.
  Tests: `network/floating_ip_allocate_test.go:TestFloatingIPAllocateOptionalServerAndLiteralFixedSelection`.
- Complete ports discovery precedes mutation: the tested late403, missing explicit NAT and no usable IPv4 paths send no allocation POST. Default external selection uses the shared role candidate or, when unavailable, the first enabled router gateway, skipping a disabled router.
  Tests: `network/floating_ip_allocate_test.go:TestFloatingIPAllocateLatePortsAndNATFailurePreventMutation`, `network/floating_ip_allocate_test.go:TestFloatingIPAllocateDefaultRoleAndRouter`.
- Without Server or explicit port, FixedAddress/NATDestination and Wait=true with a negative wait timeout are ignored. This accepted unattached allocation sends no poll or cleanup; a missing response ID is logical null while Wire preserves actual absence.
  Tests: `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreateUnattachedIgnoresWaitAndFixedNAT`.
- For configured Nova and None, nil Network reads the first pool, while explicit empty or named Network is a literal pool with no pool query. Creation is fresh POST followed by the mandatory raw member compatibility GET of numeric ID7; ignored Server/fixed/NAT/wait values do not trigger attachment or readiness work. The tested response permits null address and a different pool/attached instance rather than imposing assignment-candidate validation.
  Tests: `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreateNovaFreshCompatibilityAndPoolPresence`.
- The final Nova compatibility response is normalized through the shared cloud normalizer, not the fresh allocation candidate verifier. The new HTTP tests keep wire ERROR and exact vendor numbers beside source-synthetic logical ACTIVE. Existing shared pure normalization assertions cover canonical/null alias precedence, strict/non-strict aliases, properties/location ownership and configured-Neutron mode after Nova fallback; this is composition evidence, not additional new Create fixtures.
  Tests: `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreateNovaFreshCompatibilityAndPoolPresence`, `compute/floating_ip_query_normalize_test.go:TestFloatingIPQueryNormalizationCanonicalNullPropertiesAndStrictAliases`, `compute/floating_ip_query_normalize_test.go:TestFloatingIPQueryNormalizationUsesConfiguredNeutronAfterNovaFallback`.
- Pure Neutron NotFound before accepted allocation can fall through from missing ordinary-network lookup or allocation POST to Nova. The original error and configured-Neutron normalization remain visible. A nonempty explicit port instead returns ErrUnsupported before any Nova allocation, and the tested fallback discovery is Network then Compute.
  Tests: `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreatePreacceptNotFoundFallbackAndPortPolicy`, `connectiontest/connection_floating_ip_create_boundaries_test.go:TestFloatingIPCreatePostNotFoundCanFallbackBeforeAcceptance`.
- An accepted Neutron POST202 with malformed JSON, Close error, custom caller cancellation or cached Network endpoint drift retains Allocated=true and actual allocation status/header/envelope with ResponseError202. It sends no wait, cleanup, compatibility GET or alternate allocation; tested Close/cancel/source causes remain observable.
  Tests: `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreateAcceptedDecodeCloseCancelAndSourceFailures`.
- Nova compatibility403 or an accepted malformed compatibility model preserves the initial accepted allocation and the actual compatibility failure. Accepted Nova POST with null or object-valued ID retains raw Allocation and ResponseError200 but performs no unsafe compatibility request, cleanup or concealed reallocation.
  Tests: `connectiontest/connection_floating_ip_create_boundaries_test.go:TestFloatingIPCreateNovaCompatibilityAndUnsafeIDKeepAllocation`.
- Present null, false or different response port overrides the request seed. No-wait creation returns an error beside its actual accepted201 receipt and unchanged response port in Wire/logical Resource; no cleanup occurs.
  Tests: `connectiontest/connection_floating_ip_create_boundaries_test.go:TestFloatingIPCreateResponsePortOverridesSeed`.
- With a selected port and Wait=true, the ordinary public-Get loop polls immediately, continues past an empty list and logical ERROR, accepts JSON-semantic exact ACTIVE, then verifies the selected port. The successful fixture retains three observations; lookup403 or final port mismatch returns a known allocation without cleanup, Waited success or Nova compatibility.
  Tests: `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreateWaitPublicGetSkipsErrorAndChecksFinalPort`.
- Direct UUID wait uses the public member GET. A member404 after an accepted Neutron allocation retains that allocation and observation, with no Nova fallback, second allocation or timeout cleanup. This is the deliberate post-acceptance safety boundary.
  Tests: `connectiontest/connection_floating_ip_create_boundaries_test.go:TestFloatingIPCreateDirectWait404DoesNotAllocateAgain`.
- A consumed zero/negative or real20-ms SDK wait timeout returns FloatingIPCreateTimeoutError with the allocated ID and DeadlineExceeded. Cleanup runs one default-policy DELETE and public absence verification on the still-live original context rather than the expired wait child. Successful absence and cleanup403 are separate results; original lookup/POST and cleanup DELETE have no SDK wait-child deadline.
  Tests: `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreateSDKWaitTimeoutCleansWithLiveParent`.
- Caller custom cancellation during a public wait remains Canceled with its cause and accepted allocation, without detached cleanup. Expiry of a20-ms overall cap is DeadlineExceeded but not FloatingIPCreateTimeoutError and sends no cleanup; the SDK distinguishes this earlier budget from its own post-allocation wait timer.
  Tests: `connectiontest/connection_floating_ip_create_test.go:TestFloatingIPCreateCallerCancellationNeverDetachesCleanup`, `connectiontest/connection_floating_ip_create_boundaries_test.go:TestFloatingIPCreateParentDeadlineIsNotSDKWaitTimeout`.
- Bulk factories own Source/Location pointers and project raw JSON; returned policy edits do not mutate retained factory data. Original option callbacks run once. Preparation rejects nil options, invalid source, nonpositive dedicated overall timeout/interval and negative bulk interval; a pre-cancelled context preserves its cause and invokes no callback.
  Tests: `compute/floating_ip_create_options_test.go:TestFloatingIPCreateOptionsOwnSnapshotsAndApplyOnce`, `compute/floating_ip_create_options_test.go:TestFloatingIPCreateOptionsPreflightAndCancellation`.
- A direct Compute Service captures its source before original callbacks and snapshots request Network before a callback changes the caller string. The tested earlier parent deadline remains exact on lookup/allocation, and a subsequent Connection call reuses already selected Compute/Network dependencies without another locator. Clearing the captured Servers collection in an original option fails before HTTP.
  Tests: `connectiontest/connection_floating_ip_create_boundaries_test.go:TestFloatingIPCreateWholeBudgetSnapshotAndCachedService`, `connectiontest/connection_floating_ip_create_boundaries_test.go:TestFloatingIPCreateOptionsCannotReplaceCapturedService`.
- The initial cloud allocation success policy cannot be expanded by a native retry callback: expanded404 remains a terminal failure after two physical POST attempts, without allocation acceptance or Nova fallback. Changing the cached endpoint in that callback stops before the second physical POST and retains ErrInvalidOption; original callbacks are invoked once.
  Tests: `connectiontest/connection_floating_ip_create_boundaries_test.go:TestFloatingIPCreateNativeRetryCannotExpandCodesOrChangeSource`.

Python/Go differences:

- Only the named cloud create_floating_ip declaration is mapped: fresh Neutron/Nova allocation, consumed Neutron destination/wait/port checks, Nova compatibility fetch and SDK-wait-timeout cleanup. Independent availability, DeleteUnattached, attach/detach, native/Proxy operations and inherited Resource/session/cache APIs retain their own tracking and are not requirements missing from this signature.
- Go replaces arbitrary supplied Python Server/Resource/Munch objects with safe resource.Ref ID/name selectors and concrete option factories. Typed string fixed/NAT/port inputs, owned JSON views/actual Wire and structured allocation/compatibility/observation/cleanup evidence are deliberate API mappings rather than Python object identity or exception/return-value equivalence.
- Network pointer presence preserves Nova None versus empty pool while Neutron nil/empty both select defaults. Source None explicitly creates through Nova. Typed invalid path IDs and unknown source/interval/budget policies return local errors; ignored destination values are validated only on their consuming branch. Dynamic Python numeric/object inputs, unsafe/null routing and timeout=None TypeError corner are not exposed.
- Neutron Resource.create request seeds are projected only into missing returned keys and never actual Wire. Flat object and named object responses are supported; malformed JSON, wrong-shaped envelopes and unusable executable IDs retain accepted proof and fail rather than reproducing Python Resource malformed-body swallowing or unsafe URL stringification. Present response null remains authoritative.
- Python catches Neutron helper NotFound even after accepted allocation and may allocate again through Nova. Go permits only pure pre-acceptance absence fallback and rejects mixed/terminal source/context/processing/native retry failures; post-acceptance errors cannot hide a second allocation. This intentional mutation-safety mapping is documented and regression tested.
- Go uses a library-owned child context for the post-allocation60-second wait, with optional positive interval and separate whole-operation cap. It interrupts in-flight HTTP and performs a final guard rather than copying Python wall-clock yield/sleep late-success behavior. Only its own wait-timeout cause with a live original guarded context triggers cleanup; parent/whole timeout or caller cancellation never detaches deletion.
- Timeout cleanup reuses the mapped public Delete default retry1 and ordinary Get semantics. Python logs cleanup exceptions and rethrows only ResourceTimeout; Go retains Cleanup/CleanupError and joins the original typed timeout, transport and context causes. DOWN policy success is not actual absence, and no unrelated cleanup operation is implied.
- Legacy Nova is checked against the selected Compute microversion below2.36 without a hidden downgrade. Its final view uses the configured cloud normalization mode: direct Nova/None may synthesize logical ACTIVE while actual Wire remains unchanged, and configured-Neutron fallback remains Neutron mode. This function does not promise real Nova lifecycle/readiness or full endpoint/version negotiation.
- Automatic destination selection keeps source-style created_at lexical reverse order and the first valid IPv4 while explicit fixed selection is literal and can be IPv6. Typed parsing and stable string order replace arbitrary Python malformed or mixed created_at/object comparison corners. Ordinary canonical HTTP/model behavior is mapped without reproducing every dynamic Resource descriptor/runtime case.
- Owned allocation/compatibility transport accepts final HTTP200 through399, preserving actual status/body/header and guarded native retries. Source literal status<400 also includes informational1xx; normal net/http informational handling and redirect/session combinations are deliberate transport boundaries, not claims of exact Python session behavior. Standard query/lookup reads retain their documented typed status/envelope/paging rules.
- Go exposes explicit per-call source/location/strict/direct-get options and existing immutable Connection configuration. Generic Python has_service negotiation, arbitrary mutable session/profile overrides and inherited Resource/cache/discovery facilities remain separate SDK goal scope, rather than an unimplemented declared Create argument.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
networkName := "public"
result, err := conn.CreateFloatingIP(ctx,
	compute.CreateFloatingIPRequest{Network: &networkName},
	compute.WithFloatingIPCreateServer(resource.ID(serverID)),
	compute.WithFloatingIPCreateWait(true),
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
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	networkName := flag.String("network", "", "network name/ID or literal Nova pool")
	serverID := flag.String("server", "", "optional server ID")
	portID := flag.String("port", "", "optional explicit Neutron port ID")
	fixed := flag.String("fixed", "", "optional fixed IP literal")
	nat := flag.String("nat", "", "optional NAT destination name/ID")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	wait := flag.Bool("wait", false, "wait for selected Neutron port allocation")
	waitTimeout := flag.Duration("wait-timeout", time.Minute, "post-allocation wait timeout")
	timeout := flag.Duration("timeout", 0, "optional whole SDK operation cap")
	flag.Parse()

	input := compute.CreateFloatingIPRequest{}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "network" {
			input.Network = networkName
		}
	})
	opts := []compute.FloatingIPCreateOption{
		compute.WithFloatingIPCreatePort(*portID),
		compute.WithFloatingIPCreateFixedAddress(*fixed),
		compute.WithFloatingIPCreateNATDestination(*nat),
		compute.WithFloatingIPCreateWait(*wait),
		compute.WithFloatingIPCreateWaitTimeout(*waitTimeout),
	}
	if *serverID != "" {
		opts = append(opts, compute.WithFloatingIPCreateServer(resource.ID(*serverID)))
	}
	if *source != "" {
		opts = append(opts, compute.WithFloatingIPCreateSource(compute.FloatingIPSource(*source)))
	}
	if *timeout != 0 {
		opts = append(opts, compute.WithFloatingIPCreateTimeout(*timeout))
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
	result, createErr := conn.CreateFloatingIP(ctx, input, opts...)
	var summary map[string]any
	if result != nil {
		summary = map[string]any{
			"backend": result.Backend, "allocated": result.Allocated,
			"waited": result.Waited, "observations": len(result.Observations),
			"compatibility_attempted": result.Compatibility != nil,
		}
		if result.AllocationResponse != nil {
			summary["allocation_status"] = result.AllocationResponse.StatusCode
		}
		if result.Cleanup != nil {
			summary["cleanup_deleted"] = result.Cleanup.Deleted
			summary["cleanup_absent"] = result.Cleanup.Absent
			summary["cleanup_down"] = result.Cleanup.Down
			summary["cleanup_attempts"] = len(result.Cleanup.Attempts)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(createErr, encoder.Encode(summary))
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
ip = conn.create_floating_ip(network="public")
attached = conn.create_floating_ip(
    network="public", server={"id": "SERVER_ID"}, wait=True, timeout=60
)
# explicit port wins over server/fixed/NAT in the pinned implementation
on_port = conn.create_floating_ip(network="public", port="PORT_ID", wait=True)
```

## References

- [AvailableFloatingIP](floating-ip-available.md)
- [IP helper](server-ip-helpers.md)
- [Network Ensure](../network/floating-ip-ensure.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [DeleteFloatingIP](floating-ip-delete.md)
- [public/backend selection](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L775-L842)
- [Neutron create/wait/cleanup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L850-L949)
- [Nova POST/compat GET](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
