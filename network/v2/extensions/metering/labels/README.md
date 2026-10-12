# Neutron metering label

이 패키지는 pinned openstacksdk의 network metering label Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0에는 Neutron `/v2.0/metering/metering-labels` 바인딩이 없으므로, 이 패키지가 SDK에서 직접 요청을 보냅니다. 기준 소스는 [`_proxy.py` Metering Labels 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L2861-L3002)과 [`metering_label.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label.py)입니다. label에 속한 rule은 [`rules`](../rules/README.md) 패키지에 있습니다.

| Python `conn.network` | Go `labels.API` | 요청 |
|---|---|---|
| `create_metering_label(**attrs)` | `Create(ctx, options...)` | `POST metering/metering-labels` |
| `get_metering_label(metering_label, fields=None)` | `Get(ctx, id, options...)` | `GET metering/metering-labels/{id}` |
| `update_metering_label(metering_label, **attrs)` | `Update(ctx, id, options...)` | `PUT metering/metering-labels/{id}` |
| `delete_metering_label(metering_label, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE metering/metering-labels/{id}` |
| `metering_labels(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET metering/metering-labels` |
| `find_metering_label(name_or_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | GET 뒤 목록 |

```python
label = conn.network.create_metering_label(name="web", is_shared=False)
for item in conn.network.metering_labels(project_id=project_id):
    print(item.name, item.is_shared)
conn.network.delete_metering_label(label)
```

```go
api := labels.New(networkClient)
label, err := api.Create(ctx,
    labels.WithCreateField("name", "web"),
    labels.WithCreateField("is_shared", false))
if err != nil { return err }
for item, err := range api.List(ctx, labels.WithListQuery("project_id", projectID)) {
    if err != nil { return err }
    fmt.Println(*item.Name, item.IsShared != nil && *item.IsShared)
}
_, err = api.Delete(ctx, label.ID) // 없는 label은 무시합니다
```

## 요청과 응답

생성은 `{"metering_label": {...}}` 본문으로 POST합니다. 키로 Python 속성 이름(`is_shared`, `project_id`)과 Body 키(`shared`)를 모두 쓸 수 있고, 둘 다 Body 키로 바뀌어 전송됩니다. 선언되지 않은 키도 본문에 그대로 넣으며 Resource 제어 이름은 HTTP 전에 거부합니다.

수정은 지정한 속성만 PUT 본문에 담고, 보낼 속성이 없으면 요청 없이 ID만 채운 결과를 돌려줍니다. `id`와 `metering_label`은 속성으로 줄 수 없습니다. `Get`의 `WithGetFields`는 Python 이름을 Body 키로 바꿔 `fields` query로 보냅니다(`is_shared`는 `shared`가 됩니다).

Go는 Python처럼 200부터 399까지를 성공으로 받습니다. `MeteringLabel`의 typed 필드는 JSON 타입이 맞을 때만 채워지고, `project_id`는 값이 없으면 `tenant_id`를 대신 읽습니다. `Resource`는 Source 속성을 Python 이름으로, `Wire`와 `Envelope`는 실제 응답을 보관합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `meteringlabels`)로 감쌉니다.

## 삭제

`Delete`는 Python 기본값 `ignore_missing=True`를 따라 깨끗한 404를 성공으로 바꾸고 `IgnoredMissing`을 표시합니다. `WithDeleteIgnoreMissing(false)`이면 404도 오류이고, 다른 4xx와 5xx는 언제나 오류입니다.

## 목록

`List`는 lazy iterator입니다. `limit`, `marker`, `description`, `fields`, `name`, `project_id`, `sort_key`, `sort_dir`, `is_shared`(`shared`로 전송)는 server query가 되고, 값은 Requests 방식으로 인코딩하므로 `true`는 `shared=True`로 나갑니다. `id`와 `tenant_id`는 Python처럼 응답 행을 로컬에서 거르고 그 밖의 이름은 버립니다. 페이지는 `metering_labels_links`의 next 링크를 따라가며 링크 query를 원래 query 위에 합치고, limit만큼 행이 오면 마지막 ID를 marker로 이어 갑니다. `WithListMaxItems`와 `WithListPaginated(false)`도 쓸 수 있습니다.

## 찾기

`Find`는 먼저 `GET metering/metering-labels/{name_or_id}`를 보내고, 깨끗한 400, 403, 404이면 목록을 끝까지 읽어 ID나 이름이 같은 항목을 고릅니다. query mapping에 `name`이 있으므로 Python과 같이 목록 요청에 `name=<name_or_id>`를 붙입니다. 없으면 기본으로 `nil, nil`이고 `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`입니다.

## Python과 다른 점

Go는 ID 문자열만 받고, ID를 URL segment 하나로 escape하며 빈 ID와 제어 문자를 HTTP 전에 거부합니다. 마이크로버전은 협상하지 않고 설정된 client 값을 보내며, 헤더와 마이크로버전 옵션은 Go 확장입니다. 삭제 결과의 `DeleteResult`는 Python의 None 대신 돌려주는 Go 편의 값이고, `find_metering_label`의 deprecated 경고는 재현하지 않습니다.
