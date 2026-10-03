# Glance namespace tags

`MetadefTags`는 namespace의 tag 정의를 다루는 SDK 소유 leaf입니다. `conn.ImageV2(ctx).MetadefTags`, `image.Service.API.MetadefTags`, `metadeftags.New(client)`에서 사용합니다. `InNamespace(ctx, namespace)`는 HTTP 없이 literal parent와 서비스 source를 scope 수명 동안 고정합니다. 이름 검색·Ref·parent GET을 수행하지 않습니다.

| Go 호출 | 실제 wire 요청·정상 응답 | 고정 Python 연결 |
| --- | --- | --- |
| `Create` | child bodyless POST, 201 | `add_tag_to_metadef_namespace` |
| `Get` | child bodyless GET, 200 | 별도 tag GET proxy 없음; inherited `check_tag`는 존재 확인 |
| `Update` | 현재 child PUT `{name:...}`, 200 | 별도 rename proxy 없음 |
| `Delete` | child DELETE, 204 | `remove_tag_from_metadef_namespace` |
| `DeleteAll` | collection DELETE, 204 | `remove_tags_from_metadef_namespace` |
| `Set` | collection POST `{tags:[{name:...}]}`, 201 | Namespace Resource의 `set_tags` |
| `List`·`All` | collection GET, 200 | inherited Resource의 `fetch_tags`; 별도 목록 proxy 없음 |

서버에는 [7개 실제 action](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L355-L396)이 있습니다. 고정 Python proxy에는 위의 3개 write만 있으며, 나머지 SDK 호출에 Python operation을 만들지 않습니다.

## 호출과 독립적인 결과

예제의 `setNames`는 호출자가 선택한 collection 입력입니다. `Append=true`를 명시하며 이미 있는 이름을 다시 보내면 서버가 conflict를 반환할 수 있습니다. `clearNamespace=true`는 전체 tags 삭제를 별도로 선택합니다. 각 요청은 독립적이고, 오류가 나면 이미 받은 결과를 반환합니다.

```go
package example

import (
    "context"

    "gophercloudsdk/image"
    "gophercloudsdk/image/v2/metadeftags"
)

type TagEvidence struct {
    Created *metadeftags.Tag
    Read *metadeftags.Tag
    Updated *metadeftags.Tag
    Set *metadeftags.SetResult
    Rows []*metadeftags.Tag
    All []*metadeftags.Tag
    Deleted *metadeftags.Acknowledgement
    Cleared *metadeftags.Acknowledgement
}

func manageNamespaceTags(ctx context.Context, service *image.Service, namespace, current, replacement string, setNames []string, clearNamespace bool) (out TagEvidence, err error) {
    scope, err := service.API.MetadefTags.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    out.Created, err = scope.Create(ctx, current,
        metadeftags.WithCreateOpts(metadeftags.CreateOpts{}),
        metadeftags.WithCreateHeader("X-Request-Source", "example"),
        metadeftags.WithCreateHeaders(map[string]string{"X-Trace": "create"}))
    if err != nil { return out, err }
    out.Read, err = scope.Get(ctx, current,
        metadeftags.WithGetOpts(metadeftags.GetOpts{}),
        metadeftags.WithGetHeader("X-Request-Source", "example"),
        metadeftags.WithGetHeaders(map[string]string{"X-Trace": "get"}))
    if err != nil { return out, err }
    out.Updated, err = scope.Update(ctx, current,
        metadeftags.WithUpdateOpts(metadeftags.UpdateOpts{}),
        metadeftags.WithUpdateHeader("X-Request-Source", "example"),
        metadeftags.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        metadeftags.WithUpdateName(replacement))
    if err != nil { return out, err }
    out.Set, err = scope.Set(ctx, setNames,
        metadeftags.WithSetOpts(metadeftags.SetOpts{}),
        metadeftags.WithSetHeader("X-Request-Source", "example"),
        metadeftags.WithSetHeaders(map[string]string{"X-Trace": "set"}),
        metadeftags.WithSetAppend(true))
    if err != nil { return out, err }

    listOptions := []metadeftags.ListOption{
        metadeftags.WithListOpts(metadeftags.ListOpts{}),
        metadeftags.WithListHeader("X-Request-Source", "example"),
        metadeftags.WithListHeaders(map[string]string{"X-Trace": "list"}),
        metadeftags.WithListLimit(2),
        metadeftags.WithListMarker(""),
        metadeftags.WithListSortKey("name"),
        metadeftags.WithListSortDir("asc"),
        metadeftags.WithListMaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }
    out.Deleted, err = scope.Delete(ctx, replacement,
        metadeftags.WithDeleteOpts(metadeftags.DeleteOpts{}),
        metadeftags.WithDeleteHeader("X-Request-Source", "example"),
        metadeftags.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadeftags.WithDeleteAllOpts(metadeftags.DeleteAllOpts{}),
        metadeftags.WithDeleteAllHeader("X-Request-Source", "example"),
        metadeftags.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

`List`와 `All`은 각각 새 iteration을 시작하며 cache를 공유하지 않습니다. 예제는 positive limit을 지정했으므로 source-backed marker page를 따라가고 로컬 cap을 적용합니다. 두 Delete는 기본부터 strict missing입니다.

고정 openstacksdk의 실제 proxy 3개와 별도의 Resource method를 구분한 비교입니다.

```python
from openstack.image.v2.metadef_namespace import MetadefNamespace


