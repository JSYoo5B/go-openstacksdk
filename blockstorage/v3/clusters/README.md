# Cinder cluster

이 패키지는 pinned openstacksdk의 cluster Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0에는 `clusters` 바인딩이 없으므로 이 패키지가 요청을 SDK에서 직접 보냅니다. 기준 소스는 [`_proxy.py` Clusters 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1957-L2065)과 [`cluster.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/cluster.py)입니다.

| Python `conn.block_storage` | Go `clusters.API` | 요청 |
|---|---|---|
| `get_cluster(cluster)` | `Get(ctx, name, options...)` | `GET clusters/{name}` |
| `find_cluster(name_or_id, ignore_missing=True)` | `Find(ctx, nameOrID, options...)` | GET 뒤 `clusters/detail?name=...` 목록 |
| `clusters(details=True, **query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET clusters/detail` 또는 `GET clusters` |
| `enable_cluster(cluster)` | `Enable(ctx, target, options...)` | `PUT clusters/enable` |
| `disable_cluster(cluster, reason=None)` | `Disable(ctx, target, options...)` | `PUT clusters/disable` |

```python
cluster = conn.block_storage.get_cluster("cluster1@lvm")
conn.block_storage.disable_cluster(cluster, reason="maintenance")
```

```go
api := clusters.New(volumeClient)
cluster, err := api.Get(ctx, "cluster1@lvm")
if err != nil { return err }
_, err = api.Disable(ctx, cluster.Target(), clusters.WithDisableReason("maintenance"))
```

## Microversion

`Get`, `List`, `Find`는 `Cluster._get_microversion`을 따릅니다. client에 microversion이 설정되어 있으면 discovery 없이 그 값을 보내고, 없으면 Cinder version 문서를 한 번 읽어 3.26과 서버 최대값 중 작은 값을 `OpenStack-API-Version: volume X`와 `X-OpenStack-Volume-API-Version: X`로 보냅니다. `Enable`과 `Disable`은 Python `Cluster._action`처럼 `session.default_microversion`, 즉 client에 설정된 microversion만 보내고 discovery는 하지 않습니다. 설정이 없으면 microversion 헤더 없이 보내므로 Cinder가 요구하는 3.7 미만으로 처리될 수 있으며, 이는 Python과 같은 동작입니다.

## 요청과 응답

`Cluster`는 `name`을 alternate id로 씁니다. `Get(ctx, name)`은 Python처럼 요청한 값을 `id`로 채운 뒤 응답 `cluster` 객체의 선언된 키를 덮어씁니다. `Body`는 선언된 Body 키 14개를 raw JSON으로 보관하고 값이 없으면 null입니다. `Binary`, `State`, `Frozen`, `NumHosts` 같은 typed 필드는 JSON 타입이 맞을 때만 채워집니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 비어 있거나 JSON이 아닌 본문은 요청 값을 그대로 두고, JSON이지만 객체가 아니면 실제 응답을 담은 `resource.ResponseError`로 실패합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고 모든 오류는 `resource.OperationError`(Resource `clusters`)로 감쌉니다.

`List`는 기본으로 `clusters/detail`을 읽고 `WithListDetails(false)`이면 `clusters`를 읽습니다. `WithListQuery`로 넘긴 `name`, `binary`, `is_up`, `disabled`, `num_hosts`, `num_down_hosts`, `replication_status`, `frozen`, `active_backend_id`는 Requests 방식으로 인코딩한 server query가 됩니다. `id`, `created_at`, `disabled_reason`, `last_heartbeat`, `state`, `status`, `updated_at`은 Python처럼 응답 행을 로컬에서 거르고 그 밖의 이름은 무시합니다. limit, marker, max_items, paginated 옵션도 Python과 같은 규칙으로 동작합니다.

`Find`는 먼저 `GET clusters/{name_or_id}`를 보냅니다. 응답이 깨끗한 400, 403, 404이면 Python `Resource.find`가 `name`을 query에 넣는 규칙에 따라 `GET clusters/detail?name={name_or_id}` 전체 목록에서 ID나 이름이 정확히 같은 항목을 고릅니다. 없으면 기본으로 `nil, nil`이고, `WithFindIgnoreMissing(false)`이면 `resource.ErrNotFound`입니다.

## Enable과 Disable

Python은 cluster 인스턴스의 `name`과 `binary`로 본문을 만듭니다. Go는 이 두 값을 `Target`으로 받습니다. `NewTarget(name, binary)`로 직접 만들거나 `Get`, `Find`, `List` 결과의 `Cluster.Target()`을 쓸 수 있습니다. 본문은 `{"name": name}`이고 binary가 비어 있지 않으면 `binary`, `WithDisableReason`의 사유가 비어 있지 않으면 `disabled_reason`을 더합니다. 응답의 `cluster` 객체는 보낸 `name`과 `binary` 위에 덮어씁니다.

Python에 cluster를 문자열로 넘기면 그 문자열은 `id`가 되고 `name`은 비어 있으므로 본문은 `{"name": null}`이 됩니다. Go에서 같은 요청이 필요하면 `Name`이 nil인 `Target{}`을 넘기면 됩니다.

## Python과 다른 점

Python은 Resource 인스턴스나 문자열을 받지만 Go는 `Get`과 `Find`에 이름 문자열을, 동작에는 `Target`을 받습니다. Go는 이름을 URL segment 하나로 escape하고 빈 값, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. 로컬 필터는 JSON 값을 비교하므로 Python의 타입 변환은 하지 않습니다. `find_cluster`의 deprecated 경고와 Resource의 `location` 속성은 재현하지 않습니다. 헤더 옵션은 Go 확장이며 인증, microversion, content 헤더는 SDK가 소유합니다.
