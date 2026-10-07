# cloud Floating IP 가용 후보 조회·할당

`conn.AvailableFloatingIP`는 cloud의 floating IP source에 따라 **첫 free IP를 반환하거나 새 IP를 할당**합니다. Connection이 필요한 Neutron/Compute endpoint, 인증 scope와 공유 network 역할을 연결하므로 애플리케이션에서 builder나 resolver를 구현할 필요가 없습니다.

free IP를 반환하는 것은 연결이나 예약이 아닙니다. Neutron의 optional `Server`는 새 allocation에서만 사용하고, Nova는 server를 사용하지 않습니다. 이 API는 IP/server ACTIVE 대기, Nova add action, raw 서버 주소 관측을 하지 않습니다. 서버에 IP를 연결하는 작업은 [IP helper](server-ip-helpers.md) 또는 [Ensure·readiness](server-ip-dispatch.md)를 사용합니다.

## 공개 API와 옵션

`conn.AvailableFloatingIP(ctx, request, options...)`와 `service.AvailableFloatingIP(ctx, request, options...)`는 같은 `(*compute.AvailableFloatingIPResult, error)`를 반환합니다. `service`는 `*compute.Service`입니다. Connection entry는 backend가 필요한 시점에 서비스를 발견하므로 Neutron free 재사용을 위해 Compute를 먼저 요청하지 않습니다.

| 요청·옵션 | 의미 |
|---|---|
| `compute.AvailableFloatingIPRequest{Networks: []resource.Ref{...}, Server: ...}` | Network 요청 타입의 alias. Server는 optional |
| `compute.WithAvailableIPSource(compute.FloatingIPNeutron)` | per-call backend source override. 생략하면 Connection의 server-address/cloud source 설정 사용 |
| `compute.WithAvailableIPSource(compute.FloatingIPNova)` | standalone legacy Nova pool/list/allocation 경로 |
| `compute.WithAvailableIPSource(compute.FloatingIPNone)` | 이 explicit getter에서는 Nova 경로. 자동 needs의 source=None skip와 구분 |
| `compute.WithAvailableIPNetworkOptions(...)` | Neutron의 concrete Available 옵션을 준비하여 전달. 여러 그룹은 순서대로 적용 |
| `compute.WithAvailableIPLocation(location)` | 반환 view에 쓸 `resource.CloudLocation` 전체 facts override. 생략하면 Connection의 cloud/auth 기록 사용 |
| `compute.WithAvailableIPStrict(true)` | Nova 정규화의 호환 alias·extra top-level 복원 제외. 기본 false; properties와 Wire는 보존 |
| `compute.WithAvailableIPTimeout(time.Minute)` | backend 발견·network 선택·fallback·목록·allocation·Nova compat GET 전체의 SDK deadline |
| `compute.WithUnlimitedAvailableIPTimeout()` | SDK 전체 deadline 해제. 부모 context·transport·Neutron 개별 timeout은 유지 |

Neutron에서는 `Networks`의 caller 순서가 우선입니다. 각 typed `Name`/`ID`는 공유 external-floating 역할의 이름/ID로만 exact 비교합니다. 첫 matching network에 free IP가 없으면 그 network에 할당합니다. 빈 목록은 첫 floating 역할, 그 역할이 없으면 enabled router의 첫 external gateway를 사용합니다. [Neutron lower 계약](../network/floating-ip-available.md)에 scope/null·port·allocation 정책을 설명합니다.

Nova에서는 `Networks`가 zero 또는 **한 개**여야 합니다. 한 개의 `Name`/`ID` 문자열 자체를 literal pool로 사용하며 Neutron network로 해석하지 않습니다. 빈 목록이면 `/os-floating-ip-pools`의 첫 name을 사용합니다. Neutron 후보가 여러 개였더라도 Nova로 fallback한 뒤 여러 값을 pool로 직렬화하지 않고 `ErrInvalidOption`을 반환합니다.

Neutron fixed/NAT/project 옵션은 `WithAvailableIPNetworkOptions(network.WithAvailableFixedAddress(...), network.WithAvailableNATDestination(...), network.WithAvailableProject(...))`로 전달합니다. 이 그룹은 공통 preparation에서 한 번 검증하지만 Nova에서는 적용하지 않습니다. Nova available은 API가 반환한 목록에서 같은 pool의 첫 free IP를 선택하며 별도 project filter를 적용하지 않습니다. Neutron project filter·fixed/NAT 목적지·allocation association을 합성하지 않습니다.

## 독립 Go 예제

SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장합니다. `-source`를 생략하면 cloud 설정을 사용하며 `neutron`, `nova`, `none`으로 호출별 source를 지정할 수 있습니다. `-strict`는 Nova 반환 view의 호환 alias를 생략합니다. location은 Connection의 cloud/auth 기록을 사용합니다. `-network`는 Neutron의 정확한 이름 또는 Nova의 literal pool입니다. 빈 값이면 backend의 default 선택을 사용합니다. 이 프로그램은 실제 Available 작업을 호출하므로 free 후보가 없으면 새 IP를 생성할 수 있습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/compute"
    "gophercloudsdk/network"
    "gophercloudsdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    source := flag.String("source", "", "cloud default, neutron, nova or none")
    pool := flag.String("network", "public", "network name or literal Nova pool")
    server := flag.String("server-id", "", "optional server for new Neutron allocation")
    project := flag.String("project", "", "optional Neutron reuse filter only")
    strict := flag.Bool("strict", false, "omit Nova view compatibility aliases")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *source, *pool, *server, *project, *strict); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, source, pool, serverID, project string, strict bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud),
        sdk.WithMicroversion(sdk.Compute, "2.35"))
    if err != nil {
        return err
    }
    input := compute.AvailableFloatingIPRequest{}
    if pool != "" {
        input.Networks = []resource.Ref{resource.Name(pool)}
    }
    if serverID != "" {
        input.Server = resource.ID(serverID)
    }
    options := []compute.AvailableFloatingIPOption{
        compute.WithAvailableIPTimeout(time.Minute),
        compute.WithAvailableIPStrict(strict),
    }
    if source != "" {
        options = append(options, compute.WithAvailableIPSource(compute.FloatingIPSource(source)))
    }
    if project != "" {
        options = append(options, compute.WithAvailableIPNetworkOptions(
            network.WithAvailableProject(project)))
    }
    result, err := conn.AvailableFloatingIP(ctx, input, options...)
    if result != nil {
        summary := map[string]any{
            "backend": result.Backend, "id": result.ID, "address": result.Address,
            "reused": result.Reused, "allocated": result.Allocated,
        }
        if record := result.FloatingIP; record != nil {
            summary["normalization_source"] = record.NormalizationSource
            summary["normalized"] = record.Normalized
            if record.Resource != nil {
                summary["resource"] = record.Resource.Body
            }
            if record.Wire != nil {
                summary["wire"] = record.Wire.Body
            }
        }
        if result.FallbackError != nil {
            summary["fallback_error"] = result.FallbackError.Error()
        }
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        if encodeErr := encoder.Encode(summary); encodeErr != nil {
            return errors.Join(err, encodeErr)
        }
    }
    return err
}
```

이 예제는 Nova fallback에도 호환되도록 selected Compute microversion2.35를 명시합니다. Neutron free 재사용만 수행하면 이 설정 때문에 Compute endpoint를 조회하지 않습니다. Nova endpoint를 실제 제공하는 cloud가 필요하며, SDK가 선택된 버전을 몰래 낮추지 않습니다. 초기 Connect는 Available 시간 예산 밖에 있고 부모 context가 제한합니다. Network/Compute endpoint 발견부터 Available 내부의 1분 예산을 적용합니다.

## backend 선택과 fallback

- source=Neutron이면 lazy Network를 발견하여 [직접 lower Available](../network/floating-ip-available.md)을 실행합니다. 재사용 free IP에서는 서버 이름·ports·Compute를 조회하지 않습니다. 새 allocation에서만 optional server 이름과 port/fixed IPv4를 사용합니다.
- 정확한 missing Network catalog endpoint는 Nova를 선택합니다. 단일 wrapped 부재와 다른 원인이 Join된 catalog 오류를 구분하며, mixed failure는 원인을 유지하고 Nova로 진행하지 않습니다.
- Neutron lower가 **결과 없이 순수 NotFound**로 종료하면 Nova로 fallback합니다. 명시 external network 무일치·default external network 부재와 HTTP404도 포함할 수 있습니다. 이때 `FallbackError`에 원래 오류를 남깁니다. clean catalog absence 자체는 lower NotFound가 아니어서 이 필드가 nil일 수 있습니다.
- Neutron allocation/model/응답이 알려진 결과가 있거나 read/Close·decode·source·취소·여러 cause 또는 terminal SDK 정책 오류이면 fallback하지 않습니다. RetryFunc가404를 성공 코드에 추가해도 SDK의 목록200 정책 실패를 Nova 전환으로 숨기지 않습니다. 403/204/malformed 결과를 free 후보 없음으로 숨기지 않습니다.
- source=Nova/None이면 Neutron 역할·owner를 선조회하지 않고 Compute legacy 경로를 사용합니다. None은 이 standalone API를 disabled skip로 바꾸지 않습니다.

Nova 경로는 raw 목록에 `instance_id: null` → literal pool 순서로 필터를 적용하고, 모든 matching row를 정규화한 뒤 첫 결과를 반환합니다. 빈 문자열 instance ID는 null과 다르며, 제외된 행의 ID·주소는 정규화하지 않습니다. 첫 행이 정상이어도 뒤 matching row의 정규화가 실패하면 오류와 실제 목록 응답을 반환합니다. 조회 행의 null ID·비 IPv4 주소·확장 값을 연결용 ID/IPv4 verifier로 거부하지 않습니다.

clean Nova 목록404는 빈 후보로 처리하여 fresh POST와 mandatory compatibility GET을 진행합니다. 명시 pool은 lookup 없이 사용하고, default 첫 name은 목록보다 먼저 조회하며 보통 fresh에서 재조회하지 않습니다. default name이 null이면 Python None처럼 Create의 기본 pool 조회를 다시 수행합니다. 목록200의 malformed/accepted 처리 오류와403 등은 빈 후보로 숨기지 않습니다. 생성·호환 조회는 [독립 Create](floating-ip-create.md)의 raw 응답 엔진을 사용하고 접수된 HTTP200..399를 처리합니다. 호환 GET의 실제 반환 ID·주소·pool이 달라도 passive cloud view에 보존하며 이미 접수된 allocation의 증거와 분리합니다. server action·관측·wait·cleanup은 수행하지 않습니다.

selected Compute2.36 이상은 이 legacy IP/pool 경로를 사용할 수 없어 `ErrUnsupported`입니다. 버전 정책과 raw Nova 모델은 [Nova backend 가이드](server-nova-floating-ip.md)의 기반을 사용하지만, 이 Available entry는 association 소비자를 호출하지 않습니다.

## optional server·scope·부분 결과

Neutron free 선택은 network/current project/port null만 검사합니다. status나 주소 family를 추가 필터하지 않으므로 ERROR 상태나 IPv6 주소가 반환될 수 있습니다. `Server`가 주어져도 reused free IP를 PUT하지 않습니다. 새 allocation만 서버 port 목록에서 목적지를 선택하며 no-ports 또는 명시 fixed 무일치에서는 unattached로 할당할 수 있습니다. 여러 port는 NAT network로 좁힌 뒤 최근 `created_at`의 첫 유효 IPv4를 선택합니다. 이 policy를 기존 Ensure의 strict unique 선택과 혼동하지 않습니다.

기본 Neutron reuse project는 이미 기록된 Keystone scope이며 scope를 새로 조회·추측하지 않습니다. 없는 scope는 null 필터이고 빈 문자열과 다릅니다. `WithAvailableProject`는 **reuse filter만** 바꾸며 새 POST는 project_id override 없이 인증 scope에 할당합니다. 이 옵션은 Nova의 API project 범위를 바꾸지 않습니다.

| 결과 field | 의미 |
|---|---|
| `Backend` | 현재 선택한 Neutron/Nova branch. 준비 실패에서 이 값만으로 HTTP 실행을 입증하지 않음 |
| `ID`, `Address` | 알려진 실제 모델의 string 편의 값. permissive raw 값이 typed 모델로 표현되지 않으면 비어 있을 수 있으므로 `FloatingIP.Resource/Wire` 확인 |
| `Reused`, `Allocated` | free 후보 반환 또는 새 POST 접수. 연결·예약·ACTIVE 성공을 뜻하지 않음 |
| `FloatingIP` | `*compute.FloatingIPRecord`; 공개 Resource view와 실제 Wire를 분리. view 오류에서는 Wire만 남을 수 있음 |
| `Neutron` | `*network.FloatingIPAvailability`; native 모델, raw Metadata 및 allocation Envelope/header/status |
| `Nova` | `*compute.NovaFloatingIPAvailability`; optional typed IP와 allocation response, `Inventory`·`PoolQuery`·`Creation` |
| `FallbackError` | 결과 없이 Neutron pure NotFound에서 Nova를 선택한 원인. 성공 반환과 동시에 남을 수 있음 |

allocation 접수 뒤 응답 검증·read/Close·후속 GET·source·취소 오류가 발생하면 알려진 모델과 실제 receipt를 result 옆에 보존합니다. 성공과 실패에서 실제 backend를 읽고, error가 있어도 `Allocated`와 모델/응답 증거를 확인합니다. 오류가 있는 partial을 유효한 다음 mutation 대상으로 간주하지 않습니다. 이미 할당한 자원을 자동 삭제하거나 다른 backend/network로 재할당하지 않습니다.

Nova raw 모델에는 합성 IP ACTIVE나 합성 InstanceID/association을 넣지 않습니다. canonical 주소 키의 present-null 우선순위와 raw JSON의 누락/null·큰 정수·extension은 `Metadata.Body`와 `FloatingIP.Wire.Body`에 남습니다. Neutron의 native string 필드는 null과 누락을 빈 문자열로 보일 수 있으므로 원래 의미가 필요하면 Wire를 확인합니다.

## Resource·Wire·location·strict

`result.FloatingIP.Resource`는 공개 반환용 owned view이고 `Wire`는 이미 받은 실제 row입니다. view 변환을 위해 목록·단건을 다시 조회하거나 allocation을 다시 실행하지 않습니다. Neutron view는 누락된 known Body 기본값, name/project alias, tags/revision/port_details 변환과 location을 제공합니다. Neutron은 `Normalized=false`이며 strict 옵션으로 이 view의 alias를 제거하지 않습니다. Resource·Wire·lower 모델을 각각 수정해도 다른 view의 JSON bytes를 변경하지 않습니다.

Nova는 `Normalized=true`입니다. 직접 Nova/None은 Python과 같이 Resource의 canonical status ACTIVE를 합성합니다. 실제 Wire의 status가 DOWN이어도 view가 ACTIVE일 수 있으므로 이를 IP ACTIVE·연결·예약 증거로 사용하지 않습니다. Neutron 서비스 발견 후의 외부 fallback에서 실제 backend가 Nova여도 `NormalizationSource=FloatingIPNeutron`을 유지하고, attached는 port, missing status는 UNKNOWN 규칙을 사용합니다. `Backend`는 실제 반환 모델의 backend이며 정규화 규칙과 구분합니다.

`WithAvailableIPStrict(true)`는 Nova view의 port_id/router_id/project_id/tenant_id/floating_network_id 호환 alias와 properties의 extra top-level 복원만 생략합니다. properties 객체 자체와 unknown·nullable Wire는 보존합니다. 기본 false에서는 두 위치에 extension을 읽을 수 있습니다. canonical present-null은 legacy alias보다 우선합니다. Nova raw status 등 소비되지 않은 key는 properties에 남을 수 있으며 canonical 합성 status를 덮어쓰지 않습니다.

location은 Connection의 기록된 cloud·region·project facts로 구성하고 알 수 없는 값은 null로 둡니다. `WithAvailableIPLocation(location)`은 `resource.CloudLocation` 전체 값을 owned snapshot으로 지정합니다. 이 옵션은 인증 scope나 Neutron reuse project를 바꾸지 않습니다. `WithAvailableIPNetworkOptions(network.WithAvailableProject(...))`의 reuse filter와 구분합니다. 옵션 생성 후 원본 location의 pointer·JSON을 변경해도 준비된 값은 바뀌지 않습니다.

location·descriptor·정규화 오류 또는 source/context 종료로 성공한 view를 만들 수 없으면 `FloatingIP.Resource`는 nil일 수 있습니다. 이미 받은 `Wire`와 알려진 raw backend 모델·allocation receipt는 보존하며 원래 오류와 함께 반환합니다. view 오류 때문에 다른 backend로 fallback하거나 새 allocation을 추가하지 않습니다. accepted 응답 처리 오류가 있어도 독립적으로 변환 가능한 Resource가 남을 수 있으므로 nil 여부만으로 작업 성공을 판단하지 않습니다.

## 준비한 옵션과 시간 정책

outer Compute 옵션과 nested Network 옵션은 한 번 준비하고 fallback에서 다시 application closure를 실행하지 않습니다. network 후보와 옵션 목록은 적용 전에 복사하고 source/location/Network 옵션은 각 적용 단계에서 owned snapshot으로 유지합니다. 각 callback 전후 source·context를 검사하며 준비 중 교체·취소가 발생하면 뒤 callback과 HTTP를 실행하지 않습니다. 작업 중 source·endpoint·version 변화도 오류로 종료합니다. 공통 유효성 검증은 실제 backend에서 사용하지 않는 Network 옵션에도 적용됩니다.

기본 SDK deadline은 없습니다. `WithAvailableIPTimeout`과 더 이른 parent deadline은 backend discovery부터 fallback/Nova compat GET까지 같은 context를 제한하며 단계마다 다시 시작하지 않습니다. nested `network.WithAvailableTimeout`은 Neutron lower 단계에만 더 짧은 제한을 줄 수 있고 Nova에 적용하지 않습니다. timeout/취소가 Neutron NotFound fallback으로 변환되지 않습니다. unlimited 옵션은 SDK 제한만 제거하고 부모 context 및 transport timeout을 없애지 않습니다.

## Python 고정 소스와 남은 범위

```python
import openstack

