# Keystone OS-EP-FILTER endpoint group

`endpointgroups.API`는 Gophercloud v2.15.0이 바인딩하지 않는 Keystone `OS-EP-FILTER` 경로를 SDK가 직접 호출하는 API입니다. pinned openstacksdk의 endpoint group proxy 메서드 12개와 cloud layer의 endpoint group helper 6개를 한 package에서 제공합니다. `api := endpointgroups.New(identityClient)`처럼 Keystone v3 service client로 만들며, 모든 요청은 그 client의 endpoint 아래 `OS-EP-FILTER/...`로 보냅니다.

## Proxy 메서드

| Python `conn.identity` | Go | 요청 |
|---|---|---|
| `create_endpoint_group(**attrs)` | `Create(ctx, options...)` | `POST OS-EP-FILTER/endpoint_groups` |
| `get_endpoint_group(endpoint_group)` | `Get(ctx, id, options...)` | `GET OS-EP-FILTER/endpoint_groups/{id}` |
| `update_endpoint_group(endpoint_group, **attrs)` | `Update(ctx, id, options...)` | `PATCH OS-EP-FILTER/endpoint_groups/{id}` |
| `delete_endpoint_group(endpoint_group, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE OS-EP-FILTER/endpoint_groups/{id}` |
| `endpoint_groups(**query)` | `List(ctx, options...)` | `GET OS-EP-FILTER/endpoint_groups` |
| `find_endpoint_group(name_or_id, ignore_missing=True)` | `Find(ctx, nameOrID, options...)` | ID로 GET한 뒤 `?name=` 목록 |
| `endpoint_group_endpoints(endpoint_group, **query)` | `ListEndpoints(ctx, id, options...)` | `GET OS-EP-FILTER/endpoint_groups/{id}/endpoints` |
| `endpoint_group_projects(endpoint_group, **query)` | `ListProjects(ctx, id, options...)` | `GET OS-EP-FILTER/endpoint_groups/{id}/projects` |
| `project_endpoint_groups(project, **query)` | `ListProjectEndpointGroups(ctx, projectID, options...)` | `GET OS-EP-FILTER/projects/{id}/endpoint_groups` |
| `endpoint_projects(endpoint, **query)` | `ListEndpointProjects(ctx, endpointID, options...)` | `GET OS-EP-FILTER/endpoints/{id}/projects` |
| `associate_project_with_endpoint_group(endpoint_group, project)` | `AssociateProject(ctx, id, projectID, options...)` | `PUT OS-EP-FILTER/endpoint_groups/{id}/projects/{project_id}` |
| `disassociate_project_from_endpoint_group(endpoint_group, project)` | `DisassociateProject(ctx, id, projectID, options...)` | `DELETE OS-EP-FILTER/endpoint_groups/{id}/projects/{project_id}` |

`project_endpoints(project)`의 `OS-EP-FILTER/projects/{id}/endpoints`는 기존 generated [`projectendpoints`](../projectendpoints) package가 담당합니다.

## Cloud helper

| Python `conn` | Go | 동작 |
|---|---|---|
| `create_endpoint_group(name, filters, description=None)` | `CloudCreate(ctx, name, filters, description, options...)` | `name`과 `filters`는 항상 보내고 `description`은 비어 있지 않을 때만 보냄 |
| `update_endpoint_group(endpoint_group_id, **kwargs)` | `Update(ctx, id, WithUpdateField(...)...)` | proxy update와 같은 PATCH |
| `list_endpoint_groups(**kwargs)` | `List(ctx, WithListName(name))` | proxy list와 같은 GET |
| `get_endpoint_group(id)` | `Find(ctx, id)` | proxy find와 같은 조회 |
| `delete_endpoint_group(id)` | `CloudDelete(ctx, id, options...)` | `Find`로 찾지 못하면 DELETE 없이 `false`, 찾으면 인자로 받은 `id`를 DELETE하고 `true` |
| `search_endpoint_groups(name_or_id=None, filters=None)` | `CloudSearch(ctx, nameOrID, options...)` | 전체 목록 뒤 `_utils._filter_list`와 같은 local 선택 |

```python
group = conn.identity.create_endpoint_group(name="web", filters={"interface": "public"})
conn.identity.associate_project_with_endpoint_group(group, project_id)
for endpoint in conn.identity.endpoint_group_endpoints(group):
    print(endpoint.url)
conn.identity.delete_endpoint_group(group)
```

