# 인터페이스 연결 native 호출

`attachinterfaces.New(client)`(또는 `service.AttachInterfaces`)의 generated `Create/Get/List/Delete`는 Gophercloud `v2.15.0`의 [attachinterfaces 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/attachinterfaces/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "attachinterfaces"}` 문맥만 더하며 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. 서버 ID와 port ID는 escape 없이 경로에 이어 붙입니다.

| 메서드 | 요청 | 성공 status | 반환 |
|---|---|---|---|
| `Create(ctx, serverID, opts, options...)` | `POST servers/{id}/os-interface`, `{"interfaceAttachment": {...}}` | 200 | 응답의 `interfaceAttachment` |
| `Get(ctx, serverID, portID)` | `GET servers/{id}/os-interface/{portID}` | 200 | 응답의 `interfaceAttachment` |
| `List(ctx, serverID)` | `GET servers/{id}/os-interface` | native pager 200, 204, 300 | `interfaceAttachments` 행 |
| `Delete(ctx, serverID, portID)` | `DELETE servers/{id}/os-interface/{portID}` | 202, 204 | error |

`CreateOpts`에는 필수 필드가 없습니다. 모두 비우면 `{"interfaceAttachment": {}}`를 보내 Nova가 네트워크를 고르게 합니다. `PortID`와 `NetworkID`를 함께 줄 수 없다는 제약은 SDK가 검사하지 않고 Nova가 검증합니다. `FixedIPs`의 빈 `SubnetID`는 생략합니다. `tag` 같은 추가 필드는 `WithCreateField`로 `interfaceAttachment` envelope 안에 넣으며, `port_id` 같은 기존 key와 겹치거나 nil 옵션이면 HTTP 전에 거부됩니다.

`List`는 단일 페이지로 다루므로 `interfaceAttachments_links`가 있어도 다음 페이지를 요청하지 않습니다. 빈 목록은 아무 값도 내보내지 않고, 본문 없는 204는 `io.EOF` 오류 하나로 끝납니다. 목록 오류는 operation 문맥 없이 native 오류 그대로 전달됩니다. 연결·해제의 완료 대기는 이 호출에 포함되지 않습니다.
