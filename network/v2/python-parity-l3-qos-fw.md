# Python openstacksdk Neutron L3 agent, QoS rule, FWaaS 대응

이 문서는 openstacksdk [`network/v2/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)의 Proxy 메서드 가운데 L3 agent 스케줄링, QoS rule 생성·수정·삭제, FWaaS v2, conntrack helper, NDP proxy, 기본 보안 그룹 규칙, 보안 그룹 기본 statefulness 62개를 이 SDK의 Go 호출과 맞춰 봅니다. Go 호출은 모두 `conn.NetworkV2(ctx)`에서 얻는 서비스의 필드(`Agents`, `Routers`, `QoSRules`, `FirewallGroups`, `FirewallPolicies`, `FirewallRules`)를 기준으로 적었습니다.

요청 경로, query, 본문과 결과 필드는 아래 테스트가 고정합니다.

- [`extensions/fwaas_v2/groups/python_parity_test.go`](extensions/fwaas_v2/groups/python_parity_test.go)
- [`extensions/fwaas_v2/policies/python_parity_test.go`](extensions/fwaas_v2/policies/python_parity_test.go)
- [`extensions/fwaas_v2/rules/python_parity_test.go`](extensions/fwaas_v2/rules/python_parity_test.go)
- [`extensions/qos/rules/python_parity_test.go`](extensions/qos/rules/python_parity_test.go)
- [`extensions/agents/python_parity_test.go`](extensions/agents/python_parity_test.go)
- [`extensions/layer3/routers/python_parity_test.go`](extensions/layer3/routers/python_parity_test.go)

판정은 두 가지입니다. `대응`은 Python 인자의 효과를 공개 Go API로 재현할 수 있다는 뜻이며, 아래 절에 적은 차이는 그대로 남습니다. `미해결`은 Go에 해당 요청이 없거나 Python 기본 호출과 같은 요청을 보낼 수 없다는 뜻입니다.

## 대응표

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `routers_hosting_l3_agents(router, **query)` | `Routers.ListL3Agents(ctx, routerID)` | 대응 |
| `agent_hosted_routers(agent, **query)` | `Agents.ListL3Routers(ctx, agentID)` | 대응 |
| `add_router_to_agent(agent, router, *, ha_chassis_priority=None)` | `Agents.ScheduleL3Router(ctx, agentID, ScheduleL3RouterOpts{RouterID}, WithScheduleL3RouterField("ha_chassis_priority", n))` | 대응 |
| `update_router_in_agent(agent, router, *, ha_chassis_priority=None)` | 없음 | 미해결 |
| `remove_router_from_agent(agent, router)` | `Agents.RemoveL3Router(ctx, agentID, routerID)` | 대응 |
| `create_qos_bandwidth_limit_rule(qos_policy, **attrs)` | `QoSRules.CreateBandwidthLimitRule(ctx, policyID, CreateBandwidthLimitRuleOpts{...})` | 대응 |
| `update_qos_bandwidth_limit_rule(qos_rule, qos_policy, **attrs)` | `QoSRules.UpdateBandwidthLimitRule(ctx, policyID, ruleID, UpdateBandwidthLimitRuleOpts{...})` | 대응 |
| `delete_qos_bandwidth_limit_rule(qos_rule, qos_policy, ignore_missing=True)` | `resource.IgnoreMissing(QoSRules.DeleteBandwidthLimitRule(ctx, policyID, ruleID))` | 대응 |
| `create_qos_dscp_marking_rule(qos_policy, **attrs)` | `QoSRules.CreateDSCPMarkingRule(ctx, policyID, CreateDSCPMarkingRuleOpts{...})` | 대응 |
| `update_qos_dscp_marking_rule(qos_rule, qos_policy, **attrs)` | `QoSRules.UpdateDSCPMarkingRule(ctx, policyID, ruleID, UpdateDSCPMarkingRuleOpts{...})` | 대응 |
| `delete_qos_dscp_marking_rule(qos_rule, qos_policy, ignore_missing=True)` | `resource.IgnoreMissing(QoSRules.DeleteDSCPMarkingRule(ctx, policyID, ruleID))` | 대응 |
| `create_qos_minimum_bandwidth_rule(qos_policy, **attrs)` | `QoSRules.CreateMinimumBandwidthRule(ctx, policyID, CreateMinimumBandwidthRuleOpts{...})` | 대응 |
| `update_qos_minimum_bandwidth_rule(qos_rule, qos_policy, **attrs)` | `QoSRules.UpdateMinimumBandwidthRule(ctx, policyID, ruleID, UpdateMinimumBandwidthRuleOpts{...})` | 대응 |
| `delete_qos_minimum_bandwidth_rule(qos_rule, qos_policy, ignore_missing=True)` | `resource.IgnoreMissing(QoSRules.DeleteMinimumBandwidthRule(ctx, policyID, ruleID))` | 대응 |
| `create_qos_minimum_packet_rate_rule(qos_policy, **attrs)` | 없음 | 미해결 |
| `update_qos_minimum_packet_rate_rule(qos_rule, qos_policy, **attrs)` | 없음 | 미해결 |
| `delete_qos_minimum_packet_rate_rule(qos_rule, qos_policy, ignore_missing=True)` | 없음 | 미해결 |
| `create_qos_packet_rate_limit_rule(qos_policy, **attrs)` | 없음 | 미해결 |
| `update_qos_packet_rate_limit_rule(qos_rule, qos_policy, **attrs)` | 없음 | 미해결 |
| `delete_qos_packet_rate_limit_rule(qos_rule, qos_policy, ignore_missing=True)` | 없음 | 미해결 |
| `create_firewall_group(**attrs)` | `FirewallGroups.Create(ctx, groups.CreateOpts{...}, WithCreateField(...))` | 대응 |
| `update_firewall_group(firewall_group, **attrs)` | `FirewallGroups.Update(ctx, id, groups.UpdateOpts{...})`, `RemoveIngressPolicy`, `RemoveEgressPolicy` | 대응 |
| `delete_firewall_group(firewall_group, ignore_missing=True)` | `FirewallGroups.Remove(ctx, resource.ID(id))` | 대응 |
| `get_firewall_group(firewall_group, *, fields=None)` | `FirewallGroups.GetWithFields(ctx, id, fields...)` | 대응 |
| `find_firewall_group(name_or_id, ignore_missing=True, **query)` | `FirewallGroups.Find(ctx, resource.ID(v), resource.WithIgnoreMissing())` 다음 `Find(ctx, resource.Name(v), ...)` | 대응 |
| `firewall_groups(**query)` | `FirewallGroups.List(ctx, WithListOptions(groups.ListOpts{...}), WithListQuery(...))` | 대응 |
| `create_firewall_policy(**attrs)` | `FirewallPolicies.Create(ctx, policies.CreateOpts{...})` | 대응 |
| `update_firewall_policy(firewall_policy, **attrs)` | `FirewallPolicies.Update(ctx, id, policies.UpdateOpts{...})` | 대응 |
| `delete_firewall_policy(firewall_policy, ignore_missing=True)` | `FirewallPolicies.Remove(ctx, resource.ID(id))` | 대응 |
| `get_firewall_policy(firewall_policy, *, fields=None)` | `FirewallPolicies.GetWithFields(ctx, id, fields...)` | 대응 |
| `find_firewall_policy(name_or_id, ignore_missing=True, **query)` | `FirewallPolicies.Find` ID 다음 Name | 대응 |
| `firewall_policies(**query)` | `FirewallPolicies.List(ctx, WithListOptions(policies.ListOpts{...}), WithListQuery(...))` | 대응 |
| `insert_rule_into_policy(firewall_policy, firewall_rule, insert_after=None, insert_before=None)` | `FirewallPolicies.InsertRule(ctx, id, InsertRuleOpts{...})` (위치 필수) | 미해결 |
| `remove_rule_from_policy(firewall_policy, firewall_rule)` | `FirewallPolicies.RemoveRule(ctx, id, ruleID)` | 대응 |
| `create_firewall_rule(**attrs)` | `FirewallRules.Create(ctx, rules.CreateOpts{Protocol, Action, ...})` | 대응 |
| `update_firewall_rule(firewall_rule, **attrs)` | `FirewallRules.Update(ctx, id, rules.UpdateOpts{...})` | 대응 |
| `delete_firewall_rule(firewall_rule, ignore_missing=True)` | `FirewallRules.Remove(ctx, resource.ID(id))` | 대응 |
| `get_firewall_rule(firewall_rule, *, fields=None)` | `FirewallRules.GetWithFields(ctx, id, fields...)` | 대응 |
| `find_firewall_rule(name_or_id, ignore_missing=True, **query)` | `FirewallRules.Find` ID 다음 Name | 대응 |
| `firewall_rules(**query)` | `FirewallRules.List(ctx, WithListOptions(rules.ListOpts{...}), WithListQuery(...))` | 대응 |
| `create_conntrack_helper(router, **attrs)` | 없음 | 미해결 |
| `conntrack_helpers(router, **query)` | 없음 | 미해결 |
| `get_conntrack_helper(conntrack_helper, router)` | 없음 | 미해결 |
| `update_conntrack_helper(conntrack_helper, router, **attrs)` | 없음 | 미해결 |
| `delete_conntrack_helper(conntrack_helper, router, ignore_missing=True)` | 없음 | 미해결 |
| `create_ndp_proxy(**attrs)` | 없음 | 미해결 |
| `get_ndp_proxy(ndp_proxy, fields=None)` | 없음 | 미해결 |
| `find_ndp_proxy(ndp_proxy_id, ignore_missing=True, **query)` | 없음 | 미해결 |
| `delete_ndp_proxy(ndp_proxy, ignore_missing=True)` | 없음 | 미해결 |
| `ndp_proxies(**query)` | 없음 | 미해결 |
| `update_ndp_proxy(ndp_proxy, **attrs)` | 없음 | 미해결 |
| `create_default_security_group_rule(**attrs)` | 없음 | 미해결 |
| `delete_default_security_group_rule(default_security_group_rule, ignore_missing=True)` | 없음 | 미해결 |
| `find_default_security_group_rule(name_or_id, ignore_missing=True, **query)` | 없음 | 미해결 |
| `get_default_security_group_rule(default_security_group_rule, fields=None)` | 없음 | 미해결 |
| `default_security_group_rules(**query)` | 없음 | 미해결 |
| `create_security_groups_default_statefulness(**attrs)` | 없음 | 미해결 |
| `delete_security_groups_default_statefulness(..., ignore_missing=True)` | 없음 | 미해결 |
| `find_security_groups_default_statefulness(name_or_id, ignore_missing=True, **query)` | 없음 | 미해결 |
| `get_security_groups_default_statefulness(..., fields=None)` | 없음 | 미해결 |
| `security_groups_default_statefulness(**query)` | 없음 | 미해결 |
| `update_security_groups_default_statefulness(..., **attrs)` | 없음 | 미해결 |

대응 32개, 미해결 30개입니다.

## 공통 차이

Python 메서드는 ID 문자열과 Resource 객체를 모두 받고 변경 가능한 Resource를 돌려줍니다. Go는 문자열 ID를 받고 Gophercloud v2.15.0의 타입 모델 포인터를 돌려주므로 모델에 없는 응답 필드는 버려집니다. 이 범위의 Resource는 모두 `_max_microversion`이 없어서 microversion 협상 차이는 생기지 않습니다.

Python Proxy는 `raise_exc=False`로 요청하고 Resource 메서드가 직접 오류를 판정합니다. 그래서 `Resource.create`, `commit`, `fetch`, `delete`, `list`는 400 이상에서 예외를 던지지만 [`agent.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/agent.py)의 L3 router 메서드는 상태를 보지 않습니다. Go는 모든 호출에서 Gophercloud가 정한 성공 코드 밖의 응답을 오류로 돌려줍니다.

