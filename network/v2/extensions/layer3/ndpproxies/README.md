# Neutron router NDP proxy

`ndpproxies.API`는 Gophercloud v2.15.0이 바인딩하지 않는 Neutron `/ndp_proxies` 경로를 SDK가 직접 호출하는 API입니다. pinned openstacksdk의 NDP proxy 메서드 6개를 제공하며, `api := ndpproxies.New(networkClient)`처럼 Neutron service client로 만듭니다. 요청과 응답 처리는 기존 SDK 소유 Neutron record와 같은 `internal/neutronfind` 엔진을 사용합니다.

## Proxy 메서드

| Python `conn.network` | Go | 요청 |
|---|---|---|
| `create_ndp_proxy(**attrs)` | `Create(ctx, options...)` | `POST /v2.0/ndp_proxies` |
| `get_ndp_proxy(ndp_proxy, fields=None)` | `Get(ctx, id, options...)` | `GET /v2.0/ndp_proxies/{id}` |
| `update_ndp_proxy(ndp_proxy, **attrs)` | `Update(ctx, id, options...)` | `PUT /v2.0/ndp_proxies/{id}` |
| `delete_ndp_proxy(ndp_proxy, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE /v2.0/ndp_proxies/{id}` |
| `ndp_proxies(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET /v2.0/ndp_proxies` (첫 페이지만) |
| `find_ndp_proxy(ndp_proxy_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | ID로 GET한 뒤 `?name=` 목록 |

```python
proxy = conn.network.create_ndp_proxy(
    router_id=router.id, port_id=port.id, ip_address="2001:db8::10")
conn.network.update_ndp_proxy(proxy, description="web")
for item in conn.network.ndp_proxies(router_id=router.id):
    print(item.ip_address)
conn.network.delete_ndp_proxy(proxy)
```

```go
api := ndpproxies.New(networkClient)
proxy, err := api.Create(ctx,
    ndpproxies.WithCreateField("router_id", routerID),
    ndpproxies.WithCreateField("port_id", portID),
    ndpproxies.WithCreateField("ip_address", "2001:db8::10"))
if err != nil {
    return err
}
id := proxy.Text("id")
if _, err := api.Update(ctx, id, ndpproxies.WithUpdateField("description", "web")); err != nil {
    return err
}
for item, err := range api.List(ctx, ndpproxies.WithListQuery("router_id", routerID)) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("ip_address"))
}
_, err = api.Delete(ctx, id)
return err
```

## 결과 Record

모든 조회와 생성, 수정 메서드는 `*Record`를 돌려줍니다. `Resource.Body`는 Python Resource의 `to_dict()`와 같은 attribute 이름의 view이며 선언된 attribute는 `id`, `name`, `created_at`, `description`, `ip_address`, `port_id`, `project_id`, `revision_number`, `router_id`, `updated_at`입니다. 응답에 없는 attribute는 `null`입니다. NDPProxy는 `NetworkResource`가 아닌 `Resource`를 직접 상속하므로 `revision_number`에 정수 변환이 없고 `if_match` header attribute도 없습니다. 그래서 서버가 보낸 `revision_number` 값을 그대로 둡니다. 클래스가 `_allow_unknown_attrs_in_body`를 켜므로 `tenant_id`처럼 선언하지 않은 응답 key도 view에 남습니다. `location`은 항상 `null`이며 `Wire`, `Envelope`, `Header`, `StatusCode`는 실제 응답 값입니다.

## 생성, 수정, 조회, 삭제

`Create`와 `Update`는 `WithCreateField`와 `WithUpdateField`로 준 attribute만 `ndp_proxy` 아래에 보냅니다. 명시한 `null`, `false`, `0`과 선언하지 않은 attribute도 그대로 보내고, attribute가 없으면 `Create`는 `{"ndp_proxy":{}}`를 보냅니다. `microversion` attribute는 Python `Resource.new`처럼 버리고, Resource 제어 이름은 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다. `Update`는 바꿀 attribute가 없으면 HTTP 요청 없이 ID만 담긴 view를 돌려주고, `id` attribute는 update 대상과 겹치므로 거부합니다.

`Get`의 `WithGetFields(...)`는 `fields` query를 반복해서 보냅니다. `Delete`는 깨끗한 404를 기본으로 무시하고 `DeleteResult.IgnoredMissing`을 채우며, `WithDeleteIgnoreMissing(false)`이면 404를 돌려줍니다. 200부터 399까지를 성공으로 받고, 400 이상은 원래 status와 body를 담은 Gophercloud `ErrUnexpectedResponseCode`입니다. 모든 오류는 `resource.OperationError`(`Resource`는 `ndpproxies`)로 감쌉니다.

## 목록과 find

Python `ndp_proxies`는 `Resource.list`를 항상 `paginated=False`로 호출합니다. 그래서 Go `List`도 첫 페이지만 읽고 `ndp_proxies_links`나 `next`를 따라가지 않습니다. `paginated`를 query로 주면 Python에서는 같은 keyword가 두 번 전달되어 `TypeError`가 나므로 Go는 HTTP 전에 `resource.ErrInvalidOption`을 돌려주며, 이 package에는 `WithListPaginated`가 없습니다. `WithListLimit`, `WithListMarker`, `WithListMaxItems`는 첫 페이지 요청과 raw row 상한에만 쓰입니다.

서버 query는 `QueryParameters`에 선언된 `sort_key`, `sort_dir`, `name`, `description`, `fields`, `project_id`, `router_id`, `port_id`, `ip_address`, `limit`, `marker`입니다. `id`, `created_at`, `revision_number`, `updated_at` 같은 그 밖의 선언된 attribute는 local filter로 적용하고, 선언되지 않은 key는 버립니다. 목록 요청에는 `Accept: application/json`을 붙입니다.

`Find`는 먼저 `GET /ndp_proxies/{name_or_id}`를 보내고, 깨끗한 400, 403, 404가 오면 `?name={name_or_id}` 목록을 끝까지 읽어 ID나 이름이 정확히 같은 row를 찾습니다. Python `Resource.find`는 `paginated=False`를 쓰지 않으므로 이 목록은 다음 페이지 링크를 따라갑니다. 찾지 못하면 기본은 `nil, nil`, `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`이고, 둘 이상이면 `resource.ErrAmbiguous`입니다.

## Python과 다른 점

- Python은 ID 문자열이나 Resource 객체를 받지만 Go는 ID 문자열만 받고 한 path segment로 escape합니다.
- Neutron에는 microversion이 없어서 Python은 version header를 보내지 않습니다. Go는 선택한 client에 `Microversion`이 설정되어 있을 때만 `OpenStack-API-Version`을 보내며, `With*Microversion`과 `With*Headers`는 Go 확장입니다.
- find 목록에서 다음 페이지 링크가 처음 filter를 바꾸거나 같은 페이지를 반복하면 Go는 실패합니다.
- local filter의 숫자 비교는 정확한 10진수 비교입니다.
- `location`은 항상 `null`이고, 응답 status 100부터 199는 성공으로 받지 않습니다.
- Python delete는 `None`을 돌려주지만 Go는 실제 응답을 담은 `DeleteResult`를 돌려줍니다.
