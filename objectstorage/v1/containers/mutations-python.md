# Swift Container 생성·삭제: openstacksdk와 Go

`Containers.CreateRecord`와 `Containers.DeleteRecord`는 pinned Python의 이름 기반 `create_container`·`delete_container` 호출을 SDK가 소유한 옵션·변환·응답 규칙으로 제공합니다. 기존 native `Create`·`Delete`, strict ACK `CreateContainer`·`DeleteContainer`는 기존 계약을 유지합니다.

```python
import openstack

conn = openstack.connect()
container = conn.object_store.create_container(
    "assets", read_ACL=".r:*", is_content_type_detected=False,
    history_location="", metadata={"owner": "team", "disabled": False},
)
print(container.name, container.metadata)
conn.object_store.delete_container("assets")
conn.object_store.delete_container("assets", ignore_missing=False)
```

Go에서 같은 named workflow를 사용하는 standalone 예제입니다. 실행에는 인증 환경과 해당 container의 생성·삭제 권한이 필요합니다.

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }
    created, err := swift.Containers.CreateRecord(ctx, "assets", map[string]any{
        "read_ACL": ".r:*",
        "is_content_type_detected": false,
        "history_location": "",
        "metadata": map[string]any{"owner": "team", "disabled": false},
    }, containers.WithRecordCreateHeader("X-Client-Request-Id", "create-assets"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("PUT status=%d name=%s observed_metadata=%v\n",
        created.StatusCode, created.Resource.Body["name"], created.Metadata)

    deleted, err := swift.Containers.DeleteRecord(ctx, "assets")
    if err != nil { log.Fatal(err) }
    fmt.Printf("DELETE status=%d ignored_missing=%t\n", deleted.StatusCode, deleted.IgnoredMissing)

    // To reproduce ignore_missing=False, select the concrete option explicitly.
    _, err = swift.Containers.DeleteRecord(ctx, "assets",
        containers.WithRecordDeleteIgnoreMissing(false))
    if err != nil { fmt.Println("strict deletion:", err) }
}
```

| Python | Go | 계약 |
|---|---|---|
| `create_container(name, **attrs)` | `CreateRecord(ctx, name, attrs, options...)` | one bodyless PUT, declared Resource projection과 physical HTTP evidence |
| `delete_container(name)` | `DeleteRecord(ctx, name, options...)` | one DELETE; clean404는 기본 무시 |
| `ignore_missing=False` | `WithRecordDeleteIgnoreMissing(false)` | 404 원래 HTTP 오류 유지 |
| strict ACK convenience | 기존 `CreateContainer`·`DeleteContainer` | 생성201/202·삭제204의 기존 좁은 성공 규칙 |

`CreateRecord`의 attrs는 Container에 선언된 Body·Header 이름을 받습니다. Header의 wire 이름도 대소문자 구분 없이 받으며 `count`·`bytes`는 Body, `object_count`·`bytes_used`는 Header seed입니다. `name`은 이미 필수 argument이므로 attrs에서 중복 지정하면 사전 오류이고, `id`는 Source `Container.new`처럼 별도의 route override로 쓰지 않습니다. unknown attrs는 Source처럼 무시하며 unknown attribute에 들어 있는 custom JSON callback도 호출하지 않습니다.

Body는 서버에 보내지 않습니다. Source `_transform_create_request`가 body를 None으로 바꾸지만, 해당 Body seed는 생성 결과에 남습니다. 일반 Header scalar는 JSON-domain Python `str()` spelling을 사용하므로 null→`None`, bool→`True`·`False`이며 string list는 `", "`로 join합니다. Header list의 null·number는 사전 오류입니다. nil attrs와 nil/null/빈 metadata 모두 bodyless PUT을 수행하고, 명시한 false·빈 문자열·null Header는 생략하지 않습니다.

생성 metadata는 setter와 다릅니다. Source constructor가 키를 먼저 `X-Container-Meta-`로 prefix하기 때문에 `metadata={"content_type": ...}`는 custom metadata가 됩니다. 실제 Content-Type은 `attrs["content_type"]`으로 지정합니다. exact prefixed metadata key는 그대로 사용하지만 lowercase already-prefixed 키는 후속 case-sensitive 계산에 의해 다시 prefix되는 pinned Source 동작을 유지합니다. metadata는 요청 Header를 overlay하지만 저장된 Header seed를 바꾸지 않습니다. Go map에서 동일한 field/header를 가리키는 alias가 충돌하면 HTTP 전에 거부합니다. 이는 Python 입력 순서에 의존하는 마지막 값 선택을 Go map의 불안정한 순서로 재현하지 않기 위한 명시적 차이입니다.

반환값은 다음 carrier를 구분합니다.

- `SeedBody`·`SeedHeaders`: caller의 실제 선언된 입력 snapshot입니다.
- `Resource.Body`: seed와 응답의 declared Body/Header를 overlay하고 descriptor를 적용한 view입니다. unknown field는 포함하지 않습니다.
- `Wire.Body`: 응답에서 읽은 모든 JSON field를 유지합니다. Body가 JSON으로 parse되지 않으면 빈 map이며 원본은 `Envelope`에 있습니다.
- `Envelope`·`Header`·`StatusCode`: 실제 PUT 응답입니다. decoded view와 별도의 bytes/header를 소유합니다.
- `Metadata`: 응답에서 관측한 custom metadata이며 입력을 그대로 성공 결과로 간주하지 않습니다. Source가 Header descriptor로 소비한 temp URL key는 이 custom metadata map에서 제외합니다.

응답 Body가 빈 값 또는 malformed JSON이면 Source처럼 Body seed를 유지합니다. valid JSON scalar·array·null은 Resource mapping으로 번역할 수 없어 실제 응답 증거가 포함된 오류가 됩니다. 응답 declared Body field는 seed를 덮어씁니다. 명시한 null은 값이 없는 field와 다르고 alias fallback을 차단합니다. `count`↔`object_count`, `bytes`↔`bytes_used`는 한쪽이 없을 때만 alias를 사용하며 integer descriptor는 shared Source conversion을 사용합니다. bool Header의 응답 문자열 `"false"`는 Python의 `bool("false")`에 따라 true입니다. header가 응답에 없고 seed가 false이면 false를 유지합니다.

성공 status는 Source `raise_from_response`의 `<400`, 실제 HTTP100–399 범위입니다. 최종1xx 응답을 노출하지 않는 실제 HTTP client의 정상화는 별도 전송 계층 계약입니다. 성공은 PUT ACK이며 추가 HEAD·존재 조회·자동 container refresh를 수행하지 않습니다. 실제 Header가 없는 속성에 seed가 남는 것은 새로 관측한 서버 상태를 의미하지 않습니다. `location`의 nullable placeholder와 mutable Resource·Adapter/session/cache·dirty 재사용 전체 동작은 공통 SDK 계약의 별도 추적 항목입니다.

`DeleteRecord`는 404를 성공 status로 넣지 않습니다. native 오류·retry handling을 거친 뒤 원인이 clean404만 있을 때 기본값으로 무시하고 실제404 receipt의 `IgnoredMissing`을 설정합니다. physical read/Close failure, cancellation, source 변경이 섞인404는 무시하지 않습니다. 403·409 같은 서버 정책·nonempty 오류도 그대로 남습니다. accepted 응답 read/Close/source 오류는 부분 receipt와 `resource.ResponseError`를 보존하며 성공 응답 handling 실패를 replay하지 않습니다.

서비스 source는 custom JSON 변환과 option callback 전에 고정하고, attrs·nested metadata를 options 전에 snapshot합니다. Request의 Source attr·metadata Header 및 bodyless framing은 retry에서도 보존합니다. Go는 instance별 metadata를 소유해 pinned Source class-level metadata 공유로 인해 다른 생성에 입력이 유출되는 문제를 피합니다. dictionary `str()`는 Go JSON snapshot의 안정된 key 순서를 사용하며 Python dict insertion order와 다를 수 있습니다. HTTP Header alias/repeated value의 모호성은 실제 응답 증거를 남긴 projection 오류로 반환합니다.

[계약 테스트](python_record_test.go)는 실제 wire/header/default status·seed/response/alias/null projection·native404 retry·물리 실패·source/callback·retry ownership과 독립 metadata defaults를 검증합니다. 기존 Swift fixture와 Gophercloud public `testhelper` 기반 `internal/testcloud`를 재사용합니다. 서버 ACL·quota·storage policy·replication·container emptiness와 authenticated cloud acceptance는 별도 확인 단계입니다.
