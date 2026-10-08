# Glance import·기본 저장소 레코드: Python과 Go

`conn.ImageV2(ctx).ServiceInfo`의 `GetImportInfoRecord`, `ListStoreRecords`, `AllStoreRecords`는 import discovery와 기본 저장소 목록을 declared Resource로 제공합니다. SDK가 기본값·descriptor 변환·로컬 Body 필터·페이지 순회·현재 location을 처리하며 실제 Wire와 응답 증거를 별도로 보존합니다.

| Python `conn.image` | Go `serviceinfo.API` | 고정 GET 경로 | Resource 필드 수 |
|---|---|---|---:|
| `get_import_info()` | `GetImportInfoRecord(ctx, options...)` | `/info/import` | 4 |
| `stores(details=False, **query)` | `ListStoreRecords(ctx, options...)` | `/info/stores` | 6 |
| `list(stores(details=False, **query))` | `AllStoreRecords(ctx, options...)` | 같은 목록 순회를 수집 | 6 |

기존 typed `GetImportInfo`와 `ListStores/AllStores`의 strict200·schema·부분 결과 정책은 유지합니다. 이 owned 목록은 기본 stores 경로를 고정하며 `details=True`는 후속 핵심 admin 범위입니다. 기존 typed `WithListStoresDetails(true)`는 계속 사용할 수 있습니다. Python `stores`의 상세 분기가 남아 있으므로 기본 목록을 구현했다는 이유로 해당 API 전체 완료를 주장하지 않습니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
info = conn.image.get_import_info()
print(info.to_dict())

for store in conn.image.stores(details=False):
    print(store.to_dict())

for store in conn.image.stores(
    details=False, is_default=True, properties={"region": "east"},
    limit=10, max_items=20, paginated=True
):
    print(store.to_dict())
```

고정 [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2272-L2295)는 stores의 query를 generic Resource 목록에 전달합니다. `limit`·`marker`는 서버 query이고 canonical `id`, `name`, `description`, `is_default`, `properties`는 로컬 Body 조건입니다. remote spelling `default`와 ordinary unknown 조건은 로컬 필터가 아닙니다. `is_default` 조건은 descriptor 변환을 거친 값을 비교합니다.

## 독립 Go main

[외부 모듈 설치 안내](../../../docs/install.md)를 따라 아래 내용을 `main.go`로 저장해 빌드할 수 있습니다. 실제 실행은 `clouds.yaml`의 설정을 준비하고 `go run . -cloud dev`처럼 호출합니다. `-store`는 선택한 ID의 로컬 조건이고 `-limit=-1`은 이 예제 CLI의 생략 표기입니다. `-limit=0`은 library의 explicit zero limit을 지정합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    storeID := flag.String("store", "", "optional local store ID filter")
    limit := flag.Int("limit", -1, "server limit; -1 omits the option")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows before filtering")
    paginated := flag.Bool("paginated", true, "follow store list continuations")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *storeID, *limit, *maxItems, *paginated); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(view, wire *resource.RawResource, statusCode int) error {
    value := map[string]any{"resource": view, "wire": wire, "status_code": statusCode}
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func run(ctx context.Context, cloud, storeID string, limit, maxItems int, paginated bool) error {
    if limit < -1 { return fmt.Errorf("limit must be -1 or nonnegative") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.ImageV2(ctx)
    if err != nil { return err }
    info, err := service.ServiceInfo.GetImportInfoRecord(ctx,
        serviceinfo.WithImportRecordHeader("X-Request-Source", "serviceinfo-records-example"))
    if err != nil { return err }
    if err := printRecord(info.Resource, info.Wire, info.StatusCode); err != nil { return err }

    options := []serviceinfo.StoreRecordListOption{
        serviceinfo.WithStoreRecordListOpts(serviceinfo.StoreRecordListOpts{MaxItems: maxItems}),
        serviceinfo.WithStoreRecordListHeader("X-Request-Source", "serviceinfo-records-example"),
        serviceinfo.WithStoreRecordListPaginated(paginated),
    }
    if limit >= 0 {
        options = append(options, serviceinfo.WithStoreRecordListLimit(limit))
    }
    if storeID != "" {
        options = append(options, serviceinfo.WithStoreRecordListFilter("id", storeID))
    }
    for record, readErr := range service.ServiceInfo.ListStoreRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := printRecord(record.Resource, record.Wire, record.StatusCode); err != nil { return err }
    }
    return nil
}
```

