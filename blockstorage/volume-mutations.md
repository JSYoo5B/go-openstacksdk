# 볼륨 속성 변경과 bootable 설정

`Connection.UpdateVolume`은 볼륨을 한 번 찾아 **실제로 달라진 known fields**만 PUT하고,
기존 상태·요청값·부분 서버 응답을 합친 owned 결과를 제공합니다.
`Connection.SetVolumeBootable`은 같은 기본 조회 후 bootable action을 보냅니다.
기본값과 raw field 처리는 라이브러리가 담당하며 interface builder를 구현할 필요는 없습니다.

| openstacksdk cloud | Go |
|---|---|
| `conn.update_volume("data", name="renamed")` | `conn.UpdateVolume(ctx, input, blockstorage.WithUpdateVolumeName("renamed"))` |
| `conn.update_volume("data", metadata={})` | `blockstorage.WithUpdateVolumeMetadata(map[string]string{})` |
| known attribute kwargs | `blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{...})` |
| `conn.set_volume_bootable("data")` | `conn.SetVolumeBootable(ctx, input)` |
| `conn.set_volume_bootable("data", False)` | `blockstorage.WithSetVolumeBootable(false)` |
| 호출별 location | `WithUpdateVolumeLocation` / `WithSetVolumeBootableLocation` |

다음은 Connection과 직접 package 호출을 함께 보여주는 독립 Go 예제입니다.
Connection은 `sdk.Connect` 또는 `sdk.FromProvider`로 준비합니다.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func InspectUpdate(result *blockstorage.UpdateVolumeResult) {
    if result == nil { return }
    fmt.Println("fixed mutation ID:", result.VolumeID)
    if result.Resolved != nil {
        fmt.Println("lookup pages:", len(result.Resolved.Pages))
        if result.Resolved.Observed != nil {
            fmt.Println("lookup status:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil {
        fmt.Println("actual PUT response:", result.Applied.StatusCode,
            "raw bytes:", len(result.Applied.Body))
    }
}

func Update(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        blockstorage.WithUpdateVolumeName("renamed"),
        blockstorage.WithUpdateVolumeMetadata(map[string]string{}),
        blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{
            "image_id": json.RawMessage(`"image-id"`),
            "scheduler_hints": json.RawMessage(`{"same_host":["peer-volume"]}`),
        }))
    InspectUpdate(result)
    if err != nil { return err }
    fmt.Println("merged normalized JSON:", string(result.Value))
    owned := result.Volume.Clone()
    fmt.Println("merged raw metadata:", string(owned.Body["metadata"]))

    // Empty kwargs still resolve the volume but do not send a PUT.
    unchanged, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"})
    InspectUpdate(unchanged)
    if err != nil { return err }
    fmt.Println("no PUT:", unchanged.Applied == nil)
    return nil
}

func Bootable(ctx context.Context, conn *sdk.Connection) error {
    input := blockstorage.SetVolumeBootableRequest{NameOrID: "data"}
    enabled, err := conn.SetVolumeBootable(ctx, input) // Default: true.
    if enabled != nil && enabled.Applied != nil {
        fmt.Println("enable acknowledgement:", enabled.Applied.StatusCode)
    }
    if err != nil { return err }
    disabled, err := conn.SetVolumeBootable(ctx, input,
        blockstorage.WithSetVolumeBootable(false))
    if disabled != nil && disabled.Applied != nil {
        fmt.Println("disable acknowledgement:", disabled.Applied.StatusCode)
    }
    return err
}

func UpdateWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    name := "renamed"
    location := resource.CloudLocation{
        Cloud: &cloud,
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        blockstorage.WithUpdateVolumeOptions(blockstorage.UpdateVolumeOpts{
            Attributes: blockstorage.UpdateVolumeAttributes{Name: &name},
            Location: &location,
        }))
    InspectUpdate(result)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    result, err := blockstorage.UpdateVolume(ctx, cinder,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        blockstorage.WithUpdateVolumeDescription("updated"))
    InspectUpdate(result)
    if err != nil { return err }
    _, err = blockstorage.SetVolumeBootable(ctx, cinder,
        blockstorage.SetVolumeBootableRequest{NameOrID: "data"},
        blockstorage.WithSetVolumeBootable(false))
    return err
}
```

실제 Python cloud 사용은 다음과 같습니다. `update_volume`은 변경된 Resource를 반환하고,
`set_volume_bootable`은 `None`을 반환합니다.

```python
updated = conn.update_volume("data", name="renamed", metadata={})
conn.set_volume_bootable("data")
conn.set_volume_bootable("data", False)
```

[pinned cloud update_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L234)과
[set_volume_bootable](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L254)의 실제 실행 경로를 기준으로 합니다.
둘 다 filter 없이 기본 볼륨 조회를 한 번 수행합니다. Go도 [기본 볼륨 조회](volume-reads.md)의
exact identity/name 정책을 공유하며 안전한 ID 경로를 쓸 수 없는 이름은 목록으로만 찾습니다.
조회가 완료되어도 볼륨이 없으면 `ErrNotFound`이고 mutation을 보내지 않습니다.

`VolumeID`는 입력이 아니라 **실제 canonical 조회 응답의 안전한 nonempty string ID**입니다.
missing·null·nonstring·unsafe ID는 `Resolved`를 남기는 local `ErrInvalidOption`이며 입력 ID를 seed하지 않습니다.
query 없는 PUT `/volumes/{VolumeID}`와 POST `/volumes/{VolumeID}/action`을 고정합니다.
응답이 나중에 다른 ID를 반환해도 `VolumeID`는 이 실제 요청 대상을 유지하고, update 결과의 raw `id`는
서버가 반환한 값으로 병합될 수 있습니다. Python의 mutable Resource ID와 route/runtime 변경은 별도 경계입니다.

`UpdateVolumeAttributes`의 `Name`·`Description`은 nullable string pointer이고,
`Metadata`는 nil이면 미지정, nonnil `{}`이면 빈 object를 지정합니다.
`Fields`는 모든 pinned Volume Body attribute와 wire alias의 raw JSON을 표현합니다.
명시적 metadata null은 `WithUpdateVolumeFields(map[string]json.RawMessage{"metadata": json.RawMessage("null")})`로 전달합니다.
구체적인 Name·Description·Metadata 입력이 대응하는 raw `Fields`보다 우선합니다.

Python의 cloud alias 정규화처럼 canonical `name`·`description`이 **존재하면** display alias보다 우선합니다.
선택한 값이 null·false·0·빈 string·빈 list·빈 object이면 해당 name/description을 업데이트에서 제외합니다.
따라서 `WithUpdateVolumeName("")`로 이름을 비우는 동작을 만들지 않습니다.
[pinned _get_volume_kwargs](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L494)가 이 규칙을 정의합니다.
다른 known fields의 null·falsey 값은 각 field의 의미와 dirty 비교를 유지합니다.

raw known fields는 다음과 같습니다. 같은 key는 attribute와 wire가 같습니다.
전체 37 Body descriptors 중 `id`는 입력 변경을 거부하며, 나머지 36개를 지원합니다.
서버가 이 속성 변경을 허용하는지는 Cinder가 판정합니다. 생성 helper의 required size를 주입하지 않습니다.

| attribute | wire key |
|---|---|
| `attachments`, `availability_zone`, `backup_id` | 같은 key |
| `consistency_group_id` | `consistencygroup_id` |
| `consumes_quota`, `cluster_name`, `created_at`, `description`, `encryption_key_id` | 같은 key |
| `extended_replication_status` | `os-volume-replication:extended_status` |
| `group_id` | 같은 key |
| `host` | `os-vol-host-attr:host` |
| `image_id` | `imageRef` |
| `is_bootable` / `is_encrypted` / `is_multiattach` | `bootable` / `encrypted` / `multiattach` |
| `migration_id` / `migration_status` | `os-vol-mig-status-attr:name_id` / `os-vol-mig-status-attr:migstat` |
| `project_id` | `os-vol-tenant-attr:tenant_id` |
| `replication_driver_data` | `os-volume-replication:driver_data` |
| `provider_id`, `replication_status` | 같은 key |
| `scheduler_hints` | `OS-SCH-HNT:scheduler_hints` |
| `service_uuid`, `shared_targets`, `size`, `snapshot_id` | 같은 key |
| `source_volume_id` | `source_volid` |
| `status`, `updated_at`, `user_id`, `volume_image_metadata`, `volume_type`, `volume_type_id`, `name`, `metadata` | 같은 key |

Go map 입력에서 attribute와 wire key가 둘 다 있으면 wire key가 우선합니다.
Python kwargs의 삽입 순서와 다른 명시적인 Go 규칙입니다. unknown 입력은 source처럼 무시하며
생성용 native 확장 `source_replica`도 이 update에서는 무시합니다.
`id`, `location`, `base_path`, `microversion`, `headers`, `uri`, `prepend_key`, `has_body`,
`retry_on_conflict`, `commit_method`, `allow_commit`은 고정 target·SDK policy를 바꾸므로 HTTP 전에 local error로 거부합니다.

dirty 판정은 normalized `Value` 대신 **원본 raw fields**를 비교합니다. 기존 fields의 absent와 explicit null은
서로 다르며, bool/number의 Python식 recursive equality, object key 순서 무관, list 순서를 적용합니다.
예를 들어 원본 numeric `1`과 요청 `true`는 같아서 기존 raw 값을 유지하지만 string `"3"`과 numeric `3`은 다릅니다.
Go는 JSON decimal을 정확히 비교하므로 Python float rounding·underflow 경계와는 차이가 있습니다.
빈 kwargs·동일 값·ignored unknown fields만 있으면 PUT을 생략하고 `Applied=nil`의 성공 결과를 반환합니다.
이 값은 서버 효과를 추측하는 `Changed` bool이 아니라 **실제 PUT이 없었다는 증거**입니다.
[pinned Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)의 no-op 경로를 따릅니다.

known 원본 lookup aliases는 선택된 실제 JSON row의 member 순서로 소비하며 마지막 textual alias가 우선합니다.
중복 key를 먼저 collapse하는 Python JSON object와 interleaved duplicate alias 순서는 다를 수 있습니다.
unknown 원본 fields는 Go raw 증거 확장으로 유지하고, unknown mutation 응답 fields는 병합하지 않습니다.
PUT 전에 proposed merged fields의 37 nullable descriptors와 computed location을 미리 만듭니다.
BoolStr 등 malformed known descriptor는 `Resolved`를 남기는 local error이고 PUT을 보내지 않습니다.
이 nullable view와 변환은 기존 [볼륨 조회·검색](search-volumes.md)의 JSON descriptor 정책을 공유하며,
더 엄격한 native typed Volume 모델로 먼저 축소하지 않습니다.

PUT body의 `volume`에는 dirty raw fields만 들어갑니다. dirty scheduler hints는 volume에서 제거하고,
truthy일 때만 top-level `OS-SCH-HNT:scheduler_hints`로 보냅니다. dirty hints가 null·빈 object 등 falsey이면
상위 hint는 생략하지만 `{"volume":{}}` PUT 자체는 발생할 수 있습니다.
[pinned Volume request body](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L446)를 반영합니다.

accepted update response는 canonical `volume` object 또는 flat object의 recognized fields만
요청이 반영된 기존 fields에 덮어씁니다. 생략한 fields는 유지하고 null·false·다른 ID도 recognized 값이면 반영합니다.
빈 body와 UTF-8 malformed JSON은 Python JSON decoder ValueError 처리처럼 허용하여 requested/prior 상태를 반환합니다.
반면 유효한 JSON의 null·scalar·array 또는 잘못된 `volume` envelope는 `Applied`를 보존하는 response error입니다.
잘못된 UTF-8는 JSON 대체문자나 Python decoder encoding 추정을 재현하지 않고 response error로 거부합니다.
최종 `Volume`과 `Value`는 병합·descriptor view 전체가 성공했을 때만 제공되며 `Resolved`·`Applied`와 독립적으로 소유합니다.
추가 GET으로 상태를 다시 가져오지 않습니다.
[pinned response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338)이 부분 병합과 malformed JSON tolerance의 기준입니다.

`SetVolumeBootableOpts.Bootable=nil`은 **true**입니다. `WithSetVolumeBootable(false)`는
`{"os-set_bootable":{"bootable":false}}`를 그대로 보냅니다.
Python은 bool annotation이 있어도 런타임 string·null·숫자·container를 그대로 전달할 수 있지만,
Go 공개 API는 bool domain으로 제한합니다. action 응답은 빈 값·non-JSON·binary를 opaque bytes로 유지합니다.
두 helper 모두 wait·timeout 옵션과 자동 polling·refresh·검증·rollback을 제공하지 않습니다.
Python bootable docstring의 ResourceTimeout 문구와 달리 실제 함수에는 wait/timeout 실행 분기가 없습니다.
[pinned Volume bootable action](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L177)을 따릅니다.

선택한 v3의 mutation 단계는 source `raise_from_response`의 `<400` 조건을
독립적인 `100..399` status policy로 유지합니다. lookup의 GET `200` policy와 별개입니다.
1xx 최종 응답 노출은 `net/http`를 따르고 다른 method·URL로 이동하는 redirect는 고정 요청 guard가 거부합니다.
기존 native `volumes.Update`와 `SetBootable`은 `200`만 받으며,
native `BootableOpts{}`의 zero bool은 false입니다. 기존 `CreateVolume`의 true-only readiness 후 bootable 단계도
별도의 `200` 정책입니다. 이 두 새 cloud helper로 그 leaf·생성 workflow의 정책을 바꾸지 않습니다.

`Resolved`는 조회의 실제 member·fallback page 증거이고 `Applied`는 실제 mutation 응답입니다.
accepted read·Close·cancellation/custom cause·선택 source 변경 오류에서도 이미 끝난 조회와 해당 mutation
body·header·status를 유지합니다. rejected native HTTP error는 실제 status·body·header를 유지하며
이전 lookup `200`을 mutation 실패의 accepted proof로 합성하지 않습니다.
local target·owned location·proposed descriptor 오류도 이전 성공 응답을 `ResponseError`로 빌리지 않습니다.
실제 응답 decode/view 오류는 자신의 accepted `ResponseError`를 가집니다.
`Applied`는 권한/속성/bootable의 서버 처리 완료나 변화 발생을 주장하는 `Changed`·완료 bool이 아닙니다.

Connection은 원본 옵션을 한 번 적용한 다음 `CurrentLocation` 또는 owned override를 snapshot하고,
그 뒤 cached Cinder v3를 선택합니다.
factory와 callback의 pointer·map·raw JSON·원본 옵션 slice를 소유하며, 호출별 location은 Connection 기본값을 바꾸지 않습니다.
직접 package 호출은 selected client source를 원본 callback 전에 capture한 뒤 recorded provider project를 한 번 snapshot합니다.
lookup와 mutation 사이에 token scope가 바뀌어도 이 호출의 computed location은 다시 읽지 않습니다.
실제 foreign volume의 project/zone은 기존 location 계산 규칙을 따릅니다.
Provider·Endpoint·ResourceBase·Type·Microversion 변경은 terminal error이고 live native auth·retry를 유지합니다.
전체 workflow를 재실행하지 않으며 method·URL·body·response decoder 소유권을 지킵니다.

Python `update_volume`은 조회 뒤 configured proxy가 v3인지 확인합니다. Go Connection은 처음부터 cached v3를 선택하며,
직접 `ServiceClient`는 그 자체로 API 버전을 증명하지 않으므로 이 문서의 v3 계약에 맞는 client를 전달합니다.
Python bootable의 configured v2/v3 분기, generic Resource/session의 mutable 상태·cache·microversion negotiation을
이 고정 source snapshot과 동일하다고 주장하지 않습니다.
raw 입력의 truthiness·equality는 exact decimal을 사용하므로 Python JSON float rounding·underflow와 다를 수 있습니다.
이 차이는 dirty equality뿐 아니라 raw name/description과 scheduler hints의 falsey 선택에도 적용됩니다.
lookup·response의 aliases는 원본 member 순서로 처리하지만 Go map 입력은 wire key 우선입니다.
unknown 원본 raw fields 보존은 Go 증거 확장이고 unknown response fields는 무시합니다.
위 두 pinned cloud 선언의 Go 매핑과 별도 native·Proxy·Resource 지원 상태를 독립적으로 검토하며,
전체 openstacksdk 구현 완료를 의미하지 않습니다.
