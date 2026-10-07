# 부모가 필요한 리소스

부모와 자식 ID를 매번 조합하는 대신 부모를 한 번 지정한 객체를 사용합니다. 이름을 지정하면 정확한 이름 검색과 중복 검사를 수행합니다. ID를 지정하면 부모 조회 요청 없이 그 ID를 보관합니다. 부모가 실제로 없는 경우 후속 리소스 요청의 404를 그대로 처리합니다.

Heat stack은 API가 name+ID를 모두 요구하므로 예외입니다. `Stacks.InStack(ctx, resource.ID(id))`도 Heat identity GET으로 이름을 확인합니다. 두 값을 이미 알고 있으면 `Stacks.ForStack(stacks.StackIdentity{Name: name, ID: id})`로 조회 없이 고정합니다. Nova·Cinder·Neutron·Octavia project quota는 singleton이며, Connection의 서비스별 quota helper가 별도 Keystone client로 프로젝트 이름을 해석합니다. Neutron·Octavia의 실제 quota 목록은 singleton과 별개인 API iterator를 사용합니다.

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

이 코드는 context와 연결 객체 `conn`, `github.com/JSYoo5B/gophercloudsdk/resource`와 `github.com/JSYoo5B/gophercloudsdk/dns/v2/recordsets`를 사용하는 함수 안에서 작성합니다. 범위 객체의 Create/Update는 기존 API의 concrete options와 `With...` 함수를 그대로 사용하며 요청 builder 구현은 필요하지 않습니다.

openstacksdk의 `conn.dns.get_recordset(record_id, zone=zone)`, `conn.load_balancer.members(pool=pool)`처럼 부모를 전달하는 동작에 대응합니다. Go에서는 아래처럼 부모 범위를 객체에 고정합니다.

| 리소스 | Go 접근 | 실제 자식 식별자 |
|---|---|---|
| DNS record set | `dns.RecordSets.InZone(ctx, zone)` | record set ID |
| Nova interface | `compute.AttachInterfaces.InServer(ctx, server)` | port ID |
| Nova volume attachment | `compute.VolumeAttachments.InServer(ctx, server)` | volume ID |
| Nova instance action | `compute.InstanceActions.InServer(ctx, server)` | request ID |
| Nova server tags | `compute.Tags.InServer(ctx, server)` | 태그 문자열 집합 |
| Nova project quota | `conn.ProjectQuotas(ctx, project)` / `CurrentProjectQuotas(ctx)` | 고정 project ID; Get·Defaults·Detail·Update·Reset |
| Nova user quota | `projectScope.InUser(ctx, user)` | 고정 project+user; Get·Detail·Update·Reset, project-wide Defaults/Reset 미노출 |
| Cinder project quota | `conn.BlockStorageProjectQuotas(ctx, project)` / `CurrentBlockStorageProjectQuotas(ctx)` | 고정 project ID; Get·Defaults·Usage·Update·Reset |
| Neutron project quota | `conn.NetworkProjectQuotas(ctx, project)` / `CurrentNetworkProjectQuotas(ctx)` | 고정 project ID; Get·Defaults·Detail·Update·Delete, API의 별도 override 목록 |
| Octavia project quota | `conn.LoadBalancerProjectQuotas(ctx, project)` / `CurrentLoadBalancerProjectQuotas(ctx)` | 고정 project ID; Get·Update·Reset, Defaults는 전역 별도 타입, API의 별도 quota 목록 |
| Manila project quota | `conn.SharedFileSystemProjectQuotas(ctx, project)` / `CurrentSharedFileSystemProjectQuotas(ctx)` | 고정 project ID; Get·Defaults·Detail·Update·Reset, SDK 직접 구현 |
| Manila user/share type quota | `projectScope.InUser(ctx, user)` / `InShareType(ctx, shareType)` | 고정 project+selector; Get·Detail·Update·Reset, share type은 2.39 이상 |
| Manila quota class | `shared.QuotaClassSets.InClass(ctx, "default")` | 고정 class 이름; Get·Update, 프로젝트 조회 없음 |
| Designate project quota | `conn.DNSProjectQuotas(ctx, project)` / `CurrentDNSProjectQuotas(ctx)` | 고정 project ID와 sudo-project header; Get·PATCH Update·DELETE Reset |
| Magnum project/resource quota | `conn.ContainerInfraProjectQuotas(ctx, project)` → `ForResource(quotas.Cluster)` | 고정 project+resource; Create·Get·PATCH Update·Delete, explicit hard limit |
| Nova project limits | `conn.ProjectLimits(ctx, project)` / `CurrentProjectLimits(ctx)` | 고정 tenant_id query; Get, 일반 current 조회는 Limits.Fetch |
| Cinder project limits | `conn.BlockStorageProjectLimits(ctx, project)` / `CurrentBlockStorageProjectLimits(ctx)` | 고정 project_id query; Get, 선택한 숫자 버전3.39 이상·admin context 필요 |
| Heat stack | `orchestration.Stacks.InStack(ctx, ref)` / `ForStack(identity)` | stack name + UUID |
| Heat stack resource | `orchestration.StackResources.InStack(ctx, ref)` / `ForStack(identity)` | 실제 소속 stack pair + resource_name |
| Heat stack event | `orchestration.StackEvents.InStack(ctx, ref)` → `ForResource(name)` | stack pair + resource_name + event ID |
| Manila share access rule | `shared.ShareAccessRules.InShare(ctx, share)` | access ID; 응답 share ID 검증 |
| Trove database | `database.Databases.InInstance(ctx, instance)` | 데이터베이스 이름 |
| Trove user | `database.Users.InInstance(ctx, instance)` | 사용자·host 식별자 (서비스 설명 참조) |
| Magnum node group | `magnum.NodeGroups.InCluster(ctx, cluster)` | UUID |
| Barbican secret consumer | `keymanager.SecretConsumers.InSecret(ctx, secret)` | service + resource_type + resource_id association; 별도 consumer ID 없음 |
| Zaqar subscription | `messaging.Subscriptions.InQueue(ctx, queueName)` | 구독 ID; caller가 지정한 queue 이름을 추가 조회 없이 고정 |
| Masakari host | `ha.Hosts.InSegment(ctx, segment)` | Host UUID; Segment UUID는 한 번 고정 |
| Masakari VMove | `ha.VMoves.InNotification(ctx, notification)` | VMove UUID; Notification UUID와 numeric1.3 이상 필요 |
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
| Swift object | `swift.Objects.InContainer(ctx, container)` | object 이름 |

