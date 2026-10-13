# Keystone Role 목록·검색: openstacksdk와 Go

고정 기준은 [openstacksdk ef55d7d](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/role.py)와 Gophercloud v2.15.0입니다. 이번 범위는 proxy `roles`, cloud `list_roles/search_roles` 세 선언입니다. native `Role`, `List/All`, `Get/FindIdentity`는 기존 계약을 유지하며 새 API가 Source 속성·기본값·필터·페이지·옵션 복사와 실제 응답 증거를 소유합니다.

| Python | Go | 반환 |
|---|---|---|
| `conn.identity.roles(**query)` | `service.Roles.ListRecords(ctx, options...)` | lazy `RoleRecord` iterator |
| `list(conn.identity.roles(**query))` | `service.Roles.AllRecords(ctx, options...)` | 성공한 empty도 nonnil slice, 실패 시 collected slice 폐기 |
| `conn.identity.roles(jmespath_filters=...)` | `service.Roles.ProxyListRecords(ctx, WithRecordListJMESPath(...))` | JSON 또는 아직 읽지 않은 opaque generator |
| `conn.list_roles(**kwargs)` | `identity.NewRoleReadCloud(service.Roles).ListRoles(ctx, options...)` | `RoleListResult.Value/JSON/Roles/Inventory` |
| `conn.search_roles(name_or_id, filters, domain_id)` | `RoleReadCloud.SearchRoles(ctx, nameOrID, WithRoleSearch...(...))` | 선택 JSON, 선택 Roles와 actual Inventory |

`RoleRecord.Resource.Body`는 name·description·domain_id·options·links와 inherited id, computed location을 제공합니다. 누락된 descriptor와 explicit null은 null이며 options의 nonmapping 값은 Source dict descriptor처럼 `{}`가 됩니다. Role은 unknown attrs를 Body에 허용하지 않으므로 extension·unknown project_id·서버가 보낸 location은 `Wire.Body`에 남습니다. `_max_microversion=None`이며 자체 maximum-version discovery를 요구하지 않습니다.

`Wire`, `Envelope`, `Header`, `StatusCode`는 원래 행·전체 페이지·실제 HTTP header/status를 보존합니다. `Clone`은 각 view와 byte/header 채널을 복사하며 응답 id와 links.self는 후속 경로를 바꾸지 않습니다. Role에 Header descriptor가 없으므로 response header를 Body 필드로 만들어 넣지 않습니다. 직접 native client 기반 location은 null입니다. `WithRecordListLocation/WithRoleSearchLocation`은 명시적으로 기록된 cloud/region/current project를 복사하는 Go 편의 기능입니다. Role에 project_id/zone descriptor가 없어 unknown 응답 값으로 current project를 바꾸지 않고, 이 편의 옵션의 zone은 null로 정규화합니다. 인증과 endpoint에서 cloud/project 이름을 추측하지 않습니다.

## Query·페이지·오류

server query는 name·domain_id와 inherited limit·marker입니다. `WithRecordListQuery`는 선언 query의 배열을 반복 값으로, bool을 `True/False`로 보내고 null은 생략합니다. description·options·links·id는 local Body filter입니다. dict는 insertion-order recursive subset이고 CPython json.loads의 integer/float/bool equality를 적용합니다. falsey actual은 dict와 불일치하며 empty dict는 truthy scalar/list에도 일치합니다. unknown kwargs는 JSON marshaler를 호출하기 전에 버립니다. Server name filter를 SDK가 다시 local filter로 강제하지 않습니다.

기본은 paginated=true, limit 생략, max_items=0 무제한입니다. explicit 양수 limit이면 광고한 next가 없어도 짧은 nonempty page의 마지막 raw ID를 marker로 요청하고 empty page에서 끝납니다. local filter로 버린 row도 raw cap·marker에 포함합니다. break·cap·single page 뒤 추가 페이지를 읽지 않습니다. 이미 받은 iterator row는 늦은 오류에도 보관할 수 있습니다.

