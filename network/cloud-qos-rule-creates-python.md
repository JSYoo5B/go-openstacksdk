# Cloud QoS 규칙 생성: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud `v2.15.0`입니다. `conn.Network(ctx)`의 cloud facade가 kwargs 검증·전체 확장 조회·정책 검색·direction 번역·규칙 POST를 소유합니다. 별도 application builder/interface를 구현하거나 확장 목록과 policy ID를 직접 연결할 필요가 없습니다.

| Python cloud method | Go service method | 필수값 / 선택 kwargs |
| --- | --- | --- |
| `create_qos_bandwidth_limit_rule(policy, max_kbps, **kwargs)` | `CreateQoSBandwidthLimitRuleRecord(ctx, policy, maxKBPS, options...)` | max_kbps / max_burst_kbps, direction |
| `create_qos_dscp_marking_rule(policy, dscp_mark)` | `CreateQoSDSCPMarkingRuleRecord(ctx, policy, dscpMark, options...)` | dscp_mark / 없음 |
| `create_qos_minimum_bandwidth_rule(policy, min_kbps, **kwargs)` | `CreateQoSMinimumBandwidthRuleRecord(ctx, policy, minKBPS, options...)` | min_kbps / direction |
| Python rule Resource | `result.Rule` | owned descriptor/seed view |
| 실제 응답 | `result.Wire`, `Envelope`, `Header`, `StatusCode` | seed를 서버 반환값으로 표시하지 않음 |
| 이번 호출의 확장·정책 단계 | `result.Extensions`, `result.Policy` | 규칙 POST와 별도 증거 |

필수값은 JSON-domain any를 받습니다. 일반 호출은 `..., 1000, network.WithQoSRuleCreateDirection("egress")`처럼 primitive와 concrete 편의 옵션만 사용하면 됩니다. 0도 명시된 필수값이며 누락으로 처리하지 않습니다. null, bool, string 또는 큰 integer를 Source처럼 전달해야 하면 nil, json.RawMessage, json.Number 또는 JSON 옵션을 사용합니다. SDK가 Source에 없는 int 변환·범위 보정·필수 양수 검사나 burst/direction default를 만들지 않습니다. 서버가 실제 값·driver 지원 범위를 검증합니다.

선택 옵션은 `WithQoSRuleCreateDirection(string)`, `WithQoSRuleCreateMaxBurstKBPS(int)`, `WithQoSRuleCreateAttribute(key, value)`, `WithQoSRuleCreateJSON(rawObject)`, `WithQoSRuleCreateOpts(value)`입니다. JSON/bulk 옵션은 선택 kwargs 전체를 대체하고 뒤 개별 옵션은 같은 키를 덮습니다. SDK가 map과 JSON bytes를 복제합니다. Direction:null 또는 raw false/0/array/object는 Attribute/JSON 옵션으로 표현합니다. DSCP에 추가 kwargs를 전달하면 HTTP 전에 오류입니다. 이 Go variadic 옵션은 일관된 사용법을 위한 편의이며 pinned Python DSCP method에는 kwargs가 없습니다.

## 요청 순서와 기본 동작

BW는 max_burst_kbps/direction, Minimum은 direction만 선택 kwargs로 허용합니다. Required key나 policy_name_or_id를 선택 kwargs에 다시 넣거나 unknown key를 보내면 Source의 인수 충돌/valid_kwargs처럼 extension HTTP 전에 오류입니다. 잘못된 UTF-8 JSON과 nil 옵션도 선검증합니다.

매 호출은 cache 없이 전체 extension 목록을 읽어 qos alias를 확인합니다. 첫 page에 qos가 있어도 후반 page 오류나 unhashable alias를 숨기지 않습니다. 없는 qos는 `network.ErrUnavailableExtension`이고 policy/rule HTTP를 수행하지 않습니다.

그 뒤 기본 Source policy Find를 수행합니다. BW/DSCP는 ignore_missing=True로 찾은 뒤 nil이면 오류이고 Minimum은 ignore_missing=False입니다. 성공 GET의 반환 ID를 사용하며 clean400/403/404에서만 전체 list fallback으로 exact ID/name을 찾습니다. Duplicate·list failure·physical/cancel/source failure는 missing으로 바꾸지 않습니다.

