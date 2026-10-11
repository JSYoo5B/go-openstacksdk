# SubnetPool: Python과 Go Record 사용법

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud
`v2.15.0`입니다. `conn.NetworkV2(ctx).SubnetPools`가 옵션과 응답의 소유권을
제공하므로 별도 builder나 확장 interface를 구현할 필요가 없습니다.

| Python proxy | Go | 기본 동작 |
|---|---|---|
| `update_subnet_pool(id, **attrs)` | `UpdateRecord(ctx, id, options...)` | 공급한 attrs만 `PUT /subnetpools/{id}`의 `subnetpool` envelope로 전송; attrs가 없으면 HTTP 생략 |
| `find_subnet_pool(name_or_id, ignore_missing=True, **query)` | `FindRecord(ctx, identity, options...)` | literal GET 먼저; 깨끗한 400/403/404에만 전체 목록의 정확한 ID/이름 검색 |
| `subnet_pools(**query)` | `ListRecords(ctx, options...)` | 지연 iterator, 모든 페이지, 선언된 서버 query와 로컬 Body filter |
| `list(conn.network.subnet_pools(...))` | `AllRecords(ctx, options...)` | 목록 수집; 오류가 나면 부분 결과를 반환하지 않음 |

이미 제공되는 native Create/Delete/Get과 별개로 이 세 연산은 느슨한 Source Resource의
속성·기본값을 보존합니다. native `Update`의 revision 옵션과 달리 `UpdateRecord`의
`if_revision`/`if_match` attrs는 일반 JSON Body 속성입니다. 고정 Python `SubnetPool`은
`NetworkResource`를 상속하지 않아 `If-Match` Header descriptor를 갖지 않습니다.
조건부 HTTP 갱신을 원하면 native API의 명시적 revision 기능을 선택합니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
pool = conn.network.find_subnet_pool("POOL_NAME_OR_ID")
rows = list(conn.network.subnet_pools(is_shared=False, default_quota=128))
if pool is not None:
    updated = conn.network.update_subnet_pool(pool.id, name="renamed", default_quota=0)
    # 전달할 attrs가 없으면 Source commit 역시 HTTP를 생략합니다.
    unchanged = conn.network.update_subnet_pool(pool.id)
