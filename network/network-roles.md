# 네트워크 역할 조회와 공유 snapshot

`Connection.GetNetworkRoles(ctx)`는 Neutron 네트워크를 한 번 탐색하여 외부·내부 IPv4/IPv6, Floating IP용 외부 IPv4, NAT source/destination, 기본 인터페이스 역할을 함께 반환합니다. 개별 getter와 `network.Service.Roles.Discover(ctx)`도 같은 성공 결과를 공유합니다. 애플리케이션에서 builder나 resolver를 구현할 필요 없이 `clouds.yaml` 또는 SDK의 concrete 옵션으로 정책을 지정할 수 있습니다.

이 단위는 openstacksdk의 네트워크 역할 getter와 설정 분류를 제공합니다. 서버에 Floating IP가 필요한지 판단하는 `_needs_floating_ip`, 자동 생성·연결, 서버의 `public_v4`·`private_v4`·`interface_ip` 계산까지 구현되었다는 의미는 아닙니다.

## clouds.yaml로 사용하기

Python과 Go에서 같은 `dev` 항목을 선택할 수 있습니다. 아래 `public`·`private`는 실제 클라우드의 정확한 네트워크 이름 또는 ID로 바꿉니다. 인증 값은 해당 클라우드의 설정을 사용합니다.

```yaml
clouds:
  dev:
    auth:
      auth_url: https://identity.example.com/v3
      username: demo
      password: secret
      project_name: demo
      user_domain_name: Default
      project_domain_name: Default
    region_name: RegionOne
    use_external_network: true
    use_internal_network: true
    networks:
      - name: public
        routes_externally: true
        nat_source: true
      - name: private
        routes_externally: false
        nat_destination: true
        default_interface: true
```

Python:

```python
import openstack

conn = openstack.connect(cloud="dev")
for net in conn.get_external_ipv4_networks():
    print(net.id, net.name)
print(conn.get_nat_source())
print(conn.get_nat_destination())
print(conn.get_default_network())
```

Go에서 YAML 정책만 사용하려면 `sdk.Connect(ctx, sdk.WithCloud("dev"))`로 연결합니다. 다음 독립 예제는 같은 역할을 concrete 옵션으로 지정합니다. `sdk.WithNetworkRoles`는 YAML 역할 정책 전체를 교체합니다.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/network"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    conn, err := sdk.Connect(ctx,
        sdk.WithCloud("dev"),
        sdk.WithNetworkRoles(
            network.WithConfiguredNetworks(
                network.ConfiguredNetwork{
                    Name: "public",
                    RoutesIPv4Externally: true,
                    RoutesIPv6Externally: true,
                    NATSource: true,
                },
                network.ConfiguredNetwork{
                    Name: "private",
                    NATDestination: true,
                    DefaultInterface: true,
                },
            ),
        ),
    )
    if err != nil {
        return err
    }
    fmt.Printf("external=%t internal=%t\n",
        conn.UseExternalNetwork(), conn.UseInternalNetwork())

    roles, err := conn.GetNetworkRoles(ctx)
    if err != nil {
        return err
    }
    for _, net := range roles.ExternalIPv4 {
        fmt.Printf("external IPv4: %s %s router_external=%t provider=%q\n",
            net.ID, net.Name, net.RouterExternal, net.ProviderPhysicalNetwork)
    }
    printNetwork("NAT source", roles.NATSource)
    printNetwork("NAT destination", roles.NATDestination)

    // 성공한 탐색 결과를 공유합니다. 다시 HTTP 조회하지 않습니다.
    defaultNetwork, err := conn.GetDefaultNetwork(ctx)
    if err != nil {
        return err
    }
    printNetwork("default", defaultNetwork)

    // Reset 자체는 HTTP 요청을 하지 않습니다. 다음 조회가 새로 탐색합니다.
    conn.ResetNetworkRoles()
    fresh, err := conn.GetNetworkRoles(ctx)
    if err != nil {
        return err
    }
    fmt.Printf("fresh external IPv4 networks: %d\n", len(fresh.ExternalIPv4))
    return nil
}

