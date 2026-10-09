# Image

Glance v2 이미지의 조회, iterator, 삭제, 상태 대기와 메타데이터 생성·직접 업로드·staging과 import, checksum 검사를 포함한 다운로드를 제공합니다. `conn`과 `ctx`는 [전체 README](../README.md)처럼 준비합니다. Go 조각은 `fmt`, `os`, `time`, `image`, `resource` 등을 필요한 만큼 import한 오류 반환 함수 안에서 사용합니다.

## openstacksdk 대응

| openstacksdk | go-openstacksdk |
|---|---|
| `conn.image.get_image(id)` | `service.Images.Get(ctx, id)` |
| `conn.image.find_image(name_or_id, ignore_missing=False)` | `service.Images.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(false))` |
| `conn.image.images(status="active")` | `service.Images.List(ctx, resource.WithStatus("active"))` |
| `conn.image.delete_image(id)` | `service.DeleteImage(ctx, resource.ID(id))`; [삭제 옵션·결과 비교](delete.md) |
| `conn.image.delete_image(id, store="archive")` | `service.DeleteImage(ctx, resource.ID(id), image.WithDeleteImageStore("archive"))` |
| `conn.image.download_image(record_or_id, output=path_or_file, stream=False)` | `service.DownloadImageRecord(ctx, image.ImageRecordDownloadRequest{ID: id, Filename: path}, ...)`; [owned 네 모드·checksum 비교](image-record-download.md) |
| `conn.image.download_image(id, output=file)` | `service.DownloadTo(ctx, resource.ID(id), writer, ...)`; [다운로드 옵션·결과 비교](download.md) |
| `conn.image.create_image(name, data=data, use_import=False, allow_duplicates=True)` | `service.Upload(ctx, image.UploadImageRequest{Name: name, Data: reader}, ...)` |
| `conn.create_image(..., wait=True, timeout=300)` | 업로드 호출에 `image.WithWait(resource.WithTimeout(5*time.Minute))` 추가 |
| `conn.image.create_image(..., use_import=True)` | `service.CreateAndImport(ctx, image.CreateAndImportRequest{...}, ...)`; [단계별 옵션·결과 비교](create-import.md) |
| `conn.image.upload_image(container_format="bare", disk_format="qcow2", data=file, **attrs)` | `service.UploadImageRecord(ctx, image.ImageRecordUploadRequest{Data: file, Attributes: attrs}, image.WithImageRecordUploadContainerFormat("bare"), image.WithImageRecordUploadDiskFormat("qcow2"))`; [owned 생성·전송 비교](image-record-upload.md) |
| `conn.image.import_image(record, ...)` | `service.ImportImageRecord(ctx, image.ImageRecordImportRequest{Record: record}, ...)`; [owned 입력·옵션·응답 비교](image-record-import.md) |

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

`Service.CreateAndImport`는 metadata 생성·직접 staging·import 접수를 연결하고 remote URL/Glance 소스와 저장소 선택도 concrete 옵션으로 처리합니다. [생성·import 사용법](create-import.md)은 실제 단계별 접수 증거와 부분 실패, 기본값·소유권·선택적 active 대기를 설명합니다. Swift task 업로드, checksum 계산·검증, 기존 이미지 재사용과 vendor/cloud 설정의 자동 적용은 남은 비교 범위입니다. 다운로드 결과의 `Body`는 사용자가 닫아야 합니다.

[image_test.go](image_test.go)는 Glance의 envelope 없는 응답, 추가 Properties, 실패 상태를 검증합니다. [upload_test.go](upload_test.go)는 업로드 HTTP 계약과 검증·실패 정책을, [upload_retry_test.go](upload_retry_test.go)는 단일 PUT·지연된 Body.Close·현재 offset·Reader 소유권과 원인 오류 보존을 검증합니다. [서버 생성 통합 테스트](../server_create_test.go)는 Compute에서 이미지 이름을 해석하는 과정을 검증합니다.

`image.WaitForState(ctx, service.Images, ref, target)`와 Image v2 leaf의 `WaitForState`는 정확한 ERROR 실패 상태와 무제한 SDK timeout을 기본으로 사용합니다. `WaitForDelete`는 삭제 요청 없이 기본 120초 동안 삭제 완료를 관찰합니다. 생성·업로드의 기존 `WithWait` 기본값은 유지합니다. [서비스별 대기 비교](../docs/service-waits.md)에 옵션·context·Python 대응을 설명합니다.

