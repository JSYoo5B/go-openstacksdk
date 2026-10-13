# Keystone Project 조회·검색: Python과 Go

기준은 [openstacksdk ef55d7d](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/project.py)와 Gophercloud v2.15.0입니다. 기존 native `Project`, `List`, `FindIdentity`에 더해 SDK가 Project descriptor·query·local filter·pagination·옵션 복사를 소유하는 별도 record API를 제공합니다. 이번 문서는 조회·검색 네 선언을 다루며 생성·수정·삭제·role assignment를 구현 완료로 세지 않습니다.

| Python | Go | 결과 |
|---|---|---|
| `conn.identity.projects(**query)` | `Projects.ListRecords(ctx, options...)` | lazy `ProjectRecord` iterator |
| `list(conn.identity.projects(**query))` | `Projects.AllRecords(ctx, options...)` | 성공 시 nonnil slice, 실패 시 collected slice 폐기 |
| `conn.identity.projects(jmespath_filters=...)` | `Projects.ProxyListRecords(ctx, WithRecordListJMESPath(...))` | JSON 또는 아직 읽지 않은 generator |
| `conn.identity.find_project(name_or_id, **query)` | `Projects.FindRecord(ctx, nameOrID, options...)` | GET 우선, clean 400/403/404 뒤 LIST, 기본 missing은 nil |
| `conn.list_projects(domain_id=..., name_or_id=..., filters=...)` | `conn.ListProjects(ctx, identity.WithProjectRead...(...))` | `ProjectListResult`의 원래 목록·JSON·실제 records |
| `conn.search_projects(name_or_id=..., filters=..., domain_id=...)` | `conn.SearchProjects(ctx, nameOrID, options...)` | 선택 JSON과 Projects, 원래 노출된 Inventory |

`ProjectRecord.Resource.Body`는 Python 속성 view입니다. `enabled`는 `is_enabled`로 읽으며, `is_domain/is_enabled`는 Python JSON truthiness, `options`는 dict descriptor, `tags`는 list descriptor로 변환합니다. 누락된 tags는 `[]`, 다른 선언 속성은 `null`입니다. explicit null은 유지하고 scalar tags는 단일 요소 배열, nonmapping options는 `{}`가 됩니다. response에 canonical 이름과 wire 이름이 같이 있으면 마지막 member가 이깁니다. extension의 큰 정수·null·unknown fields도 남습니다.

`Wire.Body`는 변환 전 spelling과 값입니다. `Envelope`, `Header`, `StatusCode`에는 실제 member 또는 전체 list page가 보존됩니다. 서로 다른 record의 header와 두 view는 독립 복사입니다. Project는 Header descriptor를 선언하지 않으므로 HTTP response header를 Body 속성으로 만들어 넣지 않습니다. SDK Connection이 없는 직접 record 조회의 location은 null이며, `Connection.ListProjects/SearchProjects`와 `WithProjectReadLocation`은 선택 전에 설정된 cloud/project 정보를 넣습니다. unknown extension에 실제 project_id/availability_zone가 있으면 그 리소스의 위치를 계산하며, Wire의 location을 바꾸지 않습니다.

## Query와 페이지

server query는 `domain_id`, `is_domain`, `name`, `parent_id`, `is_enabled`→`enabled`, tag aliases, inherited `limit/marker`입니다. 배열은 반복 query, boolean은 `True/False`, null은 생략합니다. canonical query가 wire alias보다 우선합니다. `description`, `options`, `links`, `id` 같은 declared nonquery Body 속성은 각 응답 행에서 local filter로 검사합니다. dict는 재귀 부분 일치이고 JSON number/boolean 비교는 CPython json.loads의 integer·float·bool equality를 따릅니다. empty dict filter는 truthy scalar/list에도 일치하며 falsey 실제 값에는 일치하지 않습니다. unknown query는 값의 JSON encoding 전에 버립니다.

`headers/microversion/base_path/__conflicting_attrs/paginated/max_items/jmespath_filters`는 Source formal argument를 SDK 소유 옵션으로 처리합니다. runtime Resource class/session/self 변경과 중복 formal 인자는 HTTP 전에 오류입니다. `WithRecordListHeaders/Microversion/BasePath/Limit/Marker/MaxItems/Paginated`가 구체적인 옵션을 제공합니다. Query·Headers·pointer 값과 callback 결과는 복사하며 선택 API/client가 바뀌거나 취소되면 요청 전에 종료합니다.

