# Neutron QoS DSCP marking rule: Python과 Go

고정 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud v2.15.0, Neutron 27.0.0입니다. `QoSRules.GetDSCPMarkingRecord`·`FindDSCPMarkingRecord`·`ListDSCPMarkingRecords`는 named `get_qos_dscp_marking_rule`, `find_qos_dscp_marking_rule`, `qos_dscp_marking_rules`의 Source-shaped record/default/query 계약을 제공합니다. 기존 native typed 호출과 parent resource Collection을 다시 완료 API로 집계하지 않습니다.

| Python | Go | 기본 동작 |
|---|---|---|
| `get_qos_dscp_marking_rule(rule_id, policy_id)` | `GetDSCPMarkingRecord(ctx, policyID, ruleID)` | 해당 policy의 literal member GET, query-free, fallback·missing suppression 없음 |
| `find_qos_dscp_marking_rule(rule_id, policy_id)` | `FindDSCPMarkingRecord(ctx, policyID, identity, options...)` | GET-first, clean400/403/404만 fallback, 기본 ignore-missing true |
| `qos_dscp_marking_rules(policy_id, **query)` | `ListDSCPMarkingRecords(ctx, policyID, options...)` | lazy full list, `Accept: application/json` |
| `ignore_missing=False` | `WithDSCPMarkingFindIgnoreMissing(false)` | complete fallback search가 비면 ErrNotFound |
| `limit`, `marker` | `WithDSCPMarkingListLimit/Marker` | inherited server query |
| `id`, `name`, `dscp_mark` | `WithDSCPMarkingListQuery(key, value)` | server 쿼리로 보내지 않고 각 row를 로컬 필터 |
| `max_items`, `paginated=False` | `WithDSCPMarkingListMaxItems/Paginated` | raw-row cap·단일 페이지 |

Policy는 ID로 명시합니다. pinned Proxy의 `_get_resource(QoSPolicy, policy)`는 parent GET/find를 수행하지 않고 URI seed만 만듭니다. Go에서도 부모 정책을 먼저 조회하지 않습니다. Python은 policy Resource/Munch를 받을 수 있지만 Go의 이 concrete API는 literal ID string을 받아 경로를 고정합니다. parent name 자동 resolve가 필요한 기존 `DSCPMarkingRules(ctx, resource.ByName(...))` scope와 사용 목적이 다릅니다.

아래 Python 전체는 독립 실행 가능한 비교 예제입니다. 기존 OpenStack 인증 환경을 사용합니다.

```python
import argparse
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--policy", required=True)
    parser.add_argument("--operation", choices=("list", "get", "find"), default="list")
    parser.add_argument("--rule", default="")
    parser.add_argument("--page-size", type=int, default=0)
    parser.add_argument("--max-items", type=int, default=0)
    parser.add_argument("--mark", type=int)
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect()
    if args.operation == "get":
        result = conn.network.get_qos_dscp_marking_rule(args.rule, args.policy)
        print(result.to_dict())
    elif args.operation == "find":
        result = conn.network.find_qos_dscp_marking_rule(
            args.rule, args.policy, ignore_missing=not args.strict
        )
        print(None if result is None else result.to_dict())
    else:
        query = {"limit": args.page_size, "max_items": args.max_items}
        if args.mark is not None:
            query["dscp_mark"] = args.mark
        for rule in conn.network.qos_dscp_marking_rules(args.policy, **query):
            print(rule.qos_policy_id, rule.to_dict())


if __name__ == "__main__":
    main()
```

Go 전체 예제를 `main.go`에 저장해 실행할 수 있습니다. `--mark=-1` 기본값은 로컬 mark 필터를 생략합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
)

func show(record *rules.DSCPMarkingRecord) {
    if record == nil { fmt.Println("missing"); return }
    body, err := json.Marshal(record.Resource.Body)
    if err != nil { log.Fatal(err) }
    fmt.Printf("policy=%s HTTP=%d body=%s\n", record.PolicyID, record.StatusCode, body)
}

