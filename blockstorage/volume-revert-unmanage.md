# Volume revert·unmanage

`RevertVolumeToSnapshot`과 `UnmanageVolume`은 현재 volume ID를 받아 Cinder v3 action 하나를 호출합니다. 옵션이나 caller builder가 없습니다. package·Connection·service 경로가 같은 `*VolumeActionResult`를 반환합니다.

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 실제 [revert Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1220-L1239), [unmanage Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1293-L1303), [Volume.revert_to_snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L226-L232), [Volume.unmanage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L311-L315)입니다.

| 실제 Python Proxy | Go Connection / package | action body |
|---|---|---|
| `conn.block_storage.revert_volume_to_snapshot(volume_id, snapshot_id)` | `RevertVolumeToSnapshot(ctx, request, snapshotID)` | `{"revert":{"snapshot_id":snapshotID}}` |
| `conn.block_storage.unmanage_volume(volume_id)` | `UnmanageVolume(ctx, request)` | `{"os-unmanage":null}` |

package 함수는 `ctx` 다음에 선택한 `*gophercloud.ServiceClient`를 받습니다. `conn.BlockStorageV3(ctx)`의 `Volumes`는 API 필드이며 같은 이름의 service method는 request 대신 `volumeID string`을 받습니다.

## 한 action 실행

다음 프로그램은 clouds.yaml의 `CLOUD_NAME`, 현재 `VOLUME_ID`, `VOLUME_LIFECYCLE_ACTION=revert|unmanage`를 요구합니다. `VOLUME_LIFECYCLE_PATH=connection|direct|service`의 기본값은 connection입니다. 한 실행에서 action 하나를 전송합니다. 30초 parent context는 인증·discovery·action을 포함하는 application 제한입니다.

revert에서는 `SNAPSHOT_ID` 환경변수의 존재가 필수입니다. 예제는 빈 값도 허용하여 SDK의 literal body 입력과 동일하게 전달합니다. 입력 문자열을 실제 snapshot으로 받아들일지, 현재 volume 상태와 snapshot 관계가 적절한지는 Cinder가 판단합니다.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
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
    if cloudName == "" || volumeID == "" {
        return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
    }
    action := os.Getenv("VOLUME_LIFECYCLE_ACTION")
    if action != "revert" && action != "unmanage" {
        return fmt.Errorf("VOLUME_LIFECYCLE_ACTION must be revert or unmanage")
    }
    path := os.Getenv("VOLUME_LIFECYCLE_PATH")
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown VOLUME_LIFECYCLE_PATH %q", path)
    }
    var snapshotID string
    if action == "revert" {
        value, supplied := os.LookupEnv("SNAPSHOT_ID")
        if !supplied { return fmt.Errorf("SNAPSHOT_ID is required; explicit empty is allowed") }
        snapshotID = value
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, path, action, input, snapshotID)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
    if path == "connection" {
        if action == "revert" { return conn.RevertVolumeToSnapshot(ctx, input, snapshotID) }
        return conn.UnmanageVolume(ctx, input)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        if action == "revert" { return blockstorage.RevertVolumeToSnapshot(ctx, cinder.RawClient(), input, snapshotID) }
        return blockstorage.UnmanageVolume(ctx, cinder.RawClient(), input)
    }
    if action == "revert" { return cinder.Volumes.RevertVolumeToSnapshot(ctx, input.VolumeID, snapshotID) }
    return cinder.Volumes.UnmanageVolume(ctx, input.VolumeID)
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
        if errors.As(err, &proof) {
            fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body))
        }
    }
}
```

Python 호출은 다음과 같습니다. 각 줄은 독립 요청입니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.revert_volume_to_snapshot(volume_id, snapshot_id)
conn.block_storage.unmanage_volume(volume_id)
```

## 입력과 microversion

두 helper는 volume 또는 snapshot 이름을 찾거나 member GET을 수행하지 않습니다. 현재 `VolumeID`는 안전한 UTF-8 단일 route segment로 검증합니다. revert의 `snapshotID`는 필수 UTF-8 body string이므로 빈 문자열·control 문자·path처럼 보이는 문자열도 그대로 JSON에 넣습니다. 이 값에 현재 volume route의 제한을 적용하거나 generated ID를 만들지 않습니다.

revert는 매 호출에서 **required3.40** support 검사를 수행합니다. selected microversion이 있어도 서버 advertisement의 min/max 둘 다가 required3.40을 포함해야 합니다. selected도 같은 major의3.40 이상이어야 하므로 selected3.39 또는 global `latest`는 gate를 통과하지 못합니다. `3.latest`는 same-major 비교를 통과할 수 있지만, 원래 header literal을 서버가 받아 준다는 보장은 아닙니다. 서버 range 검사는 required3.40에 대한 검사이므로 advertised max3.50과 selected3.90도 이 gate를 통과하면 selected literal3.90을 action에 사용합니다.

