# Image

Glance v2 이미지의 조회, iterator, 삭제, 상태 대기와 메타데이터 생성부터 직접 업로드까지 묶는 작업을 제공합니다. `conn`과 `ctx`는 [전체 README](../README.md)처럼 준비합니다. Go 조각은 `fmt`, `os`, `time`, `image`, `resource` 등을 필요한 만큼 import한 오류 반환 함수 안에서 사용합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.image.get_image(id)` | `service.Images.Get(ctx, id)` |
| `conn.image.find_image(name_or_id, ignore_missing=False)` | `service.Images.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(false))` |
| `conn.image.images(status="active")` | `service.Images.List(ctx, resource.WithStatus("active"))` |
| `conn.image.delete_image(id)` | `service.Images.Delete(ctx, resource.ID(id))` |
| `conn.image.create_image(name, data=data, use_import=False, allow_duplicates=True)` | `service.Upload(ctx, image.UploadImageRequest{Name: name, Data: reader}, ...)` |
| `conn.create_image(..., wait=True, timeout=300)` | 업로드 호출에 `image.WithWait(resource.WithTimeout(5*time.Minute))` 추가 |

조회 대상은 이미지 메타데이터이며 이미지 데이터 다운로드 자체가 아닙니다. [공식 Image API](https://docs.openstack.org/openstacksdk/latest/user/proxies/image_v2.html)

## 조회와 목록

`service.Images.FindIdentity(ctx, "ubuntu")`는 ID GET부터 시도하고 GET400·403·404 뒤
정확한 ID/이름으로 일반 목록을 검색합니다. 정상적으로 끝난 일반 목록에 없을 때만
`os_hidden=true`인 숨김 이미지 목록을 한 번 더 검색합니다. 두 번째 검색에는 자동 이름
필터나 추가 GET을 넣지 않습니다. 오류·중복·취소는 즉시 반환하며 기본 미존재는 `nil, nil`입니다.
Python의 직접 `find_image`는 문자열과 `ignore_missing`을 받고, Go raw wire query는
추가 옵션입니다. [자동 조회 예제와 옵션](../docs/finding-identities.md#glance-숨김-이미지-검색)을 참고하세요.

아래 `resource.Name` 예제는 이름만 명시해 조회합니다. 자동 ID 판별과 숨김 이미지 검색이
필요하면 위의 `FindIdentity`를 사용합니다.

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

## 메타데이터 생성과 직접 업로드

Python:

```python
with open("ubuntu.qcow2", "rb") as data:
    uploaded = conn.create_image(
        name="ubuntu",
        data=data,
        disk_format="qcow2",
        container_format="bare",
        visibility="private",
        allow_duplicates=True,
        disable_vendor_agent=False,
        use_import=False,
        wait=True,
        timeout=300,
    )
```

Go:

```go
service, err := conn.Image(ctx)
if err != nil { return err }
data, err := os.Open("ubuntu.qcow2")
if err != nil { return err }
defer data.Close()

uploaded, err := service.Upload(ctx, image.UploadImageRequest{
    Name: "ubuntu",
    Data: data,
}, image.WithWait(resource.WithTimeout(5*time.Minute)))
if err != nil {
    if uploaded != nil { fmt.Println("created image:", uploaded.ID) }
    return err
}
fmt.Println(uploaded.ID, uploaded.Status)
```

`Upload`는 메타데이터를 `POST /images`로 생성하고 `PUT /images/{id}/file`에 `application/octet-stream`으로 데이터를 보냅니다. 기본값은 `disk_format: "qcow2"`, `container_format: "bare"`, `visibility: "private"`입니다. Python의 디스크 기본값은 cloud 설정에 따르지만 이 작업은 명시된 SDK 기본값을 사용하며, 데이터 내용이나 파일 확장자로 형식을 추측하지 않습니다.

항상 새 이미지를 생성하므로 이름 중복을 허용합니다. Python의 checksum 비교를 통한 기존 이미지 재사용과 vendor agent 속성 자동 추가는 이 작업에 포함되지 않습니다. `Images`의 조회·목록·삭제 인터페이스는 그대로 사용할 수 있습니다.

`WithDiskFormat`, `WithContainerFormat`, `WithVisibility`, `WithMinDisk`, `WithMinRAM`, `WithProtected`, `WithHidden`, `WithTags`, `WithProperties`, `WithProperty`, `WithUploadSize`, `WithWait`로 옵션을 설정합니다. 최소 디스크는 GB, 최소 RAM은 MB이며 음수를 허용하지 않습니다. 명시한 최소값 `0`과 boolean `false`도 JSON에 보존합니다. 지원 형식과 속성 스키마는 Glance가 검증합니다.

```go
uploaded, err := service.Upload(ctx, image.UploadImageRequest{
    Name: "arm-image",
    Data: data,
}, image.WithDiskFormat("raw"),
   image.WithMinDisk(0), image.WithMinRAM(512),
   image.WithVisibility(image.VisibilityShared),
   image.WithProtected(false), image.WithHidden(false),
   image.WithTags("linux", "arm64"),
   image.WithProperty("hw_architecture", "aarch64"))
