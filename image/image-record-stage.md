# Glance 이미지 레코드 staging: Python과 Go

`Connection.StageImageRecord`와 `image.Service.StageImageRecord`는 SDK가 반환한 queued `ImageRecord`에 binary 데이터를 한 번 전송하고, 성공한 staging 뒤 같은 private 이미지 ID로 메타데이터를 조회합니다. concrete request·옵션이 filename 열기, reader 선택, 크기 추론과 단계별 응답 보존을 담당합니다. 별도 builder나 resolver 구현은 필요하지 않습니다.

먼저 `GetImageRecord`를 명시적으로 호출하고 `ImageRecordStageRequest{Record: record}`를 전달합니다. 고정 Python의 `stage_image("id", ...)`는 새 Resource에 queued 상태가 없어 실패하며 자동 GET을 하지 않습니다. Go의 literal `ID` 입력도 같은 경로에서 IO·HTTP 전에 실패합니다. 이름 조회는 이 helper에 포함되지 않습니다.

| 단계 | Python `conn.image` | Go Service·Connection |
|---|---|---|
| 명시적 조회 | `record = conn.image.get_image(id)` | `service.GetImageRecord(ctx, image.ImageRecordRequest{ID: id})` |
| SDK가 여는 파일 | `stage_image(record, filename=path)` | `StageImageRecord(ctx, image.ImageRecordStageRequest{Record: record, Filename: path})` |
| caller reader | `stage_image(record, data=data)` | `ImageRecordStageRequest{Record: record, Data: reader}` |
| 명시적 크기 | `size=0` 또는 signed int | `image.WithImageRecordStageSize(0)` 또는 signed int64 |
| 필수 후속 조회 | helper 내부 `image.fetch` | 같은 captured source로 private ID GET |

기존 [`API.ImageData.StageImage/StageKnownImage`](v2/imagedata/README.md)은 native typed profile을 유지합니다. `StageImage`는 초기 GET을 수행하고, `StageKnownImage`는 native Image의 ID/status를 복사합니다. 두 API는 PUT204·최종 GET200과 명시적인 nonnegative size를 사용하며 private pending Body 전체·borrowed data 참조를 보존하지 않습니다. 새 owned API의 자동 total-size 추론·filename·header-only ACK·whole seeded fetch와 구분해서 선택합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
record = conn.image.stage_image(record, filename="image.qcow2")

