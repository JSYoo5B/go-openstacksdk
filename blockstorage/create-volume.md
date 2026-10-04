# 데이터 볼륨 생성

`conn.CreateVolume`은 Cinder v3 볼륨을 만들고, 기본적으로 새 볼륨의 `available` 상태까지 기다립니다. 라이브러리가 생성 속성·기본값·이미지 이름 해석·상태 대기를 처리합니다. Bootable을 명시하면 대기를 반드시 수행하며 true일 때만 완료 뒤 bootable 액션을 보냅니다. 별도 request builder를 구현하지 않습니다.

이미지 이름이 없을 때의 기본 순서는 Cinder `POST /volumes` 202 → `GET /volumes/{created_id}` 200입니다. 대기 중에는 고정 ID 조회를 반복합니다. 이미지 이름은 생성 전에 Glance의 기존 Collection에서 한 번 해석합니다. 이미지 ID, snapshot ID, source volume ID와 backup ID는 그 값을 사용하며 추가 조회를 만들지 않습니다.

다음 함수들은 별도 호출 예입니다. `conn`은 [전체 README](../README.md)처럼 준비한 Connection입니다.

```go
package example

import (
    "context"
    "encoding/json"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "gophercloudsdk"
    "gophercloudsdk/blockstorage"
    "gophercloudsdk/resource"
)

func createDefault(ctx context.Context, conn *sdk.Connection) (*blockstorage.CreateVolumeResult, error) {
    return conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 10},
        blockstorage.WithCreateVolumeName("data"),
        blockstorage.WithCreateVolumeMetadata(map[string]string{"owner": "worker"}))
}

func createWithoutWaiting(ctx context.Context, cinder *gophercloud.ServiceClient) (*blockstorage.CreateVolumeResult, error) {
    return blockstorage.CreateVolume(ctx, cinder, nil,
        blockstorage.CreateVolumeRequest{Size: 10},
        blockstorage.WithCreateVolumeWait(false))
}

func createBootableFromImage(ctx context.Context, conn *sdk.Connection) (*blockstorage.CreateVolumeResult, error) {
    image := resource.Name("ubuntu")
    return conn.CreateVolume(ctx,
        blockstorage.CreateVolumeRequest{Size: 20, Image: &image},
        blockstorage.WithCreateVolumeBootable(true))
}

func createWithPolicy(ctx context.Context, conn *sdk.Connection, progress func(int) error) (*blockstorage.CreateVolumeResult, error) {
    timeout, interval := 5*time.Minute, time.Second
    return conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 30},
        blockstorage.WithCreateVolumeSnapshot("snapshot-id"),
        blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{
            "volume_type_id": json.RawMessage(`"type-id"`),
            "is_multiattach": json.RawMessage(`false`),
        }),
        blockstorage.WithCreateVolumeSchedulerHints(map[string]json.RawMessage{
            "same_host": json.RawMessage(`["source-volume-id"]`),
        }),
        blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{
            Timeout: &timeout, PollInterval: &interval,
            ProgressCallback: progress,
        }))
}
```

Python의 실제 대응 API는 cloud helper인 `conn.create_volume`입니다.

```python
volume = conn.create_volume(10, name="data", metadata={"owner": "worker"})
boot_volume = conn.create_volume(20, image="ubuntu", bootable=True)
```

`wait=False`는 생성 접수 뒤 반환하고 `timeout=300`은 available 대기에 300초를 전달합니다. `bootable=False`도 wait를 true로 만들지만 false 액션은 보내지 않습니다. Bootable을 지정하지 않은 경우에만 wait=False가 대기를 끕니다. 이 helper는 Nova 서버를 만들거나 볼륨을 서버에 연결하지 않습니다.

## 입력과 생성 속성

`CreateVolumeRequest.Size`는 생성 JSON에 항상 들어갑니다. 0과 음수도 생략하거나 snapshot 크기로 추론하지 않으며, 서버가 유효성을 판단합니다. 이는 Size0을 `omitempty`로 생략하는 기존 native CreateOpts와 다릅니다. `CreateVolumeRequest.Image`의 ID는 직접 사용하고 Name은 한 번 해석합니다. 선택한 이미지가 Attributes의 imageRef를 덮어씁니다. Glance client는 이미지 Name 해석에만 필요합니다.