def proxy_namespace_tags(conn, namespace, tag, clear_namespace=False):
    conn.image.add_tag_to_metadef_namespace(namespace, tag)
    conn.image.remove_tag_from_metadef_namespace(namespace, tag)
    if clear_namespace:
        conn.image.remove_tags_from_metadef_namespace(namespace)


def resource_namespace_tags(conn, namespace, names):
    resource = MetadefNamespace.new(id=namespace)
    resource.set_tags(conn.image, names, append=True)
    resource.fetch_tags(conn.image)
    if names:
        resource.check_tag(conn.image, names[0])
    return resource
```

[실제 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1487-L1539)는 세 write 모두 None을 반환합니다. `_get_resource`는 전달된 Namespace Resource를 재사용하거나 이름으로 bare Resource를 만들며 parent를 조회하지 않습니다. [Namespace.add_tag·set_tags](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py#L109-L152)와 [inherited TagMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L28-L142)은 같은 Resource를 돌려주며 로컬 tags를 수정합니다. Go의 독립 Tag·SetResult·Acknowledgement와 cached Resource identity를 동일하게 취급하지 않습니다.

## literal 경로와 collection 입력

namespace와 개별 child 경로 이름은 nonempty valid UTF-8, 최대 80 rune입니다. colon·Unicode·space를 그대로 두고 path component를 한 번 escape합니다. 정확한 `.`·`..`, slash·backslash·percent·query·fragment 문자와 ASCII control/DEL은 HTTP 전에 거부합니다. trim·UUID 해석·Find·parent discovery는 없습니다. Create는 body 없이 POST하고 Update는 current 경로에 `{name:current}`를 PUT합니다. optional Name으로 rename하더라도 현재 경로를 바꾸지 않습니다. 서버는 [개별 name schema의 80 제한](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L288-L337)과 DB supplementary Unicode 제약을 별도로 판단하며 SDK가 유효한 4-byte Unicode 전체를 차단하지 않습니다. API schema 예제의 255 표기를 실제 controller의 80 제한과 혼동하지 않습니다.

Set의 `[]string`은 경로 이름이 아닌 JSON 입력입니다. valid UTF-8 literal string만 검사하고 80-rune·nonempty·control·경로 문자 제한을 추가하지 않습니다. empty·comma·newline·slash 같은 값을 normalize·trim·dedupe하지 않고 그대로 name object로 보냅니다. SQL length·Unicode·중복·권한은 서버가 판단하므로 저장과 round trip을 보장하지 않습니다. nil·empty slice는 모두 `{tags:[]}`로 직렬화하며 option callback 전에 이름 배열을 복사합니다.

`SetOpts.Append *bool`은 nil이면 false이고 SDK가 `X-OpenStack-Append: False` 또는 `True`를 소유해 보냅니다. Set source/caller header로 같은 header를 넣을 수 없습니다. [공식 Create tags 설명](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#create-tags)은 true를 append, 나머지를 replacement로 설명합니다. 고정 [SQL create_tags](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/metadef_api/tag.py#L114-L141)는 nonempty 목록에서만 기존 행을 지우므로 empty Set은 기존 tags를 clear하지 않습니다. 전체 삭제는 explicit `DeleteAll`입니다. SetResult.Tags는 제출된 항목의 응답이며 append 뒤 전체 union이나 DB transaction 성공 범위를 합성하지 않습니다. Python은 append 여부와 관계없이 로컬 tags를 제출한 목록으로 바꿉니다.

## 목록 query와 marker 진행

List는 생성 시 HTTP가 없는 lazy iterator입니다. nil Limit이면 query를 생략하고 한 GET으로 끝냅니다. explicit zero는 `limit=0` 한 요청 후 끝내고, positive Limit은 받은 raw 행 수가 limit 이상이면 마지막 raw canonical name을 다음 marker로 사용합니다. short·empty page에서 멈추며 oversized response를 강제로 오류로 만들지 않습니다. 서버 [index](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L139-L183)와 [공식 List tags](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-tags)와 [deserializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L411-L429)는 실제 limit·marker·sort query를 지원합니다. serializer는 next·first·Link를 만들지 않으므로 SDK는 그런 link를 따르거나 합성하지 않습니다.

Limit·Marker·SortKey·SortDir는 nil omission과 explicit 0·empty를 구분합니다. Limit 음수는 오류입니다. Marker·SortKey는 control 없는 valid UTF-8 query literal이며 child-route의 80 제한을 재사용하지 않습니다. SortDir은 asc·desc만 허용하고 nil이면 서버 기본값을 둡니다. sort 기본 created_at/desc와 전체 namespace 정책은 서버가 판단합니다. 다음 page에서는 원래 limit·sort를 유지하고 marker만 교체합니다.

전체 UTF-8·JSON syntax와 required nonnull tags array를 먼저 검사한 뒤 소비하는 Tag row만 strict object로 decode합니다. 페이지를 모두 소비한 뒤 continuation이 필요할 때 마지막 raw name을 caller mutation 전에 독립 보존하며 missing·empty·invalid·반복 marker를 오류로 처리합니다. `MaxItems=0`은 로컬 cap 없음, 양수는 소비 수 제한, 음수는 HTTP 전 오류입니다. cap이나 break는 사용하지 않는 malformed row·marker 검사·후속 요청을 건너뜁니다. All은 빈 성공에서 nonnil empty slice, 오류에서 부분 slice를 버립니다. 로컬 filter·tag cache·response identity retargeting을 수행하지 않습니다.

## 모델·source·오류 증거

Tag는 exact canonical optional nullable Name과 literal CreatedAt·UpdatedAt string pointer를 가집니다. valid UTF-8의 nonnull JSON object에서 잘못된 nonnull canonical type은 atomic 오류입니다. unknown fields·큰 숫자·null은 독립 Metadata.Body에 남으며 ID·parent·date parse·Self·schema·Links를 합성하지 않습니다. 반환 Name은 passive이고 다음 경로를 바꾸지 않습니다.

SetResult는 root Metadata와 required nonnull Tags array를 가집니다. 모든 nonnull object row를 eager·atomic decode하고 empty 성공은 nonnil Tags입니다. root와 각 nested Tag의 Body·Header·pointer를 독립 소유하며 잘못된 row에서 부분 result를 반환하지 않습니다. List는 소비하는 행만 decode하므로 Set의 eager 계약과 다릅니다.

scope는 원래 API/client/provider·type·endpoint·base·microversion을 수명 동안 고정합니다. context는 저장하지 않으며 callback 전·accepted body 뒤·각 소비 행에 source/context를 검사합니다. 요청·page마다 최신 유효 ordinary source headers를 복사하고 원래 provider의 live auth를 사용합니다. 7개 concrete 옵션 family는 FullOpts 전체 교체·last wins이며 pointer·map·optionslice·callback config를 독립 복사합니다. callback은 요청 또는 iteration마다 한 번, nil/error callback·소유 header·invalid input은 HTTP 전에 오류입니다.

실제 Create·Set201/Get·List·Update200만 decode합니다. accepted read·Close·UTF-8·JSON·canonical·context·source 오류는 실제 전체 Body·Header·StatusCode와 원인을 `resource.ResponseError`에 보존합니다. 실제 Delete·DeleteAll204는 opaque Acknowledgement에 선택한 Namespace와 개별 child Name을 남기며 bulk Name은 nil입니다. handling 실패에서도 ACK와 오류를 함께 반환합니다. 두 Delete는 모든 404를 오류로 유지하고 IgnoreMissing·private 404 bypass가 없습니다. hidden parent/tag가 404로 표현될 수 있으므로 부재나 권한을 추정하지 않습니다. SDK가 unprotect·enumerate·count·wait·cleanup을 수행하지 않습니다.

configured native pre-body retry·reauth·backoff·동일 target redirect는 유지하고 method·origin·path·query·serialized body·owned response fields·accepted status를 바꾸는 hook은 차단합니다. RetryFunc의 `MoreHeaders` 변경은 기존 native provider policy로 유지되므로 option/source ordinary header 제한과 구분합니다. accepted body는 한 번 닫고 오류 뒤 replay하지 않습니다. read·Close·transport·ctx.Err/custom context.Cause와 native reauth ErrOriginal·ErrReauth 필드를 보존합니다. Python `raise_from_response`의 status<400 처리, TagMixin adapter/microversion과 local cache 변경은 별도 계약입니다.

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Glance `57f7dd9e76ef24e1e9013eceaa703bd442469a24`, native Gophercloud `v2.15.0`입니다. native namespace-tag CRUD는 없으며 이 leaf는 SDK 소유입니다. 실제 3개 Python proxy의 partial review는 직접 연결되는 13개 callable만 다룹니다. 나머지 24개 workflow는 서버·Resource 소스로 검토한 별도 SDK 계약이고 synthetic Python operation을 만들지 않습니다. Python 전체 Resource·session·cache parity나 실제 schema·권한·DB·cloud side effect 검증을 주장하지 않습니다.

실제 검증은 [core 테스트](core_test.go), [option 테스트](options_test.go), [외부 HTTP 계약](contracts_test.go), [Connection 경로·paging 계약](../../../connection_image_metadef_tags_test.go), [generator registry 계약](../../../internal/cmd/sdkgen/glance_metadef_tags_test.go)에 연결되어 있습니다. 새 Go 예제 한 개를 컴파일하고 기존 269개 source를 그대로 보존했습니다. 누적 270개 전체를 다시 실행했다는 의미는 아닙니다.
