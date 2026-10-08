# Nova availability zone: Python과 Go

`conn.Compute(ctx)`의 `ListAvailabilityZones`는 일반 availability zone을 lazy 조회하고, `ListAvailabilityZoneNames`는 Cloud의 이름 목록·unavailable 기본값·실패 처리를 SDK 안에서 조합합니다. builder나 별도 목록 엔진을 애플리케이션에 구현할 필요가 없습니다.

| 고정 openstacksdk | Go | 기본값·반환 |
|---|---|---|
| `conn.compute.availability_zones(details=False)` | `service.ListAvailabilityZones(ctx)` 또는 `service.API.AvailabilityZones.ListRecords(ctx)` | 일반 경로의 lazy owned record |
| `conn.list_availability_zone_names(unavailable=False)` | `service.ListAvailabilityZoneNames(ctx)` | available 이름만 eager 수집 |
| `conn.list_availability_zone_names(unavailable=True)` | 위 호출에 `compute.WithUnavailableZones(true)` | unavailable 이름도 포함 |
| `conn.compute.availability_zones(details=True)` | 기존 `service.API.AvailabilityZones.ListDetail(ctx)` | native typed 상세 목록; 관리자 분기는 후속 검토 범위 |

일반 조회는 `GET /os-availability-zone`이고 상세 조회는 `/os-availability-zone/detail`입니다. [Nova 2024.1 기본 정책](https://docs.openstack.org/nova/2024.1/configuration/policy.html)은 일반 list에 `@`, 상세 detail에 `rule:context_is_admin`을 지정합니다. 따라서 일반 목록·Cloud 이름 조회를 핵심 user 순서에 두고 상세 host/service 목록은 핵심 admin 순서로 유지합니다. 실제 접근 권한은 해당 배포의 정책이 결정합니다. 고정 Proxy의 하나인 `availability_zones` 선언은 상세 분기까지 포함하므로 일반 경로를 구현한 것만으로 전체 선언을 완료했다고 표시하지 않습니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
for zone in conn.compute.availability_zones():
    print(zone.name, zone.state, zone.hosts)

print(conn.list_availability_zone_names())
print(conn.list_availability_zone_names(unavailable=True))
```

## 독립 Go main

`-cloud`로 clouds.yaml의 cloud를 선택합니다. 첫 호출은 일반 record 목록을 출력하고, 두 번째는 기본 available 이름 목록을 출력합니다. `-unavailable`을 지정하면 두 번째 호출에 `WithUnavailableZones(true)`를 적용합니다. 두 호출은 각각 별개의 HTTP 목록 작업이며 같은 inventory를 재사용하는 cache를 만들지 않습니다. 예제의 검증 범위는 컴파일이며 실제 인증·OpenStack 실행 결과는 별도입니다.

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
)

func main() {
    cloudName := flag.String("cloud", "dev", "clouds.yaml cloud name")
    unavailable := flag.Bool("unavailable", false, "include unavailable zone names")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloudName, *unavailable); err != nil {
        log.Fatal(err)
    }
}

func printJSON(value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloudName string, unavailable bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }

    records := make([]map[string]any, 0)
    for record, readErr := range service.ListAvailabilityZones(ctx) {
        if record != nil {
            records = append(records, map[string]any{
                "resource": record.Resource, "wire": record.Wire,
                "envelope": string(record.Envelope),
                "header": record.Header, "status_code": record.StatusCode,
            })
        }
        if readErr != nil {
            return errors.Join(readErr, printJSON(map[string]any{
                "operation": "zones", "partial_records": records,
            }))
        }
    }
    if err := printJSON(map[string]any{"operation": "zones", "records": records}); err != nil {
        return err
    }

    options := make([]compute.AvailabilityZoneNamesOption, 0)
    if unavailable { options = append(options, compute.WithUnavailableZones(true)) }
    result, readErr := service.ListAvailabilityZoneNames(ctx, options...)
    if result != nil {
        suppressed := ""
        if result.SuppressedError != nil { suppressed = result.SuppressedError.Error() }
        printErr := printJSON(map[string]any{
            "operation": "names", "names": result.Names, "value": result.Value,
            "inventory_count": len(result.Inventory), "suppressed_error": suppressed,
        })
        if printErr != nil { return errors.Join(readErr, printErr) }
    }
    return readErr
}
```

이름의 raw 값을 담은 `Names`와 `Value`는 JSON으로, Envelope는 실제 응답 문자열로 출력합니다. error와 result가 함께 반환되어도 알려진 기록을 표시한 뒤 error를 반환합니다.

## concrete 옵션과 모델

일반 leaf 목록의 `AvailabilityZoneListOpts`는 `Microversion *string`만 갖습니다. `WithAvailabilityZoneListOptions`·`WithAvailabilityZoneMicroversion`·`WithAvailabilityZoneHeader`를 사용하며 Service의 `ListAvailabilityZones`도 같은 leaf 옵션을 받습니다. query·Body 확장·상세 경로를 일반 목록에 섞는 옵션은 없습니다.

Cloud 이름 목록의 `AvailabilityZoneNamesOpts`는 `Unavailable bool`, `Microversion *string`을 갖습니다. `WithAvailabilityZoneNamesOptions`는 typed 값을 설정하고 `WithUnavailableZones`, `WithAvailabilityZoneNamesMicroversion`, `WithAvailabilityZoneNamesHeader`는 개별 값을 지정합니다. 옵션은 SDK가 소유 복사하고 한 logical call에 한 번 적용합니다. microversion nil은 캡처한 client의 선택 버전을 사용하고, 명시 empty는 해당 호출의 버전 header를 생략합니다. 일반 AZ 조회에 version discovery 요청을 추가하지 않으며 호출별 설정으로 원래 client를 변경하지 않습니다.

`availabilityzones.AvailabilityZoneRecord`의 `Resource`는 `id`, `name`, `state`, `hosts`, `location`을 갖는 owned response view입니다. `zoneName`은 `name`, `zoneState`는 `state`로 대응하며 누락 필드는 JSON null입니다. **name으로 id를 새로 만들지 않습니다.** state·hosts·name의 실제 JSON은 문자열·bool·typed map으로 강제 변환하지 않아 null과 큰 숫자·비문자열도 보존합니다. `Wire`는 알려진 필드뿐 아니라 vendor 값을 포함한 실제 행이고, `Envelope`·`Header`·`StatusCode`는 실제 응답의 증거입니다.

Service는 옵션 적용 전에 Connection의 현재 location을 한 번 snapshot하여 각 Resource에 독립적으로 추가합니다. standalone leaf에는 null location을 사용하고 실제 Wire·Envelope의 location 값은 바꾸지 않습니다. 반환 view는 caller가 수정할 수 있는 소유 복사이며 Python의 mutable Resource 전체 lifecycle이나 session/cache를 구현했다는 뜻은 아닙니다. lazy 순회는 공통 guarded pager를 재사용하며 소비자가 중단하면 다음 페이지를 조회하지 않습니다. owned 읽기는 Source의 성공 범위인 200..399를 받아 object/행을 검증합니다. 빈 204도 JSON representation을 요구하므로 원래 응답 증거를 가진 오류로 드러냅니다. dictionary 형태의 links.next·singular collection object 같은 공통 Go 호환 처리와 native pager의 단일 페이지 정책은 서로 구분합니다.

canonical `name`/`state`와 wire `zoneName`/`zoneState`가 함께 오면 Source JSON dictionary 순서상 나중 alias가 이깁니다. 같은 key의 중복은 최초 위치·최종 값을 유지합니다. unknown 필드는 actual Wire에만 남습니다. 공통 pager는 body·header next 후보가 충돌하면 오류로 처리하고 origin·고정 collection·필터 변경을 제한합니다. 이는 Source의 첫 body next 우선 선택보다 엄격한 Go 정책입니다.

## Cloud 이름과 실패 처리

Source는 `zone.state['available'] or unavailable` 순서로 평가합니다. `unavailable=true`도 먼저 state의 available 값을 조회하므로 누락·null·잘못된 state를 건너뛰지 않습니다. available에는 JSON 영역의 Python truthiness를 적용하며 포함한 이름은 raw 값으로 보존합니다. source 순서·중복 이름을 유지하고 name 누락/null도 문자열을 합성하지 않습니다. Python의 `list[str]` annotation과 실제 descriptor 값의 범위를 구분하기 위해 Go의 `Names`는 `[]json.RawMessage`입니다.

완료 결과의 `Value`는 Names의 JSON 배열이고 정상 빈 목록은 `[]`입니다. `Inventory`에는 소비한 owned record를 보존합니다. 고정 Cloud 함수는 SDKException을 잡아 빈 배열을 반환하며, 늦은 목록 실패라면 앞서 수집한 이름도 버립니다. Go는 실제 HTTP 4xx/5xx 실패와 source의 반복 marker SDKException에 대응하는 순수 pagination cycle을 이 범위로 처리하여 error는 nil, `Names`·`Value`는 완료된 빈 배열, `SuppressedError`는 원래 원인으로 반환합니다. partial Inventory는 진단 자료로 남습니다. 이 함수 자체에는 retained inventory cache가 없습니다.

malformed state·JSON decode·accepted body read/Close 오류·그 안의 nested HTTP 원인·취소·source guard 실패는 suppression 대상이 아닙니다. retry callback이 원래 HTTP 오류를 그대로 반환해 같은 error 값이 중복된 경우는 순수 HTTP 실패로 처리합니다. HTTP 또는 cycle에 다른 terminal 실패 원인이 함께 있으면 그 원인을 빈 성공으로 숨기지 않습니다. 이때 error와 알려진 partial Inventory를 반환하고 Value를 완결 성공으로 채우지 않습니다. `SuppressedError`가 있는 빈 목록과 정상적인 빈 목록을 구분하고, 부모 context 취소 원인도 보존합니다.

## native 호환과 남은 범위

기존 `AvailabilityZones.List(ctx)`는 native `AvailabilityZone` DTO와 SinglePageBase를 사용합니다. `ZoneName` string·`ZoneState.Available` bool·Hosts/service timestamp의 typed decode, native pager의 200/204/300 정책과 단일 페이지 처리는 owned raw record 목록과 구분합니다. missing/null native bool이 false가 되는 것과 Python의 untyped state descriptor가 원문을 유지하는 것은 서로 다른 반환 정책입니다. 일반 source 목록의 continuation은 SDK의 owned reader에서 처리합니다.

`ListDetail`의 관리자 host/service 조회, 이를 포함하는 전체 Proxy 선언의 판정과 일반 mutable Resource/session 표면은 후속 범위입니다. 일반 사용자 두 흐름의 구현·테스트·문서는 [단계별 현황](../docs/implementation-plan.md)과 [지원 판정대장](../docs/sdk-support-ledger.md)에서 실제 검증 결과로 관리합니다. 이 가이드의 존재만으로 전체 availability zone 가족이나 SDK 전체를 완료했다고 세지 않습니다.

고정 소스는 [Proxy availability_zones](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1854-L1874), [AvailabilityZone descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/availability_zone.py#L16-L36), [Cloud list_availability_zone_names](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L255-L276)입니다.