`CreateVolumeAttributes`는 자주 쓰는 속성을 concrete 필드로 받습니다.

| 속성 | Go 필드 | 요청 이름 |
|---|---|---|
| 이름·설명 | `Name`, `Description` | `name`, `description` |
| AZ | `AvailabilityZone` | `availability_zone` |
| snapshot | `SnapshotID` | `snapshot_id` |
| 복제 원본 | `SourceVolumeID` | `source_volid` |
| 이미지 | `ImageID` | `imageRef` |
| backup | `BackupID` | `backup_id` |
| 볼륨 type | `VolumeType`, `VolumeTypeID` | `volume_type`, `volume_type_id` |
| group | `GroupID`, `ConsistencyGroupID` | `group_id`, `consistencygroup_id` |
| multiattach | `Multiattach` | `multiattach` |
| 문자열 메타데이터 | `Metadata` | `metadata` |
| scheduler hints | `SchedulerHints` | 본문 최상위 `OS-SCH-HNT:scheduler_hints` |
| 그 외 알려진 속성 | `Fields` | Python 속성 이름 또는 wire key를 매핑 |

Fields는 고정 Python Volume의 직접 Body 속성 34개와 inherited `id`·`name`·`metadata`를 받습니다. `source_volume_id` → `source_volid`, `image_id` → `imageRef`, `is_bootable` → `bootable`, `is_encrypted` → `encrypted`, `is_multiattach` → `multiattach`와 `consistency_group_id` → `consistencygroup_id`를 SDK가 매핑합니다. `host`, `project_id`, migration·replication 속성도 선언된 확장 wire key로 변환합니다. 입력 가능한 알려진 속성을 제한된 native CreateOpts 필드만으로 축소하지 않습니다. 해당 속성의 생성 허용 여부·microversion·권한은 서버가 판단합니다.

같은 속성의 wire key와 Python alias가 Fields에 함께 있으면 wire key가 우선합니다. concrete Attributes의 nonnil 필드는 raw Fields의 해당 canonical 값을 먼저 교체합니다. canonical name·description이 있으면 display_name·display_description보다 우선하며, 선택한 null·false·0·빈 문자열·빈 배열·빈 객체는 이름·설명에서 생략합니다. 필수 Size는 Request의 값으로 고정하고 선택한 Image가 최종 imageRef를 고정합니다.

Fields의 `bootable`은 생성 본문의 속성이며, 완료 대기와 후속 액션을 선택하는 `WithCreateVolumeBootable`과 별도입니다. 이 namespace는 Python 함수의 인자 바인딩을 다시 수행하지 않습니다.

Fields의 알려진 값은 원문 JSON으로 전송합니다. Python Resource의 int·bool·BoolStr·dict/list 변환을 그대로 수행하지 않습니다. JSON false·zero·null·빈 객체·배열과 큰 숫자는 선택한 값대로 보존합니다. 문자열 Metadata는 nil이면 생략하고 nonnil empty map이면 `{}`를 보냅니다. Fields의 `metadata:null`도 보존합니다. SchedulerHints는 객체 또는 null이어야 하며 null·빈 객체는 생략하고 값이 있는 객체만 volume envelope 밖으로 이동합니다. 임의 dict coercion은 하지 않습니다.

모르는 필드나 `base_path`, `microversion`, private Resource 제어 입력은 요청 전에 거부합니다. source·URL·본문 envelope를 kwargs로 바꾸지 않습니다. `SourceReplica`/`source_replica`는 native가 지원하는 명시적인 Go 확장이고, 고정 Python Volume descriptor에서 빠진 입력을 누락한 것으로 판정하지 않습니다.

## 대기와 bootable

