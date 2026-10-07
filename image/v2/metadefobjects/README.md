# Glance 메타데이터 객체

`MetadefObjects`는 literal namespace 안의 객체를 다루는 SDK 소유 leaf입니다. `conn.ImageV2(ctx).MetadefObjects`, `image.Service.API.MetadefObjects`, `metadefobjects.New(client)`에서 같은 API를 사용합니다. 먼저 `InNamespace(ctx, namespace)`로 scope를 만들며 이 호출은 HTTP를 보내지 않습니다. namespace와 객체 이름은 UUID나 Ref가 아니라 그대로 사용할 이름이고, parent 조회나 Find를 실행하지 않습니다.

| Go 호출 | 고정 Python proxy | 요청·정상 응답 |
| --- | --- | --- |
| `scope.Create` | `create_metadef_object` | collection POST, 201 |
| `scope.Get` | `get_metadef_object` | child GET, 200 |
| `scope.Update` | `update_metadef_object` | 현재 child PUT, 200 |
| `scope.Delete` | `delete_metadef_object` | child DELETE, 204 |
| `scope.DeleteAll` | `delete_all_metadef_objects` | collection DELETE, 204 |
| `scope.List` | `metadef_objects` | lazy한 단일 collection GET, 200 |
| `scope.All` | 같은 generator의 수집 | 단일 collection GET, 200 |

## 생성·조회·목록·교체·삭제

다음 예제는 호출자가 선택한 namespace와 객체 이름을 사용합니다. `replacementName`은 같은 이름 또는 새 이름입니다. 각 요청은 독립적이며 실패하면 이미 받은 증거를 반환합니다. `clearNamespace=true`는 해당 namespace의 모든 객체를 삭제하는 별도 선택입니다. SDK가 중간 실패를 복구하거나 다른 객체를 자동 정리하지 않습니다.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/gophercloudsdk/image"
    "github.com/JSYoo5B/gophercloudsdk/image/v2/metadefobjects"
)

type ObjectEvidence struct {
    Created *metadefobjects.Object
    Read *metadefobjects.Object
    Rows []*metadefobjects.Object
    All []*metadefobjects.Object
    Updated *metadefobjects.Object
    Deleted *metadefobjects.Acknowledgement
    Cleared *metadefobjects.Acknowledgement
}

