# Nova-network Floating IP 연결

기존 서버 IP helper가 legacy Nova-network backend도 실행합니다. Connection에 `FloatingIPNova`와 호환 Compute microversion을 설정하고 `AddIPsToServer` 또는 `AddIPList`를 호출합니다. SDK가 pool 선택·재사용·할당·재검증·action·선택적인 주소 관측을 담당하므로 애플리케이션이 builder나 resolver를 구현할 필요가 없습니다.

Nova는 Neutron과 다른 모델을 반환합니다. `AutomaticServerIPResult.NovaAssignment`와 각 `Attempts[i].NovaAssignment`에서 실제 Nova IP와 action 증거를 읽습니다. 기존 Neutron `Assignment`에 Nova port나 합성 `ACTIVE`를 넣지 않습니다.

## 설정과 선택

Connection 생성 시 다음 concrete 옵션을 사용합니다.

- `sdk.WithMicroversion(sdk.Compute, "2.35")`: legacy IP endpoint를 사용할 정확한 버전.
- `sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNova))`: 주소 계산과 자동 IP 분기의 source.
- `compute.WithServerIPAutomaticOptions(...)`: pool/list/auto, reuse, fixed 주소, 시간 정책을 묶어 전달.
- `network.WithEnsureFixedAddress("10.0.0.10")`, `network.WithEnsureReuse(false)`: 특정 fixed IPv4 또는 새 allocation 선택.
- `compute.WithServerIPWait(true)`: raw Nova에서 이번 주소의 출현을 관측.

`WithServerAddressPolicy`는 file/environment의 전체 주소 정책을 교체합니다. private/IPv6 등 다른 주소 설정이 필요하면 같은 그룹에 포함하거나 per-call `WithAutomaticAddressOptions`로 필요한 항목만 override합니다. YAML을 사용할 경우 `clouds.yaml`의 해당 cloud에 `floating_ip_source: nova`를 설정하고 주소 정책 override를 생략할 수 있습니다.

`AddIPsToServer`의 순서는 **nonzero pool > nonempty IP 목록 > auto**입니다. 명시 pool과 IP는 private·auto=false·기존 public/floating 같은 자동 skip를 우회합니다. `AddIPList`는 positional 목록을 사용하며 순서와 중복을 유지합니다. 빈 positional 목록은 공통 검증 뒤 no-op이고 다른 selector나 자동 할당으로 바뀌지 않습니다. [독립 helper 계약](server-ip-helpers.md)과 [선택 우선순위](server-ip-dispatch.md)를 참고하세요.

Nova에서 `WithFloatingIPPool(resource.Name("public"))`과 `resource.ID(...)`는 전달한 문자열 자체를 pool 값으로 사용합니다. Neutron network 이름/ID로 조회하지 않습니다. pool을 생략하면 `GET /os-floating-ip-pools`의 첫 name을 사용하고, 빈 목록은 오류입니다. reuse=true이면 같은 pool의 `instance_id: null` IP를 선택하고, 없으면 할당합니다. 빈 문자열 instance ID를 null과 같은 free 상태로 보지 않습니다. reuse=false는 available-IP 목록을 조회하지 않고 새로 할당합니다.

명시 IP 목록은 unfiltered Nova IP 목록에서 주소를 정확히 찾습니다. 같은 주소가 여러 개면 모호성 오류이고 새 IP를 할당하지 않습니다. 선택한 IP의 ID·주소·pool·association metadata를 재GET으로 확인하며, 다른 서버로 바뀐 candidate를 새 대상으로 재선택하지 않습니다. Neutron port·NAT network·project override는 Nova action에 표현할 수 없으므로 `resource.ErrUnsupported`입니다. Nova API의 현재 project 범위를 Neutron owner 필터로 합성하지 않습니다.

## 독립 Go 예제

SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장합니다. 기본 `-mode dispatch`는 pool → IP 목록 → auto를 적용합니다. `-mode list`는 `-ips`의 positional 목록만 연결합니다. `-wait`는 주소 관측을 켜고, `-reuse=false`는 pool/auto에서 새 IP를 할당합니다. `-fixed`는 선택 사항입니다. 이 프로그램은 실제 Add 작업을 호출하며 첫 server GET에도 Compute endpoint를 사용합니다.

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "strings"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/compute"
    "gophercloudsdk/network"
    "gophercloudsdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    id := flag.String("server-id", "", "existing Nova server ID")
    mode := flag.String("mode", "dispatch", "dispatch or list")
    pool := flag.String("pool", "", "literal Nova pool")
    ips := flag.String("ips", "", "comma-separated floating IPv4 addresses")
    fixed := flag.String("fixed", "", "fixed IPv4 for addFloatingIp")
    reuse := flag.Bool("reuse", true, "reuse a free IP for pool/auto")
    wait := flag.Bool("wait", false, "observe each address in raw Nova")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *id, *mode, *pool, *ips, *fixed, *reuse, *wait); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, id, mode, pool, ips, fixed string, reuse, wait bool) error {
    if id == "" || (mode != "dispatch" && mode != "list") {
        return fmt.Errorf("provide -server-id and -mode dispatch or list")
    }
    conn, err := sdk.Connect(ctx,
        sdk.WithCloud(cloud),
        sdk.WithMicroversion(sdk.Compute, "2.35"),
        sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNova)),
    )
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
    if server == nil || server.ID != id {
        return fmt.Errorf("server GET did not return %q", id)
    }
    addresses := []string{}
    if ips != "" {
        for _, address := range strings.Split(ips, ",") {
            addresses = append(addresses, strings.TrimSpace(address))
        }
    }
    destination := []network.EnsureFloatingIPOption{network.WithEnsureReuse(reuse)}
    if fixed != "" {
        destination = append(destination, network.WithEnsureFixedAddress(fixed))
    }
    automatic := []compute.AutomaticFloatingIPOption{
        compute.WithAutomaticEnsureOptions(destination...),
    }
    if pool != "" {
        automatic = append(automatic, compute.WithFloatingIPPool(resource.Name(pool)))
    }
    if len(addresses) != 0 {
        automatic = append(automatic, compute.WithFloatingIPAddresses(addresses...))
    }
    options := []compute.ServerIPOption{
        compute.WithServerIPAutomaticOptions(automatic...),
        compute.WithServerIPWait(wait),
    }
    var result *compute.AutomaticServerIPResult
    if mode == "list" {
        result, err = conn.AddIPList(ctx, server, addresses, options...)
    } else {
        result, err = conn.AddIPsToServer(ctx, compute.AutomaticFloatingIPRequest{Server: server}, options...)
    }
    if result != nil {
        fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
        printAssignment("latest", result.NovaAssignment)
        for _, attempt := range result.Attempts {
            fmt.Printf("attempt=%d requested=%s completed=%t observed=%t error=%v\n",
                attempt.Index, attempt.RequestedAddress, attempt.Completed, attempt.Observed, attempt.Error)
            printAssignment("attempt", attempt.NovaAssignment)
        }
    }
    return err
}

func printAssignment(label string, assignment *compute.NovaFloatingIPAssignment) {
    if assignment == nil {
        return
    }
    fmt.Printf("%s reused=%t allocated=%t already_attached=%t action_accepted=%t\n",
        label, assignment.Reused, assignment.Allocated, assignment.AlreadyAttached, assignment.ActionAccepted)
    if ip := assignment.FloatingIP; ip != nil {
        fmt.Printf("ip=%s address=%s pool=%s fixed=%s instance=%s\n",
            ip.ID, ip.Address, ip.Pool, optional(ip.FixedAddress), optional(ip.InstanceID))
    }
    if response := assignment.ActionResponse; response != nil {
        fmt.Printf("action_status=%d\n", response.StatusCode)
    }
}

