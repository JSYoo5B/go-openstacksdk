# Volume bootable·readonly flags

`SetVolumeBootableStatus`와 `SetVolumeReadonly`는 명시적인 volume ID로 Cinder v3 flag action을 호출합니다. SDK가 기본값·옵션 복사·microversion 선택·실제 응답 보존을 처리하며, caller가 요청 builder를 구현할 필요가 없습니다. 이름 조회·Resource 변환·CurrentLocation·후속 상태 대기는 수행하지 않습니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 실제 [bootable Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1149), [readonly Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1111), [Volume methods](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L177)입니다.

| openstacksdk Proxy | Go Connection / package | 전송 body |
|---|---|---|
| `conn.block_storage.set_volume_bootable_status(id, False)` | `SetVolumeBootableStatus(ctx, VolumeActionRequest{VolumeID: id}, false)` | `{"os-set_bootable":{"bootable":false}}` |
| `conn.block_storage.set_volume_readonly(id)` | `SetVolumeReadonly(ctx, VolumeActionRequest{VolumeID: id})` | `{"os-update_readonly_flag":{"readonly":true}}` |
| `conn.block_storage.set_volume_readonly(id, False)` | `SetVolumeReadonly(ctx, VolumeActionRequest{VolumeID: id}, WithVolumeReadonly(false))` | `{"os-update_readonly_flag":{"readonly":false}}` |

package 함수에는 `ctx` 다음에 `cinder.RawClient()`를 전달합니다. v3 service의 `cinder.Volumes`는 API **필드**이며 같은 이름의 method가 request 대신 `volumeID string`을 받습니다. `VolumeActionResult`는 세 호출 경로에서 공통입니다.

## 한 flag 실행

아래 프로그램은 clouds.yaml의 `CLOUD_NAME`과 실제 `VOLUME_ID`를 사용합니다. `FLAG_ACTION`은 필수이며 `bootable` 또는 `readonly`를 지정합니다. `FLAG_ACTION_PATH`는 `connection`·`direct`·`service` 중 하나이고 기본값은 `connection`입니다. `FLAG_VALUE`는 `true` 또는 `false`로 지정하며, bootable에는 필수이고 readonly에서는 생략 시 SDK 기본값 `true`를 사용합니다. 환경 변수의 문자열을 bool로 파싱하는 것은 이 예제 application의 입력 처리입니다.

