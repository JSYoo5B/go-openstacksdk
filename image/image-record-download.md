# Glance 이미지 레코드 다운로드: Python과 Go

`Connection.DownloadImageRecord`와 `image.Service.DownloadImageRecord`는 이미지 Resource를 준비하고 metadata를 조회한 뒤 파일·borrowed writer·메모리·stream으로 데이터를 받습니다. SDK가 descriptor 변환, 저장소 preference, checksum 선택과 단계별 응답을 처리합니다. 기본 사용에는 builder나 hash interface 구현이 필요 없습니다.

Python의 `conn.image.download_image` 공개 함수 전체 흐름에 대응합니다. 모든 모드에서 metadata GET → binary GET 순서이며, hash 선택은 binary 응답을 받은 뒤 수행합니다. binary 응답으로 Record를 다시 변환하거나 마지막 GET을 수행하지 않습니다. lower image proxy의 ID·Resource 입력을 받으며 이름 검색은 별도 [레코드 검색](image-records.md) 작업입니다.

| 작업 | Python `conn.image.download_image` | Go Service·Connection |
|---|---|---|
| ID | `download_image(id, ...)` | `ImageRecordDownloadRequest{ID: id}` |
| 기존 Resource | `download_image(record, ...)`; dict/Munch도 `_get_resource`에서 처리 | SDK가 반환한 `Record` 또는 raw JSON `Resource` 중 하나 |
| 경로에 저장 | `output="ubuntu.qcow2"` | `ImageRecordDownloadRequest{Filename: "ubuntu.qcow2"}` |
| caller writer | `output=file` | `ImageRecordDownloadRequest{Output: writer}` |
| 기본 메모리 | output 생략, `stream=False`; `Response.content` | output 생략, 기본 Stream=false; `Downloaded.Body` |
| caller stream | output 생략, `stream=True`; caller가 소비·close | `WithImageRecordDownloadStream(true)`; `Downloaded.Stream`을 caller가 소비·Close |
| 복사 chunk | `chunk_size=1048576` | 기본 1MiB; `WithImageRecordDownloadChunkSize(n)` |
| 저장소 preference | `store_preferences=["fast", "archive"]` | `WithImageRecordDownloadStorePreferences("fast", "archive")` |
| checksum | 사용할 수 있는 hash를 소비 후 검증 | 기본 같은 선택 순서; explicit proof와 factory 옵션 제공 |
| 반환값 | 세 분기 모두 실제 `Response` | fetched `Record`, `Metadata`, binary `Downloaded`, 복사량·checksum proof |

Source docstring의 bytes/Image 반환 설명과 달리 고정 구현은 모든 분기에서 Response를 반환합니다. output이 있으면 stream=true여도 저장 분기가 우선합니다. Go도 Output 또는 Filename이 있으면 Stream보다 우선하며, `Output`과 `Filename`을 동시에 지정하면 첫 요청 전에 오류를 반환합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
response = conn.image.download_image(
    "literal-image-id",
    output="ubuntu.qcow2",
    store_preferences=["fast", "archive"],
)
print(response.status_code)
```

다음은 각각 별도 실행 예입니다. borrowed file은 caller가 열고 닫습니다. 기본 메모리 모드는 전체 이미지를 메모리에 보관합니다. stream 모드는 SDK가 checksum을 자동 검증하지 않으므로 소비와 close를 caller가 담당합니다.

```python
with open("ubuntu.qcow2", "wb") as output:
    response = conn.image.download_image("literal-image-id", output=output)

response = conn.image.download_image("literal-image-id")
data = response.content

response = conn.image.download_image("literal-image-id", stream=True)
try:
    with open("ubuntu.qcow2", "wb") as output:
        for chunk in response.iter_content(chunk_size=1048576):
            output.write(chunk)
finally:
    response.close()
