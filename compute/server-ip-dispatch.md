# 서버의 pool·순차 Floating IP·자동 선택

기존 `PlanServerFloatingIP`, `EnsureServerFloatingIP`, `GetActiveServer`, `WaitForServer`, `CreateWithAutomaticFloatingIP`에서 같은 IP 선택 옵션을 사용합니다. Connection이 인증·cloud 설정·서비스 연결·공유 네트워크 역할을 준비하므로 애플리케이션에서 builder나 resolver를 구현할 필요가 없습니다.

독립 [AddIPsToServer·AddIPList](server-ip-helpers.md)도 같은 선택 기반을 사용합니다. 해당 entry는 기본60초·비동기이고 wait는 ACTIVE 조건 없이 raw 주소만 관측하므로 이 문서의 기존 Ensure/ready/create 정책과 구분합니다.

## 선택 순서와 입력

| 입력 | 선택되는 작업 |
|---|---|
| `compute.WithFloatingIPPool(resource.Name("public"))` | Neutron 외부 network 이름; Nova는 literal pool 값으로 재사용/할당 |
| `compute.WithFloatingIPPool(resource.ID("external-id"))` | Neutron 외부 network ID; Nova는 문자열 자체를 pool 값으로 사용 |
| `compute.WithFloatingIPAddresses("198.51.100.10", "198.51.100.11")` | 이미 존재하는 두 IP를 제공한 순서대로 연결 |
| 둘 다 생략 | 기존 자동 needs/skip 정책 적용 |

선택 순서는 **nonzero pool > nonempty IP 목록 > auto**이며 옵션을 넣은 순서와 독립입니다. 각 selector의 마지막 옵션이 자신의 값을 교체합니다. `WithFloatingIPPool(resource.Ref{})`은 pool을 지우고, 인자 없는 `WithFloatingIPAddresses()`는 목록을 지웁니다. pool과 목록은 서로를 지우지 않으므로 pool을 지운 뒤 기존 목록을 사용할 수 있습니다.

pool/IP 선택에서는 최종 winning selector만 검증합니다. 공통 address/Ensure/wait 옵션의 유효성 검증은 계속 수행합니다. 유효한 pool이 있으면 무시되는 IP 문자열을 검사하지 않습니다. IP 목록이 이기면 각 원소가 IPv4여야 하며 순서와 중복을 유지합니다. 목록은 옵션 생성·준비 시 복사합니다. IPv4 주소를 floating IP 이름이나 ID로 추측하지 않고, 기존 IP의 `floating_ip_address`를 정확히 조회합니다.

`AutomaticFloatingIPRequest.Network`와 생성 옵션의 `FloatingIPNetwork`는 기존 **automatic 분기의 allocation network**이며 Nova에서는 literal pool 값입니다. 명시 pool은 이를 대체하고, Neutron 명시 IP 목록은 각 IP에 기록된 floating network를 사용합니다. Nova는 IP ID·주소·pool·instance association을 재검증합니다. 이 입력들은 서버 NIC network를 바꾸지 않습니다.

pool/IP 목록은 `WithAutomaticIPEnabled(false)`, private cloud, 기존 public/floating 주소 등의 **자동 skip 조건을 우회**합니다. 서버 readiness 계약은 계속 적용됩니다. GetActiveServer의 non-ACTIVE는 nil,nil이고 ERROR나 ACTIVE 주소 미준비는 오류입니다. Ensure의 실제 연결은 supplied ACTIVE를 요구하며, Wait/Create는 같은 ID의 raw 현재 상태와 주소 준비를 먼저 기다립니다.

## 기존 API에 같은 옵션 전달하기

`opts`는 `[]compute.AutomaticFloatingIPOption`, `input`은 `compute.AutomaticFloatingIPRequest{Server: server}`로 구성합니다.

