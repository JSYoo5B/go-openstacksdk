# Flavor 단일 extra-spec과 native 조회

`conn.Compute(ctx)`의 `service.API.Flavors.GetExtraSpecsProperty`는 지정한 extra-spec 값 하나를 조회합니다. `service.GetFlavorByID`는 Cloud의 ID 전용 flavor 조회이며, 기존 `service.API.Flavors.Get`·`GetExtraSpec`·`ListDetail`은 native 반환형과 상태 코드를 유지합니다.

## Python과 Go의 대응

아래 표의 `api`는 `service.API.Flavors`입니다.

| 고정 openstacksdk 또는 native Gophercloud | Go 호출 | 반환 |
|---|---|---|
| `conn.compute.get_flavor_extra_specs_property("7", "hw:cpu_policy")` | `api.GetExtraSpecsProperty(ctx, flavors.FlavorExtraSpecsRequest{ID: "7"}, "hw:cpu_policy")` | raw 선택 값과 실제 응답의 `*flavors.FlavorExtraSpecPropertyRecord` |
| 같은 호출에 기존 Python Flavor 전달 | 입력의 `Flavor` 또는 `Resource` 필드 사용 | 기존 입력을 변경하지 않는 독립 결과 |
| Cloud `conn.get_flavor_by_id("7")` | `service.GetFlavorByID(ctx, "7")` | native `*flavors.Flavor`, 기본 specs 보강 없음 |
| `conn.get_flavor_by_id("7", get_extra=True)` | 위 호출에 `compute.WithFlavorByIDExtraSpecs(true)` | inline specs가 비어 있을 때만 보강 |
| native `flavors.Get(...).Extract()` | `api.Get(ctx, "7")` | native `*flavors.Flavor` |
| native `flavors.GetExtraSpec(...).Extract()` | `api.GetExtraSpec(ctx, "7", "hw:cpu_policy")` | flat 전체 `map[string]string` |
| native `flavors.ListDetail(...)` | `api.ListDetail(ctx, flavors.WithListDetailOptions(...))` | native Flavor를 순서대로 반환하는 iterator |

Python의 단일 property 호출에는 query·필터·`ignore_missing` 옵션이 없습니다. ID나 기존 Flavor에서 경로를 만들며 flavor GET·이름 검색·전체 extra-specs GET을 추가하지 않습니다.

