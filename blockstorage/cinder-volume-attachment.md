# Cinder 직접 volume attach·detach

`AttachCinderVolume`과 `DetachCinderVolume`은 명시적인 현재 volume ID를 받아 Cinder v3 action을 호출합니다. SDK가 옵션 기본값·복사·microversion·응답 proof를 처리하며 caller builder는 필요하지 않습니다. 기존 `AttachVolume`·`DetachVolume`은 서버와 볼륨을 연결하는 Nova+Cinder workflow입니다.

기준 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy attach/detach](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1241-L1280)와 [Volume.attach/detach](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L234-L271)입니다.

| 사용할 작업 | 실제 Python | Go Connection / package | HTTP·대기 |
|---|---|---|---|
| 직접 Cinder attach | `conn.block_storage.attach_volume(volume, mountpoint, instance=...)` | `AttachCinderVolume(ctx, request, mountpoint, options...)` | Cinder `os-attach` POST; wait 없음 |
| 직접 Cinder detach | `conn.block_storage.detach_volume(volume, attachment, force=False)` | `DetachCinderVolume(ctx, request, attachmentID, options...)` | Cinder `os-detach` / `os-force_detach` POST; wait 없음 |
| 서버에 볼륨 연결 workflow | `conn.attach_volume(server, volume, wait=True)` | `AttachVolume(ctx, AttachVolumeRequest, options...)` | Cinder 상태 확인→Nova POST→기본 Cinder 대기 |
| 서버에서 볼륨 분리 workflow | `conn.detach_volume(server, volume, wait=True)` | `DetachVolume(ctx, DetachVolumeRequest, options...)` | Nova DELETE→기본 Cinder 대기 |

직접 helper의 `VolumeActionRequest{VolumeID: ...}`는 현재 Cinder volume ID입니다. detach의 추가 인자는 body의 attachment ID이며 기존 Nova workflow의 server/volume ref와 다릅니다. package 함수는 `ctx` 다음에 선택한 `*gophercloud.ServiceClient`를 받고, `conn.BlockStorageV3(ctx)`의 `Volumes` **API 필드**는 request 대신 `volumeID string`을 받습니다. 세 호출 경로의 결과는 `*VolumeActionResult`입니다.

## 한 action 실행

다음 독립 프로그램은 clouds.yaml의 `CLOUD_NAME`, 현재 `VOLUME_ID`, `CINDER_ATTACHMENT_ACTION=attach|detach`를 요구합니다. `CINDER_ATTACHMENT_PATH=connection|direct|service`는 기본 connection이며 한 실행에서 action 하나만 수행합니다. 30초 parent context는 인증·discovery·action을 포함하는 application 제한입니다.

attach의 `CINDER_MOUNTPOINT`, detach의 `CINDER_ATTACHMENT_ID`는 환경변수의 존재가 필수이며 명시한 빈 문자열도 전달합니다. attach selector는 `CINDER_INSTANCE`와 `CINDER_HOST_NAME`의 생략/빈 값을 구분합니다. `CINDER_FORCE` 기본값은 false입니다. force=true일 때만 예제가 `CINDER_CONNECTOR_JSON` object를 읽습니다. bool·JSON 파싱은 예제 application의 입력 처리입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "strconv"
    "time"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
    if cloudName == "" || volumeID == "" { return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required") }
    action, path := os.Getenv("CINDER_ATTACHMENT_ACTION"), os.Getenv("CINDER_ATTACHMENT_PATH")
    if action != "attach" && action != "detach" { return fmt.Errorf("CINDER_ATTACHMENT_ACTION must be attach or detach") }
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" { return fmt.Errorf("unknown CINDER_ATTACHMENT_PATH %q", path) }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var bodyValue string
    var attach blockstorage.CinderVolumeAttachOpts
    var detach blockstorage.CinderVolumeDetachOpts
    if action == "attach" {
        value, supplied := os.LookupEnv("CINDER_MOUNTPOINT")
        if !supplied { return fmt.Errorf("CINDER_MOUNTPOINT is required; explicit empty is allowed") }
        bodyValue = value
        var options []blockstorage.CinderVolumeAttachOption
        if value, supplied := os.LookupEnv("CINDER_INSTANCE"); supplied {
            options = append(options, blockstorage.WithCinderVolumeAttachInstance(value))
        }
        if value, supplied := os.LookupEnv("CINDER_HOST_NAME"); supplied {
            options = append(options, blockstorage.WithCinderVolumeAttachHostName(value))
        }
        prepared, err := blockstorage.PrepareCinderVolumeAttachOptions(ctx, options...)
        if err != nil { return err }
        attach = prepared
    } else {
        value, supplied := os.LookupEnv("CINDER_ATTACHMENT_ID")
        if !supplied { return fmt.Errorf("CINDER_ATTACHMENT_ID is required; explicit empty is allowed") }
        bodyValue = value
        force := false
        if text, supplied := os.LookupEnv("CINDER_FORCE"); supplied {
            value, err := strconv.ParseBool(text)
            if err != nil { return fmt.Errorf("CINDER_FORCE: %w", err) }
            force = value
        }
        options := []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachForce(force)}
        if text, supplied := os.LookupEnv("CINDER_CONNECTOR_JSON"); force && supplied {
            var connector map[string]json.RawMessage
            if err := json.Unmarshal([]byte(text), &connector); err != nil { return fmt.Errorf("CINDER_CONNECTOR_JSON: %w", err) }
            options = append(options, blockstorage.WithCinderVolumeDetachConnector(connector))
        }
        prepared, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx, options...)
        if err != nil { return err }
        detach = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    result, err := apply(ctx, conn, path, action, blockstorage.VolumeActionRequest{VolumeID: volumeID}, bodyValue, attach, detach)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, bodyValue string, attach blockstorage.CinderVolumeAttachOpts, detach blockstorage.CinderVolumeDetachOpts) (*blockstorage.VolumeActionResult, error) {
    a := blockstorage.WithCinderVolumeAttachOptions(attach)
    d := blockstorage.WithCinderVolumeDetachOptions(detach)
    if path == "connection" {
        if action == "attach" { return conn.AttachCinderVolume(ctx, input, bodyValue, a) }
        return conn.DetachCinderVolume(ctx, input, bodyValue, d)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        if action == "attach" { return blockstorage.AttachCinderVolume(ctx, cinder.RawClient(), input, bodyValue, a) }
        return blockstorage.DetachCinderVolume(ctx, cinder.RawClient(), input, bodyValue, d)
    }
    if action == "attach" { return cinder.Volumes.AttachCinderVolume(ctx, input.VolumeID, bodyValue, a) }
    return cinder.Volumes.DetachCinderVolume(ctx, input.VolumeID, bodyValue, d)
}

