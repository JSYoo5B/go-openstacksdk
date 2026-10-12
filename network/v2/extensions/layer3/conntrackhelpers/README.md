# Neutron router conntrack helper

`conntrackhelpers.API`는 Gophercloud v2.15.0이 바인딩하지 않는 Neutron `/routers/{router_id}/conntrack_helpers` 경로를 SDK가 직접 호출하는 API입니다. pinned openstacksdk의 conntrack helper 메서드 5개를 제공하며, `api := conntrackhelpers.New(networkClient)`처럼 Neutron service client로 만듭니다. Python에는 `find_conntrack_helper`가 없으므로 Go에도 `Find`가 없습니다.

## Proxy 메서드

| Python `conn.network` | Go | 요청 |
|---|---|---|
| `create_conntrack_helper(router, **attrs)` | `Create(ctx, routerID, options...)` | `POST /v2.0/routers/{router_id}/conntrack_helpers` |
| `get_conntrack_helper(conntrack_helper, router)` | `Get(ctx, routerID, id, options...)` | `GET .../conntrack_helpers/{id}` |
| `update_conntrack_helper(conntrack_helper, router, **attrs)` | `Update(ctx, routerID, id, options...)` | `PUT .../conntrack_helpers/{id}` |
| `delete_conntrack_helper(conntrack_helper, router, ignore_missing=True)` | `Delete(ctx, routerID, id, options...)` | `DELETE .../conntrack_helpers/{id}` |
| `conntrack_helpers(router, **query)` | `List(ctx, routerID, options...)`, `All(ctx, routerID, options...)` | `GET .../conntrack_helpers` |

Python의 get, update, delete는 helper를 먼저, router를 나중에 받습니다. Go는 기존 SDK의 하위 resource API처럼 router ID를 먼저 받습니다. Python `_get_resource(Router, router)`는 HTTP 없이 ID만 꺼내므로 Go도 router를 조회하지 않고 받은 router ID를 한 path segment로 escape해서 씁니다.

```python
helper = conn.network.create_conntrack_helper(
    router, helper="ftp", protocol="tcp", port=21)
conn.network.update_conntrack_helper(helper, router, port=2121)
for item in conn.network.conntrack_helpers(router, helper="ftp"):
    print(item.id, item.port)
conn.network.delete_conntrack_helper(helper, router)
```

```go
api := conntrackhelpers.New(networkClient)
helper, err := api.Create(ctx, routerID,
    conntrackhelpers.WithCreateField("helper", "ftp"),
    conntrackhelpers.WithCreateField("protocol", "tcp"),
    conntrackhelpers.WithCreateField("port", 21))
if err != nil {
    return err
}
id := helper.Text("id")
if _, err := api.Update(ctx, routerID, id, conntrackhelpers.WithUpdateField("port", 2121)); err != nil {
    return err
}
for item, err := range api.List(ctx, routerID, conntrackhelpers.WithListQuery("helper", "ftp")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["port"]))
}
_, err = api.Delete(ctx, routerID, id)
return err
```

## 결과 Record

`Record.Resource.Body`는 Python `to_dict()`와 같은 view이며 `id`, `name`, `helper`, `protocol`, `port`, `location`만 담습니다. ConntrackHelper는 `_allow_unknown_attrs_in_body`를 켜지 않은 `Resource`이므로 응답의 `project_id` 같은 선언하지 않은 key는 view에서 빠지고 `Wire`에만 남습니다. `router_id`는 URI attribute라서 `to_dict()`에 나오지 않으며, Go는 고정된 router ID를 `Record.RouterID`에 둡니다. `port`는 타입 선언이 없어 서버 값을 그대로 둡니다. `location`은 항상 `null`이고 `Envelope`, `Header`, `StatusCode`는 실제 응답 값입니다.

## 생성과 수정

`Create`와 `Update`는 선언된 attribute(`id`, `name`, `helper`, `protocol`, `port`)만 `conntrack_helper` 아래에 보냅니다. Python `Resource`가 선언하지 않은 attribute를 버리므로 Go도 다른 key를 HTTP 전에 버립니다. 명시한 `null`, `false`, `0`은 그대로 보내고, attribute가 없으면 `Create`는 `{"conntrack_helper":{}}`를 보냅니다. `router_id`를 attribute로 주면 Python에서 같은 keyword가 두 번 전달되므로 Go는 `resource.ErrInvalidOption`으로 거부합니다. Resource 제어 이름도 HTTP 전에 거부합니다.

`Update`는 보낼 attribute가 없으면 Python `commit`처럼 HTTP 없이 ID만 담긴 view를 돌려줍니다. `id`는 update 대상과 겹치므로 거부합니다. ConntrackHelper는 `NetworkResource`가 아니어서 Python `_update`가 `if_revision`을 받아도 `If-Match`를 만들지 않으며, Go도 이 값을 버립니다.

## 조회, 삭제, 목록

`Get`은 Python처럼 query 없이 한 번 GET합니다. `Delete`는 깨끗한 404를 기본으로 무시하고 `DeleteResult.IgnoredMissing`을 채우며, `WithDeleteIgnoreMissing(false)`이면 404를 돌려줍니다. 200부터 399까지를 성공으로 받고, 400 이상은 원래 status와 body를 담은 Gophercloud `ErrUnexpectedResponseCode`입니다. 모든 오류는 `resource.OperationError`(`Resource`는 `conntrackhelpers`)로 감쌉니다.

ConntrackHelper는 `QueryParameters`를 선언하지 않으므로 서버 query는 기본값인 `limit`과 `marker`뿐입니다. `helper`, `protocol`, `port`, `id`, `name`을 `WithListQuery`로 주면 받은 row에 local filter로 적용하고, 그 밖의 key는 버립니다. `router_id`는 bound 인자와 겹치므로 거부합니다. `List`는 `conntrack_helpers_links`, top-level `next`, HTTP `Link` header를 따라가며, 링크가 없고 limit가 있으면 마지막 row ID를 marker로 삼아 빈 페이지까지 읽습니다. `WithListMaxItems`는 raw row 상한이고 `WithListPaginated(false)`는 첫 페이지만 읽습니다.

## Python과 다른 점

- Python은 helper와 router에 ID나 Resource 객체를 받지만 Go는 ID 문자열만 받고, 인자 순서는 router ID가 먼저입니다.
- Neutron에는 microversion이 없어서 Python은 version header를 보내지 않습니다. Go는 선택한 client에 `Microversion`이 설정되어 있을 때만 `OpenStack-API-Version`을 보내며, `With*Microversion`과 `With*Headers`는 Go 확장입니다.
- 다음 페이지 링크가 처음 router 경로나 filter를 바꾸거나 같은 페이지를 반복하면 Go는 실패합니다.
- `location`은 항상 `null`이고, 응답 status 100부터 199는 성공으로 받지 않습니다.
- Python delete는 `None`을 돌려주지만 Go는 실제 응답을 담은 `DeleteResult`를 돌려줍니다.
