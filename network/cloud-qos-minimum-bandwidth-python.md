# Cloud QoS minimum bandwidth 규칙: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud `v2.15.0`입니다. [고정 cloud Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)는 SDK가 확장 조회·정책 해석·규칙 조회·검색을 순서대로 수행하는 기준입니다.

| Python cloud | Go network.Service |
| --- | --- |
| `list_qos_minimum_bandwidth_rules(policy, filters=None)` | `ListQoSMinimumBandwidthRules(ctx, policy, options...)` |
| `search_qos_minimum_bandwidth_rules(policy, rule_id=None, filters=None)` | `SearchQoSMinimumBandwidthRules(ctx, policy, ruleID, options...)` |
| `get_qos_minimum_bandwidth_rule(policy, rule_id)` | `GetQoSMinimumBandwidthRule(ctx, policy, ruleID, options...)` |
| 생략·falsey filters | 옵션 생략 또는 `WithMinimumBandwidthCloudFiltersJSON(raw)` |
| dictionary filters와 Proxy formal kwargs | `WithMinimumBandwidthCloudFiltersJSON(raw)` |
| 전체 DTO 교체 | `WithMinimumBandwidthCloudOpts` |
| 연결된 computed location | `WithMinimumBandwidthCloudCurrentLocation` |

전체 SDK는 `sdk.Connect`와 `conn.Network(ctx)`로 같은 facade를 사용하고, 이미 선택한 client는 `network.New(client)`로 독립적으로 사용합니다. 페이지별 builder나 개발자가 작성하는 decoder·확장 분기는 필요하지 않습니다. 각 메서드는 `GET extensions`의 **전체 페이지**를 매번 읽고 `qos` 존재를 확인한 다음 정책을 찾습니다. 정책 Find는 GET-first와 정확한 ID/name 목록 fallback이며 중복 이름을 임의로 고르지 않습니다. 목록·검색의 정책 Find는 ignore_missing=True이고 단건 조회는 Source의 ignore_missing=False를 사용합니다. 누락 정책이나 unavailable extension은 규칙 요청 전에 typed 오류입니다. 단건 규칙 GET은 누락을 오류로 유지하고 규칙 목록 fallback을 하지 않습니다.

[Neutron 27 고정 read 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)은 규칙의 project scope와 parent-owner reader/admin 규칙을 정의합니다. 실제 배포의 정책과 인증이 요청 권한을 결정합니다. 이 단위는 목록·검색·조회만 구현합니다.

## Source filters와 Proxy generator

생략·null·false·0·CPython float underflow·빈 string/list/dict는 필터 없음입니다. truthy filters는 mapping이어야 하며 문자열 expression이나 truthy scalar/list는 extension/policy 단계 후 `**filters` 오류가 됩니다. `id`, `name`, `min_kbps`, `direction`는 untyped Body local filters이며 server query로 바꾸지 않습니다. server query는 선언된 limit/marker입니다. unknown kwargs는 list에서 버리지만 search의 추가 cloud mapping에서는 missing attribute 오류가 될 수 있습니다. 순서별 short-circuit와 nested dictionary, CPython JSON-loaded 숫자/bool/float 비교를 보존합니다. 검색은 모든 목록 페이지를 수집한 뒤 ID/name 정확 일치 또는 fnmatch glob을 검사하고 원래 filters로 다시 mapping을 수행합니다. callback은 한 번만 실행합니다.

`ProxyMinimumBandwidthRecords`와 `WithMinimumBandwidthProxyQuery`는 Source Proxy lane입니다. filters object의 headers·microversion·base_path·paginated·max_items 및 한 번의 `__conflicting_attrs` 복원을 처리합니다. bound qos_policy/qos_policy_id, Proxy resource_type/self와 Resource cls/session 중복은 오류입니다. 복원된 qos_policy_id는 Source처럼 URI 값을 바꿀 수 있습니다. `%(<key>)s` 경로는 JSON-domain kwargs에서 SDK가 바인딩하며 URI 값을 안전하게 escape합니다. 빈 base_path는 rule HTTP 없이 빈 목록입니다. falsey max_items는 제한 없음이고 true는 1입니다. continuation도 선택한 collection 경로에 고정합니다. 기존 typed List…Records의 raw formal key 거부 정책은 유지합니다.

Source Proxy JMESPath는 아직 읽지 않은 **opaque generator**에 적용됩니다. HTTP를 먼저 읽어 JSON 배열로 바꾸지 않습니다. 아래 HTTP 수는 extension/policy 이후의 rule 요청만 셉니다.

