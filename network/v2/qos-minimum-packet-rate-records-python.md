# QoS Minimum Packet Rate Rule의 Python/Go 조회

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud는 `v2.15.0`입니다. Go leaf는 `network/v2/extensions/qos/rules`이고 `network.QoSRules`에서 사용합니다. 이 문서는 세 Source 조회 연산에 대응합니다.

| Python | Go |
| --- | --- |
| `get_qos_minimum_packet_rate_rule(rule, policy)` | `GetMinimumPacketRateRecord(ctx, policyID, ruleID, options...)` |
| `find_qos_minimum_packet_rate_rule(rule_id, policy, ignore_missing=True, **query)` | `FindMinimumPacketRateRecord(ctx, policyID, identity, options...)` |
| `qos_minimum_packet_rate_rules(policy, **query)` | `ListMinimumPacketRateRecords(ctx, policyID, options...)` / `AllMinimumPacketRateRecords` |

Go는 기존 native QoS API처럼 parent-first 인수를 사용합니다. `policyID`는 literal ID이며 이름 검색이나 parent HTTP 요청을 추가하지 않습니다. Python Resource 객체 대신 읽은 policy의 ID를 전달합니다. 빈 값, dot segment, control, invalid UTF-8은 사전 오류이고 slash/percent/space는 한 URL segment로 escaping합니다. 경로는 `/qos/policies/{policyID}/minimum_packet_rate_rules[/{ruleID}]`입니다.

