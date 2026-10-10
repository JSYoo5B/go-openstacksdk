# Cinder native 관리자 호출

이 문서는 Gophercloud `v2.15.0`의 Cinder 관리 호출을 그대로 감싼 generated 메서드를 설명합니다. 대상은 v3 volume type 관리, 기존 볼륨 가져오기(manage), v2·v3 scheduler pool과 service 목록, v2 quota와 limit입니다. SDK는 오류에 `resource.OperationError` 문맥을 더하고 `With...Field` 확장 필드를 합칠 뿐이며, native URL 구성·고정 status·paging·decode는 바꾸지 않습니다. 목록 stream의 page 오류에는 operation 문맥이 붙지 않습니다. 경로는 client `ResourceBase`(예: `.../v3/{project}/`) 아래에 붙습니다.

v2 quota 조회와 limit 조회를 제외하면 모두 기본 Cinder 정책상 관리자 호출입니다. 조회 전용 volume type 메서드는 [volume type 조회](native-volume-types.md)에서 다룹니다.

## volume type 관리

`service.VolumeTypes`(`blockstorage/v3/volumetypes`)의 관리 메서드는 [v3 volumetypes 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/requests.go)을 호출합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST types`, `{"volume_type": {...}}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT types/{id}`, `{"volume_type": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE types/{id}` | 202, 204 |
| `CreateExtraSpecs(ctx, id, specs, options...)` | `POST types/{id}/extra_specs`, `{"extra_specs": {...}}` | 200 |
| `UpdateExtraSpec(ctx, id, spec, options...)` | `PUT types/{id}/extra_specs/{key}`, `{"key": "value"}` | 200 |
| `DeleteExtraSpec(ctx, id, key)` | `DELETE types/{id}/extra_specs/{key}` | 202 |
| `AddAccess(ctx, id, opts, options...)` | `POST types/{id}/action`, `{"addProjectAccess": {"project": ...}}` | 202 |
| `RemoveAccess(ctx, id, opts, options...)` | `POST types/{id}/action`, `{"removeProjectAccess": {"project": ...}}` | 202 |
| `ListAccesses(ctx, id)` | `GET types/{id}/os-volume-type-access` | native pager 200, 204, 300 |
| `CreateEncryption(ctx, id, opts, options...)` | `POST types/{id}/encryption`, `{"encryption": {...}}` | 200 |
| `GetEncryption(ctx, id)` | `GET types/{id}/encryption` | 200 |
| `GetEncryptionSpec(ctx, id, key)` | `GET types/{id}/encryption/{key}` | 200 |
| `UpdateEncryption(ctx, id, encryptionID, opts, options...)` | `PUT types/{id}/encryption/{encryptionID}`, `{"encryption": {...}}` | 200 |
| `DeleteEncryption(ctx, id, encryptionID)` | `DELETE types/{id}/encryption/{encryptionID}` | 202, 204 |

### 입력 규칙

`CreateOpts.Name`은 필수라 비어 있으면 HTTP 전에 오류입니다. 생성의 공개 여부는 `os-volume-type-access:is_public` key로 보내지만, `UpdateOpts.IsPublic`은 `is_public` key로 보냅니다. 수정 필드는 모두 pointer라 nil이면 생략하고 빈 문자열 pointer는 그대로 보냅니다. 그래서 빈 `UpdateOpts`는 `{"volume_type": {}}`입니다.

`CreateExtraSpecs`의 nil map은 `{"extra_specs": null}`로 나갑니다. 이 envelope 값은 문자열 map이라 `WithCreateExtraSpecsField` 확장 필드는 `extra_specs` 안이 아니라 그 옆 최상위에 붙고, `extra_specs` 이름의 확장은 충돌 오류입니다. `UpdateExtraSpec`은 key와 value 한 쌍만 받습니다. 비어 있거나 두 쌍 이상이면 native 입력 오류이고, 경로의 key도 그 한 쌍에서 정합니다. 이 호출은 확장 필드를 받지 않으므로 같은 옵션 타입인 `WithCreateExtraSpecsField`를 넘기면 HTTP 전에 거부합니다. extra spec key와 `GetEncryptionSpec` key는 escape 없이 경로에 붙습니다.

`AddAccessOpts`·`RemoveAccessOpts`의 `Project`에는 필수 표시가 없어 빈 값도 `{"project": ""}`로 보냅니다. `CreateEncryptionOpts.Provider`는 필수지만 `KeySize`·`ControlLocation`·`Cipher`에는 omitempty가 없어 0과 빈 문자열도 그대로 보냅니다. `UpdateEncryptionOpts`는 필수 필드도 omitempty도 없으므로 바꾸지 않을 값까지 네 필드를 모두 채워야 합니다.

확장 필드는 typed 옵션에 선언된 key와 겹치면, 그 필드를 생략했더라도 HTTP 전에 거부합니다. nil 옵션도 HTTP 전에 오류입니다.

### decode

생성·수정 응답은 `volume_type` key를 직접 찾고, 응답이 `{}`이면 빈 type을 돌려줍니다. `CreateExtraSpecs`는 `extra_specs` 문자열 map을 돌려주고 key가 없으면 nil입니다. `UpdateExtraSpec`은 envelope 없는 `{"key": "value"}` 전체를 map으로 돌려주며 값이 문자열이 아니면 decode 오류입니다.

`ListAccesses`는 한 페이지만 읽고 `volume_type_access` 배열을 돌려줍니다. 응답에 링크가 있어도 따라가지 않습니다. 본문 없는 204는 `io.EOF` 오류이고, 200·204·300 밖의 status는 operation 문맥 없는 native 오류입니다.

`CreateEncryption`·`UpdateEncryption`은 `encryption` key 안의 `EncryptionType`을 돌려줍니다. 반면 `GetEncryption`은 envelope 없는 본문을 `GetEncryptionType`으로 읽으므로, 암호화가 없는 type의 `{}` 응답은 빈 값입니다. 이 결과의 `created_at`·`updated_at`·`deleted_at`은 시각으로 해석하지 않은 문자열이고 null은 빈 문자열입니다. `GetEncryptionSpec`은 본문 전체를 `map[string]any`로 돌려주므로 숫자는 `float64`입니다.

## 기존 볼륨 가져오기

`service.ManageableVolumes.ManageExisting(ctx, opts, options...)`는 [manageablevolumes 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/manageablevolumes/requests.go)대로 `POST manageable_volumes`에 `{"volume": {...}}`를 보내고 202만 성공으로 받습니다. Cinder는 이 경로를 microversion 3.8에서 추가했으므로 client에 3.8 이상을 설정해야 합니다.

모든 필드에 omitempty가 있고 필수 표시가 없습니다. Cinder는 `host`나 `cluster` 중 하나와 `ref`를 요구하지만 native builder가 검사하지 않으므로 빈 옵션도 `{"volume": {}}`로 전송됩니다. `Bootable`은 true일 때만 보냅니다. 응답은 `volume` key의 v3 `volumes.Volume`이며, 생성·수정 시각은 시간대 없는 형식만 받아서 끝에 `Z`가 붙으면 decode 오류입니다.

## scheduler pool과 service 목록

`service.SchedulerStats.List`와 `service.Services.List`는 v2·v3 package가 같은 구현입니다. 소스는 [schedulerstats](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/schedulerstats/requests.go)와 [services](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/services/requests.go)입니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `SchedulerStats.List(ctx, options...)` | `GET scheduler-stats/get_pools?detail=...&tenant_id=...` | native pager 200, 204, 300 |
| `Services.List(ctx, options...)` | `GET os-services?binary=...&host=...` | native pager 200, 204, 300 |

두 목록 모두 한 페이지만 읽고 `pools_links`·`services_links`를 따라가지 않습니다. `ListOpts.Detail`은 bool이라 false이면 `detail` query를 생략하고, 서버 기본값인 이름만 있는 pool 목록을 받습니다. 추가 filter는 `WithListQuery`로 더합니다.

pool의 `free_capacity_gb`·`total_capacity_gb`·`allocated_capacity_gb`는 숫자일 때만 그대로 읽습니다. 문자열 `infinite`는 양의 무한대, `unknown`이나 숫자 문자열은 0이 됩니다. `max_over_subscription_ratio`는 숫자든 문자열이든 문자열로 바꿉니다(예: `20.0`은 `"20"`). 다른 capability 필드는 선언된 타입과 다르면 decode 오류입니다.

service의 `updated_at`은 시간대 없는 millisecond 형식만 받으므로 `Z`가 붙으면 decode 오류이고, null은 0 시각입니다.

## v2 quota와 limit

Cinder v2는 deprecated legacy API입니다. `service.QuotaSets`(`blockstorage/v2/quotasets`)와 `service.Limits`(`blockstorage/v2/limits`)는 [v2 quotasets](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v2/quotasets/requests.go)와 [v2 limits](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v2/limits/requests.go)를 직접 호출합니다. v3에는 project 이름 해석과 원문 보존을 더한 [SDK 소유 quota](quotasets/README.md)·[limit](limits/README.md) 호출이 따로 있습니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `QuotaSets.Get(ctx, projectID)` | `GET os-quota-sets/{projectID}` | 200 |
| `QuotaSets.GetDefaults(ctx, projectID)` | `GET os-quota-sets/{projectID}/defaults` | 200 |
| `QuotaSets.GetUsage(ctx, projectID)` | `GET os-quota-sets/{projectID}?usage=true` | 200 |
| `QuotaSets.Update(ctx, projectID, opts, options...)` | `PUT os-quota-sets/{projectID}`, `{"quota_set": {...}}` | 200 |
| `QuotaSets.Delete(ctx, projectID)` | `DELETE os-quota-sets/{projectID}` | 200 |
| `Limits.Get(ctx)` | `GET limits` | 200 |

quota 조회는 해당 project 사용자도 할 수 있지만 수정과 초기화(`Delete`)는 관리자 호출입니다. `Delete`는 다른 삭제와 달리 200만 성공으로 받습니다. limit 조회는 현재 token의 project 기준이라 관리자 권한이 필요 없습니다.

`UpdateOpts`의 한도 필드는 pointer라 nil이면 생략하고 0과 -1은 그대로 보냅니다. `Force`는 true일 때만 보냅니다. `Extra` map은 typed 필드를 만든 다음 `quota_set` 안에 합치므로 type별 key(`volumes_ssd` 등)를 넣을 수 있고, typed 필드와 같은 key면 `Extra` 값이 이깁니다. `WithUpdateField` 확장은 typed key(생략된 것 포함)나 `Extra`에 이미 있는 key와 겹치면 HTTP 전에 거부합니다.

`Get`·`GetDefaults`·`Update`는 `quota_set` 안에서 typed 필드를 읽고, 나머지 key는 `Extra`에 JSON 숫자(`float64`) 그대로 남깁니다. `quota_set` key가 없으면 오류 없이 nil을 돌려줍니다. `GetUsage`는 pointer가 아닌 `QuotaUsageSet` 값을 돌려주며 type별 사용량 key는 담을 곳이 없어 버립니다. envelope가 없으면 빈 값입니다.

`Limits.Get`은 `limits` key의 `absolute` 열 개 정수 필드와 `rate` 목록을 읽습니다. `next-available`은 시각으로 해석하지 않은 문자열입니다. `limits` key가 없으면 오류 없이 nil입니다.
