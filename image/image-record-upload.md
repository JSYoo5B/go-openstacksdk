# Glance 이미지 레코드 생성·업로드: Python과 Go

`Connection.UploadImageRecord`와 `image.Service.UploadImageRecord`는 새 이미지 metadata를 생성하고 caller의 `io.Reader`를 한 번 binary PUT로 전송합니다. SDK가 전체 Image descriptor view·raw 생성 속성·크기 추론·단계별 응답을 처리하므로 builder interface를 구현할 필요가 없습니다. Python의 deprecated `conn.image.upload_image`에 대응하는 owned API입니다.

`container_format`과 `disk_format`은 명시적으로 지정합니다. name·visibility·형식·데이터의 SDK 기본값은 없으며, 생략된 name이나 nil Data도 입력으로 표현할 수 있습니다. 이 가이드의 `qcow2`, `bare`, `private`는 예제가 선택한 값입니다. 실제 schema·형식 enum·권한은 서버가 판단합니다.

| 작업 | Python `conn.image.upload_image` | Go Service·Connection |
|---|---|---|
| 형식 지정 | `container_format="bare", disk_format="qcow2"` | `WithImageRecordUploadContainerFormat("bare")`, `WithImageRecordUploadDiskFormat("qcow2")` |
| 생성 속성 | `name=..., visibility=..., **attrs` | `ImageRecordUploadRequest.Attributes`와 `WithImageRecordUploadAttribute(s)` |
| caller 파일 | `data=file` | `ImageRecordUploadRequest{Data: file}`; caller가 열고 닫음 |
| 자동 크기 | size 생략 시 seek 가능한 파일의 전체 길이 추론 | 기본 동일한 전체 길이 추론, 현재 cursor 복원 |
| 명시 크기 | `size=0` 등 Python int | `WithImageRecordUploadSize(int64)`; 음수·0·양수 그대로 전달 |
| 크기 추론 끄기 | 직접 대응 옵션 없음 | `WithImageRecordUploadSizeInference(false)` |
| 반환값 | 같은 mutable 생성 Image; PUT 응답은 버림 | 독립 `Record`, `Metadata`, `Uploaded` |

호출 순서는 metadata `POST /images` → 생성 응답의 Resource 변환 → Data 할당·선택적 size 추론 → `PUT /images/{id}/file`입니다. 초기·최종 GET은 없습니다. PUT 응답으로 Record를 다시 변환하거나 status를 active로 바꾸지 않습니다. 완료 상태가 필요하면 [레코드 조회·대기](image-record-waits.md)를 별도 호출합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
with open("ubuntu.qcow2", "rb") as source:
    record = conn.image.upload_image(
        container_format="bare",
        disk_format="qcow2",
        data=source,
        name="ubuntu-example",
        visibility="private",
        filename="ubuntu.qcow2",
    )
