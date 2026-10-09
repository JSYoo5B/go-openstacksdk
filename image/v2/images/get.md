# Glance native 이미지 조회

`service.API.Images.Get(ctx, imageID)`는 Gophercloud `v2.15.0`의 native Get과 `Extract()`를 호출하고 native `*images.Image`를 반환합니다. 준비한 client는 `images.New(client).Get`을 사용합니다. 전체 v2 facade는 아래처럼 반환값과 오류를 먼저 처리한 뒤 `api.Images.Get(ctx, imageID)`로 같은 메서드를 사용합니다.

```go
api, err := conn.ImageV2(ctx)
if err != nil { return err }
```

이미 준비한 Image Service와 context에서 호출합니다.

```go
image, err := service.API.Images.Get(ctx, imageID)
if err != nil {
    var native gophercloud.ErrUnexpectedResponseCode
    if errors.As(err, &native) {
        fmt.Printf("HTTP %d, expected=%v, response bytes=%d\n",
            native.Actual, native.Expected, len(native.Body))
    }
    return err
}
fmt.Println(image.ID, image.Status, image.SizeBytes, image.Properties, image.OpenStackImageImportMethods)
```

위 코드에는 Go `errors`, `fmt`와 `github.com/gophercloud/gophercloud/v2`를 사용합니다. Python `get_image`와 같은 descriptor 기본값·location·properties packing이 필요하면 [owned 레코드 조회](../../image-records.md)의 `GetImageRecord`를, Cloud 계층의 strict 조회는 [Cloud 이미지 조회](../../image-record-cloud.md)의 `GetImageRecordByID`를 사용합니다.

## 요청과 성공 응답

ID string을 [upstream Get](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L228-L233)의 `ServiceURL` 경로로 그대로 전달합니다. SDK가 ID를 검증하거나 하나의 URI segment로 escape하지 않으므로 빈 ID·slash·query delimiter도 upstream URL 처리에 맡깁니다. 이름 조회와 fallback 목록도 없습니다. 기존 client의 ResourceBase/Endpoint, MoreHeaders, microversion과 live auth token을 사용하고 기본 Accept는 `application/json`입니다.

기본 성공 status는 200뿐입니다. 201·203·204·301·404·503 등은 native `gophercloud.ErrUnexpectedResponseCode`이며 SDK는 이를 `resource.OperationError{Operation: "Get", Resource: "images"}`로 감쌀 뿐 다른 형식으로 바꾸지 않습니다. owned `resource.ResponseError`나 Record를 만들지 않습니다.

## native projection

[Extract](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/results.go#L161-L179)는 응답 header의 `OpenStack-image-import-methods`와 `OpenStack-image-store-ids`를 body에 덮어쓴 뒤 [Image.UnmarshalJSON](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/results.go#L101-L154)으로 decode합니다. 두 값은 전체 공백을 지운 뒤 쉼표로 나누고 빈 항목을 버립니다. `size`는 숫자면 int64, null이면 0이며 문자열 같은 다른 타입은 오류입니다. 이때 native Extract가 이미 일부 채운 Image를 오류와 함께 반환할 수 있습니다.

선언되지 않은 key는 `Properties`에 모으고 `self`, `size`, 두 header key는 제외합니다. body의 문자열 `properties` 필드는 `Properties["properties"]`로 남습니다. 시간 필드는 RFC3339 `time.Time`으로 decode합니다. 이 projection은 Python Image Resource의 descriptor 변환·`properties` packing·location과 다르며 SDK가 그 차이를 메우지 않습니다.

## 오류와 재시도

transport·context·응답 decode 오류는 원래 원인을 그대로 감쌉니다. native `RetryFunc`, 401 재인증, 429 `RetryBackoffFunc`와 거부 응답 Read/Close 처리도 upstream 정책을 따르며 SDK가 source guard나 재시도를 추가하지 않습니다. 같은 정책의 세부는 [native 삭제 가이드](delete.md#오류와-native-재시도)와 같습니다.
