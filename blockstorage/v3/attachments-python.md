# Cinder attachment reservation과 summary 목록

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [Attachment proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1495-L1551)와 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/attachment.py)입니다.

Cinder attachment API는 볼륨을 소비하는 Nova·Glance·Ironic 등의 서비스가 사용하는 API입니다. 일반 서버에 볼륨을 연결하는 흐름에는 프로젝트의 cloud `AttachVolume` helper를 사용합니다. 이 문서의 reservation은 서버 연결 완료를 기다리거나 Nova attach를 실행하지 않습니다.

| openstacksdk | Go | 요청 |
|---|---|---|
| `create_attachment(volume_id)` | `Attachments.CreateRecord(ctx, volumeID)` | `POST /attachments`, instance 없이 volume_uuid만 |
| `create_attachment(volume_id, instance=server_id, connector=connector, mode="rw")` | `CreateRecord`의 concrete field 옵션 | truthy instance를 instance_uuid로 변환 |
| `attachments(volume_id=volume_id, all_projects=True)` | `Attachments.ListRecords(ctx, options...)` | `GET /attachments?volume_id=...&all_tenants=True` |
| `attachments(paginated=False)` | `WithRecordListPaginated(false)` | 첫 페이지만 |
| `attachments(max_items=100)` | `WithRecordListMaxItems(100)` | local filter 이전 raw 행 cap |

```python
reserved = conn.block_storage.create_attachment(volume_id)
attached = conn.block_storage.create_attachment(
    volume_id, instance=server_id, connector={"host": "compute-1"}, mode="rw")
for attachment in conn.block_storage.attachments(volume_id=volume_id):
    print(attachment.id, attachment.attach_mode)
```