`service.CreateTask/GetTask/Tasks/AllTasks`는 생성 입력 기본값과 concrete 옵션, 실제 응답 원문, Task 목록 순회를 제공합니다. [Task 생성·조회·목록 사용법](tasks.md)에 Python 대응과 native API의 차이를 설명합니다.

`service.ImageTasks/AllImageTasks`는 필수 이미지 ID 또는 이름과 concrete 옵션으로 이미지별 Task를 한 번 조회합니다. [이미지별 Task 사용법](image-tasks.md)에 삭제 여부·시각, 로컬 목록 한도와 Python 대응을 설명합니다.

`service.GetImage/ListImages/AllImages`는 이미지 ID·이름 조회와 lazy 목록 순회를 제공합니다. [이미지 조회·목록 사용법](images.md)에 nullable 값·확장 속성·concrete 필터·서버 next의 반복 값 보완과 Python 대응을 설명합니다.

`service.GetImageRecord/ListImageRecords/AllImageRecords/FindImageRecord`는 Image의65필드 declared Resource와 실제 Wire를 분리하고 descriptor 변환·properties packing·공통 pager·ID/이름·숨김 이미지 검색을 처리합니다. [레코드의 Python/Go 비교와 독립 main](image-records.md)에 seed·기본값·concrete 옵션·부분 결과와 기존 typed API의 차이를 설명합니다.

`service.WaitForImageRecordStatus/WaitForImageRecordDelete`는 owned ImageRecord를 받아 초기 상태·fresh GET·시간 예산·삭제404의 마지막 관측을 처리합니다. [레코드 대기의 Python/Go 비교와 독립 main](image-record-waits.md)에 concrete duration/header/attribute/callback 옵션, nullable 상태·부분 결과와 기존 typed 대기의 차이를 설명합니다.

`service.UpdateImageRecord`와 `conn.UpdateImageRecord`는 ID 또는 SDK가 반환한 ImageRecord에 변경 속성을 적용해 same-value 요청을 생략하고 JSON Patch를 자동 생성합니다. [레코드 수정의 Python/Go 비교와 독립 main](image-record-update.md)에 raw baseline·properties 교체·concrete 옵션·응답과 다음 수정의 관계를 설명합니다.

`service.UpdateImagePropertiesRecord`와 Connection의 같은 메서드는 SDK Record의 기존 properties를 복사하고 keyword 변환·참조 이미지 검색·meta overlay·조건부 자동 PATCH를 처리합니다. [속성 helper 비교·독립 main](image-record-properties.md)에 bool/no-op, ordered JSON·map 옵션과 literal ID의 properties seed 실패를 설명합니다.

`service.AddImageRecordTag/RemoveImageRecordTag`는 literal ID 또는 owned ImageRecord를 받아 태그 하나의 접수와 성공 후 로컬 tags 변경을 구분합니다. [레코드 태그의 Python/Go 비교와 독립 main](image-record-tags.md)에 concrete header 옵션, 중복·첫 일치 삭제, 기존 조회 증거와 후행 오류의 접수를 설명합니다.

`service.UpdateImage/SetImageProperties`는 concrete 패치와 필드별 helper로 이미지를 수정합니다. [이미지 수정 사용법](update.md)에 패치 순서·속성 이름 escape·raw 값·빈 변경의 실제 응답과 Python dirty Resource·coercion의 차이를 설명합니다.

[Native Image PATCH의 Python/Go 비교·독립 main](v2/images/update.md)은 `service.API.Images.Update`의9종 concrete patch·기본200·nil/empty·옵션 교체와 native partial Extract를 설명합니다.

`service.UploadImageRecord`와 `conn.UploadImageRecord`는 명시 형식·raw constructor 속성으로 metadata를 생성하고 borrowed reader를 한 번 전송합니다. [owned 업로드 비교·독립 main](image-record-upload.md)에 기본 total-size 추론·signed size·optout, pending/translated Record와 Metadata·Uploaded 접수·후행 오류를 설명합니다. 형식·name·visibility 기본값과 필수 reader 조건은 이 owned API에 추가하지 않습니다.

`service.UploadImage`는 concrete 옵션으로 메타데이터를 생성하고 caller Reader를 한 번 전송합니다. [직접 업로드 사용법](upload-image.md)에 기본값·명시 size·단계별 응답과 Python 자동 길이 추론의 차이를 설명합니다.

Task는 `service.API.Tasks.WaitForTask(ctx, resource.ID(id), options...)` 또는 `conn.ImageV2(ctx)`의 `Tasks`에서 기다립니다. [Task 전용 사용법](v2/tasks/README.md)은 success·failure·120초·2초 기본값, 정확한 396 오류의 재생성, 같은 시간 제한으로 새 ID 조회, 실제 응답과 부분 실패를 설명합니다. Python의 cached Task 대신 fresh ID를 받고 `tasks.WithTaskWait...` 옵션을 사용합니다. 공통 이미지 상태 대기와 import 제출은 각각 별도 호출입니다.

