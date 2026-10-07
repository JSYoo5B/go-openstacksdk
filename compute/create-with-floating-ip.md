# 서버 생성과 Floating IP 연결

`service.Servers.CreateWithFloatingIP`은 서버 생성, 실제 서버 `ACTIVE` 확인, floating IPv4 재사용 또는 할당·연결, 실제 IP `ACTIVE` 확인을 하나의 작업으로 수행합니다. Connection이 Compute와 Network 연결을 준비하고 이름을 조회하므로 애플리케이션에서 builder나 resolver를 구현할 필요가 없습니다. 이미지 부팅, 기존 볼륨 부팅, 이미지에서 새 볼륨을 만드는 부팅에 같은 흐름을 적용합니다.

`Servers.Create`는 기존대로 기본 비동기 생성 응답을 반환하며 `compute.WithWait`로 서버 대기를 선택합니다. `CreateWithFloatingIP`은 서버와 IP 대기가 항상 필수입니다. `WithWait`와 `WithEnsureWait`는 이 메서드에서 대기의 timeout·간격·실패 상태·callback 정책을 설정합니다.

## Python cloud 호출과 대응

Python에서 외부 pool을 명시해 서버 생성과 IP 연결을 요청하는 예제입니다. `auto_ip=False`여도 명시한 `ip_pool` 분기가 먼저 적용됩니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    network="private",
    ip_pool="public",
    auto_ip=False,
    reuse_ips=True,
    wait=True,
    timeout=180,
)
print(server.id)
```

Go에서는 서버 생성 입력과 외부 floating network를 구분한 요청을 사용합니다. 결과에 실제 Nova 서버와 IP assignment를 각각 보관합니다.

| Python cloud 입력/동작 | Go |
| --- | --- |
| `name`, `image`, `flavor` | `request.Server`의 `CreateServerRequest` |
| 서버용 `network` / `nics` | `WithServerOptions(WithNetworks(...))` / `WithNetworkInterfaces(...)` |
| `ip_pool="public"` | `request.FloatingIPNetwork = resource.Name("public")` |
| 외부 네트워크 자동 선택 | `FloatingIPNetwork` 생략; 현재 Ensure의 외부 네트워크 선택 정책 사용 |
| `reuse_ips=True` | 기본값; `WithFloatingIPOptions(network.WithEnsureReuse(true))` |
| `reuse_ips=False` | `WithFloatingIPOptions(network.WithEnsureReuse(false))` |
| `wait=True`, `timeout=180`의 작업 의도 | 항상 서버·IP ACTIVE 확인; `WithWorkflowTimeout(180*time.Second)` |
| `boot_volume=...` | `WithServerOptions(compute.WithBootVolume(ref))` |
| `boot_from_volume=True`, `volume_size=50` | 이미지 요청에 `WithServerOptions(compute.WithBootVolumeSize(50))` |
| 서버를 확장한 단일 반환 객체 | `result.Server`와 `result.Assignment`를 별도 반환 |

이 대응은 명시적으로 서버 생성과 floating IPv4 연결을 요청하는 부분입니다. Python의 기본 `create_server(auto_ip=True)`에서 **IP가 필요한지 판단하는 조건**까지 같다는 의미는 아닙니다.

## 독립 Go 예제

`clouds.yaml`의 `dev` 인증 설정과 `ubuntu`, `small`, `private`, `public` 이름을 실제 환경에 맞게 바꿉니다. 예제는 Compute microversion 2.37을 명시하므로 선택한 cloud가 이 버전을 지원해야 합니다. `resource.Name`은 정확한 이름 조회이고 `resource.ID`는 지정한 ID를 사용합니다.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/compute"
    "gophercloudsdk/network"
    "gophercloudsdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    conn, err := sdk.Connect(ctx,
        sdk.WithCloud("dev"),
        sdk.WithMicroversion(sdk.Compute, "2.37"),
    )
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    result, err := service.Servers.CreateWithFloatingIP(ctx,
        compute.CreateServerWithFloatingIPRequest{
            Server: compute.CreateServerRequest{
                Name:   "web-01",
                Image:  resource.Name("ubuntu"),
                Flavor: resource.Name("small"),
            },
            FloatingIPNetwork: resource.Name("public"),
        },
        compute.WithWorkflowTimeout(3*time.Minute),
        compute.WithServerOptions(
            compute.WithNetworks(resource.Name("private")),
            compute.WithWait(resource.WithPollInterval(time.Second)),
        ),
        compute.WithFloatingIPOptions(
            network.WithEnsureWait(resource.WithPollInterval(time.Second)),
        ),
    )
    if err != nil {
        if result != nil {
            if result.Server != nil {
                fmt.Fprintf(os.Stderr, "server=%s status=%s\n",
                    result.Server.ID, result.Server.Status)
            }
            if result.Assignment != nil && result.Assignment.FloatingIP != nil {
                assignment := result.Assignment
                fmt.Fprintf(os.Stderr, "floating-ip=%s address=%s reused=%t allocated=%t\n",
                    assignment.FloatingIP.ID, assignment.FloatingIP.FloatingIP,
                    assignment.Reused, assignment.Allocated)
            }
        }
        return err
    }
    fmt.Printf("server=%s floating-ip=%s\n",
        result.Server.ID, result.Assignment.FloatingIP.FloatingIP)
    return nil
}
```