```

상위 cloud `conn.download_image(name_or_id, output_path=..., output_file=...)`는 별도 이름 검색과 정확히 한 output 검사를 갖는 다른 공개 함수입니다. 이 lower proxy 대응 API와 같은 완료 항목으로 세지 않습니다.

## 독립 Go main

[설치 안내](../docs/install.md)를 따라 아래 코드를 `main.go`로 저장합니다. 기본 file 모드는 SDK가 경로를 열고 닫습니다. writer 모드는 caller가 파일을 먼저 열어 빌려주고 직접 닫습니다. buffer 모드는 SDK가 전체 bytes를 보관하며 예제는 크기만 출력합니다. stream 모드는 반환된 `io.ReadCloser`를 caller가 파일에 복사하고 닫습니다.

```sh
go run . -cloud dev -id literal-image-id -mode file -output ubuntu.qcow2
go run . -cloud dev -id literal-image-id -mode writer -output ubuntu.qcow2 -service
go run . -cloud dev -id literal-image-id -mode buffer -prefer fast -prefer archive
go run . -cloud dev -id literal-image-id -mode stream -output ubuntu.qcow2 -timeout 10m
```

각 명령은 별도 실행 예이며 한 실행에서 Download를 한 번 호출합니다. `-service`로 같은 작업을 Image Service에서 실행할 수 있습니다. context는 인증·두 GET·반환 stream 소비가 모두 끝날 때까지 유지합니다. 기본 2분 예산은 파일 크기와 환경에 맞춰 변경합니다. `-prefer`를 반복하면 순서와 중복을 보존하고, `-prefer ''`도 명시할 수 있습니다. file/writer/stream 모드는 선택한 출력 파일을 덮어쓸 수 있습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
)

type preferences []string

func (value *preferences) String() string { return fmt.Sprint([]string(*value)) }
func (value *preferences) Set(store string) error {
    *value = append(*value, store)
    return nil
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    id := flag.String("id", "", "required literal image ID")
    mode := flag.String("mode", "file", "file, writer, buffer or stream")
    output := flag.String("output", "", "output path for file, writer or stream")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    chunk := flag.Int("chunk-size", 1<<20, "SDK consumed-mode copy chunk in bytes, up to 64 MiB")
    verify := flag.Bool("verify-checksum", true, "verify available hashes in SDK consumed modes")
    timeout := flag.Duration("timeout", 2*time.Minute, "authentication, GETs and consumption deadline")
    var stores preferences
    flag.Var(&stores, "prefer", "store preference; repeat to preserve order and duplicates")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required by this example") }
    if *mode != "file" && *mode != "writer" && *mode != "buffer" && *mode != "stream" {
        log.Fatal("-mode must be file, writer, buffer or stream")
    }
    if *mode != "buffer" && *output == "" { log.Fatal("this mode requires -output") }
    if *timeout <= 0 { log.Fatal("-timeout must be positive") }
    options := []image.ImageRecordDownloadOption{
        image.WithImageRecordDownloadOpts(image.ImageRecordDownloadOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-download-example"},
        }),
        image.WithImageRecordDownloadStream(*mode == "stream"),
        image.WithImageRecordDownloadChunkSize(*chunk),
        image.WithImageRecordDownloadChecksumVerification(*verify),
        image.WithImageRecordDownloadStorePreferences(stores...),
    }
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *id, *mode, *output, *viaService, options); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, id, mode, output string, viaService bool, options []image.ImageRecordDownloadOption) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordDownloadRequest{ID: id}
    var writer *os.File
    if mode == "file" {
        input.Filename = output
    } else if mode == "writer" {
        // Caller creates/truncates this file before the SDK metadata GET.
        writer, err = os.Create(output)
        if err != nil { return err }
        input.Output = writer
    }
    var result *image.ImageRecordDownloadResult
    var operationErr error
    if viaService {
        service, serviceErr := conn.Image(ctx)
        if serviceErr != nil { operationErr = serviceErr } else {
            result, operationErr = service.DownloadImageRecord(ctx, input, options...)
        }
    } else {
        result, operationErr = conn.DownloadImageRecord(ctx, input, options...)
    }
    if writer != nil { operationErr = errors.Join(operationErr, writer.Close()) }
    var callerBytes int64
    if result != nil && result.Downloaded != nil && result.Downloaded.Stream != nil {
        // Consumption and Close happen while the original context is alive.
        // These caller bytes do not update result.BytesWritten or Checksum.
        var consumeErr error
        callerBytes, consumeErr = saveStream(result.Downloaded.Stream, output)
        operationErr = errors.Join(operationErr, consumeErr)
    }
    return printDownload(result, operationErr, callerBytes)
}

func saveStream(stream io.ReadCloser, filename string) (count int64, err error) {
    defer func() { err = errors.Join(err, stream.Close()) }()
    output, err := os.Create(filename)
    if err != nil { return 0, err }
    defer func() { err = errors.Join(err, output.Close()) }()
    return io.Copy(output, stream)
}

func printDownload(result *image.ImageRecordDownloadResult, operationErr error, callerBytes int64) error {
    value := map[string]any{
        "partial": result != nil && operationErr != nil,
        "caller_stream_bytes_written": callerBytes,
    }
    if result != nil {
        value["bytes_written"] = result.BytesWritten
        if result.Metadata != nil {
            value["metadata"] = map[string]any{
                "body": result.Metadata.Body, "header": result.Metadata.Header,
                "status_code": result.Metadata.StatusCode,
            }
        }
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource, "wire": result.Record.Wire,
                "status_code": result.Record.StatusCode,
                "import_methods": result.Record.ImportMethods,
            }
        }
        if result.Downloaded != nil {
            value["downloaded"] = map[string]any{
                "buffered_bytes": len(result.Downloaded.Body),
                "status_code": result.Downloaded.StatusCode,
                "header": result.Downloaded.Header,
                "compatibility_header": result.Downloaded.CompatibilityHeader,
                "content_md5": result.Downloaded.ContentMD5,
            }
        }
        if result.Checksum != nil {
            value["checksum"] = map[string]any{
                "algorithm": result.Checksum.Algorithm,
                "expected": result.Checksum.Expected, "actual": result.Checksum.Actual,
                "complete": result.Checksum.Complete, "verified": result.Checksum.Verified,
            }
        }
    }
    if operationErr != nil {
        value["error"] = operationErr.Error()
        value["checksum_mismatch"] = errors.Is(operationErr, image.ErrChecksumMismatch)
        value["hash_length_required"] = errors.Is(operationErr, image.ErrImageRecordDownloadHashLengthRequired)
    }
    body, err := json.MarshalIndent(value, "", "  ")
    if err == nil { _, err = fmt.Fprintln(os.Stdout, string(body)) }
    return errors.Join(operationErr, err)
}
```