func printNetwork(label string, net *network.RoleNetwork) {
    if net == nil {
        fmt.Printf("%s: none\n", label)
        return
    }
    fmt.Printf("%s: %s %s\n", label, net.ID, net.Name)
}
```

## getter와 반환 모델

Go의 모든 조회에는 `context.Context`가 필요하며 오류를 별도로 반환합니다. 목록 getter는 `[]*network.RoleNetwork`, 단일 getter는 `*network.RoleNetwork`를 반환합니다. 해당 역할이 없으면 빈 목록 또는 `nil`이 정상 결과입니다. 설정한 네트워크가 실제 목록에 없거나 단일 역할의 selector가 여러 네트워크와 일치하면 오류입니다.

| openstacksdk Connection | gophercloudsdk Connection |
| --- | --- |
| `get_external_ipv4_networks()` | `GetExternalIPv4Networks(ctx)` |
| `get_internal_ipv4_networks()` | `GetInternalIPv4Networks(ctx)` |
| `get_external_ipv6_networks()` | `GetExternalIPv6Networks(ctx)` |
| `get_internal_ipv6_networks()` | `GetInternalIPv6Networks(ctx)` |
| `get_external_ipv4_floating_networks()` | `GetExternalIPv4FloatingNetworks(ctx)` |
| `get_external_networks()` | `GetExternalNetworks(ctx)` |
| `get_internal_networks()` | `GetInternalNetworks(ctx)` |
| `get_nat_source()` | `GetNATSource(ctx)` |
| `get_nat_destination()` | `GetNATDestination(ctx)` |
| `get_default_network()` | `GetDefaultNetwork(ctx)` |

`GetNetworkRoles(ctx)`는 이 역할들을 `NetworkRoleSnapshot` 하나로 읽는 추가 API입니다. 목록은 `ExternalIPv4`, `InternalIPv4`, `ExternalIPv6`, `InternalIPv6`, `ExternalIPv4Floating`, 단일 값은 `NATSource`, `NATDestination`, `DefaultNetwork`입니다. Aggregate getter는 IPv4 목록 뒤에 IPv6 목록을 붙입니다. 두 family에 속하는 네트워크는 두 번 나타나며, Python 소스와 같이 ID로 중복 제거하지 않습니다.

`RoleNetwork`는 기존 native `network.Network` 모델을 embed하고 `RouterExternal`과 `ProviderPhysicalNetwork`를 추가합니다. 일반 `service.Networks`의 반환 타입은 그대로 유지됩니다. 역할 탐색 decoder는 native 모델의 custom JSON decoder가 extension 필드를 소비하지 않도록 응답 객체 전체를 native 모델과 extension에 각각 전달합니다. 따라서 native timestamp 처리와 `router:external`·`provider:physical_network` 필드를 함께 보존합니다. 잘못된 JSON field 타입은 오류입니다.

서비스를 이미 얻었다면 같은 탐색과 cache를 직접 사용할 수 있습니다.

```go
service, err := conn.Network(ctx)
if err != nil {
    return err
}
roles, err := service.Roles.Discover(ctx)
if err != nil {
    return err
}
_ = roles
service.Roles.Reset()
```

`conn.UseExternalNetwork()`와 `conn.UseInternalNetwork()`는 HTTP 조회 없이 설정값을 반환합니다. 두 값 모두 false이면 Connection getter는 endpoint를 찾거나 네트워크를 조회하지 않고 빈 결과를 반환합니다. 한 값만 false인 경우에는 탐색이 진행되며 family별 역할 목록을 모두 분류합니다. 이 값은 각 목록을 필터링하는 옵션이 아닙니다. Connection getter는 network catalog endpoint가 없을 때도 빈 결과를 반환하지만, HTTP 403/404나 일반 endpoint 구성 오류를 빈 결과로 바꾸지 않습니다. 직접 `conn.Network(ctx)`를 호출하면 endpoint 구성 오류는 그대로 반환됩니다.

## 역할을 정하는 규칙

IPv4/IPv6는 실제 subnet의 IP 버전이나 현재 통신 가능 여부를 뜻하지 않습니다. 고정 Python 소스처럼 설정된 routing 역할과 네트워크 extension으로 분류합니다. 설정에서 family 역할을 지정하면 해당 selector의 이름 또는 ID에 일치하는 네트워크에 적용됩니다.

| 별도 역할 설정이 없는 네트워크 | IPv4 역할 | IPv6 역할 |
| --- | --- | --- |
| `router:external = true` | external | external |
| router external은 false이고 `provider:physical_network`가 비어 있지 않음 | external | internal |
| router external은 false이고 provider physical network도 없음 | internal | internal |

YAML의 `routes_externally`는 두 family의 기본값이며, `routes_ipv4_externally`와 `routes_ipv6_externally`가 있으면 각각 덮어씁니다. Typed `ConfiguredNetwork`에서는 `RoutesIPv4Externally`와 `RoutesIPv6Externally`를 별도로 지정합니다. false도 해당 family의 internal 역할을 명시한 값입니다.

- NAT source를 지정하지 않으면 router external 네트워크들이 `ExternalIPv4Floating`에 들어가고, 응답 순서의 첫 번째가 `NATSource`가 됩니다. Provider physical network만 있는 네트워크는 자동 Floating IP source가 아닙니다.
- `NATSource: true`인 설정 행이 여러 개이면 첫 설정 행의 selector를 사용합니다. 지정한 source는 Floating IP 목록의 유일한 항목이 됩니다. 설정된 네트워크가 실제 Floating IP 할당에 적합한지는 이 getter가 검증하지 않습니다.
- NAT destination을 지정하지 않으면 모든 subnet 페이지를 조회하고 비어 있지 않은 gateway IP가 있는 네트워크를 찾습니다. 네트워크 응답 순서의 마지막 후보가 `NATDestination`입니다. Gateway 존재만 확인하며 router 연결은 검증하지 않습니다.
- NAT destination을 지정하면 subnet 조회를 완전히 건너뜁니다. Default network는 `DefaultInterface` 설정으로만 정합니다. Gateway가 있다고 자동 기본 인터페이스로 삼지 않습니다.

NAT source, NAT destination, default interface selector는 전체 네트워크 목록에서 정확한 이름 **또는** ID로 찾습니다. 이름 중복이나 한 리소스의 ID와 다른 리소스의 이름이 일치하는 경우 단일 값을 임의로 고르지 않고 `resource.AmbiguousError`를 반환합니다. 설정된 family 목록의 selector도 이름 또는 ID를 지원하되, 같은 이름의 여러 네트워크를 목록에 포함할 수 있습니다. 고정 Python 소스는 초기 분류에서 ID를 허용하지만 family 목록의 마지막 검증은 이름만 비교하므로, Go는 이 검증도 ID를 지원하도록 일관되게 처리합니다.

네트워크와 필요한 subnet의 모든 페이지를 읽습니다. 빈 중간 페이지에 다음 링크가 있으면 계속 진행하며, 다음 링크 순환·HTTP 실패·decode 실패는 오류로 반환합니다. NAT destination을 지정하지 않은 경우에는 family 목록 getter를 호출하더라도 전체 snapshot을 만들기 위한 subnet 조회가 필요할 수 있습니다.

## 설정 교체와 서버 기본 NIC

`sdk.WithNetworkRoles(options...)`는 YAML에서 받은 네트워크 행과 discovery flag를 **정책 전체 단위로 교체**합니다. Typed 옵션에서 생략한 discovery flag는 true이며 생략한 행은 빈 목록입니다. `sdk.WithNetworkRoles()`는 YAML 역할 설정을 제거하고 기본 탐색 정책을 사용합니다. `network.WithConfiguredNetworks()` 역시 행 목록을 비웁니다. 여러 옵션에서는 같은 설정의 마지막 값이 적용되지만 잘못된 앞선 옵션을 뒤의 옵션으로 숨길 수는 없습니다.

`network.WithExternalNetworkDiscovery(false)`와 `network.WithInternalNetworkDiscovery(false)`로 탐색 사용 여부를 지정할 수 있습니다. `network.PrepareNetworkRoleOptions(...)`는 입력을 검증하고 복사한 `NetworkRolePolicy`를 만듭니다. 옵션 factory도 입력 slice를 복사하므로 호출자가 이후 원본 행을 바꾸어도 저장된 정책은 바뀌지 않습니다. 기본 인터페이스와 NAT destination은 각각 한 행만 허용합니다.

Cloud 파일의 우선순위는 `secure.yaml` > 선택한 `clouds.yaml` 항목 > `clouds-public.yaml` profile입니다. 각 단계의 `networks`는 행을 병합하지 않고 목록 전체를 교체합니다. `networks: []`는 상속 목록을 지우며 `networks: null`은 설정 오류입니다. 인증과 네트워크 정책은 Connect 시점에 읽은 동일한 파일 bytes를 사용합니다. 이후 파일 변경이 기존 Connection의 정책을 바꾸지 않습니다. Typed 정책을 제공해도 잘못된 YAML 설정 자체를 숨기지는 않습니다.

네트워크 flag는 YAML boolean 또는 문자열을 받습니다. 문자열은 대소문자를 무시한 정확한 `"true"`만 true이며, `"false"`를 비롯한 다른 문자열은 false입니다. 문자열 공백은 제거하지 않습니다. null은 false, 숫자·목록·객체는 설정 오류입니다. 특히 `use_external_network: "false"`는 Go에서 false입니다. 고정 Python `NetworkCommonCloudMixin`은 이 상위 flag를 raw 값으로 읽으므로 비어 있지 않은 문자열 `"false"`가 truthy하게 동작할 수 있습니다. 의도한 동작을 위해 YAML boolean을 권장합니다.

기존 `external_network`·`internal_network` 설정도 지원하지만 `networks`와 함께 지정하면 오류입니다. Legacy external은 두 family의 external과 기본 인터페이스, legacy internal은 두 family의 internal과 NAT destination으로 처리합니다. Legacy external이 NAT source를 명시하는 것은 아닙니다.

`DefaultInterface`는 서버 생성의 기본 NIC selector로도 전달됩니다. 서버의 명시적 `WithNetworks`·`WithNetworkInterfaces`·`WithNetworkMode`가 가장 우선하며, `sdk.WithDefaultNetwork` 또는 `sdk.WithoutDefaultNetwork`가 이 정책의 default selector보다 우선합니다. 선택할 default가 없으면 Nova의 선택된 microversion 2.37 이상에서 `auto`, 그 이전에는 networks 생략 동작을 사용합니다.

YAML 또는 typed-role configured default를 사용하는 `Servers.Create`는 getter와 같은 성공 snapshot의 `DefaultNetwork`를 사용합니다. 명시 기본 Ref·NIC·mode와 default selector가 없는 경로는 역할 조회를 우회합니다. `FloatingIPs.Ensure`의 zero external은 공유 floating 후보를 먼저 사용하고, 성공한 후보가 비어 있을 때 router gateway를 찾습니다. `Create`와 `Ensure`의 자동 NAT는 명시 port·fixed address·NAT destination이 없고 요청 서버 소유 port가 여러 개일 때만 공유 `NATDestination`으로 좁힙니다. 명시 destination 옵션은 추론 NAT 조회를 우회하지만 zero external의 source 조회까지 끄지는 않습니다.

`CreateWithFloatingIP`에서도 기본 NIC와 이후 source/NAT 선택이 같은 성공 snapshot을 사용할 수 있습니다. IP 후보·port 조회와 연결·할당은 실제 서버 ACTIVE 이후에 수행합니다. [Connection 소비 테스트](../connection_network_role_consumers_test.go), [Network 소비 테스트](floating_ip_roles_test.go), [복합 workflow 테스트](../connection_server_network_roles_test.go)에 이 경계를 기록했습니다. `WithoutDefaultNetwork`는 서버 기본 NIC만 비활성화하며 getter의 `DefaultNetwork` 설정을 지우지는 않습니다.

## 서버 생성에 같은 설정 사용하기

위 YAML의 private default와 public NAT source를 그대로 사용할 수 있습니다. Python에서는 역할 getter 후 cloud `create_server`가 같은 역할을 사용합니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
print(conn.get_default_network())
server = conn.create_server(
    name="web", image="ubuntu", flavor="c2", wait=True, auto_ip=True,
)
print(server.id)
```