성공 시에는 실제 서버와 IP의 ACTIVE를 확인했습니다. 이 시점에 반환된 서버의 `Addresses`에는 floating IP가 아직 없을 수 있습니다. SDK가 주소를 합성하거나 Nova 주소 반영·SSH 접속 가능성까지 기다리지는 않습니다.

## 기본값과 옵션

서버 NIC는 기존 Create와 같은 선택 순서를 사용합니다. 명시 `WithNetworks`·`WithNetworkInterfaces`·`WithNetworkMode`가 먼저이고, 없으면 Connection의 `WithDefaultNetwork`/`WithoutDefaultNetwork`, YAML 기본 네트워크, 선택 microversion에 따른 auto/생략 순서입니다. 외부 `FloatingIPNetwork`는 서버 NIC의 기본값을 바꾸지 않습니다. [서버 기본 네트워크](server-default-network.md)와 [NIC 입력](server-network-interfaces.md)에 세부 규칙을 기록했습니다.

외부 floating network를 생략하면 Ensure는 응답 순서상 첫 외부 네트워크를 선택하고, 목록에 없으면 첫 enabled router의 외부 gateway를 사용합니다. 목록 페이지의 오류를 숨기지 않습니다. 명시한 external network ID는 이름 목록 조회를 생략하지만 이 workflow에는 Neutron endpoint와 port/IP 작업이 여전히 필요합니다.

floating IP는 기본적으로 현재 project에서 재사용합니다. 같은 destination에 이미 연결된 IP, 첫 free 후보, 새 allocation 순서이며 전체 후보 페이지를 확인한 뒤 mutation합니다. `WithEnsureReuse(false)`는 reuse 목록과 기본 project scope 판정을 생략하고 새 IP를 할당합니다. `WithEnsureProject("project-id")`는 재사용과 할당의 owner를 명시합니다. 일반적인 project-scoped Connection은 owner를 직접 지정할 필요가 없습니다.

선택 조건에 맞는 `(port, fixed IPv4)` 후보가 여러 개이면 임의로 하나를 고르지 않고 `ErrAmbiguous`를 반환합니다. 서버에 연결된 port, NAT destination network, fixed IPv4를 `WithFloatingIPOptions` 안의 `WithEnsurePort`, `WithEnsureNATDestination`, `WithEnsureFixedAddress`로 좁힐 수 있습니다. fixed address 옵션은 연결할 서버 주소를 선택하며 서버 NIC에 새 주소를 할당하는 옵션은 아닙니다. 명시 port가 다른 서버 소유이면 실패합니다.

`WithServerOptions`와 `WithFloatingIPOptions`는 여러 번 사용하면 해당 옵션들을 순서대로 추가합니다. 같은 service 설정의 마지막 옵션이 우선하고 앞선 invalid option은 뒤 옵션으로 숨기지 않습니다. 전달된 option slice는 snapshot하므로 원래 slice의 원소를 교체해도 준비한 workflow option은 바뀌지 않습니다.