func optional(value *string) string {
    if value == nil {
        return "<null>"
    }
    return *value
}
```

## 반환 모델과 접수 증거

| 값 | 의미 |
|---|---|
| `NovaAssignment.FloatingIP` | 실제 list/POST/GET에서 받은 모델. 정상 연결 시 최신 pre-action 재GET 모델이며 post-action association을 합성하지 않음 |
| `Reused` / `Allocated` | 기존 IP 선택 / Nova POST200 allocation 접수 |
| `AlreadyAttached` | 재GET이 같은 서버 association과 지정한 fixed 조건을 확인하여 action을 생략 |
| `ActionAccepted` | Nova가 `addFloatingIp` action을202로 접수했다는 증거 |
| `AllocationResponse` / `ActionResponse` | 실제 body/header/status 복사본. accepted read/Close 오류에서도 알려진 응답을 보존 |
| `Observed` | 해당 entry의 raw Nova 목표 주소 관측 조건을 만족한 역사적 증거 |

Nova wire ID의 정수와 문자열을 모두 처리하며 큰 정수를 float64로 변환하지 않습니다. `Metadata.Body`는 원래 JSON의 null·누락·extension을 보존합니다. canonical 주소 키가 present이면 null도 legacy `ip`/`fixed_ip`보다 우선합니다. 선택·allocation·재GET에서 association metadata가 없으면 free 상태로 추측하지 않습니다.

action202 응답에는 IP 모델이 없습니다. `ActionAccepted=true`만으로 IP의 `InstanceID`가 바뀌었다거나 Nova가 목표 주소를 반영했다고 주장하지 않습니다. `ActionAccepted=true`와 error가 함께 올 수도 있으므로 error와 응답 증거를 함께 확인합니다. 합성 `Status="ACTIVE"`는 제공하지 않습니다. Neutron 결과는 기존 `Assignment`를 사용하며 Nova assignment와 섞지 않습니다.

allocation POST200 뒤에는 같은 IP의 GET을 수행하고, action 전에도 선택한 IP를 재검증합니다. POST의 decoded 모델이 request와 모순되거나 후속 GET/action/관측이 실패해도 알려진 allocation과 응답 증거를 error 옆에 보존합니다. 오류가 있는 partial 모델은 실행할 유효 대상으로 취급하지 않습니다. 실패 뒤 다른 IP를 할당하거나 자동 DELETE하지 않습니다.

## 시간·wait·버전 범위

standalone Add의 기본은 **전체60초, raw poll5초, wait=false**입니다. waitfalse는 접수 또는 확인된 already-attached 결과를 반환하며 raw Nova 수렴을 확인하지 않습니다. waittrue는 같은 server ID의 모든 network row에서 이번 IPv4의 exact `version=4`, `OS-EXT-IPS:type=floating`, `addr=target`를 확인합니다. standalone은 서버 상태가 BUILD/ERROR여도 목표 주소를 관측하면 완료합니다. 서버 ACTIVE나 존재하지 않는 Nova IP ACTIVE 상태를 요구하지 않습니다. SDK의 입력 상태 gate와 별개로 실제 Nova action의 서버 상태·권한 제약은 API가 검증합니다.

기존 `EnsureServerFloatingIP`, `GetActiveServer`, `WaitForServer`, `CreateWithAutomaticFloatingIP`도 같은 Nova 분기를 소비하지만 entry별 서버 readiness 조건을 유지합니다. Ensure/ready/create의 동기 경로는 실제 서버 ACTIVE와 raw 목표 주소를 요구하고 GetActive 기본 async는 IP 접수 뒤 반환합니다. Neutron 경로의 실제 IP ACTIVE 조건을 Nova 합성 상태로 대신하지 않습니다. [기존 readiness](server-ready.md), [생성 흐름](create-with-automatic-floating-ip.md)을 참고하세요.

IP helper의 선택·list·allocate·호환 GET·action·관측·목록 전체는 하나의 budget을 사용합니다. parent deadline이 더 짧으면 parent가 우선합니다. `WithAutomaticIPTimeout` 또는 unlimited 옵션은 기존 concrete 정책으로 설정합니다. 예제의 Connect와 첫 서버 GET은 Add 호출 전에 실행하므로 helper의60초에 포함되지 않으며 프로그램의 parent context가 제한합니다. 늦은 항목 실패에서도 앞선 Attempts와 마지막 matching Server를 보존합니다. 각 Observed는 처리 당시의 관측이며 마지막 Server에 모든 이전 IP가 동시에 남아 있다는 검증은 아닙니다.

legacy `/os-floating-ips`와 `/os-floating-ip-pools` 성공은200, allocation도200, `addFloatingIp`는202로 처리합니다. 빈 collection envelope와204 응답을 같은 free-IP 부재로 취급하지 않습니다. source/client/header 변경·취소·잘못된 응답 뒤에 다음 대상이나 backend로 진행하지 않습니다.

공식 Nova API는 IP/pool endpoint가 microversion2.36부터404, add/remove action은2.44부터404입니다. 이 소비자는 list/allocation endpoint를 함께 사용하므로 selected2.36이상에서 `resource.ErrUnsupported`를 반환합니다. selected microversion을 몰래 낮추지 않습니다. 위 예제는2.35를 명시하며, 해당 legacy API를 실제 제공하는 cloud가 필요합니다. [공식 버전 history](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html#microversion-2-36), [action reference](https://docs.openstack.org/api-ref/compute/#add-associate-floating-ip-addfloatingip-action-deprecated).

## Python 고정 소스와 비교

비교 pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. cloud 설정에 `floating_ip_source: nova`와 호환 Compute API 버전을 사용합니다. [pool·목록·자동 dispatcher](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1284-L1525), [Nova available/create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L673-L712), [Nova action와raw 대기](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1212)를 대조합니다.

```python
import openstack

