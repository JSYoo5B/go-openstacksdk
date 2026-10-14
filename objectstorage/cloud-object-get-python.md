# Swift cloud 객체 조회·파일 출력: openstacksdk와 Go

`Connection.GetObjectText(ctx, container, object, options...)`는 Swift cloud `get_object`의 닫힌 문자열 이름 동선을 즉시 실행합니다. 기본값은 소문자 header map과 decoded text를 반환하고, `WithObjectGetOutfile`은 SDK가 열고 닫는 파일에 binary bytes를 씁니다. `WithObjectGetWriter`는 caller가 소유한 `io.Writer`에 씁니다. Source에서 선언만 하고 실제로 사용하지 않는 `query_string`과 `stream`은 Go에서도 무시합니다.

이 동선은 유의미한 구현·테스트를 제공하지만 named `get_object` 전체 판정은 **unresolved**입니다. 기본 `response.text`의 Requests 통계적 `apparent_encoding` 전체와 CPython의 전체 text-codec/alias registry는 재현하지 않았습니다. Source가 성공하는 기본 문자셋 추정에 Go `ErrUnsupported`가 나올 수 있으며, 이를 UTF8 구현이나 raw GET으로 완료했다고 세지 않습니다. 일반 `Proxy.get_object`의 mutable Object/Container·dirty attrs/session 경계와 기존 native `GetObject`·`DownloadObject` 판정도 그대로 둡니다.

Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [cloud get_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L512), [Proxy.get_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L288)입니다. Gophercloud `v2.15.0`, Go `1.25`, `golang.org/x/text v0.41.0`을 사용합니다. 비교 runtime은 CPython 3.14.8, Requests 2.34.2, urllib3 2.8.0, charset_normalizer 3.5.2입니다. 원 Source cloud/Proxy/Object/Requests 메서드를 실행한 65개 관찰에서 HTTP adapter는 메모리 `http.client.HTTPResponse`, path open은 recording writer로만 바꿨습니다. 실제 cloud 인증·파일시스템·ACL·릴리즈 판정은 이 Source 관찰에 포함하지 않습니다.

아래 Python은 독립 실행 예제입니다. 옵션을 생략하면 `(headers, body)`의 `body`는 `response.text`이고, clean 404이면 전체 반환값이 `None`입니다. 파일 또는 falsey non-None 출력 대상을 넘기면 body는 `None`입니다.

```python
import argparse
import json
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--stream", action="store_true")
parser.add_argument("--chunk-size", type=int, default=1024)
output = parser.add_mutually_exclusive_group()
output.add_argument("--outfile")
output.add_argument("--discard-content", action="store_true")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    result = conn.get_object(
        args.container, args.object,
        query_string=args.query, stream=args.stream,
        resp_chunk_size=args.chunk_size,
        outfile="" if args.discard_content else args.outfile,
    )
    if result is None:
        print("missing", file=sys.stderr)
    else:
        headers, body = result
        print(json.dumps(headers, indent=2), file=sys.stderr)
        if body is not None:
            sys.stdout.write(body)
```

아래 Go는 독립 `main.go`입니다. [설치 안내](../docs/install.md)에 따라 module/clouds.yaml을 설정하고 `go run . --cloud devstack --container assets --object part.bin --outfile part.bin`처럼 실행할 수 있습니다. binary 출력에는 `--outfile`을 사용합니다. 기본 text 출력에서는 헤더·부재·쓰기 수를 stderr에, 지원되는 decoded text를 stdout에 씁니다. GetObjectText는 반환 전에 physical HTTP body와 자신이 연 파일을 정리합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func run(ctx context.Context, cloud, container, object, query, outfile string,
    stream, discard bool, chunkSize int) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    options := []objectstorage.ObjectGetOption{
        objectstorage.WithObjectGetQueryString(query),
        objectstorage.WithObjectGetStream(stream),
        objectstorage.WithObjectGetChunkSize(chunkSize),
    }
    if outfile != "" { options = append(options, objectstorage.WithObjectGetOutfile(outfile)) }
    if discard {
        if outfile != "" { return fmt.Errorf("choose outfile or discard-content") }
        options = append(options, objectstorage.WithoutObjectGetContent())
    }
    result, getErr := conn.GetObjectText(ctx, container, object, options...)
    if result != nil {
        summary := struct {
            Headers map[string]string `json:"headers"`
            Missing bool `json:"missing"`
            BytesWritten int64 `json:"bytes_written"`
        }{result.Headers, result.Missing, result.BytesWritten}
        encoded, err := json.MarshalIndent(summary, "", "  ")
        if err != nil { return err }
        fmt.Fprintln(os.Stderr, string(encoded))
    }
    if getErr != nil { return getErr }
    if result != nil && result.Body != nil {
        written, err := io.WriteString(os.Stdout, *result.Body)
        if err != nil { return err }
        if written != len(*result.Body) { return io.ErrShortWrite }
    }
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    object := flag.String("object", "logs/part.bin", "literal object name")
    query := flag.String("query", "", "ignored by pinned cloud get_object")
    stream := flag.Bool("stream", false, "ignored by pinned cloud get_object")
    outfile := flag.String("outfile", "", "SDK-owned binary output path")
    discard := flag.Bool("discard-content", false, "falsey non-None output: headers only")
    chunkSize := flag.Int("chunk-size", 1024, "output-only decoded chunk size; 1..16777216")
    timeout := flag.Duration("timeout", 5*time.Minute, "whole request deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *container, *object, *query, *outfile,
        *stream, *discard, *chunkSize); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

