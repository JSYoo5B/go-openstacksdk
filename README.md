# gophercloudsdk

Gophercloud 위에 연결, 서비스, 리소스, 복합 작업의 일관된 사용 방식을 제공하는 Go SDK 프로젝트입니다. 애플리케이션이 기본값, 이름 조회, 페이지네이션, 상태 대기, 요청 builder를 반복해서 구현하지 않도록 하는 것이 목적입니다.

현재는 **개발 중**입니다. 고정한 Gophercloud의 공개 API 호출은 제공하며, openstacksdk 수준의 리소스·복합 작업 계층을 확장하고 있습니다. Go 1.25 이상과 Gophercloud v2.15.0을 사용합니다. 모듈 이름 `gophercloudsdk`는 로컬 개발용이며, 저장소 공개 시 실제 모듈 경로로 변경해야 합니다. 옆 디렉토리의 개발 브랜치에 의존하는 `replace`는 사용하지 않습니다.

## 디렉토리와 지원 범위

```text
gophercloudsdk/
├── connection*.go           # 인증, 설정, 24개 서비스 접근과 캐시
├── accelerator/             # SDK 소유 Cyborg v2 모델·transport
├── clustering/              # SDK 소유 Senlin v1 profile·policy·cluster·node·receiver·action
├── instanceha/              # SDK 소유 Masakari v1 failover 리소스
├── compute/                 # 서버, flavor, 서버 생성 흐름
├── network/                 # Neutron 네트워크
├── image/                   # Glance 이미지
├── blockstorage/            # Cinder v3 볼륨
├── identity/, dns/, ...     # 서비스별 버전 패키지와 전체 API
├── resource/                # 공통 조회, iterator, 대기, 옵션, 오류
├── request/                 # SDK 소유 옵션, 확장 필드·query·header
├── api/                     # API/리소스 지원 목록과 HTTP 계약 테스트
├── internal/cmd/sdkgen/     # API, 공통 정책, 참조 문서 생성
├── internal/testcloud/      # HTTP 테스트 fixture
├── examples/                # 서버 생성과 floating IP 연결 실행 예제
└── docs/                    # 설계와 테스트 설명
```

고정한 Gophercloud API는 **21개 서비스의 23개 API 버전**, **194개 리소스 패키지**, **1,126개 공개 연산**을 제공합니다. SDK가 직접 구현한 Cyborg v2, Senlin v1, Masakari v1을 포함해 연결 가능한 서비스는 24개이며 API 버전은 26개입니다. Senlin은 69개 직접 선언 연산, Masakari는 17개에 대응하는 API를 제공합니다. 상속한 Resource 동작의 전체 비교는 계속 진행합니다. Manila quota·quota class도 SDK가 직접 구현합니다. 공통 정책은 108개 native 일반 Collection과 SDK 소유 리소스의 실제 조회·삭제·대기 capability, 부모 범위, Heat 복합 식별자·자식·이벤트, project quota·limits·Magnum project+resource quota와 server tag 집합에 적용합니다. native 연산 수에는 인증 함수와 URL 도우미도 포함되며 HTTP endpoint 수를 뜻하지 않습니다. [API 설명](api/README.md), [공통 정책 지원 목록](api/resource_inventory.json), [openstacksdk 비교 기준](api/openstacksdk/README.md)에서 범위를 확인합니다.

아래 표는 추가 이름 해석과 서버 생성 흐름을 제공하는 기존 상위 서비스의 범위입니다. 모든 API와 공통 정책은 이어지는 버전별 서비스 패키지에 있습니다.

| 서비스 | 리소스 | Get / Find / List / All | Delete | Wait | Create |
|---|---|---|---|---|---|
| [Compute](compute/README.md) | 서버 | 지원 | 지원 | 지원 | 이미지·기존 볼륨·이미지에서 만든 새 볼륨 부팅, 이름 해석, 선택적 대기 |
| Compute | flavor | 지원 | 미지원 | 미지원 | 미지원 |
| [Network](network/README.md) | 네트워크 | 지원 | 지원 | 지원 | 미지원 |
| Network | 포트 | 지원 | 지원 | 지원 | 개별 API |
| Network | floating IP | ID 조회·Find·목록 | 지원 | 지원 | 네트워크·서버·포트 해석, IPv4 선택, 선택적 대기 |
| [Image](image/README.md) | 이미지 | 지원 | 지원 | 지원 | metadata 생성·직접 업로드·선택적 대기 |
| [Block Storage](blockstorage/README.md) | 볼륨 | 지원 | 지원 | 지원 | 미지원 |

