# Cloud QoS 규칙 수정: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud `v2.15.0`입니다. `conn.Network(ctx)`로 얻은 network.Service의 아래 세 Update 메서드가 kwargs 검증, 전체 확장 확인, policy lookup, current rule strict GET, raw 변경 비교와 dirty-only PUT까지 처리합니다. 애플리케이션에 builder/interface 구현이나 확장·이름 검색 연결을 요구하지 않습니다.

| Python | Go | 허용 kwargs |
| --- | --- | --- |
| `update_qos_bandwidth_limit_rule(policy, rule_id, **kwargs)` | `UpdateQoSBandwidthLimitRuleRecord(ctx, policy, id, options...)` | max_kbps, max_burst_kbps, direction |
| `update_qos_dscp_marking_rule(policy, rule_id, **kwargs)` | `UpdateQoSDSCPMarkingRuleRecord(ctx, policy, id, options...)` | dscp_mark |
| `update_qos_minimum_bandwidth_rule(policy, rule_id, **kwargs)` | `UpdateQoSMinimumBandwidthRuleRecord(ctx, policy, id, options...)` | min_kbps, direction |
| empty kwargs의 None | `NoOp=true`, Rule/Current=nil | gate와 policy lookup은 수행 |
| raw attrs가 기존 값과 같음 | `NoOp=true`, Rule/Current 있음 | strict GET 수행, PUT 없음 |
| 반환 Resource | `result.Rule.Resource` | owned view, 실제 Wire/Envelope/Header/StatusCode 별도 |

`WithQoSRuleUpdateJSON(rawObject)`, `WithQoSRuleUpdateOpts`, `WithQoSRuleUpdateAttribute(key, value)`가 map/JSON bytes를 소유합니다. Bulk/JSON은 전체 kwargs를 대체하고 뒤 개별 옵션이 같은 키를 덮습니다. Primitive 편의 옵션은 WithQoSRuleUpdateMaxKBPS(int), MinKBPS(int), DSCPMark(int), MaxBurstKBPS(int), Direction(string)입니다. 명시 0과 null을 누락으로 바꾸지 않습니다. JSON/Attribute 옵션으로 null·bool·string·큰 integer를 전달할 수 있으며 Source에 없는 강제 int 변환·양수 검사·default는 넣지 않습니다. 실제 값과 driver 범위는 서버가 검증합니다.

## 정확한 요청 순서

Unknown kwargs나 bound policy_name_or_id/rule_id 충돌, 잘못된 JSON/nil 옵션은 HTTP 전에 오류입니다. 이후 각 호출은 fresh extension 전체 페이지를 소비해 qos를 확인합니다. 첫 alias에서 중단하거나 cache하지 않아 후반 malformed alias/HTTP 실패가 남습니다. 없는 qos는 network.ErrUnavailableExtension이며 policy/rule HTTP가 없습니다.

BW/DSCP policy Find는 ignore_missing=True 뒤 nil이면 NotFound이고 Minimum은 ignore_missing=False입니다. Successful member GET→clean400/403/404의 전체 list fallback, exact name/ID와 후반 duplicate 검사를 SDK가 처리합니다. Missing policy는 **kwargs가 비어 있어도** 오류입니다.

BW만 policy lookup 뒤 nonnull direction에 대해 fresh 전체 qos-bw-limit-direction gate를 수행합니다. false/0/empty string·array·object도 nonnull입니다. 미지원이면 direction만 제거하고 DirectionDropped=true를 반환합니다. Null direction은 제거하지 않고 추가 gate도 없습니다. Optional gate의 HTTP/physical 오류를 미지원으로 숨기지 않습니다. Minimum direction에는 이 추가 gate가 없습니다.

번역 후 kwargs가 비면 None/no current GET/no PUT입니다. Rule ID가 빈 문자열이어도 이 branch에서는 조회하거나 검증하지 않습니다. 남은 kwargs가 있으면 current rule을 strict GET하며 child404를 nil로 바꾸거나 list fallback하지 않습니다. GET 결과의 실제 rule ID로 PUT합니다. Caller rule ID나 공개 Resource를 조작해 새 PUT 대상을 만드는 방식이 아닙니다.

