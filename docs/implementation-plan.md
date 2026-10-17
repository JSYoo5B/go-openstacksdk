# 구현 순서와 단계별 현황

목표는 고정한 Gophercloud와 openstacksdk의 API·리소스·복합 작업을 일관된 Go SDK로 제공하는 것입니다.
SDK가 기본값·구체적인 옵션·확장 처리와 공통 리소스 동작을 소유하며, 애플리케이션에 builder 구현 책임을 넘기지 않습니다.

구현 순서는 **핵심 서비스 user API → 핵심 서비스 admin API → 매니지드·후속 서비스 user API → 매니지드·후속 서비스 admin API**입니다.
기존 구현은 유지하고 실제 근거가 있는 연산의 판정만 갱신합니다.

고정 기준은 Gophercloud `v2.15.0`과 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다.
정확한 source pin과 선언 fingerprint는 [catalog](../api/sdk_support_catalog.json)와 [판정 JSON](../api/sdk_reviews.json)에서 관리합니다.

## 현재 집계와 진행 중인 작업

숫자는 다음 세 의미로 구분합니다.

- **API 매핑:** 고정 선언에 대응하는 공개 Go API가 정해진 profile로 연결되고 검토된 연산 수입니다. 전체 openstacksdk 동등성이나 전체 SDK 완료 수가 아닙니다.
- **미해결 항목:** 연산별 필수 동작에 실제 누락이 있거나 아직 검토하지 않은 선언입니다. 부분 구현·테스트·문서가 있어도 남은 차이가 있으면 유지합니다.
- **검증 동작 기록:** 판정 JSON의 `contracts` 항목입니다. 한 연산에 여러 기록이 있을 수 있으므로 API 완료 수·테스트 함수 수·전체 구현률로 사용하지 않습니다.

`go_mapping`과 `supported`는 서로 다른 판정입니다. `supported`는 고정 Source 동작 그대로의 지원이며, `go_mapping`에는 명시한 Go profile과 차이가 있습니다.
서비스 표의 **API 대응**은 `go_mapping`과 `supported`를 합한 수이며, 서비스 전체나 user/admin 단계 완료를 뜻하지 않습니다.

<!-- sdk-progress:start -->
| 지표 | 현재 값 | 해석 |
|---|---:|---|
| 고정 소스 전체 선언 | 3,362 | Gophercloud·openstacksdk 선언; inherited/descriptor/Resource 표면은 별도 추적 |
| 구현 대상 선언 | 3,228 | 전체 선언에서 폐기된 서버 API 등 `unsupported`로 제외한 선언을 뺀 수 |
| API 매핑 | 2,059 (63.8%) | 구현 대상 대비 `go_mapping` 판정. 부분 검증 기록을 추가해도 이 수는 늘지 않음 |
| source 그대로 지원 | 0 | `supported` 판정 |
| 미해결 / 제외(미지원) | 1,169 / 134 | 미검토 선언도 미해결 집계에 포함 |
| 연산별 검토 기록 | 2,487 | 아직 개별 기록 없는 선언 875 |
| 검증 동작 기록 | 7,570 | [판정 JSON](../api/sdk_reviews.json)의 `contracts` 항목 수; 기록 분할 크기가 달라 API 완료 수·구현률로 사용하지 않음 |
<!-- sdk-progress:end -->

