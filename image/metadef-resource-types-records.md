# Glance Metadef resource type 목록: Python과 Go

`image.Service.API.MetadefResourceTypes.ListRecords`는 global resource type을, `InNamespace`로 만든 scope의 `ListRecords`는 해당 namespace의 association을 순회합니다. 두 API는 openstacksdk 목록 호출의 서버 query·로컬 Body 필터·페이지 처리를 SDK가 담당하고, 각 결과에 declared `Resource`와 실제 `Wire`·응답 receipt를 함께 제공합니다.

| Python `conn.image` | Go | 고정 GET 경로 | 응답의 목록 키 |
|---|---|---|---|
| `metadef_resource_types(**query)` | `api.ListRecords(ctx, options...)` | `/metadefs/resource_types` | `resource_types` |
| `metadef_resource_type_associations(namespace, **query)` | `scope.ListRecords(ctx, options...)` | `/metadefs/namespaces/{namespace}/resource_types` | `resource_type_associations` |

`AllRecords`는 같은 순회를 slice로 모읍니다. 기존 typed `List`·`All`은 query 없는 한 페이지·strict200 동작을 유지합니다. Association의 생성·삭제와 그 typed 목록은 [기존 leaf 가이드](v2/metadefresourcetypes/README.md)에 설명합니다. 새 레코드 목록을 위해 다른 서버 경로를 추가하지 않습니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"

for value in conn.image.metadef_resource_types(
    limit=10, max_items=20, paginated=False
):
    print(value.to_dict())

for value in conn.image.metadef_resource_type_associations(
    namespace, limit=10, max_items=20, name="OS::Nova::Server", paginated=False
):
    print(value.to_dict())
```

`limit`·`marker`는 inherited query mapping을 통해 서버에 보냅니다. `name` 같은 declared Body 속성은 응답을 받은 뒤 로컬에서 필터링합니다. `max_items`는 **필터를 통과한 결과 수가 아니라 소비한 원본 row 수**를 제한합니다. 따라서 위 association 호출이 최대20개를 반환한다는 의미는 아니며, 처음20개가 모두 필터에 맞지 않으면 결과는 비어 있습니다.

예제의 `paginated=False`는 첫 페이지까지만 읽습니다. `max_items`가 양수이고 explicit limit이 없으면 첫 요청에 max_items를 limit hint로 보냅니다. Glance의 고정 resource-type controller는 limit·marker를 읽지 않고 기본 목록에 next를 만들지 않지만, Python의 inherited list 동작에는 continuation 처리가 있습니다. explicit limit을 사용하면 short nonempty page에서도 marker fallback으로 추가 요청할 수 있습니다.

## 독립 Go main

아래 예제는 global 목록을 iterator로 출력한 뒤 namespace association을 `AllRecords`로 수집합니다. `-name`은 association에만 적용하는 로컬 필터입니다. `-limit`와 `-max-items`의0은 각각 explicit limit과 소비 cap을 생략합니다. 고정 Glance의 finite 목록에 맞춰 `-paginated`는 기본 false이며, continuation을 이용하려면 true로 설정합니다. OpenStack의 인증 정보는 `-cloud`로 선택한 clouds.yaml 설정을 사용합니다.

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

    "github.com/JSYoo5B/go-openstacksdk"
    types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    name := flag.String("name", "", "optional association name filter")
    limit := flag.Int("limit", 0, "explicit server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows per list")
    paginated := flag.Bool("paginated", false, "follow list continuations")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *name, *limit, *maxItems, *paginated); err != nil {
        log.Fatal(err)
    }
}

func printRecord(record *types.Record) error {
    if record == nil { return nil }
    data, err := json.MarshalIndent(map[string]any{
        "namespace": record.Namespace,
        "resource": record.Resource,
        "wire": record.Wire,
        "envelope": record.Envelope,
        "header": record.Header,
        "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, name string, limit, maxItems int, paginated bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    api := service.API.MetadefResourceTypes
    options := []types.RecordListOption{
        types.WithRecordListOpts(types.RecordListOpts{Limit: limit, MaxItems: maxItems}),
        types.WithRecordListHeader("X-Request-Source", "metadef-records-example"),
        types.WithRecordListPaginated(paginated),
    }
    for record, readErr := range api.ListRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := printRecord(record); err != nil { return err }
    }
    scope, err := api.InNamespace(ctx, namespace)
    if err != nil { return err }
    scopedOptions := []types.RecordListOption{
        types.WithRecordListOpts(types.RecordListOpts{
            Limit: limit, MaxItems: maxItems,
            Filters: associationFilters(name),
        }),
        types.WithRecordListHeader("X-Request-Source", "metadef-records-example"),
        types.WithRecordListPaginated(paginated),
    }
    records, readErr := scope.AllRecords(ctx, scopedOptions...)
    for _, record := range records {
        if err := printRecord(record); err != nil { return errors.Join(readErr, err) }
    }
    return readErr
}

func associationFilters(name string) []resource.ListOption {
    if name == "" { return nil }
    return []resource.ListOption{resource.WithFilter("name", name)}
}
```

