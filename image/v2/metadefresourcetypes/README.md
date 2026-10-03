# Metadef resource types와 namespace association

`image.Service.API.MetadefResourceTypes`는 global resource type 목록과 namespace에 연결된 association을 구분합니다. 고정 Python의 네 proxy와 같은 네 서버 경로를 다루며 개별 Get·Update·Find·global Delete를 추가하지 않습니다.

| 사용 범위 | 실제 요청 | 성공 응답 |
|---|---|---|
| global List/All | GET `/metadefs/resource_types` | 200, `resource_types` array |
| scoped List/All | GET namespace `/resource_types` | 200, `resource_type_associations` array |
| scoped Create | POST namespace `/resource_types` | 201, association object |
| scoped Delete | DELETE namespace `/resource_types/{name}` | 204, opaque ACK |

[공식 API](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#metadata-definition-resource-types)와 [고정 controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py)는 association 생성 시 없는 global type을 서버에서 생성할 수 있음을 보여줍니다. Association 삭제는 global type을 지우지 않습니다. 이 연결 자체가 image·volume·flavor의 실제 metadata를 변경하지는 않습니다.

## Go 사용

아래 함수는 이미 구성한 image Service를 받습니다. List와 All은 각각 새 finite GET을 수행합니다. 반환 evidence에는 완료된 단계의 결과가 남고, 실제204 handling 실패에서도 Delete 결과와 오류를 함께 보존합니다. 예제의 Delete는 explicit false로 missing을 오류로 처리합니다.

```go
package example

import (
    "context"

    "gophercloudsdk/image"
    types "gophercloudsdk/image/v2/metadefresourcetypes"
)

type AssociationEvidence struct {
    GlobalRows []*types.ResourceType
    GlobalAll  []*types.ResourceType
    Created    *types.Association
    Rows       []*types.Association
    All        []*types.Association
    Deleted    *types.Acknowledgement
}

func manageResourceTypeAssociation(
    ctx context.Context,
    service *image.Service,
    namespace, resourceType, prefix, target string,
) (*AssociationEvidence, error) {
    evidence := &AssociationEvidence{}
    api := service.API.MetadefResourceTypes
    listOptions := []types.ListOption{
        types.WithListOpts(types.ListOpts{MaxItems: 20}),
        types.WithListHeader("X-Example", "resource-types"),
        types.WithListHeaders(map[string]string{"X-Request-Group": "example"}),
        types.WithListMaxItems(20),
    }
    for row, err := range api.List(ctx, listOptions...) {
        if err != nil {
            return evidence, err
        }
        evidence.GlobalRows = append(evidence.GlobalRows, row)
    }
    var err error
    evidence.GlobalAll, err = api.All(ctx, listOptions...)
    if err != nil {
        return evidence, err
    }
    scope, err := api.InNamespace(ctx, namespace)
    if err != nil {
        return evidence, err
    }
    evidence.Created, err = scope.Create(ctx, resourceType,
        types.WithCreateOpts(types.CreateOpts{}),
        types.WithCreateHeader("X-Example", "association"),
        types.WithCreateHeaders(map[string]string{"X-Request-Group": "example"}),
        types.WithCreatePrefix(prefix),
        types.WithCreatePropertiesTarget(target),
    )
    if err != nil {
        return evidence, err
    }
    for row, err := range scope.List(ctx, listOptions...) {
        if err != nil {
            return evidence, err
        }
        evidence.Rows = append(evidence.Rows, row)
    }
    evidence.All, err = scope.All(ctx, listOptions...)
    if err != nil {
        return evidence, err
    }
    evidence.Deleted, err = scope.Delete(ctx, resourceType,
        types.WithDeleteOpts(types.DeleteOpts{}),
        types.WithDeleteHeader("X-Example", "association"),
        types.WithDeleteHeaders(map[string]string{"X-Request-Group": "example"}),
        types.WithDeleteIgnoreMissing(false),
    )
    return evidence, err
}
```

고정 Python의 실제 네 proxy를 비교하면 다음과 같습니다. 서버가 finite response를 반환하므로 비교 예제는 두 list에 `paginated=False`를 명시합니다. Go는 서버 query·client filter·generic pagination을 합성하지 않습니다.

```python
def resource_type_association(conn, namespace, resource_type, prefix, target):
    global_types = list(conn.image.metadef_resource_types(paginated=False))
    created = conn.image.create_metadef_resource_type_association(
        namespace,
        name=resource_type,
        prefix=prefix,
        properties_target=target,
    )
    associations = list(
        conn.image.metadef_resource_type_associations(namespace, paginated=False)
    )
    conn.image.delete_metadef_resource_type_association(
        resource_type, namespace, ignore_missing=False
    )
    return global_types, created, associations
```

[실제 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1696-L1786)의 parent 인자는 `Resource._get_id`로 처리하며 parent Name 검색을 하지 않습니다. [두 Resource class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py)는 같은 파일에 있고 name을 alternate ID로 사용합니다. Create는 mutable Resource를 반환하고 Delete proxy는 결과를 버려 None을 반환합니다. Resource 재사용·dirty fields·adapter/session·microversion·generic list query와 Body filter는 Go의 passive DTO·literal scope와 별도 계약입니다.

## 입력과 concrete 옵션

`InNamespace`는 HTTP 없이 parent와 원래 API/client/provider·type·endpoint·ServiceURL·microversion을 scope 수명 동안 고정합니다. context는 저장하지 않습니다. namespace와 selected child는 nonempty valid UTF-8, 최대80rune입니다. colon·Unicode·space를 보존하고 path component를 한 번 escape하며 정확한 `.`·`..`, slash·backslash·percent·query·fragment 문자와 ASCII control/DEL은 HTTP 전에 거부합니다. trim·Ref·UUID 해석·lookup은 없습니다.

Create body는 항상 name을 포함합니다. Prefix·PropertiesTarget pointer가 nil이면 생략하고 explicit empty string은 보냅니다. valid UTF-8 JSON string, 최대80rune만 검사하며 colon separator·trim·경로 규칙을 붙이지 않습니다. control·multiline도 JSON scalar로 처리합니다. 고정 [schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py#L263-L317)는 optional string의80 제한을 적용하고 read-only dates·unknown request fields는 서버에서 거부합니다. 임의 raw 속성·schema discovery·parent GET·global type precreate·read-modify-write·rollback을 수행하지 않습니다.

Create·Delete·List는 concrete option family 세 개이며 두 list는 같은 ListOption을 사용합니다. FullOpts는 전체 교체, 순서는 last wins입니다. pointer·headers map·option slice·helper factory·callback config를 독립 복사합니다. callback은 요청 또는 iteration마다 한 번 실행하고 nil/error callback·invalid input·소유 header 충돌은 HTTP 전에 실패합니다. 요청마다 최신 유효 ordinary source headers를 capture하고 원래 provider의 live authentication을 사용합니다. Global API list는 iteration마다 source를 capture하므로 NamespaceScope의 수명 고정과 구분합니다.

## finite 목록과 passive 모델

두 목록은 lazy one GET이며 body·query가 없습니다. 실제 [index·show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py#L49-L119)는 limit·marker·sort를 읽지 않고 serializer는 next·first·Link를 만들지 않습니다. wire 순서·반복을 보존하며 DB 정렬을 약속하지 않습니다. `MaxItems=0`은 로컬 cap 없음, 양수는 소비 수 제한, 음수는 preflight 오류입니다. 전체 UTF-8·JSON과 required nonnull array를 먼저 검사한 뒤 소비하는 row만 strict decode합니다. cap·break는 미소비 malformed row·불필요한 HTTP를 건너뜁니다. All은 빈 성공에서 nonnil empty slice, 오류에서 부분 slice를 버립니다. client filter·paging fallback·link follow·cache를 합성하지 않습니다.

ResourceType은 exact canonical optional nullable Name·CreatedAt·UpdatedAt string pointer를 가지고 Association은 Prefix·PropertiesTarget을 추가합니다. canonical nonnull wrong type은 atomic 오류입니다. unknown fields·큰 숫자·null은 독립 Metadata.Body에 보존하며 protected·ID·Namespace·Self·Links·date parse를 합성하지 않습니다. Create result를 input identity로 seed하지 않고 returned Name은 passive입니다. raw Body·Header·pointer는 caller와 row 사이에서 독립 소유합니다.

## 실제 응답과 missing

원래 List200·Create201만 decode합니다. accepted read·Close·UTF-8·JSON·canonical·context·source 실패는 실제 전체 Body·Header·Status와 원인을 `resource.ResponseError`에 보존합니다. 실제204 Delete만 opaque ACK를 만들고 선택한 Namespace·Name과 raw evidence를 남깁니다. handling 실패에도 ACK+error를 반환합니다.

Delete `IgnoreMissing *bool`은 nil이면 true입니다. 직접 소유한 fixed DELETE404에서 전체 read·Close·context·source 처리가 모두 성공했을 때 `(nil,nil)`로 처리합니다. 그404 handling 실패는 `(nil,error)`이고 실제404 ResponseError를 보존합니다. explicit false는 native404 ownership·RetryFunc을 유지합니다. default true의 private404 수용은 native404 hook을 우회하는 좁은 Go 차이입니다. hidden parent도404일 수 있어 부재 증명이 아니며 자동 unprotect·GET·cleanup을 수행하지 않습니다. 삭제는 association만 제거하고 global type은 남습니다.

configured native pre-body retry·reauth·backoff·동일 target redirect는 유지하고 method·origin·path·query·serialized body·owned response fields·원래 accepted status를 바꾸는 hook은 차단합니다. native RetryFunc의 MoreHeaders 변경은 기존 provider policy로 유지합니다. accepted body는 한 번 닫고 handling 실패 뒤 replay하지 않습니다. read·Close·transport·ctx.Err/custom context.Cause와 native reauth ErrOriginal·ErrReauth 필드를 보존합니다.

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Glance `57f7dd9e76ef24e1e9013eceaa703bd442469a24`, Gophercloud `v2.15.0`입니다. native image resource-type CRUD는 없으며 SDK 소유 leaf입니다. Python 전체 Resource/cache/session/query parity나 실제 schema·WSME·DB·Unicode·권한·cloud side effect 검증을 주장하지 않습니다.