buffer 결과의 `Downloaded.Body`는 caller가 사용하는 owned bytes입니다. 크기 제한이 없어 큰 이미지에는 file 또는 writer 모드를 사용합니다. `bytes.Buffer`를 직접 `Output`으로 빌려주어도 되며 이 경우 bytes는 caller buffer에 남고 `Downloaded.Body`는 nil입니다. stream 예제의 `caller_stream_bytes_written`는 caller의 `io.Copy` 결과로, SDK의 `BytesWritten`와 별개입니다. 출력 JSON은 binary 전체를 출력하지 않고 두 칸 들여쓰기로 metadata·headers·checksum proof·부분 오류를 보여 줍니다. metadata receipt의 raw `[]byte` Body는 JSON에서 base64로 표현됩니다.

## 입력 Resource와 private 상태

`ImageRecordDownloadRequest`에는 정확히 하나의 `ID`, `Record`, `Resource`를 지정합니다. literal ID에는 name fallback이 없습니다. owned Record는 `GetImageRecord`·`FindImageRecord`·생성 등 SDK가 반환한 값이어야 하며, public 필드만 조립한 Record에는 private raw 상태가 없어 local 오류를 반환합니다. 이미 조회한 Record를 전달해도 Download의 필수 metadata GET은 생략하지 않습니다.