| 호출 | 완료 의미와 옵션 위치 |
|---|---|
| `conn.PlanServerFloatingIP(ctx, input, opts...)` | 읽기 전용 선택; Neutron 명시 IP는 ordered `Decision.AttachmentSelections`, pool은 `Decision.Selection`; Nova는 backend 진단 |
| `conn.EnsureServerFloatingIP(ctx, input, opts...)` | 각 assignment의 실제 Neutron IP ACTIVE와 exact raw Nova floating IPv4 관측까지 순차 완료 |
| `conn.GetActiveServer(ctx, input, compute.WithServerReadyAutomaticIPOptions(opts...))` | supplied 상태 판정 후 기본 비동기 접수; IP/raw 관측 대기 없음 |
| 위 Get에 `compute.WithActiveServerWait(true)` 추가 | Neutron IP ACTIVE와 raw Nova 관측도 요구 |
| `conn.WaitForServer(ctx, input, compute.WithServerReadyAutomaticIPOptions(opts...))` | raw 서버 readiness 후 Neutron IP ACTIVE·Nova 관측 필수 |
| `conn.CreateWithAutomaticFloatingIP(ctx, request, compute.AutomaticServerCreateOptions{AutomaticIP: opts, Server: serverOptions})` | 생성·raw readiness·같은 IP 선택과 순차 관측; 기존 부팅/NIC 옵션 사용 |

`service`의 같은 이름 메서드도 동일합니다. `service.Servers.WaitForServer`는 별도 collection 상태 대기로 IP dispatch를 수행하지 않습니다. 기존 `Servers.CreateWithFloatingIP`의 옵션과 반환 타입도 이 새 옵션 그룹으로 바뀌지 않습니다.

대상 선택은 `compute.WithAutomaticEnsureOptions(...)`로 공유합니다. 다음 port/NAT/project/revision·IP waiter 설명은 Neutron branch의 계약입니다. Nova는 fixed/reuse를 소비하며 port/NAT/project override를 거부합니다. `network.WithEnsurePort`, `WithEnsureNATDestination`, `WithEnsureFixedAddress`, `WithEnsureWait`가 pool/auto 및 명시 IP 목적지·대기에 적용됩니다. `WithEnsureProject`와 `WithEnsureReuse`는 pool/auto 재사용·할당에만 적용되며 명시 기존 IP의 검색을 현재 project/free IP로 제한하지 않습니다. 이미 다른 포트에 연결된 기존 IP도 Neutron 권한 아래 이동할 수 있습니다. 공유 역할과 목적지 정책을 적용한 뒤에도 port/fixed IPv4 후보가 여러 개 남으면 명시 selector가 필요합니다.

Plan은 owner-free로 목적지를 선택하고 mutation을 보내지 않습니다. 이후 Ensure를 새로 호출하면 새 작업으로 다시 조회·준비합니다. 공개 Decision의 selection은 실행 가능한 plan handle이 아닙니다. `Needed=true`와 Nova backend를 반환하는 diagnostic Plan도 가능하므로 Plan 성공이 해당 backend의 실행 지원을 뜻하지 않습니다.

## Python과 비교

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1527), [add_ip_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1343-L1392), [_add_ip_from_pool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1284-L1342), [기존 서버 readiness](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1481)입니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

# pool이 먼저이므로 ips와 auto_ip=False는 pool 실행을 막지 않습니다.
ready = conn.wait_for_server(
    server,
    ip_pool="public",
    ips=["198.51.100.10", "198.51.100.11"],
    auto_ip=False,
    reuse=True,
    timeout=120,
)