`AllRecords`는 뒤 페이지나 row에서 실패해도 이미 반환한 record를 오류와 함께 남깁니다. 예제는 이를 출력한 뒤 오류를 반환합니다. 빈 성공은 nonnil empty slice입니다. iterator에서 `break`하면 미소비 row를 decode하거나 다음 페이지를 요청하지 않습니다. 예제의 검증 범위는 컴파일과 로컬 HTTP 계약이며 인증된 cloud·Python 실행을 증명하지는 않습니다.

## 옵션과 필터

`RecordListOpts`에는 `Headers`, `Limit`, `Marker`, `MaxItems`, `Paginated *bool`, `Filters []resource.ListOption`이 있습니다. `WithRecordListOpts`는 전체 설정을 교체하고 개별 `WithRecordListHeader(s)`·`WithRecordListLimit`·`WithRecordListMarker`·`WithRecordListMaxItems`·`WithRecordListPaginated`·`WithRecordListFilter(s)`는 설정을 추가하거나 교체합니다. 같은 concrete 설정이나 같은 semantic 필터 안에서는 뒤 옵션이 우선하며 map·slice·pointer·callback 설정은 호출별로 snapshot합니다. lazy iterator의 옵션 callback은 실제 순회 시작 때 한 번 적용됩니다.

`Paginated=nil`은 true, false는 첫 페이지까지만 읽습니다. `Limit=0`은 생략, 양수는 한 개의 positive limit, `MaxItems=0`은 cap 없음입니다. 음수와 잘못된 marker·옵션은 HTTP 전에 오류입니다. `Marker=""`는 생략하며, semantic marker를 설정한다면 한 개의 nonempty literal query 값이어야 합니다. typed Limit/Marker와 semantic 필터의 같은 query를 함께 지정하면 순서와 무관하게 충돌 오류입니다. 보호된 인증·version·representation·framing header는 SDK가 소유하며 임의 base_path·namespace_name·microversion·session 같은 제어값을 필터로 전달할 수 없습니다.

| semantic filter | 처리 |
|---|---|
| `limit`, `marker` | 서버 query |
| `id`, `name`, `created_at`, `updated_at` | 두 목록의 로컬 Body 필터 |
| `prefix`, `properties_target` | association의 로컬 Body 필터 |
| 선언되지 않은 일반 속성 | query로 보내거나 Wire를 검색하지 않고 무시 |
| 경로·parent·session·version 제어 속성 | 오류 |

`Filters`에는 공통 `resource.WithFilter`·`resource.WithFilters`를 사용합니다. 공통 helper는 값을 JSON으로 snapshot합니다. 알 수 없는 일반 속성에 대해 capture한 encoding 오류는 선택된 query·Body 필터의 검증에 반영하지 않습니다. arbitrary query·별도의 Body filter·Find·Name 검색 옵션을 이 carrier로 추가하지 않습니다.

로컬 object 필터는 nonempty 실제 object에 대해 recursive subset을 비교하고, scalar·array는 값으로 비교합니다. missing 필드는 null로 비교합니다. Go는 JSON bool과 number를 구분하고 숫자를 정확한 decimal 값으로 비교하므로 Python의 `True == 1`이나 binary float rounding과 차이가 있습니다. object 필터에 실제 nonobject 값이 들어오면 mismatch이며 Python의 일부 nested 값 처리에서 발생할 수 있는 예외를 재현하지 않습니다.

## 결과와 location

`Record`의 `Resource`·`Wire`는 각각 독립 `*resource.RawResource`입니다. `Envelope`는 해당 physical page의 전체 JSON이고 `Header`·`StatusCode`는 실제 응답입니다. 같은 페이지의 여러 record는 같은 페이지 내용을 각각 소유하며 한 record나 한 채널의 수정이 다른 값에 전파되지 않습니다.

| Resource 필드 | global | association |
|---|---|---|
| `id`, `name`, `created_at`, `updated_at`, `location` | 포함 | 포함 |
| `prefix`, `properties_target`, `namespace_name` | 제외 | 포함 |

누락한 declared 값은 null입니다. id는 응답에 키가 있으면 null을 포함해 그 값을 유지하고, **id 키가 없을 때만 name을 alternate ID로 사용**합니다. name·date·prefix·target은 Source의 untyped descriptor처럼 원래 JSON 값을 유지하며 string coercion이나 date parsing을 하지 않습니다. `self`·unknown field는 Wire에만 남습니다. raw 큰 숫자와 명시 null도 보존합니다.

Association의 `namespace_name`과 `Record.Namespace`는 고정 scope의 parent입니다. Wire의 namespace_name은 실제 응답에 남지만 declared parent를 바꾸지 않습니다. global record의 Namespace는 nil입니다. Python 기본 `to_dict()`는 URI 속성을 제외하지만 Go view는 `namespace_name`을 요청 provenance로 추가합니다. namespace scope는 HTTP 없이 만들고 literal parent를 검증하므로 parent 조회나 이름 검색을 수행하지 않습니다.

