# 고정 endpoint의 schema 조회

`image.Service`의 16개 schema getter는 각자의 고정 endpoint로 body/query 없는 GET을 보내 실제 200의 schema document를 읽습니다. 이미지·member·task·metadef의 ID나 Name을 받지 않습니다. 복수형 getter도 schema document 하나를 반환하며 resource 목록이나 pager를 만들지 않습니다.

```go
package example

import (
    "context"

    "gophercloudsdk/image"
)

func getAllSchemas(ctx context.Context, service *image.Service) (map[string]*image.Schema, error) {
    options := []image.GetSchemaOption{
        image.WithGetSchemaOpts(image.GetSchemaOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithGetSchemaHeader("X-Schema-Request", "discovery"),
        image.WithGetSchemaHeaders(map[string]string{"X-Client-Workflow": "schemas"}),
    }
    getters := []struct {
        path string
        get  func(context.Context, ...image.GetSchemaOption) (*image.Schema, error)
    }{
        {"schemas/images", service.GetImagesSchema},
        {"schemas/image", service.GetImageSchema},
        {"schemas/members", service.GetMembersSchema},
        {"schemas/member", service.GetMemberSchema},
        {"schemas/tasks", service.GetTasksSchema},
        {"schemas/task", service.GetTaskSchema},
        {"schemas/metadefs/namespace", service.GetMetadefNamespaceSchema},
        {"schemas/metadefs/namespaces", service.GetMetadefNamespacesSchema},
        {"schemas/metadefs/resource_type", service.GetMetadefResourceTypeSchema},
        {"schemas/metadefs/resource_types", service.GetMetadefResourceTypesSchema},
        {"schemas/metadefs/object", service.GetMetadefObjectSchema},
        {"schemas/metadefs/objects", service.GetMetadefObjectsSchema},
        {"schemas/metadefs/property", service.GetMetadefPropertySchema},
        {"schemas/metadefs/properties", service.GetMetadefPropertiesSchema},
        {"schemas/metadefs/tag", service.GetMetadefTagSchema},
        {"schemas/metadefs/tags", service.GetMetadefTagsSchema},
    }
    results := make(map[string]*image.Schema, len(getters))
    for _, getter := range getters {
        schema, err := getter.get(ctx, options...)
        if err != nil {
            return results, err
        }
        results[getter.path] = schema
    }
    return results, nil
}
```

이 예제 함수는 16개 독립 getter를 순서대로 호출하며 실패하면 이미 얻은 값과 오류를 반환합니다. SDK의 atomic batch나 cache API가 아닙니다. 옵션을 생략한 각 getter의 기본 설정은 빈 일반 header map입니다. `WithGetSchemaOpts`는 전체 concrete 설정을 교체하고 `Header`·`Headers` helper는 일반 header를 추가합니다. header map과 option slice를 snapshot하고 각 callback config를 한 번 적용한 뒤 다시 복사합니다. 보호된 auth·framing·version·representation header, 충돌하는 alias, nil/error callback은 HTTP 전에 거부합니다.

## Python의 16개 Resource fetch

```python
def get_all_schemas(conn):
    getters = (
        ("schemas/images", conn.image.get_images_schema),
        ("schemas/image", conn.image.get_image_schema),
        ("schemas/members", conn.image.get_members_schema),
        ("schemas/member", conn.image.get_member_schema),
        ("schemas/tasks", conn.image.get_tasks_schema),
        ("schemas/task", conn.image.get_task_schema),
        ("schemas/metadefs/namespace", conn.image.get_metadef_namespace_schema),
        ("schemas/metadefs/namespaces", conn.image.get_metadef_namespaces_schema),
        ("schemas/metadefs/resource_type", conn.image.get_metadef_resource_type_schema),
        ("schemas/metadefs/resource_types", conn.image.get_metadef_resource_types_schema),
        ("schemas/metadefs/object", conn.image.get_metadef_object_schema),
        ("schemas/metadefs/objects", conn.image.get_metadef_objects_schema),
        ("schemas/metadefs/property", conn.image.get_metadef_property_schema),
        ("schemas/metadefs/properties", conn.image.get_metadef_properties_schema),
        ("schemas/metadefs/tag", conn.image.get_metadef_tag_schema),
        ("schemas/metadefs/tags", conn.image.get_metadef_tags_schema),
    )
    return {path: getter() for path, getter in getters}
```

고정 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [16개 proxy getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1951-L2161)는 `_get(..., requires_id=False, base_path=...)`를 호출합니다. 첫 6개는 `Schema`, 나머지 10개는 `MetadefSchema` Resource를 fetch합니다. mutable Resource·descriptor coercion·dirty/cache·adapter/session 동작은 Go의 passive DTO와 다릅니다.

Python `Schema.additional_properties`는 dict descriptor이고 `MetadefSchema.additional_properties`는 bool descriptor입니다. properties/definitions/required에도 변환이 적용될 수 있습니다. Go는 알려진 canonical field의 형태를 엄격히 읽고 `additionalProperties`는 임의의 raw JSON으로 보존하므로 Python의 변환 결과를 full parity로 주장하지 않습니다.

## 실제 서버의 raw·minimal 형태

