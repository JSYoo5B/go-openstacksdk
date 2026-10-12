# Neutron security groups default statefulness

`defaultstatefulness.API`는 Gophercloud v2.15.0이 바인딩하지 않는 Neutron `/security-groups-default-statefulness` 경로를 SDK가 직접 호출하는 API입니다. 새 security group의 `stateful` 기본값을 project별로, 또는 `project_id`가 `null`이면 전체 시스템에 대해 정하는 설정입니다. pinned openstacksdk의 메서드 6개를 제공하며 `api := defaultstatefulness.New(networkClient)`처럼 Neutron service client로 만듭니다.

## Proxy 메서드

| Python `conn.network` | Go | 요청 |
|---|---|---|
| `create_security_groups_default_statefulness(**attrs)` | `Create(ctx, options...)` | `POST /v2.0/security-groups-default-statefulness` |
| `get_security_groups_default_statefulness(value, fields=None)` | `Get(ctx, id, options...)` | `GET .../{id}` |
| `update_security_groups_default_statefulness(value, **attrs)` | `Update(ctx, id, options...)` | `PUT .../{id}` |
| `delete_security_groups_default_statefulness(value, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE .../{id}` |
| `security_groups_default_statefulness(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET /v2.0/security-groups-default-statefulness` |
| `find_security_groups_default_statefulness(name_or_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | ID로 GET한 뒤 전체 목록 |

```python
setting = conn.network.create_security_groups_default_statefulness(
    project_id=project.id, stateful=False)
conn.network.update_security_groups_default_statefulness(
    setting, stateful=True, if_revision=setting.revision_number)
for item in conn.network.security_groups_default_statefulness(project_id=project.id):
    print(item.id, item.stateful)
conn.network.delete_security_groups_default_statefulness(setting)
```

```go
api := defaultstatefulness.New(networkClient)
setting, err := api.Create(ctx,
    defaultstatefulness.WithCreateField("project_id", projectID),
    defaultstatefulness.WithCreateField("stateful", false))
if err != nil {
    return err
}
id := setting.Text("id")
var revision int64
_ = json.Unmarshal(setting.Resource.Body["revision_number"], &revision)
if _, err := api.Update(ctx, id,
    defaultstatefulness.WithUpdateField("stateful", true),
    defaultstatefulness.WithUpdateRevision(revision)); err != nil {
    return err
}
for item, err := range api.List(ctx, defaultstatefulness.WithListQuery("project_id", projectID)) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["stateful"]))
}
_, err = api.Delete(ctx, id)
return err
```

## 결과 Record

`Resource.Body`는 Python `to_dict()`와 같은 view로 `id`, `name`, `revision_number`, `project_id`, `stateful`, `if_match`, `location`을 담습니다. 클래스는 `NetworkResource`를 상속하므로 `revision_number`는 정수로, `stateful`은 truthiness로 bool이 되고, `if_match` header attribute는 요청에서 준 값이 없으면 `null`입니다. 선언하지 않은 응답 key(`tenant_id` 등)도 view에 남습니다. `project_id`에는 `tenant_id` alias가 없습니다. `location`은 항상 `null`이며 `Wire`, `Envelope`, `Header`, `StatusCode`는 실제 응답 값입니다.

## 생성과 수정

`Create`와 `Update`는 준 attribute만 `security_group_default_statefulness` 아래에 보냅니다. 명시한 `null`, `false`, `0`과 선언하지 않은 attribute도 그대로 보내며, attribute가 없는 `Create`는 `{"security_group_default_statefulness":{}}`를 보냅니다. `if_match`(또는 `if-match`) attribute, `WithCreateIfMatch`, `WithUpdateIfMatch`는 Body가 아니라 `If-Match` header가 되고, list는 `", "`로 이어집니다.

`Update`의 `WithUpdateRevision(n)`과 `WithUpdateField("if_revision", n)`은 Python `update_security_groups_default_statefulness(..., if_revision=n)`과 같습니다. Python `Proxy._update`처럼 순서와 관계없이 `if_match`를 `revision_number=n`으로 바꾸고 `If-Match: revision_number=n`을 보냅니다. `if_revision`이 `null`이면 Python처럼 쓰지 않으며 정수가 아니면 HTTP 전에 거부합니다. Python `commit`은 dirty header가 있으면 Body가 비어도 요청하므로, Go도 `If-Match`만 있으면 `{"security_group_default_statefulness":{}}`로 PUT합니다. Body와 header가 모두 없으면 HTTP 없이 ID만 담긴 view를 돌려줍니다. `id` attribute는 update 대상과 겹치므로 거부합니다.

## 조회, 삭제, 목록, find

`Get`의 `WithGetFields(...)`는 `fields` query를 반복해서 보냅니다. `Delete`는 깨끗한 404를 기본으로 무시하고 `DeleteResult.IgnoredMissing`을 채우며, `WithDeleteIgnoreMissing(false)`이면 404를 돌려줍니다. 200부터 399까지를 성공으로 받고, 400 이상은 원래 status와 body를 담은 Gophercloud `ErrUnexpectedResponseCode`입니다. 모든 오류는 `resource.OperationError`(`Resource`는 `defaultstatefulness`)로 감쌉니다.

서버 query는 `fields`, `id`, `project_id`, `stateful`, `sort_dir`, `sort_key`, `limit`, `marker`입니다. `name`과 `revision_number`는 local filter로 적용하고 선언되지 않은 key는 버립니다. `List`는 `security_groups_default_statefulness_links`, top-level `next`, HTTP `Link` header를 따라가고, 링크가 없고 limit가 있으면 marker로 빈 페이지까지 읽습니다. `WithListMaxItems`는 raw row 상한이고 `WithListPaginated(false)`는 첫 페이지만 읽습니다.

`Find`는 먼저 ID로 GET하고, 깨끗한 400, 403, 404가 오면 전체 목록에서 ID나 `name`이 정확히 같은 row를 찾습니다. 클래스가 `name` query를 선언하지 않으므로 목록에 `name` query를 붙이지 않습니다. 찾지 못하면 기본은 `nil, nil`, `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`이고, 둘 이상이면 `resource.ErrAmbiguous`입니다.

## Python과 다른 점

- Python은 ID 문자열이나 Resource 객체를 받지만 Go는 ID 문자열만 받고 한 path segment로 escape합니다.
- Neutron에는 microversion이 없어서 Python은 version header를 보내지 않습니다. Go는 선택한 client에 `Microversion`이 설정되어 있을 때만 `OpenStack-API-Version`을 보내며, `With*Microversion`과 `With*Headers`는 Go 확장입니다.
- Python은 `if_revision`에 어떤 값이든 문자열로 넣지만 Go는 JSON 정수만 받습니다. `if_match`의 숫자 같은 JSON 값도 HTTP 전에 거부합니다.
- 다음 페이지 링크가 처음 filter를 바꾸거나 같은 페이지를 반복하면 Go는 실패합니다.
- `location`은 항상 `null`이고, 응답 status 100부터 199는 성공으로 받지 않습니다.
- Python delete는 `None`을 돌려주지만 Go는 실제 응답을 담은 `DeleteResult`를 돌려줍니다.
