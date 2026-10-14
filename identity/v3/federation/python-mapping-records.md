# Keystone federation mapping record API

이 문서는 `federation.API`에 SDK가 직접 추가한 mapping record 메서드를 설명합니다. Gophercloud v2.15.0 `federation.Mapping`에는 `schema_version`이 없고 rules를 고정된 `MappingRule` 구조체로 decode하므로, 구조체 밖의 rule 키와 `schema_version`이 사라집니다. record 메서드는 응답 JSON을 그대로 보관하는 `MappingRecord`를 돌려줍니다. 기준 소스는 [`_proxy.py` Mapping 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2982-L3020)과 [`mapping.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/mapping.py)입니다.

| Python `conn.identity` | Go `federation.API` | 요청 |
|---|---|---|
| `get_mapping(mapping)` | `GetMappingRecord(ctx, id, options...)` | `GET OS-FEDERATION/mappings/{id}` |
| `mappings(**query)` | `ListMappingRecords(ctx, options...)`, `AllMappingRecords(ctx, options...)` | `GET OS-FEDERATION/mappings` |
| `update_mapping(mapping, **attrs)` | `UpdateMappingRecord(ctx, id, options...)` | `PATCH OS-FEDERATION/mappings/{id}` |
| `delete_mapping(mapping, ignore_missing=True)` | `DeleteMappingRecord(ctx, id, options...)` | `DELETE OS-FEDERATION/mappings/{id}` |
| `find_mapping(name_or_id, ignore_missing=True)` | `FindMappingRecord(ctx, nameOrID, options...)` | GET 뒤 목록 |

```go
api := federation.New(identityClient)
m, err := api.GetMappingRecord(ctx, "corp-mapping")
if err != nil { return err }
fmt.Println(*m.SchemaVersion, string(m.Rules))
```

## 응답

`MappingRecord.ID`는 Python의 `id`이자 `name`입니다. `SchemaVersion`은 JSON 문자열일 때만 채워지고, `Rules`는 Keystone이 돌려준 rules 배열의 raw JSON이라서 `MappingRule`에 없는 키도 그대로 남습니다. `Body`는 선언된 Body 키 3개(`id`, `rules`, `schema_version`)를 raw JSON으로 보관하고 값이 없으면 null입니다. `Wire`는 실제 응답 객체 전체, `Envelope`, `Header`, `StatusCode`는 실제 응답입니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답에 `mapping` 객체가 있으면 그 안을, 없으면 최상위 객체를 읽습니다. 비어 있거나 JSON이 아닌 본문은 요청 값만 남기고, JSON이지만 객체가 아니면 실제 응답을 담은 `resource.ResponseError`로 실패합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `mappings`, Operation `Get`, `List`, `Update`, `Delete`, `Find`)로 감쌉니다.

## 목록

`ListMappingRecords`는 lazy iterator입니다. Mapping은 limit과 marker 말고 server query를 선언하지 않으므로, `WithMappingRecordListQuery`로 넘긴 `id`, `name`, `rules`, `schema_version`은 Python `Resource.list`처럼 응답 행을 로컬에서 거르고 그 밖의 이름은 무시합니다. 요청에는 `Accept: application/json`이 붙습니다. `WithMappingRecordListLimit`, `WithMappingRecordListMarker`, `WithMappingRecordListMaxItems`, `WithMappingRecordListPaginated(false)`는 Python의 `limit`, `marker`, `max_items`, `paginated`와 같고, limit을 주면 빈 페이지가 나올 때까지 마지막 ID를 marker로 삼아 다음 페이지를 요청합니다.

## 수정, 삭제, 찾기

`UpdateMappingRecord`는 지정한 `rules`와 `schema_version`만 PATCH 본문에 담습니다. `id`와 `name`은 같은 Body 키이고 commit에서 dirty가 될 수 없으므로 본문에 넣지 않으며, 보낼 속성이 없으면 요청하지 않습니다. `DeleteMappingRecord`는 깨끗한 404를 nil로 바꾸고 `WithMappingRecordDeleteIgnoreMissing(false)`이면 404도 오류입니다. `FindMappingRecord`는 먼저 member GET을 보내고 깨끗한 400, 403, 404이면 query 없는 전체 목록에서 ID가 정확히 같은 항목을 고릅니다.

## Python과 다른 점

Python은 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받고, ID를 URL segment 하나로 escape하며 빈 ID와 제어 문자를 HTTP 전에 거부합니다. Python은 `schema_version`을 `str`로, `rules`를 `list`로 변환하지만 Go는 JSON 값을 그대로 두고 로컬 필터도 JSON 값으로 비교합니다. 목록은 Keystone `links.next`가 문자열이면 따라가는데, pinned Python 페이지네이터는 이 dict 형태를 무시합니다. `create_mapping`은 Python이 PUT 본문에 `id`를 남기는 방식이 공용 PUT helper와 달라 이 record API에 넣지 않았습니다. 헤더 옵션은 Go 확장입니다.