## L3 agent 스케줄링

`add_router_to_agent`는 `POST /agents/{agent_id}/l3-routers`에 `{"router_id": ...}`를 보내고, `ha_chassis_priority`가 None이 아닐 때만 그 키를 더합니다. Go에서는 `ScheduleL3Router`에 `WithScheduleL3RouterField("ha_chassis_priority", n)`을 붙이면 같은 본문이 나갑니다. Python은 응답 JSON을 dict로 그대로 돌려주고 오류 상태에서도 예외를 던지지 않습니다. Go는 201만 성공으로 받고 응답 본문은 버린 채 오류만 돌려줍니다.

`remove_router_from_agent`는 `DELETE /agents/{agent_id}/l3-routers/{router_id}`를 보냅니다. Python은 이 DELETE에 `{"router_id": ...}` 본문을 함께 싣고 어떤 상태든 None을 돌려줍니다. Go `RemoveL3Router`는 본문 없이 보내고 404나 409를 오류로 돌려주므로, Python처럼 결과를 무시하려면 호출자가 오류를 버려야 합니다.

`agent_hosted_routers`는 `GET /agents/{agent_id}/l3-routers`, `routers_hosting_l3_agents`는 `GET /routers/{router_id}/l3-agents`입니다. 두 Go 호출은 query를 받지 않으므로 Python `**query` 필터는 돌아온 목록에서 호출자가 직접 걸러야 합니다. `ListL3Routers`는 한 번의 GET으로 슬라이스를 돌려주고 `routers_links`를 따라가지 않으며, `ListL3Agents`도 한 페이지만 읽습니다. 결과 모델은 각각 `routers.Router`와 `routers.L3Agent`라서 Python Router의 `ha`, `flavor_id`나 Agent의 `ha_chassis_priority` 같은 필드는 남지 않습니다.