## 독립 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`에 저장합니다. `go run . -cloud dev -volume VOLUME_ID`는 최대100행을 조회합니다. `-reserve`는 별도로 명시해야 POST하며, `-instance`를 생략하면 instance_uuid를 보내지 않습니다. 실제 Cinder API가 허용하는 권한과 connector 내용은 사용하는 서비스가 정합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    volume := flag.String("volume", "", "volume ID")
    instance := flag.String("instance", "", "optional attaching server ID")
    version := flag.String("microversion", "3.71", "Cinder microversion supported by the deployment")
    reserve := flag.Bool("reserve", false, "create a volume attachment reservation")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Attachments
    for value, err := range api.ListRecords(ctx,
        attachments.WithRecordListQuery("volume_id", *volume),
        attachments.WithRecordListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
    if *reserve {
        options := []attachments.RecordCreateOption{}
        if *instance != "" {
            options = append(options, attachments.WithRecordCreateField("instance", *instance))
        }
        value, err := api.CreateRecord(ctx, *volume, options...)
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

## 생성 속성·응답

`CreateRecord`는 native `CreateOpts.InstanceUUID`의 빈 UUID 기본값을 사용하지 않습니다. 필수 volumeID에서 volume_uuid를 만들고, instance 속성을 지정하지 않으면 instance_uuid 자체를 생략합니다. Python처럼 null·false·0·빈 문자열·빈 array/object instance는 원래 instance key로 보내며, truthy instance만 instance_uuid로 변환합니다. 응답 view의 volume_id·instance는 생성 attrs를 seed로 유지하고 실제 응답에서 온 속성을 덮어씁니다.

`WithRecordCreateField`는 Source의 id·name·status·instance·attach_mode/mode·attached_at/attach_time·detached_at/detach_time·connection_info·connector를 받습니다. alias가 있는 field 옵션은 적용 순서의 마지막 값을 사용합니다. volume_id는 positional volumeID와 충돌하므로 사전에 오류로 반환합니다. `WithRecordCreateOpts`의 raw Fields map에 같은 속성의 alias 두 개를 넣는 경우 Go map에는 순서가 없으므로 오류입니다.

Attachment는 `_allow_unknown_attrs_in_body=False`입니다. 따라서 Python처럼 unknown attrs를 버리며, 문서의 instance_id도 Body descriptor가 아니므로 무시합니다. 서버를 지정할 때는 instance를 사용합니다. known 필드의 explicit null·false·zero·빈 값과 정확한 JSON 정수를 보존합니다. native model의 required-field 검사나 string 변환을 더하지 않습니다.

Falsey mode는 Source `supports_microversion("3.54")`가 참이면 request에서 제거되지만, Resource view에는 입력값이 남습니다. `CreateRecord`는 falsey mode를 지정한 경우 공통 Cinder discovery를 자동으로 실행합니다. 서버가 **min_version ≤3.54≤ max_version**을 광고하고, configured default가 있다면 그 default의 finite major도3이며3.54 이상인 경우에만 제거합니다. 따라서 selected3.80이어도 advertised minimum3.60이거나 한쪽 bound가 없으면 `mode:false`를 유지합니다. finite-major 비교에서 global latest는3.54와 match하지 않으므로 mode를 유지하고3.latest는 match할 수 있습니다.

Capability GET은 실제 catalog v3 endpoint와 필요한 경우 같은 reverse-proxy prefix의 root를 최대 두 번 조회합니다. version header와 body를 보내지 않고 clean404/405 또는 eligible v3 row가 없는 경우에만 다음 endpoint로 이동합니다. 발견한 bounds가 없으면 capability는 false입니다. default가 없으면 발견한 범위에서 Source Attachment의 ceiling3.71로 operation version을 선택하되 원본 client를 변경하지 않습니다. 명시적3.80은3.71로 제한하지 않습니다. `WithRecordCreateModeSupported(bool)`는 자동 discovery 대신 호출자가 이미 확인한 capability를 적용하는 명시적 Go override이며 일반 호출에는 필요하지 않습니다. 그 override는 configured request version을 그대로 유지합니다. truthy mode 또는 생략한 mode는 capability GET을 추가하지 않습니다.

`AttachmentRecord.Discovery`는 실제 accepted capability 응답의 Envelope/Header/StatusCode를 독립적으로 보관합니다. capability 단계에서 Read·Close·decode·source·취소 오류가 나면 POST를 보내지 않고 Discovery 증거만 반환합니다. 이때 attachment의 StatusCode는0이고 Envelope/Resource/Wire는 없으므로 discovery200을 attachment 생성 ACK로 취급하지 않습니다. 이후 POST가 실패해도 이미 관찰한 Discovery는 남고, accepted POST의 처리 오류는 그 POST ACK와 함께 반환됩니다. negotiation 후 POST의 selected version header 또는 header 부재는 retry와 physical attempt에서도 고정합니다.


`AttachmentRecord.Resource`는 id/name/status/instance/volume_id/attached_at/detached_at/attach_mode/connection_info/connector/location의11개 declared 값을 제공합니다. 누락된 값은 null이며, location은 Connection 기반 Python location을 합성하지 않습니다. connection_info는 Source dict descriptor처럼 null은 null, nonnull nonobject는{}로 보여 줍니다. `Wire`에는 실제 응답의 unknown fields·원래 connection_info·큰 정수·문자열 timestamp를 보존합니다. `Envelope`는 전체 실제 HTTP 본문, Header/StatusCode는 실제 ACK입니다. view·Wire·Envelope·각 header는 독립 복사합니다. Wire가 없는 accepted204에서도 seed view가 실제 서버 응답값이라고 주장하지 않습니다.

Create는 singular attachment 객체 또는 flat object를 받아 seed와 합칩니다. Source `_translate_response`처럼 빈 본문·잘못된 JSON syntax는 seed를 유지합니다. 유효한 nonobject/null envelope, 잘못된 UTF-8는 실제 accepted 응답을 가진 오류입니다. 원래200..399를 허용하며 native default status 범위로 좁히지 않습니다.

## 목록·소유권·오류

`ListRecords`는 기존 native `/attachments/detail` 목록과 독립인 Source summary `/attachments`를 사용합니다. id/status/project_id/volume_id/instance_id는 서버 query이며, all_projects는 all_tenants로 변환합니다. wire alias all_tenants도 받아들입니다. 둘을 함께 넘기면 값이null이어도 all_projects가 우선하며 null은 query에서 생략합니다. Requests query encoder를 재사용해 bool을 True/False로 표현하고 iterable을 반복 query 값으로 인코딩합니다. 나머지 declared Body attribute name·instance·attached_at·detached_at·attach_mode·connection_info·connector는 local filter입니다. wire Body alias mode·attach_time과 unknown query는 Python hasattr 계약대로 버립니다.

Limit·Marker는 서버 query, MaxItems는 filter 이전에 검사한 raw 행 수입니다. 양수 MaxItems는 생략했거나0인 limit의 초기 힌트입니다. null server 값은 생략하고 dictionary local filter는 subset 비교합니다. 공통 JSON filter는 Python과 달리 bool과 number를 구분합니다. typed limit/cap은 음수를 허용하지 않으며 marker/limit은 단일 값이어야 합니다.

Iterator는 lazy이고 소비자가break하면 후속 요청을 하지 않습니다. plural attachments는 array 또는 singleton object를 받으며 null/scalar 행은 거부합니다. attachments_links·top-level next·HTTP Link를 따라가고, 양수 initial limit의 short page도 wire id를 사용해 다음 marker를 요청합니다. empty page·local cap·single page는 광고된 next를 평가하기 전에 종료합니다. 반환 view의 id를 바꿔도 marker는 원래 page에서 선택합니다. 같은 origin과 collection을 유지하고 filter 변경·추가·반복 cursor를 차단하는 기존 REST guard를 사용합니다. dictionary links는 Source paginator보다 넓은 Go 호환 기능입니다.

공통 cloudread/REST Source capture·microversions discovery/Supports와 option snapshot을 재사용합니다. per-request list microversion·header는 복사한 client에만 적용하고 원본은 바꾸지 않습니다. 소스 route/provider/type/version 변경, 취소의 custom cause, accepted body Read/Close 오류, 거부 HTTP body의 Read/Close 오류, transport·retry callback 오류를 원래 HTTP ACK/상태/본문/헤더와 함께 유지합니다. 거부 상태400..599를 shared rejected-body observer로 관찰하며 깨진 응답을 성공으로 허용하지 않습니다. 입력·옵션·Header 오류는 HTTP 전에 실패합니다.

Python mutable Resource/Munch 인자·동일 객체 dirty/cache/commit/session hooks·base_path retargeting 및 capability 결정이 필요 없는 다른 Resource/list 호출의 class/session microversion negotiation은 공유 SDK 범위로 별도 추적합니다. 이 API의 volumeID는 유효한 nonblank UTF-8 string을 명시적으로 받습니다. 기존 generated Create/Get/List/Update/Complete와 native Resources는 별도 API 계약으로 유지합니다.

[계약 테스트](attachments/python_records_contracts_test.go)는 기존 public Gophercloud testhelper 기반 attachment fixture를 재사용합니다. Reservation omission, JSON presence/alias/descriptor, microversion3.0~3.80 및 advertised range/default를 분리한 mode capability, versionless discovery·finite fallback·source 원복과 오류 증거, Source summary/query/pagination/raw cap, 실제 부분 ACK/거부 body Read/Close/callback/cancellation/source 오류와 옵션 동시 재사용을 검사합니다. 실제 Cinder 배포의 권한·서비스 connector·endpoint discovery 확인은 로컬 fixture 검증과 별개입니다.