print(record.id, record.status)
```

고정 Source는 이 deprecated 메서드에서 warning을 발생시킵니다. `filename`은 이 메서드의 formal 인자가 없어 생성 metadata property로 전달됩니다. 파일은 `data`에 caller가 열어 전달합니다. 현대 `create_image`의 filename·checksum·중복 재사용·Task/Swift/import 분기와 이 호출을 구분합니다.

## 독립 Go main

[설치 안내](../docs/install.md)를 따라 아래 코드를 `main.go`로 저장합니다. 기본 실행은 Connection에서 새 이미지를 한 번 생성·업로드합니다. `-service`는 같은 호출을 Image Service에서 실행합니다. 두 진입점을 중복 호출하지 않습니다. 2분 context는 인증·metadata 생성·binary 전송의 전체 시간 예산입니다.

```sh
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example -service -size 0
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example -infer-size=false
```

각 명령은 별도 실행 예이며 새 이미지를 생성합니다. 실제 cloud·파일·이름으로 값을 바꿉니다. `-size 0`은 명시적인 0을 전송하고, size 생략은 추론을 사용합니다. 음수 size도 그대로 표현되며 서버가 처리합니다. `-infer-size=false`는 명시 size가 없을 때 모든 SDK length/seek 추론을 끕니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    filename := flag.String("file", "", "required caller-owned image file")
    name := flag.String("name", "ubuntu-example", "image metadata name")
    visibility := flag.String("visibility", "private", "server image visibility")
    container := flag.String("container-format", "bare", "explicit image container format")
    disk := flag.String("disk-format", "qcow2", "explicit image disk format")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    size := flag.Int64("size", 0, "explicit signed size hint; omission uses inference")
    inferSize := flag.Bool("infer-size", true, "infer total seekable length when size is omitted")
    flag.Parse()
    if *filename == "" { log.Fatal("-file is required by this example") }
    sizePresent := false
    flag.Visit(func(value *flag.Flag) {
        if value.Name == "size" { sizePresent = true }
    })
    options := []image.ImageRecordUploadOption{
        image.WithImageRecordUploadOpts(image.ImageRecordUploadOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-upload-example"},
        }),
        image.WithImageRecordUploadContainerFormat(*container),
        image.WithImageRecordUploadDiskFormat(*disk),
        image.WithImageRecordUploadSizeInference(*inferSize),
    }
    if sizePresent { options = append(options, image.WithImageRecordUploadSize(*size)) }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *filename, *name, *visibility, *viaService, options); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, filename, name, visibility string, viaService bool, options []image.ImageRecordUploadOption) (err error) {
    source, err := os.Open(filename)
    if err != nil { return err }
    defer func() { err = errors.Join(err, source.Close()) }()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordUploadRequest{
        Data: source,
        Attributes: map[string]any{
            "name": name, "visibility": visibility,
            // This is metadata; the SDK does not open a filename attribute.
            "filename": filepath.Base(filename),
        },
    }
    var result *image.ImageRecordUploadResult
    if viaService {
        service, serviceErr := conn.Image(ctx)
        if serviceErr != nil { return serviceErr }
        result, err = service.UploadImageRecord(ctx, input, options...)
    } else {
        result, err = conn.UploadImageRecord(ctx, input, options...)
    }
    return printUpload(result, err)
}

func printUpload(result *image.ImageRecordUploadResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        for name, receipt := range map[string]*image.ImageUploadResponse{
            "metadata": result.Metadata, "uploaded": result.Uploaded,
        } {
            if receipt != nil {
                value[name] = map[string]any{
                    // Response bytes may be empty or non-JSON; []byte is base64.
                    "body": receipt.Body, "header": receipt.Header,
                    "status_code": receipt.StatusCode,
                }
            }
        }
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource, "wire": result.Record.Wire,
                "status_code": result.Record.StatusCode,
                "envelope_bytes": len(result.Record.Envelope),
                "import_methods": result.Record.ImportMethods,
            }
        }
    }
    if operationErr != nil { value["error"] = operationErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return errors.Join(operationErr, err) }
    fmt.Println(string(body))
    return operationErr
}
```

출력은 두 칸 들여쓰기 JSON입니다. 두 receipt의 `Body`는 `[]byte`이므로 base64로 출력합니다. binary ACK가 JSON·UTF-8이라고 가정하지 않습니다. 예제 검증은 독립 소비자의 컴파일 범위이며 실제 인증·OpenStack/Python 실행을 의미하지 않습니다.

## concrete 옵션과 생성 속성

`ImageRecordUploadRequest`는 `Data io.Reader`, `Attributes map[string]any`를 받습니다. 형식은 별도 `With…ContainerFormat`·`With…DiskFormat` 또는 full opts의 `json.RawMessage`로 지정합니다. 형식의 생략·null·false·0·빈 문자열/배열/object는 Python truthiness 기준으로 metadata POST 전에 실패합니다. nonempty 문자열뿐 아니라 true·nonzero number·nonempty 배열/object도 이 precheck를 통과하며 서버가 실제 형식 유효성을 판단합니다. name·visibility·owner·readonly·범위 값에 별도 client enum/default 규칙을 추가하지 않습니다.

| 옵션 | 동작 |
|---|---|
| `WithImageRecordUploadOpts(ImageRecordUploadOpts)` | option policy 전체 교체; request Attributes는 유지 |
| `WithImageRecordUploadContainerFormat(any)`, `WithImageRecordUploadDiskFormat(any)` | 형식의 raw JSON 값 지정; `nil`은 명시적 null |
| `WithImageRecordUploadAttribute(key, value)` | literal 생성 속성 한 개 덮어쓰기 |
| `WithImageRecordUploadAttributes(map[string]any)` | option 속성 map 병합 |
| `WithImageRecordUploadHeader(key, value)`, `WithImageRecordUploadHeaders(map[string]string)` | ordinary header 병합 |
| `WithImageRecordUploadSize(int64)` | signed 크기 hint 설정; 자동 추론 생략 |
| `WithoutImageRecordUploadSize()` | 앞선 explicit size만 해제; inference 설정은 유지 |
| `WithImageRecordUploadSizeInference(bool)` | `false`로 추론 중지; `true`로 허용 |

