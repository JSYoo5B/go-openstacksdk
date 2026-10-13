# Cloud QoS rule type 목록: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud v2.15.0입니다. `network.Service.ListQoSRuleTypeValues`가 cloud helper의 extension 확인, filters 기본값, 원본 kwargs, `list()` 결과 변환을 소유합니다. 별도 interface builder나 호출자 측 extension 검색이 필요하지 않습니다.

| Python | Go |
|---|---|
| `conn.list_qos_rule_types(filters=...)` | `conn.Network(ctx)` 뒤 `service.ListQoSRuleTypeValues(ctx, network.WithQoSRuleTypeListFilters(raw))` |
| 원본 filters JSON / 기본 인자 | `WithQoSRuleTypeListFilters(raw)` / 옵션 생략 |
| concrete 옵션 전체 복사 | `WithQoSRuleTypeListOpts(network.QoSRuleTypeListOpts{Filters: raw})` |
| 일반 실제 Resource 목록 | `result.RuleTypes`, 실제 전체 목록 `result.Inventory` |
| JSON literal·문자열·object의 `list()` 변환 | `result.Values`, 완전히 JSON인 경우 `result.JSON` |
| 수행한 extension 조회 | `result.Extensions` |
| service proxy `conn.network.qos_rule_types(**query)` | `service.API.QoSRuleTypes.ProxyListRecords(ctx, ruletypes.WithProxyListQuery(...))` |
| typed 목록의 명시적 route 변경 | `ruletypes.WithRecordListBasePath(path)` |

## 기본 정책과 관리자 후속 범위

[Neutron 27.0.0 기본 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)은 `get_rule_type`에 project scope의 `role:reader`를 적용합니다. Rule type은 특정 project의 소유물이 아니므로 policy owner 조건을 붙이지 않습니다. 목록의 실제 종류와 driver 지원 정보, 활성 extension은 배포가 결정합니다.

