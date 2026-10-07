# Floating IP 선택 계획과 실행

`FloatingIPs.PrepareEnsure`는 floating IPv4를 연결할 기존 서버와 외부 network·port·fixed IPv4를 먼저 선택합니다. `EnsurePrepared`는 그 선택을 유지하여 재사용 또는 allocation을 실행합니다. Connection이 서비스를 연결하므로 builder나 resolver interface를 애플리케이션에서 구현하지 않습니다.

| 호출 | 하는 일 |
|---|---|
| `service.FloatingIPs.PrepareEnsure(ctx, request, opts...)` | context·입력 검증, 외부 network와 server-owned fixed IPv4 선택; `(network.FloatingIPPlan, error)` |
| `plan.Selection()` | 선택한 concrete ID와 주소의 복사본 반환 |
| `service.FloatingIPs.EnsurePrepared(ctx, plan)` | 실행 시 owner 준비, 선택한 port 재검증, 기존 IP 재사용/새 allocation·선택적 ACTIVE 대기 |
| 기존 `service.FloatingIPs.Ensure(ctx, request, opts...)` | 한 호출로 연결하는 기존 흐름; 재사용 owner를 선택 조회 전에 준비 |

계획 준비에는 recorded project 조회, floating IP 후보 목록, POST·PUT·DELETE가 없습니다. 인증된 unscoped provider도 **선택할 network와 port를 조회할 권한이 있으면** 준비할 수 있습니다. project owner가 필요한 실행은 별도입니다. PrepareEnsure는 서버 ACTIVE나 port 생성을 기다리지 않으며, 준비된 기존 서버의 port가 필요합니다.

## Python cloud와 비교

