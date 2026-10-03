# Image 생성과 직접 업로드

`image.Service.UploadImage`는 `POST /images`로 metadata를 생성한 뒤, 응답의 canonical `id`로 `PUT /images/{id}/file`을 한 번 전송합니다. 기본값은 `qcow2`, `bare`, `private`입니다. 완료 후 metadata를 다시 조회하지 않으므로 `Image.Status`는 생성 응답의 `queued` 등으로 남을 수 있습니다. 서버 상태를 추측하여 `active`로 바꾸지 않습니다.

[pinned Python `Proxy.upload_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L583)는 deprecated이며 disk/container format을 명시해야 합니다. `_create`가 반환한 Image에 데이터를 지정하고 `Image.upload`를 호출한 다음 같은 Image 객체를 반환합니다. 이 공개 메서드에는 checksum 검증, Task/Swift, import, wait 또는 실패 시 삭제가 없습니다. 해당 동작은 별도의 `create_image` private 경로에 있습니다.

```python
# deprecated proxy 자체의 두 단계 동작을 비교하는 예제입니다.
# 새 Python 코드는 pinned SDK 안내에 따라 create_image를 검토하세요.
with open("image.qcow2", "rb") as source:
    created = conn.image.upload_image(
        container_format="bare",
        disk_format="qcow2",
        data=source,
        name="upload-example",
        visibility="private",
        tags=["example"],
    )
print(created.id, created.status)  # PUT 이후 fetch한 metadata가 아닙니다.
```

Go에서는 생성 metadata와 binary acknowledgement를 각각 보존합니다. 다음 함수의 `knownSize`는 이미 알고 있는 업로드 byte 수이고, `owner`는 지정할 권한이 있을 때만 전달합니다. 예제는 옵션 전체 교체, field map 전체 교체, header 병합과 개별 override를 보여 줍니다.

```go
package main

import (
    "context"
    "encoding/json"
    "os"

    "gophercloudsdk/image"
)

func uploadFile(ctx context.Context, service *image.Service, filename string,
    knownSize *int64, owner *string) (*image.ImageUploadResult, error) {
    source, err := os.Open(filename)
    if err != nil { return nil, err }
    defer source.Close() // Reader는 호출자가 소유합니다.

    options := []image.ImageUploadOption{
        image.WithImageUploadOpts(image.ImageUploadOpts{
            Headers: map[string]string{"X-Request-Label": "example"},
            Fields: map[string]json.RawMessage{"build": json.RawMessage(`"initial"`)},
        }),
        image.WithImageUploadHeaders(map[string]string{"X-Trace-Label": "upload"}),
        image.WithImageUploadHeader("X-Request-Label", "direct-upload"),
        image.WithImageUploadFields(map[string]json.RawMessage{
            "build": json.RawMessage(`"release"`),
        }),
        image.WithImageUploadField("build", "2026-10-03"),
        image.WithImageUploadDiskFormat("qcow2"),
        image.WithImageUploadContainerFormat("bare"),
        image.WithImageUploadVisibility(image.VisibilityPrivate),
        image.WithImageUploadProtected(false),
        image.WithImageUploadHidden(false),
        image.WithImageUploadMinDisk(0),
        image.WithImageUploadMinRAM(0),
        image.WithImageUploadTags("example"),
    }
    if owner != nil {
        options = append(options, image.WithImageUploadOwner(*owner))
    }
    if knownSize == nil {
        options = append(options, image.WithoutImageUploadSize())
    } else {
        options = append(options, image.WithImageUploadSize(*knownSize))
    }
    result, err := service.UploadImage(ctx,
        image.UploadImageRequest{Name: "upload-example", Data: source}, options...)
    // err가 있어도 result의 완료된 metadata/acknowledgement를 확인할 수 있습니다.
    return result, err
}

func main() {}
```

`UploadImageRequest.Name`은 유효한 UTF-8이고 공백만으로 이루어지지 않아야 하며 실제 문자열은 그대로 전송합니다. `Data`는 nil 또는 typed nil일 수 없습니다. 이 검사는 option callback 전에 이루어지고, preflight는 Reader의 `Read`, `Close`, `Seek`, 길이 계산을 호출하지 않습니다.

`ImageUploadOpts`는 `Headers`, `Fields`, `Size`를 가진 concrete 옵션입니다. callback에는 초기화된 두 map이 제공되며 callback마다 결과를 복사합니다. `WithImageUploadOpts`는 전체 설정을, `WithImageUploadFields`는 override field map 전체를 교체합니다. nil/빈 field map은 override를 비우므로 기본 metadata는 유지됩니다. 개별 field/helper는 같은 exact key를 덮어쓰고, plural header는 ordinary header를 병합합니다. helper는 입력 map, slice, raw bytes와 pointer를 소유한 snapshot으로 보존하므로 호출자가 나중에 원본을 변경해도 재사용되는 옵션에 반영되지 않습니다.

metadata는 root JSON object입니다. `name`은 request가 소유하므로 Fields에 지정할 수 없습니다. 다른 canonical/custom key는 alias나 강제 `properties` envelope 없이 전달합니다. raw `null`, `false`, `0`, 음수 및 고정밀 숫자는 그대로 보존하며 server가 schema, enum, readonly, 범위와 권한을 검사합니다. 단, 최종 `disk_format`과 `container_format`은 비어 있지 않은 UTF-8 JSON string이어야 합니다. Visibility helper도 server enum을 대신 검사하지 않습니다.

field key는 비어 있지 않은 UTF-8 literal이며 Python `strip()`으로 달라지는 가장자리 공백과 U+001C–U+001F를 허용하지 않습니다. trim, alias, percent decode를 수행하지 않습니다. nil/빈 raw value, 잘못된 JSON/UTF-8은 POST 전에 실패합니다. `WithImageUploadField`의 `any` 값은 helper를 만들 때 Go `encoding/json`으로 한 번 encode하며 marshal 원인을 보존합니다. Python descriptor 변환이나 property coercion과 같은 의미를 보장하지 않습니다.

Size가 nil이면 PUT size header를 생략하고, 0을 포함한 nonnegative 값이면 `X-OpenStack-Image-Size`를 PUT에만 전송합니다. format이나 크기를 filename/Reader에서 추론하지 않습니다. [pinned `Image.upload`와 `get_file_size`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462)는 size를 생략하면 seek 가능한 파일을 끝까지 이동했다가 원래 위치로 돌려 전체 길이를 구할 수 있습니다. Go는 현재 cursor부터 읽고 caller의 Reader를 닫거나 이동시키지 않습니다. 전송량과 size의 일치 여부는 server가 검증합니다.

원래 service/provider, Type, Endpoint, reverse prefix, base, microversion과 ordinary header를 callback 전에 고정하고 단계마다 확인합니다. token은 실제 전송 시 원래 provider의 live authentication을 사용합니다. auth/media/size/length 관련 header는 SDK가 소유하며 원본 client의 map과 설정을 변경하지 않습니다. POST에는 private `Content-Type`/`Accept: application/json`, PUT에는 binary Content-Type과 빈 Accept를 적용합니다. metadata POST의 명시적 native `RetryFunc.MoreHeaders` 정책은 기존 advanced provider 경계로 남습니다.

POST는 실제 201만 받아들이고, `Metadata`에 실제 전체 bytes/header/status를 보존합니다. 기존 atomic `ImageInfo` decoder가 25 nullable canonical field와 독립적인 raw Body/Properties를 해석합니다. 날짜/URL/ID는 passive response data이며 header를 JSON field로 주입하지 않습니다. decode가 실패하면 Image는 제공하지 않습니다. 성공한 response의 canonical lowercase `id`만 ImageID와 PUT route로 사용하며 `Location`, `self`, `file`, 기타 ID alias는 routing 근거가 아닙니다. ID는 비어 있지 않은 안전한 UTF-8 path segment여야 합니다. 공간, colon, percent, `?`, `#` 등 literal은 한 번 escape하며 dot/dotdot, slash, backslash, control은 거부합니다. 잘못된 ID는 생성 증거와 Image를 남기고 PUT/Reader 사용 전에 실패합니다.

binary PUT은 timeout/transport 설정을 유지한 private client에서 한 번 전송합니다. SDK retry/backoff/reauth/redirect를 적용하지 않습니다. 실제 204만 acknowledgement로 인정하며 opaque body를 JSON으로 해석하지 않습니다. body read/Close/context/custom cause 또는 source check 실패가 있어도 받은 `Acknowledgement`와 이전 `Metadata`, `Image`, `ImageID`를 보존합니다. 응답 body는 한 번 닫고 원인을 합친 `resource.ResponseError`에서 실제 bytes/header/status를 확인할 수 있습니다. 예상하지 않은 HTTP status와 transport 오류는 native 원인을 유지합니다. caller Reader의 막힌 `Read`를 취소가 직접 해제하지는 않으므로 caller의 해제 협력이 필요할 수 있습니다.

[pinned API-ref](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-data.inc#L14)와 [현재 공식 API](https://docs.openstack.org/api-ref/image/v2/index.html#upload-binary-image-data)는 직접 binary PUT을 API v2.0부터, 정상 응답을 204로 설명합니다. queued 상태, formats, quota와 backend 정책은 server가 검사합니다. server는 실패 시 data를 복원하거나 삭제할 수 있어 두 단계 원자성이나 rollback을 보장하지 않습니다. SDK는 추가 GET/status gate/wait/cleanup을 수행하지 않습니다.

기존 `Service.Upload`, `UploadImageOption`, Reader request와 native/generated Create/ImageData.Upload는 그대로 사용할 수 있습니다. Python deprecated default/Resource alias·coercion·session/cache·자동 길이 추론까지 완전한 parity는 아닙니다. 이 facade는 명시적인 두 단계 요청과 실제 응답 증거를 제공합니다.