예를 들어 `CLOUD_NAME=devstack VOLUME_ID=actual-volume-id FLAG_ACTION=readonly FLAG_VALUE=false FLAG_ACTION_PATH=connection go run ./volume-flag-example`로 실행합니다. 아래 프로그램을 SDK를 사용하는 모듈의 별도 example 디렉터리에 저장합니다. 한 실행에서 선택한 action 한 개를 호출합니다. parent context의 30초는 authentication·discovery·action 전체에 대한 application 제한입니다.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "strconv"
    "time"

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
    action := os.Getenv("FLAG_ACTION")
    if action != "bootable" && action != "readonly" {
        return fmt.Errorf("FLAG_ACTION must be bootable or readonly")
    }
    path := os.Getenv("FLAG_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown FLAG_ACTION_PATH %q", path)
    }
    text, supplied := os.LookupEnv("FLAG_VALUE")
    if action == "bootable" && !supplied {
        return fmt.Errorf("FLAG_VALUE is required for bootable")
    }
    flag := true
    if supplied {
        parsed, err := strconv.ParseBool(text)
        if err != nil {
            return fmt.Errorf("FLAG_VALUE: %w", err)
        }
        flag = parsed
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var policy blockstorage.VolumeReadonlyOpts
    if action == "readonly" {
        var originals []blockstorage.VolumeReadonlyOption
        if supplied {
            originals = append(originals, blockstorage.WithVolumeReadonly(flag))
        }
        prepared, err := blockstorage.PrepareVolumeReadonlyOptions(ctx, originals...)
        if err != nil {
            return err
        }
        policy = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input, flag, policy)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest, flag bool, policy blockstorage.VolumeReadonlyOpts) (*blockstorage.VolumeActionResult, error) {
    options := []blockstorage.VolumeReadonlyOption{blockstorage.WithVolumeReadonlyOptions(policy)}
    if path == "connection" {
        if action == "bootable" {
            return conn.SetVolumeBootableStatus(ctx, input, flag)
        }
        return conn.SetVolumeReadonly(ctx, input, options...)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        if action == "bootable" {
            return blockstorage.SetVolumeBootableStatus(ctx, cinder.RawClient(), input, flag)
        }
        return blockstorage.SetVolumeReadonly(ctx, cinder.RawClient(), input, options...)
    }
    if action == "bootable" {
        return cinder.Volumes.SetVolumeBootableStatus(ctx, input.VolumeID, flag)
    }
    return cinder.Volumes.SetVolumeReadonly(ctx, input.VolumeID, options...)
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
    }
}
```

`PrepareVolumeReadonlyOptions`는 HTTP나 서비스 선택 없이 original 옵션을 한 번 실행해 owned policy를 만듭니다. 위 예제는 준비한 policy를 full replacement factory에 넘겨 재사용하는 방법을 보여 줍니다. 보통은 `conn.SetVolumeReadonly(ctx, input, blockstorage.WithVolumeReadonly(false))`처럼 직접 전달하면 충분합니다. service 패키지 `blockstorage/v3/volumes`에도 같은 이름의 `VolumeReadonlyOpts`·`VolumeReadonlyOption` alias, 두 factory, `PrepareVolumeReadonlyOptions`가 있습니다.

Python은 실제 Proxy를 다음처럼 사용합니다. 각 줄은 독립 action이며 두 flag를 자동으로 묶는 workflow가 아닙니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.set_volume_bootable_status(volume_id, False)
conn.block_storage.set_volume_readonly(volume_id)       # default True
conn.block_storage.set_volume_readonly(volume_id, False)
```

## 기본값·옵션 소유권과 실행 순서

Bootable의 `bool`은 필수 인자이고 `false`도 그대로 body에 포함됩니다. Proxy에는 bootable 기본값이 없지만 Source Resource의 `Volume.set_bootable_status`만 defaulttrue입니다. Readonly는 Proxy의 defaulttrue를 nullable `VolumeReadonlyOpts.Readonly *bool`로 표현합니다. nil 또는 옵션 생략은 true이고, false pointer와 `WithVolumeReadonly(false)`는 false입니다.

`WithVolumeReadonlyOptions(value)`는 생성 시 입력 pointer를 복사하고 호출할 때 전체 policy를 복사하여 교체합니다. 뒤에 온 옵션이 선택을 바꾸며, `WithVolumeReadonly(false)` 뒤에 `WithVolumeReadonlyOptions(VolumeReadonlyOpts{})`를 적용하면 최종 기본값 true로 돌아갑니다. Prepare가 반환한 default도 owned true pointer입니다. SDK는 callback마다 pointer와 original 옵션 slice를 복사하므로 caller가 보관한 입력·중간 policy가 이후 HTTP 값을 바꾸지 않습니다. nil 옵션과 callback 오류는 오류로 반환합니다.

직접 package/service readonly는 context·선택 source를 capture하고 ID를 검사한 **뒤** originals를 실행합니다. callback 전후 source/context guard를 확인합니다. Connection readonly는 context·Connection·provider preflight → originals Prepare 한 번 → ID 검사 → cached Cinder v3 선택 순서이고, core에는 owned complete policy를 전달합니다. 따라서 두 경로의 callback과 ID 검사 순서를 같다고 가정하지 않습니다. Bootable에는 옵션 callback이 없으며 Connection preflight와 ID 검사 후 cached Cinder를 선택합니다.

## ID·version·opaque 결과

`VolumeID`는 비어 있지 않은 UTF-8 단일 unescaped path segment입니다. slash·backslash·query·fragment·percent escape·공백·control·`.`·`..`을 거부하며 UUID 모양만 요구하지는 않습니다. 이름처럼 보이는 문자열도 조회 없이 literal ID로 POST `volumes/{id}/action`에 사용합니다. 입력 오류나 nil client/Connection은 action 이름과 `volumes` label의 library error로 반환됩니다. Nova·Identity·CurrentLocation을 선택하지 않습니다.