## 생성 전 확인과 생성 후 실패

SDK는 server create의 입력·부팅 조합·선택 microversion·wait 옵션, external Ref와 Ensure 옵션을 Nova POST 전에 검증합니다. Connection의 Network 서비스 확보와 기본 reuse project의 recorded auth scope 확인도 POST 전에 합니다. 잘못된 옵션, 없는 Neutron endpoint, 기본 reuse인데 project scope를 알 수 없는 provider는 서버를 만들기 전에 실패합니다. project를 endpoint URL이나 반환 서버의 TenantID에서 추측하지 않습니다.

SDK는 내부적으로 `FloatingIPs.PrepareEnsureActive`를 사용합니다. 이 preparation은 재사용 owner를 정책에 바인딩하고 필수 IP 대기를 켜며, 지정한 IP wait 옵션은 보존합니다. 앱은 이 helper나 builder를 직접 만들 필요가 없습니다. 이미 준비한 일반 Ensure policy로 설정을 교체하더라도 이 workflow의 필수 IP ACTIVE 확인을 끌 수 없습니다.

이미지·flavor·서버 network 등의 이름 조회는 기존 Create처럼 POST 전에 수행합니다. 새 서버의 port/fixed IPv4 선택, 외부 floating network 조회, IP 후보 조회·연결·할당은 서버 ACTIVE 이후에 수행합니다. 이 단계의 403, missing/ambiguous resource, IP quota, revision 충돌, timeout은 서버 생성 이후 실패일 수 있습니다. 생성 전 검증이 모든 cloud 상태를 보장한다고 해석하지 않습니다.

새 서버 ID는 Ensure에 `resource.ID`로 전달하며 서버 이름을 다시 조회하지 않습니다. server Wait 성공 뒤에도 원래 생성 ID와 응답 ID, 실제 `Status == ACTIVE`를 확인합니다. IP도 실제 Status와 ID·owner·external network·port·fixed address를 확인합니다. `resource.WithStatusAttribute`로 다른 필드를 선택해 waiter가 먼저 완료되더라도 실제 서버/IP가 ACTIVE가 아니면 성공으로 반환하지 않습니다.

## 부분 결과 처리

후속 작업에 실패하면 error와 함께 SDK가 확인한 리소스가 남습니다. 결과를 먼저 확인한 다음 오류를 처리하면, 재생성에 앞서 기존 서버와 IP를 조사할 수 있습니다.

| 실패 지점 | 보존되는 결과 |
| --- | --- |
| 생성 전 입력/endpoint/scope/이름 조회 오류 | 생성한 리소스 없음 |
| Nova 생성 후 서버 대기 오류 | `result.Server`에 원래 생성 서버; Assignment 없음 |
| 서버 ACTIVE 이후 외부 network/port/IP 목록 오류 | 확인된 ACTIVE server; Assignment 없음 |
| IP 재사용 후보 선택 뒤 guarded PUT 실패 | ACTIVE server + 선택한 Reused IP |
| 새 IP 응답 확인 뒤 검증/ACTIVE 대기 실패 | ACTIVE server + Allocated IP |

`Assignment.Reused`는 free IP뿐 아니라 해당 destination에 이미 연결된 IP를 재사용한 경우도 포함합니다. `Assignment.Allocated`는 새 allocation 결과를 확인했다는 뜻이고 전체 workflow의 성공 flag는 아닙니다. 응답이 유실돼 리소스 ID를 확인하지 못한 경우까지 SDK가 식별한 것으로 표시하지 않습니다.

SDK는 실패한 서버/IP를 자동 DELETE하지 않으며, 재사용 PUT이 409/412 등으로 실패해도 새 allocation으로 우회하지 않습니다. 알려진 revision은 guarded PUT에 사용하지만 free IP 선택 자체를 예약하거나 여러 프로세스를 잠그지는 않습니다. 애플리케이션이 필요에 따라 명시적으로 조사·정리하거나 재시도 정책을 적용합니다. 별도 `FloatingIPs.Ensure`의 자세한 owner/revision/selection 계약은 [Ensure 가이드](../network/floating-ip-ensure.md)를 참고하세요.