func inspect(result *blockstorage.VolumeActionResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for index, page := range result.Discovery {
            if page != nil { fmt.Println("admitted discovery response:", index, page.StatusCode, len(page.Body)) }
        }
        if result.Applied != nil {
            fmt.Println("admitted action response:", result.Applied.StatusCode)
            fmt.Printf("opaque body: %q\n", result.Applied.Body)
        }
        fmt.Println("helper acknowledgement completed:", result.Completed)
    }
    if err != nil {
        var proof *resource.ResponseError
        if errors.As(err, &proof) { fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body)) }
    }
}
```

일반 호출에서는 `conn.AttachCinderVolume(ctx, request, "/dev/vdb", blockstorage.WithCinderVolumeAttachInstance(serverID))`처럼 factory를 바로 전달하면 충분합니다. Prepare는 HTTP 없이 original 옵션을 한 번 실행하고 owned policy를 준비합니다. attach Prepare는 선택한 selector의 존재·UTF-8를, detach Prepare는 실제 force branch의 Connector를 검사합니다. 오류에는 zero policy를 반환합니다. 위 예제는 그 policy를 전체 교체 factory로 재사용합니다. service 패키지 `blockstorage/v3/volumes`에도 같은 opts/option alias·factory·Prepare 이름이 있습니다.

Python에서는 직접 Proxy를 다음처럼 사용합니다. 각 줄은 별도 요청입니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.attach_volume(volume_id, "/dev/vdb", instance=server_id)
conn.block_storage.attach_volume(volume_id, "", instance="", host_name="ignored")
conn.block_storage.detach_volume(volume_id, attachment_id)
conn.block_storage.detach_volume(volume_id, attachment_id, force=True,
                                 connector={"host": "", "multipath": False, "extra": None})
```

## 옵션 기본값과 소유권

attach의 `Instance`·`HostName`은 nullable string입니다. `Instance!=nil`이면 명시한 빈 값도 `instance_uuid`로 보내며 HostName을 소비하지 않습니다. Instance가 nil일 때만 nonnil HostName을 `host_name`으로 보냅니다. 둘 다 nil이면 action 전에 local error가 납니다. 선택한 selector만 UTF-8를 검사하므로 비활성 HostName의 잘못된 UTF-8가 선택한 Instance 요청을 막지 않습니다. mountpoint는 항상 literal UTF-8 body string이며 빈 값·control·path-like 문자열도 보냅니다. 이 직접 Source API에는 Mode 옵션이 없습니다.

detach의 `Force`는 nil/default=false입니다. false이면 `os-detach`에 `attachment_id`만 보내고 Connector를 검증하거나 소비하지 않습니다. 따라서 unused Connector의 잘못된 raw JSON도 normal detach를 막지 않습니다. true이면 `os-force_detach`를 사용합니다. Connector nil 또는 빈 map은 생략하며, nonempty map은 member 값이 `null`·false·0·빈 string·빈 배열·빈 object여도 포함합니다. 실제로 전송하는 connector의 UTF-8 key와 complete JSON raw value만 검사합니다. attachmentID는 빈 값을 포함하여 항상 literal UTF-8 body string으로 보냅니다.

