# Cinder group type

이 패키지는 pinned openstacksdk의 group type Proxy 메서드와 group_specs 메서드를 Go로 제공합니다. Gophercloud v2.15.0에는 `group_types` 바인딩이 없으므로 이 패키지가 요청을 SDK에서 직접 보냅니다. 기준 소스는 [`_proxy.py` Group Types 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2809-L3025)과 [`group_type.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_type.py)입니다.

| Python `conn.block_storage` | Go `grouptypes.API` | 요청 |
|---|---|---|
| `get_group_type(group_type)` | `Get(ctx, id, options...)` | `GET group_types/{id}` |
| `find_group_type(name_or_id, ignore_missing=True)` | `Find(ctx, nameOrID, options...)` | `GET group_types/{name_or_id}?is_public=none` 뒤 목록 |
| `group_types(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET group_types` |
| `create_group_type(**attrs)` | `Create(ctx, options...)` | `POST group_types` |
| `update_group_type(group_type, **attrs)` | `Update(ctx, id, options...)` | `PUT group_types/{id}` |
| `delete_group_type(group_type, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE group_types/{id}` |
| `fetch_group_type_group_specs(group_type)` | `FetchGroupSpecs(ctx, id, options...)` | `GET group_types/{id}/group_specs` |
| `create_group_type_group_specs(group_type, group_specs)` | `CreateGroupSpecs(ctx, id, specs, options...)` | `POST group_types/{id}/group_specs` |
| `get_group_type_group_specs_property(group_type, prop)` | `GetGroupSpecsProperty(ctx, id, prop, options...)` | `GET group_types/{id}/group_specs/{prop}` |
| `update_group_type_group_specs_property(group_type, prop, val)` | `UpdateGroupSpecsProperty(ctx, id, prop, val, options...)` | `PUT group_types/{id}/group_specs/{prop}` |
| `delete_group_type_group_specs_property(group_type, prop, ignore_missing=True)` | `DeleteGroupSpecsProperty(ctx, id, prop, options...)` | `DELETE group_types/{id}/group_specs/{prop}` |

```python
gt = conn.block_storage.create_group_type(name="gold", is_public=True)
conn.block_storage.create_group_type_group_specs(gt, {"consistent_group_snapshot_enabled": "<is> True"})
conn.block_storage.delete_group_type(gt)
```

```go
api := grouptypes.New(volumeClient)
gt, err := api.Create(ctx,
    grouptypes.WithCreateField("name", "gold"),
    grouptypes.WithCreateField("is_public", true))
if err != nil { return err }
_, err = api.CreateGroupSpecs(ctx, gt.ID, map[string]any{"consistent_group_snapshot_enabled": "<is> True"})
if err != nil { return err }
err = api.Delete(ctx, gt.ID) // 없는 group type은 무시합니다
```

## Microversion

모든 호출은 `GroupType._get_microversion`을 따릅니다. client에 microversion이 설정되어 있으면 discovery 없이 그 값을 보내고, 없으면 Cinder version 문서를 한 번 읽어 3.11과 서버 최대값 중 작은 값을 `OpenStack-API-Version: volume X`와 `X-OpenStack-Volume-API-Version: X`로 보냅니다. 서버가 버전을 알리지 않으면 microversion 헤더 없이 보냅니다. 실제 discovery 응답은 결과의 `Discovery`에 남습니다.

## 요청과 응답

생성은 `Resource.create`처럼 `{"group_type": {...}}` 본문으로 POST합니다. `WithCreateField`의 키는 선언된 속성 `id`, `name`, `description`, `group_specs`, `is_public`이고 그 밖의 키는 Python `Resource.new`처럼 버립니다. 값은 JSON 그대로 보내므로 null과 false도 전달됩니다.

수정은 `Resource.commit`처럼 지정한 속성만 PUT 본문에 담습니다. `id`는 dirty 속성이 될 수 없어서 무시하며, 보낼 속성이 없으면 Python처럼 discovery와 요청을 모두 생략하고 ID만 채운 결과를 돌려줍니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답이 `group_type` 객체를 가지면 그 안을, 아니면 최상위 객체를 읽어 선언된 키만 요청 값 위에 덮어씁니다. 비어 있거나 JSON이 아닌 본문은 요청 값을 그대로 두고, JSON이지만 객체가 아니면 실제 응답을 담은 `resource.ResponseError`로 실패합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고 모든 오류는 `resource.OperationError`(Resource `group_types`)로 감쌉니다.

`GroupType.Body`는 선언된 Body 키 5개를 raw JSON으로 보관합니다. `group_specs`는 Python 기본값처럼 값이 없으면 `{}`이고 나머지는 null입니다. `GroupSpecs`, `Description`, `IsPublic` 같은 typed 필드는 JSON 타입이 맞을 때만 채워지며, `Wire`는 실제 응답 객체, `Envelope`는 실제 응답 본문입니다.

## group_specs

`FetchGroupSpecs`와 `CreateGroupSpecs`는 응답이 JSON 객체여야 하며, 그 `group_specs` 값(키가 없으면 `{}`)을 ID와 함께 담은 `GroupType`을 돌려줍니다. 이 결과의 `Wire`는 nil입니다. Python도 응답을 `response.json()`으로 읽으므로 JSON이 아니거나 객체가 아닌 본문은 오류입니다.

`GetGroupSpecsProperty`는 응답 객체에서 `prop` 멤버를 꺼내 `SpecProperty.Value`에 담고, 멤버가 없으면 Python `dict.get`처럼 JSON null을 줍니다. `UpdateGroupSpecsProperty`는 `{prop: val}`을 PUT하고 응답에 `prop` 멤버가 없으면 Python의 `KeyError`처럼 오류를 돌려줍니다. `DeleteGroupSpecsProperty`는 Python과 같이 깨끗한 404만 무시하며 `WithDeleteIgnoreMissing(false)`이면 404도 오류입니다.

## 삭제와 목록

`Delete`는 기본으로 깨끗한 404를 nil로 바꾸고, `WithDeleteIgnoreMissing(false)`이면 404를 돌려줍니다. 다른 4xx와 5xx는 언제나 오류입니다.

`List`는 lazy iterator이며 반복할 때마다 요청을 다시 보냅니다. `WithListQuery`로 넘긴 `is_public`, `offset`, `sort_dir`, `sort_key`, `sort`는 server query가 되고 `all_projects`는 `all_tenants`로 나갑니다. 값은 Requests 방식으로 인코딩하므로 `true`는 `True`가 됩니다. `id`, `name`, `description`, `group_specs`는 Python처럼 응답 행을 로컬에서 거르고 그 밖의 이름은 무시합니다. `WithListLimit`, `WithListMarker`, `WithListMaxItems`, `WithListPaginated(false)`는 Python의 `limit`, `marker`, `max_items`, `paginated`와 같고, `group_types_links`의 next 링크나 marker 규칙으로 다음 페이지를 따라갑니다. 요청에는 `Accept: application/json`이 붙습니다.

## 찾기

`Find`는 Python과 같이 `is_public=none`을 붙여 `GET group_types/{name_or_id}`를 먼저 보냅니다. 응답이 깨끗한 400, 403, 404이면 `GET group_types?is_public=none` 전체 목록에서 ID나 이름이 정확히 같은 항목을 고릅니다. 없으면 기본으로 `nil, nil`이고, `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`입니다. 같은 이름이 둘 이상이면 `resource.AmbiguousError`이며, 500 같은 다른 실패에서는 목록으로 넘어가지 않습니다.

## Python과 다른 점

Python은 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. Go는 ID와 spec 이름을 URL segment 하나로 escape하고 빈 값, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. Python `utils.urljoin`은 escape 없이 경로에 붙입니다. 같은 Body 키에 서로 다른 값을 주면 Python은 하나를 임의로 고르지만 Go는 HTTP 전에 거부합니다. 로컬 필터는 JSON 값을 비교하므로 Python의 타입 변환은 하지 않습니다. `find_group_type`의 deprecated 경고와 Resource의 `location` 속성은 재현하지 않습니다. 헤더 옵션은 Go 확장이며 인증, microversion, content 헤더는 SDK가 소유합니다.