Create/Update와 각 서비스의 API 호출은 버전별 패키지에서 concrete options로 사용합니다. [microversion 범위 협상](docs/microversions.md)과 [볼륨 부팅 옵션](compute/README.md)을 제공하며, [floating IP 생성·연결](network/README.md)과 [이미지 직접 업로드](image/README.md)를 제공합니다. Senlin Profile·Policy는 [변경 추적과 Commit](clustering/v1/tracking/README.md), Cluster·Node는 [비동기 변경 추적과 accepted Operation](clustering/v1/tracking/async/README.md)을 제공합니다. 다른 리소스의 변경 추적, floating IP 재사용·서버 생성과 자동 연결, 이미지 import 흐름과 안전한 바이너리 자동 재시도는 계속 구현할 대상입니다. [SDK 지원 판정대장](docs/sdk-support-ledger.md)은 확인한 차이와 전체 완료의 기준을 기록합니다. Senlin은 [전용 상태·삭제 대기](clustering/v1/waiting/README.md), [고정 cluster의 policy 조회](clustering/v1/clusterpolicies/README.md), [node attribute 수집](clustering/v1/clusterattributes/README.md), [cluster metadata 관리](clustering/v1/clusters/metadata/README.md)와 [목록 행 수·첫 페이지 제어](clustering/v1/listing/README.md)를 제공합니다. [서버 계약 비교](docs/senlin-server-contracts.md)는 API reference와 실제 release 코드의 성공 코드·metadata 경로 차이를 기록합니다.

## 모든 서비스의 사용 문서

목록의 로컬 조건도 SDK 옵션으로 선택합니다. QoS rules·Address Group addresses·Subnet Pool prefixes·Network subnets는
`resource.WithBodyFilter`/`WithBodyFilters`로 field 선택·snapshot·배열 비교를 처리합니다.
Subnet은 [선언된 query 24개·로컬 필드 9개](network/v2/subnets/README.md)를
`resource.WithFilter`/`WithFilters`로 자동 분류합니다. Python 이름의 query 변환과 원본
응답 비교를 SDK가 처리하며, 값 복사·bulk 교체·충돌 검사도 같은 옵션에서 제공합니다.
AddressGroup은 [query 8개·로컬 필드 3개](network/v2/extensions/security/addressgroups/listing/README.md)를
분류합니다. `project_id`는 서버 query이고 `tenant_id`는 원본 응답의 로컬 조건입니다.
`id`·`addresses`도 원문으로 비교하며 기존 이름 조회와 native 결과 모델을 유지합니다.
QoS Policy는 [query 15개·로컬 필드 2개](network/v2/extensions/qos/policies/listing/README.md)를
분류합니다. `is_shared`·태그 이름을 wire query로 변환하고 `rules`·`tenant_id`는 원문으로 비교합니다.
큰 rule 숫자는 정확하게 비교하며 반환값은 기존 typed Policy입니다.
Subnet Pool은 [query 16개·로컬 필드 10개](network/v2/extensions/subnetpools/listing/README.md)를
분류합니다. prefix length의 Python 이름을 변환하고 `id`·`tenant_id`·timestamp·prefix 배열을
원문으로 비교합니다. 정수 응답은 정확한 정수 정책을 사용하며 반환값은 기존 typed SubnetPool입니다.
Network도 [query 23개·로컬 필드 14개](network/v2/networks/listing/README.md)를 분류합니다.
`is_shared`·provider·address scope·태그 이름을 query로 변환하고 `subnet_ids`·확장 필드·timestamp는
원문 응답에서 비교합니다. boolean 응답의 truthiness와 두 정수 속성의 정확한 변환을 SDK가
처리하며, `conn.Network(ctx).Networks`와 versioned `Networks.Resources`에 같은 옵션을 사용합니다.
기존 이름 검색·상태 대기·native 반환 모델도 유지합니다.
Router는 [query 18개·로컬 필드 10개](network/v2/extensions/layer3/routers/listing/README.md)를
분류합니다. `is_admin_state_up`·`is_distributed`·`is_ha`와 태그 이름을 query로 변환하고,
gateway·routes·availability zone·timestamp·tenant ID를 원문에서 비교합니다.
Python `revision_number`는 원문 `revision`을 선택하며 native 반환값의 `RevisionNumber`와
구분합니다. `conn.Network(ctx).API.Routers.Resources`와 `conn.NetworkV2(ctx).Routers.Resources`에
같은 옵션을 사용합니다.

