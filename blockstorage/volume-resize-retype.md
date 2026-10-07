# Volume extend·retype·extend completion

`ExtendVolume`, `RetypeVolume`, `CompleteVolumeExtend`는 명시적인 volume ID로 Cinder v3 action을 호출합니다. SDK가 옵션 기본값·값 복사·microversion 선택·실제 응답 보존을 처리하며 caller가 요청 builder를 구현할 필요가 없습니다. 각 호출은 독립 action이고 이름·Type 조회, Resource 변환, CurrentLocation, 완료 대기를 수행하지 않습니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 실제 [extend/completion Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1084), [retype Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1126), [Volume.extend/complete_extend](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L165), [Volume.retype](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L317)입니다.

| 실제 Python Proxy | Go Connection / package | 전송 body·기본값 |
|---|---|---|
| `conn.block_storage.extend_volume(id, size)` | `ExtendVolume(ctx, VolumeActionRequest{VolumeID: id}, newSize)` | `{"os-extend":{"new_size":size}}`; size 필수, 0·음수도 포함 |
| `conn.block_storage.retype_volume(id, type_id)` | `RetypeVolume(ctx, request, newType)` | `{"os-retype":{"new_type":type_id,"migration_policy":"never"}}` |
| `conn.block_storage.retype_volume(id, type_id, migration_policy="")` | `RetypeVolume(ctx, request, newType, WithVolumeRetypeMigrationPolicy(""))` | `migration_policy` 생략, `new_type`는 항상 포함 |
| `conn.block_storage.complete_volume_extend(id)` | `CompleteVolumeExtend(ctx, request)` | `{"os-extend_volume_completion":{"error":false}}`; false도 포함 |

package 함수는 `ctx` 다음에 `cinder.RawClient()`를 받습니다. v3 service의 `cinder.Volumes`는 API **필드**이며 같은 이름의 method가 request 대신 `volumeID string`을 받습니다. 세 경로는 공통 `VolumeActionResult`를 반환합니다.

## 한 action 실행

다음 프로그램은 clouds.yaml의 `CLOUD_NAME`, 실제 `VOLUME_ID`, `RESIZE_ACTION=extend|retype|complete-extend`를 요구합니다. `RESIZE_ACTION_PATH=connection|direct|service`의 기본값은 connection입니다. extend에는 `EXTEND_SIZE`가 필수이며 0·음수도 전달합니다. retype에는 `NEW_VOLUME_TYPE` 변수의 존재가 필수이고 supplied empty string도 허용합니다. `MIGRATION_POLICY`를 생략하면 never, 빈 문자열로 설정하면 field를 생략합니다. completion의 `COMPLETION_ERROR`는 생략 시 false입니다. 문자열을 int/bool로 파싱하는 것은 예제 application의 입력 처리입니다.

예를 들어 `CLOUD_NAME=devstack VOLUME_ID=actual-volume-id RESIZE_ACTION=retype NEW_VOLUME_TYPE=actual-type-id MIGRATION_POLICY=never go run ./volume-resize-example`로 실행합니다. 아래 프로그램은 SDK를 사용하는 모듈의 별도 example 디렉터리에 저장합니다. 한 실행에서 선택한 action 하나만 호출하며 parent context의 30초는 authentication·discovery·action 전체에 대한 application 제한입니다.