서비스를 이미 골랐다면 `swift, err := conn.ObjectStorageV1(ctx)` 다음 `swift.Cloud.GetObjectText(...)`를 사용합니다. 가장 낮은 닫힌 동선은 `swift.Objects.GetCloudObjectText(...)`와 `objects.WithCloudGet...` 옵션입니다. caller가 builder interface나 body wrapper를 구현할 필요가 없습니다. borrowed `*bytes.Buffer`, `*bufio.Writer` 또는 `*os.File`은 표준 `io.Writer`로 전달하며 자신이 연 파일의 Close는 caller가 수행합니다.

| Python | Go | 동작 |
| --- | --- | --- |
| `get_object("assets", "file")` | `GetObjectText(ctx, "assets", "file")` | eager GET, optional text, lowercase headers |
| `query_string="x=1"`, `stream=True` | `WithObjectGetQueryString("x=1")`, `WithObjectGetStream(true)` | 둘 다 실제 동작에서 무시 |
| `resp_chunk_size=32768`, `outfile="part.bin"` | `WithObjectGetChunkSize(32768)`, `WithObjectGetOutfile("part.bin")` | binary 출력만 chunk 크기를 사용 |
| `outfile=file_object` | `WithObjectGetWriter(writer)` | borrowed writer, 성공 후 지원하는 Flush 호출, Close 안 함 |
| `outfile=""` | `WithObjectGetOutfile("")` 또는 `WithoutObjectGetContent()` | falsey non-None: headers만, payload·text read·write 없음 |
| `outfile=None` | 옵션 생략 또는 `WithoutObjectGetOutfile()` | 기본 text 동선 |
| 전체 옵션 교체 | `WithObjectGetOpts(ObjectGetOpts{...})` | ChunkSize·Outfile pointer 값은 SDK가 복사 |
| clean HTTP 404 → `None` | `result.Missing == true`, `err == nil` | actual 404는 별도 Receipt에 보존 |

`ObjectGetResult.Headers`는 Source `_last_headers`의 소문자 map이며 HTTP의 반복 header values는 `", "`로 합칩니다. `Body`는 `*string`이라 성공한 빈 text `""`와 출력 동선의 nil을 구분합니다. `Receipt`는 실제 status/header, `BytesWritten`은 destination이 받아들인 bytes만 셉니다. clean 404는 Missing과 실제 receipt를 남기고 Headers/Body는 nil입니다. 반환값은 partial error evidence를 담을 수 있으므로 err를 먼저 확인합니다. Header/Receipt/error proof는 서로 공유하는 mutable map/bytes가 아닙니다.

기본 text는 헤더의 명시적 charset을 우선하며 Requests의 느슨한 `;` 분리와 마지막 charset 값 우선 정책을 사용합니다. media substring 비교는 Source처럼 case-sensitive입니다. `text`가 포함되면 정확한 Latin-1, `application/json`이면 UTF8 기본값입니다. Latin-1의 `0x80`은 `U+0080`이며 Windows-1252의 euro로 바꾸지 않습니다. 알려지지 않은 빈/잘못된 charset의 UTF8 replacement fallback, UTF8/UTF8-SIG·ASCII·Latin-1·Windows-1252·UTF16/UTF32와 일부 IANA 이름/alias를 지원합니다. invalid UTF8·명시적 UTF16/UTF32의 짧은 마지막 입력은 replacement 문자로 처리합니다. UTF16의 마지막 high-surrogate와 남은 한 byte는 Source처럼 replacement 하나로 묶습니다. `shift_jis`처럼 underscore가 실제 IANA 이름에 필요한 경우에는 원래 이름으로 lookup합니다. CPython 전체 codec/alias 의미와 Go IANA decoding이 모두 같다는 뜻은 아닙니다. `unicode_escape`, `cp1006`, `palmos` 같은 알려진 미지원 Python codec은 `ErrUnsupported`와 actual received bytes를 남깁니다.