이미 선택된 Cinder microversion이 nonempty이면 literal 문자열 그대로 사용하고 discovery를 생략합니다. 선택 version이 없으면 [Volume class maximum 3.71](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L146)을 ceiling으로 유한한 bodyless discovery 후보의 accepted200·300에서 maximum/optional minimum을 선택합니다. clean native404·405만 다음 후보로 넘어가며 accepted parse/Read/Close/source/context 오류는 현재 discovery proof를 유지하고 terminal입니다. 선택 문자열에 3.71을 강제하거나 원본 client를 덮어쓰지 않습니다. 유한 discovery shapes/status, server maximum 먼저 파싱, missing maximum의 no-version 처리와 keystoneauth canonicalization/cache/DiscoveryFailure 차이는 [state action version 정책](volume-actions.md#action-microversion)에 설명합니다.

`VolumeActionResult`의 `Discovery`는 accepted discovery HTTP들의 증거이고, `Applied`는 accepted action HTTP 한 단계의 증거입니다. `VolumeActionPage`의 Body·Header·StatusCode는 실제 응답의 owned snapshot입니다. action은 Source status<400을 HTTP100..399로 옮기며 body를 JSON으로 파싱하지 않습니다. empty·malformed JSON·invalid UTF-8도 action 응답에서는 opaque bytes입니다.

`Completed=true`는 accepted action의 Read/Close와 source/context 검사가 끝난 helper acknowledgement입니다. 서버의 bootable/readonly field가 실제 변경됐다는 조회 증거나 완료 대기 결과가 아닙니다. accepted Read/Close/source/context 오류에서는 `Applied`와 앞선 `Discovery`를 유지하고 `Completed=false`입니다. action 전 단계 실패에는 `Applied`가 없고, native>=400 또는 native OkCodes 확장에 따른 original-policy rejection도 `Applied`를 채우지 않습니다. native HTTP evidence와 현재 admitted ResponseError의 원인을 구분하여 반환하며, 모든 rejected body IO 오류를 보존한다는 뜻은 아닙니다.

captured ProviderClient·Endpoint·ResourceBase·Type·Microversion 변경은 terminal입니다. ordinary header snapshot과 fixed action method/URL/serialized body·framing·version header 정책은 physical 요청과 native retry에서도 지키며 authentication token은 live provider를 사용합니다. 대기·후속 GET·자동 rollback 또는 실패한 action의 SDK replay는 없습니다. native authentication/retry 정책에 따라 전송 시도가 발생할 수 있습니다.

## 기존 API와 Source 차이

기존 [cloud `SetVolumeBootable`](volume-mutations.md)은 NameOrID 조회와 defaulttrue를 가진 별도 helper입니다. 새 `SetVolumeBootableStatus`는 명시적 ID와 required bool을 받으며 그 cloud API를 변경하지 않습니다. 기존 [native `SetBootable`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go#L644) 및 service `SetBootable`은 BootableOpts와 accepted200 계약을 유지합니다. native `AttachMode.ReadOnly`는 attach mode이며 readonly flag setter가 아닙니다.

Source Proxy는 ID 또는 supplied Volume Resource를 `_get_resource`로 구성·갱신하고 두 Resource method는 response를 반환하지 않습니다. Go는 typed safe ID와 owned acknowledgement를 제공하며 mutable Resource/dict·constructor/location/descriptor side effects·custom urljoin coercion을 복제하지 않습니다. Source method는 bool annotation만으로 값을 변환하지 않고 caller 값을 그대로 body에 전달합니다. Go bool API는 dynamic null·숫자·container 입력을 받지 않는 명시적 typed domain입니다.

이 문서는 두 실제 v3 Proxy flag action의 사용과 Go mapping 차이를 설명합니다. v2/native leaf/Resource 의존 선언의 전체 지원, authenticated Python/OpenStack 실행, 전체 SDK 완료를 뜻하지 않습니다. Source 정적 검토·Go 예제 컴파일·실제 cloud 실행은 각각 별도 검증입니다.
