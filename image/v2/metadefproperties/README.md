# Glance 메타데이터 property

`MetadefProperties`는 literal namespace 안의 property 정의를 다루는 SDK 소유 leaf입니다. `conn.ImageV2(ctx).MetadefProperties`, `image.Service.API.MetadefProperties`, `metadefproperties.New(client)`에서 같은 API를 사용합니다. `InNamespace(ctx, namespace)`는 HTTP 없이 scope를 만들고 parent와 서비스 target을 고정합니다. namespace와 property 이름은 직접 사용할 이름이며 Find·Ref·parent 조회를 수행하지 않습니다.

| Go 호출 | 고정 Python proxy | 요청·정상 응답 |
| --- | --- | --- |
| `scope.Create` | `create_metadef_property` | collection POST, 201 |
| `scope.Get` | `get_metadef_property` | child GET, 200 |
| `scope.Update` | `update_metadef_property` | 현재 child PUT, 200 |
| `scope.Delete` | `delete_metadef_property` | child DELETE, 204 |
| `scope.DeleteAll` | `delete_all_metadef_properties` | collection DELETE, 204 |
| `scope.List` | `metadef_properties` | lazy한 단일 collection GET, 200 |
| `scope.All` | 같은 generator의 수집 | 단일 collection GET, 200 |

## 생성·조회·목록·교체·삭제

다음 예제는 호출자가 선택한 namespace와 property 이름을 사용합니다. `replacementName`은 같은 이름 또는 새 이름이고, `clearNamespace=true`는 해당 namespace의 모든 property를 삭제하는 별도 선택입니다. 각 요청은 독립적이며 실패하면 이미 받은 증거를 반환합니다. Update는 생략한 keyword를 보존하지 않으므로 유지할 값을 다시 명시합니다.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/gophercloudsdk/image"
    "github.com/JSYoo5B/gophercloudsdk/image/v2/metadefproperties"
)

type PropertyEvidence struct {
    Created *metadefproperties.Property
    Read *metadefproperties.Property
    Rows []*metadefproperties.Property
    All []*metadefproperties.Property
    Updated *metadefproperties.Property
    Deleted *metadefproperties.Acknowledgement
    Cleared *metadefproperties.Acknowledgement
}

