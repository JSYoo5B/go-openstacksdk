# Neutron Extension의 Python/Go 목록·검색

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, native 기준은 Gophercloud `v2.15.0`입니다. `conn.NetworkV2(ctx).Extensions`에서 기존 native `Get`·`List`와 owned record API를 함께 사용할 수 있습니다.

| Python | Go |
| --- | --- |
| `conn.network.extensions(**query)` | `ProxyListRecords(ctx, options...)` |
| 기본 Resource generator 순회 | `ListRecords(ctx, options...)` |
| `list(generator)` | `AllRecords(ctx, options...)` |
| `conn.network.find_extension(name_or_id)` | `FindRecord(ctx, identity)` |
| `ignore_missing=False` | `WithRecordFindIgnoreMissing(false)` |
| Find의 `list_base_path` | `WithRecordFindListBasePath(path)` |
| List의 `limit`, `marker`, `max_items`, `paginated` | `WithRecordListLimit`, `WithRecordListMarker`, `WithRecordListMaxItems`, `WithRecordListPaginated` |
| Body kwargs의 로컬 비교 | `WithRecordListQuery(key, value)` |
| deprecated Proxy `jmespath_filters` | `WithRecordListJMESPath`와 `ProxyListRecords` |
| `value.to_dict()`와 실제 HTTP 증거 | `Resource`와 별도의 `Wire`·`Envelope`·`Header`·`StatusCode` |

Neutron 27.0.0 Extension controller는 GET 목록과 alias 조회에 별도 ADMIN 정책 검사를 넣지 않습니다. 서비스의 인증/middleware/catalog 노출을 그대로 따르며 익명 접근을 보장하지 않습니다. 실제 alias 목록은 배포가 활성화한 extension입니다. SDK가 관리자 역할을 요구하거나 extension/backend를 활성화하지 않습니다. [고정 controller](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/extensions.py#L100-L127).

## 지원 범위와 남은 Find kwargs

`extensions(**query)`의 named list 동작은 owned list/proxy 옵션으로 매핑했습니다. `FindRecord`의 기본 alias/name 검색, GET-first fallback, ignore_missing, 선택 microversion·ListBasePath와 실제 응답 증거는 구현했지만 `find_extension(**query)` 전체는 unresolved입니다. 이 Find 행은 완성된 API 개수에 포함하지 않습니다.

고정 Neutron Proxy는 `**query`를 Resource.find에 전달합니다. 그중 kwargs key `headers`는 member fetch에서 HTTP header 옵션이 아니라 ordinary GET query/constructor seed이며, GET 실패 뒤 Resource.list에 넘어가면 list의 `headers` formal로 소비됩니다. 현재 shared find 엔진은 이 raw kwargs를 예약 key로 거부합니다. `WithRecordFindHeaders`의 concrete HTTP header 옵션은 이 두 단계의 Source kwargs 동작을 대체하지 않습니다.

Fetch formal `requires_id`·`base_path`·`error_message`·`skip_cache`·`resource_response_key`도 named kwargs로 도달할 수 있지만 현재 shared find 엔진은 거부합니다. Source에서는 각각 member URI의 ID 포함 여부·member base path·오류 메시지·adapter cache 제어·response envelope key를 바꾸며, fallback에서는 다시 list 단계의 formal/unknown kwargs 규칙을 따릅니다. Member `base_path`와 이미 구현된 fallback-only `list_base_path`는 별도 동작입니다. 이 kwargs는 후속 공통 구현과 실제 단계별 계약 테스트가 필요합니다.

## 독립 Python

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "find", "proxy"), default="list")
parser.add_argument("--extension", default="qos")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--jmes", default="")
parser.add_argument("--list-base-path")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        query = {}
        if args.list_base_path is not None:
            query["list_base_path"] = args.list_base_path
        value = conn.network.find_extension(
            args.extension, ignore_missing=not args.strict, **query
        )
        print(json.dumps(None if value is None else value.to_dict(), indent=2))
    else:
        query = {}
        if args.mode == "list":
            query["alias"] = args.extension
        else:
            query["jmespath_filters"] = args.jmes
        result = conn.network.extensions(**query)
        if isinstance(result, types.GeneratorType):
            for value in result:
                print(json.dumps(value.to_dict(), indent=2))
        else:
            print(result)
