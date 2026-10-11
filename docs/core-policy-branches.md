# 이번 핵심 API 묶음의 권한 분기

고정 SDK catalog의 API 개수와 서버 권한은 별개입니다. 이번 구현42개는 핵심 user/admin 합산 증가이며 user 단계 전체 완료를 의미하지 않습니다. 아래 정책 기준은 OpenStack2025.2의 Cinder27.0.0·Nova32.0.0·Neutron27.0.0 stock 소스입니다. 배포의 policy override와 리소스 소유권은 실제 서버가 판단합니다.

| SDK 호출 | 일반 프로젝트 경로 | 추가 권한 또는 용도 |
|---|---|---|
| Cinder attachment `CreateRecord/ListRecords` | 생성은 project member, 목록은 project 범위 | pinned Python 문서의 서비스 내부용 연산; 일반 user 흐름 완료로 분류하지 않음 |
| Cinder `InitializeWithConnector/TerminateWithConnector` | project member | 저수준 connector 연산; stock admin 전용으로 분류하지 않음 |
| Cinder type `FindRecord` | project reader | 기본 type 조회 정책 |
| Cinder encryption 생성·수정, QoS `FindRecord` | 없음 | stock admin API3개 |
| Nova volume attachment `UpdateRecord` | 2.85 이상·같은 volume ID로 delete_on_termination 수정 | 2.85 미만 또는 volume ID 교체는 service/admin; Cinder migration/swap 분기 |
| Nova `UnshelveWithOptions` | host를 지정하지 않는 경로 | 비어 있지 않은 host 지정은 추가 admin 정책 |
| Nova server group `FindRecord` | 현재 project 검색 | all_projects fallback은 admin 정책 |
| Nova 직접 fixed/floating IP action | 선택 버전2.43 이하의 프로젝트 연산 | 2.44에서 해당 action 제거; 다른 inventory proxy의2.36 제거와 구분 |
| Neutron port forwarding6개 동작·12개 Python 이름 | project member/reader 경로 | 부모 floating IP의 project와 서버 정책 적용 |

정책 감사는 Cinder/Nova/Neutron37개 선언을 포함합니다. 이번 신규34개와 기존 Transfer3개 계약 보강을 합친 범위이며, 분류는 user29·조건부user/admin/service3·admin3·Source 내부용2입니다. 이37개를 신규 API 증가량 또는 순수 user 완료량으로 더하지 않습니다. Swift1·Credential1·Trust5·cloud backup1은 별도 구현/판정8개입니다.

공식 근거:

- [Cinder attachment 정책](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/attachments.py), [volume action 정책](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/volume_actions.py), [기본 정책](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/base.py), [2025.2 정책 안내](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html).
- [Nova attachment controller](https://github.com/openstack/nova/blob/32.0.0/nova/api/openstack/compute/volumes.py), [attachment 정책](https://github.com/openstack/nova/blob/32.0.0/nova/policies/attach_volume.py), [기본 정책](https://github.com/openstack/nova/blob/32.0.0/nova/policies/base.py), [2025.2 정책 안내](https://docs.openstack.org/nova/2025.2/configuration/policy.html).
- [Neutron2025.2 정책 안내](https://docs.openstack.org/neutron/2025.2/configuration/policy.html).

정확한 operation ID·Source fingerprint·45개 정책 binding·공식 소스 SHA·행 근거는 이번 실행의 `/private/tmp/go-openstacksdk-resume-policy-classification.json`에 있습니다. 정책 AST 대조는 정적 감사이며 실제 인증된 OpenStack 호출을 실행한 결과는 아닙니다.
