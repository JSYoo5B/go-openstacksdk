# Swift object 바이트 읽기

`objects.API.GetObject`는 응답을 `[]byte`로 읽고, `DownloadObject`는 caller가 넘긴 `io.Writer`에 복사하며, `StreamObject`는 즉시 GET을 시작한 뒤 caller가 닫을 `io.ReadCloser`를 반환합니다. 세 메서드 모두 문자 디코딩, 캐시 조회, 사전 HEAD 없이 literal container/object 경로로 GET을 보냅니다. container는 비어 있거나 `/`, `\`, dot segment, ASCII control을 포함할 수 없습니다. object는 `/`를 포함할 수 있지만 비어 있거나 dot segment, `\`, ASCII control, 잘못된 UTF-8을 포함할 수 없습니다. 전체 object 이름은 한 번 escape되므로 `/`와 literal `%2F`를 구분합니다.

성공 상태는 200, 206, 304입니다. 206의 단일 range·multipart body도 그대로 바이트로 전달합니다. 304는 `NotModified=true`인 성공이며 body를 읽거나 writer에 쓰지 않고 닫습니다. `Complete`는 EOF까지 읽고 해당 바이트를 전달했거나 304를 처리했다는 뜻입니다. ETag, Content-Length, multipart 구조를 검증했다는 뜻은 아니며, EOF 뒤 Close 오류가 있어도 `Complete`는 유지될 수 있습니다.

각 결과의 `Header`와 `StatusCode`는 실제 응답입니다. `Metadata`는 기존 `MetadataInfo`의 nullable header projection을 사용합니다. 누락과 빈 문자열을 구분하고, malformed count나 선택된 header의 중복은 전체 projection을 실패시킵니다. 이때 body를 임의로 drain하지 않고 닫으며, 결과에는 실제 header/status가 남고 `Metadata`는 nil입니다. body 읽기 이후 오류가 생기면 이미 관찰한 metadata와 실제 바이트·counter는 보존합니다. 에러 반환과 non-nil 결과가 함께 올 수 있습니다.

다음 예제는 구성된 API, 실제 ETag·기준 시각·version ID를 받습니다. full options의 날짜 조건을 각 요청에 맞게 해제하고 바꿉니다. `VersionID`, `Symlink`, `MultipartManifest`, `Filename`은 server에 전달할 literal query이며 middleware 설치나 대상 object의 종류를 확인하지 않습니다.

```go
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"gophercloudsdk/objectstorage/v1/objects"
)

func readObjects(ctx context.Context, api *objects.API, etag string, asOf time.Time, version string) error {
	common := []objects.ObjectReadOption{
		objects.WithObjectReadOpts(objects.ObjectReadOpts{
			Headers:           map[string]string{"X-Audit": "initial"},
			IfModifiedSince:   &asOf,
			IfUnmodifiedSince: &asOf,
		}),
		objects.WithObjectReadHeaders(map[string]string{"X-Audit": "read"}),
		objects.WithObjectReadHeader("X-Audit", "read-final"),
		objects.WithObjectReadNewest(false),
		objects.WithObjectReadBufferSize(32 * 1024),
	}
	getOpts := append(append([]objects.ObjectReadOption(nil), common...),
		objects.WithoutObjectReadIfUnmodifiedSince(),
		objects.WithObjectReadIfModifiedSince(asOf),
		objects.WithObjectReadIfNoneMatch(etag),
	)
	buffered, err := api.GetObject(ctx, "backups", "report.txt", getOpts...)
	if buffered != nil {
		fmt.Printf("GET status=%d bytes=%d not-modified=%t complete=%t\n",
			buffered.StatusCode, len(buffered.Body), buffered.NotModified, buffered.Complete)
	}
	if err != nil {
		return err
	}

	copyOpts := append(append([]objects.ObjectReadOption(nil), common...),
		objects.WithoutObjectReadIfModifiedSince(),
		objects.WithObjectReadIfUnmodifiedSince(asOf),
		objects.WithObjectReadIfMatch(etag),
		objects.WithObjectReadRange("bytes=0-1023"),
		objects.WithObjectReadFilename("report.part"),
		objects.WithObjectReadVersionID(version),
	)
	var output bytes.Buffer
	copied, err := api.DownloadObject(ctx, "backups", "report.txt", &output, copyOpts...)
	if copied != nil {
		fmt.Printf("copied=%d complete=%t\n", copied.BytesWritten, copied.Complete)
	}
	if err != nil {
		return err
	}

	plain := append(append([]objects.ObjectReadOption(nil), common...),
		objects.WithoutObjectReadIfModifiedSince(),
		objects.WithoutObjectReadIfUnmodifiedSince(),
		objects.WithoutObjectReadNewest(),
	)
	manifestOpts := append(append([]objects.ObjectReadOption(nil), plain...),
		objects.WithObjectReadMultipartManifest("get"),
	)
	streamed, err := api.StreamObject(ctx, "backups", "manifest", manifestOpts...)
	if err != nil {
		return err
	}
	_, readErr := io.Copy(io.Discard, streamed.Body)
	closeErr := streamed.Body.Close()
	fmt.Printf("stream bytes=%d complete=%t\n", streamed.BytesRead, streamed.Complete)
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}

	linkOpts := append(append([]objects.ObjectReadOption(nil), plain...),
		objects.WithObjectReadSymlink("get"),
	)
	_, err = api.GetObject(ctx, "backups", "report-link", linkOpts...)
	return err
}

