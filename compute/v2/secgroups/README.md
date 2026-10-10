# Nova 보안 그룹 proxy native 호출

`secgroups.New(client)`(또는 `service.SecurityGroups`)의 generated 메서드는 Gophercloud `v2.15.0`의 [secgroups 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/secgroups/requests.go)을 바꾸지 않고 호출합니다. Nova의 `os-security-groups` API는 Neutron 보안 그룹을 대신 호출하던 proxy이며 Compute microversion 2.36부터 제거됐습니다. native 호출은 microversion을 고르지 않으므로 client의 `Microversion`을 2.35 이하로 두거나 [Neutron 보안 그룹](../../../network/v2/native-crud.md#security-grouprulefloating-ip)을 직접 사용합니다. SDK는 오류에 `resource.OperationError{Resource: "secgroups"}` 문맥만 더합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST os-security-groups`, `{"security_group": {...}}` | 200 |
| `Get(ctx, id)` | `GET os-security-groups/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT os-security-groups/{id}` | 200 |
| `Delete(ctx, id)` | `DELETE os-security-groups/{id}` | 202, 204 |
| `List(ctx)` | `GET os-security-groups` | 한 페이지 |
| `ListByServer(ctx, serverID)` | `GET servers/{server}/os-security-groups` | 한 페이지 |
| `CreateRule(ctx, opts, options...)` | `POST os-security-group-rules` | 200 |
| `DeleteRule(ctx, id)` | `DELETE os-security-group-rules/{id}` | 202, 204 |
| `AddServer(ctx, serverID, name)`·`RemoveServer(...)` | `POST servers/{server}/action`, `{"addSecurityGroup"\|"removeSecurityGroup": {"name": ...}}` | 201, 202 |

`CreateOpts.Name`은 필수입니다. `UpdateOpts.Description`은 pointer라 빈 설명을 보낼 수 있습니다. `CreateRuleOpts`는 `ParentGroupID`와 `IPProtocol`이 필수이고 `CIDR`과 `FromGroupID`(`group_id`) 중 하나 이상이 있어야 합니다. `FromPort`·`ToPort`는 omitempty가 없어 0도 보냅니다. 서버 action은 그룹을 ID가 아닌 이름으로 지정합니다.

응답의 그룹·rule ID와 `parent_group_id`는 nova-network의 숫자와 Neutron의 UUID 문자열을 모두 받아 문자열로 바꿉니다. 두 목록은 한 페이지만 읽습니다.
