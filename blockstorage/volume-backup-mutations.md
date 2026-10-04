# 볼륨 backup 생성·삭제

`CreateVolumeBackup`과 `DeleteVolumeBackup`은 Cinder v3 backup의 요청, 응답 병합,
정확한 이름·ID 조회와 대기를 라이브러리에서 처리합니다. Connection은 인증과 cached Cinder
선택을 담당하며 직접 package 함수는 준비된 `*gophercloud.ServiceClient`를 받습니다.
호출자는 builder 인터페이스를 구현하지 않고 필요한 `With...` 옵션을 지정합니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
[실제 cloud 생성](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L600)과
[삭제](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L758)입니다.
native transport의 기준은 Gophercloud `v2.15.0`입니다.

| 실제 Python cloud 호출 | Connection | 직접 package 함수 |
|---|---|---|
| `conn.create_volume_backup(volume_id)` | `conn.CreateVolumeBackup(ctx, blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID})` | `blockstorage.CreateVolumeBackup(ctx, cinder, request, options...)` |
| `conn.create_volume_backup(volume_id, force=False, incremental=True, wait=False)` | 같은 요청과 `WithCreateVolumeBackupForce(false)`, `WithCreateVolumeBackupIncremental(true)`, `WithCreateVolumeBackupWait(false)` | 같은 옵션 |
| `conn.delete_volume_backup("nightly")` | `conn.DeleteVolumeBackup(ctx, blockstorage.DeleteVolumeBackupRequest{NameOrID: "nightly"})` | `blockstorage.DeleteVolumeBackup(ctx, cinder, request, options...)` |
| `conn.delete_volume_backup("nightly", force=True, wait=True, timeout=600)` | 같은 요청과 `WithDeleteVolumeBackupForce(true)`, `WithDeleteVolumeBackupWait(true)`, `WithDeleteVolumeBackupWaitPolicy(...)` | 같은 옵션 |

| 생략한 값 | 생성 | 삭제 |
|---|---|---|
| `Force` | false를 body에 명시 | false: 일반 DELETE |
| `Incremental` | false를 body에 명시 | 해당 옵션 없음 |
| `Name`, `Description`, `SnapshotID` | JSON null | 해당 옵션 없음 |
| `Wait` | true | false |
| `WaitPolicy.Timeout` | nil만 무제한 | nil만 무제한 |
| `WaitPolicy.PollInterval` | nil: 2초 | nil: 2초 |
| `Location` | 호출에서 captured 현재 scope | 호출에서 captured 현재 scope |

다음 독립 Go 예제는 clouds.yaml로 Connection을 만드는 방법과 이미 인증한 provider를
받는 대안, 전체 정책 준비, 직접 package 호출 및 부분 결과 확인을 함께 보여줍니다.
`cloudName`은 clouds.yaml의 항목이고 volume/snapshot ID와 backup 이름은 호출자가 지정합니다.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "gophercloudsdk"
    "gophercloudsdk/blockstorage"
    "gophercloudsdk/resource"
)

func Configured(ctx context.Context, cloudName string) (*sdk.Connection, error) {
    return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func Explain(err error) {
    if err == nil { return }
    fmt.Println("workflow error:", err)
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("SDK wait timeout or parent context deadline")
    }
    if errors.Is(err, context.Canceled) {
        fmt.Println("parent context canceled")
    }
    if errors.Is(err, resource.ErrInvalidOption) {
        fmt.Println("local input, location, status or route error")
    }
}

