# Neutron TaaS tap flow

이 패키지는 pinned openstacksdk의 tap flow Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0은 tap-as-a-service의 `/taas/tap_flows` 경로를 바인딩하지 않으므로, 이 패키지가 SDK에서 직접 요청을 보냅니다. 기준 소스는 [`_proxy.py` Tap Flows 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L8985-L9062)과 [`tap_flow.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_flow.py)입니다. 같은 TaaS 확장의 tap mirror는 generated [`tapmirrors`](../tapmirrors) 패키지가, tap service는 [`tapservices`](../tapservices) 패키지가 담당합니다.

| Python `conn.network` | Go `tapflows.API` | 요청 |
|---|---|---|
| `create_tap_flow(**attrs)` | `Create(ctx, options...)` | `POST taas/tap_flows` |
| `get_tap_flow(tap_flow, fields=None)` | `Get(ctx, id, options...)` | `GET taas/tap_flows/{id}` |
| `update_tap_flow(tap_flow, **attrs)` | `Update(ctx, id, options...)` | `PUT taas/tap_flows/{id}` |
| `delete_tap_flow(tap_flow, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE taas/tap_flows/{id}` |
| `tap_flows(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET taas/tap_flows` |
| `find_tap_flow(name_or_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | ID로 GET한 뒤 `?name=` 목록 |

```python
flow = conn.network.create_tap_flow(
    name="mirror", tap_service_id=service.id, source_port=port.id, direction="BOTH")
conn.network.update_tap_flow(flow, description="web 트래픽")
conn.network.delete_tap_flow(flow)
```

```go
api := tapflows.New(networkClient)
flow, err := api.Create(ctx,
    tapflows.WithCreateField("name", "mirror"),
    tapflows.WithCreateField("tap_service_id", serviceID),
    tapflows.WithCreateField("source_port", portID),
    tapflows.WithCreateField("direction", "BOTH"))
if err != nil { return err }
_, err = api.Update(ctx, flow.ID, tapflows.WithUpdateField("description", "web 트래픽"))
if err != nil { return err }
err = api.Delete(ctx, flow.ID) // 없는 tap flow는 무시합니다
```

## 요청과 응답

생성은 `Resource.create`처럼 지정한 속성만 `{"tap_flow": {...}}`에 담아 POST합니다. `WithCreateField`의 키로 Python 속성 이름과 wire 이름을 모두 쓸 수 있고, TapFlow는 `_allow_unknown_attrs_in_body`를 켜 두었으므로 선언되지 않은 키도 본문에 그대로 들어갑니다. 값은 JSON 그대로 보내므로 null, false, 빈 문자열도 구분해서 전달됩니다. `microversion` 키는 Python 생성자가 소비하는 값이라서 본문에 넣지 않고, `location`이나 `base_path` 같은 Resource 제어 이름은 HTTP 전에 거부합니다.

수정은 `Resource.commit`처럼 지정한 속성만 PUT 본문에 담습니다. 보낼 속성이 없으면 요청을 보내지 않고 ID만 채운 결과를 돌려줍니다. Python에서 `update_tap_flow(tap_flow, id=...)`나 `tap_flow=...`를 attrs로 넘기면 TypeError가 나므로, Go도 `id`와 `tap_flow` 필드를 invalid option으로 거부합니다.

조회는 `_get_with_fields`처럼 `WithGetFields`의 Python 속성 이름을 wire 이름으로 바꾸어 `?fields=a&fields=b`로 보냅니다. 빈 목록이면 query를 붙이지 않습니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답 본문의 `tap_flow` 객체(없으면 최상위 객체)를 요청 값 위에 덮어써 Python Resource 보기를 만듭니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `tapflows`, Operation `Create`, `Get`, `Update`, `Delete`, `List`, `Find`)로 감쌉니다.

`TapFlow`의 typed 필드는 JSON 타입이 맞을 때만 채워집니다. `Resource.Body`는 선언된 Body 키 9개와 알 수 없는 응답 키, 그리고 null인 `location`을 raw JSON으로 보관합니다. `project_id`가 없고 `tenant_id`만 있으면 Python alias처럼 `project_id`에도 그 값이 들어갑니다. `Wire`는 실제 응답 객체, `Envelope`는 실제 응답 본문입니다.

## 삭제

`Delete`는 Python 기본값 `ignore_missing=True`를 따라 깨끗한 404를 nil로 바꿉니다. `WithDeleteIgnoreMissing(false)`를 주면 404도 오류로 돌려주고, 409나 500 같은 다른 status는 언제나 오류입니다.

## 목록

`List`는 lazy iterator이며 반복할 때마다 요청을 다시 보냅니다. Python의 `**query`는 `WithListQuery(key, value)`로 넘깁니다. `sort_key`, `sort_dir`, `fields`, `name`, `project_id`, `limit`, `marker`는 Requests 방식으로 인코딩한 server query가 되고, `id`, `description`, `tenant_id`, `tap_service_id`, `direction`, `status`, `source_port`는 Python처럼 응답 행을 로컬에서 거릅니다. 그 밖의 이름은 무시합니다. 요청에는 `Accept: application/json`이 붙습니다.

다음 페이지는 Neutron `tap_flows_links`의 `rel=next` 링크나 `Link` 헤더로 찾고, `/v2.0/...` 형태의 상대 링크도 같은 collection으로 따라갑니다. 링크가 없고 limit을 주었으면 Python처럼 마지막 행의 ID를 marker로 삼아 빈 페이지가 나올 때까지 계속 요청합니다. `WithListMaxItems`만 주면 그 값을 limit으로 보내고 로컬 필터보다 먼저 행 수를 세며, `WithListPaginated(false)`는 첫 페이지만 읽습니다.

## 찾기

`Find`는 `Resource.find`처럼 먼저 `GET taas/tap_flows/{name_or_id}`를 보내고, `WithFindQuery`로 준 값은 이 GET의 query에도 붙습니다. 응답이 깨끗한 400, 403, 404이면 `?name={name_or_id}`를 붙인 전체 목록을 읽어 ID나 이름이 정확히 같은 항목을 고릅니다. 없으면 기본으로 `nil, nil`이고, `WithFindIgnoreMissing(false)`이면 `resource.NotFoundError`입니다. 같은 이름이 둘 이상이면 오류이며, 500 같은 다른 실패에서는 목록으로 넘어가지 않습니다.

## Python과 다른 점

Python은 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. Go는 ID를 URL segment 하나로 escape하고 빈 ID, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. Python은 ID를 escape하지 않고 경로에 그대로 붙입니다. Python `_consume_attrs`는 속성 이름을 대소문자 구분 없이 맞추지만 Go는 정확히 같은 이름만 선언된 속성으로 봅니다. Python은 microversion을 협상하지만 Go는 client에 설정된 microversion이 있을 때만 `OpenStack-API-Version: network ...`을 보냅니다. 헤더와 microversion 옵션은 Go 확장이고, `Delete`는 Python의 None 대신 오류만 돌려줍니다. `find_tap_flow`의 deprecated 경고는 재현하지 않습니다.
