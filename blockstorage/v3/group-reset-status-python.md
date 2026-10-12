# Group과 group snapshot 상태 재설정: Python과 Go

`groups.API.ResetStatusRecord`와 `groupsnapshots.API.ResetStatusRecord`는 pinned openstacksdk의 상태 재설정 Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0에는 두 `reset_status` 동작의 바인딩이 없으므로 SDK가 요청을 직접 보냅니다. 기준 소스는 [`_proxy.py` reset_group_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2250-L2272), [`reset_group_snapshot_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2756-L2786), [`group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py)와 [`group_snapshot.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_snapshot.py)입니다.

| Python `conn.block_storage` | Go | 요청 |
|---|---|---|
| `reset_group_status(group, status)` | `groups.API.ResetStatusRecord(ctx, id, status, options...)` | `POST groups/{id}/action` |
| `reset_group_state(group, status)` (deprecated) | 위와 같음 | 위와 같음 |
| `reset_group_snapshot_status(group_snapshot, status)` | `groupsnapshots.API.ResetStatusRecord(ctx, id, status, options...)` | `POST group_snapshots/{id}/action` |
| `reset_group_snapshot_state(group_snapshot, state)` (deprecated) | 위와 같음 | 위와 같음 |

```python
conn.block_storage.reset_group_status(group_id, "available")
conn.block_storage.reset_group_snapshot_status(snapshot_id, "error")
```

```go
if _, err := groups.New(volumeClient).ResetStatusRecord(ctx, groupID, "available"); err != nil { return err }
if _, err := groupsnapshots.New(volumeClient).ResetStatusRecord(ctx, snapshotID, "error"); err != nil { return err }
```

## 요청

두 메서드 모두 `{"reset_status": {"status": status}}`를 보내며 status 문자열은 검사하지 않습니다. deprecated 별칭은 같은 요청을 보내므로 Go에는 메서드가 하나뿐이고, Python의 deprecated 경고는 재현하지 않습니다.

Group은 `Group._get_microversion`을 따릅니다. client에 microversion이 설정되어 있으면 discovery 없이 그 값을, 없으면 Cinder version 문서를 읽어 3.38과 서버 최대값 중 작은 값을 보냅니다. Group snapshot은 `GroupSnapshot._action`을 따라 같은 규칙에 상한 3.29를 씁니다. `WithRecordResetStatusMicroversion`으로 microversion을 직접 정할 수 있지만 이는 Go 확장입니다.

Python `GroupSnapshot._action`은 `Accept` 헤더를 빈 문자열로 보냅니다. Go도 group snapshot 요청에 빈 `Accept` 헤더를 보내며, 이때 client나 옵션에 지정한 `Accept`는 Python처럼 이 값으로 대체됩니다. Group 요청에는 Gophercloud 기본값인 `Accept: application/json`이 붙습니다.

## 결과와 오류

Python은 None을 돌려주지만 Go는 요청한 ID(`Reference`), 실제 응답 본문, 헤더, status와 discovery 응답을 담은 `ResetStatusRecordResult`를 돌려줍니다. Group 쪽 결과 타입은 replication 동작의 `ReplicationRecordResult`와 같습니다. Python `raise_from_response`처럼 400 미만은 모두 성공이고, 400 이상은 Gophercloud `ErrUnexpectedResponseCode`를 `resource.OperationError`(Operation `ResetStatusRecord`, Resource `groups` 또는 `group_snapshots`)로 감싼 오류입니다. 이때도 결과에는 실제 거절 응답이 남습니다.

## Python과 다른 점

Python은 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. Go는 ID를 URL segment 하나로 escape하고 빈 값, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. 헤더 옵션은 Go 확장이며 인증과 microversion 헤더는 SDK가 소유합니다.