```

## 독립 Go main

[설치 안내](../../docs/install.md)에 따라 모듈을 준비하고 다음 코드를 `main.go`로 저장합니다.
기본 `-action find`는 읽기만 합니다. update는 명시적인 action과 literal ID를 받습니다.

```sh
go run . -cloud dev -action find -pool POOL_NAME_OR_ID
go run . -cloud dev -action list
go run . -cloud dev -action update -pool POOL_ID -name renamed -quota 0
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    action := flag.String("action", "find", "find, list, update")
    pool := flag.String("pool", "", "find identity or update literal ID")
    name := flag.String("name", "", "optional update name")
    quota := flag.Int("quota", 0, "optional update quota; supply -quota to send zero")
    flag.Parse()
    sent := map[string]bool{}
    flag.Visit(func(f *flag.Flag) { sent[f.Name] = true })
    if (*action == "find" || *action == "update") && *pool == "" {
        log.Fatal("-pool is required")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    var rows []*subnetpools.SubnetPoolRecord
    switch *action {
    case "find":
        row, findErr := network.SubnetPools.FindRecord(ctx, *pool)
        err = findErr
        if row != nil { rows = append(rows, row) }
    case "list":
        rows, err = network.SubnetPools.AllRecords(ctx)
    case "update":
        options := []subnetpools.RecordMutationOption{}
        if sent["name"] { options = append(options, subnetpools.WithRecordAttribute("name", *name)) }
        if sent["quota"] { options = append(options, subnetpools.WithRecordAttribute("default_quota", *quota)) }
        row, updateErr := network.SubnetPools.UpdateRecord(ctx, *pool, options...)
        err = updateErr
        if row != nil { rows = append(rows, row) }
    default:
        log.Fatal("select find, list or update")
    }
    if err != nil { log.Fatal(err) }
    for _, row := range rows {
        raw, marshalErr := json.MarshalIndent(row.Resource.Body, "", "  ")
        if marshalErr != nil { log.Fatal(marshalErr) }
        fmt.Printf("HTTP %d\n%s\n", row.StatusCode, raw)
    }
}
```

## 옵션과 관측 값

`WithRecordAttribute`는 canonical 이름과 wire 이름을 받습니다. 예를 들어
`maximum_prefix_length`는 `max_prefixlen`으로 전송합니다. 이후 옵션이 같은 속성을
덮어쓰며, 명시한 null/false/0/빈 문자열/빈 배열과 알려지지 않은 확장 속성은 그대로 남습니다.
기본 `is_default`/`shared`를 만들지 않습니다. Bulk `WithRecordMutationOpts`의 map은
순서가 없으므로 충돌하는 canonical/wire 키를 정렬해서 적용합니다. 정확한 입력 순서가
필요하면 개별 `WithRecordAttribute`를 사용합니다.

`SubnetPoolRecord.Resource.Body`는 canonical Source descriptor view입니다. 빠진 Body
속성은 null, 빠진 tags는 `[]`이며 명시한 tags null은 그대로 유지합니다. project_id가
빠졌을 때만 tenant_id로 보완합니다. 문자열 숫자의 Unicode digit/int 변환, float의 truncation,
bool·list의 Source 변환도 적용합니다. 알려지지 않은 JSON 숫자는 RawMessage로 보존합니다.
`Wire.Body`는 관측한 row 자체입니다. member의 sparse 응답은 요청 seed와 합쳐 Resource를
만들지만 Wire에는 seed를 넣지 않습니다. `Envelope`, `Header`, `StatusCode`는 실제 전체
member 응답 또는 목록 페이지이며 서로 독립된 사본입니다. HTTP를 생략한 update는 상태 0이고
Wire/Envelope/Header가 없습니다. 빈/문법상 잘못된 accepted member JSON은 Source처럼 seed를
유지하되 실제 body를 보존합니다. 유효 JSON의 잘못된 객체 shape와 잘못된 UTF-8은 오류입니다.

`WithRecordFindQuery`는 원래 kwargs를 member query/seed로 사용하고 목록 단계에서 별도
분류합니다. `WithRecordListQuery`의 선언되지 않은 kwargs는 Source처럼 버립니다.
`is_shared`→`shared`, `any_tags`→`tags-any` 등은 서버 query이며 canonical 키가 wire alias보다
우선합니다. `default_quota`, `id`, `tenant_id`, `prefixes` 등 서버 mapping이 없는 **canonical**
Body 속성은 로컬 filter입니다. 로컬 비교는 Source dict subset 및 Python bool/number equality를
사용합니다. wire Body alias `default_prefixlen`은 list filter로 해석하지 않습니다.
`fields` 목록 query의 값은 그대로 전송하며 `GetWithFields`의 별도 field-name 변환과 다릅니다.
unknown Find query는 GET에서 전송되지만 목록 단계에서는 버립니다. `all_projects`는 Source
find의 소비된 인수로 HTTP에 넣지 않으며 권한을 늘리지 않습니다.

`WithRecordListLimit/Marker/MaxItems/Paginated`와 같은 Find 옵션은 목록 단계에만 적용합니다.
기본은 전체 페이지이며 cap은 로컬 filter 전에 받은 raw row 수를 셉니다. nonzero limit에서는
짧은 페이지에도 실제 wire ID marker로 계속 검색합니다. body/HTTP Link와 top-level next를
읽으며, owned pager는 표준 `links: {next: URL}`도 허용합니다. pinned Source의 dict 변환 루프는
이 형태를 따라가지 못하므로 이 지원은 명시적인 Go 개선입니다. 충돌·순환·다른 origin 링크는
증거를 가진 오류로 판정합니다. iterator를 중단하면 다음 페이지를 요청하지 않습니다.

`WithRecordFindIgnoreMissing(false)`는 완전히 성공한 검색에서 누락을 오류로 반환합니다.
후행 페이지의 중복 이름·오류나 Read/Close/cancel/Source 변화는 누락으로 숨기지 않습니다.
4xx/5xx에는 원래 Gophercloud 오류의 method/URL/body/header/status를 유지하고, accepted 응답
처리 실패는 `resource.ResponseError`에 실제 증거를 남깁니다. 해당 처리 실패는 재전송하지 않습니다.

Find의 `WithRecordFindMicroversion`은 member GET에만 적용됩니다. fallback LIST와 다음 페이지는
원래 선택한/default version을 사용하며, Header 옵션은 양 단계에 적용합니다. 이는 Source
`Resource.find`가 microversion을 fetch 인수로 소비하고 list에 전달하지 않는 계약입니다.
직접 `ListRecords`와 `UpdateRecord`의 Microversion 옵션은 각각 해당 호출에 적용합니다.
source를 callback 전에 캡처하고 재시도,
재인증, 물리 Read/Close 경계에서 검사합니다. protocol version을 바꾸거나 없던 version을 주입하는
재시도, auth/framing Header 주입, update JSON/Content-Type 변경은 차단하며 원래 오류를 보존합니다.
일반 retry Header와 새 인증 token은 native 정책을 유지합니다. bulk 옵션, raw 입력과 callback 결과는
snapshot을 소유하므로 같은 옵션을 병렬 호출에 재사용할 수 있습니다.

## 범위와 정책

[Neutron 27.0.0 SubnetPool 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnetpool.py)은
GET/list에 admin·해당 프로젝트 reader 또는 shared pool을, update에 admin·해당 프로젝트 member를
허용합니다. `update_subnetpool:is_default`는 admin 분기이며 이 예제에서는 설정하지 않습니다.
service 역할만으로 허용되는 별도 분기는 없습니다.
[neutron-lib 3.21.1 schema](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/subnetpool.py)는
shared, ip_version, tenant_id, id를 PUT 속성으로 허용하지 않습니다. address_scope_id는
[address-scope extension](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/address_scope.py)이
추가하는 user update 속성입니다. SDK는 raw Source attrs를 전송하고 배포 서버가 schema·정책을
판정합니다. fixture 통과를 실제 Neutron 배포의 권한/동작 검증으로 보지 않습니다.

이 대응은 literal ID/이름 및 JSON 속성을 위한 세 callable 계약입니다. Python의 mutable Resource
cache, arbitrary 객체·session/Connection/location 계산과 런타임 deprecation warning까지 재현하지
않습니다. location은 null입니다. SDK가 소유한 constructor/control 인수는 JSON 확장 키로 받아
route·class를 바꾸지 않으며 concrete Go 옵션으로 처리합니다. Map 기반 로컬 filter의 평가 순서는
정렬됩니다. Python/Go float repr·Unicode runtime 및 CPython int-string 제한 차이는 공통 변환
범위에 남습니다. native typed 모델과 Record를 혼합해 전 SDK parity가 완료됐다고 집계하지 않습니다.