아래 Python은 독립 실행 예제입니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", default="")
parser.add_argument("--mode", choices=("get", "find", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "get":
        value = conn.network.get_qos_minimum_packet_rate_rule(args.rule, args.policy)
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "find":
        value = conn.network.find_qos_minimum_packet_rate_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(value.to_dict(), indent=2))
    else:
        for value in conn.network.qos_minimum_packet_rate_rules(
            args.policy, direction="any", max_items=100
        ):
            print(json.dumps(value.to_dict(), indent=2))
```

아래 Go는 독립 `main.go`입니다. [설치 안내](../../docs/install.md)에 따라 module 안에서 실행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, mode, policyID, identity string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "get":
        value, err := network.QoSRules.GetMinimumPacketRateRecord(ctx, policyID, identity)
        if err != nil { return nil, err }
        return value.Resource, nil
    case "find":
        value, err := network.QoSRules.FindMinimumPacketRateRecord(
            ctx, policyID, identity, rules.WithMinimumPacketRateFindIgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.QoSRules.AllMinimumPacketRateRecords(
            ctx, policyID, rules.WithMinimumPacketRateListQuery("direction", "any"),
            rules.WithMinimumPacketRateListMaxItems(100),
        )
        if err != nil { return nil, err }
        rows := make([]*resource.RawResource, 0, len(values))
        for _, value := range values { rows = append(rows, value.Resource) }
        return rows, nil
    default:
        return nil, fmt.Errorf("unknown mode %q", mode)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    policyID := flag.String("policy", "", "literal QoS policy ID")
    identity := flag.String("rule", "", "literal rule ID or Find identity")
    mode := flag.String("mode", "list", "get, find or list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *policyID, *identity)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## Source Body와 실제 증거

`QoSMinimumPacketRateRule`은 Resource를 직접 상속하며 TagMixin/NetworkResource를 상속하지 않습니다. Body 선언은 `direction`, `min_kpps`와 inherited `id`, `name`이고 모두 untyped/default null입니다. `min_kpps`는 kpps 단위입니다. 숫자 문자열, null, false, zero, 배열/객체와 unknown extension JSON을 native 정수 struct로 강제 변환하지 않습니다. Tags/revision/timestamp/Header descriptor 기본값을 만들지 않습니다. 실제 unknown fields는 보존하고 실제 `If-Match` HTTP Header는 증거에 남습니다.

`Resource`는 Source descriptor/default view, `Wire`는 수신한 literal row, `Envelope`은 member의 실제 body 또는 각 row를 수신한 whole page body입니다. `Header`와 `StatusCode` 역시 실제 관찰값입니다. 서로 독립 snapshot이므로 view를 수정해도 receipt는 바뀌지 않습니다. accepted empty/JSON parse 실패 member 응답은 Source처럼 constructor/default view를 유지하고 Wire는 nil입니다. 유효 JSON의 잘못된 shape/UTF-8/constructor 실패는 실제 응답 증거를 보존한 오류입니다.

`qos_policy_id`는 URI descriptor입니다. 생성자는 이를 Body/to_dict 밖에서 소비합니다. LIST는 고정 parent를 row에 주입한 후 URI로 소비하므로 Resource Body에 표시하지 않습니다. member가 이 key를 unknown Body로 반환하면 Source to_dict는 bound URI를 읽어 고정 parent를 출력하므로 Go도 그 경우만 고정 parent를 Body에 표시합니다. Wire/Envelope은 서버가 보낸 foreign/null parent를 보존하고 `record.QoSPolicyID`는 언제나 고정 parent입니다.

`location`은 Connection이 소유하는 computed 값이며 이 immutable profile에서는 항상 null입니다. 서버가 위조한 location은 Resource에 적용하지 않고 Wire/Envelope에 원문을 보존합니다. Python Connection의 mutable current_location은 프로젝트 차원의 별도 잔여 항목입니다.

## Get·Find·List의 query와 기본 동작

Get proxy는 kwargs를 받지 않습니다. 고정 parent는 URI 생성자에만 사용하고 member query에 넣지 않습니다. Go Get의 안전한 Header/Microversion 옵션은 명시적인 Go 편의입니다. missing/forbidden에서 LIST로 전환하지 않습니다.

Find는 member GET에 고정 `qos_policy_id`와 원래 kwargs를 넣으며 kwargs의 Body/unknown 값을 constructor seed로 유지합니다. None 값은 GET query에서 생략되지만 seed의 explicit null은 보존합니다. 깨끗한 400/403/404에서만 같은 parent의 전체 LIST로 진행하고 exact ID 또는 inherited name으로 검색합니다. name server query가 없으므로 자동 name hint를 넣지 않습니다. 후반 페이지 중복은 `resource.ErrAmbiguous`, 깨끗한 전체 검색의 없음은 기본 `nil, nil`이며 `WithMinimumPacketRateFindIgnoreMissing(false)`에서는 `resource.ErrNotFound`입니다. 실제 forbidden/list/physical 실패는 logical absence로 숨기지 않습니다.

정확한 server QueryParameters는 inherited `limit`, `marker`입니다. `direction`, `min_kpps`, `name`, List `id`는 local Body filter입니다. fields/tags/project/sort/다른 unknown List kwargs는 전송하거나 필터링하지 않으며 marshal 전에 버립니다. Find member는 원래 unknown kwargs를 전달하므로 JSON 표현 가능한 값이어야 합니다. query의 parent/positional/constructor/session/route controls는 사전 오류입니다. Python deprecated JMESPath/`__conflicting_attrs` 복구, mutable Resource/Adapter/cache/session 전체는 이 concrete record mapping 밖이며 별도로 추적합니다.

local dictionary filter는 Source nested subset와 falsey/empty semantics, truthy scalar에 대한 empty-dict match, nonempty-dict mapping.get 실패를 따릅니다. bool-number equality를 지원합니다. decimal/exponent truthiness와 numeric next는 CPython binary64 underflow/overflow를 따르지만 일반 숫자 equality는 shared exact-decimal JSON이라 Python float rounding/nonfinite/custom-object 전체 비교는 지원 판정하지 않습니다.

List는 lazy iterator, All은 late error 때 partial rows를 버리는 collector입니다. 기본 full paging, MaxItems=0 무제한, positive cap은 local filter 전에 raw row를 셉니다. limit=0에 positive cap을 주면 cap이 limit hint입니다. short page도 raw row ID marker로 이어가고 empty page에서 끝납니다. cap/single-page는 관찰 범위의 유일성만 보장합니다. Source HTTP Link/plural links/top-level next를 지원하고 dictionary continuation은 Go 편의입니다. 같은 origin·고정 parent collection·원래 filters와 exact `/v2.0/qos/policies/{parent}/minimum_packet_rate_rules` alias만 허용합니다. foreign parent/query 변경/cycle는 whole page proof를 가진 오류입니다.

## 옵션·버전·실패 처리

구체적인 `MinimumPacketRateGetOpts`, `MinimumPacketRateFindOpts`, `MinimumPacketRateListOpts` 및 With 함수를 사용하고 interface builder를 구현할 필요가 없습니다. JSON/maps/pointers와 callback 결과를 snapshot하며 selected source는 callback 전에 캡처합니다. 옵션을 재사용해도 반환 view/receipt가 공유되지 않습니다.

Find per-call microversion은 member GET에만 적용하고 fallback은 캡처된 service default를 유지합니다. List override는 모든 page에 적용하고 Get override는 해당 요청에만 적용합니다. explicit empty override는 version Header를 생략하는 Go 옵션입니다. class ceiling/discovery를 추가하지 않으며 원래 service client를 수정하지 않습니다. List Accept 기본은 application/json이며 explicit Header가 우선합니다.

실제 완료 status 200..399를 성공으로 처리하고 모든 400..599 거절의 native status/body/header를 보존합니다. Source final <400의 informational 100..199는 이 Go 완료 profile 밖입니다. `StopFaultRetryCallbacks`를 활성화하여 read/Close/cancellation/Source 실패가 있는 실제 응답은 retry/reauth callback으로 진행하지 않습니다. 모든 실패 원인과 actual receipt를 유지하고 요청/Close는 한 번입니다. 깨끗한 native retry와 live token reauthentication은 허용하되 fixed route/query/status/version/auth/framing/Header policy를 지킵니다.

## 권한과 검증 경계

Stock Neutron `27.0.0`의 `get_policy_minimum_packet_rate_rule`은 project scope에서 admin 또는 부모 정책 소유 project reader입니다. shared policy visibility만으로 rule 접근 권한이 생기지 않습니다. [공식 QoS policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L488)와 [parent-owner base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)가 근거입니다. deprecated defaults/운영자 override에 따른 배포별 권한은 실제 응답으로 확인합니다.

테스트는 기존 nativeRuleAPI/nativeRuleWire 및 Gophercloud 공개 testhelper.SetupHTTP를 사용하는 internal/testcloud를 재사용합니다. Root가 focused/full race 및 위 독립 main 빌드를 실행합니다. fixture 성공은 실제 authenticated cloud의 extension·백엔드 지원·권한 검증이 아니며 다른 QoS 종류/Create/Update/Delete 전체 지원 판정으로 확대하지 않습니다.
