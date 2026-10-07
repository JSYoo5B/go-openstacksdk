# Glance 이미지 PATCH·property upsert

`image.Service.UpdateImage`은 순서가 있는 `ImagePatch` 배열을 보내고, `SetImageProperties`는 전달한 map의 key를 정렬하여 `add` operation으로 보냅니다. 둘 다 실제 PATCH 200 응답을 [ImageInfo](images.md)로 반환합니다. 기존 [generated/native Update](v2/images/README.md)는 그대로 사용할 수 있습니다.

[고정 공식 API-ref](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L559-L682)는 `/v2/images/{image_id}` PATCH와 정상 응답 200을 정의합니다. route는 API v2.0부터, 사용하는 `application/openstack-images-v2.1-json-patch` media type은 v2.1부터 제공됩니다. 서버는 deprecated v2.0 media type도 받지만 두 형식의 operation 표현과 `add` 동작이 다릅니다. SDK는 v2.1 media type을 사용하며 discovery·Schema 조회·version negotiation을 먼저 수행하지 않습니다.

| 호출·옵션 | 동작과 기본값 |
|---|---|
| `UpdateImage(ctx, ref, ...)` | Ref가 필수입니다. `Changes` 순서와 중복 path를 보존합니다. 같은 값을 보내도 local 비교나 dedup을 하지 않습니다. |
| `SetImageProperties(ctx, ref, ...)` | Properties의 exact wire key를 Go string 순서로 정렬하여 `add`합니다. 기존 값도 v2.1 `add`로 덮어씁니다. GET·read/modify/write·Python type coercion을 수행하지 않습니다. |
| nil/empty Changes·Properties | nonnil `[]` PATCH를 보내 실제 `ImageInfo`를 받습니다. [서버 controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L617-L655)는 빈 changes일 때 save하지 않습니다. |
| FullOpts / plural helper | FullOpts는 해당 options 전체를 교체합니다. `WithUpdateImageChanges`는 전체 배열을, `WithSetImagePropertiesProperties`는 전체 map을 교체합니다. |

아래 첫 함수는 수정 가능한 queued 이미지의 예제입니다. owner·visibility·format 변경 권한과 이미지 상태는 서버가 판단합니다. 이 예제의 `build`는 add 이후 replace하고, `legacy`는 같은 PATCH에서 add 이후 remove하므로 기존 custom property의 존재를 가정하지 않습니다. 두 함수는 독립적인 요청입니다.

```go
package examples

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/gophercloudsdk/image"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func updateQueuedImageExample(ctx context.Context, svc *image.Service, imageID, ownerID string) (*image.ImageInfo, error) {
    return svc.UpdateImage(ctx, resource.ID(imageID),
        image.WithUpdateImageOpts(image.UpdateImageOpts{
            Headers: map[string]string{"X-Example": "ordered-image-patch"},
        }),
        image.WithUpdateImageHeaders(map[string]string{"X-Trace-Id": "image-update"}),
        image.WithUpdateImageHeader("X-Request-Id", "image-update-example"),
        image.WithUpdateImageChanges(image.ImagePatch{
            Op: "add", Path: "/build", Value: json.RawMessage(`"draft"`),
        }),
        image.WithUpdateImageChange(image.ImagePatch{
            Op: "replace", Path: "/build", Value: json.RawMessage(`"ready"`),
        }),
        image.WithUpdateImageName("release-image"),
        image.WithUpdateImageVisibility("private"),
        image.WithUpdateImageProtected(false),
        image.WithUpdateImageHidden(false),
        image.WithUpdateImageOwner(ownerID),
        image.WithUpdateImageContainerFormat("bare"),
        image.WithUpdateImageDiskFormat("qcow2"),
        image.WithUpdateImageMinDisk(0),
        image.WithUpdateImageMinRAM(0),
        image.WithUpdateImageTags("release", "ready"),
        image.WithUpdateImageField("build~origin", "ci"),
        image.WithUpdateImageFields(map[string]json.RawMessage{
            "build/id": json.RawMessage(`"2026.10"`),
            "legacy": json.RawMessage(`"temporary"`),
        }),
        image.WithUpdateImageRemoveField("legacy"),
    )
}

func setImagePropertiesExample(ctx context.Context, svc *image.Service, imageID string) (*image.ImageInfo, error) {
    return svc.SetImageProperties(ctx, resource.ID(imageID),
        image.WithSetImagePropertiesOpts(image.SetImagePropertiesOpts{
            Headers: map[string]string{"X-Example": "property-upsert"},
        }),
        image.WithSetImagePropertiesHeaders(map[string]string{"X-Trace-Id": "image-properties"}),
        image.WithSetImagePropertiesHeader("X-Request-Id", "image-properties-example"),
        image.WithSetImagePropertiesProperties(map[string]json.RawMessage{
            "team": json.RawMessage(`"platform"`),
            "release": json.RawMessage(`"2026"`),
        }),
        image.WithSetImagePropertiesProperty("release", "2026.10"),
    )
}
```

