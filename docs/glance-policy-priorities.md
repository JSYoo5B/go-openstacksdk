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

`metadef_objects`와 `metadef_namespaces`의 owned 목록2개는 핵심 user 단계에서 완료했고 현재 전체269/3,362입니다. object getter와 schema getter는 이미 완료했으므로 재집계하지 않습니다. object의 생성·수정·개별/전체 삭제4개는 핵심 admin 단계로 옮깁니다. 다른 metadata 쓰기도 같은 단계에서 처리하고, 다음 후보의 권한 근거가 부족하면 먼저 서버 rule을 확인합니다.

member owned 목록·검색2개를 완료했습니다. 이 범위는 고정 [Glance image 조회 정책](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L251-L278)의 `get_member`·`get_members`가 admin 또는 project/shared member reader를 허용하므로 핵심 user에 배치합니다. metadata 생성·수정·삭제의 admin 순서는 유지합니다.

[구현 계획](implementation-plan.md)과 [지원 판정대장](sdk-support-ledger.md)에서 현재 수치와 검증 범위를 확인합니다.

## 다음 import info와 store 조회

같은 고정 Glance revision의 실제 구현은 `api/v2/info.py`가 아닌 `api/v2/discovery.py`입니다. [import 정보41–51행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L41-L51)과 [기본 stores53–93행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L53-L93)은 별도 per-operation policy 호출이 없어 핵심 user 후보로 분류합니다. 이는 controller의 admin 제한 부재에서 도출한 분류이며 인증이나 실제 접근 결과를 보장하지 않습니다.

상세 `/info/stores/detail`은 [controller165–193행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L165-L193)에서 [Discovery API policy125–133행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L125-L133)의 `stores_info_detail`을 호출합니다. [정책21–33행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/discovery.py#L21-L33)은 `ADMIN_OR_SERVICE_ROLE`과 project scope를 요구합니다. 따라서 `stores(details=True)`는 핵심 admin/service 분기입니다.

읽은 controller/discovery policy/API policy의 SHA256은 각각 `11f24d5a5d592f27234fb6d138157e97c27b755a03ead5d8796d350a39593b5b`·`b06d81a0834f8b7392579b929641806f766500edd1f163ca120cf96f25c8a1f1`·`6abba4e3adfc602c8a199a2d15187e21b8bdaa3109421a7558a51c5445a86940`입니다. 두 후보의 지원 판정은 아직 올리지 않았습니다. stores는 같은 선언의 상세 경로도 구현·검증한 뒤 전체 mapping으로 판정하며, 실제 policy override와 multi-backend 상태는 서버가 판단합니다.

## 이미지 조회·목록·검색의 기본 정책

같은 고정 Glance revision의 [image 정책52–78행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L52-L78)은 `get_image`에 `ADMIN_OR_PROJECT_READER_GET_IMAGE`, `get_images`에 `ADMIN_OR_PROJECT_READER`를 적용하며 두 rule 모두 project scope입니다. [공통 정책22–67행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/base.py#L22-L67)의 project reader와 image member/public/community/shared 조건을 근거로 이미지 조회·목록과 이를 사용하는 Find를 핵심 user 범위에 배치합니다. 관리자 경로도 허용하지만 admin 전용 작업으로 분류하지 않습니다.

[controller549–566행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L549-L566)은 현재 프로젝트에 대한 `get_images` 검사 후 각 행의 `get_image` 정책으로 목록을 다시 제한합니다. [직접 show587–598행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L587-L598)도 `get_image`를 호출하고 [API policy262–266행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L262-L266)이 같은 rule을 enforce합니다. 숨김 이미지 query가 이 권한 검사를 우회하지 않습니다. 실제 role·소유권·visibility와 policy override는 배포 서버가 결정하며 SDK role preflight 또는 실제 cloud 접근을 검증한 것으로 확대하지 않습니다.

읽은 image 정책과 controller SHA256은 각각 `e082ee840732842af1cc9c8811cb956466a6972149268a2f38f1874ef1dae38a`·`685a1a176203d3fcd92580106a2ffeee1911234c0f99db2ad04f8d253499e2cc`입니다. [이미지 레코드 가이드](../image/image-records.md)는 이 분류와 별도로 literal GET·Source descriptor·페이지·검색 계약 및 Go 확장을 설명합니다. 이 정책 확인만으로 API 구현 완료 수를 변경하지 않습니다.