# pool이 없으면 제공한 목록을 순서대로 연결합니다.
ready = conn.wait_for_server(
    server,
    ips=["198.51.100.10", "198.51.100.11"],
    auto_ip=False,
    timeout=120,
)
```

Go Neutron pool은 [Ensure](../network/floating-ip-ensure.md)의 current-project/reuse 정책과 같은 고정 tuple 실행을 사용합니다. 같은 목적지에 이미 붙은 IP를 free IP보다 먼저 재사용하며, 모든 후보 페이지를 읽고 revision이 있으면 조건부 PUT을 보냅니다. Python의 [available IP 경로](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L671)는 free IP를 선택하거나 할당하고, pool helper는 별도 attach/Compute refresh 단계를 이어갑니다. 두 정책을 동일하다고 주장하지 않습니다.

Go Neutron 명시 IP 검색·연결은 [Attach](../network/floating-ip-attach.md)를 사용합니다. 목록 조회의 모든 페이지와 exact address를 검사하고 모호한 IP나 목적지는 오류로 반환합니다. project owner를 자동 조회하거나 새 IP를 할당하지 않습니다. 이미 연결된 IP의 안정된 association을 확인하며 준비한 revision 조건과 목적지를 유지하며 unrelated association drift가 있으면 재선택하지 않고 오류를 반환합니다.

## 독립 Go 예제

SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장하여 실행합니다. 기본은 서버 GET과 읽기 전용 Plan입니다. `-execute`는 같은 옵션으로 Ensure를 실행하며 서버가 supplied ACTIVE여야 합니다. 아직 준비되지 않은 서버는 `WaitForServer`를 사용하세요. `-pool`은 정확한 network **이름**, `-ips`는 쉼표로 구분한 기존 IPv4 목록입니다. 둘 다 없으면 auto이고, 둘 다 있으면 pool이 먼저입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    serverID := flag.String("server-id", "", "existing Nova server ID")
    pool := flag.String("pool", "", "exact external network name")
    ips := flag.String("ips", "", "comma-separated existing IPv4 addresses")
    execute := flag.Bool("execute", false, "execute assignment and observation")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *serverID, *pool, *ips, *execute); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, id, pool, ips string, execute bool) error {
    if id == "" {
        return fmt.Errorf("-server-id is required")
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    server, err := service.Servers.Get(ctx, id)
    if err != nil {
        return err
    }
    opts := []compute.AutomaticFloatingIPOption{
        compute.WithAutomaticIPTimeout(2 * time.Minute),
        compute.WithAutomaticEnsureOptions(network.WithEnsureWait(
            resource.WithPollInterval(time.Second))),
    }
    if ips != "" {
        addresses := strings.Split(ips, ",")
        for i := range addresses {
            addresses[i] = strings.TrimSpace(addresses[i])
        }
        opts = append(opts, compute.WithFloatingIPAddresses(addresses...))
    }
    if pool != "" {
        opts = append(opts, compute.WithFloatingIPPool(resource.Name(pool)))
    }
    input := compute.AutomaticFloatingIPRequest{Server: server}
    if !execute {
        decision, err := conn.PlanServerFloatingIP(ctx, input, opts...)
        if decision != nil {
            encoder := json.NewEncoder(os.Stdout)
            encoder.SetIndent("", "  ")
            if printErr := encoder.Encode(decision); printErr != nil {
                return printErr
            }
        }
        return err
    }
    result, err := conn.EnsureServerFloatingIP(ctx, input, opts...)
    if result != nil {
        fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
        if result.Server != nil {
            fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
        }
        for _, attempt := range result.Attempts {
            fmt.Printf("step=%d requested=%s completed=%t observed=%t error=%v\n",
                attempt.Index, attempt.RequestedAddress, attempt.Completed,
                attempt.Observed, attempt.Error)
            if attempt.Assignment != nil && attempt.Assignment.FloatingIP != nil {
                ip := attempt.Assignment.FloatingIP
                fmt.Printf("ip=%s address=%s port=%s\n", ip.ID, ip.FloatingIP, ip.PortID)
            }
        }
    }
    return err
}
```

## 순차 완료·부분 결과·시간 예산

동기 실행은 IP #1의 lookup·association·Neutron IP ACTIVE·같은 server ID의 raw Nova exact floating IPv4 관측을 끝낸 뒤 #2를 조회합니다. async Get은 각 association 접수를 끝낸 뒤 다음 조회로 진행하며 IP/raw 대기를 하지 않습니다. 어느 경로에서도 #2 실패 뒤 #3를 시작하거나 자동 DELETE·대체 allocation·Nova fallback을 보내지 않습니다.

| 결과 | 의미 |
|---|---|
| `Mode` | `ServerIPAutomatic`, `ServerIPPool`, `ServerIPExplicit` |
| `Attempts` | 시작한 explicit pool/IP 항목만 기록; Index는 0부터, pool의 RequestedAddress는 빈 문자열 |
| `Attempts[i].Assignment` / `NovaAssignment` | 해당 backend의 알려진 IP·reused/allocated evidence; 연결 전 모델이나 접수 후 partial일 수 있음 |
| `Completed` | 해당 항목 완료; async에서는 접수 완료를 포함 |
| `Observed` | 해당 항목 완료 당시 actual ACTIVE raw Nova의 exact version4/type=floating/address 관측 |
| scalar `Assignment` / `NovaAssignment` | 해당 backend의 가장 최근 알려진 assignment; #2 lookup 실패면 #1을 유지하고 #2 partial이 있으면 이를 보존 |
| `Server` | supplied 또는 마지막 matching raw Nova 서버; 생성의 최초 응답은 별도 `Creation` |
| `Attempts[i].Error` | 실패한 시작 항목의 오류; 사전 backend/capability 오류면 Attempts가 비어 있을 수 있음 |

