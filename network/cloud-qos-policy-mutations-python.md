# Cloud QoS Policy Mutations Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateQoSPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindCloudMutationRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.UpdateCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.PolicyRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMutationJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyName`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyDescription`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyShared`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyDefault`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyProjectID`

- Cloud valid_kwargs name/description/shared/default/project_id are validated before any extension HTTP. Complete JSON null/presence and concrete options are library-owned, nil/unknown cloud keys including is_shared/is_default are rejected with HTTP0 and later options overwrite prior keys from owned snapshots.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationValidKwargsBeforeGateAndOwnedOptions`.
- Every valid call consumes a fresh full qos extension discovery; every nonnull default including false/0/[]/{} triggers an independent fresh full qos-default gate. Supported default maps raw to is_default; unavailable default is removed without hiding gate errors, with completed Extensions evidence and DefaultDropped separately retained.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyCreateDefaultNonnullFreshGatesAndEmptyPOST`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyUpdateEmptyAfterDefaultSkipsLookupAndMissingRaises`.
- New cloud Source mutation lane uses selected original service/API/source and header snapshots captured before option callbacks. Source100..399 completion preserves status199. Clean503 retry and401 token renewal remain possible; changed route aborts renewal/retry with original rejected evidence. Prior typed/native profiles are unchanged.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationSourceStatusPolicyCleanRetryAndImmutableHeaders`, `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset`.
- POST/PUT rejected400/403/404/409/503 retain actual status/body/header and native value ErrUnexpectedResponseCode rather than fabricating Resource/Wire or swallowing404. Accepted/rejected200/401/404/503 read+Close/cancel/source fault matrix retains actual physical evidence and causes with zero retry/reauth callbacks. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal (value native response proof) and ErrReauth despite native absence of Unwrap.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationRejectedStatusPreservesActualEvidence`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationPhysicalFaultsPreserveActualEvidenceWithoutHooks`.
- After default handling, empty kwargs returns Source None without Find/PUT (including default:null and unavailable nonnull default). Nonempty attrs use default ignore_missing=True Source Find; absent policy raises SDK not-found with no PUT, retaining completed extension/lookup stage evidence.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyUpdateEmptyAfterDefaultSkipsLookupAndMissingRaises`.
- Dirty comparison uses private raw Source Body presence rather than coerced/default/alias getters: missing name→null is dirty; tenant_id-only→same project_id is dirty; shared:"false"→true is dirty; shared:true→1 is equal. Only changed keys enter PUT, partial response seed retains old/new raw state, and equal update returns original private receipt without invented PUT proof.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyUpdateRawDirtyMissingAliasAndOnlyChangedPayload`, `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationListConstructorRawLedgerAndPartialResponseSeed`.
- GET response raw Body and LIST constructor canonical/wire consumption have separate private ledgers. Borrowed seed is bound to original API/client/source guard and cloned private evidence; caller mutations to exported Resource/Wire/Header/Envelope or Clone cannot create false dirtiness or forge transport provenance. Another API or changed original route rejects before PUT.
  Tests: `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationBorrowedLedgerIgnoresExportedTamperingAndBindsOwner`, `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationListConstructorRawLedgerAndPartialResponseSeed`.
- Source _update resets microversion to None when no dirty PUT occurs; successful dirty PUT restores selected commit version. Actual unknown response microversion stays in Wire and cannot shadow Resource state.
  Tests: `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset`.

Python/Go differences:

- Concrete Go result preserves Policy, Lookup, Extensions, NoOp and DefaultDropped rather than Python Resource/None only. Wire/Envelope/Header/StatusCode remain actual observed response evidence; a prepared seed/no-op is not represented as a server PUT response.
- The new Source mutation lane preserves legacy typed/native defaults and policies. Complete UTF8 JSON/concrete owned options cover the named kwargs domain; custom runtime objects and mutable Python Resource/class/adapter/session/cache/current_location remain SDK-R1/SDK-C1/SDK-S1 shared foundation boundaries, with supported increment 0 and no generic full-SDK claim.
- Go source-relative URI guards escape returned member identity as one segment. Null/false/0/underflow/empty dirty member ID is rejected before PUT; Python null raises InvalidRequest but falsey nonnull URL joining can target a collection path. Go does not issue collection PUT from such a member seed. Equal updates do not validate or require a nonnull ID before returning no-op.
- JSON object wire order is not guaranteed. Source constructor conversion reads raw row insertion order, which is retained for declared/wire alias ledger conversion; dirty payload contains only changed keys and Python-loaded numeric equality preserves bool/int equality and CPython floating-point behavior without converting arbitrary integers through float64.
- Stock Neutron27 create_policy/update_policy are project ADMIN_OR_PROJECT_MANAGER, distinct from deprecated ADMIN_ONLY. Project_id field/cross-project privileges remain independent server validation (fixed neutron-lib3.21.0 reference); no mock/race/Source/main result proves authenticated live deployment/backend/driver acceptance.