raw Resource는 Python의 synchronized dict/Munch constructor 입력에 대응합니다. 예를 들어 다음과 같이 raw JSON을 보존해 준비할 수 있습니다.

```go
var seed resource.RawResource
if err := json.Unmarshal([]byte(`{
  "id": "literal-image-id",
  "os_hash_algo": "sha256",
  "os_hash_value": "expected-hex",
  "properties": {"purpose": "download-example"}
}`), &seed); err != nil {
    return err
}
result, err := service.DownloadImageRecord(ctx, image.ImageRecordDownloadRequest{
    Resource: &seed,
    Filename: "ubuntu.qcow2",
})
```

이 조각은 `resource`, `encoding/json`, `image`를 import한 오류 반환 함수 안에서 사용합니다. hash 예시는 입력 형태를 보여 주며 실제 metadata가 같은 값을 유지한다면 실제 digest와 일치해야 합니다. caller의 raw map·JSON bytes는 option callback 전에 복사합니다. raw constructor는 전체 Body를 보존하고, `Record.Resource`는 known descriptor·unknown properties·location을 처리한 65필드 declared view입니다. view 기본값이 raw pending Body에 자동으로 기록되는 것은 아닙니다. `self`, `connection`, `_synchronized`, `microversion` 같은 constructor 인자 충돌은 Go의 local 입력 경계에서 거부합니다.

metadata 응답은 private raw Body에 sparse overlay합니다. `{}` 같은 sparse object도 Source properties 처리와 clean baseline을 거치므로, 응답이 `id`를 바꾸면 이후 binary 경로는 바뀐 private ID를 사용합니다. 입력 ID와의 equality check는 없습니다. JSON syntax가 잘못된 응답은 body overlay·clean을 하지 않고 private/pending 상태와 실제 metadata receipt·header를 유지합니다. 유효한 non-object JSON이나 descriptor 변환 실패는 후속 binary GET 전에 오류를 반환합니다. import-method header 처리도 적용하며 header가 없으면 plain 목록을 빈 값으로 reset합니다.

초기 ID와 fetched ID는 nonblank valid Unicode 문자열이어야 하며 control·`.`·`..`는 local 오류입니다. ID는 captured `/images/{id}/file` 경로의 한 segment로 한 번 escape합니다. public Resource·Wire·Envelope·Location·`self`·`file`을 고쳐도 private identity나 source가 바뀌지 않습니다. 이전 Record의 borrowed data도 private clone에 남으며 Download가 그 값을 새 binary bytes로 대체하지 않습니다. caller가 이전 data의 수명과 cursor를 관리합니다.

## concrete 옵션

| helper | 의미 |
|---|---|
| `WithImageRecordDownloadOpts(ImageRecordDownloadOpts)` | 전체 option policy 교체 |
| `WithImageRecordDownloadStream(bool)` | output이 없을 때 unread stream 반환 |
| `WithImageRecordDownloadChunkSize(int)` | SDK consumed-mode 복사 chunk; 1byte..64MiB |
| `WithImageRecordDownloadStorePreferences(...string)` | ordered preference 목록 교체 |
| `WithImageRecordDownloadChecksumVerification(bool)` | false이면 checksum 선택·factory·검증 생략 |
| `WithImageRecordDownloadHashFactory(ImageRecordDownloadHashFactory)` | primary algorithm constructor를 함수로 확장 |
| `WithImageRecordDownloadHeader(key, value)` | ordinary header 한 개 병합 |
| `WithImageRecordDownloadHeaders(map[string]string)` | ordinary header map 병합 |

full opts는 `Stream bool`, `ChunkSize *int`, `StorePreferences []string`, `VerifyChecksum *bool`, `HashFactory`, `Headers`를 갖습니다. nil chunk는 1MiB, nil verification은 true입니다. pointer·slice·header map을 snapshot으로 복사하며 full opts는 이전 policy를 교체합니다. leaf helper는 뒤에 지정한 값이 우선하고 reusable option을 여러 호출에 적용할 수 있습니다. factory 함수 참조는 snapshot에 보존하며 closure 안의 상태는 caller가 관리합니다.

