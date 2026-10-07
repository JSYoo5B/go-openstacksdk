# Floating IP 재사용과 서버 연결

`service.FloatingIPs.Ensure`는 서버의 fixed IPv4에 floating IP를 연결합니다. 기본적으로 같은 대상에 이미 연결된 IP를 사용하고, 없으면 사용 가능한 IP를 재사용하며, 그것도 없으면 새 IP를 할당합니다. Connection이 서버 이름 해석과 서비스 연결을 처리하므로 애플리케이션에서 builder나 resolver를 구현할 필요가 없습니다.

새 IP를 항상 생성하는 기존 `FloatingIPs.Create`는 그대로 사용합니다. Ensure는 기존 서버에 대한 연결 작업이며 서버의 ACTIVE 상태나 포트 생성을 기다리는 메서드는 아닙니다. 사용할 서버의 포트가 준비되어 있어야 합니다.

조회와 연결을 나누면서 선택한 대상이 바뀌지 않게 하려면 [PrepareEnsure·EnsurePrepared](floating-ip-plan.md)를 사용합니다. read-only 준비에는 reuse owner 조회나 IP mutation이 없고, 실행 시 owner를 확인하고 선택한 port를 다시 검증합니다. 기존 Ensure의 owner 선확인과 한 호출 동작은 유지합니다.

독립 [AddIPsToServer·AddIPList](../compute/server-ip-helpers.md)는 별도 기본60초·비동기 entry입니다. 선택적 wait는 서버/IP ACTIVE 없이 raw 목표 주소를 확인하며, 이 문서의 직접 API나 기존 상위 readiness 조건을 바꾸지 않습니다.

## Python과 Go의 대응

| 고정 openstacksdk cloud 호출 | gophercloudsdk |
|---|---|
| `conn.add_ips_to_server(server, ip_pool="public", reuse=True)` | `Ensure(ctx, EnsureFloatingIPRequest{Server: ref, Network: resource.Name("public")})` |
| `conn.add_auto_ip(server, reuse=True)` | Network를 생략한 Ensure의 자동 외부 네트워크 선택 |
| `reuse=False` | `network.WithEnsureReuse(false)` |
| `fixed_address="10.0.0.10"` | `network.WithEnsureFixedAddress("10.0.0.10")` |
| `nat_destination="private"` | `network.WithEnsureNATDestination(resource.Name("private"))` |
| port를 먼저 선택하여 연결 | `network.WithEnsurePort(resource.ID("port-id"))` 또는 정확한 port 이름 |
| `wait=True, timeout=60` | `network.WithEnsureWait(resource.WithTimeout(time.Minute))`; 관측 대상 차이는 아래 설명 |

Python은 server Resource 또는 주소 문자열을 반환하는 흐름을 사용합니다. Go는 `*network.FloatingIPAssignment`를 반환하고 그 안의 `FloatingIP`, `Reused`, `Allocated`를 제공합니다. 호출 형태의 대응이며 전체 cloud 동작이 동일하다는 뜻은 아닙니다.

## 실행 예제

예제의 cloud·server·외부/내부 네트워크 이름을 환경에 맞게 바꿉니다. 재사용의 기본 owner는 기록된 Keystone 인증 project이므로 project scope가 있는 `dev` cloud를 준비합니다.

Python cloud 예제:

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ips_to_server(
    server,
    ip_pool="public",
    nat_destination="private",
    reuse=True,
    wait=True,
    timeout=60,
)
print(server.interface_ip)
```

독립 Go 예제:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "gophercloudsdk"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	assignment, err := service.FloatingIPs.Ensure(ctx, network.EnsureFloatingIPRequest{
		Server: resource.Name("web-01"), Network: resource.Name("public"),
	}, network.WithEnsureNATDestination(resource.Name("private")),
		network.WithEnsureWait(resource.WithTimeout(time.Minute)))
	if err != nil {
		if assignment != nil && assignment.FloatingIP != nil {
			log.Printf("floating IP remains: id=%s reused=%t allocated=%t",
				assignment.FloatingIP.ID, assignment.Reused, assignment.Allocated)
		}
		return err
	}
	floating := assignment.FloatingIP
	fmt.Println(floating.ID, floating.FloatingIP, floating.PortID, floating.FixedIP, floating.Status)
	fmt.Println("reused", assignment.Reused, "allocated", assignment.Allocated)
	return nil
}
```