**BW의 direction 추가 gate는 정책 조회 뒤** 수행합니다. Omitted 또는 null direction은 그대로 추가 gate 없이 POST합니다. Nonnull false/0/empty string/array/object도 fresh 전체 `qos-bw-limit-direction` 조회를 수행합니다. 확장이 없으면 direction만 제거하고 `DirectionDropped=true`로 기록합니다. 확장이 있으면 raw 값을 그대로 유지합니다. Optional gate의 HTTP/physical 오류는 direction 제거로 숨기지 않습니다. Minimum의 direction에는 이 추가 gate가 없습니다. Required max_kbps/min_kbps/dscp_mark와 명시 burst 0/null은 POST body에 보존됩니다.

Parent ID가 null/false/0일 때 child create는 Source의 URI `%s` 서식을 따라 각각 None/False/0 segment를 사용합니다. 이는 정책 member DELETE의 falsey URL join과 다른 계약입니다. 큰 integer를 float64로 바꾸지 않습니다. Go는 parent를 한 segment로 escape하고 blank/control/traversal 경로를 거부합니다. Source의 slash/percent URI 포맷이 임의 경로로 바뀔 수 있는 차이는 아래의 경로 정책 차이로 명시합니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=("bandwidth", "dscp", "minimum"), default="bandwidth")
parser.add_argument("--policy", required=True)
parser.add_argument("--value", default="0")
parser.add_argument("--kwargs", default="{}")
args = parser.parse_args()
value = json.loads(args.value)
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "bandwidth":
        result = conn.create_qos_bandwidth_limit_rule(args.policy, value, **kwargs)
    elif args.kind == "dscp":
        result = conn.create_qos_dscp_marking_rule(args.policy, value, **kwargs)
    else:
        result = conn.create_qos_minimum_bandwidth_rule(args.policy, value, **kwargs)
    print(json.dumps(result.to_dict(), indent=2))
