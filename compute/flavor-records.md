# Flavor 목록·이름 조회: Python과 Go

`conn.Compute(ctx)`의 `service.ListFlavors`와 `service.FindFlavor`는 openstacksdk의 Compute Proxy 목록/find를 SDK가 소유하는 옵션·응답 view·페이지·조건부 extra specs로 연결합니다. 애플리케이션이 builder나 resolver interface를 구현할 필요가 없습니다. 옵션 타입은 `compute/v2/flavors`의 concrete 타입을 그대로 사용합니다.

| 고정 openstacksdk | Go |
|---|---|
| `conn.compute.flavors(details=True, get_extra_specs=False, **query)` | `service.ListFlavors(ctx, flavors.WithFlavorListFilters(query))` |
| `conn.compute.flavors(details=False, get_extra_specs=True)` | `ListFlavors(ctx, flavors.WithFlavorListDetails(false), flavors.WithFlavorListExtraSpecs(true))` |
| `conn.compute.find_flavor(name_or_id, ignore_missing=True, get_extra_specs=False, **query)` | `service.FindFlavor(ctx, nameOrID, flavors.WithFlavorFindFilters(query))` |
| `conn.compute.find_flavor(name_or_id, ignore_missing=False, get_extra_specs=True)` | `FindFlavor(ctx, nameOrID, flavors.WithFlavorFindIgnoreMissing(false), flavors.WithFlavorFindExtraSpecs(true))` |

Python의 `query` dictionary는 Go에서 `map[string]any`로 표현합니다. 아래 예제의 `min_ram`은 서버 query이고 `disk`는 응답의 로컬 필터입니다. 일반 사용자에게 보이는 flavor 범위와 extra specs 권한은 Nova의 배포 정책이 판단하며, SDK가 role을 추정하거나 권한을 확장하지 않습니다.

```python
import openstack

conn = openstack.connect(compute_api_version="2.55")
for flavor in conn.compute.flavors(min_ram=1024, disk=20):
    print(flavor.to_dict())

found = conn.compute.find_flavor("m1.small", ignore_missing=True,
                                 get_extra_specs=True)
if found is not None:
    print(found.to_dict())
```

## 독립 Go main