서버 ID를 이미 알고 있다면 `Server: resource.ID(serverID)`로 Nova 이름 조회를 생략합니다. port와 network의 ID도 명시할 수 있습니다. `resource.Name`은 정확한 이름을 조회하며 UUID 형태의 이름도 ID로 추정하지 않습니다.

## 재사용 owner와 선택 순서

재사용 기본값은 true입니다. owner를 지정하지 않으면 ProviderClient에 기록된 Keystone v3 project 또는 v2 token tenant를 읽습니다. project를 읽기 위한 HTTP 요청이나 재인증을 추가하지 않고 endpoint URL·token 문자열·서버 이름에서 owner를 추정하지 않습니다.

manual token을 사용하는 adopted provider, system/domain/unscoped 인증 등에서 recorded project를 얻을 수 없으면 재사용을 시작하기 전에 `ErrUnsupported`를 반환합니다. `network.WithEnsureProject("project-id")`로 owner를 명시할 수 있습니다. 이 옵션은 재사용 목록의 project 범위와 신규 allocation의 `project_id`를 설정하며 인증 자체를 바꾸지는 않습니다. 다른 project를 지정할 권한은 Neutron이 검사합니다.

`WithEnsureReuse(false)`는 사용 가능한 IP 목록과 recorded-project 해석을 건너뛰고 새 allocation을 요청합니다. 이때 명시 owner가 있으면 이를 보내고, 없으면 Neutron의 인증 scope에 따른 기본 할당을 사용합니다. 같은 설정의 옵션은 마지막 값이 우선하며 잘못된 옵션은 이후 옵션으로 숨기지 않습니다.

재사용 목록은 owner와 floating network query를 전송하고 **모든 페이지를 읽은 뒤** 선택합니다. 빈 JSON 페이지에도 next 링크가 있으면 계속 읽고, 204는 종료 응답으로 처리합니다. 반복 링크는 다시 조회하기 전에 오류로 반환합니다. 실제 응답에서도 project/tenant 일관성, 외부 network, 유효한 floating IPv4, 연결 대상을 다시 확인합니다. `ERROR` 상태, 다른 owner/network, IPv6, 다른 port/fixed IP에 연결된 IP는 재사용 후보가 아닙니다.

선택은 다음 순서입니다.

1. 같은 port와 fixed IPv4에 이미 연결된 첫 번째 IP. 뒤 페이지에 있어도 free IP보다 우선하며 PUT을 생략합니다.
2. 응답 순서의 첫 번째 unattached IP. 선택한 port/fixed IPv4로 PUT합니다.
3. 후보가 없을 때 새 IP를 port/fixed IPv4와 함께 POST합니다.

IP 선택의 성공·실패 결과를 cache하지 않습니다. 서버 port와 가용 IP 후보는 호출마다 조회하고, 네트워크 역할의 성공 snapshot만 공유합니다. 뒤 페이지 오류가 있으면 앞 페이지에 후보가 있어도 연결이나 새 할당을 진행하지 않습니다. `Reused`는 이미 같은 대상에 연결된 경우도 포함하며 `Allocated`는 새 allocation이 접수된 경우를 나타냅니다.

## 외부 네트워크와 목적지 포트

`EnsureFloatingIPRequest.Network`를 지정하면 이름은 `router:external=true` 목록의 정확한 이름과 실제 external flag를 확인합니다. ID는 추가 network 조회 없이 전달하며 Neutron이 allocation 시 외부 network 조건을 검사합니다.

Network가 zero Ref이면 [공유 역할 snapshot](network-roles.md)의 `ExternalIPv4Floating`에서 첫 후보를 선택합니다. Configured NAT source가 있으면 그 네트워크가 유일한 floating 후보이며, 없으면 router external 네트워크들이 후보입니다. Provider physical network만 있는 네트워크는 자동 floating source가 아닙니다. **성공한 역할 목록이 비어 있을 때만** enabled router의 첫 external gateway를 찾습니다. 두 discovery flag가 모두 false여서 역할 조회를 건너뛴 경우에도 router fallback은 별도로 실행합니다. 어느 쪽도 없으면 `ErrNotFound`이며, 역할·subnet 오류를 router나 Nova fallback으로 바꾸지 않습니다.

