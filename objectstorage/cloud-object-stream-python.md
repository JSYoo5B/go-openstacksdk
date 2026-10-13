# Swift cloud 객체 스트리밍: openstacksdk와 Go

`Connection.StreamObjectChunks(ctx, container, object, options...)`는 고정 Source cloud `stream_object`의 문자열 container/object 동선을 lazy binary iterator로 제공합니다. 기본 chunk size는 SDK가 1024바이트로 고르고, `query_string`은 Source처럼 사용하지 않습니다. 스트림 생성·`Chunks()` 호출 자체는 Swift 발견이나 GET을 실행하지 않으며 첫 순회에서 실행합니다.

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [cloud stream_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L484), [Object.stream/_download](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L319), Gophercloud `v2.15.0`입니다. codec 비교 runtime은 CPython 3.14.8, Requests 2.34.2, urllib3 2.8.0입니다. Source 메서드와 Requests/urllib3를 실제로 실행하고 HTTP adapter만 메모리 응답으로 바꾼 41개 관찰에서 기본 헤더·chunk·codec·404·lazy 동작을 확인했습니다. 실제 cloud의 인증이나 ACL 검증은 이 증거에 포함하지 않습니다.

아래 Python은 독립 실행 예제입니다. `query_string`을 넘겨도 전송 URL에 붙지 않으며, 정상적으로 처리한 HTTP 404는 빈 순회로 종료합니다.

```python
import argparse
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--chunk-size", type=int, default=1024)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    stream = conn.stream_object(
        args.container, args.object,
        query_string=args.query, resp_chunk_size=args.chunk_size,
    )
    try:
        for chunk in stream:
            sys.stdout.buffer.write(chunk)
    finally:
        stream.close()
```

아래 Go는 독립 `main.go`입니다. [설치 안내](../docs/install.md)의 module/clouds.yaml 설정 후 `go run . --cloud devstack --container assets --object logs/part.bin > part.bin`처럼 실행할 수 있습니다. stdout에는 binary bytes, stderr에는 실제 HTTP status와 원래 Content-Encoding을 출력합니다. 쓰기 오류가 발생해도 `run`의 defer에서 body를 닫은 다음 `main`이 종료하므로 Close 오류를 함께 반환합니다.

