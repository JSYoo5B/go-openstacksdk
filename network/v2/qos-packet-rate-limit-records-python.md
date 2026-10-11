# QoS Packet Rate Limit Rule의 Python/Go 조회

고정 Source는 `openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, 인증·HTTP client pin은 Gophercloud `v2.15.0`입니다. 실제 Go leaf는 `network/v2/extensions/qos/rules`이며 `network.QoSRules`를 사용합니다. 아래 세 proxy만 이번 named API 판정 대상입니다. Gophercloud v2.15.0에는 PacketRateLimit의 native typed 호출이 없어 기존 `QoSRules` client와 공통 REST/Neutron engine을 재사용합니다. PacketRateLimit Create/Update/Delete는 별도 미완료 API이며 이번 범위에 포함하지 않습니다.

| Python | Go | 경로 |
| --- | --- | --- |
| `find_qos_packet_rate_limit_rule(rule, policy, ignore_missing=True, **query)` | `QoSRules.FindPacketRateLimitRecord(ctx, policyID, identity, options...)` | member GET 후 필요한 경우 동일 parent collection |
| `get_qos_packet_rate_limit_rule(rule, policy)` | `QoSRules.GetPacketRateLimitRecord(ctx, policyID, ruleID, options...)` | member GET 한 번 |
| `qos_packet_rate_limit_rules(policy, **query)` | `QoSRules.ListPacketRateLimitRecords(ctx, policyID, options...)` / `AllPacketRateLimitRecords` | collection GET |

Go의 parent-first 인수 순서는 기존 native rule API와 같습니다. `policyID`는 문자 그대로의 ID이며 정책 이름 검색이나 parent GET을 수행하지 않습니다. Python Resource 인스턴스를 전달하는 대신 이미 읽은 정책의 ID를 넘길 수 있습니다. 빈 ID·dot segment·control·잘못된 UTF-8은 사전 오류이고 `/`·`%`·space는 한 번의 URL segment escaping으로 보존합니다. 실제 경로는 `/qos/policies/{policyID}/packet_rate_limit_rules[/{ruleID}]`입니다.

다음 Python 예제는 각각 독립 실행할 수 있습니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", default="")
parser.add_argument("--mode", choices=("find", "get", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        rule = conn.network.find_qos_packet_rate_limit_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(rule.to_dict(), indent=2))
    elif args.mode == "get":
        rule = conn.network.get_qos_packet_rate_limit_rule(args.rule, args.policy)
        print(json.dumps(rule.to_dict(), indent=2))
    else:
        for rule in conn.network.qos_packet_rate_limit_rules(
            args.policy, direction="egress", max_items=100
        ):
            print(json.dumps(rule.to_dict(), indent=2))
```

아래 코드는 독립 `main.go`입니다. 결과의 `Resource`는 Source descriptor 뷰, `Wire`는 수신한 literal row이며 `QoSPolicyID`는 고정한 URI parent입니다.

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

