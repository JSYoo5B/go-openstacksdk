# Neutron floating-IP Port Forwarding

`conn.NetworkV2(ctx).PortForwarding`의 `Record` 메서드는 고정 openstacksdk의
PortForwarding proxy를 제공합니다. Python의 두 naming family는 같은 resource와
`/floatingips/{floatingip_id}/port_forwardings`를 사용합니다. Go는 parent를 항상
앞의 명시적 ID 인수로 고정하므로 Python의 서로 다른 인수 순서를 기억할 필요가 없습니다.

| openstacksdk | Go | 기본 동작 |
|---|---|---|
| `create_port_forwarding(floatingip_id=fip_id, **attrs)` / `create_floating_ip_port_forwarding(fip_id, **attrs)` | `CreateRecord(ctx, fipID, options...)` | 주어진 Body attrs만 envelope POST |
| `update_port_forwarding(pf_id, fip_id, **attrs)` / `update_floating_ip_port_forwarding(fip_id, pf_id, **attrs)` | `UpdateRecord(ctx, fipID, pfID, options...)` | 변경 attrs만 PUT; 생략하면 HTTP 없이 ID-only 결과 |
| `get_port_forwarding(pf_id, fip_id, fields=[...])` / `get_floating_ip_port_forwarding(fip_id, pf_id, fields=[...])` | `GetRecord(ctx, fipID, pfID, WithRecordFields(...))` | literal scoped GET와 반복 fields query |
| `port_forwardings(fip_id, **query)` / `floating_ip_port_forwardings(fip_id, **query)` | `ListRecords(ctx, fipID, options...)` | lazy pagination, 서버 query와 local filter 분리 |
| `find_port_forwarding(pf_id, fip_id)` / `find_floating_ip_port_forwarding(fip_id, pf_id)` | `FindRecord(ctx, fipID, nameOrID, options...)` | GET 후 clean 400/403/404만 scoped 목록의 exact ID/name 검색 |
| `delete_port_forwarding(pf_id, fip_id)` / `delete_floating_ip_port_forwarding(fip_id, pf_id)` | `DeleteRecord(ctx, fipID, pfID, options...)` | clean 404를 기본으로 무시 |

```python
created = conn.network.create_floating_ip_port_forwarding(
    floating_ip_id, internal_port_id=port_id,
    internal_ip_address="10.0.0.5", internal_port=22,
    external_port=2222, protocol="tcp", description="SSH")
selected = conn.network.get_floating_ip_port_forwarding(
    floating_ip_id, created.id, fields=["id", "external_port"])
for forwarding in conn.network.floating_ip_port_forwardings(
    floating_ip_id, protocol="tcp", description="SSH"):
    print(forwarding.id)
found = conn.network.find_floating_ip_port_forwarding(floating_ip_id, created.id)
# 필요할 때: conn.network.delete_floating_ip_port_forwarding(
#     floating_ip_id, created.id, ignore_missing=False)
```

## 독립 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`로 저장합니다.
`go run . -cloud dev -floating-ip FIP_ID -id PF_ID`는 목록·GET·Find를 실행합니다.
생성·수정·삭제는 각각 `-create`, `-description`, `-delete`로 명시한 경우 실행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding"
)