[공식 API 문서](https://docs.openstack.org/api-ref/image/v2/index.html#show-image-schema)는 정상 200의 schema document를 설명하며 예제보다 실제 응답을 기준으로 삼도록 안내합니다. 고정 Glance commit `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/schemas.py#L58-L104)와 [schema serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/schema.py#L82-L240)는 아래 형태를 사용합니다.

| endpoint 종류 | 서버 형태 | 데이터에서 확인할 경계 |
| --- | --- | --- |
| image | `PermissiveSchema.raw()` | top-level `additionalProperties`는 `{"type":"string"}`이며 custom image property 설정에 따라 내용이 달라질 수 있습니다. |
| images | `CollectionSchema.raw()` | `properties.images.items`에 item schema가 중첩됩니다. top-level `additionalProperties`는 생략됩니다. |
| member·task | `Schema.minimal()` | 일반 top-level `additionalProperties`와 links가 생략됩니다. |
| members·tasks | `CollectionSchema.minimal()` | item schema와 describedby template가 중첩됩니다. tasks의 partial item schema는 single task schema와 다릅니다. |
| metadef 단수형 | `Schema.raw()` | top-level `additionalProperties:false`와 필요한 definitions/required를 반환합니다. `resource_type`의 schema name은 `resource_type_association`입니다. |
| metadef 복수형 | `CollectionSchema.raw()`; properties는 `DictCollectionSchema.raw()` | 일반 collection은 array item schema이고, `properties.properties`는 object와 item `additionalProperties`로 표현됩니다. child definitions가 root로 이동할 수 있습니다. |

서버는 optional definitions·required·links를 truthy일 때만 넣습니다. 따라서 응답에 없는 필드를 client default로 만들어 채우지 않습니다. collection schema의 `first`·`next`·`schema`·links는 JSON Schema 문서 안의 설명·template이며 실제 다음 페이지 요청이 아닙니다.

## canonical field와 독립 raw 데이터

모든 getter는 `*image.Schema`를 반환합니다. `Name *string`, `Properties`·`Definitions map[string]json.RawMessage`, `Required []string`, `AdditionalProperties json.RawMessage`와 embedded `resource.Metadata`를 가집니다. document의 name은 passive 값이며 고정 path와 일치하도록 강제하지 않습니다.

`Name`은 missing/null이면 nil, 그 외에는 string만 허용합니다. `Properties`·`Definitions`는 missing/null이면 nil이고 그 외에는 object만 허용합니다. `Required`는 missing/null이면 nil이며 그 외에는 null이 아닌 string들로 된 array여야 합니다. 명시적인 `""`·`{}`·`[]`는 각각 nonnil empty 값으로 구분됩니다. 잘못된 canonical type을 Python처럼 coercion하지 않습니다.

`AdditionalProperties`는 absent일 때만 nil입니다. false·true·object·array·string·number와 명시적인 JSON null도 그 raw bytes를 유지합니다. `Body map[string]json.RawMessage`에는 canonical·unknown·대소문자 decoy를 포함한 전체 root field가 남습니다. properties/definitions 안의 null·false·nested value·`9007199254740993`·fraction도 float64로 변환하지 않습니다. Body 값, typed map 값, AdditionalProperties와 실제 response Header는 서로 독립적으로 소유합니다.

canonical decoder는 embedded Metadata의 links·CreatedAt·UpdatedAt를 해석하지 않으며 이 필드는 nil로 남습니다. `links`, `created_at`, `updated_at`, `id`, `$schema`, `$ref`, URL·extension field는 형태에 관계없이 valid JSON 데이터로 Body에만 보존합니다. 중첩 JSON Schema의 유효성·draft·format·required 의미를 SDK가 검증하거나 다른 request의 preflight gate로 사용하지 않습니다.

## 실제 200 증거와 기존 source 정책

전체 body는 valid UTF-8의 nonnull JSON object여야 합니다. empty body·null·array·scalar·잘못된 JSON·canonical schema 오류와 accepted read/Close/context 실패는 nil typed Schema와 실제 200 bytes/header/status를 보존한 `resource.ResponseError`를 반환합니다. 원래 read·Close·`ctx.Err()`·custom cause를 유지하고 body는 한 번 닫으며 accepted body 실패 뒤 replay하지 않습니다. 다른 actual status와 transport/hook 오류는 기존 native 오류로 반환하고 missing을 숨기거나 cache/fallback 결과로 바꾸지 않습니다.

source/context/client/provider·prefix·일반 header를 callback 전에 검증·capture합니다. helper/전체 설정/각 callback의 map을 복사하고 준비 후 context와 원래 provider identity를 다시 확인합니다. source가 안전하게 바뀌어도 준비한 route/header를 유지하며 원래 provider의 최신 token을 사용합니다. body·query·Ref/ID·arbitrary schema path·filter·pager 옵션은 없습니다.

공통 `DoJSON`과 fixed request policy는 configured pre-body retry·reauth·backoff 및 같은 target의 native redirect 동작을 유지합니다. method·origin·path·query 변경은 transport 전에 차단합니다. RetryFunc가 nil body를 JSON null로 바꾸거나 RawBody·KeepResponseBody·JSONResponse 등 SDK 소유 필드를 바꾸면 다음 요청 전에 원래 오류와 hook/encoding cause를 보존해 거부합니다. callback이 native OkCodes를 넓혀도 실제 200만 accepted합니다. native reauth의 원인들은 기존 `ErrOriginal`·`ErrReauth` 필드에 남습니다.

이 getter들은 schema 조회만 수행합니다. resource CRUD·Name lookup·URL/ref following·paging·cache mutation·recursive schema validation·자동 quota/권한 enforcement·discovery gate·wait·cleanup을 추가하지 않습니다. local 계약과 컴파일 검증은 실제 cloud의 인증·deployment policy나 full Python Resource/cache/session 동등성을 증명하지 않습니다.

실제 local 계약은 [외부 HTTP tests](schemas_contracts_test.go), [core tests](schemas_core_test.go), [option tests](schemas_options_test.go)에서 검증합니다. [Connection test](../connection_image_schemas_test.go)는 공유 native client와 passive discovery를, [generator test](../internal/cmd/sdkgen/glance_schemas_test.go)는 기존 native binding 보존과 16개 getter 문서를 확인합니다.
