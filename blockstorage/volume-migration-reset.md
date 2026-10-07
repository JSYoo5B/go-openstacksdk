# Volume status reset·migration·migration completion

`ResetVolumeStatus`, `MigrateVolume`, `CompleteVolumeMigration`는 명시적인 volume ID로 Cinder v3 action을 호출합니다. SDK가 기본값·옵션 복사·microversion policy·실제 응답 보존을 처리하므로 caller가 builder를 구현할 필요가 없습니다. 각 호출은 독립 action입니다.

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 실제 [reset Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1199), [migration/completion Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1336), [Volume.reset_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L209), [Volume.migrate/complete_migration](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L330)입니다.

| 실제 Python Proxy | Go Connection / package | body·기본값 |
|---|---|---|
| `reset_volume_status(id)` | `ResetVolumeStatus(ctx, request)` | `{"os-reset_status":{}}`; status 기본값 없음 |
| `reset_volume_status(id, status="available")` | `WithVolumeStatusResetStatus("available")` | nonempty status만 포함; enum 검사 없음 |
| `migrate_volume(id, host="host@backend")` | `MigrateVolume(ctx, request, WithVolumeMigrationHost("host@backend"))` | host 포함, false flags 생략 |
| `migrate_volume(id, cluster="")` | `WithVolumeMigrationCluster("")` | 빈 cluster도 포함하고 required3.16 검사 |
| `complete_volume_migration(id, new_id)` | `CompleteVolumeMigration(ctx, request, newVolume)` | new_volume과 error:false 항상 포함 |

package 함수는 `ctx` 다음에 선택한 `*gophercloud.ServiceClient`를 받습니다. `conn.BlockStorageV3(ctx)`로 얻은 service의 `Volumes`는 **API 필드**이며 같은 이름의 method가 request 대신 `volumeID string`을 받습니다. 세 경로는 같은 `*VolumeActionResult`를 반환합니다.

## 한 action 실행

다음 프로그램은 clouds.yaml의 `CLOUD_NAME`, 실제 `VOLUME_ID`, `VOLUME_MIGRATION_ACTION=reset|migrate|complete`를 요구합니다. `VOLUME_MIGRATION_PATH=connection|direct|service`의 기본값은 connection입니다. 한 실행에서 선택한 action 하나를 호출합니다. `ctx`의 30초는 인증·discovery·action을 포함하는 application 제한입니다.

