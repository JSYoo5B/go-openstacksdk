# Neutron networking-sfc port chain

이 패키지는 pinned openstacksdk의 `sfc_port_chain` Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0에는 networking-sfc 확장의 `/v2.0/sfc/port_chains` 바인딩이 없으므로, 이 패키지가 요청을 SDK에서 직접 보냅니다. 기준 소스는 [`_proxy.py` port chain 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9367-L9499)과 [`sfc_port_chain.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_chain.py)입니다. 다섯 networking-sfc 패키지는 같은 내부 구현을 공유하므로 요청 규칙이 모두 같습니다.

| Python `conn.network` | Go `portchains.API` | 요청 |
|---|---|---|
| `create_sfc_port_chain(**attrs)` | `Create(ctx, options...)` | `POST /v2.0/sfc/port_chains` |
| `get_sfc_port_chain(port_chain)` | `Get(ctx, id, options...)` | `GET /v2.0/sfc/port_chains/{id}` |
| `update_sfc_port_chain(port_chain, **attrs)` | `Update(ctx, id, options...)` | `PUT /v2.0/sfc/port_chains/{id}` |
| `delete_sfc_port_chain(port_chain, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE /v2.0/sfc/port_chains/{id}` |
| `sfc_port_chains(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET /v2.0/sfc/port_chains` |
| `find_sfc_port_chain(name_or_id, ignore_missing=True, **query)` | `Find(ctx, nameOrID, options...)` | GET 뒤 `name` 목록 |

```python
pc = conn.network.create_sfc_port_chain(
    name="web-chain", port_pair_groups=[ppg.id], flow_classifiers=[fc.id],
    chain_parameters={"correlation": "mpls", "symmetric": False})
conn.network.update_sfc_port_chain(pc, flow_classifiers=[fc.id, fc2.id])
conn.network.delete_sfc_port_chain(pc)
```

```go
api := portchains.New(networkClient)
pc, err := api.Create(ctx,
    portchains.WithCreateField("name", "web-chain"),
    portchains.WithCreateField("port_pair_groups", []string{ppg.ID}),
    portchains.WithCreateField("flow_classifiers", []string{fc.ID}),
    portchains.WithCreateField("chain_parameters", map[string]any{"correlation": "mpls", "symmetric": false}))
if err != nil { return err }
_, err = api.Update(ctx, pc.ID, portchains.WithUpdateField("flow_classifiers", []string{fc.ID, fc2.ID}))
if err != nil { return err }
err = api.Delete(ctx, pc.ID) // 없는 port chain은 무시합니다
```

## 요청과 응답

생성은 Python `Resource.create`처럼 `POST /v2.0/sfc/port_chains`에 `{"port_chain": {...}}` 본문을 보냅니다. `WithCreateField`의 키로 Python 속성 이름과 Body 키를 모두 쓸 수 있고, 선언되지 않은 키는 Python `Resource.new`처럼 버립니다. POST 생성이므로 `id`를 주면 본문에 그대로 들어갑니다. 값은 JSON 그대로 보내므로 null, false, 빈 목록도 그대로 전달됩니다. 같은 Body 키에 서로 다른 값을 주면 Python은 둘 중 하나를 임의로 고르지만 Go는 HTTP 전에 거부합니다.

수정은 Python `Resource.commit`의 기본 `commit_method`인 PUT으로 지정한 속성만 보냅니다. ID는 dirty 속성이 될 수 없으므로 `id` 필드는 무시하고, 보낼 속성이 없으면 요청 없이 ID만 채운 결과를 돌려줍니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답 본문이 `port_chain` 객체를 가지면 그 안을, 아니면 최상위 객체를 읽어 선언된 키만 요청 값 위에 덮어씁니다. 비어 있거나 JSON이 아닌 본문은 요청 값을 그대로 두고, JSON이지만 객체가 아니면 실제 응답을 담은 `resource.ResponseError`로 실패합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `port_chains`, Operation `Create`, `Get`, `Update`, `Delete`, `List`, `Find`)로 감쌉니다.

`PortChain`의 typed 필드는 JSON 타입이 맞을 때만 채워집니다. `Body`는 선언된 Body 키 8개(`id`, `name`, `description`, `project_id`, `tenant_id`, `port_pair_groups`, `flow_classifiers`, `chain_parameters`)를 Python 속성 값으로 보관하고 값이 없으면 null입니다. `port_pair_groups`와 `flow_classifiers`는 Python `type=list`라서 목록이 아닌 값은 한 원소짜리 목록으로 감싸고, `chain_parameters`는 `type=dict`라서 객체가 아니면 `{}`가 됩니다. `project_id`는 Python alias처럼 리소스가 `project_id`를 받은 적이 없을 때만 `tenant_id` 값을 씁니다. 서버가 `project_id: null`을 보내면 null을 그대로 둡니다. `Wire`는 실제 응답 객체 전체, `Envelope`는 실제 응답 본문입니다.

## 삭제

`Delete`는 Python 기본값 `ignore_missing=True`를 따라 깨끗한 404를 nil로 바꿉니다. `WithDeleteIgnoreMissing(false)`를 주면 404도 오류로 돌려주고, 409나 500 같은 다른 status는 언제나 오류입니다.

## 목록

`List`는 lazy iterator이며 반복할 때마다 요청을 다시 보냅니다. Python의 `**query`는 `WithListQuery(key, value)`로 넘깁니다. QueryParameters에 있는 `name`, `description`, `project_id`, `tenant_id`와 `limit`, `marker`는 Requests 방식으로 인코딩한 server query가 됩니다. QueryParameters에 없는 Body 속성(`id`, `port_pair_groups`, `flow_classifiers`, `chain_parameters`)은 Python `Resource.list`처럼 응답 행을 로컬에서 거르고, 그 밖의 이름은 무시합니다. 목록 속성은 목록 전체가 같아야 하고, `chain_parameters`를 객체로 거르면 Python `_dict_filter`와 같이 지정한 키만 비교합니다. `WithListName`과 `WithListProjectID`는 자주 쓰는 query의 편의 함수입니다. 요청에는 `Accept: application/json`이 붙습니다.

다음 페이지는 Neutron이 보내는 `port_chains_links`의 `rel: next` 링크를 따라갑니다. `/v2.0/...`처럼 버전으로 시작하는 상대 링크는 Python과 같이 endpoint 아래 경로로 풀고, HTTP `Link` 헤더도 따릅니다. `WithListLimit`, `WithListMarker`, `WithListMaxItems`, `WithListPaginated(false)`는 `limit`, `marker`, `max_items`, `paginated`와 같습니다. limit을 주면 다음 링크가 없어도 Python처럼 빈 페이지가 나올 때까지 마지막 ID를 marker로 삼아 다음 페이지를 요청합니다. max_items만 주면 그 값을 limit으로 보내고, 로컬 필터보다 먼저 행 수를 셉니다.

## 찾기

`Find`는 `Resource.find`처럼 먼저 `GET /v2.0/sfc/port_chains/{name_or_id}`를 보냅니다. 응답이 깨끗한 400, 403, 404이면 `name=<name_or_id>` query로 목록 전체를 읽고, ID나 이름이 정확히 같은 항목을 고릅니다. 없으면 기본으로 `nil, nil`이고, `WithFindIgnoreMissing(false)`이면 `resource.NotFoundError`입니다. 같은 항목이 둘 이상이면 `resource.AmbiguousError`이며, 500 같은 다른 실패에서는 목록으로 넘어가지 않습니다.

Python find의 `**query`는 `WithFindQuery(key, value)`로 넘깁니다. Python `Resource.fetch`처럼 모든 이름이 ID 조회 GET의 query로 나가고, 목록 단계에서는 목록과 같은 규칙으로 server query와 로컬 필터로 나뉩니다. query에 `name`이 있으면 Python과 같이 `name_or_id` 대신 그 값을 목록 query로 씁니다. `all_projects`는 Python에서도 어느 요청에도 실리지 않으므로 무시하고, `id`와 `headers` 같은 Resource 제어 이름은 HTTP 전에 거부합니다.

## Python과 다른 점

Python은 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. Go는 ID를 URL segment 하나로 escape하고 빈 ID, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. Python은 ID를 escape하지 않고 경로에 그대로 붙입니다. 로컬 필터는 JSON 값을 비교하므로 Python과 달리 bool과 숫자를 구분합니다. Python의 networking-sfc 리소스에는 microversion이 없어 버전 헤더를 보내지 않지만, Go는 선택한 client에 microversion이 설정되어 있으면 Gophercloud가 `OpenStack-API-Version: network <version>`을 붙입니다. 생성과 수정 본문의 키 순서는 Python의 인자 순서가 아니라 정렬된 순서입니다. 속성 이름의 대소문자를 무시하는 Python 매칭과 find의 deprecated 경고, Resource의 `location` 속성은 재현하지 않습니다. 헤더 옵션은 Go 확장이며 인증과 content 헤더는 SDK가 소유하므로 거부합니다.
