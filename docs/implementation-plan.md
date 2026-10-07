# 구현 순서와 단계별 현황

2026-10-07에 조정한 작업 순서입니다. **핵심 서비스 user API → 핵심 서비스 admin API → 매니지드 서비스 user API → 매니지드 서비스 admin API** 순으로 진행합니다. 목표는 고정한 Gophercloud와 openstacksdk의 전체 API·리소스·복합 작업을 일관된 Go SDK로 제공하는 것입니다. 관리형 서비스의 구현도 전체 목표에 포함하며, 기존 구현과 지원 판정은 보존합니다.

이 문서는 **앞으로 할 작업의 순서와 진행 단계**, [SDK 지원 판정대장](sdk-support-ledger.md)은 **검증한 계약과 실행 근거**, [판정 JSON](../api/sdk_reviews.json)은 **연산별 지원 여부**를 관리합니다. 세 자료의 상태는 서로 다른 의미를 갖습니다.

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

단계는 일부 겹칠 수 있습니다. 예를 들어 계약 테스트와 문서는 구현과 함께 작성하고, 커밋·push도 단위가 완전히 끝날 때까지 미루지 않습니다. 코드가 바뀌면 영향받는 단계의 검증을 다시 확인하며, 이전 revision의 성공을 변경 후 성공으로 옮겨 적지 않습니다.

## 현재 작업 현황

아래는 2026-10-07, 코드 기준 `a35f92f`에서 확인한 최근 단위와 다음 작업입니다. 전체 API의 단계별 보드는 아직 없으며 이 표부터 단위별 진행 상태를 저장소에 남깁니다. 이후 작업을 시작하거나 단계가 바뀔 때 행과 근거를 함께 갱신합니다.

| 작업 단위 | 소스 검토 | 구현 | 테스트 | 문서 | 최종 검토·판정 | 커밋·push / 다음 행동 |
|---|---|---|---|---|---|---|
| Cinder `UploadVolumeToImage` | 완료 | 완료 | 집중 14그룹·전체 race·vet 완료 | Python 비교·3개 호출 경로·예제 컴파일 완료 | 해당 Python Proxy 1개 연산 `go_mapping`; native/v2/Resource 등은 별도 | `f9892f5`까지 push 완료. [검증 기록](sdk-support-ledger.md#cinder-v3-volume-image-export), [사용법](../blockstorage/volume-upload-image.md) |
| Volume 순수 모델 변환 공통화 | 기존 변환 계약 비교 완료 | 완료 | 기존 계약 회귀·전체 race·vet 완료 | 내부 변경을 지원대장에 기록 | 의미 보존 검토 완료, API 지원 승격 없음 | `a35f92f` push 완료. [검증 기록](sdk-support-ledger.md#volume-모델-변환의-공통-내부-계층) |
| 핵심 user API의 미해결 계약 선별 | 예정 | 대기 | 대기 | 대기 | 대기 | 1단계. Identity·Nova·Neutron·Glance·Cinder·Barbican·Swift와 관련 cloud/Resource 계약 및 권한을 조사해 다음 작은 구현 단위를 확정 |
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

2026-10-07의 지원 판정은 직접 선언 **3,362개** 중 `supported` 0개, `go_mapping` 159개, `unsupported` 1개, `unresolved` 3,202개입니다. 저장된 review 466개에는 Go 매핑 159개·부분 검토 306개·미지원 1개가 있고, review가 없는 2,896개도 미해결에 포함됩니다. 이 개수는 두 소스의 선언 집계이며 중복을 제거한 HTTP endpoint 수나 상속 표면을 포함한 전체 SDK 완료율이 아닙니다. 구현·테스트가 있는 연산도 전체 계약 검토가 남으면 미해결로 유지합니다.

전체 완료는 [기존 완료 기준](sdk-support-ledger.md#완료-판정)을 따릅니다. 우선순위 조정으로 `unsupported`·`unresolved`를 제외하거나 생성 transport 수를 SDK 완료 수로 바꾸지 않습니다.
