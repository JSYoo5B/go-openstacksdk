# Trunk subport와 address group address action

pinned openstacksdk의 네 action을 builder interface 없이 호출합니다. 각 메서드는 기존 resource ID와 JSON으로 표현 가능한 action 값을 받아 한 번의 PUT을 전송합니다. 입력은 JSON snapshot으로 고정되며, 실제 반환 row와 HTTP 증거는 별도로 보존합니다.

| openstacksdk | go-openstacksdk | 경로·body |
|---|---|---|
| `conn.network.add_trunk_subports(trunk, subports)` | `service.Trunks.AddSubportsRecord(ctx, trunkID, subports)` | PUT `/trunks/{id}/add_subports`, `{"sub_ports":...}` |
| `conn.network.delete_trunk_subports(trunk, subports)` | `service.Trunks.RemoveSubportsRecord(ctx, trunkID, subports)` | PUT `/trunks/{id}/remove_subports`, `{"sub_ports":list(subports)}` |
| `conn.network.add_addresses_to_address_group(group, addresses)` | `service.SecurityAddressGroups.AddAddressesRecord(ctx, groupID, addresses)` | PUT `/address-groups/{id}/add_addresses`, `{"addresses":...}` |
| `conn.network.remove_addresses_from_address_group(group, addresses)` | `service.SecurityAddressGroups.RemoveAddressesRecord(ctx, groupID, addresses)` | PUT `/address-groups/{id}/remove_addresses`, `{"addresses":...}` |

```python
trunk = conn.network.add_trunk_subports(
    trunk_id,
    [{"port_id": child_port_id, "segmentation_type": "vlan", "segmentation_id": 100}],
)
trunk = conn.network.delete_trunk_subports(trunk, [{"port_id": child_port_id}])
group = conn.network.add_addresses_to_address_group(group_id, ["192.0.2.0/24"])
group = conn.network.remove_addresses_from_address_group(group, ["192.0.2.0/24"])
```

## 실행 가능한 Go main

