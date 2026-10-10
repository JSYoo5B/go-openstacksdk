# Cinder v3 native 볼륨 호출

`volumes.New(client)`(또는 `service.Volumes`)의 generated 메서드는 Gophercloud `v2.15.0`의 [v3 volumes 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "volumes"}` 문맥만 더하며, 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID는 escape 없이 경로 segment로 이어 붙입니다. 볼륨 생성·삭제 대기, metadata, 크기 변경처럼 SDK가 소유한 상위 흐름은 [Block Storage 사용법](../README.md)을 참고합니다.

## 생성·조회·목록·수정·삭제

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST volumes`, `{"volume": {...}}` | 202 |
| `Get(ctx, id)` | `GET volumes/{id}` | 200 |
| `List(ctx, options...)` | `GET volumes/detail` | native pager 200, 204, 300 |
| `Update(ctx, id, opts, options...)` | `PUT volumes/{id}`, `{"volume": {...}}` | 200 |
| `Delete(ctx, id, options...)` | `DELETE volumes/{id}` | 202, 204 |

`Create`는 202만 성공으로 보며, 다른 Block Storage 생성과 달리 201도 오류입니다. `CreateOpts`의 빈 값은 모두 생략하고 `ImageID`는 `imageRef` key로 보냅니다. `WithCreateField`의 확장 필드는 `volume` envelope 안에 들어가고 기존 key와 겹치면 HTTP 전에 거부됩니다. `WithCreateHintOpts`의 scheduler hints는 envelope 밖 최상위 `OS-SCH-HNT:scheduler_hints`로 보냅니다. `DifferentHost`·`SameHost`·`LocalToInstance`는 UUID 형식이 아니면 HTTP 전에 오류이고, 모든 hint가 비면 key 자체를 보내지 않습니다.

`UpdateOpts`의 이름과 설명은 pointer라 빈 문자열을 보낼 수 있고, `Metadata`는 비어 있으면 생략합니다. `Delete`의 `DeleteOpts{Cascade: true}`는 `cascade=true` query를 보내며 false는 생략합니다. `WithDeleteQuery`로 다른 query를 더할 수 있습니다.

목록은 상세 경로 `volumes/detail`을 쓰고 `volumes_links`의 next href를 따라갑니다. `Metadata` 필터는 `{'k':'v'}`처럼 Python dict 모양 문자열 하나로 보내며 여러 key의 순서는 Go map 순회 순서를 따릅니다. `AllTenants`는 `all_tenants=true`, `Bootable` pointer는 false도 보냅니다. `all_tenants`는 기본 정책에서 관리자에게만 의미가 있습니다.

응답은 `volume` key를 직접 찾아 decode합니다. 본문이 `{}`이거나 값이 null이면 오류 없이 빈 볼륨을 돌려주고, 다른 key만 있으면 오류입니다. 생성·수정 시각과 attachment의 `attached_at`은 시간대 없는 `2006-01-02T15:04:05.999999` 형식만 받아서 끝에 `Z`가 붙으면 decode 오류입니다. null 시각은 zero time입니다. `bootable`은 `"true"`/`"false"` 문자열 그대로이고 null `backup_id`는 nil pointer입니다.

## 사용자 action

아래 action은 모두 `POST volumes/{id}/action`에 action 이름 하나를 key로 둔 본문을 보냅니다. `With{Action}Field` 확장 필드는 그 action 객체 안에 들어가며 기존 key와 겹치거나 nil 옵션이면 HTTP 전에 거부됩니다.

| 메서드 | 본문 | 결과 | 성공 status |
|---|---|---|---|
| `Attach(ctx, id, opts, options...)` | `{"os-attach": {...}}` | 없음 | 202 |
| `BeginDetaching(ctx, id)` | `{"os-begin_detaching": {}}` | 없음 | 202 |
| `Detach(ctx, id, opts, options...)` | `{"os-detach": {...}}` | 없음 | 202 |
| `Reserve(ctx, id)`·`Unreserve(ctx, id)` | `{"os-reserve": {}}`·`{"os-unreserve": {}}` | 없음 | 200, 201, 202 |
| `InitializeConnection(ctx, id, opts, options...)` | `{"os-initialize_connection": {"connector": {...}}}` | `connection_info` map | 200, 201, 202 |
| `TerminateConnection(ctx, id, opts, options...)` | `{"os-terminate_connection": {"connector": {...}}}` | 없음 | 202 |
| `ExtendSize(ctx, id, opts, options...)` | `{"os-extend": {"new_size": N}}` | 없음 | 202 |
| `UploadImage(ctx, id, opts, options...)` | `{"os-volume_upload_image": {...}}` | `VolumeImage` | 202 |
| `SetImageMetadata(ctx, id, opts, options...)` | `{"os-set_image_metadata": {"metadata": ...}}` | 없음 | 200 |
| `SetBootable(ctx, id, opts, options...)` | `{"os-set_bootable": {"bootable": ...}}` | 없음 | 200 |
| `ChangeType(ctx, id, opts, options...)` | `{"os-retype": {...}}` | 없음 | 202 |
| `ReImage(ctx, id, opts, options...)` | `{"os-reimage": {...}}` | 없음 | 202 |

connection 두 호출은 connector를 `connector` 안에 한 번 더 감싸므로 확장 필드는 `connector` 옆에 붙습니다. `Multipath`는 pointer라 false도 보내고 빈 connector는 `{}`입니다. `InitializeConnection`은 응답의 `connection_info` 객체를 그대로 map으로 돌려주며, key가 없으면 nil map과 nil 오류입니다.

`ExtendSize`의 `NewSize`와 `ChangeType`의 `NewType`은 필수 tag가 있지만 native 검사는 빈 문자열만 거부하고 정수 0은 그대로 보냅니다. `SetImageMetadata`의 nil metadata는 `null`, `SetBootable`의 false와 `ReImage`의 빈 image ID·false 예약 플래그도 항상 보냅니다. `Detach`의 빈 attachment ID는 빈 action 객체가 됩니다.

`UploadImage`는 `os-volume_upload_image` 객체를 `VolumeImage`로 decode하며, key가 없으면 오류 없이 zero 값을 돌려줍니다. 응답 시각은 볼륨과 같은 시간대 없는 형식만 받습니다. force delete, reset status, unmanage는 기본 정책상 관리자 action이라 이 절에서 다루지 않습니다.

## 상태 대기

`WaitForStatus(ctx, id, status)`는 먼저 `Get`을 한 번 보내고, 상태가 정확히 같지 않으면 1초 간격으로 다시 조회합니다. 상태 비교는 대소문자를 구분합니다. `Get` 오류는 즉시 반환하고, timeout은 호출자의 `ctx` deadline으로만 정합니다. 오류 상태(`error`)를 만나도 멈추지 않으므로 실패 상태 감지와 기본 timeout이 필요하면 SDK의 [서비스 대기](../../docs/service-waits.md)를 사용합니다.