func main() {}
```

`DownloadObject`는 writer를 빌리고 Close·Flush·Seek·truncate를 하지 않습니다. 읽은 buffer를 쓰기 전에 context/source를 다시 확인하며, 실제 성공한 write 바이트만 `BytesWritten`에 셉니다. `GetObject`는 읽기 오류 전까지 확보한 바이트를 `Body`에 남깁니다. 둘의 loop buffer는 기본 32 KiB이고 `BufferSize`는 1부터 16 MiB까지 지정할 수 있으며 0은 기본값입니다. invalid read/write count, short write, 반복적인 no-progress도 오류로 처리합니다. 이미 body가 노출되거나 쓰기가 시작되면 HTTP를 재시도하지 않습니다.

스트림은 한 caller가 순서대로 읽어야 하며 Read와 Close를 동시에 호출하지 않습니다. EOF, 읽기 오류, context 취소, source 변경을 관찰하면 underlying body를 한 번 닫습니다. 도중에 읽기를 그만두면 caller가 Close해야 합니다. 반복 Close는 같은 cleanup 결과를 반환하고, 읽기가 끝난 뒤에는 terminal 상태를 유지합니다. `BytesRead`는 전달한 실제 바이트 수입니다. Download/Stream의 오류 증거는 body 전체를 누적하지 않으며, caller가 exported result를 바꿔도 내부의 원래 wire header/status로 오류를 만듭니다.

full options는 설정을 교체하고 각 helper/callback 뒤의 map·pointer는 복사됩니다. `Newest`의 nil은 header 생략, `false`는 명시한 false입니다. `Without...`는 presence를 해제합니다. 날짜는 UTC의 HTTP GMT 형식으로 보냅니다. Range와 ETag 조건은 literal header로 전달하며 locally 해석하거나 trim하지 않습니다. 네 typed query는 UTF-8을 보존하되 ASCII control/DEL을 거부합니다. 일반 source/options header로 auth, framing, metadata mutation, range/조건/newest header를 덮어쓰지 않고 전용 helper를 사용합니다.

원래 API/client/provider와 endpoint/resource base/type/microversion을 workflow와 실제 Read/Write/Close 경계에서 확인하고, configured native retry·reauth·redirect 정책은 body를 노출하기 전의 기존 동작을 따릅니다. 명시한 native RetryFunc의 advanced header 변경은 유지하지만 method/URL/body ownership guard와 200·206·304 제한을 넓힐 수 없습니다. 클라이언트 설정은 병렬 요청 전에 완료해야 합니다. 별도 goroutine이 abandoned stream을 자동 정리한다는 보장은 없습니다.

Python의 [고정된 세 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L288-L396)는 다른 반환 계약입니다. `get_object`는 매번 GET하고 mutable Object의 header를 갱신합니다. 기본 `remember_content=False`에서는 내용을 저장하지 않고, True이면 `response.text`를 `data`에 저장합니다. `outfile`은 경로 문자열을 열고 닫거나 borrowed file-like writer에 쓰고 성공 후 flush하는 별도 옵션입니다. 기존 Object의 `container`가 있으면 explicit container보다 먼저 사용합니다. Go는 container/object 문자열을 명시하고 bytes/writer/closeable stream을 분리합니다.

```python
import openstack

conn = openstack.connect()

remembered = conn.object_store.get_object(
    "report.txt", container="backups", remember_content=True
)
print("remembered text characters:", len(remembered.data))

data = conn.object_store.download_object("report.txt", container="backups")
print("downloaded bytes:", len(data))

chunks = conn.object_store.stream_object(
    "report.txt", container="backups", chunk_size=32 * 1024
)
for chunk in chunks:
    print("chunk bytes:", len(chunk))
```

Python `download_object`는 `response.content` bytes를 반환하고, `stream_object`는 호출 때 GET을 시작한 뒤 `iter_content(..., decode_unicode=False)` iterator를 반환합니다. 이 세 proxy의 Resource/session/coercion·텍스트 기억·파일 경로 처리와 Go 결과/소유권은 완전한 parity가 아닙니다. 기존 generated `Objects.Download`도 이미 caller-owned stream을 지원하고 native `Get`은 HEAD metadata 조회이므로, 그 API를 대체하지 않습니다.

[Swift GET API reference](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L16-L151)는 body와 header metadata를 설명합니다. 실제 [range/conditional 응답](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/swob.py#L1350-L1478)과 [기존 native Download](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L103-L171)가 206·304를 포함합니다. symlink/version/large-object middleware 결과는 server가 결정하며 client는 받은 payload를 그대로 전달합니다.
