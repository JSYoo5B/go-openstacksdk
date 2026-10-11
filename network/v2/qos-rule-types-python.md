# QoS rule types: pinned Python and Go

`NetworkV2.QoSRuleTypes`의 `GetRecord`, `FindRecord`, `ListRecords`는 QoS rule type을 Source 속성·기본값과 실제 응답으로 구분하여 반환합니다. 비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud v2.15.0입니다.

| Python | Go |
| --- | --- |
| `conn.network.get_qos_rule_type(name, fields=...)` | `api.GetRecord(ctx, name, WithRecordGetFields(...))` |
| `conn.network.find_qos_rule_type(name)` | `api.FindRecord(ctx, name)` |
| `ignore_missing=False` | `WithRecordFindIgnoreMissing(false)` |
| `conn.network.qos_rule_types(**query)` | `api.ListRecords(ctx, WithRecordListQuery(key, value), ...)` |
| `list(iterator)` | `api.AllRecords(ctx, ...)` |
| `paginated=False`, `max_items=n` | `WithRecordListPaginated(false)`, `WithRecordListMaxItems(n)` |
| `value.to_dict()` | `json.MarshalIndent(value.Resource, "", "  ")` |
| 응답의 원래 JSON·HTTP metadata | `Wire`, `Envelope`, `Header`, `StatusCode` |