if err != nil { return err }
fmt.Println(uploaded.ID)
```

Properties는 Glance의 envelope 없는 본문에 추가합니다. 값을 JSON으로 복사하므로 옵션 생성 후 map이나 slice를 변경해도 요청은 변하지 않습니다. 이름·형식·visibility 등 core 필드와 응답의 상태·크기 같은 필드는 속성으로 덮어쓸 수 없습니다. 빈 이름·nil Reader·잘못된 옵션과 대기 설정은 메타데이터 POST 전에 오류로 처리합니다.

업로드 PUT가 끝날 때까지 `Upload`는 실행을 계속합니다. `WithWait`를 추가하면 공통 정책으로 `active`까지 대기하며, 기본 대기 제한은 5분입니다. 이를 생략하면 메타데이터 생성 응답을 반환하므로 `Status`는 업로드가 끝났어도 `queued`일 수 있습니다. 업로드나 대기에 실패하면 생성 이미지와 원인 오류를 함께 반환하며, 이미지를 자동 삭제하지 않습니다. 생성 응답을 해석하다 실패하더라도 ID를 이미 얻었다면 해당 이미지와 해석 오류를 보존합니다. Python의 직접 업로드 실패 시 삭제 정책과 다른 점입니다. HTTP 오류는 `errors.As`, context·Reader 오류는 `errors.Is`로 확인할 수 있습니다.

## Reader 소유권

`Service.Upload`는 Reader의 현재 위치에서 읽으며 호출자가 제공한 Reader나 파일을 닫거나 seek하지 않습니다. 위 예제처럼 호출자가 `defer data.Close()`로 관리합니다. 같은 Reader를 다른 요청과 동시에 사용하지 마세요.

바이너리 PUT는 한 번만 시도하며 원래 provider의 재인증·`RetryBackoffFunc`·`RetryFunc`를 자동 적용하지 않습니다. HTTP redirect도 자동으로 따라가지 않습니다. 같은 스트림을 일부 소비한 후 다시 전송하면 데이터가 누락될 수 있기 때문입니다. `401`·`429` 또는 다른 업로드 오류는 생성 이미지와 원래 HTTP 오류로 반환합니다. 인증을 갱신하거나 입력을 다시 준비한 뒤 기존 이미지에 업로드하려면 `service.API.ImageData.Upload(ctx, uploaded.ID, reader)`를 사용하세요. 이 별도 API는 기본 Gophercloud 재시도·Reader 처리 정책을 사용합니다. 메타데이터 생성과 상태 조회에는 기존 provider의 재인증·재시도 설정을 유지합니다.

`WithUploadSize(byteCount)`는 `X-OpenStack-Image-Size` 헤더를 보내 Glance의 저장 공간 사전 할당을 돕습니다. 입력 크기를 알아내기 위해 데이터를 메모리에 모으거나 Reader의 끝으로 seek하지 않으며, 이 크기는 가상 디스크 크기와 구분합니다. Reader 자체의 막힌 읽기를 중단하는 처리는 호출자가 관리합니다.

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

## 전체 API와 남은 작업

생성·속성 수정, 데이터 업로드·다운로드, import, task, 멤버 관리는 `service.API`의 [Image v2 API](v2/README.md)에서 제공합니다. `service.API.Images`, `ImageData`, `ImageImport`, `Tasks`, `Members`에서 각 호출을 사용하며, 멤버는 `Members.InImage(ctx, ref)`로 부모 이미지를 고정할 수 있습니다. 서버 생성의 이미지 이름 해석은 이 패키지의 Find를 사용합니다.

stage→import, 원격 URL import, 여러 저장소 선택, Swift task 업로드, checksum 계산·검증을 묶는 상위 작업은 아직 없습니다. 각각의 API는 위 `service.API`를 통해 사용할 수 있습니다. 다운로드 결과의 `Body`는 사용자가 닫아야 합니다.

[image_test.go](image_test.go)는 Glance의 envelope 없는 응답, 추가 Properties, 실패 상태를 검증합니다. [upload_test.go](upload_test.go)는 업로드 HTTP 계약과 검증·실패 정책을, [upload_retry_test.go](upload_retry_test.go)는 단일 PUT·지연된 Body.Close·현재 offset·Reader 소유권과 원인 오류 보존을 검증합니다. [서버 생성 통합 테스트](../server_create_test.go)는 Compute에서 이미지 이름을 해석하는 과정을 검증합니다.

`image.WaitForState(ctx, service.Images, ref, target)`와 Image v2 leaf의 `WaitForState`는 정확한 ERROR 실패 상태와 무제한 SDK timeout을 기본으로 사용합니다. `WaitForDelete`는 삭제 요청 없이 기본 120초 동안 삭제 완료를 관찰합니다. 생성·업로드의 기존 `WithWait` 기본값은 유지합니다. [서비스별 대기 비교](../docs/service-waits.md)에 옵션·context·Python 대응을 설명합니다.

Task는 `service.API.Tasks.WaitForTask(ctx, resource.ID(id), options...)` 또는 `conn.ImageV2(ctx)`의 `Tasks`에서 기다립니다. [Task 전용 사용법](v2/tasks/README.md)은 success·failure·120초·2초 기본값, 정확한 396 오류의 재생성, 같은 시간 제한으로 새 ID 조회, 실제 응답과 부분 실패를 설명합니다. Python의 cached Task 대신 fresh ID를 받고 `tasks.WithTaskWait...` 옵션을 사용합니다. 공통 이미지 상태 대기와 import 제출은 각각 별도 호출입니다.

`service.API.ImageImport.ImportImage(ctx, ref, options...)`는 기존 이미지의 ID/Name 참조를 해결하고 format을 조회한 뒤 import를 제출합니다. 이미 보유한 native Image는 `ImportKnownImage`로 조회 없이 사용할 수 있습니다. [Import 사용법](v2/imageimport/README.md)에 기본 glance-direct, web/remote 소스, 저장소 선택, 명시적 false와 실제 202 응답을 설명합니다. 결과는 접수 응답이며 이미지가 active라는 뜻은 아닙니다. 준비된 이미지의 완료 대기는 `service.API.Images.WaitForState(ctx, ref, "active", options...)`로 별도 선택합니다.
