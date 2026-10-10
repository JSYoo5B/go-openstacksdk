# Cinder v3 native snapshot 호출

`snapshots.New(client)`(또는 `service.Snapshots`)의 generated 메서드는 Gophercloud `v2.15.0`의 [v3 snapshots 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/snapshots/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "snapshots"}` 문맥만 더하며, 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. metadata는 [Snapshot metadata](snapshots/README.md), 실패 상태를 감지하는 대기는 SDK의 `WaitForState`·`WaitForAvailable`을 참고합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST snapshots`, `{"snapshot": {...}}` | 202 |
| `Get(ctx, id)` | `GET snapshots/{id}` | 200 |
| `List(ctx, options...)` | `GET snapshots` | native pager 200, 204, 300 |
| `ListDetail(ctx, options...)` | `GET snapshots/detail` | native pager 200, 204, 300 |
| `Update(ctx, id, opts, options...)` | `PUT snapshots/{id}`, `{"snapshot": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE snapshots/{id}` | 202, 204 |

`CreateOpts`의 `VolumeID`는 필수라 비어 있으면 HTTP 전에 오류입니다. `Force`는 bool이라 false는 생략되고 true일 때만 보냅니다. `UpdateOpts`의 이름과 설명은 pointer라 빈 문자열을 보낼 수 있습니다. `With...Field` 확장 필드는 `snapshot` envelope 안에 들어가고 기존 key와 겹치면 HTTP 전에 거부됩니다.

`List`와 `ListDetail`은 같은 `ListOpts`를 각각 요약 경로와 상세 경로로 보내며 둘 다 `snapshots_links`의 next href를 따라갑니다. `AllTenants`는 `all_tenants=true`로 보내고 기본 정책에서 관리자에게만 의미가 있습니다.

응답의 진행률과 프로젝트는 `os-extended-snapshot-attributes:` 접두사 key에서 읽습니다. 시각은 볼륨과 같은 시간대 없는 형식만 받아서 끝에 `Z`가 붙으면 decode 오류입니다. 볼륨과 달리 snapshot 응답은 일반 envelope decode라 `snapshot` key가 없거나 null이면 오류 없이 nil을 돌려줍니다.

`WaitForStatus(ctx, id, status)`는 즉시 한 번, 이후 1초 간격으로 `Get`을 반복하며 대소문자를 구분해 정확히 같은 상태에서 멈춥니다. `Get` 오류와 호출자 `ctx` deadline에서도 멈춥니다. 다만 응답에 `snapshot` key가 없으면 native 구현이 nil snapshot의 상태를 읽다가 **panic**합니다. 이 경로는 아직 SDK가 막지 않으므로 신뢰할 수 없는 endpoint에서는 SDK의 `WaitForState`·`WaitForAvailable`을 사용합니다.

## 관리자 action

기본 정책상 관리자 호출인 세 action은 모두 `POST snapshots/{id}/action`에 보내고 202만 받습니다. `ForceDelete(ctx, id)`는 `{"os-force_delete": {}}`, `ResetStatus(ctx, id, opts, options...)`는 `{"os-reset_status": {"status": ...}}`, `UpdateStatus(ctx, id, opts, options...)`는 `{"os-update_snapshot_status": {"status": ..., "progress": ...}}`를 보냅니다. 두 상태 action의 `Status`는 omitempty가 없어 빈 값도 보내고 `Progress`는 비면 생략합니다.