이 예제는 declared 값과 실제 wire 값을 함께 출력합니다. Import GET이 빈/invalid JSON을 허용한 경우 Wire는 nil이고 Resource는 기본값을 가지며 실제 bytes는 Envelope에 남습니다. Envelope가 JSON이 아닐 수 있으므로 전체 record를 JSON으로 출력할 때는 해당 raw bytes를 별도로 처리합니다. 예제의1분 제한은 caller context의 선택이며 library가 추가한 기본 timeout이 아닙니다.

같은 API는 상위 `conn.Image(ctx)`의 `API.ServiceInfo`에서도 사용합니다. 직접 `serviceinfo.New(client)`는 Connection dependency 없이 사용할 수 있고 `NewWithDependencies(client, Dependencies{CloudLocation: getter})`는 현재 cloud 사실을 공급합니다. getter는 각 GET/순회에서 옵션 callback 전에 한 번 읽고 복사하므로 callback이나 후속 page가 변경한 값을 현재 operation에 섞지 않습니다.

## 옵션과 필터

`ImportRecordOpts`에는 일반 `Headers`만 있고 `WithImportRecordOpts/Header/Headers`를 제공합니다. ID·이름·query·입력 Resource를 요구하지 않는 고정 singleton GET입니다.

`StoreRecordListOpts`는 `Headers map[string]string`, `Limit *int`, `Marker string`, `MaxItems int`, `Paginated *bool`, `Filters map[string]json.RawMessage`를 제공합니다. `WithStoreRecordListOpts`는 전체 설정을 교체하고 `Header/Headers`, `Limit`, `Marker`, `MaxItems`, `Paginated`, `Filter/Filters` helper는 순서대로 값을 추가·교체합니다. `Filters`는 Body 조건 전용이며 서버 query는 typed Limit·Marker로 지정합니다.

- nil Limit과 빈 Marker는 생략합니다. `WithStoreRecordListLimit(0)`은 explicit `limit=0`을 보내며 음수는 오류입니다.
- nil Paginated는 true이고 false는 첫 page까지만 읽습니다. MaxItems0은 무제한입니다.
- 양수 MaxItems는 **필터 전 소비한 원본 행 수**를 제한합니다. limit이 없거나 explicit0이면 첫 요청의 limit을 cap hint로 설정하고 양수 explicit limit은 유지합니다.
- canonical `id`, `name`, `description`, `is_default`, `properties`만 로컬 조건으로 비교합니다. `default`·unknown은 encoding 전에 무시하며 route·details 선택을 만들지 않습니다.

입력 map·pointer·raw bytes와 옵션 slice를 복사하고 callback은 각 순회에서 한 번 적용합니다. nil/error callback, 음수 limit/cap, invalid marker/header·선택한 필터의 잘못된 JSON은 HTTP 전에 오류입니다. 보호된 auth·framing·version·representation header를 일반 Header 옵션으로 덮어쓸 수 없습니다.

필터는 descriptor 변환 뒤 비교합니다. nonempty 실제 object에는 recursive subset을 적용하고 scalar·array는 값 비교를 수행합니다. missing은 null로 비교합니다. empty 실제 object는 object 조건에 맞지 않습니다. Go는 bool과 number를 구분하고 숫자를 exact decimal로 비교하므로 Python의 `True == 1`, binary float rounding, 일부 nested 값의 예외와 차이가 있습니다. caller의 필터 값은 descriptor 타입으로 자동 변환하지 않습니다.

## Resource와 Wire

`ImportRecord`와 `StoreRecord`는 각각 `Resource`·`Wire *resource.RawResource`, `Envelope`, `Header`, `StatusCode`를 제공합니다. declared view·wire fields·전체 physical body·headers의 map와 raw bytes는 서로 독립 복사합니다. 같은 page의 행끼리도 증거를 공유하지 않으며 반환값 수정이 다음 요청 target을 바꾸지 않습니다.

