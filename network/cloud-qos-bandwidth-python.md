# Cloud QoS bandwidth-limit 목록·검색·조회

기준은 openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1535)와 Gophercloud v2.15.0입니다. 세 기능은 read API이며 생성·수정·삭제를 포함하지 않습니다.

| Python | Go network Service | 반환 |
|---|---|---|
| `conn.list_qos_bandwidth_limit_rules(policy, filters)` | `service.ListQoSBandwidthLimitRules(ctx, policy, options...)` | eager 목록, 실제 rules·parent·extension 증거 |
| `conn.search_qos_bandwidth_limit_rules(policy, rule_id, filters)` | `service.SearchQoSBandwidthLimitRules(ctx, policy, ruleID, options...)` | 같은 목록 후 ID/glob/name와 local filter |
| `conn.get_qos_bandwidth_limit_rule(policy, rule_id)` | `service.GetQoSBandwidthLimitRule(ctx, policy, ruleID, options...)` | literal rule GET, 404는 오류 |

전체 SDK에서는 `conn.Network(ctx)`로 같은 Service에 접근합니다. 각 호출은 QoS extension의 **전체 목록을 먼저 소비**하고 `find_qos_policy(..., ignore_missing=True)` 순서로 policy를 해석합니다. extension이 앞 페이지에 있어도 뒤 페이지를 읽습니다. 뒤 페이지 오류·잘못된 alias는 숨기지 않습니다. policy를 찾지 못하면 typed NotFound 오류이며 rule 요청은 하지 않습니다. ambiguous policy도 rule을 선택하지 않습니다. 조회에서도 extension·policy 단계를 생략하거나 cache하지 않습니다.

`QoSBandwidthLimitResult`의 `Extensions`, `Policy`, `Inventory`는 실제 확인한 단계 증거입니다. `Rules`는 실제 선택한 records이고 `Rule`은 단일 GET record입니다. `Value`는 JSON 결과이며 `Values`에는 JSON 값이나 opaque generator를 담을 수 있습니다. literal JSON에 resource·HTTP receipt를 합성하지 않습니다. 단계 오류에는 성공한 선택 결과를 표시하지 않으며 이미 확인한 extension/policy/rule inventory는 보존합니다. policy None에 대한 HTTP 응답을 새로 만들지 않습니다.

`WithQoSBandwidthLimitFilters(raw)`는 concrete JSON filter입니다. SDK가 값을 복사하고 extension/policy 단계 뒤에 kwargs를 확장합니다. 생략·null·false·0·빈 문자열/목록/객체와 CPython float underflow는 필터 없음입니다. truthy 값은 object여야 합니다. `direction`, `max_kbps`, `max_burst_kbps`, `id`, `name`은 Source Body local filter이며 서버로 보내지 않습니다. 서버 query는 limit/marker입니다. unknown kwargs는 Source list가 버리지만 search의 두 번째 dictionary filter에서는 unknown key 오류가 될 수 있습니다. 필터 없는 빈 성공 결과는 `[]`입니다.

Search는 Source의 list를 그대로 호출한 뒤 `_filter_list`의 ID/name exact·glob와 ordered dictionary 부분 일치를 적용합니다. callback을 두 번 실행하지 않습니다. 숫자·bool과 float precision은 CPython JSON-loaded 비교를 사용합니다. truthy 문자열 필터는 list 단계의 `**filters`가 불가능하므로 policy 조회 뒤에 오류입니다. 이 helper의 문자열 필터를 standalone JMESPath search처럼 처리하지 않습니다.

응답은 기존 [bandwidth-limit record API](v2/qos-bandwidth-limit-records-python.md)를 재사용합니다. max_kbps/max_burst_kbps/direction/id/name은 Source에서 untyped이며 숫자로 강제 변환하지 않습니다. 누락 Body는 null, unknown extension은 보존합니다. Source proxy 경로의 computed location은 standalone Service에서 null이며 raw location도 logical view에서 null로 바뀝니다. actual Wire에는 raw 값을 보존합니다. qos_policy_id는 URI parent 상태이며 list Body에 합성하지 않습니다. 실제 envelope/header/status와 논리 속성을 구별합니다.

## Source proxy의 formal kwargs와 generator

`ProxyBandwidthLimitRecords`와 `WithBandwidthLimitProxyQuery`는 Source 형식을 처리하는 별도 lane입니다. 기존 `ListBandwidthLimitRecords`의 typed raw base_path 거부·200..399 상태 정책을 유지합니다. concrete `WithBandwidthLimitListBasePath`는 실제 service 내부 상대 경로를 소유합니다. Source proxy는 base_path·headers·microversion을 formal에서 소비하고, 입력 순서대로 뒤 concrete 옵션이 우선합니다. 선택 경로는 continuation에서도 유지하며 외부 host·query·fragment·위험한 경로는 preflight 오류입니다. Source proxy lane은 100..399, 기존 literal getter는 200..399 상태 정책을 사용합니다.