reset의 세 환경변수는 생략하면 해당 필드를 보내지 않고, 명시한 빈 문자열도 보내지 않습니다. migrate의 `MIGRATION_HOST`·`MIGRATION_CLUSTER`는 생략과 빈 문자열을 구분합니다. `MIGRATION_FORCE_HOST_COPY`·`MIGRATION_LOCK_VOLUME`는 생략/false이면 body에서 빠집니다. completion의 `NEW_VOLUME_ID`는 환경변수의 존재가 필수이며 빈 값도 literal로 전달하고, `MIGRATION_ERROR` 기본값은 false입니다. bool 문자열 파싱은 이 예제 application의 입력 처리입니다.

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
    action := os.Getenv("VOLUME_MIGRATION_ACTION")
    if action != "reset" && action != "migrate" && action != "complete" {
        return fmt.Errorf("VOLUME_MIGRATION_ACTION must be reset, migrate or complete")
    }
    path := os.Getenv("VOLUME_MIGRATION_PATH")
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown VOLUME_MIGRATION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var reset blockstorage.VolumeStatusResetOpts
    var migration blockstorage.VolumeMigrationOpts
    var completion blockstorage.VolumeMigrationCompletionOpts
    var newVolume string
    switch action {
    case "reset":
        var options []blockstorage.VolumeStatusResetOption
        if value, supplied := os.LookupEnv("RESET_STATUS"); supplied {
            options = append(options, blockstorage.WithVolumeStatusResetStatus(value))
        }
        if value, supplied := os.LookupEnv("RESET_ATTACH_STATUS"); supplied {
            options = append(options, blockstorage.WithVolumeStatusResetAttachStatus(value))
        }
        if value, supplied := os.LookupEnv("RESET_MIGRATION_STATUS"); supplied {
            options = append(options, blockstorage.WithVolumeStatusResetMigrationStatus(value))
        }
        prepared, err := blockstorage.PrepareVolumeStatusResetOptions(ctx, options...)
        if err != nil { return err }
        reset = prepared
    case "migrate":
        var options []blockstorage.VolumeMigrationOption
        if value, supplied := os.LookupEnv("MIGRATION_HOST"); supplied {
            options = append(options, blockstorage.WithVolumeMigrationHost(value))
        }
        if value, supplied := os.LookupEnv("MIGRATION_CLUSTER"); supplied {
            options = append(options, blockstorage.WithVolumeMigrationCluster(value))
        }
        for _, control := range []struct {
            name string
            option func(bool) blockstorage.VolumeMigrationOption
        }{
            {"MIGRATION_FORCE_HOST_COPY", blockstorage.WithVolumeMigrationForceHostCopy},
            {"MIGRATION_LOCK_VOLUME", blockstorage.WithVolumeMigrationLockVolume},
        } {
            if text, supplied := os.LookupEnv(control.name); supplied {
                value, err := strconv.ParseBool(text)
                if err != nil { return fmt.Errorf("%s: %w", control.name, err) }
                options = append(options, control.option(value))
            }
        }
        prepared, err := blockstorage.PrepareVolumeMigrationOptions(ctx, options...)
        if err != nil { return err }
        migration = prepared
    case "complete":
        value, supplied := os.LookupEnv("NEW_VOLUME_ID")
        if !supplied { return fmt.Errorf("NEW_VOLUME_ID is required; explicit empty is allowed") }
        newVolume = value
        var options []blockstorage.VolumeMigrationCompletionOption
        if text, supplied := os.LookupEnv("MIGRATION_ERROR"); supplied {
            value, err := strconv.ParseBool(text)
            if err != nil { return fmt.Errorf("MIGRATION_ERROR: %w", err) }
            options = append(options, blockstorage.WithVolumeMigrationCompletionError(value))
        }
        prepared, err := blockstorage.PrepareVolumeMigrationCompletionOptions(ctx, options...)
        if err != nil { return err }
        completion = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, path, action, input, newVolume, reset, migration, completion)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, newVolume string, reset blockstorage.VolumeStatusResetOpts, migration blockstorage.VolumeMigrationOpts, completion blockstorage.VolumeMigrationCompletionOpts) (*blockstorage.VolumeActionResult, error) {
    r := blockstorage.WithVolumeStatusResetOptions(reset)
    m := blockstorage.WithVolumeMigrationOptions(migration)
    c := blockstorage.WithVolumeMigrationCompletionOptions(completion)
    if path == "connection" {
        switch action {
        case "reset": return conn.ResetVolumeStatus(ctx, input, r)
        case "migrate": return conn.MigrateVolume(ctx, input, m)
        default: return conn.CompleteVolumeMigration(ctx, input, newVolume, c)
        }
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        switch action {
        case "reset": return blockstorage.ResetVolumeStatus(ctx, cinder.RawClient(), input, r)
        case "migrate": return blockstorage.MigrateVolume(ctx, cinder.RawClient(), input, m)
        default: return blockstorage.CompleteVolumeMigration(ctx, cinder.RawClient(), input, newVolume, c)
        }
    }
    switch action {
    case "reset": return cinder.Volumes.ResetVolumeStatus(ctx, input.VolumeID, r)
    case "migrate": return cinder.Volumes.MigrateVolume(ctx, input.VolumeID, m)
    default: return cinder.Volumes.CompleteVolumeMigration(ctx, input.VolumeID, newVolume, c)
    }
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

Prepare 함수는 HTTP 없이 original 옵션을 한 번 실행하고 owned policy를 반환합니다. 위 프로그램은 그 policy를 전체 교체 factory로 재사용합니다. 일반 호출에서는 `conn.MigrateVolume(ctx, request, blockstorage.WithVolumeMigrationHost("host@backend"))`처럼 직접 옵션을 전달하면 충분합니다. service 패키지 `blockstorage/v3/volumes`에도 같은 opts/option alias, factory, Prepare 이름이 있습니다.

Python에서는 실제 Proxy를 다음처럼 호출합니다. 각 줄은 별도 요청입니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.reset_volume_status(volume_id, status="available", attach_status="detached")
conn.block_storage.migrate_volume(volume_id, host="host@backend", force_host_copy=True)
conn.block_storage.complete_volume_migration(volume_id, new_volume_id, error=False)
```

## 옵션과 microversion

reset의 `Status`·`AttachStatus`·`MigrationStatus`는 nullable string입니다. nil과 빈 문자열은 생략합니다. SDK가 available/detached 같은 기본 status나 enum을 만들지 않습니다. 아무 옵션이 없어도 빈 os-reset_status object를 전송합니다.

migration의 `Host`·`Cluster`는 nil이면 생략하고 nonnil이면 빈 문자열도 보냅니다. `ForceHostCopy`·`LockVolume`는 기본 false이며 true만 보냅니다. SDK는 host/cluster가 둘 다 없거나 둘 다 있는 요청을 막거나, host/cluster 이름을 resolve하지 않습니다. 이 요청의 상태·권한·backend 조건은 서버가 판단합니다.

**Cluster가 nonnil이면**, 빈 문자열을 포함하여 required3.16 검사를 수행합니다. 선택한 microversion이 있어도 서버 advertisement의 min/max 둘 다를 확인하고 required3.16이 그 범위에 포함되어야 합니다. 선택 버전도 required와 호환되는 같은 major의3.16 이상이어야 합니다. 서버 range 검사는 선택 버전 자체가 아니라 required3.16에 대한 검사이므로, server max3.20과 선택3.90도 이 gate를 통과하면 선택 literal3.90을 POST에 사용합니다. 이는 서버가 그 action을 받아 준다는 보장이 아닙니다. 고정 [Keystoneauth5.16.0 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)는 same-major와 tuple 비교를 사용하므로 global `latest`는 required3.16과 major가 달라 실패하고 `3.latest`는 그 비교를 통과할 수 있습니다. 원래 selected 문자열의 POST 사용과 Keystoneauth Session의 실제 header canonicalization은 별도 경계입니다. 이 dependency pin은 Source 정적 비교이며 Python runtime 실행 근거가 아닙니다.

Cluster가 nil이면 이 minimum gate는 없습니다. 비어 있지 않은 selected 버전은 그대로 사용하며, selected가 없으면 finite discovery에서 Volume ceiling3.71과 server maximum/optionalminimum으로 선택합니다. cluster gate가 필요한 미선택 호출도 같은 operation-owned advertisement에서 버전을 선택합니다. source/cache의 microversion은 변경하지 않습니다. gate 실패나 discovery 오류는 action POST 전에 끝나며 확보한 discovery proof는 결과에 남습니다. required gate의 bounds/selected 불일치는 `resource.ErrUnsupported`를 포함합니다. 유한 후보 소진으로 usable advertisement가 없으면 gate도 실패하며, accepted data가 있으면 그 data의 proof를 보존합니다. 이 실패 정책은 일반 cap 선택의 미지원 버전 생략과 다릅니다.

completion의 `Error`는 기본 false이고 false/true 모두 body에 포함됩니다. `newVolume`은 UTF-8 literal body string으로 필수이며 빈 문자열도 전달합니다. 이 값을 lookup하거나 현재 volume route처럼 제한하지 않습니다. `error=true`는 Cinder에 오류를 알리는 입력이며 SDK가 cleanup이나 rollback을 수행한다는 뜻이 아닙니다.

## 결과와 경계

`Discovery`는 확보한 discovery 응답이며, `Applied`는 실제 action 응답의 owned Body/Header/StatusCode입니다. action body는 opaque이므로 empty/malformed/invalid-UTF8 응답도 원문 그대로 보존합니다. 원래 policy의 HTTP100..399와 read/close·source 검증이 성공하면 `Completed=true`입니다. backend migration 완료, volume 상태 확인이나 cleanup 성공을 의미하지 않습니다.

`Applied`가 있으면서 error가 있을 수 있습니다. accepted Read/Close 오류, custom context cancellation과 source 변경은 이미 받은 응답 proof를 남기고 Completed를 false로 유지합니다. 원래 policy가 거부한 HTTP 응답은 native error이며 action Applied를 만들지 않습니다. 이전 discovery 응답을 뒤의 rejected action 오류의 현재 proof로 빌리지 않습니다. SDK는 accepted-fault action을 자동 재전송하거나 undo하지 않습니다.

세 action은 GET으로 volume/host/cluster/new-volume을 조회하지 않고, CurrentLocation·Resource 모델 변환·wait·poll을 호출하지 않습니다. Connection은 cached Cinder v3를 사용하며 옵션 준비와 service 선택을 소유합니다. 패키지 함수는 선택한 Source와 현재 route ID를 검증한 뒤 original options를 실행합니다. completion의 newVolume UTF-8도 이 단계에서 검사합니다. Connection은 context/provider 선행 검증 후 original options를 준비하고, completion의 newVolume UTF-8와 route ID를 검사한 다음 cached Cinder를 선택합니다. SDK는 original 옵션 slice를 실행 전에 복사하고 callback마다 pointer policy를 복사하며, callback을 요청 retry마다 다시 실행하지 않습니다.

Go는 안전한 UTF-8 현재 volume ID와 typed string/bool 입력을 사용합니다. Python의 supplied Resource/dict/Munch·descriptor·constructor/location side effect·동적 nonstring/null 입력을 재현하지 않습니다. Python supports_microversion의 explicit default=""와 None을 Go ServiceClient의 빈 문자열 하나로 모두 구분할 수 없으며, empty는 Go 미선택 policy입니다. 원본은 require stage와 ordinary selection에서 endpoint-data getter를 각각 호출할 수 있으나 physical GET 횟수는 cache가 결정합니다. Go는 유한하고 call-owned인 discovery를 한 번 공유하며 Keystoneauth 전체 cache/link/envelope/canonicalization과 exception 순서·HTTP taxonomy를 동일하게 주장하지 않습니다.

기존 native `cinder.Volumes.ResetStatus(ctx, id, ResetStatusOpts, ...)`는 status string을 항상 보내고 202만 받아 error를 반환하는 독립 API입니다. 새 세 helper가 native/v2/Resource 선언을 자동으로 지원 판정하지 않습니다. 고정 Source 정적 비교와 Go 컴파일·로컬 HTTP 검증은 인증된 OpenStack/Python runtime 검증 또는 전체 SDK 완성을 뜻하지 않습니다.

[다른 볼륨 action](volume-actions.md), [크기·type 변경](volume-resize-retype.md), [service API](v3/volumes/README.md)를 함께 참고하세요.