기본 pagination은 true, limit 생략, max_items=0은 무제한입니다. limit이 truthy면 next link가 없는 짧은 페이지에서도 마지막 **raw** 행 ID를 marker로 재요청하고 빈 페이지에서 끝납니다. local filter로 제외한 행도 raw cap과 marker에 포함합니다. consumer break·cap·single-page 후에는 다음 페이지를 요청하지 않습니다.

이번 Project profile은 고정 Source의 링크 우선순위를 적용합니다. root `links`가 있으면 plural `projects_links`를 가리고, array의 첫 literal `rel="next"/href` 또는 top-level next를 사용합니다. Keystone의 `links:{"next":...}`는 Python 코드가 rel/href 없는 항목으로 바꾸기 때문에 따라가지 않습니다. Go native `List`의 dictionary-next 호환 동작과 구분됩니다. Requests RFC Link의 `uri`/`url` 오류는 Go에서 유효한 Link를 읽는 호환성 개선으로 처리합니다. continuation은 원래 origin·collection·필터·limit을 유지해야 하며 cycle·다른 route·query 변경은 실제 page proof와 함께 실패합니다.

`FindRecord`는 ID처럼 보이는 문자열도 처음 GET하고, 정상 GET 결과를 이름으로 재검사하지 않습니다. query의 constructor seed를 Body view에 반영한 뒤 actual response가 덮어쓰며 unknown fields도 남습니다. clean 400/403/404만 name-filtered LIST로 이어집니다. 모든 페이지의 exact ID 또는 name match를 확인하고 두 번째 일치가 있으면 `resource.ErrAmbiguous`, 엄격한 missing 옵션이면 `resource.ErrNotFound`입니다. `WithRecordFindListBasePath`는 GET 성공 시 무시하며 fallback LIST만 다른 service-relative 경로로 바꿉니다. find의 per-call microversion은 GET에만 적용하고 LIST는 선택한 서비스 기본값을 씁니다.

읽기·Close·callback·선택 source 변경·취소가 섞인 HTTP 오류는 missing이나 fallback으로 숨기지 않습니다. native 401 reauthentication과 clean retry/backoff는 사용할 수 있으며, physical body/Close/source/cancel fault는 해당 hook에 재전송하지 않습니다. 실제 status 100..399를 Source 성공 범위로 처리하되 malformed JSON·descriptor·row constructor 오류는 원래 header/status/body와 함께 반환합니다.

## Cloud helper의 실제 규칙

[Source list_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L82)는 먼저 `dict(**filters)`를 만들고 truthy name/domain 인자로 덮어씁니다. null·false·0·empty string/list/object는 `{}`가 되지만, **truthy JMESPath 문자열도 mapping 확장 오류로 HTTP 전에 실패합니다.** `search_projects`도 같은 list를 먼저 호출하므로 문자열 JMESPath 지원을 문서만 보고 가정하지 않습니다.

Search는 name을 server query에 넣지 않고 목록을 받은 뒤 exact ID/name 또는 glob, insertion-order nested dict filter를 적용합니다. 예를 들어 `filters={"enabled": True}`는 server query로는 valid하지만 local cloud 단계에서는 `is_enabled` 속성의 wire alias라 membership에 실패합니다. `filters={"is_enabled": True}`를 사용합니다. location filter는 query로 보내지 않고 Connection 기본값을 넣은 뒤 검사합니다.

`ProjectListResult.Value`는 Source의 `list()` 결과를 보존하는 `[]any`입니다. 일반 경로에서는 Resource view, Proxy JSON literal 경로에서는 literal value, `to_array(@)`에서는 unconsumed generator를 담습니다. 전체가 JSON일 때만 `JSON`을 채우며 literal 값에는 HTTP receipt나 Project association을 만들지 않습니다. JSON object를 Proxy 표현식으로 반환해 list로 바꾸는 경우 Go는 key를 정렬해 결정적 순서를 제공합니다. Python insertion order와의 차이입니다. opaque generator를 JSON 검색 결과로 변환하거나 nonobject literal을 Resource filter로 검사할 때는 concrete option error를 반환합니다. 이 경계는 문서화된 Go 표현 차이이며 실제 기록을 만들어 내지 않습니다.

직접 facade가 필요한 경우 `identity.NewProjectReadCloud(service.Projects)`를 사용합니다. Connection은 cached `IdentityV3().Projects`를 사용하고 cloud/project location을 기본값으로 공급합니다. `WithProjectReadLocation`의 per-call 값은 복사해 유지하고, 전체 opts 또는 nil Location reset은 Connection 기본값으로 돌아갑니다. catalog endpoint가 없으면 실제 설정 오류를 반환합니다.