location은 고정 Python의 collector 규칙을 따릅니다. **Association은 항상 Connection location을 사용**하며, global resource type은 응답의 declared Body 키 유무에 따라 달라집니다. 응답의 location은 어느 경우에도 Wire에 그대로 남습니다.

| row 조건 | Resource의 location |
|---|---|
| association | snapshot한 Connection location |
| global에 `id`·`name`·`created_at`·`updated_at` 중 하나라도 있음 | snapshot한 Connection location |
| global에 위 네 키가 모두 없고 explicit `location`이 있음 | 그 raw location, null도 유지 |
| global에 위 네 키와 location이 모두 없음 | snapshot한 Connection location |

`id:null`·`name:null`도 declared 키가 있는 경우입니다. 고정 [Resource collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L805-L844)는 Body·Header·URI 속성을 수집하면 남은 computed 속성을 다시 계산합니다. association에는 fixed namespace URI가 항상 있어 응답 location이 declared Resource에 남지 않습니다.

Connection location은 옵션·HTTP 전에 한 번 snapshot합니다. `conn.Image(ctx).API.MetadefResourceTypes`와 `conn.ImageV2(ctx).MetadefResourceTypes` 두 진입 경로가 모두 CurrentLocation dependency를 연결합니다. 직접 생성한 `types.New(client)`는 Connection location 대신 null을 사용하며 `types.NewWithDependencies(client, types.Dependencies{CloudLocation: getter})`로 concrete dependency를 공급할 수 있습니다. Go의 CloudLocation에 명시한 Zone은 유지되지만 고정 Python의 current_location은 zone=None을 사용합니다.

## 페이지 처리와 오류

레코드 목록은 actual HTTP200..399를 받은 뒤 **전체 physical page의 UTF-8을 검증하고 mandatory JSON을 파싱**합니다. unknown envelope 값이나 미소비 row의 invalid UTF-8도 오류입니다. root는 object여야 하고 해당 plural key가 있어야 하며, 목록은 array 또는 Source처럼 단일 row object를 수용합니다. consumed row는 valid UTF-8의 nonnull object여야 합니다. 빈 body·invalid JSON·parsed nonobject·잘못된 plural shape를 defaults나 빈 성공으로 바꾸지 않습니다. 기존 typed 목록은 actual200과 nonnull array만 수용합니다.

페이지 처리는 Body의 links·plural_links·next와 HTTP Link를 사용하고, explicit limit의 marker fallback은 마지막 소비 row의 id/name을 사용합니다. cap에 도달하거나 첫 페이지 전용 옵션을 사용하면 continuation을 평가하기 전에 멈춥니다. 빈 페이지에서도 멈춥니다. fallback에 쓸 identity가 없거나 적합한 string이 아니면 오류이며 passive JSON identity를 강제로 문자열로 만들지 않습니다.

Go의 공통 paging guard는 같은 origin·collection 경로와 initial filter를 고정하고, 충돌하는 link·filter 변경·marker 반복·cycle을 오류로 처리합니다. 고정 Python은 Body link에 우선순위를 주고 상대적으로 자유롭게 URL·query를 처리하므로 이 guard 정책까지 동일하다고 주장하지 않습니다. namespace나 service target이 관찰된 뒤 바뀌면 진행 중인 작업은 오류이며 다른 target으로 이어가지 않습니다.

accepted 응답의 read·Close·UTF-8·JSON·shape·context·source·continuation 오류는 실제 receipt가 있을 때 `*resource.ResponseError`에 Body·Header·StatusCode와 원인을 보존합니다. rejected status는 native Gophercloud HTTP 오류로 남습니다. HTTP 전에 발생한 오류에는 response receipt가 없습니다. 앞에서 성공한 record는 유지하며 오류 뒤 다른 namespace·schema 조회나 accepted body replay를 하지 않습니다.

## 비교와 검증 범위

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Glance `57f7dd9e76ef24e1e9013eceaa703bd442469a24`, Gophercloud `v2.15.0`입니다. 고정 소스의 [두 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1696-L1786), [Resource class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py), [inherited list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2459)가 기준입니다.

새 [레코드 목록 테스트](v2/metadefresourcetypes/list_records_test.go)와 [Connection 진입 경로 테스트](../connection_image_metadef_records_test.go)는 기존 [leaf core fixture](v2/metadefresourcetypes/core_test.go)·[외부 HTTP 계약](v2/metadefresourcetypes/contracts_test.go)·[공통 list control](../internal/rest/list_control_test.go)·[short page 처리](../internal/rest/list_short_page_test.go)·[로컬 JSON 필터](../internal/jsonfilter/filter_test.go)를 재사용합니다. 실제 통과 결과·예제 빌드·연산별 판정은 [판정대장](../docs/sdk-support-ledger.md)에 검증 후 기록합니다. 두 연산의 Go mapping과 Python 전체 Resource lifecycle·session·cache·동적 경로의 동등성은 별도 범위이며 실제 cloud의 권한·DB·side effect 검증을 주장하지 않습니다.
