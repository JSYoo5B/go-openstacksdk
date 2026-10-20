# Security Group Rule List Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/security_group_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordListPaginated`

- Pinned Source SecurityGroupRule MRO reuses existing SecurityGroupRuleRecord and canonical descriptor mapping:18 Body fields, NetworkResource row Header if_match list, TagMixin tags defaultempty and Computed location ownednull. Only port ranges/revision have Source int conversion; bool/null/rawexact/Unicode string/digit faults, ether canonical/wire orderedlastvalue and project absent-only tenant fallback remain distinct from Wire. Wholepage headers never overwrite row Header constructor attributes. Unknown/bodyIF_MATCH/self/forgedlocation are retained only in appropriate independent carriers and clones.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListSourceDescriptorsHeadersAndOwnedEvidence`, `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListPagesEnvelopeAndConstructorProof`.
- Lazy Source-owned List maps declared server queries and canonical aliases ether_type/any_tags/not_tags/not_any_tags, preserving Requests repeated/null/bool values and canonicalpriority, allowing id serverquery and not inventing namehint. Declared nonquery Body localfilters are evaluated without client re-filtering serverfields. Unknown/Header/all_projects kwargs are discarded before marshaling and do not expand authority.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListServerAliasesLocalFieldsAndUnknownDiscard`, `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListOrderedDictionaryFiltersAndExactEquality`.
- Ordered local recursive mapping filters preserve top-level/nested order and short-circuit. Empty mapping matches truthfulscalar, nonemptymapping ontruthfulscalar yields actualpage failure, Python float underflow truthiness and bool-number behavior are reused. Shared exactdecimal equality intentionally distinguishes actual9007199254740992 fromrawfilter9007199254740993.0.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListOrderedDictionaryFiltersAndExactEquality`.
- Shared Engine.List retains plural/singleton/next/plurallinks/HTTPLink/versionedalias/shortpage marker, empty stop and falseydecimal continuation, rawcap includingfilteredrows withzero-limit hint, singlepage/consumerbreak and no evaluationof unusedmalformedrow/unsafe next. Wrongshape/invalidconstructor cls/connection/microversion/_synchronized/digitfaults/unsafeorigincollectionfilter/cycle preserve actualwholepage proof; laterdenial remainsvisible and AllRecords discards partial slice.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListRawCapAndBreakAvoidUnconsumedWork`, `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListPagesEnvelopeAndConstructorProof`, `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListStatusPoliciesAndLateDenial`.
- Shared Engine accepted200..399 and actualnative informational199/400..599 errors remain distinct; no findfallback or ignore404 for list. Actual status/body/header/methodURL and source/cancel/Read/Close/mixedcauses are preserved. StopFaultRetryCallbacks=true suppresses originalRetryFunc before physicalfault exposure across accepted/rejected70lanes including401 and mixed Read/Close/Source/cancel faults; exactlyoneHTTP/onephysicalClose, no replay or manufactured Resource. A willing original ReauthFunc stays uncalled on physical401. Native *ErrUnableToReauthenticate ErrOriginal keeps actual401 method/URL/status/body/Header; non-unwrapped ErrReauth preserves exact guard/physical causes and one close.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListStatusPoliciesAndLateDenial`, `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListPhysicalFaultsStopRetryCallbacks`.
- Selected/default network version or absence and explicit per-list override use no invented ceiling/discovery. Source AcceptJSON overridescapturedAccept and explicit safeHeaderwins. Cleanretry/reauth/tokenrenewal preserve request; auth/framing/foreignNovaCinder/versionOmit/bodytamper retainsactualrejection and callbackcause withoutsecondHTTP. Native OkCodes expansion canresend but original acceptedpolicy rejectsactual400.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListVersionsHeadersRetryAndReauthentication`.
- ConcreteJSON/header/version/optionslice and retainedcallback states are snapshotted on eachlazyexecution; API/source capturedbeforecallback, nilctxAPIoption andbadcontrols/header/query/negativecap/limit/cancellation fail beforeHTTP withoutfakeproof. Existing nativeRuleRecorder and publicGophercloudtesthelpertestcloud reused; concurrentfactory reuse exercisesrealHTTP andsourceimmutability withoutnewtransport/pager/mockserver framework. SDK owns initialized Query/Headers maps before the first customcallback and after nil reset/empty DTO factory; direct writes and later retained-map mutation are isolated.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListPreflightAndOwnedOptionSnapshots`, `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListPublicFixtureConcurrentReuse`, `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListCustomCallbacksReceiveOwnedMaps`.
- SourceFindPolicy fixes all four conditional headers (If-Match, If-None-Match, If-Modified-Since, If-Unmodified-Since) to their selected values or required absence. Clean native retry preserves explicitly selected conditions and ordinary new headers. Injection of an absent condition, replacement or omission of a selected condition fails before HTTP attempt two, preserving the actual503 status/body/Header and callback cause; caller client headers remain unchanged.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListConditionalHeadersRemainSelected`.
- Source Proxy._list jmespath_filters is a named deprecated arbitrary-transform control, not unknown query. Bounded Go record iterator explicitly rejects key presence (null/false/empty/string/malformed raw/unmarshalable value) as unsupported SDK-S1 before JSON marshaling or HTTP, consistently for query factory, bulk DTO and custom callback. No manufactured response proof; Source falsey no-transform behavior is an explicit Go difference. Existing known wire aliases are unchanged.
  Tests: `network/v2/extensions/security/rules/python_list_contracts_test.go:TestPythonSecurityRuleListJMESPathTransformPresenceIsExplicitlyUnsupported`.

