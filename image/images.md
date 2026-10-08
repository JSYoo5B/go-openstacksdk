# Glance 이미지 조회·목록

`image.Service.GetImage`, `ListImages`, `AllImages`는 이미지의 canonical 필드와 raw JSON·실제 header·status를 소유하는 `ImageInfo`를 반환합니다. 기존 `Service.Images` collection과 [generated/native Image API](v2/images/README.md)는 계속 사용할 수 있습니다. 새 iterator의 이름은 `ListImages`입니다.

[고정 공식 API-ref](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L243-L308)는 개별 조회를 `/v2/images/{image_id}`, [목록](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L353-L550)을 `/v2/images`의 GET 200으로 정의합니다. 두 route는 API v2.0부터 제공되며 필터 기능은 이후 확장됐습니다. SDK는 version·policy·Schema discovery를 먼저 요청하지 않습니다.

| 호출·옵션 | 동작과 기본값 |
|---|---|
| `GetImage(ctx, ref, ...)` | Ref가 필수입니다. ID는 별도 lookup 없이 고정한 ID를 한 번 escape하여 GET합니다. Name은 기존 정확한 resolver 이후 새 GET을 수행합니다. 404는 오류입니다. |
| `ListImages` / `AllImages` | lazy iterator / 동일 iterator의 수집입니다. iteration마다 새로 준비합니다. 성공한 빈 All 결과는 nonnil 빈 slice이며 오류 시 nil/error입니다. |
| Scalar query | string·integer·bool pointer가 nil이면 생략합니다. explicit 빈 string, integer 0, bool false는 전송합니다. Marker의 빈 string은 생략합니다. |
| Local controls | `MaxItems = 0`은 무제한, 양수는 local cap, 음수는 사전 오류입니다. `SinglePage = false`가 기본이며 true는 첫 page만 소비합니다. 둘 다 wire query를 만들지 않습니다. |

아래 함수들은 서로 독립적인 요청 예제입니다. classic 정렬과 comma 형태의 modern 정렬은 별도 options에서 사용합니다. 필터 preset은 필요한 조건만 남겨 사용할 수 있습니다.

```go
package examples

import (
    "context"
    "net/url"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func getImageExample(ctx context.Context, svc *image.Service, imageID string) (*image.ImageInfo, error) {
    return svc.GetImage(ctx, resource.ID(imageID),
        image.WithGetImageOpts(image.GetImageOpts{}),
        image.WithGetImageHeaders(map[string]string{"X-Example": "image-get"}),
        image.WithGetImageHeader("X-Request-Id", "image-get-example"),
    )
}

func classicImagesExample(ctx context.Context, svc *image.Service, imageName, ownerID string) ([]*image.ImageInfo, error) {
    options := []image.ListImagesOption{
        image.WithListImagesOpts(image.ListImagesOpts{MaxItems: 0}),
        image.WithListImagesHeaders(map[string]string{"X-Example": "image-list"}),
        image.WithListImagesHeader("X-Request-Id", "image-list-example"),
        image.WithListImagesLimit(2),
        image.WithListImagesMarker(""),
        image.WithListImagesName(imageName),
        image.WithListImagesVisibility("private"),
        image.WithListImagesMemberStatus("all"),
        image.WithListImagesOwner(ownerID),
        image.WithListImagesStatus("active"),
        image.WithListImagesSizeMin(0),
        image.WithListImagesSizeMax(1 << 40),
        image.WithListImagesProtected(false),
        image.WithListImagesHidden(false),
        image.WithListImagesSortKeys("name", "created_at"),
        image.WithListImagesSortDirs("asc", "desc"),
        image.WithListImagesTags("ready", "approved"),
        image.WithListImagesCreatedAt("gte:2020-01-01T00:00:00Z"),
        image.WithListImagesUpdatedAt("lte:2030-01-01T00:00:00Z"),
        image.WithListImagesContainerFormat("bare"),
        image.WithListImagesDiskFormat("qcow2"),
        image.WithListImagesFilters(url.Values{
            "os_distro": {"ubuntu"}, "unused": {"remove-me"},
        }),
        image.WithListImagesFilter("architecture", "x86_64"),
        image.WithListImagesFilter("unused"),
        image.WithListImagesMaxItems(20),
        image.WithListImagesSinglePage(false),
    }
    rows := make([]*image.ImageInfo, 0)
    for value, err := range svc.ListImages(ctx, options...) {
        if err != nil {
            return nil, err
        }
        rows = append(rows, value)
    }
    return rows, nil
}

func modernImagesExample(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageInfo, error) {
    return svc.AllImages(ctx,
        image.WithListImagesID(imageID),
        image.WithListImagesSort("name:asc,created_at:desc"),
        image.WithListImagesMaxItems(20),
    )
}
```