Proxy는 JMESPath를 아직 읽지 않은 generator에 평가합니다. cloud helper는 그 값에 Python `list()` 변환을 적용합니다. 전체 cloud 호출은 먼저 extension/policy HTTP를 수행하므로 아래 표의 HTTP 수는 **rule 요청만** 셉니다.

| object 안의 jmespath_filters | cloud list 결과 | rule HTTP |
|---|---|---|
| 생략·falsey·문자열 아닌 값, `@` | 실제 rules 전체 수집 | 목록 GET |
| `` `[1,null,{"id":"literal"}]` `` | JSON 값 목록, Rules 없음 | 0 |
| `'한😀'` | Unicode 문자 목록 | 0 |
| object literal | key 목록, Go lexical 순서 | 0 |
| `to_array(@)` | generator pointer 배열, Value 없음 | 반환 시 0, 명시 순회 시 GET |
| `[*].id`, `type(@)`, scalar/null literal | noniterable 오류 | 0 |
| `length(@)`, `to_string(@)` | opaque 타입/직렬화 오류 | 0 |

Search의 ID 단계가 없을 때 dictionary filter는 falsey literal 항목을 제외합니다. nested generator의 key membership는 실제 iterator를 소비한 뒤 unknown key 오류를 반환합니다. ID가 있으면 generator에 `.get`을 할 수 없어 rule HTTP 전에 오류입니다.

nested generator의 `.Records`는 캡처한 context·원래 service guard를 그대로 사용합니다. 소비 전에 취소하거나 client/API를 바꾸면 요청이 나가지 않습니다. unknown extension과 local 필터는 전체 paging과 실제 response handling을 공통 계층에서 처리합니다. 물리 read/close 실패는 실제 bytes·native 오류와 원인을 보존하고 retry/reauth callback을 실행하지 않습니다. 일반 404 rule GET은 Find fallback이나 missing 허용으로 바꾸지 않습니다.

## 독립 Python

```python
import argparse
import json
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", default="")
    parser.add_argument("--mode", choices=("list", "search", "get"), default="list")
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters)
    if args.mode == "list":
        values = conn.list_qos_bandwidth_limit_rules(args.policy, filters)
    elif args.mode == "search":
        values = conn.search_qos_bandwidth_limit_rules(args.policy, args.rule or None, filters)
    else:
        if not args.rule:
            parser.error("--rule required for get")
        values = [conn.get_qos_bandwidth_limit_rule(args.policy, args.rule)]
    for value in values:
        print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
```

## 전체 Connection에서 실행하는 독립 Go main

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
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "rule ID or search pattern")
	mode := flag.String("mode", "list", "list, search or get")
	filters := flag.String("filters", "null", "Source cloud filters JSON")
	flag.Parse()
	if *policy == "" {
		log.Fatal("--policy required")
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
	var result *network.QoSBandwidthLimitResult
	switch *mode {
	case "list":
		result, err = service.ListQoSBandwidthLimitRules(ctx, *policy, network.WithQoSBandwidthLimitFilters(json.RawMessage(*filters)))
	case "search":
		result, err = service.SearchQoSBandwidthLimitRules(ctx, *policy, *rule, network.WithQoSBandwidthLimitFilters(json.RawMessage(*filters)))
	case "get":
		if *rule == "" {
			log.Fatal("--rule required for get")
		}
		result, err = service.GetQoSBandwidthLimitRule(ctx, *policy, *rule)
	default:
		log.Fatal("--mode must be list, search or get")
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.Value == nil {
		log.Fatal("opaque result: consume QoSBandwidthLimitGenerator.Records explicitly")
	}
	output, err := json.MarshalIndent(result.Value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(output))
}
```

## 검증과 차이

기존 `pythonCloudBClient`와 공개 Gophercloud testhelper 기반 testcloud를 재사용합니다. 고정 Source의 실제 cloud/Proxy methods로 kwargs·generator·선택·호출 순서 41건을 비교했습니다. 이 Python 비교는 extension availability/policy lookup과 transport 대신 일부 실제 Resource generator를 사용하므로 실제 HTTP 검증과 구별합니다. 테스트는 extension 전체 소비·policy missing/ambiguity·parent/paging·Source local 필터와 raw defaults·literal GET404·generator/JSON 값 구별·base_path·옵션 소유/취소/client guard·실제 truncation401과 늦은 rule page 오류를 검증합니다. 실제 cloud의 QoS policy/권한은 실행하지 않았습니다.

Go context/error와 typed JSON 입력, protected route/header 선택은 Python 동적 class/session/임의 객체 입력과 다릅니다. JMESPath object key는 Go lexical이며 Source insertion order와 다를 수 있습니다. 기존 공통 strict syntax·exact JSON numeric evaluator·custom Resource type 함수 차이는 유지합니다. 계산된 전체 Connection location 및 mutable Resource/session 전체 모델은 공통 SDK-C1/R1 범위이며 standalone Service의 null location과 구별합니다. Source URI나 constructor formal 충돌은 처리하지만 caller가 Resource builder나 extension interface를 구현할 필요는 없습니다.