```go
package main

import (
    "context"
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
    if cloudName == "" || volumeID == "" {
        return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
    }
    action := os.Getenv("RESIZE_ACTION")
    if action != "extend" && action != "retype" && action != "complete-extend" {
        return fmt.Errorf("RESIZE_ACTION must be extend, retype or complete-extend")
    }
    path := os.Getenv("RESIZE_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown RESIZE_ACTION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var size int
    var newType string
    var retype blockstorage.VolumeRetypeOpts
    var completion blockstorage.VolumeExtendCompletionOpts
    switch action {
    case "extend":
        text, supplied := os.LookupEnv("EXTEND_SIZE")
        if !supplied {
            return fmt.Errorf("EXTEND_SIZE is required")
        }
        parsed, err := strconv.Atoi(text)
        if err != nil {
            return fmt.Errorf("EXTEND_SIZE: %w", err)
        }
        size = parsed
    case "retype":
        text, supplied := os.LookupEnv("NEW_VOLUME_TYPE")
        if !supplied {
            return fmt.Errorf("NEW_VOLUME_TYPE is required; an explicit empty string is allowed")
        }
        newType = text
        var originals []blockstorage.VolumeRetypeOption
        if policy, supplied := os.LookupEnv("MIGRATION_POLICY"); supplied {
            originals = append(originals, blockstorage.WithVolumeRetypeMigrationPolicy(policy))
        }
        prepared, err := blockstorage.PrepareVolumeRetypeOptions(ctx, originals...)
        if err != nil {
            return err
        }
        retype = prepared
    case "complete-extend":
        var originals []blockstorage.VolumeExtendCompletionOption
        if text, supplied := os.LookupEnv("COMPLETION_ERROR"); supplied {
            flag, err := strconv.ParseBool(text)
            if err != nil {
                return fmt.Errorf("COMPLETION_ERROR: %w", err)
            }
            originals = append(originals, blockstorage.WithVolumeExtendCompletionError(flag))
        }
        prepared, err := blockstorage.PrepareVolumeExtendCompletionOptions(ctx, originals...)
        if err != nil {
            return err
        }
        completion = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input, size, newType, retype, completion)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest, size int, newType string, retype blockstorage.VolumeRetypeOpts, completion blockstorage.VolumeExtendCompletionOpts) (*blockstorage.VolumeActionResult, error) {
    r := blockstorage.WithVolumeRetypeOptions(retype)
    c := blockstorage.WithVolumeExtendCompletionOptions(completion)
    if path == "connection" {
        switch action {
        case "extend": return conn.ExtendVolume(ctx, input, size)
        case "retype": return conn.RetypeVolume(ctx, input, newType, r)
        default: return conn.CompleteVolumeExtend(ctx, input, c)
        }
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        switch action {
        case "extend": return blockstorage.ExtendVolume(ctx, cinder.RawClient(), input, size)
        case "retype": return blockstorage.RetypeVolume(ctx, cinder.RawClient(), input, newType, r)
        default: return blockstorage.CompleteVolumeExtend(ctx, cinder.RawClient(), input, c)
        }
    }
    switch action {
    case "extend": return cinder.Volumes.ExtendVolume(ctx, input.VolumeID, size)
    case "retype": return cinder.Volumes.RetypeVolume(ctx, input.VolumeID, newType, r)
    default: return cinder.Volumes.CompleteVolumeExtend(ctx, input.VolumeID, c)
    }
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
            fmt.Printf("opaque body: %q\n", result.Applied.Body)
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

Prepare 함수는 HTTP 없이 original 옵션을 한 번 실행하여 owned default policy를 만듭니다. 위 예제는 준비한 policy를 전체 교체 factory로 재사용합니다. 일반 호출에서는 `conn.RetypeVolume(ctx, input, newType, blockstorage.WithVolumeRetypeMigrationPolicy(""))`나 `conn.CompleteVolumeExtend(ctx, input, blockstorage.WithVolumeExtendCompletionError(true))`처럼 직접 옵션을 전달하면 충분합니다. service 패키지 `blockstorage/v3/volumes`에도 같은 opts/option alias, factory와 Prepare 이름이 있습니다.

Python에서는 실제 Proxy를 다음처럼 호출합니다. 이 세 줄도 각각 별도 요청입니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.extend_volume(volume_id, 20)
conn.block_storage.retype_volume(volume_id, "actual-type-id", migration_policy="never")
conn.block_storage.complete_volume_extend(volume_id, error=False)
```

## 입력·기본값과 옵션 소유권

Extend는 필수 `newSize int`를 `new_size`에 항상 포함합니다. 0·음수, 기존 크기보다 큰지 또는 서버 지원 여부를 로컬 검사하지 않습니다. Python Source도 supplied size를 변환·truthiness 검사 없이 전달합니다. Go machine int 밖의 arbitrary integer와 null·bool·string·tuple 등 Python dynamic 입력은 명시적인 typed domain 차이입니다.

Retype의 필수 `newType string`은 UTF-8이면 빈 문자열·공백·slash·control도 body에 허용하며 Type GET·이름 해석이나 safe-route 제한을 추가하지 않습니다. Source의 [Resource._get_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1016)는 supplied Type Resource의 실제 id 또는 일반 입력을 그대로 반환합니다. Go string은 그 Resource.id의 null·bool·container domain을 복제하지 않습니다.

`VolumeRetypeOpts.MigrationPolicy *string`의 nil은 Proxy defaultnever이며 explicit empty string은 body에서 migration_policy를 생략합니다. Source Resource.retype 자체의 기본값은 None이고 truthy policy만 포함합니다. Go에서는 Python의 explicit None/False/0 대신 empty string으로 같은 omitted field를 요청합니다. 최종 policy는 UTF-8만 검사하고 never/on-demand enum은 제한하지 않습니다. `VolumeExtendCompletionOpts.Error *bool`의 nil은 false이고 explicit true/false 모두 body에 포함됩니다. [readonly 기본값 true](volume-flags.md)와 다릅니다.