singular Change·typed helper·Fields는 현재 Changes 끝에 append합니다. Fields는 전달한 map 안의 key를 정렬하여 append하고 nil/empty map은 아무 operation도 추가하지 않습니다. typed helper는 exact wire key의 `add`입니다. Name·Visibility·Owner·ContainerFormat·DiskFormat은 string, Protected·Hidden은 bool, MinDisk·MinRAM은 int64, Tags는 string array입니다. 빈 string·false·0은 명시적으로 전송하며 `WithUpdateImageTags()`는 nonnil 빈 `[]`로 tags를 비웁니다. Property helper는 한 key를 교체합니다. 전체 map 교체는 이 helper보다 먼저 사용해야 앞에서 지정한 값을 유지할 수 있습니다.

map·slice·raw bytes는 helper 생성 시 snapshot하고 callback 전후에도 복사합니다. UpdateImage callback에는 초기화된 Headers가, SetImageProperties callback에는 초기화된 Headers와 Properties가 제공됩니다. nil callback·custom 오류·marshal 오류는 cause를 유지하며 Name lookup이나 PATCH 전에 중단합니다. 공유 입력을 동시에 변경하지 않는 범위에서 helpers를 재사용·병렬 실행할 수 있습니다. `any` helper는 생성 시 `encoding/json`으로 한 번 serialize하며 Python descriptor·문자열 변환을 적용하지 않습니다. raw 입력은 큰 숫자·null을 그대로 표현할 수 있습니다.

`ImagePatch.Op`는 exact `add`, `replace`, `remove`입니다. add/replace의 Value는 nonnil·nonempty·UTF8·valid JSON이어야 하며 explicit JSON `null`도 값입니다. remove의 Value는 반드시 nil이어야 합니다. nonnil 길이 0 RawMessage는 nil로 바꾸지 않고 오류로 처리합니다. nullable base field를 null로 설정하는 일과 해당 field를 remove하는 일은 다릅니다. [서버](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L657-L713)는 없는 custom property의 replace/remove에 409, domain base attribute의 remove에 403을 반환합니다.

Field key는 nonempty UTF8이고 [Python `str.strip()`에 해당하는 서버 token 정리](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1406-L1442)로 identity가 바뀌지 않아야 합니다. SDK는 앞뒤 whitespace와 U+001C–U+001F를 제거하는 대신 거부합니다. 내부 whitespace/control은 JSON data로 남습니다. key의 `~`는 `~0`, `/`는 `~1`로 escape합니다. generic Path는 `/`로 시작하는 literal JSON Pointer이며 decoded token이 비어 있거나 같은 edge 문제가 있으면 오류입니다. `~1` 다음 `~0`을 한 번 decode하며 알 수 없는 escape를 거부합니다. `%2F`는 percent decode하지 않습니다. URI path·URL과 JSON Pointer는 별개입니다.

generic Path의 여러 token은 허용하지만 위치 depth/index·operation 지원은 서버가 판단합니다. [parser](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1450-L1551)는 일반 root field의 depth 1, locations add/remove의 depth 2, locations replace의 depth 1을 적용하며 tags 원소 append는 구현하지 않습니다. SDK는 readonly·reserved key·enum·name 255·음수 min 값·custom property type·location 권한을 추론하거나 Schema validation을 대체하지 않습니다. 기본 서버 custom value schema는 string이므로 arbitrary JSON을 전송할 수 있다는 사실이 숫자·array·null의 성공적인 저장을 보장하지 않습니다. 배포의 custom schema와 property protection도 서버 정책입니다.

Python의 실제 두 proxy 호출은 다음과 같습니다.

