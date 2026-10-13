# Neutron service providers

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `conn.network.service_providers(**query)`에 대응합니다. Gophercloud v2.15.0에는 이 endpoint의 native binding이 없어 SDK가 concrete API를 제공합니다. SDK의 공통 Neutron reader·descriptor·옵션·paging·JMESPath 엔진을 재사용하며 application builder나 decoder를 구현하지 않습니다.

| Python | Go |
| --- | --- |
| `conn.network.service_providers(**query)` | `api.ProxyListRecords(ctx, options...)` |
| 기본 resource generator 순회 | `api.ListRecords(ctx, options...)` |
| `list(generator)` | `api.AllRecords(ctx, options...)` |
| `service_type`, `name`, `fields`, `is_default` | `WithRecordListQuery(key, value)` |
| `max_items`, `paginated` | `WithRecordListMaxItems`, `WithRecordListPaginated` |
| `headers`, `microversion`, `base_path` | `WithRecordListHeaders`, `WithRecordListMicroversion`, `WithRecordListBasePath` |
| `__conflicting_attrs` | `WithRecordListConflictingAttrs(json.RawMessage(...))` |
| deprecated `jmespath_filters` | `WithRecordListJMESPath`와 `ProxyListRecords` |
| Source 속성/default view | `ServiceProviderRecord.Resource` |
| 실제 row·whole page·HTTP metadata | `Wire`, `Envelope`, `Header`, `StatusCode` |

읽기 전용 Source class입니다. 이 package에 Get/Create/Update/Delete를 추가하거나 존재하지 않는 provider ID를 만들어내지 않습니다.

