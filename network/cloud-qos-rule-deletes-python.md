# Cloud QoS rule 삭제: Python과 Go

기준은 openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1723)와 Gophercloud v2.15.0입니다. 이 문서는 bandwidth-limit·DSCP marking·minimum-bandwidth의 **삭제 3개**만 다룹니다.

| Python cloud | Go network Service | Source policy find |
|---|---|---|
| `conn.delete_qos_bandwidth_limit_rule(policy, rule_id)` | `DeleteQoSBandwidthLimitRule(ctx, policy, ruleID, options...)` | ignore_missing=True, None이면 NotFound |
| `conn.delete_qos_dscp_marking_rule(policy, rule_id)` | `DeleteQoSDSCPMarkingRule(ctx, policy, ruleID, options...)` | ignore_missing=True, None이면 NotFound |
| `conn.delete_qos_minimum_bandwidth_rule(policy, rule_id)` | `DeleteQoSMinimumBandwidthRule(ctx, policy, ruleID, options...)` | ignore_missing=False, lookup 자체의 NotFound |

전체 SDK에서는 `conn.Network(ctx)`로 같은 Service에 접근합니다. 각 호출은 QoS extension 전체 목록을 먼저 읽고 policy 이름/ID를 Source Find로 해석합니다. extension을 앞 페이지에서 찾았어도 뒤 페이지까지 확인합니다. policy가 없거나 ambiguous이면 DELETE를 보내지 않습니다. policy의 404는 cloud false가 아닙니다. [minimum-bandwidth Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2115)의 strict policy lookup 차이를 보존합니다.

성공하면 Source true, **깨끗한 rule DELETE404만** false입니다. cloud helper는 leaf에 항상 ignore_missing=False를 전달하고, rule NotFound를 받은 뒤 bool을 결정합니다. 다른 HTTP400/403/408/409/500, 물리 read/close 실패, 취소, 잘못된 ID, callback/client guard 실패는 오류입니다. retry callback의404나 native callback이 OkCodes에404를 추가해 받아들인 응답도 false로 변환하지 않습니다. shared missing 정책이 실제 거부 응답과 accepted/transport/context/혼합 실패를 구별합니다.

`QoSRuleDeleteResult`에는 `Completed`, `Deleted`, `Extensions`, `Policy`, `Receipt`가 있습니다. 오류가 없을 때 Completed=true이며 Deleted가 Source bool입니다. 오류가 있는 result의 기본 Deleted=false를 Source의 missing false로 읽지 않습니다. extension/policy/DELETE의 실제 확인한 증거만 남습니다. missing policy None에 가상의 policy response를 만들지 않으며 preflight 실패에는 DELETE Receipt가 없습니다.

`Receipt`의 Envelope는 actual DELETE bytes이고 Header·StatusCode도 actual response입니다. ACK는 empty·opaque·비JSON일 수 있으므로 Resource/Wire Body로 합성하거나 JSON decode하지 않습니다. successful ACK와 native rejected response, native reauthentication의 ErrOriginal에서 실제 응답을 소유합니다. 물리 실패의 bytes·Read/Close/cancellation 원인과 원래 native401을 유지합니다. retry/reauth callback은 물리 실패 뒤에 실행하지 않으며 깨끗한 retry/renewal은 같은 bodyless method·parent/rule route·선택 header/version을 사용합니다.

## Direct leaf와 옵션

`service.API.QoSRules.DeleteBandwidthLimitRecord`, `DeleteDSCPMarkingRecord`, `DeleteMinimumBandwidthRecord`는 literal policy/rule ID를 받아 **별도 extension·policy 조회 없이** bodyless DELETE를 보냅니다. Source proxy처럼 ignore_missing의 기본값은 true이며 clean404 receipt의 IgnoredMissing=true입니다. `rules.WithQoSRuleDeleteIgnoreMissing(false)`를 주면 native404와 실제 receipt를 반환합니다. cloud helper는 이 strict 설정을 소유하므로 호출자가 missing 정책을 다시 구현하지 않습니다.

network의 `WithQoSRuleDeleteOpts/Headers/Microversion`과 leaf의 같은 이름 factory는 concrete 값과 callback 결과를 복사합니다. Headers는 nil이어도 callback이 직접 key를 할당할 수 있습니다. option error와 동시에 생긴 취소/client 변경을 함께 보존합니다. 인증·framing·다른 서비스 version 제어와 선택 route 변경은 거부하며 헤더/버전은 retry에도 고정합니다. Source cloud helper에는 이런 transport kwargs가 없으므로 Go convenience입니다.

