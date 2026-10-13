# Swift container cloud 검색: openstacksdk와 Go

고정 Source `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `search_containers(name=None, filters=None)`는 모든 container 페이지를 먼저 수집한 뒤 cloud `_utils._filter_list`를 적용합니다. Go `Cloud.SearchContainers`가 같은 순서를 처리하며 `Connection.SearchContainers`는 연결의 설정된 location도 자동으로 제공합니다.

```python
import openstack

conn = openstack.connect()
print([c.name for c in conn.search_containers(name="logs-*", filters={"count": 1})])
print(conn.search_containers(filters="[].name"))
print(conn.search_containers(filters="length(@)"))
```

전체 Go 예제입니다. 실제 실행에는 Swift account 접근 권한과 인증 환경이 필요합니다. 예제 빌드와 fixture 테스트는 live cloud 실행을 증명하지 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    matched, err := conn.SearchContainers(ctx,
        objectstorage.WithContainerSearchName("logs-*"),
        objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(`{"count":1}`)))
    if err != nil { log.Fatal(err) }
    for _, row := range matched.Records {
        fmt.Printf("name=%s page_status=%d\n", row.Resource.Body["name"], row.StatusCode)
    }
    names, err := conn.SearchContainers(ctx, objectstorage.WithContainerSearchExpression("[].name"))
    if err != nil { log.Fatal(err) }
    total, err := conn.SearchContainers(ctx, objectstorage.WithContainerSearchExpression("length(@)"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("names=%s total=%s\n", names.Value, total.Value)
}
```

서비스를 이미 얻었다면 `swift.Cloud.SearchContainers(ctx, options...)`를 호출합니다. 직접 client는 `objectstorage.New(client)`로 같은 API를 제공합니다. 이 직접 경로는 recorded token project ID를 요청 없이 읽고, 알 수 없는 cloud·region·project 이름은 null로 유지합니다. full configured location을 가진 호출은 `WithContainerSearchLocation(location)`으로 제공합니다. Root `Connection.SearchContainers`가 `CurrentLocation()` snapshot을 자동 주입하므로 application이 provider/session getter를 직접 구성할 필요는 없습니다.

| Python | Go | 계약 |
| --- | --- | --- |
| `name="logs-*"` | `WithContainerSearchName("logs-*")` | id/name의 정확한 일치 후 fnmatch glob; 중복 보존 |
| `filters={"count":1}` | `WithContainerSearchFiltersJSON(...)` | 선언된 Resource getter의 cloud mapping 비교 |
| `filters="[].name"` | `WithContainerSearchExpression("[].name")` | full JMESPath의 임의 JSON 결과 |
| filters 생략 | options 생략 또는 `WithoutContainerSearchFilters()` | 모든 행 반환 |
| 전체 options 교체 | `WithContainerSearchOpts(...)` | Headers·Filters·Location을 소유하는 snapshot |

`SearchContainers`는 기존 container `ListRecords`를 수집합니다. 기본 GET은 Swift account root로 가고 `Accept: application/json`을 설정합니다. 이름과 filters를 서버 query로 보내거나 조회 전에 서버 prefix로 바꾸지 않습니다. `X-Account-Container-Count` 및 실제 marker/link continuation은 기존 listing이 처리합니다. 중간 페이지가 malformed JSON, constructor collision, physical read/Close 오류, 취소 또는 source drift로 실패하면 성공한 부분 검색 결과를 만들지 않습니다. `AllRecords`에는 이미 수집한 실제 행, `LastPage`에는 마지막 실제 응답 receipt가 남고 `Value`·`Records`는 nil입니다. 기존 listing이 nil receipt를 제공하는 native pointer401도 `ErrOriginal`에서 마지막 실제 응답만 복구합니다. HTTP를 다시 보내거나 body를 다시 읽지 않습니다.