charset이 없을 때 Go가 처리하는 valid UTF8/BOM의 제한 정책도 Requests의 전체 `apparent_encoding` 구현을 뜻하지 않습니다. Source는 모든 bytes에 charset_normalizer를 적용합니다. 예를 들어 `caf\xe9`를 unmarked `application/octet-stream`으로 받으면 이 runtime은 UTF16BE `"\u6361\u66e9"`를 선택하고, UTF8-valid `00 41 00 42`도 `"AB"`로 추정합니다. malformed BOM에서도 다른 code page를 고를 수 있습니다. Go는 unmarked nonUTF8·NUL·malformed BOM 같은 detector-required 입력을 explicit `ErrUnsupported`로 돌려주며 전체 decoded bytes/status/header를 `resource.ResponseError`에 보존합니다. 지원되는 UTF8/BOM 사례가 전체 detector branch를 완료하는 증거는 아닙니다. binary 데이터에는 writer/path 동선을 사용하여 text 추정을 하지 않습니다.

성공 응답은 Source의 100..399 정책과 실제 body를 따르고, 별도 native `StreamObject`의 200/206/304 정책을 바꾸지 않습니다. 같은 SDK-owned gzip/x-gzip·zlib/raw deflate·zstd 및 최대 5개 encoding-chain decoder를 재사용합니다. actual 원래 Content-Encoding/Content-Length를 유지하기 위해 `Accept-Encoding: gzip, deflate, zstd`를 고정하고 selected client의 override는 스냅샷으로 보존합니다. clean 404가 되려면 거절 본문 Read·Close·압축 해제도 성공해야 합니다. corrupt compressed 404, read/Close/취소/source/native callback 실패를 부재로 숨기지 않습니다.

출력 순서는 accepted HTTP/status 뒤 원래 headers를 보존하고 path를 열고, 그 후에 accepted decoder를 생성·읽고 씁니다. 따라서 missing path open 오류는 깨진 gzip이나 여섯 encoding chain의 decoder 오류보다 먼저입니다. falsey outfile branch는 decoder도 시작하지 않습니다. clean 404이면 path를 열거나 기존 파일을 truncate하지 않습니다. Writer는 read 실패 chunk를 받지 않으며 write/short-write 실패 뒤 Flush하지 않습니다. 성공한 borrowed writer가 `Flush() error`를 지원하면 한 번 Flush하고, 지원하지 않는 unbuffered Go writer는 추가 interface 구현 없이 사용할 수 있습니다. Source는 short write 반환값을 무시하지만 Go는 `io.ErrShortWrite`로 명확히 실패합니다.

SDK가 연 path는 오류가 발생해도 닫으며 physical HTTP body도 한 번 정리합니다. Source path/write/iterator 실패의 finally cleanup 부재를 재현하지 않는 Go 개선입니다. write/flush 뒤에도 selected source/context를 guard하고, 취소·잠시 바뀌었다 복원된 endpoint·native 실제 read/Close 원인을 error chain에 보존합니다. HTTP 오류에는 native actual status/header/body가 남고 401 `ErrUnableToReauthenticate`의 `ErrOriginal`과 `ErrReauth`는 별도로 확인합니다. writer/output 실패에는 실제 receipt와 현재 decoded chunk 또는 accepted destination bytes가 남습니다. 경로 side effect 뒤 자동으로 GET을 재전송하지 않습니다.

Go 옵션은 output chunk size를 1..16MiB로 제한하고 실제 파일을 열기 전에 검증합니다. Python은 truthy path를 연 뒤 dynamic chunk type을 검사할 수 있으므로 이 invalid-input side-effect 순서는 Go 정책 차이입니다. memory 또는 falsey output에서는 resp_chunk_size를 Source처럼 사용하지 않습니다. container/object는 기존 SDK의 안전한 문자열 이름 검증과 전체 segment PathEscape를 따릅니다. 임의 mutable Resource/dirtyattrs, statistical detector, CPython codec registry와 session/cache foundation SDK-R1·C1·S1은 전체 지원으로 세지 않습니다.

[leaf 계약 테스트](v1/objects/python_cloud_get_test.go)와 [cloud 계약 테스트](cloud_object_get_test.go)는 기존 physical object fixtures 및 public Gophercloud testhelper 기반 testcloud를 재사용합니다. 첫 구현의 UTF32 alias와 PathError concrete type 수리를 Root focused/race 검사에서 확인한 뒤, Source-backed accepted decoder phase·NUL/BOM/known codec negative cases를 추가했습니다. 최종 수정본의 Go gate·예제 compile·전체 저장소 검사는 Root가 별도 기록하며 문서와 Source oracle만으로 PASS를 주장하지 않습니다.

최종 Root focused race 검사에서 Swift leaf 18개·cloud 3개 신규 test group과 Connection 연결 테스트가 통과했습니다. 같은 통합 run의 8개 package 전체는 67개 group·249개 subcase PASS였습니다. Source 65개 관찰, Python 예제 AST 및 문서/독립 파일의 byte 일치는 별도 증거입니다. 통계 detector와 CPython codec namespace의 두 named remaining은 이 테스트 성공 이후에도 unresolved로 유지합니다. 전체 저장소 검사·실제 cloud acceptance는 이 집중 검사와 구분합니다.