func manageProperties(ctx context.Context, service *image.Service, namespace, name, replacementName string, clearNamespace bool) (out PropertyEvidence, err error) {
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    propertyType, title := "string", "Hypervisor Type"
    description := "First line\nSecond line"
    attributes := map[string]any{"enum": []string{"kvm", "qemu"}, "default": "kvm"}
    out.Created, err = scope.Create(ctx, name,
        metadefproperties.WithCreateOpts(metadefproperties.CreateOpts{
            Type: &propertyType, Title: &title,
            Attributes: map[string]json.RawMessage{"enum": json.RawMessage(`["kvm","qemu"]`)},
        }),
        metadefproperties.WithCreateHeader("X-Request-Source", "example"),
        metadefproperties.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        metadefproperties.WithCreateType(propertyType),
        metadefproperties.WithCreateTitle(title),
        metadefproperties.WithCreateDescription(description),
        metadefproperties.WithCreateAttributes(attributes),
        metadefproperties.WithCreateAttribute("readonly", false))
    if err != nil { return out, err }

    out.Read, err = scope.Get(ctx, name,
        metadefproperties.WithGetOpts(metadefproperties.GetOpts{}),
        metadefproperties.WithGetHeader("X-Request-Source", "example"),
        metadefproperties.WithGetHeaders(map[string]string{"X-Trace": "get"}),
        metadefproperties.WithGetResourceType(""))
    if err != nil { return out, err }

    listOptions := []metadefproperties.ListOption{
        metadefproperties.WithListOpts(metadefproperties.ListOpts{}),
        metadefproperties.WithListHeader("X-Request-Source", "example"),
        metadefproperties.WithListHeaders(map[string]string{"X-Trace": "list"}),
        metadefproperties.WithListMaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = scope.Update(ctx, name,
        metadefproperties.WithUpdateOpts(metadefproperties.UpdateOpts{}),
        metadefproperties.WithUpdateHeader("X-Request-Source", "example"),
        metadefproperties.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        metadefproperties.WithUpdateName(replacementName),
        metadefproperties.WithUpdateType(propertyType),
        metadefproperties.WithUpdateTitle(title),
        metadefproperties.WithUpdateDescription(description),
        metadefproperties.WithUpdateAttributes(attributes),
        metadefproperties.WithUpdateAttribute("readonly", false))
    if err != nil { return out, err }

    out.Deleted, err = scope.Delete(ctx, replacementName,
        metadefproperties.WithDeleteOpts(metadefproperties.DeleteOpts{}),
        metadefproperties.WithDeleteHeader("X-Request-Source", "example"),
        metadefproperties.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        metadefproperties.WithDeleteIgnoreMissing(false))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadefproperties.WithDeleteAllOpts(metadefproperties.DeleteAllOpts{}),
        metadefproperties.WithDeleteAllHeader("X-Request-Source", "example"),
        metadefproperties.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

`List`와 `All`은 각각 새 GET을 실행하고 cache를 공유하지 않습니다. 예제는 Get에 explicit empty `resource_type=`를 보내며 개별 Delete에는 strict missing을 선택했습니다. Type·Title은 Create와 Update 모두 필수 pointer이므로 zero/full opts만으로 호출하면 HTTP 전에 오류입니다.

고정 openstacksdk의 실제 6개 proxy 호출은 다음과 같습니다.

```python
def manage_properties(conn, namespace, name, replacement_name, clear_namespace=False):
    created = conn.image.create_metadef_property(
        namespace, name=name, type="string", title="Hypervisor Type",
        description="First line\nSecond line", enum=["kvm", "qemu"], default="kvm", readonly=False)
    fetched = conn.image.get_metadef_property(name, namespace, resource_type="")
    rows = list(conn.image.metadef_properties(namespace))
    updated = conn.image.update_metadef_property(
        name, namespace, name=replacement_name, type="string", title="Hypervisor Type",
        description="First line\nSecond line", enum=["kvm", "qemu"], default="kvm", readonly=False)
    conn.image.delete_metadef_property(replacement_name, namespace, ignore_missing=False)
    if clear_namespace:
        conn.image.delete_all_metadef_properties(namespace)
    return created, fetched, rows, updated
```

Python의 [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1790-L1947)는 namespace와 property 이름 또는 Resource를 받습니다. create/get/update는 `MetadefProperty`, list는 generator, 개별·전체 delete는 None을 반환합니다. Update와 개별 Delete는 property 인자를 `_get_id`로 줄여 generic 호출에 넘깁니다. Python의 mutable Resource·dirty cache·반환 객체 identity를 Go의 독립 DTO나 실제 204 acknowledgement와 동일하게 취급하지 않습니다.

## 필수 scalar와 raw keyword

namespace와 property 이름은 nonempty valid UTF-8, 최대 80 rune입니다. colon·Unicode·space를 유지하고 path component를 한 번 escape합니다. 정확한 `.`·`..`, slash·backslash·percent·query·fragment 문자와 ASCII control/DEL을 거부하며 자동 trim·UUID 해석·이름 검색을 하지 않습니다. route와 source/context는 callback 전에, optional rename과 payload는 callback 뒤 HTTP 전에 검사합니다.

Create·Update의 `Type`와 `Title`은 nonnil이어야 합니다. Type은 nonempty valid UTF-8이고 Go가 enum을 고정하지 않습니다. Title은 valid UTF-8이면 empty도 명시적으로 허용합니다. Description은 optional pointer이며 nil 생략, empty·multiline 전송, 길이 제한을 추가하지 않습니다. 서버의 [schema와 deserializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L254-L389)가 최종 허용 여부를 판단합니다.

`CreateOpts.Attributes`·`UpdateOpts.Attributes`는 `map[string]json.RawMessage`입니다. nil은 keyword 없음, nonnil empty map도 추가 keyword 없음이며 각각의 key는 valid UTF-8, 값은 valid UTF-8의 JSON이어야 합니다. null·큰 숫자·array·object 같은 literal JSON을 보존합니다. exact key name·type·title·description·self·schema·created_at·updated_at·namespace_name은 SDK가 소유하므로 거부합니다. 그 외 key는 로컬 whitelist 없이 flat body로 보냅니다. `readonly` 같은 schema keyword는 이 raw 입력에 해당합니다.

`WithCreateAttributes`·`WithUpdateAttributes`는 `map[string]any`, singular `WithCreateAttribute`·`WithUpdateAttribute`는 key와 any 값을 받아 helper 생성 시 `encoding/json`으로 snapshot합니다. FullOpts의 raw map과 입력 형태가 다릅니다. plural helper는 원래 map key의 UTF-8을 marshal 전에 검사하고 attribute map 전체를 교체하며 singular helper는 해당 key를 추가·교체합니다. marshal 실패는 보존했다가 옵션 적용 시 `ErrInvalidOption`과 원인으로 반환하며 HTTP를 보내지 않습니다. `json.RawMessage`를 any 안에 넣으면 literal JSON을 유지할 수 있습니다. 일반 Go 값은 `encoding/json`의 실제 변환 규칙을 따르며 원래 값의 malformed UTF-8을 추가로 검출한다고 보장하지 않습니다.

raw keyword를 전송할 수 있다는 사실은 서버의 저장·round trip을 보장하지 않습니다. 고정 [Schema.raw](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/schema.py#L86-L103)는 `additionalProperties=false`이고 [property schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L692-L806)와 [WSME PropertyType](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/model/metadef_property_type.py#L23-L62)은 표현 범위가 다릅니다. unknown request key는 보통 400이며, schema의 required와 WSME confidential, number minimum/maximum과 WSME integer, arbitrary default와 WSME bytes 표현이 일치하지 않습니다. SDK는 JSON Schema·WSME를 재현하거나 fractional/unknown/default 값이 그대로 복원된다고 약속하지 않습니다.

`Update(ctx, current, ...)`는 current 경로에 한 번 PUT하며 body name은 기본 current, optional Name으로 rename을 지정합니다. Type·Title을 다시 보내야 하고 생략한 description·keyword는 full replacement로 reset됩니다. [controller.update](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L185-L221)가 저장 schema 전체를 교체합니다. SDK는 GET/RMW·merge·PATCH·rollback을 하지 않습니다. Python [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)의 dirty-only/no-op 동작은 별도 계약입니다.

## Get query와 dictionary 목록

`GetOpts.ResourceType`은 nil이면 query를 생략하고 empty pointer도 `resource_type=`로 보냅니다. valid UTF-8/control-free literal 값을 query로 한 번 encode합니다. nonempty 값을 지정하면 고정 [server.show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L98-L146)가 resource-type association의 prefix를 확인하고 요청 property 이름에서 prefix를 제거해 조회합니다. prefix가 없거나 맞지 않으면 NotFound가 될 수 있습니다. SDK는 association discovery나 로컬 prefix 재작성을 하지 않습니다. [공식 Show Property 설명](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#show-property-definition)도 이 query를 설명합니다.

`List`는 iterator 생성 시 HTTP를 보내지 않습니다. iteration마다 query와 body 없는 collection GET 하나를 실행합니다. 전체 UTF-8·JSON syntax와 required nonnull `properties` dictionary를 먼저 검사하고, value는 소비할 때만 strict nonnull Property object로 decode합니다. `MaxItems=0`은 로컬 cap 없음, 양수는 소비 수 제한, 음수는 HTTP 전 오류입니다. cap이나 break는 사용하지 않는 malformed value를 decode하지 않습니다. 빈 성공 `All`은 nonnil empty slice, 오류에서는 부분 slice를 버립니다.

dictionary는 wire encounter order로 소비하며 같은 key가 반복되면 첫 slot을 유지하고 마지막 value를 사용합니다. lexical sort를 추가하지 않습니다. `Property.Key *string`은 List에서만 제공하는 literal dictionary key provenance이며 CRUD에서는 nil입니다. canonical Name은 별도 optional field로 유지하고 raw Body에 이름을 seed하거나 덮어쓰지 않습니다. Key·Name·self·schema 어느 것도 다음 route를 자동 결정하지 않습니다.

고정 [MetadefProperty.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L86-L185)는 `params={}`로 한 번 GET하고 dictionary key를 name으로 seed한 뒤 value attribute가 이를 덮어쓸 수 있습니다. recognized Body/dict filter는 Python에서 로컬 적용합니다. Go는 이런 filter와 descriptor coercion·minLength/minItems 0·uniqueItems false 기본값을 추가하지 않습니다. [실제 server index](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L64-L96)와 [공식 List Properties](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-properties)는 dictionary 응답이며 Go가 next·Link·schema·marker·wire limit·sort·filter·fallback을 해석하거나 합성하지 않습니다.

## 모델·scope·옵션·오류 증거

Property의 Name·Type·Title·Description·Self·Schema와 Metadata.CreatedAt·UpdatedAt은 exact canonical optional nullable string pointer입니다. 전체 응답은 valid UTF-8의 nonnull JSON object이며 잘못된 nonnull canonical type은 atomic 오류입니다. raw keyword·null·큰 숫자의 precision은 독립 Metadata.Body에 남습니다. 날짜 parse·ID·namespace·Metadata.Links 합성·schema fetch·nested keyword normalization을 하지 않습니다. List의 Key도 독립 pointer이며 Name이나 Body와 alias하지 않습니다.

scope는 API/client/provider·type·endpoint·base·microversion을 수명 동안 고정하고 context를 저장하지 않습니다. callback 전·accepted body 뒤·소비하는 행 전후에 source/target/context를 재검사합니다. 각 요청은 최신 유효 ordinary source header를 복사하고 원래 provider의 live auth를 사용합니다. 6개 concrete 옵션 family의 FullOpts는 전체 설정을 교체하고 나중 옵션이 우선합니다. pointer·map·RawMessage bytes·option slice·callback config를 독립 복사하며 callback은 요청 또는 iteration마다 한 번 실행합니다. nil/error callback·보호된 header·invalid input은 HTTP 전에 오류입니다.

실제 Create201/Get·List·Update200만 decode합니다. accepted read·Close·UTF-8·JSON·model·context·source 오류는 typed Property를 반환하지 않고 `resource.ResponseError`에 실제 전체 Body·Header·StatusCode와 원인을 보존합니다. Delete·DeleteAll의 실제 204는 opaque Acknowledgement에 선택한 Namespace와 실제 증거를 남기며, Name은 개별 요청 child이고 DeleteAll에서는 nil입니다. accepted handling 실패에서도 ACK와 오류를 함께 반환합니다.

개별 Delete의 `IgnoreMissing *bool`은 nil이면 true입니다. true는 SDK가 직접 소유한 physical 404의 read·Close·context·source 검사가 모두 성공했을 때만 nil/nil로 끝내고 그 404에서만 native RetryFunc를 우회합니다. false는 native 오류·retry 정책을 유지합니다. DeleteAll은 모든 404를 오류로 유지합니다. hidden parent/property가 404로 표현될 수 있으므로 quiet 404는 실제 부재의 증거가 아닙니다. protected namespace 정책도 서버가 판단하며 SDK가 unprotect·enumerate·count·wait·cleanup을 수행하지 않습니다.

configured native pre-body retry·reauth·backoff·동일 target redirect 정책은 유지하고 method·origin·path·query·serialized body·owned response fields 또는 accepted status를 바꾸는 callback은 차단합니다. accepted body를 한 번 닫고 실패 뒤 replay하지 않습니다. read·Close·transport 원인과 `ctx.Err()`·custom `context.Cause`를 보존하며 native reauth 원인은 기존 `ErrOriginal`·`ErrReauth` 필드에서 확인합니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Glance commit `57f7dd9e76ef24e1e9013eceaa703bd442469a24`, native Gophercloud `v2.15.0`입니다. native에는 property CRUD/list API가 없어 이 leaf를 SDK에서 소유합니다. Python 전체 Resource·client filter·adapter/session parity나 실제 서버 schema·권한·DB·cloud side effect를 검증했다고 주장하지 않습니다.

소스·요청·응답 경계는 [core tests](core_test.go), [option tests](options_test.go), [외부 HTTP contracts](contracts_test.go)에서 검증합니다. Connection 경로와 생성 registry는 [Connection contract](../../../connection_image_metadef_properties_test.go)와 [generator contract](../../../internal/cmd/sdkgen/glance_metadef_properties_test.go)가 별도로 확인합니다. 이 테스트는 로컬 transport 증거이며 Python 실행이나 실제 cloud parity 검증은 아닙니다.