| declared view | 필드 |
|---|---|
| Import4개 | `id`, `name`, `import_methods`, `location` |
| Store6개 | `id`, `name`, `description`, `is_default`, `properties`, `location` |

[Import/Store 선언](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L22-L44)의 inherited id/name과 description은 untyped입니다. 생략은 null이고 explicit null·빈 문자열·숫자·array·object를 그대로 보존합니다. Store에는 alternate ID가 없으므로 name을 id로 합성하지 않습니다. 날짜 속성을 추가하거나 id로 다른 endpoint를 자동 조회하지 않습니다.

Remote `import-methods`는 `import_methods`, `default`는 `is_default`로 옮깁니다. canonical/remote 두 spelling이 있으면 파싱한 JSON dictionary insertion 순서에서 나중 key가 우선합니다. 같은 key를 반복하면 마지막 값을 쓰되 첫 insertion 위치를 유지합니다. unknown·self·remote spelling·큰 숫자와 서버의 read-only·type·weight는 실제 Wire에 남고 declared 속성으로 추가하지 않습니다.

`import_methods`와 `properties`의 dict descriptor는 생략/null이면 null, nonnull object이면 nested 값을 보존하고 nonobject이면 `{}`로 변환합니다. Import는 description/type/value를 별도 typed 구조나 string-array로 강제하지 않습니다. `is_default`는 생략/null이면 null이며 nonnull 값은 Python bool truthiness로 변환합니다. 따라서 서버의 `"false"`도 nonempty string이므로 true입니다. wire value는 변환하지 않습니다. 이 규칙은 [고정 descriptor 구현](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L122)에 따른 것이며 Go의 숫자 truthiness는 exact decimal을 사용합니다.

Import는 fresh singleton Resource의 Connection location을 유지하고 response raw location을 overlay하지 않습니다. Store는 declared Body key가 하나라도 소비되면 captured current location을 쓰며, 그런 key가 없는 location-only/unknown-only 행에서는 explicit raw location을 보존합니다. raw location도 없으면 current location을 사용합니다. 이는 [Resource collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L808-L839)의 Body·computed 소비 규칙입니다. 직접 New에 dependency가 없으면 recomputed location은 null입니다. Go의 명시 Zone은 유지하며 Source current_location 기본 zone=None과 구분합니다.

## 응답·페이지·부분 결과

Import GET은 actual HTTP200..399를 받으면 Body 속성을 overlay합니다. 빈/invalid JSON은 [Source fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1356-L1396)처럼 기본 Resource와 실제 receipt를 반환하고 Wire는 nil입니다. valid JSON null·array·scalar나 invalid UTF-8은 처리 오류입니다. 응답 id/name/import-methods가 있으면 declared view로 처리하지만 그 값이나 schema·location으로 다른 URL을 요청하지 않습니다.

Store 목록은 actual HTTP200..399에서 valid UTF-8의 JSON nonnull object와 required `stores` key를 요구합니다. 배열 외 단일 행 object도 generic Source처럼 받습니다. consumed row는 nonnull object이고 raw `connection`, `microversion`, `_synchronized`는 [Source constructor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2305-L2323)의 중복 인자 오류에 대응해 거부합니다. caller break·cap 이후 미소비 행을 projection하지 않지만 전체 page의 JSON/UTF-8은 이미 검증합니다.

공통 목록은 Body links·stores_links·next와 HTTP Link를 처리합니다. top-level next의 null·false·숫자0·빈 string/array/object는 무시하고 다른 continuation을 검사합니다. truthy nonstring next는 처리 오류입니다. `links:{next:"URL"}`는 Go 확장입니다. 양수 초기 limit 또는 max-items hint가 있으면 마지막 소비 행의 id를 marker fallback으로 사용하며 short nonempty page도 이어갑니다. local 필터에 실패한 행도 raw cap·marker에 기여하고 후속 link에서 처음 받은 limit만으로 fallback을 새로 활성화하지 않습니다. explicit limit0은 fallback을 만들지 않지만 광고된 링크는 따를 수 있습니다.