Neutron 27.0.0의 `get_service_provider`는 project scope `role:reader`입니다. Provider는 project 소유 리소스가 아니므로 owner project_id 검사를 추가하지 않습니다. 실제 provider/service type과 driver 지원은 서버가 제공하며 SDK가 권한이나 backend를 생성하지 않습니다. [고정 Neutron service type 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/service_type.py#L22-L41).

## 독립 Python

`clouds.yaml`의 `dev` cloud를 사용합니다.

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--service-type")
parser.add_argument("--default", choices=("any", "yes", "no"), default="any")
parser.add_argument("--jmes")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    query = {}
    if args.service_type is not None:
        query["service_type"] = args.service_type
    if args.default != "any":
        query["is_default"] = args.default == "yes"
    if args.jmes is not None:
        query["jmespath_filters"] = args.jmes
    result = conn.network.service_providers(**query)
    if isinstance(result, types.GeneratorType):
        for value in result:
            print(json.dumps(value.to_dict(), indent=2))
    else:
        print(result)
```

## 독립 Go main

[설치 안내](../../../docs/install.md)대로 module을 준비하고 `go run . -cloud dev -service-type QOS`로 실행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    serviceType := flag.String("service-type", "", "server service type")
    defaultMode := flag.String("default", "any", "any, yes or no")
    expression := flag.String("jmes", "", "deprecated Python Proxy expression")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []serviceproviders.RecordListOption{
        serviceproviders.WithRecordListJMESPath(*expression),
    }
    if *serviceType != "" {
        options = append(options, serviceproviders.WithRecordListQuery("service_type", *serviceType))
    }
    switch *defaultMode {
    case "any":
    case "yes", "no":
        options = append(options, serviceproviders.WithRecordListQuery("is_default", *defaultMode == "yes"))
    default:
        log.Fatal("default must be any, yes or no")
    }
    result, err := network.ServiceProviders.ProxyListRecords(ctx, options...)
    if err != nil { log.Fatal(err) }
    if result.JSON != nil {
        fmt.Println(string(result.JSON))
        return
    }
    generator, ok := result.Value.(*serviceproviders.RecordListGenerator)
    if !ok {
        fmt.Printf("opaque expression result: %T\n", result.Value)
        return
    }
    for value, err := range generator.Records {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

## 선언·기본값·query

Source의 네 Body 속성은 inherited id/name과 service_type/is_default입니다. `is_default`의 wire 이름은 default이고 누락/null은 null입니다. Response `"false"` 문자열은 Python bool truthiness로 true, 0·빈 container·float underflow는 false입니다. Provider name으로 id를 대체하지 않습니다. NetworkResource/TagMixinNetwork를 상속하지 않으므로 revision_number·tags·if_match defaults를 넣거나 그런 unknown 확장을 임의로 cast하지 않습니다.

Unknown JSON, 큰 정수, nullable 값은 Resource와 원래 Wire에 독립 복사로 남습니다. self는 Resource에서 제거하며 location은 현재 SDK의 owned null view입니다. Source mutable Resource/current_location/cache lifecycle는 프로젝트의 공통 계약에서 별도로 추적합니다. Eager owned JSON 투영은 Source의 lazy descriptor getter 오류보다 먼저 실패할 수 있습니다.

Server query는 inherited limit/marker 및 fields·service_type·name·is_default(default)입니다. Canonical is_default가 wire default보다 presence로 우선하고 false는 False, null은 URL에서 생략합니다. Fields 배열은 literal 반복 query이며 name/service_type/default는 서버 조건입니다. Body id는 로컬 조건이고 unknown kwargs는 값의 marshaler를 실행하기 전에 버립니다. Predicate와 nested bool-number/CPython loaded-float 비교는 공통 Source 프로필을 사용합니다.

Named headers/microversion/base_path와 전용 factories는 마지막 값을 소유하며 원래 selected client를 바꾸지 않습니다. __conflicting_attrs는 한 번만 ordered merge하고 restored paginated/base_path는 Source 중복 인수 오류, restored JMES는 Resource.list unknown으로 처리합니다. Bound cls/session/self/resource_type 충돌을 presence로 거부하며 allow_unknown_params와 다른 비 Body kwargs는 Source처럼 버립니다. Concrete header는 안전한 string map만 받습니다. None-valued header removal·Resource subclass/adapter injection·URI percent-format는 이 JSON 옵션 범위 밖입니다.

## Paging·JMESPath·오류

기본 목록은 unbounded입니다. 실제 next/HTTP Link와 limit/marker fallback, raw cap·single page·consumer break를 공통 pager가 처리합니다. Positive max_items가 limit을 힌트로 넣고 제외한 행도 raw cap에 셉니다. Stock provider rows는 ID가 없을 수 있습니다. 서버가 limit을 무시하고 next 없이 cap보다 적은 행을 보내면 marker fallback이 ID를 만들지 않고 실패합니다. 이 경우 한 페이지를 원하면 `WithRecordListPaginated(false)`를 명시합니다. 고정 Source도 None marker의 반복을 오류로 처리하며 provider name을 ID로 대체하지 않습니다.

Route는 같은 origin의 selected collection과 filter/cursor에 고정하며 foreign/rewrite/cycle은 실제 whole-page 증거를 가진 오류입니다. Empty base_path는 Go에서 기본 경로로 처리하고 Source의 empty URI·arbitrary adapter routing과 차이가 있습니다. Dictionary links는 Go 편의입니다. AllRecords는 late error에서 partial slice를 버립니다.

실제 Source Proxy는 Resource.list를 수집하지 않고 JMESPath에 generator 그대로 넘깁니다. `[*].name`은 HTTP 없이 null, `length(@)`는 HTTP 없이 type error, `@`와 `to_array(@)`는 lazy generator/그것을 담은 배열입니다. ProxyListRecords.Value는 그 opaque identity를 보존하고 fully JSON일 때만 JSON을 채웁니다. Typed ListRecords/AllRecords의 truthy string expression은 ProxyListRecords로 안내하는 오류입니다.

Completed status 200..399와 실제 400..599 거부 응답을 구별합니다. Decode/constructor/local predicate 오류에는 실제 status/body/header가 남습니다. Physical read/Close/source/cancel 실패는 RetryFunc/ReauthFunc를 실행하지 않고 cause와 receipt를 유지합니다. Native pointer401의 ErrOriginal/ErrReauth도 보존합니다. 깨끗한 retry/token 갱신은 공통 계약을 따르며 새로운 body/auth/route/조건/version을 넣어 성공으로 바꾸지 않습니다. Synthetic informational responses와 replacement UTF8 decoding은 명시적 Go 차이입니다.

테스트는 Gophercloud testhelper를 사용하는 internal/testcloud의 실제 HTTP server와 기존 공통 reader를 재사용합니다. Physical fault fixture는 real transport의 response body만 관찰·장식합니다. Fixture/race/main build는 실제 배포의 인증/권한/extension/backend 검증과 구별하며 이 문서 자체는 실행하지 않은 검사를 PASS라고 기록하지 않습니다.

Source: [ServiceProvider](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/service_provider.py#L16), [Proxy.service_providers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7274), [실제 Proxy 목록/JMES 동작](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866).