[설치 안내](../../docs/install.md)로 모듈을 준비하고 아래를 `main.go`로 저장합니다. 예를 들어 `go run . -cloud dev -group GROUP_ID -add-address 192.0.2.0/24`로 address를 추가하거나, `-trunk TRUNK_ID -add-subport PORT_ID -segmentation-id 100`으로 VLAN subport를 추가합니다. `-remove-address`·`-remove-subport`는 각각 제거를 실행합니다. 생성·삭제 대기나 별도의 GET refresh는 수행하지 않습니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    trunkID := flag.String("trunk", "", "trunk ID")
    addPort := flag.String("add-subport", "", "subport to add")
    removePort := flag.String("remove-subport", "", "subport to remove")
    segmentationID := flag.Int("segmentation-id", 100, "VLAN segmentation ID")
    groupID := flag.String("group", "", "address group ID")
    addAddress := flag.String("add-address", "", "address or CIDR to add")
    removeAddress := flag.String("remove-address", "", "address or CIDR to remove")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    var trunk *trunks.TrunkActionRecord
    if *addPort != "" {
        if *trunkID == "" { log.Fatal("-trunk is required") }
        trunk, err = service.Trunks.AddSubportsRecord(ctx, *trunkID,
            []map[string]any{{"port_id": *addPort, "segmentation_type": "vlan",
                "segmentation_id": *segmentationID}})
        if err != nil { log.Fatal(err) }
        printJSON(trunk.Resource)
    }
    if *removePort != "" {
        if *trunkID == "" { log.Fatal("-trunk is required") }
        var options []trunks.SubportActionOption
        if trunk != nil {
            options = append(options, trunks.WithSubportActionSeed(trunk.Resource))
        }
        trunk, err = service.Trunks.RemoveSubportsRecord(ctx, *trunkID,
            []map[string]any{{"port_id": *removePort}}, options...)
        if err != nil { log.Fatal(err) }
        printJSON(trunk.Resource)
    }
    var group *addressgroups.AddressActionRecord
    if *addAddress != "" {
        if *groupID == "" { log.Fatal("-group is required") }
        group, err = service.SecurityAddressGroups.AddAddressesRecord(ctx,
            *groupID, []string{*addAddress})
        if err != nil { log.Fatal(err) }
        printJSON(group.Resource)
    }
    if *removeAddress != "" {
        if *groupID == "" { log.Fatal("-group is required") }
        var options []addressgroups.AddressActionOption
        if group != nil {
            options = append(options, addressgroups.WithAddressActionSeed(group.Resource))
        }
        group, err = service.SecurityAddressGroups.RemoveAddressesRecord(ctx,
            *groupID, []string{*removeAddress}, options...)
        if err != nil { log.Fatal(err) }
        printJSON(group.Resource)
    }
}
```

## Source 동작과 반환 값

필수 `subports`·`addresses` 인자는 메서드에서 생략할 수 없습니다. 빈 slice는 `[]`, nil slice와 nil은 JSON null이고, false·0·빈 string도 입력 그대로 남습니다. Source add-subports와 address 두 action은 annotation 외에 client schema 검사를 하지 않으므로 이 값들을 서버까지 전달합니다. 중첩 subport map의 unknown extension·생략한 key·명시 null을 임의로 보충하거나 삭제하지 않습니다.

Trunk 제거만 Source proxy의 `list(subports)` 변환이 있습니다. Go도 JSON 배열은 그대로, JSON object는 순서대로 key 배열, string은 Unicode codepoint 배열로 변환합니다. null·bool·number는 iterable이 아니므로 `resource.ErrInvalidOption`이며 HTTP를 보내지 않습니다. `json.RawMessage`를 사용하면 dictionary의 원래 순서를 보존할 수 있습니다. 일반 Go map은 JSON encoder의 key 순서를 사용합니다. Python의 임의 generator·custom object 대신 Go API는 JSON으로 표현 가능한 값을 받습니다.

`Resource`는 선언된 Source descriptor 뷰이고 `Wire`는 실제 response row입니다. `Envelope`·`Header`·`StatusCode`는 실제 HTTP 응답, `Input`은 제출한 action 값입니다. `Input`을 반환된 resource 속성으로 간주하지 않습니다. 미수신 declared field는 null이며 trunk `tags`만 Source default `[]`입니다. list descriptor는 nonnull scalar를 한 원소 list로, trunk `is_admin_state_up`은 Python truthiness로 투영합니다. `project_id`가 없을 때만 `tenant_id`를 alias fallback으로 사용하고, 명시 null은 유지합니다.

Trunk Source action은 envelope-free response object를 `_body`에 직접 update합니다. 그래서 응답 `trunk` envelope를 해제하지 않고 새 unknown field나 canonical alias를 Resource translator처럼 정규화하지 않습니다. 이러한 값은 Wire에서 확인할 수 있습니다. Address group은 `address_group` envelope를 해제하고 canonical/wire Body aliases, unknown extensions와 `self` 제거를 Source translator처럼 처리합니다. 반환시 seed에 있던 속성은 응답으로 갱신되지 않으면 보존됩니다. `WithSubportActionSeed`·`WithAddressActionSeed`는 기존 owned Resource 뷰를 복사하여 Python의 cached Resource 입력을 연결하는 옵션이며, 명시 route ID를 변경하거나 caller record를 수정하지 않습니다. alias가 충돌하는 seed는 모호한 Go map 순서를 사용하지 않고 HTTP 전에 거부합니다. record `location`은 null이며 완전한 Connection location 계약은 공통 진행 항목입니다.

Source처럼 HTTP 200..399를 성공 범위로 처리합니다. Trunk는 항상 `resp.json()`을 호출하므로 body 없는 204도 JSON decode 오류이며 실제 성공 HTTP 증거를 유지합니다. Address group은 Resource translator가 non-JSON success body를 무시하므로 body 없는 204에서는 seed view와 HTTP 증거만 있고 Wire는 nil입니다. physical Read·Close·cancel·source 변경 오류는 이 decode/no-body 처리보다 먼저 보존되며, 오류를 성공으로 바꾸거나 mutation을 다시 보내지 않습니다. clean 404도 두 action에서 오류입니다. native retry가 JSON body를 바꾸면 최초 rejection과 `ErrInvalidOption`을 함께 반환합니다. `errors.As`로 Gophercloud HTTP rejection 또는 `resource.ResponseError`를 확인할 수 있습니다.

옵션은 concrete function과 snapshot으로 처리합니다. Headers·seed·retained option callback을 나중에 변경해도 적용된 값이 바뀌지 않습니다. selected provider·endpoint·ResourceBase·service type·microversion을 operation 동안 고정하고, custom JSON marshaler나 retry hook이 이를 변경하면 중단합니다. ID는 literal string으로 받아 한 번 escape하며 credentials·auth·route headers는 SDK가 관리합니다.

## 서비스 우선순위와 정책 근거

이 네 action은 핵심 Network user 작업에 배치합니다. 고정 [Neutron 27.0.0 trunk policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/trunk.py)는 두 subport mutation에 project scope의 admin 또는 해당 project member를 명시합니다. 27.0.0 [address group policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/address_group.py)는 address mutation을 개별 등록하지 않습니다. action controller·[기본 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py)의 경로와 oslo.policy의 missing-rule fallback을 합쳐보면 stock default는 admin 또는 resource owner라고 판단됩니다. 최신 [정책 문서](https://docs.openstack.org/neutron/2026.2/configuration/policy.html)는 address mutation에도 member rule을 명시하므로 이 최신 내용을 27.0.0 규칙으로 소급하지 않습니다. 실제 deployment policy override는 서버가 판단합니다.

테스트는 기존 leaf HTTP recorder와 Gophercloud 공개 `testhelper` 기반 `internal/testcloud`를 재사용합니다. 두 package의 신규 계약 테스트는 요청 body와 route, descriptor view·unknown 보존, Source bodyless 차이, option snapshot, cancellation·marshal mutation, 실제 accepted/rejected Read·Close·source-restoration 오류, retry body mutation 및 HTTP 오류 증거를 검증합니다. 문서 main build와 fixture 검증은 인증된 실제 cloud acceptance를 대신하지 않습니다.
