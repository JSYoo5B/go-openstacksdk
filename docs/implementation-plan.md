# 구현 순서와 단계별 현황

2026-10-07에 조정한 작업 순서입니다. **핵심 서비스 user API → 핵심 서비스 admin API → 매니지드 서비스 user API → 매니지드 서비스 admin API** 순으로 진행합니다. 목표는 고정한 Gophercloud와 openstacksdk의 전체 API·리소스·복합 작업을 일관된 Go SDK로 제공하는 것입니다. 관리형 서비스의 구현도 전체 목표에 포함하며, 기존 구현과 지원 판정은 보존합니다.

이 문서는 **앞으로 할 작업의 순서와 진행 단계**, [SDK 지원 판정대장](sdk-support-ledger.md)은 **검증한 계약과 실행 근거**, [판정 JSON](../api/sdk_reviews.json)은 **연산별 지원 여부**를 관리합니다. 세 자료의 상태는 서로 다른 의미를 갖습니다.

[단계별 사용·릴리즈 기준](release-milestones.md)은 `make smoke`로 확인하는 핵심 user 개발 preview와 외부 설치용 alpha의 완료 조건을 관리합니다. 현재 5개 흐름·9개 기존 테스트 그룹이 로컬 HTTP race 검증을 통과했고, 공개 모듈 경로와 외부 소비자 설치 검증은 다음 배포 준비 항목입니다.

## 현재 집계와 진행 중인 작업

집계는 [판정 JSON](../api/sdk_reviews.json)과 고정 catalog에서 `make progress`로 생성합니다. `make check`는 집계가 판정 JSON과 다르면 실패합니다. **API 완료 수**와 **진행 중인 구현 단계**를 함께 확인할 수 있도록 아래에 현재 작업을 기록합니다.

**현재 작업 (2026-10-08): 핵심 user Barbican 삭제3개·목록1개·고정 getter2개 완료.** 기존 기반의 실제 named 계약을 닫아 **전체179→185(+6)·핵심105→111·Barbican5→11/67**로 반영했습니다. 삭제와 목록의 누락은 공유 engine과 기존 fixture로 보완했고, 이미 구현한 getter2개는 기존 HTTP·공유 Connection·디코더 증거를 재사용해 완료 심사했습니다.

삭제는 `5e7cb7e`·`96b76d4`·`e336575`·`97e205e`, 목록은 `026967a`·`ee67234`·`dfcba00`·`e9670e5`·`a1218bb`, 고정 getter2개 완료 판정은 `c3cf874`로 작은 의미 단위로 커밋하고 push했습니다. 최종 source의 집중4 package race·전체40 package check·문서 main build·기존 preview5흐름/9그룹이 PASS했습니다. 조회 예제의 기존2개 Go 함수도 그대로 추출해 컴파일했습니다. 판정·prose만 바뀐 getter 단계에서는 같은 Go 전체 테스트를 반복하지 않고 final parity/집계만 확인합니다. Source pin·기존 계약/anchors를 유지하며 generic Resource/session과 Go typed/seeded ID·strict status의 차이를 명시합니다.

**진행 중: Barbican Container/Order/Secret 사용자 생성3개 최종 검증.** 소스 대조에서 ref-only POST 응답 뒤 입력 속성이 결과에서 사라지는 누락과 accepted 응답 원문·header·status/error 증거 부재를 확인했습니다. Python은 입력으로 만든 Resource에 응답 속성을 병합하며 추가 GET/payload GET을 하지 않습니다. 기존 native Create는 호환성을 유지하고, 기존 guarded REST·metadata projection·옵션 snapshot·HTTP fixture를 재사용하는 공통 엔진과 세 leaf의 동일한 `CreateRecord` 옵션을 `1eae268`로, Python 비교 가이드·문서 main·재생성 hook을 `9c14ed4`로 커밋·push했습니다. 기존 shared decoder·source guard·옵션 JSON snapshot·leaf/cached Connection·body fault wrapper를 재사용하며, 새 harness 없이 서비스별 binding과 공통 오류를 나누어 검증합니다. 새 HTTP6그룹과 기존 계약을 포함한 집중3 package race 및 문서 main build가 PASS했습니다. 공통 package는 컴파일 확인이며 자체 테스트가 있다고 세지 않습니다. 전체 gate와 최종 named 판정은 남아 있으므로 아직 완료 수에 추가하지 않습니다. 공통 Go context cause 일관성도 남은 SDK 기반 과제로 추적합니다. 이 후보를 완료 수에 미리 더하지 않습니다. 핵심 user→핵심 admin→후속 user→후속 admin 순서와 외부 설치용 alpha 준비는 유지합니다.

<!-- sdk-progress:start -->
| 지표 | 현재 값 | 해석 |
|---|---:|---|
| 고정 소스 전체 선언 | 3,362 | Gophercloud·openstacksdk 선언; inherited/descriptor/Resource 표면은 별도 추적 |
| 검증된 Go 매핑 | 185 (5.5%) | 전체 연산의 `go_mapping` 판정. 부분 계약 추가만으로 이 수를 늘리지 않음 |
| source 그대로 지원 | 0 | `supported` 판정 |
| 미해결 / 미지원 | 3,176 / 1 | 미검토 선언도 미해결 집계에 포함 |
| 연산별 검토 기록 | 510 | 아직 개별 기록 없는 선언 2,852 |
| 기록한 부분·전체 계약 | 3,221 | [판정 JSON](../api/sdk_reviews.json)의 계약 항목 수; 테스트 함수 수나 전체 API 완료 수와 다름 |
<!-- sdk-progress:end -->

최근 완료 수 변화는 다음과 같습니다. 아래 수치는 해당 커밋 시점의 이력입니다.

| 완료 단위 | 완료 수 변화 | 구현·검증 근거 | 원격 반영 |
|---|---:|---|---|
| Floating IP 조회 6개·삭제 1개·생성 1개 최종 판정 | 161 → 169 (+8) | 신규 생성 24그룹·조회 회귀 3그룹, 전체 40 package check, 실행 예제 컴파일 | `73bcfba` |
| 미연결 Floating IP 일괄 정리 | 169 → 170 (+1) | 신규 HTTP 12그룹, 집중 race 169그룹·전체 40 package check, 실행 예제 컴파일 | `9f1cb3d` |
| Secret getter·Glance schema getter 4개 | 170 → 175 (+5) | 기존 25개 계약·실제 HTTP/raw/옵션 테스트 재검토, 관련 4 package 집중 race·전체 40 package check PASS | `4d89708` push 완료 |
| AvailableFloatingIP named getter | 175 → 176 (+1) | Query/Create/Allocate 재사용·Neutron 내부/Nova fallback, 집중 race·전체40 package check·smoke5흐름 PASS | `c61acae` push 완료 |
| Barbican effective quota getter | 176 → 177 (+1) | 기존 strict/error fixture·context guard 재사용, 집중3 package race·전체40 package check·조회 main build PASS | `1546e7d` push 완료 |
| Barbican Container/Order metadata getter | 177 → 179 (+2) | 공유 Fetch·기존 fixture6그룹, 집중3 package race·전체40 package check·main build·smoke5흐름 PASS | `156b907` push 완료 |
| Barbican Container/Order/Secret 삭제 | 179 → 182 (+3) | 공유 Delete·기존 fixture6그룹, 집중4 package race·전체40 package vet/race·나머지 gate·main build·smoke5흐름 PASS | `97e205e` push 완료 |
| Barbican SecretStore 목록 semantic 필터 | 182 → 183 (+1) | 공통 classifier/matcher/pagination·기존 fixture의4 HTTP그룹·cached Connection, 집중4 package race·전체40 package check·main build·smoke5흐름 PASS | `a1218bb` push 완료 |
| Barbican SecretStore fixed getter2개 | 183 → 185 (+2) | 기존3계약씩·HTTP/Connection·shared decoder 재사용, 같은 최종40 package gate·보존2 Go 함수 build·ID mapping 비교 | `c3cf874` push 완료 |

