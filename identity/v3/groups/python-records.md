# Keystone Group reads: openstacksdk 비교

비교 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `Proxy.find_group`, `Proxy.groups`, `IdentityCloudMixin.search_groups`, `Group`/`Resource.find`/`Resource.list`입니다. Native 기반은 Gophercloud v2.15.0입니다. `FindRecord`, `ListRecords`/`AllRecords`/`ProxyListRecords`, `identity.NewGroupReadCloud(...).SearchGroups`는 선언 속성과 원래 HTTP 증거를 함께 제공하는 새 API입니다. 기존 native `Get`·`List`·`FindIdentity`는 각각의 기존 계약을 유지합니다.

Keystone 28.0.0 stock 정책에서 Group get/list는 admin 또는 system reader 또는 **대상 Group 도메인의 domain reader**를 허용합니다. 단순 project reader만으로 허용되는 정책은 아닙니다. List controller는 domain-scoped token의 domain ID를 정책 대상에 넣고 그 도메인으로 결과를 제한합니다. 서버의 실제 정책·토큰 scope를 SDK가 대체하지 않습니다. 확인 원문: [Keystone 정책 문서](https://docs.openstack.org/keystone/2025.2/configuration/policy.html), [고정 28.0.0 Group policy](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/common/policies/group.py), [고정 Group controller](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/api/groups.py).

## 실행 가능한 Python 예제

`OS_CLOUD`/clouds.yaml 또는 OpenStack 인증 환경을 설정하고 고정 Source와 일치하는 openstacksdk 환경에서 실행합니다.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    found = conn.identity.find_group("ops", domain_id="domain-id")
    if found is not None:
        print(json.dumps(found.to_dict(), indent=2, ensure_ascii=False))

    for row in conn.identity.groups(
        domain_id="domain-id", description="operations", limit=100,
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

    selected = conn.search_groups(
        name_or_id="ops-*", domain_id="domain-id",
        filters={"description": "operations"},
    )
    print(json.dumps([row.to_dict() for row in selected], indent=2))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go 예제

선택한 Identity v3 service에서 concrete API를 사용합니다. 아래를 `main.go`로 저장한 Go module에서 SDK dependency를 추가한 뒤 실행합니다. 인증 환경은 Python 예제와 같습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
)

func printJSON(value any) {
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", encoded)
}

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }

    found, err := service.Groups.FindRecord(ctx, "ops",
        groups.WithRecordFindQuery("domain_id", "domain-id"),
    )
    if err != nil { log.Fatal(err) }
    if found != nil { printJSON(found.Resource.Body) }

    for row, err := range service.Groups.ListRecords(ctx,
        groups.WithRecordListQuery("domain_id", "domain-id"),
        groups.WithRecordListQuery("description", "operations"),
        groups.WithRecordListLimit(100),
        groups.WithRecordListMaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        printJSON(row.Resource.Body)
    }

    cloud := identity.NewGroupReadCloud(service.Groups)
    selected, err := cloud.SearchGroups(ctx, "ops-*",
        identity.WithGroupSearchDomainID("domain-id"),
        identity.WithGroupSearchFilters(json.RawMessage(`{"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(selected.Value)
}
```

Cloud factory는 이미 선택된 Identity v3 API를 바인딩하며 HTTP를 보내지 않습니다. Python `ensure_service_version(..., '3')`처럼 v2 binding을 자동 변환하거나 version discovery를 수행하지 않습니다. 현재 Connection location을 자동 주입하는 bridge는 없고 명시적 location 옵션을 제공합니다.

## API와 선언 속성

| Python 호출/속성 | Go 대응/정책 |
|---|---|
| `identity.find_group(name_or_id, ignore_missing=True, **query)` | `Groups.FindRecord(ctx, nameOrID, ...RecordFindOption)` |
| `identity.groups(**query)` | `ListRecords` lazy iterator; `AllRecords` owned slice; polymorphic 결과는 `ProxyListRecords` |
| `search_groups(name_or_id, filters, domain_id)` | `NewGroupReadCloud(service.Groups).SearchGroups(ctx, nameOrID, ...GroupSearchOption)` |
| server query | `limit`, `marker`, `domain_id`, `name`; 별도 alias 없음 |
| local Body filter | query mapping에 없는 선언 `description`, `id`; `WithRecordListQuery` |
| Group Body | `description`, `domain_id`, `name`, `id`: 모두 untyped; 누락 null, 명시 null/false/숫자/배열/객체 보존 |
| unknown 응답 속성 | Source declared view에서 버림; 실제 `Wire`에 보존 |
| unknown list kwargs | JSON marshaler 호출 전에 버림; `FindRecord`의 일반 unknown query는 GET에만 전송 |
| `location` | 기본 null; `WithRecordFindLocation`, `WithRecordListLocation`, `WithGroupSearchLocation`으로 immutable 명시 |

`GroupRecord.Resource.Body`는 선언 속성과 계산된 location, `Wire`는 원래 응답 행, `Envelope`는 원래 전체 body, `Header`·`StatusCode`는 실제 receipt를 각각 독립적으로 보관합니다. `Clone`은 모든 채널을 복제합니다. 응답 `id`와 HREF는 passive data이며 후속 member 경로를 바꾸지 않습니다. 응답 id가 null/비문자열이어도 view에 유지하고 routing ID를 합성하지 않습니다. GET에서 응답 id가 없으면 Source constructor에 넣은 조회 ID가 view에 남습니다.

서버의 unknown `location`·`project_id`는 `Wire`에만 남습니다. Group에는 `project_id`/zone descriptor가 없으므로 이 값으로 명시적인 현재 project를 바꾸지 않습니다. Location 옵션은 생성·적용·결과에서 별도 소유하며 cloud/region/current project 이름을 native client에서 추정하지 않습니다.

## Find와 페이지 동작

`FindRecord`는 literal ID/name으로 GET을 먼저 보내고 **물리적으로 완전히 읽고 닫힌 GET 400·403·404**에서만 LIST로 전환합니다. 기본 logical absence는 `(nil, nil)`이고 `WithRecordFindIgnoreMissing(false)`는 `resource.ErrNotFound`입니다. LIST 403/404나 body 읽기·Close·취소·Source 변경 오류는 숨기지 않습니다. LIST의 ID 또는 name exact match를 전체 페이지에서 확인하며 같은 ID의 반복 행도 `resource.ErrAmbiguous`입니다. 첫 match 뒤의 늦은 페이지 오류도 최종 오류입니다.

GET의 일반 query는 seed의 선언 Body 값과 요청 params에 반영됩니다. Fallback에서는 Group의 네 query만 서버로 보내고 known Body filter를 로컬에서 평가하며 unknown kwargs를 버립니다. `name` 키가 없을 때만 조회 문자열을 자동 name query로 추가합니다. 명시된 null/empty name 키는 자동 hint를 막습니다. `WithRecordFindMicroversion`은 GET에만 적용되고 fallback LIST는 원래 session version을 사용합니다. `WithRecordFindListBasePath`는 fallback 경로만 바꿉니다.

`ListRecords`는 생성 시 HTTP 0회이며 실제 순회 시 옵션/Source 검증과 HTTP를 시작합니다. `break`는 다음 페이지 요청을 막고 `AllRecords`는 오류에 부분 slice를 버립니다. Iterator에서 이미 받은 record나 Cloud `Inventory`로 부분 결과를 보관할 수 있습니다. 기본 paginated=true, max_items=0(무제한), limit 생략입니다. `WithRecordListMaxItems`는 **로컬 필터 전 raw 행 수**를 제한합니다.

Pinned Source는 Keystone root `links` dictionary 안의 `next`를 다음 rel/href로 인식하지 않습니다. 이를 무시하고 root `links` 키가 있으면 `groups_links`보다 우선합니다. 명시적인 root `links:null`은 iterable이 아닌 값이므로 Source처럼 실제 page receipt와 함께 오류입니다. Top-level `next`·rel/href 배열은 공통 Source 정책을 따릅니다. HTTP Link는 Go의 명시적 호환 정책입니다. Pinned Source가 Requests의 next link에서 `uri`를 찾는 반면 일반 Requests는 `url`을 제공하므로 그 KeyError 결함을 재현하지 않고 RFC Link를 읽습니다. explicit 양수 limit이 있으면 짧은 nonempty page도 마지막 **raw wire ID**를 marker로 재요청하여 빈 페이지에서 종료합니다. local filter가 그 마지막 행을 버려도 동일한 marker를 사용합니다.

Go는 선택한 origin·collection path·query/header/version을 소유하며 외부 URL, path traversal, cycle, 잘못된 marker를 실제 page receipt와 함께 거절합니다. `WithRecordListBasePath`/`WithRecordFindListBasePath`는 같은 서비스의 안전한 상대 경로만 받습니다. Amount는 비음수 Go integer입니다. Source의 비정상적인 음수/비정수 paging이나 bool/null/빈 marker 재전송은 재현하지 않고 실제 receipt와 함께 종료/오류를 반환합니다.

Group `_max_microversion=None`이므로 기본 호출에서 discovery가 필요하지 않습니다. 명시적 version은 선택한 session 설정을 기본으로 사용하며 `identity <version>` 헤더를 소유합니다. 다른 서비스 version, 인증·route·protocol header를 override하여 재시도 경로를 바꿀 수 없습니다. Group list의 일반 `headers`·`microversion`·`base_path` formal kwargs와 `__conflicting_attrs` 한 번 복원은 공통 normalization에 따릅니다. 복원된 paginated/base_path의 bound argument 중복은 HTTP 전에 거절합니다.

List의 dictionary filter는 ordered recursive subset이며 falsey actual은 dictionary 필터와 불일치합니다. Truthy scalar에 empty dict 필터는 일치하고 nonempty dict는 오류입니다. CPython JSON float rounding/underflow와 bool/integer equality를 사용합니다. 입력 순서의 첫 불일치는 뒤의 잘못된 shape를 평가하지 않습니다. Cloud의 mapping 필터는 `_filter_list`의 별도 missing-key 오류 정책을 유지합니다.

## Proxy expression과 Cloud search

Source `Proxy._list`는 JMESPath를 아직 읽지 않은 Resource generator에 적용합니다. `[*].id`, `[0]`, `type(@)`는 **HTTP 0회로 null**이고 `length(@)`는 type 오류입니다. `@`·`not_null(@,[] literal)`은 generator를, `to_array(@)`는 generator가 들어 있는 배열을 반환합니다. `RecordProxyListResult.JSON`은 전체가 JSON일 때만 채워지고 opaque generator를 강제로 행 배열로 만들지 않습니다. Expression이 falsey 또는 non-string이면 lazy generator를 유지합니다.

```go
proxy, err := service.Groups.ProxyListRecords(ctx,
    groups.WithRecordListJMESPath("[*].id"),
)
if err != nil { log.Fatal(err) }
fmt.Println(string(proxy.JSON)) // null; HTTP 0회
```

`ListRecords`는 typed 행을 반환하므로 truthy string JMESPath에는 `ProxyListRecords`를 사용하라는 오류를 반환합니다. Go `iter.Seq2`를 다시 순회하면 캡처된 요청을 다시 실행하며 Python generator의 한 번 소비 상태를 복제하지 않습니다. 순수 expression 반환 전에 Go는 SDK-owned option/Source preflight를 수행합니다.

Cloud `SearchGroups`는 **domain query만 서버에 보내 모든 페이지를 먼저 모은 후** exact/glob ID/name, ordered nested mapping 또는 truthy string JMESPath를 적용합니다. Name/filters를 서버로 보내지 않습니다. `WithGroupSearchFilters(json.RawMessage(`"length(@)"`))`는 실제 전체 행 수를 JSON 숫자로 반환하고 `Expression=true`입니다. `Groups`는 ordinary selection의 복제된 record만 보관하며 arbitrary expression에는 nil입니다. `Inventory`는 실제 원래 행을 독립적으로 남기고 local 오류/늦은 HTTP 오류에도 보존합니다. 늦은 HTTP 오류는 잘못된 local expression보다 먼저 반환됩니다.

## 검증과 남은 범위

`python_records_test.go`와 `identity/group_search_records_test.go`는 기존 `internal/testcloud`/Gophercloud public testhelper/native fixture를 재사용합니다. GET-first/seed/null ID, duplicate/late HTTP, root link presence, limit/marker/raw cap/break, loaded local filters, opaque Proxy와 materialized Cloud, clone/location, callback/source/cancel snapshot을 검증합니다. Physical Read·Close·Source 복구·custom cancel matrix는 200/400/403/404/401/429/503 실제 receipt와 native 401 ErrOriginal/ErrReauth를 확인합니다. Clean 401/503/429/498은 정상 retry/re-auth, live token, frozen header를 확인합니다. root가 현재 branch의 Go gate를 수행한 결과만 검증 완료로 기록합니다.

이 finite Group read 구현은 전체 SDK 완료나 실제 배포 cloud 통합 완료가 아닙니다. Python Connection/session response cache와 synchronized generic Resource lifecycle은 별도 SDK-R1/C1 계약입니다. Python의 임의 `Resource.fetch` control(kwargs requires_id/base_path/skip_cache/resource_response_key 등), 객체 입력과 runtime attribute extension을 전부 재현한다고 주장하지 않습니다. 사용자 API는 concrete typed 옵션과 안전한 경로를 소유합니다. Find raw control kwargs의 미지원은 named review의 remaining에 정확히 유지합니다.
