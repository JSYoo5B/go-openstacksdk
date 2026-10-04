# Volume state actions

`ReserveVolume`, `UnreserveVolume`, `BeginVolumeDetaching`, `AbortVolumeDetaching`는 명시적인 volume ID로 Cinder v3 action을 호출합니다. 이름 조회·Resource 변환·CurrentLocation·완료 대기를 수행하지 않습니다. 각 호출은 독립 요청이며 네 함수를 자동으로 실행하는 workflow나 rollback은 아닙니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 실제 [v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1422)와 [Volume methods](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L405)입니다. 이 네 action에는 별도 옵션·Prepare·builder가 필요하지 않습니다.

| Python Proxy | Go Connection / package | 정확한 action body |
|---|---|---|
| `conn.block_storage.reserve_volume(id)` | `ReserveVolume(ctx, VolumeActionRequest{VolumeID: id})` | `{"os-reserve":null}` |
| `conn.block_storage.unreserve_volume(id)` | `UnreserveVolume(ctx, VolumeActionRequest{VolumeID: id})` | `{"os-unreserve":null}` |
| `conn.block_storage.begin_volume_detaching(id)` | `BeginVolumeDetaching(ctx, VolumeActionRequest{VolumeID: id})` | `{"os-begin_detaching":null}` |
| `conn.block_storage.abort_volume_detaching(id)` | `AbortVolumeDetaching(ctx, VolumeActionRequest{VolumeID: id})` | `{"os-roll_detaching":null}` |

package 함수는 `ctx, cinder.RawClient(), request`를 받습니다. v3 service의 `cinder.Volumes`는 API **필드**이며 같은 이름의 method가 `ctx, volumeID string`을 받습니다. 세 경로 모두 같은 `VolumeActionResult`를 반환합니다.

## 한 action 실행

다음 프로그램은 clouds.yaml의 `CLOUD_NAME`과 실제 `VOLUME_ID`를 사용합니다. `STATE_ACTION`은 `reserve`·`unreserve`·`begin-detaching`·`abort-detaching` 중 하나이며 기본값은 `reserve`입니다. `STATE_ACTION_PATH`는 `connection`·`direct`·`service`이고 기본값은 `connection`입니다. 한 실행에서 선택한 action만 한 번 호출합니다. parent context의30초는 예제 application이 authentication·discovery·action 전체에 적용하는 제한이며 SDK wait timeout이 아닙니다.

예를 들어 `CLOUD_NAME=devstack VOLUME_ID=actual-volume-id STATE_ACTION=reserve STATE_ACTION_PATH=connection go run ./volume-action-example`로 실행합니다. 아래 프로그램은 SDK를 사용하는 모듈의 별도 example 디렉터리에 저장합니다. 명시적인 기존 microversion 정책은 Connection 구성에서 지정하며, 이 예제는 선택 version이 없으면 action이 자신의 정책으로 discovery하도록 둡니다.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "gophercloudsdk"
    "gophercloudsdk/blockstorage"
    "gophercloudsdk/resource"
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
    action := os.Getenv("STATE_ACTION")
    if action == "" {
        action = "reserve"
    }
    switch action {
    case "reserve", "unreserve", "begin-detaching", "abort-detaching":
    default:
        return fmt.Errorf("unknown STATE_ACTION %q", action)
    }
    path := os.Getenv("STATE_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    switch path {
    case "connection", "direct", "service":
    default:
        return fmt.Errorf("unknown STATE_ACTION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
    if path == "connection" {
        calls := map[string]func(context.Context, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error){
            "reserve": conn.ReserveVolume,
            "unreserve": conn.UnreserveVolume,
            "begin-detaching": conn.BeginVolumeDetaching,
            "abort-detaching": conn.AbortVolumeDetaching,
        }
        return calls[action](ctx, input)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        calls := map[string]func(context.Context, *gophercloud.ServiceClient, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error){
            "reserve": blockstorage.ReserveVolume,
            "unreserve": blockstorage.UnreserveVolume,
            "begin-detaching": blockstorage.BeginVolumeDetaching,
            "abort-detaching": blockstorage.AbortVolumeDetaching,
        }
        return calls[action](ctx, cinder.RawClient(), input)
    }
    calls := map[string]func(context.Context, string) (*blockstorage.VolumeActionResult, error){
        "reserve": cinder.Volumes.ReserveVolume,
        "unreserve": cinder.Volumes.UnreserveVolume,
        "begin-detaching": cinder.Volumes.BeginVolumeDetaching,
        "abort-detaching": cinder.Volumes.AbortVolumeDetaching,
    }
    return calls[action](ctx, input.VolumeID)
}

func inspect(result *blockstorage.VolumeActionResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for i, page := range result.Discovery {
            if page != nil {
                fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
            }
        }
        if result.Applied != nil {
            fmt.Println("admitted action response:", result.Applied.StatusCode)
            fmt.Printf("opaque action body: %q\n", result.Applied.Body)
        }
        fmt.Println("helper acknowledgement completed:", result.Completed)
    }
    if err != nil {
        fmt.Println("action error:", err)
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
        }
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted the call")
        }
    }
}
```

오류가 있어도 이미 받아들인 `Discovery`·`Applied`를 확인할 수 있습니다. 예제의 action 선택 map은 application의 command dispatch이며 SDK가 요구하는 builder interface가 아닙니다. `Completed`를 현재 서버 상태의 증거로 사용하지 않습니다.

Python에서는 실제 Proxy를 다음처럼 호출합니다. 아래 네 줄도 각각 별도의 action입니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.reserve_volume(volume_id)
conn.block_storage.unreserve_volume(volume_id)
conn.block_storage.begin_volume_detaching(volume_id)
conn.block_storage.abort_volume_detaching(volume_id)
```

