# Cinder v3 허용 Resource 필터

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [resource_filters Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3620-L3628), [ResourceFilter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/resource_filter.py), [공통 Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2359)입니다. `/resource_filters`는 배포가 허용하는 volume/snapshot 등의 필터 이름을 반환하며 Gophercloud v2.15.0에는 native binding이 없어 SDK가 concrete API·기본값·결과를 소유합니다.

| Python | Go | 동작 |
|---|---|---|
| `conn.block_storage.resource_filters()` | `filters.ListRecords(ctx)` | lazy 목록, server query 없음 |
| `resource_filters(resource="volume")` | `WithRecordListResource("volume")` | `?resource=volume` |
| `resource_filters(resource=None)` | query에 JSON null | wire 생략 |
| `resource_filters(filters=["name"])` | `WithRecordListQuery(map[string]any{"filters": []string{"name"}})` | 응답 Body에 대한 로컬 predicate |
| `paginated=False` | `WithRecordListControl(resource.ListControl{SinglePage: true})` | 첫 페이지 후 종료 |
| `max_items=N` | `WithRecordListControl(resource.ListControl{MaxItems: N})` | 필터 전 raw-row cap, 초기 limit hint |
| `microversion=version` | `WithRecordListMicroversion(version)` | 모든 페이지의 override, 빈 문자열은 헤더 생략 |
| `headers={...}` | `WithRecordListHeaders(...)` | 일반 헤더 복사, protocol/auth는 library 소유 |
| 정상 목록 수집 | `filters.AllRecords(ctx)` | 오류면 부분 행을 버리고 반환 |
| 고정 `resource_filters(jmespath_filters=expr)` | `filters.ProxyListRecords(ctx, WithRecordListJMESPath(expr))` | 아직 소비하지 않은 opaque generator에 평가, service HTTP 0 |
| falsey 또는 non-string `jmespath_filters` | `WithRecordListProxyJMESValue(value)` | 원래 lazy generator 유지 |
| 배열 수집 후 `jmespath.search(expr, data)` | `filters.ListResult(ctx, WithRecordListJMESPath(expr))` | 전체 selected row의 임의 JSON 변환 |

`filters := resourcefilters.New(service.RawClient())`로 생성할 수 있습니다. API 사용자에게 builder나 인터페이스 구현을 요구하지 않습니다.

## 독립 Python main

고정 Proxy의 `jmespath_filters`는 generator를 그대로 JMESPath에 전달하는 결함이 있습니다. 아래 main은 이 결함을 명시적으로 개선하여 먼저 `list(...)`를 수집한 뒤 `row.to_dict()` 배열을 검색합니다. `resource_filters(..., jmespath_filters=expr)` 원본 호출과 동일하다고 주장하지 않습니다.

```python
import argparse
import json
import jmespath
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.33")
    parser.add_argument("--resource", default=None)
    parser.add_argument("--single-page", action="store_true")
    parser.add_argument("--expression", default="")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    rows = list(conn.block_storage.resource_filters(
        resource=args.resource,
        paginated=not args.single_page,
    ))
    data = [row.to_dict() for row in rows]
    # Explicit repair: the pinned Proxy passes a generator to JMESPath.
    value = jmespath.search(args.expression, data) if args.expression else data
    print(json.dumps(value, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용한 별도 모듈에서 `go run . -cloud dev -resource volume`로 실행합니다. 예: `-expression 'length(@)'`는 행 개수 JSON을 반환합니다.

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.33", "selected Cinder version")
    resourceName := flag.String("resource", "", "resource name; empty omits query")
    single := flag.Bool("single-page", false, "stop after the first page")
    expression := flag.String("expression", "", "optional JMESPath result")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    filters := resourcefilters.New(service.RawClient())
    options := []resourcefilters.RecordListOption{
        resourcefilters.WithRecordListControl(resource.ListControl{SinglePage: *single}),
        resourcefilters.WithRecordListJMESPath(*expression),
    }
    if *resourceName != "" {
        options = append(options, resourcefilters.WithRecordListResource(*resourceName))
    }
    result, err := filters.ListResult(ctx, options...)
    if err != nil { log.Fatal(err) }
    fmt.Println(string(result.Value))
}
```

## Source 계약과 Go 제어

