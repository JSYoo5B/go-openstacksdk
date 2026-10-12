# Cinder default volume type

이 패키지는 pinned openstacksdk의 project 기본 volume type Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0에는 `default-types` 바인딩이 없으므로 이 패키지가 요청을 SDK에서 직접 보냅니다. 기준 소스는 [`_proxy.py` Default Types 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L738-L818)과 [`default_type.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/default_type.py)입니다.

| Python `conn.block_storage` | Go `defaulttypes.API` | 요청 |
|---|---|---|
| `default_types()` | `List(ctx, options...)`, `All(ctx, options...)` | `GET default-types` |
| `show_default_type(project)` | `Get(ctx, projectID, options...)` | `GET default-types/{project_id}` |
| `set_default_type(project, type)` | `Set(ctx, projectID, volumeTypeID, options...)` | `PUT default-types/{project_id}` |
| `unset_default_type(project)` | `Unset(ctx, projectID, options...)` | `DELETE default-types/{project_id}` |

```python
conn.block_storage.set_default_type(project_id, volume_type_id)
default = conn.block_storage.show_default_type(project_id)
conn.block_storage.unset_default_type(project_id)
```

```go
api := defaulttypes.New(volumeClient)
if _, err := api.Set(ctx, projectID, volumeTypeID); err != nil { return err }
dt, err := api.Get(ctx, projectID)
if err != nil { return err }
fmt.Println(*dt.VolumeTypeID)
err = api.Unset(ctx, projectID) // 기본 type이 없어도 오류가 아닙니다
```

## Microversion

Python의 네 메서드는 먼저 `utils.supports_microversion(3.67)`을 확인하고 거짓이면 요청 없이 예외를 냅니다. Go도 매 호출마다 Cinder version 문서를 읽어 서버가 3.67을 지원하는지 보고, client에 microversion이 설정되어 있으면 그 값이 3.67 이상인지도 확인합니다. 조건이 맞지 않으면 `resource.ErrUnsupported`로 실패하고 default-types 요청은 보내지 않습니다. 통과하면 `DefaultType._get_microversion`처럼 설정된 microversion을, 없으면 3.67과 서버 최대값 중 작은 값을 `OpenStack-API-Version: volume X`와 `X-OpenStack-Volume-API-Version: X`로 보냅니다. 실제 discovery 응답은 결과의 `Discovery`에 남습니다.

## 요청과 응답

`Set`은 Python `_create(DefaultType, id=project, volume_type_id=type)`과 같이 `PUT default-types/{project_id}`에 `{"default_type": {"volume_type": type}}`을 보냅니다. Python `_prepare_request_body`는 값이 비어 있을 때 `volume_type_id`를 `volume_type`으로 바꾸지 않으므로, Go도 빈 volume type ID는 `{"default_type": {"volume_type_id": ""}}`로 보냅니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답이 `default_type` 객체를 가지면 그 안을, 아니면 최상위 객체를 읽어 선언된 키만 요청 값 위에 덮어씁니다. 비어 있거나 JSON이 아닌 본문은 요청 값을 그대로 두고, JSON이지만 객체가 아니면 실제 응답을 담은 `resource.ResponseError`로 실패합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고 모든 오류는 `resource.OperationError`(Resource `default_types`)로 감쌉니다.

`DefaultType.ID`는 경로에 쓴 project ID(Python `id`)이고, `ProjectID`와 `VolumeTypeID`는 응답 본문의 값입니다. `Body`는 선언된 Body 키 4개(`id`, `name`, `project_id`, `volume_type_id`)를 raw JSON으로 보관하며 값이 없으면 null입니다. 목록 행은 경로 ID가 없으므로 `ID`가 비어 있습니다.

`Unset`은 Python `_delete`의 기본 `ignore_missing=True`를 따라 깨끗한 404를 nil로 바꿉니다. Python `unset_default_type`에는 이 인자가 없지만 Go는 `WithDeleteIgnoreMissing(false)`로 404를 돌려받을 수 있습니다.

`List`는 lazy iterator이며 반복할 때마다 버전 확인과 요청을 다시 합니다. Python `default_types()`는 query를 받지 않으므로 Go도 query 옵션을 제공하지 않고, `WithListMaxItems`, `WithListPaginated`, `WithListHeaders`만 있습니다. 요청에는 `Accept: application/json`이 붙습니다.

## Python과 다른 점

Python은 project와 volume type에 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. Go는 project ID를 URL segment 하나로 escape하고 빈 값, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. Python이 지원하지 않는 버전에서 내는 예외는 `SDKException`이고 Go는 `resource.ErrUnsupported`입니다. 헤더 옵션은 Go 확장이며 인증, microversion, content 헤더는 SDK가 소유합니다.