비교 소스는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. Python public cloud에는 같은 선택을 opaque plan으로 고정하는 두 단계 API가 없습니다. [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1525)는 pool → 명시 IP 목록 → auto+needs 순서로 연결 작업을 선택합니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ips_to_server(
    server,
    ip_pool="public",
    reuse=True,
    wait=True,
    timeout=60,
)
print(server.interface_ip)
```

[needs 검사](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1527-L1606)와 이후 연결은 network/port 선택을 다시 수행할 수 있습니다. Go 계획은 한 번 선택한 ID·주소를 실행까지 보존하여 이 사이에 다른 대상을 고르는 일을 막는 기반입니다. Python의 [public add_auto_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394-L1424)는 needs 검사 없이 연결을 요청하고 interface 주소 문자열을 반환하므로 조건부 auto branch와 구분합니다.

이 가이드의 API는 **Neutron 선택·실행 기반**입니다. Compute의 기존 public/floating/fixed 주소·private/source/서비스 조건에 따른 자동 필요성 판단, 조건부 dispatch, raw Nova 주소 수렴은 아직 이 메서드의 기능이 아닙니다. Python 예제의 전체 반환 Server/Resource·pool·기본값·cleanup 동작에 대한 일대일 완료를 뜻하지 않습니다.

## 독립 Go 예제

프로그램은 기본적으로 계획만 준비하고 선택 결과를 출력합니다. `-execute`를 지정하면 같은 계획으로 연결합니다. `-server-id`에는 기존 Nova 서버 ID를 넣습니다. `-network`가 비어 있으면 공유 floating 역할과 router gateway를 사용하고, 이름을 지정하면 정확한 외부 network 이름을 조회합니다. 예제는 실행 시 Neutron IP ACTIVE까지 대기하도록 옵션을 준비합니다.

```go
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "gophercloudsdk"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	serverID := flag.String("server-id", "", "existing Nova server ID")
	external := flag.String("network", "", "exact external network name; empty uses roles/router")
	project := flag.String("project-id", "", "explicit reuse/allocation owner; empty uses recorded scope at execution")
	reuse := flag.Bool("reuse", true, "reuse an attached/free floating IP before allocating")
	execute := flag.Bool("execute", false, "execute the prepared assignment")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *serverID, *external, *project, *reuse, *execute); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, serverID, external, project string, reuse, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	request := network.EnsureFloatingIPRequest{Server: resource.ID(serverID)}
	if external != "" {
		request.Network = resource.Name(external)
	}
	options := []network.EnsureFloatingIPOption{
		network.WithEnsureReuse(reuse),
		network.WithEnsureWait(resource.WithTimeout(time.Minute)),
	}
	if project != "" {
		options = append(options, network.WithEnsureProject(project))
	}
	plan, err := service.FloatingIPs.PrepareEnsure(ctx, request, options...)
	if err != nil {
		var absence *network.FloatingIPPlanUnavailableError
		if errors.As(err, &absence) {
			log.Printf("no prepared target: %s", absence.Reason)
		}
		// 진단을 출력해도 원래 오류와 함께 온 다른 원인을 숨기지 않습니다.
		return fmt.Errorf("prepare floating IP: %w", err)
	}
	selection := plan.Selection()
	fmt.Printf("server=%s external=%s port-network=%s port=%s fixed4=%s\n",
		selection.ServerID, selection.NetworkID, selection.PortNetworkID,
		selection.PortID, selection.FixedIPv4)
	if !execute {
		return nil
	}

	assignment, err := service.FloatingIPs.EnsurePrepared(ctx, plan)
	if assignment != nil {
		fmt.Printf("reused=%t allocated=%t\n", assignment.Reused, assignment.Allocated)
		if assignment.FloatingIP != nil {
			ip := assignment.FloatingIP
			fmt.Printf("ip-id=%s floating4=%s port=%s fixed4=%s status=%s\n",
				ip.ID, ip.FloatingIP, ip.PortID, ip.FixedIP, ip.Status)
		}
	}
	if err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response evidence: status=%d", response.StatusCode)
		}
		return fmt.Errorf("execute floating IP: %w", err)
	}
	return nil
}
```

`Selection()`에는 `ServerID`, `NetworkID`, `PortNetworkID`, `PortID`, `FixedIPv4`가 있습니다. `NetworkID`는 외부 allocation network, `PortNetworkID`는 실제 목적지 port의 network입니다. 반환 구조체를 수정해도 plan은 바뀌지 않습니다. 직접 만든 selection을 실행 API에 전달하거나 plan 내부 옵션을 다시 지정하지 않습니다.

## 외부 network·port 선택과 typed absence

Network가 zero Ref이면 같은 Connection의 [공유 역할 snapshot](network-roles.md)에서 첫 floating IPv4 network를 선택하고, 성공한 역할 결과가 비었을 때만 enabled router의 첫 external gateway를 선택합니다. 외부 network 이름은 정확한 이름과 실제 external flag를 전체 목록에서 확인합니다. 명시 ID는 추가 network 조회를 생략하며 Neutron이 mutation 조건을 검사합니다.

목적지는 서버 소유 port와 유효한 fixed IPv4를 선택합니다. 명시 port·fixed·NAT destination 옵션을 적용하고, 별도 destination 제약이 없는 여러 server-owned port는 공유 NAT 역할로 좁힙니다. 범위 안에 여러 `(port, fixed IPv4)` 쌍이 있으면 `ErrAmbiguous`입니다. IPv6 전용 port도 여러 port인지 판단할 때 개수에 포함합니다. Python의 최근 port/첫 IPv4 선택과 달리 Go는 모호한 선택을 확정하지 않습니다.

| `FloatingIPPlanUnavailableError.Reason` | 만들어지는 근거 |
|---|---|
| `NoFloatingIPExternalNetwork` | 자동 shared 역할과 모든 router 조회가 성공했지만 외부 allocation network가 없음 |
| `NoFloatingIPServerPorts` | 전체 자동 device_id port 목록 조회가 성공했지만 요청 서버의 port가 없음 |
| `NoFloatingIPFixedMatch` | 서버 port 목록은 있지만 지정 fixed IPv4가 없고, 명시 port/NAT 제약 없이 전체 후보를 검사함 |

이 타입은 완성된 탐색의 대상 부재를 표현합니다. 일반 `errors.Is(err, resource.ErrNotFound)`에는 명시 이름·port의 미존재, HTTP 404 등도 포함되므로 이것만으로 자동 skip을 결정하지 않습니다. typed absence에 context 취소나 source 오류가 함께 있으면 그 오류도 처리해야 합니다. 예제는 reason을 출력하되 전체 오류를 반환합니다.

HTTP/decode/read/Close·뒤 페이지 오류, 설정 오류, source 변경, 취소, 모호한 선택은 absence로 숨기지 않습니다. nonempty IPv6-only inventory나 명시 NAT 범위의 빈 결과도 일반 오류입니다. 명시 port/NAT 조회·소유권·제약의 불일치는 오류입니다. port/NAT 제약 없이 explicit fixed를 찾는 완료된 탐색의 부재만 위 `NoFloatingIPFixedMatch`로 구분합니다.

## 실행 시 owner와 목적지 재검증

PrepareEnsure는 reusable project owner를 읽지 않습니다. `EnsurePrepared`가 실행을 시작할 때 `WithEnsureProject`의 명시 owner 또는 그 시점 ProviderClient의 기록된 Keystone project/tenant를 사용합니다. reuse=true인데 owner를 얻을 수 없으면 `ErrUnsupported`이며 port 재조회나 floating IP 요청 전에 종료합니다. 별도의 HTTP project 조회나 재인증을 추가하지 않습니다. 미리 owner를 고정한 `PrepareEnsureActive` 정책을 전달한 경우 그 owner를 유지합니다.

reuse=false이면 가용 IP 목록을 생략합니다. 명시 owner가 있으면 allocation에 보내고 없으면 Neutron 인증 scope의 기본값을 사용합니다. 변경 가능한 인증 결과와 선택한 port/외부 network는 서로 다른 정책이며 token scope가 바뀌었다고 plan이 다른 대상을 자동 선택하지 않습니다.

실행은 선택한 port를 GET하고 정확한 port ID·서버 DeviceID·port network ID·fixed IPv4의 존재를 다시 확인합니다. 사라진 port나 대상 변화는 오류이며 다른 port로 재선택하지 않습니다. 이 재검증은 서버 존재·router 연결성·프로젝트 권한을 미리 예약하거나 보장하는 동작은 아닙니다. 실제 mutation의 권한과 network 조건은 Neutron이 판단합니다.

plan은 준비한 동일 `FloatingIPs` 객체에 속합니다. zero plan이나 다른 service의 plan은 실행하지 못합니다. API·client·provider·endpoint·resource base·microversion·공개 collection/service 포인터의 변경을 감지하면 오류를 유지하며 이전 값으로 되돌려도 이미 거부된 plan을 되살리지 않습니다. plan은 원자적 예약·분산 lock·일회성 실행 token이 아니므로 반복/동시 실행의 중복 allocation 방지를 보장하지 않습니다.

## 재사용·revision·ACTIVE·부분 결과

실행 시 floating IP 후보의 owner/external network 목록을 모든 페이지에서 확인하고, 같은 port/fixed IPv4에 이미 붙은 후보를 첫 free IP보다 우선합니다. 같은 대상에 붙어 있으면 association PUT을 생략합니다. free IP를 사용하면 관측한 `revision_number`가 있는 경우 0도 포함하여 `If-Match: revision_number=N`으로 PUT합니다. 뒤 페이지 오류나 revision 충돌이 있으면 대체 allocation을 하지 않습니다.

후보가 없으면 선택한 network/port/fixed IPv4로 POST합니다. 새 allocation의 성공 코드는 native와 같이 201/202이며, 접수 이후 decode/read/Close·source·취소 오류가 생기면 `Allocated=true`인 assignment와 실제 response error를 함께 보존합니다. 유효한 body를 디코드할 수 있으면 read/Close·source·취소 오류와 함께 알려진 IP 모델을 반환합니다. malformed·truncated body로 모델을 확인할 수 없을 때만 `FloatingIP`가 nil일 수 있습니다. 재사용 PUT은 ID·owner·network·port·fixed IPv4 검증을 통과한 응답만 기존 후보를 대체하며, 접수 후 처리 오류가 있어도 검증한 최신 모델을 보존합니다. 잘못된 응답은 원래 후보를 유지합니다. 원래 처리 오류와 추가 decode·검증 원인을 함께 반환하고, 오류 뒤 대기·관측·재할당·DELETE를 보내지 않습니다.

`WithEnsureWait`가 없으면 실제 association 응답을 반환하며 `DOWN`일 수 있습니다. 옵션이 있으면 같은 IP의 실제 ACTIVE를 GET으로 관측하고 ID·owner·network·port·fixed IPv4를 검증합니다. 공통 waiter의 상태 attribute를 바꿔도 실제 IP Status가 ACTIVE여야 성공합니다. caller context는 전체 준비·실행을 제한하고, `resource.WithTimeout`은 waiter의 추가 제한입니다. 생성하거나 재사용한 IP를 오류 시 자동 DELETE하지 않습니다.

이 wait는 Neutron IP 상태이며 raw Nova server `addresses`에 새 floating IP가 나타나는지 확인하지 않습니다. [주소 view 보충](../compute/server-addresses.md)이 만든 Supplemental 행도 Nova 관측을 대신하지 않습니다. 기존 [Ensure](floating-ip-ensure.md), [CreateWithFloatingIP](../compute/create-with-floating-ip.md)의 공개 계약은 유지합니다.

## 조회 cache·HTTP 보호와 남은 범위

plan이 직접 실행하는 owned REST 목록·명시 Neutron lookup·port 재검증·mutation·waiter 요청은 source guard를 사용합니다. 성공 응답 처리 오류를 allocation 재시도로 바꾸지 않으며 retry/reauth/redirect 시 source·입력 변경을 검사합니다. 준비에서 시작하는 uncached role network/subnet 조회도 guarded REST를 사용하고 한 번 받은 역할 snapshot을 재사용합니다.

공유 role cache의 성공값·Reset·동시 discovery는 기존 정책을 따릅니다. 이미 일반 role discovery가 진행 중인 경우 그 결과를 기다려 공유할 수 있으며, 모든 기존 public getter/native/raw 요청이 이 plan의 보호 정책으로 바뀌는 것은 아닙니다. 서버 이름 조회는 기존 Connection resolver를 사용합니다. 환경의 외부 변경을 자동 탐지하거나 cache를 매번 강제로 새로 읽는 정책은 아니므로 필요한 topology 변경에는 [Reset 규칙](network-roles.md)을 적용합니다.

고정 source `_needs_floating_ip`가 일부 network SDKException을 false로 숨기고 이후 재선택하는 동작과 비교하여, Go는 명확한 완료 탐색 부재와 실패를 구분하고 concrete 선택을 유지합니다. 별도 [Compute 자동 메서드](../compute/server-automatic-ip.md)는 기존 서버의 조건부 Neutron 실행과 raw Nova 관측을 제공합니다. [자동 IP 서버 생성](../compute/create-with-automatic-floating-ip.md)과 [GetActive/Wait](../compute/server-ready.md)도 같은 Neutron 정책을 소비합니다. 일반 Create의 전체 source dispatch, pool/명시 IP 우선순위, Nova-network fallback, full has_service/session/Resource 모델 및 timeout cleanup은 계속 남습니다. 이번 plan 기반만으로 전체 cloud 연산을 지원 완료로 세지 않습니다.

[plan HTTP 테스트](floating_ip_plan_test.go)와 [설정·응답 경계 테스트](floating_ip_plan_boundaries_test.go)는 owner 지연·owned 선택·typed absence·목적지 재GET·전체 후보·revision·실제 ACTIVE·접수 후 부분 결과·source guard를 확인하도록 작성되어 있습니다. 테스트 실행 결과와 최종 예제 컴파일은 [지원 판정대장](../docs/sdk-support-ledger.md)에 실제 검증 revision과 함께 기록합니다. Python 비교는 고정 소스 정적 검토이며 Python 예제·실클라우드 실행 근거는 별도입니다.

[Compute 자동 floating IPv4](../compute/server-automatic-ip.md)는 `FloatingIPs.NewPlanner`의 한 guarded 역할 snapshot을 주소 분류와 대상 선택에 공유하고, 필요할 때 같은 plan을 실행합니다. `NewPlanner` 자체는 조회·owner 확인을 하지 않으며 `NetworkRoles`는 호출자가 소유하는 복사본을 반환합니다. 한 planner가 확보한 snapshot은 Reset 이후에도 유지되고 다음 planner는 새 cache를 읽을 수 있습니다. `WithEnsureActive`는 기존 wait 옵션을 유지하며 actual ACTIVE를 필수로 만들지만 owner를 미리 바인딩하지 않습니다. 원래 직접 Plan API에는 자동 필요성이나 raw Nova 관측 책임이 없습니다.

접수 후 IP 모델·오류 보존의 [후속 검증 기록](../docs/sdk-support-ledger.md#neutron-접수-응답의-ip-모델-보존)은 POST201/202·PUT200의 complete/invalid/truncated body와 서버 Get/Wait/Create/Ensure 부분 결과를 구분합니다.