## 사용 예제

다음 Python과 Go 예제는 OS_CLOUD와 OS_PROJECT_DOMAIN_ID가 설정된 clouds.yaml 환경을 사용합니다. Project 목록의 domain reader target은 domain_id로 제한합니다.

```python
import json
import os
import openstack

conn = openstack.connect(cloud=os.environ["OS_CLOUD"])
domain = os.environ["OS_PROJECT_DOMAIN_ID"]

for project in conn.identity.projects(domain_id=domain, limit=2, max_items=2):
    print(json.dumps(project.to_dict(), indent=2))

project = conn.identity.find_project("demo", domain_id=domain)
if project is not None:
    print(json.dumps(project.to_dict(), indent=2))

listed = conn.list_projects(domain_id=domain, filters={"is_enabled": True})
print(json.dumps([project.to_dict() for project in listed], indent=2))
searched = conn.search_projects("demo*", domain_id=domain,
                                filters={"is_enabled": True})
print(json.dumps([project.to_dict() for project in searched], indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    domain := os.Getenv("OS_PROJECT_DOMAIN_ID")

    for project, err := range service.Projects.ListRecords(ctx,
        projects.WithRecordListQuery("domain_id", domain),
        projects.WithRecordListLimit(2), projects.WithRecordListMaxItems(2)) {
        if err != nil { log.Fatal(err) }
        printJSON(project.Resource)
    }
    project, err := service.Projects.FindRecord(ctx, "demo",
        projects.WithRecordFindQuery("domain_id", domain))
    if err != nil { log.Fatal(err) }
    if project != nil { printJSON(project.Resource) }

    filters := identity.WithProjectReadFilters(json.RawMessage(`{"is_enabled":true}`))
    listed, err := conn.ListProjects(ctx,
        identity.WithProjectReadDomainID(domain), filters)
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)
    searched, err := conn.SearchProjects(ctx, "demo*",
        identity.WithProjectReadDomainID(domain), filters)
    if err != nil { log.Fatal(err) }
    printJSON(searched.Value)
}
```

## 권한과 지원 범위

[Keystone 28.0.0 project 정책](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py#L157-L174)의 `identity:list_projects`는 admin 외에 system reader와 대상 domain의 reader를 허용합니다. 일반 project reader가 전체 `/v3/projects` 목록 권한을 갖는다는 뜻은 아닙니다. `identity:get_project`는 system/domain reader 외에 own-project 사용자 경로도 있습니다. Find의 fallback과 두 cloud helper는 목록 권한을 필요로 합니다. [기본 role 정책](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)과 서버의 policy override가 실제 접근을 결정합니다. 이번 user read 묶음은 이 reader 경로이며 write/admin-only 연산은 별도 단계입니다.

지원 판정은 proxy `projects`와 cloud `list_projects/search_projects` 세 선언이 `go_mapping`입니다. `FindRecord`의 concrete 옵션·GET 우선·clean fallback·중복/missing·실제 receipt 경로는 검증하지만, named `find_project`는 **unresolved**입니다. Source가 `**query`로 받는 `requires_id/base_path/skip_cache/error_message/resource_response_key` 같은 Resource.fetch control과 raw `headers`의 GET query→fallback LIST formal header 분기를 공통 classifier가 축소하거나 거부합니다. `WithRecordFindHeaders`가 있다는 이유로 그 raw kwargs 분기까지 완료로 세지 않습니다.

공통 검토에서 추적하는 SDK-R1/C1/S1(Connection 전체 Resource runtime·선택적 response cache·세션/negotiation) 범위도 별도로 남습니다. native transport wrapper, 세 named mapping과 이 FindRecord 부분 구현을 전체 SDK parity나 실제 cloud 실행 성공으로 부르지 않습니다.

검증은 기존 `testcloud`의 Gophercloud public testhelper와 Project transport fixture를 재사용합니다. `python_records_test.go`의 12개 그룹, `identity/project_record_cloud_test.go`의 6개 그룹, `connection_project_record_lists_test.go`의 3개 그룹이 계약을 검사합니다. 고정 Python runtime 비교는 descriptor/alias/default·Keystone dictionary link·short-page marker·falsey filters·truthy nonmapping 오류·argument override·wire alias local failure·opaque Proxy 결과 26건과 stock reader policy AST 2건을 포함합니다. Go 실행 결과와 지원 수 집계는 해당 실행의 중앙 검증 기록을 따릅니다.
