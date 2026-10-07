# 기존 서버에 Floating IP 추가

`AddIPsToServer`는 pool > IP 목록 > auto 정책을 적용하고, `AddIPList`는 제공한 기존 IPv4 목록만 순서대로 연결합니다. 기본은 **전체60초·raw poll5초·비동기 접수**입니다. Connection이 cloud 설정·공유 역할·서비스와 목적지 해석을 제공하므로 애플리케이션에서 builder나 resolver interface를 구현하지 않습니다.

| 공개 호출 | 입력 |
|---|---|
| `service.AddIPsToServer(ctx, input, options...)` | `compute.AutomaticFloatingIPRequest{Server: server, Network: optionalAutoNetwork}` |
| `service.AddIPList(ctx, server, addresses, options...)` | `*compute.Server`, `[]string`의 기존 floating IPv4 주소 |
| `conn.AddIPsToServer` / `conn.AddIPList` | 같은 입력·옵션·결과를 사용하는 Connection facade |

반환은 `(*compute.AutomaticServerIPResult, error)`입니다. nil/canceled context, nil server, 안전하지 않은 server ID와 적용되는 옵션의 오류는 작업 준비 시 거부합니다. 일부 작업이 접수된 뒤 실패하면 알려진 Server·Assignment·Attempts와 오류를 함께 반환합니다.

## 옵션·선택·빈 목록

`compute.ServerIPOption`의 두 concrete 옵션을 사용합니다.

- `WithServerIPWait(bool)`: 기본 false; 마지막 값이 적용됩니다. true이면 각 IP의 정확한 주소가 같은 server ID의 raw Nova에 나타날 때까지 관측합니다.
- `WithServerIPAutomaticOptions(...compute.AutomaticFloatingIPOption)`: 기존 pool/list/auto·주소 설정·Neutron 목적지/reuse·시간·progress 옵션을 공유합니다. 여러 그룹은 전달한 순서대로 옵션을 모읍니다.

`AddIPsToServer`에서는 `WithFloatingIPPool(resource.Name("public"))` 또는 명시 `resource.ID`의 pool이 먼저이고, 없으면 `WithFloatingIPAddresses(...)`, 둘 다 없으면 기본 auto=true와 기존 needs/skip 정책입니다. reuse는 기본 true입니다. 명시 pool/IP 목록은 auto=false·private·기존 public/floating 주소 등 자동 skip를 우회합니다. 최종 winning selector만 검증하며 공통 address/Ensure/wait 옵션 검증은 계속 수행합니다. [selector 우선순위와 pool 정책 차이](server-ip-dispatch.md)를 참고하세요.

`AddIPList`의 positional `addresses`는 옵션보다 우선합니다. 순서와 중복을 유지하고, nested pool/list 옵션이 positional 목록을 바꾸거나 지우지 못합니다. `nil` 또는 `[]string{}`는 공통 검증 뒤 no-op입니다. `Mode=ServerIPExplicit`, `Reason=AutomaticIPEmptyAddressList`, `Needed=false`와 supplied Server를 반환하며 backend/owner/서비스 조회·HTTP·자동 할당을 하지 않습니다. 반면 `AddIPsToServer`의 빈 IP selector는 auto 분기로 돌아갑니다.

목적지는 `WithAutomaticEnsureOptions(network.WithEnsurePort(...), network.WithEnsureNATDestination(...), network.WithEnsureFixedAddress(...))`로 선택합니다. 명시 기존 IP는 새로 할당하지 않고 current-project/free-IP 필터를 적용하지 않습니다. Neutron 권한 아래 기존 association을 이동할 수 있으며, 목적지 필터를 적용한 뒤에도 후보가 여러 개면 오류입니다. pool/auto의 project/reuse와 [Attach](../network/floating-ip-attach.md)·[Ensure](../network/floating-ip-ensure.md)의 재검증·revision·부분 결과 정책은 유지합니다.

## wait는 주소 관측이며 서버 준비 대기가 아닙니다

standalone Add는 supplied ACTIVE를 요구하지 않으며 BUILD·SHUTOFF·ERROR나 주소 미준비라는 이유만으로 GetActiveServer처럼 거부하지 않습니다. `wait=false`는 association 접수 뒤 반환하며 IP ACTIVE와 raw 주소 관측을 하지 않습니다. IP가 DOWN이어도 알려진 association을 결과로 받을 수 있습니다.