```go
api := endpointgroups.New(identityClient)
group, err := api.Create(ctx,
    endpointgroups.WithCreateField("name", "web"),
    endpointgroups.WithCreateField("filters", map[string]string{"interface": "public"}))
if err != nil {
    return err
}
id := group.Text("id")
if err := api.AssociateProject(ctx, id, projectID); err != nil {
    return err
}
for endpoint, err := range api.ListEndpoints(ctx, id) {
    if err != nil {
        return err
    }
    fmt.Println(endpoint.Text("url"))
}
return api.Delete(ctx, id)
```

## 결과 Record

모든 메서드는 `*Record`를 돌려줍니다. `Resource.Body`는 Python Resource의 `to_dict()`와 같은 attribute 이름의 view입니다. 예를 들어 endpoint의 `enabled`는 `is_enabled`로 바뀌고 Python `type=bool`처럼 truthiness로 변환됩니다. Project의 `options`는 dict, `tags`는 list로 변환되며 `tags`가 없으면 `[]`입니다. 범위가 있는 목록은 URI attribute(`endpoint_group_id`, `project_id`, `endpoint_id`)를 각 row에 넣습니다. Project 계열은 `_allow_unknown_attrs_in_body`가 켜져 있으므로 선언하지 않은 응답 key도 view에 남깁니다. `location`은 connection 정보가 없는 단독 package이므로 항상 JSON `null`입니다. `Wire`는 응답 객체 전체, `Envelope`는 응답 body 전체를 보관하고 `Header`와 `StatusCode`는 실제 응답 값입니다.

`Create`와 `Update`는 선언된 EndpointGroup attribute(`id`, `name`, `description`, `filters`, `links`)만 `endpoint_group` 아래에 보냅니다. 알 수 없는 attribute는 버리며 명시한 `null`, `false`, 빈 값은 그대로 보냅니다. `Update`는 `id`도 버리고, 보낼 attribute가 하나도 없으면 Python `commit`처럼 HTTP 요청 없이 ID만 담긴 view를 돌려줍니다. 응답이 JSON 객체이면 선언된 key가 요청 값을 덮어쓰고, 비어 있거나 JSON이 아니면 요청 값이 남습니다. JSON이지만 객체가 아닌 응답은 실제 응답을 담은 `resource.ResponseError`로 실패합니다.

## 상태 코드와 오류

Python `raise_from_response`처럼 400 미만의 모든 상태를 받아들입니다. 400 이상은 Gophercloud의 `ErrUnexpectedResponseCode`를 원래 status와 body와 함께 돌려줍니다. 모든 오류는 `resource.OperationError`로 감싸며 `Operation`은 Go 메서드 이름, `Resource`는 `endpointgroups`입니다. ID가 비어 있거나 제어 문자를 포함하면 HTTP 전에 `resource.ErrInvalidOption`으로 실패합니다.

`Delete`는 Python의 `ignore_missing=True` 기본값처럼 깨끗한 404를 무시합니다. `WithDeleteIgnoreMissing(false)`를 주면 404를 그대로 돌려줍니다. `DisassociateProject`는 Python에 `ignore_missing` 인자가 없으므로 404를 항상 돌려줍니다. `AssociateProject`와 `DisassociateProject`는 body 없이 요청합니다.

## 목록 query와 local filter

`WithListQuery(key, value)`는 Python list 메서드의 `**query` 하나를 받습니다. 각 목록은 해당 Resource class의 `QueryParameters`에 있는 attribute를 wire query로 보내고, 선언된 그 밖의 Body attribute는 받은 row에 local filter로 적용합니다. 나머지 key는 버립니다. query 값은 Requests와 같은 방식으로 인코딩하므로 `true`는 `True`, 배열은 같은 key의 반복이 됩니다. attribute 이름과 wire 이름이 함께 주어지면 attribute 이름이 우선합니다.

| 메서드 | wire query | local filter |
|---|---|---|
| `List`, `ListProjectEndpointGroups` | `name` | `id`, `description`, `filters`, `links` |
| `ListEndpoints` | `interface`, `region_id`, `service_id`, `limit`, `marker` | `id`, `name`, `is_enabled`, `links`, `url` |
| `ListProjects`, `ListEndpointProjects` | `domain_id`, `is_domain`, `name`, `parent_id`, `is_enabled`(`enabled`), `tags`, `any_tags`(`tags-any`), `not_tags`(`not-tags`), `not_any_tags`(`not-tags-any`), `limit`, `marker` | `id`, `description`, `options`, `links` |

