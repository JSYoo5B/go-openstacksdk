# Swift 원시 객체 조회: openstacksdk와 Go

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [`get_object_raw(container, obj, query_string=None, stream=False)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L449-L484)이고 Gophercloud는 `v2.15.0`입니다. `swift.Objects.GetObjectRaw(ctx, container, object, options...)`는 기본 쿼리 생략과 buffered 조회를 SDK가 소유합니다. `WithRawObjectQueryString`은 이미 인코딩한 쿼리를 추가하고 `WithRawObjectStream(true)`는 읽을 스트림을 넘깁니다. Catalog의 `python:cloud/_object_store/ObjectStoreCloudMixin/get_object_raw` 한 행을 다루며 기존 `GetObject`, `DownloadObject`, `StreamObject`의 제한된 성공 상태나 metadata projection을 바꾸지 않습니다.

아래 Python은 독립 실행 예제입니다. HTTP 404·503 같은 응답도 원래 상태와 내용을 출력하며 HTTP 상태만으로 오류를 발생시키지 않습니다.

```python
import argparse
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--stream", action="store_true")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    response = conn.get_object_raw(
        args.container, args.object,
        query_string=args.query, stream=args.stream,
    )
    try:
        print(f"status={response.status_code}", file=sys.stderr)
        for chunk in response.iter_content(chunk_size=32768):
            sys.stdout.buffer.write(chunk)
    finally:
        response.close()
```

아래 Go는 독립 `main.go`입니다. 기본값은 이미 읽고 물리 body를 닫은 reader를 반환하며 `--stream`을 지정하면 동일한 결과 구조에 guarded reader를 반환합니다. [설치 안내](../../../docs/install.md)에 따라 module과 clouds.yaml을 설정합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func run(ctx context.Context, cloud, container, object, query string, stream bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { return err }
    response, requestErr := swift.Objects.GetObjectRaw(ctx, container, object,
        objects.WithRawObjectQueryString(query),
        objects.WithRawObjectStream(stream),
    )
    if response == nil { return requestErr }
    fmt.Fprintf(os.Stderr, "status=%d\n", response.StatusCode)
    _, readErr := io.Copy(os.Stdout, response.Body)
    closeErr := response.Body.Close()
    return errors.Join(requestErr, readErr, closeErr)
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    object := flag.String("object", "logs/part.bin", "literal object name; empty selects container")
    query := flag.String("query", "", "already encoded query string")
    stream := flag.Bool("stream", false, "transfer a guarded stream")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *container, *object, *query, *stream); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

| Python | Go | 동작 |
| --- | --- | --- |
| `get_object_raw("assets", "part.bin")` | `GetObjectRaw(ctx, "assets", "part.bin")` | 쿼리 없이 bodyless GET, 기본 buffered |
| `query_string="a=1&a=2"` | `WithRawObjectQueryString("a=1&a=2")` | 반복 값과 순서를 포함한 문자열을 그대로 보냅니다 |
| `stream=False` | 옵션 생략 또는 `WithRawObjectStream(false)` | opaque bytes를 모두 읽고 physical body를 한 번 닫습니다 |
| `stream=True` | `WithRawObjectStream(true)` | 헤더 직후 body를 넘기고 caller Read/Close 경계를 guard합니다 |
| `obj=""` | object 인자 `""` | container endpoint를 GET합니다 |
| 전체 옵션 교체 | `WithRawObjectOpts(RawObjectOpts{...})` | SDK 소유 값으로 QueryString·Stream을 교체합니다 |

`RawObjectResponse.StatusCode`와 `Header`는 실제 응답이고 `Body`는 caller가 닫아야 합니다. 응답을 JSON·문자·metadata로 decode하지 않아 NUL·잘못된 UTF-8·binary payload도 보존합니다. 유효한 HTTP 상태 100..999는 상태만으로 오류로 바꾸지 않으므로 304·401·404·429·498·503도 최종 깨끗한 응답이면 raw 결과입니다. Gophercloud의 깨끗한 401 재인증과 429/498 backoff는 설정한 provider 정책을 먼저 수행합니다. 상태 외에 read·Close·취소·source 변경·callback 실패가 섞인 응답은 오류이며 실제 status/header/partial bytes를 보존합니다. native rejection으로 처리한 최종 401/429/498의 physical body는 이미 읽고 닫혔으므로 stream 모드에서도 그 실제 바이트를 보관한 reader로 반환합니다. `ErrUnableToReauthenticate`는 두 원인을 Unwrap하지 않으므로 `ErrOriginal`과 `ErrReauth`를 별도로 확인합니다.

Buffered 오류에는 non-nil raw 결과와 오류가 함께 올 수 있습니다. 이미 받은 바이트는 결과 `Body`에 남고 underlying body는 닫힙니다. Streaming 오류는 `Read` 또는 `Close`에서 `resource.OperationError`·실제 `resource.ResponseError`로 나타납니다. 스트림은 전체 body를 오류 증거용으로 누적하지 않고 현재 읽기의 partial chunk, status/header와 원인을 보존합니다. source/취소/physical 실패 후에는 재요청하지 않으며 underlying body는 한 번만 Close합니다. 한 caller가 순서대로 읽고 도중에 그만두면 Close해야 합니다. 결과의 exported header를 바꿔도 오류의 wire 증거는 바뀌지 않습니다.

QueryString은 문자열이며 공백·fragment·ASCII control·잘못된 UTF-8을 HTTP 전에 거부합니다. 쿼리를 map으로 바꾸거나 다시 인코딩하지 않습니다. Python Requests의 자동 URI 재인코딩을 기대하는 입력은 caller가 먼저 인코딩해야 합니다. container는 SDK의 literal container 검증을 따르고 object는 `/`를 포함할 수 있습니다. Python `urllib.parse.quote`는 object 내부 `/`를 남기지만 Go는 한 opaque 이름을 `url.PathEscape`로 escape하므로 `/`와 literal `%2F`를 구분합니다. 이 Source cloud 호출의 container와 object는 문자열입니다. Requests cookie/session cache와 arbitrary mutable response object는 Go 호출 signature에 포함하지 않습니다.

이 메서드는 인증·route·framing·조건·선택한 version header의 SDK 경계를 유지합니다. HEAD/metadata 조회나 미리 존재 확인을 하지 않습니다. Swift 배포의 읽기 ACL·middleware·backend 처리는 서버가 결정하고 실제 cloud 권한/동작 확인은 fixture 검증과 별개입니다. [계약 테스트](python_raw_get_test.go)가 Source 기본값·상태·opaque bytes·옵션·물리 오류·native renewal·stream 수명을 검사합니다. Guide build와 fixture PASS만으로 실제 인증, cloud acceptance 또는 release readiness를 주장하지 않습니다.
