# Glance Metadef 객체·namespace 레코드 목록: Python과 Go

객체 scope의 `ListRecords`와 namespace API의 `ListRecords`는 고정 openstacksdk의 목록 처리에 맞춰 서버 query, declared Body 필터, 기본값과 descriptor 변환, 페이지 순회를 라이브러리가 담당합니다. 각 행에는 declared `Resource`와 실제 `Wire`, physical page의 응답 증거를 함께 제공합니다.

| Python `conn.image` | Go | 고정 GET 경로 | 목록 키 |
|---|---|---|---|
| `metadef_objects(namespace)` | `scope.ListRecords(ctx, options...)` | `/metadefs/namespaces/{namespace}/objects` | `objects` |
| `metadef_namespaces(**query)` | `api.ListRecords(ctx, options...)` | `/metadefs/namespaces` | `namespaces` |

`AllRecords`는 같은 순회를 수집하고 후속 오류에서도 앞서 성공한 행을 반환합니다. 객체의 기존 typed `List`·`All`은 query 없는 한 페이지·strict200, namespace의 typed `List`·`All`은 strict200·body `next`만 따르는 계약을 유지합니다. CRUD는 각각 [객체](v2/metadefobjects/README.md)·[namespace](v2/metadefnamespaces/README.md) 문서를 참고합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"

for value in conn.image.metadef_namespaces(
    visibility="private", owner="project-id", is_protected=False,
    limit=10, max_items=20, paginated=True
):
    print(value.to_dict())

for value in conn.image.metadef_objects(namespace):
    print(value.to_dict())
```

Namespace의 `visibility`, `resource_types`, `sort_key`, `sort_dir`, `limit`, `marker`는 서버 query입니다. `owner`·`is_protected` 같은 declared Body 속성은 응답을 받은 뒤 로컬에서 비교합니다. `protected`는 remote Body 이름이며 로컬 필터 key는 `is_protected`입니다. `any_tags` 같은 tag query는 이 class의 query mapping에 병합되지 않아 일반 unknown 속성처럼 무시됩니다.

**고정 public `metadef_objects(namespace)`에는 query 인자가 없습니다.** 객체의 query·Body 필터·호출별 헤더·limit·max_items·paginated 옵션은 Go에서 추가한 기능입니다. 이를 Python named proxy에 전달할 수 있는 kwargs로 설명하지 않습니다. 기본 객체 호출 자체도 inherited Resource generator의 기본 `paginated=True`를 사용합니다.

## 독립 Go main

아래 예제는 namespace 레코드를 iterator로 출력한 뒤 한 namespace의 객체를 `AllRecords`로 수집합니다. `-namespace-filter`·`-owner`는 namespace의 로컬 Body 필터, `-object-name`은 Go가 추가한 객체 로컬 필터입니다. `-limit`와 `-max-items`의0은 각각 explicit limit과 소비 cap을 생략하며 `-paginated` 기본값 true는 inherited 기본값과 같습니다. 인증은 `-cloud`로 선택한 clouds.yaml 설정을 사용합니다.

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
    namespaces "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
    objects "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal object parent")
    namespaceFilter := flag.String("namespace-filter", "", "optional namespace Body filter")
    owner := flag.String("owner", "", "optional namespace owner Body filter")
    objectName := flag.String("object-name", "", "optional object name Body filter")
    visibility := flag.String("visibility", "", "optional namespace server query")
    limit := flag.Int("limit", 0, "explicit server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows per list")
    paginated := flag.Bool("paginated", true, "follow list continuations")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    err := run(ctx, *cloud, *namespace, *namespaceFilter, *owner, *objectName,
        *visibility, *limit, *maxItems, *paginated)
    if err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            if printErr := printJSON(map[string]any{
                "error_status": response.StatusCode, "error_header": response.Header,
                "error_body": response.Body,
            }); printErr != nil { log.Print(printErr) }
        }
        log.Fatal(err)
    }
}

func printJSON(value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, namespaceFilter, owner,
    objectName, visibility string, limit, maxItems int, paginated bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.ImageV2(ctx)
    if err != nil { return err }
    namespaceOptions := []namespaces.RecordListOption{
        namespaces.WithRecordListOpts(namespaces.RecordListOpts{MaxItems: maxItems}),
        namespaces.WithRecordListHeader("X-Request-Source", "metadef-record-lists-example"),
        namespaces.WithRecordListPaginated(paginated),
    }
    if limit > 0 {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListLimit(limit))
    }
    if visibility != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListVisibility(visibility))
    }
    if namespaceFilter != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListFilter("namespace", namespaceFilter))
    }
    if owner != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListFilter("owner", owner))
    }
    for record, readErr := range service.MetadefNamespaces.ListRecords(ctx, namespaceOptions...) {
        if readErr != nil { return readErr }
        if err := printJSON(record); err != nil { return err }
    }
    scope, err := service.MetadefObjects.InNamespace(ctx, namespace)
    if err != nil { return err }
    objectOptions := []objects.RecordListOption{
        objects.WithRecordListOpts(objects.RecordListOpts{Limit: limit, MaxItems: maxItems}),
        objects.WithRecordListHeader("X-Request-Source", "metadef-record-lists-example"),
        objects.WithRecordListPaginated(paginated),
    }
    if objectName != "" {
        objectOptions = append(objectOptions, objects.WithRecordListFilter("name", objectName))
    }
    records, readErr := scope.AllRecords(ctx, objectOptions...)
    for _, record := range records {
        if err := printJSON(record); err != nil { return errors.Join(readErr, err) }
    }
    return readErr
}
```