`service.ImportImageRecord`와 `conn.ImportImageRecord`는 SDK Record의 private formats·pending Body를 유지하고 초기 조회 없이 한 번 import POST를 제출합니다. [owned import 비교·독립 main](image-record-import.md)은 raw JSON 기본값·presence·완전한 remote triple·store constructor/private ID와 singular JSON+header, ordinary3/copy admin method 분기·opaque ACK/부분 오류를 설명합니다. 아래 native typed import profile과 구분해서 사용합니다.

`service.API.ImageImport.ImportImage(ctx, ref, options...)`는 기존 이미지의 ID/Name 참조를 해결하고 format을 조회한 뒤 import를 제출합니다. 이미 보유한 native Image는 `ImportKnownImage`로 조회 없이 사용할 수 있습니다. [Import 사용법](v2/imageimport/README.md)에 기본 glance-direct, web/remote 소스, 저장소 선택, 명시적 false와 실제 202 응답을 설명합니다. 결과는 접수 응답이며 이미지가 active라는 뜻은 아닙니다. 준비된 이미지의 완료 대기는 `service.API.Images.WaitForState(ctx, ref, "active", options...)`로 별도 선택합니다.

`service.StageImageRecord`와 `conn.StageImageRecord`는 queued SDK Record의 private state를 유지하고 filename 또는 borrowed reader를 한 번 staging한 뒤 필수 metadata GET을 수행합니다. [owned staging 비교·독립 main](image-record-stage.md)은 literal ID의 queued seed 실패·명시적 GET, 기본 total-size 추론·signed size·optout, owned file close·borrowed data 유지와 partial receipt를 설명합니다. 아래 `StageImage/StageKnownImage`의 native typed profile과 구분해서 사용합니다.

`service.API.ImageData.StageImage(ctx, ref, data, options...)`는 queued 이미지를 확인하고 `io.Reader`를 한 번 전송한 뒤 최신 이미지를 조회합니다. `StageKnownImage`는 이미 보유한 native Image의 ID/status를 복사해 첫 조회를 생략합니다. [Staging 사용법](v2/imagedata/README.md)은 선택적 크기 헤더, caller의 Reader 소유권, 실제 PUT204 접수와 후속 GET200 결과·부분 실패를 설명합니다. staged 데이터의 import 제출과 active 상태 대기는 이어서 선택할 수 있습니다.

`service.DownloadImageRecord`와 `conn.DownloadImageRecord`는 ID·owned Record·raw constructor Resource의 필수 metadata fetch 뒤 file/writer/buffer/stream을 처리합니다. [owned 다운로드 비교·독립 main](image-record-download.md)에 sparse overlay의 private ID·after-GET hash 선택·registry/factory·200..399 응답·부분 오류·borrowed writer와 caller stream Close를 설명합니다. 기본 memory에는 전체 크기 cap이 없으며 stream-only는 자동 checksum proof를 만들지 않습니다.

`service.DownloadTo(ctx, ref, writer, options...)`는 fresh metadata를 먼저 조회하고 기본 1MiB chunk로 writer에 전송하며 가능한 checksum을 검사합니다. [다운로드 사용법](download.md)은 저장소 우선순위·hash 우선순위·caller의 writer 소유권과 실제 metadata/binary 응답·바이트 수·부분 오류를 설명합니다. 기존 raw `service.API.ImageData.Download`를 사용할 때는 반환된 Body를 직접 닫습니다.

`service.DeleteImage(ctx, ref, options...)`는 ID 조회 없이 전체 이미지 또는 선택한 저장소 복사본을 삭제합니다. Name은 기존 정확한 이름 조회로 해석하며 기본 미존재는 `nil, nil`입니다. [삭제 사용법](delete.md)은 concrete 옵션, 실제204 응답 원문·헤더·상태와 응답 처리 오류, 저장소의 마지막 복사본 정책을 설명합니다. 기존 `Images.Delete`와 native `API.Images.Delete`는 계속 사용할 수 있습니다.

`service.DeleteImageRecord`와 `conn.DeleteImageRecord`는 literal ID 또는 SDK ImageRecord의 private identity를 받아 전체 이미지와 특정 store 삭제를 처리합니다. [owned 삭제 비교·독립 main](image-record-delete.md)은 whole Record+ACK와 store ACK-only, opaque200..399·기본 missing·whole user/store admin 분기를 설명합니다. [native 삭제 가이드](v2/images/delete.md)는 `API.Images.Delete`의 error-only·기본202/204·raw ID와 native Read/Close·재시도 경계를 설명합니다.