Python/Go differences:

- Exactlyone unresolvednamed python:network/v2/security_group_rules is mapped; existing typedList/All/generatedCollection and SourceFind/mutations are not recounted or changed. Library-owned profile reuses existing canonicaldescriptors/SecurityGroupRuleRecord plus shared neutronfind.Engine.List/SourceFindPolicy and StopFaultRetryCallbacks=true; no sharedexisting/generatorwrites or newtransport/pager implementation.
- SecurityGroupRule derives NetworkResource/TagMixinNetwork:18 declaredBody + row Header if_match + Computedlocation. Resource defaults/casts/unknownextensions are separate fromphysicalWire/Envelope/status/Header. Null ownedconnectionview and exactcomputedlocation ownership do not claim Python Connection/currentLocation/Resourcemutablecache/dirtyState/session/runtimeSubclass parity; shared SDK-R1/C1/S1 remainsindependent.
- SequentialJSONquerylocalfilter order is retained, bulkGoMap/newcallbackkey order sorted; numeric equalityuses sharedSDK-C1 exactdecimal, not arbitrary CPythonfloatrounding/nonfinite/runtimeobjects. Intdescriptors use auditedUnicode16 tables/arbitraryprecision rather than caller interface casts or configurablePython integerstringlimit.
- UnknownLISTkwargs ignoredbeforemarshal and noHeaderfilter; safeconcreteoptions reject Resourceconstructor/session/basepath override. Go rawmax_items and limit are finite nonnegative machineintegers under sharedpager; safeorigin/collection/filter/cycle checks and finitepagebound replace Source broadretargeting. HTTPLinkRFC/dictionarylinks are explicitGocompatibility with exactversionedalias.
- Engine acceptedfinal200..399 differs from pinnedgeneric valid100..399, strictUTF8 differs Requests replacementdecode. Actualnative status/body/header/methodURL and physicalsource/cancelmixedcauses remainobservable. StopFaultRetryCallbacks is explicitGo guard preventing faultcallback/resend; cleanRetryFunc/reauth remain. Ordinary new retry headers remain allowed; all four conditional headers are fixed to selected values or required absence, and injection/replacement/OmitHeaders fails before HTTP attempt two. Existing selected headers and protocol absence remain protected. OkCodes expansion mayretry but cannot establishsuccess outsideoriginalacceptance. Faulty401 has native *ErrUnableToReauthenticate public ErrOriginal/ErrReauth, which do not Unwrap; inspect the original receipt and guard cause separately. A willing hook is suppressed while existing clean401 renewal remains.
- PinnedNeutron27 get_security_group_rule ADMIN_OR_SG_OWNER_READER/projectscope applies collection/member. StockDB collectsfilteredcandidates usingadmincontext, controllerfilters eachrow byactualpolicy; emptysuccessisnormal and notproofof fullauthority. No localRBAC inference/expansion; policyoverride/extensions/livecloudacceptance remainactualdeploymentwork distinctfromfixtures.
- Source Proxy._list openstack/proxy.py866–916 accepts named deprecated jmespath_filters and applies a truthy-string JMESPath expression with arbitrary output. This bounded Resource record API does not implement the transform (SDK-S1) and rejects presence before marshaling/HTTP instead of silently losing it. Explicit null/false/empty presence is also rejected, unlike Source falsey passthrough. Unknown kwargs and known canonical/wire aliases retain their existing behavior.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--group", default="")
    parser.add_argument("--direction", default="")
    parser.add_argument("--name", default="")
    parser.add_argument("--page-size", type=int, default=-1)
    parser.add_argument("--max-items", type=int, default=0)
    parser.add_argument("--single-page", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {"max_items": args.max_items, "paginated": not args.single_page}
    if args.group:
        query["security_group_id"] = args.group
    if args.direction:
        query["direction"] = args.direction
    if args.name:
        query["name"] = args.name
    if args.page_size >= 0:
        query["limit"] = args.page_size
    for rule in conn.network.security_group_rules(**query):
        print(json.dumps(rule.to_dict(), ensure_ascii=False))


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
    "os"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    group := flag.String("group", "", "server security_group_id filter")
    direction := flag.String("direction", "", "server direction filter")
    name := flag.String("name", "", "local inherited name filter")
    pageSize := flag.Int("page-size", -1, "server limit; -1 omits it")
    maxItems := flag.Int("max-items", 0, "raw cap before local filtering")
    single := flag.Bool("single-page", false, "read first page only")
    flag.Parse()
    if err := run(*cloud, *group, *direction, *name, *pageSize, *maxItems, *single); err != nil { log.Fatal(err) }
}

