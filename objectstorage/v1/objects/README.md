# Swift 리소스와 container 범위

container와 object는 이름이 곧 식별자입니다. 공통 Collection은 `resource.ID(name)`과 `resource.Name(name)`을 구분합니다. ID는 바로 HEAD/DELETE를 실행하고, Name은 prefix 목록을 검색한 뒤 정확한 이름을 비교합니다. object 이름의 `/`, 공백, `?`, `#`, `%`는 SDK가 URL에 인코딩하므로 원래 이름을 전달합니다.

| 작업 | openstacksdk | Go |
|---|---|---|
| container metadata 조회 | `conn.object_store.get_container_metadata(name)` | `service.Containers.Resources.Get(ctx, name)` |
| container 목록 | `conn.object_store.containers()` | `service.Containers.Resources.List(ctx)` |
| object 목록 | `conn.object_store.objects(container)` | `scope.List(ctx)` |
| object metadata 조회 | `conn.object_store.get_object_metadata(object, container=container)` | `scope.Get(ctx, key)` |
| object 삭제 | `conn.object_store.delete_object(object, container=container)` | `scope.Delete(ctx, resource.ID(key))` |

```go
// ctx context.Context, service *swiftv1.Service를 사용하는 오류 반환 함수 안에서
scope, err := service.Objects.InContainer(ctx, resource.ID("assets"))
if err != nil { return err }
object, err := scope.Get(ctx, "images/logo ?.png")
if err != nil { return err }
fmt.Println(object.Name, object.Bytes, object.ContentType, object.Metadata)

for object, err := range scope.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(object.Name, object.Hash)
}

header, err := scope.Create(ctx, "docs/readme.txt", objects.CreateOpts{
    Content: strings.NewReader("hello"),
    Metadata: map[string]string{"Owner": "sdk"},
})
if err != nil { return err }
_ = header

download, err := scope.Download(ctx, resource.ID("docs/readme.txt"))
if err != nil { return err }
defer download.Close()
_, err = io.Copy(destination, download)
if err != nil { return err }
```

`swiftv1`은 `gophercloudsdk/objectstorage/v1`, `objects`는 `gophercloudsdk/objectstorage/v1/objects`, `resource`는 `gophercloudsdk/resource`입니다. `fmt`, `strings`, `io`는 표준 라이브러리이며 `destination`은 `io.Writer`입니다. Connection의 `ObjectStorage(ctx)`가 서비스 객체를 제공합니다.

ID로 parent를 지정하면 사전 조회를 하지 않습니다. `resource.Name(...)`은 container를 한 번 조회해 범위를 고정합니다. 목록의 `ContainerResource/ObjectResource`에는 Swift가 목록에 제공한 필드가 들어 있습니다. `Get`은 HEAD 요청으로 `Details`, 사용자 `Metadata`, 전체 `Header`까지 채웁니다. 목록에서 얻은 모델의 `Details/Metadata/Header`는 nil입니다. metadata key의 대소문자는 HTTP 헤더의 canonical 형태를 따릅니다.

delimiter 목록에서 얻은 디렉터리 항목은 `Object.Subdir`에 보관하며 실제 object 조회 응답을 뜻하지 않습니다. 상태 필드가 없으므로 `Wait`와 `WithStatus`는 `ErrUnsupported`입니다. `WaitDeleted`는 HEAD의 404를 삭제 완료로 판단합니다. Delete는 기본적으로 404를 무시하고, 403 등의 오류는 원래 응답 오류를 보존합니다.

범위 객체는 `Update`, `Copy`, `Download`에도 같은 container와 object 참조를 사용합니다. `CopyOpts.Destination`은 `/container/object` 형식으로 지정합니다. 입력 확장은 기존 API의 `objects.With...Header/Query`를 전달합니다. `Create`는 업로드 응답 헤더를 반환하며 서버의 객체를 추가 HEAD로 읽지 않습니다. 특정 object version, SLO/DLO와 temp URL 등의 API 입력은 전체 API에서 사용할 수 있으며, 이 공통 Collection의 조회·삭제 기본값은 현재 object version입니다.