Security Group은 [query 17개·로컬 필드 3개](network/v2/extensions/security/groups/listing/README.md)를
분류합니다. `revision_number`·`tenant_id`·`stateful`·`is_shared`는 서버 조건이며, timestamp 두 개와
`security_group_rules`만 원문에서 비교합니다. rule의 추가 필드·큰 숫자·null 요소를 비교에
보존하고 기존 typed Security Group을 반환합니다. SDK가 concrete native 목록의 raw query도
처리하므로 호출자가 builder를 구현하지 않습니다.

Snapshot v2/v3의 `UpdateMetadata`는 실제 `metadata` 응답을 `map[string]any`로 반환합니다.
이전의 잘못된 Snapshot 반환형을 교정하고, 잘못된 응답은 오류로 처리합니다.
[v2 사용법](blockstorage/v2/snapshots/README.md)과 [v3 사용법](blockstorage/v3/snapshots/README.md)에
반환형 변경과 native PUT 교체·Python POST 병합의 차이를 설명합니다.

Volume·Snapshot v2/v3의 `MetadataIn(ctx, ref)`는 고정된 리소스의 메타데이터를 Get·Merge·Replace·DeleteKeys로 관리합니다. v3는 `conn.VolumeMetadata`와 `conn.SnapshotMetadata`로 바로 범위를 만들 수 있습니다. SDK가 문자열 map과 헤더 옵션, 빈 입력·전체 삭제·순서별 부분 성공을 처리하고 실제 응답을 반환합니다. [공통 metadata 사용법](blockstorage/metadata/README.md)과 [서버 계약 비교](docs/cinder-metadata-server-contracts.md)에 Python 대응·ETag·Backup의 경로 차이를 설명합니다.

Trunk는 [query 14개·로컬 필드 2개](network/v2/extensions/trunks/listing/README.md)를
분류합니다. `project_id`·`sub_ports`·`status`는 서버 조건이며 `id`·`tenant_id`만 원문에서
비교합니다. SDK가 admin state·태그 별칭과 옵션 snapshot을 처리하고 기존 이름·상태 필터와
typed Trunk 반환값을 유지합니다.