완료 수가 그대로인 동안에도 구현·테스트·문서 단계는 갱신합니다. 부분 계약·테스트 수를 API 완료 수에 더하지 않습니다.

## 서비스 우선순위별 API 집계

`make progress`가 아래 표도 함께 생성합니다. 완료는 `supported`와 `go_mapping`, 부분·미해결 검토는 개별 review가 있지만 `unresolved`인 선언입니다. 미검토는 개별 review가 없는 선언입니다. **서비스별 수는 user/admin 합산이며 1단계 완료율로 해석하지 않습니다.**

고정 Gophercloud/Python 직접 선언과 Cloud 복합 작업을 소스의 주관 서비스에 한 번씩 포함합니다. `_network_common`의 Nova·Neutron 공통 IP 작업은 Network 그룹에서 한 번만 세며, `_coe`는 Container Infra입니다. 실행 시 다른 서비스도 호출하는 작업을 중복 집계하지 않습니다. 고정 소스의 `FileSegment` 4개(`cloud/_utils`)는 Swift, server inventory 3개는 Nova, Identity 버전을 고르는 native `utils.ChooseVersion`은 Keystone에 배정합니다. Python `connection`·`openstackcloud`와 나머지 native common/version utilities는 서비스 공통 기반으로 분리합니다. 상속·descriptor·Resource 전체 표면은 이 분모에 포함되지 않습니다.

<!-- sdk-service-progress:start -->
| 우선순위 묶음 | 완료 / 전체 | 완료 판정률 | 부분·미해결 검토 | 미검토 | 미지원 |
|---|---:|---:|---:|---:|---:|
| 핵심 서비스 · 1·2단계 합산 | 111 / 2,292 | 4.8% | 225 | 1,955 | 1 |
| 후속 네트워크 · 3·4단계 합산 | 5 / 254 | 2.0% | 13 | 236 | 0 |
| 후속 베어메탈 · 3·4단계 합산 | 1 / 207 | 0.5% | 1 | 205 | 0 |
| 후속 나머지 · 3·4단계 합산 | 68 / 580 | 11.7% | 85 | 427 | 0 |
| 서비스 공통 기반 | 0 / 29 | 0.0% | 0 | 29 | 0 |
| 전체 | 185 / 3,362 | 5.5% | 324 | 2,852 | 1 |

**핵심 서비스**

| 서비스 | 완료 / 전체 | 부분·미해결 검토 | 미검토 | 미지원 |
|---|---:|---:|---:|---:|
| Identity / Keystone | 2 / 389 | 7 | 380 | 0 |
| Compute / Nova | 6 / 333 | 23 | 304 | 0 |
| Placement | 0 / 71 | 0 | 71 | 0 |
| Network / Neutron | 23 / 758 | 52 | 683 | 0 |
| Image / Glance | 4 / 120 | 75 | 41 | 0 |
| Block Storage / Cinder | 65 / 480 | 29 | 386 | 0 |
| Key Manager / Barbican | 11 / 67 | 10 | 46 | 0 |
| Object Storage / Swift | 0 / 74 | 29 | 44 | 1 |

**후속 서비스 · 네트워크 → 베어메탈 → 나머지**

| 서비스 | 완료 / 전체 | 부분·미해결 검토 | 미검토 | 미지원 |
|---|---:|---:|---:|---:|
| Load Balancer / Octavia | 3 / 144 | 7 | 134 | 0 |
| DNS / Designate | 2 / 110 | 6 | 102 | 0 |
| Bare Metal / Ironic | 0 / 189 | 0 | 189 | 0 |
| Bare Metal Introspection | 1 / 18 | 1 | 16 | 0 |
| Database / Trove | 1 / 55 | 2 | 52 | 0 |
| Container Infra / Magnum | 1 / 56 | 0 | 55 | 0 |
| Container / Zun | 0 / 4 | 0 | 4 | 0 |
| Clustering / Senlin | 52 / 69 | 17 | 0 | 0 |
| Orchestration / Heat | 6 / 66 | 23 | 37 | 0 |
| Messaging / Zaqar | 0 / 38 | 4 | 34 | 0 |
| Workflow / Mistral | 0 / 30 | 0 | 30 | 0 |
| Shared File System / Manila | 0 / 196 | 6 | 190 | 0 |
| Instance HA / Masakari | 8 / 17 | 9 | 0 | 0 |
| Metric | 0 / 9 | 0 | 9 | 0 |
| Reservation / Blazar | 0 / 5 | 0 | 5 | 0 |
| Accelerator / Cyborg | 0 / 35 | 24 | 11 | 0 |
<!-- sdk-service-progress:end -->

단계별 진행은 현재 다음과 같습니다. 판정 JSON에 권한·user/admin 분류가 없어 각 단계의 완료/전체 수는 **아직 미분류**입니다. 같은 서비스 함수를 user/admin 양쪽에 중복 집계하거나 대기 중인 단계의 기존 완료 기능을 0으로 표시하지 않습니다.

| 구현 순서 | 현재 상태 | 단계별 API 개수 |
|---|---|---|
| 1. 핵심 서비스 user API | 진행 중 · 현재 AvailableFloatingIP의 남은 선택 순서 보완 | user/admin 분리 집계 대기 |
| 2. 핵심 서비스 admin API | 추가 작업 대기 · 기존 구현·판정 유지 | user/admin 분리 집계 대기 |
| 3. 매니지드·후속 서비스 user API | 추가 작업 대기 · 네트워크 → 베어메탈 → 나머지 | user/admin 분리 집계 대기 |
| 4. 매니지드·후속 서비스 admin API | 추가 작업 대기 · 같은 내부 순서 | user/admin 분리 집계 대기 |

후속 서비스의 기존 완료 수에는 우선순위 조정 전에 구현한 Senlin·Masakari 등의 기능이 포함됩니다. 앞으로의 작업 순서를 뜻하는 단계와 누적 완료 판정을 구분합니다. 0개 완료인 서비스에도 API 파사드나 부분 구현이 있을 수 있으며, 전체 연산의 필수 계약을 닫기 전에는 완료로 세지 않습니다.

