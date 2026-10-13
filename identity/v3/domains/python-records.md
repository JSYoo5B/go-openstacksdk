# Keystone Domain list/search: openstacksdk 비교

고정 비교 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `Proxy.domains`, `IdentityCloudMixin.list_domains`/`search_domains`, Domain/Resource.list/Proxy._list입니다. 기반은 Gophercloud v2.15.0입니다. 새 `ListRecords`, `AllRecords`, `ProxyListRecords`와 `identity.NewDomainReadCloud(...).ListDomains`/`SearchDomains`는 SDK 소유 defaults/options와 선언 속성·실제 응답 증거를 제공합니다. 기존 native API의 동작은 유지합니다.

Keystone 28.0.0 stock `identity:list_domains`는 admin 또는 system reader 또는 **대상 domain의 domain reader**를 허용합니다. Domain-scoped token의 domain ID를 controller가 정책 대상에 넣고 같은 domain 결과로 제한합니다. 일반 project reader만으로 전체 domain 조회를 허용하는 정책은 아닙니다. 서버가 실제 인증·scope·정책을 결정합니다. 확인 원문: [2025.2 정책 문서](https://docs.openstack.org/keystone/2025.2/configuration/policy.html), [고정 Domain policy](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/domain.py), [고정 Domain controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/domains.py). 이 가이드는 사용자 list/search만 다룹니다.

## 실행 가능한 Python

`OS_CLOUD`/clouds.yaml 또는 OpenStack 인증 환경을 설정하고 openstacksdk를 설치합니다.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    for row in conn.identity.domains(
        is_enabled=True, description="operations", limit=100, max_items=100
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

    listed = conn.list_domains(is_enabled=True, description="operations")
    print(json.dumps([row.to_dict() for row in listed], indent=2))

    # Source distinguishes None from an explicitly empty identifier.
    server_filtered = conn.search_domains(filters={"name": "team"})
    locally_filtered = conn.search_domains(
        filters={"description": "operations"}, name_or_id="team-*"
    )
    print(json.dumps([row.to_dict() for row in server_filtered], indent=2))
    print(json.dumps([row.to_dict() for row in locally_filtered], indent=2))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go

동일한 인증 환경에서 아래를 `main.go`로 저장한 Go module에 SDK dependency를 추가하여 실행합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", data)
}

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Domains.ListRecords(ctx,
        domains.WithRecordListQuery("is_enabled", true),
        domains.WithRecordListQuery("description", "operations"),
        domains.WithRecordListLimit(100),
        domains.WithRecordListMaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        printJSON(row.Resource.Body)
    }

    cloud := identity.NewDomainReadCloud(service.Domains)
    listed, err := cloud.ListDomains(ctx,
        identity.WithDomainReadFilters(json.RawMessage(`{"is_enabled":true,"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)

    serverFiltered, err := cloud.SearchDomains(ctx,
        identity.WithDomainReadFilters(json.RawMessage(`{"name":"team"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(serverFiltered.Value)

    locallyFiltered, err := cloud.SearchDomains(ctx,
        identity.WithDomainReadNameOrID("team-*"),
        identity.WithDomainReadFilters(json.RawMessage(`{"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(locallyFiltered.Value)
}
```

Factory는 이미 선택된 Keystone v3 API를 바인딩하며 HTTP를 보내지 않습니다. Python `ensure_service_version(..., '3')`의 임의 v2 binding 전환이나 Connection current location 자동 계산은 주장하지 않습니다. `WithDomainReadLocation(resource.CloudLocation)`과 leaf `WithRecordListLocation`으로 명시적 immutable location을 제공합니다.

## 선언 속성과 목록 정책

| 항목 | Source 및 Go 계약 |
|---|---|
| server query | `name`, `is_enabled` → `enabled`, inherited `limit`·`marker`; raw wire `enabled`도 인식하며 canonical alias presence 우선 |
| local Body filters | `description`, `options`, `links`, `id`; 입력 순서대로 평가 |
| nullable Body | description, is_enabled, options, links, id, name; 누락 null과 명시 null 유지 |
| is_enabled | Python bool cast: falsey false, truthy true; CPython float underflow 적용 |
| options | Python dict getter: object 유지, nonobject `{}`, null 유지 |
| unknown kwargs | 선언 query/Body가 아니면 marshaler 평가 전 버림 |
| unknown 응답 | declared `Resource.Body`에서 버림; 원래 `Wire`에 유지 |

`DomainRecord.Resource`는 선언 속성과 계산된 location, `Wire`는 원래 행, `Envelope`는 전체 실제 page body, `Header`·`StatusCode`는 실제 receipt를 독립적으로 보관합니다. `Clone`은 전 채널을 복제합니다. `options`는 기존 native Domain 모델에 없는 필드까지 유지하며 별도 interface builder가 필요하지 않습니다. 응답 id/HREF는 passive 값이고 후속 routing이나 marker를 caller-mutated view에서 가져오지 않습니다.

`ListRecords`는 호출 시 HTTP 없이 lazy iterator를 반환합니다. `break`는 다음 페이지를 읽지 않고 `AllRecords`는 오류 시 부분 slice를 버립니다. 기본 paginated=true, max_items=0, limit 생략입니다. Raw `max_items` cap은 local filter 전에 적용됩니다. explicit 양수 limit이 있으면 링크 없는 짧은 nonempty page도 마지막 **raw wire ID**를 marker로 재요청하여 empty page에서 끝납니다.

Pinned Source처럼 root links dictionary의 next를 rel/href로 인식하지 않고, root `links` 키가 있으면 `domains_links`를 대신 사용하지 않습니다. `links:null`은 iterable이 아니므로 실제 page receipt를 가진 오류입니다. Rel/href 배열과 top-level next는 Source 우선순위에 따릅니다. HTTP Link는 명시적 Go 호환 정책으로 읽습니다. Source의 Requests next['uri'] 접근에 대한 일반 next['url'] shape 불일치 결함을 재현하지 않습니다.

이 새 Profile만 valid HTTP 100..399를 받아들이는 `SourceStatusPolicy`를 켭니다. 기존 Profile과 native API 상태 정책은 유지합니다. Domain의 `_max_microversion=None`이므로 기본 discovery가 없습니다. Typed headers/version/base_path와 Proxy formal kwargs normalization을 재사용하며 `__conflicting_attrs`는 한 번 복원합니다. Bound paginated/base_path 중복과 허용되지 않은 transport control은 HTTP 전에 거절합니다.

Go는 origin/collection route/query/header/version을 소유합니다. 외부 URL·traversal·cycle·잘못된 marker를 실제 page proof와 함께 종료하며 amount는 비음수 Go integer입니다. 명시적 location은 서버의 unknown location/project_id/zone 값을 신뢰하지 않습니다. Domain에는 project_id/zone descriptor가 없으므로 configured current project를 유지합니다.

Local dictionary filter는 recursive subset와 actual truthiness를 사용합니다. CPython loaded bool/integer·float rounding/underflow equality를 보존하며 첫 mismatch는 뒤의 잘못된 shape를 평가하지 않습니다. Reached truthy scalar/nonempty dict filter 오류는 실제 page proof와 함께 반환합니다.

## Proxy와 Cloud의 서로 다른 반환

`ProxyListRecords`는 JMESPath를 **unconsumed generator**에 적용합니다. Projection·type(@)는 HTTP 0회로 null, length(@)는 HTTP 0회 type 오류입니다. @와 not_null은 generator를, to_array(@)는 nested generator를 유지합니다. 전체가 JSON일 때만 `RecordProxyListResult.JSON`을 채웁니다. Typed `ListRecords`의 truthy string expression에는 `ProxyListRecords`를 사용합니다. Falsey/non-string expression은 lazy generator를 유지합니다. Go iterator 재순회는 요청을 다시 실행하며 Python의 single-use 상태를 공유하지 않습니다.

Cloud `ListDomains`는 keyword filters를 Source Proxy에 전달한 뒤 Python list()처럼 eagerly 소비합니다. Literal list·string·object expression도 HTTP 없이 Python iterable 형태로 반환할 수 있습니다. Object key의 Python insertion order 대신 Go는 deterministic sorted order를 사용합니다. Nested generator는 `Value`에 유지하고 JSON을 합성하지 않습니다.

Cloud `SearchDomains`는 `NameOrID *string` presence를 구분합니다.

- 기본 nil: Source `name_or_id=None`처럼 filters를 `list_domains(**filters)`에 전달합니다. Filters는 mapping이어야 하며 truthy string뿐 아니라 false/0/empty string/array도 kwargs expansion 오류입니다. Source의 정확한 None만 empty mapping으로 바뀝니다. Go `WithDomainReadFilters(null)`은 이 Search None 의미를 보존합니다.
- `WithDomainReadNameOrID("")` 포함 nonnil: 서버에 filters/name을 보내지 않고 모든 domains를 먼저 모은 뒤 exact/glob identifier와 local nested mapping 또는 truthy string JMESPath를 적용합니다. 잘못된 local filter도 HTTP 완료 후 평가되며 늦은 HTTP 실패가 우선합니다.

Arbitrary local expression은 `Expression=true` 및 JSON `Value`를 반환하고 Domain record를 합성하지 않습니다. Ordinary selection의 `Domains`는 `Inventory`와 별도로 복제합니다. Nil-identifier branch의 Proxy nested generator는 `ValueObjects`에 유지하며 JSON Value는 nil입니다. Inventory는 실제 yielded records/receipt를 보관하여 늦은 HTTP/local 오류에서도 남습니다. Resource local filter로 제외한 행은 원래 전체 Envelope에 남고 yielded Inventory에는 포함되지 않습니다.

## 검증 범위

새 tests는 기존 `internal/testcloud`, Gophercloud public testhelper 및 Domain native transport fixture를 재사용합니다. Descriptor/default/null·bool/dict casts, alias query, options 유지, Source links/marker/raw cap, loaded local filters, opaque Proxy, Cloud identifier presence/kwargs/JMESPath, clone/location, option/source/cancel/header snapshot을 검사합니다. 28 physical fault lanes와 clean401/503/429/498 retry lanes는 실제 body/header/status·Close once·no replay·live token·owned header를 확인합니다. Native *ErrUnableToReauthenticate는 Unwrap가 없으므로 값형 ErrOriginal과 ErrReauth causes를 각각 검사합니다.

Worker는 Go 테스트를 실행하지 않으며 현재 root gate 결과만 검증 완료로 기록합니다. 실제 cloud 통합 및 Connection/session cache·동적 generic Resource lifecycle·자동 Connection bridge는 별도 SDK-R1/C1 remaining입니다. 이 세 concrete mapping은 `go_mapping` 후보이고 supported 증가는 0입니다.