conn = openstack.connect(cloud="dev")
ip = conn.available_floating_ip(network="public")
print(ip["id"], ip["floating_ip_address"])

server = conn.compute.find_server("web-01", ignore_missing=False)
ip = conn.available_floating_ip(network="public", server=server)
# Neutron free 재사용은 연결하지 않으며 Nova available는 server를 사용하지 않음.
```

비교 pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`다. [public Available·backend free 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L773), [Nova allocation+compat GET](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975), [normalization](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1732-L1830)을 대조한다.

Python public Available의 인자는 network/server뿐이며 자체 wait/reuse/timeout 인자가 없다. Go의 concrete project/fixed/NAT/time 옵션은 추가 기능이다. Python의 network 문자열 name OR ID 및 dynamic list 입력과 Go typed Ref/단일 Nova pool 정책은 같은 입력 표면이 아니다. Go는 source None의 standalone Nova 선택·Neutron free 재사용과 optional allocation server·NotFound fallback을 제공하지만 partial receipt와 joined 원인을 보존하며 일부 fallback의 오류 처리 차이가 있다.

Python Neutron 성공은 network FloatingIP Resource를 그대로 반환한다. Go의 `FloatingIP`는 SDK 소유 반환 view로 location·기본값·Nova strict aliases/properties를 제공하고 실제 backend 모델과 Wire를 함께 보존한다. Nova view의 합성 ACTIVE와 configured-Neutron fallback의 정규화 규칙은 실제 접수/연결 증거와 별도로 읽는다. 이 값 모델은 Python mutable Resource의 inherited fetch/commit/session 전체를 제공하지 않는다.