```

## 독립 Go main

[설치 안내](../../../docs/install.md)의 module에서 `go run . -cloud dev -mode find -extension qos -strict`로 실행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "list", "list, find or proxy")
    identity := flag.String("extension", "qos", "alias or exact display name")
    strict := flag.Bool("strict", false, "return an error when missing")
    expression := flag.String("jmes", "", "deprecated Proxy expression")
    listPath := flag.String("list-base-path", "", "Find fallback service-relative path")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := network.Extensions
    switch *mode {
    case "find":
        options := []extensions.RecordFindOption{
            extensions.WithRecordFindIgnoreMissing(!*strict),
        }
        if *listPath != "" {
            options = append(options, extensions.WithRecordFindListBasePath(*listPath))
        }
        value, err := api.FindRecord(ctx, *identity, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil); return }
        printJSON(value.Resource)
    case "list":
        for value, err := range api.ListRecords(
            ctx, extensions.WithRecordListQuery("alias", *identity),
        ) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    case "proxy":
        result, err := api.ProxyListRecords(ctx, extensions.WithRecordListJMESPath(*expression))
        if err != nil { log.Fatal(err) }
        if result.JSON != nil { fmt.Println(string(result.JSON)); return }
        generator, ok := result.Value.(*extensions.RecordListGenerator)
        if !ok { fmt.Printf("opaque expression result: %T\n", result.Value); return }
        for value, err := range generator.Records {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    default:
        log.Fatal("mode must be list, find or proxy")
    }
}
```

## 모델·기본값·alias

Extension은 NetworkResource/TagMixinNetwork를 상속하지 않습니다. 선언 Body는 inherited id/name 및 alias·description·links·updated_at 여섯 개이며 누락/default는 null입니다. Tags·revision·if_match를 선언값으로 만들거나 cast하지 않습니다. 그런 unknown 확장이 실제 JSON에 있으면 그대로 보존합니다. HTTP Header는 실제 evidence에 남습니다.

Alias는 Source alternate ID입니다. 실제 id가 없을 때만 Resource.id가 alias를 읽고, 실제 id:null·false·0·empty string이 있으면 그 값을 우선합니다. Find member 요청은 caller identity를 id seed로 사용하므로 body에 alias만 있으면 Resource.id는 요청한 문자열이고 Wire에는 실제 alias만 있습니다. List는 caller kwargs를 row seed로 사용하지 않습니다. 마지막으로 나타나는 canonical updated_at/wire updated가 Resource.updated_at을 결정합니다.

Links는 `type=list, list_type=dict`입니다. Links 자체 null은 null, mapping은 한 항목 배열, scalar는 한 항목 배열로 만들고 각 element의 nonmapping 값(null 포함)은 빈 객체로 바꿉니다. 예를 들어 `[null, false, {"href":"x"}]`는 `[{}, {}, {"href":"x"}]`가 됩니다. Wire에는 원래 배열이 남습니다. Unknown nested JSON과 큰 정수, timestamp text를 유지합니다.

Resource·Wire·Envelope·Header는 각각 owned snapshot이며 Resource의 값 변경이 실제 증거를 바꾸지 않습니다. Self는 Resource에서 제거하며 현재 location은 SDK-owned null입니다. Python의 mutable Resource/Computed/current_location/cache/dirty lifecycle와 arbitrary class/session injection은 공통 SDK 잔여 계약으로 따로 추적합니다. Owned JSON view를 먼저 완성하므로 lazy descriptor getter를 나중에 읽는 Source보다 malformed 값의 오류가 빨라질 수 있습니다.

## Find·missing·fallback

Literal identity를 한 URL segment로 escape하여 GET을 먼저 수행합니다. Query는 member GET과 constructor seed에 보존하며 응답의 실제 값이 seed를 덮습니다. Accepted empty/non-JSON body는 Source처럼 seed/default view를 유지하고 Wire는 nil입니다. Actual valid JSON의 잘못된 shape나 UTF8 오류는 실제 응답을 포함한 실패입니다.

