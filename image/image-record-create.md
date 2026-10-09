# Glance 이미지 생성: Python과 Go

`Connection.CreateImageRecord`와 `image.Service.CreateImageRecord`는 현대 `conn.image.create_image`의 전체 흐름을 제공하는 진입점입니다. SDK가 cloud 기본값·checksum 계산·기존 이미지 검색과 재사용·메타데이터 변환·직접 업로드·import·설정된 Swift Task 흐름을 선택합니다. 호출자는 request와 `With…` 옵션을 전달하며 builder나 resolver interface를 구현하지 않습니다.

고정 비교 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L178-L436)입니다. 아래는 현재 구현의 사용법과 표현 경계입니다. 연산의 최종 지원 판정·실행 검증은 [지원 판정대장](../docs/sdk-support-ledger.md)과 [현재 진행 단계](../docs/implementation-plan.md#현재-집계와-진행-중인-작업)에서 관리합니다.

| Python `conn.image.create_image` | Go Service·Connection |
|---|---|
| `name`, `filename`, `data`, `**kwargs` | `ImageRecordCreateRequest{Name, Filename, Data, Attributes}` |
| `data=bytes`, `data=str`, `data=file` | `ImageRecordCreateBytes`, `ImageRecordCreateText`, `ImageRecordCreateReader` |
| `container`, `md5`, `sha256`, `disk_format`, `container_format`, `tags` | 대응하는 `WithImageRecordCreate…` 옵션 |
| `disable_vendor_agent`, `allow_duplicates`, `validate_checksum` | 대응하는 raw JSON 값 옵션; 기본값과 적용 순서는 SDK 소유 |
| `meta={...}` | `WithImageRecordCreateMeta(map[string]any)` |
| `use_import`, `import_method`, `uri`, `remote_*`, `stores`, `all_stores*` | `WithImageRecordCreateUseImport`와 `WithImageRecordCreateImportOptions(image.WithImageRecordImport…)` |
| `wait`, `timeout`, `size` | `WithImageRecordCreateWait`, `Timeout`, `Size`; timeout 단위는 초 |
| `conn.image.create_image(...)` | `service.CreateImageRecord(ctx, request, options...)` |
| `conn.create_image(...)`의 modern proxy 흐름 | `conn.CreateImageRecord(ctx, request, options...)`; 같은 request·옵션·결과 |
| mutable Image 또는 wait 없는 Task 반환 | `ImageRecordCreateResult`가 선택된 Record·Task·단계별 실제 응답을 분리 |

기존 [Upload](upload-image.md), [CreateAndImport](create-import.md), deprecated proxy에 대응하는 [UploadImageRecord](image-record-upload.md), 개별 [StageImageRecord](image-record-stage.md)·[ImportImageRecord](image-record-import.md)는 각각의 계약을 유지합니다. 현대 흐름이 필요한 경우 이 문서의 `CreateImageRecord`를 사용합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.create_image(
    name="ubuntu-example",
    filename="ubuntu.qcow2",
    visibility="private",
    allow_duplicates=True,
)

# 선택적인 checksum 계산·중복 재사용.
record = conn.image.create_image(
    name="binary-example",
    data=b"image bytes",
    validate_checksum=True,
)

# 동일 흐름에서 명시적으로 interoperable import를 선택.
record = conn.image.create_image(
    name="import-example",
    filename="ubuntu.qcow2",
    use_import=True,
    import_method="glance-direct",
)

# image_api_use_tasks가 활성화된 cloud에서는 Swift Task 경로를 사용.
# wait=False가 실제 함수 signature의 기본값이다.
```

위 호출은 독립적인 사용 예입니다. 파일과 bytes는 실제 이미지 데이터로 바꿉니다. Task API 접근 권한과 import 지원 방식은 cloud의 서버 정책으로 결정됩니다.

## 독립 Go main

[설치 안내](../docs/install.md)를 따라 다음 코드를 `main.go`에 저장합니다. 한 번 실행할 때 선택한 scenario 하나만 호출합니다. 기본 scenario는 `direct`이고 filename을 SDK가 열고 닫습니다. `-service`는 같은 호출을 Image Service에서 실행합니다. cloud를 생략하면 `OS_CLOUD` 또는 `OS_*` 설정을 사용합니다.

```sh
go run . -cloud dev -scenario direct -file ubuntu.qcow2 -name ubuntu-example
go run . -cloud dev -scenario bytes -file ubuntu.qcow2 -name binary-example -service
go run . -cloud dev -scenario import -file ubuntu.qcow2 -name import-example
go run . -cloud task-cloud -scenario task -file ubuntu.qcow2 -name task-example -wait=true
go run . -cloud dev -scenario metadata -name metadata-example
```

명령마다 실제 해당 API를 실행합니다. `bytes` 예제는 checksum 계산을 보여주기 위해 파일 전체를 읽으며, 큰 이미지에는 `direct`의 filename을 권합니다. `task` scenario는 cloud의 `image_api_use_tasks` 설정을 따릅니다. `-force-task-policy`를 명시하면 concrete policy로 Task를 활성화하고 object-store 사용을 지정합니다. 이 override는 cloud 이미지 policy 전체를 대체하므로 예제에서 지정하지 않은 이미지 기본값은 SDK 기본값이 됩니다.

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
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
)

type createImage func(context.Context, image.ImageRecordCreateRequest, ...image.ImageRecordCreateOption) (*image.ImageRecordCreateResult, error)

func main() {
    cloud := flag.String("cloud", "", "clouds.yaml entry; omission uses OS_CLOUD/OS_* settings")
    scenario := flag.String("scenario", "direct", "direct, bytes, import, task, or metadata")
    filename := flag.String("file", "", "local image file for direct/bytes/import/task")
    name := flag.String("name", "image-example", "image name")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    wait := flag.Bool("wait", true, "wait for Task completion in the task scenario")
    forceTask := flag.Bool("force-task-policy", false, "replace cloud image policy with explicit Task policy")
    flag.Parse()
    if *scenario != "metadata" && *filename == "" {
        log.Fatal("this scenario requires -file")
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *scenario, *filename, *name, *viaService, *wait, *forceTask); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, scenario, filename, name string, viaService, wait, forceTask bool) error {
    options := []openstack.ConnectionOption{}
    if cloud != "" {
        options = append(options, openstack.WithCloud(cloud))
    }
    if forceTask {
        options = append(options, openstack.WithImageCreatePolicy(
            image.WithImageCreatePolicyTasks(true),
            image.WithImageCreatePolicyObjectStoreEnabled(true),
        ))
    }
    conn, err := openstack.Connect(ctx, options...)
    if err != nil { return err }
    var create createImage = conn.CreateImageRecord
    if viaService {
        service, serviceErr := conn.Image(ctx)
        if serviceErr != nil { return serviceErr }
        create = service.CreateImageRecord
    }
    var result *image.ImageRecordCreateResult
    switch scenario {
    case "direct":
        result, err = direct(ctx, create, filename, name)
    case "bytes":
        result, err = validatedBytes(ctx, create, filename, name)
    case "import":
        result, err = imported(ctx, create, filename, name)
    case "task":
        result, err = task(ctx, create, filename, name, wait)
    case "metadata":
        result, err = metadata(ctx, create, name)
    default:
        return fmt.Errorf("unknown scenario %q", scenario)
    }
    return show(result, err)
}

func direct(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Filename: filename, Attributes: map[string]any{"visibility": "private"},
    }, image.WithImageRecordCreateAllowDuplicates(true))
}

func validatedBytes(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    data, err := os.ReadFile(filename)
    if err != nil { return nil, err }
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Data: image.ImageRecordCreateBytes(data),
    }, image.WithImageRecordCreateValidateChecksum(true))
}

func imported(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{Name: name, Filename: filename},
        image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateUseImport(true),
        image.WithImageRecordCreateImportOptions(image.WithImageRecordImportMethod("glance-direct")),
    )
}

func task(ctx context.Context, create createImage, filename, name string, wait bool) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{Name: name, Filename: filename},
        image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateContainer("images"),
        image.WithImageRecordCreateWait(wait),
        image.WithImageRecordCreateTimeout(300),
    )
}

func metadata(ctx context.Context, create createImage, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Attributes: map[string]any{"min_disk": "0", "hw_qemu_guest_agent": false},
    }, image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateTags([]string{"sdk-example"}),
        image.WithImageRecordCreateMeta(map[string]any{"vendor_raw": map[string]any{"enabled": false}}),
    )
}

func recordSummary(record *image.ImageRecord) any {
    if record == nil { return nil }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "status_code": record.StatusCode, "header": record.Header,
        "envelope_bytes": []byte(record.Envelope), "import_methods": record.ImportMethods,
    }
}

func taskSummary(value *image.ImageRecordCreateTask) any {
    if value == nil { return nil }
    return map[string]any{
        "resource": value.Resource, "status_code": value.StatusCode,
        "header": value.Header, "envelope_bytes": []byte(value.Envelope),
    }
}

func show(result *image.ImageRecordCreateResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if operationErr != nil { value["error"] = operationErr.Error() }
    if result != nil {
        value["outcome"], value["reused"], value["warnings"] = result.Outcome, result.Reused, result.Warnings
        value["record"], value["updated"] = recordSummary(result.Record), recordSummary(result.Updated)
        value["task"], value["task_diagnostic"] = taskSummary(result.Task), taskSummary(result.TaskDiagnostic)
        value["checksum"], value["imported"] = result.Checksum, result.Imported
        for phase, receipt := range map[string]*image.ImageUploadResponse{
            "created": result.Created, "uploaded": result.Uploaded, "staged": result.Staged,
            "checksum_fetched": result.ChecksumFetched, "task_image_fetched": result.TaskImageFetched,
            "task_diagnostic_response": result.TaskDiagnosticResponse,
        } {
            if receipt != nil { value[phase] = receipt }
        }
        if result.TaskWait != nil {
            value["task_wait"] = map[string]any{
                "task": taskSummary(result.TaskWait.Task), "original_task": taskSummary(result.TaskWait.OriginalTask),
                "fetches": result.TaskWait.Fetches, "recreations": result.TaskWait.Recreations,
                "fetch_responses": result.TaskWait.FetchResponses, "recreation_responses": result.TaskWait.RecreationResponses,
            }
        }
        if result.Swift != nil {
            value["swift"] = map[string]any{
                "container_discovery": result.Swift.ContainerDiscovery,
                "container_created": result.Swift.ContainerCreated,
                "container_fetched": result.Swift.ContainerFetched,
                "object": result.Swift.Object, "cleanup": result.Swift.Cleanup,
            }
        }
        if result.Cleanup != nil { value["image_cleanup_ack"] = result.Cleanup.Acknowledgement }
    }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return errors.Join(operationErr, err) }
    fmt.Println(string(body))
    return operationErr
}
```

응답 bytes는 base64로 출력해 빈·opaque 바이너리 ACK를 JSON으로 가정하지 않습니다. `Outcome`은 `reused`, `metadata-only`, `uploaded`, `imported`, `task`, `task-completed`를 구분합니다. 직접 업로드와 import는 성공해도 반환 Record의 `status`를 active로 합성하지 않습니다. Task를 기다리지 않은 `task` 결과는 ImageRecord나 완료된 import를 보장하지 않습니다.

## 입력·기본값·concrete 옵션

request에는 `Name`, `Filename` 문자열, tagged `Data`, `Attributes map[string]any`가 있습니다. Data의 zero value와 `ImageRecordCreateReader(nil)`은 Python None입니다. `ImageRecordCreateBytes(nil)`·빈 bytes·빈 text는 **present이지만 falsey**입니다. bytes/text는 복사되며 Reader는 현재 cursor에서 빌려 씁니다. Reader를 capture하면서 읽거나 seek하거나 닫지 않습니다. typed nil Reader는 오류입니다.

기본 container는 `images`, container format은 falsey일 때 `bare`, disk format은 falsey일 때 cloud의 `image_format`이며 설정이 생략되면 `qcow2`입니다. tags는 truthy일 때만 formal root에 들어갑니다. `disable_vendor_agent=true`, `allow_duplicates=false`, `validate_checksum=false`, `use_import=false`, **wait=false**, Task timeout은 3600초가 실제 기본값입니다. Source docstring의 wait 설명보다 함수 signature를 따릅니다. visibility에 SDK private 기본값을 추가하지 않습니다.

| `WithImageRecordCreate…` helper | 적용 |
|---|---|
| `Opts(ImageRecordCreateOpts)` | 전체 option policy를 복사·교체; 이후 helper로 overlay |
| `Container(string)` | Swift container; 빈 문자열도 presence 보존 |
| `MD5(any)`, `SHA256(any)` | raw checksum 값; 생략과 falsey 값 구분 |
| `DiskFormat(any)`, `ContainerFormat(any)`, `Tags(any)` | formal raw JSON 값 |
| `DisableVendorAgent(any)`, `AllowDuplicates(any)` | Python truthiness로 선택 |
| `Wait(any)`, `Timeout(any)` | Task wait 선택과 초 단위 예산 |
| `ValidateChecksum(any)`, `UseImport(any)` | 계산·검증과 non-Task import 선택 |
| `Size(any)` | 현재 구현은 signed arbitrary-precision JSON integer·bool; null/생략은 선택적 추론 |
| `Meta(map[string]any)` | 변환 후 raw overlay; mapping 전체 교체 |
| `Attribute(key, value)`, `Attributes(map[string]any)` | kwargs overlay; exact key를 병합 |
| `ImportOptions(...ImageRecordImportOption)` | import method/URI/remote/store/확장값; nested callback은 한 번 적용 |
| `Header(key, value)`, `Headers(map[string]string)` | Glance 단계별 일반 헤더 |
| `SwiftHeader(key, value)`, `SwiftHeaders(map[string]string)` | 선택된 Swift 단계의 일반 헤더 |

총 22개 helper는 같은 `ImageRecordCreateOpts`를 사용합니다. Raw 필드는 `json.RawMessage`이며 nil은 생략, JSON `null`은 명시된 값입니다. `Meta`·`Attributes`는 JSON-domain copy를 사용하고 raw number를 보존할 때 `json.RawMessage`를 넘깁니다. formal request 값과 option 값을 나눠 받아 Python의 keyword 바인딩 충돌을 Go에서 그대로 표현할 수 없는 곳에서는 SDK의 명시한 formal 값이 우선합니다.

`ImageCreatePolicy`의 concrete 필드는 `ImageFormat`, `UseTasks`, `DisableVendorAgent`, `ObjectStoreEnabled`, `CloudName`, late shape/type 확인을 위한 `RawVendorAgent`, `RawObjectStoreEnabled`입니다. `PrepareImageCreatePolicy`와 `WithImageCreatePolicyOpts/Format/Tasks/VendorAgent/ObjectStoreEnabled`로 만들며 Connection에서는 `openstack.WithImageCreatePolicy(...)`를 사용합니다. merged cloud의 `image_format`, `image_api_use_tasks`, `disable_vendor_agent`, `has_object_store`를 읽습니다. vendor shape 확인은 중복 재사용 뒤, object-store bool 확인과 서비스 구성은 실제 Task 분기에서 일어납니다. Service를 직접 만들 때 `image.Dependencies{CreatePolicy: policy, ObjectStorage: ...}`로 concrete Swift service 함수를 공급할 수 있으며 Connection이 이 연결을 소유합니다.

## 순서·재사용·메타데이터

1. truthy filename과 truthy Data가 동시에 있으면 검색·hash·HTTP 전에 실패합니다. 둘 다 falsey면 [Source filename helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L50-L63)를 실행합니다. 기존 regular file인 Name은 filename으로 선택하고 확장자를 뺀 basename을 name으로 사용합니다. `.disk`처럼 선행 점만 있는 이름은 보존합니다. Source의 두 번째 분기는 `exists(name+format)`과 `isfile(name)`을 조합하는 typo가 있어, 원래 name이 없는 상태에서 `name.qcow2`만 존재해도 자동 선택하지 않습니다. 실제 고정 동작을 유지합니다. format의 문자열 확인도 이 helper가 필요한 시점에만 적용합니다.
2. checksum validation을 켜고 truthy Data가 bytes가 아니면 거부합니다. Reader와 text는 supplied checksum이 있어도 이 조건에 걸립니다. MD5·SHA256 둘 다 falsey일 때만 filename 또는 truthy bytes에서 둘을 계산합니다. 한 hash가 supplied이면 다른 hash를 자동 계산하지 않습니다. 빈 bytes는 데이터 hash를 계산하지 않습니다.
3. `allow_duplicates=false`이면 literal GET → 일반 목록 전체 검색 → 정상 부재 뒤 hidden 목록 검색을 사용합니다. 이름 중복·페이지 오류는 생성으로 넘어가지 않습니다. supplied truthy hash 하나 이상이 모두 일치해야 재사용합니다. modern `owner_specified.openstack.*` key가 존재하면 null·빈 값이어도 legacy `owner_specified.shade.*`를 대체합니다. 재사용은 vendor/meta/import/store/Task 단계의 뒤쪽 오류를 검사하기 전에 반환합니다.
4. kwargs → enabled cloud vendor overwrite → legacy `properties`를 pop·flatten → synthesized `owner_specified.openstack.md5`, `sha256`, `object` → 변환 → raw Meta → formal name → Resource constructor hook 순으로 처리합니다. hash가 falsey이면 synthesized 값은 빈 문자열이고 object 값은 `container/name`입니다. 정수 key인 `min_disk`, `min_ram`, `size`, `virtual_size`는 Python int 정책, `is_protected`·`protected`·`tags`는 raw 유지, 나머지는 None 유지 또는 Python string 변환입니다. 정수 null 오류는 Meta가 해당 값을 덮어쓸 예정이어도 먼저 발생합니다. hook의 truthy `__conflicting_attrs`는 그 뒤 constructor에서 적용됩니다.
5. filename, truthy Data 또는 truthy import method가 없으면 메타데이터만 POST합니다. use_import, URI, stores, wait를 지정하는 것만으로 업로드를 선택하지 않습니다. metadata-only는 ID/status/formats를 요구하지 않으며 binary 전송·최종 GET·wait가 없습니다. `is_public`은 이 분기에서 일반 property 변환을 따릅니다. 업로드를 선택하면 presence가 있는 `is_public`을 pop하고 Python truthiness로 root visibility를 정하며 `Warnings`에 deprecated 사용을 남깁니다.

## non-Task 업로드·import·오류

truthy stores·all_stores·all_stores_must_succeed는 선택된 non-Task 업로드에서 use_import를 켭니다. truthy all_stores와 nonempty stores는 metadata POST 전에 충돌합니다. use_import를 선택하고 method가 falsey이면 `glance-direct`가 됩니다. filename은 metadata 변환·POST 전에 열고 모든 종료 경로에서 SDK가 한 번 닫습니다. borrowed Reader는 닫지 않습니다.

직접 업로드는 metadata POST → 생성 private ID로 `PUT /images/{id}/file`입니다. 새 Record의 queued gate, active wait, 최종 GET이 없습니다. `wait`·`timeout`은 이 경로에서 사용하지 않습니다. use_import는 생성 응답이 광고한 import methods의 exact membership을 **POST 뒤** 확인합니다. 지원하지 않는 method이면 이미 생성한 metadata와 오류를 반환하고 이미지를 삭제하지 않습니다.

`glance-direct`는 같은 private ID에 raw stage PUT → header translation → import POST를 합니다. 개별 StageImageRecord의 queued gate·후속 GET을 추가하지 않습니다. web/glance download는 해당 method에 맞는 원격 source를 제출합니다. nonnil empty stores는 명시한 빈 선택으로 유지되어 all_stores flags를 suppress합니다. import는 접수 응답이며 active 완료를 뜻하지 않습니다.

Size는 현재 integer/bool 표현을 `X-OpenStack-Image-Size`에 보냅니다. arbitrary precision과 음수·0을 유지하며 bool은 Python `True`·`False` 문자열입니다. null/생략이면 SDK filename 또는 borrowed file-like Reader의 **전체 길이**를 추론하고 cursor를 복원합니다. tagged bytes/text는 Go 내부 reader가 seek 가능해도 Source bytes/text에 없는 seek API를 가정해 길이를 추론하지 않습니다. 이 header는 가상 디스크 size와 구분합니다.

validate_checksum과 변환·Meta 뒤, constructor hook 이전 flat metadata의 truthy hash가 있으면 binary/import 뒤 metadata GET을 한 번 추가합니다. Meta로 덮어쓴 MD5·SHA256도 비교할 expected 값이 됩니다. truthy remote `checksum`을 MD5 또는 SHA256와 비교하며 falsey remote checksum은 오류 없이 받아들이지만 **내용 검증 성공을 주장하지 않습니다**. `Checksum.Compared`·`Matched`를 확인합니다. OS hash 선택이나 이미지 다운로드 검증은 이 helper의 계약에 포함되지 않습니다.

binary/stage/import/checksum try 내부 실패는 현재 private ID의 **전체 이미지 DELETE**를 시도합니다. 파일 열기·metadata POST·지원하지 않는 method의 앞쪽 실패는 이 cleanup을 실행하지 않습니다. clean delete404는 무시합니다. 원래 오류와 cleanup 실패는 `errors.Join`으로 보존하고 `Cleanup`에 실제 접수 결과를 남깁니다. context/source 실패 뒤 background cleanup으로 source를 우회하지 않습니다. metadata JSON·원격 제어 요청은 기존 provider 정책을 사용하고 borrowed binary stream은 replay 없이 한 번 전송합니다.

## Swift Task 구성과 대기

cloud Task 설정이 truthy면 explicit truthy use_import와 충돌합니다. Task route가 선택될 때만 Swift availability를 확인하고 service를 구성합니다. container HEAD → clean404이면 PUT → HEAD로 준비하며 container rollback은 없습니다. object에는 `application/octet-stream`, TTL 86400초, `x-object-meta-x-sdk-autocreated=true`를 SDK가 적용합니다.

ordinary tagged Data는 한 번의 Swift PUT로 전송하고 supplied file hashes를 적용하지 않습니다. borrowed Reader는 spool·seek·close·replay하지 않습니다. filename은 기존 [Swift CreateObject 엔진](../objectstorage/v1/objects/create.md)의 capability·stale·ordinary/SLO 경로를 사용합니다. 현대 흐름의 absent hash는 빈 문자열이므로 GenerateChecksums=false를 사용합니다. HEAD404에서는 계산한 hash metadata를 새로 만들지 않으며 existing HEAD 뒤 필요한 hash 계산은 비교용입니다. 이 composition은 유효한 UTF-8 hash 문자열을 받아 standalone CreateObject의 hex 길이 제한을 적용하지 않습니다.

preface에서 filename+empty bytes/text는 truthiness 충돌을 통과하지만 Swift는 Data presence를 검사합니다. 따라서 **container 준비 뒤** filename+present empty Data가 실패할 수 있습니다. 입력 형태의 구분과 실패 시점을 보존합니다.

Task POST에는 type import와 `input={import_from: container/name, image_properties: {name}}`를 제출합니다. 이 시점에 Glance image metadata POST는 하지 않습니다. wait=false는 ID/status/result의 완료 조건을 요구하지 않는 raw Task와 실제 응답을 반환하고 object를 남깁니다. 이후 image를 관측·정리하는 호출은 caller가 선택합니다.

wait=true는 성공한 Task POST 뒤 대기와 finally cleanup에 진입합니다. seeded success는 GET·timeout·ID 검사 전에 반환합니다. 기본 2초 간격과 3600초 예산을 사용하며 raw timeout null은 SDK 제한 없음, 0·음수는 즉시 timeout입니다. caller context가 전체 요청을 제한합니다. 정확한 396 message만 같은 raw type/input으로 재생성하고 **원래 시간 예산**에서 새 Task를 조회합니다. 재생성 응답이 success여도 다음 poll을 수행합니다. `TaskWait`가 latest Task, accepted fetch/recreation receipt와 count를 보존합니다. timeout 숫자·bool과 Go timer 범위의 표현 정책은 JSON/Go 경계에 속합니다.

success의 result.image_id로 image를 GET하고 fetched custom properties에 원래 **변환 전** properties를 병합한 뒤 raw UPDATE/PATCH를 합니다. formal disk/container format은 Task update에서 제거하고 tags·visibility는 유지합니다. Meta·Size는 Task 후처리에 적용하지 않습니다. ResourceFailure가 발생할 때만 outer original Task의 마지막 identity로 진단 GET을 추가하고 `TaskDiagnostic`·`TaskDiagnosticResponse`에 남깁니다. 일반 HTTP/type/timeout 실패에는 진단 GET을 합성하지 않습니다. Task Resource의 location은 operation에서 capture한 현재 Connection location입니다.

entered wait의 성공·실패·timeout·후처리 오류는 모두 finally에서 Swift object HEAD/DeleteObject를 시도합니다. Task 예산·HTTP 요청 타이머의 만료는 해당 요청의 오류로 보존하고, 부모 context가 살아 있으면 정리를 수행합니다. caller 취소나 실제 service/source 변경이 발생하면 후속 HTTP 요청을 중단합니다. SLO이면 manifest/segment cleanup의 기존 계약을 사용하며 실제 결과는 `Swift.Cleanup`입니다. object upload·Task POST가 실패하거나 wait=false일 때는 이 finally에 진입하지 않습니다. image/container rollback을 추가하지 않습니다.

| 결과 | 의미 |
|---|---|
| `Record`, `Updated` | 재사용/metadata/binary 결과와 Task 후처리의 owned ImageRecord; Resource·Wire·응답 증거 분리 |
| `Created`, `Uploaded`, `Staged`, `ChecksumFetched`, `TaskImageFetched` | 실제 metadata 또는 Task 생성, binary/stage, 선택 checksum GET, Task-success image GET 응답; final Update와 독립 |
| `Imported` | 실제 import 접수 결과 |
| `Task`, `TaskWait`, `TaskDiagnostic`, `TaskDiagnosticResponse` | 최초 Task, latest wait/396 재생성, 선택적인 원래 Task 진단 |
| `Swift.ContainerDiscovery/ContainerCreated/ContainerFetched` | 실제 container establishment 단계; 필요한 단계만 nonnil |
| `Swift.Object.Created/Uploaded` | filename/SLO·ordinary Data 업로드 증거; skipped·attempts 등 기존 결과 유지 |
| `Swift.Cleanup`, `Cleanup` | 별개의 Swift wait-finally cleanup와 non-Task 전체 이미지 삭제 증거 |
| `Checksum`, `Warnings`, `Reused`, `Outcome` | 실제 선택·비교 여부·deprecated property 관측; 성공 상태를 합성하지 않음 |

## 의도적인 Go 차이와 검증 경계

파일/SLO에서는 기존 Go engine의 immutable spool, unique segment namespace, 고정 worker 수와 정확한 이름의 보수적인 cleanup을 재사용합니다. Source의 mutable filename/segment cursor·넓은 prefix cleanup을 그대로 복제하지 않습니다. [Swift 차이·복구 경계](../objectstorage/v1/objects/create.md)를 함께 읽습니다. borrowed Data는 이 filename engine을 거치지 않습니다.

Source adapter가 import/direct ACK의 일부 HTTP 실패를 조용히 반환하는 곳에서도 Go는 native HTTP 오류를 반환합니다. Source cloud helper가 cleanup SDKException을 억제하는 경우 Go는 원래 실패와 cleanup 실패를 함께 반환합니다. 성공 뒤 cleanup만 실패하면 부분 결과와 오류가 함께 돌아오며 image 성공·Swift 정리를 하나의 성공 값으로 합치지 않습니다. 일반 headers가 auth·framing·TTL·autocreated 같은 SDK-owned control을 덮어쓸 수 없습니다.

Python의 모든 동적 객체/비문자 dictionary key/session·class mutation 대신 fixed JSON-domain 값·유효한 UTF-8·literal Go URL identity를 사용합니다. arbitrary Python integers 전반이나 임의 File-like callback 등은 각 bridge의 명시된 표현 범위를 따릅니다. 전체 SDK의 SDK-R1/C1/S1과 standalone Swift CreateObject·WaitForTask의 미해결 비교 행은 이 composition을 추가했다고 완료로 바꾸지 않습니다. generated transport 전체를 SDK parity로 세지 않습니다.

테스트는 기존 Gophercloud public testhelper의 assertions와 taskCore/shared native HTTP transport fixtures를 재사용합니다. 새로운 server/HTTP engine을 각 기능마다 작성하지 않으며, Source 순서·요청 payload·phase receipt·callback·reader/cleanup 오류의 관측을 검사합니다. 실제 실행 결과·whole-project gates·외부 소비자 example 빌드 증거는 구현 단계 문서에서 별도로 기록합니다. 이 가이드 자체는 실제 cloud 인증·Python 실행·전체 지원 완료의 증거가 아닙니다.
