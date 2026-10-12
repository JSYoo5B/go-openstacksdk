# Neutron network segment range

`networksegmentranges.API`는 Gophercloud v2.15.0이 바인딩하지 않는 Neutron `/network_segment_ranges` 경로를 SDK가 직접 호출하는 API입니다. pinned openstacksdk의 network segment range proxy 메서드 6개를 제공하며, `api := networksegmentranges.New(networkClient)`처럼 Neutron service client로 만듭니다. 요청과 응답 처리는 기존 SDK 소유 Neutron record(`rbacpolicies` 등)와 같은 `internal/neutronfind` 엔진을 사용합니다.

## Proxy 메서드

| Python `conn.network` | Go | 요청 |
|---|---|---|
| `create_network_segment_range(**attrs)` | `Create(ctx, options...)` | `POST /v2.0/network_segment_ranges` |
| `get_network_segment_range(range, fields=None)` | `Get(ctx, id, options...)` | `GET /v2.0/network_segment_ranges/{id}` |
| `update_network_segment_range(range, **attrs)` | `Update(ctx, id, options...)` | `PUT /v2.0/network_segment_ranges/{id}` |
| `delete_network_segment_range(range, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE /v2.0/network_segment_ranges/{id}` |
| `network_segment_ranges(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET /v2.0/network_segment_ranges` |
| `find_network_segment_range(name_or_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | ID로 GET한 뒤 `?name=` 목록 |

```python
rng = conn.network.create_network_segment_range(
    name="vlan-100", network_type="vlan", physical_network="physnet1",
    minimum=100, maximum=199, shared=True)
conn.network.update_network_segment_range(rng, maximum=299)
for item in conn.network.network_segment_ranges(network_type="vlan"):
    print(item.name, item.available)
conn.network.delete_network_segment_range(rng)
```

```go
api := networksegmentranges.New(networkClient)
rng, err := api.Create(ctx,
    networksegmentranges.WithCreateField("name", "vlan-100"),
    networksegmentranges.WithCreateField("network_type", "vlan"),
    networksegmentranges.WithCreateField("physical_network", "physnet1"),
    networksegmentranges.WithCreateField("minimum", 100),
    networksegmentranges.WithCreateField("maximum", 199),
    networksegmentranges.WithCreateField("shared", true))
if err != nil {
    return err
}
id := rng.Text("id")
if _, err := api.Update(ctx, id, networksegmentranges.WithUpdateField("maximum", 299)); err != nil {
    return err
}
for item, err := range api.List(ctx, networksegmentranges.WithListQuery("network_type", "vlan")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("name"), string(item.Resource.Body["available"]))
}
_, err = api.Delete(ctx, id)
return err
```

## 결과 Record

`Create`, `Get`, `Update`, `Find`, `List`는 `*Record`를 돌려줍니다. `Resource.Body`는 Python Resource의 `to_dict()`와 같은 attribute 이름의 view입니다. 선언된 attribute는 `id`, `name`, `default`, `shared`, `project_id`, `network_type`, `physical_network`, `minimum`, `maximum`, `used`, `available`이며 응답에 없으면 `null`입니다. Python 선언처럼 `default`와 `shared`는 truthiness로 bool이 되고, `minimum`과 `maximum`은 정수로, `used`는 dict로, `available`은 list로 변환됩니다. 클래스가 `_allow_unknown_attrs_in_body`를 켜므로 `tenant_id`처럼 선언하지 않은 응답 key도 view에 남습니다. `project_id`에는 `tenant_id` alias가 없어서 서버가 `tenant_id`만 보내면 `project_id`는 `null`입니다. `location`은 connection 정보가 없는 단독 package이므로 항상 `null`이고, 서버가 보낸 값은 `Wire`에만 남습니다. `Wire`는 응답 객체 원문, `Envelope`은 응답 body 전체이며 `Header`와 `StatusCode`는 실제 응답 값입니다.

## 생성과 수정

`Create`와 `Update`는 `WithCreateField`와 `WithUpdateField`로 준 attribute만 `network_segment_range` 아래에 보냅니다. 명시한 `null`, `false`, `0`은 그대로 보내며, 선언하지 않은 attribute도 클래스가 허용하므로 Body에 넣습니다. Neutron이 필수 값을 검사하므로 attribute 없이 `Create`를 호출하면 `{"network_segment_range":{}}`를 보냅니다. `microversion` attribute는 Python `Resource.new`처럼 버리고, `base_path`, `connection`, `location` 같은 Resource 제어 이름은 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다. 응답에 없는 attribute는 요청 값이 view에 남고, 응답이 비어 있거나 JSON이 아니면 요청 값으로 만든 view를 돌려줍니다.

`Update`는 바꿀 attribute가 하나도 없으면 Python `commit`처럼 HTTP 요청 없이 ID만 담긴 view를 돌려줍니다. `id`를 attribute로 주면 update 대상 ID와 겹치므로 HTTP 전에 실패합니다.

## 조회, 삭제, 상태 코드

`Get`은 `WithGetFields(...)`를 받으면 Python `_get_with_fields`처럼 attribute 이름을 wire 이름으로 바꿔 `fields` query를 반복해서 보냅니다. 이 클래스는 attribute 이름과 wire 이름이 같습니다. ID는 한 path segment로 escape하며 비어 있거나 제어 문자를 포함하면 HTTP 전에 실패합니다.

`Delete`는 Python `ignore_missing=True` 기본값처럼 깨끗한 404를 무시하고 `DeleteResult.IgnoredMissing`을 `true`로 채웁니다. `WithDeleteIgnoreMissing(false)`를 주면 404를 그대로 돌려줍니다. Python은 `None`을 돌려주지만 Go는 실제 응답 status, header, body를 `DeleteResult`로 보여 줍니다.

200부터 399까지의 응답을 성공으로 받습니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`에 원래 status와 body를 담아 돌려줍니다. 모든 오류는 `resource.OperationError`로 감싸며 `Operation`은 Go 메서드 이름, `Resource`는 `networksegmentranges`입니다.

