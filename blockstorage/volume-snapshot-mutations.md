# 볼륨 snapshot 생성·삭제

`CreateVolumeSnapshot`과 `DeleteVolumeSnapshot`은 Cinder v3 snapshot의 요청, 이름·ID 조회,
응답 병합과 대기를 라이브러리에서 처리합니다. Connection은 인증과 cached Cinder 선택을
담당하며, 직접 package 함수는 이미 준비한 `*gophercloud.ServiceClient`를 받습니다.
호출자는 Gophercloud builder를 구현하지 않고 필요한 `With...` 옵션만 지정합니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
[실제 cloud 생성 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L505)와
[삭제 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L799)입니다.
native transport의 기준은 Gophercloud `v2.15.0`입니다.

| 실제 Python cloud 호출 | Connection | 직접 package 함수 |
|---|---|---|
| `conn.create_volume_snapshot(volume_id)` | `conn.CreateVolumeSnapshot(ctx, blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID})` | `blockstorage.CreateVolumeSnapshot(ctx, cinder, request, options...)` |
| `conn.create_volume_snapshot(volume_id, force=True, wait=False)` | 같은 요청과 `WithCreateVolumeSnapshotForce(true)`, `WithCreateVolumeSnapshotWait(false)` | 같은 두 옵션 |
| `conn.delete_volume_snapshot("nightly")` | `conn.DeleteVolumeSnapshot(ctx, blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "nightly"})` | `blockstorage.DeleteVolumeSnapshot(ctx, cinder, request, options...)` |
| `conn.delete_volume_snapshot("nightly", wait=True, timeout=600)` | 같은 요청과 `WithDeleteVolumeSnapshotWait(true)`, `WithDeleteVolumeSnapshotWaitPolicy(...)` | 같은 두 옵션 |

| 생략한 값 | 생성 | 삭제 |
|---|---|---|
| `Force` | false를 요청에 명시 | 해당 옵션 없음 |
| `Wait` | true | false |
| `WaitPolicy.Timeout` | nil: 시간 제한 없음 | nil: 시간 제한 없음 |
| `WaitPolicy.PollInterval` | nil: 2초 | nil: 2초 |
| `Location` | 호출 시 captured 현재 scope | 호출 시 captured 현재 scope |