func run(cloud, group, direction, name string, pageSize, maxItems int, single bool) error {
    if pageSize < -1 || maxItems < 0 { return fmt.Errorf("invalid page-size or max-items") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    options := []rules.RecordListOption{
        rules.WithRecordListMaxItems(maxItems),
        rules.WithRecordListPaginated(!single),
    }
    if pageSize >= 0 { options = append(options, rules.WithRecordListLimit(pageSize)) }
    for _, filter := range []struct{ key, value string }{
        {"security_group_id", group}, {"direction", direction}, {"name", name},
    } {
        if filter.value != "" { options = append(options, rules.WithRecordListQuery(filter.key, filter.value)) }
    }
    out := json.NewEncoder(os.Stdout)
    for record, readErr := range network.SecurityRules.ListRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := out.Encode(map[string]any{
            "resource": record.Resource.Body,
            "wire": record.Wire.Body,
            "envelope": record.Envelope,
            "headers": record.Header,
            "status_code": record.StatusCode,
        }); err != nil { return err }
    }
    return nil
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6820-L6842)
- [Source SecurityGroupRule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group_rule.py)
- [Source Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866-L916)
- [Neutron 27.0.0 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py#L228-L247)
- [stock DB](https://github.com/openstack/neutron/blob/27.0.0/neutron/db/securitygroups_db.py#L1021-L1050)
- [controller](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/v2/base.py#L292-L331)
- [Python List Contracts Test](extensions/security/rules/python_list_contracts_test.go)