`WithGetImageOpts`와 `WithListImagesOpts`는 해당 options 전체를 교체하며 마지막 helper가 우선합니다. map·slice·pointer는 helper 생성 시 snapshot하고 각 callback 전후에도 복사합니다. callback에는 초기화된 Headers·Filters가 제공되며 nil callback·custom 오류는 작업을 중단합니다. iterator는 생성 시 option slice를 소유하고 각 iteration에서 options를 적용합니다. caller가 공유 입력을 동시에 변경하지 않는 범위에서 재사용·병렬 실행할 수 있습니다.

`WithListImagesFilters`는 Filters 전체를 교체합니다. `WithListImagesFilter(key, values...)`는 한 key의 values를 교체하고, values가 없으면 그 key를 삭제합니다. FullOpts의 nil/empty value slice는 오류입니다. Filters는 unknown wire key를 위한 concrete `url.Values`이며 `limit`, `marker`, `name`, `tag`, `os_hidden`, `deleted`, local control 등 SDK가 소유한 key는 거부합니다. 알려진 key는 typed helper를 사용합니다. builder·임의 URL·Python alias나 client Body filter를 받지 않습니다.

query value는 UTF8 literal data로 escape하므로 explicit 빈 값·newline·percent·slash도 데이터로 전송할 수 있습니다. key는 비어 있지 않은 UTF8이고 control 문자를 금지합니다. SDK는 query value의 enum·UUID·날짜·최대 길이를 추론하지 않습니다. Limit·SizeMin/Max는 음수를 거부하지만 min/max 관계를 추론하지 않습니다. Tags·SortKeys·SortDirs는 순서·반복·빈 원소를 보존합니다. Sort pointer가 present이면 classic 정렬과 함께 사용할 수 없으며 SortDirs가 여러 개면 SortKeys 개수와 같아야 합니다. 나머지 정렬 문법·허용 값은 서버가 판단합니다.

Go는 nil query를 서버 기본값으로 채우거나 local cap을 limit hint로 전송하지 않습니다. [고정 controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L517-L585)는 기본 hidden=false·member_status=accepted·created_at desc를 적용합니다. limit 생략 시 [설정](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L282-L329)의 `limit_param_default`를 사용하고 `api_limit_max`로 제한합니다. 고정 source의 기본값은 각각 25·1000이며 배포 설정은 다를 수 있습니다. explicit limit 0은 전송되고 서버가 처리합니다. tags의 `.strip()` 등 서버 normalization을 client에서 대신 수행하지 않습니다.

Python의 실제 두 proxy 호출은 다음과 같습니다.

```python
def images_example(conn, image_id, image_name, owner_id):
    image = conn.image.get_image(image_id)
    rows = list(conn.image.images(
        name=image_name, owner=owner_id, tag=["ready", "approved"],
        sort_key=["name", "created_at"], sort_dir=["asc", "desc"], limit=2,
    ))
    return image, rows
```

[고정 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1034-L1055)의 `get_image`는 string ID 또는 Image Resource를 `_get`으로 fetch하며 Name lookup을 하지 않습니다. 기존 mutable Resource를 재사용하더라도 fetch는 Adapter GET을 호출합니다. Adapter의 설정된 cache·session 동작까지 Go가 복제한다는 의미는 아닙니다. Go Name은 명시적 확장입니다. 캡처한 client의 기존 `Images.ResolveID`가 정확한 이름·중복/없음·native model/pager 정책으로 ID를 고르고, source와 ID를 확인한 뒤 새 raw GET을 합니다. resolver의 native body·pagination 소유권은 기존 경계에 남습니다.

직접 ID는 비어 있지 않은 안전한 UTF8 URI segment입니다. dot/dotdot·slash·backslash·control·invalid UTF8는 사전 오류이고 space·colon·percent·question·hash는 한 번 escape합니다. UUID나 read ID 길이를 강제하지 않습니다. 응답 ID/Name과 요청값의 일치를 강제하거나 response self/file/schema/direct_url을 다음 요청 주소로 사용하지 않습니다.

`ImageInfo`의 25개 canonical 필드는 exact case로 decode합니다. string·bool·int64 필드는 missing/null이면 nil이고 explicit 빈 string·false·0은 present입니다. int64는 부동소수점 변환 없이 정확히 읽습니다. Tags와 Locations는 missing/null이면 nil, `[]`이면 nonnil 빈 slice입니다. Tags 원소는 nonnull string이고 Locations 원소는 nonnull object로 [기존 ImageLocation](locations.md) decoder를 사용합니다. URL·metadata는 passive data입니다. 잘못된 nonnull canonical 타입은 receiver를 부분 변경하지 않는 decode 오류입니다.