EndpointGroup은 `include_pagination_defaults=False`로 선언되어 있어서 `List`와 `ListProjectEndpointGroups`는 `WithListLimit`와 `WithListMarker`를 Python처럼 버립니다. 다만 `WithListMaxItems(n)`은 Python `max_items`처럼 `limit=n`을 보냅니다. limit가 있으면 마지막 row의 ID를 marker로 삼아 빈 페이지가 나올 때까지 다음 요청을 보내고, raw row 수가 `max_items`에 이르면 멈춥니다. local filter는 이 raw row 상한을 적용한 뒤에 걸립니다. `WithListPaginated(false)`는 첫 페이지만 읽습니다. 목록 요청에는 Python처럼 `Accept: application/json`을 붙입니다.

## Find

`Find`는 먼저 `GET OS-EP-FILTER/endpoint_groups/{name_or_id}`를 보냅니다. 깨끗한 400, 403, 404가 오면 EndpointGroup이 `name` query를 선언하므로 `GET OS-EP-FILTER/endpoint_groups?name={name_or_id}` 목록을 끝까지 읽고 ID나 이름이 정확히 같은 row를 찾습니다. 찾지 못하면 기본값에서는 `nil, nil`을 돌려주고, `WithFindIgnoreMissing(false)`이면 `resource.NotFoundError`를 돌려줍니다. 둘 이상이면 `resource.AmbiguousError`입니다. 500 같은 다른 오류는 목록으로 넘어가지 않습니다.

## Cloud search

`CloudSearch`는 query 없이 `List`를 끝까지 읽은 뒤 `_utils._filter_list`와 같은 규칙으로 고릅니다. `nameOrID`가 비어 있지 않으면 ID나 이름이 정확히 같거나 glob 패턴에 맞는 row만 남깁니다. `WithSearchFilters`의 JSON 객체는 중첩 dictionary filter이며, row에 없는 key를 만나면 Python의 `AttributeError`처럼 `resource.ErrInvalidOption`으로 실패합니다. `WithSearchExpression`은 deprecated JMESPath 문자열이며 결과는 임의의 JSON일 수 있습니다. `SearchResult.Value`는 선택된 view 배열이나 JMESPath 결과이고, `EndpointGroups`는 선택된 record(식일 때는 `nil`), `Inventory`는 읽은 모든 record입니다.

## Python과 다른 점

- Python은 ID 문자열이나 Resource 객체를 받지만 Go는 ID 문자열만 받고, ID를 한 path segment로 escape합니다.
- Python은 resource와 client의 최대 microversion을 협상하지만 Keystone v3에는 microversion이 없으며 Go는 선택한 client 설정을 그대로 씁니다.
- Python의 일반 paginator는 Keystone의 `links: {"next": ...}` 형식을 무시하지만, Go는 기존 SDK 소유 identity 목록과 같이 이 링크를 따라갑니다. 그 밖에 top-level `next`, HTTP `Link` header와 limit marker 반복은 Python과 같습니다.
- local filter는 JSON boolean과 숫자를 구분합니다. Python에서는 `True == 1`이 참입니다.
- `location`은 항상 `null`입니다. Python은 connection의 cloud, region, project 정보를 넣습니다.
- `CloudDelete`는 DELETE가 실패하면 `false`와 그 오류를 함께 돌려줍니다. Python은 오류를 log로 남기고 `False`만 돌려줍니다.
- cloud `update_endpoint_group`과 `list_endpoint_groups`는 `valid_kwargs`로 허용하지 않은 인자를 HTTP 전에 거부합니다. Go의 `Update`와 `List`는 proxy 규칙을 따르므로 다른 attribute를 버리거나 local filter로 씁니다.
- Python cloud helper는 list를 즉시 만들어 돌려주지만 Go의 `List`는 lazy iterator입니다.
- `WithReadHeaders`, `WithCreateHeaders` 같은 header option은 Go 확장입니다. 인증, host, 연결 관련 header는 SDK가 관리하므로 지정하면 HTTP 전에 `resource.ErrInvalidOption`으로 실패합니다.