## 전체 timeout과 대기 정책

기본 workflow timeout은 5분입니다. `WithWorkflowTimeout`은 Network 서비스/preparation, create의 의존 리소스 조회·POST, 서버 대기, IP 조회·연결·할당·대기에 하나의 context deadline을 적용합니다. 부모 context의 deadline이 더 빠르면 그 시간이 우선합니다. 각 단계마다 전체 timeout을 다시 시작하지 않습니다. Python cloud의 timeout은 서버 대기 시작 뒤의 remaining 시간을 IP 단계에 전달하지만 Go의 전체 제한은 서비스 준비·조회·Nova POST도 포함하므로 정확한 시간 범위가 다릅니다.

`compute.WithWait(resource.WithTimeout(...))`와 `network.WithEnsureWait(resource.WithTimeout(...))`는 각 대기 단계의 더 짧은 제한을 설정할 수 있지만 공통 deadline을 늘리지 않습니다. `WithUnlimitedWorkflowTimeout`은 전체 SDK 제한을 해제하며 부모 context와 개별 waiter의 제한은 계속 적용됩니다. `resource.WithUnlimitedWait`만 지정해도 공통 workflow deadline은 유지됩니다. timeout과 cancellation은 알려진 부분 결과와 함께 전달하며 `errors.Is(err, context.DeadlineExceeded)`/`context.Canceled`로 확인할 수 있습니다.

## Python 대비 남은 범위와 검증

비교 기준은 고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. [cloud create_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L915-L1245), [서버 대기와 remaining timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1481), [IP 선택 분기와 필요성 판정](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1606), [NAT port 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1709)의 소스를 확인했습니다.

다음 차이는 남아 있으므로 cloud `create_server`와 `_network_common` 전체 지원 판정은 계속 partial입니다.

- `_needs_floating_ip`의 기존 public/floating 주소·fixed 주소·private cloud·service/flags에 따른 자동 생략, 외부 network/NAT 가능성 판정. 이 Go 메서드는 연결을 명시적으로 요청하는 흐름입니다.
- Python shared network role/NAT·subnet 관계, 외부/내부·IPv4/IPv6 분류, `has_service`, cache/reset과 cloud flags의 전체 정책.
- `ips` 목록, pool/auto 선택 우선순위의 전체 조합, Nova floating IP fallback, cloud의 일부 생성 확장 입력과 추가 볼륨/server group 동작.
- Python의 일부 activation timeout 이후 신규 IP 삭제, ACTIVE지만 주소가 없을 때 서버 삭제. Go workflow는 부분 결과를 보존합니다.
- Python `_attach_ip_to_server(wait=True)`의 서버 주소 재조회·수렴 및 `public_v4`/`interface_ip` 같은 반환값 확장. Go는 실제 Nova server와 Neutron IP를 각각 반환합니다.
- Python NAT port의 최근 생성 port/첫 IPv4 선택. Go Ensure는 여러 후보를 엄격히 실패시키고 명시 선택을 받습니다.

Python 비교는 고정 소스를 읽은 결과이며 Python 예제 실행이나 인증된 OpenStack 검증 결과가 아닙니다. 로컬 HTTP 테스트는 옵션 검증과 호출 순서, 공통 deadline/cancellation, 실제 ACTIVE·identity 검사, 재사용/새 allocation, revision 충돌, 실패 후 서버·IP 보존을 확인하는 범위입니다. 독립 Go 예제의 컴파일 확인과 해당 HTTP 테스트 실행 결과는 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록합니다. 실환경의 quota·라우터 연결성·Nova 주소 반영·네트워크 도달성은 이 로컬 검증에 포함하지 않습니다.

[공유 네트워크 역할 조회](../network/network-roles.md)는 별도 getter로 제공됩니다. 현재 이 workflow의 Ensure 선택과 자동 IP 필요성 판단에는 연결되지 않았으므로, 역할 조회 자체와 위 remaining의 상위 정책 연계를 구분합니다.
