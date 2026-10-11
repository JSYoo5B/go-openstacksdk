# Nova attachment 목록·갱신과 server group 검색

이 API는 pinned openstacksdk Compute proxy의 `volume_attachments`, `update_volume_attachment`, `find_server_group`에 대응합니다. 기존 Gophercloud 기반 native `List`, `Get`, `Create`와 SDK 공통 `Resources`는 계속 사용할 수 있습니다. Python Resource의 raw 값·별칭·기본값이 필요하면 아래의 owned record API를 사용합니다.

| openstacksdk | go-openstacksdk | 기본 동작 |
|---|---|---|
| `conn.compute.volume_attachments(server, **query)` | `service.VolumeAttachments.ListRecords(ctx, serverID, options...)` | lazy iterator, `limit`·`offset`·`marker`, Body 필터, 응답 next link와 limit 기반 marker fallback |
| `conn.compute.update_volume_attachment(server, volume, volume_id=None, **attrs)` | `service.VolumeAttachments.UpdateRecord(ctx, serverID, currentVolumeID, options...)` | 기존 volume ID를 PUT 경로에 사용, swap 생략 시 같은 volume ID를 `volumeId`에 전송 |
| `conn.compute.find_server_group(name_or_id, ignore_missing=True, all_projects=False)` | `service.ServerGroups.FindRecord(ctx, nameOrID, options...)` | literal GET부터 시작; clean 400/403/404만 목록으로 fallback, 정확한 ID 또는 이름 비교 |

```python
for attachment in conn.compute.volume_attachments(
    server_id, limit=20, offset=0, max_items=100,
):
    print(attachment.to_dict())

# volume은 현재 연결된 볼륨 ID입니다. attachment_id는 경로 ID가 아닙니다.
conn.compute.update_volume_attachment(
    server_id, current_volume_id, delete_on_termination=False,
)
conn.compute.update_volume_attachment(
    server_id, current_volume_id, volume_id=replacement_volume_id,
)
group = conn.compute.find_server_group(group_name, ignore_missing=False)
```

