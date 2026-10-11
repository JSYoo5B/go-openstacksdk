# Neutron Resource 속성 생성·수정·삭제

이 문서는 OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
AddressGroup, AddressScope, SubnetPool, Trunk, QoSPolicy proxy 선언 12개에
대응하는 Go 호출을 설명합니다. 기존 typed `Create`·`Update`는 그대로 사용하고,
Python의 `**attrs` 요청 본문을 재현할 때는 `CreateWithAttributes`와
`UpdateWithAttributes`를 사용합니다.

| Python 호출 | Connection 아래 Go API | 경로·envelope |
|---|---|---|
| `create_address_group(**attrs)` | `service.API.SecurityAddressGroups.CreateWithAttributes(ctx, options...)` | `address-groups`·`address_group` |
| `update_address_group(id, **attrs)` | `service.API.SecurityAddressGroups.UpdateWithAttributes(ctx, id, options...)` | `address-groups/{id}`·`address_group` |
| `delete_address_group(id, ignore_missing=True)` | `service.API.SecurityAddressGroups.DeleteWithOptions(ctx, id, options...)` | `address-groups/{id}` |
| `create_address_scope(**attrs)` | `service.API.AddressScopes.CreateWithAttributes(ctx, options...)` | `address-scopes`·`address_scope` |
| `update_address_scope(id, **attrs)` | `service.API.AddressScopes.UpdateWithAttributes(ctx, id, options...)` | `address-scopes/{id}`·`address_scope` |
| `delete_address_scope(id, ignore_missing=True)` | `service.API.AddressScopes.DeleteWithOptions(ctx, id, options...)` | `address-scopes/{id}` |
| `create_subnet_pool(**attrs)` | `service.API.SubnetPools.CreateWithAttributes(ctx, options...)` | `subnetpools`·`subnetpool` |
| `delete_subnet_pool(id, ignore_missing=True)` | `service.API.SubnetPools.DeleteWithOptions(ctx, id, options...)` | `subnetpools/{id}` |
| `create_trunk(**attrs)` | `service.API.Trunks.CreateWithAttributes(ctx, options...)` | `trunks`·`trunk` |
| `delete_trunk(id, ignore_missing=True)` | `service.API.Trunks.DeleteWithOptions(ctx, id, options...)` | `trunks/{id}` |
| `create_qos_policy(**attrs)` | `service.API.QoSPolicies.CreateWithAttributes(ctx, options...)` | `qos/policies`·`policy` |
| `delete_qos_policy(id, ignore_missing=True)` | `service.API.QoSPolicies.DeleteWithOptions(ctx, id, options...)` | `qos/policies/{id}` |

각 leaf의 `New(client)` 또는 `network/v2.New(client)`에서 같은 호출을 사용할 수
있습니다. ID는 직접 경로에 붙이고 이름 조회는 수행하지 않습니다. 이름을 가진 입력은
먼저 해당 API의 `FindIdentity`로 해결한 뒤 반환된 `ID`를 전달합니다.

## 속성의 기본값과 별칭

`WithAttribute(key, value)`는 옵션을 만들 때 JSON 값을 복사합니다. 옵션을 재사용해도
원래 map·slice 변경이 요청을 바꾸지 않습니다. 같은 wire key를 여러 번 지정하면
뒤의 옵션 값이 적용됩니다. 각 요청은 독립 본문을 만듭니다.

| 입력 | 본문 |
|---|---|
| 옵션 미지정 | 해당 key 없음 |
| `WithAttribute("name", "")` | `"name": ""` |
| `WithAttribute("is_shared", false)` | `"shared": false` |
| `WithAttribute("default_prefix_length", 0)` | `"default_prefixlen": 0` |
| `WithAttribute("description", nil)` | `"description": null` |
| `WithAttribute("addresses", []string{})` | `"addresses": []` |
| `WithAttribute("addresses", []string(nil))` | `"addresses": null` |

canonical Body 별칭은 기존 선택 조회에 쓰는 pinned Source MRO 표를 재사용합니다.
AddressScope도 `id`, `name`, `ip_version`, `project_id`, `tenant_id`, `is_shared`를
같은 방식으로 처리합니다. SubnetPool의 `minimum_prefix_length`와
`maximum_prefix_length`는 `min_prefixlen`과 `max_prefixlen`으로, Trunk의
`is_admin_state_up`은 `admin_state_up`으로 바뀝니다. wire 이름도 그대로 쓸 수 있습니다.

다섯 Python Resource 모두 `_allow_unknown_attrs_in_body=True`이므로 알 수 없는
확장 key도 본문에 남습니다. `fields` 같은 query 전용 이름을 생성 속성으로 주면
본문 key로 보내며 조회 query로 바꾸지 않습니다. 확장 schema와 필수값 검증은
Neutron이 수행합니다. 값은 Go의 JSON 표현대로 보내고 Python 객체의 descriptor
접근·변환을 수행하지 않습니다.

