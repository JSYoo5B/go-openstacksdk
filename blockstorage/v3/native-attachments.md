# Cinder v3 native attachment 호출

`attachments.New(client)`(또는 `service.Attachments`)의 generated 메서드는 Gophercloud `v2.15.0`의 [v3 attachments 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/attachments/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "attachments"}` 문맥만 더합니다. attachment API는 Cinder microversion 3.27 이상, `Complete`는 3.44 이상이 필요하지만 native 호출은 microversion을 고르지 않습니다. 호출자가 client의 `Microversion`을 정하면 `OpenStack-API-Version: volume X.Y` header로 보냅니다. 서버 연결 흐름 전체는 [Cinder attachment 사용법](../cinder-volume-attachment.md)을 참고합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST attachments`, `{"attachment": {...}}` | 200, 202 |
| `Get(ctx, id)` | `GET attachments/{id}` | 200 |
| `List(ctx, options...)` | `GET attachments/detail` | native pager 200, 204, 300 |
| `Update(ctx, id, opts, options...)` | `PUT attachments/{id}`, `{"attachment": {"connector": ...}}` | 200 |
| `Complete(ctx, id)` | `POST attachments/{id}/action`, `{"os-complete": null}` | 204 |
| `Delete(ctx, id)` | `DELETE attachments/{id}` | 200 |

`CreateOpts`의 `VolumeUUID`와 `InstanceUUID`는 omitempty와 필수 검사가 모두 없어 비어 있어도 `""`로 보냅니다. `Connector`와 `Mode`는 비면 생략합니다. `UpdateOpts.Connector`는 omitempty가 없어 nil이면 `"connector": null`입니다. `Delete`는 일반 DELETE와 달리 200만 받고 202·204는 오류입니다.

목록은 상세 경로 `attachments/detail`을 쓰지만 Gophercloud의 page가 `NextPageURL`을 정의하지 않아 `attachments_links` 배열을 읽지 않고 `{"links": {"next": "..."}}` 문자열만 따라갑니다. 그래서 Cinder 응답에서는 첫 페이지만 반환합니다. `AllTenants`는 `all_tenants=true`를 보냅니다.

응답은 `attachment` key를 직접 찾아 `{}`·null을 빈 attachment로 돌려주고 다른 key만 있으면 오류입니다. `connection_info`는 raw map이고 시각은 시간대 없는 형식만 받습니다. `WaitForStatus(ctx, id, status)`는 즉시 한 번, 이후 1초 간격으로 조회해 정확히 같은 상태에서 멈춥니다. envelope가 없는 응답은 빈 상태로 보고 deadline까지 계속 조회하며, Get 오류는 즉시 반환합니다.