## 변경 감지·borrowed ledger·증거

Private ledger는 raw Source Body의 key presence/값과 원래 URI parent, actual receipt, original API/client/source guard를 보관합니다. Missing rate→null은 dirty이고 true→1, false→0은 Python bool/int equality로 같아 PUT하지 않습니다. CPython-loaded float equality/underflow를 따르고 arbitrary integer를 float64로 바꾸지 않습니다. Descriptor getter나 누락 default만 비교하지 않습니다. Body/URI/receipt 복제를 통해 공개 Resource/Wire/QoSPolicyID/Envelope/Header나 Clone 변경은 private ledger를 바꾸지 못합니다.

Low-level `rules.API.GetCloud{BandwidthLimit,DSCPMarking,MinimumBandwidth}UpdateRecord`가 만든 record/Clone만 같은 API/family의 `UpdateCloud{...}Record`에 빌릴 수 있습니다. 다른 API/client/family나 source drift는 PUT 전에 거부합니다. 일반 사용자는 cloud method만 호출합니다. 이 leaf들은 좁은 cloud update domain이며 general Proxy.update raw attrs 전체 지원 판정을 확대하지 않습니다.

Equal branch는 private current receipt를 반환하고 PUT proof를 만들지 않습니다. Source `_update`처럼 unknown microversion getter는 None/null로 재설정합니다. Dirty PUT 성공은 selected commit version을 유지합니다. Response microversion/foreign qos_policy_id/location의 실제 값은 Wire에 보존하고 view만 selected version/원래 raw URI parent/owned null location을 읽습니다. PUT response가 일부 필드만 반환하면 Resource는 raw seed에서 다른 필드를 유지하고 Wire에 seed를 추가하지 않습니다. Current·Policy·Rule receipt는 별도 owned evidence입니다.

Source Resource._prepare_request는 dirty ID가 None일 때만 InvalidRequest입니다. Equal null ID는 PUT 없이 반환합니다. Dirty falsey nonnull ID(0/false/empty/underflow/empty array·object)는 utils.urljoin에 따라 collection 경로 끝 `/`로 PUT합니다. 이를 문서상의 편의 차이로 숨기지 않고 **Source 전용 copied profile의 SourceCollectionPUT opt-in**으로 같은 selected collection route를 구현했습니다. Legacy profile 기본값, POST/DELETE 및 일반 옵션에는 영향이 없습니다. URI parent null/false/0는 child `%s` 포맷의 None/False/0 segment를 사용하되 view는 원래 JSON 값을 유지합니다.

Go의 경로 정책은 literal parent/rule identity를 한 segment로 escape하고 blank/control/traversal 경로는 거부합니다. Source의 slash/percent 경로 포맷과 다르며, source origin/protocol/header guard를 유지합니다. Collection PUT도 selected fixed profile.Path+'/'에만 허용합니다. 서버가 그 route나 값을 거부하면 실제 응답 오류를 반환합니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=("bandwidth", "dscp", "minimum"), default="bandwidth")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", default="")
parser.add_argument("--kwargs", default="{}")
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "bandwidth":
        result = conn.update_qos_bandwidth_limit_rule(args.policy, args.rule, **kwargs)
    elif args.kind == "dscp":
        result = conn.update_qos_dscp_marking_rule(args.policy, args.rule, **kwargs)
    else:
        result = conn.update_qos_minimum_bandwidth_rule(args.policy, args.rule, **kwargs)
    print(json.dumps(None if result is None else result.to_dict(), indent=2))