`WithServerIPWait(true)`는 같은 ID의 raw Nova `GET /servers/{id}`에서 exact version4·type=floating·이번 assigned IPv4를 확인합니다. raw 상태가 BUILD나 ERROR여도 해당 주소가 보이면 관측을 완료합니다. 다른 floating 주소·fixed 행·IPv6·AccessIPv4·Neutron ACTIVE만으로 완료하지 않습니다. **서버나 IP의 ACTIVE는 이 entry의 완료 조건이 아닙니다.** `WithEnsureWait`의 옵션은 검증되지만 이 entry에서 IP ACTIVE polling을 켜지 않습니다. raw budget/poll/progress는 `WithAutomaticIPTimeout`, `WithUnlimitedAutomaticIPTimeout`, `WithAutomaticIPPollInterval`, `WithAutomaticIPProgress`로 설정합니다.

기존 `EnsureServerFloatingIP`, `GetActiveServer`의 상태 판정, `WaitForServer`, `CreateWithAutomaticFloatingIP`의 stronger readiness 조건은 바꾸지 않습니다. [기존 서버 readiness](server-ready.md)와 [생성 흐름](create-with-automatic-floating-ip.md)은 실제 서버 ACTIVE·주소 준비 및 각 entry의 Neutron IP ACTIVE/관측 조건을 계속 적용합니다.

## Python과 비교

고정 비교 소스는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1525), [add_ip_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1343-L1392), [attach helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1163)입니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

# standalone defaults: timeout=60, wait=False, auto_ip=True, reuse=True
added = conn.add_ips_to_server(server, ip_pool="public", auto_ip=False)
added = conn.add_ip_list(server, ["198.51.100.10", "198.51.100.11"], wait=True)
```

Python의 add_ip_list는 각 IP에 같은 timeout을 전달하며 wait=true일 때 각 주소 관측에 사용합니다. Go는 async HTTP를 포함해 목록 전체가 하나의 overall budget을 소비하며 항목마다60초를 다시 시작하지 않습니다. source helper의 첫 floating 주소 선택과 달리 Go는 모든 network row에서 이번 target 주소를 찾습니다. source `wait=min(5, timeout)`과 Go의 configurable raw interval도 완전히 같은 정책은 아닙니다.

Python attach helper는 supplied floating 주소가 없고 selected IP의 port_id가 있으면 wait=false에서도 Compute GET을 한 번 하여 이미 붙은 주소인지 확인합니다. pool의 reuse=false 분기도 create 뒤 Compute GET을 수행하며, create(wait=true)가 IP ACTIVE를 기다릴 수 있습니다. Go standalone async에는 이 refresh가 없고, sync도 raw 주소 관측만 요구합니다. pool의 기존 Ensure 재사용·할당 정책과 이 차이는 계속 구분합니다.

## 독립 Go 예제

이 예제는 **서버 GET을 먼저 하므로 Compute endpoint가 필요**합니다. 모델을 이미 가진 애플리케이션은 바로 Add를 호출할 수 있으며 async 명시 IP 연결이나 빈 AddIPList에는 Compute 초기화가 필요하지 않습니다. SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장합니다. `-mode dispatch`가 기본이고 `-mode list`는 positional 목록만 연결합니다. `-pool`은 정확한 외부 network 이름이며 `-ips`는 쉼표로 구분한 기존 IPv4 목록입니다. `-wait`는 주소 관측을 켭니다. 아래 프로그램은 Add 작업을 직접 호출합니다.

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
    "gophercloudsdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    id := flag.String("server-id", "", "existing Nova server ID")
    mode := flag.String("mode", "dispatch", "dispatch or list")
    pool := flag.String("pool", "", "exact external network name")
    ips := flag.String("ips", "", "comma-separated existing IPv4 addresses")
    wait := flag.Bool("wait", false, "observe each address in raw Nova")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *id, *mode, *pool, *ips, *wait); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, id, mode, pool, ips string, wait bool) error {
    if id == "" || (mode != "dispatch" && mode != "list") {
        return fmt.Errorf("provide -server-id and -mode dispatch or list")
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
    if server == nil || server.ID != id {
        return fmt.Errorf("Nova returned a different or missing server")
    }
    var addresses []string
    if ips != "" {
        addresses = strings.Split(ips, ",")
        for i := range addresses {
            addresses[i] = strings.TrimSpace(addresses[i])
        }
    }
    var automatic []compute.AutomaticFloatingIPOption
    if len(addresses) != 0 {
        automatic = append(automatic, compute.WithFloatingIPAddresses(addresses...))
    }
    if pool != "" {
        automatic = append(automatic, compute.WithFloatingIPPool(resource.Name(pool)))
    }
    options := []compute.ServerIPOption{
        compute.WithServerIPAutomaticOptions(automatic...),
        compute.WithServerIPWait(wait),
    }
    var result *compute.AutomaticServerIPResult
    if mode == "list" {
        result, err = conn.AddIPList(ctx, server, addresses, options...)
    } else {
        result, err = conn.AddIPsToServer(ctx,
            compute.AutomaticFloatingIPRequest{Server: server}, options...)
    }
    if result != nil {
        fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
        if result.Assignment != nil && result.Assignment.FloatingIP != nil {
            ip := result.Assignment.FloatingIP
            fmt.Printf("ip=%s address=%s status=%s\n", ip.ID, ip.FloatingIP, ip.Status)
        }
        for _, attempt := range result.Attempts {
            fmt.Printf("step=%d requested=%s completed=%t observed=%t error=%v\n",
                attempt.Index, attempt.RequestedAddress, attempt.Completed,
                attempt.Observed, attempt.Error)
        }
    }
    return err
}
```