`Records`는 일반 mapping/name 검색의 행이며 각 `ContainerRecord`는 `Resource`, `Wire`, `Envelope`, `Header`, `StatusCode`를 구분합니다. Source에 선언된 Body/Header getter와 computed location은 `Resource.Body`에 있고, unknown 필드와 remote location은 `Wire.Body`에만 남습니다. 계정 HTTP Header를 각 container의 Header descriptor로 번역하지 않습니다. 새 Search view의 bool Header는 row SeedHeaders 원본에 CPython JSON-loaded truthiness를 적용하므로 `is_newest=1e-400`은 false, 문자열 `"false"`는 true이며 null은 null입니다. 기존 ListRecords의 exact decimal bool view는 변경하지 않고 Wire에는 원래 값이 남습니다. `Value`는 selection의 완전한 JSON입니다. JMESPath expression이면 null, bool, string, number, object 또는 임의 list일 수 있으므로 `Expression=true`, `Records=nil`입니다. 결과에 가상의 resource 행이나 HTTP 요청을 연결하지 않습니다. `AllRecords`는 expression 입력의 실제 행이며 선택된 행과 각각 독립된 bytes/maps/header를 소유합니다.

순서는 name/id selection 다음 filters입니다. filters가 null, false, 0, empty string/list/object이면 Source Python처럼 필터를 생략합니다. decimal/exponent 숫자의 underflow·overflow truthiness와 mapping equality에는 공통 `SelectLoaded`의 CPython JSON-loaded 정책을 사용합니다. 중첩 dict는 Source 삽입 순서대로 검사하므로 앞 조건의 불일치가 뒤의 잘못된 필드를 검사하지 않을 수 있습니다. 선택한 행이 없으면 잘못된 mapping field가 접근되지 않습니다. Truthy scalar/list filters는 선택한 실제 행에서 mapping keys가 없어 오류가 되지만, 비어 있는 선택에선 빈 결과입니다. 선언되지 않은 server field는 Wire에 있어도 Source getter가 아니므로 mapping filter에서 AttributeError에 해당하는 Go 오류가 됩니다. 비어 있지 않은 JMESPath 문자열은 선택이 비어 있어도 평가하므로 문법 오류를 유지합니다.

서비스 source는 ListRecords의 실제 preflight에서 옵션보다 먼저 캡처됩니다. Source registration 뒤 library-owned callback에서 search 옵션과 location을 복사하므로 validation을 위한 가짜 container 이름이나 HTTP 요청이 없습니다. Callback마다 maps/raw bytes/location pointer를 다시 소유하고 nil Header map은 초기화합니다. nil options, 취소, route 변경, Header 충돌은 HTTP 전에 실패합니다. 일단 수집한 상태에서 취소/source 변경이 생기면 선택 결과를 성공으로 반환하지 않습니다. 기록된 location은 callback·HTTP 이전에 snapshot하며 서버가 보낸 location으로 덮어쓰지 않습니다.

Go의 typed name/string·raw JSON 필터는 Python의 동적 객체 인자와 다릅니다. malformed JSON 입력에는 Python 객체로 직접 대응하는 값이 없으며 Go 오류로 남깁니다. mutable Python Resource·dirty/session/cache 및 공통 SDK-R1/C1/S1 계약은 별도 작업입니다. JMESPath 자체의 deprecated 경고는 Go에서 출력하지 않습니다. 기존 listing의 동일 origin/path/filter 고정, page bound와 더 강한 cycle detection은 계속 적용합니다. 인증과 Swift ACL/정책을 포함한 실제 cloud acceptance는 별도입니다.

[검색 계약 테스트](cloud_container_search_test.go)는 전체 페이지 수집, glob과 aliases, falsey·loaded numeric mapping, insertion-order 오류, 모든 JMESPath 결과 유형, location 소유권과 remote Wire 분리, malformed 뒤 페이지, source/callback 취소 및 원본 필터 소유권을 검증합니다.

실제 HTTP에서 Content-Length보다 짧은 후속 page로 `io.ErrUnexpectedEOF`를 만들고 200·203·401·404·429·503의 마지막 physical receipt와 native401 두 cause, callback 차단을 검증합니다. Close/source 실패는 재사용한 containers listing 계약 테스트의 별도 검증도 유지합니다.