| object의 `jmespath_filters` | cloud list 결과 | rule HTTP |
| --- | --- | --- |
| 생략·falsey·문자열 아닌 값, `@` | 실제 규칙 목록 | 전체 목록 GET |
| JSON array literal | JSON 값 목록 | 0 |
| Unicode 문자열 literal | Unicode 문자 목록 | 0 |
| JSON object literal | key 목록; Go lexical 순서 | 0 |
| `to_array(@)` 또는 `[@]` | opaque generator를 포함하는 Values; Value 없음 | 반환 시 0, Records 순회 시 GET |
| `[*].id`, `[0]`, `type(@)`, scalar/null literal | noniterable 오류 | 0 |
| `length(@)` | opaque generator type 오류 | 0 |
| `to_string(@)` | Go의 opaque 직렬화 오류 | 0 |

검색의 ID phase가 비었으면 dictionary filter는 falsey literal(null/false/0/빈 list/dict)을 먼저 건너뜁니다. ID가 있으면 그 전에 get() 오류가 발생합니다. nested opaque generator에서는 key membership 검사가 실제 generator를 소비하고 unknown-key 오류가 되며, 소비한 규칙의 실제 Inventory·receipt를 보존합니다.

`to_string(@)`가 Python generator의 프로세스 주소를 포함한 repr 문자열을 만드는 동작은 Go에서 합성하지 않습니다. JSON domain 밖의 Python 객체·repr/session/mutable Resource 계약은 SDK-R1/C1/S1 범위에 남습니다. JMESPath literal을 사용해도 extension/policy 조회는 먼저 수행합니다. top-level 문자열 filters가 JMESPath에 직접 전달된다고 해석하지 않습니다.

## 반환·소유권·오류

`MinimumBandwidthCloudResult`는 `Extensions`·`Policy`·`Inventory`와 선택 결과 `Rules`·`Rule`·`Values`·`Value`를 구분합니다. literal JSON이나 opaque generator에 가짜 Wire/Envelope/Header/status를 만들지 않습니다. `Value`는 완전히 JSON으로 표현할 수 있는 성공 결과에만 있으며 빈 목록은 `[]`, 실패는 nil입니다. `LastRule`은 마지막으로 실제 yield된 규칙 receipt입니다. 후속 HTTP 실패의 body/Header/status는 오류에 원문으로 남고 앞 페이지 Inventory는 partial evidence로 남습니다. partial inventory를 검색 성공으로 표시하지 않습니다.

record의 `Resource`는 Source default/alias view, `Wire`는 실제 JSON fields이며 `Envelope`, `Header`, `StatusCode`는 실제 응답입니다. 누락 Body 값은 null, `min_kbps`/`direction` 값을 int로 강제 변환하지 않고 unknown extension fields도 보존합니다. list의 qos_policy_id는 URI 상태여서 Body에 합성하지 않습니다. standalone client는 computed location null을 유지합니다. 연결된 사용은 CurrentLocation을 SDK 옵션으로 snapshot합니다. 이 Source 클래스에는 project_id/availability_zone descriptor가 없으므로 unknown wire 필드로 location의 project/zone을 바꾸지 않으며 configured current location의 zone을 보존합니다. Wire의 remote location은 관측값으로 남습니다.

선택 결과와 Inventory/LastRule은 raw bytes, Resource/Wire, Header를 독립적으로 소유합니다. 입력 JSON·옵션 DTO·location pointer를 이후 수정해도 이미 캡처한 결과를 바꾸지 않습니다. original service/client/header와 context guard는 모든 종속 요청과 나중에 소비하는 generator까지 유지됩니다. 취소의 원인과 source drift는 성공으로 바꾸지 않습니다. 실제 read/Close fault에서는 native pointer401의 ErrOriginal/ErrReauth와 원문 body/Header/status를 보존하며 retry/backoff/reauth callback을 실행하지 않습니다. 이미 처리된 body를 다시 읽거나 재전송하지 않습니다.

Go는 policy/rule ID를 안전한 nonblank string으로 받고 header를 string-valued mapping, microversion/base_path를 nullable string, truthy max_items를 nonnegative integer 또는 true로 제한합니다. 문자열 경로 formatting 이외의 Python numeric percent formatting은 명시적 unsupported 오류이며 route의 외부 origin·query·fragment·traversal는 거부합니다. 이런 typed/경계 검증이 일부 Python 동적 오류보다 먼저 발생할 수 있습니다. 안전한 페이지 한도·cycle 검증·HTTP Link 상호운용은 기존 공통 계층을 재사용합니다. dict key 결과와 key 기반 JMESPath 함수는 Go lexical 순서를 사용하며 deprecated 경고는 출력하지 않습니다. 실제 인증 cloud acceptance는 별도 검증입니다.