preferences는 empty·comma·control·중복도 valid UTF-8 문자열이면 보존합니다. 목록을 comma로 join하고 `prefer` 한 값으로 percent encode합니다. nil/빈 목록은 query를 생략합니다. 이는 서버가 backend를 선택한 순서나 실제 가용성을 보장하지 않습니다. 고정 Glance는 comma를 분리하고 whitespace/empty 항목을 정리하며, cache 경로는 preference 처리를 우회할 수 있습니다. [정책·store 근거](../docs/glance-policy-priorities.md)를 참고합니다.

nil option·typed-nil writer·잘못된 JSON/descriptor/ID/header·허용 범위 밖 chunk 등은 가능한 첫 요청 전에 실패합니다. `Filename`은 독립 출력 경로이며 metadata의 `filename` property와 관계없습니다. auth·media·framing·version·size header는 SDK가 소유합니다. ordinary header에는 captured native 요청 정책을 적용하며 이 API는 range resume나 부분 이미지 checksum policy를 추가하지 않습니다.

## checksum 선택·registry·확장

binary GET을 받아들인 뒤 새로 fetched Resource의 raw `hash_algo`, `hash_value`, `checksum`을 읽습니다. Source getter의 wire key `os_hash_algo`·`os_hash_value`는 declared view의 `hash_algo`·`hash_value`로 project합니다. 세 값은 문자열로 미리 좁히지 않고 Python truthiness를 적용합니다.

1. truthy algorithm·hash-value 쌍이 있으면 algorithm만 Python `str`에 대응하는 문자열로 변환해 primary factory를 한 번 호출합니다.
2. primary pair가 없거나 factory가 `ErrImageRecordDownloadHashUnsupported`를 반환하면 truthy metadata checksum을 선택합니다. 없으면 실제 binary `Content-MD5`를 사용합니다. 이 fallback은 SDK의 MD5 constructor를 사용합니다.
3. 사용할 hash가 없으면 `Checksum`은 nil이며 bytes를 검증 없이 전달합니다. Source는 warning을 기록하지만 Go는 선택 결과로 availability를 표현합니다.

기본 registry는 아래 이름을 대소문자 구분 없이 처리합니다. 앞뒤 whitespace는 trim하지 않습니다. 배포 환경 OpenSSL의 임의 algorithm 전체를 보장하지 않습니다.

| family | 이름·alias |
|---|---|
| MD5 | `md5` |
| SHA1·SHA2 | `sha1`/`sha-1`, `sha224`/`sha-224`, `sha256`/`sha-256`, `sha384`/`sha-384`, `sha512`/`sha-512` |
| SHA512 truncation | `sha512_224`/`sha512-224`, `sha512_256`/`sha512-256` |
| SHA3 | `sha3_224`/`sha3-224`, `sha3_256`/`sha3-256`, `sha3_384`/`sha3-384`, `sha3_512`/`sha3-512` |
| BLAKE2 | `blake2b`/`blake2b512`/`blake2b-512`, `blake2s`/`blake2s256`/`blake2s-256` |
| SHAKE | `shake_128`/`shake128`/`shake-128`, `shake_256`/`shake256`/`shake-256` |

BLAKE2는 `golang.org/x/crypto v0.55.0` 구현을 사용하며 indirect `golang.org/x/sys v0.47.0`도 포함합니다. 배포 라이선스·고지는 기존 14개에 두 BSD-3-Clause 원문을 더한 16개 파일이며, [라이선스 적용 범위](../docs/licensing.md)에 출처를 기록합니다. 설치 문서의 이전 revision 증거는 그 당시 14개 파일에 대한 기록입니다. 추가 algorithm은 `func(name string) (hash.Hash, error)` 형태의 `ImageRecordDownloadHashFactory`로 제공합니다. custom factory는 default primary constructor를 교체하며, truthy primary pair가 있을 때만 한 번 실행합니다. custom factory가 fallback을 원하면 unsupported sentinel을 반환합니다. absent/incomplete primary나 MD5 fallback에 custom factory를 재호출하지 않습니다. 다른 factory 오류나 nil/typed-nil hash는 accepted binary receipt를 보존하고 소비 전에 실패합니다. 임의 factory의 algorithm 구현·성능은 caller 책임입니다.

