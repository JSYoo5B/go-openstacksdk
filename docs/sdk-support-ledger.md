# SDK 지원 판정대장

이 문서는 **API 매핑**, **미해결 동작**, **검증 동작 기록**을 구분합니다. 고정 비교 대상은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**입니다. 생성된 호출 함수 수나 테스트 수를 완료 API 수로 사용하지 않습니다.

- [연산별 판정 JSON](../api/sdk_reviews.json): 실제 공개 API, 테스트·문서 근거, 차이와 미해결 항목
- [구현 계획의 자동 집계](implementation-plan.md#현재-집계와-진행-중인-작업): 서비스 우선순위별 API 매핑과 미해결 진행량
- [이전 검증 이력](history/sdk-support-ledger-2026-10-18.md): 상세 구현 이력, 첫 실패·수리 후 통과, Source 관찰 범위와 당시 집계

`make progress`가 판정 JSON에서 집계를 갱신하고 `make check`가 불일치를 검사합니다. **검증 동작 기록 수는 구현 완료 API 수와 다른 지표**입니다. 한 API의 일부 동작을 검증했어도 나머지 동작이 남으면 해당 연산은 `unresolved`를 유지합니다.

## 판정 상태

| 상태 | 의미 | 완료로 인정할 증거 |
|---|---|---|
| `supported` | 고정 소스의 동작을 Go API로 제공 | 입력·기본값·결과·오류·해당 리소스의 공통 정책을 구현하고 동작을 검증한 테스트와 사용 문서 |
| `go_mapping` | Go의 타입·context·오류·옵션에 맞게 형태나 기본값을 의도적으로 변경 | 실제 사용 가능한 대체 API, Python과 달라지는 의미의 명시, 동일 기능을 검증한 테스트와 사용 문서 |
| `unsupported` | 필요한 기능이 구현되지 않았음을 확인 | 빠진 동작과 이유, 구현할 대상 또는 의존 API. 지원 완료 상태가 아님 |
| `unresolved` | 동작 또는 근거를 아직 판정하지 못함 | 비교할 소스와 다음 확인 작업. 이름 기반 후보 매칭도 이 상태에 해당 |

`unsupported`는 검토를 마쳤다는 뜻이지 프로젝트 목표를 달성했다는 뜻이 아닙니다. 사용자에게 필요한 기능을 제외하는 결정이나 Go에서 제공하기 어려운 구현을 회피하는 데 `go_mapping`을 사용하지 않습니다. 일부 기능만 지원한 연산은 전체 연산을 `supported`로 표시하지 않고 세부 동작을 나눕니다.

## 현재 판정과 미해결

API 매핑·미해결·명시적 제외 항목의 최신 수치는 [구현 계획의 자동 집계](implementation-plan.md#현재-집계와-진행-중인-작업)를 따릅니다.

아래 두 연산에는 공개 helper와 검증 동작 기록을 각각 2개 추가했습니다. 남은 차이가 있어 두 연산 모두 `unresolved`를 유지하며, API 매핑 수는 늘리지 않았습니다.

공통 미해결은 다음과 같습니다.

- SDK-R1/C1/S1의 Resource·adapter·session·cache·동적 descriptor 수명 주기
- 원래 mutable Resource 반환, unknown·부분·빈·잘못된 성공 응답의 병합과 원본 mutation receipt
- 전체 accepted status·Requests encoding·비유한 JSON·Python 임의 객체/iterable/ID 동작
- raw Find의 headers·microversion·fetch/constructor controls와 기존 port 조회 profile 차이
- 인증된 실클라우드 acceptance, 외부 설치·릴리즈의 남은 조건

서비스 진행 순서는 핵심 user → 핵심 admin → 매니지드 user → 매니지드 admin입니다. Swift는 핵심에 유지하고 후속 네트워크·베어메탈을 우선합니다. [Glance 정책 우선순위](glance-policy-priorities.md)와 [릴리즈 기준](release-milestones.md)을 함께 따릅니다.

## 2026-10-18 방화벽 후속 두 사이클: group 수정

[`UpdateFirewallGroupRecord`](../network/cloud_firewall_group_update.go)는 strict 대상 조회 → egress/ingress policy 조회 → 입력 순서의 port 조회 → 변경 필드 PUT을 제공합니다. 대상 필터를 참조 조회에 전파하지 않고 중복 참조도 다시 조회합니다. policy null은 조회 없이 ID null로 바꾸며 `ports=[]`는 port 조회 없이 비웁니다. Source docstring과 달리 실제 decorator가 금지하는 direct policy ID 인자도 거부합니다. 사용법과 지원 범위는 [비교 가이드](../network/cloud-firewall-group-update-python.md)에 있습니다.

| 검증 단계 | 결과와 범위 |
|---|---|
| 집중 검사 | group 9그룹·29사례와 policy 회귀 8그룹·31사례, 합계 17그룹·60사례 최초 PASS |
| Source 관찰 | 실제 method/helper AST와 stub 7사례로 순서·null·빈 목록·중복·필터 분리를 확인; decorator·Resource commit·HTTP·실클라우드 검증은 아님 |
| 예제·전체 검사 | 독립 Go main/Python AST 각 1개와 전체 205 test package·license/vet/race/parity/progress/gofmt 최초 PASS |
| 판정 | `unresolved` 유지, 검증 동작 기록 2개 추가; 구현 `593d3f04` |

동결 전 잘못된 port 응답 ID 오류에 원본 body/header/status를 보존하도록 보강했습니다. 집중 검사 실패는 없었습니다. native HTTP 200 전용 응답·typed model·응답 병합·mutation receipt·port lookup profile·Python 참조 타입과 공통 미해결은 남아 있습니다.

## 2026-10-18 방화벽 후속 두 사이클: policy 수정

[`UpdateFirewallPolicyRecord`](../network/cloud_firewall_policy_update.go)는 strict 대상 policy 조회 뒤 rule 참조를 입력 순서와 중복 그대로 조회해 ID로 바꾸고, 변경 필드만 PUT합니다. `firewall_rules=[]`는 rule 조회 없이 비우며 같은 ID 목록도 참조 조회를 마친 뒤 PUT을 생략합니다. 중간 실패는 앞선 조회 증거를 보존하고 이후 조회·PUT을 중단합니다. [비교 가이드](../network/cloud-firewall-policy-update-python.md)에 문자열 참조 범위와 null의 대상 조회 후 오류 순서를 설명합니다.

| 검증 단계 | 결과와 범위 |
|---|---|
| 집중 검사 | policy 8그룹·31사례와 rule 회귀 5그룹·21사례, 합계 13그룹·52사례 최초 PASS |
| Source 관찰 | method/helper AST와 stub 7사례로 대상 우선·필터 분리·참조 순서·빈 목록·null 오류를 확인; decorator·Resource commit·HTTP·실클라우드 검증은 아님 |
| 예제 | 독립 Go main 빌드와 Python AST 각 1개 PASS |
| 전체 검사 | 205개 테스트 package, license·vet·race·parity·progress·gofmt 최초 PASS |
| 판정 | `unresolved` 유지, 검증 동작 기록 2개 추가; 구현 `b8da817e`, 마감 `837ee16d` |

이 묶음에는 검사 실패나 수리가 없었습니다. native HTTP 200 전용 응답·typed model·Resource 병합·mutation receipt·raw Find controls·Python iterable/ID와 공통 미해결은 남아 있습니다.

## 완료 판정

프로젝트 완료를 주장하려면 다음 근거가 함께 필요합니다.

1. 두 고정 소스의 연산뿐 아니라 확인한 상속·descriptor·Resource 표면을 빠짐없이 추적하고, 미확정 범위를 공개합니다.
2. 각 API의 필수 입력, 생략/false/빈 값/zero 구분, library-owned builder, 기본값, query/header/body 확장, 결과와 오류를 실제 계약 테스트로 확인합니다.
3. 이름 조회, 중복, 미존재, pagination, cancellation, 삭제·상태 대기는 해당 리소스가 지원하는 동작에 맞게 검증합니다. 표면이 비슷한 단일 서비스의 테스트를 전체 서비스의 증거로 사용하지 않습니다.
4. 서비스 간 작업은 정상 흐름과 각 단계의 실패·부분 성공·재시도·대기·정리 정책을 검증합니다. 자동 삭제나 rollback은 문서화한 정책을 따라야 합니다.
5. 모든 서비스와 전체 사용 문서가 구현된 공개 API를 사용하고, 예제 빌드와 문서 링크를 확인합니다.
6. `supported`/`go_mapping` 항목마다 실제 증거가 있고, 요청 범위의 `unsupported`/`unresolved` 항목이 남아 있으면 전체 목표를 완료로 표시하지 않습니다.

microversion 자동 선택·협상과 요청 필드 capability 검증 역시 공통 SDK 계약입니다. [WithMicroversion](../connection_options.go)의 명시 선택 기능만으로 자동 협상까지 지원한다고 판정하지 않습니다. 이 대장의 각 조사 항목은 구현과 검증이 추가될 때 갱신해야 하며, 생성된 transport 함수 수는 완료율의 대체 지표로 사용하지 않습니다.

## 이전 검증 기록

[2026-10-18 분리 이력](history/sdk-support-ledger-2026-10-18.md)에 이전 구현·검사·Source 대조 기록을 보존합니다. 최초 실패, 수리 뒤 통과, 부분 검증과 실클라우드 미검증의 구분도 이력에 남겼습니다. 과거 quota·임시 로그 경로·반복된 파일 SHA는 현재 진행 지표에서 제외했으며 이력에는 보존합니다.