conn = openstack.connect(cloud="dev")  # configured floating_ip_source: nova
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

# standalone defaults: timeout=60, wait=False, reuse=True
server = conn.add_ips_to_server(server, ip_pool="public", reuse=True)
server = conn.add_ip_list(
    server, ["198.51.100.10"],
    fixed_address="10.0.0.10", wait=True, timeout=60,
)
```

Python Nova path는 unfiltered 목록의 `instance_id=None`과 pool을 비교하고 첫 free IP를 재사용합니다. pool이 없으면 첫 pool name을 사용하며 [create는 POST 뒤 compat GET](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975)을 수행합니다. Go는 선택한 객체의 정확한 ID/주소/pool/association을 재검증하고 조회 실패와 성공 빈 목록을 구분합니다. source의 list404→빈 목록과 달리 legacy API 부재를 알려진 오류로 반환하여 새 allocation으로 숨기지 않습니다.

Python의 Nova normalization은 `status="ACTIVE"`를 합성하고 `instance_id`를 attached 계산에만 사용합니다. Go는 실제 raw 필드를 보존하고 action 접수와 raw 목표 관측을 나눕니다. Python의 wait는 첫 floating 주소를 비교하지만 Go는 모든 row의 정확한 IPv4 target를 찾습니다. source의 per-item timeout·dynamic `min(5, timeout)`과 Go의 전체 budget·configurable interval도 같은 정책이 아닙니다.

automatic source가Nova여도 Network 서비스가 있으면 source needs는 외부 network/NAT topology를 확인합니다. Go도 이 판단과 Nova pool 값을 분리합니다. explicit IP/pool은 자동 needs를 우회하며 Neutron endpoint나 owner를 미리 요구하지 않습니다. 순수한 native endpoint 부재와 단일 wrapped 부재만 Nova를 선택합니다. 다른 오류가 Join된 catalog 실패는 원인을 보존하고 Compute/HTTP/mutation으로 진행하지 않습니다. Neutron HTTP 실패 후 자동 Nova fallback은 구분합니다.

이 단위는 기존 Compute IP 소비자의 Nova list/선택/할당/add action 경로를 제공합니다. 별도 [AvailableFloatingIP](floating-ip-available.md)은 서버 연결 없이 free 선택 또는 새 할당을 반환하며 configured source=None에서도 explicit getter의 Nova 경로를 제공합니다. 독립 public IP list/search/get와 pool list/search는 [Floating IP query 가이드](floating-ip-queries.md)의 별도6개 API로 제공합니다. 독립 IP delete와 공개 조회 재검증은 [삭제 가이드](floating-ip-delete.md)에서 제공합니다. direct proxy add/remove 및 전체 Resource 지원은 미해결로 추적합니다. 이 가이드로 전체 source operation을 완료로 승격하지 않습니다. Python 함수별 fallback, async already-attached/pool refresh, full Resource/interface/location/session normalization·mutable model·lookup retry/error/cleanup 정책은 계속 남는 비교 범위입니다.

Python 비교는 고정 source 정적 확인입니다. 예제 컴파일·로컬 HTTP fixture·최종 revision의 gate 결과는 실제 확인된 근거만 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록합니다. Python 예제나 인증된 cloud 실행을 확인했다는 뜻은 아닙니다.

새 IP 할당·선택적 대기·timeout 정리는 [CreateFloatingIP](floating-ip-create.md)에서 제공합니다.

[미연결 Floating IP 일괄 정리](floating-ip-unattached-delete.md)는 Neutron 전체 목록을 확보한 뒤 port가 비어 있는 항목을 순차 삭제합니다. 개별 false는 계속 처리하고 오류는 중단하며 SDK 소유 옵션·한 deadline·항목별 부분 결과를 제공합니다.