속성 없이 생성하면 `{"address_group": {}}` 같은 빈 resource envelope를 보냅니다.
AddressGroup의 `addresses`·Trunk의 `port_id`를 Go에서 미리 필수 검사하지 않으며,
Trunk의 `sub_ports: []`, AddressScope의 `name: ""`·`ip_version: 0`,
SubnetPool의 `name: ""`·`prefixes: null`, QoSPolicy의 `name: ""`도 자동 추가하지
않습니다. 이러한 native builder 동작이 필요한 기존 호출은
[typed native CRUD](native-crud.md)를 사용합니다.

AddressGroup과 AddressScope의 속성 없는 수정은 HTTP를 보내지 않고 ID만 담긴 새
native 모델을 반환합니다. 빈 문자열·false·null을 지정한 속성은 실제 PUT을 보냅니다.
기존 Resource 객체의 같은 값을 다시 지정했을 때 변경 여부를 판단하는 캐시·dirty
tracking은 제공하지 않습니다. 수정 ID는 위치 인자로 전달하며 `WithAttribute("id", ...)`
옵션은 HTTP 전에 거부합니다.

```python
# Python: canonical Body 속성만 지정하면 필요한 key만 본문에 들어간다.
ag = conn.network.create_address_group(name="web", addresses=[])
scope = conn.network.create_address_scope(name="scope", ip_version=4, is_shared=False)
pool = conn.network.create_subnet_pool(
    name="pool", prefixes=["10.60.0.0/16"], default_prefix_length=24,
    minimum_prefix_length=16, maximum_prefix_length=28, is_shared=False,
)
trunk = conn.network.create_trunk(port_id=parent_port_id, is_admin_state_up=False, sub_ports=[])
policy = conn.network.create_qos_policy(name="gold", is_shared=False, is_default=False)

# ID만 주고 변경 속성이 없으면 Resource.commit은 요청하지 않는다.
unchanged = conn.network.update_address_group(ag.id)
conn.network.update_address_group(ag.id, description="")
conn.network.update_address_scope(scope.id, is_shared=False)
conn.network.delete_address_group(ag.id)  # 미존재 기본 무시
conn.network.delete_address_scope(scope.id, ignore_missing=False)
```

## 삭제 옵션과 응답

다섯 `DeleteOpts`는 `IgnoreMissing *bool`만 제공합니다. nil이면 기본값 true입니다.
`WithDeleteIgnoreMissing(false)`는 404를 오류로 반환하고, `WithDeleteOptions`는 전체
설정을 복사해 교체합니다. 옵션 순서에서 마지막 설정이 적용됩니다. 이 다섯 Python
선언에는 revision 인자가 없어 public `Revision` 옵션을 추가하지 않았습니다.

```go
err := api.DeleteWithOptions(ctx, id) // 기본값: 미존재 무시
err = api.DeleteWithOptions(ctx, id, addressgroups.WithDeleteIgnoreMissing(false))
strict := false
option := addressgroups.WithDeleteOptions(addressgroups.DeleteOpts{IgnoreMissing: &strict})
// option은 위 false를 복사했으므로 이후 strict 변수의 변경에 영향받지 않는다.
```

새 호출의 HTTP 성공 범위는 200..399입니다. 생성·수정은 native typed 모델로 decode하고,
삭제는 성공 본문을 decode하지 않습니다. 실패는 `resource.OperationError`로 감싸며
`errors.As`로 원래 `gophercloud.ErrUnexpectedResponseCode`의 status·body·header를 읽을 수
있습니다. 취소는 `errors.Is(err, context.Canceled)` 및 사용자 지정
`context.Cause(ctx)`를 보존합니다. 실제 404와 함께 응답 read/close·transport·취소 오류가
발생하면 미존재 성공으로 처리하지 않습니다.

기존 `Delete`와 `Remove`는 native 202·204 규칙을 유지합니다. `Remove(ctx, resource.ID(id))`
역시 미존재를 기본 무시하지만, 위 새 호출은 성공 status와 응답 본문 정책까지 통일합니다.

## 전체 Connection main

아래 프로그램은 `NEUTRON_MUTATION_KIND`를 `address-group`, `address-scope`,
`subnet-pool`, `trunk`, `qos-policy` 중 하나로 선택해서 생성합니다. `trunk`는
`PARENT_PORT_ID`를 사용합니다. `NEUTRON_DELETE_ID`가 있으면 해당 ID 삭제를 수행하고,
`NEUTRON_UPDATE_ID`가 있으면 AddressGroup 또는 AddressScope의 빈 값 수정을 수행합니다.
환경 변수를 설정하지 않으면 인증 후 사용법만 출력합니다.