Compute/Network/Image의 전체 API 객체는 `conn.ComputeV2(ctx)`, `conn.NetworkV2(ctx)`, `conn.ImageV2(ctx)`로 가져오거나 기존 서비스의 `API` 필드를 사용합니다. 표의 부모는 `resource.ID(...)` 또는 `resource.Name(...)`입니다. floating IP처럼 이름 필드가 없는 부모는 ID로 지정합니다.

Barbican의 [SecretConsumer](../keymanager/v1/secretconsumers/README.md)는 secret ID로 범위를 고정합니다. 명시적 Name은 기존 Secrets 목록으로 한 번 해석하는 Go 편의 기능이며 Python consumer 문자열의 ID 해석과 구분합니다. Create·Delete는 실제 HTTP200의 secret 응답을 반환하고 List/All은 advertised offset next로 순회합니다. 별도 consumer Get·ID·Find·Wait는 제공하지 않습니다.

Zaqar의 [Subscription](../messaging/v2/subscriptions/README.md) 부모는 `resource.Ref` 대신 queue 이름 문자열입니다. `conn.MessagingV2(ctx)`의 `Subscriptions.InQueue(ctx, "jobs")`는 queue를 조회하지 않고 Create·Get·List/All·Delete의 경로를 고정합니다. 연결의 안정적인 Client-ID를 공유하며 호출별 typed project header는 공유 client를 수정하지 않습니다. 목록은 Python MessageResource의 limit·marker 동작을 사용하고 TTL 생략은 서버 기본값을 유지합니다.

Masakari의 `ha`는 `conn.InstanceHA(ctx)`로 얻습니다. [Host scope](../instanceha/v1/hosts/README.md)는 Segment 이름이나 UUID를 한 번 해석하며 [VMove scope](../instanceha/v1/vmoves/README.md)는 이름이 없는 Notification의 UUID를 사용합니다. 데이터베이스 `ID`와 URI UUID를 구별하고, scope의 부모 ID를 응답 body나 확장 query로 바꾸지 않습니다.