Barbican은 `conn.KeyManagerV1(ctx)`의 `SecretStores`로 목록·global default·preferred 조회를,
`Quotas`로 현재 프로젝트 quota와 고정 프로젝트의 override 조회·교체·삭제를 제공합니다.
SDK가 concrete options·기본 404 정책·요청 snapshot·실제 응답 증거를 처리합니다.
[SecretStore 비교](keymanager/v1/secretstores/README.md)와
[Quota 비교](keymanager/v1/quotas/README.md)에 Python/Go 사용법과 범위를 설명합니다.
`SecretConsumers.InSecret(ctx, resource.ID(secretID))`는 consumer association의 생성·삭제와
offset 목록을 제공합니다. [SecretConsumer 비교](keymanager/v1/secretconsumers/README.md)에서
association 입력과 실제 secret 응답을 구별합니다.
`Secrets.Fetch(ctx, resource.ID(secretID))`는 metadata를 조회한 뒤 content type이 있을 때만
payload를 가져옵니다. [Secret 조회 비교](keymanager/v1/secrets/README.md)에 기본 선택 규칙,
metadata 전용 옵션, 텍스트·바이너리 결과와 실패 시 응답 보존을 설명합니다.
`Secrets.FindIdentity(ctx, identity, options...)`는 직접 조회와 이름 검색을 SDK가 처리하며
공통 `resource.WithIdentityFindIgnoreMissing/Fallback` 옵션을 받습니다.
[Secret 자동 조회 비교](keymanager/v1/secrets/finding/README.md)에 두 조회 경로의 결과와 오류 정책을 설명합니다.
`Secrets.Resources.List/All`은 `resource.WithFilter/WithFilters`의 Python 속성 이름을
서버 query 또는 원문 응답 필터로 자동 분류합니다. [Secret 목록 비교](keymanager/v1/secrets/listing/README.md)에
`algorithm` 별칭, timestamp 원문과 `id`·`secret_id`의 서로 다른 비교 규칙을 설명합니다.
`Containers.Resources.List/All`도 같은 옵션으로 서버의 `limit`·`marker`와 로컬 Body 속성
10개를 분류합니다. `name`은 로컬 조건이며 원문 `secret_refs`·`consumers` 배열도 비교합니다.
[Container 목록 비교](keymanager/v1/containers/listing/README.md)에 `WithName`·raw query와의
차이, `id`·`container_id` 및 native decoder의 경계를 설명합니다.
`Orders.Resources.List/All`은 서버의 `limit`·`marker`와 로컬 Body 속성 14개를 분류합니다.
최상위 `name`과 `meta.name`, 전체 `id`와 별도 `order_id`·`secret_id`를 구분하며 원문
metadata도 비교합니다. [Order 목록 비교](keymanager/v1/orders/listing/README.md)에
사용 예제와 native 응답 모델·ID 조회의 경계를 설명합니다.
Zaqar는 `conn.MessagingV2(ctx)`의 `Subscriptions.InQueue(ctx, queueName)`으로 queue를 고정하고
구독 생성·조회·목록·삭제를 제공합니다. [Subscription 비교](messaging/v2/subscriptions/README.md)에
Client-ID·project header·TTL 기본값·marker 순회와 Python/Go 사용법을 설명합니다.
[Python/Go 목록 비교](docs/listing.md#명시적인-native-body-필터)에 raw query·로컬 cap·nil/빈 값과
native 응답 모델의 경계를 설명합니다.

Nova 서버/flavor·Cinder v3 볼륨·Glance v2 이미지·Neutron 네트워크/subnet/포트/router/security group/subnet pool/trunk/QoS policy/address group·Keystone 프로젝트/사용자/그룹/domain/role·
고정 zone의 Designate recordset·고정 pool의 Octavia member는 문자열을 그대로 받는 `FindIdentity`를 제공합니다.
[서비스별 Python/Go 자동 조회 비교](docs/finding-identities.md)에 공통 기본값·옵션·오류와
전체 서비스 사용 예제를 설명합니다.

| 서비스 | API 버전 문서 | Connection |
|---|---|---|
| Accelerator (Cyborg) | [v2](accelerator/v2/README.md) | `Accelerator(ctx)` |
| Bare Metal | [v1](baremetal/v1/README.md) | `BareMetal(ctx)` |
| Bare Metal Introspection | [v1](baremetalintrospection/v1/README.md) | `BareMetalIntrospection(ctx)` |
| Block Storage | [v2](blockstorage/v2/README.md), [v3](blockstorage/v3/README.md) | `BlockStorageV2(ctx)`, `BlockStorageV3(ctx)` |
| Clustering (Senlin) | [v1](clustering/v1/README.md) | `Clustering(ctx)` |
| Compute | [v2](compute/v2/README.md) | `ComputeV2(ctx)` |
| Container | [v1](container/v1/README.md) | `Container(ctx)` |
| Container Infra | [v1](containerinfra/v1/README.md) | `ContainerInfra(ctx)` |
| Database | [v1](db/v1/README.md) | `Database(ctx)` |
| DNS | [v2](dns/v2/README.md) | `DNS(ctx)` |
| Identity | [v2](identity/v2/README.md), [v3](identity/v3/README.md) | `IdentityV2(ctx)`, `Identity(ctx)` |
| Image | [v2](image/v2/README.md) | `ImageV2(ctx)` |
| Instance HA (Masakari) | [v1](instanceha/v1/README.md) | `InstanceHA(ctx)` |
| Key Manager | [v1](keymanager/v1/README.md) | `KeyManager(ctx)` |
| Load Balancer | [v2](loadbalancer/v2/README.md) | `LoadBalancer(ctx)` |
| Messaging | [v2](messaging/v2/README.md) | `Messaging(ctx)` |
| Metric (Aetos) | [v1](metric/v1/README.md) | `Metric(ctx)` |
| Object Storage | [v1](objectstorage/v1/README.md) | `ObjectStorage(ctx)` |
| Orchestration | [v1](orchestration/v1/README.md) | `Orchestration(ctx)` |
| Placement | [v1](placement/v1/README.md) | `Placement(ctx)` |
| Network | [v2](network/v2/README.md) | `NetworkV2(ctx)` |
| Reservation | [v1](reservation/v1/README.md) | `Reservation(ctx)` |
| Shared File System | [v2](sharedfilesystems/v2/README.md) | `SharedFileSystem(ctx)` |
| Workflow | [v2](workflow/v2/README.md) | `Workflow(ctx)` |

각 문서에는 openstacksdk와의 입력·결과 형식 비교, 실제 서비스 필드, API 패키지 링크, 이름 조회·삭제·대기가 적용되는 리소스를 기록합니다. DNS zone이나 Octavia pool의 자식 리소스는 [부모 범위를 지정](docs/scoped-resources.md)해서 사용합니다.

[Senlin 목록](clustering/v1/listing/README.md)은 11개 typed `List/All`의 `WithListFilter`, `WithListMaxItems`, `WithListPaginated`, `WithListHeader`, `WithListMicroversion`으로 로컬 필터·소비량·호출별 헤더와 버전을 선택합니다. [Manila access scope](sharedfilesystems/v2/shareaccessrules/README.md)의 목록도 raw cap과 한 응답 제어를 제공하며, 실제 service type·version header·부모 식별자를 SDK가 검증합니다. 기본값과 옵션 병합은 라이브러리가 처리하므로 별도 builder나 resource interface를 구현할 필요가 없습니다.

Heat는 `orchestration.Stacks.InStack(ctx, ref)` 또는 이름·ID가 모두 있는 `ForStack(identity)`로 대상을 고정합니다. Manila access rule은 `shared.ShareAccessRules.InShare(ctx, share)`에서 부모를 검증합니다. Nova quota는 `conn.ProjectQuotas(ctx, resource.Name("tenant"))`, Cinder는 `BlockStorageProjectQuotas`, Neutron은 `NetworkProjectQuotas`, Octavia는 `LoadBalancerProjectQuotas`, Manila는 `SharedFileSystemProjectQuotas`, Designate는 `DNSProjectQuotas`가 Keystone을 통한 이름 해석을 맡습니다. 각각의 `Current...ProjectQuotas(ctx)`는 기록된 Keystone project 인증 결과를 사용합니다. Nova·Manila의 `scope.InUser(ctx, user)`는 project·user를 함께 고정하고 Manila의 `InShareType`은 project·share type을 고정합니다. Quota singleton에는 Find/Wait가 없으며, 실제 목록 endpoint가 있는 Neutron·Octavia는 별도의 `API.ListProjects/AllProjects`를 제공합니다. Defaults의 프로젝트별·전역 구분과 [범위별 연산 및 사용법](docs/scoped-resources.md)을 참고합니다. 읽기 전용 limits는 Nova의 `conn.ProjectLimits(ctx, project)`와 Cinder의 `BlockStorageProjectLimits`로 고정하며, 일반 current-project 조회는 서비스의 `Limits.Fetch(ctx)`를 사용합니다. Cinder 프로젝트 필터는 선택한 숫자 버전 3.39 이상과 admin context가 필요합니다. Magnum은 `conn.ContainerInfraProjectQuotas(ctx, project)`에서 프로젝트를 고정한 뒤 `ForResource(quotas.Cluster)`로 리소스를 선택하며 Create/Get/Update/Delete를 제공합니다. [Magnum 사용법](containerinfra/v1/quotas/README.md)은 Python에 없는 quota API와 explicit hard limit 정책을 설명합니다.

## openstacksdk와 전체 사용 방식 비교

openstacksdk는 `Connection`에서 서비스 Proxy에 접근합니다. 이 프로젝트는 Go에서 오류와 context를 명시하도록 서비스 접근을 메서드로 제공합니다. 리소스 연산은 `Servers`, `Networks` 등의 typed collection에 모읍니다. [openstacksdk 사용 계층](https://docs.openstack.org/openstacksdk/latest/user/)

| 작업 | openstacksdk | gophercloudsdk |
|---|---|---|
| 연결 | `openstack.connect(cloud="dev")` | `sdk.Connect(ctx, sdk.WithCloud("dev"))` |
| 서비스 접근 | `conn.compute` | `conn.Compute(ctx)` → `*compute.Service, error` |
| ID 조회 | `conn.compute.get_server(id)` | `computeService.Servers.Get(ctx, id)` |
| 이름 조회 | `conn.compute.find_server("web", ignore_missing=False)` | `computeService.Servers.Find(ctx, resource.Name("web"))` |
| ID 참조 조회 | `conn.compute.find_server(id)` | `computeService.Servers.Find(ctx, resource.ID(id))` |
| 목록 | `conn.compute.servers(status="ACTIVE")` | `computeService.Servers.List(ctx, resource.WithStatus("ACTIVE"))` |
| 목록 수집 | `list(conn.compute.servers())` | `computeService.Servers.All(ctx)` |
| 목록 제한 | `max_items=50`, `paginated=False` | `resource.WithMaxItems(50)`, `resource.WithPaginated(false)`; Senlin typed API는 `WithListMaxItems`, `WithListPaginated` |
| 선택 인자 | keyword argument / 기본값 | 작업별 `With...` 옵션 / 문서화된 기본값 |
| 확장 입력 | `**attrs`, `**query` | `compute.WithField(...)`, `resource.WithQuery(...)` |
| 오류 | 예외 | 반환된 `error`, `errors.Is`, `errors.As` |
| 리소스 재사용 | Resource 인스턴스 전달 가능 | `resource.ID(server.ID)`로 명시적으로 참조 |
| 상태 대기 선택 인자 | `attribute`, `failures`, `callback`, `wait=None` | `WithStatusAttribute`, `WithFailureStates`, `WithProgressCallback`, `WithUnlimitedWait` |

이름과 ID를 하나의 문자열로 추측하지 않습니다. UUID처럼 생긴 이름도 `resource.Name(...)`이면 이름으로 찾습니다. 기본 응답 모델은 Gophercloud 타입의 alias를 사용합니다. 인증·virtual media·Swift 리소스에는 SDK가 추가 정보를 보관하는 모델을 제공합니다. 응답 변경을 추적하고 자동 저장하는 Resource 객체는 아직 제공하지 않습니다. 대기 옵션은 SDK가 모델의 필드와 진행률을 읽으며 호출자가 builder를 구현하지 않습니다. [초기 조회·callback·context의 Go 정책](resource/README.md)은 Python의 캐시된 Resource 대기와 다른 부분을 설명합니다.

### 서비스 여러 개를 사용하는 서버 생성

Python의 상위 연결 API:

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    network="private",
    wait=True,
    timeout=300,
)
print(server.id)
```

Go에서는 연결이 서비스 의존성을 주입하고, 서버 생성 계층이 이미지·flavor·네트워크 이름을 해석합니다:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/compute"
    "gophercloudsdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()

    conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
    if err != nil { log.Fatal(err) }
    computeService, err := conn.Compute(ctx)
    if err != nil { log.Fatal(err) }

    server, err := computeService.Servers.Create(ctx, compute.CreateServerRequest{
        Name: "web-01",
        Image: resource.Name("ubuntu"),
        Flavor: resource.Name("small"),
    }, compute.WithNetworks(resource.Name("private")),
       compute.WithMetadata(map[string]string{"team": "infra"}),
       compute.WithWait(resource.WithTimeout(5*time.Minute)))
    if err != nil {
        if server != nil { log.Printf("created server: %s", server.ID) }
        log.Fatal(err)
    }
    fmt.Println(server.ID)
}
```

ID를 이미 알고 있다면 `resource.ID(...)`를 사용하면 됩니다. 생성 의존성이 ID로 지정되면 사전 조회를 생략합니다. 위 두 예제의 공통 범위는 기존 이미지·flavor·네트워크를 사용한 생성과 상태 대기입니다. Python 상위 API의 추가 네트워크 자동 구성이나 floating IP 관리까지 구현한 것은 아닙니다. [openstacksdk Connection API](https://docs.openstack.org/openstacksdk/latest/user/connection.html)

빌드 가능한 예제는 [examples/create-server](examples/create-server/main.go)에 있습니다. 인자를 주고 실행하면 실제 클라우드에 서버를 생성합니다:

```sh
go run ./examples/create-server -name web-01 -image ubuntu -flavor small -network private
```

## 기본값과 공통 계약

| 항목 | 기본 동작 |
|---|---|
| 인증 | `OS_CLOUD`가 있으면 clouds.yaml, 없으면 Gophercloud의 `OS_*` 인증 파서 |
| 명시 인증 | `sdk.WithAuth(gophercloud.AuthOptions{...})`가 기본 인증 소스를 대체 |
| 명시 cloud | `sdk.WithCloud("dev")`가 환경의 cloud 선택을 대체 |
| 서비스 선택 | cloud/environment의 region/interface, 명시 옵션으로 덮어쓰기 |
| 서비스 연결 | 최초 접근 시 구성하고 성공한 서비스만 캐시 |
| Find | 정확한 이름 또는 명시 ID, 미존재는 `ErrNotFound` |
| 중복 이름 | 항상 `ErrAmbiguous`; 임의로 선택하지 않음 |
| Collection.Delete | 미존재는 성공; `WithMissingError()`로 엄격하게 변경. 비동기 Senlin 삭제는 service API가 Submission을 반환하며 force 기본값은 서비스 문서 참조 |
| List | 모든 페이지를 lazy iterator로 순회; `break`하면 후속 페이지를 읽지 않음 |
| All | 모든 결과를 메모리에 수집; 빈 목록은 빈 slice |
| 페이지 크기 | `WithPageSize`는 페이지 크기이며 전체 결과 개수 제한이 아님 |
| 로컬 목록 제한 | `WithMaxItems(50)`은 로컬 name/status 필터 전 raw 행 cap; 0 무제한, 음수는 순회 시 HTTP 전에 오류 |
| 첫 페이지 | `WithPaginated(false)`는 continuation을 읽지 않음; 기본 true, page 경계 없는 custom iterator는 `ErrUnsupported` |
| Wait | 대상 상태를 필수로 지정; 최대 5분, 간격 2초, context로 취소 가능 |
| Create | 생성 응답 즉시 반환; `compute.WithWait()`면 ACTIVE까지 대기 |
| 생성 후 대기 실패 | 생성된 서버와 오류를 함께 반환, 자동 삭제하지 않음 |

같은 설정에 대한 옵션은 뒤에 지정한 값이 우선합니다. `WithName`은 항상 정확한 이름 필터로 적용되며 `WithQuery("name", ...)`보다 우선합니다. `WithQuery`로 전달하는 이름 필터는 서비스 자체의 의미를 따릅니다.

목록 cap은 필터를 통과한 결과 수를 채우는 옵션이 아닙니다. Senlin은 raw 행을 decode·검증한
뒤 cap에서 즉시 멈추지만 native Gophercloud는 페이지 전체 extraction 때문에 뒤쪽 malformed
행의 오류도 반환할 수 있습니다. limit hint와 빈 페이지 종료는 서비스별 정책입니다.
[공통 목록 정책](resource/README.md), [서비스별 Python/Go 목록 비교](docs/listing.md)와
[Senlin/Python 비교](clustering/v1/listing/README.md)를 참고하세요. 생성된 native 일반 Collection
107개와 부모 scope 14개, Swift·Trove·Nova 이력의 수동 binding 5개에 같은 로컬 목록 옵션을
연결했습니다. native typed List의 기존 옵션은 유지하며 `Resources.List/All` 또는 scope의
`List/All`에서 `resource.WithMaxItems`와 `resource.WithPaginated`를 사용합니다.

공통 explicit Ref 조회는 원래 HTTP 오류를 보존합니다. [Senlin의 다섯 FindIdentity](clustering/v1/finding/README.md)는 Python처럼 GET 400·403·404 후 목록 검색을 제공하며, 목록의 권한 오류나 실패를 미존재로 숨기지 않습니다. `WithIgnoreMissing()`을 사용한 Find는 미존재일 때 `nil, nil`을 반환하므로 결과의 nil 여부를 확인해야 합니다.

```go
server, err := computeService.Servers.Find(ctx, resource.Name("web-01"))
switch {
case errors.Is(err, resource.ErrNotFound):
    // 이름이 없음
case errors.Is(err, resource.ErrAmbiguous):
    // 중복 이름: ID를 지정해야 함
case err != nil:
    return err
default:
    fmt.Println(server.ID)
}
```

위 조각은 `errors`, `fmt`, `resource`를 import하고, `ctx`와 `computeService`를 준비한 오류 반환 함수 안에서 사용합니다. 공통 사용법은 [resource README](resource/README.md)를 참고하세요.

## Go에 맞춘 선택 옵션과 확장

필수 입력은 concrete request struct, 선택 입력은 작업별 functional options로 받습니다. `WithConfigDrive(false)`처럼 false를 명시한 경우에도 값을 전송합니다. 옵션 종류를 분리해 List 옵션을 Create에 전달하면 컴파일 오류가 나도록 합니다.

```go
// 서비스별 query 확장: 애플리케이션의 builder 구현이 필요하지 않음
networks, err := networkService.Networks.All(ctx,
    resource.WithQuery("router:external", "true"))

// 서버 생성 확장: server 객체 내부에 JSON 필드를 추가
vendorOption := compute.WithField("vendor_hint", map[string]any{"pool": "fast"})
// vendorOption을 Servers.Create의 옵션으로 전달
```

`vendor_hint`는 확장 경로를 보여주는 예시이며 표준 Nova 필드가 아닙니다. 해당 필드를 지원하는 클라우드에서만 사용해야 합니다. 기본 생성 필드와 충돌하거나 JSON으로 변환할 수 없는 값은 요청 전에 거부합니다. JSON 필드의 실제 스키마와 microversion 요구는 서버가 검증합니다. 현재 Gophercloud CreateOpts의 필드는 core 필드로 보호되며, 일부 core 필드는 아직 SDK의 typed 옵션으로 노출하지 않았습니다.

현재 미지원 API는 서비스의 `RawClient()`로 Gophercloud concrete options를 사용할 수 있습니다. 이를 상위 SDK의 지원 기능으로 간주하지 않습니다. 공유 클라이언트 설정은 요청을 동시에 실행하기 시작한 뒤 변경하지 마세요.

서버 생성 뒤 새 floating IP를 연결하는 전체 실행 예제는 [create-server-and-ip](examples/create-server-and-ip/main.go)에 있습니다. `-name`, `-image`, `-flavor`, `-network`, `-external-network`를 지정하며, 포트의 IPv4가 여러 개면 `-fixed-address`로 선택합니다. 실패한 단계에서 앞서 생성한 리소스를 자동 삭제하지 않습니다.

## 개발과 검증

```sh
go mod download
make check
go test -coverpkg=./... ./...
go build ./examples/...
```

테스트는 로컬 `httptest.Server`를 사용합니다. 실클라우드 자격 증명이 필요하지 않으며 OpenStack 리소스를 생성하지 않습니다. 테스트 환경은 localhost 포트 바인딩을 허용해야 합니다. [테스트 구성](docs/testing.md), [설계 및 확장 계획](docs/design.md)을 참고하세요.

Senlin의 이름·UUID·짧은 ID 자동 조회는 `Profiles/Policies/Clusters/Nodes/Receivers.FindIdentity(ctx, identity, options...)`로 사용합니다. SDK가 GET-first·목록 fallback·정확한 ID/이름·전체 페이지 중복 검사를 담당하며 기본 미존재는 `nil, nil`입니다. [서비스별 Python/Go 사용 비교와 호출별 옵션](clustering/v1/finding/README.md)을 참고합니다.