```go
package main

import (
    "context"
    "fmt"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    kind := os.Getenv("NEUTRON_MUTATION_KIND")
    deleteID, updateID := os.Getenv("NEUTRON_DELETE_ID"), os.Getenv("NEUTRON_UPDATE_ID")
    switch kind {
    case "address-group":
        api := service.API.SecurityAddressGroups
        value, err := api.CreateWithAttributes(ctx, addressgroups.WithAttribute("name", "web"), addressgroups.WithAttribute("addresses", []string{}))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if updateID != "" {
            if _, err := api.UpdateWithAttributes(ctx, updateID, addressgroups.WithAttribute("description", "")); err != nil { panic(err) }
        }
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "address-scope":
        api := service.API.AddressScopes
        value, err := api.CreateWithAttributes(ctx, addressscopes.WithAttribute("name", "scope"), addressscopes.WithAttribute("ip_version", 4), addressscopes.WithAttribute("is_shared", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if updateID != "" {
            if _, err := api.UpdateWithAttributes(ctx, updateID, addressscopes.WithAttribute("is_shared", false)); err != nil { panic(err) }
        }
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID, addressscopes.WithDeleteIgnoreMissing(false)); err != nil { panic(err) }
        }
    case "subnet-pool":
        api := service.API.SubnetPools
        value, err := api.CreateWithAttributes(ctx, subnetpools.WithAttribute("name", "pool"), subnetpools.WithAttribute("prefixes", []string{"10.60.0.0/16"}), subnetpools.WithAttribute("default_prefix_length", 24), subnetpools.WithAttribute("minimum_prefix_length", 16), subnetpools.WithAttribute("maximum_prefix_length", 28), subnetpools.WithAttribute("is_shared", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "trunk":
        api := service.API.Trunks
        value, err := api.CreateWithAttributes(ctx, trunks.WithAttribute("port_id", os.Getenv("PARENT_PORT_ID")), trunks.WithAttribute("is_admin_state_up", false), trunks.WithAttribute("sub_ports", []map[string]any{}))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "qos-policy":
        api := service.API.QoSPolicies
        value, err := api.CreateWithAttributes(ctx, policies.WithAttribute("name", "gold"), policies.WithAttribute("is_shared", false), policies.WithAttribute("is_default", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    default:
        fmt.Println("set NEUTRON_MUTATION_KIND to address-group, address-scope, subnet-pool, trunk or qos-policy")
    }
}
```

## Go profile의 경계와 남은 Source 계약

반환값은 Python의 mutable Resource가 아닌 기존 native 모델입니다. 생성·수정 응답에
요청 속성을 합성하거나 unknown 응답 필드를 저장하지 않습니다. 선택한 ID와 응답 ID가
다르면 서버가 반환한 ID를 유지합니다. 빈 수정의 반환값만 요청 ID를 담습니다.
Python은 bodyless 또는 JSON 해석 불가 성공 응답에서도 요청 속성 view를 유지할 수 있으나,
Go 생성·수정은 native decode 실패를 반환합니다. SubnetPool 생성은 native decoder가
`default_prefixlen`·`min_prefixlen`·`max_prefixlen`을 모두 요구하고, Trunk 시각은 native
RFC3339 규칙을 따릅니다. [선택 조회](python-read-delete.md)의 sparse SubnetPool 보완은
생성·수정 decoder에 적용하지 않습니다.

`base_path`·`microversion`·`connection`·`_synchronized`·`resource_type`·
`__conflicting_attrs`·`location`은 이 Body API에서 HTTP 전에 거부합니다. Source의
Proxy/Resource 제어 인자, computed location과 metadata 처리, Resource 인스턴스 입력과
동일 객체 반환, 저장된 기존 속성에 대한 no-op/dirty 판단, adapter/session 정책·동적
microversion과 Python 예외 클래스는 별도 구현 대상으로 남습니다. 어떤 kwargs도
완전히 같은 Resource 객체 동작을 제공한다는 의미는 아닙니다.

검증은 기존 public Gophercloud testhelper 기반 `internal/testcloud`와 각 leaf의 기존
native wire fixture를 재사용합니다. 새 테스트는 omitted 본문, canonical 별칭,
false/0/null/빈 값, unknown Body key, 옵션 snapshot, 빈 수정 무요청, 성공 상태·native
decode 실패·원래 HTTP 오류·취소 원인, 삭제의 404 및 physical read 실패를 확인합니다.
실제 OpenStack 인증과 API 실행은 별도 환경에서 검증해야 합니다.

Gophercloud는 HTTP204 응답 body를 모델로 decode하지 않습니다. 새 create/update profile은 이 native 결과를 유지하여 `nil, nil`을 반환하고 입력 attrs로 관측한 응답 모델을 합성하지 않습니다. 최종 200..399 허용은 모든 상태에서 nonnil 모델을 보장한다는 뜻이 아닙니다. Source의 mutable Resource seed/cache 결과와 이 경계가 다릅니다.
