# Neutron service flavor

이 패키지는 pinned openstacksdk의 network service flavor Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0에는 Neutron `/v2.0/flavors` 바인딩이 없으므로, 이 패키지가 SDK에서 직접 요청을 보냅니다. 기준 소스는 [`_proxy.py` Flavors 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1630-L1794)과 [`flavor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/flavor.py)입니다.

| Python `conn.network` | Go `flavors.API` | 요청 |
|---|---|---|
| `create_flavor(**attrs)` | `Create(ctx, options...)` | `POST flavors` |
| `get_flavor(flavor, fields=None)` | `Get(ctx, id, options...)` | `GET flavors/{id}` |
| `update_flavor(flavor, **attrs)` | `Update(ctx, id, options...)` | `PUT flavors/{id}` |
| `delete_flavor(flavor, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE flavors/{id}` |
| `flavors(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET flavors` |
| `find_flavor(name_or_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | GET 뒤 목록 |
| `associate_flavor_with_service_profile(flavor, service_profile)` | `AssociateServiceProfile(ctx, flavorID, serviceProfileID, options...)` | `POST flavors/{id}/service_profiles` |
| `disassociate_flavor_from_service_profile(flavor, service_profile)` | `DisassociateServiceProfile(ctx, flavorID, serviceProfileID, options...)` | `DELETE flavors/{id}/service_profiles/{service_profile_id}` |

```python
flavor = conn.network.create_flavor(
    name="gold", service_type="L3_ROUTER_NAT", is_enabled=True)
conn.network.associate_flavor_with_service_profile(flavor, profile_id)
for f in conn.network.flavors(service_type="L3_ROUTER_NAT"):
    print(f.name, f.service_profile_ids)
conn.network.delete_flavor(flavor)
```

```go
api := flavors.New(networkClient)
flavor, err := api.Create(ctx,
    flavors.WithCreateField("name", "gold"),
    flavors.WithCreateField("service_type", "L3_ROUTER_NAT"),
    flavors.WithCreateField("is_enabled", true))
if err != nil { return err }
if _, err := api.AssociateServiceProfile(ctx, flavor.ID, profileID); err != nil { return err }
for f, err := range api.List(ctx, flavors.WithListQuery("service_type", "L3_ROUTER_NAT")) {
    if err != nil { return err }
    fmt.Println(*f.Name, f.ServiceProfileIDs)
}
_, err = api.Delete(ctx, flavor.ID) // 없는 flavor는 무시합니다
```

## 요청과 응답

생성은 `{"flavor": {...}}` 본문으로 POST합니다. `WithCreateField`의 키로 Python 속성 이름(`is_enabled`, `service_profile_ids`)과 Body 키(`enabled`, `service_profiles`)를 모두 쓸 수 있고, 둘 다 Body 키로 바뀌어 전송됩니다. Flavor는 `_allow_unknown_attrs_in_body`가 켜져 있으므로 선언되지 않은 키도 본문에 그대로 넣습니다. 값은 JSON 그대로 보내므로 null, false, 빈 목록도 구분됩니다. `base_path`, `connection`, `location` 같은 Resource 제어 이름은 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다.

수정은 `Resource.commit`처럼 지정한 속성만 PUT 본문에 담습니다. 보낼 속성이 없으면 요청 없이 ID만 채운 결과를 돌려줍니다. ID는 메서드 인자이므로 `id`와 `flavor`를 속성으로 주면 거부합니다. `Get`의 `WithGetFields`는 Python `_get_with_fields`처럼 Python 이름을 Body 키로 바꿔 `fields` query로 보냅니다.

Python은 400 미만의 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답이 `flavor` 객체를 가지면 그 안을, 아니면 최상위 객체를 읽습니다. `Flavor`의 typed 필드는 JSON 타입이 맞을 때만 채워지고, `Resource`는 Source 속성을 Python 이름으로 보관합니다(없는 값은 null이며 모르는 키도 남습니다). `service_profile_ids`는 list 타입이라 문자열 하나가 오면 목록 하나로 감쌉니다. `Wire`는 실제 응답 객체, `Envelope`는 실제 응답 본문입니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `flavors`)로 감쌉니다.

## 삭제

`Delete`는 Python 기본값 `ignore_missing=True`를 따라 깨끗한 404를 성공으로 바꾸고, 결과의 `IgnoredMissing`을 true로 표시합니다. `WithDeleteIgnoreMissing(false)`를 주면 404도 오류로 돌려줍니다. 409나 500 같은 다른 status는 언제나 오류입니다. Python은 None을 돌려주지만 Go는 실제 응답의 status와 header를 담은 `DeleteResult`를 함께 돌려줍니다.

## 목록

`List`는 lazy iterator이며 반복할 때마다 요청을 다시 보냅니다. Python의 `**query`는 `WithListQuery(key, value)`로 넘깁니다. `limit`, `marker`, `description`, `fields`, `name`, `service_type`, `sort_key`, `sort_dir`, `is_enabled`는 server query가 되고, `is_enabled`는 `enabled`로 이름이 바뀝니다. 값은 Requests 방식으로 인코딩하므로 `true`는 `enabled=True`로 나갑니다. `id`와 `service_profile_ids`는 server query가 아니므로 Python처럼 응답 행을 로컬에서 거르고, 그 밖의 이름은 무시합니다. 요청에는 `Accept: application/json`이 붙습니다.

페이지는 `flavors_links`의 next 링크, 응답의 `next`, HTTP `Link` 헤더를 따라가며 링크 query를 원래 query 위에 합칩니다. 링크가 없어도 limit만큼 행이 왔으면 마지막 ID를 marker로 삼아 다음 페이지를 요청합니다. `WithListMaxItems`와 `WithListPaginated(false)`는 `max_items`, `paginated`와 같고, max_items는 로컬 필터보다 먼저 행 수를 셉니다.

## 찾기

`Find`는 `Resource.find`처럼 먼저 `GET flavors/{name_or_id}`를 보냅니다. 응답이 깨끗한 400, 403, 404이면 목록을 끝까지 읽어 ID나 이름이 정확히 같은 항목을 고릅니다. Flavor의 query mapping에 `name`이 있으므로 이 목록 요청에는 `name=<name_or_id>`가 붙습니다. `WithFindQuery`의 값은 member GET과 목록에 함께 쓰입니다. 없으면 기본으로 `nil, nil`이고 `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`입니다. 같은 항목이 둘 이상이면 오류이고, 500 같은 다른 실패에서는 목록으로 넘어가지 않습니다.

## service profile 연결

`AssociateServiceProfile`은 `flavors/{flavor_id}/service_profiles`에 `{"service_profile": {"id": service_profile_id}}`를 POST하고, 응답 JSON 전체를 `Body`에 담아 돌려줍니다. `ServiceProfileID`는 응답의 `service_profile.id`가 문자열일 때 채워집니다. `DisassociateServiceProfile`은 `flavors/{flavor_id}/service_profiles/{service_profile_id}`에 본문 없는 DELETE를 보냅니다.

Python의 두 메서드는 Proxy 세션의 `raise_exc=False`로 요청하므로 HTTP 오류에서도 예외를 내지 않습니다. 그래서 associate는 오류 응답의 JSON을 그대로 dict로 돌려주고, disassociate는 어떤 status에서도 None을 돌려줍니다. Go는 200부터 399까지만 성공으로 받고, 400 이상은 status와 본문을 담은 `ErrUnexpectedResponseCode`를 `resource.OperationError`(Operation `AssociateServiceProfile` 또는 `DisassociateServiceProfile`)로 감싸 돌려줍니다. disassociate의 404도 오류입니다. 성공 응답이 JSON이 아니면 Python `resp.json()`이 예외를 내듯 Go도 실제 응답을 담은 `resource.ResponseError`로 실패합니다.

## Python과 다른 점

Python은 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. Go는 ID를 URL segment 하나로 escape하고 빈 ID, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. Python은 service profile이 None이면 associate에 `{"id": null}`을, disassociate에 마지막 segment가 빈 경로를 보내지만 Go는 빈 ID를 거부합니다. Python은 연결 경로를 만들 때 ID 앞뒤의 `/`를 지우고 escape하지 않지만, Go는 ID를 그대로 escape합니다. 마이크로버전은 Python처럼 협상하지 않고 설정된 client 값을 보내며, 헤더와 마이크로버전 옵션은 Go 확장입니다. `find_flavor`의 deprecated 경고는 재현하지 않습니다.