`WithEnsurePort`, `WithEnsureFixedAddress`, `WithEnsureNATDestination`을 명시하면 추론 NAT 조회를 우회하고 명시 입력의 제약을 적용합니다. Network가 zero이면 자동 floating source 조회는 여전히 필요합니다. 어느 destination 옵션도 없고 전체 목록에서 요청 서버 소유 port가 여러 개이면 공유 `NATDestination`의 network로 좁힙니다. IPv4 필터링 전에 port 개수를 세므로 IPv6 전용 두 번째 port도 이 분기를 켭니다. NAT 역할이 없으면 `ErrAmbiguous`, 선택된 network에 IPv4 대상이 없으면 `ErrNotFound`이며 할당·연결하지 않습니다. 서버 소유 port가 하나이면 추론 NAT 조회를 하지 않습니다. 다른 서버의 port는 개수와 후보에서 제외합니다.

범위를 좁힌 뒤에도 유효한 `(port, fixed IPv4)` 쌍이 여러 개이면 `ErrAmbiguous`입니다. 명시 port도 서버 소유인지 확인하고, 명시 NAT·fixed·port의 조건은 함께 적용합니다. IPv6와 malformed fixed address 옵션은 검증에서 거부합니다. 같은 자동 NAT 규칙은 `FloatingIPs.Create(..., WithServer(...))`에도 적용됩니다.

Python의 `_nat_destination_port`는 최근 port와 첫 IPv4를 선택하는 분기가 있지만 Go는 모호한 선택을 오류로 반환합니다. network·router·port 선택도 빈 중간 페이지의 next 링크를 따라가며 뒤 페이지 오류가 있으면 mutation 전에 종료합니다. 외부 allocation network와 private NAT destination은 서로 다른 입력이며 서버 생성의 default network와 자동으로 같은 값이 되지 않습니다.

## revision 조건과 실패 결과

재사용 목록에 `revision_number`가 있으면 association PUT에 `If-Match: revision_number=N`을 보냅니다. 명시한 0도 보존하고 누락/null이면 조건을 보내지 않습니다. 음수 revision은 오류입니다. 이는 관측한 revision으로 변경을 보호하는 것이며 IP 선택에 별도의 전역 lock이나 예약을 추가하지 않습니다. revision이 없는 경우의 List → PUT은 원자적 claim을 보장하지 않습니다.

목록 실패, association 409, revision 412 등은 원래 HTTP 오류를 보존합니다. 실패했다고 다른 IP를 자동 선택하거나 새 allocation/Nova fallback으로 전환하지 않습니다. 자동 DELETE, detach, server 삭제도 하지 않습니다.

| 결과 시점 | 반환값 |
|---|---|
| 입력·owner·network·destination·목록 선택 실패 | assignment nil과 error |
| 기존 IP를 선택한 뒤 PUT·응답 검증·대기 실패 | 선택한 IP와 `Reused: true`, error |
| 새 allocation 후 응답 검증·대기 실패 | 알려진 allocation과 `Allocated: true`, error |
| 모두 성공 | 검증한 IP와 Reused/Allocated metadata |

PUT 실패나 다른 리소스의 응답 때문에 선택한 IP ID를 버리지 않습니다. 대기 실패 때의 반환 IP는 마지막으로 검증한 선택/연결 결과이며 최신 실패 상태가 반영되었다고 가정하지 않습니다. 오류가 있을 때도 먼저 반환값을 확인하면 caller가 알려진 리소스를 관리할 수 있습니다. transport 실패로 생성 결과 자체를 얻지 못한 경우에는 확인하지 못한 IP ID를 만들어 반환하지 않습니다.

## WithEnsureWait의 상태 계약

기본 호출은 ACTIVE 대기를 하지 않습니다. ID·external network·owner·floating IPv4·port/fixed IPv4를 검증한 association 응답을 반환하며 상태가 `DOWN`일 수 있습니다. `network.WithEnsureWait(...)`를 사용하면 같은 IP가 ACTIVE가 될 때까지 공통 waiter로 조회하고 다시 ID와 association을 검증합니다. wait attribute 같은 공통 옵션을 바꾸더라도 최종 IP `Status == "ACTIVE"` 조건은 유지합니다.