```python
from openstack.image.v2.image import Image

def update_image_example(conn, image_id):
    cached = Image.existing(id=image_id, name="old-name", team="platform")
    updated = conn.image.update_image(cached, name="release-image", is_protected=False)
    did_update = conn.image.update_image_properties(
        image=updated, meta={"team": "platform"}, min_ram=0,
    )
    return updated, did_update
```

[고정 Python `update_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)는 전달한 mutable Resource를 재사용하고 attrs를 변경한 뒤 commit합니다. ID string이면 current image를 fetch하지 않고 ID-only Resource를 만듭니다. [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L2011)은 dirty 상태가 없으면 HTTP를 생략하고, 있으면 original/current body의 JSON Patch를 path 순서로 정렬합니다. unknown client Properties는 remote root로 펼쳐집니다. caller가 가진 상태와 현재 서버 상태가 같다는 뜻은 아닙니다.

[`update_image_properties`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1139-L1179)는 기존 `image.properties`를 seed로 사용하고 bool을 반환합니다. kwargs의 min_disk/min_ram/size/virtual_size는 int, protected/is_protected/tags는 raw, 다른 nonnull 값은 string으로 바꾸며 meta가 그 결과를 덮어씁니다. truthy kernel/ramdisk는 Connection image lookup 이후 *_id로 바뀝니다. copied properties가 빈 경우에만 False이고, nonempty unchanged properties이면 True여도 Resource.commit이 PATCH를 생략할 수 있습니다. Go는 이런 cached dirty state·alias·coercion·lookup·boolean 반환을 복제하지 않습니다.

Ref는 필수입니다. 직접 ID는 safe literal UTF8 segment로 한 번 escape하며 empty·dot/dotdot·slash·backslash·control·invalid UTF8를 거부하고 UUID·길이를 강제하지 않습니다. space·colon·percent·question·hash는 literal data입니다. Name은 명시적인 Go 확장입니다. options·value·pointer 검증 이후 캡처한 client의 기존 정확한 `Images.ResolveID`를 사용하고 고정한 ID로 PATCH합니다. resolver의 native model/body/pager·중복/없음 정책은 기존 경계에 남습니다.

source/provider·Type·Endpoint·base·microversion·ordinary headers를 callback 전에 캡처하고 원래 provider auth는 live로 유지합니다. source/context는 작업 전·Name 후·accepted 응답 후·decode 후에 확인합니다. Name lookup에는 캡처한 source/option header 정책을 사용하고, 이후 private client에 v2.1 Content-Type과 `Accept: application/json`을 설정합니다. [native Request](https://github.com/gophercloud/gophercloud/blob/v2.15.0/service_client.go#L144-L163)가 service headers로 request headers를 덮어쓰기 때문입니다. 원래 client/header는 변경하지 않습니다. 명시적으로 설정된 native RetryFunc의 advanced MoreHeaders 변경과 same-target redirect 정책은 유지됩니다.

actual 200만 decode합니다. reused `ImageInfo`는 25개 canonical 필드의 nullable string/bool/int64·Tags/Locations와 independent Body/Properties/header/status를 소유합니다. 날짜·URL·응답 ID/Name은 passive이고 요청값을 seed하거나 일치를 강제하지 않습니다. Properties는 noncanonical root field의 raw projection이며 wire `properties` object를 강제하지 않습니다. Header를 pseudo JSON field로 주입하지 않습니다. canonical 오류는 atomic decode 오류입니다.

accepted read·Close·context/custom cause·source·JSON/model 오류는 whole actual body/header/status의 `ResponseError`와 nil model을 유지하며 body를 한 번 닫고 accepted PATCH를 replay하지 않습니다. native unexpected status·transport·prebody retry/reauth·method/URL/body/status guard는 기존 정책을 유지합니다. 응답 self/file/schema나 raw property는 다음 요청 주소가 아닙니다. nonempty changes는 서버가 순서대로 적용하고 save하지만 위치 backend 효과까지 포함하는 cloud rollback·snapshot consistency·원자성을 보장하지 않습니다. 후속 GET·wait·checksum·cleanup도 수행하지 않습니다.

계약은 [core 테스트](update_core_test.go), [options 테스트](update_options_test.go), [외부 HTTP 계약 테스트](update_contracts_test.go), [Connection 통합 테스트](../connection_image_update_test.go), [native·registry 보존 테스트](../internal/cmd/sdkgen/glance_image_update_test.go)에서 고정 source와 로컬 fixture로 검증합니다.