func printForwarding(value *portforwarding.PortForwardingRecord) {
    if value == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("parent=%s status=%d\n%s\n", value.ParentID, value.StatusCode, body)
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    floatingIP := flag.String("floating-ip", "", "floating IP ID")
    id := flag.String("id", "", "port forwarding ID to get and find")
    port := flag.String("port", "", "internal Neutron port ID")
    address := flag.String("address", "10.0.0.5", "internal IP address")
    create := flag.Bool("create", false, "create a port forwarding")
    description := flag.String("description", "", "replace description, including empty")
    remove := flag.String("delete", "", "port forwarding ID to delete")
    flag.Parse()
    if *floatingIP == "" { log.Fatal("-floating-ip is required") }
    update := false
    flag.Visit(func(f *flag.Flag) { if f.Name == "description" { update = true } })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.PortForwarding
    for row, err := range api.ListRecords(ctx, *floatingIP,
        portforwarding.WithRecordListQuery("protocol", "tcp"),
        portforwarding.WithRecordListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printForwarding(row)
    }
    if *id != "" {
        row, err := api.GetRecord(ctx, *floatingIP, *id,
            portforwarding.WithRecordFields("id", "external_port"))
        if err != nil { log.Fatal(err) }
        printForwarding(row)
        found, err := api.FindRecord(ctx, *floatingIP, *id)
        if err != nil { log.Fatal(err) }
        printForwarding(found)
        if update {
            row, err := api.UpdateRecord(ctx, *floatingIP, *id,
                portforwarding.WithRecordAttribute("description", *description))
            if err != nil { log.Fatal(err) }
            printForwarding(row)
        }
    }
    if *create {
        row, err := api.CreateRecord(ctx, *floatingIP,
            portforwarding.WithRecordAttribute("internal_port_id", *port),
            portforwarding.WithRecordAttribute("internal_ip_address", *address),
            portforwarding.WithRecordAttribute("internal_port", 22),
            portforwarding.WithRecordAttribute("external_port", 2222),
            portforwarding.WithRecordAttribute("protocol", "tcp"))
        if err != nil { log.Fatal(err) }
        printForwarding(row)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *floatingIP, *remove,
            portforwarding.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## 옵션과 결과

`WithRecordAttribute`와 `WithRecordMutationOpts`는 JSON을 복사하여 생략과 명시적인
`null`, `false`, `0`, 빈 문자열·배열·object를 구별합니다. Source처럼 필수 필드의
유효성은 Neutron에 맡깁니다. `external_port_range` 같은 미선언 extension은 Body와
결과에 보존합니다. 이 Resource는 `NetworkResource`를 상속하지 않으므로
`if_match` / `if_revision` attrs는 일반 Body extension이며 조건부 Header가 아닙니다.

`PortForwardingRecord.Resource`는 `id`, `name`, 여섯 port forwarding descriptor,
nullable `location`과 unknown Body extensions의 Python view입니다. `internal_port`와
`external_port`는 Source int descriptor 규칙을 적용합니다. 숫자 문자열은 정수로,
float는 절삭한 정수로 보이며 bool은 bool로 유지됩니다. `Wire`는 원래 row의 JSON을,
`Envelope`, `Header`, `StatusCode`는 실제 응답을 보존합니다. `ParentID`는 URI scope를
따로 보존합니다. 서버가 주지 않은 값은 GET의 ID 또는 mutation attrs에서만 유지하고,
200..399의 bodyless·malformed JSON 응답에는 실제 wire row를 만들어내지 않습니다.
수정 no-op은 `StatusCode=0`, `Wire=nil`로 HTTP가 없었다는 점이 드러납니다.

List의 서버 query는 `fields`, `internal_port_id`, `external_port`, `protocol`,
`sort_key`, `sort_dir`, `limit`, `marker`입니다. 반복값과 Requests의 bool 표현을
보존하고 `null` query 값은 생략합니다. 그 외 선언된 Body attrs는 Python view에
local filter를 적용하고 알 수 없는 query는 무시합니다. `WithRecordListLimit`,
`WithRecordListMarker`, `WithRecordListPaginated(false)`, `WithRecordListMaxItems`를
지원합니다. max-items는 local filter 이전의 raw row 수를 제한합니다. 반복을 멈추면
다음 페이지를 읽지 않으며, 같은 Seq를 다시 반복하면 독립적으로 조회합니다.
Source plural `port_forwardings_links`, `links`, top-level `next`, HTTP Link와
limit/마지막 wire ID marker fallback을 사용합니다. Dictionary `links.next`는 Go의
명시적인 호환 확장이며 Source의 dictionary-link 구현 오류를 재현하지 않습니다.

Find는 기본적으로 미존재에 `nil`을 반환합니다. `WithRecordFindIgnoreMissing(false)`로
미존재 오류를 요청할 수 있습니다. Source의 GET-first 단계는 `floatingip_id`와
`WithRecordFindQuery`의 kwargs를 query로 보내고, fallback에서는 일반 List 규칙으로
분류합니다. 이 점은 일반 `GetRecord`의 query와 다릅니다. 전체 fallback을 끝까지
읽어 duplicate와 나중 페이지 실패를 검사합니다. GET의 mixed handling failure는
fallback하지 않습니다. Delete도 실제 clean 404만 무시하며 read/close·retry·취소 오류를
미존재로 덮어쓰지 않습니다.

Go는 literal UTF-8 ID를 받으며 mutable Python Resource 객체 대신 독립된 값을
반환합니다. parent/member ID는 각 segment로 escape하며 빈·control·dot segment를
HTTP 전에 거부합니다. `location`에 Python connection의 계산된 프로젝트 정보를
주입하지 않고 null을 유지합니다. 선택한 endpoint/provider/type/version의 변경,
다른 parent/path/origin의 pagination, route/transport control attrs와 protocol/auth
Header override는 오류입니다. Find의 `base_path`, `list_base_path`, `microversion`
kwargs는 고정 scope를 바꾸므로 거부합니다. 상속된 mutable Resource/session API 전체
동등성은 별도 R1 항목이며 이 여섯 named proxy API의 구현과 구분합니다.

기존 generated `Create/Get/List/Update/Delete`, native `PortForwarding` 타입과
`InFloatingIP(ctx, resource.Ref)` scope는 그대로 사용할 수 있습니다.
[전체 서비스 안내](README.md), [native CRUD](native-crud.md)를 함께 참고합니다.