## 서비스 우선순위

공통 기반은 해당 작업에 필요한 범위부터 보완합니다. 앞 단계의 미해결 계약을 우선 닫고 다음 단계로 이동합니다. 단계 안에서는 기본 사용 흐름과 그 흐름을 막는 계약을 먼저 처리합니다.

| 순서 | API 범위 | 먼저 완성할 동작 |
|---|---|---|
| 1 | **핵심 서비스 user API** | 인증·프로젝트 선택, 이름/ID 조회, 사용자가 소유하거나 사용할 수 있는 리소스의 기본 CRUD·목록·페이지네이션·대기, 서버 부팅·네트워크·볼륨·이미지·키 관리·객체 저장 흐름 |
| 2 | **핵심 서비스 admin API** | 운영자 권한을 요구하는 전역 조회·관리, quota·서비스/host·Placement 관리, Cinder `ManageVolume` 등 관리자 action과 관련 기본값·확장·오류 계약 |
| 3 | **매니지드 서비스 user API** | LBaaS·DBaaS·컨테이너·클러스터 등에서 사용자가 요청·조회·수정·삭제하는 리소스와 상위 작업, 부모 범위·비동기 완료·부분 실패 |
| 4 | **매니지드 서비스 admin API** | 해당 서비스의 운영자용 전역 관리·quota·서비스/host·관리자 action과 관련 계약 |

핵심 서비스 묶음은 **Identity(Keystone), Compute(Nova), Placement, Network(Neutron), Image(Glance), Block Storage(Cinder), Key Manager(Barbican), Object Storage(Swift)**입니다.

후속 서비스 묶음은 **Load Balancer(Octavia), DNS(Designate), Bare Metal(Ironic), Bare Metal Introspection, Database(Trove), Container Infra(Magnum), Container(Zun), Clustering(Senlin), Orchestration(Heat), Messaging(Zaqar), Workflow(Mistral), Shared File System(Manila), Instance HA(Masakari), Metric, Reservation(Blazar), Accelerator(Cyborg)** 순서입니다. 이 중 인프라 확장·특화 서비스도 핵심 서비스 뒤에서 같은 user → admin 순서로 진행합니다. 모든 후속 서비스를 배포 형태까지 관리형이라고 분류하는 것은 아닙니다.

**후속 user API 단계와 후속 admin API 단계 각각에서 네트워크 관련 서비스(Octavia·Designate) → 베어메탈(Ironic·Introspection) → 나머지 후속 서비스 순으로 진행합니다.** 따라서 후속 서비스의 user API를 처리한 뒤 같은 내부 순서로 admin API를 처리합니다. Neutron은 핵심 서비스 묶음에서 먼저 진행합니다. Barbican을 핵심으로 올리고 Swift도 핵심에 유지합니다.

