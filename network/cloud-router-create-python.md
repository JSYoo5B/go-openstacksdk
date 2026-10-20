# Cloud Router Create Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/create_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateRouterRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.CreateCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateName`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateAdminStateUp`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateExternalGatewayNetworkID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateEnableSNAT`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateExternalFixedIPs`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateAvailabilityZoneHints`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateRouterRecord`

- routers.API.CreateWithAttributes sends only explicit attributes: no arguments produce {"router":{}}, and admin_state_up plus a caller-built external_gateway_info are sent as given.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkCreateRouterSubnetSendOnlyExplicitAttributes`.
- Cloud default admin_state_up=True; raw null/false/nonstring values stay in request. Name/net/fixed-IPs truthiness differs from project/enable_snat nonNone presence; zero/empty project survives, enable_snat:false alone builds gateway. Known7kwargs only; options/JSON/Opts/CurrentLocation are SDK-owned snapshots, no app builder required.
  Tests: `network/cloud_router_create_test.go:TestCloudRouterCreateDefaultAndSourceJSONPresenceTruthiness`, `network/cloud_router_create_test.go:TestCloudRouterCreateGatewayAssemblyAndOwnedConcreteOptions`, `network/cloud_router_create_test.go:TestCloudRouterCreatePreflightCancellationAndSelectedSourceGuards`.
- Omitted/null AZ hints send no discovery; nonnull must be list before any HTTP. Empty list and nonstring list elements remain valid Source input. Every requested hint call fully consumes fresh router_availability_zone inventory including later aliases/pages; failure has actual extension proof but no fabricated POST receipt.
  Tests: `network/cloud_router_create_test.go:TestCloudRouterCreateAZListValidationThenFreshCompleteGate`, `network/cloud_router_create_test.go:TestCloudRouterCreateAZMissingLateAliasAndPageFailuresPreserveOnlyGates`, `network/cloud_router_create_test.go:TestCloudRouterCreateAZPhysicalFailureHasExtensionProofAndNoPostReceipt`.
- One collection POST contains only translated cloud fields. Canonical admin seed is separate from actual wire admin_state_up; sparse response keeps input, explicit null overrides. Existing Router descriptors/MRO/list/dict/int/bool/default/header/tenant aliases are reused. Wire and whole receipt remain physical response only.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreateTranslatedBodySeedCanonicalAndSourceStatus`, `network/cloud_router_create_test.go:TestCloudRouterCreateResponseSourceDefaultsAliasesAndOwnedLocation`, `network/cloud_router_create_test.go:TestCloudRouterCreateRejectedStatusesAndInvalidResponsesKeepActualReceipt`.
- Source100..399 acceptance opts in only new leaf; existing native/shared typed baseline is unchanged. Selected microversion and owned null/current location replace response members only in Resource; current foreign project metadata is computed without extra API requests. Response mutation cannot alias independent Wire evidence.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreateTranslatedBodySeedCanonicalAndSourceStatus`, `network/cloud_router_create_test.go:TestCloudRouterCreateResponseSourceDefaultsAliasesAndOwnedLocation`, `network/cloud_router_create_test.go:TestCloudRouterCreateCleanRetryTokenRenewalAndHeaderVersionCapture`.
- Actual rejected400/403/404/409/503 retain native POST error/status/body/header. Invalid response shapes error with actual accepted receipt; nonJSON/empty response retains Source seed without fabricated Wire. Nil/invalid cloud attrs/context/cancel/selected API/source fail before HTTP.
  Tests: `network/cloud_router_create_test.go:TestCloudRouterCreateRejectedStatusesAndInvalidResponsesKeepActualReceipt`, `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreatePreflightAndResponseShapeErrors`, `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreateNativeRejectionsAndStickyRestoredSourceProof`, `network/cloud_router_create_test.go:TestCloudRouterCreatePreflightCancellationAndSelectedSourceGuards`.
- Physical accepted/rejected200/401/404/503 read/Close faults, custom cancel and source change preserve actual receipts/all causes with zero retry/reauth hooks. Native401 value ErrOriginal and ErrReauth remain available. Restored source cannot erase observed mismatch; clean503 retry and401 token renewal preserve owned route/body/headers. Changed route aborts before resend with original503 receipt.
  Tests: `network/cloud_router_create_test.go:TestCloudRouterCreatePhysicalReadCloseCancellationSourceAndNative401NoHooks`, `network/cloud_router_create_test.go:TestCloudRouterCreateAZPhysicalFailureHasExtensionProofAndNoPostReceipt`, `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreateNativeRejectionsAndStickyRestoredSourceProof`, `network/cloud_router_create_test.go:TestCloudRouterCreateCleanRetryTokenRenewalAndHeaderVersionCapture`.
- Connection supplies cached Neutron and an owned default current location without caller builders; explicit location remains selectable and invalid/cancelled calls do not discover a service.
  Tests: `connection_cycle7_records_test.go:TestConnectionRouterCreateOwnsDefaultLocationAndCachedService`, `connection_cycle7_records_test.go:TestConnectionCycle7PreflightPreservesCancellationBeforeDiscovery`.

Python/Go differences:

- Concrete Go RouterCreateResult exposes owned Router/Wire/Envelope/Header/StatusCode and independently observed Extensions, unlike only mutable Python Router. Cloud7kwargs are complete UTF8 JSON; application builders are not required. Failed extension GET cannot fabricate POST evidence.
- Source float serialization/runtime domain, custom Python runtime objects, mutable Resource/class/adapter/session/API cache/current_location and dynamic class dispatch remain shared SDK-R1/SDK-C1/SDK-S1 foundation differences. Their supported increment remains0; this one cloud mapping does not promote general Proxy raw attrs create, update/delete or full SDK parity.
- Go preserves selected source/API/header/version snapshots and context cancellation. JSON object wire order is not guaranteed. Safe literal service-relative URI/protocol policy remains shared; router POST has no caller ID path segment to normalize. Invalid values fail preflight rather than silently inventing IDs/defaults.
- Standalone location defaults null; explicit owned CurrentLocation can be injected from Connection.CurrentLocation without HTTP, and never enters POST. Existing Network/Subnet role cache and mutation notifications are preserved; source-wide URL cache invalidation is not claimed closed.
- Neutron27 stock create_router and gateway/network_id use project ADMIN_OR_PROJECT_MEMBER; enable_snat/external_fixed_ips field specification requires ADMIN, including explicit false. Deprecated historic rules differ. SDK/test/runtime evidence does not assert real deployment authorization, role, ownership, driver or server acceptance.
- Actual imported pinned cloud create_router/_build_external_gateway_info + Neutron Proxy._create + Resource.new/create/translate/descriptors ran on Python3.14.8 for90cases. Only Adapter.post and extension iterator were fixture substitutions. stock27 router policy SHA2569c7436013df00879c69d0821f6fd4b127186efa7dc3774b6a0e2bdfa3d29aefa matched official tagged primary source. This is not live OpenStack acceptance.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kwargs", default="{}")
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    router = conn.create_router(**kwargs)
    print(json.dumps(router.to_dict(), indent=2))
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
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kwargs := flag.String("kwargs", "{}", "JSON create_router kwargs")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	result, err := conn.CreateRouterRecord(ctx,
		network.WithRouterCreateJSON(json.RawMessage(*kwargs)),
	)
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(result.Router, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L57-L145)
- [cloud create_router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2250-L2307)
- [gateway helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2882-L2901)
- [Router descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