소비를 완료하면 actual lowercase hex와 raw expected를 그대로 비교합니다. hex 대소문자·whitespace를 normalize하거나 Content-MD5를 base64 decode하지 않습니다. truthy 숫자·bool·object·배열 expected는 전송을 끝낸 뒤 문자열 digest와 mismatch입니다. `errors.Is(err, image.ErrChecksumMismatch)`로 공통 오류를 확인하고 `errors.As`의 `*image.ImageRecordDownloadChecksumMismatchError`에서 raw Expected·Algorithm·Actual을 읽을 수 있습니다.

SHAKE는 constructor 선택에는 성공하지만 Source의 인자 없는 `hexdigest()`는 output length가 필요합니다. 같은 흐름으로 Go의 consumed 모드는 전체 복사 후 `ErrImageRecordDownloadHashLengthRequired`를 반환합니다. `Complete=true`이며 Actual은 비어 있고 Verified는 false입니다. stream-only는 digest를 만들지 않아 이 길이 오류가 발생하지 않습니다. 원하는 길이의 custom hash를 제공하는 것은 추가 factory 정책입니다.

`WithImageRecordDownloadChecksumVerification(false)`는 Go의 추가 옵션으로 primary/fallback 선택과 factory 호출을 모두 생략합니다. 필수 metadata GET이나 descriptor 변환을 생략하지 않습니다. available checksum이 존재해도 disabled 또는 stream-only 모드의 성공은 검증 완료를 뜻하지 않습니다.

## output·stream 소유권과 결과

`ImageRecordDownloadResult`의 `Record`는 fetched metadata 상태입니다. `Metadata`는 실제 metadata 응답의 raw Body/Header/StatusCode, `Downloaded`는 독립 binary 응답의 실제 Header/StatusCode와 mode별 Body·Stream입니다. binary 응답은 Record의 Body·dirty·location·status·ImportMethods를 바꾸지 않습니다.

file 모드는 metadata GET·binary GET·hash 선택이 성공한 뒤 `os.Create(Filename)`로 열어 truncate하고 SDK가 닫습니다. 그보다 앞선 실패에서는 이 파일을 열지 않습니다. 후행 read/write/checksum/Close 오류가 생겨도 이미 쓴 partial 파일을 지우거나 원본을 복구하지 않습니다. borrowed writer는 현재 위치에 쓰며 SDK가 Seek·Truncate·Flush·Close하지 않습니다. 성공한 write byte만 세고 short write는 오류로 보존합니다. writer/file 모드에서는 `Downloaded.Body`가 nil입니다.

buffer 모드는 output 없는 기본 분기입니다. 전체 Body를 보관하고 실패하면 이미 복사된 owned partial bytes를 반환합니다. 기본 memory cap은 없으며 chunk bound는 전체 이미지 크기 제한이 아닙니다. consumed 모드의 accepted HTTP body는 SDK가 한 번 닫습니다. EOF와 successful writes를 모두 확인했을 때만 checksum Complete/Actual을 기록하고 digest가 맞아야 Verified를 true로 설정합니다. Close-only 실패는 이미 완료한 proof를 취소하지 않습니다.

stream-only는 unread `Downloaded.Stream`을 반환하므로 caller가 원래 context가 살아 있는 동안 소비하고 반드시 Close합니다. late Read/Close 오류도 operation/context·sticky source 원인을 보존하며 Close는 idempotent합니다. 취소나 source guard 실패 뒤에도 Close는 underlying HTTP body를 정리합니다. stream의 EOF나 caller의 io.Copy는 반환 `BytesWritten=0`, `Checksum.Complete/Verified=false`를 자동 변경하지 않습니다.