selected가 없으면 같은 operation-owned advertisement로 Volume cap3.71과 server maximum/optionalminimum을 적용합니다. gate 실패·bounds 부재·malformed discovery·IO/source/context 오류는 action POST 전에 끝납니다. 이미 admitted된 discovery 응답은 `Discovery`에 남습니다. advertised bounds 누락·지원범위 불충족·selected 호환성 실패에는 `resource.ErrUnsupported`가 포함됩니다. 이 minimum gate는 detached/available/latest-snapshot 조건을 확인하는 기능이 아닙니다. Python도 그 조건을 문서에 설명하지만 상태·attachment·snapshot GET으로 검증하지 않습니다.

package·service 경로는 context/provider/selected Source와 현재 route ID를 먼저 검증하고, revert의 snapshot UTF-8를 검사합니다. Connection은 context/provider 선행 검증 후 snapshot UTF-8, 현재 route ID를 검사한 다음 cached Cinder를 선택합니다. 두 helper에는 original option callback이나 option default가 없습니다. 입력 오류는 HTTP 전에 끝나며 실제 요청 전 Source/header policy를 고정합니다.

unmanage는 minimum gate가 없습니다. 비어 있지 않은 selected microversion은 그대로 사용하고, 없으면 유한 discovery로 cap3.71을 적용합니다. source client의 microversion은 수정하지 않습니다. 두 helper는 CurrentLocation·Resource 모델·wait·poll·cleanup을 호출하지 않습니다. Connection은 cached Cinder v3를 선택합니다.

## 응답과 Go 경계

`Discovery`와 `Applied`는 각 단계에서 admitted된 실제 Body/Header/StatusCode를 독립적으로 소유합니다. action 응답은 opaque이므로 JSON을 파싱하지 않으며 empty/malformed/invalid-UTF8 body도 보존합니다. 원래 policy의 HTTP100..399와 read/close·source 검증이 성공하면 `Completed=true`입니다. 이는 action acknowledgement이며 revert 상태 완료, Cinder에서 management 제거 확인 또는 backend 데이터 삭제 증명이 아닙니다. unmanage의 목적은 backend storage object를 지우지 않고 Cinder 관리에서 제거하는 것입니다.

accepted Read/Close·source·context 오류는 `Applied`가 있는 partial result와 error를 함께 반환할 수 있고 `Completed=false`를 유지합니다. original policy가 거부한 HTTP 응답은 native error로 남고 `Applied`를 만들지 않습니다. discovery proof를 뒤의 rejected action 오류의 현재 proof로 빌리지 않습니다. accepted-fault 응답을 SDK가 자동으로 replay하거나 undo하지 않습니다. native가 discarded한 rejected POST body IO를 모두 복원한다고 주장하지 않습니다.

고정 method·route·serialized body·framing·선택 version은 retry/redirect에서 검사하고, live provider authentication은 사용할 수 있습니다. Go는 typed current volume ID와 literal snapshot string을 사용하며 Python supplied Resource/dict/Munch·untyped inherited ID·constructor/descriptor/location/cache side effect를 재현하지 않습니다. Source의 `_get_resource`는 local construction/reuse이고 name finder가 아닙니다.

Source는 action 응답을 버리고 None을 반환합니다. Go는 partial proof와 acknowledgement를 노출합니다. Source의 require stage와 ordinary selection은 endpoint-data getter를 각각 호출할 수 있지만 physical GET 횟수는 Adapter cache가 결정합니다. Go는 유한한 call-owned discovery를 공유하며 Keystoneauth의 전체 cache/link/envelope/canonicalization·예외 순서·HTTP taxonomy를 동일하게 주장하지 않습니다. Python explicit selected default=""와 None을 Go ServiceClient의 빈 문자열 하나로 구분할 수 없고, Go empty는 미선택 policy입니다. 고정 [Keystoneauth5.16.0 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)의 same-major 조건은 Source 정적 비교입니다. selected literal forwarding과 Python Session의 실제 header canonicalization은 별도 경계입니다.

기존 native `cinder.Volumes.Unmanage(ctx, id)`는 `{"os-unmanage":{}}`, HTTP202, error-only result인 독립 API입니다. 새 `UnmanageVolume`의 null body와 opaque100..399 result를 위해 native API를 바꾸지 않습니다. 새 두 helper만으로 native/v2/Resource 또는 manage/list/metadata 선언의 지원 상태가 바뀌지 않습니다. Go 예제 컴파일·로컬 HTTP 검증과 Source 정적 비교는 authenticated OpenStack/Python runtime 또는 전체 SDK 완성을 뜻하지 않습니다.

[다른 볼륨 action](volume-actions.md), [status·migration 작업](volume-migration-reset.md), [service API](v3/volumes/README.md)를 함께 참고하세요.
