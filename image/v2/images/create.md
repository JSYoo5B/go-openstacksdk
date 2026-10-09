# Glance native 이미지 생성

`service.API.Images.Create(ctx, images.CreateOpts{...}, options...)`는 Gophercloud `v2.15.0`의 native Create를 호출해 metadata만 가진 이미지를 만들고 native `*images.Image`를 반환합니다. 이미지 데이터 업로드·import·Task·대기는 하지 않습니다. 준비한 client는 `images.New(client).Create`를 사용합니다.

```go
visibility := images.ImageVisibilityPrivate
image, err := service.API.Images.Create(ctx, images.CreateOpts{
    Name: "ubuntu-24.04", DiskFormat: "qcow2", ContainerFormat: "bare",
    Visibility: &visibility, Tags: []string{"base"},
    Properties: map[string]string{"os_distro": "ubuntu"},
}, images.WithCreateField("hw_qemu_guest_agent", "yes"))
if err != nil {
    return err
}
fmt.Println(image.ID, image.Status)
```

파일 업로드·checksum·cloud 기본값까지 포함한 Python `create_image`는 [현대 생성](../../image-record-create.md)의 `CreateImageRecord`를, Cinder 볼륨 분기와 생성 후 대기는 [Cloud 이미지 생성](../../image-record-cloud.md#이미지-생성)을 사용합니다.

## 요청 본문

[ToImageCreateMap](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L195-L207)은 `BuildRequestBody`로 선언 필드를 직렬화합니다. `Name`은 필수라 비어 있으면 HTTP 전에 오류이고, 나머지 값 필드는 zero 값이면 생략합니다. `Visibility`, `Hidden`, `Protected`는 pointer라 false도 명시적으로 보낼 수 있습니다. 그다음 `Properties`의 문자열 값을 최상위 key로 복사하므로 `name` 같은 선언 key도 덮어쓸 수 있습니다.

`WithCreateField(key, value)`는 이 결과 뒤에 JSON 값을 추가하는 Go 확장입니다. 값은 옵션 생성 시 JSON으로 snapshot하며 `json.RawMessage`의 원문 숫자도 그대로 보냅니다. `CreateOpts`의 선언 key나 이미 본문에 있는 key(Properties 포함)와 겹치면 HTTP 전에 `resource.ErrInvalidOption`입니다.

## 응답과 오류

POST는 client의 ResourceBase/Endpoint 아래 `images`로 보내고 [기본 OkCodes](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L209-L219)는 201뿐입니다. 200·202·409 등은 native `gophercloud.ErrUnexpectedResponseCode`이고 SDK는 `resource.OperationError{Operation: "Create", Resource: "images"}` 문맥만 더합니다. 성공 응답은 [native 조회](get.md#native-projection)와 같은 header merge와 `Image.UnmarshalJSON` projection을 거칩니다. native 재시도·재인증 정책은 [native 삭제 가이드](delete.md#오류와-native-재시도)와 같습니다.
