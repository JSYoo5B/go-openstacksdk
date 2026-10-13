# Swift object cloud 검색: openstacksdk와 Go

openstacksdk 고정 commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 cloud `search_objects(container, name=None, filters=None)`는 `list_objects(container)`의 모든 페이지를 먼저 수집하고 `_utils._filter_list`를 적용합니다. Go `Cloud.SearchObjects`와 `Connection.SearchObjects`가 이 순서를 소유합니다.

```python
import openstack

conn = openstack.connect()
objects = conn.search_objects("assets", name="logs-*", filters={"content_length": 5})
print([obj.name for obj in objects])
print(conn.search_objects("assets", filters="[].name"))
print(conn.search_objects("assets", filters="length(@)"))
```

전체 Go 연결 예제입니다. 실행에는 Swift account 접근 권한과 인증 환경이 필요합니다. 문서 예제 빌드·HTTP fixture 검증은 실제 cloud acceptance를 의미하지 않습니다.

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
    matched, err := conn.SearchObjects(ctx, "assets",
        objectstorage.WithObjectSearchName("logs-*"),
        objectstorage.WithObjectSearchFiltersJSON(json.RawMessage(`{"content_length":5}`)))
    if err != nil { log.Fatal(err) }
    for _, row := range matched.Records {
        fmt.Printf("name=%s bytes=%s page_status=%d\n",
            row.Resource.Body["name"], row.Resource.Body["content_length"], row.StatusCode)
    }
    names, err := conn.SearchObjects(ctx, "assets", objectstorage.WithObjectSearchExpression("[].name"))
    if err != nil { log.Fatal(err) }
    total, err := conn.SearchObjects(ctx, "assets", objectstorage.WithObjectSearchExpression("length(@)"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("names=%s total=%s\n", names.Value, total.Value)
}
```

이미 선택한 서비스는 `swift.Cloud.SearchObjects(ctx, "assets", options...)`, 직접 client는 `objectstorage.New(client).SearchObjects(...)`로 같은 API를 사용합니다. 별도 builder나 페이지별 decoder·selector를 개발자가 만들지 않습니다.

| Source cloud 입력 | Go 옵션 | 동작 |
| --- | --- | --- |
| container | 필수 string argument | URI의 container를 한 번 escape하고 고정 |
| name 또는 fnmatch glob | `WithObjectSearchName` | id/name 정확 일치 후 glob; 중복 보존 |
| dictionary filters | `WithObjectSearchFiltersJSON` | Resource getter의 ordered/nested mapping equality |
| JMESPath string filters | `WithObjectSearchExpression` | null/scalar/object/list 등 임의 JSON 결과 |
| filters 생략 또는 falsey | 생략·`WithoutObjectSearchFilters` | 전체 수집한 행 반환 |
| 전체 옵션 교체 | `WithObjectSearchOpts` | raw filters·Headers·Location 모두 snapshot |

Source object listing은 선택한 container에 bodyless `GET`을 보내고 `format=json`, `Accept: application/json`을 사용합니다. Cloud name/filters는 서버 prefix/query로 바꾸지 않습니다. 기존 `Objects.ListRecords`의 `X-Container-Object-Count`·marker/limit/link continuation과 physical response 처리를 재사용합니다. 중간 page가 실패하면 이미 수집한 `AllRecords` 및 마지막 실제 `LastPage` receipt를 제공하되 성공한 `Value`나 `Records`를 만들지 않습니다. 성공한 페이지에서 발견한 행과 실패한 page를 섞어 검색 성공으로 반환하지 않습니다.

`ObjectSearchResult.Records`는 일반 name/dictionary 선택에서 HTTP 행과 대응하는 independent `ObjectRecord`입니다. `Resource.Body`는 Source Object getter 이름인 `_bytes`, `_content_type`, `_hash`, `content_length`, `content_type`, `etag`, `updated_at`, 선언된 Header와 computed location 등을 제공합니다. 응답의 `bytes`·`content_type`·`hash`가 Body에 있으면 해당 private getter와 Header alias가 연결됩니다. 선언되지 않은 `symlink_path`·`subdir`·foreign 필드는 `Wire.Body`에만 남습니다. `ObjectRecord.Container`는 실제 URI container이고 remote row의 container는 Wire에 보존합니다.

URI의 `container`와 BaseResource의 plain `metadata`는 Source `to_dict()`에 들어가는 Body/Header/Computed descriptor가 아닙니다. `metadata`는 `ObjectRecord.Metadata`에서 원본 JSON 값과 `X-Object-Meta-` prefix를 유지하며, Source cloud mapping에서 `metadata`, `container` 또는 server spelling `bytes`를 필드로 검사하면 잘못된 filter 오류가 됩니다. `Resource.Body["_bytes"]` 또는 `content_length`처럼 실제 getter 이름을 사용합니다. Metadata를 가상의 Resource 필드로 넣거나 계정 response Header를 각 object의 Header descriptor로 바꾸지 않습니다.

Location의 기본값은 Source 결합에 따라 구분합니다. Source `Proxy._get_connection()`이 None인 직접 Resource는 `location=null`입니다. Object의 container URI가 있으므로 remote row location은 computed로 받아들이기 전에 버려집니다. 연결된 Source Proxy는 Resource `_collect_attrs`에서 `Connection.current_location`을 computed 기본값으로 넣습니다. 따라서 직접 Go `Cloud.SearchObjects`는 실제 기본null을 유지하고, `Connection.SearchObjects`는 SDK가 `CurrentLocation()`을 한 번 snapshot해 자동 전달합니다. 구체적인 `WithObjectSearchLocation` override는 별도 소유하며 whole DTO reset 뒤 location=nil이면 Connection의 기본값으로 돌아갑니다. 인증·추측으로 별도의 location을 만들지 않습니다.

Filters는 전체 페이지를 수집한 다음 name/id 선택 뒤 처리합니다. null·false·0·underflow float·빈 string/list/dict는 Source truthiness로 생략합니다. 공통 `cloudfilter.SelectLoaded`를 재사용해 CPython JSON-loaded numeric equality와 float truthiness를 적용합니다. mapping의 조건은 raw JSON 삽입 순서로 검사하므로 앞의 불일치가 뒤의 잘못된 key 접근을 피할 수 있고, 이름 선택에 결과가 없으면 dictionary의 잘못된 key를 검사하지 않습니다. JMESPath 문자열은 이름 선택이 비어 있어도 평가하므로 잘못된 문법은 오류입니다. Header bool/int/list/dict/str descriptor와 Body/Header alias는 기존 Object record 모델이 소유합니다.

JMESPath 출력은 임의 JSON이라 `Expression=true`, `Value`에 결과를 저장하고 `Records=nil`로 둡니다. null은 JSON `null`, 빈 행 결과는 JSON `[]`, 실패한 선택은 `Value=nil`로 서로 다릅니다. Expression 결과가 object/list여도 가상의 HTTP 행과 연결하지 않습니다. `AllRecords`는 실제 expression 입력이며 `Resource`·`Wire`·`Metadata`·`Envelope`·`Header` maps/bytes는 selected Records와 독립적으로 소유합니다.

실제 object list source와 literal container는 search option callback보다 먼저 캡처합니다. 첫 library-owned leaf option에서 raw filters/Header/location을 callback별로 다시 복사하며, retained option pointer나 raw input의 뒤늦은 수정이 요청·결과를 바꾸지 않습니다. Source drift·취소·fault가 발생하면 partial inventory는 증거로만 남기고 성공한 선택을 만들지 않습니다. 고정 method/path/query/Accept/조건/version과 bodyless framing은 기존 list guards가 지킵니다. Native pointer401의 실제 `ErrOriginal`·`ErrReauth` 및 response bytes/Header/status를 그대로 유지하고 read/Close를 다시 실행하거나 acknowledged 요청을 replay하지 않습니다.

Go typed name/container와 raw JSON 필터는 Python의 동적 인자와 다릅니다. malformed JSON 필터는 직접 대응하는 Python 객체가 없어 Go 오류로 남깁니다. JMESPath deprecated 경고는 출력하지 않습니다. Object metadata는 instance별로 소유해 Source class-level mutable metadata 공유가 다른 행에 유출되는 문제를 피합니다. 기존 list의 같은 origin/container/filter 고정, page bound 및 더 강한 cycle detection은 유지하고 Source Requests Link `uri` key 문제는 listing의 문서화된 수리를 재사용합니다. Mutable Resource·dirty/session/cache SDK-R1/C1/S1 및 실제 인증 cloud acceptance는 별도 작업입니다.

[서비스 계약](cloud_object_search_test.go)과 [Connection 계약](../connection_swift_object_search_test.go)은 전체 페이지 수집, fnmatch/aliases, falsey·loaded numeric mapping, 순서별 오류, 임의 JMESPath JSON, 기본null/자동configured location과 remote Wire 분리, late constructor 오류, callback/source cancellation, 입력·결과 소유권을 검증합니다. testcloud의 실제 HTTP Content-Length 단절을 통해 후속 200·203·401·404·429·503 page에서 원문/body/Header/status와 native401 두 cause를 보존하고 fault retry/backoff/reauth를 실행하지 않는 것도 확인합니다. 기존 Object list의 Close/source/clean retry 계약을 함께 재사용합니다.
