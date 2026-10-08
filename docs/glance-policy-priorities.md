# Glance user/admin 구현 우선순위

2026-10-09에 metadata 쓰기 API의 분류를 교정했습니다. Python SDK에 admin 전용 설명이 없다는 사실만으로 user 단계에 넣었던 판단은 잘못됐습니다. 교정 당시 구현·테스트와 API 완료 수265/3,362는 보존하고, 이후 구현 순서를 서버 기본 정책에 맞춥니다.

## 확인한 기준

Glance server revision `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [metadata 정책](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/metadef.py)과 [공통 role 정책](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/base.py)을 직접 확인했습니다. 비교용 SDK inventory의 source pin은 변경하지 않습니다. 정책 파일 SHA256은 각각 `bc557a604741274df7bf0cde7e1be599de4450faf2c3e66d81d358b80c99c6ff`·`d30a1d9cdaba689c2aa7a5c37bae5879ea2071443548d754724de89a3954566a`입니다.

metadata 쓰기의 `METADEF_ADMIN`은 `rule:metadef_admin` → `base.ADMIN` → `rule:context_is_admin` → `role:admin`으로 이어집니다. 아래 분류는 이 기본 정책을 기준으로 하며 실제 배포의 policy override와 가시성·소유권은 서버가 판단합니다. SDK에서 role을 검사하거나 server policy를 재구현하지 않습니다.

| 범위 | 서버 기본 정책 | 구현 단계 |
|---|---|---|
| Namespace 목록·조회 | admin 또는 project reader | 핵심 user |
| Object·property·tag·association 목록·조회 | admin 또는 접근 가능한 namespace의 project reader | 핵심 user |
| Namespace/object/property/tag 생성·수정·삭제 | metadef_admin | 핵심 admin |
| Resource type association 생성·삭제 | metadef_admin | 핵심 admin |

association의 정확한 쓰기 rule은 `add_metadef_resource_type_association`·`remove_metadef_resource_type_association`입니다. property는 `add_metadef_property`·`modify_metadef_property`·`remove_metadef_property`, object는 `add_metadef_object`·`modify_metadef_object`·`delete_metadef_object`입니다. 각 rule의 check_str와 reader 조회 rule을 AST로 읽어 위 관계를 확인했습니다. 실제 인증 cloud 테스트를 했다는 뜻은 아닙니다.

## 완료 작업과 다음 순서

완료한 property 생성·수정·삭제와 association 생성·삭제는 admin 범위의 구현으로 유지합니다. property 목록·조회와 resource type/association 목록은 user 범위입니다. 기존 서비스별 완료 수는 user/admin 합산이므로 숫자를 감소시키거나 다시 세지 않습니다.

`metadef_objects`와 `metadef_namespaces`의 owned 목록2개는 핵심 user 단계에서 완료했고 현재 전체267/3,362입니다. object getter와 schema getter는 이미 완료했으므로 재집계하지 않습니다. object의 생성·수정·개별/전체 삭제4개는 핵심 admin 단계로 옮깁니다. 다른 metadata 쓰기도 같은 단계에서 처리하고, 다음 후보의 권한 근거가 부족하면 먼저 서버 rule을 확인합니다.

다음 member 목록·검색은 고정 [Glance image 조회 정책](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L251-L278)의 `get_member`·`get_members`가 admin 또는 project/shared member reader를 허용하므로 핵심 user에 배치합니다. metadata 생성·수정·삭제의 admin 순서는 유지합니다.

[구현 계획](implementation-plan.md)과 [지원 판정대장](sdk-support-ledger.md)에서 현재 수치와 검증 범위를 확인합니다.