## QoS rule 생성, 수정, 삭제

bandwidth limit, DSCP marking, minimum bandwidth rule은 `/qos/policies/{policy_id}/<rule 종류>_rules`에 Python과 같은 POST, PUT, DELETE를 보냅니다. policy ID는 URI에만 쓰이고 본문에는 들어가지 않습니다. 기존 조회 대응은 [qos-bandwidth-limit-records-python.md](qos-bandwidth-limit-records-python.md)와 같은 문서에 있습니다.

Go 생성 옵션은 `max_kbps`, `dscp_mark`, `min_kbps`를 항상 보냅니다. Python에서 이 값을 빼면 키도 빠지지만 Neutron은 어차피 이 값을 요구하므로 성공하는 요청에서는 차이가 없습니다. `direction`은 빈 문자열이면 빠지고, 이미 선언된 필드라서 `WithCreate...Field`로 null을 덮어쓸 수도 없습니다.

수정에서 Python은 바뀐 속성이 없으면 HTTP 요청을 보내지 않습니다. Go `Update...Rule`은 빈 옵션이어도 `{"bandwidth_limit_rule": {}}` 같은 PUT을 보냅니다.

삭제는 Python 기본값 `ignore_missing=True`를 맞추기 위해 `resource.IgnoreMissing`으로 감쌉니다. 이 함수는 순수한 404만 nil로 바꾸고 409 같은 오류는 그대로 둡니다. `ignore_missing=False`가 필요하면 감싸지 않은 호출이 404 오류를 그대로 돌려줍니다.

