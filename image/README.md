# Image

Glance v2 이미지의 조회, iterator, 삭제, 상태 대기를 제공합니다. `conn`과 `ctx`는 [전체 README](../README.md)처럼 준비합니다. Go 조각은 `fmt`, `resource` 등을 import한 오류 반환 함수 안에서 사용합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.image.get_image(id)` | `service.Images.Get(ctx, id)` |
| `conn.image.find_image(name, ignore_missing=False)` | `service.Images.Find(ctx, resource.Name(name))` |
| `conn.image.images(status="active")` | `service.Images.List(ctx, resource.WithStatus("active"))` |
| `conn.image.delete_image(id)` | `service.Images.Delete(ctx, resource.ID(id))` |

조회 대상은 이미지 메타데이터이며 이미지 데이터 다운로드 자체가 아닙니다. [공식 Image API](https://docs.openstack.org/openstacksdk/latest/user/proxies/image_v2.html)

## 조회와 목록

Python:

```python
image = conn.image.find_image("ubuntu", ignore_missing=False)
for image in conn.image.images(status="active"):
    print(image.id, image.name)
```

Go:

```go
service, err := conn.Image(ctx)
if err != nil { return err }
image, err := service.Images.Find(ctx, resource.Name("ubuntu"))
if err != nil { return err }
fmt.Println(image.ID, image.SizeBytes)

for image, err := range service.Images.List(ctx, resource.WithStatus("active")) {
    if err != nil { return err }
    fmt.Println(image.ID, image.Name)
}
```

이름이 중복되면 `ErrAmbiguous`입니다. 전체 slice가 필요하면 `service.Images.All(ctx, ...)`를 사용합니다. 응답은 Gophercloud 이미지 타입을 재사용하며, 추가 이미지 속성은 `image.Properties`에서 확인할 수 있습니다.

## 대기와 삭제

```go
ready, err := service.Images.Wait(ctx, resource.ID(image.ID), "active")
if err != nil { return err }
fmt.Println(ready.Status)

if err := service.Images.Delete(ctx, resource.ID(image.ID)); err != nil {
    return err
}
```

`Wait`는 모든 서비스에서 같은 형태를 사용하는 이 프로젝트의 공통 기능입니다. 상태 비교는 대소문자를 구분하지 않으며 `killed`·`deleted`를 실패로 처리합니다. 삭제된 ID 조회가 404면 `ErrNotFound`로 반환합니다.

## 전체 API와 남은 업로드 작업

생성·속성 수정, 데이터 업로드·다운로드, import, task, 멤버 관리는 `service.API`의 [Image v2 API](v2/README.md)에서 제공합니다. `service.API.Images`, `ImageData`, `ImageImport`, `Tasks`, `Members`에서 각 호출을 사용하며, 멤버는 `Members.InImage(ctx, ref)`로 부모 이미지를 고정할 수 있습니다. 서버 생성의 이미지 이름 해석은 이 패키지의 Find를 사용합니다.

이미지 메타데이터 생성, 데이터 업로드 또는 import, 상태 대기, 실패 정리를 묶는 상위 작업은 아직 없습니다. 다운로드 결과의 `Body`는 사용자가 닫아야 합니다.

[image_test.go](image_test.go)는 Glance의 envelope 없는 응답, 추가 Properties, 실패 상태를 검증합니다. [서버 생성 통합 테스트](../server_create_test.go)는 Compute에서 이미지 이름을 해석하는 과정을 검증합니다.
