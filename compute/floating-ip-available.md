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
| `compute.WithAvailableIPTimeout(time.Minute)` | backend 발견·network 선택·fallback·목록·allocation·Nova compat GET 전체의 SDK deadline |
| `compute.WithUnlimitedAvailableIPTimeout()` | SDK 전체 deadline 해제. 부모 context·transport·Neutron 개별 timeout은 유지 |

Neutron에서는 `Networks`의 caller 순서가 우선입니다. 각 typed `Name`/`ID`는 공유 external-floating 역할의 이름/ID로만 exact 비교합니다. 첫 matching network에 free IP가 없으면 그 network에 할당합니다. 빈 목록은 첫 floating 역할, 그 역할이 없으면 enabled router의 첫 external gateway를 사용합니다. [Neutron lower 계약](../network/floating-ip-available.md)에 scope/null·port·allocation 정책을 설명합니다.

Nova에서는 `Networks`가 zero 또는 **한 개**여야 합니다. 한 개의 `Name`/`ID` 문자열 자체를 literal pool로 사용하며 Neutron network로 해석하지 않습니다. 빈 목록이면 `/os-floating-ip-pools`의 첫 name을 사용합니다. Neutron 후보가 여러 개였더라도 Nova로 fallback한 뒤 여러 값을 pool로 직렬화하지 않고 `ErrInvalidOption`을 반환합니다.

Neutron fixed/NAT/project 옵션은 `WithAvailableIPNetworkOptions(network.WithAvailableFixedAddress(...), network.WithAvailableNATDestination(...), network.WithAvailableProject(...))`로 전달합니다. 이 그룹은 공통 preparation에서 한 번 검증하지만 Nova에서는 적용하지 않습니다. Nova available은 API가 반환한 목록에서 같은 pool의 첫 free IP를 선택하며 별도 project filter를 적용하지 않습니다. Neutron project filter·fixed/NAT 목적지·allocation association을 합성하지 않습니다.

## 독립 Go 예제

SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장합니다. `-source`를 생략하면 cloud 설정을 사용하며 `neutron`, `nova`, `none`으로 호출별 source를 지정할 수 있습니다. `-network`는 Neutron의 정확한 이름 또는 Nova의 literal pool입니다. 빈 값이면 backend의 default 선택을 사용합니다. 이 프로그램은 실제 Available 작업을 호출하므로 free 후보가 없으면 새 IP를 생성할 수 있습니다.

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
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *source, *pool, *server, *project); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, source, pool, serverID, project string) error {
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

Nova 경로는 같은 pool의 `instance_id: null` free IP를 재사용하며 없으면 POST200 allocation 뒤 같은 ID의 compat GET200을 수행합니다. 빈 문자열 instance ID는 free가 아니고 선택한 row의 ID·IPv4·pool·association metadata를 확인합니다. server를 생략하거나 지정해도 addFloatingIp/action 및 server GET·주소 관측은 수행하지 않습니다. Nova 목록/allocate/read의 성공 코드는200이고204를 free 후보 부재로 취급하지 않습니다. legacy API404는 알려진 오류이며 Python처럼 목록404를 빈 후보로 바꾸어 allocation을 추가 시도하지 않습니다.

selected Compute2.36 이상은 이 legacy IP/pool 경로를 사용할 수 없어 `ErrUnsupported`입니다. 버전 정책과 raw Nova 모델은 [Nova backend 가이드](server-nova-floating-ip.md)의 기반을 사용하지만, 이 Available entry는 association 소비자를 호출하지 않습니다.

## optional server·scope·부분 결과

Neutron free 선택은 network/current project/port null만 검사합니다. status나 주소 family를 추가 필터하지 않으므로 ERROR 상태나 IPv6 주소가 반환될 수 있습니다. `Server`가 주어져도 reused free IP를 PUT하지 않습니다. 새 allocation만 서버 port 목록에서 목적지를 선택하며 no-ports 또는 명시 fixed 무일치에서는 unattached로 할당할 수 있습니다. 여러 port는 NAT network로 좁힌 뒤 최근 `created_at`의 첫 유효 IPv4를 선택합니다. 이 policy를 기존 Ensure의 strict unique 선택과 혼동하지 않습니다.

기본 Neutron reuse project는 이미 기록된 Keystone scope이며 scope를 새로 조회·추측하지 않습니다. 없는 scope는 null 필터이고 빈 문자열과 다릅니다. `WithAvailableProject`는 **reuse filter만** 바꾸며 새 POST는 project_id override 없이 인증 scope에 할당합니다. 이 옵션은 Nova의 API project 범위를 바꾸지 않습니다.

| 결과 field | 의미 |
|---|---|
| `Backend` | 현재 선택한 Neutron/Nova branch. 준비 실패에서 이 값만으로 HTTP 실행을 입증하지 않음 |
| `ID`, `Address` | 해당 backend의 알려진 실제 모델에서 읽은 공통 값. model 미확인 allocation에서는 비어 있을 수 있음 |
| `Reused`, `Allocated` | free 후보 반환 또는 새 POST 접수. 연결·예약·ACTIVE 성공을 뜻하지 않음 |
| `Neutron` | `*network.FloatingIPAvailability`; native 모델, raw Metadata 및 allocation Envelope/header/status |
| `Nova` | `*compute.NovaFloatingIPAvailability`; actual raw Nova IP와 allocation response |
| `FallbackError` | 결과 없이 Neutron pure NotFound에서 Nova를 선택한 원인. 성공 반환과 동시에 남을 수 있음 |

allocation 접수 뒤 응답 검증·read/Close·후속 GET·source·취소 오류가 발생하면 알려진 모델과 실제 receipt를 result 옆에 보존합니다. 성공과 실패에서 실제 backend를 읽고, error가 있어도 `Allocated`와 모델/응답 증거를 확인합니다. 오류가 있는 partial을 유효한 다음 mutation 대상으로 간주하지 않습니다. 이미 할당한 자원을 자동 삭제하거나 다른 backend/network로 재할당하지 않습니다.

Nova 모델에는 합성 IP ACTIVE나 합성 InstanceID/association을 넣지 않습니다. canonical 주소 키의 present-null 우선순위와 raw JSON의 누락/null·큰 정수·extension은 `Metadata.Body`에 남습니다. Neutron의 native string 필드는 null과 누락을 빈 문자열로 보일 수 있으므로 원래 의미가 필요하면 lower `Metadata.Body`를 확인합니다.

## 준비한 옵션과 시간 정책

outer Compute 옵션과 nested Network 옵션은 한 번 준비하고 fallback에서 다시 application closure를 실행하지 않습니다. network 후보 slice는 options 전에 복사합니다. 준비 중 service/client source 교체와 작업 중 source·endpoint·version 변화는 오류로 종료합니다. 공통 유효성 검증은 실제 backend에서 사용하지 않는 Network 옵션에도 적용됩니다.

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

Python Neutron 성공은 network FloatingIP Resource를 그대로 반환한다. Nova path의 normalizer는 configured source를 다시 확인하므로 configured-Neutron fallback은 실제 Nova data에도 Neutron식 status/attached를 적용할 수 있다. Go는 **실제 backend 모델**을 유지하고 Nova ACTIVE·association을 합성하지 않는다. Python location/project·strict aliases·properties·nullable aliases·mutable Resource의 inherited fetch/commit/session·query semantics를 이 result 구조만으로 완료하지 않는다.

Python의 unfiltered list 내부404→Nova와 바깥 Available fallback이 중첩되는 순서는 Go의 직접 backend 전환과 다르다. cloud has_service/version/config 전체, configured API GET cache, 모든 함수별 fallback·lookup/cleanup/error 정책, standalone pool/IP CRUD·delete verification 및 full Resource/session 표면은 계속 비교할 범위다. [기존 Compute IP consumer](server-ip-dispatch.md)의 연결·wait나 [Python의 별도 public create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L775-L842)의 주변 계약을 이 Available 구현만으로 완료 처리하지 않는다.

Python 비교는 고정 source 정적 확인이다. 위 Python 예제나 인증된 OpenStack 실행의 확인을 뜻하지 않는다. 실제 HTTP fixture·정확한 독립 main 컴파일·최종 revision gate 결과는 확인된 근거만 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록한다.

전체 IP 목록·검색·단건과 pool 조회는 [Floating IP query 가이드](floating-ip-queries.md)의 별도6개 API를 사용합니다. 이 Available entry는 free-first 재사용 또는 allocation이며, query의 nullable row·Nova 논리 정규화 view와 반환 계약이 다릅니다.