full opts는 `ContainerFormat`, `DiskFormat` raw JSON, `Attributes`, `Headers`, `Size *int64`, `DisableSizeInference bool`을 갖습니다. request 속성을 먼저 복사하고 option 속성을 같은 exact key에 덮어씁니다. 그 다음 formal 형식을 constructor 속성에 바인딩합니다. Python의 같은 formal 인자와 kwargs 중복 전달은 binding 오류지만, Go Attributes map의 같은 형식 key는 formal 옵션이 초기 값을 소유합니다. map·raw bytes·size pointer·helper 입력은 독립 snapshot으로 복사하며 option을 여러 호출에서 재사용할 수 있습니다.

형식 precheck는 constructor의 `__conflicting_attrs`·properties overlay 전에 한 번 수행합니다. truthy `__conflicting_attrs` object는 constructor 속성을 덮어쓰고 hook 자체를 제거하며, falsey hook은 unknown metadata로 남습니다. 고정 Source의 hook은 `.items()`를 사용하므로 truthy JSON 배열은 실패합니다. custom Python 객체의 `.items()` 실행은 공통 동적 class 범위입니다. `properties` object는 flat POST의 같은 wire key를 덮어씁니다. 따라서 formal 형식이 truthy여도 properties/hook이 그 값을 null로 바꾸면 null이 그대로 제출됩니다. 최종 wire 형식에 같은 검사를 다시 적용하지 않습니다. truthy string `properties`는 그 문자열 property로 보내며 그 밖의 scalar와 falsey 값은 Source packing 규칙을 따릅니다.

SDK는 captured `/images` route와 source를 고정합니다. top-level `base_path`, `resource_type`, `self`, `connection`, `_synchronized`, `microversion` 같은 Python constructor control은 local 오류로 거부합니다. inner hook의 `base_path`·`resource_type`는 ordinary metadata가 되고 source를 바꾸지 않습니다. JSON/UTF-8·descriptor projection·안전한 헤더 제약은 Go의 입력 경계이며 임의 Python class/constructor 객체 실행과 같지 않습니다. auth·media·framing·version·size는 SDK가 소유하는 헤더입니다.

## raw POST와 Record view

생성 입력의 raw current Body와 65필드 declared `Record.Resource`를 구분합니다. owner/owner_id와 wire 별칭, unknown properties, bool/list/dict descriptor 변환과 현재 Connection location은 SDK가 project합니다. view의 기본값·null·변환된 getter 값이 자동으로 raw POST에 추가되지는 않습니다. constructor에 supplied `id`도 생성 Body에는 남습니다.

정상 JSON object metadata 응답은 `self`를 제외한 raw 값을 constructor 상태에 sparse overlay하고 기준을 clean합니다. `{}`도 properties의 Source 기본 처리를 거칩니다. metadata의 import-method header는 constructor의 plain 목록을 교체하며 생략되면 빈 목록으로 reset합니다. invalid JSON syntax는 Source처럼 body overlay/clean을 하지 않고 pending constructor 상태·실제 receipt·헤더를 유지합니다. 유효한 non-object JSON이나 descriptor 변환 오류는 실패로 반환합니다.

PUT route는 metadata 변환 후 private canonical `id`를 사용합니다. 응답이 값을 덮어쓰지 않으면 constructor의 raw ID가 남을 수 있습니다. public Resource·Wire·Envelope·Location·self·file을 수정해도 route가 바뀌지 않습니다. Go는 nonblank Unicode 문자열 ID와 안전한 단일 경로 segment를 요구하고 한 번 escape합니다. Source의 임의 타입·slash ID urljoin과 이 경계는 다릅니다. ID 오류는 생성 Record·Metadata를 보존하고 binary 읽기 전에 멈춥니다.

## Data·크기·수명

metadata 변환이 끝난 후 Data를 Record의 plain private data로 할당합니다. 이는 Body·dirty 속성과 별개입니다. nil Data는 앞선 데이터를 재사용하지 않고 빈 PUT를 보냅니다. typed-nil reader는 local 입력 오류입니다. bytes·문자열은 caller가 `bytes.NewReader`·`strings.NewReader`로 전달할 수 있습니다. `Attributes["data"]`·`Attributes["filename"]`는 metadata이고 binary source나 파일 경로가 아닙니다.

