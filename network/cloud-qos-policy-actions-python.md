# Cloud QoS Policy Actions Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/get_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindListBasePath`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Every cloud call completes fresh uncached qos extension discovery before options/member/deletion and preserves completed extension evidence. Unavailable extension has the shared sentinel and no policy HTTP. Context/source/selected private API guards remain active through dependent leaf requests and caller option snapshots.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot`.
- New Source action profile clones class Body/Query and opts Source100..399, link precedence, CPython-loaded equality and physical-stop policies in without changing prior typed API baselines. Actual accepted/rejected response receipts and native401 pointer ErrOriginal/value ErrUnexpectedResponseCode plus ErrReauth causes survive read/Close/cancel/source failures with zero retry/reauth and no missing/success fallback.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsPhysicalFaultsNeverFallbackMissingOrRetry`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`.
- Clean503 retry and401 live token renewal are retained for Get/DELETE while changed selected route before retry preserves the original503 receipt. Source status199 opt-in is separately exercised over the public testhelper HTTP fixture with a response observer; no resource/response is fabricated.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsCleanRetryRenewalAndRouteGuard`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`.
- Cloud Get default missing is a nil Policy and nil error. Falsey JSON filters become {} after the gate; truthy nonmapping filters fail as kwargs rather than being interpreted from docstring as JMES. Ordinary raw member params seed constructor fields and actual body overrides them, keeping explicit null/defaults/unknown values/private Wire/Header/Envelope evidence independent.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetDefaultMissingAndMemberEvidence`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot`.
- Clean400/403/404 member failures alone trigger complete exact ID/name list fallback with declared name hint and server/local Body classification. Raw microversion is GET-only; list_base_path is fallback-only and an unused foreign fallback route is ignored on successful member GET. CPython-loaded local equality and late duplicate ambiguity are handled by the owned library.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFallbackListPathVersionLocalFiltersAndLateAmbiguity`.
- After a complete fresh gate, cloud-bound name_or_id/ignore_missing kwargs collide by presence including null, while the six raw fetch controls are now consumed by the dedicated FindSourceRecord lane; ordinary legacy FindRecord retains its reviewed classifier.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot`.
- Each six-control cloud lookup fully repeats extension discovery before a collection/member policy GET. Policy descriptor/tenant alias/rules/defaults, unknown constructor controls and computed Resource.location=null are preserved in the Resource view; actual server location JSON remains in physical Wire/Envelope/status/header evidence.
  Tests: `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsFreshFullGateAndPolicyProfile`.
- Cloud filters raw headers are member GET mapping-key query values and fallback HTTP headers, with independent raw base_path/list_base_path and GET-only microversion. Invalid truthy headers fail only at fallback, after completed fresh gate evidence.
  Tests: `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsRawHeadersHaveSeparatePhases`, `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsInvalidHeadersWaitForFallback`.
- Copied cloud filter/header inputs, falsey response-key fallback and fresh uncached repeated reads retain independent results. Custom HTTP messages preserve native rejected status/body/header causes or clean404 missing semantics. Physical accepted/rejected response faults never replay, fall back or issue deletion.
  Tests: `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsOwnedOptionsAndFalseyEnvelope`, `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsCustomErrorKeepsNativeProof`, `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsPhysicalFaultNeverReplaysOrDeletes`.

Python/Go differences:

- The concrete Go carrier preserves completed Extensions, the owned PolicyRecord and independent deletion receipt rather than returning only Python Resource/None/bool. Failed operations retain observed evidence and contextual causes; successful bodyless deletion does not invent Resource/Wire.
- Go opts source100..399/loaded equality/link/fault contracts into a new private action lane while preserving legacy typed Find/List/Delete. Options and records are independently owned JSON snapshots; Python mutable Resource/cache/custom class/adapter/current_location state remains the shared SDK-R1/SDK-C1/SDK-S1 foundation boundary, not a named operation support count.
- Literal IDs/continuations remain in the selected service origin/path/query and are escaped as one member segment. Source URL joining/URI percent formatting can rewrite arbitrary paths; concrete Go source/header/protocol guard and strict UTF8/JSON policies remain explicit reviewed differences.
- For actual returned null ID, Source raises InvalidRequest and Go rejects before DELETE. Source urljoin maps nonnull false/0/underflow/empty ID to a collection route; Go rejects such IDs as a member-route policy instead of issuing collection DELETE, returning false or fabricating success. Nonempty numeric/true JSON IDs use the shared PythonString conversion and preserve large integer precision.
- Stock Neutron27 get_policy permits owner project reader or shared policy (and admin); delete_policy requires ADMIN_OR_PROJECT_MANAGER in project scope with historic deprecated ADMIN_ONLY. Mock/source/race/main validation does not assert authenticated backend success or elevated permissions.
- Cloud filters remain kwargs expanded after fresh full extension discovery and are not a local Cloud _filter_list/JMES predicate. Only the dedicated FindSourceRecord opts into six fetch controls, leaving ordinary QoS FindRecord unchanged. The existing source profile computes Policy Resource.location as null and does not synthesize Connection facts; actual server location JSON remains only in physical Wire/Envelope evidence.
- The fetch-control lane uses finite UTF-8 JSON/string identities and service-local safe routes with only %(id)s and %% URI formatting and string-valued fallback headers. skip_cache is consumed without an invented cache header; each lookup invokes the stateless reader, without claiming bypass of caller custom transport caches. error_message decorates Go HTTP causes and leaves transport/accepted decode/final missing errors distinct. SDK-R1/C1/S1 remain whole-SDK goals.