## ID·route와 기본 정책

Go는 `VolumeID`가 비어 있지 않은 UTF-8의 단일 unescaped URL path segment인지 검사합니다. slash·backslash·query·fragment·percent escape·공백·control·`.`·`..`을 허용하지 않습니다. UUID 형식만 요구하거나 ID가 실제로 존재하는지 먼저 조회하지 않습니다. 같은 문자열을 이름으로 재해석하지 않고 고정 POST `volumes/{id}/action`에 사용하며 서버가 존재 여부와 허용 상태를 판정합니다.

Connection은 입력/context/provider를 검사한 뒤 cached Cinder v3만 선택합니다. 이 helper는 Nova·Identity lookup·현재 location snapshot을 소비하지 않습니다. service nil과 package nil client도 library error로 전달하며 원래 action 이름과 `volumes` resource label을 유지합니다.

[Source `_get_resource`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563)는 ID 또는 supplied Volume Resource에서 객체를 만들거나 갱신합니다. 이름 검색은 하지 않지만 constructor·computed location·to_dict·descriptor 변환과 mutable Resource identity 효과가 생길 수 있습니다. Go는 명시적인 ID-only domain과 safe route로 이를 대체하며 supplied mutable Resource/dict 및 Source의 custom urljoin coercion을 복제하지 않습니다. 이 차이가 다른 Resource 선언의 지원 완료를 뜻하지 않습니다.

## Action microversion

[Volume의 class maximum은3.71](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L146)입니다. 이미 선택된 Cinder client microversion이 nonempty이면 그 문자열을 그대로 사용하고 discovery를 생략합니다. version을3.71로 강제하거나 client의 원본 설정을 변경하지 않습니다. selected literal 정책은 Source session default 우선 선택에 대응하지만 keystoneauth Session의 후속 문자열 canonicalization까지 재현하지 않습니다.

선택 version이 없으면 captured endpoint의 유한한 후보에 bodyless GET discovery를 수행합니다. catalog의 끝 project/version만 제거하므로 encoded reverse-proxy prefix를 보존합니다. accepted200·300의 flat object, `version`, `versions` 또는 `versions.values`에서 처음 사용할 수 있는 v3 행을 선택합니다. status는 empty 또는 CURRENT·SUPPORTED·STABLE·DEPRECATED이며 대소문자를 구분하지 않습니다. accepted `{}`·usable v3 없는 페이지는 다음 후보로 넘어갑니다. usable 행의 `max_version`이 비어 있으면 `version`을 사용하고, 그 maximum도 없으면 즉시 version을 생략합니다.

maximum이 있으면 서버 maximum을 먼저 파싱하고 optional minimum을 파싱합니다. [Source utility](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L358)처럼 ceiling3.71과 server maximum 중 작은 tuple을 사용하며 minimum>3.71이면 version을 생략합니다. missing maximum에서는 unused minimum을 파싱하지 않습니다. tuple의 길이를 보존하며 max/min major3 제한이나 min≤max 조건을 추가하지 않습니다. finite major 아래 `latest`가 남아 header를 만들 수 없으면 action POST 전에 오류가 납니다. `Microversion`의 empty string은 version 생략이며, 이 필드는 action POST가 이루어졌다는 flag가 아닙니다.