## 실행 가능한 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 `main.go`로 저장합니다. `go run . -cloud dev -server SERVER_ID -group NAME`은 attachment 목록과 group 검색을 실행합니다. `-update-volume CURRENT_VOLUME_ID`를 지정하면 PUT을 실행하며, `-swap-to NEW_VOLUME_ID`로 대상 볼륨을 바꿉니다. 갱신은 Cinder detach/attach 완료를 기다리지 않으므로 반환 HTTP 응답과 실제 attachment 상태를 구분해야 합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    serverID := flag.String("server", "", "server ID to list attachments")
    groupName := flag.String("group", "", "server group exact ID or name")
    allProjects := flag.Bool("all-projects", false, "search server groups across projects")
    updateVolume := flag.String("update-volume", "", "current volume ID to update")
    replacement := flag.String("swap-to", "", "replacement volume ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.Compute, "2.89"))
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    if *serverID != "" {
        for value, err := range service.VolumeAttachments.ListRecords(ctx, *serverID,
            volumeattach.WithAttachmentLimit(20),
            volumeattach.WithAttachmentOffset(0),
            volumeattach.WithAttachmentMaxItems(100)) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    }
    if *groupName != "" {
        value, err := service.ServerGroups.FindRecord(ctx, *groupName,
            servergroups.WithGroupFindIgnoreMissing(false),
            servergroups.WithGroupFindAllProjects(*allProjects))
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
    }
    if *updateVolume != "" {
        if *serverID == "" { log.Fatal("-server is required for -update-volume") }
        options := []volumeattach.AttachmentUpdateOption{
            volumeattach.WithAttachmentAttribute("delete_on_termination", false),
        }
        if *replacement != "" {
            options = append(options, volumeattach.WithAttachmentSwapTo(*replacement))
        }
        value, err := service.VolumeAttachments.UpdateRecord(ctx,
            *serverID, *updateVolume, options...)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
        fmt.Printf("HTTP %d request-id=%s\n", value.StatusCode, value.Header.Get("X-Openstack-Request-Id"))
    }
}
```

`api.InServer(ctx, resource.ID(serverID))`로 만든 scope에서도 `ListRecords`·`UpdateRecord`를 호출할 수 있습니다. 명시 ID는 parent lookup을 보내지 않습니다. 이름으로 scope를 만들면 기존 공통 서버 resolver를 사용합니다.

## 반환 값과 옵션 계약

`AttachmentRecord`·`ServerGroupRecord`의 `Resource`는 선언된 Python 속성 이름으로 투영한 뷰입니다. 미수신 필드는 JSON null이며 attachment의 `volumeId` → `volume_id`, `bdm_uuid` → `bdm_id`, group의 `members` → `member_ids`를 변환합니다. 동일 별칭이 응답 객체에 함께 있으면 Python dictionary 순서처럼 마지막 값을 사용합니다. Group `rules`는 Source dict descriptor를 따라 nonnull 비객체를 `{}`로 투영합니다. Attachment의 ID는 `id`가 **없을 때만** `volumeId`로 대체합니다. 명시된 null·false·0을 다른 값으로 바꾸지 않습니다.

`Wire`는 서버가 실제로 전송한 row와 unknown 필드를 보존합니다. `Envelope`·`Header`·`StatusCode`는 실제 HTTP 증거입니다. Attachment `ServerID`는 고정 URI parent이며 응답의 `serverId`와 섞지 않습니다. body 없는 갱신 성공은 입력 seed가 있는 Resource와 HTTP 증거를 반환하며, 실제 수신 row가 없으므로 Wire는 nil입니다. 이 record 뷰의 `location`은 null이며 Connection의 전체 Python cloud-location 계약은 별도 공통 진행 항목입니다.

목록은 `WithAttachmentQuery("device", nil)`처럼 canonical Body 속성으로 로컬 필터를 전달합니다. Wire alias와 unknown query는 Source처럼 버립니다. `WithAttachmentPaginated(false)`는 한 page만 소비하고, `WithAttachmentMaxItems(n)`은 **필터 적용 전 받은 row 개수**를 제한합니다. `MaxItems=0`은 무제한입니다. `WithAttachmentLimit(0)`은 명시 zero를 보존하며 양수 cap이 있으면 Source처럼 limit 힌트로 대체합니다. 양수 limit으로 받은 짧은 page도 marker로 이어가며 빈 page에서 멈춥니다. next link는 같은 service origin·collection 안에서만 허용하고 반복 cursor를 차단합니다.

갱신 옵션은 omitted·JSON null·false·0·빈 값을 구분해 snapshot을 저장합니다. `WithAttachmentAttribute("volume_id", nil)`은 Python의 named `volume_id=None`처럼 기존 volume 유지이고, wire alias `WithAttachmentAttribute("volumeId", nil)`은 실제 null Body 값입니다. unknown attrs는 marshal 전에 버리며 ID·parent route를 attrs로 바꾸면 HTTP 전에 `resource.ErrInvalidOption`입니다. `WithAttachmentSwapTo`와 다른 옵션의 마지막 값이 적용됩니다.

Find는 성공한 GET을 바로 반환하고, fallback에서 후보 하나를 찾더라도 뒤 page까지 확인합니다. 중복은 `resource.AmbiguousError`, strict missing은 `resource.ErrNotFound`이며 기본 missing은 nil,nil입니다. `all_projects=True`는 fallback 목록에만 전송하고 direct GET query에는 들어가지 않습니다. 목록 HTTP 오류와 read·Close·cancel·source mutation·retry callback 오류가 섞인 GET rejection은 missing으로 억제하거나 fallback하지 않습니다. `errors.As`로 native Gophercloud rejection 또는 `resource.ResponseError`의 실제 status·body·headers를 확인할 수 있습니다.

선택한 client microversion은 그대로 사용합니다. 미선택 시 공통 Nova discovery로 Source resource ceiling과 advertised maximum 중 작은 버전을 사용합니다(Attachment 2.89, ServerGroup 2.64). 원본 service client를 변경하지 않습니다. 이 helper는 Source의 local Resource mutable-state API를 재현하지 않으며 explicit string ID와 owned records를 반환합니다. typed pagination은 음수·중복/비정수 limit·offset을 HTTP 전에 거부합니다. 로컬 필터는 공통 Go JSON 계약을 사용하므로 Python의 `True == 1`과 달리 bool과 number를 구분하고 exact decimal을 비교합니다. Python의 malformed nonmapping attribute exception 대신 비객체의 object filter는 mismatch입니다.

검증은 기존 Gophercloud testhelpers를 사용하는 `internal/testcloud` fixture와 기존 leaf request recorder를 재사용합니다. 신규 계약 테스트는 lazy paging, offset·marker, cap-before-filter, raw view·alias, PUT defaults·snapshots, discovery, Find missing·duplicates·late errors 및 실제 body read/Close 실패를 검사합니다. 문서 example 빌드는 실제 OpenStack에 대한 실행·인증 성공을 의미하지 않습니다.