```

`python main.py --cloud dev --policy gold --rule RULE_UUID --kind bandwidth --kwargs '{"max_kbps":1000,"max_burst_kbps":0}'`

`python main.py --cloud dev --policy gold --rule RULE_UUID --kind dscp --kwargs '{"dscp_mark":32}'`

`python main.py --cloud dev --policy gold --rule RULE_UUID --kind minimum --kwargs '{"min_kbps":1000,"direction":"egress"}'`

## 독립 Go main

[설치 안내](../docs/install.md)에 따라 module을 준비하고 main.go로 저장합니다. kwargs는 raw JSON으로 읽어 integer precision을 유지합니다.

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
	kind := flag.String("kind", "bandwidth", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "literal policy ID or exact name")
	rule := flag.String("rule", "", "literal rule ID")
	kwargs := flag.String("kwargs", "{}", "JSON cloud update kwargs")
	flag.Parse()
	if *policy == "" {
		log.Fatal("policy is required")
	}
	if *kind != "bandwidth" && *kind != "dscp" && *kind != "minimum" {
		log.Fatal("unknown kind")
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
	option := network.WithQoSRuleUpdateJSON(json.RawMessage(*kwargs))
	var result *network.QoSRuleUpdateResult
	switch *kind {
	case "bandwidth":
		result, err = service.UpdateQoSBandwidthLimitRuleRecord(ctx, *policy, *rule, option)
	case "dscp":
		result, err = service.UpdateQoSDSCPMarkingRuleRecord(ctx, *policy, *rule, option)
	case "minimum":
		result, err = service.UpdateQoSMinimumBandwidthRuleRecord(ctx, *policy, *rule, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	var value any
	if result.Rule != nil {
		value = result.Rule.Resource
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

`go run . -cloud dev -policy gold -rule RULE_UUID -kind bandwidth -kwargs '{"max_kbps":1000,"max_burst_kbps":0}'`

`go run . -cloud dev -policy gold -rule RULE_UUID -kind dscp -kwargs '{"dscp_mark":32}'`

`go run . -cloud dev -policy gold -rule RULE_UUID -kind minimum -kwargs '{"min_kbps":1000,"direction":"egress"}'`

이 main은 actual GET 및 필요하면 PUT을 수행합니다. kwargs '{}'이면 gate와 policy lookup만 수행하고 None을 출력합니다.

## 권한·실패·검증 범위

[고정 Neutron27 BW update 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L244-L260), [DSCP](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L376-L392), [Minimum](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L452-L468)은 project ADMIN_OR_PARENT_OWNER_MANAGER입니다. [parent owner manager 규칙](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L56-L68)은 parent 소유 project manager 또는 admin을 요구합니다. Deprecated ADMIN_ONLY와 구분하며 shared policy 읽기 권한으로 수정 권한을 추론하지 않습니다. SDK가 역할·driver 수락을 만들지 않습니다.

Accepted/rejected response의 actual status/body/header를 보존합니다. Native401은 Unwrap이 없는 ErrUnableToReauthenticate의 ErrOriginal(value ErrUnexpectedResponseCode)과 ErrReauth를 함께 유지합니다. Strict current GET과 PUT의 read/Close·caller cancel·source drift는 success/None/missing으로 바꾸지 않고 retry/reauth hooks를 중단합니다. Clean retry와 live token renewal은 유지하고 changed route는 원래 rejected proof와 함께 중단합니다. Source/header snapshots는 option callbacks 전에 고정합니다.

Actual pinned cloud decorator·Proxy find/get/update·Resource find/fetch/_update/commit을 Python3.14.8에서 74 cases 실행했습니다. HTTP adapter와 extension generator만 fixture이며 live backend/authorization 증거가 아닙니다. Go는 기존 public Gophercloud testhelper-backed internal/testcloud/nativeRuleAPI와 actual response body observer를 재사용합니다. Cloud6+leaf4의 10그룹, GET/PUT physical/cancel/source72사례, falsey collection21사례, clean retry/reauth/route/Source199 12사례를 준비했습니다. Root Go/main/full repository gate는 실제 실행 ledger를 따르며 이 문서는 미실행 PASS를 주장하지 않습니다.

Go concrete JSON/owned snapshot과 JSON wire object order 차이는 명시한 정책입니다. Mutable Python Resource/class/adapter/cache/session/current_location 및 custom runtime 객체는 SDK-R1/SDK-C1/SDK-S1 공통 foundation remaining으로 별도 추적합니다. Generic Source fetch/update kwargs나 mutable class dispatch 전체의 완료를 이 세 named cloud method로 승격하지 않고 supported 증가는 0입니다.

Source: [BW update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1666-L1731), [DSCP update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1877-L1930), [Minimum update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2070-L2129), [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py).