## 독립 Python main

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
    with openstack.connect(cloud=args.cloud) as conn:
        filters = json.loads(args.filters)
        if args.mode == "list":
            values = conn.list_qos_minimum_bandwidth_rules(args.policy, filters)
        elif args.mode == "search":
            values = conn.search_qos_minimum_bandwidth_rules(args.policy, args.rule or None, filters)
        else:
            if not args.rule:
                parser.error("--rule required for get")
            values = [conn.get_qos_minimum_bandwidth_rule(args.policy, args.rule)]
        for value in values:
            print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value,
                             indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 전체 SDK 연결에서 실행하는 독립 Go main

```go
package main

import (
 "context"
 "encoding/json"
 "flag"
 "fmt"
 "log"
 "os"
 "time"

 sdk "github.com/JSYoo5B/go-openstacksdk"
 "github.com/JSYoo5B/go-openstacksdk/network"
)
func main() {
 cloud:=flag.String("cloud",os.Getenv("OS_CLOUD"),"clouds.yaml cloud")
 policy:=flag.String("policy","","QoS policy ID or name")
 rule:=flag.String("rule","","rule ID or fnmatch pattern")
 mode:=flag.String("mode","list","list, search or get")
 filters:=flag.String("filters","null","JSON-domain Source filters")
 flag.Parse()
 if *policy=="" {log.Fatal("-policy required")}
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
 defer cancel()
 var connect []sdk.ConnectionOption
 if *cloud!=""{connect=append(connect,sdk.WithCloud(*cloud))}
 conn,err:=sdk.Connect(ctx,connect...)
 if err!=nil{log.Fatal(err)}
 service,err:=conn.Network(ctx)
 if err!=nil{log.Fatal(err)}
 location,err:=conn.CurrentLocation()
 if err!=nil{log.Fatal(err)}
 options:=[]network.MinimumBandwidthCloudOption{network.WithMinimumBandwidthCloudCurrentLocation(location)}
 var result *network.MinimumBandwidthCloudResult
 switch *mode {
 case "list":
  options=append(options,network.WithMinimumBandwidthCloudFiltersJSON(json.RawMessage(*filters)))
  result,err=service.ListQoSMinimumBandwidthRules(ctx,*policy,options...)
 case "search":
  options=append(options,network.WithMinimumBandwidthCloudFiltersJSON(json.RawMessage(*filters)))
  result,err=service.SearchQoSMinimumBandwidthRules(ctx,*policy,*rule,options...)
 case "get":
  if *rule==""{log.Fatal("-rule required for get")}
  result,err=service.GetQoSMinimumBandwidthRule(ctx,*policy,*rule,options...)
 default:log.Fatal("mode must be list, search or get")
 }
 if err!=nil{
  if result!=nil{fmt.Fprintf(os.Stderr,"observed extensions=%d rules=%d\n",len(result.Extensions),len(result.Inventory))}
  log.Fatal(err)
 }
 if result.Value==nil{fmt.Printf("opaque result: %d values\n",len(result.Values));return}
 fmt.Println(string(result.Value))
}
```

예: `python main.py --policy gold --mode search --rule 'rule-*' --filters '{"direction":"egress"}'` 또는 `go run . -policy gold -mode search -rule 'rule-*' -filters '{"direction":"egress"}'`. getter는 `-mode get -rule <literal-id>`로 실행합니다. 코드 예제의 compile/fixture 검증은 인증된 실제 cloud 실행을 의미하지 않습니다.

[계약 테스트](cloud_qos_minimum_bandwidth_test.go)는 기존 Gophercloud testhelper 기반 testcloud/Neutron scripted HTTP fixture를 재사용합니다. 17개 그룹은 fresh 전체 extension gate, policy fallback/strictness, literal getter404, 전체 paging, ordered/nested/loaded mapping, reachable formal kwargs와 URI 복원, opaque JMES 및 lazy generator, falsey max_items와 경로 binding, raw filter/location/result ownership, source/cancel guard, 실제 후속 200·203·401·404·429·503 Content-Length 단절과 fault callback 차단을 검증합니다.