func main() {
    policy := flag.String("policy", "", "parent policy ID")
    operation := flag.String("operation", "list", "list, get or find")
    rule := flag.String("rule", "", "literal rule ID or find identity")
    pageSize := flag.Int("page-size", 0, "server page size")
    maxItems := flag.Int("max-items", 0, "raw row cap; zero means no cap")
    mark := flag.Int("mark", -1, "optional local DSCP mark filter")
    strict := flag.Bool("strict", false, "find absence is an error")
    flag.Parse()
    if *policy == "" { log.Fatal("--policy is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    switch *operation {
    case "get":
        record, err := network.QoSRules.GetDSCPMarkingRecord(ctx, *policy, *rule)
        if err != nil { log.Fatal(err) }
        show(record)
    case "find":
        record, err := network.QoSRules.FindDSCPMarkingRecord(ctx, *policy, *rule,
            rules.WithDSCPMarkingFindIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        show(record)
    case "list":
        options := []rules.DSCPMarkingListOption{
            rules.WithDSCPMarkingListLimit(*pageSize),
            rules.WithDSCPMarkingListMaxItems(*maxItems),
        }
        if *mark >= 0 {
            options = append(options, rules.WithDSCPMarkingListQuery("dscp_mark", *mark))
        }
        for record, err := range network.QoSRules.ListDSCPMarkingRecords(ctx, *policy, options...) {
            if err != nil { log.Fatal(err) }
            show(record)
        }
    default:
        log.Fatal("--operation must be list, get or find")
    }
}
```

[Source rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_dscp_marking_rule.py)는 Resource를 직접 상속합니다. declared Body는 inherited `id`·`name`과 `dscp_mark`이며 모두 untyped/null default입니다. dscp_mark를 Go int로 강제하지 않아 null·false·0·빈 문자열·큰 정수·향후 확장 shape를 Wire와 함께 유지합니다. Header descriptor, tags/default empty list, revision/timestamp cast를 다른 Neutron resource에서 가져오지 않습니다. location은 owned null로 제공합니다. unknown Body extension도 보존합니다.

`DSCPMarkingRecord.PolicyID`는 고정된 URI attribute입니다. Source `to_dict()`는 URI descriptor를 기본 Body field로 출력하지 않으므로 Go도 목록이나 constructor seed에서 `qos_policy_id`를 Body default로 합성하지 않습니다. member 응답에 같은 unknown key가 실제로 있으면 Source unknown-key getter가 URI parent를 읽는 동작에 맞춰 Resource view에 bound parent를 표시하고, Wire에는 서버의 실제 값을 보존합니다. LIST는 Source가 URI props를 row에 덮어쓴 뒤 constructor가 URI를 소비하는 동작을 유지합니다. actual Wire row는 원본을 별도로 보존합니다.

Get은 ID만 response constructor seed로 사용합니다. Go의 `WithDSCPMarkingGetQuery/Headers/Microversion`은 명시적 transport 확장으로, pinned named Python getter의 query-free signature보다 넓습니다. Get query는 response Body seed가 되지 않으며 404가 clean이어도 list fallback하거나 무시하지 않습니다. accepted empty/malformed JSON은 Source ValueError suppression처럼 ID/default를 유지한 부분 Resource이고, valid null/scalar/wrong envelope나 invalid UTF8는 실제 ACK의 `resource.ResponseError`입니다.

Find에는 필수 `qos_policy_id`가 Resource.find의 constructor kwargs와 member fetch params에 함께 들어갑니다. 따라서 첫 GET은 `?qos_policy_id=...`도 보내지만, LIST에서는 URI를 경로에만 사용합니다. caller가 bound `qos_policy_id`·`qos_policy` 또는 Find의 `qos_rule_id`를 다시 옵션 query에 넣으면 HTTP 전에 거부합니다. pinned named Proxy에서 중복 required kwargs를 전달해도 오류입니다. Get/find/list 모두 literal ID를 한 번 escape해 endpoint의 reverse prefix와 policy 경로를 유지하며, 빈 값·dot segment·control·invalid UTF8를 Go 사전 보호로 거부합니다.

Find의 첫 GET은 원래 JSON kwargs를 Requests 쿼리로 보내고 Source constructor Body seed로 보유합니다. unknown GET kwargs는 유지하지만 fallback LIST에서는 unknown을 버립니다. Resource의 `_query_mapping`에는 limit/marker만 있어 name 자동 server hint, fields/dscp_mark server query를 추가하지 않습니다. `id`·`name`·`dscp_mark`는 실제 선언된 Body local filter이며 unknown extension은 class-declared filter가 아닙니다. local filters는 옵션 순서와 nested raw JSON 순서를 보존해 단락 평가하고 bool/number·recursive dictionary 규칙을 공통 engine으로 처리합니다. 수치 equality는 기존 SDK-C1 exact-decimal 정책으로, Python float binary64 반올림과 완전 동치가 아닙니다. 예를 들어 9007199254740992와 raw9007199254740993.0을 다르게 비교합니다.

clean HTTP400/403/404만 fallback을 허용합니다. raw cap/단일 페이지를 별도로 지정하지 않으면 모든 페이지에서 ID 또는 name의 정확한 일치를 확인하고 두 개면 ErrAmbiguous를 반환합니다. 첫 후보를 찾고도 뒤 페이지 오류를 숨기지 않습니다. 기본 ignore-missing true는 clean search가 끝나고도 후보가 없을 때만 nil입니다. raw MaxItems는 local filter에서 제외된 행도 세며 falsey limit에 최초 hint를 제공합니다. iterator `break`, cap, single-page는 미소비 row와 next link work를 멈춥니다. 공통 list pager는 singleton/plural envelope, body/HTTP next links, 짧은 nonempty page의 truthy limit marker fallback, 빈 page 종료와 원래 policy 경로/filter/cycle 보호를 제공합니다.

`WithDSCPMarkingFindMicroversion`은 GET에만 적용하고 fallback은 원래 service default를 사용합니다. 이는 Source Resource.find가 explicit version을 fetch에만 전달하는 동작입니다. List/Get의 explicit version은 해당 호출에 적용하며 기본값은 selected client의 default 또는 부재입니다. max microversion·새 discovery 요청·임의 version ceiling을 추가하지 않습니다. 공통 Engine의 accepted status200..399 정책은 pinned generic `<400`이 transport-exposed1xx까지 허용하는 것보다 좁습니다. 이 공통 전송 정책 차이를 실제 status 증거와 함께 유지합니다.

Resource/Wire JSON, whole Envelope, Header/status는 독립 소유합니다. Source/client를 option callback 전에 고정하고 nested raw JSON·header map·pointer 및 callback 결과를 snapshot합니다. native HTTP 오류의 method/URL/status/body/header, Read/Close·취소 cause와 sticky source 변경을 보존합니다. clean native retry와 live provider token/reauth는 유지하되 SDK-owned auth/framing/foreign Nova·Cinder version, selected network version 또는 부재와 bodyless route를 바꾸면 실패합니다. 성공 처리·physical/source 오류는 후속 fetch/fallback을 만들지 않습니다. 전체 mutable Source Resource/cache/dirty state/Connection/location synthesis와 custom Adapter/session lifecycle은 별도 SDK-R1/C1/S1 범위입니다.

[Neutron 27.0.0 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L337-L357)는 collection/member GET 모두 `ADMIN_OR_PARENT_OWNER_READER`, project scope입니다. QoS policy 자체의 `get_policy`에 있는 shared-policy read 조건과 다르며, 다른 프로젝트가 공유 정책을 볼 수 있다는 이유로 그 정책의 rule 읽기까지 허용되지 않습니다. SDK는 권한을 추정하거나 client-side allow/deny하지 않고 실제 403 등을 유지합니다. Find는 Source 기본대로 clean member403 후 LIST를 시도할 수 있으나 LIST403은 권한 오류입니다. 실제 배포의 정책 override, QoS extension enablement·supported marks와 authenticated cloud acceptance는 fixture 결과와 별도입니다.

[계약 테스트](python_dscp_marking_contracts_test.go)는 기존 QoS native fixture, public Gophercloud testhelper 기반 `internal/testcloud`, 공통 neutronfind/cloudread/REST를 재사용합니다. parent route·required kwargs·untyped/default/raw evidence, complete find/404 policy·local/paging/cap, accepted/rejected physical/source/cancel faults, native retry/version/header, 실제 HTTP escaping와 동시 옵션 재사용을 검증합니다. 신규 transport 또는 pagination 구현은 없습니다. 집중 race·독립 Go main compile은 루트 통합 단계에서 확인합니다.