#1 완료 뒤 #2 실패해도 #1의 `Completed/Observed`와 해당 backend assignment는 유지하며 result와 error를 함께 반환합니다. 전체 `Observed`는 성공한 동기 explicit 흐름에서 true이고, async 또는 오류에서 false입니다. **각 Observed는 처리 당시의 역사적 관측입니다.** 마지막 `Server` snapshot에 모든 이전 IP가 동시에 남아 있다는 추가 검증은 하지 않습니다.

같은 작업의 생성/readiness·선택·여러 IP association·대기·raw 관측은 하나의 overall context budget을 소비합니다. 항목마다 overall timeout을 새로 시작하지 않습니다. `WithAutomaticIPTimeout` 또는 더 이른 parent deadline이 전체 상한이며, 서버/Neutron IP waiter의 별도 timeout은 해당 stage를 더 짧게 제한할 수 있습니다. 옵션은 한 번 준비하여 소비하고, callback/source 변경·취소 오류와 접수한 HTTP 증거·부분 리소스를 보존합니다. Plan 이후 새 Ensure 호출은 별도 작업·budget입니다.

기존 기본값은 유지합니다. Plan/Ensure/Create는 전체5분·raw poll2초, Get/Wait는 전체180초·raw poll5초입니다. Get의 IP wait는 기본 false이고 `WithActiveServerWait(true)`로 켭니다. Ensure/Wait/Create는 Neutron IP ACTIVE·실제 raw 서버 ACTIVE/목표 관측을 강제하므로 하위 `WithEnsureNoWait`로 끌 수 없습니다. 명시 branch의 configured Nova/None나 정확한 Network endpoint 부재는 [legacy Nova backend](server-nova-floating-ip.md)를 실행합니다. Create는 이미 알려진 selected2.36 이상·Neutron 전용 port/NAT/project override를 서버 POST 전에 거부하며, endpoint 발견은 lazy로 유지합니다. Nova IP에는 ACTIVE 상태를 합성하지 않습니다.

## 확인 범위와 남은 작업

이 단위는 기존 Service/Connection 소비자에 Neutron/Nova pool > ordered IP > auto 분기를 연결합니다. 독립 [AddIPsToServer·AddIPList](server-ip-helpers.md)는 별도 기본60초·wait=false entry를 제공합니다. Python의 dynamic 입력·full returned Resource 계약은 계속 비교합니다. Nova의 별도 공개 CRUD·detach/cleanup·함수별 fallback, Python async helper의 선택적 Compute refresh, cloud get_server의 Resource/interface/location/session normalization·lookup retry·fault/extra_data·cleanup 전체 계약은 계속 남습니다. 서버 Create/Wait 전체 연산의 완료를 이 부분 구현만으로 주장하지 않습니다.

Python 비교는 고정 source의 정적 검토입니다. 로컬 HTTP fixtures는 순서·duplicates·pool priority·read-only Plan·selector 검증·async/sync 차이·옵션1회·고정 ID·공통 deadline 동일성/취소·partial response/source guard를 검증합니다. Python 예제나 인증된 OpenStack cloud 실행을 뜻하지 않습니다. 독립 main의 컴파일과 최종 검증 revision은 실제 실행 결과만 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록합니다.

Nova의 pool은 literal 값이며 기본 pool은 `/os-floating-ip-pools`의 첫 name입니다. `NovaAssignment`는 실제 raw IP와 allocation200/action202 증거를 보존합니다. `NovaSelections`는 실행 중 확인한 진단이며 공개 실행 handle이 아닙니다. Plan의 Nova branch는 IP 목록/할당/action 요청을 보내지 않습니다. Neutron HTTP 실패 뒤에 Nova로 전환하거나 실패한 주소 뒤 다음 항목을 시작하지 않습니다.