### `python:cloud/_network/NetworkCloudMixin/create_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateQoSPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.CreateCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMutationJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyName`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyDescription`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyShared`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyDefault`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyProjectID`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Cloud valid_kwargs name/description/shared/default/project_id are validated before any extension HTTP. Complete JSON null/presence and concrete options are library-owned, nil/unknown cloud keys including is_shared/is_default are rejected with HTTP0 and later options overwrite prior keys from owned snapshots.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationValidKwargsBeforeGateAndOwnedOptions`.
- Every valid call consumes a fresh full qos extension discovery; every nonnull default including false/0/[]/{} triggers an independent fresh full qos-default gate. Supported default maps raw to is_default; unavailable default is removed without hiding gate errors, with completed Extensions evidence and DefaultDropped separately retained.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyCreateDefaultNonnullFreshGatesAndEmptyPOST`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyUpdateEmptyAfterDefaultSkipsLookupAndMissingRaises`.
- New cloud Source mutation lane uses selected original service/API/source and header snapshots captured before option callbacks. Source100..399 completion preserves status199. Clean503 retry and401 token renewal remain possible; changed route aborts renewal/retry with original rejected evidence. Prior typed/native profiles are unchanged.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationSourceStatusPolicyCleanRetryAndImmutableHeaders`, `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset`.
- POST/PUT rejected400/403/404/409/503 retain actual status/body/header and native value ErrUnexpectedResponseCode rather than fabricating Resource/Wire or swallowing404. Accepted/rejected200/401/404/503 read+Close/cancel/source fault matrix retains actual physical evidence and causes with zero retry/reauth callbacks. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal (value native response proof) and ErrReauth despite native absence of Unwrap.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationRejectedStatusPreservesActualEvidence`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationPhysicalFaultsPreserveActualEvidenceWithoutHooks`.
- Create always POSTs the translated cloud attrs even empty {}, without requiring name or sending synthesized Body defaults. Explicit null and raw default/is_default values remain raw in the request; actual partial response updates the Source seed while Wire remains actual response only.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyCreateDefaultNonnullFreshGatesAndEmptyPOST`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationValidKwargsBeforeGateAndOwnedOptions`.
- Actual unknown response microversion cannot shadow selected Source create version: Resource view exposes selected version while Wire keeps the original actual member. Cloud helper uses library-owned descriptor/default materialization and preserves selected header/version protocol.
  Tests: `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset`.

Python/Go differences:

- Concrete Go result preserves Policy, Lookup, Extensions, NoOp and DefaultDropped rather than Python Resource/None only. Wire/Envelope/Header/StatusCode remain actual observed response evidence; a prepared seed/no-op is not represented as a server PUT response.
- The new Source mutation lane preserves legacy typed/native defaults and policies. Complete UTF8 JSON/concrete owned options cover the named kwargs domain; custom runtime objects and mutable Python Resource/class/adapter/session/cache/current_location remain SDK-R1/SDK-C1/SDK-S1 shared foundation boundaries, with supported increment 0 and no generic full-SDK claim.
- Go source-relative URI guards escape returned member identity as one segment. Null/false/0/underflow/empty dirty member ID is rejected before PUT; Python null raises InvalidRequest but falsey nonnull URL joining can target a collection path. Go does not issue collection PUT from such a member seed. Equal updates do not validate or require a nonnull ID before returning no-op.
- JSON object wire order is not guaranteed. Source constructor conversion reads raw row insertion order, which is retained for declared/wire alias ledger conversion; dirty payload contains only changed keys and Python-loaded numeric equality preserves bool/int equality and CPython floating-point behavior without converting arbitrary integers through float64.
- Stock Neutron27 create_policy/update_policy are project ADMIN_OR_PROJECT_MANAGER, distinct from deprecated ADMIN_ONLY. Project_id field/cross-project privileges remain independent server validation (fixed neutron-lib3.21.0 reference); no mock/race/Source/main result proves authenticated live deployment/backend/driver acceptance.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "update"), default="create")
parser.add_argument("--identity", default="")
parser.add_argument("--attrs", default="{}")
args = parser.parse_args()
if args.mode == "update" and not args.identity:
    parser.error("update identity is required")
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        value = conn.create_qos_policy(**attrs)
    else:
        value = conn.update_qos_policy(args.identity, **attrs)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
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

func printJSON(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "create", "create or update")
	identity := flag.String("identity", "", "update: literal ID or exact name")
	attrs := flag.String("attrs", "{}", "JSON cloud mutation kwargs")
	flag.Parse()
	if *mode != "create" && *mode != "update" {
		log.Fatal("mode must be create or update")
	}
	if *mode == "update" && *identity == "" {
		log.Fatal("update identity is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var result *network.QoSPolicyMutation
	option := network.WithQoSPolicyMutationJSON(json.RawMessage(*attrs))
	if *mode == "create" {
		result, err = service.CreateQoSPolicyRecord(ctx, option)
	} else {
		result, err = service.UpdateQoSPolicyRecord(ctx, *identity, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.Policy == nil {
		printJSON(nil)
		return
	}
	printJSON(result.Policy.Resource)
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L90-L134)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L28-L46)
- [Attributes](https://github.com/openstack/neutron-lib/blob/3.21.0/neutron_lib/api/attributes.py#L24-L56)
- [QoS](https://github.com/openstack/neutron-lib/blob/3.21.0/neutron_lib/api/definitions/qos.py#L69-L76)
- [Install](../docs/install.md)
- [cloud create/update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1432-L1510)
- [Neutron Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5080-L5210)
- [QoSPolicy descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py)
- [Resource dirty/create/commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
