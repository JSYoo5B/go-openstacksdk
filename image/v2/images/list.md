# Glance native 이미지 목록

`service.API.Images.List(ctx, options...)`는 Gophercloud `v2.15.0`의 native List pager를 SDK의 lazy `iter.Seq2[*images.Image, error]`로 순회합니다. 준비한 client는 `images.New(client).List`를 사용합니다. 각 행은 [native 조회](get.md)와 같은 `Image.UnmarshalJSON` projection을 거칩니다.

```go
for image, err := range service.API.Images.List(ctx,
    images.WithListOptions(images.ListOpts{Status: images.ImageStatusActive, Limit: 50}),
    images.WithListQuery("size_min", "1073741824")) {
    if err != nil {
        return err
    }
    fmt.Println(image.ID, image.Name, image.SizeBytes)
}
```

위 코드에는 Go `fmt`와 `github.com/JSYoo5B/go-openstacksdk/image/v2/images`를 사용합니다. Python `images(**query)`와 같은 descriptor 변환·로컬 Body 필터·location이 필요하면 [owned 레코드 목록](../../image-records.md)의 `ListImageRecords`를, Cloud 계층의 deleted 제외 목록은 [Cloud 이미지 목록](../../image-record-cloud.md)의 `AllCloudImageRecords`를 사용합니다.

## query

`WithListOptions`의 `ListOpts`는 [ToImageListQuery](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L98-L123)로 직렬화합니다. zero 값 필드는 보내지 않으며 Tags는 `tag`를 반복합니다. `CreatedAtQuery`, `UpdatedAtQuery`는 RFC3339 시각 앞에 `gt`, `gte`, `lt`, `lte` 같은 filter를 붙입니다. `WithListQuery(key, value)`는 이 결과 뒤에 추가 query를 덧붙이는 Go 확장입니다.

고정 [BuildQueryString](https://github.com/gophercloud/gophercloud/blob/v2.15.0/params.go#L418-L421)은 `int` 필드만 숫자 query로 바꾸므로 int64인 `SizeMin`, `SizeMax`는 값이 있어도 전송되지 않습니다. SDK는 native 동작을 바꾸지 않습니다. 크기 범위가 필요하면 위 예제처럼 `WithListQuery("size_min", ...)`, `WithListQuery("size_max", ...)`를 사용합니다.

## 페이지와 중단

첫 요청은 client의 ResourceBase/Endpoint 아래 `images`로 보내고 기본 Accept, MoreHeaders, microversion, live token을 사용합니다. 응답 body의 `next`는 [nextPageURL](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/urls.go#L43-L65)이 경로와 query만 취해 client ServiceURL 기준 주소로 다시 만듭니다. 그래서 다른 host를 가리키는 next도 같은 서비스 주소로 요청합니다.

`images`가 빈 페이지는 next가 있어도 순회를 끝냅니다. 본문 없는 204 응답은 native pager가 JSON을 먼저 decode하므로 `io.EOF` 오류가 됩니다. 뒤 페이지의 HTTP 오류는 이미 반환한 행 뒤에 native `ErrUnexpectedResponseCode`로 나오고, 행 decode 오류도 그대로 반환합니다. 호출자가 순회를 멈추면 다음 페이지를 요청하지 않습니다. 이 순회의 오류에는 SDK operation 문맥을 추가하지 않으며 nil 옵션 같은 준비 오류만 `List/images` 문맥으로 감쌉니다.