고정 Source의 root links 존재가 roles_links보다 우선하고 Keystone `links:{"next":...}`는 rel/href가 없는 항목이 되어 무시합니다. rel/href 배열·top-level next·limit fallback을 Source 우선순위로 처리하며 explicit root links:null은 실제 page 오류입니다. Go는 원래 origin/path/query/limit을 고정하고 순환·다른 URL·필터 변경을 actual page proof와 함께 거절합니다. RFC HTTP Link는 pinned Requests의 next[uri]/next[url] 차이를 교정하는 Go 호환 동작입니다. 비음수 Go integer cap·안전한 marker/route·iterator 재순회는 문서화된 Go 정책입니다.

`headers/microversion/base_path/paginated/max_items/jmespath_filters/__conflicting_attrs`는 concrete owned 옵션으로 처리합니다. `WithRecordListOpts/Headers/Microversion/BasePath/Limit/Marker/MaxItems/Paginated`가 있으며 callbacks·maps·raw JSON·pointer 값은 복사합니다. conflicting attrs는 한 번 복원하고 bound formal 중복을 거절합니다. nil 옵션/ctx·외부 route·잘못된 JSON·selected API/client/source 변경·custom cancel은 오류입니다. 인증·framing·다른 서비스 version header는 SDK가 소유하며 `identity <version>`은 선택값과 per-call 옵션에 고정됩니다.

새 Role profile은 유효 HTTP100..399를 받아들입니다. accepted Read/Close/decode/descriptor 오류는 실제 `resource.ResponseError`, rejected 오류는 actual native `ErrUnexpectedResponseCode`와 원인을 보존합니다. 401은 `ErrUnableToReauthenticate.ErrOriginal/ErrReauth`를 함께 유지합니다. physical Read/Close/source/cancel fault는 retry/backoff/reauth나 다른 페이지 요청으로 숨기지 않습니다. clean retry·reauth·429/498 backoff는 고정 path/query/header/version과 live token으로 처리합니다.

## Proxy와 cloud의 다른 expression 순서

[Source Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)는 아직 소비하지 않은 Resource generator를 JMESPath에 전달합니다. `[*].id/[0]/type(@)`는 HTTP0회로 null, `length(@)`는 type 오류, `@`는 같은 generator, `to_array(@)`는 generator가 들어 있는 배열입니다. falsey/non-string expression은 원래 generator를 유지합니다. `RecordProxyListResult.JSON`은 전체가 JSON일 때만 채우고 `Value`의 `*RecordListGenerator.Records`를 순회해야 HTTP가 나갑니다. typed ListRecords에 truthy string expression을 주면 ProxyListRecords를 사용하도록 오류를 반환합니다.

cloud `list_roles`는 Source `list()`로 그 polymorphic Proxy 결과를 소비합니다. `RoleListResult.Value`는 항상 성공한 목록의 `[]any`이고 JSON literal·문자열 iterator·nested generator도 보존합니다. 전체가 JSON일 때만 JSON을 채우며 literal 값에 Role association·HTTP receipt를 만들지 않습니다. object-valued Proxy 결과는 Go에서 key를 정렬해 결정적인 list 순서를 제공하며 Python insertion order와 다른 정책입니다. 실제 record 경로의 Value/Roles/Inventory는 독립 복사되고 late HTTP에도 Inventory가 남습니다.

cloud `search_roles`는 **domain_id만 server query로 보내 모든 행을 먼저 수집한 뒤** exact/glob ID/name와 nested mapping 또는 truthy JMESPath 문자열을 평가합니다. name과 filters를 서버로 보내지 않습니다. `length(@)` 같은 expression은 actual row 배열에서 임의 JSON scalar/object/list를 반환하고 Expression=true, Roles=nil, Inventory에는 원래 records가 남습니다. name은 expression보다 먼저 필터링하며 late HTTP 오류는 잘못된 local expression보다 먼저 반환합니다. 이는 Project cloud helper의 filters kwargs expansion 순서와 다릅니다.

## 실행 예제