기본 size 추론은 `io.Seeker`의 현재 cursor를 읽고 끝까지 seek한 전체 길이를 얻은 뒤 기존 cursor를 복원합니다. 전송은 원래 위치부터 하므로 전체 길이 hint와 실제 남은 bytes는 다를 수 있습니다. non-seekable reader와 `ESPIPE`는 size를 생략합니다. 다른 seek/복원 오류는 생성 결과를 보존하고 PUT 전에 반환합니다. SDK는 end-seek 실패에도 가능한 cursor 복원을 시도합니다. Python의 duck-typed file·복원 실패 동작과 Go의 `io.Seeker` 경계는 문서화된 차이입니다.

명시 size는 음수·0·양수 `int64`를 그대로 `X-OpenStack-Image-Size`에 보내며 추론하지 않습니다. `Without…Size` 뒤에도 `DisableSizeInference`가 true이면 추론을 중지합니다. Python `int` subclass·bool의 문자열 header 표현은 Go `int64` 옵션의 입력 범위 밖입니다. size header는 binary 단계만 소유하며 ordinary header로 위장할 수 없습니다.

caller reader는 전송 중 현재 위치에서 소비되며 SDK가 닫지 않습니다. binary transport에는 Read만 노출합니다. 원래 transport·timeout·live auth를 유지하고 SDK binary reauth·retry·backoff·redirect·GetBody replay를 끕니다. metadata JSON POST는 native retry/auth 정책을 유지합니다. 반환 Record의 private borrowed data 참조도 후속 owned 호출에 보존되므로 재사용한다면 caller가 파일 수명과 cursor를 관리합니다. 이 main은 `run`이 끝난 뒤 caller가 닫으므로 그 Record로 같은 파일을 재전송하지 않습니다.

## 접수·부분 실패·서버 권한

`ImageRecordUploadResult{Record, Metadata, Uploaded}`는 metadata 생성과 upload 접수를 구분합니다. `Metadata`는 실제 accepted POST의 raw Body/Header/StatusCode입니다. 이후 ID·size·읽기·transport·HTTP 오류에도 생성 Record와 Metadata를 보존합니다. accepted 응답 처리의 read/Close/context 오류에도 해당 실제 receipt를 보존합니다. `Uploaded`는 실제 accepted PUT 응답이 있을 때만 존재하는 opaque 접수입니다. malformed/non-UTF-8 bytes도 Record의 Body·dirty·status·location·ImportMethods·이전 receipt를 바꾸지 않습니다.

두 단계는 Go의 accepted 200..399와 native 400..599 오류 정책을 따릅니다. 고정 Python은 metadata 응답을 변환하면서 4xx/5xx를 raise하지만 Image.upload의 PUT에는 별도 translate/raise가 없어 raw HTTP 오류 응답을 그대로 받을 수 있습니다. Go의 명시적 native rejection 오류는 의도한 차이입니다. 오류는 `errors.Is/As`로 확인하며 native rejection 증거와 accepted 처리 오류의 `resource.ResponseError`를 구분합니다.

실패한 이미지를 자동 삭제하거나 metadata를 rollback하지 않습니다. checksum·Task/Swift·import·대기·상태 합성·보상 GET도 수행하지 않습니다. binary 접수 후 Record의 status는 생성 응답에서 본 `queued` 등 이전 값일 수 있습니다. 현대 공개 `create_image` 전체 분기와 기존 `Upload`·`UploadImage`의 기본값/strict201·204 profile은 별도 범위입니다. [기존 직접 업로드](upload-image.md)와 [생성·import](create-import.md)에 해당 API를 설명합니다.

ordinary 생성·upload는 핵심 user 경로입니다. 같은 metadata 입력의 public visibility·다른 프로젝트 owner는 기본 정책에서 admin 분기이며 community visibility는 project member/admin 범위입니다. client는 role을 검사하거나 visibility/owner를 강제로 바꾸지 않습니다. 실제 소유권·이미지 상태·quota·backend·policy override는 서버가 판단합니다. [고정 정책 근거](../docs/glance-policy-priorities.md)에 Source 범위를 설명합니다.

비교는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 공개 `upload_image` 전체 graph와 Glance `57f7dd9e76ef24e1e9013eceaa703bd442469a24`, Gophercloud v2.15.0을 기준으로 합니다. 동적 Python class 확장·mutable cache·전체 Adapter/session 계약은 공통 후속 범위입니다. [구현 계획](../docs/implementation-plan.md)과 [지원 판정대장](../docs/sdk-support-ledger.md)의 최신 실제 검증을 기준으로 지원 수치를 확인합니다.
