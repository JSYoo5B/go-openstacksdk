# 부모가 필요한 리소스

부모와 자식 ID를 매번 조합하는 대신 부모를 한 번 지정한 객체를 사용합니다. 이름을 지정하면 정확한 이름 검색과 중복 검사를 수행합니다. ID를 지정하면 부모 조회 요청 없이 그 ID를 보관합니다. 부모가 실제로 없는 경우 후속 리소스 요청의 404를 그대로 처리합니다.

```go
dns, err := conn.DNS(ctx)
if err != nil { return err }
records, err := dns.RecordSets.InZone(ctx, resource.Name("example.org."))
if err != nil { return err }

record, err := records.Find(ctx, resource.Name("www.example.org."))
if err != nil { return err }
ttl := 300
record, err = records.Update(ctx, resource.ID(record.ID),
    recordsets.UpdateOpts{TTL: &ttl},
    recordsets.WithUpdateField("vendor:enabled", false),
)
if err != nil { return err }
```

이 코드는 context와 연결 객체 `conn`, `gophercloudsdk/resource`와 `gophercloudsdk/dns/v2/recordsets`를 사용하는 함수 안에서 작성합니다. 범위 객체의 Create/Update는 기존 API의 concrete options와 `With...` 함수를 그대로 사용하며 요청 builder 구현은 필요하지 않습니다.

openstacksdk의 `conn.dns.get_recordset(record_id, zone=zone)`, `conn.load_balancer.members(pool=pool)`처럼 부모를 전달하는 동작에 대응합니다. Go에서는 아래처럼 부모 범위를 객체에 고정합니다.

| 리소스 | Go 접근 | 실제 자식 식별자 |
|---|---|---|
| DNS record set | `dns.RecordSets.InZone(ctx, zone)` | record set ID |
| Nova interface | `compute.AttachInterfaces.InServer(ctx, server)` | port ID |
| Nova volume attachment | `compute.VolumeAttachments.InServer(ctx, server)` | volume ID |
| Magnum node group | `magnum.NodeGroups.InCluster(ctx, cluster)` | UUID |
| Keystone application credential | `identity.ApplicationCredentials.InUser(ctx, user)` | credential ID |
| Keystone access rule | `identity.ApplicationCredentials.AccessRules(ctx, user)` | access rule ID |
| Keystone EC2 credential | `identity.EC2Credentials.InUser(ctx, user)` | access ID |
| Glance image member | `image.Members.InImage(ctx, imageRef)` | member ID |
| Octavia pool member | `loadBalancer.Pools.Members(ctx, pool)` | member ID |
| Octavia L7 rule | `loadBalancer.L7Policies.Rules(ctx, policy)` | rule ID |
| Neutron port forwarding | `network.PortForwarding.InFloatingIP(ctx, floatingIP)` | port forwarding ID |
| Neutron bandwidth limit rule | `network.QoSRules.BandwidthLimitRules(ctx, policy)` | rule ID |
| Neutron DSCP marking rule | `network.QoSRules.DSCPMarkingRules(ctx, policy)` | rule ID |
| Neutron minimum bandwidth rule | `network.QoSRules.MinimumBandwidthRules(ctx, policy)` | rule ID |

Compute/Network/Image의 전체 API 객체는 `conn.ComputeV2(ctx)`, `conn.NetworkV2(ctx)`, `conn.ImageV2(ctx)`로 가져오거나 기존 서비스의 `API` 필드를 사용합니다. 표의 부모는 `resource.ID(...)` 또는 `resource.Name(...)`입니다. floating IP처럼 이름 필드가 없는 부모는 ID로 지정합니다.

모든 범위 객체는 Get/Find/List/All/Delete/Wait/WaitDeleted/ResolveID를 공유합니다. 이름이나 상태가 없는 모델은 해당 기능에 `ErrUnsupported`를 반환합니다. Delete는 기본적으로 404를 무시하며 `WithMissingError()`로 엄격한 동작을 선택합니다. Create/Update는 원래 API에 해당 연산이 있는 범위에 제공됩니다.

범위는 호출 사이에도 고정됩니다. 부모 이름을 매번 다시 찾거나 다른 부모의 동일한 자식 이름으로 대체하지 않습니다. 일반 Collection처럼 context 취소와 원래 HTTP 오류를 보존합니다. HTTP 테스트는 [scoped_contracts_test.go](../api/scoped_contracts_test.go)와 [qos_contracts_test.go](../api/qos_contracts_test.go)에서 14개 범위의 실제 URL, 이름 해석, 기본 TTL과 JSON Patch를 검증합니다.