Wait의 nil은 true, Bootable의 nil은 선택 없음입니다. Bootable이 nonnil이면 false도 최종 wait를 true로 만듭니다. false는 available까지 기다리고 액션 없이 반환하며, true는 대기 성공 후 고정 ID에 `POST /volumes/{id}/action` 200과 `{"os-set_bootable":{"bootable":true}}`를 보냅니다. 명시적인 false 변경이 필요하면 기존 [Volume SetBootable API](v3/README.md)를 별도로 사용합니다.

`CreateVolumeWaitOpts`는 기존 attachment 대기와 같은 concrete 정책입니다. Timeout의 nil·명시적인 zero는 무제한, PollInterval의 nil은 2초이며 명시적인 zero·음수 interval과 음수 timeout은 사전에 거부합니다. FailureStates의 nil은 정확한 `error`, nonnil empty slice는 실패 검사를 끕니다. 목표 available과 실패는 대소문자를 구별하지 않고 정확하게 비교하며 목표를 먼저 확인합니다. `error_deleting` 같은 접두사를 자동으로 실패로 확대하지 않습니다. 생성 응답의 literal lowercase `error`는 wait=False에서도 즉시 실패입니다.

양수 timeout은 생성 접수의 body·Close·identity·source 검사가 끝난 뒤 대기 단계에서 시작합니다. 대기 HTTP와 timer는 그 context로 취소됩니다. 이후 bootable 액션은 원래 caller context를 사용하며 대기 timeout으로 제한하지 않습니다. 전체 이미지 이름 해석·생성·대기·액션의 시간 제한은 caller context로 지정합니다. ProgressCallback은 nonterminal 관찰에서만 `0`을 받고 오류를 반환해 중단할 수 있습니다.

`WithCreateVolumeOptions`는 전체 설정을, `WithCreateVolumeAttributes`와 `WithCreateVolumeWaitPolicy`는 해당 namespace를 교체합니다. Wait·Bootable helper는 자신의 값만 바꿉니다. 이름·설명·snapshot·type·metadata·hint·Fields helper도 각 필드에 값을 설정하고 소유하므로 caller가 기본 pointer나 builder를 직접 조립할 필요가 없습니다. pointer·map·slice·raw JSON은 옵션을 한 번 적용한 뒤 복사해 실행합니다. `PrepareCreateVolumeOptions(ctx, options...)`는 서비스 선택 없이 원래 callback을 한 번 실행하고 소유한 전체 유효 정책을 반환합니다. 이를 전체 정책 factory로 다시 사용해도 원래 callback은 다시 실행하지 않습니다. Connection은 완전한 옵션을 준비한 후 Cinder와 필요한 Glance를 선택합니다. 직접 package 호출은 제공한 source를 먼저 capture한 뒤 원래 callback을 적용합니다. caller가 보유한 옵션·retained 설정 pointer로 실행 정책을 바꾸지 않습니다.

## 실제 결과와 실패

| `CreateVolumeResult` 필드 | 의미 |
|---|---|
| `VolumeID` | 생성 응답의 canonical ID로 고정한 후속 경로 |
| `Created` | 실제 POST202 접수 응답과 생성 볼륨 모델 |
| `LastAccepted` | 대기에서 마지막으로 받아들인 Cinder GET200 |
| `Ready` | available을 확인한 실제 `VolumeInfo` |
| `BootableSet` | 실제 bootable action POST200 접수 응답 |

Created·LastAccepted는 실제 Body·Header·StatusCode와 `VolumeInfo`를 각각 보존합니다. 생성 응답의 lowercase `id`는 안전한 nonempty 경로 식별자여야 하며 이후 응답이나 Location이 경로를 바꾸지 않습니다. accepted 응답의 read·Close·decode·source·context 오류에서는 raw 증거와 error를 함께 반환합니다. 모델은 전체 decode 후에만 넣으며, canonical 상태·대상 검사가 이후 실패할 수도 있습니다. Ready는 LastAccepted.Volume과 독립적으로 소유합니다.

