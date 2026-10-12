# Neutron metering label rule

이 패키지는 pinned openstacksdk의 network metering label rule Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0에는 Neutron `/v2.0/metering/metering-label-rules` 바인딩이 없으므로, 이 패키지가 SDK에서 직접 요청을 보냅니다. 기준 소스는 [`_proxy.py` Metering Label Rules 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3005-L3157)과 [`metering_label_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label_rule.py)입니다. rule이 속한 label은 [`labels`](../labels/README.md) 패키지에 있습니다.

| Python `conn.network` | Go `rules.API` | 요청 |
|---|---|---|
| `create_metering_label_rule(**attrs)` | `Create(ctx, options...)` | `POST metering/metering-label-rules` |
| `get_metering_label_rule(metering_label_rule, fields=None)` | `Get(ctx, id, options...)` | `GET metering/metering-label-rules/{id}` |
| `update_metering_label_rule(metering_label_rule, **attrs)` | `Update(ctx, id, options...)` | `PUT metering/metering-label-rules/{id}` |
| `delete_metering_label_rule(metering_label_rule, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE metering/metering-label-rules/{id}` |
| `metering_label_rules(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET metering/metering-label-rules` |
| `find_metering_label_rule(name_or_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | GET 뒤 목록 |

```python
rule = conn.network.create_metering_label_rule(
    metering_label_id=label.id, direction="egress",
    destination_ip_prefix="0.0.0.0/0", is_excluded=False)
for item in conn.network.metering_label_rules(metering_label_id=label.id):
    print(item.direction, item.destination_ip_prefix)
conn.network.delete_metering_label_rule(rule)
```

```go
api := rules.New(networkClient)
rule, err := api.Create(ctx,
    rules.WithCreateField("metering_label_id", labelID),
    rules.WithCreateField("direction", "egress"),
    rules.WithCreateField("destination_ip_prefix", "0.0.0.0/0"),
    rules.WithCreateField("is_excluded", false))
if err != nil { return err }
for item, err := range api.List(ctx, rules.WithListQuery("metering_label_id", labelID)) {
    if err != nil { return err }
    fmt.Println(*item.Direction, *item.DestinationIPPrefix)
}
_, err = api.Delete(ctx, rule.ID) // 없는 rule은 무시합니다
```

## 요청과 응답

생성은 `{"metering_label_rule": {...}}` 본문으로 POST합니다. 키로 Python 속성 이름(`is_excluded`, `project_id`)과 Body 키(`excluded`)를 모두 쓸 수 있고, 둘 다 Body 키로 바뀌어 전송됩니다. Python에서 deprecated인 `remote_ip_prefix`도 그대로 보낼 수 있습니다. 선언되지 않은 키도 본문에 넣으며 Resource 제어 이름은 HTTP 전에 거부합니다.

수정은 지정한 속성만 PUT 본문에 담고, 보낼 속성이 없으면 요청 없이 ID만 채운 결과를 돌려줍니다. `id`와 `metering_label_rule`은 속성으로 줄 수 없습니다. `Get`의 `WithGetFields`는 Python 이름을 Body 키로 바꿔 `fields` query로 보냅니다(`is_excluded`는 `excluded`가 됩니다).

Go는 Python처럼 200부터 399까지를 성공으로 받습니다. `MeteringLabelRule`의 typed 필드는 JSON 타입이 맞을 때만 채워지고, `project_id`는 값이 없으면 `tenant_id`를 대신 읽습니다. `Resource`는 Source 속성을 Python 이름으로, `Wire`와 `Envelope`는 실제 응답을 보관합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `meteringlabelrules`)로 감쌉니다.

## 삭제

`Delete`는 Python 기본값 `ignore_missing=True`를 따라 깨끗한 404를 성공으로 바꾸고 `IgnoredMissing`을 표시합니다. `WithDeleteIgnoreMissing(false)`이면 404도 오류이고, 다른 4xx와 5xx는 언제나 오류입니다.

## 목록

`List`는 lazy iterator입니다. `limit`, `marker`, `direction`, `fields`, `metering_label_id`, `remote_ip_prefix`, `source_ip_prefix`, `destination_ip_prefix`, `project_id`, `sort_key`, `sort_dir`는 server query가 됩니다. `is_excluded`는 query mapping에 없으므로 Python과 같이 `id`, `name`, `tenant_id`와 함께 응답 행을 로컬에서 거르는 데 쓰이고, Body 키인 `excluded`처럼 선언되지 않은 이름은 버립니다. 페이지는 `metering_label_rules_links`의 next 링크를 따라가며 링크 query를 원래 query 위에 합치고, limit만큼 행이 오면 마지막 ID를 marker로 이어 갑니다. `WithListMaxItems`와 `WithListPaginated(false)`도 쓸 수 있습니다.

## 찾기

`Find`는 먼저 `GET metering/metering-label-rules/{name_or_id}`를 보내고, 깨끗한 400, 403, 404이면 목록을 끝까지 읽어 ID나 이름이 같은 항목을 고릅니다. query mapping에 `name`이 없으므로 목록 요청에 이름 query를 붙이지 않습니다. `WithFindQuery`의 값(예를 들어 `metering_label_id`)은 Python과 같이 member GET과 목록 양쪽에 query로 붙습니다. 없으면 기본으로 `nil, nil`이고 `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`입니다.

## Python과 다른 점

Go는 ID 문자열만 받고, ID를 URL segment 하나로 escape하며 빈 ID와 제어 문자를 HTTP 전에 거부합니다. 마이크로버전은 협상하지 않고 설정된 client 값을 보내며, 헤더와 마이크로버전 옵션은 Go 확장입니다. Python은 `remote_ip_prefix`를 쓰면 deprecation 경고를 내지만 Go는 경고 없이 보냅니다. 삭제 결과의 `DeleteResult`는 Python의 None 대신 돌려주는 Go 편의 값이고, `find_metering_label_rule`의 deprecated 경고는 재현하지 않습니다.