깨끗한 member 400/403/404에서만 full list fallback을 수행합니다. Extension은 name server query가 없어 name hint를 보내지 않습니다. Alias-derived ID 또는 name의 exact match를 전체 목록에서 찾아 후반 중복도 ErrAmbiguous로 처리합니다. 성공한 전체 조회의 absence는 기본 nil,nil이며 strict는 ErrNotFound입니다. LIST403·late error·physical fault·source 변경·취소는 missing으로 숨기지 않습니다.

Find microversion은 member GET만 덮고 LIST는 선택된 원래/default version입니다. ListBasePath는 fallback만 바꾸며 member route는 유지합니다. Path는 safe service-relative 범위이고 actual selected collection에 continuation을 고정합니다. Header는 concrete 안전 정책을 따르는 Go 옵션입니다. Source runtime/session/constructor controls 전체를 arbitrary 객체로 바꾸는 기능은 구현하지 않습니다.

## List·paging·Proxy

Inherited limit/marker만 server query입니다. Id·name·alias·description·links·updated_at은 Source Body local filters입니다. Wire updated는 query/local attr로 선언되지 않아 unknown kwargs로 버립니다. Fields도 선언 query가 아니므로 버립니다. Unknown 값은 marshaler를 실행하기 전에 버리고 local predicate는 kwargs 순서로 평가합니다. Nested dictionary subset과 bool-number/CPython-loaded numeric equality를 공통 엔진이 처리합니다.

Positive max_items는 local filtering 전 raw rows를 세며 limit hint를 보낼 수 있습니다. Single page와 consumer break는 continuation을 피합니다. Marker는 실제 primary id presence를 우선하고 absence에서 alias를 사용합니다. Actual primary null/empty를 alias로 바꾸어 이어가지 않습니다. Source처럼 root links key가 있으면 extensions_links를 가리고 dictionary links는 순회해도 next 후보를 만들지 않습니다. List links의 첫 literal rel="next"·href 후보를 우선하며 truthy URL이 없을 때 top-level next로 내려갑니다. Relation은 대소문자를 구별합니다. HTTP Link interoperability와 limit fallback은 공통 pager를 사용하며 origin/selected path/filter/cycle을 검사합니다. AllRecords는 terminal error에서 partial slice를 버립니다.

List Headers/Microversion/BasePath, 한 번의 ordered __conflicting_attrs merge, Source bound argument/duplicate controls는 공통 concrete 옵션으로 처리합니다. Header string-map 제한·safe path·empty-path default·HTTP Link interoperability는 명시적 Go 정책이며 Source arbitrary adapter/URI percent formatting과 차이가 있습니다.

고정 Proxy._list는 collection을 수집하지 않고 실제 generator를 JMESPath에 넘깁니다. `[0]`·`[*].alias`·`type(@)`는 HTTP 없이 null, `length(@)`는 HTTP 없이 type error, `@`는 lazy generator, `to_array(@)`는 generator를 담은 배열입니다. ProxyListRecords의 Value가 opaque identity를 보존하며 fully JSON일 때만 JSON이 채워집니다. Typed ListRecords/AllRecords는 truthy-string expression을 ProxyListRecords로 안내하는 오류입니다.

## 실제 오류·검증 범위

Source profile은 completed 100..399와 rejected 400..599를 구분하고 decode/local predicate/constructor 오류에는 실제 status/body/Header를 남깁니다. Physical read/Close/custom cancel/source failure는 native retry/reauth를 실행하지 않으며 Source404도 fallback이 되지 않습니다. Native pointer401의 ErrOriginal/ErrReauth는 각각 실제 receipt와 처리 실패를 유지합니다. Clean503 retry와401 token renewal은 허용하되 callback의 body/auth/route/protocol/version 변경은 실제 원본 응답과 함께 실패합니다.

테스트는 기존 Gophercloud testhelper의 internal/testcloud HTTP fixture 및 real response-body observer를 재사용합니다. Source/policy 고정 검토·race·독립 main build는 실제 배포에서 인증된 Neutron/backend 검증과 구별합니다. 실행 결과는 root의 검증 ledger에 기록하며 이 문서 자체는 미실행 검사를 PASS라고 표시하지 않습니다.

Source: [Extension class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/extension.py#L16), [find/list proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1588-L1626), [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489), [list element cast](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L113).