새 Source delete lane은 **유효 HTTP100..399를 허용**합니다. Source `raise_from_response`의 <400 동작에 맞춘 opt-in이며 기존 generated delete·typed record 정책을 바꾸지 않습니다. policy lookup은 기존 Source Find와 captured immutable API를 재사용합니다. body를 만들거나 rule GET/Find로 삭제 전에 검사하지 않습니다.

Go ID는 UTF-8 string이며 blank·control·dot 이동을 거부합니다. Source의 arbitrary Python 객체/None/falsey Resource state와 동일하다고 주장하지 않습니다. slash·space·literal %2F가 있는 **nonblank literal ID**는 한 URL segment로 escape합니다. whitespace-only ID는 Source가 요청할 수 있어도 Go는 오류입니다. 빈 ID를 누락 false로 바꾸거나 이름으로 추측하지 않습니다. URI parent는 실제 resolved policy ID입니다.

## 서버의 권한

[Neutron27 QoS policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)의 세 delete_policy_*_rule은 project scope의 `ADMIN_OR_PARENT_OWNER_MANAGER`입니다. [base policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py)는 parent policy를 소유한 manager 또는 admin을 뜻합니다. deprecated ADMIN_ONLY 규칙과 현재 기본값을 구별합니다. 공유 policy를 읽을 수 있다는 사실만으로 삭제 권한이 생기지 않습니다. 배포의 policy override·QoS driver와 실제 인증 권한을 SDK가 에뮬레이션하지 않으며 403도 false로 숨기지 않습니다.

## 독립 Python

이 예제는 실제 규칙을 삭제합니다. `--kind`, `--policy`, `--rule`을 명시합니다.

```python
import argparse
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--kind", choices=("bandwidth", "dscp", "minimum"), required=True)
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", required=True)
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    methods = {
        "bandwidth": conn.delete_qos_bandwidth_limit_rule,
        "dscp": conn.delete_qos_dscp_marking_rule,
        "minimum": conn.delete_qos_minimum_bandwidth_rule,
    }
    print(methods[args.kind](args.policy, args.rule))


if __name__ == "__main__":
    main()
```

## 전체 Connection에서 실행하는 독립 Go main

Envelope는 opaque일 수 있으므로 아래 예제는 body를 JSON string으로 출력합니다.

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	kind := flag.String("kind", "", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "literal QoS rule ID")
	flag.Parse()
	if *policy == "" || *rule == "" || *kind == "" {
		log.Fatal("--kind, --policy and --rule required")
	}
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var result *network.QoSRuleDeleteResult
	switch *kind {
	case "bandwidth":
		result, err = service.DeleteQoSBandwidthLimitRule(ctx, *policy, *rule)
	case "dscp":
		result, err = service.DeleteQoSDSCPMarkingRule(ctx, *policy, *rule)
	case "minimum":
		result, err = service.DeleteQoSMinimumBandwidthRule(ctx, *policy, *rule)
	default:
		log.Fatal("--kind must be bandwidth, dscp or minimum")
	}
	if err != nil {
		if result != nil && result.Receipt != nil {
			log.Printf("actual DELETE response: %d %s", result.Receipt.StatusCode, string(result.Receipt.Envelope))
		}
		log.Fatal(err)
	}
	output := map[string]any{"completed": result.Completed, "deleted": result.Deleted}
	if result.Receipt != nil {
		output["status"] = result.Receipt.StatusCode
		output["body"] = string(result.Receipt.Envelope)
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## 검증과 공통 경계

공개 Gophercloud testhelper 기반 testcloud와 기존 nativeRuleAPI/nativeRuleWire/pythonBandwidthFaultBody를 재사용합니다. 테스트는 opaque ACK와 Source status opt-in, default/strict missing, 전체 gate·strict minimum policy, bodyless bound URI, 소유권/guard, 실제 physical read/close/native401, accepted/callback404, clean renewal/retry를 검사합니다. 고정 Source의 실제 cloud→Proxy→Resource.delete method를 adapter response로 실행한 51건은 false/true/error와 strict lookup·ID 동작을 비교합니다. 실제 cloud deletion/권한·QoS driver는 실행하지 않았습니다.

Go partial carrier/context/errors와 safe typed ID/header/route 정책은 Source 동적 Resource/class/session state와 다릅니다. mutable Resource·전체 cloud 설정·session/adapter behavior는 SDK-R1/C1/S1 공통 과제입니다. 이 세 helper의 bool·HTTP 단계와 실제 receipt를 그 전체 parity로 부풀리지 않습니다. DELETE에는 새 Resource Body나 computed location을 만들 필요가 없으며 SDK가 prerequisite와 strict child/missing 판정을 소유합니다.