최근 두 작업은 핵심 Neutron cloud의 firewall policy/group 수정 helper입니다.
정확한 검증 범위는 아래 [현재 작업 현황](#현재-작업-현황)과 판정 JSON을 봅니다.
과거 사이클별 quota·commit·소스 hash와 반복 보고는 이 계획에서 제거했습니다. 필요한 이력은 Git과 [지원 판정대장](sdk-support-ledger.md)의 검증 근거에서 확인합니다.

## 서비스 우선순위별 API 집계

`make progress`가 생성하는 아래 표는 **서비스별 누적 API 매핑**을 비교하기 위한 자료입니다.
서비스별 수는 user/admin 합산이며 각 구현 단계의 완료율이 아닙니다.
판정 JSON에 권한별 분류가 완성되지 않아 user/admin 단계별 API 수는 아직 분리 집계하지 않습니다.

고정 소스의 직접 선언을 주관 서비스에 한 번씩 배정합니다.
복합 작업이 다른 서비스를 호출하더라도 중복으로 세지 않으며, 상속·descriptor·Resource 전체 표면은 이 분모와 별도로 추적합니다.

<!-- sdk-service-progress:start -->
| 우선순위 묶음 | API 대응 / 대상 | API 대응률 | 부분·미해결 검토 | 미검토 | 제외(미지원) |
|---|---:|---:|---:|---:|---:|
| 핵심 서비스 · 1·2단계 합산 | 1,981 / 2,158 | 91.8% | 176 | 1 | 134 |
| 후속 네트워크 · 3·4단계 합산 | 5 / 254 | 2.0% | 13 | 236 | 0 |
| 후속 베어메탈 · 3·4단계 합산 | 1 / 207 | 0.5% | 1 | 205 | 0 |
| 후속 나머지 · 3·4단계 합산 | 68 / 580 | 11.7% | 85 | 427 | 0 |
| 서비스 공통 기반 | 4 / 29 | 13.8% | 19 | 6 | 0 |
| 전체 | 2,059 / 3,228 | 63.8% | 294 | 875 | 134 |

**핵심 서비스**

| 서비스 | API 대응 / 대상 | 부분·미해결 검토 | 미검토 | 제외(미지원) |
|---|---:|---:|---:|---:|
| Identity / Keystone | 358 / 379 | 20 | 1 | 10 |
| Compute / Nova | 287 / 333 | 46 | 0 | 0 |
| Placement | 68 / 71 | 3 | 0 | 0 |
| Network / Neutron | 660 / 728 | 68 | 0 | 30 |
| Image / Glance | 107 / 109 | 2 | 0 | 11 |
| Block Storage / Cinder | 379 / 398 | 19 | 0 | 82 |
| Key Manager / Barbican | 61 / 67 | 6 | 0 | 0 |
| Object Storage / Swift | 61 / 73 | 12 | 0 | 1 |

**후속 서비스 · 네트워크 → 베어메탈 → 나머지**

| 서비스 | API 대응 / 대상 | 부분·미해결 검토 | 미검토 | 제외(미지원) |
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

현재 1단계 핵심 user 동작을 보완 중입니다. 2단계 이후에도 기존에 매핑한 기능은 보존합니다.
후속 서비스의 누적 값에는 우선순위 조정 전에 구현한 기능이 포함되므로 작업 순서와 누적 매핑을 구분합니다.

## 서비스 우선순위

| 순서 | API 범위 | 우선 작업 |
|---|---|---|
| 1 | 핵심 서비스 user API | 인증·프로젝트 선택, 이름/ID 조회, 기본 CRUD·목록·페이지네이션·대기, 서버·네트워크·볼륨·이미지·키·객체 저장 흐름 |
| 2 | 핵심 서비스 admin API | 전역 관리·quota·서비스/host·Placement, Cinder ManageVolume 등 관리자 action과 관련 기본값·오류 처리 |
| 3 | 매니지드·후속 서비스 user API | 네트워크 → 베어메탈 → 나머지의 사용자 리소스·복합 작업, 부모 범위·비동기 완료·부분 실패 |
| 4 | 매니지드·후속 서비스 admin API | 같은 내부 서비스 순서의 전역 관리·quota·운영자 action |

핵심 서비스는 Identity(Keystone), Compute(Nova), Placement, Network(Neutron), Image(Glance), Block Storage(Cinder), Key Manager(Barbican), Object Storage(Swift)입니다.

후속 단계 안의 서비스 순서는 다음과 같습니다.

1. 네트워크: Load Balancer(Octavia), DNS(Designate).
2. 베어메탈: Bare Metal(Ironic), Bare Metal Introspection.
3. 나머지: Database(Trove), Container Infra(Magnum), Container(Zun), Clustering(Senlin), Orchestration(Heat), Messaging(Zaqar), Workflow(Mistral), Shared File System(Manila), Instance HA(Masakari), Metric, Reservation(Blazar), Accelerator(Cyborg).

후속 user API를 처리한 뒤 같은 서비스 순서로 후속 admin API를 처리합니다.
Neutron은 핵심에, Barbican과 Swift도 핵심에 유지합니다. 모든 후속 서비스를 배포 형태까지 관리형으로 분류하는 것은 아닙니다.

Swift는 일반 Glance 업로드의 필수 의존은 아니지만 고정 Source의 Swift 경유 이미지 Task 업로드에 필요합니다.
이 경로는 container/object 생성 → Glance import Task → 선택적 대기·이미지 갱신·객체 정리를 연결합니다.
관련 남은 범위는 [Image 안내](../image/README.md#전체-api와-남은-작업)에서 관리하며, Source의 admin-only 설명과 서버 정책을 검토해 핵심 admin 단계에 배치합니다.

user/admin은 함수 이름이나 CRUD만으로 판단하지 않습니다.
정책·사용 목적을 근거로 분류하고 `all_projects`·force처럼 한 함수에 관리자 분기가 있으면 검증 동작을 나누어 추적합니다.
배포별 권한이 미확정이면 근거와 차이를 남깁니다.

핵심 user 단계는 다음 흐름의 남은 항목을 우선합니다.

- 인증·접근 가능한 프로젝트·공유 토큰·endpoint 선택과 기본 조회.
- 이미지·flavor·network·port 해석, 서버 생성·대기·삭제, floating IP와 부분 실패.
- Cinder 생명주기·서버 연결/분리·볼륨/snapshot 부팅·이미지 연계.
- Barbican secret/container·payload·권한·옵션.
- Swift container/object·metadata·스트림 소유권·무결성.
- 사용자 quota/limits·확장·이전 버전과 공통 Resource 동작.

이미 검증한 기능은 재구현하지 않고 연결된 `remaining`을 확인합니다.
앞선 흐름을 막는 의존 기능이나 회귀는 필요한 범위만 먼저 처리합니다. 이는 후속 단계 전체를 앞당기는 지시가 아닙니다.

## 구현 단위와 검증 반복의 조정

작업을 시작할 때 연산과 필요한 동작 목록을 고정합니다.
동일한 공통 엔진·profile·descriptor·converter·fixture를 재사용하고, 새 요청·입력·결과·실패 분기에 필요한 검증을 추가합니다.
공개 Gophercloud testhelper를 재사용할 수 있으면 새 HTTP 테스트 프레임워크를 만들지 않습니다.

의존성 없는 서비스나 검토 작업은 파일 소유를 나누어 병렬 처리할 수 있습니다.
공통 계약과 통합 범위는 먼저 정하고 root가 공통 수정·Go 검사·중앙 판정·Git 작업을 순서대로 처리합니다.
승인된 사이클의 범위를 늘려 여러 사이클을 하나로 합치지 않습니다.

집중 검증이 통과한 구현은 작은 의미 있는 커밋으로 보존하고 main에 push합니다.
전체 검사는 공통 기반과 변경 영향에 맞춰 묶어서 수행합니다.
같은 Go 소스의 문구·판정 갱신만으로 전체 검사를 반복하지 않으며, 새 코드·실패·미해결 위험이 있으면 필요한 검증을 다시 합니다.

생성 transport나 검증 동작 기록 증가만으로 API 매핑을 올리지 않습니다.
기존 미해결 연산도 실제 필수 누락을 다시 확인하고, 구현·테스트·문서가 모두 갖춰졌을 때 판정을 바꿉니다.
범용 Resource/session 목표를 모든 선언의 포괄적인 blocker로 중복해서 붙이지 않습니다.

## 공통 SDK의 남은 계약

아래 공통 범위는 연산별 API 매핑과 별도로 추적합니다.
미완료인 동안 전체 openstacksdk 동등성이나 전체 SDK 목표 완료를 주장하지 않습니다.

| 추적 ID | 남은 구현·검증 |
|---|---|
| SDK-R1 | 같은 mutable Resource 객체의 fetch/update, Body/Header/URI component와 dirty state·baseline·기본값의 서비스 공통 lifecycle. Owned view·snapshot와 일부 Image 수정 동작은 있으나 전체 공통화는 미완료 |
| SDK-C1 | 선택적 Adapter cache, conditional GET·ETag/Last-Modified·304 재사용·invalidation과 호출 간 소유권. 현재 fresh HTTP이며 캐시 동등성 없음 |
| SDK-J1 | 고정 Proxy 목록의 deprecated JMESPath·임의 결과 shape·오류·paging binding. 공통 기반은 있으나 ImageRecord 목록의 jmespath_filters는 미지원 |
| SDK-S1 | 동적 per-call session·base_path·microversion, adapter/Resource override 우선순위. 현재 source capture/guard와 구체적인 옵션 범위만 구현 |

연산별로 실제 누락인 status·encoding·raw control·부분 응답·참조 타입 차이는 해당 서비스 가이드와 `remaining`에 명시합니다.
실클라우드 acceptance와 릴리즈 기준도 로컬 모의 HTTP 검증과 분리해 남깁니다.

## 작업 단위의 진행 단계

| 단계 | 완료 기준 |
|---|---|
| 소스 검토 | 연산 ID·source pin·권한/서비스 단계와 입력·기본값·반환·오류·버전·상속의 필수 동작을 확인 |
| 구현 | SDK 소유 공개 옵션·기본값·모델·공통 정책을 구현하고 작은 커밋으로 보존 |
| 테스트 | 요청 path/body/query/header, 응답·실패·취소·부분 성공의 의미 있는 집중/회귀 검증 |
| 문서 | 서비스 사용법과 Python/Go 비교·기본값·차이·부분 결과, 변경한 실행 예제의 빌드 확인 |
| 최종 검토·판정 | Source·실제 assertion·문서·실행 결과를 대조해 API/검증 동작/차이/remaining 갱신, 정합성 검사 |
| 공유 | 승인된 push, 자동 진행 집계와 현재 단계 갱신 |

단계는 겹칠 수 있습니다. 구현과 테스트·문서를 함께 작성하고 전체 단위 종료까지 커밋을 미루지 않습니다.
소스가 바뀌면 영향받는 검증을 갱신하며 이전 revision의 성공을 변경 후 성공으로 옮겨 적지 않습니다.

대화의 중간 보고에는 다음만 짧게 알립니다.

- 현재 기능과 진행 단계.
- 실제로 통과·실패한 검증과 아직 실행 중인 검사.
- 커밋·push 상태와 현재 묶음의 남은 작업.
- 숫자를 보고할 때 API 매핑 변화와 검증 동작 기록 변화를 별도로 표시.

## 현재 작업 현황

| 대상 | 구현·검증 상태 | 연산 판정 |
|---|---|---|
| Cloud firewall policy 수정 | target-first strict 조회·순서 있는 rule 해석·raw 변경 감지 구현. 집중 8그룹·31사례와 전체 205 test package·예제 검사 통과 | 응답·Resource·Find 차이가 남아 unresolved |
| Cloud firewall group 수정 | target→egress→ingress→ports, null/빈 목록·부분 실패 구현. 집중 9그룹·29사례 통과; 전체 205 test package·예제 검사 통과 | 응답·Resource·port/Find 차이가 남아 unresolved |

이번 두 연산은 API 매핑을 늘린 것으로 보고하지 않습니다. 실제 추가한 동작은 연산별 검증 동작 기록과 테스트로 연결합니다.
전체 목표는 미완료이며 다음 범위는 핵심 user 단계의 남은 필수 동작에서 별도로 고정합니다.
반복 자동화는 PAUSED입니다. 이번 승인 범위가 끝나면 새 사이클을 임의로 시작하지 않습니다.

## 지원 여부와 검증 근거의 관리

| 자료 | 역할 |
|---|---|
| [전체 catalog](../api/sdk_support_catalog.json) | 고정 소스의 직접 선언 ID·fingerprint와 전체 대상 |
| [연산별 판정](../api/sdk_reviews.json) | supported/go_mapping/unsupported/unresolved, 공개 API, 검증 동작별 테스트·문서·차이·remaining |
| [지원 판정대장](sdk-support-ledger.md) | 검증 요약과 이력 근거. 상세 과거 기록은 여기와 Git 이력을 참조 |
| 이 문서 | 서비스 우선순위·현재 작업·단계별 관리와 자동 집계 |
| [테스트 안내](testing.md), [검증기 안내](../internal/cmd/paritycheck/README.md) | 실행 방법·자동 검증의 범위 |
| [릴리즈 기준](release-milestones.md), [설치 안내](install.md) | 로컬 preview와 외부 alpha의 별도 완료 조건·설치 검증 |

`make check`는 license·vet·race test·판정 정합성·진행 집계·gofmt 등을 확인합니다.
정합성 검사는 API/테스트 함수/문서 존재와 pin·fingerprint·판정 관계를 검사합니다.
테스트가 동작을 충분히 증명하는지는 별도 검토가 필요하며, 실제 실행 성공도 실행 결과로 확인합니다.

Source AST+stub 관찰은 실제 Source의 해당 본문·순서만 증명하며 전체 Python SDK 실행이나 HTTP 검증으로 보고하지 않습니다.
로컬 모의 HTTP, 예제 Go 빌드·Python AST, 실클라우드 acceptance와 릴리즈 검증을 구분합니다.
최초 실패와 수리 후 성공도 구분해 필요한 검증 기록에 보존합니다.

최신 누적 숫자는 자동 집계 두 표에서만 관리합니다.
이 분모는 중복 제거한 HTTP endpoint 수나 상속 표면 전체가 아니며 검증 동작 기록 수에도 전체 구현률을 붙이지 않습니다.
전체 완료는 [완료 판정](sdk-support-ledger.md#완료-판정)을 따릅니다. 우선순위 조정으로 미해결·미지원을 지우거나 생성 코드 수를 SDK 완료 수로 바꾸지 않습니다.
