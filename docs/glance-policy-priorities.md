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

## 이미지 태그 추가·삭제의 기본 정책

여기의 image tag는 앞선 `metadef_tags`와 별도 API입니다. 같은 고정 Glance revision의 [tag controller47–54·75–85행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_tags.py#L47-L85)은 PUT/DELETE 모두 `modify_image`를 검사합니다. [image 정책81–83행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L81-L83)의 rule은 project scope의 `ADMIN_OR_PROJECT_MEMBER`이며 [API policy288–293행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L288-L293)이 enforce합니다. 이 기본 정책에서 접근 가능한 project-owned 이미지의 태그 추가·삭제를 핵심 user 단위에 배치합니다. 실제 소유권·가시성·policy override와 서버 제한은 서버가 판단합니다.

읽은 tag controller·image 정책·API policy SHA256은 각각 `043179d6db6c35cd770fb98beea1ef6aacf9f9dfb6c121e34df491975a3b1fd6`·`e082ee840732842af1cc9c8811cb956466a6972149268a2f38f1874ef1dae38a`·`6abba4e3adfc602c8a199a2d15187e21b8bdaa3109421a7558a51c5445a86940`입니다. `/private/tmp/go-openstacksdk-image-record-tags-policy-audit.json`에 원문 범위와 hash를 기록했습니다. 고정 openstacksdk의 `add_tag/remove_tag`는 아직 unresolved이며, 기존 endpoint 호출에 supplied Image의 성공 후 local tags 갱신을 더하는 owned profile이 다음 작업입니다. 정책 확인만으로 완료 수를 늘리지 않습니다.


## 이미지 삭제의 전체/store 기본 정책

같은 고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`에서 [전체 route425–429행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L425-L429)은 `ImagesController.delete`로 이어집니다. [controller874–876행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L874-L876)의 `delete_image()`와 [API policy268–273행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L268-L273)은 [delete_image 정책38–51행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L38-L51)의 project scope `ADMIN_OR_PROJECT_MEMBER`를 적용하므로 기본 전체 삭제는 핵심 user 경로입니다.

특정 store 삭제는 핵심 admin 분기입니다. [store route442–446행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L442-L446)→[controller769–773행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L769-L773)의 `delete_locations()`→[API policy224–229행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L224-L229)→[delete_image_location 정책159–172행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L159-L172)은 project scope `ADMIN`을 요구합니다. 실제 policy override·소유권·backend·이미지 상태/location 제한은 서버가 판단합니다.

AST와 SHA256으로 네 원문을 확인했습니다. router SHA256은 `4ebf33ca94f39509252e668e7a9744ce5d05b2cc275f9a09dd67fda0a6098731`이며 controller·API policy·image 정책은 위에 기록한 동일 hash입니다. 공개 Python `delete_image`는 whole/store를 포함하는 하나의 operation이므로 store 분기를 별도 API로 재집계하지 않습니다. [owned 삭제 가이드](../image/image-record-delete.md)와 [native 삭제 가이드](../image/v2/images/delete.md)는 서로 다른 입력·반환·status 경계를 설명합니다. 현재 진행 수치와 지원 분류는 [구현 계획](implementation-plan.md)·[지원 판정대장](sdk-support-ledger.md)의 최신 업데이트를 기준으로 확인합니다.


## 이미지 비활성화·재활성화의 기본 정책

같은 고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [action routes453–462행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L453-L462)은 POST deactivate/reactivate를 `ImageActionsController`로 연결합니다. [controller47–102행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_actions.py#L47-L102)의 `deactivate_image()`·`reactivate_image()` → [API policy295–307행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L295-L307) → [deactivate/reactivate 정책301–328행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L301-L328)은 두 작업 모두 project scope의 `ADMIN_OR_PROJECT_MEMBER`를 적용합니다. 따라서 기본 정책에서 핵심 user 작업으로 배치합니다. 비활성화의 active·재활성화의 deactivated 상태 검사와 실제 role·소유권·policy override·backend 제한은 서버가 판단합니다.

AST와 SHA256으로 route·controller·policy를 확인했으며 action controller hash는 `96d4b2da6975a5f88f10ec3e3eeb7217478b5c34b2f6f70ad0584fc728bcfc48`입니다. router·API policy·image 정책 hash는 위의 동일 pin 값이며 `/private/tmp/go-openstacksdk-image-core-user-policy-audit.json`에 원문 범위와 hash를 기록했습니다. [owned 상태 action 가이드](../image/image-record-actions.md)는 권한 분류와 별도로 Python 기본 status 처리·Go native 오류 및 local Record/ACK 계약을 설명합니다. 현재 구현 수치와 판정은 [구현 계획](implementation-plan.md)·[지원 판정대장](sdk-support-ledger.md)의 최신 업데이트를 기준으로 확인합니다.


## 이미지 property helper의 일반·admin 필드 범위

`UpdateImagePropertiesRecord`는 [고정 API policy213–229행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L213-L229)의 `update_property` → `modify_image`와 [modify 정책80–93행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)의 project scope `ADMIN_OR_PROJECT_MEMBER`를 근거로 ordinary property 갱신을 핵심 user에 배치합니다. 같은 helper의 public visibility 변경은 [publicize 정책94–103행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L94-L103)의 admin 분기이며, retained/pending body에서 location removal이 발생하면 [delete location 정책159–172행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L159-L172)의 admin 분기입니다. 값 변환·전체 목록 기반 kernel/ramdisk 해석·Source bool/PATCH 관계는 [속성 helper 가이드](../image/image-record-properties.md)에 설명합니다. 실제 소유권·property protection·readonly/schema·policy override·서버 제한은 서버가 판단하며 client에서 role을 검사하지 않습니다.


## 이미지 staging의 기본 정책

같은 고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [stage route486–489행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L486-L489)은 `PUT /images/{image_id}/stage`를 `ImageDataController.stage`로 연결합니다. [controller330–360행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_data.py#L330-L360)은 staging에 독립 policy rule이 없음을 설명하고 repository의 이미지 접근과 `modify_image()`를 검사합니다. [API policy288–293행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L288-L293) → [modify 정책80–93행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)은 project scope `ADMIN_OR_PROJECT_MEMBER`이므로 ordinary staging을 핵심 user 범위에 배치합니다. 별도의 admin-only stage 옵션은 없으며 실제 소유권·상태·quota·backend·policy override는 서버가 판단합니다.

원본 SHA/AST와 full public graph는 `/private/tmp/go-openstacksdk-image-data-workflows-source-audit.json`에 고정했습니다. [owned staging 가이드](../image/image-record-stage.md)는 Source의 명시적 queued Record·자동 total-file size·header-only 응답·필수 fetch와 Go의 filename close·borrowed data·opaque200..399/부분 결과를 설명합니다. 기존 native `StageImage/StageKnownImage` 구현과 같은 public graph로 판정하지 않습니다. policy 확인이나 새 문서만으로 완료 수를 올리지 않으며 현재 판정은 [구현 계획](implementation-plan.md)·[지원 판정대장](sdk-support-ledger.md)의 최종 검증을 기준으로 합니다.


## 이미지 import의 ordinary·copy method 기본 정책

같은 고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [import controller317–390행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L317-L390)은 method별 상태·형식 검사 뒤 ordinary `glance-direct`·`web-download`·`glance-download`에 `modify_image()`, `copy-image`에 `copy_image()`를 적용합니다. [API policy288–312행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L288-L312)과 [modify 정책80–93행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)의 project `ADMIN_OR_PROJECT_MEMBER`를 근거로 ordinary3 methods는 핵심 user입니다. [copy 정책330–343행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L330-L343)의 `copy_image`는 project `ADMIN`을 요구하므로 핵심 admin 분기입니다.

`ImportImageRecord`는 같은 공개 helper에서 네 method·store 조합을 처리하며 copy 분기를 별도 operation으로 세지 않습니다. client에서 role·status를 검사하지 않고 server native 오류를 보존합니다. 실제 소유권·method 활성화·backend/store 가용성·quota·URL authorization·policy override·비동기 storage 성공은 서버가 판단합니다. [owned import 비교·독립 main](../image/image-record-import.md)에 초기 GET 없는 private Record·Python truthy formats·singular JSON/header·plural raw ID와 native typed profile의 차이를 설명합니다. Source·policy 근거는 `/private/tmp/go-openstacksdk-image-record-import-source-audit.json`에 고정하며 문서나 policy 확인만으로 지원 수를 올리지 않습니다. 현재 판정은 [구현 계획](implementation-plan.md)·[지원 판정대장](sdk-support-ledger.md)의 최종 gate를 기준으로 합니다.


## 이미지 생성·upload의 user·admin metadata 분기

같은 고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`에서 [add_image 정책24–37행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L24-L37)은 `ADMIN_OR_PROJECT_MEMBER_CREATE_IMAGE`, [upload_image 정책144–157행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L144-L157)은 project scope `ADMIN_OR_PROJECT_MEMBER`를 요구합니다. 따라서 ordinary metadata POST·file PUT는 핵심 user 작업입니다. [base 정책72–74행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/base.py#L72-L74)의 create 조건은 project member의 owner가 인증 프로젝트와 일치해야 하므로 다른 프로젝트 owner는 같은 공개 입력의 admin 분기입니다.

[API add_image243–260행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L243-L260)은 visibility 검사를 함께 적용하고, [visibility helper111–115행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/policy.py#L111-L115)은 public/community에 각각 publicize/communitize를 적용합니다. [publicize·communitize 정책94–117행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L94-L117)은 public에 project `ADMIN`, community에 `ADMIN_OR_PROJECT_MEMBER`를 요구합니다. 같은 `UploadImageRecord`의 metadata 속성으로 전달하며 branch를 별도 operation으로 세지 않습니다.

[owned 업로드 가이드](../image/image-record-upload.md)는 deprecated Python `upload_image`의 private create→binary PUT 전체 graph, 명시 형식 precheck·raw 속성 overlay·signed size와 GET 없는 반환을 설명합니다. client role·enum·상태 검사를 추가하지 않으며 실제 소유권·schema·quota·backend·policy override는 서버가 판단합니다. Source SHA/AST는 `/private/tmp/go-openstacksdk-image-record-upload-source-audit.json`에 고정했습니다. 현대 공개 `create_image` 전체 분기는 별도 미해결 범위이고, 지원 수치는 [구현 계획](implementation-plan.md)·[지원 판정대장](sdk-support-ledger.md)의 실제 최종 검증을 기준으로 확인합니다.


## 이미지 다운로드의 user·deactivated admin 분기

같은 고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`에서 [get_image 정책53–65행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L53-L65)은 `ADMIN_OR_PROJECT_READER_GET_IMAGE`, [download_image·download_from_store 정책119–142행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L119-L142)은 project scope `ADMIN_OR_PROJECT_MEMBER_DOWNLOAD_IMAGE`를 사용합니다. [base 정책64–71행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/base.py#L64-L71)은 owner/member·public·community·shared 및 admin 분기를 포함합니다. 일반적인 metadata 조회·binary 다운로드는 핵심 user 작업입니다.

[ImageDataController.download479–510행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_data.py#L479-L510)은 deactivated 이미지를 non-admin에게 거부하고 admin은 정상 download policy로 진행합니다. 같은 공개 helper의 admin 분기이며 별도 operation으로 세지 않습니다. client가 상태·role gate를 추가하지 않으므로 실제 policy override와 서버 상태가 결과를 결정합니다.

[prefer deserializer598–612행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_data.py#L598-L612)은 cache 경로의 preference 우회를 설명하고 comma 분리·whitespace/empty 제거를 수행합니다. 이후 controller가 backend 설정·scheme과 location을 검증합니다. client의 ordered preference 값 전달이 backend 순서·cache 우회·storage 성공을 보장하지 않습니다.

[owned 다운로드 가이드](../image/image-record-download.md)는 lower Python `download_image`의 필수 metadata fetch→binary GET→hash 선택과 output/stream/memory 전체 분기를 설명합니다. Source SHA/AST는 `/private/tmp/go-openstacksdk-image-record-download-source-audit.json`에 고정하며 별도 cloud wrapper·현대 공개 create와 공통 SDK-R1/C1/S1은 후속 범위입니다. 정책 확인이나 새 문서만으로 지원 수를 올리지 않습니다. 실제 판정은 [구현 계획](implementation-plan.md)·[지원 판정대장](sdk-support-ledger.md)의 최종 gate를 기준으로 합니다.