`service.DeactivateImageRecord/ReactivateImageRecord`와 Connection의 같은 메서드는 공통 owned ID/Record·concrete header 옵션으로 고정 action을 제출합니다. [owned 상태 action 비교·독립 main](image-record-actions.md)은 Python 기본4xx 처리와 Go200..399/native 오류, local pending 상태·이전 receipt와 이번 ACK의 경계를 설명합니다.

`service.API.ServiceInfo.ListStores/AllStores(ctx, options...)`는 기본 저장소 목록과 선택적 상세 목록을 읽고, `GetImportInfo(ctx)`는 현재 서버의 import 방식을 조회합니다. [ServiceInfo 사용법](v2/serviceinfo/README.md)은 Python 대응·concrete 옵션·실제 응답 증거·목록 제어를 설명합니다. discovery 결과로 import 실행을 자동 제한하거나 캐시하지 않습니다.

`ServiceInfo.GetImportInfoRecord/ListStoreRecords/AllStoreRecords`는 import와 기본 stores의 declared Resource·Wire·응답 증거를 구분하고 descriptor 변환·현재 location·공통 페이지·로컬 Body 조건을 처리합니다. [레코드 사용법과 독립 main](v2/serviceinfo/records.md)은 concrete 옵션·explicit zero limit·부분 결과와 후속 admin 상세 목록의 범위를 설명합니다.

`service.API.ServiceInfo.GetUsageInfo(ctx, options...)`는 인증된 프로젝트의 limit·사용량을 조회합니다. [사용량 사용법](v2/serviceinfo/usage.md)은 nullable int64·unknown resource·빈 응답과 concrete header 옵션을 설명합니다.

`service.AddTag/RemoveTag(ctx, ref, tag, options...)`는 태그 한 개를 변경하며 `DeactivateImage/ReactivateImage(ctx, ref, options...)`는 고정 action을 제출합니다. [태그·상태 변경 사용법](mutations.md)은 공통 concrete header 옵션·literal escaping·미존재 오류·actual204 acknowledgement와 Python cache 차이를 설명합니다.

`service.AddImageLocation(ctx, ref, url, options...)`는 외부 저장소 URL과 선택 validation hash를 제출하고 실제202 응답을 보존합니다. `GetImageLocations(ctx, ref, options...)`는 유한 배열을 한 번 조회합니다. [Location 사용법](locations.md)에 서버의 비동기 검증·concrete 옵션·Python Resource와의 차이를 설명합니다.

`service.GetImageSchema/GetImagesSchema`와 member·task·metadef의 기존 16개 strict schema getter는 공통 `GetSchemaOption`으로 조회합니다. [Schema 사용법](schemas.md)에 고정 경로·실제200·기존 typed `Schema` 필드와 raw Body 보존을 설명합니다. Source의 class별 descriptor 변환은 별도 정책이며 반환값의 links·이름·schema URL은 자동으로 따라가지 않습니다.

`service.GetSchemaRecord(ctx, image.SchemaMetadefNamespace)`는 같은 고정 경로에서 Source class의 nullable·dict/bool/list 값을 가진 Resource와 실제 Wire·응답 receipt를 분리합니다. ordinary와 metadata 16종은 concrete `SchemaKind`로 선택합니다. [Python/Go 비교·독립 main](schema-records.md)에 raw 기본값·ordered 별칭·현재 location·accepted 빈/불완전 JSON과 오류의 차이를 설명합니다. 기존 strict getter의 API·200 정책은 유지합니다.

`service.GetImageCache/QueueImage/CacheDeleteImage/ClearCache/CachedImageNodes/CleanCache/PruneCache`는 캐시 조회와 정리를 제공합니다. [캐시 사용법](cache.md)에 concrete 옵션·미존재 삭제 기본값·zero target의 cache/queue 선택·실제202/204/200과 passive node URL을 설명합니다. Queue 결과는 작업 접수이며 완료 여부는 서버의 후속 상태로 확인합니다.

`service.AddImageMember/GetImageMember/UpdateImageMember/RemoveImageMember/FindImageMember/ListImageMembers/AllImageMembers`는 부모 이미지 Ref와 멤버 ID를 명시해 사용합니다. [멤버 사용법](members.md)에 직접 ID 조회·미존재 기본값·local MaxItems·timestamp 문자열과 기존 `API.Members.InImage`의 차이를 설명합니다.