`ImageV2`와 `Image(ctx).API`는 같은 shared provider와 concrete leaf를 사용합니다. iterator 생성과 `InNamespace`는 HTTP를 보내지 않습니다. scope는 literal parent를 검증하며 parent 조회·Find를 수행하지 않습니다. 응답의 namespace·name·self·schema를 다음 요청 경로로 사용하지 않습니다. 예제의 오류 Body는 `[]byte`이므로 JSON 출력에서는 base64입니다.

## 옵션과 필터

두 leaf는 `RecordListOpts`, concrete `RecordListOption`, `WithRecordListOpts`·`Header`·`Headers`·`Limit`·`Marker`·`MaxItems`·`Paginated`·`Filter`·`Filters`를 제공합니다. 서버 query 편의를 위해 `WithRecordListVisibility`·`ResourceTypes`·`SortKey`·`SortDir`도 있습니다. 객체는 이 네 문자열을 `RecordListOpts`에도 저장하고 namespace helper는 semantic filter를 추가합니다. `Filters`에는 공통 `resource.WithFilter`·`resource.WithFilters`만 넣습니다.

`WithRecordListOpts`는 전체 설정을 교체하고 개별 helper는 순서대로 추가·교체합니다. map·slice·pointer·JSON 값과 callback 설정은 독립 snapshot을 사용하며 옵션 callback은 iteration마다 한 번 실행합니다. nil/error callback, invalid 값, 보호된 header, reserved 제어 속성은 HTTP 전에 오류입니다. ordinary unknown semantic 속성은 무시하고 선택된 query·Body 필터만 encoding 오류를 검증합니다.

| 분류 | namespace | 객체 |
|---|---|---|
| 서버 query | `limit`, `marker`, `visibility`, `resource_types`, `sort_key`, `sort_dir` | 같은 여섯 key; Go extension |
| 로컬 declared Body | `id`, `name`, `created_at`, `description`, `display_name`, `is_protected`, `namespace`, `owner`, `resource_type_associations`, `updated_at`, `tags` | `id`, `name`, `created_at`, `updated_at`, `description`, `properties`, `required`; Go extension |
| 서버 query 우선 | `visibility`는 로컬 비교하지 않음 | 네 class query 속성은 로컬 비교하지 않음 |

Namespace의 `RecordListOpts.Limit`은 `*int`입니다. nil이면 생략하고 `WithRecordListLimit(0)`은 explicit `limit=0`을 보내며 음수를 거부합니다. semantic `WithRecordListFilter("limit", 0)`도 같은0을 보낼 수 있습니다. 객체의 Limit은 Go extension인 `int`이며0은 생략합니다. 위 공통 CLI는 양수일 때만 namespace limit helper를 추가해 `-limit=0`을 생략으로 취급합니다. 빈 Marker는 생략하고 음수 MaxItems를 거부합니다.

`Paginated=nil`은 true, false는 첫 페이지까지만 읽습니다. 양수 MaxItems는 **필터 적용 전 소비한 원본 row 수**를 제한하고 limit이 없거나 explicit0이면 첫 요청에 limit hint를 보냅니다. 양수 explicit limit은 cap보다 크더라도 유지합니다. 따라서 Body 필터가 모두 실패하면 cap에 도달한 뒤 빈 결과로 끝날 수 있습니다. Source의 arbitrary query 값·Python truthiness와 Go의 정수·문자열 옵션은 구분합니다.

