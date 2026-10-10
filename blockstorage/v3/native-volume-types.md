# Cinder v3 native volume type 조회

`service.VolumeTypes`(`blockstorage/v3/volumetypes`)의 generated 조회 메서드는 Gophercloud `v2.15.0`의 [v3 volumetypes 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "volumetypes"}` 문맥만 더합니다. type 생성·수정·삭제, extra spec 쓰기, 암호화 설정, 비공개 type 접근 관리는 기본 정책상 관리자 호출이라 이 문서에서 다루지 않습니다.

| 메서드 | 요청 | 결과 | 성공 status |
|---|---|---|---|
| `Get(ctx, id)` | `GET types/{id}` | `*VolumeType` | 200 |
| `List(ctx, options...)` | `GET types?is_public=...` | `VolumeType` stream | native pager 200, 204, 300 |
| `ListExtraSpecs(ctx, id)` | `GET types/{id}/extra_specs` | `map[string]string` | 200 |
| `GetExtraSpec(ctx, id, key)` | `GET types/{id}/extra_specs/{key}` | `map[string]string` | 200 |

`List`는 `ListOpts.IsPublic`이 비어 있으면 `is_public=None`을 항상 보내 공개·비공개 type을 모두 요청합니다. `VisibilityPublic`·`VisibilityPrivate`는 `true`·`false`를 보냅니다. 페이지 링크는 Gophercloud가 단수형 `volume_type_links` key에서만 읽으므로, 서버가 복수형 `volume_types_links`로 링크를 주면 다음 페이지를 요청하지 않습니다.

`Get` 응답은 `volume_type` key를 직접 찾아 `{}`·null을 빈 type으로 돌려줍니다. `is_public`과 `os-volume-type-access:is_public`은 각각 `IsPublic`·`PublicAccess`로 decode합니다. `ListExtraSpecs`는 `extra_specs` 객체를 돌려주고 key가 없으면 nil map입니다. `GetExtraSpec`은 envelope 없는 `{"key": "value"}` 본문 전체를 map으로 돌려주며, 값이 문자열이 아니면 decode 오류입니다. `key`는 escape 없이 경로에 붙습니다.