[공식 Cinder API](https://docs.openstack.org/api-ref/block-storage/v3/#list-resource-filters)는 **3.33+**에서 해당 경로를 제공합니다. [ResourceFiltersController.index](https://github.com/openstack/cinder/blob/master/cinder/api/v3/resource_filters.py)에는 별도의 관리자 전용 authorize 호출이 없고, 일반 인증된 프로젝트 사용자가 현재 설정된 허용 필터를 읽습니다. SDK가 서버의 role/policy 판정을 대신하지 않습니다.

Source `_max_microversion=3.33`에 따라 selected version이 있으면 높은 버전도 유지하고, 없으면 공통 Cinder discovery로 server maximum과 3.33 중 작은 버전을 선택합니다. 명시적인 version override는 모든 페이지에 적용하며 `""`는 auto-discovery 없이 버전 헤더를 생략합니다. selected `3.32`에서는 Source처럼 실제 서버 오류를 보존합니다. `Discovery`와 member `Envelope`·status는 분리된 실제 증거입니다.

query mapping은 `include_pagination_defaults=False`로 **resource만** 서버에 보냅니다. 요청의 null은 생략하고 JSON iterable은 Requests 방식의 반복값으로 변환합니다. `filters`, inherited `id`·`name`은 공통 Cinder `MatchLocal`로 Body에 적용합니다. 다른 kwargs와 사용자 `limit`·`marker`는 버리고 그 값의 marshaler도 호출하지 않습니다. local cap은 필터에서 탈락한 raw row도 세며 cap 뒤의 malformed row/next를 소비하지 않습니다. max_items hint로만 limit이 들어가며, cap 이전에 서버가 짧은 페이지를 주고 continuation/유효 ID가 없으면 strict marker 오류가 발생할 수 있습니다. stock resource-filter 행에는 ID가 없으므로 평소에는 unlimited 기본값을 사용합니다.

`Resource`는 untyped id/name, `filters` list descriptor(스칼라는 한 요소 배열, null은 null), Python `str` 방식의 resource descriptor와 default null을 제공합니다. unknown/self/location은 literal `Wire`에 보존합니다. standalone leaf의 computed location은 null인 명시적 SDK-R1 차이입니다. 모든 row에 독립 whole-page Envelope·HTTP Header/status가 있으며 response constructor collision·잘못된 모양·UTF-8·descriptor·local filter 오류는 accepted whole-page proof를 보존합니다. JSON 숫자의 로컬 equality·dictionary truthiness는 공통 `MatchLocalLoaded` opt-in을 사용합니다. decimal/exponent는 CPython json.loads처럼 binary64로 해석하고, integer는 임의 정밀도를 유지하며 bool은 integer subclass처럼 비교합니다. `1e-400 == 0`, overflow끼리의 비교, float 반올림 후 인접 bigint와의 비교와 nested subset에도 같은 정책을 적용합니다. 기존 다른 바인딩의 exact-decimal 기본 정책은 바꾸지 않습니다.

lazy 순회는 break/single-page/cap에서 추가 HTTP를 중단하고, 원래 query·same-origin·같은 collection route를 유지합니다. body/HTTP Link, falsey `next`(binary64 float underflow 포함), empty-page 종료와 cycle 검사에는 공통 pager를 사용합니다. dict `links.next`/실제로 사용 가능한 HTTP Link 해석은 고정 Source의 dict-link 무시와 Requests `uri` key 결함을 개선한 호환 동작입니다. 교차 origin/route/filter 변경 및 불완전한 marker는 인증 토큰 노출이나 루프를 만들기 전에 거부합니다.

`ListResult`는 typed JMESPath 옵션으로 arbitrary JSON을 제공합니다. `Value`의 scalar/null/object에는 HTTP proof를 꾸며 넣지 않고, 실제 receipt는 `Records`에 남습니다. 표준 `ListRecords`·`AllRecords`는 typed row를 반환하므로 nonempty expression을 전달하면 HTTP 전 `ProxyListRecords` 또는 `ListResult`를 사용하라는 오류를 반환합니다. 빈 expression은 Source의 falsey 조건처럼 전체 배열을 반환합니다. raw `jmespath_filters`는 `WithRecordListQuery`의 JSON 값으로도 받아 server query와 분리하며 `ProxyListRecords`에서 Source의 falsey/non-string 규칙을 적용합니다. base_path/session/__conflicting_attrs 등 route/session control 주입은 해당 typed 제어를 사용하도록 HTTP와 value marshaler 전에 거부합니다. route 교체 및 Python monkeypatch는 지원 계약에 포함하지 않습니다.

`ProxyListRecords`는 원본 `_list`의 polymorphic 결과를 별도로 보존합니다. `[*].resource`는 JSON null, `length(@)`는 TypeError에 대응하는 Go 오류, `@`는 `*RecordListGenerator`, `to_array(@)`나 `{lazy: @}`는 그 동일 pointer를 포함한 컨테이너를 반환합니다. 이는 버전 discovery와 member HTTP 전에 결정됩니다. `RecordProxyListResult.JSON`은 전체가 JSON일 때만 채웁니다. opaque generator를 JSON으로 직렬화하면 오류이며, `generator.Records`를 순회할 때 비로소 discovery/목록을 요청하고 당시의 실제 receipt를 기록합니다. generator identity는 공통 `internal/jmespath.OpaqueIdentity` ABI로 유지합니다. `ListResult`의 eager 개선과 원본 Source 결과를 섞지 않습니다.

옵션 생성·callback 경계에서 알려진 query JSON과 raw bytes/map/pointer를 소유하며 nil map은 안전한 빈 map으로 재구성합니다. selected 원본 client는 수정하지 않습니다. read/Close/source/cancellation 장애에는 retry/backoff/reauth를 호출하지 않고 native 401 `ErrOriginal`·`ErrReauth` 필드를 보존합니다. 깨끗한 재인증·retry는 가능하며 body·protocol·version·금지 헤더 injection은 첫 실제 rejection 증거와 함께 중단됩니다.

회귀는 [일반 호출 계약](resourcefilters/python_contracts_test.go) 13그룹과 [Source opaque·loaded 숫자 계약](resourcefilters/python_proxy_contracts_test.go) 4그룹에 있습니다. 보통 호출은 기존 `testcloud` 및 public Gophercloud testhelper HTTP fixture를 사용하며 물리 Read/Close 장애만 원래 Transport/body를 감쌉니다. 실제 cloud 정책·인증 호출과 Python runtime 검증은 별도 acceptance입니다.