### `python:cloud/_network/NetworkCloudMixin/delete_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteQoSPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.DeleteSourceRecord`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Every cloud call completes fresh uncached qos extension discovery before options/member/deletion and preserves completed extension evidence. Unavailable extension has the shared sentinel and no policy HTTP. Context/source/selected private API guards remain active through dependent leaf requests and caller option snapshots.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot`.
- New Source action profile clones class Body/Query and opts Source100..399, link precedence, CPython-loaded equality and physical-stop policies in without changing prior typed API baselines. Actual accepted/rejected response receipts and native401 pointer ErrOriginal/value ErrUnexpectedResponseCode plus ErrReauth causes survive read/Close/cancel/source failures with zero retry/reauth and no missing/success fallback.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsPhysicalFaultsNeverFallbackMissingOrRetry`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`.
- Clean503 retry and401 live token renewal are retained for Get/DELETE while changed selected route before retry preserves the original503 receipt. Source status199 opt-in is separately exercised over the public testhelper HTTP fixture with a response observer; no resource/response is fabricated.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsCleanRetryRenewalAndRouteGuard`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`.
- After one fresh qos gate, default Source Find ignore_missing=True runs once. Missing lookup returns Deleted=false without DELETE. A found lookup uses returned Resource.id, sends one bodyless literal member DELETE and returns true after success or clean404 according to Proxy default ignore_missing=True. Lookup and deletion keep separate actual receipts, including clean404 status/body/header/IgnoredMissing.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyDeleteMissingFoundAndIgnored404HaveDifferentResults`.
- Deletion consumes the returned JSON ID without a second lookup or caller-name substitution; large integer and true IDs use PythonString without float64 loss. Empty/null/false/zero/underflow IDs fail before member deletion under the explicit Go path policy. Header/body edits in the deletion carrier cannot alter the lookup receipt.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyDeleteReturnedJSONIDAndIndependentActualReceipts`.

Python/Go differences:

- The concrete Go carrier preserves completed Extensions, the owned PolicyRecord and independent deletion receipt rather than returning only Python Resource/None/bool. Failed operations retain observed evidence and contextual causes; successful bodyless deletion does not invent Resource/Wire.
- Go opts source100..399/loaded equality/link/fault contracts into a new private action lane while preserving legacy typed Find/List/Delete. Options and records are independently owned JSON snapshots; Python mutable Resource/cache/custom class/adapter/current_location state remains the shared SDK-R1/SDK-C1/SDK-S1 foundation boundary, not a named operation support count.
- Literal IDs/continuations remain in the selected service origin/path/query and are escaped as one member segment. Source URL joining/URI percent formatting can rewrite arbitrary paths; concrete Go source/header/protocol guard and strict UTF8/JSON policies remain explicit reviewed differences.
- For actual returned null ID, Source raises InvalidRequest and Go rejects before DELETE. Source urljoin maps nonnull false/0/underflow/empty ID to a collection route; Go rejects such IDs as a member-route policy instead of issuing collection DELETE, returning false or fabricating success. Nonempty numeric/true JSON IDs use the shared PythonString conversion and preserve large integer precision.
- Stock Neutron27 get_policy permits owner project reader or shared policy (and admin); delete_policy requires ADMIN_OR_PROJECT_MANAGER in project scope with historic deprecated ADMIN_ONLY. Mock/source/race/main validation does not assert authenticated backend success or elevated permissions.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("get", "delete"), default="get")
parser.add_argument("--identity", required=True)
parser.add_argument("--filters", default='{"skip_cache":true,"error_message":"QoS policy lookup failed"}')
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "delete":
        print(json.dumps(conn.delete_qos_policy(args.identity)))
    else:
        value = conn.get_qos_policy(args.identity, filters=json.loads(args.filters))
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
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "get", "get or delete")
    identity := flag.String("identity", "", "literal ID or exact name")
    filters := flag.String("filters", `{"skip_cache":true,"error_message":"QoS policy lookup failed"}`, "get-only JSON kwargs")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    switch *mode {
    case "get":
        result, err := service.GetQoSPolicyRecord(
            ctx, *identity, network.WithQoSPolicyFilters(json.RawMessage(*filters)),
        )
        if err != nil { log.Fatal(err) }
        if result.Policy == nil { printJSON(nil); return }
        printJSON(result.Policy.Resource)
    case "delete":
        result, err := service.DeleteQoSPolicyRecord(ctx, *identity)
        if err != nil { log.Fatal(err) }
        printJSON(result.Deleted)
    default:
        log.Fatal("mode must be get or delete")
    }
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L42-L165)
- [Install](../docs/install.md)
- [Cloud get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L220-L250)
- [Cloud delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1512-L1536)
- [Resource.find/fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Proxy delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L670-L713)
