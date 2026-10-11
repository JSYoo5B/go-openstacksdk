# Swift container 목록: openstacksdk와 Go

`Containers.ListRecords`는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 named `conn.object_store.containers(**query)`를 제공합니다. 목록의 SDK 기본값·쿼리 분류·descriptor·pagination을 소유하며, 기존 Gophercloud native `List`와 별도의 callable입니다. 이 문서는 catalog `python:object_store/v1/containers` 한 행을 다룹니다.

Python의 독립 실행 예제입니다. 기존 OpenStack 인증 환경을 사용하고 계정 목록만 조회합니다.

```python
import argparse
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prefix", default="assets/")
    parser.add_argument("--page-size", type=int)
    parser.add_argument("--max-items", type=int, default=0)
    args = parser.parse_args()
    query = {"prefix": args.prefix, "max_items": args.max_items}
    if args.page_size is not None:
        query["limit"] = args.page_size
    conn = openstack.connect()
    for container in conn.object_store.containers(**query):
        print(container.name, container.count, container.bytes)


if __name__ == "__main__":
    main()
```

동일한 Go 예제는 아래 전체를 `main.go`에 저장해 실행할 수 있습니다. Go 기본 page size 0은 명시한 `limit=0`이며, Source에서는 falsey여서 limit 자체의 후속 페이지 fallback을 만들지 않습니다. 실제 account count나 next link가 있으면 계속 조회합니다. page size 양수는 짧은 페이지에도 marker fallback을 허용합니다.

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func main() {
    prefix := flag.String("prefix", "assets/", "container name prefix")
    pageSize := flag.Int("page-size", 0, "server page size; zero disables limit fallback")
    maxItems := flag.Int("max-items", 0, "raw rows to consume; zero means no row cap")
    flag.Parse()
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }
    options := []containers.RecordListOption{
        containers.WithRecordListPrefix(*prefix),
        containers.WithRecordListLimit(*pageSize),
        containers.WithRecordListMaxItems(*maxItems),
    }
    for record, err := range swift.Containers.ListRecords(ctx, options...) {
        if err != nil {
            if record != nil {
                log.Printf("actual page status=%d body=%s", record.StatusCode, record.Envelope)
            }
            log.Fatal(err)
        }
        fmt.Printf("name=%s count=%s bytes=%s HTTP=%d\n",
            record.Resource.Body["name"], record.Resource.Body["count"],
            record.Resource.Body["bytes"], record.StatusCode)
    }
}
```

| Python | Go | 동작 |
|---|---|---|
| `containers()` | `ListRecords(ctx)` | lazy account-root GET, `Accept: application/json`(service-level Accept보다 우선), 기본 `format` 쿼리 없음 |
| `prefix`, `format`, `limit`, `marker` | `WithRecordListPrefix/Format/Limit/Marker` | server query; pointer가 없는 기본값과 명시한 0·빈 문자열 구분 |
| `name=...`, `id=...`, `count=...`, `bytes=...` | `WithRecordListQuery(key, value)` | 각 row의 descriptor 변환 후 로컬 필터 |
| `headers={...}` | `WithRecordListHeaders` | 기본 Accept overlay, SDK 소유 인증·framing·version 제외 |
| `max_items=...` | `WithRecordListMaxItems` | 필터 제외 행을 포함한 raw cap; falsey limit에는 최초 요청 hint 추가 |
| `microversion=...` | `WithRecordListMicroversion` | 해당 호출의 private client만 선택; 공용 client 변경 없음 |
| `end_marker`, `full_listing` | 같은 key의 `WithRecordListQuery` | pinned named Source의 unknown query로 무시 |
| `paginated=...` | raw query에서 사전 오류 | named Source가 이미 `paginated=True`를 전달하므로 Python도 중복 인자 오류 |
| Go의 단일 페이지·안전 한도 | `WithRecordListSinglePage`, `WithRecordListMaxPages` | 명시적 Go 제어; 기본 최대 10,000페이지 |

`end_marker`는 Swift REST API/native transport에서 사용할 수 있어도 pinned Container의 `_query_mapping`에는 없습니다. `full_listing=False`도 이 named 함수의 pagination을 끄지 않습니다. 목록을 중간에 멈추려면 iterator에서 `break`하거나 raw cap을 사용합니다. `format="plain"`·`"xml"`을 지정해 서버가 JSON이 아닌 응답을 보내면 pinned Source처럼 실제 page의 decode 오류를 반환합니다. SDK가 `format=json`을 임의로 추가하지 않습니다.

`WithRecordListQuery`의 server/local 값은 JSON-domain Python Requests 규칙을 사용합니다. null은 쿼리에서 생략하고 bool은 `True`·`False`, 배열은 null을 제외한 반복 key, 객체는 key 목록, 큰 정수는 원래 정확한 decimal을 보냅니다. typed `WithRecordListLimit`은 음수를 사전 거부하지만 raw `WithRecordListQuery("limit", -1)`은 Source처럼 실제 서버 응답에 맡깁니다. fallback은 초기 limit의 Python truthiness를 따르므로 숫자 0·false와 문자열 `"0"`·`"False"`는 다릅니다. unknown query는 encoding 전에 무시하며, unknown value의 custom marshaler도 실행하지 않습니다.

로컬 필터는 number/bool 동치와 재귀 dictionary predicate를 공유합니다. JSON 수치 비교는 공통 SDK의 정확한 decimal 정책을 적용합니다. Python JSON float의 binary64 반올림과 완전히 같지 않으며, 예를 들어 실제 정수 9007199254740992는 raw 필터 `9007199254740993.0`에 일치하지 않습니다(Python에서는 해당 float가 반올림되어 일치할 수 있습니다). 이는 기존 SDK-C1의 명시적 수치 정책이며 raw evidence는 변하지 않습니다. 빈 dictionary는 truthy scalar에도 일치하지만, nonempty dictionary가 truthy scalar에서 `.get()`을 필요로 하면 실제 page 증거가 있는 오류입니다. `WithRecordListQuery` 호출 순서대로 먼저 실패한 predicate에서 단락 평가합니다. `RecordListOpts.Query`의 Go map에는 입력 순서가 없으므로 해당 map의 나머지 predicate는 key를 정렬합니다. nested predicate의 정확한 순서는 `json.RawMessage`로 보존할 수 있습니다. `object_count`·Header attribute는 Body local filter가 아니므로 query로 전달해도 ignored입니다.

각 행은 `ContainerRecord`를 독립 소유합니다.

- `Resource.Body`: inherited id, name/count/bytes, 16 Header descriptor, location을 포함한 21개 declared field와 null defaults입니다. count·bytes와 integer Header는 Source int descriptor 변환, bool Header는 Python truthiness를 사용합니다. int descriptor는 bool을 Python int subclass로 유지하고, 일반 비숫자 문자열·컨테이너를 0으로 해석합니다. digit-only 문자열 `"²"`처럼 int로 바꿀 수 없는 Unicode digit은 실제 page 오류입니다. `count`↔`object_count`, `bytes`↔`bytes_used`, name/id alias는 해당 field가 없을 때만 fallback합니다. explicit null·false·0·빈 값은 그대로 구분합니다.
- `SeedBody`·`SeedHeaders`: row에 실제로 존재한 declared attribute의 raw snapshot입니다. `id`는 `Container.new`와 달리 list의 `existing`에서 유지하며, literal id presence가 alternate name보다 우선합니다.
- `Wire.Body`: unknown field·원래 JSON type까지 포함한 실제 row입니다. `Envelope`는 실제 account 응답 전체, `Header`·`StatusCode`는 실제 해당 page입니다. carrier 사이에 bytes/map/header를 공유하지 않습니다.
- `Metadata`: row의 `metadata` constructor 값만 instance별로 보유하며, custom prefix와 JSON-domain Python `str()`를 적용한 `map[string]string`입니다. 원래 값은 `Wire.Body`에 있습니다. Python의 class-level mutable metadata 공유와 달리 다른 행으로 유출하지 않습니다.

Account GET의 `X-Container-*`, Content-Type, custom metadata Header를 개별 container Header descriptor로 합성하지 않습니다. pinned `Resource.list`는 row로 `existing`을 만들고 BaseResource의 response translation을 호출하지 않습니다. row 안에 `object_count`·`x-container-object-count` 같은 선언된 Header 값이 있으면 그 값만 projection합니다. 개별 container의 현재 ACL/metadata가 필요하면 별도 metadata API로 조회합니다. 목록은 자동 HEAD를 하지 않습니다.

Pagination은 body next link, RFC Link, `X-Account-Container-Count`, truthy limit fallback 순으로 처리합니다. root array뿐 아니라 Source가 허용하는 singleton object도 읽습니다. 계정 count는 필터에 남은 행이 아닌 소비한 raw 행과 비교하고, 실제 마지막 row의 id/name을 다음 marker로 고정합니다. 소비자가 반환된 Resource를 바꾸어도 다음 marker를 바꾸지 않습니다. empty array는 malformed count·next link를 평가하기 전에 끝나며, raw cap·단일 페이지·consumer break는 미소비 row나 continuation을 읽지 않습니다.

Go는 next link의 origin·escaped account path와 초기 server filter·limit을 고정하고, 생략한 쿼리를 유지합니다. 다른 account/path, auth 정보·fragment, 변경된 filter/limit, 반복 page/marker는 실제 응답 증거와 함께 거부합니다. pinned Source는 일부 version-prefixed body URL을 잘라 연결하고, Requests의 일반 Link parser 결과에 없는 `uri` key를 읽는 결함이 있습니다. Go는 유효한 RFC Link를 직접 지원하고 계정 범위를 검증하는 명시적 개선을 제공합니다. Python의 임의 URL/Adapter traversal 전체를 재현하지 않습니다.

성공 status는 pinned generic `raise_from_response`의 유효 HTTP100..399입니다. 최종 1xx 노출은 transport 동작에 따라 다릅니다. accepted 204라도 빈 목록 body가 JSON이 아니면 actual decode 오류입니다. 400..599 목록 오류에는 ignore-missing을 적용하지 않습니다. 공통 REST가 거부 응답을 native 오류로 돌려주면 그 오류에 담긴 실제 status/body/header를 page receipt로 복사합니다. 본문을 다시 읽거나 Close하지 않으며 성공 페이지로 해석하지 않습니다. body Read/Close, 취소 cause, sticky source 변경, callback 실패는 원래 method/URL/status/body/header와 native HTTP 원인을 유지하고 Close는 한 번만 실행합니다. clean rejection의 native retry는 유지하지만 성공 응답 처리·physical/source 오류는 replay하지 않습니다.

Go의 요청 ownership은 native retry에서도 Authorization·X-Service-Token·Host/Cookie·framing/connection 및 foreign Nova/Cinder/Manila/Ironic version header의 부재, 선택한 Swift version 또는 부재, operation Accept, 네 가지 conditional Header(If-Match/If-None-Match/If-Modified-Since/If-Unmodified-Since)의 선택값 또는 부재와 bodyless GET을 고정합니다. retry에서 이를 변경하거나 선택된 version을 OmitHeaders에 넣으면 실제 첫 응답과 callback 원인을 보존하고 replay하지 않습니다. ordinary retry header와 provider의 live X-Auth-Token/reauth는 계속 사용할 수 있습니다. option factory와 callback 결과는 nested JSON/header/pointer를 복사하며 iterator는 최초 소비 시 source를 고정합니다.

`headers`·`max_items`·`microversion` 같은 method control은 전용 concrete options로 사용합니다. 임의 `base_path`, JMESPath, mutable Resource cache/dirty-state/Connection/location synthesis와 전체 Proxy/Adapter/session lifecycle은 기존 공통 SDK-R1/C1/S1 범위에 남습니다. 여기서 기존 native List나 생성된 transport wrapper를 다시 완료 API로 집계하지 않습니다. authenticated Swift ACL/quota/replication과 실제 cloud acceptance는 fixture 결과와 별도입니다.

[계약 테스트](python_list_contracts_test.go)는 기존 Container transport/fault-body와 public Gophercloud `testhelper` 기반 `internal/testcloud`를 재사용합니다. 실제 HTTP escaping/account-root pagination, raw receipt/default/query/row projection, Source predicate 순서·cls/connection/microversion/_synchronized constructor 충돌, cap·count·Link, 물리 오류·retry ownership, 동시 option 재사용을 포함합니다. 집중 race와 standalone Go 예제의 compile 검증은 루트 통합 단계에서 기록합니다.