QoS rule type은 클라우드가 제공하는 종류 이름을 사용합니다. `get_rule_type`은 project scope의 `role:reader` 정책을 따릅니다. Rule type은 특정 프로젝트에 소유되지 않으므로 QoS policy의 owner 검사와 다릅니다. [Neutron 2025.2 정책](https://docs.openstack.org/neutron/2025.2/configuration/policy.html).

`drivers`에는 배포의 backend 지원 정보를 담을 수 있습니다. 이 정보는 `qos-rule-type-details` 확장이 제공하며 실제 지원 종류와 값은 서버가 결정합니다. [Networking API의 QoS rule types](https://docs.openstack.org/api-ref/network/v2/#qos-rule-types).

## 독립 Python 프로그램

`clouds.yaml`에 `dev` cloud를 준비하고 `python qos_rule_types.py --operation get --name bandwidth_limit --fields type,drivers`로 실행합니다. 기본 동작은 목록입니다.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--operation", choices=["get", "find", "list"], default="list")
parser.add_argument("--name", default="bandwidth_limit")
parser.add_argument("--fields")
parser.add_argument("--type")
parser.add_argument("--all-rules", action="store_true")
parser.add_argument("--all-supported", action="store_true")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
fields = args.fields.split(",") if args.fields else None
if args.operation == "get":
    value = conn.network.get_qos_rule_type(args.name, fields=fields)
    print(value.to_dict())
elif args.operation == "find":
    # The pinned public find method has no **query or fields argument.
    value = conn.network.find_qos_rule_type(args.name, ignore_missing=not args.strict)
    print(None if value is None else value.to_dict())
else:
    query = {}
    if fields:
        query["fields"] = fields
    if args.type is not None:
        query["type"] = args.type
    if args.all_rules:
        query["all_rules"] = True
    if args.all_supported:
        query["all_supported"] = True
    if args.max_items:
        query["max_items"] = args.max_items
    for value in conn.network.qos_rule_types(**query):
        print(value.to_dict())
```

## 독립 Go main

[설치 안내](../../docs/install.md)로 module을 준비하고 `go run . -operation get -name bandwidth_limit -fields type,drivers`로 실행합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
)

func printType(value *ruletypes.QoSRuleTypeRecord) {
    if value == nil { fmt.Println("not found"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    operation := flag.String("operation", "list", "get/find/list")
    name := flag.String("name", "bandwidth_limit", "literal rule type name")
    fields := flag.String("fields", "", "comma-separated fields")
    typeFilter := flag.String("type", "", "server type filter")
    allRules := flag.Bool("all-rules", false, "server all_rules control")
    allSupported := flag.Bool("all-supported", false, "server all_supported control")
    maxItems := flag.Int("max-items", 0, "raw row cap; zero is unlimited")
    strict := flag.Bool("strict", false, "find reports missing")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.QoSRuleTypes
    var selected []string
    if *fields != "" { selected = strings.Split(*fields, ",") }
    switch *operation {
    case "get":
        value, err := api.GetRecord(ctx, *name, ruletypes.WithRecordGetFields(selected...))
        if err != nil { log.Fatal(err) }
        printType(value)
    case "find":
        value, err := api.FindRecord(ctx, *name, ruletypes.WithRecordFindIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        printType(value)
    case "list":
        options := []ruletypes.RecordListOption{ruletypes.WithRecordListMaxItems(*maxItems)}
        if len(selected) != 0 { options = append(options, ruletypes.WithRecordListQuery("fields", selected)) }
        if *typeFilter != "" { options = append(options, ruletypes.WithRecordListQuery("type", *typeFilter)) }
        if *allRules { options = append(options, ruletypes.WithRecordListQuery("all_rules", true)) }
        if *allSupported { options = append(options, ruletypes.WithRecordListQuery("all_supported", true)) }
        for value, err := range api.ListRecords(ctx, options...) {
            if err != nil { log.Fatal(err) }
            printType(value)
        }
    default: log.Fatal("unknown operation")
    }
}
```

## ID·기본값·확장 처리

모델은 상속된 `id`, `name`과 선언된 `type`, `drivers`를 읽고 누락 값을 null로 보존합니다. 리스트의 id 키가 **실제로 없으면** `Resource.id`는 type을 사용합니다. 실제 id가 null·false·0·빈 문자열이면 그 값을 우선합니다. 멤버 Get/Find는 Source처럼 요청한 이름을 id seed로 사용하므로 응답에 type만 있을 때도 Resource.id는 요청한 이름입니다. 응답이 실제 id를 보내면 그 값이 seed를 교체합니다. type 자체는 합치거나 이름을 덮어쓰지 않습니다.

`drivers`는 Source list descriptor를 따릅니다. null과 omission은 null이고, 배열 외 값은 단일 항목 배열로 변환합니다. 중첩 driver·supported_parameters·parameter_values의 객체·배열·null·정수 정밀도는 raw JSON으로 유지합니다. 선언되지 않은 확장도 Resource와 Wire에 독립 복사로 보존합니다. `self`는 Source처럼 Resource에서 제거하고 Wire에 남깁니다. Native typed `GetRuleType`/`ListRuleTypes`는 그대로 별도 계약을 유지합니다.

`Resource.location`은 SDK가 소유하는 null이며 응답의 원래 location은 Wire에 있습니다. Python Connection의 location 및 row constructor의 computed 값을 일반적으로 재현하지 않는 SDK-R1 경계입니다. 멤버의 unknown microversion 역시 Wire에는 실제 값, Resource에는 선택된 버전 문자열 또는 null이 들어갑니다. LIST의 microversion·connection·_synchronized·cls 키는 presence만으로 constructor 충돌 오류이며 null·false도 예외가 아닙니다. 오류에는 실제 페이지 응답이 붙습니다.

Get의 nil/빈 fields는 query를 생략합니다. nonempty fields는 순서·중복·unknown field를 유지하며 Body 속성명과 wire명이 같은 이 모델에서는 그대로 전송합니다. fields는 Resource의 constructor seed가 아닙니다. Find는 pinned 공개 메서드에 임의 **query가 없으므로 Go `RecordFindOpts`도 Query를 제공하지 않습니다. 헤더·microversion 설정은 Go의 명시적 추가 옵션입니다. Find의 override는 GET 단계만 바꾸며 fallback LIST는 원래 선택 버전을 사용합니다.

## 목록·검색·HTTP 증거

목록의 server query는 limit·marker·type·drivers·fields·all_rules·all_supported입니다. id·name은 선언 Body의 local filter이며 unknown kwargs는 값 평가 전에 버립니다. null은 query omission, false는 `False`, zero는 `0`로 구분합니다. `max_items`는 local 조건을 적용하기 전 raw row를 제한하고 limit hint를 줄 수 있습니다. `paginated=false`는 첫 페이지로 종료하며 consumer break도 다음 페이지를 요청하지 않습니다.

Source의 inherited pager를 공통 SDK pager로 재사용합니다. 실제 링크, 서버 limit, 짧은 페이지의 marker fallback을 처리합니다. Marker 역시 **원래 Wire에 id가 없을 때만** type을 사용합니다. 실제 id가 null이면 type으로 바꾸지 않습니다. 고정 origin/path/filter, cursor 반복·변경 검사와 단일 nonempty query 값으로 변환되는 marker 규칙은 명시적 Go guard입니다. Source의 임의 iterable/object marker 인코딩과 permissive adapter는 재현하지 않습니다. Dictionary next links는 Go 확장입니다. 로컬 JSON 숫자 비교는 exact decimal 정책이므로 Python binary64 반올림에 따른 `9007199254740992 == 9007199254740993.0`이나 underflow에 따른 `0 == 1e-4000`은 같지 않습니다. 목록 control/next의 float truthiness는 별도로 Source binary64 규칙을 따릅니다.

Find는 멤버 GET 후 깨끗한 HTTP400·403·404에서만 목록으로 넘어가고 완전한 목록에서 정확한 id 또는 name을 찾습니다. 기본 missing은 nil이며 strict missing은 ErrNotFound, 여러 일치는 ErrAmbiguous입니다. Get에는 fallback이 없습니다. 물리 Read·Close·callback·sticky source 변경·취소 오류가 있으면 404도 missing/fallback이 아니고 실제 status/body/header를 보존합니다. 새 프로필은 이런 오류가 이미 관찰됐을 때 native RetryFunc를 호출하지 않도록 opt-in합니다. 깨끗한 503의 정상 native retry, 일반 retry 헤더와 live token 갱신은 유지합니다.

공통 Neutron reader는 기존 200..399 admission을 유지합니다. Source의 `<400` 처리와 비교하면 synthetic 100..199는 명시적 Go 차이입니다. 모든 400..599 거부 응답의 처리 실패를 관찰합니다. 이 가이드는 실제 클라우드 동작, 임의 Resource subclass·Munch·mutable lifecycle·HTTP cache 또는 전체 SDK 완료를 주장하지 않습니다.

Source: [three proxy declarations](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5212-L5285), [QoSRuleType schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_rule_type.py#L16-L42), [Resource alternate id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L654-L672).

custom 옵션 콜백에 전달되는 Header map과 LIST Query map은 처음부터 초기화되며 이전 콜백이나 factory가 nil로 reset한 뒤에도 새 owned map을 제공합니다. 콜백이 보관한 map을 HTTP reader에서 변경해도 실행 중인 요청·필터나 원래 ServiceClient를 바꾸지 않습니다. 물리적 Read/Close·sticky source·custom cancellation이 함께 생긴 401은 원래 ReauthFunc와 RetryFunc를 호출하지 않고 한 요청·한 Close로 끝납니다. Gophercloud의 `*ErrUnableToReauthenticate`는 Unwrap이 없으므로 `ErrOriginal`에서 실제 GET method/URL/status/body/header를, `ErrReauth`에서 함께 보존된 물리적 원인과 guard를 확인합니다.