전체 교체 factory는 이전 selector/map/force를 누적하지 않고 완전한 policy로 교체합니다. nil Force는 다시 기본 false가 됩니다. factories와 callback 경계는 pointer·Connector map·각 RawMessage bytes를 복사합니다. SDK는 original 옵션 slice를 실행 전에 복사하고 각 callback 전후에 policy를 복사하므로 caller가 factory 입력이나 retained config를 나중에 변경해도 캡처한 요청 policy가 바뀌지 않습니다. original callback은 native retry마다 다시 실행하지 않습니다.

## 전송 결과와 경계

package·service는 selected Source와 현재 route ID, required mountpoint/attachment UTF-8를 먼저 검사한 다음 original options를 실행합니다. Connection은 context/provider 선행 검사 후 originals를 Prepare로 한 번 실행하고 required body string, 현재 route ID를 검사하여 cached Cinder를 선택합니다. Connection이 internal engine에 전달하는 완전한 owned policy는 original callback을 다시 호출하지 않습니다.

두 action에는 minimum microversion gate가 없습니다. 비어 있지 않은 selected microversion은 그대로 사용하고, 없으면 유한 discovery로 Volume cap3.71과 server maximum/optionalminimum을 적용합니다. source client/cache의 version은 변경하지 않습니다. 직접 helper는 Nova·Identity·member GET·name finder·CurrentLocation·Resource 모델 변환·상태/attachment 확인·wait·poll을 호출하지 않습니다. Connection이 cached Cinder v3와 옵션 준비를 소유합니다.

`Discovery`와 `Applied`는 각 단계에서 admitted된 실제 Body/Header/StatusCode를 독립적으로 소유합니다. action 응답은 opaque이므로 empty/malformed/invalid-UTF8 body도 원문을 보존합니다. 원래 policy의 HTTP100..399와 read/close·source 검증이 성공하면 `Completed=true`입니다. 이는 helper acknowledgement이며 attachment 생성/분리 완료, device 연결 상태나 backend readiness 확인이 아닙니다.

accepted Read/Close·source·custom context 오류에는 partial `Applied`가 남고 Completed는 false일 수 있습니다. original policy가 거부한 HTTP 응답은 native error이며 Applied를 만들지 않습니다. 이전 discovery proof를 뒤의 rejected action 오류의 현재 proof로 빌리지 않습니다. SDK는 accepted-fault action을 자동 replay하거나 cleanup/rollback하지 않습니다. force detach 입력의 Source 설명은 실패한 detach를 되돌리는 Cinder action 의미이며 SDK가 backend disconnect나 별도 rollback을 실행한다는 뜻이 아닙니다.

Go는 안전한 현재 volume route ID, typed nullable selector/bool 및 owned `map[string]json.RawMessage`를 사용합니다. Python의 arbitrary truthy force·non-JSON connector·Resource/dict/Munch·None·untyped inherited ID·constructor/descriptor/location/cache side effect를 재현하지 않습니다. Connector raw JSON 값의 숫자 precision/표현과 JSON serialization은 Python의 동적 connector 값·JSON 직렬화와 다른 Go 경계입니다. 응답 JSON은 두 helper 모두 파싱하지 않습니다. fixed method/URL/serialized body/framing/version을 retry/redirect에서 검사하고 live provider authentication은 사용할 수 있습니다. finite discovery·Keystoneauth cache/canonicalization/exception ordering·native rejected POST body IO 보존은 전체 동등성을 주장하지 않습니다.

기존 native `cinder.Volumes.Attach(ctx, id, AttachOpts, ...)`·`Detach(ctx, id, DetachOpts, ...)`는 omitempty string과 HTTP202/error-only API입니다. native attach의 Mode, 두 nonempty selector 동시 전송, empty mountpoint/attachment 생략 및 normal-only detach 계약을 새 helper로 바꾸지 않습니다. 새 두 직접 Proxy 선언의 구현은 native/v2/Resource 또는 기존 cloud workflow의 추가 지원 승격 근거가 아닙니다. Source 정적 비교·Go 예제 컴파일·로컬 HTTP 검증은 인증된 OpenStack/Python runtime이나 전체 SDK 완성을 뜻하지 않습니다.

서버에 실제 볼륨을 연결하는 workflow는 [AttachVolume](attach-volume.md)와 [DetachVolume](detach-volume.md)를 참고하세요. [다른 직접 action](volume-actions.md), [service API](v3/volumes/README.md)도 별도 사용법을 설명합니다.