```

`python main.py --cloud dev --policy gold --kind bandwidth --value 1000 --kwargs '{"max_burst_kbps":0,"direction":"egress"}'`

`python main.py --cloud dev --policy gold --kind dscp --value 32`

`python main.py --cloud dev --policy gold --kind minimum --value 1000 --kwargs '{"direction":"egress"}'`

## 독립 Go main

[설치 안내](../docs/install.md)에 따라 module을 준비한 뒤 main.go로 저장합니다.

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
	value := flag.String("value", "0", "JSON required max_kbps/dscp_mark/min_kbps")
	kwargs := flag.String("kwargs", "{}", "JSON optional cloud kwargs")
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
	required := json.RawMessage(*value)
	option := network.WithQoSRuleCreateJSON(json.RawMessage(*kwargs))
	var result *network.QoSRuleCreateResult
	switch *kind {
	case "bandwidth":
		result, err = service.CreateQoSBandwidthLimitRuleRecord(ctx, *policy, required, option)
	case "dscp":
		result, err = service.CreateQoSDSCPMarkingRuleRecord(ctx, *policy, required, option)
	case "minimum":
		result, err = service.CreateQoSMinimumBandwidthRuleRecord(ctx, *policy, required, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(result.Rule, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

`go run . -cloud dev -policy gold -kind bandwidth -value 1000 -kwargs '{"max_burst_kbps":0,"direction":"egress"}'`

`go run . -cloud dev -policy gold -kind dscp -value 32`

`go run . -cloud dev -policy gold -kind minimum -value 1000 -kwargs '{"direction":"egress"}'`

이 main은 실제 POST를 수행합니다. 공개 low-level helper `rules.API.CreateCloudBandwidthLimitRecord`, `CreateCloudDSCPMarkingRecord`, `CreateCloudMinimumBandwidthRecord`는 이미 resolved parent와 translated cloud attrs를 받는 좁은 leaf이며, 일반 Proxy raw attrs create 전체의 완료 판정으로 승격하지 않습니다. 일반 애플리케이션은 위 cloud method를 사용합니다.

## 권한과 실제 증거

[고정 Neutron 27 QoS 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L226-L245)의 create_policy_bandwidth_limit_rule, [DSCP](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L361-L380), [Minimum](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L438-L457)은 project scope의 ADMIN_OR_PARENT_OWNER_MANAGER입니다. [manager 규칙](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L56-L68)은 QoS parent 소유 project의 manager 또는 admin을 요구합니다. Historic deprecated ADMIN_ONLY와 구분합니다. Shared policy를 읽을 권한이 child rule 생성 권한을 보장하지 않습니다. 배포 정책과 driver·서버 검증이 최종 수락을 결정하며 SDK가 역할을 확대하지 않습니다.

Request body는 uncoerced cloud 입력이고 Resource는 Source descriptors/defaults와 seed를 병합한 view입니다. Partial response가 required rate를 생략하면 Resource의 rate는 입력 seed에서 유지되고 Wire에 그 rate를 추가하지 않습니다. Response id:null은 null로 덮습니다. Missing name/direction/burst는 Source descriptor의 None/null이며 SDK가 egress나 burst 0을 response로 만들지 않습니다.

Response의 unknown qos_policy_id는 actual Wire에 그대로 보관하지만 Resource getter view는 원래 JSON URI parent를 읽습니다. Request URI formatting으로 None/False/0 spelling을 사용하더라도 URI getter는 원래 null/false/0입니다. Selected microversion과 owned null location이 response의 같은 이름을 덮는 경우에도 Wire의 actual 값은 유지합니다. Carrier Policy는 lookup의 별도 owned evidence이며 rule envelope/header 변경이 그 evidence를 바꾸지 않습니다.

Selected source/API/header를 option callback 전에 고정합니다. 응답에 대한 read/Close 오류·취소·selected source 변경은 success/missing으로 숨기지 않고 retry/reauth hooks를 중단합니다. Native401은 Unwrap이 없는 ErrUnableToReauthenticate의 ErrOriginal(value ErrUnexpectedResponseCode), ErrReauth를 보존합니다. Failed mutation의 status/body/header를 유지하고 Resource/Wire를 지어내지 않습니다. Clean retry와 live token renewal은 유지하며 route 변경은 원래 rejected receipt와 함께 중단합니다.

## 지원 범위와 검증

완성 대상은 위 cloud create 세 method입니다. 기존 native API, typed baseline, update/delete/details helper, general Proxy raw attrs 생성 판정은 변경하지 않습니다. Generic mutable Python Resource/class/adapter/cache/current_location 및 custom runtime object는 SDK-R1/SDK-C1/SDK-S1의 공통 기반 remaining으로 별도 추적합니다. Complete UTF-8 JSON, selected service-relative URI/header protocol, owned immutable snapshots와 JSON object wire order는 명시한 Go 정책입니다. 동적 class 속성 dispatch를 일반 raw unknown Body preservation과 같은 것으로 주장하지 않습니다. Full SDK supported 개수를 이 세 mapping으로 올리지 않습니다.

실제 pinned Python cloud decorator·Neutron Proxy.find/create·Resource.find/new/create를 Python 3.14.8에서 실행한 62 cases를 비교했습니다. HTTP adapter/extension generator만 fixture로 대체했으며 live OpenStack/Keystone 권한 증거는 아닙니다. Go 계약은 기존 Gophercloud public testhelper-backed internal/testcloud, nativeRuleAPI fixture와 실제 response body observer를 재사용합니다. Cloud 8그룹/leaf 2그룹, 36 physical/cancel/source subcases, 12 clean retry/reauth/route/Source199 subcases를 준비했습니다. 실제 Go 집중 검증·독립 main build·전체 repository gate는 root의 실행 ledger를 기준으로 기록하며 이 문서가 미실행 PASS를 주장하지 않습니다.

Source: [BW cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1620-L1668), [DSCP cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1844-L1879), [Minimum cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2040-L2077), [Source Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py).


최종 root 검증에서 정확한 가이드 Go main2개 build·Python AST2개와 저장소204개 실제 test package 전체 make check가 PASS입니다. 실행 로그는 `/private/tmp/go-openstacksdk-cycle6-check-first.log`, 예제 hash receipt는 `...cycle6-exact-guides/build-receipt.json`입니다. 실제 인증·live cloud·전체 SDK supported 판정은 별도 잔여로 유지합니다.