`service.GetImageMemberRecord/AddImageMemberRecord/UpdateImageMemberRecord/RemoveImageMemberRecord`는 raw declared 속성과 fresh constructor, actual200..399 응답·부분 오류·삭제404 기본값을 제공합니다. [Python/Go 비교·독립 main](member-record-mutations.md)에 같은 Connection API와 owner/recipient 사용법을 설명합니다.

`service.ListImageMemberRecords/AllImageMemberRecords/FindImageMemberRecord`는 Member의9필드 declared Resource·실제 Wire, 일반 페이지 순회와 ID/name 검색 fallback·중복 판정을 제공합니다. [레코드 목록·검색의 Python/Go 비교와 독립 main](member-records.md)에 무옵션 기본값·concrete Go 확장·부분 결과·직접 GET seed를 설명합니다.

`service.API.MetadefNamespaces.Create/Get/Update/Delete`는 literal namespace 이름을 사용하고 `List/All`은 서버가 광고한 다음 페이지를 순회합니다. [Namespace 사용법](v2/metadefnamespaces/README.md)에 생략한 PUT 필드의 초기화·rename·`limit=0`·일반 헤더와 raw 응답을 설명합니다.

`service.API.MetadefObjects.InNamespace(ctx, namespace)`는 HTTP 없이 namespace와 서비스 대상을 고정합니다. 범위의 `Create/Get/Update/Delete/DeleteAll/List/All`과 [Object 사용법](v2/metadefobjects/README.md)에 nested raw property·생략/빈 값·PUT 교체·유한 목록·삭제 기본값을 설명합니다.

`service.API.MetadefProperties.InNamespace(ctx, namespace)`는 고정 namespace의 `Create/Get/Update/Delete/DeleteAll/List/All`을 제공합니다. [Property 사용법](v2/metadefproperties/README.md)에 flat JSONSchema 입력·필수 Type/Title·raw와 any 옵션의 차이·dictionary key provenance·삭제 기본값을 설명합니다.

`MetadefResourceTypes.ListRecords/AllRecords`와 namespace scope의 같은 메서드는 declared Resource·실제 Wire/receipt, semantic 필터와 공통 paging을 제공합니다. [Python/Go 비교·독립 main](metadef-resource-types-records.md)에서 두 Connection 경로와 옵션을 설명합니다. `MetadefProperties` scope의 `GetRecord`는 초기 Resource 속성·descriptor 기본값과 변환·현재 location을 SDK가 처리하고 실제 응답과 구분합니다. [Property 레코드 조회·독립 main](metadef-property-records.md)과 [기존 raw 조회](metadef-property.md)에 두 API의 사용법을 설명합니다.

`MetadefObjects` scope와 `MetadefNamespaces`의 `ListRecords/AllRecords`는 raw Body 기본값·descriptor 변환·현재 location·공통 페이지 순회와 부분 결과를 제공합니다. [객체·namespace 목록의 Python/Go 비교와 독립 main](metadef-object-namespace-record-lists.md)에 두 진입점과 concrete 옵션 사용법을 설명합니다.

namespace association의 [owned 생성·삭제와 독립 main](metadef-resource-type-association-mutations.md)은 명시한6개 raw Body 값,8필드 declared Resource와 실제 응답을 구분합니다. `CreateRecord`는 name/id의 SDK 필수 조건 없이 dirty Body를 보내고, `DeleteRecord`는 복사한 identity와 고정 parent로 실제 opaque ACK를 남깁니다. 기본 physical404 receipt와 Source의 mutable Resource 재사용 차이도 설명합니다.

`MetadefProperties` scope의 `ListRecords/AllRecords`는 finite dictionary를 선언 Resource로 변환하고 로컬 조건·raw cap·부분 결과를 처리합니다. `DeleteRecord/DeleteAllRecords`는 같은 concrete 입력·기존 옵션으로 실제 opaque 삭제 receipt를 반환합니다. [목록 비교·독립 main](metadef-property-record-list.md)과 [삭제 예제](v2/metadefproperties/README.md#owned-목록과-삭제)를 참고하세요.

property의 owned `CreateRecord/UpdateRecord`는 명시한 raw Body 속성만 전송하고 같은 descriptor view·현재 location을 반환합니다. 변경 속성이 없는 수정은 HTTP 없이 local snapshot을 제공합니다. [Python/Go 비교·독립 main](metadef-property-record-write.md)에 fresh identity·dirty payload·응답과 raw CRUD의 차이를 설명합니다.
