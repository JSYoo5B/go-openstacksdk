# Flavor extra specs: Python과 Go

`conn.Compute(ctx)`의 `service.API.Flavors.FetchExtraSpecs`는 openstacksdk의 `conn.compute.fetch_flavor_extra_specs(flavor)`에 대응합니다. flavor ID 또는 기존 flavor를 concrete 입력으로 받고, 기존 값에 inline extra specs가 있어도 `/flavors/{id}/os-extra_specs`를 GET합니다. flavor member 조회·목록·이름 검색으로 ID를 찾거나 inline 값을 재사용하지 않습니다.

아래 표의 api는 `service.API.Flavors`입니다.

| 고정 openstacksdk 호출 | Go 호출 | 반환 |
|---|---|---|
| `conn.compute.fetch_flavor_extra_specs("7")` | `api.FetchExtraSpecs(ctx, flavors.FlavorExtraSpecsRequest{ID: "7"})` | `*flavors.FlavorExtraSpecsRecord` |
| `conn.compute.fetch_flavor_extra_specs(existing)` | `api.FetchExtraSpecs(ctx, flavors.FlavorExtraSpecsRequest{Flavor: existing})` 또는 `{Resource: existingRaw}` | 입력 snapshot의 extra specs를 갱신한 독립 결과 |
| `existing.fetch_extra_specs(conn.compute)` | 같은 `FetchExtraSpecs` | Python은 기존 Resource를 갱신하고 Go는 기존 입력을 보존 |

Python 예제는 inline 값이 있는 기존 Resource에도 별도 조회가 일어나는 경우입니다.

```python
import sys
import openstack
from openstack.compute.v2.flavor import Flavor

conn = openstack.connect()
existing = Flavor.existing(id=sys.argv[1], extra_specs={"inline": "old"})
updated = conn.compute.fetch_flavor_extra_specs(existing)
print(updated.extra_specs)
print(updated.to_dict())
```

## 독립 Go main