이 조회는 핵심 user 단계에서 구현합니다. [Nova 기본 정책](https://docs.openstack.org/nova/latest/configuration/policy.html)의 extra-specs show/index는 `project_reader_or_admin`을 사용하며, 생성·수정·삭제는 admin 단계에 둡니다. 실제 접근 권한은 배포한 cloud의 정책이 결정합니다.

```python
import openstack

conn = openstack.connect()
flavor = conn.get_flavor_by_id("7", get_extra=False)
value = conn.compute.get_flavor_extra_specs_property(flavor, "hw:cpu_policy")
print(flavor.id, value)
```

## 독립 Go main

인증 설정을 준비하고 `-flavor-id 7 -property hw:cpu_policy`를 전달합니다. `-extra`를 지정하면 Cloud ID 조회의 조건부 specs 보강도 수행합니다. 예제는 Compute2.55를 명시하고 인증·조회 전체에 같은1분 context를 사용합니다. native 목록은 처음20개를 출력합니다. 실제 cloud 실행은 예제의 빌드 검증과 별도입니다.

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

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    id := flag.String("flavor-id", "", "flavor ID")
    property := flag.String("property", "hw:cpu_policy", "single extra-spec property")
    extra := flag.Bool("extra", false, "conditionally fetch missing inline flavor specs")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *id, *property, *extra); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, id, property string, extra bool) error {
    if id == "" {
        return fmt.Errorf("-flavor-id is required")
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    cloudFlavor, err := service.GetFlavorByID(ctx, id,
        compute.WithFlavorByIDExtraSpecs(extra))
    if err != nil {
        return fmt.Errorf("cloud flavor by ID: %w", err)
    }
    api := service.API.Flavors
    record, propertyErr := api.GetExtraSpecsProperty(ctx,
        flavors.FlavorExtraSpecsRequest{ID: id}, property)
    if record != nil {
        data, printErr := json.MarshalIndent(map[string]any{
            "value": record.Value,
            "present": record.Present,
            "wire": record.Wire,
            "response_body": string(record.Envelope),
            "header": record.Header,
            "status_code": record.StatusCode,
        }, "", "  ")
        if printErr != nil {
            return errors.Join(propertyErr, printErr)
        }
        fmt.Println(string(data))
    }
    if propertyErr != nil {
        return fmt.Errorf("single property: %w", propertyErr)
    }
    nativeFlavor, err := api.Get(ctx, id)
    if err != nil {
        return fmt.Errorf("native flavor: %w", err)
    }
    nativeProperty, err := api.GetExtraSpec(ctx, id, property)
    if err != nil {
        return fmt.Errorf("native property string map: %w", err)
    }
    listed := make([]*flavors.Flavor, 0, 20)
    for flavor, listErr := range api.ListDetail(ctx,
        flavors.WithListDetailOptions(flavors.ListOpts{Limit: 20})) {
        if listErr != nil {
            return fmt.Errorf("native detailed list: %w", listErr)
        }
        listed = append(listed, flavor)
        if len(listed) == 20 {
            break
        }
    }
    data, err := json.MarshalIndent(map[string]any{
        "cloud_flavor": cloudFlavor,
        "native_flavor": nativeFlavor,
        "native_property_map": nativeProperty,
        "native_detailed_flavors": listed,
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}
```

## 단일 property의 raw 값과 옵션

`Value`는 `json.RawMessage`입니다. property가 응답에 없으면 `Value`는 JSON `null`, `Present`는 false입니다. 명시 `null`은 같은 Value에 Present true이며, 빈 문자열·숫자·배열·객체를 string이나 dict로 바꾸지 않습니다. Python의 type annotation은 `str | None`이지만 실제 helper는 `response.json().get(prop)`에 `cast`만 적용하므로 raw 값이 그대로 반환됩니다. 숫자를 문자열로 바꾸거나 dict descriptor 기본값을 넣지 않습니다.

`Wire`는 실제 flat 응답 object이고 `Envelope`·`Header`·`StatusCode`는 해당 응답의 receipt입니다. Value와 Wire의 raw JSON은 서로 독립된 복사입니다. 오류가 있으면 receipt가 반환되어도 Value/Wire가 준비되지 않았을 수 있으므로 error를 먼저 확인합니다. 완전하지 않은 응답을 정상 값으로 합성하거나 accepted read/Close/decode 실패 뒤 SDK가 GET을 추가로 보내지 않습니다. provider의 인증 재시도 정책은 별도로 유지됩니다.

입력과 `WithFlavorExtraSpecsOptions`·`WithFlavorExtraSpecsMicroversion`·`WithFlavorExtraSpecsHeader`는 [전체 specs 조회](flavor-extra-specs.md#concrete-입력과-옵션)와 공유합니다. 입력 ID/Flavor/RawResource를 호출 시작에 복사하며 명시 ID 또는 기존 입력의 id→name→original_name을 사용합니다. 반환에 Resource view를 새로 만들거나 supplied Flavor의 ExtraSpecs를 갱신하지 않습니다.

property는 literal path segment로 escape합니다. 공백·slash·percent·예약 문자를 문자 그대로 선택할 수 있고 빈 값·`.`·`..`·control·잘못된 UTF-8은 HTTP 전에 거부합니다. Python은 `utils.urljoin`으로 slash를 strip해 경로를 결합하므로 이 입력 안전성 정책은 Go 차이입니다.

선택된 Compute microversion은 유지합니다. 버전이 미선택이면 최대2.61까지 조회된 광고 범위를 사용하며 명시 빈 microversion은 versionless 요청입니다. 추가 query/body를 보내지 않고 고정 member GET의 final200..399를 수용합니다. root는 UTF-8 JSON object여야 하며 빈 body·malformed JSON·null/array root는 receipt가 있는 오류입니다. 이 정책을 native strict200과 구분하세요.

## Cloud ID 조회와 native ABI

`GetFlavorByID`의 `FlavorByIDOpts.GetExtraSpecs` 기본값은 false입니다. `compute.WithFlavorByIDOptions`·`WithFlavorByIDExtraSpecs`로 concrete 선택을 전달합니다. SDK가 GET-only FindFallbackNever/IgnoreMissing(false) 기본값을 소유하므로400/403/404를 이름 목록 fallback이나 정상 미존재로 바꾸지 않습니다. true이면 성공한 flavor의 inline ExtraSpecs가 nil/빈 map일 때 canonical 반환 ID의 전체 specs를 한 번 조회합니다. inline nonempty map이면 추가 GET이 없습니다. 이 조건부 보강은 단일 property 조회와 별도 호출입니다. query/body/header/argument callback을 이 bool 전용 API에 추가하면 HTTP 전에 거부합니다.

Cloud 결과는 native `*flavors.Flavor`입니다. source Cloud `get_flavor_by_id`가 위임하는 Proxy `get_flavor`와 같은 [GET-only 대응과 DTO 차이](user-read-apis.md#flavor의-get-전용-정책)를 사용합니다. safe 명시·반환 ID 검사와 선택된 버전 유지, native nil map/기본값은 Python seeded mutable Flavor와 구분합니다.

기존 native `api.Get`은200만 성공하고 flavor wrapper가 missing/null이면 native nil 결과를 유지합니다. `api.GetExtraSpec`은200의 flat `map[string]string` 전체를 반환합니다. requested key를 자동으로 고르는 scalar API가 아니며, null string member는 native 빈 문자열이고 비문자열 member는 typed decode 오류입니다. typed decode에서 부분 Flavor/map과 error가 함께 반환될 수 있습니다. 오류가 있으면 이 값을 성공으로 취급하지 않습니다. HTTP 상태·body·header·method·URL의 native 원인을 wrapped error에서 확인할 수 있습니다.

## Native 상세 목록

`api.ListDetail`은 `/flavors/detail`과 native `FlavorPage`를 사용합니다. `WithListDetailOptions(flavors.ListOpts{...})`의 ChangesSince/MinDisk/MinRAM/SortDir/SortKey/Marker/Limit/AccessType은 각각 `changes-since`/`minDisk`/`minRam`/`sort_dir`/`sort_key`/`marker`/`limit`/`is_public`으로 전달합니다. zero/default 값은 생략하며 `WithListDetailQuery`로 raw 확장 query를 추가합니다. 기본 is_public=None 또는 항목별 specs 보강을 자동 적용하지 않습니다.

목록은200/204/300 native page 상태와 `flavors_links` continuation을 유지합니다. iterator에서 break하면 사용하지 않는 다음 페이지를 요청하지 않습니다. 반환 모델·typed 전체 page decode와 empty-page 종료는 native 정책입니다. 명시 private/all access query는 해당 cloud의 정책 권한을 요구할 수 있습니다.

Python `conn.compute.flavors(details=False, get_extra_specs=True, **query)`의 summary 선택·항목별 specs 보강·semantic query/Body 필터 및 전체 inherited continuation은 이 native 상세 목록과 별도 지원 범위입니다. [Flavor 자동 조회](../docs/finding-identities.md#nova-flavor와-extra-specs)의 FindIdentity fallback 정책이나 모든 Resource/session API가 이 단위로 완료되었다고 해석하지 않습니다.