다음 독립 Go 예제는 실제 `sdk.Connect`·`sdk.FromProvider` 생성 방식, Connection 호출,
직접 package 대안과 부분 결과 확인을 함께 보여줍니다. `cloudName`은 clouds.yaml 항목이며,
이미 인증된 provider를 받는 대안은 `Adopt`입니다.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Configured(ctx context.Context, cloudName string) (*sdk.Connection, error) {
    return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectCreate(result *blockstorage.CreateVolumeSnapshotResult) {
    if result == nil { return }
    fmt.Println("latest logical ID JSON:", string(result.SnapshotID))
    if result.Created != nil {
        fmt.Println("POST:", result.Created.StatusCode, "bytes:", len(result.Created.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    fmt.Println("created-stage logical JSON:", string(result.CreatedValue))
    if result.CreatedSnapshot != nil {
        actual := result.CreatedSnapshot.Clone()
        fmt.Println("actual POST ID field:", string(actual.Body["id"]))
        var fields map[string]json.RawMessage
        if err := actual.Decode(&fields); err != nil {
            fmt.Println("optional projection:", err)
        } else {
            fmt.Println("actual POST field count:", len(fields))
        }
    }
    fmt.Println("final logical JSON:", string(result.Value), "ready:", len(result.Ready) != 0)
}

func InspectDelete(result *blockstorage.DeleteVolumeSnapshotResult) {
    if result == nil { return }
    if result.Deleted != nil { fmt.Println("requested deletion completed:", *result.Deleted) }
    if result.Resolved != nil {
        fmt.Println("lookup pages:", len(result.Resolved.Pages), "seeded ID:", result.Resolved.SeededID)
        if result.Resolved.Observed != nil {
            fmt.Println("lookup member:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil { fmt.Println("DELETE acknowledged:", result.Applied.StatusCode) }
    if result.Absent != nil { fmt.Println("clean polling absence:", result.Absent.StatusCode) }
    fmt.Println("latest logical ID JSON:", string(result.SnapshotID))
}

func InspectError(err error) {
    if err == nil { return }
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Println("accepted response failure:", response.StatusCode, len(response.Body))
    }
    var sdkTimeout *blockstorage.SnapshotWaitTimeoutError
    if errors.As(err, &sdkTimeout) {
        fmt.Println("SDK wait budget:", sdkTimeout.Timeout)
    }
    fmt.Println("deadline cause:", errors.Is(err, context.DeadlineExceeded))
}

func ViaConnection(ctx context.Context, cloudName, volumeID string) error {
    conn, err := Configured(ctx, cloudName)
    if err != nil { return err }
    created, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotName("nightly"),
        blockstorage.WithCreateVolumeSnapshotDescription("nightly volume snapshot"))
    InspectCreate(created) // Inspect completed physical stages even when err is nonnil.
    if err != nil { InspectError(err); return err }

    // Default deletion resolves exactly one snapshot and returns after acknowledgement.
    deleted, err := conn.DeleteVolumeSnapshot(ctx,
        blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "nightly"})
    InspectDelete(deleted)
    InspectError(err)
    return err
}

func DirectNoWait(ctx context.Context, cinder *gophercloud.ServiceClient, volumeID string) error {
    result, err := blockstorage.CreateVolumeSnapshot(ctx, cinder,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotForce(false),
        blockstorage.WithCreateVolumeSnapshotWait(false),
        blockstorage.WithCreateVolumeSnapshotFields(map[string]json.RawMessage{
            "display_name": json.RawMessage(`"nightly"`),
            "description": json.RawMessage(`"created without polling"`),
        }))
    InspectCreate(result)
    InspectError(err)
    return err
}

func WaitForDeletion(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    budget, interval := 10*time.Minute, time.Second
    result, err := conn.DeleteVolumeSnapshot(ctx,
        blockstorage.DeleteVolumeSnapshotRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeSnapshotWait(true),
        blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{
            Timeout: &budget, PollInterval: &interval,
        }))
    InspectDelete(result)
    InspectError(err)
    return err
}

func ZeroWaitBudget(ctx context.Context, conn *sdk.Connection, volumeID string) (*blockstorage.CreateVolumeSnapshotResult, error) {
    zero := time.Duration(0)
    result, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: &zero}))
    InspectCreate(result) // POST occurs first; an initially available result may already succeed.
    InspectError(err)
    return result, err
}

func WithCallerDeadline(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    parent, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    result, err := conn.CreateVolumeSnapshot(parent,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID})
    InspectCreate(result)
    InspectError(err)
    return err
}

func OwnedLocation(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Project: resource.CloudProject{
            ID: json.RawMessage(`"token-scope-id"`), Name: &project,
        },
    }
    wait := false
    result, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotOptions(blockstorage.CreateVolumeSnapshotOpts{
            Wait: &wait, Location: &location,
        }))
    InspectCreate(result)
    return err
}

func EmptyDelete(ctx context.Context) error {
    var conn *sdk.Connection
    result, err := conn.DeleteVolumeSnapshot(ctx, blockstorage.DeleteVolumeSnapshotRequest{})
    InspectDelete(result) // Valid context, no service selection, successful false.
    return err
}
```

대응하는 실제 Python 사용은 다음과 같습니다. 각 호출은 같은 cloud Connection에 대한 대안입니다.

```python
import openstack