Go의 아래 독립 예제도 설정에서 기본 NIC와 floating source/NAT를 선택합니다. `CreateWithFloatingIP`는 floating IPv4 연결을 명시적으로 요청하며, Python `auto_ip`의 기존 주소·private cloud 등에 따른 자동 생략을 아직 적용하지 않습니다. 서버와 IP의 실제 ACTIVE는 확인하지만 Nova 주소 수렴은 별도 남은 범위입니다. `ubuntu`·`c2`는 사용할 이미지와 flavor 이름으로 바꿉니다.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/compute"
    "gophercloudsdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
    if err != nil {
        return err
    }
    roles, err := conn.GetNetworkRoles(ctx)
    if err != nil {
        return err
    }
    if roles.DefaultNetwork != nil {
        fmt.Printf("default NIC: %s\n", roles.DefaultNetwork.ID)
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    result, err := service.Servers.CreateWithFloatingIP(ctx,
        compute.CreateServerWithFloatingIPRequest{
            Server: compute.CreateServerRequest{
                Name: "web",
                Image: resource.Name("ubuntu"),
                Flavor: resource.Name("c2"),
            },
            // FloatingIPNetwork를 생략하면 configured source를 사용합니다.
        },
    )
    if err != nil {
        if result != nil && result.Server != nil {
            fmt.Printf("known server: %s\n", result.Server.ID)
        }
        if result != nil && result.Assignment != nil && result.Assignment.FloatingIP != nil {
            fmt.Printf("known floating IP: %s\n", result.Assignment.FloatingIP.ID)
        }
        return err
    }
    fmt.Printf("server=%s floating IPv4=%s\n",
        result.Server.ID, result.Assignment.FloatingIP.FloatingIP)
    return nil
}
```

선행 getter는 필수 호출이 아닙니다. 생략하면 configured 기본 NIC 선택이 처음 탐색하고 이후 source/NAT 선택이 그 성공 cache를 재사용합니다. [상위 네트워크 CRUD](network-mutations.md)의 접수된 생성·수정·삭제는 같은 cache를 자동 초기화합니다. Raw/native/API 호출이나 외부 변경 뒤에는 `conn.ResetNetworkRoles()`를 호출합니다. 이 예제는 실제 리소스를 생성하는 사용법이며, 문서 검증에서는 컴파일과 로컬 HTTP fixture를 확인합니다.

## cache, 복사, Reset과 오류

성공한 전체 탐색만 Connection의 network service에 저장하며 TTL로 자동 갱신하지 않습니다. 개별 getter를 이어서 호출하면 같은 성공 결과를 사용합니다. 네트워크/subnet HTTP 오류, 설정 selector 불일치, decode 오류, 취소된 탐색은 저장하지 않으므로 이후 호출이 새로 시도할 수 있습니다. 고정 Python 소스가 네트워크 목록 오류를 빈 cache 상태로 남기거나 subnet 오류를 빈 목록으로 처리하는 것과 달리, Go는 오류를 호출자에게 보여줍니다.

반환한 snapshot의 목록, 네트워크 모델, native `Subnets`·`Tags`·`AvailabilityZoneHints` slice는 호출자가 소유합니다. 값을 수정해도 cache나 다른 호출자의 결과는 바뀌지 않습니다. 이 native slice 필드의 JSON null과 명시적 빈 배열 `[]` 차이도 복사 후 유지됩니다. 한 snapshot의 목록과 scalar가 같은 네트워크를 가리키더라도 각각 독립적으로 복사되므로 수정이 서로 전파되지 않습니다.

동시 조회는 진행 중인 탐색 하나를 기다립니다. 대기자의 context 취소는 해당 대기자만 종료하며 먼저 시작한 탐색을 취소하지 않습니다. 먼저 시작한 호출 자체가 취소되면 해당 탐색은 실패하고 cache를 남기지 않습니다. 이미 cache가 있어도 nil context나 취소된 context를 정상 조회로 취급하지 않습니다.

`conn.ResetNetworkRoles()`와 `service.Roles.Reset()`은 HTTP 요청 없이 성공 cache를 무효화합니다. Reset은 진행 중인 호출을 취소하지 않습니다. Reset 전에 시작한 탐색은 원래 호출자에게 결과를 반환할 수 있으나 그 결과를 새 cache에 넣지 못하며, 후속 getter는 기존 탐색이 끝난 뒤 새로 탐색합니다. Python의 대응 cache 초기화 메서드 `_reset_network_caches()`는 private 메서드이고 Go는 이를 public API로 제공합니다.

## openstacksdk 비교 범위

비교 기준은 openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [NetworkCommonCloudMixin 역할 분류와 getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L95-L432), [CloudRegion의 역할 설정 selector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/cloud_region.py#L1427-L1495)입니다. Source의 family 분류, aggregate 중복, NAT 선택 순서를 따르면서 오류 전파·호출자 소유 복사·취소·Reset의 동시성 계약을 Go API로 명시합니다.

[상위 네트워크 CRUD](network-mutations.md)와 기존 `service.Networks.Delete`의 접수 후 cache hook은 구현했습니다. Raw/native/API·외부 변경 뒤에는 명시 Reset이 필요합니다. [서버 주소 view](../compute/server-addresses.md)는 같은 cache와 concrete private/IPv6/source 설정으로 주소 선택·기존 association 보충을 제공합니다. 전체 `has_service`·session 정책, 생성·대기 lifecycle에 주소 확장을 자동 적용하는 작업, 자동 Floating IP 필요 판단·생략·Nova 주소 수렴은 후속 범위입니다. 기본 NIC·source/NAT 소비와 명시적 `CreateWithFloatingIP`만으로 cloud `create_server` 전체 동작의 동등성을 주장하지 않습니다. 위 비교는 고정 소스에 대한 확인 범위이며 실제 클라우드의 통신 가능성이나 전체 Python parity를 검증한 결과는 아닙니다.

[Network 계약 테스트](roles_test.go)와 [Connection 통합 테스트](../connection_network_roles_test.go)는 분류·페이지·오류·동시 탐색·cache/Reset·설정과 반환값 소유권을 로컬 HTTP fixture로 검증합니다. 전체 검사와 위 독립 Go 예제의 컴파일 결과는 [지원 판정대장](../docs/sdk-support-ledger.md#공유-네트워크-역할-조회와-설정)에 기록했습니다. 인증된 OpenStack 또는 Python 예제 실행 결과는 포함하지 않습니다.