로컬 object 필터는 recursive subset, scalar·array는 값 비교이며 missing은 null로 비교합니다. 실제 nonobject 값에 object 필터를 적용하면 mismatch입니다. Go는 bool과 number를 구분하고 숫자를 정확한 decimal로 비교합니다. Python의 `True == 1`, binary float rounding, 일부 nested 필터 예외와 차이가 있습니다. namespace의 `is_protected` 응답 truthiness 변환 뒤 비교하되 caller 필터 값을 bool로 자동 변환하지 않습니다.

## Resource·Wire·응답 증거

`Record.Resource`와 `Record.Wire`는 독립 `*resource.RawResource`이고 `Envelope`는 해당 physical page의 전체 JSON, `Header`와 `StatusCode`는 실제 응답입니다. 같은 페이지의 행끼리도 증거 bytes·map·header를 공유하지 않습니다. unknown·self·remote alias·명시 null·큰 숫자는 Wire에 그대로 남습니다. 객체 record의 `Namespace`는 고정 parent이고 namespace record에는 별도 parent 필드가 없습니다.

| declared Resource view | 필드 |
|---|---|
| 객체 9개 | `id`, `name`, `created_at`, `updated_at`, `description`, `properties`, `required`, `namespace_name`, `location` |
| namespace 13개 | `id`, `name`, `created_at`, `description`, `display_name`, `is_protected`, `namespace`, `owner`, `resource_type_associations`, `updated_at`, `visibility`, `tags`, `location` |

객체의 일곱 Body 필드는 untyped JSON이며 properties·required도 object/string-array로 강제 변환하지 않습니다. 누락값은 null이고 id 키가 있으면 null을 포함해 그 값을 유지하며 **키가 없을 때만 name으로 fallback**합니다. namespace도 id 키가 없을 때만 namespace를 alternate ID로 쓰며 inherited `name`은 별도 nullable 필드입니다. 날짜를 parse하거나 unknown key를 declared 속성으로 추가하지 않습니다.

Namespace의 `protected`는 declared `is_protected`로 옮기고 nonnull 값을 Python bool truthiness처럼 변환합니다. 두 spellings가 한 row에 있으면 파싱한 JSON dictionary의 insertion 순서에서 나중 key가 우선합니다. 동일 key를 반복하면 마지막 값을 쓰되 처음 insertion 위치를 유지합니다. 누락·명시 null은 null입니다. 숫자의 zero/nonzero를 정확한 decimal로 판정하므로 극단적인 소수가 Python float에서0으로 underflow하는 경우에는 차이가 있습니다. `resource_type_associations`는 scalar를 한 요소 list로 만들며 요소 중 nonobject는 `{}`로 변환합니다. `tags`는 누락이면 `[]`, 명시 null이면 null, nonnull scalar이면 한 요소 list이고 array의 각 값은 그대로 둡니다. 이 변환은 Wire를 바꾸지 않습니다.

객체의 namespace_name은 항상 고정 scope의 parent여서 응답 parent를 덮어쓰지 않습니다. Python 기본 `to_dict()`는 URI를 제외하지만 Go는 이를 provenance로 추가합니다. location은 Source collector의 Body·URI 소비 규칙을 따릅니다. 객체는 namespace URI가 항상 소비되어 옵션 전에 snapshot한 Connection 위치를 사용합니다. Namespace는 declared Body key가 하나라도 있으면 Connection 위치를 쓰고, 그 key가 전혀 없는 location-only/unknown-only row에서는 explicit raw location을 유지합니다. 어느 쪽도 response location을 다음 HTTP target으로 사용하지 않습니다. 직접 `New(client)`는 Connection dependency가 없어 recomputed location은 null입니다. `NewWithDependencies(client, Dependencies{CloudLocation: getter})`로 concrete getter를 공급할 수 있습니다. Go의 CloudLocation에 명시한 Zone은 유지하며 Source current_location의 기본 zone=None과 구분합니다.

## 페이지와 오류

actual HTTP200..399의 전체 physical body가 valid UTF-8이어야 하며 JSON 파싱은 필수입니다. root는 nonnull object이고 해당 plural key가 있어야 합니다. array 외 단일 row object도 Source처럼 받지만 consumed row는 nonnull object여야 합니다. 빈 body, invalid JSON, nonobject root/row, 잘못된 envelope shape를 빈 성공으로 바꾸지 않습니다. caller break·cap 이후 미소비 행은 row projection하지 않지만 전체 page UTF-8·JSON 검증은 이미 수행합니다.