날짜는 literal optional string입니다. 출력 enum·값 범위·image name 255·UUID·URL·Schema를 client에서 검증하지 않습니다. [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1756-L1827)는 schema를 validate하지 않고 filter하며 flat extra properties와 설정에 따른 locations/direct_url/stores를 반환합니다. `Metadata.Body`는 null·unknown field·큰 숫자의 raw field bytes를 보존합니다. `Properties`는 canonical key를 제외한 root field의 독립적인 raw projection이고 유효한 응답에서는 nonnil map입니다. wire `properties`가 object라고 강제하거나 root/body를 다시 구성하지 않습니다. `owner_id`, uppercase alias·`metadata`·unknown JSON도 raw-only입니다. Header는 raw로 보존하고 import/store header를 pseudo JSON field로 주입하지 않습니다. Metadata.Links는 passive raw-only입니다.

이 모델은 Python Image의 bool/int coercion·owner alias·unknown property packing·self 제거·header import parsing과 다릅니다. 기존 native `Image`의 time parsing·size 부동소수점 경유·remaining Properties normalization도 유지되며 새 모델과 섞지 않습니다. Python의 client Body/JMESPath filters·max_items limit hint·일반 Resource pager/cache 전체 parity는 남아 있습니다.

목록은 whole UTF8·JSON syntax·nonnull root object와 canonical `images` nonnull array를 먼저 검사하고 소비한 row만 decode합니다. cap·early break·SinglePage는 사용하지 않은 row와 next를 검사하지 않습니다. 다음 요청은 canonical body `next`만 사용하며 short/empty page에서도 next가 있으면 따라갑니다. row count·limit·HTTP Link·first/schema/self·last ID로 continuation을 만들거나 sorting/dedup/client filter를 추가하지 않습니다. next missing/null/empty는 끝이며, 소비해야 하는 nonempty next는 string이어야 합니다.

[서버 serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1855-L1871)는 `dict(request.params)`로 next를 만들어 반복 query를 하나로 줄입니다. Go는 이를 명시적으로 보완합니다. next의 nonmarker key set이 최초 query와 같고 각 values가 원래 ordered sequence 전체이거나 반복 값 중 하나의 singleton일 때만 허용한 뒤, 원래 query 전체를 복원하고 새 marker만 사용합니다. 임의 subset·추가·누락·변경은 오류입니다. raw Body는 서버가 보낸 값을 유지합니다. 이 복원은 Python/native의 반환 URL 추종과 다른 Go 정책입니다.

raw next URL은 UTF8·noncontrol·same origin·정확한 captured collection escaped path를 요구합니다. relative exact `/v2/images`만 캡처한 reverse prefix에 연결합니다. userinfo·opaque·fragment·network path·dot segment·path alias·다른 collection/origin과 query parse 오류를 거부합니다. marker는 하나의 nonempty UTF8 값이어야 하고 URL/marker cycle을 막습니다. [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L564-L574)는 policy로 줄어든 짧은 visible page에도 next를 줄 수 있으므로 길이만으로 종료하지 않습니다.

callback 전에 source/provider·Type·Endpoint·base·microversion과 ordinary headers를 캡처합니다. headers는 Name resolver와 첫 GET 동안 고정되며 option headers가 덮어씁니다. 다음 page는 고정 target/provider를 유지하며 최신 ordinary headers를 다시 읽습니다. 원래 provider auth는 live이고 요청 전·resolver 후·accepted 응답 후·소비 row 전후에 source/context를 확인합니다. native RetryFunc의 명시적 ordinary header 정책과 설정된 same-target redirect 동작은 유지됩니다.

actual 200만 decode합니다. accepted read·Close·context/custom cause·source·JSON/model/envelope/next/cycle 오류는 nil row와 whole actual body·header·status를 가진 `ResponseError`를 보존하며 한 번 닫고 accepted 요청을 replay하지 않습니다. Get/All은 부분 결과를 반환하지 않습니다. native unexpected status와 prebody retry/reauth/backoff·method/URL/body/status ownership guard는 기존 정책을 유지합니다. 404는 policy로 숨겨진 이미지일 수도 있으므로 없음의 증거로 해석하지 않습니다. 조회는 image import·checksum 검증·대기·cleanup·실제 cloud 실행을 포함하지 않습니다.

계약은 [core 테스트](images_core_test.go), [options 테스트](images_options_test.go), [외부 HTTP 계약 테스트](images_contracts_test.go), [Connection 통합 테스트](../connection_image_read_test.go), [native·registry 보존 테스트](../internal/cmd/sdkgen/glance_image_read_test.go)에서 고정 source와 로컬 fixture로 검증합니다.
