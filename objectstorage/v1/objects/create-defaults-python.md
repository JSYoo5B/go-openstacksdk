# 빈 Swift 객체와 native PUT 기본값

SDK의 `Objects.Create`는 `CreateOpts.Content`를 생략하면 실제 빈 객체 PUT을 보냅니다. Gophercloud v2.15.0의 nil reader MD5 계산 panic을 SDK가 막고, native ETag·header·query·응답 decode 계약은 유지합니다. typed nil reader는 `resource.ErrInvalidOption`이며 HTTP 요청을 보내지 않습니다.

다음 예제는 별도 모듈에서 빌드할 수 있습니다. 실제 실행에는 인증 가능한 OpenStack 환경과 기존 `backups` container가 필요합니다.

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }

    // Content omitted: one ordinary PUT, with the MD5 of empty bytes.
    empty, err := swift.Objects.Create(ctx, "backups", "empty.bin", objects.CreateOpts{})
    if err != nil { log.Fatal(err) }
    fmt.Printf("PUT ETag=%q transaction=%q\n", empty.ETag, empty.TransID)

    // The option snapshots these bytes. Reusing it starts a new reader.
    data := []byte("hello")
    body := objects.WithCreateBytes(data)
    data[0] = 'X'
    for _, name := range []string{"first.txt", "second.txt"} {
        result, err := swift.Objects.Create(ctx, "backups", name,
            objects.CreateOpts{ContentType: "text/plain"}, body,
            objects.WithCreateHeader("X-Object-Meta-Category", "example"))
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s PUT ETag=%q\n", name, result.ETag)
    }

    // Source-shaped bytes workflow: no native ETag calculation, and raw PUT
    // acknowledgement is retained separately from metadata observations.
    uploaded, err := swift.Objects.CreateObject(ctx, "backups", "source-empty.bin",
        objects.CreateObjectInput{Data: []byte{}})
    if err != nil { log.Fatal(err) }
    if uploaded.Ordinary != nil && uploaded.Ordinary.Acknowledgement != nil {
        fmt.Printf("source PUT status=%d\n", uploaded.Ordinary.Acknowledgement.StatusCode)
    }
}
```

openstacksdk의 동일한 빈 data 사용법은 다음과 같습니다.

```python
import openstack

conn = openstack.connect()
obj = conn.object_store.create_object("backups", "empty.bin", data=b"")
conn.object_store.create_object("backups", "first.txt", data=b"hello",
                                content_type="text/plain")
```

| 호출 | 생략 또는 빈 입력 | ETag 기본값 | 반환값 |
|---|---|---|---|
| Go `Objects.Create` | nil `Content`는 빈 객체, typed nil은 사전 오류 | native MD5 계산 | 실제 PUT 응답의 `CreateHeader` |
| Go `WithCreateBytes(nil)` 또는 `WithCreateBytes([]byte{})` | 빈 객체 | native MD5 계산 | 해당 `Create` 호출의 결과 |
| Go `Objects.CreateObject` | nonnil 빈 `Data`는 빈 객체, source 전체 생략은 사전 오류 | bytes 경로에서 계산하지 않음 | phase별 HTTP 증거를 보유한 `CreateObjectResult` |
| Python `create_object(data=b"")` | 빈 객체 | data 경로에서는 checksum 생성이 기본 false | mutable `Object` |
| Python `create_object(name=..., data=None)` | filename이 없으면 name을 로컬 파일 경로로 사용 | 파일 경로에서는 checksum 생성이 기본 true | 파일 업로드 경로에서 `None` |

`Create`와 `CreateObject`는 기존 container에 PUT합니다. pinned Python data 경로도 container 생성 호출 없이 `Object.create`에 위임합니다. Python docstring의 container 자동 생성 설명은 이 data 실행 경로의 증거가 아닙니다.

`Create`의 options는 순서대로 적용합니다. 마지막 `WithCreateOptions`로 Content를 nil로 바꾸면 빈 객체 기본값이 적용되며, 마지막 `WithCreateBytes`는 앞선 typed Content를 대체합니다. `NoETag: true`는 native checksum 생성을 생략하고, 명시한 `ETag`는 native 계약대로 `NoETag`와 함께 있어도 보냅니다. `ContentLength`·system header·query 값은 typed 입력과 기존 `WithCreateHeader`·`WithCreateQuery` 계약을 유지하며 SDK가 임의로 바꾸지 않습니다. nil/빈 Content의 native 0 길이 header는 별도 명시 없이 전송 계층이 처리합니다.

`WithCreateBytes`는 option 생성 시 입력 bytes를 복사하고 매 호출마다 새 reader를 생성합니다. 호출 도중 option 내부 reader를 공유하지 않습니다. 이 기본값 수정은 Gophercloud의 전체 retry 동작을 다시 구현하지 않습니다. 비어 있지 않은 body의 physical retry, 읽기·seek·close 소유권과 응답 증거를 제어해야 하면 [owned create workflow](create.md)의 `CreateObject`를 사용합니다. 기존 `CreateOpts.Content` reader는 upstream 방식으로 읽거나 seek할 수 있고, 요청 body로 넘겨지는 reader의 closure는 Go HTTP transport 계약을 따릅니다. MD5 사전 계산 중의 read/seek 오류는 원래 cause와 `Create/objects` 문맥을 유지합니다.

`Create`는 201·202만 성공으로 받고 반환 header는 그 PUT 응답에서 읽습니다. `CreateObject.Ordinary.Acknowledgement` 역시 실제 PUT 증거이며 별도의 HEAD 결과로 대체하지 않습니다. 이 문서는 native nil Content 미완료 항목을 해결합니다. Python `create_object`의 mutable Resource·adapter/session·image integration 등 전체 parity 미완료 항목은 [기존 create 계약](create.md)에 계속 기록합니다.

검증은 [create defaults 계약](create_defaults_test.go)의 실제 PUT·header·옵션 override·원인 오류·empty retry와 [기존 native 계약](native_contracts_test.go)을 함께 사용합니다. fixture는 저장소의 Gophercloud `testhelper` 기반 `internal/testcloud`를 재사용합니다.