단일 rule type 상세 조회는 추가 제한이 있습니다. [같은 버전의 stock QoSPlugin](https://github.com/openstack/neutron/blob/27.0.0/neutron/services/qos/qos_plugin.py)의 `get_rule_type`은 `context.is_admin`이 아니면 `NotAuthorized`를 발생시킵니다. 따라서 cloud `get_qos_rule_type_details`는 관리자 단계로 보류했고 이번 사용자 API 구현이나 완료 수에 포함하지 않습니다. 기존 낮은 수준의 `API.QoSRuleTypes.GetRecord`가 있는 것과 cloud helper를 완성한 것은 구별합니다.

고정 Source의 details helper는 `qos`와 `qos-rule-type-details`를 각각 **새 전체 extension 목록**으로 확인한 뒤 member GET을 수행하고, `filters`는 읽지 않습니다. 이 두 확인을 하나로 합치거나 필터를 상세 GET에 붙이는 대응으로 집계하지 않습니다. Source 일반 Resource·cloud configuration/runtime 전체는 별도 SDK-R1 과제로 남습니다. 실제 배포의 인증·policy·backend 실행은 이번 계약 테스트가 증명하지 않습니다.

## 매 호출의 extension gate와 kwargs

목록 호출은 먼저 `GET /extensions`의 전체 목록과 필요한 다음 페이지를 읽어 `qos` alias가 있는지 확인합니다. 첫 일치 뒤에도 읽으며, 후반의 잘못된 alias나 HTTP 실패를 숨기지 않습니다. 결과를 cache하지 않습니다. Alias가 없으면 `network.ErrUnavailableExtension`에 대응하는 오류이고 관찰한 extension records는 결과에 남습니다. 이 확인이 실패하면 옵션 callback과 filters JSON 해석도 실행하지 않습니다.

옵션 생략·JSON null/false/0/0.0/빈 문자열/배열/객체와 CPython binary64에서 0으로 underflow한 수치는 `{}` kwargs로 취급합니다. Truthy 값은 JSON object여야 합니다. Raw JSON object의 원래 key 순서를 보존하며 server query·Body 로컬 조건·unknown kwargs 분류는 기존 [RuleTypes record reader](v2/qos-rule-types-python.md)와 공통 Source helper를 재사용합니다. null query omission, False/True 값, 반복 fields, id/name 로컬 비교, raw row cap·pagination을 호출자가 직접 구현하지 않습니다.

새 `ProxyListRecords`는 `jmespath_filters`, `headers`, `microversion`, `base_path`, 한 번의 `__conflicting_attrs` 복원을 처리합니다. 기존 typed `ListRecords`·`AllRecords`의 raw reserved key 거부는 유지하고 route가 필요하면 typed `WithRecordListBasePath`를 사용합니다. 원본 Proxy 표현식은 복원한 같은 이름의 attr로 교체되지 않으며 복원한 paginated/base_path는 formal 중복 오류입니다. Header/version/path 값의 실제 검증은 generator 순회 시 수행합니다. JSON 상수 표현식이면 쓰이지 않는 formal 값과 QoS 목록 HTTP를 실행하지 않습니다.

Source와 같은 이름으로 임의 Python class/session을 교체하는 기능은 없고 concrete JSON 옵션과 캡처한 service client/context를 사용합니다. 인증·framing·version 헤더, URL origin/path, UTF-8, cap과 continuation 검증은 공통 Go 정책입니다. 이 프로필은 기존 200..399 상태 범위를 유지하며 Source `<400`의 synthetic 100..199와는 다릅니다. 기존 id/name 로컬 equality는 exact decimal 정책으로, Python loaded float 반올림/underflow equality와 차이가 있습니다.

## Source `list()`의 다형 결과

| Proxy 표현식 | 결과 | HTTP |
|---|---|---|
| 생략·falsey·nonstring·`@` | 실제 records의 Values·RuleTypes·Inventory | 전체 extension GET 후 실제 rule type GET |
| `` `[1,null,true]` `` | `[1,null,true]`, RuleTypes 없음 | extension GET만 |
| `` `{"z":1,"a":2}` `` | object의 key 목록 | extension GET만 |
| `'한😀'`, `''` | Unicode 문자 목록, `[]` | extension GET만 |
| `to_array(@)` | 미소비 `*ruletypes.ProxyListGenerator`를 담은 Values, JSON 없음 | 반환 시 extension GET만, 명시 순회 시 rule type GET |
| `[0]`, `[*].type`, `type(@)`, `` `null` ``, `` `false` `` | noniterable 오류 | extension GET만 |
| `length(@)` | generator 타입 오류 | extension GET만 |

Object key 순서는 명시적 차이입니다. Python `list(dict)`는 삽입 순서여서 위 literal은 `["z","a"]`이고 Go는 lexical 정렬로 `["a","z"]`입니다. 배열과 Unicode 문자 순서는 유지합니다. JMESPath 숫자 evaluator·엄격한 문법과 custom Python 객체 type 함수의 차이는 공통 계약을 따릅니다.

JSON literal은 실제 QoSRuleTypeRecord로 합성하지 않습니다. Record에만 실제 Wire·Envelope·Header·StatusCode가 있고 `Resource`는 기본값·alias·선언 속성의 독립 view입니다. 늦은 페이지 오류면 관찰한 Inventory와 Extensions를 보존하며 성공한 Values·RuleTypes·JSON으로 표시하지 않습니다. 성공한 빈 실제 목록은 nonnil slice와 JSON `[]`입니다. Opaque generator의 명시 순회는 추가 GET을 수행하며 Source와 달리 Go iterator는 재사용할 수 있습니다.

## 독립 Python 프로그램

`clouds.yaml`의 cloud를 지정해 `python qos_rule_types.py --cloud dev --filters='{"type":"bandwidth_limit"}'`로 실행합니다. JSON 상수 목록을 보려면 `` --filters='{"jmespath_filters":"`[1,null]`"}' ``를 사용합니다.

```python
import argparse
import json
import os
import types
import openstack


def print_value(value):
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, ensure_ascii=False, indent=2))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    for value in conn.list_qos_rule_types(filters=json.loads(args.filters)):
        if isinstance(value, types.GeneratorType):
            for record in value:
                print_value(record)
        else:
            print_value(value)


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../docs/install.md)로 module을 준비하고 `go run . -cloud dev -filters='{"type":"bandwidth_limit"}'`로 실행합니다. Source filters는 JSON object 문자열 그대로 전달하며 동일 main으로 `to_array(@)`의 opaque generator를 명시 순회할 수 있습니다.

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
)

func printJSON(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}

func main() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	filters := flag.String("filters", "null", "original Source cloud filters JSON")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, err := service.ListQoSRuleTypeValues(ctx, network.WithQoSRuleTypeListFilters(json.RawMessage(*filters)))
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON != nil {
		var decoded any
		decoder := json.NewDecoder(bytes.NewReader(result.JSON))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err != nil {
			log.Fatal(err)
		}
		printJSON(decoded)
		return
	}
	for _, value := range result.Values {
		if generator, ok := value.(*ruletypes.ProxyListGenerator); ok {
			for record, err := range generator.Records {
				if err != nil {
					log.Fatal(err)
				}
				printJSON(record.Resource)
			}
		} else {
			printJSON(value)
		}
	}
}
```

## 검증 범위

테스트는 공개 Gophercloud testhelper를 사용하는 기존 testcloud·native transport fixture를 재사용합니다. Cloud 10개·Proxy 5개 테스트 그룹은 per-call 전체 extension gate, kwargs/falsey/Source formal 지연, 단일·다형 결과, 옵션 소유권과 guard, 실제 페이지 및 native401의 ErrOriginal/ErrReauth와 물리 Read/Close 오류, faulty retry/reauth callback 억제, 기존 typed 상태/옵션 정책을 검증합니다. Source의 실제 AST 메서드와 실제 QoSRuleType descriptor를 사용해 Python 3.14/JMESPath 1.0.1에서 33개 사례를 대조했고, 그 비교에서는 lazy transport/list만 대체했습니다. 전체 Resource runtime이나 라이브 OpenStack 검증은 아닙니다. Go 테스트·race·독립 main build 결과는 root 검증 ledger에 기록합니다.

Source: [cloud 목록·상세](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L281-L332), [network proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5270), [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866).