timeout, context 취소, 실패 상태, poll HTTP 오류, 다른 ID/owner/destination 응답은 IP와 error를 함께 반환합니다. 이 wait는 **Neutron IP의 ACTIVE**를 확인하며 Nova server의 `addresses`·`interface_ip` 갱신이나 접속 가능성까지 기다리지 않습니다. Python `_attach_ip_to_server(wait=True)`가 서버를 다시 조회해 주소 출현을 기다리는 흐름과 다릅니다.

일반 caller는 With 옵션만 전달하면 됩니다. `PrepareEnsureFloatingIPOptions`와 `WithEnsureFloatingIPPolicy`는 서버 생성 같은 SDK workflow가 옵션을 사전 검증하고 재사용하기 위한 선택적인 경로이며 builder 구현을 요구하지 않습니다.

## 고정 소스와 남은 범위

비교 대상은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [project 범위와 available IP 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L671), [auto IP 흐름](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394-L1487), [포트 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1709), [서버 주소 대기](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1163)입니다. Python은 일부 Neutron NotFound에서 Nova fallback을 하고 새 IP의 activation timeout에서 삭제를 시도하지만 Go Ensure는 오류와 알려진 리소스를 보존합니다.

직접 Ensure는 automatic 필요성 판정이나 raw Nova 주소 관측을 수행하지 않습니다. [Compute IP dispatch](../compute/server-ip-dispatch.md)가 기존 automatic 판단과 Neutron pool·순차 명시 IPv4 소비자를 연결하고 기존 Ensure/Get/Wait/Create의 Neutron 동기 작업에서 IP ACTIVE와 raw Nova 목표 주소를 확인합니다. 전체 `has_service`/service/config/session 정책, Nova의 별도 공개 CRUD·detach/cleanup·함수별 fallback, standalone cloud IP helper의 전체 Resource 입력·반환 정규화은 남은 범위입니다. [상위 network CRUD](network-mutations.md)의 공유 cache hook은 구현했으며 다른 mutation 경로의 hook은 남습니다. `create_server`의 전체 wait/response/cleanup 및 다른 생성 옵션까지 동일하게 구현했다고 판정하지 않습니다. Ensure와 Compute 소비자의 부분 계약을 이 메서드들의 전체 지원으로 올리지 않습니다.

[역할 getter와 공유 cache](network-roles.md)는 기본 NIC·자동 floating source·조건부 NAT 선택에 함께 쓰입니다. [상위 network CRUD](network-mutations.md)의 접수된 변경은 cache를 자동 초기화합니다. Raw/native/API 또는 외부 변경은 `ResetNetworkRoles()` 또는 `service.Roles.Reset()`을 호출해 반영합니다. 성공한 역할만 cache하며, IP 후보와 port 선택 결과는 cache하지 않습니다. [역할 소비 테스트](floating_ip_roles_test.go)는 source override·cache/Reset·명시 우회·다중 port/NAT·IPv6 전용 port·오류/취소·빈 역할 router fallback을 실제 요청과 mutation 여부로 검증합니다.

[Ensure HTTP 테스트](floating_ip_ensure_test.go), [선택·대기 테스트](floating_ip_selection_test.go), [Connection 통합 테스트](../connection_floating_ip_ensure_test.go)는 다중·빈 페이지와 반복 링크, owner/local 제약·revision presence, 이미 연결된 IP·새 allocation, 실패 후 보존·fallback 금지·취소·timeout과 현재 인증 scope를 검증합니다. Python 비교는 고정 소스 확인이며 Python 예제나 인증된 OpenStack 실행을 검증한 것은 아닙니다. 실행 근거는 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록합니다.

[Compute legacy Nova backend](../compute/server-nova-floating-ip.md)는 IP list·pool/할당·add action을 직접 소비하고 별도 NovaAssignment를 반환합니다. 이 Network Ensure나 executable Neutron plan을 Nova 모델로 변환하지 않습니다.