system-scoped reader가 구성된 OS_CLOUD/clouds.yaml 또는 OpenStack 인증 환경에서 실행합니다. 첫 목록의 limit/max_items는 raw 행 기준입니다. 아래는 각각 독립적인 Python 프로그램과 Go main입니다.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    for role in conn.identity.roles(description="operations", limit=2,
                                    max_items=2):
        print(json.dumps(role.to_dict(), indent=2, ensure_ascii=False))

    listed = conn.list_roles(description="operations")
    print(json.dumps([role.to_dict() for role in listed], indent=2))
    selected = conn.search_roles("read*", filters={"options": {"immutable": False}})
    print(json.dumps([role.to_dict() for role in selected], indent=2))
    count = conn.search_roles(filters="length(@)")
    print(json.dumps(count, indent=2))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
)

func printJSON(value any) {
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }

    for role, err := range service.Roles.ListRecords(ctx,
        roles.WithRecordListQuery("description", "operations"),
        roles.WithRecordListLimit(2), roles.WithRecordListMaxItems(2)) {
        if err != nil { log.Fatal(err) }
        printJSON(role.Resource)
    }
    cloud := identity.NewRoleReadCloud(service.Roles)
    listed, err := cloud.ListRoles(ctx,
        roles.WithRecordListQuery("description", "operations"))
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)
    selected, err := cloud.SearchRoles(ctx, "read*",
        identity.WithRoleSearchFilters(json.RawMessage(`{"options":{"immutable":false}}`)))
    if err != nil { log.Fatal(err) }
    printJSON(selected.Value)
    count, err := cloud.SearchRoles(ctx, "",
        identity.WithRoleSearchFilters(json.RawMessage(`"length(@)"`)))
    if err != nil { log.Fatal(err) }
    printJSON(count.Value)
}
```

## 권한과 지원 판정

[Keystone28.0.0 Role policy](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/role.py#L99-L119)의 `identity:list_roles`는 admin 또는 **system-scoped reader**, 또는 domain_id가 있는 domain-manager token 경로를 허용합니다. 단순 project reader/domain reader 권한을 전체 Role 목록 권한으로 해석하지 않습니다. [controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/roles.py#L45-L72)는 truthy domain_id query를 받으면 `identity:list_domain_roles`로 전환하며 이 정책은 admin 또는 system reader입니다. domain-manager의 global-role 목록 경로가 그 domain-specific 목록에 자동 적용되지는 않습니다. domain_id가 없을 때 서버는 domain_id=None filter로 global roles를 선택합니다. SDK는 token scope와 실제 서버 policy를 바꾸지 않습니다. [기본 role 규칙](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)과 [2025.2 정책 문서](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)가 이 user read 분류의 근거입니다.

검토 범위는 proxy roles와 cloud list_roles/search_roles의 세 named `go_mapping` 후보이며 실제 중앙 판정과 지원 수는 해당 실행의 검증 기록을 따릅니다. 새 FindRecord/find_role mapping은 제공하지 않습니다. 기존 native FindIdentity가 있다는 이유로 raw Resource.fetch controls·header kwargs·generic synchronized Resource/cache 계약까지 완료로 세지 않습니다. SDK-R1/C1/S1의 Connection-wide runtime·response cache·세션/버전 협상은 별도로 추적하고 전체 `supported` parity를 주장하지 않습니다.

기존 testcloud의 Gophercloud public testhelper와 Role transport fixture를 재사용한 leaf10개·cloud11개 새 그룹을 root가 검증했습니다. actual evidence·descriptor/default/null·loaded filter·raw marker/cap/break·Source link·opaque Proxy·actual cloud selection·clone·option/source/cancel·clean native retry 및 mixed physical 오류를 포함합니다. `go-openstacksdk-cycle3-identity-first.jsonl`에서 Role leaf와 cloud Role 그룹은 통과했으며 그 실행의 다른 Identity 실패를 전체 PASS로 바꾸지 않습니다. 수리 후 `go-openstacksdk-cycle3-identity-cloud-repaired.jsonl`에서 identity package가 통과했습니다. CPython3.14에서 고정 Role/Resource.list/Proxy._list/cloud methods의 32건을 HTTP adapter와 v3 selection collaborator만 대체해 비교했고, Keystone policy 두 rule와 controller 분기도 AST로 대조했습니다. 실제 cloud 인증 호출과 Source cache/lifecycle의 통합 성공으로 확대하지 않습니다.
