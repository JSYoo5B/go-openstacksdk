# Cloud Flavor 목록·검색·조회: Python과 Go

`conn.Compute(ctx)`의 `AllFlavors`, `SearchFlavors`, `GetFlavor`는 Cloud의 세 함수에 대응합니다. SDK가 기존 Flavor reader·조건부 extra specs·Connection location과 Cloud 필터를 조합하므로 애플리케이션에 builder interface를 요구하지 않습니다. Proxy의 lazy `ListFlavors`와 이름 조회 `FindFlavor`는 [owned Flavor 가이드](flavor-records.md)의 API로 그대로 사용합니다.

| 고정 openstacksdk | Go | 실제 기본값과 순서 |
|---|---|---|
| `conn.list_flavors(get_extra=False)` | `service.AllFlavors(ctx)` | detail 전체 목록을 eager 수집; 기본 extra specs 보충 없음 |
| `conn.search_flavors(name_or_id=None, filters=None, get_extra=True)` | `service.SearchFlavors(ctx, pattern, options...)` | 전체 detail 목록과 조건부 extra specs를 먼저 완료한 뒤 이름·Cloud 필터 |
| `conn.get_flavor(name_or_id, filters=None, get_extra=True)` | `service.GetFlavor(ctx, identity, options...)` | falsey filters를 비운 뒤 기존 Find에 직접 위임; missing은 기본 nil |

고정 소스 `list_flavors`의 docstring에는 default True와 cloud 설정 override가 적혀 있지만 **실제 signature는 False이고 함수는 그 설정을 읽지 않습니다.** Search/Get은 실제 default True입니다. Source의 Search는 filter를 목록 query나 Body 필터로 먼저 보내지 않습니다. 따라서 검색에서 제외되는 flavor도 먼저 필요한 extra specs 조회를 수행합니다. Get의 deprecated filters는 Cloud search나 JMESPath 단건 선택 경로가 아닙니다.

```python
import openstack

conn = openstack.connect(compute_api_version="2.55")
all_flavors = conn.list_flavors()  # get_extra=False
matched = conn.search_flavors("m1.*", filters={"disk": 20})
found = conn.get_flavor("m1.small")  # get_extra=True, ignore_missing=True
for flavor in matched:
    print(flavor.to_dict())
if found is not None:
    print(found.to_dict())
```

## 독립 Go main

`clouds.yaml`의 cloud를 `-cloud`로 선택하고, `-pattern`의 기본값 `*`로 검색합니다. AllFlavors는 기본 extra=false, Search/Get은 기본 true를 사용하며 선택 버전2.55를 유지합니다. `-flavor ID_OR_NAME`을 추가하면 Get도 수행합니다. 출력은 raw Value, 실제 응답 Envelope와 inventory 수·각 record 증거를 구별합니다. 이 main의 검증 범위는 컴파일이며 실제 OpenStack 실행은 별도입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/compute"
    "github.com/JSYoo5B/gophercloudsdk/compute/v2/flavors"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    pattern := flag.String("pattern", "*", "flavor name or ID glob")
    identity := flag.String("flavor", "", "optional flavor ID or name to get")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *pattern, *identity); err != nil {
        log.Fatal(err)
    }
}

func recordValue(record *flavors.FlavorRecord) any {
    if record == nil { return nil }
    var enrichment any
    if child := record.Enrichment; child != nil {
        enrichment = map[string]any{
            "resource": child.Resource, "wire": child.Wire,
            "extra_specs": child.ExtraSpecs, "envelope": string(child.Envelope),
            "header": child.Header, "status_code": child.StatusCode,
        }
    }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope), "header": record.Header,
        "status_code": record.StatusCode, "enrichment": enrichment,
    }
}