# 별도 실행에서 reader를 빌리는 경우:
# with open("image.qcow2", "rb") as data:
#     record = conn.image.stage_image(record, data=data, size=0)
```

고정 Source의 filename 경로는 파일을 열어 `image.data`에 보관하며 helper에 explicit close가 없습니다. Go는 SDK가 연 파일을 필수 최종 GET 이후 닫고 반환 Record에 closed file을 남기지 않습니다. Python의 caller file도 수명을 직접 관리하는 것이 좋습니다.

## 독립 Go main

[설치 안내](../docs/install.md)를 따라 아래 코드를 `main.go`로 저장합니다. `go run . -cloud dev -image-id ID -filename image.qcow2`는 GET으로 SDK Record를 얻고 staging을 **한 번** 호출합니다. 기본은 SDK-owned filename입니다. `-borrow`는 예제가 직접 연 파일을 reader로 빌려주며, `-service`는 동일 작업을 Service에서 실행합니다. Connection과 Service를 중복 호출하지 않습니다.

`-size 0`·`-size -1`은 명시적인 signed size이며, 생략하면 seek 가능한 입력의 **전체** 길이를 추론합니다. `-infer-size=false`는 이 seek를 끕니다. 예제의 60초 context는 인증·명시적 GET·PUT·최종 GET 전체 예산입니다. 문서 검증은 consumer 컴파일이며 실제 cloud 인증·데이터 전송을 실행한 검증은 아닙니다.

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
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    filename := flag.String("filename", "", "required binary image filename")
    borrow := flag.Bool("borrow", false, "open a caller-owned file and lend its reader")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    infer := flag.Bool("infer-size", true, "infer total seekable length and restore cursor")
    size := flag.Int64("size", 0, "explicit signed X-OpenStack-Image-Size")
    flag.Parse()
    explicitSize := false
    flag.Visit(func(value *flag.Flag) { if value.Name == "size" { explicitSize = true } })
    if *imageID == "" || *filename == "" { log.Fatal("-image-id and -filename are required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *filename, *borrow, *viaService, *infer, *size, explicitSize); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "native HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "accepted HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID, filename string, borrow, viaService, infer bool, size int64, explicitSize bool) (err error) {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordStageRequest{Record: record, Filename: filename}
    mode := "filename"
    if borrow {
        data, openErr := os.Open(filename)
        if openErr != nil { return openErr }
        defer func() { err = errors.Join(err, data.Close()) }()
        input = image.ImageRecordStageRequest{Record: record, Data: data}
        mode = "borrowed-reader"
    }
    options := []image.ImageRecordStageOption{
        image.WithImageRecordStageOpts(image.ImageRecordStageOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-stage-example"},
        }),
        image.WithImageRecordStageHeader("X-SDK-Example-Mode", mode),
        image.WithImageRecordStageHeaders(map[string]string{"X-SDK-Example-Operation": "stage"}),
        image.WithImageRecordStageSizeInference(infer),
    }
    if explicitSize {
        options = append(options, image.WithImageRecordStageSize(size))
    } else {
        options = append(options, image.WithoutImageRecordStageSize())
    }
    var result *image.ImageRecordStageResult
    if viaService {
        result, err = service.StageImageRecord(ctx, input, options...)
    } else {
        result, err = conn.StageImageRecord(ctx, input, options...)
    }
    return printStage(result, err)
}

func printStage(result *image.ImageRecordStageResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        value["staged"] = result.Staged
        value["metadata"] = result.Metadata
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
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

출력은 두 칸 들여쓰기 JSON입니다. receipt의 `Body []byte`는 base64로 표현되며 binary ACK를 JSON 이미지로 해석하지 않습니다. 예제의 caller file close 오류는 호출 종료 시 반환 오류에 합칩니다.

## 요청과 concrete 옵션

`ImageRecordStageRequest{ID string; Record *ImageRecord; Data io.Reader; Filename string}`는 ID/Record 중 하나와 선택적인 데이터 소스를 받습니다. SDK-produced Record의 private canonical `id`와 raw current/original/dirty 상태를 사용합니다. public Resource·Wire·Envelope·Header·properties.id를 수정해도 대상을 바꾸지 않습니다. private ID는 안전한 escaped URI segment 하나로 전달하며 `Location`·self·file·schema를 routing fallback으로 쓰지 않습니다. [공통 Record identity 경계](image-records.md)를 따릅니다.

source와 입력 Record·옵션 slice는 callback 전에 capture합니다. private Body의 descriptor 변환과 현재 location projection, `ImportMethods=[]` reset을 먼저 수행하고 정확한 lowercase `queued`를 검사합니다. missing/null/numeric·`Queued`/`QUEUED` 상태와 descriptor 오류는 SDK filename open·reader seek/read·PUT 전에 실패합니다. SDK Record를 직접 만든 public view로 대체할 수 없습니다.

Filename과 nonnil Data는 동시에 전달할 수 없습니다. Data가 nonnil이면 그 reader를 사용하며 typed nil reader는 IO 전에 거부합니다. Data와 Filename을 모두 생략하면 Record의 private borrowed reader를 재사용합니다. 그 참조도 nil이면 빈 PUT을 보냅니다. 명시적인 빈 reader도 유효한 입력입니다. 이는 Python 임의 object의 `__bool__`와 다르게 Go interface의 nil 여부를 기준으로 하는 경계입니다.

`ImageRecordStageOpts`는 `Headers map[string]string`, `Size *int64`, `DisableSizeInference bool`을 제공합니다. 기본 추가 header 없음·Size nil·추론 활성입니다.

- `WithImageRecordStageOpts`: map·size pointer를 snapshot하고 전체 옵션을 교체합니다.
- `WithImageRecordStageSize`: signed int64를 명시합니다. 0과 음수도 그대로 size header로 보냅니다.
- `WithoutImageRecordStageSize`: 명시 Size만 nil로 되돌립니다. 추론 활성 여부는 바꾸지 않습니다.
- `WithImageRecordStageHeader`: ordinary header 하나를 추가하고 같은 HTTP 이름의 이전 값을 교체합니다.
- `WithImageRecordStageHeaders`: map을 snapshot해 병합합니다. supplied map 안의 충돌하는 case alias는 거부합니다.
- `WithImageRecordStageSizeInference(false)`: 자동 seek를 끕니다. true는 다시 활성화합니다.

옵션은 순서대로 한 번 적용합니다. retained callback config·원본 map·size pointer 변경이 준비된 요청에 영향을 주지 않습니다. ordinary headers는 PUT과 최종 GET에 사용하며 인증·framing·Content-Type/Accept·version·size header를 덮어쓰는 옵션은 거부합니다. nil/error callback과 header 오류도 IO 전에 실패합니다. 별도의 routing·session·binary injection builder는 없습니다.

## 크기와 데이터 수명

Size가 nil이고 추론이 활성일 때 `io.Seeker`의 현재 위치를 구하고 EOF로 이동해 **전체 길이**를 얻은 뒤 현재 위치로 복원합니다. 실제 PUT은 그 현재 cursor부터 읽습니다. 예를 들어 전체 10바이트 중 cursor가4이면 size header는10이고 전송 데이터는 남은6바이트입니다. 남은 길이를 보내려면 caller가 명시 Size를 선택합니다. `X-OpenStack-Image-Size`는 저장 공간 힌트이며 HTTP Content-Length를 설정하지 않습니다.

current/end/restore seek의 ESPIPE는 unknown size로 처리해 header를 생략합니다. 다른 seek 오류는 PUT 전에 반환합니다. 실패한 end seek에도 cursor 복원을 best effort로 시도하며 별도 restore 오류는 원인과 합칩니다. Source는 예외 시 이 추가 restore를 하지 않으므로 이는 Go cleanup 개선입니다. source/context 실패를 cleanup으로 없애거나 전송을 계속하지 않습니다. non-seekable reader는 추론 없이 전송하고, borrowed reader의 seek 자체를 피하려면 inference를 끕니다.

Python의 `isinstance(size, int)`는 음수·0·bool·int subclass·큰 정수도 받을 수 있습니다. Go signed int64는 음수·0을 보존하지만 Python bool·arbitrary precision 범위까지 받지는 않습니다. 서버의 size·quota·저장 공간 검증을 SDK가 대체하지 않습니다.

SDK filename은 queued 검사를 통과한 뒤 `os.Open`으로 엽니다. 디렉터리·open 오류는 PUT 전에 실패합니다. 파일은 필수 최종 GET 뒤 return 시 한 번 닫으며, 오류 경로도 닫습니다. close 실패는 이미 얻은 결과와 원인·source/context 오류에 합치고 추가 fetch나 replay를 하지 않습니다. 닫은 파일은 반환 Record의 fallback 참조에서 제거합니다.

borrowed reader는 SDK가 닫지 않습니다. 반환 Record에는 동일 borrowed data 참조가 유지되며 owned `UpdateImageRecord`, property helper, wait 결과를 거쳐도 보존됩니다. 데이터 자체를 복제하거나 cursor를 되감지 않으므로 다음 staging은 그때의 cursor를 사용합니다. 파일 수명·blocking Read 해제·동시 사용은 caller가 관리합니다. 재사용하는 Record는 caller의 원본과 독립된 body/receipt snapshot이지만 reader는 빌린 동일 객체입니다.

binary PUT에는 retry/reauth/backoff·HTTP redirect replay를 적용하지 않습니다. live token과 원래 transport/timeout을 유지하고 reader Read 전후와 response 처리 경계에서 source/context를 검사합니다. final JSON GET은 기존 안전한 auth/retry 정책을 유지합니다. SDK 한 번의 stage 시도와 caller transport 내부 재시도는 다른 범위입니다.

## 실제 receipt와 pending Body

Source `Image.stage`는 `_translate_response(has_body=False)`를 호출하므로 `400..599`는 오류이고 최종 GET을 실행하지 않습니다. Go도 이 native rejection 정책을 사용합니다. action helper가 raw 응답을 버리는 Python 기본 정책과 혼동하지 않습니다. Go는 actual `200..399`를 accepted receipt로 보존하며, redirect를 따라갔다는 뜻은 아닙니다. final metadata GET에도 owned Record의 accepted200..399 계약을 적용합니다.

stage ACK의 Body는 empty/invalid JSON/array/null/non-UTF-8도 opaque bytes입니다. Record에는 실제 ACK Header·StatusCode·Envelope를 보관하고 Wire는 nil입니다. `ImportMethods` header만 소비하며 current/original/dirty Body와 pending 속성을 clean하지 않습니다. SDK가 `uploading`·`active` 상태를 합성하지 않습니다.

성공한 header translation 뒤 private fixed ID로 필수 GET을 수행합니다. valid object 응답은 private seed 전체에 overlay하고 Body 기준을 clean하며 최신 Header·ImportMethods·location을 project합니다. 응답에 없는 기존 seed 속성은 남을 수 있습니다. malformed JSON은 body baseline·pending 변경을 보존하고 실제 새 receipt를 반환합니다. valid JSON의 잘못된 형태·descriptor/identity 불일치는 actual GET 증거와 오류로 반환합니다. 다른 응답 ID·Location이나 public view로 후속 경로를 바꾸지 않습니다.

| 종료 지점 | `ImageRecordStageResult` |
|---|---|
| prepare/open/seek 실패, native PUT rejection, accepted PUT 없는 transport 실패 | nil result + error |
| accepted PUT body Read/Close·source/context 또는 header translation 실패 | `Staged` 실제 receipt, `Record=nil`, `Metadata=nil` + error |
| stage header translation 성공 뒤 최종 GET의 native/transport/dispatch 실패 | `Staged`와 prepared Record 유지, `Metadata=nil` + error |
| accepted 최종 GET 처리·projection 실패 | `Staged`·actual `Metadata`와 가능한 partial Record + error |
| 최종 GET 성공 | translated Record·Staged·Metadata |
| SDK-owned file Close 실패 | 해당 시점 actual result 유지 + joined error |

`Staged`와 `Metadata`는 각각 `*ImageUploadResponse{Body, Header, StatusCode}`이며 Record receipt와도 독립적인 snapshot입니다. nonnil result나 staging ACK만으로 전체 workflow 성공을 판단하지 않고 err를 확인합니다. `errors.As`로 native `gophercloud.ErrUnexpectedResponseCode`와 accepted `resource.ResponseError`, `errors.Is`로 reader/seek/Close/context 원인을 확인할 수 있습니다.

stage는 metadata 생성·import 제출·Task 생성·active 대기·checksum·자동 DELETE를 실행하지 않습니다. 이후 작업은 별도 API를 선택합니다. 실패한 필수 fetch 때문에 이미 전송한 reader를 restage하거나 생성 이미지/임시 데이터를 SDK가 정리하지 않습니다.

## 고정 Source와 정책 범위

openstacksdk pin `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.stage_image538–581행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L538-L581), [Image.stage325–366행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L325-L366), [get_file_size462–491행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462-L491)을 비교했습니다. whole seed/header response 규칙은 [Resource._translate_response1338–1403행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1403), 필수 fetch는 [Resource.fetch1778–1837행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)입니다.

Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [stage controller330–360행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_data.py#L330-L360)은 repository 조회와 `modify_image`를 사용합니다. [modify 정책80–93행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)은 project scope `ADMIN_OR_PROJECT_MEMBER`이므로 기본 stage는 핵심 user 범위입니다. 실제 소유권·접근·상태·quota·backend·policy override는 서버가 판단합니다. [권한 분류 근거](../docs/glance-policy-priorities.md)를 참고하세요.

immutable Go Record·strict Unicode/private identity·io.Reader/int64 경계·owned file close·binary replay 방지·실패한 seek의 추가 restore는 명시적인 Go 선택입니다. arbitrary mutable Resource/subclass와 공통 dirty/Header lifecycle은 SDK-R1, Adapter cache/invalidation은 SDK-C1, 동적 session/microversion/transport kwargs는 SDK-S1에서 별도 추적합니다. 이 profile의 source 계약·consumer 빌드·fixture 테스트가 Python runtime 전체나 실제 OpenStack 배포의 동등성을 증명하지는 않습니다.