`VolumeInfo`는 identity·상태·이름·설명·크기·AZ·source/type/group·flag 등의 nullable 필드를 제공합니다. `MetadataFields`는 volume의 `metadata` 객체 값들을 raw JSON으로 유지하고 embedded `Metadata.Body`는 전체 리소스의 알려진·알려지지 않은 필드와 생략·명시적인 null을 보존합니다. timestamps는 literal 문자열이며 큰 숫자는 float64로 바꾸지 않습니다. bootable·encrypted 응답은 JSON bool 또는 대소문자를 구별하지 않는 정확한 true/false 문자열을 받고, 다른 값은 거부합니다. 나머지 typed bool은 JSON bool만 받습니다. 알려진 타입 오류와 잘못된 attachment 배열은 모델을 부분 변경하지 않습니다. 모든 Python 응답 속성을 typed 필드로 제공한다고 주장하지 않습니다.

BootableSet은 opaque body·header·200 접수 증거이며 새로운 볼륨 모델이 아닙니다. 성공한 액션만으로 Ready.IsBootable을 true로 덮어쓰거나 추가 확인 없이 서버 관찰을 합성하지 않습니다. Wait(false)이고 Bootable이 nil이면 LastAccepted·Ready·BootableSet이 없습니다.

오류가 발생해도 result를 함께 확인합니다. nonnil result 자체는 생성 접수를 뜻하지 않습니다. 실제 Created의 202, BootableSet의 200과 독립적인 Ready가 각 단계의 증거입니다. 후속 대기·액션이 거부되면 원래 native 오류의 상태·본문·header를 보존하고 마지막 accepted 관찰을 거부 응답으로 덮어쓰지 않습니다. LastAccepted는 모든 물리적인 전송 시도의 기록이 아닙니다. 자동 DELETE·두 번째 논리적 생성·보상 액션은 수행하지 않습니다. caller가 고정 VolumeID와 보존한 결과로 확인·정리를 결정합니다.

## openstacksdk와의 범위

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [BlockStorageCloudMixin.create_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L175-L232), [이름·설명 alias helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L494-L503), [Volume 속성과 scheduler hint 이동](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54-L144), [hint request preparation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L446-L464)입니다. Gophercloud는 v2.15.0에 고정합니다.

Python은 image 문자열을 strict find_image로 해석하고 image dict의 ID를 직접 읽습니다. Go는 Ref로 ID·Name 선택을 명시하며 ID는 Glance 없이 사용합니다. Python waiter가 이미 available인 Resource에서 cache로 반환할 수 있는 반면 Go는 새 GET을 수행합니다. Python은 mutable Volume을 반환하고 예외에서 접수 결과를 별도로 돌려주지 않으며, Go는 모델·raw phase 결과와 오류를 함께 남깁니다.

Python의 timeout=None은 무제한이지만 [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L102)의 숫자 0·음수는 cached target 단축을 제외하면 첫 poll 전에 timeout입니다. Go는 zero를 무제한으로, 음수를 생성 전 오류로 처리합니다. 양수 Go 대기 context는 HTTP와 timer를 중단하며 Python loop는 fetch 사이에 시간을 검사합니다. 이미지 문자열 heuristic, Python coercion과 insertion order, route/private kwargs 제어 대신 위의 명시적인 Go 정책을 적용합니다.

이 직접 helper의 size·wait·timeout·image·bootable과 알려진 생성 kwargs를 Go 대응 정책으로 제공합니다. `go_mapping`은 위 차이를 기록한 판정이며 별도의 Python Cinder create/wait/set_bootable, Glance find와 inherited Resource/session/cache 선언의 판정을 바꾸지 않습니다. Connection은 Cinder v3를 선택하며 Python의 v2 Resource 객체 모델도 완료했다고 확대하지 않습니다. 실제 quota·backend·source·microversion·권한·생성 가능 여부는 서버가 판단합니다. 이름 해석은 기존 Collection의 native pagination·전체 페이지 decoder 경계를 유지합니다.

[볼륨 연결](attach-volume.md)과 [분리](detach-volume.md)는 별도 workflow입니다. client의 설정은 동시 사용 전에 준비하며 SDK는 capture한 source·고정 method·URL·본문과 원래 provider의 live 인증을 사용합니다. configured native retry·reauthentication은 유지하고 임의 transport 내부 replay나 packet-level exactly-once를 주장하지 않습니다.