func printQuery(operation string, result *compute.FlavorQueryResult) error {
    if result == nil { return nil }
    inventory := make([]any, len(result.Inventory))
    for i, record := range result.Inventory { inventory[i] = recordValue(record) }
    data, err := json.MarshalIndent(map[string]any{
        "operation": operation, "value": result.Value,
        "matched_count": len(result.Flavors), "inventory_count": len(result.Inventory),
        "inventory": inventory,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, pattern, identity string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    all, err := service.AllFlavors(ctx)
    if printErr := printQuery("all", all); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("all flavors: %w", err) }
    matched, err := service.SearchFlavors(ctx, pattern)
    if printErr := printQuery("search", matched); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("search flavors: %w", err) }
    if identity == "" { return nil }
    found, err := service.GetFlavor(ctx, identity)
    if found != nil {
        data, printErr := json.MarshalIndent(map[string]any{"operation": "get", "record": recordValue(found)}, "", "  ")
        if printErr != nil { return errors.Join(err, printErr) }
        fmt.Println(string(data))
    }
    if err != nil { return fmt.Errorf("get flavor: %w", err) }
    if found == nil { fmt.Println("flavor not found") }
    return nil
}
```

error와 result가 함께 있으면 예제는 부분 증거를 출력한 뒤 error를 반환합니다. Envelope는 accepted opaque/불완전 body도 나타낼 수 있어 JSON 값으로 재해석하지 않고 문자열로 출력합니다. 이 예제의 세 public 호출은 각각 별개의 logical call과 location/version 준비를 갖습니다.

## concrete 옵션

`FlavorQueryOpts`는 `GetExtra *bool`, `Filters *json.RawMessage`, `Microversion *string`을 갖습니다. nil GetExtra는 각 함수의 Source 기본값이고, `WithFlavorQueryExtraSpecs(false)`로 Search/Get의 보충을 끌 수 있습니다. `WithFlavorQueryOptions`는 pointer/raw JSON을 소유 복사하고 typed 옵션을 교체합니다. 개별 옵션은 적용 순서대로 반영하며 별도 Header는 유지합니다.

`WithFlavorQueryFilters(raw)`는 **nil이면 필터를 비우고**, `json.RawMessage("null")`는 명시 null을 보존합니다. `WithFlavorQueryExpression(expression)`은 JSON 문자열 필터를 만들며 Search에서 사용합니다. Filter bytes와 callback carrier는 SDK가 소유하고 한 logical call에 한 번 적용합니다. `WithFlavorQueryMicroversion`/`WithFlavorQueryHeader`는 기존 Flavor reader에 연결합니다. AllFlavors는 Source 목록 함수의 입력처럼 필터를 받지 않습니다. explicit null/object를 포함해 Filters가 present이면 HTTP 전에 오류이며 `WithFlavorQueryFilters(nil)`로 비운 경우는 허용합니다. 필터링은 SearchFlavors를 사용합니다.

| 함수 | 필터 의미 |
|---|---|
| `AllFlavors` | filters 없음; 완결 detail inventory의 eager 수집 |
| `SearchFlavors` | 이름/ID exact-or-glob 이후 dictionary subset 또는 JMESPath; 이 단계 전에 inventory·보충 완료 |
| `GetFlavor` | absent/null/falsey는 무필터 Find; truthy object는 member Find kwargs, truthy 비객체는 HTTP 전 입력 오류 |

Search dictionary는 [owned List](flavor-records.md)의7개 query/12개 Body classifier에 조기 전달하지 않습니다. Cloud의 declared Resource/location을 검사하므로 arbitrary 속성 오류와 ordered nested predicate는 공통 `cloudfilter.Select`의 계약을 사용합니다. 이름/ID는 Source의 문자열 표현·glob 단계를 사용하며 owned Find의 exact-string 비교와 구분합니다. JMESPath는 배열뿐 아니라 scalar/object/null도 반환할 수 있습니다.

Get은 Source의 `if not filters: filters={}`를 따라 false·0·빈 문자열·빈 배열·빈 객체도 무필터 경로로 처리합니다. **truthy 문자열은 JMESPath로 실행하지 않습니다.** Python의 `**filters` TypeError를 Go의 명시적 입력 오류로 표현합니다. truthy dictionary의 원래 속성 spelling은 member GET과 Resource seed에 전달하고, clean400/403/404 fallback에서는 기존 query alias·Body 필터를 사용합니다. Source상 duplicate keyword인 `name_or_id`, `ignore_missing`, `get_extra_specs`는 HTTP 전에 거부합니다. Get의 string-valued `limit`/marker 같은 query도 기존 Find의 typed 검증과 collision 정책을 유지합니다. dict-valued member query는 공유 encoder의 scalar/null/scalar-array 범위를 넘으므로 Go에서 입력 오류이며, List/Search의 dictionary subset과 구별합니다.

## 반환·부분 결과·재사용

All/Search는 `FlavorQueryResult{Value, Flavors, Inventory}`를 반환합니다. Value는 완료된 결과의 `json.RawMessage`이며 기본 목록은 빈 결과도 `[]`입니다. Inventory는 처리한 실제 record 순서를 유지합니다. ordinary Search의 Flavors는 Value와 연결된 행이며 JMESPath arbitrary JSON 결과에는 Flavor를 합성하지 않습니다. late page·추가 specs·Cloud filter 오류가 발생하면 partial Inventory를 보존하고 Value/Flavors를 완결 성공으로 채우지 않습니다. Get은 기존 `*flavors.FlavorRecord`를 직접 반환하고 missing은 nil, accepted 오류/추가 specs 오류는 record와 error가 함께 올 수 있습니다.

각 record는 Service의14필드 Resource와 실제 Wire/row 또는 member Envelope, 별도 Enrichment receipt를 유지합니다. Source의 이 세 함수는 별도 legacy flavor normalizer를 호출하지 않으며 기존 Proxy Resource를 반환합니다. SDK도 기존 projector와 Connection의 computed location을 재사용합니다. location은 옵션·discovery 전에 한 번 snapshot하고 Resource와 Enrichment.Resource에만 주입하며 원문 증거를 바꾸지 않습니다. Search가 local selection을 마칠 때까지 같은 source/context/binding guard를 유지합니다.

version nil의 selected 유지·미선택 finite discovery2.61·명시 empty의 versionless override, list의 page/cycle/origin·alias/default·필터 타입, Get의 clean fallback·missing·duplicate·partial response는 [owned Flavor](flavor-records.md)와 동일한 reader를 재사용합니다. Cloud eager API는 별도의 조회/pagination/normalization 엔진을 만들지 않습니다. native `Get`·`ListDetail`·`ListExtraSpecs`·typed collection은 그대로 유지합니다.

source 값과 Go 값의 bool/number·JSON precision·query bool spelling·비유한 float·controls·escaped route·dictionary-links 호환 및 full Python mutable Resource/session 차이는 기존 owned guide에 명시합니다. Cloud Search는 공통 Python-shaped Cloud filter 엔진을 사용하고, Get kwargs는 owned Find의 typed query 범위를 사용한다는 구분이 필요합니다. SDK가 Nova 권한이나 보이는 flavor 범위를 대신 판단하지 않습니다.

고정 소스는 [Cloud list_flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L278-L289), [search_flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L165-L181), [get_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L524-L561), [_filter_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)입니다. 테스트는 기존 public HTTP fixture와 공통 계약을 재사용하고 실제 Cloud 검증·예제 빌드·named 지원 판정의 결과는 [판정대장](../docs/sdk-support-ledger.md)에 따로 기록합니다.
