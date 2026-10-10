# Cinder v3 native QoS spec 호출

`service.QoS`(`blockstorage/v3/qos`)의 generated 메서드는 Gophercloud `v2.15.0`의 [qos 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/qos/requests.go)을 바꾸지 않고 호출합니다. QoS spec 관리는 기본 Cinder 정책상 관리자 호출입니다. SDK는 오류에 `resource.OperationError{Resource: "qos"}` 문맥만 더합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST qos-specs`, `{"qos_specs": {...}}` | 200 |
| `Get(ctx, id)` | `GET qos-specs/{id}` | 200 |
| `List(ctx, options...)` | `GET qos-specs` | native pager |
| `Update(ctx, id, opts, options...)` | `PUT qos-specs/{id}` | 200 |
| `Delete(ctx, id, options...)` | `DELETE qos-specs/{id}` | 202, 204 |
| `DeleteKeys(ctx, id, options...)` | `PUT qos-specs/{id}/delete_keys`, `{"keys": [...]}` | 202 |
| `Associate(ctx, id, opts, options...)` | `GET qos-specs/{id}/associate?vol_type_id=` | 202 |
| `Disassociate(ctx, id, opts, options...)` | `GET qos-specs/{id}/disassociate?vol_type_id=` | 202 |
| `DisassociateAll(ctx, id)` | `GET qos-specs/{id}/disassociate_all` | 202 |
| `ListAssociations(ctx, id)` | `GET qos-specs/{id}/associations` | 한 페이지 |

`CreateOpts.Specs`와 `UpdateOpts.Specs`의 key·값은 `qos_specs` 객체에 바로 펼쳐 넣는데, typed 필드를 만든 뒤에 덮어쓰므로 `name` 같은 key가 있으면 `Name`보다 우선합니다. `Name`은 omitempty가 없어 빈 이름도 보냅니다. `Update`의 결과는 QoS 객체가 아니라 서버가 돌려준 spec map(`map[string]string`)입니다.

`DeleteKeys`는 `WithDeleteKeysOptions`로 지운 key 목록을 주며, 목록 없이 호출하면 `{"keys": null}`을 보냅니다. association 변경 세 호출은 상태를 바꾸지만 HTTP GET이고, `VolumeTypeID`는 필수라 비어 있으면 HTTP 전에 오류입니다. `Delete`의 `DeleteOpts{Force: true}`는 `force=true` query를 보냅니다. 목록은 `qos_specs_links`의 next href를 따라가고 association 목록은 한 페이지만 읽습니다.
