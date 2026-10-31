# Cloud Subnet Create Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/create_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateSubnetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateSubnetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.CreateCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithCloudCreateOrder`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.FindSubnetSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.FindSubnetSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateCIDR`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateIPVersion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateDHCP`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateName`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateTenantID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateGatewayIP`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateDisableGateway`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateAllocationPools`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateDNSNameservers`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateHostRoutes`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateIPv6RAMode`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateIPv6AddressMode`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreatePrefixLength`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateDefaultSubnetPool`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateSubnetPool`

- subnets.API.CreateWithAttributes sends only explicit attributes, so ip_version, enable_dhcp and a null gateway_ip appear only when the caller sets them.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkCreateRouterSubnetSendOnlyExplicitAttributes`.
- Library-owned concrete options accept Source formal keywords and arbitrary extension kwargs, snapshot JSON/maps/ordered aliases/location/client before callbacks, own defaults ip_version=4 and enable_dhcp=False, and require no application builder/interface. Positional network/receiver duplicates, nil options and malformed complete UTF8 JSON fail before HTTP. JSON/individual options preserve input order; map-only order is deterministic lexical.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateDefaultsOwnedKeywordsRawBodyAndIndependentEvidence`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreatePreflightBoundArgumentsCancellationAndImmutableService`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateOwnedLocationUsesActualProjectsWithoutForgedWire`.
- Default network lookup occurs before gateway/CIDR/pool/IP semantic validation. A tenant_id that is not None/null remains a raw member query including false/zero/empty string. New fixed Source helpers opt into valid100..399, Source root-links precedence, loaded JSON policy and physical-fault callback suppression without altering baseline FindRecord/native/list bindings. Only clean400/403/404 fall back to all-page exact ID/name find; duplicate/late-page errors never POST or accept an early match.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateNetworkLookupPrecedesAllSemanticValidation`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateFalseyFormalsTenantPresenceAndGatewayNull`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateDefaultLookupsConsumeWholePagesAndKeepPartialFailure`.
- Gateway truthy conflict is checked before CIDR/pool requirements and mutually-exclusive default/named pool selection. Named pool lookup precedes string ip_version int conversion; missing pool masks later invalid IP as in Source. Unicode decimal digits/signs/spaces/underscores use the library PythonIntegerString, while nonstring None/bool/list/float retains raw dirty JSON. Required network/pool data is taken from the returned Source getter rather than caller label.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateSubnetPoolLookupThenPythonStringIPConversion`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateNetworkLookupPrecedesAllSemanticValidation`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateFalseyFormalsTenantPresenceAndGatewayNull`.
- Body starts from network_id/ip_version/DHCP defaults, applies arbitrary kwargs, then truthy formal optional fields in Source order. Falsey formal fields omit; disable gateway forces explicit null; default pool forces true and named pool uses resolved ID. Raw bool/int/list dirty values remain uncoerced in request; only getter descriptors cast. Ordered canonical/wire aliases and response aliases use the final Source value, with partial/empty responses retaining seed only in Resource while Wire/Envelope stay actual.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateOrderedAliasesConflictRestorationAndRawDescriptorValues`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateResponseCanonicalAliasesDoNotLoseActualHeaderOrWire`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateDefaultsOwnedKeywordsRawBodyAndIndependentEvidence`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateOwnedTranslatedBodySeedAndSourceStatuses`.
- Proxy base_path is consumed before one-level truthy __conflicting_attrs restoration; falsey conflicting attrs remain ordinary unknown Body, and restored base_path remains Body while the fixed class getter is /subnets. Source constructor collisions are rejected after network lookup; microversion constructor does not override selected create version. Header aliases select the ordered winner before formatting/validation, including shadowed invalid lists, null=>None and string-list comma-space. Actual HTTP If-Match overrides request header seed; absent response header retains seed.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateSourceProxyControlsHeadersAndOriginalBasePath`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateResponseCanonicalAliasesDoNotLoseActualHeaderOrWire`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateOnlyWinningHeaderAliasIsValidated`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateHeaderSeedAndActualResponsePrecedence`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateBasePathAndRestoredUnknownClassGetter`.
- Lookup observations and subnet POST expose independently owned Resource/Wire/Envelope/Header/Status. Actual or foreign project IDs compute SDK-owned location without overwriting actual Wire location/microversion. Public Connection supplies current scope; standalone nil location is null. Malformed remote/body/header projection or late selected-source/cancellation errors retain actual accepted receipt as ResponseError and return no successful Resource, never a fabricated server acknowledgement.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateOwnedLocationUsesActualProjectsWithoutForgedWire`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateResponseCanonicalAliasesDoNotLoseActualHeaderOrWire`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreatePhysicalFaultsKeepActualLookupOrMutationAndStopHooks`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateHeaderSeedAndActualResponsePrecedence`.
- Actual GET/POST200/401/404/503 read+Close/cancel/source failures preserve response bytes/header/status and original causes with zero retry/reauth callbacks. Pointer native401 retains ErrOriginal value ErrUnexpectedResponseCode and ErrReauth despite absent Unwrap. Clean503 retry and401 token renewal remain supported; route changes stop before resend with original rejection receipt. Selected headers remain captured when callback mutates the original valid header map.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreatePhysicalFaultsKeepActualLookupOrMutationAndStopHooks`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateCleanRetryRenewalSource199AndFixedHeaderCapture`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateConstructorGetterFailuresAndNativeRejectedReceipts`.
- An actual accepted subnet POST100..399 invalidates the original private successful network role/subnet-gateway topology cache even when response decode/Read/Close/cancel/source handling subsequently fails. Rejected POST and pre-POST lookup/validation errors do not invalidate success cache. Replacing public Service.Roles does not redirect the private mutation cache target.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateAcceptedMutationInvalidatesOnlyPrivateSuccessfulRoleCache`.
- Connection supplies cached Neutron and an owned default location for lookup/create while the service invalidates only actual accepted topology mutations.
  Tests: `connectiontest/connection_cycle7_records_test.go:TestConnectionSubnetCreateSharesNetworkCacheAndLocation`, `connectiontest/connection_cycle7_records_test.go:TestConnectionCycle7PreflightPreservesCancellationBeforeDiscovery`.

Python/Go differences:

- This maps one fixed cloud create_subnet declaration with library-owned orchestration and returned owned Resource/Wire/actual receipt plus independent Network/SubnetPool observations. It does not promote native Create/CreateWithAttributes, old general FindRecord, arbitrary raw Proxy.Find/constructor/class/adapter/session/cache behavior or any additional Source declaration. Full SDK supported delta remains 0; shared SDK-R1/SDK-C1/SDK-S1 foundation remaining persists separately.
- Go network/named pool references are validated literal string IDs or exact names. Truthy nonstring/mutable Resource reference branches remain the shared Resource/class foundation; falsey pool values still skip lookup as in the fixed cloud flow. Go escapes slash/percent as a single route segment and rejects blank/control/traversal references. base_path accepts only literal service-relative routes, including empty endpoint-root route; Source arbitrary percent-template/query/external path formatting is not reproduced.
- Go exposes concrete With options, safe immutable snapshots, captured service/header/protocol guards, independent actual transport receipts and accepted-mutation topology cache invalidation instead of Python mutable Resource/Connection side effects. CurrentLocation is SDK-owned configured context; unknown raw location remains request/Wire data while the getter is null or computed scope. General dynamic class/member dispatch and arbitrary computed properties are not claimed by unknown JSON extension Body preservation.
- Ordered JSON and individual keyword options preserve Source insertion order; map-only remaining keys use Go lexical order, and JSON wire key order is not guaranteed. Raw JSON numeric tokens retain precision instead of first converting to float64; Python configurable int-string digit limit and generic JSON float/runtime-domain serialization remain explicit Go/foundation differences. No new semantic CIDR/IP/range/driver validation beyond the Source cloud rules is claimed.
- Stock Neutron27 create_subnet is project ADMIN_OR_NET_OWNER_MEMBER, with independent admin segment_id/service_types and separate tags policy. Readable/shared network or library lookup success does not prove write authorization, other-tenant/pool access or server/driver acceptance. Live authenticated OpenStack policy is not part of fixture/runtime/main/race proof.
- Actual imported pinned cloud get_network/get_subnetpool/create_subnet plus Neutron Proxy.find/create and Resource.find/new/create ran under CPython3.14.8 for 127 cases. Only HTTP adapter methods were substituted. This is not live OpenStack acceptance.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--network", required=True)
parser.add_argument("--kwargs", default='{"cidr":"10.61.0.0/24"}')
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    subnet = conn.create_subnet(args.network, **kwargs)
    print(json.dumps(subnet.to_dict(), indent=2))
```

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func run(ctx context.Context, cloud, networkName, keywords string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	result, err := conn.CreateSubnetRecord(ctx, networkName,
		network.WithSubnetCreateJSON(json.RawMessage(keywords)))
	if result != nil {
		if result.Network != nil {
			fmt.Fprintf(os.Stderr, "network lookup status=%d\n", result.Network.StatusCode)
		}
		if result.SubnetPool != nil {
			fmt.Fprintf(os.Stderr, "subnetpool lookup status=%d\n", result.SubnetPool.StatusCode)
		}
		if result.StatusCode != 0 {
			fmt.Fprintf(os.Stderr, "subnet POST status=%d request-id=%q\n",
				result.StatusCode, result.Header.Get("X-Request-Id"))
		}
	}
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result.Resource, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	networkName := flag.String("network", "", "literal network ID or exact name")
	keywords := flag.String("kwargs", `{"cidr":"10.61.0.0/24"}`, "Source cloud keyword JSON object")
	flag.Parse()
	if *networkName == "" {
		fmt.Fprintln(os.Stderr, "network is required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *cloud, *networkName, *keywords); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## References

- [Install](../docs/install.md)
- [Subnet](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnet.py#L60-L118)
- [cloud create_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2401-L2569)
- [Subnet descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet.py)
- [Proxy._create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L771-L800)
