# 볼륨 연결 native 호출

`volumeattach.New(client)`(또는 `service.VolumeAttachments`)의 generated `Create/Get/List/Delete`는 Gophercloud `v2.15.0`의 [volumeattach 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/volumeattach/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "volumeattach"}` 문맥만 더하며 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. 서버 ID와 볼륨 ID는 escape 없이 경로에 이어 붙입니다.

| 메서드 | 요청 | 성공 status | 반환 |
|---|---|---|---|
| `Create(ctx, serverID, opts, options...)` | `POST servers/{id}/os-volume_attachments`, `{"volumeAttachment": {...}}` | 200 | 응답의 `volumeAttachment` |
| `Get(ctx, serverID, volumeID)` | `GET servers/{id}/os-volume_attachments/{volumeID}` | 200 | 응답의 `volumeAttachment` |
| `List(ctx, serverID)` | `GET servers/{id}/os-volume_attachments` | native pager 200, 204, 300 | `volumeAttachments` 행 |
| `Delete(ctx, serverID, volumeID)` | `DELETE servers/{id}/os-volume_attachments/{volumeID}` | 202, 204 | error |

`CreateOpts.VolumeID`는 필수이며 비어 있으면 HTTP 전에 오류입니다. 빈 `Device`·`Tag`와 false인 `DeleteOnTermination`은 생략하므로 false를 명시해 보낼 수 없습니다. `WithCreateField` 확장 필드는 `volumeAttachment` envelope 안에 들어가고 `volumeId` 같은 기존 key와 겹치거나 nil 옵션이면 HTTP 전에 거부됩니다. 응답에 `tag`나 `delete_on_termination`이 없으면 해당 pointer는 nil입니다.

`List`는 단일 페이지로 다루므로 `volumeAttachments_links`가 있어도 다음 페이지를 요청하지 않습니다. 빈 목록은 아무 값도 내보내지 않고, 본문 없는 204는 `io.EOF` 오류 하나로 끝납니다. 목록 오류는 operation 문맥 없이 native 오류 그대로 전달됩니다. 연결·해제의 완료 대기는 이 호출에 포함되지 않습니다.
