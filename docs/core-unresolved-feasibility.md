# 핵심 서비스 미해결 연산의 구현 가능성 분류

이 문서는 2026-10-12 기준 핵심 서비스(Keystone, Nova, Placement, Neutron, Glance, Cinder, Barbican, Swift와 해당 cloud 계층 helper)에 남아 있는 미해결 판정 722개를 구현 가능 여부에 따라 나눈 결과입니다. 판정 JSON의 `remaining` 문장과 고정 Gophercloud v2.15.0, 고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` 소스를 서비스별로 다시 읽어 분류했습니다. 연산별 분류 결과는 [분류 데이터](core-unresolved-feasibility.json)에 있으며, 각 항목은 범주·작업 규모·공통 수정 이름·한 문장 설명을 가집니다.

**2026-10-13 상태 갱신:** 아래722개와 분류 데이터는 10-12 최초 조사 범위를 보존한 것입니다. 현재 남은 작업 목록으로 다시 해석하지 않습니다. 현재 판정과 대조하면 이 조사 중 Go 매핑363개·미지원133개·부분/미해결226개입니다. 직전 묶음19개에 이어 추가 사이클18개와 quota 반복 첫 사이클21개·두 번째 사이클14개·세 번째 사이클8개·네 번째 사이클4개를 매핑했으며, Project·Group·Neutron Extension Find의 raw fetch control은 남은 범위로 보존합니다. 최신 서비스별 수치와 다음 작업은 [구현 진행표](implementation-plan.md#현재-집계와-진행-중인-작업) 및 [판정 JSON](../api/sdk_reviews.json)을 기준으로 합니다. Python/Go 표현 차이나 upstream 결함만으로 추가 `unsupported` 판정을 확정하지 않습니다.

## 요약

| 구분 | 범주 | 개수 | 판단 |
|---|---|---:|---|
| 구현 가능 | 판정 재검토 | 72 | 같은 요청을 이미 보내고 있어 계약 테스트를 보강하면 `go_mapping`이 됩니다 |
| 구현 가능 | SDK 소유 호출 | 224 | 서버 API는 있지만 Gophercloud에 바인딩이 없습니다 |
| 구현 가능 | 공통 helper | 221 | find fallback, 로컬 Body 필터, 페이지 순회 같은 SDK 기능이 빠져 있습니다 |
| 구현 가능 | 입력 표현 | 26 | typed 입력이 Python이 보내는 null·false·빈 값을 표현하지 못합니다 |
| 구현 가능 | 응답 모델 | 23 | 요청은 같지만 응답 모델이 필드를 버리거나 다른 모양으로 읽습니다 |
| 구현 대상 제외 | 서버 API 폐기 | 130 | upstream에서 제거된 API라 `unsupported`로 기록했습니다 |
| 구현하지 않음 | Python 전용 | 15 | Go에 대응 개념이 없거나 Go식 대안이 이미 있습니다 |
| 구현하지 않음 | upstream 결함 | 11 | Python 또는 native 쪽 결함이라 그대로 재현하지 않습니다 |

구현 가능한 항목은 566개이고, 그중 작업 규모가 작은(S) 항목이 421개입니다. 서버 API 폐기 130개와 구현하지 않음 26개를 합친 156개는 `unsupported`로 기록하는 편이 정확합니다. 다만 upstream 결함 가운데 Cinder snapshot 대기 2개는 SDK 쪽에서 고칠 수 있으므로 구현 대상에 포함합니다.

서비스 묶음별 분포는 다음과 같습니다.

| 묶음 | 재검토 | SDK 호출 | helper | 입력 | 모델 | API 폐기 | Python 전용 | 결함 | 합계 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Keystone | 14 | 40 | 37 | 2 | 11 | 10 | 0 | 0 | 114 |
| Neutron | 7 | 126 | 115 | 8 | 7 | 30 | 0 | 0 | 293 |
| Cinder | 2 | 41 | 6 | 2 | 0 | 79 | 1 | 8 | 139 |
| Nova·Placement·Glance·Barbican·Swift와 cloud 나머지 | 49 | 17 | 63 | 14 | 5 | 11 | 14 | 3 | 176 |

## 구현 가능한 항목

### 판정 재검토 72개

남은 사유가 응답 캐시, descriptor 변환, seeded Resource 병합처럼 감사 수준의 항목뿐이고 Go 호출이 같은 요청과 같은 기본 결과를 만드는 경우입니다. Glance cache·metadef·task, Swift temp URL과 object 조회·삭제·대기, Nova 대기, Barbican quota 같은 예전 엄격 판정 49개가 가장 많습니다. Keystone은 cloud helper 12개와 `find_domain` 등 14개가 해당합니다. 이 helper들이 실제로 넘기는 find 인자는 query로 표현되는 값뿐이라 남은 사유가 도달하지 않습니다. Neutron quota 5개와 trunk·QoS policy 수정 2개, Cinder quota 기본값·수정 2개도 같은 경우입니다. 새 기능 없이 계약 테스트를 더하고 재판정하면 됩니다.

### SDK 소유 호출 224개

서버 API는 현재 OpenStack에 있지만 Gophercloud v2.15.0에 바인딩이 없는 경로입니다. 저장소에는 이미 Glance action, Barbican secret ACL, Neutron owned record처럼 SDK가 직접 요청을 만드는 선례가 있어 같은 방식으로 구현할 수 있습니다. 경로 묶음 단위로 작업하면 효율이 높습니다.

| 경로 묶음 | 개수 |
|---|---:|
| Neutron networking-sfc | 30 |
| Keystone OS-FEDERATION identity provider·protocol·service provider | 18 |
| Keystone OS-EP-FILTER endpoint group·project endpoint | 18 |
| Neutron flavor·service profile | 14 |
| Neutron metering label·rule | 12 |
| Neutron tap flow·tap service | 12 |
| Neutron local IP·association | 11 |
| Cinder group type·group spec | 11 |
| Cinder service action | 7 |
| Neutron QoS packet rate rule | 6 |
| Neutron NDP proxy | 6 |
| Neutron 보안 그룹 기본 statefulness | 6 |
| Neutron network segment range | 6 |
| Neutron conntrack helper, 기본 보안 그룹 rule, Cinder cluster | 각 5 |
| Keystone domain config, Neutron port binding, Cinder default type·group reset, Nova server migration·share attachment | 각 4 |
| 나머지 소규모 경로 21종 | 28 |

### 공통 helper 221개

한 번 만들면 여러 연산이 함께 해소되는 SDK 기능입니다. 개수가 많은 순서는 다음과 같습니다.

| 공통 수정 | 개수 | 내용 |
|---|---:|---|
| find fallback | 32 | ID 조회가 400·403·404이면 목록에서 ID나 이름으로 찾는 helper를 해당 collection에 연결합니다 |
| extension 확인 | 28 | cloud QoS·router helper가 먼저 보내는 `GET /v2.0/extensions` 확인을 SDK가 대신합니다 |
| cloud 목록 필터 | 22 | cloud 계층 `_filter_list`의 glob·dict·JMESPath 필터를 공개 helper로 제공합니다 |
| 로컬 Body 필터 | 21 | `resource.WithFilter` 계열을 Keystone 등 아직 바인딩이 없는 collection에 연결합니다 |
| `fields` 조회 | 16 | `fields` query를 보내는 조회 함수를 추가합니다 |
| marker 반복과 로컬 필터 | 12 | Keystone 목록에 `limit`·`marker` 반복 요청과 로컬 Body 필터를 함께 붙입니다 |
| find query 전달 | 9 | find 단계에 Python의 추가 query를 전달합니다 |
| 대기 | 8 | Placement inventory·trait, Barbican quota 등에 대기 helper를 추가합니다 |
| marker 반복 | 7 | Python이 `limit`을 받으면 빈 페이지까지 `marker`로 반복하는 순회를 collection에 추가합니다 |
| 기타 | 66 | 서버 주소 확장, cloud floating IP·firewall·object workflow, 상태 코드 무시 등 |

### 입력 표현 26개와 응답 모델 23개

입력 표현은 typed 구조체가 `omitempty`나 필수 표시 때문에 null, false, 빈 문자열, 빈 목록, 반복 query key를 보내지 못하는 경우입니다. Nova evacuate의 `onSharedStorage`, service `forced_down=false`, aggregate의 availability zone null, BGP speaker의 bool 필드가 대표적입니다. SDK 소유 옵션이나 요청 builder로 해결합니다.

응답 모델은 요청이 같아도 결과에서 필드가 빠지는 경우입니다. Keystone domain의 `options` 7개, federation mapping의 `schema_version`과 원본 rule 4개, Neutron network·port의 확장 속성 7개가 큰 묶음입니다. SDK 소유 모델이나 원문 보존 결과로 해결합니다.

## 구현은 가능하지만 대상 서버가 없는 항목 130개

| 폐기된 API | 개수 | 근거 |
|---|---:|---|
| Cinder v2 | 79 | Cinder Xena(19.0.0)에서 v2 API 전체가 제거되었습니다 |
| Neutron LBaaS v2 | 30 | neutron-lbaas 프로젝트가 retire되었고 Octavia가 대체했습니다 |
| Glance v1 | 11 | Glance v1 API가 제거되었습니다 |
| Keystone v2.0 | 10 | Keystone Queens에서 v2.0 API가 제거되었습니다 |

요청 자체는 SDK 소유 호출로 만들 수 있습니다. 그러나 지원되는 OpenStack 배포가 이 경로를 제공하지 않으므로 계약을 검증할 대상이 없습니다. 그래서 2026-10-12에 이 항목들을 구현 대상에서 제외했습니다. 판정 JSON에는 `unsupported`와 폐기 근거를 기록했고, Cinder v2의 결함·Python 전용 항목 3개(`transfers`, native `backups.Update`, `update_type`)도 같은 v2 API라 함께 제외해 모두 133개입니다. 진행 집계의 완료율은 `unsupported`를 뺀 구현 대상 선언을 분모로 계산합니다. 이미 `go_mapping`으로 판정한 Keystone v2·Cinder v2 연산은 Gophercloud 바인딩이 있어 기존 판정을 유지합니다.

## 구현하지 않는 항목 26개

### Python 전용 15개

`pprint`, `pformat`, keystoneauth `session`, `close`, 내부 업로드 helper `FileSegment`의 네 메서드, 런타임 서비스 등록 `add_service`는 Go에 대응 개념이 없어 `unsupported`가 맞습니다. 원본 `service_catalog` 객체는 typed endpoint 조회로 대신할 수 있습니다. Swift `get_object`·`stream_object`는 이미 있는 `io.Writer` 다운로드와 `io.ReadCloser` 결과를 Go식 대안으로 문서화합니다. `set_global_request_id`와 `global_request`는 Python에서도 Connection 단위 상태이므로 Connection 옵션으로 `X-OpenStack-Request-ID`를 붙이는 방식이면 구현할 수 있어 공통 helper로 옮기는 것을 검토합니다. Cinder v2 `update_type`은 Python이 요청 없이 항상 예외를 내므로 대응할 동작이 없습니다.

### upstream 결함 11개

Cinder v3 backup metadata 4개는 Python이 Cinder에 등록되지 않은 `/backups/{id}/metadata` 경로를 호출합니다. Python v2 `transfers`는 v2에 없는 `/volume-transfers` 경로를 씁니다. native v2 `backups.Update`는 v2 서버 경로가 없고 본문에 envelope도 없습니다. 이 6개는 재현하지 않고 `unsupported`로 기록합니다. Barbican `update_container`, `update_order`, `update_secret`은 서버가 Python 요청을 받아들이지 않는 것으로 보입니다. 다만 고정 Barbican 소스로 확인한 판단이 아니므로 확인 뒤에 `unsupported`로 바꿉니다.

native Cinder `snapshots.WaitForStatus` v2·v3 2개는 envelope 없는 응답에서 nil 참조로 panic합니다. 생성 facade에 nil 결과 검사를 넣으면 해결되므로 구현 대상입니다.

## 정해야 할 기준

분류 중에 묶음 사이의 판정이 엇갈린 지점을 찾았습니다.

- 변경할 값이 없을 때 Python은 수정 요청을 보내지 않지만 Go는 항상 보냅니다. `update_tap_mirror`는 이를 차이로 기록하고 대응으로 판정했고, cloud `update_port`는 같은 이유로 미해결입니다. 한쪽 기준으로 맞춰야 합니다.
- Neutron quota 상세 조회는 native가 `/details.json`을, Python이 `/details`를 씁니다. Neutron이 둘을 같게 처리하므로 같은 요청으로 보았습니다.

## 구현 순서

효과가 큰 순서로 진행합니다. 먼저 판정 재검토 72개와 Cinder snapshot 대기 2개처럼 새 기능 없이 끝나는 항목을 처리합니다. 다음으로 find fallback, 로컬 Body 필터, `fields` 조회, marker 반복, 대기처럼 여러 서비스에 걸치는 공통 helper를 만듭니다. 그 뒤 SDK 소유 호출을 경로 묶음별로 병렬 작성합니다. 구현하지 않는 나머지 항목은 기준을 정한 뒤 판정 JSON에 `unsupported`로 기록합니다.