conn = openstack.connect(cloud="configured-cloud")
snapshot = conn.create_volume_snapshot(
    "volume-id", name="nightly", description="nightly volume snapshot"
)  # force=False, wait=True, timeout=None
accepted = conn.create_volume_snapshot("volume-id", display_name="nightly", wait=False)
deleted = conn.delete_volume_snapshot("nightly")  # wait=False
deleted_after_wait = conn.delete_volume_snapshot("nightly", wait=True, timeout=600)
absent_input = conn.delete_volume_snapshot()  # False, no lookup
```

생성 입력 `VolumeID`는 POST body의 literal 값입니다. 이름으로 Volume을 조회하거나 경로 ID로
검사하지 않습니다. 빈 문자열도 서버로 전달하며, invalid UTF-8 문자열은 local 오류입니다.
`Force: nil`은 false이며 `{ "snapshot": { "volume_id": "...", "force": false } }`처럼
false를 반드시 보냅니다. 이름·설명·snapshot ID를 자동 생성하지 않습니다. 실제 Python
docstring의 이름 생성 설명과 달리 고정한 구현에는 그 처리가 없습니다.

생성의 추가 attribute는 정확히 `name`, `display_name`, `description`, `display_description`
네 가지입니다. `WithCreateVolumeSnapshotName`·`DisplayName`·`Description`·`DisplayDescription`은
string 편의 옵션이며, `WithCreateVolumeSnapshotAttributes` 또는 `WithCreateVolumeSnapshotFields`는
같은 네 key의 일반 JSON 값을 받습니다. `Attributes.Name` 같은 nonnil pointer는 같은 key의
raw `Fields` 값을 덮어씁니다. 원본 kwargs 검증과 같이 `metadata`, `size`, `id`, `base_path`,
runtime control·unknown·case-variant key는 거부합니다. 더 넓은 Proxy/native create 옵션은
이 cloud helper의 attribute가 아닙니다.

canonical `name`·`description`이 **존재하면** display alias보다 우선합니다. 선택한 값이 null·false·0·
빈 문자열·빈 배열·빈 object이면 해당 canonical 필드를 요청에서 생략하며, truthy display alias로
되돌아가지 않습니다. `Fields`의 nil entry는 JSON null입니다. truthy 숫자·배열·object는
string으로 변환하지 않고 그대로 전달합니다. 사용하지 않은 잘못된 alias 값은 변환하거나
JSON validation하지 않습니다. raw JSON에서는 decimal truthiness를 정확히 계산하므로
`1e-9999`가 truthy인 Go와 동일 JSON을 Python float로 읽었을 때 underflow하는 경계는 다릅니다.

`WithCreateVolumeSnapshotOptions`와 `WithDeleteVolumeSnapshotOptions`는 complete 정책을
교체합니다. 개별 factory는 해당 값만 지정하고, 나중 옵션이 같은 항목을 결정합니다.
`PrepareCreateVolumeSnapshotOptions`·`PrepareDeleteVolumeSnapshotOptions`는 original callback을
한 번 실행하고 pointer·map·raw JSON·옵션 slice를 소유하지만 service I/O나 실행 기본값
적용을 하지 않습니다. callback 오류·nil callback·nil/cancelled context는 오류입니다.
Connection 실행 순서는 context/필요한 Connection 검사 → original 옵션 → literal 생성 입력 검사
또는 빈 삭제 분기 → owned Location/`CurrentLocation` snapshot → cached Cinder v3 선택입니다.
직접 package의 nonempty 작업은 selected source와 ordinary header를 original 옵션보다 먼저 capture하며,
Location은 옵션 뒤에 복사합니다. 직접 호출의 Location 생략은 provider의 recorded token project ID를
사용하고, Connection은 configured cloud·region·project 이름 정보도 현재 location에 반영합니다.
native 재인증 이후 token은 갱신되어도 그 호출의 location snapshot은 바뀌지 않습니다.

정상 생성과 polling 응답은 envelope의 `snapshot` object 또는 flat object를 받습니다.
Source처럼 알려진 필드만 이전 상태에 덮어쓰며, 생략한 필드는 요청/이전 poll 값이 남고
명시적 null은 덮어씁니다. JSON duplicate key는 마지막 값이 첫 삽입 위치를 유지하고,
normalized attribute와 wire alias는 그 parsed 순서로 소비해 마지막 alias가 결정합니다.
알려진 필드의 descriptor 변환은 전체 logical view를 만들 때 즉시 수행합니다.
변환된 값은 실제 raw 응답 필드에 써넣지 않습니다. 예를 들어 wire `force: "FALSE"`,
`size: "٧"`, `metadata: "opaque"`는 actual resource에는 그대로 있고 logical view에서는
`is_forced: false`, `size: 7`, `metadata: {}`가 됩니다.

logical view는 [pinned v3 Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py)의
nullable 16필드 `consumes_quota`, `created_at`, `description`, `group_snapshot_id`, `is_forced`,
`progress`, `project_id`, `size`, `status`, `updated_at`, `user_id`, `volume_id`, `id`, `name`,
`location`, `metadata`입니다. 이 중 force는 bool 또는 caseless true/false string을 변환하고,
size는 source int descriptor를 따르며, metadata는 object를 유지하고 nonobject nonnull은
빈 object로 변환합니다. 다른 Body 필드는 untyped 일반 JSON이며 누락/null은 null입니다.
timestamp·ID·quota 값에 임의 string/bool schema를 추가하지 않습니다.
Snapshot에는 availability-zone descriptor가 없으므로 computed location의 zone은 항상 null입니다.
truthy foreign project ID는 raw 값으로 보존하고 현재 프로젝트의 이름·domain 정보를 지웁니다.
wire `location`·unknown 필드는 actual raw 증거에 남지만 logical view의 computed location을
대체하지 않습니다. Source request의 dirty 필드는 set에서 순회하므로 Python의 body key
순서를 보장하지 않으며, Go의 deterministic JSON serialization도 그 순서와 같다고 주장하지 않습니다.

빈 body 또는 UTF-8로 유효하지만 JSON 문법이 잘못된 body는 source의 JSON `ValueError`
tolerance에 맞춰 이전 logical 상태를 유지합니다.
실제 object를 관찰하지 않았으므로 해당 `RawResource`는 nil입니다. 반면 parsed `{}` 또는
`{"snapshot":{}}`는 실제 nonnil empty object입니다. valid JSON scalar·array·null·잘못된 envelope와
invalid UTF-8 bytes는 accepted 응답의 decode 오류이며 실제 bytes/header/status를 유지합니다.
완전한 known descriptor 변환 오류도 mutation 이후 발생할 수 있으므로 부분 결과를 함께 확인합니다.

| 생성 결과 필드 | 의미 |
|---|---|
| `Created` | 실제 admitted POST의 bytes/header/status. accepted read/Close 오류가 있어도 남을 수 있음 |
| `CreatedValue` | 생성 응답을 성공적으로 병합·변환한 named partial-stage logical JSON |
| `CreatedSnapshot` | 실제 POST member의 raw 필드. unknown/원래 타입 유지; empty/malformed body에서는 nil |
| `SnapshotID` | 마지막으로 소비한 merged logical ID의 raw JSON. 누락은 literal null이며 string으로 강제하지 않음 |
| `LastAccepted` | 마지막 admitted POST 또는 poll GET. decode/read/Close 오류에도 실제 응답이 있으면 그 증거 |
| `Value`, `Snapshot` | 요청한 생성 workflow가 성공한 뒤의 logical JSON과 마지막 실제 member. 실패에서는 commit하지 않음 |
| `Ready`, `ReadySnapshot` | 기다린 `available` 완료 단계의 logical JSON과 실제 member. no-wait에서는 nil |

생성은 기본적으로 기다립니다. 초기 merged status가 caseless `available`이면 GET 없이 성공하고,
ID 경로 검사나 timeout iterator도 사용하지 않습니다. 따라서 초기 available에서 missing/null ID와
timeout 0이 있어도 성공할 수 있습니다. `Wait(false)`도 후속 ID·status 경로 검사를 하지 않습니다.
전체 descriptor 변환은 두 경로에서도 이미 수행되므로 잘못된 force·size를 숨기지 않습니다.

그 밖에는 첫 poll GET을 즉시 수행합니다. 초기 `error`는 미리 실패 판정하지 않으며,
fresh poll 뒤 `available`을 먼저 검사하고 exact caseless `error`이면 `FailedStateError`입니다.
누락한 status는 이전 값이 남고, 명시적 null은 생성에서는 nonterminal입니다. nonnull nonstring status는
실제로 status를 검사하는 wait 단계의 오류입니다. poll에서 ID가 바뀌면 다음 GET은 최신 logical ID를
사용합니다. 다음 경로를 필요로 할 때만 nonempty UTF-8 단일 unescaped string segment를 검사하므로,
terminal status가 먼저 확인되면 사용하지 않을 새 ID를 경로로 검증하지 않습니다.
Python `urljoin`의 falsey/container coercion·leading slash 제거는 Go의 안전한 route domain으로
대신하며, 이 오류에서도 이미 완료한 생성·조회 증거는 남습니다.

삭제의 `NameOrID`는 trim하지 않고 같은 captured reader에서 정확히 찾습니다. 안전한 identity는
member GET을 먼저 시도하며 clean original 400·403·404에서만 name 목록으로 fallback합니다.
경로로 안전하지 않은 이름은 exact name 목록으로 찾고, 여러 일치는 `ErrAmbiguous`입니다.
이 조회의 필터·pagination·seeded member 정책은 [snapshot 조회 가이드](volume-snapshots.md)와 같습니다.
추가 확인 GET이나 volume/Identity 조회를 넣지 않습니다. 성공한 조회의 `Value`에 logical ID가
있으면 그 값을 사용하며, member에서 wire ID가 누락된 경우의 requested-ID seed도 유지합니다.
actual `Resolved.Snapshot.Body`에는 seed를 넣지 않습니다. 이 logical 상태를 삭제 prior state로
사용하는 것은 의미적으로 동일한 값의 재사용이며, converted 값을 원래 wire provenance로 설명하지 않습니다.

빈 `NameOrID`는 유효한 context와 original 옵션 처리 뒤 서비스 없이 `Deleted: &false`를 반환합니다.
직접 package의 nil client와 Connection의 nil receiver도 이 빈 입력에 사용할 수 있습니다.
사용하지 않는 Location·Cinder 설정·WaitPolicy 값은 검증하거나 선택하지 않습니다.
nil/cancelled context·nil callback·callback 오류는 빈 입력에서도 오류입니다. 목록까지 완료한
실제 부재 역시 false이며, 조회 실패는 false로 바꾸지 않습니다.

찾은 snapshot에 bodyless DELETE를 보내며 force·cascade·os-force_delete 옵션은 없습니다.
기본 `Wait`는 false라 admitted acknowledgement 이후 true를 반환합니다. 응답 body는 opaque이며
JSON이나 UTF-8을 요구하지 않습니다. 고정한 Source의 `<400` 정책을 HTTP 100..399로 적용합니다.
native 오류·retry callback에서 확장한 OkCodes로 원래 거부할 응답을 성공시킬 수는 없습니다.
실제 DELETE404는 `ignore_missing=False`에 해당하는 terminal 오류이고, 조회 완료 부재와 다릅니다.

삭제에서 `Wait(true)`이면 cached `deleted` 상태만으로 끝내지 않습니다. timeout·context·route
검사를 통과한 뒤 fresh GET을 수행합니다.
caseless `deleted` 또는 clean original GET404가 완료입니다. `error`·`error_deleting`은 실패
predicate가 아니며 계속 기다립니다. missing status는 이전 값을 유지하지만, fresh merged null이나
nonstring status는 생성의 nullable 규칙과 달리 오류입니다. changed logical ID는 다음 poll에 반영합니다.

poll GET404는 native rejection 경로를 유지하여 retry·reauthentication·live token 동작을 보존합니다.
body read와 Close 오류를 함께 관찰하므로 단순히 오류 안에 native404가 있다는 이유로 부재로
판정하지 않습니다. 깨끗한 original 직접 404를 `Absent`에 기록하고, 마지막 source/context guard까지
통과하면 삭제 완료(true)입니다. 기록한 404 자체만으로 오류와 함께 반환된 `Deleted`를 true로 바꾸지 않습니다.
404 body read/Close 실패·callback wrapper·expanded OkCodes rejection·source 변경·parent cancellation은
완료를 증명하지 못하며 `Deleted`는 nil입니다. 앞선 `Applied`와 admitted `LastAccepted`는 유지하고,
현재 rejection의 원래 native 증거와 추가 원인을 오류로 반환합니다.

| 삭제 결과 필드 | 의미 |
|---|---|
| `Deleted` | 성공 부재/빈 입력 false, 요청한 삭제 workflow 성공 true, 오류 nil |
| `Resolved` | 조회의 logical 값과 실제 member/list 증거; DELETE 전 실패·route 오류에도 완료한 조회 유지 |
| `SnapshotID` | 선택한 뒤 마지막으로 소비한 logical raw ID; lookup 부재에서는 nil |
| `Applied` | 실제 admitted DELETE의 opaque bytes/header/status; 원격 membership·삭제 완료 상태를 보증하지 않음 |
| `LastAccepted` | admitted DELETE 또는 마지막 admitted poll GET; 현재 rejected404로 덮어쓰지 않음 |
| `Ready`, `ReadySnapshot` | fresh GET 뒤 병합된 logical status가 `deleted`인 완료 단계. `ReadySnapshot`은 실제 member이며 empty/malformed이면 nil. 404 완료에서는 둘 다 nil |
| `Absent` | clean polling GET404의 실제 bytes/header/status. rejected DELETE404의 증거가 아님 |

wait timeout은 mutation acknowledgement와 초기 변환 뒤 시작합니다. **nil만 무제한**이며,
명시적 0/음수는 사용한 wait loop에서 첫 poll 전에 timeout입니다. 생성의 initial-ready shortcut과
no-wait, 삭제 lookup 부재/빈 입력에는 사용되지 않습니다. `SnapshotWaitTimeoutError`는
`context.DeadlineExceeded`를 unwrap하지만 SDK positive timeout으로 in-flight HTTP에 별도 deadline을
만들지는 않습니다. loop 경계에서 확인하므로 느린 GET 뒤 target을 관찰하면 성공할 수 있고,
nonterminal sleep 뒤 다음 경계에서 expiry를 알아차릴 수 있습니다. caller의 parent context deadline·
cancel cause는 별도로 HTTP와 sleep을 중단하고 `errors.Is`로 확인할 수 있습니다.
Python cloud 삭제도 `timeout=None`을 Proxy에 명시적으로 전달하므로 Proxy 자체의 120초 기본값을
이 경로의 대기 제한으로 사용하지 않습니다.

`PollInterval`은 Python cloud helper에 없는 Go 편의 옵션입니다. nil은 2초이고 0은 source
iterator의 zero-sleep normalization을 따라 100ms, timeout이 더 짧으면 그 값입니다.
음수는 실제 nonterminal sleep을 필요로 할 때 오류이며, no-wait·initial-ready·terminal 첫 관찰에는
사용하지 않습니다. 실패 목록이나 progress callback을 호출자가 구현할 필요는 없습니다.

한 workflow의 ProviderClient pointer·Endpoint·ResourceBase·Type·Microversion과 fixed method/URL/body를
검사하고, ordinary header는 capture한 copy를 사용합니다. auth/route/header transport control이나
충돌한 microversion header를 옵션으로 바꾸지 못하게 하며, native retry·reauth callback에서도
같은 선택과 요청 소유권을 확인합니다. provider의 live token·native 인증/재시도 정책은 계속 사용합니다.
accepted read·Close·decode·context/source 오류에 대한 실제 response 증거는 해당 단계에 남습니다.
라이브러리가 workflow를 재실행하거나 이미 생성한 snapshot을 자동 삭제/rollback하지 않습니다.

Connection은 Cinder v3를 선택하며 selected microversion을 그대로 보존합니다. 자동 upgrade나
3.65 cap을 추가하지 않습니다. pinned Snapshot의 `_max_microversion = "3.65"`는 Python의
session default가 **없을 때** discovery와 협상하는 상한이고, explicit session default는 별도입니다.
Go helper는 selected native 설정을 보존하며 그 server negotiation이나 v2 모델을 추가 구현하지 않습니다.
raw number spelling·exact decimal truthiness/equality와 Python parsed float, descriptor Unicode/임의정밀도
integer의 runtime 경계, strict UTF-8/safe route, 소유한 raw 응답/phase 결과와 Python의 mutable
Resource/session/cache는 명시적 Go mapping입니다. 이 두 direct helper 설명으로 Proxy·native leaves·
v2·generic Resource 또는 전체 SDK 지원 완료를 함께 선언하지 않습니다.