확인한 실제 Glance basic controller는 limit·marker query를 읽지 않고 전체 저장소 집합을 반환합니다. 무옵션 목록은 보통 한 페이지로 끝납니다. 명시적인 양수 Limit의 generic Source marker fallback은 같은 저장소 목록을 다시 받을 수 있고, Go는 반복 marker·cycle을 terminal 오류로 막으며 그 전에 소비한 행은 iterator와 AllStoreRecords의 부분 결과에 남습니다. MaxItems hint도 서버의 실제 paging 지원을 보장하지 않습니다. 이 Source fallback과 달리 기존 typed ListStores는 광고된 링크만 따르고 marker나 limit hint를 합성하지 않습니다. 필요한 한 페이지 소비는 WithStoreRecordListPaginated(false)로 명시할 수 있습니다.

빈 page·cap·Paginated=false·caller break는 continuation 평가 전에 끝납니다. Source의 Python truthiness/arbitrary query 값과 Go의 정수·문자열 옵션은 구분합니다. Go는 origin·captured reverse prefix·고정 collection·초기 query를 유지하고 conflicting links·query drift·반복 marker·cycle·다른 target을 다음 HTTP 전에 거부합니다. 정확한 `/v2/info/stores` alias는 captured service prefix의 같은 collection으로 대응합니다. Source의 임의 base_path·session·microversion·dynamic URL을 모두 허용하지 않습니다.

`ListStoreRecords`는 lazy이고 재순회마다 새 capture·옵션 적용·요청을 수행합니다. `AllStoreRecords`는 nonnil empty slice로 시작하고 후속 오류에도 앞서 성공한 record를 함께 반환합니다. 이 수집 정책은 partial slice를 버리는 기존 typed `AllStores`와 다릅니다.

accepted read·Close·UTF-8·JSON·shape·descriptor·context·source·continuation 오류에는 실제 receipt가 있으면 `*resource.ResponseError`의 Body·Header·StatusCode와 원인이 남습니다. rejected 응답은 native HTTP 오류이며 요청 전 오류에는 receipt가 없습니다. rejected body I/O는 기존 Gophercloud 정책을 유지합니다. body는 한 번 닫고 accepted 처리 실패 후 replay하지 않습니다. configured native pre-body retry·reauth·backoff·live auth와 operation의 sticky source/ancestor guard를 사용합니다.

## 권한과 비교 범위

고정 Glance controller는 [Import41–51행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L41-L51)과 [기본 stores53–93행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L53-L93)에 별도 operation policy를 호출하지 않습니다. 이 기본 조회를 핵심 user에 배치하되 실제 인증·multi-backend 설정·policy override는 서버가 판단합니다.

상세 stores는 [controller165–193행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L165-L193)이 `stores_info_detail`을 강제하며 [기본 정책21–33행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/discovery.py#L21-L33)은 project scope의 admin/service role입니다. 기본 목록 이후 상세 목록을 자동 요청하거나403에서 basic으로 fallback하지 않습니다. SDK는 caller role을 preflight하거나 저장소 정보로 import 실행을 자동 허용·차단하지 않습니다.

Mutable Python Resource의 재사용·dirty/cache·configured session/adapter·arbitrary subclass·override 전체는 별도 남은 범위입니다. local 계약과 문서 빌드는 Go 동작의 증거이며 Python runtime 실행이나 인증된 cloud의 권한·capability를 증명하지 않습니다.

Import의 descriptor·별칭·syntax tolerance·응답 증거는 [owned Import 테스트](import_record_test.go), 기본 stores의 필터·location·raw cap·페이지·동일 목록 반복과 legacy detail 보존은 [owned Store 테스트](store_records_test.go)에서 비교합니다. [Connection 테스트](../../../connection_image_serviceinfo_records_test.go)는 상위/버전 API의 공유 client·현재 location을 확인합니다. 새 HTTP/fault engine을 만들지 않고 기존 serviceinfo fixture와 공통 REST·filter 계약을 재사용합니다.