```go
net, err := conn.NetworkV2(ctx)
if err != nil {
	return err
}
err = resource.IgnoreMissing(net.QoSRules.DeleteBandwidthLimitRule(ctx, policyID, ruleID))
```

## FWaaS v2 group, policy, rule

세 Resource는 모두 `/fwaas/firewall_groups`, `/fwaas/firewall_policies`, `/fwaas/firewall_rules` 아래의 일반 Resource이고 `NetworkResource`를 상속하지 않습니다. 그래서 `if_revision` 같은 조건부 헤더는 없습니다.

생성과 수정은 `{"firewall_group": {...}}` 형태의 envelope 본문을 보냅니다. Python Resource는 `_allow_unknown_attrs_in_body=True`라서 선언되지 않은 키도 본문에 넣는데, Go에서는 `WithCreateField`, `WithUpdateField`로 같은 키를 더할 수 있습니다. 다만 Go 옵션 구조체에 이미 있는 키는 확장 필드로 다시 쓸 수 없습니다. 그래서 생성 시 빈 문자열 `description=""`처럼 `omitempty`로 빠지는 값은 Python과 달리 보내지 않습니다. 수정은 포인터 필드라서 빈 문자열과 빈 목록을 그대로 보냅니다. `update_firewall_group(fg, ingress_firewall_policy_id=None)`처럼 null을 보내는 수정은 `RemoveIngressPolicy`와 `RemoveEgressPolicy`가 따로 맡습니다. Python은 바뀐 속성이 없으면 요청을 생략하지만 Go `Update`는 빈 본문 PUT을 보냅니다.

