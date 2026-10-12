# Neutron default security group rule

`defaultrules.API`는 Gophercloud v2.15.0이 바인딩하지 않는 Neutron `/default-security-group-rules` 경로(security group rule template)를 SDK가 직접 호출하는 API입니다. pinned openstacksdk의 default security group rule 메서드 5개를 제공하며, `api := defaultrules.New(networkClient)`처럼 Neutron service client로 만듭니다. Python 클래스가 `allow_commit = False`이므로 update 메서드는 없습니다.

## Proxy 메서드

| Python `conn.network` | Go | 요청 |
|---|---|---|
| `create_default_security_group_rule(**attrs)` | `Create(ctx, options...)` | `POST /v2.0/default-security-group-rules` |
| `get_default_security_group_rule(rule, fields=None)` | `Get(ctx, id, options...)` | `GET /v2.0/default-security-group-rules/{id}` |
| `delete_default_security_group_rule(rule, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE /v2.0/default-security-group-rules/{id}` |
| `default_security_group_rules(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET /v2.0/default-security-group-rules` |
| `find_default_security_group_rule(name_or_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | ID로 GET한 뒤 전체 목록 |

```python
rule = conn.network.create_default_security_group_rule(
    direction="ingress", ether_type="IPv4", protocol="tcp",
    port_range_min=22, port_range_max=22, used_in_default_sg=True)
for item in conn.network.default_security_group_rules(ether_type="IPv4"):
    print(item.id, item.port_range_min)
conn.network.delete_default_security_group_rule(rule)
```

```go
api := defaultrules.New(networkClient)
rule, err := api.Create(ctx,
    defaultrules.WithCreateField("direction", "ingress"),
    defaultrules.WithCreateField("ether_type", "IPv4"),
    defaultrules.WithCreateField("protocol", "tcp"),
    defaultrules.WithCreateField("port_range_min", 22),
    defaultrules.WithCreateField("port_range_max", 22),
    defaultrules.WithCreateField("used_in_default_sg", true))
if err != nil {
    return err
}
for item, err := range api.List(ctx, defaultrules.WithListQuery("ether_type", "IPv4")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["port_range_min"]))
}
_, err = api.Delete(ctx, rule.Text("id"))
return err
```

## 결과 Record

`Resource.Body`는 Python `to_dict()`와 같은 attribute 이름의 view입니다. DefaultSecurityGroupRule은 `NetworkResource`를 상속하므로 `revision_number`는 정수로 변환되고, `if_match` header attribute가 view에 들어갑니다. `port_range_min`과 `port_range_max`는 정수, `used_in_default_sg`와 `used_in_non_default_sg`는 truthiness로 bool이 됩니다. wire 이름 `ethertype`은 view에서 `ether_type`으로 바뀌며 원래 key는 `Wire`에만 남습니다. 선언하지 않은 응답 key는 `NetworkResource`가 허용하므로 view에 남습니다. 응답에 없는 선언된 attribute와 `if_match`는 `null`이고, `location`은 항상 `null`입니다. `Envelope`, `Header`, `StatusCode`는 실제 응답 값입니다.

## 생성

`Create`는 `WithCreateField`로 준 attribute만 `default_security_group_rule` 아래에 보냅니다. `ether_type`은 `ethertype`으로 바꿔 보내고, 명시한 `null`, `false`, `0`과 선언하지 않은 attribute도 그대로 보냅니다. attribute가 없으면 `{"default_security_group_rule":{}}`를 보내며 필수 값 검사는 Neutron이 합니다. `if_match`(또는 `if-match`) attribute와 `WithCreateIfMatch(values...)`는 Body가 아니라 `If-Match` header가 됩니다. Python `_prepare_request`처럼 list는 `", "`로 잇고, 문자열은 그대로, `null`은 `None`이 됩니다. 그 밖의 JSON 값은 HTTP 전에 `resource.ErrInvalidOption`입니다. `microversion` attribute는 버리고 Resource 제어 이름은 거부합니다.

## 조회, 삭제, 상태 코드

`Get`의 `WithGetFields(...)`는 Python `_get_with_fields`처럼 attribute 이름을 wire 이름으로 바꿔 보냅니다. 예를 들어 `ether_type`은 `fields=ethertype`이 됩니다. `Delete`는 깨끗한 404를 기본으로 무시하고 `DeleteResult.IgnoredMissing`을 채우며, `WithDeleteIgnoreMissing(false)`이면 404를 돌려줍니다. 200부터 399까지를 성공으로 받고, 400 이상은 원래 status와 body를 담은 Gophercloud `ErrUnexpectedResponseCode`입니다. 모든 오류는 `resource.OperationError`(`Resource`는 `defaultrules`)로 감쌉니다.

## 목록과 find

서버 query는 `QueryParameters`에 선언된 `id`, `description`, `fields`, `remote_group_id`, `remote_address_group_id`, `direction`, `protocol`, `port_range_min`, `port_range_max`, `remote_ip_prefix`, `used_in_default_sg`, `used_in_non_default_sg`, `sort_dir`, `sort_key`, `limit`, `marker`와 `ether_type`(wire `ethertype`)입니다. `name`과 `revision_number`는 선언된 attribute지만 query가 아니므로 local filter로 적용하고, 선언되지 않은 key는 버립니다. 값은 Requests처럼 인코딩해서 `true`는 `True`가 됩니다. `List`는 `default_security_group_rules_links`, top-level `next`, HTTP `Link` header를 따라가고, 링크가 없고 limit가 있으면 marker로 빈 페이지까지 읽습니다. `WithListMaxItems`는 raw row 상한이고 `WithListPaginated(false)`는 첫 페이지만 읽습니다. 목록 요청에는 `Accept: application/json`을 붙입니다.

`Find`는 먼저 `GET /default-security-group-rules/{name_or_id}`를 보냅니다. 깨끗한 400, 403, 404가 오면 목록을 끝까지 읽고 ID나 `name`이 정확히 같은 row를 찾습니다. 클래스가 `name` query를 선언하지 않으므로 Python처럼 목록에 `name` query를 붙이지 않습니다. `WithFindQuery`로 준 값은 첫 GET의 query와 목록 query 양쪽에 쓰입니다. 찾지 못하면 기본은 `nil, nil`, `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`이고, 둘 이상이면 `resource.ErrAmbiguous`입니다.

## Python과 다른 점

- Python은 ID 문자열이나 Resource 객체를 받지만 Go는 ID 문자열만 받고 한 path segment로 escape합니다.
- Neutron에는 microversion이 없어서 Python은 version header를 보내지 않습니다. Go는 선택한 client에 `Microversion`이 설정되어 있을 때만 `OpenStack-API-Version`을 보내며, `With*Microversion`과 `With*Headers`는 Go 확장입니다.
- `if_match`에 숫자 같은 JSON 값을 주면 Python은 `str()`로 보내지만 Go는 HTTP 전에 거부합니다.
- 다음 페이지 링크가 처음 filter를 바꾸거나 같은 페이지를 반복하면 Go는 실패합니다. Go는 링크를 따라갈 때 처음 filter를 다시 붙입니다.
- local filter의 숫자 비교는 정확한 10진수 비교입니다.
- `location`은 항상 `null`이고, 응답 status 100부터 199는 성공으로 받지 않습니다.
- Python delete는 `None`을 돌려주지만 Go는 실제 응답을 담은 `DeleteResult`를 돌려줍니다.