인증 환경을 준비하고 `-flavor-id 7`을 전달합니다. `-microversion`을 생략하면 SDK가 아래의 선택 정책을 적용하고, `-microversion=`을 명시하면 version header 없이 조회합니다. 이 main은 extra specs 조회만 수행합니다. 문서 예제의 빌드 검증과 실제 cloud 실행은 별도입니다.

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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    id := flag.String("flavor-id", "", "flavor ID to fetch extra specs for")
    version := flag.String("microversion", "", "optional operation version; explicit empty is versionless")
    flag.Parse()
    versionSet := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "microversion" {
            versionSet = true
        }
    })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *id, *version, versionSet); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, id, version string, versionSet bool) error {
    if id == "" {
        return fmt.Errorf("-flavor-id is required")
    }
    conn, err := sdk.Connect(ctx)
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    options := []flavors.FlavorExtraSpecsOption{}
    if versionSet {
        options = append(options, flavors.WithFlavorExtraSpecsMicroversion(version))
    }
    record, fetchErr := service.API.Flavors.FetchExtraSpecs(ctx,
        flavors.FlavorExtraSpecsRequest{ID: id}, options...)
    if record != nil {
        data, printErr := json.MarshalIndent(map[string]any{
            "resource": record.Resource,
            "wire": record.Wire,
            "extra_specs": record.ExtraSpecs,
            "status_code": record.StatusCode,
            "header": record.Header,
        }, "", "  ")
        if printErr != nil {
            return errors.Join(fetchErr, printErr)
        }
        fmt.Println(string(data))
        fmt.Printf("response body: %q\n", record.Envelope)
    }
    return fetchErr
}
```

## Concrete 입력과 옵션

`FlavorExtraSpecsRequest`의 `ID string`, `Flavor *flavors.Flavor`, `Resource *resource.RawResource`를 사용합니다. native DTO를 가진 경우 Flavor, nullable·unknown JSON 필드를 유지하려면 Resource를 전달합니다. Flavor와 Resource는 동시에 전달할 수 없습니다. builder interface를 구현할 필요가 없습니다.

명시 ID는 요청 경로를 결정하며 기존 Resource의 id를 덮지 않습니다. ID가 없으면 기존 입력의 첫 truthy `id` → `name` → `original_name`을 사용합니다. native Flavor에는 original_name이 없으므로 ID → Name 순서입니다. null·false·0·빈 값은 건너뛰고, 선택한 값은 문자열이어야 합니다. 이것은 입력의 별칭 해석이며 서버 이름 검색이 아닙니다. Go의 ID·UTF-8·control 검사를 통과한 값만 한 path segment로 escape합니다.

호출 시작 시 기존 입력을 복사하고 extra_specs만 교체합니다. RawResource의 다른 필드와 기존 metadata를 유지하며, native Flavor의 Swap도 보존하고 IsPublic/Ephemeral을 `is_public`/`ephemeral`로 표현합니다. ID만 전달하면 Resource에 id와 extra_specs만 생기며 full Flavor 기본값이나 computed location을 합성하지 않습니다. Python의 기존 객체 갱신과 달리 caller 입력·입력 map·원래 service client는 변경하지 않습니다.

`FlavorExtraSpecsOpts{Microversion: *string}`와 `WithFlavorExtraSpecsOptions`, `WithFlavorExtraSpecsMicroversion`, `WithFlavorExtraSpecsHeader`를 제공합니다. bulk 옵션의 pointer와 callback이 만든 mutable carrier는 SDK가 소유합니다. header는 caller override, 기존 source header, 기본 `Accept: application/json` 순으로 적용합니다. 인증·version 소유 header는 공통 source 정책을 따릅니다. body나 query를 추가하는 옵션은 이 고정 GET에서 허용하지 않습니다.

## Raw 값과 Resource의 dict view

| 반환 필드 | 의미 |
|---|---|
| `Resource` | 입력의 독립 snapshot에 dict descriptor를 적용한 extra_specs만 갱신 |
| `ExtraSpecs` | 실제 extra_specs의 `json.RawMessage`; key 생략 시 `{}` |
| `Wire` | unknown 필드도 포함한 실제 응답 root 객체 |
| `Envelope` | 실제 HTTP body 전체 |
| `Header`, `StatusCode` | 이번 extra specs GET의 실제 응답 metadata |

고정 Python의 `Flavor.extra_specs`는 `Body(type=dict, default={})`입니다. dict converter와 null 처리에 맞춰 Resource의 값은 아래처럼 해석합니다. `ExtraSpecs`와 Wire는 변환 전 값을 별도로 보존하므로 잘못된 타입의 응답도 확인할 수 있습니다.

| 실제 extra_specs | `record.ExtraSpecs` | `record.Resource.Body["extra_specs"]` |
|---|---|---|
| 객체 | 원문 JSON 값 | 객체 그대로 |
| `null` | `null` | `null` |
| key 생략 | `{}` | `{}` |
| 배열·문자열·수·bool | 실제 값 그대로 | `{}` |

key 생략 시 Wire에는 extra_specs를 합성하지 않습니다. 응답의 id/name/다른 필드도 기존 Resource를 덮지 않고 Wire에만 남습니다. 큰 JSON 수는 raw bytes로 유지합니다. Resource·Wire·ExtraSpecs·Envelope·Header는 독립적으로 소유하며 한 결과의 map/bytes를 수정해도 입력이나 다른 반환 carrier를 바꾸지 않습니다.

기존 RawResource metadata는 Resource에 그대로 남으므로 이번 조회의 status/header는 **record의 StatusCode/Header**를 확인합니다. Wire metadata도 이번 응답을 나타냅니다.

## 버전·경로·오류

Microversion 옵션이 nil이면 이미 선택된 source 버전을 유지합니다. 선택된2.100이나 `latest`를2.61로 낮추지 않습니다. 미선택일 때만 [공통 Nova discovery](console-selection.md)로 광고 범위 안의 버전을2.61 한도로 선택합니다. 광고 max가 없거나 min이2.61보다 높으면 version 없이 조회합니다. 명시적인 빈 문자열은 discovery 없이 versionless GET을 선택합니다. 선택은 private GET client에 적용하며 공유 client는 보존합니다.

2.61은 Flavor Resource의 자동 선택 상한이자 flavor detail의 inline extra specs 도입 버전입니다. 별도 `/os-extra_specs` endpoint의 최소 버전 gate로 사용하지 않습니다. inline 여부와 관계없이 이 함수는 지정한 endpoint를 한 번 조회합니다. 인증·선택이 필요하면 그 요청이 앞서고, native HTTP retry/reauth 정책에 따른 physical 재시도가 있을 수 있습니다.

owned GET은200..399를 수락한 뒤 UTF-8 JSON의 nonnull root 객체를 요구합니다. malformed JSON·invalid UTF-8·null/scalar/array root·빈204는 processing 오류입니다. generic Resource fetch의 nonJSON 허용 정책을 이 함수에 적용하지 않습니다. extra_specs 자체의 null/scalar/array는 위 표대로 처리합니다.

수락한 응답의 read/Close/source/context 실패는 terminal이며 실제 원문·header·status와 원래 cause를 보존합니다. 오류와 함께 receipt record가 반환될 수 있고 Resource가 nil일 수 있으므로 error를 먼저 확인합니다. accepted 실패를 missing으로 바꾸거나 fallback flavor 조회로 재시도하지 않습니다. 거절된 HTTP 응답의403/404 등은 원래 Gophercloud HTTP 오류를 유지합니다. source 변경·context 취소·invalid input/option·outer operation guard 오류는 뒤 HTTP/callback을 중단합니다.

Go discovery는 SDK의 유한 경로·captured source·공통 guard를 사용합니다. Python session의 전체 discovery cache·sorting·anonymous 전환을 재현하지 않으며 discovery JSON/physical/source 오류는 terminal입니다. 기존 [flavor find의 extra specs 옵션](user-read-apis.md)은 inline 값 재사용 정책을 유지하고, 이 명시적인 always-fetch API와 구분합니다.

## Native ListExtraSpecs와 비교

`service.API.Flavors.ListExtraSpecs(ctx, id)`는 기존 Gophercloud binding으로 계속 사용할 수 있습니다. 요청 옵션·DTO ABI를 바꾸지 않고 `map[string]string`을 반환합니다.

| 항목 | FetchExtraSpecs | native ListExtraSpecs |
|---|---|---|
| 입력 | ID 또는 기존 native/raw flavor | flavor ID 문자열 |
| 성공 status |200..399 수락 후 JSON 검사 | 기본 GET 코드200 |
| extra_specs 결과 | raw JSON과 dict Resource view | `map[string]string` |
| missing/null extra_specs | `{}` / `null` 구분 | 둘 다 nil map |
| nonstring 값 | raw 보존, dict view 적용 | typed decode 오류와 partial map 가능 |
| 응답 원문·unknown 필드 | record의 Wire/Envelope | typed 반환에 포함하지 않음 |

native의 null map member는 Go string의 빈 값으로 해석합니다. native201/204는 성공으로 확장하지 않습니다. 따라서 문자열 map만 필요한 기존 호출을 바꿀 필요가 없으며, Python Resource 수준의 갱신·nullable 값·원문이 필요한 경우 FetchExtraSpecs를 선택합니다.

## 근거와 테스트 재사용

계약은 고정 openstacksdk [`fetch_flavor_extra_specs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L334), [`Flavor.fetch_extra_specs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py#L186), [`fields._convert_type`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)와 [`_BaseComponent.__get__`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L208), Gophercloud v2.15.0의 [`ListExtraSpecs`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/flavors/requests.go#L277)와 [`Extract`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/flavors/results.go#L220)를 기준으로 합니다.

조회는 기존 `microversions.MemberGet`, REST decode/receipt, RawResource clone를 공유합니다. [고유 binding 테스트](../api/compute_flavor_extra_specs_fetch_test.go)는 public Gophercloud testhelper와 기존 testcloud·flavorIdentityClient·payloadContractTrack·secretFetchRoundTripFunc를 재사용합니다. 기존 nativefind flavor specs·guarded response·version·ownership 회귀를 함께 사용하며 서비스마다 transport fault matrix를 복제하지 않습니다. 검증 결과와 지원 판정은 [검증 기록](../docs/sdk-support-ledger.md), [구현 계획](../docs/implementation-plan.md)에 따로 남깁니다.