`create_firewall_rule`에서 Python은 `protocol`과 `action`을 생략할 수 있고 Neutron이 각각 null(any)과 `deny`로 채웁니다. Go `rules.CreateOpts`는 두 값을 필수로 요구하므로 `Protocol: rules.ProtocolAny, Action: rules.ActionDeny`를 명시해야 같은 결과가 나옵니다. 이때 `ProtocolAny`는 본문에서 `"protocol": null`로 바뀝니다.

삭제는 `Remove(ctx, resource.ID(id))`가 Python 기본값처럼 404를 무시하고, `resource.WithMissingError()`를 주면 `resource.ErrNotFound`를 돌려줍니다. 409 같은 다른 오류는 숨기지 않습니다.

조회에서 Python `get_firewall_*`은 `fields`를 주면 `?fields=...`를 붙입니다. 세 FWaaS package에는 sdkgen이 만든 `GetWithFields(ctx, id, fields...)`는 Python `_get_with_fields`처럼 Resource 상속 체인의 Body 속성 이름을 wire 이름으로 바꾼 뒤 `fields` query를 같은 키로 반복해서 보냅니다. 선언되지 않은 이름, 중복, 순서는 그대로 두고, 필드를 주지 않으면 query 없이 `Get`과 같은 GET을 보냅니다. 성공 상태는 Python 읽기처럼 200부터 399까지 받고, 일부 필드만 담긴 응답은 Gophercloud 모델로 decode되어 빠진 필드가 zero 값이 됩니다. 실패는 `resource.OperationError`로 감싼 native 오류로 돌아오므로 `gophercloud.ResponseCodeIs(err, 404)`로 Python의 `NotFoundException`을 구분합니다.

목록은 `List`에 `WithListOptions`와 `WithListQuery`를 함께 씁니다. Python QueryParameters에 있는 키(`name`, `status`, `project_id`, `fields`, `firewall_rules`, `firewall_policy_id` 등)는 그대로 query로 나갑니다. Python은 query에 없는 Body 속성, 예를 들어 group의 `admin_state_up`을 받은 행에서 직접 걸러 내는데, Go `ListOpts.AdminStateUp`은 그 값을 서버 query로 보냅니다. Python bool은 `True`/`False`로 인코딩되므로 같은 문자열이 필요하면 `WithListQuery("shared", "False")`를 씁니다. `WithListQuery`는 키마다 값을 하나만 보내므로 `fields=["id","name"]`처럼 같은 키를 반복하는 query는 만들 수 없습니다. 반면 `ListOpts.Ports`는 값마다 `ports=`를 반복합니다. 다음 페이지는 `firewall_groups_links`를 따라갑니다.

찾기는 Python `Resource.find`처럼 ID GET을 먼저 하고, 실패하면 `name=<값>` 목록에서 찾습니다. Go는 두 번의 `Find` 호출로 같은 요청을 만듭니다.