`WithVolumeRetypeOptions`와 `WithVolumeExtendCompletionOptions`는 생성 시 입력 pointer를 복사하고 호출마다 전체 policy를 owned copy로 교체합니다. 뒤에 온 옵션이 선택을 바꾸며 complete empty opts로 교체하면 각각 never/false 기본값으로 돌아갑니다. Prepare가 반환하는 default pointer도 소유한 복사본입니다. original 옵션 slice는 실행 전 한 번 복사하고 callback 전후 policy pointer를 복사합니다. nil 옵션·callback 오류·context 중단은 오류로 반환하며 caller builder가 필요하지 않습니다.

직접 package/service retype은 source capture → volume ID 검사 → newType UTF-8 검사 → originals 순서입니다. completion도 source/ID 검사 뒤 originals를 실행하고 callback 전후 source/context guard를 확인합니다. Connection은 context·Connection·provider preflight → originals Prepare 한 번 → retype의 newType 검사 → volume ID 검사 → cached Cinder v3 선택 순서이며 owned complete policy를 core에 전달합니다. Extend에는 옵션이 없고 preflight·ID 검사 후 cached Cinder를 선택합니다.

## Route·version·응답 증거

volume ID는 비어 있지 않은 UTF-8 단일 unescaped path segment로, slash·backslash·query·fragment·percent escape·공백·control·`.`·`..`을 거부합니다. UUID 형식만 요구하거나 이름으로 재해석하지 않습니다. 고정 POST `volumes/{id}/action`은 VolumeDetail의 `/detail`을 사용하지 않습니다. 세 Source method에는 명시적 minimum microversion gate가 없고 SDK도 추가하지 않습니다. nonempty selected microversion은 literal 그대로, 없으면 finite discovery에서 class ceiling3.71을 사용합니다. clean native404/405 fallback, utility bounds 선택, keystoneauth cache/canonicalization 차이는 [state action version 정책](volume-actions.md#action-microversion)에 설명합니다. 원본 client version을 덮어쓰지 않습니다.

`Discovery`는 accepted discovery HTTP들의 proof이고 `Applied`는 accepted action HTTP의 owned Body·Header·StatusCode입니다. action은 Source status<400을 HTTP100..399로 옮기며 response JSON을 파싱하지 않습니다. empty·malformed·invalid UTF-8 action body도 opaque bytes입니다. `Completed=true`는 accepted Read/Close와 source/context 검사가 끝난 helper acknowledgement이며 size·type·server status의 완료 증거가 아닙니다. completion의 `error=true`는 오류를 Cinder에 알리는 입력이고 SDK가 backend rollback을 직접 실행하거나 완료를 조회하지 않습니다.

accepted Read/Close/source/context 오류에서는 현재 Applied와 앞선 Discovery를 유지하고 Completed=false입니다. action 전 실패에는 Applied가 없고 native>=400 및 native OkCodes 확장의 original-policy rejection은 success proof를 만들지 않습니다. native HTTP evidence와 현재 admitted ResponseError를 보존하되 모든 rejected body IO 오류가 보존된다고 주장하지 않습니다. fixed method/URL/serialized body·framing·version header와 captured ProviderClient·Endpoint·ResourceBase·Type·Microversion 정책은 physical 요청과 native retry에서도 확인하며 token은 live provider를 사용합니다. 추가 poll·자동 undo·SDK 자체의 실패 후 replay는 없습니다.

기존 native/service `ExtendSize`·`ChangeType`은 기존 opts와 accepted202 계약을 유지합니다. 현재 바인딩한 native numeric-required helper는 zero를 허용하지만 이를 새로 받은 모든 upstream helper의 동일성으로 주장하지 않습니다. native ChangeType은 empty NewType required 오류와 empty migration 생략을 가진 별도 계약이며 enum const는 runtime 검증이 아닙니다. 새 helper는 세 실제 v3 Proxy 선언의 Go mapping을 제공하며 supplied mutable Resource/dict의 constructor/location/descriptor 효과와 custom urljoin coercion을 대체합니다. Source annotation은 size/string/bool을 runtime 변환하지 않습니다. native/v2/Resource 의존 선언 자동 승격, authenticated Python/OpenStack 실행과 전체 SDK 완료는 각각 별도 범위입니다.