## 부분 결과와 시간 예산

각 IP를 순서대로 준비·재검증·연결하며 wait=true이면 해당 raw 관측을 마친 뒤 다음 IP를 조회합니다. #1 완료 뒤 #2가 실패하면 #3를 시작하지 않습니다. `Attempts`는 명시 IP 목록에서는 시작한 항목을, pool에서는 단일 실행 단계를 기록합니다. auto에는 항목별 Attempts가 없습니다. #1의 Assignment·Completed·Observed는 유지하고 #2의 known partial과 Error도 보존합니다. scalar Assignment는 가장 최근 known assignment여서 #2 lookup 실패면 #1이고 #2 partial이 있으면 #2일 수 있습니다. 자세한 [Attempts/Mode 계약](server-ip-dispatch.md)을 참고하세요.

standalone `Observed`는 **처리 당시 raw 주소 관측의 역사적 증거**이며 ACTIVE를 뜻하지 않습니다. 마지막 Server snapshot에 이전 IP가 모두 동시에 남아 있다는 추가 검증도 하지 않습니다. async나 empty/no-assignment skip는 Observed=false입니다. 순차 단계가 실패하면 전체 Observed=false이지만 먼저 끝난 attempt의 historical Observed는 남습니다.

server ID/상태의 top-level snapshot과 positional 목록 복사는 옵션 실행 전에 수행합니다. 그룹·automatic·Ensure/wait·address 옵션은 한 번 준비하며 raw progress callback에는 분리한 모델을 전달합니다. default60초 또는 `WithAutomaticIPTimeout`과 더 이른 parent deadline은 분류·선택·목록 전체·association·관측을 함께 제한합니다. native 모델·주소를 확인할 수 있는 accepted raw read/Close 오류는 마지막 matching Server와 HTTP 증거를 보존합니다. source 변경·취소·응답 오류 뒤에는 다른 대상이나 backend로 진행하지 않습니다. 자동 DELETE·실패 뒤 대체 allocation·Nova fallback도 수행하지 않습니다.

## 확인 범위와 remaining

이 entry는 Neutron과 [legacy Nova backend](server-nova-floating-ip.md)의 dispatcher·positional 목록을 제공합니다. 명시 pool/IP의 configured Nova/None 또는 정확한 Network endpoint 부재는 Nova를 선택합니다. automatic None은 skip입니다. Nova는 호환 Compute endpoint가 필요하며 selected2.36 이상·Neutron 전용 port/NAT/project override는 typed `resource.ErrUnsupported`입니다. Nova의 별도 공개 CRUD·detach/cleanup·함수별 fallback, Python의 async already-attached/pool refresh, reuse=false pool create의 특수 IP ACTIVE 대기, full cloud Resource/interface/location/session normalization·lookup/error/cleanup 전체 계약은 계속 남습니다. 서버 Create/Wait와 해당 Python cloud 연산의 전체 완료를 이 단위만으로 판정하지 않습니다.

Python 비교는 고정 source의 정적 검토입니다. 로컬 HTTP fixture는 async/sync·BUILD/ERROR 주소 관측·IP DOWN·60초 공통 budget·positional 권한·empty no-op·옵션1회·부분 응답·source guard를 검증합니다. Python 예제나 인증된 OpenStack cloud 실행을 뜻하지 않습니다. 독립 main의 정확한 컴파일 및 최종 검증 revision은 실제 확인한 결과만 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록합니다.

Nova 결과는 `NovaAssignment`와 각 `Attempts[i].NovaAssignment`를 사용합니다. action202를 접수해도 pre-action 모델의 association이나 합성 IP ACTIVE를 바꾸지 않습니다. IP 접수·already-attached 확인과 raw 관측을 구분하며, allocation/후속 조회/관측 실패의 알려진 모델과 실제 HTTP 증거를 함께 보존합니다.