## 목록과 find

`WithListQuery(key, value)`는 Python `**query` 하나를 받습니다. `QueryParameters`에 선언된 `name`, `default`, `fields`, `shared`, `project_id`, `network_type`, `physical_network`, `minimum`, `maximum`, `used`, `available`, `sort_key`, `sort_dir`, `limit`, `marker`는 서버 query로 보내고, 그 밖의 선언된 attribute인 `id`는 받은 row에 local filter로 적용합니다. 선언되지 않은 key는 버립니다. query 값은 Requests처럼 인코딩하므로 `true`는 `True`가 됩니다. 목록 요청에는 `Accept: application/json`을 붙입니다.

`List`는 lazy iterator이고 `All`은 오류가 나면 부분 결과를 버리는 수집기입니다. `network_segment_ranges_links`의 `next`, top-level `next`, HTTP `Link` header를 따라가고, 링크가 없을 때는 limit가 있으면 마지막 row의 ID를 marker로 삼아 빈 페이지가 나올 때까지 읽습니다. `WithListMaxItems(n)`은 local filter 전의 raw row 수를 제한하며 limit가 없으면 `limit=n`을 보냅니다. `WithListPaginated(false)`는 첫 페이지만 읽습니다.

`Find`는 먼저 `GET /network_segment_ranges/{name_or_id}`를 보냅니다. 깨끗한 400, 403, 404가 오면 클래스가 `name` query를 선언하므로 `?name={name_or_id}`를 붙인 목록을 끝까지 읽고 ID나 이름이 정확히 같은 row를 찾습니다. `WithFindQuery`로 준 값은 Python처럼 첫 GET의 query와 목록 query 양쪽에 쓰입니다. 찾지 못하면 기본값에서는 `nil, nil`을, `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`를 돌려줍니다. 둘 이상이면 `resource.ErrAmbiguous`입니다. 500 같은 다른 오류는 목록으로 넘어가지 않습니다.

## Python과 다른 점

- Python은 ID 문자열이나 Resource 객체를 받지만 Go는 ID 문자열만 받습니다.
- Neutron에는 microversion이 없어서 Python은 version header를 보내지 않습니다. Go는 선택한 client에 `Microversion`이 설정되어 있을 때만 `OpenStack-API-Version: network ...`을 보내며, `With*Microversion`과 `With*Headers`는 Go 확장입니다.
- 다음 페이지 링크를 따라갈 때 Go는 처음 요청한 filter가 유지되는지 확인하고, 링크가 filter를 바꾸거나 같은 페이지를 반복하면 실패합니다. Python은 링크를 그대로 따라갑니다.
- local filter의 숫자 비교는 정확한 10진수 비교라서 Python float 반올림과 다를 수 있습니다.
- `location`은 항상 `null`입니다. Python은 connection의 cloud, region, project 정보를 넣습니다.
- 응답 status 100부터 199는 Go에서 성공으로 받지 않습니다.