Available의 named 매핑에는 Neutron 내부 목록 분기 하나가 남습니다. Python은 unfiltered Neutron list404에서 configured-source로 정규화한 Nova 목록을 Neutron network/project/null-port로 필터한 뒤 필요하면 Neutron allocation을 수행하지만, 현재 getter는 외부 Nova 경로로 직접 전환합니다. 직접/외부 Nova의 raw 필터→전체 matching 정규화와 clean list404 이후 fresh POST·compat GET은 구현·검증했습니다. 전체 API 승격은 남은 Neutron 분기를 완료한 뒤 판정합니다.

cloud has_service/version/config 전체, configured API GET cache와 mutable Resource/session은 별도 전체 SDK 범위다. [기존 Compute IP consumer](server-ip-dispatch.md)의 연결·wait 및 [독립 Create](floating-ip-create.md)·[Delete](floating-ip-delete.md)·[조회](floating-ip-queries.md)의 판정은 각 선언에서 추적하며 이 Available 반환 view의 완료 여부와 합산하지 않는다.

Python 비교는 고정 source 정적 확인이다. 위 Python 예제나 인증된 OpenStack 실행의 확인을 뜻하지 않는다. 실제 HTTP fixture·정확한 독립 main 컴파일·최종 revision gate 결과는 확인된 근거만 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록한다.

전체 IP 목록·검색·단건과 pool 조회는 [Floating IP query 가이드](floating-ip-queries.md)의 별도6개 API를 사용합니다. Available은 같은 `FloatingIPRecord`의 Resource·Wire와 Query/Create 엔진을 재사용하며 free-first 선택 또는 allocation을 수행합니다. Nova read에 association용 ID·IPv4 verifier를 적용하지 않습니다.

Nova의 `Inventory`는 matching 전체 view와 실제 목록 Pages·Failure·SuppressedNotFound를 보존합니다. `Creation`은 초기 Allocation/AllocationResponse와 mandatory Compatibility의 응답·오류를 분리합니다. 반환 `FloatingIP`를 수정해도 이 증거는 바뀌지 않습니다. `Nova.FloatingIP`은 표현 가능한 행에서만 제공하는 독립된 기존 typed projection이며 null/복합 raw 값의 성공을 막지 않습니다. 새 분기 검증은 [기존 fixture를 재사용한 4개 그룹](../connection_floating_ip_available_nova_test.go)에 있습니다.