func run(ctx context.Context, cloud, mode, parent, identity string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "find":
        value, err := network.QoSRules.FindPacketRateLimitRecord(
            ctx, parent, identity, rules.WithPacketRateLimitFindIgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "get":
        value, err := network.QoSRules.GetPacketRateLimitRecord(ctx, parent, identity)
        if err != nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.QoSRules.AllPacketRateLimitRecords(
            ctx, parent, rules.WithPacketRateLimitListQuery("direction", "egress"),
            rules.WithPacketRateLimitListMaxItems(100),
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
    parent := flag.String("policy", "", "literal QoS policy ID")
    identity := flag.String("rule", "", "literal rule ID or Find identity")
    mode := flag.String("mode", "list", "find, get or list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *parent, *identity)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## Find와 Get의 구분

Find는 Source `Resource.find`처럼 literal member GET을 먼저 시도하고, 깨끗한 400·403·404에서만 같은 parent의 전체 목록으로 진행합니다. Source는 Find member query에 `qos_policy_id`도 전달하므로 이 값은 SDK가 고정 parent에서 주입합니다. Get proxy는 URI constructor에만 parent를 사용하여 query에 넣지 않습니다. Get은 missing/forbidden을 숨기거나 LIST로 전환하지 않습니다.

Find fallback은 exact ID 또는 inherited `name` 일치로 하나를 선택합니다. 이 resource는 `name` server query를 선언하지 않으므로 자동 name hint를 보내지 않습니다. 나중 페이지의 중복은 `resource.ErrAmbiguous`, 깨끗한 전체 검색 뒤 없음은 기본 `nil, nil`이며 `WithPacketRateLimitFindIgnoreMissing(false)`에서는 `resource.ErrNotFound`입니다. 목록의 실제 403·500이나 처리 실패는 logical absence가 아닙니다.

## Schema와 URI projection

Source `QoSPacketRateLimitRule`은 Resource를 직접 상속하고 TagMixin/NetworkResource를 상속하지 않습니다. 정확한 Body는 `max_kpps`, `max_burst_kpps`, `direction`과 inherited `id`, `name`이며 모두 untyped/default null입니다. 따라서 `max_kpps="1000"`, false, zero, null, arbitrary extension JSON은 native 정수 struct로 강제 변환되지 않습니다. tags/revision/timestamp/Header 기본 descriptor를 합성하지 않습니다. 실제 응답에 있는 unknown fields는 Source처럼 보존합니다. `If-Match`는 실제 HTTP Header에 남으며 `if_match` Body를 만들지 않습니다.

`qos_policy_id`는 Source URI descriptor입니다. 생성자 URI는 기본 `to_dict`의 Body에 나타나지 않습니다. LIST는 parent URI를 row에 주입하고 URI로 소비하므로 `Resource.Body`에 이 field를 만들지 않습니다. member 응답이 이 key를 unknown Body로 실제 반환하면 Source `to_dict`가 URI descriptor를 읽어 고정 parent ID를 출력합니다. Go도 그 경우에만 Body에 고정 parent를 표시합니다. 항상 `record.QoSPolicyID`로 parent를 읽을 수 있고 `Wire.Body`와 `Envelope`은 서버가 보내온 foreign/null parent를 포함해 원래 bytes를 유지합니다. location은 computed null인 owned snapshot이며 Python Connection의 mutable current_location을 자동 재현하지 않습니다. 서버가 `location`을 false/string/object로 위조해도 `Resource.Body.location`은 null이고 `Wire.Body`와 `Envelope`에는 실제 원문이 남습니다.

Get은 요청 ID만 seed로 갖습니다. Find는 원래 kwargs를 member query와 constructor seed로 사용합니다. accepted body가 empty 또는 JSON parse 실패이면 Source처럼 constructor/default Resource를 유지하고 Wire는 nil입니다. 유효 JSON wrong shape/UTF-8/descriptor 실패는 actual receipt를 보존한 오류입니다. Resource/Wire/Envelope/Header는 별도 snapshot이며 전체 page Envelope은 각 LIST row가 수신된 페이지의 실제 body입니다. LIST row의 `cls`, `connection`, `microversion`, `_synchronized` constructor 충돌은 실제 whole-page203 응답을 보존하며 실패합니다. member 응답의 `microversion`은 raw Wire와 별도로 Resource에서 실제 선택 버전 또는 null을 표시합니다.

## Query·pagination·옵션

정확한 server QueryParameters는 inherited `limit`, `marker` 두 개입니다. `direction`, `max_kpps`, `max_burst_kpps`, `name`, LIST의 `id`는 알려진 Body라 **local filter**입니다. `fields`, tags, project_id, sort 및 다른 unknown LIST kwargs는 전송·local filtering 없이 버립니다. unknown 값은 marshal 이전에 버리므로 사용자가 값으로 function을 넣어도 오류를 만들지 않습니다. Find GET은 원래 kwargs를 전달하므로 unknown GET 값은 JSON 표현 가능한 값이어야 합니다. known 값의 marshal 실패는 `resource.ErrInvalidOption`과 caller marshal 원인을 함께 보존하고 HTTP 전에 종료합니다. callback이 Query/Headers를 nil로 바꿔도 다음 옵션은 독립적으로 초기화된 map을 받습니다.

Local dictionary filter는 Source falsey/empty/nested subset와 truthy scalar에 대한 empty-dict match 및 mapping.get 실패를 따릅니다. bool-number equality(`True == 1`)를 지원합니다. CPython의 float underflow truthiness와 falsey numeric next termination을 따르지만 일반 숫자 비교는 shared exact-decimal JSON profile이므로 모든 Python float rounding/nonfinite/custom-object 비교를 재현하지 않습니다.

List는 lazy iterator이며 `AllPacketRateLimitRecords`는 late error 때 partial 결과를 버립니다. 기본 full paging, MaxItems=0 무제한, positive cap은 local filter **전**의 raw row 개수입니다. `WithPacketRateLimitListLimit`, `Marker`, `MaxItems`, `Paginated`는 typed library options입니다. explicit limit=0과 cap이 함께 있으면 cap이 limit hint입니다. short page도 wire ID marker로 이어가고 empty page에서 종료합니다. Source HTTP Link/plural links/top-level next 외 dictionary links는 명시 Go 편의입니다. continuation은 같은 origin·parent collection·원래 filters를 유지해야 하고 exact `/v2.0/qos/policies/{parent}/packet_rate_limit_rules` alias만 허용합니다. foreign parent/origin·query 변경·cursor cycle는 실제 page proof와 함께 오류입니다. cap/single-page는 관찰 범위의 유일성만 보장합니다.

`qos_policy_id`, `qos_policy`, `qos_rule_id`, `qos_rule`, positional/find controls와 route/session/constructor controls는 query에서 사전 거부합니다. `allow_unknown_params`와 deprecated `jmespath_filters`도 공통 SDK foundation 범위로 남겨 이 concrete profile에서는 명시적으로 거부합니다. callback 중 선택 client/endpoint/project/header/source를 바꾸면 HTTP 전에 오류입니다. API 사용자에게 interface builder를 구현하도록 요구하지 않습니다.

Get Header/Microversion 옵션은 kwargs 없는 pinned get proxy에 추가한 명시 Go 편의입니다. Find의 per-call microversion은 member GET에만 적용하고 fallback은 캡처한 service default를 사용합니다. List override는 전체 page에 적용합니다. client 자체는 수정하지 않습니다. 안전한 explicit Headers는 snapshot이고 LIST Accept 기본은 `application/json`입니다. authentication/framing/foreign version과 fixed query/status/route policy는 native retry에서도 보존하며 clean retry와 live token reauthentication은 공통 guarded reader에서 허용합니다.

완료 응답의 성공 범위는 200..399이며 Source `raise_from_response`의 `<400`와 비교하여 informational final status 100..199는 이 Go 완료 profile 밖입니다. 실제 400..599 native rejection과 accepted/rejected read·Close·custom cancellation·captured Source 실패는 HTTP status/body/header와 원인을 보존합니다. physical fault가 있는 404는 Find fallback을 허용하지 않습니다. physical/source/cancellation fault 뒤에는 willing retry/reauth hook도 실행하지 않으며 HTTP·Close는 각 한 번입니다. native 401 재인증 guard 오류는 `ErrUnableToReauthenticate.ErrOriginal`의 실제 응답과 `ErrReauth`의 원인을 각각 보존합니다. Gophercloud가 이 두 독립 오류를 Unwrap하지 않으므로 공개 필드로 확인합니다.

## 조회 권한과 검증 경계

Stock Neutron `27.0.0`의 `get_policy_packet_rate_limit_rule`은 project scope에서 **admin 또는 부모 정책 소유 project의 reader**를 요구합니다. 정책 자체를 읽는 shared-policy 분기가 rule read 권한까지 보장하지 않습니다. 이 SDK는 선택한 인증을 그대로 사용하고 실제 403을 보존합니다. 운영자가 정책을 override하거나 deprecated defaults를 사용하면 배포별 결과가 달라질 수 있습니다. 근거는 [공식 QoS policy 선언](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L280)과 [공식 parent-owner policy base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)입니다. Create/Update/Delete 정책은 이번 세 조회 API 판정 밖입니다.

테스트는 기존 `nativeRuleAPI` 및 Gophercloud 공개 `testhelper.SetupHTTP` 기반 `internal/testcloud`를 재사용합니다. literal parent/query 차이, schema/URI projections, defaults/extensions, missing/fallback/late duplicate, local query, paging/scope, every 400..599 proof, physical errors, option snapshots, selected version and retry ownership을 18개 테스트 그룹으로 검증합니다. forged location·mixed Read/Close/sticky Source/custom cancellation·401 공개 필드·clean live401와 explicit clear version/header 경계도 포함합니다. fixture/race/external-main 빌드 검증은 실제 Neutron deployment의 인증·extension·권한 검증과 별개입니다. Python Resource instance mutation/cache/session/Connection 전체를 그대로 이식했다는 판정도 아닙니다.
