# gophercloudsdk

Gophercloud 위에 연결, 서비스, 리소스, 복합 작업의 일관된 사용 방식을 제공하는 Go SDK 프로젝트입니다. 애플리케이션이 기본값, 이름 조회, 페이지네이션, 상태 대기, 요청 builder를 반복해서 구현하지 않도록 하는 것이 목적입니다.

현재는 **개발 중**입니다. 고정한 Gophercloud의 공개 API 호출은 제공하며, openstacksdk 수준의 리소스·복합 작업 계층을 확장하고 있습니다. Go 1.25 이상과 Gophercloud v2.15.0을 사용합니다. 모듈 경로는 `github.com/JSYoo5B/gophercloudsdk`입니다. 옆 디렉토리의 개발 브랜치에 의존하는 `replace`는 사용하지 않습니다.

[구현 순서와 단계별 현황](docs/implementation-plan.md)은 **핵심 user API → 핵심 admin API → 매니지드 user API → 매니지드 admin API** 순으로 작업을 배치하고 소스 검토·구현·테스트·문서·최종 판정·커밋과 push를 구분합니다. 전체 API의 지원 범위와 완료 기준은 [지원 판정대장](docs/sdk-support-ledger.md)에서 확인합니다.

**현재 SDK 완료 수와 진행 중인 작업:** [자동 집계·현재 단계](docs/implementation-plan.md#현재-집계와-진행-중인-작업). 아래의 Gophercloud 공개 연산 수는 고정 transport 목록이며, SDK 완료 수는 위 링크의 연산별 판정으로 확인합니다.

[외부 Go 프로젝트 설치 안내](docs/install.md)에서 공개 import와 검증한 커밋의 사용법을 확인합니다.

[Keypair 목록·검색](compute/keypairs-list-find.md)은 SDK가 필터·페이지 순회·owner를 유지하는 검색 fallback과 nullable Resource·실제 응답을 처리합니다.

[Cloud keypair 조합](compute/keypairs-cloud.md)은 `conn.Compute(ctx)`의 `ListKeypairs`·`SearchKeypairs`·`GetKeypair`·`CreateKeypair`·`DeleteKeypair`를 비교합니다. eager 목록·필터 presence·public key 생략·삭제 bool을 SDK가 처리하고, leaf의9필드 Resource에 Connection location을 보충하며 실제 Wire는 보존합니다. [명시 flavor extra-specs 조회](compute/flavor-extra-specs.md)는 `service.API.Flavors.FetchExtraSpecs`로 inline 값과 관계없이 조회하는 사용법을 설명합니다.

[Console auth-token 조회](compute/console-auth-token.md)는 명시 token으로 연결 정보를 GET하고 source-shaped Resource·actual Wire·receipt를 반환합니다.

[Compute console 자동 선택](compute/console-selection.md)은 `conn.Compute(ctx)`에서 서버가 광고한 범위에 따라 modern/legacy API를 선택합니다.

**사용 가능한 개발 preview:** `make smoke`로 핵심 user 5개 흐름을 실행하고 실제 결과를 확인합니다. [단계별 사용·릴리즈 기준](docs/release-milestones.md)에 현재 검증 범위와 첫 외부 설치용 alpha의 남은 조건을 기록합니다.

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
| [Compute](compute/README.md) | 서버 | 지원 | 지원 | 지원 | 이미지·기존 볼륨·새 부팅 볼륨, [NIC·port·고정 IP·tag와 auto/none](compute/server-network-interfaces.md), [Connection·YAML 기본 네트워크](compute/server-default-network.md), 이름 해석, 선택적 대기 |
| Compute | flavor | 지원 | 미지원 | 미지원 | 미지원 |
| [Network](network/README.md) | 네트워크 | 지원 | 지원 | 지원 | 미지원 |
| Network | 포트 | 지원 | 지원 | 지원 | 개별 API |
| Network | floating IP | ID 조회·Find·목록 | 지원 | 지원 | 네트워크·서버·포트 해석, IPv4 선택, 선택적 대기 |
| [Image](image/README.md) | 이미지 | 지원 | 지원 | 지원 | metadata 생성·직접 업로드·생성→staging→import·선택적 대기·기존 이미지 import·checksum 다운로드·저장소 복사본 삭제 |
| [Block Storage](blockstorage/README.md) | 볼륨 | 지원 | 지원 | 지원 | 이미지 선택·속성 매핑·기본 available 대기·선택적 bootable 설정 |

기존 Cinder 볼륨을 서버에 연결할 때는 `conn.AttachVolume`이 이름 해석·새 상태 확인·Nova 연결 요청·기본 완료 대기를 처리합니다. `blockstorage.WithAttachVolumeWait(false)`로 접수 결과만 받거나 대기 정책을 지정할 수 있으며, 대기 실패에도 연결 응답을 보존합니다. [Python과의 호출 비교 및 부분 결과](blockstorage/attach-volume.md)를 참고하세요. 기존 볼륨 분리는 `conn.DetachVolume`이 Nova DELETE와 기본 Cinder 완료 대기를 처리합니다. 대기를 끄고 볼륨 ID를 지정하면 Cinder 초기화 없이 호출할 수 있습니다. [분리 사용법과 Python 비교](blockstorage/detach-volume.md)에 실제 timeout과 접수 결과를 설명합니다. 새 볼륨은 `conn.CreateVolume`이 생성·기본 대기·선택적인 bootable 처리를 묶습니다. [생성 사용법과 Python 비교](blockstorage/create-volume.md)에 이미지 선택과 생성 속성을 설명합니다. 삭제는 `conn.DeleteVolume`이 초기 조회·선택적 강제 요청·기본 완료 대기를 처리합니다. [삭제 사용법과 Python 비교](blockstorage/delete-volume.md)에 초기 부재와 삭제 경합, 단계별 결과를 설명합니다.

Cinder snapshot은 `conn.ListVolumeSnapshots`, `SearchVolumeSnapshots`, `GetVolumeSnapshot`, `GetVolumeSnapshotByID`로 조회합니다. SDK가 상세 목록 기본값·서버와 로컬 필터 분류·검색·nullable 변환·단계별 응답 증거를 처리합니다. [Snapshot 조회와 Python 비교](blockstorage/volume-snapshots.md)에 source의 seeded ID, maximum/pagination 경계와 raw resource 사용법을 설명합니다.

생성·삭제는 `conn.CreateVolumeSnapshot`·`DeleteVolumeSnapshot`이 속성 매핑·이름/ID 해석·선택적인 완료 대기를 처리합니다. 생성은 기본으로 기다리고 삭제는 접수 후 반환합니다. [Snapshot 생성·삭제와 Python 비교](blockstorage/volume-snapshot-mutations.md)에 기본값·timeout·단계별 부분 결과를 설명합니다.

이미 받은 owned/native 볼륨의 attachment device는 `blockstorage.GetVolumeAttachDevice`로 조회합니다. 원래 JSON 값은 `GetVolumeAttachDeviceFields`로 읽으며 두 함수 모두 HTTP 없이 동작합니다. [서버별 device 조회와 Python 비교](blockstorage/volume-attachment-device.md)에 네 가지 typed 입력, 첫 일치와 local 오류를 설명합니다.

서버에 연결된 볼륨은 `conn.GetVolumes`로 조회합니다. 전체 목록을 읽은 뒤 attachment를 비교하고 raw resource를 반환하며, 오류 시 실제 페이지 증거를 남깁니다. [서버별 볼륨 목록과 Python 비교](blockstorage/server-volumes.md)에 ServerFields·중복 pointer·Decode·Clone을 설명합니다.

[Glance Task 생성·조회·목록](image/tasks.md)은 `image.Service`의 공통 옵션과 기본 입력, 원문 응답 모델, lazy 페이지 순회를 제공합니다.

[이미지별 Task 목록](image/image-tasks.md)은 이미지 ID·이름 선택과 공통 옵션, 삭제 여부·시각과 원문 응답을 제공합니다.

[Glance 이미지 조회·목록](image/images.md)은 concrete 선택 인자, nullable 모델과 확장 속성, 반복 필터를 보존하는 페이지 순회와 Python 사용법 비교를 제공합니다.

[Glance 이미지 수정](image/update.md)은 순서가 있는 concrete 패치와 속성 upsert, 값·삭제 구분 및 Python Resource 사용법 비교를 제공합니다.

[Glance 직접 업로드](image/upload-image.md)는 concrete 옵션, 메타데이터 생성과 바이너리 전송의 단계별 응답, caller 스트림 소유권과 Python 사용법 비교를 제공합니다.

[Swift 계정 메타데이터](objectstorage/v1/accounts/README.md)는 concrete 옵션과 nullable 카운터, 조회·설정·삭제의 실제 응답 및 Python 사용법 비교를 제공합니다.

[Swift 컨테이너 메타데이터](objectstorage/v1/containers/README.md)는 literal 이름과 concrete 옵션, 시스템 헤더 및 Python 사용법 비교를 제공합니다.

Network 생성·수정·삭제는 `service.CreateNetwork`, `UpdateNetwork`, `DeleteNetwork`로 호출합니다. [Python/Go CRUD 비교](network/network-mutations.md)에 SDK 기본값·concrete 옵션·provider/AZ·생략/null/false·부분 응답과 공유 역할 cache 자동 초기화를 설명합니다.

[네트워크 역할 조회](network/network-roles.md)는 Connection이 외부·내부 IPv4/IPv6와 NAT·기본 인터페이스를 분류합니다. Getter, configured 기본 NIC, 자동 floating source와 조건부 NAT 선택이 성공한 탐색을 공유합니다. YAML 또는 `WithNetworkRoles`의 concrete 옵션을 사용하며, 별도 builder나 resolver 구현이 필요하지 않습니다. 반환 모델을 수정해도 다른 호출에 영향을 주지 않고 `ResetNetworkRoles`로 다시 탐색할 수 있습니다. [서버 주소 계산·보충](compute/server-addresses.md)은 같은 역할 cache와 concrete private/IPv6/source 설정을 사용합니다. [Floating IP 연결 계획](network/floating-ip-plan.md)은 owner-free 선택과 같은 대상 실행을 분리합니다. [기존 서버의 자동 floating IPv4](compute/server-automatic-ip.md)는 필요성·lazy skip을 판단하고 Neutron 계획 또는 legacy Nova backend를 실행한 뒤 raw Nova 응답에서 목표 주소를 관측합니다. [서버 생성과 자동 IP](compute/create-with-automatic-floating-ip.md)는 생성·실제 ACTIVE/주소 준비·조건부 연결·Nova 관측을 단일 시간 예산으로 이어가며 초기 생성 응답과 마지막 서버를 보존합니다. [기존 서버의 ACTIVE 판정·상위 대기](compute/server-ready.md)는 supplied 상태 조회의 기본 비동기 IP 접수와 raw 현재 상태·주소 준비·관측을 기다리는 180초 작업을 제공합니다. [IP 선택 순서와 부분 결과](compute/server-ip-dispatch.md)는 pool → 순차 명시 IPv4 → automatic 우선순위를 Plan/Ensure·GetActive/Wait·자동 생성에 연결합니다. 일반 Create/Wait 전체 계약과 direct proxy add/remove·함수별 fallback·전체 cloud Resource/session 정책은 남은 범위입니다.

[기존 Floating IP 연결](network/floating-ip-attach.md)은 지정한 ID 또는 IPv4 주소와 목적지를 고정해 owner 조회나 새 allocation 없이 연결합니다. 같은 IP 주소와 준비 시점 revision을 검증하며 늦은 실패에도 알려진 IP와 HTTP 증거를 반환합니다. [Compute의 IP dispatch](compute/server-ip-dispatch.md)가 이 연결 기반과 Ensure를 사용해 순차 IPv4 목록과 pool 선택·대기·Nova 관측을 수행하고 앞선 완료와 현재 부분 결과를 보존합니다.

[독립 IP helper](compute/server-ip-helpers.md)는 `conn.AddIPsToServer`·`AddIPList`로 기존 서버에 IP를 추가합니다. 기본은 전체60초·비동기이고, 선택적인 wait는 서버 상태와 무관하게 raw 주소를 관측합니다. 빈 positional 목록은 서비스 탐색 없이 반환하며 기존 Get/Wait/Create의 ACTIVE 준비 조건은 유지합니다.

[Legacy Nova floating IP](compute/server-nova-floating-ip.md)는 기존 IP 소비자에 pool 선택·재사용/할당·add action을 연결합니다. 호환 Compute microversion에서 실행하며 `NovaAssignment`에 실제 모델·allocation/action 응답을 보존합니다. Neutron IP ACTIVE를 합성하지 않고 entry별 raw 서버 주소 관측을 적용합니다.

[독립 Floating IP 조회·할당](compute/floating-ip-available.md)은 `conn.AvailableFloatingIP`으로 첫 free IP 또는 새 allocation을 반환합니다. Neutron 목록404는 내부 Nova 목록을 같은 Neutron 필터로 검사하고, 후보가 없으면 선택한 Neutron network에 할당합니다. helper의 pure pre-accept NotFound만 외부 Nova availability로 전환합니다. free Neutron IP는 Server를 지정해도 연결하지 않으며, 실제 backend와 접수 후 부분 결과를 보존합니다. [직접 Network Available](network/floating-ip-available.md)은 Neutron 전용 경로입니다.

Create/Update와 각 서비스의 API 호출은 버전별 패키지에서 concrete options로 사용합니다. [Compute·Cinder·Image의 서비스별 대기](docs/service-waits.md)는 기본 상태·실패 상태·timeout을 SDK가 적용하고 `With...` 옵션으로 호출별 변경을 받습니다. [Glance Task 대기](image/v2/tasks/README.md)는 정확한 396 오류에서 받은 type/input으로 재생성하고 새 ID를 같은 시간 제한으로 조회하며 실제 응답과 부분 실패를 반환합니다. [이미지 staging](image/v2/imagedata/README.md)은 queued 확인·단일 바이너리 전송·후속 조회를 제공하고 실제 접수 증거와 조회 결과를 분리합니다. [이미지 import](image/v2/imageimport/README.md)는 format 검증·저장소 선택·명시적 false와 root/method 확장 필드를 concrete 옵션으로 조립해 실제 202 접수 응답을 반환합니다. [이미지 삭제](image/delete.md)는 전체 또는 저장소 복사본을 선택하고 실제 삭제 응답과 미존재·응답 처리 오류를 구별합니다. [이미지 다운로드](image/download.md)는 저장소 우선순위와 checksum 검사를 SDK가 처리하고 writer에 쓴 바이트 수와 실제 응답·부분 오류를 보존합니다. [이미지 생성·import](image/create-import.md)는 metadata·stage·import·wait 정책을 첫 요청 전에 준비하고 실패 시 단계별 실제 접수 결과를 보존합니다. [microversion 범위 협상](docs/microversions.md)과 [볼륨 부팅 옵션](compute/README.md)을 제공하며, [floating IP 생성·연결](network/README.md)과 [재사용·자동 외부 network 선택](network/floating-ip-ensure.md)과 [서버 생성·IP 연결 작업](compute/create-with-floating-ip.md)과 [이미지 직접 업로드](image/README.md)를 제공합니다. Senlin Profile·Policy는 [변경 추적과 Commit](clustering/v1/tracking/README.md), Cluster·Node는 [비동기 변경 추적과 accepted Operation](clustering/v1/tracking/async/README.md)을 제공합니다. 다른 리소스의 변경 추적, 서버 생성의 direct proxy add/remove·함수별 fallback·전체 service/config/shared-network·Resource/session 정책, 이미지의 checksum·기존 이미지 재사용·Swift task 흐름과 재시도할 입력을 명시적으로 준비하는 기능는 계속 구현할 대상입니다. [SDK 지원 판정대장](docs/sdk-support-ledger.md)은 확인한 차이와 전체 완료의 기준을 기록합니다. Senlin은 [전용 상태·삭제 대기](clustering/v1/waiting/README.md), [고정 cluster의 policy 조회](clustering/v1/clusterpolicies/README.md), [node attribute 수집](clustering/v1/clusterattributes/README.md), [cluster metadata 관리](clustering/v1/clusters/metadata/README.md)와 [목록 행 수·첫 페이지 제어](clustering/v1/listing/README.md)를 제공합니다. [서버 계약 비교](docs/senlin-server-contracts.md)는 API reference와 실제 release 코드의 성공 코드·metadata 경로 차이를 기록합니다.

## 모든 서비스의 사용 문서

Keystone의 [사용자별 프로젝트·그룹 조회](identity/v3/users/memberships.md)는 native `Project`·`Group` iterator와 SDK 소유 `ListProjectRecords`·`ListGroupRecords`를 제공합니다. 프로젝트는 concrete 옵션으로 query·별칭·로컬 Body 필터·행 수·페이지 정책을 조립하고, 그룹은 공개 query 인자 없는 source 호출을 유지합니다. 두 목록 모두 사용자 범위의 Resource view와 실제 Wire 응답을 분리합니다. Python 비교와 두 독립 Go 예제에서 반환형·부분 결과·호환성을 설명합니다.

Neutron의 [9개 리소스 조건부 Update](network/v2/revision-updates.md)는 concrete `UpdateOpts.RevisionNumber`를 HTTP `If-Match` 조건으로 전달합니다. nil과 revision 0을 구별하고 `WithUpdateOptions`로 선택한 최종 조건을 사용합니다. 412 충돌은 오류로 반환하므로 호출자가 충돌 처리 정책을 선택합니다.

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
`secretstores.WithListFilter`/`WithListFilters`로 선언된 서버 query와 로컬 속성을 전달하면 SDK가 분류·원문 비교·페이지 cap을 처리합니다. [SecretStore 비교](keymanager/v1/secretstores/README.md)와
[Quota 비교](keymanager/v1/quotas/README.md)에 Python/Go 사용법과 범위를 설명합니다. 현재 프로젝트 quota만 읽을 때는 `service.Quotas.Get(ctx)`를 사용하며, 프로젝트 ID 없이 원문 값과 실제 HTTP 오류 증거를 반환합니다.
Container/Order의 원문·nullable metadata 조회는 `service.Containers.Fetch(ctx, resource.ID(id))`와 `service.Orders.Fetch(ctx, resource.ID(id))`를 사용합니다. [Python/Go 조회 비교](keymanager/v1/metadata-fetch.md)에 결과와 실행 예제를 제공합니다.
Container/Order/Secret의 `Remove(ctx, resource.ID(id))`는 기본으로 없는 객체를 무시하고 `resource.WithMissingError()`로 strict 삭제를 선택합니다. [Python/Go 삭제 비교](keymanager/v1/metadata-delete.md)에 공통 정책과 실행 예제를 제공합니다.
`SecretConsumers.InSecret(ctx, resource.ID(secretID))`는 consumer association의 생성·삭제와
offset 목록을 제공합니다. [SecretConsumer 비교](keymanager/v1/secretconsumers/README.md)에서
association 입력과 실제 secret 응답을 구별합니다.
`Secrets.GetPayload(ctx, secretID)`는 payload GET의 전체 bytes를 반환하고 응답을 닫습니다.
[payload 조회와 반환형 변경](keymanager/v1/secrets/payload.md)에 native 기본 Accept·옵션·오류와
기존 Download 반환에서 `[]byte`로 바꾸는 방법을 설명합니다.
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

Trove `Instances.IsRootEnabled(ctx, instanceID)`는 root 조회의 native 오류를 panic 없이 반환합니다.
[Python/Go root 접근 상태 사용법](db/v1/instances/root-access.md)에 boolean 판정과 오류 정책을 설명합니다.


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
| 볼륨 이미지 내보내기 | `conn.block_storage.upload_volume_to_image(volume, image_name, ...)` | `conn.UploadVolumeToImage(ctx, request, imageName, options...)` |

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

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/compute"
    "github.com/JSYoo5B/gophercloudsdk/resource"
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

ID를 이미 알고 있다면 `resource.ID(...)`를 사용하면 됩니다. 생성 의존성이 ID로 지정되면 사전 조회를 생략합니다. 위 두 예제의 공통 범위는 기존 이미지·flavor·네트워크를 사용한 생성과 상태 대기입니다. Python 상위 API의 추가 네트워크 자동 구성이나 floating IP 관리를 자동으로 포함하지 않습니다. 기존 서버에 IP를 재사용·연결하는 별도 [Ensure 작업](network/floating-ip-ensure.md)을 제공합니다. [openstacksdk Connection API](https://docs.openstack.org/openstacksdk/latest/user/connection.html)

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

서버 생성·IP 재사용·두 ACTIVE 대기를 하나로 처리하는 [SDK 작업](compute/create-with-floating-ip.md)을 제공합니다. 각 단계를 직접 호출해 서버 생성 뒤 새 floating IP를 연결하는 전체 실행 예제는 [create-server-and-ip](examples/create-server-and-ip/main.go)에 있습니다. `-name`, `-image`, `-flavor`, `-network`, `-external-network`를 지정하며, 포트의 IPv4가 여러 개면 `-fixed-address`로 선택합니다. 실패한 단계에서 앞서 생성한 리소스를 자동 삭제하지 않습니다.

## 개발과 검증

```sh
go mod download
make check
go test -coverpkg=./... ./...
go build ./examples/...
```

테스트는 로컬 `httptest.Server`를 사용합니다. 실클라우드 자격 증명이 필요하지 않으며 OpenStack 리소스를 생성하지 않습니다. 테스트 환경은 localhost 포트 바인딩을 허용해야 합니다. [테스트 구성](docs/testing.md), [설계 및 확장 계획](docs/design.md)을 참고하세요.

Senlin의 이름·UUID·짧은 ID 자동 조회는 `Profiles/Policies/Clusters/Nodes/Receivers.FindIdentity(ctx, identity, options...)`로 사용합니다. SDK가 GET-first·목록 fallback·정확한 ID/이름·전체 페이지 중복 검사를 담당하며 기본 미존재는 `nil, nil`입니다. [서비스별 Python/Go 사용 비교와 호출별 옵션](clustering/v1/finding/README.md)을 참고합니다.

Glance 저장소 목록·상세 목록과 import 정보는 `conn.ImageV2(ctx).ServiceInfo`와 상위 `conn.Image(ctx)`의 `API.ServiceInfo`에서 조회합니다. [서비스 정보 비교](image/v2/serviceinfo/README.md)는 concrete 옵션·생략/null·raw 응답·목록 소비 제어와 openstacksdk 사용법을 설명합니다.

Glance 인증 프로젝트의 limit·사용량은 `ServiceInfo.GetUsageInfo(ctx)`로 조회합니다. [사용량 조회](image/v2/serviceinfo/usage.md)는 동적 resource·정확한 int64·생략/null·빈 결과와 Python SDK의 지원 경계를 설명합니다.

Glance의 단일 태그 추가·삭제와 이미지 비활성화·재활성화는 상위 `image.Service.AddTag/RemoveTag/DeactivateImage/ReactivateImage`로 사용합니다. [태그·상태 변경 비교](image/mutations.md)는 concrete header 옵션·ID/Name 해석·실제204 접수와 부분 응답을 설명합니다.

Glance location 추가·조회는 상위 `image.Service.AddImageLocation/GetImageLocations`로 사용합니다. [Location의 Python/Go 비교](image/locations.md)는 기본 validation_data·concrete hash/header 옵션·실제202 접수와 GET200 배열을 설명합니다.

Glance schema 조회 16개는 상위 `conn.Image(ctx)`의 `Get*Schema(ctx, options...)`로 사용합니다. [Schema의 Python/Go 비교](image/schemas.md)는 공통 concrete header 옵션·실제200·nullable canonical 필드와 다양한 `additionalProperties` raw 값을 설명합니다.

`GetSchemaRecord(ctx, image.SchemaMetadefNamespace)`는 같은16경로를 concrete `SchemaKind`로 선택해 ordinary/Metadef class의 nullable·dict/bool/list 변환과 Connection location을 적용합니다. 실제 응답 Wire·Envelope·receipt는 별도로 보존합니다. [클래스별 Python/Go 비교와 독립 main](image/schema-records.md)에 기본값·ordered alias·빈/비JSON 응답과 오류 처리를 설명합니다.

Glance 캐시 API 7개는 상위 `conn.Image(ctx)`의 전용 메서드로 사용합니다. [캐시의 Python/Go 비교](image/cache.md)는 삭제 기본값·cache/queue target 옵션·정확한 숫자·실제 응답 증거와 비동기 queue 동작을 설명합니다.

Glance 이미지 공유 멤버는 상위 `conn.Image(ctx)`의 `AddImageMember/GetImageMember/UpdateImageMember/RemoveImageMember/FindImageMember/ListImageMembers/AllImageMembers`로 사용합니다. [멤버의 Python/Go 비교](image/members.md)는 필수 부모·ID·concrete 기본값·유한 목록과 실제 응답 증거를 설명합니다.

Glance metadata namespace는 `conn.Image(ctx)`의 `service.API.MetadefNamespaces` 또는 `conn.ImageV2(ctx)`의 `service.MetadefNamespaces`로 사용합니다. [Namespace의 Python/Go 비교](image/v2/metadefnamespaces/README.md)는 literal 이름·concrete 기본값·PUT 교체·명시 zero limit과 서버의 다음 페이지를 설명합니다. 생성 시 네 `WithCreate…` container 옵션으로 property·object·tag·resource type 연결을 한 요청에 담을 수 있습니다.

Glance metadata object는 `service.API.MetadefObjects.InNamespace(ctx, namespace)`로 범위를 만든 뒤 사용합니다. [Object의 Python/Go 비교](image/v2/metadefobjects/README.md)는 고정 namespace·concrete 기본값·PUT 교체·유한 목록과 개별/일괄 삭제의 차이를 설명합니다.

Glance metadata property는 `service.API.MetadefProperties.InNamespace(ctx, namespace)`에서 사용합니다. [Property의 Python/Go 비교](image/v2/metadefproperties/README.md)는 필수 Type·Title, With 기반 JSON keyword, Get의 resource_type, dictionary Key·Name과 PUT 교체를 설명합니다.

Glance metadata tag는 `service.API.MetadefTags.InNamespace(ctx, namespace)`에서 사용합니다. [Tag의 Python/Go 비교](image/v2/metadeftags/README.md)는 bodyless Create, strict 삭제, Set의 append 기본값과 빈 입력, limit·marker 기반 목록을 설명합니다.

Glance metadata resource type의 전역 목록은 `service.API.MetadefResourceTypes.List/All`로 조회하고, namespace 연결은 `InNamespace(ctx, namespace)`에서 생성·목록·해제합니다. [Resource type의 Python/Go 비교](image/v2/metadefresourcetypes/README.md)는 공통 목록 옵션, Prefix·PropertiesTarget의 생략/빈 값과 삭제 기본값을 설명합니다.

Swift object metadata는 `conn.ObjectStorage(ctx)`의 `service.Objects.GetMetadata/SetMetadata/DeleteMetadata`로 사용합니다. [객체 metadata의 Python/Go 비교](objectstorage/v1/objects/README.md)는 함수 옵션, 기존 값과 시스템 header 보존, 조회·변경 응답의 분리, 링크 객체와 동시 변경의 제약을 설명합니다.

Swift 컨테이너는 `service.Containers.CreateContainer(ctx, name)`와 `DeleteContainer(ctx, name)`으로 생성·삭제합니다. [Python/Go 생성·삭제 비교](objectstorage/v1/containers/README.md)는 메타데이터·시스템 헤더 옵션, 기본 404 허용과 strict 삭제, 실제 HTTP 응답을 설명합니다.

Swift Temp URL key는 `service.Accounts.SetTempURLKey`, `service.Containers.SetTempURLKey`와 `service.GetTempURLKey`로 설정·조회합니다. [Python/Go 키 관리 비교](objectstorage/v1/temp_url_key.md)는 기본 primary·secondary 함수 옵션, container→account 선택 순서와 단계별 실제 응답을 설명합니다.

Swift 서명은 `service.GenerateFormSignature`와 `service.GenerateTempURL`로 생성합니다. [Python/Go 서명 비교](objectstorage/v1/signing.md)는 구조체 입력과 함수 옵션, binary key·만료 시간·URL 인코딩과 자동 키 조회를 설명합니다.

Swift 객체 바이트 읽기는 `Objects.GetObject`·`DownloadObject`·`StreamObject`와 공유 함수 옵션으로 제공합니다. [Python/Go 조회·다운로드·스트리밍 비교](objectstorage/v1/objects/read.md)에 binary 반환, writer·stream 소유권과 조건부 응답을 설명합니다.

Swift 서비스 capability는 `service.GetInfo(ctx)`로 조회하고, 객체 segment 크기는 `GetObjectSegmentSize(ctx, options...)`로 선택합니다. [Python/Go 사용법](objectstorage/v1/info.md)에 기본값·확장 JSON·404/412 fallback과 실제 응답 증거를 설명합니다.

Swift 객체 삭제는 `Objects.DeleteObject(ctx, container, object, options...)`로 사용합니다. [Python/Go 삭제 비교](objectstorage/v1/objects/delete.md)에 기본 HEAD 확인, known SLO 함수 옵션, 단계별 실제 응답과 bulk 부분 실패를 설명합니다.

Swift 객체 생성은 `Objects.CreateObject(ctx, container, object, input, options...)`으로 사용합니다. [Python/Go 생성과 stale 비교](objectstorage/v1/objects/create.md)에 bytes·file·Reader 입력, checksum과 자동 SLO/DLO, 재시도·단계별 응답 및 보수적인 부분 정리를 설명합니다.

Swift 객체 대기는 `Objects.WaitForDelete`·`WaitForStatus`로 사용합니다. [Python/Go 대기 비교](objectstorage/v1/objects/wait.md)에 HEAD 조회, 기본 2초 간격·삭제 120초 제한, 명시한 상태 속성·응답 헤더와 마지막 실제 응답을 설명합니다.

Swift 디렉터리 마커는 `Objects.CreateDirectoryMarkerObject(ctx, container, name, options...)`으로 생성합니다. [Python/Go 사용법](objectstorage/v1/objects/directory-marker.md)에 SDK가 준비하는 빈 객체와 Content-Type, metadata 함수 옵션과 실제 업로드 응답을 설명합니다.

볼륨 cloud 검색은 `Connection.SearchVolumes`와 `Connection.GetVolume`이 glob·중첩 필터·JMESPath와 계산된 location을 제공합니다. 필터 생략·null의 기본 조회와 명시적 필터의 전체 검색, 임의 JSON 결과와 원본 응답은 [서비스별 Python/Go 비교](blockstorage/search-volumes.md)에 설명합니다.

볼륨 전체 목록·ID 직접 조회·존재 확인은 `conn.ListVolumes`, `conn.GetVolumeByID`, `conn.VolumeExists`로 호출합니다. [조회 사용법과 Python 비교](blockstorage/volume-reads.md)에 404 오류, 정상 부재와 조회 실패, 원본 응답·정규화 값·페이지 증거를 설명합니다.

`conn.GetVolumeID`는 이름·ID를 실제 조회하고 응답의 ID JSON을 그대로 반환합니다. [ID 조회와 Python 비교](blockstorage/volume-id.md)에 정상 부재·null ID·오류와 실제 응답 증거의 구분을 설명합니다.

볼륨 타입 목록·검색·조회는 `conn.ListVolumeTypes`, `conn.SearchVolumeTypes`, `conn.GetVolumeType`으로 호출합니다. [타입 사용법과 Python 비교](blockstorage/volume-types.md)에 함수 옵션, nullable 여섯 필드 view, 기본 조회와 명시적 필터의 검색 경로를 설명합니다.

타입의 프로젝트 접근 권한은 `conn.GetVolumeTypeAccess`, `conn.AddVolumeTypeAccess`, `conn.RemoveVolumeTypeAccess`로 조회·변경합니다. [접근 권한 사용법과 Python 비교](blockstorage/volume-type-access.md)에 이름·ID 조회, 원본 access JSON, 프로젝트 ID 전달과 단계별 응답 증거를 설명합니다.

볼륨 수정과 bootable 설정은 `conn.UpdateVolume`·`conn.SetVolumeBootable`로 호출합니다. [사용법과 Python 비교](blockstorage/volume-mutations.md)에 변경된 속성만 보내는 옵션, 부분 응답 병합, 기본 true·명시적인 false와 조회·변경 응답의 구분을 설명합니다.

볼륨 limits는 `conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{})`로 조회합니다. 프로젝트를 지정하면 SDK가 Identity v3 조회를 먼저 완료하고 실제 ID를 query에 전달합니다. [현재/다른 프로젝트 limits와 Python 비교](blockstorage/volume-limits.md)에 기본값, 함수 옵션, nullable 모델, 단계별 응답 증거와 기존 singleton 스코프의 차이를 설명합니다.


Cinder backup은 `conn.ListVolumeBackups`·`SearchVolumeBackups`·`GetVolumeBackup`으로 조회합니다.
SDK가 상세 목록 기본값·서버/local 필터 분류·검색·nullable 24-field 변환·cached Cinder 선택과 실제
응답 증거를 처리합니다. [Backup 조회와 Python 비교](blockstorage/volume-backups.md)에 ordinary Boolean,
project/zone location, raw 응답과 seeded logical ID, nonnull 필터의 전체 검색을 설명합니다.


Backup 생성·삭제는 `conn.CreateVolumeBackup`·`conn.DeleteVolumeBackup`으로 호출합니다.
SDK가 six-field 생성 기본값, 정확한 이름/ID 조회, normal DELETE와 force 3.64 action, 부분 응답 병합과
대기를 처리합니다. [Backup 생성·삭제와 Python 비교](blockstorage/volume-backup-mutations.md)에
기본 생성 wait=true·삭제 wait=false, nil timeout, nullable 24-field logical 값과 실제 phase proof를 설명합니다.

Backup export는 `conn.ExportVolumeBackup(ctx, blockstorage.ExportVolumeBackupRequest{BackupID: backupID})`로
호출합니다. [Backup export와 Python 비교](blockstorage/volume-backup-export.md)에 Cinder v3의
opaque bytes·header·status, literal ID 요청과 오류 시 실제 응답 증거를 설명합니다.

Backup export의 JSON 값은 `conn.ExportVolumeBackupRecord(ctx, blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})`로 받습니다.
[Raw/JSON/native typed export 비교](blockstorage/volume-backup-export-record.md)에 cached Cinder의
`Backups.ExportRecord`·`ExportBackup`, owned JSON literal과 오류 시 실제 응답 증거를 설명합니다.

Backup 복원과 상태 변경은 `conn.RestoreVolumeBackup`·`conn.ResetVolumeBackupStatus`로 호출합니다.
[사용법과 Python/native 비교](blockstorage/volume-backup-actions.md)에 With 옵션, cached 필드 병합,
선택한 restore microversion과 reset의 3.64 정책, 오류 시 실제 응답 증거를 설명합니다.

Backup import는 `conn.ImportVolumeBackup(ctx, blockstorage.ImportVolumeBackupRequest{BackupService: service, BackupURL: locator})`로 호출합니다. [Python·package·service 사용법 비교](blockstorage/volume-backup-import.md)에 literal record, operation microversion 선택, disconnected `location:null` 모델과 오류 시 HTTP 증거를 설명합니다.

Cinder v3의 reserve·unreserve·begin/abort detaching은 `conn.ReserveVolume`·`UnreserveVolume`·`BeginVolumeDetaching`·`AbortVolumeDetaching`으로 명시적인 ID에 호출합니다. [Volume state action과 Python/native 비교](blockstorage/volume-actions.md)에 null action body, 호출별 microversion, opaque acknowledgement와 오류 시 단계별 응답을 설명합니다. 서버 상태 완료를 기다리지 않습니다.

명시적인 Cinder volume ID의 `conn.SetVolumeBootableStatus(ctx, request, bool)`은 필수 bootable 값을 전달하고, `conn.SetVolumeReadonly(ctx, request, options...)`는 생략 시 true와 명시적 false를 구별합니다. [Flag action 사용법과 Python 비교](blockstorage/volume-flags.md)에 package·Connection·v3 service 호출, 옵션 소유권, microversion과 opaque 응답 증거를 설명합니다.

명시적인 Cinder volume ID의 `ExtendVolume`·`RetypeVolume`·`CompleteVolumeExtend`는 필수 size/type 값, migration 기본 never·명시적 생략과 completion 기본 false를 SDK가 처리합니다. [Extend·retype·completion 사용법과 Python 비교](blockstorage/volume-resize-retype.md)에 세 호출 경로, 옵션 소유권과 opaque action acknowledgement를 설명합니다.

명시 ID의 `ResetVolumeStatus`·`MigrateVolume`·`CompleteVolumeMigration`은 상태 생략, nullable host/cluster, cluster의 required 3.16 검증과 완료의 기본 error:false를 SDK가 처리합니다. [Volume status·migration 사용법과 Python 비교](blockstorage/volume-migration-reset.md)를 참고하세요.

명시 ID의 `RevertVolumeToSnapshot`·`UnmanageVolume`은 required3.40 복원과 null 관리 해제를 SDK가 처리합니다. [Python/package/Connection/service 사용법](blockstorage/volume-revert-unmanage.md)을 참고하세요.

직접 Cinder `os-attach`·`os-detach`·`os-force_detach` action은 `AttachCinderVolume`·`DetachCinderVolume`이 owned options와 acknowledgement proof를 제공합니다. 서버 연결의 Nova+Cinder workflow인 기존 `AttachVolume`·`DetachVolume`과 호출 목적·인자가 다릅니다. [직접 Cinder 사용법과 Python 비교](blockstorage/cinder-volume-attachment.md)에 기본값·selector 우선순위·force connector 경계를 설명합니다.

Volume image metadata는 `SetVolumeImageMetadata`의 문자열·raw JSON 옵션과 `DeleteVolumeImageMetadata`의 키별 부분 결과로 처리합니다. 기본 전체 삭제는 현재 이미지 메타데이터를 조회하며, 명시적 빈 키 목록은 HTTP 없이 마칩니다. [Python·Connection·package·service 사용법](blockstorage/volume-image-metadata.md)을 참고하세요.

`UploadVolumeToImage`는 Cinder의 볼륨 이미지 내보내기에 필요한 기본값, 옵션 생략 여부, 조건부 3.1 지원 확인과 응답 처리를 제공합니다. [Python·Connection·package·service 사용법](blockstorage/volume-upload-image.md)을 참고하세요.

[독립 Floating IP 목록·검색·조회](compute/floating-ip-queries.md)는 Connection과 Compute Service의 IP 조회4개·pool 조회2개를 제공합니다. Neutron dictionary와 로컬 JMESPath 검색, 함수별404 처리, 논리 Resource와 실제 Wire·페이지 증거를 구분합니다. Python 비교와 concrete 옵션 예제를 함께 설명합니다.

[독립 Floating IP 삭제](compute/floating-ip-delete.md)는 기본 DELETE·공개 조회 검증과 한 번의 추가 시도를 제공합니다. Neutron DELETE404의 false, DOWN 성공 정책과 실제 부재, 접수 뒤 오류·시도별 응답을 구분하고 concrete 옵션을 사용합니다.

[독립 Floating IP 생성](compute/floating-ip-create.md)은 가용 IP를 재사용하지 않고 새 allocation을 생성합니다. SDK가 Neutron/Nova·port 우선·optional server/NAT·공개 Get 대기·wait timeout 정리를 처리하며, 접수된 응답과 compatibility/wait/cleanup 부분 결과를 보존합니다.

[미연결 Floating IP 일괄 정리](compute/floating-ip-unattached-delete.md)는 Neutron 전체 목록을 확보한 뒤 port가 비어 있는 항목을 순차 삭제합니다. 개별 false는 계속 처리하고 오류는 중단하며 SDK 소유 옵션·한 deadline·항목별 부분 결과를 제공합니다.