인증 환경 또는 `clouds.yaml`의 cloud를 준비합니다. 기본 호출은 raw 행 최대20개를 처리하는 목록이며 `-flavor ID_OR_NAME`을 추가하면 find도 수행합니다. `-extra`는 두 호출의 조건부 extra specs 보충을 켭니다. 선택 버전2.55는 서버가 지원해야 하며 SDK가 이 값을 자동 변경하지 않습니다. 이 예제의 검증 범위는 컴파일이고 실제 OpenStack 호출은 별도입니다.

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
    "github.com/JSYoo5B/gophercloudsdk/compute/v2/flavors"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    identity := flag.String("flavor", "", "optional flavor ID or name to find after listing")
    extra := flag.Bool("extra", false, "conditionally fetch missing or empty extra specs")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *identity, *extra); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, identity string, extra bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    printRecord := func(operation string, record *flavors.FlavorRecord) error {
        if record == nil { return nil }
        var enrichment any
        if child := record.Enrichment; child != nil {
            enrichment = map[string]any{
                "resource": child.Resource, "wire": child.Wire,
                "extra_specs": child.ExtraSpecs, "envelope": string(child.Envelope),
                "header": child.Header, "status_code": child.StatusCode,
            }
        }
        data, err := json.MarshalIndent(map[string]any{
            "operation": operation, "resource": record.Resource, "wire": record.Wire,
            "envelope": string(record.Envelope), "header": record.Header,
            "status_code": record.StatusCode, "enrichment": enrichment,
        }, "", "  ")
        if err != nil { return err }
        fmt.Println(string(data))
        return nil
    }
    for record, err := range service.ListFlavors(ctx,
        flavors.WithFlavorListMaxItems(20),
        flavors.WithFlavorListExtraSpecs(extra)) {
        if printErr := printRecord("list", record); printErr != nil { return errors.Join(err, printErr) }
        if err != nil { return fmt.Errorf("list flavors: %w", err) }
    }
    if identity == "" { return nil }
    record, err := service.FindFlavor(ctx, identity, flavors.WithFlavorFindExtraSpecs(extra))
    if printErr := printRecord("find", record); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("find flavor: %w", err) }
    if record == nil { fmt.Println("flavor not found") }
    return nil
}
```

`Envelope`는 JSON이 아닌 접수 응답도 표현할 수 있어 예제에서는 문자열로 출력합니다. error와 record가 함께 반환될 때도 증거를 출력하고 error를 처리합니다. 부분 결과의 존재를 전체 성공으로 취급하지 않습니다.

## 기본값과 필터

`FlavorListOpts`는 `Details *bool`, `GetExtraSpecs bool`, `Limit`, `Marker`, `MaxItems`, `Paginated *bool`, `Microversion *string`을 갖습니다. nil Details는 true라 `/flavors/detail`을 조회하고 false는 `/flavors`를 조회합니다. nil Paginated는 전체 페이지, GetExtraSpecs는 기본 false입니다. `FlavorFindOpts`는 같은 paging/version 필드와 `IgnoreMissing *bool`, `GetExtraSpecs`를 갖고 missing 기본값은 true입니다.

`WithFlavorListOptions`/`WithFlavorFindOptions`는 typed 필드만 교체합니다. 개별 Details/ExtraSpecs/Limit/Marker/MaxItems/Paginated/Microversion helper와 Header/Query/Filter/Filters를 조합할 수 있습니다. bulk Filters는 semantic 필터만 교체하고 nil/empty로 비웁니다. 반복 개별 Filter는 마지막 값이 이깁니다. pointer·map·mutable callback carrier는 SDK가 소유하며 옵션은 logical iteration/call에 한 번 적용합니다.

| semantic 속성 | 처리 |
|---|---|
| `limit`, `marker`, `sort_key`, `sort_dir`, `is_public`, `min_disk`, `min_ram` | 7개 서버 query; `min_disk`→`minDisk`, `min_ram`→`minRam` |
| `minDisk`, `minRam` | query alias; accepted 이름은 총9개 |
| `id`, `name`, `original_name`, `description`, `disk`, `ram`, `vcpus`, `swap`, `ephemeral`, `is_disabled`, `rxtx_factor`, `extra_specs` | 12개 로컬 Body 필터 |
| unknown semantic 목록 속성, 선언하지 않은 wire Body 이름 | 목록 필터에서 버림 |

`is_public`을 **생략하거나 semantic null로 지정하면** 목록 URL에 literal `is_public=None`을 보냅니다. false와 빈 문자열은 이 기본값과 다릅니다. Python query bool의 `True`/`False` 대신 Go는 소문자 `true`/`false`를 보냅니다. bulk map에서 canonical 이름과 query alias가 함께 있으면 canonical 값이 우선합니다.

raw Query는 같은 typed Limit/Marker query를 덮어쓸 수 있습니다. semantic query와 typed/raw query가 같은 wire 이름으로 충돌하면 값이 같아도 HTTP 전에 오류입니다. raw Query는 전달할 wire spelling을 사용하며 semantic alias 변환을 적용하지 않습니다. typed Limit/MaxItems의0은 생략/무제한, 음수는 오류이고 실제 limit은 하나의 양의 정수, marker는 하나의 비어 있지 않은 값이어야 합니다.

query 값은 scalar/null/scalar-array로 표현하고 null은 일반 URL에서 생략합니다. List의 Body 필터는 JSON 객체 subset도 지원합니다. Find의 semantic 값은 직접 GET query로도 전달되므로 객체·중첩 배열처럼 공통 query encoder가 지원하지 않는 값은 HTTP 전에 오류입니다. 이것은 Python의 임의 GET-param iterable 변환과 다른 Go 입력 범위입니다.

## 페이지와 extra specs

List는 lazy iterator입니다. iterator 생성은 HTTP·location·옵션 callback을 실행하지 않고, 소비할 때 한 번의 source/location snapshot을 준비합니다. `MaxItems`는 필터에 일치한 수가 아닌 **필터 이전 raw 행 수**를 제한하고, caller limit이 없으면 서버 limit hint를 사용합니다. 순서는 raw cap → Resource projection → 로컬 Body 필터 → 조건부 extra specs 보충 → yield입니다. `break`·`Paginated(false)`·cap은 뒤 페이지와 미소비 행의 보충을 중단합니다.

rel/href `links`·`flavors_links`, top-level `next`, HTTP Link를 따릅니다. 초기 effective limit이 있을 때 마지막 처리 raw 행의 논리 ID를 fallback marker로 사용합니다. 해당 행이 로컬 필터에서 제외되거나 caller가 반환 view를 수정해도 원래 행의 id/name/original_name snapshot에서 marker를 구합니다. 처음 advertised next가 positive limit을 추가할 수 있고 이후 값은 고정하지만, server가 처음 추가한 limit만으로 marker fallback을 켜지는 않습니다. 빈 페이지는 next가 있어도 중단합니다. 충돌하는 next·cycle·다른 route/origin/query로의 전환은 오류이며 reverse proxy prefix를 유지합니다. dictionary `links:{next:...}`는 Python ordinary-next dictionary 해석과 다른 **Go 추가 호환**입니다.

ExtraSpecs 옵션이 true일 때 **projected extra_specs가 falsey인 일치 행만** `/flavors/{logicalID}/os-extra_specs`로 보충합니다. inline nonempty 객체는 추가 GET 없이 유지합니다. present null/빈 객체나 dict descriptor가 `{}`로 바꾼 비객체 값은 보충 대상입니다. Find fallback은 유일성이 확정된 뒤에만 보충하며 중복·후속 목록 오류 뒤에는 실행하지 않습니다. 명시 [FetchExtraSpecs](flavor-extra-specs.md)는 inline 값과 관계없이 항상 GET하므로 이 조건부 옵션과 다릅니다.

## Resource와 실제 응답

`FlavorRecord.Resource`는13개 Body 속성에 location을 더한14필드 view입니다. 선언된 속성은 `id`, `name`, `original_name`, `description`, `disk`, `ram`, `vcpus`, `swap`, `ephemeral`, `is_public`, `is_disabled`, `rxtx_factor`, `extra_specs`입니다. 없는 integer 필드는0, 없는 is_public은 true, 없는 extra_specs는 `{}`, 나머지 기본값은 null입니다. **명시 null은 이 기본값으로 바뀌지 않습니다.** name은 해당 키가 없을 때만 original_name을 참고하고, 논리 id는 첫 truthy id/name/original_name입니다.

integer·bool·float·dict response descriptor는 view에 적용합니다. `rxtx_factor`의 유효 숫자 문자열은 float로 변환하고 invalid/empty 문자열은 오류입니다. bool은 숫자1/0, 컨테이너는0으로 변환합니다. 비유한 float/overflow는 유효 JSON view로 표현할 수 없어 Go에서 명시적 오류이며 Wire의 실제 값과 원래 응답 증거를 유지합니다. 정수·실수의 정밀도 변환, 공유 Unicode16 숫자 표와 Python runtime에 따른 숫자 차이는 response view에만 적용합니다. caller 필터를 이 타입으로 강제 변환하지 않습니다.

응답의 canonical/wire Body alias가 함께 있으면 **선택한 JSON 객체에서 마지막으로 나온 인식된 값**이 view를 결정합니다. wire alias는 `os-flavor-access:is_public`, `OS-FLV-EXT-DATA:ephemeral`, `OS-FLV-DISABLED:disabled`입니다. bulk query canonical 우선 정책과 구별합니다. List row는 flat이며 nested `flavor`가 있어도 outer 행을 대체하지 않습니다.

| 반환 필드 | 의미 |
|---|---|
| `Resource` | descriptor·alias·default·논리 ID와 Connection location을 적용한 독립 view |
| `Wire` | List의 실제 flat 행, 또는 member의 flavor 객체/없으면 flat root; unknown·큰 수·원래 값 보존 |
| `Envelope` | List 성공 행의 원문 row JSON, member의 실제 HTTP body 전체 |
| `Header`, `StatusCode` | List 페이지 또는 직접 member의 실제 응답 |
| `Enrichment` | 추가 specs GET의 독립 Resource/Wire/ExtraSpecs/Envelope/Header/StatusCode; 호출하지 않으면 nil |

Service는 Connection의 `CloudLocation`을 옵션·discovery 전에 한 번 snapshot하고 Resource와 Enrichment.Resource에만 주입합니다. leaf `service.API.Flavors.ListRecords`/`FindFlavor`를 직접 사용하면 observed Connection location을 추가하지 않아 location은 null입니다. 두 경로의 실제 Wire/Envelope는 바뀌지 않습니다. extra specs 요청은 기존 준비된 source/version을 재사용하고 옵션·discovery를 다시 적용하지 않습니다.

## Find와 오류·버전

Find는 UUID 추정이나 name hint 없이 UTF-8 literal 입력을 한 번 escape하여 `/flavors/{identity}`를 먼저 GET합니다. 공백·reserved 문자는 경로 안에서 escape하며 입력을 trim해 다른 이름으로 바꾸지 않습니다. empty/whitespace-only·controls·dot routing은 Go에서 거부합니다. semantic id는 고정 identity와 충돌하므로 거부합니다. 직접 GET은 caller의 원래 semantic spelling을 query와 선언된 Resource seed에 사용하고, `is_public=None`의 목록 기본값을 추가하지 않습니다. 예를 들어 min_ram은 직접 GET에서 min_ram, fallback 목록에서 minRam입니다.

**clean actual400·403·404만** `/flavors/detail` 목록 fallback을 허용합니다. fallback은 canonical 서버 query·Body 필터를 적용한 결과에서 caller 문자열과 ID 또는 name이 정확히 같은 행을 찾습니다. 숫자 passive ID/name은 문자열로 바꿔 일치시키지 않습니다. 첫 match 뒤에는 선택한 목록을 계속 읽어 늦은 오류를 확인하며 두 번째 match에서 즉시 중복 오류를 반환합니다. 중복은 `resource.ErrAmbiguous`, 전체 선택 목록의0건은 기본 `nil, nil` 또는 IgnoreMissing(false)의 `resource.ErrNotFound`입니다. caps/first-page를 지정한 경우 유일성은 선택한 범위에 대한 판단입니다.

owned reader는200..399를 수락합니다. 목록은 완결 UTF-8 JSON object의 flavors array/singleton object를 요구하며 malformed·empty204·null representation은 오류입니다. member GET은 source fetch처럼 accepted empty/nonJSON/malformed JSON을 seed view와 nil Wire로 처리하고 invalid UTF-8도 unavailable JSON으로 처리하지만, parsed root/flavor가 null·scalar·array이면 오류입니다. Python response charset/replacement decoding 전체를 재현하지는 않습니다. 물리 read/Close·source/context 오류는 이 tolerant 경로로 바뀌지 않습니다. accepted 오류의 `resource.ResponseError`는 실제 body/header/status와 원래 cause를 보존하고, nested404 때문에 fallback하거나 응답을 재전송하지 않습니다. late 목록 오류 이전의 행과 error 옆 partial record/Enrichment는 증거이며 완료 상태가 아닙니다.

nil Microversion는 원래 선택된 값을 유지하고, 미선택일 때만 [공통 Nova discovery](console-selection.md)와 최대2.61 선택을 사용합니다. 광고 max가 없거나 min이 ceiling보다 높으면 versionless로 계속합니다. `WithFlavorListMicroversion("")`/`WithFlavorFindMicroversion("")`는 discovery 없이 header를 생략하는 명시 override입니다. 선택2.100/latest를 ceiling에 맞춰 바꾸지 않고 원래 client를 수정하지 않습니다. 같은 logical find의 직접 GET·fallback·보충은 한 준비된 버전을 사용합니다. live provider 인증을 유지하며 source/context/서비스 binding을 단계와 완료 시 재검사합니다.

## 기존 API와 남는 차이

native `service.API.Flavors.Get`, `ListDetail`, `ListExtraSpecs`, `GetExtraSpec`와 typed `service.Flavors.FindIdentity`는 그대로 사용할 수 있습니다. native Flavor DTO, string-map extra specs, 원래 성공 코드·`flavors_links` linked pager·selected version 정책을 유지하며 새 default/filter/view/discovery를 합성하지 않습니다. 이 guide의 Service API는 Python Resource-shaped 결과가 필요한 호출의 대응입니다. [native/property 가이드](flavor-property-and-native.md)에 기존 API 차이를 별도로 설명합니다.

공통 로컬 matcher는 Go JSON 타입을 구별해 bool과 숫자를 같다고 보지 않습니다. 객체 subset 필터의 actual scalar는 false로 처리하므로 Python의 빈 dict truthiness나 nested scalar attribute 오류와 구별됩니다. arbitrary base_path/session/resource_type, deprecated jmespath_filters, allow_unknown_params 같은 Python Resource controls는 이 고정 경로 API의 raw query/semantic 속성으로 받지 않으며 dedicated 옵션 또는 명시 unsupported 오류로 처리합니다. full Python mutable Resource, session/cache/discovery lifecycle과 같은 API 전체 구현을 이 두 named 작업의 문서만으로 주장하지 않습니다.

고정 소스는 [Proxy flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L269-L296), [Proxy find_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L165-L199), [Flavor.list와 descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py#L43-L138), Resource.list/find입니다. 로컬 HTTP 검증·예제 빌드·named 지원 판정의 결과는 [검증 기록](../docs/sdk-support-ledger.md)에 별도로 남깁니다.