```go
package main

import (
    "context"
    "errors"
    "flag"
    "fmt"
    "io"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func run(ctx context.Context, cloud, container, object, query string, chunkSize int) (err error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    stream := conn.StreamObjectChunks(ctx, container, object,
        objectstorage.WithObjectStreamChunkSize(chunkSize),
        objectstorage.WithObjectStreamQueryString(query),
    )
    defer func() {
        err = errors.Join(err, stream.Close())
        if receipt := stream.Receipt(); receipt != nil {
            fmt.Fprintf(os.Stderr, "status=%d encoding=%q\n",
                receipt.StatusCode, receipt.Header.Get("Content-Encoding"))
        }
    }()
    for chunk, readErr := range stream.Chunks() {
        var writeErr error
        if len(chunk) != 0 {
            var written int
            written, writeErr = os.Stdout.Write(chunk)
            if writeErr == nil && written != len(chunk) { writeErr = io.ErrShortWrite }
        }
        if readErr != nil || writeErr != nil {
            return errors.Join(readErr, writeErr)
        }
    }
    return stream.Err()
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    object := flag.String("object", "logs/part.bin", "literal object name")
    query := flag.String("query", "", "ignored by the pinned cloud stream_object")
    chunkSize := flag.Int("chunk-size", 1024, "decoded bytes per chunk; 1..16777216")
    timeout := flag.Duration("timeout", 5*time.Minute, "whole download deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *container, *object, *query, *chunkSize); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

서비스 API를 이미 골랐다면 `swift, err := conn.ObjectStorageV1(ctx)` 다음 `swift.Cloud.StreamObjectChunks(...)`를 사용할 수 있습니다. 가장 낮은 닫힌 동선은 `swift.Objects.StreamCloudObject(...)`이며 옵션은 `objects.WithCloudStreamChunkSize`·`WithCloudStreamQueryString`·`WithCloudStreamOpts`입니다. 어느 단계도 caller가 interface builder나 body wrapper를 구현하게 하지 않습니다.

| Python cloud 호출 | Go | 결과 |
| --- | --- | --- |
| `conn.stream_object("assets", "part.bin")` | `conn.StreamObjectChunks(ctx, "assets", "part.bin")` | lazy GET, SDK 기본 1024바이트 |
| `resp_chunk_size=32768` | `WithObjectStreamChunkSize(32768)` | decoded binary chunk의 최대 크기 |
| `query_string="symlink=get"` | `WithObjectStreamQueryString("symlink=get")` | Source처럼 무시; URL query 없음 |
| 전체 typed 옵션 | `WithObjectStreamOpts(ObjectStreamOpts{...})` | pointer 기본값을 소유한 값으로 복사 |
| generator 순회 | `for chunk, err := range stream.Chunks()` | 오류가 있으면 partial chunk와 오류를 함께 받을 수 있음 |
| 순회 도중 중단 | loop의 `break`, 이후 `stream.Err()` | physical body 한 번 Close, cleanup 오류 확인 |
| generator를 시작하지 않고 닫기 | `stream.Close()` | Swift 발견·옵션 실행·GET 없이 종료 |

`ObjectChunkStream`의 `Chunks`, `Err`, `Close`, `Receipt` 구현은 SDK가 소유합니다. 한 handle은 한 번만 소비하며 다시 순회해도 GET을 반복하지 않습니다. EOF·read/decoder 오류·consumer break·consumer의 Close 때 응답을 한 번 닫습니다. 정상 순회 후에도 `Err()`를 확인하고, 순회를 중단한 경우에는 `Err()` 또는 `Close()`로 cleanup 실패를 확인합니다. Close를 GET 중 호출하면 늦게 도착한 응답은 선읽기 없이 닫히고 실제 receipt를 남깁니다. 하나의 goroutine에서 순서대로 순회·Close하세요. 이미 진행 중인 blocking Read를 Close로 끊는 동작은 제공하지 않으며 network Read 취소에는 context를 사용합니다.

`Receipt()`는 실제 응답의 status/header만 복사하며 payload를 누적하지 않습니다. caller가 receipt 헤더나 반환한 오류의 공개 `Body`·`Header`를 바꿔도 이후 `Receipt`/`Err`의 증거는 바뀌지 않습니다. 실패한 physical Read의 현재 partial bytes는 `resource.ResponseError`에 남고, 먼저 성공한 chunks는 caller가 소유합니다. decoder 실패만 발생한 경우에는 actual status/header와 원인이 남으며 전체 성공 payload를 오류 증거용으로 쌓지 않습니다.

Source `Object._download`는 상태가 400 미만이면 성공으로 처리합니다. 이 adapter는 유효한 HTTP 상태 100..399를 받아들이며 별도 native `Objects.StreamObject`의 200/206/304 정책을 바꾸지 않습니다. 304의 실제 body가 adapter에서 제공되면 순회하며, 실제 HTTP transport가 304 body를 비웠다면 빈 순회입니다. clean 404는 빈 성공이고 actual 404 receipt를 남깁니다. 압축 본문은 decode가 성공해야 clean 404가 될 수 있습니다. 404에 read·Close·decode·source 변경·context 취소가 섞이면 오류를 그대로 남깁니다. 다른 4xx/5xx의 native actual status/header/body도 error chain에 보존합니다.

기본 request headers는 `Accept: */*`, `Accept-Encoding: gzip, deflate, zstd`이고 `X-Newest`를 임의로 추가하지 않습니다. selected client의 명시적 `Accept-Encoding`은 스냅샷으로 유지하며 native retry가 이를 바꾸면 오류입니다. explicit Accept-Encoding은 Go transport의 자동 gzip 해제를 막으므로 Receipt의 원래 Content-Encoding/Content-Length와 decoded 출력 bytes를 구분할 수 있습니다. gzip·x-gzip·zlib/raw deflate·zstd와 최대 5개 encoding chain을 점진적으로 decode합니다. 단일 unknown/br는 비교 runtime에서 Source decoder가 없는 경우처럼 opaque bytes입니다. 정상 raw deflate가 zlib처럼 보이는 prefix를 가져도 Source의 raw fallback을 적용합니다. gzip/deflate는 Source처럼 깨끗하게 끝난 응답의 불완전한 trailer를 허용하고, gzip의 partial header는 빈 결과가 될 수 있습니다. 완전한 gzip member 뒤의 trailing garbage는 점진적으로 버리며 실제 clean EOF를 확인합니다. 초기 잘못된 magic·CRC 오류와 physical read/Close·취소·source 오류는 숨기지 않습니다. optional brotli 설치 여부나 Requests의 임의 session/HTTP adapter를 Go API로 가져오지는 않습니다.

깨끗한 native 401 재인증과 429/498 backoff는 provider에 설정한 정책으로 수행합니다. physical read·Close·source·취소가 실패한 응답은 retry/reauth/backoff callback에 전달해 재전송하지 않습니다. `gophercloud.ErrUnableToReauthenticate`는 두 원인을 Unwrap하지 않으므로 `ErrOriginal`의 실제 401 receipt와 `ErrReauth` 원인을 별도로 확인합니다. 요청 container/object, selected provider/endpoint/version과 authentication/route/framing 정책은 iteration 수명 동안 guard합니다.

Go 차이는 명시적입니다. chunk size는 1..16MiB의 typed `int`이고 nil `ChunkSize`는 SDK 기본값을 뜻합니다. Python의 `None`, 음수, bool, 0, float 분기와 오류 발생 순서를 재현하지 않습니다. Go는 short reads와 HTTP framing 조각을 일관된 최대 chunk 크기로 재조립하므로 Source Requests가 실제 chunked framing 경계에서 더 짧게 나누는 segmentation과 다를 수 있으나 binary content는 보존합니다. container/object는 검증한 문자열 이름이며 기존 Go Swift 정책대로 이름 전체를 PathEscape하여 object 내부 `/`와 literal `%2F`를 구분합니다. Source utils.urljoin의 바깥 slash 제거나 arbitrary mutable Object/Container·dirty attrs를 받는 일반 Proxy 동선은 이 닫힌 signature에 포함하지 않습니다. Go는 partial bytes+error를 돌려주고 자동으로 body를 정리합니다. Source의 generator 반환과 완전히 같은 객체 수명/exception 객체를 주장하지 않습니다.

Catalog 검토 대상은 `python:cloud/_object_store/ObjectStoreCloudMixin/stream_object` 한 선언입니다. `python:object_store/v1/stream_object`의 mutable Resource/attrs/session 전반 및 기존 native `StreamObject`의 지원 판정은 그대로 둡니다. 전체 Python Resource/Proxy constructor/session/cache, Connection foundation/공통 SDK-R1·C1·S1 잔여를 완료로 세지 않습니다. [leaf 계약 테스트](v1/objects/python_cloud_stream_test.go)와 [cloud 계약 테스트](cloud_object_stream_test.go)는 기존 physical object fixture·public Gophercloud testhelper 기반 testcloud를 재사용합니다. Root 통합 Go/race 검사와 예제 build 결과는 별도 보고하며 문서나 oracle 성공만으로 release 또는 live acceptance를 주장하지 않습니다.

이번 후보의 최종 focused 통합 검사에서는 leaf 21개·cloud 3개·Connection 4개, 총 28개 신규 test group이 통과했습니다. 실제 판정 근거는 root가 실행한 3개 Go package 결과이며 Source 41개 관찰과 구분합니다. Python 예제 AST와 문서/독립 Go·Python 파일의 byte 일치도 확인했습니다. 전체 저장소 검사와 실제 cloud 검증은 별도 범위입니다.


최종 root 검증에서 정확한 가이드 Go main2개 build·Python AST2개와 저장소204개 실제 test package 전체 make check가 PASS입니다. 실행 로그는 `/private/tmp/go-openstacksdk-cycle6-check-first.log`, 예제 hash receipt는 `...cycle6-exact-guides/build-receipt.json`입니다. 실제 인증·live cloud·전체 SDK supported 판정은 별도 잔여로 유지합니다.
