# Neutron local IP association

이 패키지는 pinned openstacksdk의 local IP association Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0은 Neutron `/local_ips/{local_ip_id}/port_associations` 경로를 바인딩하지 않으므로, 이 패키지가 SDK에서 직접 요청을 보냅니다. 기준 소스는 [`_proxy.py` Local IP Associations 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1943-L2109)과 [`local_ip_association.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/local_ip_association.py)입니다. local IP 자체는 [`localips`](../localips) 패키지가 담당합니다.

| Python `conn.network` | Go `localipassociations.API` | 요청 |
|---|---|---|
| `create_local_ip_association(local_ip, **attrs)` | `Create(ctx, localIPID, options...)` | `POST local_ips/{local_ip_id}/port_associations` |
| `get_local_ip_association(local_ip_association, local_ip, fields=None)` | `Get(ctx, localIPID, id, options...)` | `GET local_ips/{local_ip_id}/port_associations/{id}` |
| `delete_local_ip_association(local_ip, fixed_port_id, ignore_missing=True, if_revision=None)` | `Delete(ctx, localIPID, fixedPortID, options...)` | `DELETE local_ips/{local_ip_id}/port_associations/{fixed_port_id}` |
| `local_ip_associations(local_ip, **query)` | `List(ctx, localIPID, options...)`, `All(ctx, localIPID, options...)` | `GET local_ips/{local_ip_id}/port_associations` |
| `find_local_ip_association(name_or_id, local_ip, ignore_missing=True, **query)` | `Find(ctx, localIPID, nameOrID, options...)` | `?local_ip_id=`를 붙인 GET 뒤 목록 |

pinned Python Proxy에는 `update_local_ip_association`이 없으므로 Go에도 수정 메서드가 없습니다.

```python
assoc = conn.network.create_local_ip_association(lip, fixed_port_id=port.id)
for item in conn.network.local_ip_associations(lip, host="compute-1"):
    print(item.fixed_ip)
conn.network.delete_local_ip_association(lip, port.id)
```

```go
api := localipassociations.New(networkClient)
_, err := api.Create(ctx, lipID, localipassociations.WithCreateField("fixed_port_id", portID))
if err != nil { return err }
for item, err := range api.List(ctx, lipID, localipassociations.WithListQuery("host", "compute-1")) {
    if err != nil { return err }
    fmt.Println(*item.FixedIP)
}
err = api.Delete(ctx, lipID, portID) // 없는 association은 무시합니다
```

## 요청과 응답

모든 요청은 첫 인자로 받은 local IP ID를 URL segment 하나로 escape해 `local_ips/{local_ip_id}/port_associations` 아래로 보냅니다. 빈 ID, `.`과 `..`, 제어 문자는 HTTP 전에 거부합니다. 결과의 `LocalIPID`는 이 인자 값입니다. Python에서 `local_ip_id`는 URI 속성이라서 `to_dict` Body에 들어가지 않으므로, Go `Resource.Body`도 이 키를 두지 않습니다. 다만 member 응답이 같은 키를 담으면 Python은 그 키를 URI 속성으로 읽어 부모 ID를 돌려주므로 Go도 그때는 `Resource.Body["local_ip_id"]`에 인자 값을 넣고, 실제 응답 값은 `Wire`에 남깁니다.

생성은 `Resource.create`처럼 지정한 속성만 `{"port_association": {...}}`에 담아 POST합니다. 선언되지 않은 키도 본문에 들어가고, 값은 JSON 그대로 보냅니다. Python에서 `local_ip`와 `local_ip_id`를 attrs로 넘기면 메서드 인자와 겹쳐 TypeError가 나므로, Go도 생성, 목록, 찾기에서 이 두 이름을 invalid option으로 거부합니다.

조회는 `_get_with_fields`처럼 `WithGetFields`의 이름을 wire 이름으로 바꾸어 `?fields=`로 보냅니다. 응답에 `id`가 없으면 Python `Resource.fetch`처럼 요청한 ID가 `ID`에 남습니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답 본문의 `port_association` 객체(없으면 최상위 객체)를 요청 값 위에 덮어씁니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `localipassociations`, Operation `Create`, `Get`, `Delete`, `List`, `Find`)로 감쌉니다. `Resource.Body`는 선언된 Body 키 6개(`id`, `name`, `fixed_port_id`, `fixed_ip`, `host`, `local_ip_address`)와 알 수 없는 응답 키, 그리고 null인 `location`을 raw JSON으로 보관합니다.

## 삭제

`Delete`는 fixed port ID로 member 경로를 만들고, Python 기본값 `ignore_missing=True`를 따라 깨끗한 404를 nil로 바꿉니다. `WithDeleteIgnoreMissing(false)`를 주면 404도 오류로 돌려주고, 409나 500 같은 다른 status는 언제나 오류입니다. Python의 `if_revision`은 LocalIPAssociation이 `NetworkResource`가 아니라서 효과가 없으므로 Go는 이 인자를 받지 않습니다.

## 목록

`List`는 lazy iterator이며 반복할 때마다 요청을 다시 보냅니다. `fields`, `fixed_port_id`, `fixed_ip`, `host`, `sort_key`, `sort_dir`, `limit`, `marker`는 server query가 되고, `id`, `name`, `local_ip_address`는 Python처럼 응답 행을 로컬에서 거릅니다. 그 밖의 이름은 무시합니다. 요청에는 `Accept: application/json`이 붙고, 다음 페이지는 `port_associations_links`의 `rel=next` 링크나 `Link` 헤더로 따라갑니다. `WithListMaxItems`만 주면 그 값을 limit으로 보내고, `WithListPaginated(false)`는 첫 페이지만 읽습니다.

링크 없이 limit을 주면 Python은 마지막 행의 `Resource.id`를 marker로 쓰는데, Neutron port association 행에는 보통 `id`가 없습니다. Python은 다음 marker가 None이 되어 이전 marker와 같다고 보고 endless pagination `SDKException`을 냅니다(처음에 marker를 주었다면 marker 없는 요청을 한 번 더 보낸 뒤입니다). Go는 marker를 만들 수 없는 첫 시점에 invalid option 오류로 멈추므로 오류 타입과 추가 요청 여부가 다릅니다. 한 페이지만 필요하면 `WithListPaginated(false)`를 함께 주면 됩니다.

## 찾기

`Find`는 `Resource.find`처럼 먼저 `GET local_ips/{local_ip_id}/port_associations/{name_or_id}?local_ip_id={local_ip_id}`를 보냅니다. Python은 부모 ID를 `fetch`의 query 인자로도 넘기기 때문에 이 query가 붙고, `WithFindQuery`로 준 값도 같은 GET에 붙습니다. 응답이 깨끗한 400, 403, 404이면 전체 목록을 읽어 `id`나 `name` Body 값이 정확히 같은 항목을 고릅니다. 이 Resource의 query mapping에는 `name`이 없으므로 목록에 `?name=`을 붙이지 않습니다. port association 행에는 보통 `id`와 `name`이 없어서 목록 단계에서는 대개 찾지 못하고, 그러면 기본으로 `nil, nil`, `WithFindIgnoreMissing(false)`이면 `resource.NotFoundError`입니다.

## Python과 다른 점

Python은 local IP와 association에 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. Go 메서드는 언제나 local IP ID를 먼저 받으므로 `get_local_ip_association(association, local_ip)`와 인자 순서가 다릅니다. Python은 부모 ID와 member ID를 escape하지 않고 경로에 그대로 붙이지만 Go는 각각 segment 하나로 escape합니다. Python `_consume_attrs`는 속성 이름을 대소문자 구분 없이 맞추지만 Go는 정확히 같은 이름만 선언된 속성으로 봅니다. Python은 microversion을 협상하지만 Go는 client에 설정된 microversion이 있을 때만 `OpenStack-API-Version: network ...`을 보냅니다. 헤더와 microversion 옵션은 Go 확장이고, `Delete`는 Python의 None 대신 오류만 돌려줍니다. `find_local_ip_association`의 deprecated 경고는 재현하지 않습니다.