Collection 기반 범위 객체는 Get/Find/List/All/Delete/Wait/WaitDeleted/ResolveID를 공유합니다. 이름·삭제·상태가 없는 모델은 해당 기능에 `ErrUnsupported`를 반환합니다. Delete는 기본적으로 404를 무시하며 `WithMissingError()`로 엄격한 동작을 선택합니다. Create/Update는 원래 API에 해당 연산이 있는 범위에 제공됩니다.

서버 tag 범위는 Collection 대신 Add/Check/List/Replace/Remove/RemoveAll을 제공하는 문자열 집합입니다. microversion 2.26 이상을 요구하며 nil·빈 교체 목록은 집합을 비웁니다. [tag의 선택적 404 정책](../compute/v2/tags/README.md)을 참고합니다. Nova action은 requestID로만 조회하고 이름·삭제·상태 대기는 지원하지 않습니다. SDK의 [ActionResource](../compute/v2/instanceactions/README.md)는 목록·상세와 event의 추가 필드·원본 JSON·헤더를 보존합니다.

Heat [자식 리소스](../orchestration/v1/stackresources/README.md)는 logical/physical ID 대신 `resource_name`으로 조회하고 nested 항목의 실제 소속을 검증합니다. Metadata·health 변경·상태 대기를 제공하며 없는 생성·삭제를 만들지 않습니다. [이벤트](../orchestration/v1/stackevents/README.md)는 stack 전체 또는 resource별 목록과 resource-scoped 단건 GET을 제공하는 읽기 전용 범위입니다. 이벤트 이름 Find·변경·대기는 없습니다. 두 범위 모두 이미 아는 canonical stack pair를 `ForStack`으로 고정할 수 있습니다.

Trove의 [database](../db/v1/databases/README.md)와 [user](../db/v1/users/README.md)는 pinned native/Python SDK가 fetch를 노출하지 않아 Get을 같은 instance의 목록 검색으로 제공합니다. 단일 Create와 CreateBatch 모두 배열 본문으로 요청하고 error를 반환합니다. 자식 상태 대기는 제공하지 않습니다. 응답이 없는 비동기 생성의 완료나 batch 원자성을 추정하지 않습니다.

Swift의 container 이름과 object 키는 해당 서비스의 식별자입니다. object 키의 `/`, 공백과 query 문자도 SDK가 URL에 인코딩하며, 애플리케이션은 원래 문자열을 `resource.ID(...)`에 전달합니다. [Swift 사용법](../objectstorage/v1/objects/README.md)에서 metadata 조회와 업로드·다운로드를 확인합니다.

범위는 호출 사이에도 고정됩니다. 부모 이름을 매번 다시 찾거나 다른 부모의 동일한 자식 이름으로 대체하지 않습니다. 일반 Collection처럼 context 취소와 원래 HTTP 오류를 보존합니다. 기존 15개 범위의 실제 URL·기본 TTL·JSON Patch·Swift 키는 [scope](../api/scoped_contracts_test.go), [QoS](../api/qos_contracts_test.go), [Swift](../api/swift_resources_contracts_test.go) 테스트에서 확인합니다. 추가 범위는 [tags](../api/server_tags_contracts_test.go), [actions](../api/instance_actions_scope_test.go), [databases](../api/trove_databases_contracts_test.go), [users](../api/trove_users_contracts_test.go) HTTP 계약에서 검증합니다.

Glance [metadata Property](../image/v2/metadefproperties/README.md)의 `InNamespace`는 HTTP 없이 literal parent와 서비스 target을 고정합니다. 목록의 `Key`는 dictionary의 출처이며 optional `Name`과 별도로 유지합니다. 이후 호출은 애플리케이션이 선택한 literal 이름을 받습니다.

Glance [metadata Tag](../image/v2/metadeftags/README.md)는 `InNamespace`로 parent와 service target을 고정합니다. 양수 limit의 다음 marker는 yield 전에 복사한 마지막 원본 이름이며 response의 next·Link를 따라가지 않습니다.

Glance [Resource type association](../image/v2/metadefresourcetypes/README.md)은 HTTP 없는 `InNamespace`로 parent를 고정합니다. 전역 catalog와 연결 목록은 별도 모델로 반환하고, 연결 해제는 전역 resource type을 지우지 않습니다.
