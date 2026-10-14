# Keystone project endpoint record API

이 문서는 `projectendpoints.API`에 SDK가 직접 추가한 record 목록을 설명합니다. Gophercloud v2.15.0 `projectendpoints.API.List`는 query 옵션이 없고 `Endpoint` 모델에 `enabled`, `region_id`, `links`가 없으므로, record 목록이 Python `project_endpoints`의 요청과 결과를 대신 제공합니다. 기준 소스는 [`_proxy.py` project_endpoints](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L537-L555)와 [`endpoint.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/endpoint.py)입니다.

| Python `conn.identity` | Go `projectendpoints.API` | 요청 |
|---|---|---|
| `project_endpoints(project, **query)` | `ListRecords(ctx, projectID, options...)`, `AllRecords(ctx, projectID, options...)` | `GET OS-EP-FILTER/projects/{project_id}/endpoints` |

```go
api := projectendpoints.New(identityClient)
values, err := api.AllRecords(ctx, projectID,
    projectendpoints.WithRecordListQuery("interface", "public"),
    projectendpoints.WithRecordListQuery("is_enabled", true))
```

## 요청

`ProjectEndpoint`는 `Endpoint`를 상속하므로 `interface`, `region_id`, `service_id`가 server query입니다. 이 값은 Requests 방식으로 인코딩되고, `id`, `name`, `is_enabled`(또는 `enabled`), `links`, `url`은 Python `Resource.list`처럼 응답 행을 로컬에서 거릅니다. 그 밖의 이름은 무시하고, 요청에는 `Accept: application/json`이 붙습니다. project ID는 URL segment 하나로 escape하며, 빈 값과 제어 문자는 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다.

`WithRecordListLimit`, `WithRecordListMarker`, `WithRecordListMaxItems`, `WithRecordListPaginated(false)`는 Python의 `limit`, `marker`, `max_items`, `paginated`와 같습니다. limit을 주면 Python처럼 빈 페이지가 나올 때까지 마지막 ID를 marker로 삼아 다음 페이지를 요청하고, max_items만 주면 그 값을 limit으로 보내며 로컬 필터보다 먼저 행 수를 셉니다.

## 응답

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고 `resource.OperationError`(Resource `projectendpoints`, Operation `List`)로 감쌉니다.

`EndpointRecord`의 `ProjectID`는 Python이 목록의 각 행에 넣는 URI 속성입니다. typed 필드(`Name`, `Interface`, `Enabled`, `RegionID`, `ServiceID`, `URL`, `Links`)는 JSON 타입이 맞을 때만 채워집니다. `Body`는 선언된 Body 키 8개를 raw JSON으로 보관하고 값이 없으면 null이며, `Wire`는 실제 행 전체라서 `region` 같은 선언되지 않은 키도 남습니다. `Envelope`, `Header`, `StatusCode`는 그 행을 담은 페이지의 실제 응답입니다.

## Python과 다른 점

Python은 Project 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. 로컬 필터는 JSON 값을 비교하므로 bool과 숫자를 구분하고 Python의 타입 변환은 하지 않습니다. 목록은 Keystone `links.next`가 문자열이면 따라가는데, pinned Python 페이지네이터는 이 dict 형태를 무시합니다. 헤더 옵션은 Go 확장이며, Resource의 `location` 속성은 재현하지 않습니다.
