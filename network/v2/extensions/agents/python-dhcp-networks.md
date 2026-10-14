# DHCP agent가 호스팅하는 network 목록

`API.ListDHCPNetworkRecords`는 openstacksdk의 [`dhcp_agent_hosting_networks(agent, **query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)를 SDK가 직접 구현한 호출입니다. 생성된 `API.ListDHCPNetworks`는 query를 받지 않고 응답 하나만 읽기 때문에 Python이 보내는 Network query와 `networks_links` 이어 읽기를 재현할 수 없었습니다. 이 호출은 [`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)가 [`DHCPAgentHostingNetwork`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py)에 대해 하는 일을 그대로 따릅니다.

## 요청

```go
for row, err := range api.ListDHCPNetworkRecords(ctx, agentID,
	agents.WithDHCPNetworkRecordsQuery("is_shared", true),
	agents.WithDHCPNetworkRecordsQuery("fields", []string{"id", "name"}),
	agents.WithDHCPNetworkRecordsQuery("mtu", 1450),
) {
	...
}
```

- `GET agents/{agent_id}/dhcp-networks`를 보냅니다. iterator는 실행할 때 요청하며 다시 실행할 수 있습니다.
- `WithDHCPNetworkRecordsQuery(key, value)`는 Python keyword 인자 하나를 JSON 값으로 받습니다. Network `_query_mapping`에 있는 키는 Python 이름(`is_shared`)이나 wire 이름(`shared`)으로 줄 수 있고, 둘 다 주면 Python 이름이 이깁니다. 값은 requests처럼 목록이면 키를 반복하고 `True`처럼 Python 표기로 보내며 null이면 보내지 않습니다.
- `mtu`, `subnet_ids`, `dns_domain` 같이 query mapping에 없는 Network Body 속성은 서버로 보내지 않고 응답 행을 걸러 냅니다. Python과 같이 dict 값은 행의 mapping에 그 키들이 재귀적으로 들어 있는지 보고, 그 밖의 값은 Python `==`로 비교하며 없는 속성은 None입니다.
- 어느 쪽에도 속하지 않는 키는 Python처럼 버립니다. `paginated`, `max_items`, `microversion`, `headers` 같은 호출 제어 이름을 query로 주면 HTTP 전에 거부합니다.
- `WithDHCPNetworkRecordsMaxItems(n)`은 Python `max_items`입니다. limit이 없으면 `limit=n`을 보내고, 걸러 낸 행까지 모두 세어 n개를 읽으면 멈춥니다. `WithDHCPNetworkRecordsPaginated(false)`는 첫 page만 읽습니다. `WithDHCPNetworkRecordsHeader`로 요청 헤더를 더할 수 있습니다.

## 이어 읽기와 응답

- `networks_links`의 next href, 최상위 `next`, Link 헤더를 따라갑니다. `/v2.0`으로 시작하는 상대 href도 같은 collection으로 받습니다.
- next가 없어도 limit이 있으면 Python처럼 `marker=<마지막 id>`로 다음 page를 요청하고 빈 page에서 멈춥니다.
- 400 미만의 모든 상태를 받되 본문은 JSON이어야 합니다. 400 이상은 Gophercloud 상태 오류를 `resource.OperationError`(Operation `ListDHCPNetworkRecords`, Resource `agents`)로 감쌉니다.
- 각 `DHCPNetworkRecord`는 Gophercloud `networks.Network`로 디코드한 `Network`, Python이 버리는 `self`를 뺀 원본 행 `Fields`, Python이 행마다 넣는 `AgentID`, 응답 헤더와 상태 코드를 담습니다.
- 빈 agent ID, nil option, 음수 max_items, JSON으로 만들 수 없는 값, nil API는 HTTP 전에 `resource.ErrInvalidOption`으로 실패합니다.

## Python과 다른 점

- Python은 agent를 ID나 Agent Resource로 받고 Go는 ID 문자열을 받습니다. `base_path`와 `jmespath_filters`는 지원하지 않습니다.
- Python은 typed Resource 속성(예: `type=int`인 `mtu`)으로 변환한 값과 비교하고, Go는 응답 JSON 값을 그대로 Python 규칙으로 비교합니다.
- Python은 DHCPAgentHostingNetwork Resource를 내주고 Go는 `DHCPNetworkRecord`를 내줍니다. Go는 query 키를 이름 순서로 보냅니다.