Body의 `links`·plural_links·`next`와 HTTP Link에서 continuation을 찾습니다. `links:{next:"URL"}`는 Go 확장이며 Source의 dictionary 순회는 rel/href가 함께 있는 항목을 찾습니다.  truthy limit(명시값 또는 max_items hint)의 marker fallback은 마지막 **소비한 row**의 id/alternate ID를 사용합니다. nonempty short page도 fallback하며 필터 실패한 마지막 행이 marker에 기여합니다. Namespace의 explicit limit0은 marker fallback을 만들지 않으며 광고된 link에 따라 이어질 수 있습니다. 빈 페이지, cap, `Paginated=false`, caller break는 후속 continuation을 평가하기 전에 종료합니다. fallback에 필요한 identity가 nonempty string이 아니면 오류입니다.

Go는 같은 origin·collection·초기 필터를 고정하고 conflicting links·query drift·반복 marker·cycle·다른 target을 다음 HTTP 전에 거부합니다. Glance의 정확한 `/v2/metadefs/namespaces` 또는 해당 namespace의 정확한 escaped 객체 collection 경로는 캡처한 service collection으로 대응시켜 reverse prefix를 유지합니다. 이는 고정 versioned alias이며 임의 `/vN` 경로나 다른 escaped alias를 허용하지 않습니다. Source의 자유로운 dynamic URL/query·marker를 그대로 재현하지 않습니다. body는 한 번 닫고 accepted handling 실패를 replay하지 않습니다. configured native pre-body retry·reauth·backoff와 현재 provider의 live token을 사용하고 source/context/상위 operation guard의 관찰된 실패는 유지합니다.

accepted read·Close·UTF-8·JSON·shape·descriptor·context·source·continuation 오류는 실제 receipt가 있으면 `*resource.ResponseError`에 Body·Header·StatusCode와 원인을 보존합니다. rejected status는 native HTTP 오류이며 HTTP 전 오류에는 receipt가 없습니다. 이미 성공한 iterator 행과 `AllRecords`의 부분 slice는 caller에게 남습니다.

## 비교 범위와 정책

기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. [두 public proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1449-L1613), [객체 class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_object.py), [namespace class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py), [inherited list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2436)를 비교합니다.

Namespace public **query는 `_list`·`Resource.list`의 `base_path`, `headers`, `microversion`, `allow_unknown_params`, `__conflicting_attrs`, deprecated `jmespath_filters`까지 연결할 수 있습니다. Go에서는 session·target·version·auth 헤더를 고정하며 이 key를 일반 semantic 필터로 넘기면 명시 오류입니다. 지원하는 호출 제어는 concrete 옵션으로 지정합니다. Source constructor에서는 응답 row의 `connection`·`microversion`·`_synchronized`가 중복 keyword로 실패할 수 있습니다. 두 레코드 API도 이 세 key를 소비할 때 실제 page receipt와 함께 오류를 반환합니다. `self`는 Source가 constructor 전달 전에 제거하고 Go도 declared 속성으로 쓰지 않으며 Wire에 보존합니다. row의 `base_path`·`session`·`resource_type`·`__conflicting_attrs`는 unknown Wire 증거로 남고 route를 바꾸지 않습니다. 응답 row 처리와 호출 옵션에서 제어 key를 거부하는 정책은 별개입니다.

Deprecated JMESPath는 Source `_list`가 **수집 전 generator 자체**에 search를 적용합니다. 선택한 [jmespath.py1.0.1 interpreter](https://github.com/jmespath/jmespath.py/blob/1.0.1/jmespath/visitor.py)의 list projection/filter/index는 이 generator에서 null을 반환해 HTTP가 실행되지 않고, identity `@`는 generator를 유지하며 constant 식은 별도 JSON을 반환할 수 있습니다. 이를 Go 레코드 목록의 row 필터로 바꾸지 않으며 이 heterogeneous 반환 분기는 typed `ListRecords`의 지원 범위 밖입니다. jmespath.py1.0.1은 비교를 위해 선택한 dependency reference로서 SDK의 추가 pin이 아닙니다.

두 목록은 [Glance 기본 정책에 따른 핵심 user 작업](../docs/glance-policy-priorities.md)입니다. 실제 접근은 배포 policy·visibility·ownership에 따라 서버가 결정하며 SDK가 role을 미리 판정하지 않습니다. mapping과 Python 전체 mutable Resource/cache/dirty/session/동적 반환·URL의 동등성은 구분합니다. 실제 cloud 권한·DB·side effect의 검증을 주장하지 않습니다. 실제 테스트·예제 빌드·연산별 판정은 검증 후 [판정대장](../docs/sdk-support-ledger.md)에 기록합니다.