stream-only에서 선택한 Algorithm이 정확히 `"md5"`일 때 raw expected는 `Downloaded.ContentMD5`에 보존합니다. Python `str` projection을 적용한 `CompatibilityHeader`는 Source의 response content-md5 교체를 표현하는 독립 header view입니다. 실제 `Downloaded.Header`는 수정하지 않습니다. primary 이름이 `"MD5"`처럼 대문자이면 default registry가 digest를 만들더라도 Source처럼 compatibility header를 교체하지 않습니다. 그 밖의 algorithm이나 checksum이 없는 경우에도 compatibility view는 실제 header의 clone이며 이 값은 자동 검증 증거가 아닙니다.

## HTTP·부분 오류·지원 경계

Go의 metadata와 binary 단계는 실제 200..399를 받아들입니다. binary 206이나 Content-Range가 있어도 local status gate를 추가하지 않으며, 204도 선택한 checksum을 지우지 않습니다. 선택한 expected에 대해 실제 수신 bytes를 검증하므로 부분 응답은 whole-image digest와 mismatch가 날 수 있습니다. 400..599는 native rejection 오류로 보존합니다.

고정 Python은 metadata fetch를 translate하면서 HTTP 오류를 raise합니다. binary DownloadMixin은 응답을 translate/raise하지 않고 Proxy.request의 기본 raise_exc=False를 사용하므로 HTTP 오류 body를 그대로 소비하거나 반환할 수 있습니다. Go의 명시적 native binary rejection은 문서화된 차이입니다. JSON 파싱 관용 범위와 string ID·header·io 타입·chunk limit·owned cleanup도 Go의 입력/수명 경계입니다.

metadata 접수 후 오류는 실제 Metadata와 만들어진 Record를 보존합니다. binary 접수 후 factory·파일 open·read/write·checksum·Close 오류는 binary receipt와 성공한 partial bytes를 보존합니다. 실제 rejection을 accepted binary receipt로 세지 않습니다. metadata의 accepted 응답 처리 오류인 `resource.ResponseError`, native rejection, checksum·io·context 원인은 `errors.Is/As`로 확인합니다. captured route/source·ordinary headers와 live auth는 native pre-body retry/reauth 정책을 유지하고, body 소비가 시작된 뒤 read/write/checksum 실패로 재다운로드하지 않습니다. redirect는 captured target을 벗어나지 않습니다.

ordinary metadata 조회·binary 다운로드는 핵심 user 경로이며 deactivated 이미지 다운로드는 기본 Glance에서 admin 분기입니다. client가 role·visibility·상태 gate를 만들지 않습니다. 실제 소유권·visibility·sharing·backend·cache·store preference·policy override는 서버가 판단합니다. [고정 정책 근거](../docs/glance-policy-priorities.md)에 관련 분기를 설명합니다.

기존 [`DownloadTo`](download.md)는 required writer·strict initial ID equality·선제 checksum 제한·200/204 profile을 갖는 별도 API입니다. raw native `service.API.ImageData.Download`도 독립적으로 사용할 수 있으며 caller가 Body를 닫습니다. 새 owned helper의 네 모드와 비교할 때 각 API의 입력/수명/HTTP 계약을 확인합니다.

비교는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 공개 `download_image` 전체 graph, Glance `57f7dd9e76ef24e1e9013eceaa703bd442469a24`, Gophercloud v2.15.0을 기준으로 합니다. 동적 Python class 확장(SDK-R1)·mutable cache(SDK-C1)·전체 Adapter/session 및 requests/hashlib runtime(SDK-S1)은 공통 후속 범위입니다. 별도 cloud download wrapper와 현대 공개 `create_image` 전체 분기도 이 helper 완료에 포함하지 않습니다. 실제 cloud/Python 호출은 이 문서 작성에서 실행하지 않았으며 지원 수치는 [구현 계획](../docs/implementation-plan.md)·[지원 판정대장](../docs/sdk-support-ledger.md)의 최신 실제 검증으로 확인합니다.