func InspectCreate(result *blockstorage.CreateVolumeBackupResult) {
    if result == nil { return }
    fmt.Println("latest logical ID JSON:", string(result.BackupID))
    if result.Created != nil {
        fmt.Println("POST:", result.Created.StatusCode, "bytes:", len(result.Created.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    fmt.Println("created-stage logical JSON:", string(result.CreatedValue))
    if result.CreatedBackup != nil {
        actual := result.CreatedBackup.Clone()
        fmt.Println("actual POST ID:", string(actual.Body["id"]))
        var fields map[string]json.RawMessage
        if err := actual.Decode(&fields); err != nil {
            fmt.Println("optional projection:", err)
        } else {
            fmt.Println("actual POST field count:", len(fields))
        }
    }
    fmt.Println("final logical JSON:", string(result.Value), "ready:", len(result.Ready) != 0)
}

func InspectDelete(result *blockstorage.DeleteVolumeBackupResult) {
    if result == nil { return }
    if result.Deleted == nil {
        fmt.Println("requested delete policy did not complete")
    } else {
        fmt.Println("deleted:", *result.Deleted)
    }
    fmt.Println("latest logical ID JSON:", string(result.BackupID))
    if result.Resolved != nil {
        fmt.Println("resolved logical JSON:", string(result.Resolved.Value))
        fmt.Println("resolved list pages:", len(result.Resolved.Pages))
        if result.Resolved.Observed != nil {
            fmt.Println("resolved member:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil {
        fmt.Println("DELETE or force POST:", result.Applied.StatusCode, "bytes:", len(result.Applied.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    if result.Absent != nil {
        fmt.Println("clean polling GET absence:", result.Absent.StatusCode)
    }
    fmt.Println("deleted-status logical JSON:", string(result.Ready))
}

func CreateDefault(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    result, err := conn.CreateVolumeBackup(ctx,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID})
    InspectCreate(result)
    Explain(err)
    return err
}

func CreateWithoutWait(ctx context.Context, conn *sdk.Connection, volumeID, snapshotID string) error {
    policy, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx,
        blockstorage.WithCreateVolumeBackupName(""),
        blockstorage.WithCreateVolumeBackupDescription(""),
        blockstorage.WithCreateVolumeBackupForce(false),
        blockstorage.WithCreateVolumeBackupIncremental(true),
        blockstorage.WithCreateVolumeBackupSnapshotID(snapshotID),
        blockstorage.WithCreateVolumeBackupWait(false))
    if err != nil { return err }
    result, err := conn.CreateVolumeBackup(ctx,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeBackupOptions(policy))
    InspectCreate(result)
    return err
}

func RemoveDefault(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.DeleteVolumeBackup(ctx,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID})
    InspectDelete(result)
    Explain(err)
    return err
}

func ForceAndWait(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    timeout := 10 * time.Minute
    policy, err := blockstorage.PrepareDeleteVolumeBackupOptions(ctx,
        blockstorage.WithDeleteVolumeBackupForce(true),
        blockstorage.WithDeleteVolumeBackupWait(true),
        blockstorage.WithDeleteVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{
            Timeout: &timeout,
        }))
    if err != nil { return err }
    result, err := conn.DeleteVolumeBackup(ctx,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeBackupOptions(policy))
    InspectDelete(result)
    Explain(err)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient, volumeID string) error {
    cloud, region := "owned-cloud", "RegionOne"
    location := resource.CloudLocation{
        Cloud: &cloud, RegionName: &region,
        Project: resource.CloudProject{ID: json.RawMessage(`"project-id"`)},
    }
    result, err := blockstorage.CreateVolumeBackup(ctx, cinder,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeBackupLocation(location),
        blockstorage.WithCreateVolumeBackupForce(false),
        blockstorage.WithCreateVolumeBackupWait(false))
    InspectCreate(result)
    return err
}

func DirectDelete(ctx context.Context, cinder *gophercloud.ServiceClient, nameOrID string) error {
    result, err := blockstorage.DeleteVolumeBackup(ctx, cinder,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeBackupForce(false),
        blockstorage.WithDeleteVolumeBackupWait(false))
    InspectDelete(result)
    return err
}

func EmptyDelete(ctx context.Context) error {
    var conn *sdk.Connection
    result, err := conn.DeleteVolumeBackup(ctx, blockstorage.DeleteVolumeBackupRequest{})
    InspectDelete(result)
    if err != nil { return err }
    if result == nil || result.Deleted == nil || *result.Deleted {
        return errors.New("empty delete must return a completed false result")
    }
    return nil
}
```

실제 Python cloud 호출은 다음과 같습니다. cloud helper 자체에는 raw `**attrs`나
volume/snapshot Resource를 받는 overload가 없습니다.

```python
import openstack

conn = openstack.connect(cloud="my-cloud")
created = conn.create_volume_backup(volume_id)
queued = conn.create_volume_backup(
    volume_id, name="", description="", force=False, wait=False,
    incremental=True, snapshot_id=snapshot_id,
)
deleted = conn.delete_volume_backup("nightly")
forced = conn.delete_volume_backup("nightly", force=True, wait=True, timeout=600)
empty = conn.delete_volume_backup()
```

생성은 `/backups`에 여섯 field를 항상 보냅니다. 기본 body는
`{"backup":{"name":null,"volume_id":"volume-id","description":null,"force":false,"incremental":false,"snapshot_id":null}}`입니다.
생략한 Name·Description·SnapshotID의 null과 명시한 빈 string을 구분하고 false도 생략하지 않습니다.
Python docstring에 이름/설명 생성 언급이 있지만 실제 helper는 null을 전달합니다.
라이브러리가 자동 이름을 만들거나 volume/snapshot을 조회하지 않습니다. `VolumeID`와 `SnapshotID`는
literal body 데이터이며 empty·slash 값도 route identity로 검사하지 않고 서버에 전달합니다.
입력 string의 UTF-8 경계는 요청 전에 검증합니다.
입력 `Incremental`은 wire `incremental`로 보내고 nullable logical 값의 이름은 `is_incremental`입니다.
[Backup 생성 transform](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L116)이 만드는 구분입니다.
읽기 응답의 `incremental`을 `is_incremental` alias로 추가하지 않습니다.

생성 옵션은 Name·Description·Force·Incremental·SnapshotID·Wait·WaitPolicy·Location입니다.
이 직접 cloud helper에 없는 metadata/container/availability-zone·raw Fields/display alias를
추가 입력으로 노출하지 않습니다. 필요한 native 기능은 [기존 v3 API](v3/README.md)에서 별도로 선택합니다.
Python이 annotation과 달리 임의 runtime 값을 여섯 field에 보낼 수 있는 범위는 Go의 typed
string·bool 입력과 다릅니다. 예를 들어 Python `force="false"`는 문자열 자체를 wire에 보내지만
Go `WithCreateVolumeBackupForce`는 bool을 받습니다. 이를 Python wire coercion의 완전한 재현이라고
설명하지 않습니다.

`WithCreateVolumeBackupOptions`·`WithDeleteVolumeBackupOptions`는 complete 정책을 교체하고
개별 factory는 해당 항목만 지정합니다. 나중 옵션이 같은 항목을 결정합니다.
`PrepareCreateVolumeBackupOptions`·`PrepareDeleteVolumeBackupOptions`는 original callback을 한 번
실행하고 pointer·Location·WaitPolicy·옵션 slice를 소유하며 서비스 I/O를 하지 않습니다.
Prepare는 nil default를 실행 값으로 채우지 않습니다. 실제 사용하는 값은 해당 workflow에서 검증합니다.
Connection 실행 순서는 context/필요한 Connection 검사 → original 옵션 → 생성 입력 검사 또는
빈 삭제 분기 → owned Location/`CurrentLocation` snapshot → cached Cinder v3 선택입니다.
direct package의 nonempty 작업은 selected source와 ordinary header를 original 옵션 전에 capture합니다.
원본 callback을 public package 호출에서 다시 실행하지 않습니다.

생성 응답과 poll GET은 `backup` envelope object 또는 flat object를 병합합니다.
알려진 Body field만 request/이전 poll 상태에 덮어쓰고 생략한 field는 유지하며 명시적 null은
덮어씁니다. 요청의 VolumeID를 Backup ID로 seed하지 않습니다. response ID가 없으면 logical
`id`는 null이고, false·0·빈 string·container ID도 raw JSON 값으로 유지합니다.
JSON duplicate key는 마지막 값으로 collapse하되 첫 insertion 위치를 유지하며 normalized/wire alias를
그 parsed 순서로 소비한 마지막 값이 선택됩니다. source response update는 이전 값과의 dirty
equality로 새 값을 생략하지 않습니다. unknown/`self` field는 logical view에 넣지 않습니다.
모든 known descriptor는 logical view를 만들 때 즉시 변환하며 actual raw 응답에는 변환값을 쓰지 않습니다.
Python request의 dirty field는 set에서 순회하므로 body key 순서를 보장하지 않으며, Go의 deterministic
JSON serialization을 그 순서와 같다고 주장하지 않습니다.

logical view는 [v3 Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L26)의
23 Body field와 computed `location`, 총 24개입니다. missing/null은 null이며 `force`,
`has_dependent_backups`, `is_incremental`은 일반 Boolean입니다. Snapshot BoolStr과 달리 응답의
`"false"`도 nonempty string이므로 true입니다. `links`는 nonnull non-list를 한 번 list로 감싸고,
`metadata`는 nonobject nonnull을 `{}`로 바꾸며 `size`·`object_count`는 descriptor integer 변환입니다.
timestamp·ID·name·나머지 Body 값은 untyped JSON입니다. 전체 필드와 숫자 변환 경계는
[Backup 조회 가이드](volume-backups.md)에 설명합니다.

location은 옵션 처리 뒤 한 번 captured 현재 scope를 기반으로 매 병합 단계의 `project_id`와
literal `availability_zone`을 반영합니다. falsey/matching project는 현재 이름/domain을 유지하고,
truthy foreign project는 raw ID를 유지하며 이름/domain을 null로 지웁니다. 실제 row zone은 임의 JSON이고
missing/null은 null입니다. caller `Location.Zone`이나 raw 응답 `location`으로 대신하지 않습니다.
Connection은 configured cloud/region/project 사실을 사용하고 direct package의 생략 Location은
provider의 recorded token project ID를 사용하며 다른 설정을 추정하지 않습니다.
진행 중 live 인증 scope 변경은 captured logical scope를 바꾸지 않으며 다음 호출은 새로 snapshot합니다.

빈 body 또는 UTF-8이 유효하지만 JSON 문법이 잘못된 body는 source의 `ValueError` tolerance에
맞춰 이전 logical 상태를 유지합니다. 실제 object가 없으므로 해당 actual `RawResource`는 nil입니다.
parsed `{}`·`{"backup":{}}`는 actual nonnil empty object입니다. valid scalar/array/null·잘못된
envelope·invalid UTF-8·descriptor 변환 오류는 accepted 응답의 오류로 실제 body/header/status를 유지합니다.
actual `RawResource`는 unknown field와 정확한 raw 숫자 spelling을 소유합니다. `Clone`은 독립 복사이며
`Decode`는 선택적인 atomic typed projection입니다. projection 실패가 이미 완료한 workflow를 취소하지 않습니다.

| 생성 결과 필드 | 의미 |
|---|---|
| `Created` | 실제 admitted POST의 bytes/header/status; accepted Read/Close 실패에서도 남을 수 있음 |
| `CreatedValue` | 생성 응답 병합·변환이 완료한 logical 단계 |
| `CreatedBackup` | 실제 POST의 raw member; logical descriptor 변환 오류에서도 남을 수 있고 empty/malformed body에서는 nil |
| `BackupID` | 마지막 소비한 merged logical ID의 raw JSON; missing은 literal null |
| `LastAccepted` | 마지막 admitted POST 또는 poll GET; 현재 rejected 응답으로 덮어쓰지 않음 |
| `Value`, `Backup` | 요청한 생성 workflow 성공 뒤 logical JSON과 마지막 실제 member; 오류에서는 publish하지 않음 |
| `Ready`, `ReadyBackup` | 기다린 available 완료 단계; no-wait는 nil, actual member 없는 tolerated body에서는 raw resource nil |

생성은 기본적으로 기다립니다. 초기 merged status가 caseless `available`이면 GET·후속 ID 검사·
timeout iterator 없이 완료합니다. 그러므로 missing/null/사용할 수 없는 ID와 timeout 0/음수도
이 경로에서는 사용하지 않습니다. `Wait(false)`도 후속 ID/status wait 검사를 하지 않습니다.
두 경로 모두 앞의 전체 descriptor 변환을 생략하지는 않습니다.

다른 초기 상태에서는 첫 fresh GET을 즉시 수행합니다. 초기 `error`를 미리 실패로 판단하지 않으며,
fresh merged `available`을 먼저 확인하고 exact caseless `error`는 `FailedStateError`입니다.
explicit null status는 생성 wait에서 nonterminal이고 nonnull nonstring status는 실제 wait 검사 오류입니다.
생략한 status/ID는 이전 값이 유지됩니다. poll에서 ID가 바뀌면 필요한 다음 GET은 최신 logical ID를
사용합니다. 후속 HTTP가 필요한 시점에만 nonempty UTF-8 단일 unescaped string segment를
검증하므로 terminal status 뒤의 사용하지 않을 임의 ID를 경로로 검사하지 않습니다.
Python `urljoin`의 falsey/container coercion·leading slash 제거와 Go safe route domain은 다릅니다.

삭제는 같은 captured reader에서 정확한 이름/ID를 한 번 찾습니다. 안전한 identity는 member GET을
먼저 시도하고 clean original native 400·403·404에서만 이름 목록으로 fallback합니다. 경로로 안전하지
않은 이름은 exact 이름 목록으로 찾으며 여러 일치는 `ErrAmbiguous`입니다. 조회 실패는 정상 부재로
바꾸지 않습니다. [기존 Backup 조회](volume-backups.md)의 member ID seed와 페이지/alias/location
정책을 재사용하며 추가 확인 GET·Volume·Snapshot·Identity 조회를 넣지 않습니다.
실제 응답 ID가 있으면 그 logical ID를 사용하고 member wire ID가 없을 때는 requested-ID seed를
유지합니다. `Resolved.Backup.Body`에는 seed를 쓰지 않습니다. explicit null/unusable ID가 필요한
mutation route를 만들 수 없으면 local 오류이며 완료한 `Resolved`는 유지합니다.
선택한 logical 값을 삭제 prior state에 재사용해도 converted 값을 원래 wire provenance로 설명하지 않습니다.

빈 `NameOrID`는 유효한 context와 original 옵션 처리 뒤 service 없이 `Deleted: &false`입니다.
nil direct client와 nil Connection도 사용할 수 있으며 사용하지 않는 Location·WaitPolicy·service
설정을 선택하거나 검증하지 않습니다. nil/cancelled context·nil callback·callback 오류는 빈 입력에도
오류입니다. 전체 조회가 완료한 실제 부재도 false이며 조회 오류는 `Deleted: nil`입니다.

일반 삭제는 `/backups/{id}`에 bodyless DELETE입니다. force는 DELETE query 옵션이 아니라
`/backups/{id}/action`에 **`{"os-force_delete":null}` POST**입니다. [실제 Backup action](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L132)은
독립 microversion **3.64**를 사용합니다. original selected version이 생략/3.60/3.80이어도 force
action은 3.64이고, original client·capture한 normal version을 수정하지 않습니다. canonical version
header도 action 정책에 맞추며 subsequent wait GET은 다시 normal selected version을 사용합니다.
normal create/delete/fetch에 자동 3.64 cap을 추가하는 규칙과 혼동하지 않습니다.
native RetryFunc가 action version을 `MoreHeaders`로 바꾸거나 `OmitHeaders`로 지우면 terminal 오류이고,
실제 physical request의 두 Cinder version header도 3.64인지 확인합니다. original selected-version
`MoreHeaders`는 source capture에만 남으며 action override가 original client나 poll로 새지 않습니다.

두 mutation acknowledgement는 source의 `<400` 정책을 finite HTTP 100..399로 적용하며 body는
opaque bytes입니다. 삭제 응답을 JSON으로 decode하거나 상태에 병합하지 않습니다. native leaves는
별도 계약입니다. [native Backup Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L60)는
202만 받고 `omitempty`로 기본 false/null을 생략하며 empty VolumeID도 허용하지 않습니다.
native Delete는 202/204, Get은 200, [ForceDelete](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L357)는
202와 `os-force_delete: {}`를 사용합니다. cloud helper가 필요한 body와 status를 라이브러리가
소유하며 native/generated API 시그니처는 독립적입니다.
normal DELETE나 force POST의 404는 `ignore_missing=False`인 terminal mutation 오류입니다.
lookup 부재(false)나 polling absence와 다르고 `Applied`를 만들거나 `Deleted`를 false로 바꾸지 않습니다.

기본 삭제 Wait는 false여서 admitted acknowledgement 뒤 true입니다. `Wait(true)`는 cached
`deleted`만으로 끝내지 않고 timeout/context/route 검사 뒤 첫 fresh GET을 합니다.
fresh merged caseless `deleted` 또는 clean original GET404가 완료입니다. `error`·`error_deleting`은
failure predicate가 아니며 기다립니다. explicit null/nonstring status는 생성의 nullable 규칙과 달리
삭제 wait 오류이고, 생략한 field는 prior 상태를 유지합니다. changed ID는 필요한 다음 GET에 사용합니다.

poll GET404는 native rejection/retry/reauth/live-auth 경로를 유지하고 body Read·Close 오류를
관찰합니다. 단지 오류 chain에 404가 있다는 이유로 삭제 완료를 인정하지 않습니다.
original clean 404만 `Absent`에 기록하며 final source/context guard까지 통과하면 true입니다.
body IO·callback wrapper·expanded OkCodes rejection·source/cancellation 오류는 `Deleted: nil`이고
앞의 `Applied`·admitted `LastAccepted`를 유지합니다. 기록된 404 proof와 오류를 함께 받은 경우도
완료 true라고 추정하지 않습니다.

| 삭제 결과 필드 | 의미 |
|---|---|
| `Deleted` | 정상 부재/빈 입력 false, 요청한 삭제 정책 완료 true, 오류 nil |
| `Resolved` | 완료한 조회의 logical 값과 실제 member/list 증거 |
| `BackupID` | 선택 뒤 마지막 소비한 logical raw ID; 정상 lookup 부재는 nil |
| `Applied` | 실제 admitted normal DELETE 또는 force POST의 opaque body/header/status; 원격 삭제 완료 증명과 별개 |
| `LastAccepted` | admitted mutation 또는 마지막 admitted poll; rejected404로 덮어쓰지 않음 |
| `Ready`, `ReadyBackup` | fresh GET에서 merged deleted 상태를 확인한 logical 값과 실제 member; 404 완료에서는 nil |
| `Absent` | clean polling GET404의 body/header/status; rejected mutation404의 proof가 아님 |

wait budget은 생성 acknowledgement/초기 병합 뒤 또는 삭제 acknowledgement 뒤 시작합니다.
**nil만 무제한**이고 explicit 0/음수는 실제 사용하는 wait loop의 첫 poll 전에 timeout입니다.
no-wait·initial-ready·빈 삭제·정상 lookup 부재는 이 값을 사용하지 않습니다.
`BackupWaitTimeoutError`는 `context.DeadlineExceeded`를 unwrap하지만 SDK positive timeout이
in-flight HTTP에 새 deadline을 만들지는 않습니다. loop 경계에서 검사하므로 slow GET 뒤 terminal
status면 완료할 수 있습니다. caller parent context의 deadline·cancel/custom cause는 별도로
HTTP와 sleep을 중단하고 오류 원인으로 남습니다. cloud `timeout=None`은 삭제 Proxy의 standalone
120초 기본값을 덮어쓰므로 이 cloud 경로의 기본 제한은 120초가 아닙니다.
`PollInterval`은 Python 직접 cloud helper에 없는 Go 편의 옵션입니다. nil은 2초이고 0은 source
zero-sleep normalization의 100ms(더 짧은 timeout이면 그 값)입니다. 음수는 실제 nonterminal
sleep을 필요로 할 때 오류이며 no-wait/initial-ready/terminal 첫 관찰에서는 사용하지 않습니다.

admitted response의 Read·Close·decode·descriptor/context/source 오류에는 실제 해당 단계의
body/header/status를 남깁니다. rejected response를 admitted proof로 바꾸지 않습니다.
Gophercloud가 unexpected-status body Read·Close 오류를 버리는 경계 때문에 ordinary member
400·403·404 lookup에는 별도 관찰을 적용해 fault가 있으면 fallback을 막고, deletion poll404에는
clean absence 관찰을 적용합니다. rejected List, 다른 rejected member status, rejected mutation
응답의 body IO는 기존 native 정책이며 **모든 거부 응답의 IO 보존을 보장하지 않습니다.**

workflow는 original ProviderClient pointer·Endpoint·ResourceBase·Type·Microversion과 fixed
method/URL/body/header 정책을 지킵니다. ordinary header는 captured copy, auth token은 live
provider 값이고 native retry·reauth/redirect에도 요청 소유권과 source facts를 확인합니다.
force의 owned 3.64 action 정책은 original source guard와 함께 사용하며 polling 정책으로 새지 않습니다.
라이브러리가 workflow를 replay하거나 생성한 backup을 자동 삭제/rollback하지 않습니다.

Connection은 Cinder v3만 선택합니다. Python Backup `_max_microversion = "3.64"`는 normal
session default가 없을 때 discovery/협상하는 상한이고 explicit default는 먼저 사용합니다.
Go는 supplied normal version을 유지하며 자동 cap/upgrade/discovery 협상을 추가하지 않습니다.
force action의 explicit 3.64는 별도의 실제 source 동작입니다. 필요한 일반 협상은
[Connection microversion 정책](../docs/microversions.md)에서 명시합니다.
typed 입력, strict UTF-8/safe route, 소유한 physical proof와 raw numeric spelling, exact JSON
decimal truthiness/equality와 Python parsed float rounding/underflow, descriptor Unicode/임의정밀도
integer의 runtime 경계 및 Python mutable Resource/session/cache는 명시적 Go mapping입니다.
이 두 cloud helper는 v2, native/Proxy/Resource 선언과 restore/import/export/reset/update/metadata 및
전체 SDK 지원 검토를 함께 완료했다고 선언하지 않습니다.