```go
found, err := net.FirewallGroups.Find(ctx, resource.ID(v), resource.WithIgnoreMissing())
if err == nil && found == nil {
	found, err = net.FirewallGroups.Find(ctx, resource.Name(v), resource.WithIgnoreMissing())
}
```

Python은 ID GET이 400이나 403이어도 목록으로 넘어가지만 Go `Find(ID)`는 404만 nil로 바꾸므로, 400과 403에서는 호출자가 직접 이름 검색을 이어야 합니다. Python 목록 비교는 ID와 이름을 모두 보지만 Go 이름 검색은 이름만 비교하고, 중복 이름은 `resource.ErrAmbiguous`가 됩니다. Python `find_*`의 `**query`는 ID GET과 목록 양쪽에 붙지만 Go `Find`에는 추가 query를 넣을 수 없습니다.

`remove_rule_from_policy`는 `PUT /fwaas/firewall_policies/{id}/remove_rule`에 `{"firewall_rule_id": ...}`를 보내고 envelope 없는 policy 응답을 읽습니다. Go `RemoveRule`도 같은 요청과 응답 형식을 씁니다. Python은 2xx와 3xx를 성공으로 보지만 Go는 200만 받습니다.

`insert_rule_into_policy`는 미해결입니다. Python은 위치를 주지 않아도 `{"firewall_rule_id": ..., "insert_after": null, "insert_before": null}`을 보내는데, Gophercloud `InsertRuleOpts`는 `InsertBefore`와 `InsertAfter` 가운데 정확히 하나를 요구하고 HTTP 전에 거부합니다. 두 키가 선언된 필드라서 `WithInsertRuleField`로 null을 넣는 것도 막힙니다. 위치 하나를 주는 호출은 Python과 같은 요청을 보내지만 null 키는 빠집니다.

## 미해결 메서드와 부족한 부분

- `update_router_in_agent`: Gophercloud v2.15.0에는 `PUT /agents/{agent_id}/l3-routers/{router_id}` 바인딩이 없어서 SDK 소유 호출이 필요합니다.
- `create_/update_/delete_qos_minimum_packet_rate_rule`, `create_/update_/delete_qos_packet_rate_limit_rule`: Gophercloud v2.15.0에는 `minimum_packet_rate_rules`, `packet_rate_limit_rules`의 쓰기 바인딩이 없습니다. 이 SDK에는 조회용 Record 헬퍼만 있습니다.
- `insert_rule_into_policy`: 위치 없는 기본 호출과 null 위치 키를 보낼 수 없습니다.
- conntrack helper 5개(`/routers/{router_id}/conntrack_helpers`), NDP proxy 6개(`/ndp_proxies`), 기본 보안 그룹 규칙 5개(`/default-security-group-rules`), 보안 그룹 기본 statefulness 6개(`/security-groups-default-statefulness`): Gophercloud v2.15.0에 이 경로의 바인딩이 전혀 없어서 SDK 소유 호출이 필요합니다.

## Python 소스

- [`openstack/network/v2/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py) (L4165 이후 QoS rule, L5806 이후 L3 agent, NDP proxy, FWaaS, L6846 이후 기본 보안 그룹 규칙과 statefulness, L8861 이후 conntrack helper)
- [`firewall_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_group.py), [`firewall_policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_policy.py), [`firewall_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_rule.py)
- [`agent.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/agent.py), [`router.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py)
- [`qos_bandwidth_limit_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_bandwidth_limit_rule.py), [`qos_dscp_marking_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_dscp_marking_rule.py), [`qos_minimum_bandwidth_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_minimum_bandwidth_rule.py), [`qos_minimum_packet_rate_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_minimum_packet_rate_rule.py), [`qos_packet_rate_limit_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_packet_rate_limit_rule.py)
- [`l3_conntrack_helper.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/l3_conntrack_helper.py), [`ndp_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/ndp_proxy.py), [`default_security_group_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/default_security_group_rule.py), [`security_groups_default_statefulness.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_groups_default_statefulness.py)
- 공통 동작: [`proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py), [`resource.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