clean native404·405만 다음 후보의 fallback이 됩니다. Read/Close·callback·source·context 오류가 join되거나 native OkCodes를 확장해 original policy 밖 응답을 받아들인 경우는 clean fallback이 아닙니다. accepted parse/IO/source/context 오류와 그 밖의 native failure는 terminal이며 action을 보내지 않습니다. malformed accepted discovery JSON의 terminal proof 정책, 유한한 후보·허용 JSON shapes/status·typed advertisement·clean404/405 taxonomy는 명시적인 Go mapping입니다. 실제 keystoneauth discovery/cache/DiscoveryFailure와 utility 호출 전 endpoint-data의 min/max 예외 순서 전체가 동일하다고 주장하지 않습니다.

## Opaque acknowledgement와 오류 증거

`VolumeActionResult`는 `VolumeID`, 호출에 선택된 `Microversion`, accepted discovery 응답들의 `Discovery`, accepted action 응답의 `Applied`, `Completed`를 제공합니다. `VolumeActionPage`는 실제 body bytes·header·status를 보존합니다. `Discovery`와 `Applied`는 다른 HTTP 단계의 증거이며 rejected discovery404/405를 accepted proof로 채우지 않습니다.

action은 Source의 status<400 영역을 finite HTTP100..399로 옮깁니다. JSON·empty·malformed·invalid-UTF8 action body도 opaque bytes로 유지하며 response 모델이나 status field를 추출하지 않습니다. `Completed=true`는 accepted action의 Read/Close와 source/context 검사가 성공한 helper acknowledgement입니다. Volume.status가 reserved/detaching/in-use인지 확인하거나 local Volume 상태를 바꾸지 않습니다. 대기·poll·후속 GET·자동 undo는 없습니다.

accepted Read/Close·source drift·context 실패에서는 현재 `Applied` 및 앞선 `Discovery`를 유지하고 `Completed=false`로 반환합니다. action에 도달하기 전 discovery 실패에는 `Applied`가 없습니다. native>=400 rejection과 caller가 확장한 native OkCodes의 original-policy rejection은 success proof가 아니므로 `Applied`를 만들지 않습니다. 에러는 원래 native status/body/header 또는 현재 admitted ResponseError와 원인을 유지합니다. native가 버리는 모든 rejected-body Read/Close 오류까지 전부 보존한다는 의미는 아닙니다.

선택 source의 ProviderClient·Endpoint·ResourceBase·Type·Microversion 변경은 terminal이며 captured ordinary headers와 action method/route/serialized null body·framing·operation version header 정책은 physical request와 native retry에서도 지킵니다. authentication token은 live provider 정책을 사용합니다. 다른 target으로 redirect하거나 body/header/source를 바꿔 재전송할 수 없습니다. 원본 client 설정은 호출의 negotiated version으로 덮어쓰지 않습니다.

## 기존 API와 이번 범위

Gophercloud v2.15.0의 [native Reserve/Unreserve/BeginDetaching](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go#L363)는 `{}` action value를 사용하고 Reserve/Unreserve의200·201·202, BeginDetaching의202만 success로 받습니다. 기존 service `Reserve`·`Unreserve`·`BeginDetaching`은 그 계약을 유지하며 새 `ReserveVolume` 등의 `null` body·sub400 opaque result와 구별됩니다. Source abort action은 이름과 달리 정확한 `os-roll_detaching:null`을 전송합니다.

기존 [cloud bootable helper](volume-mutations.md)의 `Connection.SetVolumeBootable`은 NameOrID 조회와 defaulttrue를 제공하는 별도 기능입니다. 명시적 ID의 [flag action 가이드](volume-flags.md)는 `SetVolumeBootableStatus`의 required bool과 `SetVolumeReadonly`의 defaulttrue·명시적 false를 설명합니다. Source Proxy bootable은 required이고 Resource method만 defaulttrue이며, Proxy readonly만 defaulttrue이고 Resource method에서는 required입니다. native AttachMode.ReadOnly는 readonly flag setter가 아닙니다. 두 flag action은 이 문서의 네 state action과 별도 호출입니다.

이 문서는 네 실제 v3 Proxy action의 Go 사용과 차이를 설명합니다. mutable Resource/session/cache identity, v2/native leaf 및 그 밖의 volume action 선언을 자동으로 승격하거나 authenticated OpenStack/Python 실행·전체 SDK 완료를 주장하지 않습니다. Source 검토, Go 문서 컴파일, 실제 cloud 실행은 서로 다른 검증입니다.