func manageObjects(ctx context.Context, service *image.Service, namespace, name, replacementName string, clearNamespace bool) (out ObjectEvidence, err error) {
    scope, err := service.API.MetadefObjects.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    description := "First line\nSecond line"
    properties := map[string]json.RawMessage{
        "vcpus": json.RawMessage(`{"type":"integer","title":"Virtual CPUs","minimum":1}`),
    }
    required := []string{"vcpus"}
    out.Created, err = scope.Create(ctx, name,
        metadefobjects.WithCreateOpts(metadefobjects.CreateOpts{Description: &description}),
        metadefobjects.WithCreateHeader("X-Request-Source", "example"),
        metadefobjects.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        metadefobjects.WithCreateDescription(description),
        metadefobjects.WithCreateProperties(properties),
        metadefobjects.WithCreateRequired(required))
    if err != nil { return out, err }

    out.Read, err = scope.Get(ctx, name,
        metadefobjects.WithGetOpts(metadefobjects.GetOpts{}),
        metadefobjects.WithGetHeader("X-Request-Source", "example"),
        metadefobjects.WithGetHeaders(map[string]string{"X-Trace": "get"}))
    if err != nil { return out, err }

    listOptions := []metadefobjects.ListOption{
        metadefobjects.WithListOpts(metadefobjects.ListOpts{}),
        metadefobjects.WithListHeader("X-Request-Source", "example"),
        metadefobjects.WithListHeaders(map[string]string{"X-Trace": "list"}),
        metadefobjects.WithListMaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = scope.Update(ctx, name,
        metadefobjects.WithUpdateOpts(metadefobjects.UpdateOpts{}),
        metadefobjects.WithUpdateHeader("X-Request-Source", "example"),
        metadefobjects.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        metadefobjects.WithUpdateName(replacementName),
        metadefobjects.WithUpdateDescription(description),
        metadefobjects.WithUpdateProperties(properties),
        metadefobjects.WithUpdateRequired(required))
    if err != nil { return out, err }

    out.Deleted, err = scope.Delete(ctx, replacementName,
        metadefobjects.WithDeleteOpts(metadefobjects.DeleteOpts{}),
        metadefobjects.WithDeleteHeader("X-Request-Source", "example"),
        metadefobjects.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        metadefobjects.WithDeleteIgnoreMissing(false))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadefobjects.WithDeleteAllOpts(metadefobjects.DeleteAllOpts{}),
        metadefobjects.WithDeleteAllHeader("X-Request-Source", "example"),
        metadefobjects.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

`List`와 `All`은 각각 새 GET을 실행하고 cache를 공유하지 않습니다. 예제는 개별 Delete에 strict missing을 선택했습니다. Update가 생략한 optional 설정을 보존하지 않으므로 유지할 description·properties·required를 다시 명시합니다.

고정 openstacksdk의 실제 6개 proxy 호출은 다음과 같습니다.

```python
def manage_objects(conn, namespace, name, replacement_name, clear_namespace=False):
    properties = {"vcpus": {"type": "integer", "title": "Virtual CPUs", "minimum": 1}}
    created = conn.image.create_metadef_object(
        namespace, name=name, description="First line\nSecond line",
        properties=properties, required=["vcpus"])
    fetched = conn.image.get_metadef_object(name, namespace)
    rows = list(conn.image.metadef_objects(namespace))
    updated = conn.image.update_metadef_object(
        name, namespace, name=replacement_name,
        description="First line\nSecond line", properties=properties, required=["vcpus"])
    deleted = conn.image.delete_metadef_object(
        replacement_name, namespace, ignore_missing=False)
    cleared = conn.image.delete_all_metadef_objects(namespace) if clear_namespace else None
    return created, fetched, rows, updated, deleted, cleared
```

Python의 [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1543-L1692)는 namespace와 객체 이름 또는 Resource를 받고 create/get/update는 `MetadefObject`, list는 generator를 반환합니다. 개별 delete는 삭제한 Resource 또는 missing일 때 None, bulk delete는 `MetadefNamespace`를 반환합니다. 이 반환값을 Go의 실제 204 acknowledgement와 동일하게 취급하지 않습니다.

## Literal 입력과 replacement update

namespace와 객체 이름은 nonempty valid UTF-8, 최대 80 rune입니다. colon·Unicode·space를 유지하고 path component를 한 번 escape합니다. 정확한 `.`·`..`, slash·backslash·percent·query·fragment 문자와 ASCII control/DEL은 거부합니다. 자동 trim·case 변경·UUID 해석·이름 검색은 하지 않습니다. route identity와 source/context는 callback 전에, optional rename과 payload는 callback 뒤 HTTP 전에 검사합니다.

Description은 optional string pointer이며 valid UTF-8이면 empty·multiline을 허용하고 길이 제한을 추가하지 않습니다. namespace의 500 rune 제한을 객체에 적용하지 않습니다. Properties는 nil이면 생략, nonnil empty map이면 `{}`를 보냅니다. 각 property key는 valid UTF-8, 각 RawMessage 값은 valid UTF-8의 nonnull JSON object여야 합니다. SDK가 type·title·JSON Schema keyword를 whitelist하거나 서버 schema를 가져와 검증하지 않습니다. 최종 허용 여부는 서버가 판단합니다.

Required는 nil이면 생략, nonnil empty slice이면 `[]`를 보냅니다. entry는 valid UTF-8만 검사하며 empty·comma·newline·반복 값을 literal로 유지합니다. SDK가 trim·dedupe·property membership 검사나 CSV 변환을 하지 않습니다. 고정 [repository의 저장·복원](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/__init__.py#L561-L601)은 Required를 comma로 join/split하므로 comma를 포함한 entry의 round trip을 보장할 수 없습니다.

`Update(ctx, current, ...)`는 current 경로에 한 번 PUT합니다. body의 name은 기본 current이고 `WithUpdateName`으로 rename을 지정합니다. [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_objects.py#L174-L221)는 생략한 description·required·properties도 replacement 값으로 처리하여 기존 optional 설정을 reset합니다. SDK는 GET/RMW·merge·PATCH·rollback을 하지 않습니다.

Python `update_metadef_object`는 전달된 객체를 `_get_id`로 줄인 뒤 generic `_update`에 넘깁니다. 같은 cached 객체 자체를 넘기는 namespace proxy와 호출 경로가 다릅니다. [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)은 dirty body/header가 없으면 HTTP 없이 반환할 수 있고, 객체의 mandatory name을 body에 자동으로 되넣지 않습니다. 위 Python 예제는 name을 명시합니다. Go는 current-name을 기본 body에 넣고 항상 replacement PUT을 보내는 명시적 편의 계약입니다.

## 유한 목록과 scope 수명

`List`는 iterator 생성 시 HTTP를 보내지 않습니다. iteration마다 옵션을 새로 준비하여 query와 body 없는 collection GET 하나만 실행합니다. required nonnull `objects` array의 행은 소비할 때만 decode합니다. `MaxItems=0`은 로컬 cap 없음, 양수는 로컬 소비 수 제한, 음수는 HTTP 전 오류입니다. cap이나 caller break 뒤의 사용하지 않는 malformed 행은 decode하지 않습니다. 성공한 빈 `All`은 nonnil empty slice이고 오류에서는 부분 slice를 버립니다.

고정 [repository.list](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/__init__.py#L625-L632)와 [serializer.index](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_objects.py#L430-L435)는 전체 namespace 객체를 유한 배열로 반환하며 next를 생성하지 않습니다. [공식 List Objects 문서](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-objects)의 namespace pagination 설명과 실제 객체 경로의 차이를 반영했습니다. Go는 next·first·schema·self·links·HTTP Link를 해석하거나 따라가지 않고 wire limit·marker·sort·filter, hint나 fallback을 제공하지 않습니다. 고정 Python proxy도 목록 query 인자를 노출하지 않습니다.

`InNamespace`는 API/client/provider·type·endpoint·base·microversion을 scope 수명 동안 고정하고 context를 저장하지 않습니다. 이후 target/source drift는 callback 전에, accepted body 뒤와 소비하는 행 전후에 검사합니다. 일반 source header는 각 호출 때 최신 유효 값을 복사하고 원래 provider의 live auth를 사용합니다. 이전 응답의 namespace/name/self/schema로 route를 바꾸지 않습니다. client/provider를 동시에 임의 변경할 수 있다는 계약은 제공하지 않습니다.

## Canonical 모델·옵션·오류 증거

`Object`의 Name·Description·Self·Schema와 Metadata.CreatedAt·UpdatedAt은 canonical optional nullable string pointer입니다. 날짜를 parse하지 않고 ID·namespace·이름·Metadata.Links를 합성하지 않습니다. 전체 응답은 valid UTF-8의 nonnull JSON object이고 canonical key는 정확한 대소문자만 읽습니다. Properties는 missing/null이면 nil, empty object이면 nonnil이며 inner 값은 어떤 JSON이든 독립 RawMessage로 보존합니다. 요청 property 값의 object 제한을 응답에 적용하지 않습니다. Required는 missing/null이면 nil, empty array이면 nonnil이며 nonnull string array만 허용합니다. null entry와 type coercion은 거부합니다. Unknown 값·raw links·큰 숫자의 precision도 독립적인 Metadata.Body에 남습니다.

6개 concrete 옵션 family의 `With…Opts`는 전체 설정을 교체하고 나중 옵션이 우선합니다. map·pointer·RawMessage bytes·slice와 callback config를 독립적으로 복사하며 callback은 요청 또는 iteration마다 한 번 실행합니다. nil/error callback·보호된 header·invalid input은 HTTP 전에 오류입니다. 임의 body builder·query extension·microversion override는 제공하지 않습니다.

실제 Create201/Get·List·Update200만 decode합니다. accepted read·Close·UTF-8·JSON·model·context·source 오류는 typed Object를 반환하지 않고 `resource.ResponseError`에 실제 Body·Header·StatusCode와 원인을 보존합니다. Delete·DeleteAll의 실제 204는 opaque `Acknowledgement`에 선택한 Namespace와 실제 증거를 남깁니다. Name은 개별 삭제의 요청 이름이며 DeleteAll에서는 nil입니다. accepted handling 실패에서도 acknowledgement와 오류를 함께 반환하며 count나 모든 자식의 완료 증거를 합성하지 않습니다.

개별 Delete의 `IgnoreMissing *bool`은 nil이면 true입니다. true는 SDK가 직접 소유한 physical 404의 read·Close·context·source 검사가 모두 성공했을 때만 nil/nil로 끝내고 그 404에서만 native RetryFunc를 우회합니다. false는 native 오류·retry 정책을 유지합니다. DeleteAll은 모든 404를 오류로 유지합니다. 숨겨진 parent/객체를 서버가 404로 표현할 수 있으므로 조용한 404는 실제 부재의 증거가 아닙니다. protected namespace의 삭제도 서버 정책을 따르며 자동 unprotect를 수행하지 않습니다.

configured native pre-body retry·reauth·backoff·동일한 target redirect 정책은 유지합니다. method·origin·path·query·serialized body·owned response fields 또는 accepted status를 바꾸는 callback은 차단합니다. accepted body는 한 번 닫고 실패 뒤 replay하지 않습니다. read·Close·transport 원인과 `ctx.Err()`·custom `context.Cause`를 보존하며 native reauth 원인은 기존 `ErrOriginal`·`ErrReauth` 필드에서 확인합니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Glance commit `57f7dd9e76ef24e1e9013eceaa703bd442469a24`, native Gophercloud `v2.15.0`입니다. native에는 object CRUD/list API가 없어 SDK가 이 leaf를 소유합니다. Python의 mutable Resource·dirty cache·descriptor/coercion·adapter/session·arbitrary attrs/version controls와 전체 parity를 주장하지 않습니다. 서버 JSON Schema·권한·보호·DB 기본값·CSV 저장과 실제 cloud side effect는 로컬 호출 계약과 별개입니다.

실제 route·payload·canonical raw 모델·유한 목록·응답 소유권은 [외부 HTTP 테스트](contracts_test.go), [core 증거 테스트](core_test.go), [옵션 소유권 테스트](options_test.go)에서 확인합니다. [Connection 테스트](../../../connection_image_metadef_objects_test.go)는 shared client와 zero-HTTP scope를, [생성기 테스트](../../../internal/cmd/sdkgen/glance_metadef_objects_test.go)는 SDK 소유 scope·concrete 옵션과 기존 native API 유지를 검증합니다.

Python `get_metadef_object`의 새 Resource는 request name과 namespace_name URI를 seed로 유지할 수 있습니다. Go 요청 부모는 `scope.NamespaceName()`으로 확인하며, `Object.Name`은 실제 응답만 반영합니다. Python의 untyped properties·required와 달리 Go는 object/string-array canonical 타입을 검증하고 원문은 `Metadata.Body`에 남깁니다.