Swift는 일반 Glance 업로드/import의 필수 의존은 아니지만, 고정 openstacksdk의 [Swift 경유 이미지 Task 업로드](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L835-L913)에 필요합니다. `image_api_use_tasks` 분기는 Swift container/object 생성 → Glance import Task → 선택적 대기·이미지 갱신·객체 정리를 연결하며, 이 흐름은 [남은 구현 범위](../image/README.md#전체-api와-남은-작업)에 포함됩니다. 해당 Task 경로의 권한 분류는 고정 소스의 admin-only 설명과 서버 정책을 검토해 핵심 admin 단계에 배치합니다.

현재 Connection의 24개 서비스를 모두 배치했으며, 고정 소스에만 있는 추가 버전·상속·descriptor·Resource 표면도 각 서비스와 권한 범위에 따라 조사합니다.

user/admin은 SDK 함수 이름이나 CRUD 여부만으로 판단하지 않습니다. 연산의 사용 목적과 확인한 API 권한 정책을 근거로 구분하고, 정책·배포에 따라 달라지는 권한은 차이를 기록합니다. 하나의 함수가 일반 호출과 `all_projects`·force 등 관리자 옵션을 함께 제공하면 **계약 단위로 두 단계에 나눕니다.** 공통 구현을 먼저 완료해도 관리자 분기의 검증이 남으면 해당 전체 연산의 지원 판정은 미완료로 유지합니다. 권한이 미확정인 계약은 분류 근거를 조사하고 미해결로 남깁니다.

핵심 user API는 다음 사용 흐름을 기준으로 진행합니다. 이미 검증한 부분은 재구현하지 않고 연결된 판정의 `remaining`을 확인합니다.

1. **인증과 기본 조회:** Identity의 인증·접근 가능한 프로젝트 선택, 공유 토큰과 endpoint 선택, Nova·Neutron·Glance의 이름/ID·목록·기본값 계약을 확인합니다.
2. **이미지 기반 서버와 네트워크:** 이미지·flavor·network·port 해석, 서버 생성·대기·삭제, floating IP 선택·재사용·자동 연결과 각 단계의 부분 실패를 닫습니다.
3. **볼륨과 부팅:** Cinder의 기본 생명주기, 서버 연결·분리, 볼륨·snapshot 기반 부팅과 이미지 연계에서 남은 계약을 닫습니다.
4. **키 관리:** Barbican의 secret/container 생성·조회·목록·수정·삭제, payload와 접근 권한·옵션의 계약을 닫습니다. 각 리소스가 실제로 제공하는 연산을 소스와 권한 정책으로 확인합니다.
5. **객체 저장:** Swift의 container/object 생성·목록·업로드·다운로드·삭제, metadata·스트림 소유권·무결성 계약을 닫고 Glance 연계의 공통 기반을 준비합니다. Glance Task 연계는 권한 범위에 맞춰 핵심 admin 단계에서 처리합니다.
6. **핵심 user API의 나머지 범위:** 사용자가 조회 가능한 quota·limits, 확장 API·이전 버전과 공통 Resource 동작을 끝까지 추적합니다. 프로젝트/사용자/role의 관리자 작업, quota 변경·Placement 관리·관리자 action은 2단계에서 처리합니다.

각 흐름은 작은 API/계약 단위로 구현·검증·커밋하고 다음 핵심 흐름으로 이어갑니다. 앞선 작업을 막는 의존 기능이나 기존 공개 API의 회귀가 있으면 필요한 부분을 먼저 처리하고 그 이유와 범위를 이 문서에 기록합니다. 의존 기능 처리는 후속 단계 전체를 앞당기는 것으로 해석하지 않습니다.

다음 후보였던 Cinder `ManageVolume`은 **2단계인 핵심 서비스 admin API의 후보**로 이동하고 착수 시 권한 정책 근거를 확인합니다. 이미 끝낸 순수 Volume 모델 변환 공통화는 유지합니다. Senlin·Masakari 등에서 이미 검증한 기능도 유지하며 후속 미해결 계약의 순서만 뒤로 이동합니다.

## 구현 단위와 검증 반복의 조정

완료 API를 닫는 속도를 높이기 위해 다음 원칙을 적용합니다. 검증 수준은 유지하고, 같은 기반의 비교·검증·기록을 반복하는 작업을 줄입니다.

- 착수 시 해당 선언의 입력·기본값·확장·반환·오류·페이지·대기 등 실제 필수 계약을 유한 목록으로 고정합니다. 다른 선언의 mutable Resource/session 전체를 해당 API의 포괄적인 미완료 사유로 추가하지 않습니다. Go의 명시적 타입·옵션과 Python의 동적 입력 차이는 문서화하고, 실제 필요한 동작 누락은 구현합니다.
- 이미 검증한 공통 엔진을 재사용합니다. 선언별 근거는 공통 계약과 실제 binding 검증을 연결하고, 새로운 입력·요청 경로·결과·실패 분기에 필요한 집중 테스트를 추가합니다.
- 변경에 맞는 집중 검증이 통과하면 구현을 작은 의미 있는 커밋으로 저장합니다. 전체 검증은 변경 범위와 공통 기반의 영향에 맞춰 묶어서 수행하고, 같은 소스의 전체 검증을 판정·문구 갱신마다 반복하지 않습니다. 새 실패나 미해결 위험이 있으면 추가 검증합니다.
- 기존 부분 판정도 실제 named 계약을 다시 확인합니다. 구현·테스트·문서가 갖춰졌고 필수 누락이 없으면 최종 판정까지 닫고, 실제 누락이 있으면 그 항목만 보완합니다. 함수 생성이나 숫자 증가만으로 완료 처리하지 않습니다.
- 진행 숫자는 자동 집계하고, 계획과 대장은 현재 단계·검증 근거를 단위 종료 시 함께 갱신합니다. 소스 검토·구현·검증 등의 단계 변화는 대화에도 알리며, 문서/집계 작업 때문에 다음 구현을 오래 지연하지 않습니다.

## 작업 단위의 진행 단계

지원 판정과 별도로 각 작업의 소스 검토·구현·테스트·문서·최종 검토를 구분합니다. 부분 구현은 테스트와 문서가 있어도 전체 연산의 `remaining`이 남으면 `unresolved`입니다.

| 단계 | 완료 기준과 남길 근거 |
|---|---|
| 소스 검토 | 핵심/후속 서비스와 user/admin 단계 및 권한 분류 근거를 기록; 고정 source pin의 입력·생략/null/false/zero·기본값·결과·오류·버전·상속 동작을 비교하고 대상 연산 ID와 남은 계약을 기록 |
| 구현 | 공개 Go API와 SDK 소유 concrete 옵션·기본값·모델·공통 정책을 구현하고 의미 있는 작은 커밋으로 보존 |
| 테스트 | 실제 요청 경로·body/query/header·응답·실패·취소·부분 성공을 확인하는 계약 테스트와 변경에 맞는 집중/회귀 검증을 실행하고 명령·결과·검증한 revision을 기록 |
| 문서 | 서비스 사용법과 Python/Go 비교, 기본값·차이·부분 결과를 설명하고 변경한 실행 예제의 빌드 및 링크를 확인 |
| 최종 검토·지원 판정 | 고정 소스·실제 assertion·문서·실행 결과를 함께 검토하고 `sdk_reviews.json`의 API·계약별 테스트·문서·차이·남은 기능을 갱신; `paritycheck`로 정합성 확인 |
| 공유 | 구현·테스트·문서·판정을 각각 작은 의미 있는 커밋으로 저장하고 직접 push; 단위 종료 시 이 표와 지원대장의 검증 근거 갱신 |

작업 착수, 구현·집중 검증 완료, 커밋·push, 문서·최종 검토 완료처럼 단계가 바뀔 때 대화에도 진행 상황을 알립니다. 업데이트에는 완료한 기능, 실제 검증 결과, 커밋·push 상태와 다음 작업을 포함합니다. 실행 중인 작업은 60초 이상 안내 없이 두지 않으며, 전체 연산의 남은 계약을 부분 완료와 구분합니다.

중간 업데이트는 다음 네 항목을 기준으로 짧게 작성합니다. 단계 전환이 없는 동안에도 새로 확인한 계약이나 실패 원인, 다음 검증에서 확인할 내용을 알립니다.

- **현재 작업·단계:** 서비스와 대상 기능, 소스 검토/구현/테스트/문서/최종 검토 중 현재 단계.
- **완료·검증 결과:** 실제로 구현한 범위와 실행한 검증 결과. 실행 중인 테스트와 아직 실행하지 않은 검증은 구분.
- **커밋·push 상태:** 새 커밋과 원격 반영 여부. 의미 있는 변경이 쌓이면 작업 단위 전체가 끝나기 전에도 커밋·push.
- **남은 작업·다음 행동:** 현재 단위의 미완료 계약과 다음 검증 또는 구현. API 개수를 보고할 때는 전체 연산 판정과 부분 계약 완료를 구분하고, 변화가 있으면 판정 JSON에서 다시 집계.

단계는 일부 겹칠 수 있습니다. 예를 들어 계약 테스트와 문서는 구현과 함께 작성하고, 커밋·push도 단위가 완전히 끝날 때까지 미루지 않습니다. 코드가 바뀌면 영향받는 단계의 검증을 다시 확인하며, 이전 revision의 성공을 변경 후 성공으로 옮겨 적지 않습니다.

## 현재 작업 현황

아래는 2026-10-08의 최근 완료 단위와 다음 작업입니다. Create·조회6개·삭제1개는 해당 선언의 필수 계약·실제 테스트·문서·검증을 완료해 `go_mapping`으로 판정했습니다. List/Search/Get에서 발견한 실제 오류2개는 수정과 신규 HTTP 회귀3그룹·최종 전체 check로 닫았습니다. 전체 mutable Resource/session 등 다른 선언의 목표는 계속 추적합니다. 착수와 단계 전환 때 이 표·집계·검증 기록을 함께 갱신합니다.

| 작업 단위 | 소스 검토 | 구현 | 테스트 | 문서 | 최종 검토·판정 | 커밋·push / 다음 행동 |
|---|---|---|---|---|---|---|
| Cinder `UploadVolumeToImage` | 완료 | 완료 | 집중 14그룹·전체 race·vet 완료 | Python 비교·3개 호출 경로·예제 컴파일 완료 | 해당 Python Proxy 1개 연산 `go_mapping`; native/v2/Resource 등은 별도 | `f9892f5`까지 push 완료. [검증 기록](sdk-support-ledger.md#cinder-v3-volume-image-export), [사용법](../blockstorage/volume-upload-image.md) |
| Volume 순수 모델 변환 공통화 | 기존 변환 계약 비교 완료 | 완료 | 기존 계약 회귀·전체 race·vet 완료 | 내부 변경을 지원대장에 기록 | 의미 보존 검토 완료, API 지원 승격 없음 | `a35f92f` push 완료. [검증 기록](sdk-support-ledger.md#volume-모델-변환의-공통-내부-계층) |
| Identity 사용자별 프로젝트·그룹 목록 | native page와 Python 모델·상속 비교 완료 | 반환 모델·추출기 교정 완료 | 집중 race·전체 check 완료 | Python 비교·독립 Go 예제 컴파일 완료 | native 2개 `go_mapping`, Python 2개 부분 판정·정합성 검증 완료 | 코드·테스트 `ff685e9` push 완료. 문서·판정 `4ba94ab` push 완료. [검증 기록](sdk-support-ledger.md#identity-v3-사용자별-프로젝트그룹-목록), [사용법](../identity/v3/users/memberships.md). NIC 선택으로 연결 완료 |
| Nova 서버 NIC 선택 | cloud·Proxy·native와 capability 정책 비교 완료 | concrete NIC·mode·port 이름 해석·default auto 완료 | 11개 신규 HTTP 그룹·전체 check 완료 | Python 비교·독립 Go 예제 컴파일 완료 | create 관련 3개 부분 판정, 전체 연산 unresolved | 코드·테스트 `55927d3` push 완료. [사용법](../compute/server-network-interfaces.md), [검증 기록](sdk-support-ledger.md#nova-서버-생성의-nic-선택). 문서·판정 `16d51cc` push 완료 |
| Nova 서버 생성의 기본 네트워크 | config/cloud/native의 선택·상속·오류·cache 비교 완료 | Connection concrete 옵션·파일 snapshot·YAML 전체 페이지 name/ID 선택 완료 | 신규 12그룹 집중 race·전체 check 완료 | Python 비교·독립 Go 예제 컴파일 완료 | create 3개 부분 판정 갱신·default getter 1개 부분 판정, 전체 연산 unresolved | 코드·테스트 `0355600`·`afb3023` push 완료. [사용법](../compute/server-default-network.md), [검증 기록](sdk-support-ledger.md#nova-서버-생성의-기본-네트워크). 문서·판정 `a61135f` push 완료 |
| Neutron floating IP 재사용·서버 연결 | 고정 available/create/attach·NAT·cache·cleanup 비교 완료 | Ensure·recorded owner·revision·자동 external/router·빈 페이지 선택 완료 | 신규 9그룹 집중 race·전체 check 완료 | Python 비교·독립 Go 예제 컴파일 완료 | 부분 계약 검토 완료, 전체 cloud 연산 unresolved | 코드·테스트 `51223e7`·문서·판정 `15bb5cc` push 완료. [사용법](../network/floating-ip-ensure.md), [검증 기록](sdk-support-ledger.md#neutron-floating-ip-재사용과-서버-연결) |
| 서버 생성과 floating IPv4 연결 | 고정 create/wait/available·remaining timeout·주소 수렴 차이 비교 완료 | CreateWithFloatingIP·생성 전 서비스/owner 준비·필수 서버/IP ACTIVE·단일 deadline·부분 결과 완료 | 신규 8그룹 집중 race·전체 check 완료 | Python 비교·정확한 독립 Go 예제 컴파일 완료 | cloud create/available 갱신·cloud wait 부분 판정 추가, 전체 연산 unresolved | 준비 공통화 `bc36b0a`·구현 `14e6d88`·경계 테스트 `50b155d`·문서/판정 `8af4c7b` push 완료. [사용법](../compute/create-with-floating-ip.md), [검증 기록](sdk-support-ledger.md#서버-생성과-floating-ip-연결) |
| 공유 네트워크 역할 getter·설정·cache | 고정 family·NAT/default·use flags·loader·cache 차이 비교 완료 | 10 getter·2 flag·owned snapshot·성공 cache·Reset·설정 상속/교체 완료 | 신규 9그룹·기존 설정 snapshot assertion·집중 race·전체 check 완료 | Python 비교·정확한 독립 Go 예제 컴파일 완료 | 12개 공개 선언의 부분 계약·정합성 검증 완료, 전체 연산 unresolved | Network `0fc1be8`·Connection `39e376c`·경계 `4551ea7`·문서/판정 `833d1f1` push 완료. [사용법](../network/network-roles.md), [검증 기록](sdk-support-ledger.md#공유-네트워크-역할-조회와-설정) |
| 공유 역할의 기본 NIC·floating source/NAT 소비 | 고정 default·floating 후보·다중 port NAT·캐시/오류 순서 비교 완료 | configured default·zero external·조건부 NAT가 공유 snapshot 소비 완료 | 신규 10그룹·집중 race·전체 check 완료 | Python 비교·새 독립 workflow 예제 컴파일·기존 4개 main 보존 확인 완료 | 기존 17개 부분 판정 갱신·정합성 검증 완료, 지원 승격 없음 | 기본 NIC `09ddb13`·source/NAT `db79187`·경계 테스트 `9b3bb54` push 완료. [검증 기록](sdk-support-ledger.md#공유-역할의-기본-nicfloating-sourcenat-소비), [사용법](../network/network-roles.md#서버-생성에-같은-설정-사용하기). 문서·판정은 이번 갱신에 포함 |
| 상위 Network CRUD와 공유 cache hook | 고정 cloud·Proxy·Resource의 생성/수정/삭제·AZ·기본값·cache 차이 비교 완료 | concrete 옵션·owned lookup/응답·접수 후 원래 cache Reset 완료 | 기본 7그룹·경계 12그룹·Connection 1그룹 집중 race 완료; 공통 삭제/조회·HTTP guard·204 7그룹 추가, 전체 40 package check 완료 | Python/Go 독립 예제·기본값/오류·native 모델 한계 작성, 독립 main 컴파일 완료 | cloud 3행 부분 판정 추가·기존 12행 갱신, 전체 gate·예제 컴파일·정합성 검증 완료, 지원 승격 없음 | 공통 오류 `fd863f3`·guard `2d95f3e`·CRUD `69aa03d`·조회 `8d0477d`·경계 `f71e838`·204 `5bfea18` 커밋·push 완료. [사용법](../network/network-mutations.md). 문서 `a1dc85c` push 완료. 판정·단계 기록은 이번 분리 커밋에 포함 |
| 서버 주소 선택·기존 IP 보충 | 고정 raw/cloud/meta·역할/MAC·IPv6·probe·source/설정 차이 검토 완료 | owned view·public/private/default·Neutron/Nova 보충·Connection lazy/cache·concrete policy 완료 | 신규 23그룹 집중 race·전체 40 package check 완료 | Python/Go 비교·정확한 독립 main 컴파일·상대 파일 링크 확인 완료 | cloud getter 2행 부분 판정 추가; 전체 Get/Create/Wait unresolved 유지 | `0cc3815`·`864b0e6`·`bbd39c6`·`5e889a6`·`8e982e4` 커밋·push 완료. [사용법](../compute/server-addresses.md), [검증 기록](sdk-support-ledger.md#서버-주소-선택과-기존-floating-ip-보충). 문서 `726ea3b` push 완료; 판정·단계 기록은 분리 커밋으로 보존 |
| Floating IP 선택 계획과 실행 | 고정 needs/available/NAT·owner/응답·재선택 차이 검토 완료 | owner-free PrepareEnsure·owned plan·동일 tuple 실행·port 재GET·guarded REST·부분 201/202 결과 완료 | 신규 plan 13그룹·공유 헤더 1그룹·집중 4 package race·전체 40 package check 완료 | Python/Go 비교·정확한 main 최종 코드 컴파일·상대 링크 확인 완료 | 기존 available/native Create 2행 부분 근거 갱신·상태/핀 유지, 자동 cloud 연산 unresolved | 공통 헤더 `0258a1a`·계획 `c2cf46b`·경계 `37692fc`·헤더 범위/PUT `9ea3e22` push 완료. [사용법](../network/floating-ip-plan.md)·문서 `f77250d` push 완료. [검증 기록](sdk-support-ledger.md#floating-ip-선택-계획과-실행)과 판정은 분리 커밋으로 보존 |
| 자동 IP 필요성·조건부 연결·Nova 수렴 | 고정 needs/skip/source·available/attach·raw Nova wait와 부분 실패 검토 완료 | guarded planner·lazy decision·조건부 Neutron assignment·raw Nova target 관측·부분 결과 완료 | 신규19그룹 집중3 package race·전체40 package check 완료 | Python/Go 비교·정확한 독립 main 컴파일·상대 파일 링크245개 확인 완료 | add_ips 부분 판정1행·기존 available/getter3행 갱신; 전체 연산 unresolved·지원 승격 없음 | planner `532a1be`·context `744092f`·구현 `64cb181`·경계 `a048624`·BUILD `ba78cd4`·문서 `488a84a` push 완료. [사용법](../compute/server-automatic-ip.md), [검증 기록](sdk-support-ledger.md#기존-서버의-자동-floating-ip-판단연결nova-관측). 당시 미완료한 상위 소비자 통합은 아래 후속 단위에 기록; Nova/full source 계약은 계속 추적 |
| 자동 IP와 서버 생성·ACTIVE 대기의 통합 | 고정 create/get_active/wait·metadata·fault/삭제·budget 비교 완료 | owned 생성·raw ACTIVE/주소 준비·조건부 IP·known Creation/Server/Assignment·전체 deadline 완료 | 신규16그룹·집중13그룹4 package race·전체40 package check 완료 | Python/Go 비교·정확한 main 컴파일·상대 파일 링크 확인 완료 | create/wait2행 갱신·get_active 부분1행 추가, 전체 연산 unresolved·지원 승격 없음 | 대기 정책 `745f50d`·의존 조회 `3249f71`·통합 `f3ed74a`·경계 `afee318` push 완료. [사용법](../compute/create-with-automatic-floating-ip.md), [검증 기록](sdk-support-ledger.md#서버-생성과-자동-floating-ip-통합). 문서·판정·진행표는 별도 커밋으로 보존 |
| 기존 서버의 `GetActiveServer`·`WaitForServer` | 고정 status/fault/주소·180초/5초·async/sync·삭제 차이 검토 완료 | 공개 API·비동기 IP·필수 관측·lazy Compute binding 완료 | 새17그룹·관련3 package race·전체40 package check 완료 | Python/Go 비교·정확한 main 컴파일·상대 파일 링크 확인 완료 | get_active/wait2행 근거 갱신·정합성 통과, unresolved 유지·지원 승격 없음 | 공유 `b60264d`·Compute `4a305e6`·Connection `85a3063`·경계 `61982b2` push 완료. [사용법](../compute/server-ready.md), [검증 기록](sdk-support-ledger.md#기존-서버의-active-판정과-상위-대기). 문서 `05a2e60` 보존. 판정·단계는 별도 작은 커밋 |
| Neutron 접수 후 IP 모델 보존 | owned POST201/202·PUT200·부분 모델·오류 결합 검토 완료 | accepted decode·검증한 PUT 후보 대체 완료 | 새4그룹·기존 plan 회귀2 package race·전체40 package check 완료 | 기존 plan/readiness의 부분 모델·오류 설명·상대 링크798개 확인 완료 | 기존6행 근거 갱신·정합성 통과, unresolved 유지·지원 승격 없음 | 구현 `6ffc6a3`·서버 흐름 `5c6941d`·오류 assertion `b3a7d56`·문서 `96536db`·판정/진행표 `dfa622f` push 완료. 명시 순차 연결의 부분 결과 기반; 아래 Attach 단계에 연결 |
| 기존 Floating IP의 고정 연결 | Python 주소 선택·Neutron 연결·owner/재연결·revision 차이 검토 완료 | owner-free Attach·정확 주소/ID·고정 tuple·조건부 PUT·부분 결과 완료 | 새10그룹·기존 plan/서버 흐름2 package race·전체40 package check 완료 | Python/Go 비교·정확한 독립 main 컴파일·최종 상대 링크990개 확인 완료 | HTTP 증거·Tags 소유권·오류 원인 보완, 신규 add_ip_list 부분1행 unresolved·최종 parity 통과·지원 승격 없음 | 구현 `413e83b`·기본6그룹 `2e59e28`·경계 `e68f037`·문서 `41d17ae`·판정/진행표 `d3c9ef4` push 완료. [사용법](../network/floating-ip-attach.md), [검증 기록](sdk-support-ledger.md#기존-floating-ip의-고정-연결). 후속 순차 IP·pool 소비자와 standalone source registry는 아래 별도 단위에 기록 |
| standalone IP 흐름의 lazy Compute source registry | cached API/Servers/Flavors 변경 재현 완료 | 공통 준비 단계에서 operation source registry 유지 완료 | 신규 Connection1그룹·집중3 package race 완료 | IP dispatch 문서에 검증 범위·source 실패 보존 기록 완료 | 기존 계약 근거·최종 parity 완료, 지원 승격 없음 | 수정 `71e8962` push 완료. 접수 IP·마지막 raw Server를 보존하고 source 변경 직후 후속 요청 중단 |
| 명시 IP 목록·pool의 선택·연결 흐름 | 고정 pool > ips > auto·backend·주소 순서·부분 실패 검토 완료 | concrete selector·ordered Attempts·Plan/Ensure/Get/Wait/Create 소비 완료 | 신규 Compute13·Connection2그룹·집중3 package race·전체40 package check 완료 | Python/Go 비교·정확한 독립 main 컴파일·상대 파일 링크 검증 완료 | 기존6행 근거 갱신·최종 parity 완료, unresolved 유지·지원 승격 없음 | 구현 `b2ee2af`·Compute 검증 `7028370`·Connection 검증 `bc8d096`·사용 문서 `e8dff01` push 완료. [사용법](../compute/server-ip-dispatch.md), [검증 기록](sdk-support-ledger.md#명시-ip-목록과-pool의-상위-연결-흐름). 독립60초/async entry는 아래 후속 단위에 기록; Nova/full source 계약은 계속 추적. 기존 서비스 우선순위·전체 목표 유지 |
| 독립 AddIPsToServer·AddIPList | 고정60초/async·상태 gate 부재·필수 empty 목록·raw wait·pool 차이 검토 완료 | 공개 Service/Connection·concrete options·동기 raw-only profile·강한 기존 readiness 유지 완료 | 신규 Compute11·Connection3그룹·집중 race·전체40 package check 완료 | Python/Go 비교·독립 main 컴파일·상대 파일 링크 검증 완료 | 기존3행 근거 갱신·최종 parity 완료, unresolved 유지·지원 승격 없음 | 구현 `b5b4174`·기본 검증 `4d2ae19`·강화 `f4fa8a9`·사용 문서 `c3feb51` push 완료. 진행·판정 기록은 별도 커밋으로 공유. [사용법](../compute/server-ip-helpers.md), [검증 기록](sdk-support-ledger.md#독립-서버-ip-helper). 후속 Nova 실행은 다음 행에서 검증했습니다. 남은 named cloud 계약을 계속 처리합니다 |
| Legacy Nova floating IP backend | 고정 Nova available/create/attach·pool·normalization·버전/오류 차이 검토 완료 | 실제 list/선택/할당·compat GET·add action·별도 NovaAssignment·기존 소비자 통합 완료 | 신규 Compute21·Connection4의25그룹 집중 race·전체40 package check 완료 | Python/Go 사용법·기존 설명 교정·독립 main 컴파일·상대 파일 링크 검증 완료 | 기존6행 갱신·최종 parity·정적 검토 완료; 전체 연산 unresolved 유지·지원 승격 없음 | 구현 `d8b4d10`·검증 `03b994c`/`fc0d79b`/`1d96736`/`5b13e09`·오류 경계 `611f0fe`·주석 `4f8a1bf` push 완료. 문서 `30a06ae`·`2a2ed9d` 분리 커밋. [사용법](../compute/server-nova-floating-ip.md), [검증 기록](sdk-support-ledger.md#legacy-nova-floating-ip-backend). 독립 조회·생성·삭제는 아래 완료 행에서 검증했습니다. 현재 Available 반환 view·선택 순서를 보완합니다 |
| 독립 Floating IP availability | 고정 free-first·optional Server·NotFound·normalization 비교 완료 | Network·Compute·Connection API와 actual backend·접수 부분 결과 완료 | 신규 Network7·Connection/Service8의15그룹·Nova 회귀 집중 race·전체40 package check 완료 | Python 비교·정확한 독립 main2개 컴파일·상대 파일 링크 검증 완료 | availability1행 부분 계약11개 추가·최종 parity·정적 검토 완료; unresolved 유지·지원 승격 없음 | 구현 `6de0c4d`·검증 `695ccb4`·경계 `78bd573`·상위 `3cb2689`·검증 `d4c23f7`·terminal `3949ec4` push 완료. 문서 `f7b2e8b`·`60f292a`·`c92120d` push 완료; 판정 `2274394` 분리 커밋. [상위 사용법](../compute/floating-ip-available.md)·[Neutron 사용법](../network/floating-ip-available.md)·[검증 기록](sdk-support-ledger.md#독립-floating-ip-availability). 독립 pool/IP 조회·생성·삭제는 아래 완료 행으로 연결했습니다. 현재 Available 보완은 별도 진행 행에서 추적합니다 |
| 독립 Floating IP 목록·검색·단건·pool 조회 | 고정 cloud6연산·dict/JMES·404·Resource/Nova 정규화 비교 완료 | Service·Connection API6개·owned view/Wire·공통 concrete 옵션 완료 | 기존25+신규 회귀3그룹·관련 집중 race·최종40 package check 완료 | Python/Go 비교·정확한 독립 main 컴파일·기존 Go fence/상대 파일 링크 확인 완료 | Cloud6개 go_mapping; Neutron eager descriptor/local filter·빈 nested dict 오류 수정·판정 정합성 완료 | 기존 코드 push 완료; 수정 `e4dfabe`·회귀 `f839131`·최종 문서/판정 `73bcfba`까지 push 완료. [사용법](../compute/floating-ip-queries.md)·[검증 기록](sdk-support-ledger.md#floating-ip-named-계약-재검토와-현재-집계)
| 독립 Floating IP 삭제·공개 조회 재검증 | 고정 cloud retry·DELETE helper·일반 Get·passive 응답·버전 차이 검토 완료 | Service/Connection·SDK 소유 옵션·논리 Attempts·접수/조회 proof 완료 | 신규 HTTP20·pure3의23그룹·관련88그룹 집중 race·전체40 package check 완료 | Python/Go 비교·정확한 독립 main 컴파일·기존 Go 예제/상대 파일 링크 확인 완료 | 기존23그룹·22계약 재감사 완료; named Cloud 삭제1개 go_mapping, 다른 Proxy/Resource 선언은 별도 | 공유 조회 `8d08c61`·구현 `98b5c7e`·옵션 `5a9a5ff`·기본 `eb14968`·경계 `bb52340`·success guard `0880f87`·문서 `edeabe2`/`86c79bd` push 완료. 판정 `afb6063`·진행 기록은 별도 작은 커밋. [사용법](../compute/floating-ip-delete.md)·[검증 기록](sdk-support-ledger.md#독립-floating-ip-삭제와-공개-조회-재검증). 후속 Create·wait/timeout 정리와 미연결 IP 삭제는 아래 두 완료 행에서 검증했습니다 |
| 독립 Floating IP Create·선택적 wait/timeout 정리 | fresh·port 우선·Network/pool presence·Resource seed·wait-only60초·cleanup 비교 완료 | Neutron·Nova·Service/Connection·wait/cleanup 구현 완료 | 신규24그룹·조회 회귀3그룹·集中157그룹 race·최종40 package check 완료 | Python 비교·정확한 독립 main 컴파일·상대 링크 검증 완료 | Cloud Create1개 go_mapping; 실제19계약·24신규그룹 연결 | 코드·테스트 `bf3024a`, 가이드 `6af6a88`, 판정 `e1a580a`, 집계 `73bcfba`까지 push 완료. [사용법](../compute/floating-ip-create.md)·[검증 기록](sdk-support-ledger.md#독립-floating-ip-생성과-조회-완료). 후속 미연결 IP 삭제는 다음 완료 행에서 검증했습니다 |
| 미연결 Floating IP 일괄 정리 | Neutron gate·eager inventory·port truthiness·false/error·retry·fallback backend 비교 완료 | Service/Connection·기존 concrete Delete 옵션·owned partial Count/Items 완료 | 신규 HTTP12그룹·집중 race169그룹·`b75fc9c` 전체40 package check 통과 | Python 비교·정확한 독립 main 컴파일·문서 링크 확인 완료 | Cloud named1개 go_mapping·실제14계약·12신규그룹 연결 | 공통 `b0bcc4c`·구현 `14ed0f6`·검증 `27da548`/`b75fc9c` push 완료. 문서 `158c2fd`/`9d066b4`·판정 `0136cfa`·집계 `e56409b` push 완료. [사용법](../compute/floating-ip-unattached-delete.md)·[검증 기록](sdk-support-ledger.md#미연결-floating-ip-일괄-정리). 다음: 남은 핵심 user 계약 |
| `AvailableFloatingIP` 반환 view와 남은 선택 순서 | 고정 소스 비교 완료 | **Resource/Wire·location·strict·옵션 snapshot/guard 구현 완료** | 신규 HTTP 7그룹·`239eb9e` 전체 40 package check PASS | Python 비교 갱신·정확한 독립 main 컴파일 PASS | unresolved 유지; 필수 3개 중 view 연결 1개 구현·집중 검증 완료 | 구현 `4897448`·검증 `cf2c5e1`/`239eb9e` push 완료. 문서·전체 검증 완료; Neutron 내부 fallback → Nova raw 필터·list404 생성 순서의 2항목을 이어서 보완 |
| `AvailableFloatingIP` 직접·외부 Nova 선택 | 고정 raw 필터·전체 정규화·list404/fresh 순서 검토 완료 | **Query/Create 재사용·Inventory/Creation·optional typed projection 완료** | 기존 fixture 재사용 HTTP4그룹·관련3 package 집중 race·전체40 package check PASS | Python 비교·오류/부분 결과 사용법 갱신 | unresolved 유지, 필수 remaining **2→1** | 구현 `3b00178`·검증 `b9ef579`·사용법/판정 `099caaa` push 완료. Neutron 내부 목록 fallback·로컬 필터·선택 network allocation은 다음 단위 |
| `AvailableFloatingIP` Neutron 내부 선택·named 완료 | 고정 public list·filter·fresh 소스 검토 완료 | **Query/Create/Allocate·concrete plan 재사용 완료** | 기존 fixture 재사용 HTTP9그룹·typed NAT3사례·집중 race·전체40 package check·smoke5흐름 PASS | 전체·Compute·Network 사용법 갱신; 기존 독립 main fence 보존 | **go_mapping, remaining1→0; 전체176/3,362** | 공통 준비 `d190fd6`·NAT `2c5e2c8`·구현 `c434d93`·검증 `6bf368b`·사용법 `b1cb273`. 내부 inventory와 외부 availability를 구분, accepted 오류의 재할당0 |
| 핵심 user API의 나머지 미해결 계약 선별 | 진행 | 대기 | 대기 | 대기 | 대기 | 1단계. 기존 인증·조회와 새 Identity 수정 이후 Nova·Neutron·Glance·Cinder·Barbican·Swift 및 cloud/Resource 계약을 계속 추적 |
| Cinder `ManageVolume` | 예비 소스 조사 | 공통 모델 준비만 완료, 공개 API 미구현 | API 계약 검증 대기 | 사용 문서 대기 | 미완료, 지원 승격 없음 | 2단계 후보로 이동. 재개 시 조사 결과와 admin 분류를 고정 소스·권한 정책과 비교하고 저장소에 근거 기록 |
| Glance·Swift Task 업로드 연계 | 선택 분기·Swift 의존 확인, 세부 계약 조사 대기 | 상위 연계 미완료 | 연계 계약 검증 대기 | 기존 이미지 문서에 남은 범위 기록 | 미완료, 지원 승격 없음 | 2단계 후보. Swift 기본 연산은 1단계에서 준비하고 Task 권한·대기·정리·부분 실패 계약을 함께 조사 |
| 매니지드·확장 서비스의 user 계약 | 기존 판정별 근거 유지, 권한 분류 예정 | 추가 구현 대기 | 추가 검증 대기 | 추가 문서 대기 | 남은 계약별 미완료 | 3단계. 네트워크 → 베어메탈 → 나머지. 기존 완료 기능을 그대로 유지 |
| 매니지드·확장 서비스의 admin 계약 | 기존 판정별 근거 유지, 권한 분류 예정 | 추가 구현 대기 | 추가 검증 대기 | 추가 문서 대기 | 남은 계약별 미완료 | 4단계. 네트워크 → 베어메탈 → 나머지. 기존 완료 기능을 그대로 유지 |

## 지원 여부와 검증 근거의 관리

| 자료 | 현재 맡는 역할 |
|---|---|
| [전체 catalog](../api/sdk_support_catalog.json) | 두 고정 소스의 직접 선언 ID와 fingerprint. 판정이 없는 선언도 `unresolved`로 추적 |
| [연산별 판정](../api/sdk_reviews.json) | `supported`·`go_mapping`·`unsupported`·`unresolved`, 공개 API, 계약별 테스트, 문서, Go 차이와 `remaining` |
| [지원 판정대장](sdk-support-ledger.md) | 단위별 구현 범위, 수동 소스/계약 검토, 실제 QA·예제 검증 결과와 커밋 근거 |
| 이 문서 | 서비스·사용 흐름의 우선순위와 작업 단위별 진행 단계 |
| [테스트 안내](testing.md), [검증기 안내](../internal/cmd/paritycheck/README.md) | 실행 명령과 자동 검증 범위 |

현재 실행 로그·세부 소스 대조·검증 receipt 일부는 `/private/tmp`에 보관하고, 검증 요약과 계약별 테스트·문서·커밋 연결을 저장소에 기록합니다. 임시 파일은 장기 보존을 보장하지 않으므로 후속 작업자가 필요한 결론·남은 범위·실행 결과를 저장소 기록만으로 확인할 수 있게 갱신합니다.

`make check`는 vet·race test·지원 판정 정합성·gofmt를 실행합니다. `paritycheck`는 API와 테스트 함수 및 문서의 존재, source pin/fingerprint, 판정과 남은 기능의 정합성을 확인합니다. **테스트가 계약을 충분히 증명하는지, 실제 실행이 통과했는지는 수동 검토와 실행 결과로 확인합니다.** 현재 HTTP 테스트는 로컬 모의 서버 검증이며 실클라우드 acceptance나 Python 예제 실행은 별도 미검증 범위입니다.

최신 지원 판정 개수는 위의 [현재 집계](#현재-집계와-진행-중인-작업)에만 기록하고 판정 JSON에서 다시 계산합니다. 이 집계는 두 소스의 직접 선언 수이며 중복을 제거한 HTTP endpoint 수나 상속 표면을 포함한 전체 SDK 완료율과 다릅니다. 구현·테스트가 있는 연산도 해당 선언의 필수 계약 검토가 남으면 미해결로 유지합니다. 완료된 named 선언에 별도 Resource/session 목표의 남은 작업을 중복 blocker로 붙이지 않습니다.

전체 완료는 [기존 완료 기준](sdk-support-ledger.md#완료-판정)을 따릅니다. 우선순위 조정으로 `unsupported`·`unresolved`를 제외하거나 생성 transport 수를 SDK 완료 수로 바꾸지 않습니다.
