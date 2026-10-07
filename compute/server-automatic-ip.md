# 서버의 자동 floating IPv4와 Nova 주소 관측

`PlanServerFloatingIP`는 기존 서버의 주소·cloud 설정을 보고 floating IPv4가 필요한지 판단합니다. `EnsureServerFloatingIP`는 필요한 경우에만 Neutron 선택 계획을 실행하고, 연결한 IPv4가 실제 Nova 응답에 나타날 때까지 관측합니다. Connection이 설정과 서비스 의존성을 제공하므로 애플리케이션에서 builder나 resolver interface를 구현하지 않습니다.

| 호출 | 결과와 범위 |
|---|---|
| `conn.PlanServerFloatingIP(ctx, request, opts...)` | `*compute.ServerFloatingIPDecision`; 판단·필요한 경우 concrete Neutron 대상 준비, mutation 없음 |
| `conn.EnsureServerFloatingIP(ctx, request, opts...)` | `*compute.AutomaticServerIPResult`; skip 또는 같은 호출에서 판단·조건부 assignment·raw Nova 관측 |
| Compute service의 같은 두 메서드 | 같은 SDK 정책; Connection이 연결한 의존성과 설정을 사용 |

요청은 `compute.AutomaticFloatingIPRequest{Server: server, Network: ref}`입니다. Server는 기존 Nova 모델이며 nil일 수 없습니다. Network zero Ref는 공유 floating 역할·router gateway 선택을 사용하고, 명시 name/ID는 [Neutron 선택 계획](../network/floating-ip-plan.md)의 명시 외부 network 경로를 사용합니다.

독립 Plan 호출은 실행 token이나 opaque plan을 반환하지 않습니다. 반환 Decision을 다음 Ensure 호출에 전달하는 API도 없습니다. 두 public 호출은 각각 현재 정보를 판단하며, **한 Ensure 내부**에서 주소 역할 분류·destination 선택·실행이 같은 planner와 concrete tuple을 유지합니다. Network의 직접 `PrepareEnsure`/`EnsurePrepared`는 별도의 두 단계 API입니다.

## Python cloud와 비교

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1525)는 pool → 명시 IP 목록 → `auto_ip`+needs 순서로 작업을 선택합니다. 다음은 auto branch를 요청하는 예입니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None or server.status != "ACTIVE":
    raise RuntimeError("an existing ACTIVE server is required")
server = conn.add_ips_to_server(
    server,
    auto_ip=True,
    reuse=True,
    wait=True,
    timeout=180,
)
print(server.interface_ip)
```

Python의 [needs 검사](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1527-L1606)는 source·public/floating·fixed/private·cloud private·서비스와 외부 network/NAT 조건을 확인합니다. [attach wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1164)는 raw Compute 주소를 다시 읽습니다. Go는 이 중 **기존 서버의 bounded Neutron automatic branch**를 concrete 옵션과 부분 결과로 제공합니다. pool/명시 IP 우선순위와 Nova mutation, full cloud create/get_active/wait·Resource 모델은 별도 범위입니다.

Python [add_auto_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394-L1424)는 needs 검사 없이 연결을 요청하고 interface 주소 문자열/None을 반환합니다. Go의 조건부 메서드를 이 public 함수 전체와 일대일로 대응시켜 설명하지 않습니다.

## 독립 Go 예제

프로그램은 기존 서버를 먼저 Find합니다. 기본은 판단 결과만 출력하며, `-execute`를 주면 Ensure를 한 번 호출합니다. Plan을 먼저 호출한 뒤 별도 Ensure가 그 선택을 이어받는 것처럼 사용하지 않습니다. 이 예제의 최초 Find에는 Compute endpoint가 필요하며, 미리 받은 Server를 Connection helper에 직접 전달할 때의 lazy endpoint 생략과 구분합니다.

`-network`가 비면 공유 역할/router 선택을 사용합니다. `-project-id`는 reuse/allocation owner의 명시 override이며, 비면 실행 시 기록된 Keystone scope를 사용합니다. 기존 IP 재사용이 기본입니다. reachability probing은 예제에서 해제했고 cloud의 private/source 설정은 그대로 적용합니다.

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
	"gophercloudsdk/compute"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server", "web-01", "exact server name, or ID with -server-id")
	byID := flag.Bool("server-id", false, "treat -server as an ID")
	external := flag.String("network", "", "exact external network name; empty uses roles/router")
	project := flag.String("project-id", "", "explicit reuse/allocation owner")
	reuse := flag.Bool("reuse", true, "reuse an attached/free floating IP before allocation")
	enabled := flag.Bool("auto-ip", true, "enable conditional floating IPv4 assignment")
	execute := flag.Bool("execute", false, "assign when needed and observe actual Nova addresses")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *server, *byID, *external, *project, *reuse, *enabled, *execute); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, nameOrID string, byID bool, external, project string, reuse, enabled, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	ref := resource.Name(nameOrID)
	if byID {
		ref = resource.ID(nameOrID)
	}
	server, err := service.Servers.Find(ctx, ref)
	if err != nil {
		return err
	}
	request := compute.AutomaticFloatingIPRequest{Server: server}
	if external != "" {
		request.Network = resource.Name(external)
	}
	ipOptions := []network.EnsureFloatingIPOption{network.WithEnsureReuse(reuse)}
	if project != "" {
		ipOptions = append(ipOptions, network.WithEnsureProject(project))
	}
	options := []compute.AutomaticFloatingIPOption{
		compute.WithAutomaticIPEnabled(enabled),
		compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
		compute.WithAutomaticEnsureOptions(ipOptions...),
		compute.WithAutomaticIPTimeout(3 * time.Minute),
		compute.WithAutomaticIPPollInterval(time.Second),
		compute.WithAutomaticIPProgress(func(current *compute.Server) error {
			fmt.Printf("observing server=%s status=%s\n", current.ID, current.Status)
			return nil
		}),
	}
	if !execute {
		decision, err := conn.PlanServerFloatingIP(ctx, request, options...)
		printDecision(decision)
		return err
	}

	result, err := conn.EnsureServerFloatingIP(ctx, request, options...)
	if result != nil {
		printDecision(result.Decision)
		fmt.Printf("observed=%t\n", result.Observed)
		if result.Server != nil {
			fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
		}
		if result.Assignment != nil {
			assignment := result.Assignment
			fmt.Printf("reused=%t allocated=%t\n", assignment.Reused, assignment.Allocated)
			if assignment.FloatingIP != nil {
				fmt.Printf("floating4=%s ip-id=%s status=%s\n",
					assignment.FloatingIP.FloatingIP, assignment.FloatingIP.ID,
					assignment.FloatingIP.Status)
			}
		}
	}
	if err != nil {
		if errors.Is(err, resource.ErrUnsupported) {
			log.Printf("the selected backend or scope is not supported by this workflow")
		}
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response evidence: status=%d", response.StatusCode)
		}
		return fmt.Errorf("automatic floating IPv4: %w", err)
	}
	return nil
}

func printDecision(decision *compute.ServerFloatingIPDecision) {
	if decision == nil {
		return
	}
	fmt.Printf("needed=%t reason=%s backend=%s\n",
		decision.Needed, decision.Reason, decision.Backend)
	if decision.Needed && decision.Backend == compute.FloatingIPNeutron {
		selection := decision.Selection
		fmt.Printf("external=%s port-network=%s port=%s fixed4=%s\n",
			selection.NetworkID, selection.PortNetworkID,
			selection.PortID, selection.FixedIPv4)
	}
}
```

## 판단 순서와 lazy skip

옵션·context·필수 Server 입력을 검증한 후 다음 순서로 판단합니다. `Needed=false`만으로 skip을 확정하지 않습니다. Reason이 결정되고 **오류가 nil인 경우**의 skip이며, 오류가 있으면 partial decision은 진단입니다.

1. `WithAutomaticIPEnabled(false)` 또는 address source=`FloatingIPNone`이면 `disabled`.
2. cloud private policy가 true이면 `private_cloud`.
3. 외부 주소 사용 정책이 켜져 있고 supplied AccessIPv4가 있으면 `existing_public_ipv4`.
4. supplied `Server.Addresses`가 nil이면 raw Nova GET으로 한 번 갱신합니다. 명시 빈 map/빈 row 목록과 nil을 구분합니다. 갱신 후 AccessIPv4도 다시 검사합니다.
5. wire/owned 주소에 any-family `floating` 태그가 있으면 `existing_floating_ip`. IPv6 floating도 추가 IPv4 allocation을 생략합니다.
6. 모든 주소 row가 비어 있으면 `no_fixed_address`.
7. 남은 후보에 Network/source별 기존 association 보충과 역할 기반 public/private IPv4 선택을 적용합니다. 보충한 floating 행도 allocation을 피할 증거지만 **Nova 관측 완료 증거가 아닙니다**. 보충·목록·accepted decode 등의 오류를 성공 skip으로 숨기지 않습니다.
8. public IPv4가 있으면 `existing_public_ipv4`. private IPv4와 any-family fixed 태그가 모두 없으면 `no_fixed_address`.
9. Neutron이 있으면 same planner의 external/NAT/port/fixed tuple을 준비합니다. clean completed no-external/no-owned-port 또는 unconstrained explicit-fixed 부재만 해당 absence reason의 skip입니다. ambiguous/명시 mismatch/HTTP/decode/source/context 오류는 실패입니다.
10. 대상이 있으면 `assignment_needed`입니다. Network endpoint가 없는 경우 backend는 Nova이며 필요성 true일 수 있지만 mutation은 이 SDK 흐름에서 제공하지 않습니다.

known disabled/private/supplied AccessIPv4·floating 태그/명시 empty skip은 Network role·port·floating IP·owner 조회나 mutation을 하지 않습니다. nil 주소의 raw refresh만 필요한 skip은 Compute GET이 발생할 수 있습니다. skip이어도 잘못된 명시 Ref·옵션·nil Server·취소 등 입력 오류를 성공으로 바꾸지 않습니다. 일반 주소별 role 선택에는 [주소 정책](server-addresses.md)의 use flags·MAC·probe·정렬/explicit order 차이가 적용됩니다.

역할과 Network 사용이 필요한 경우 한 planner가 classification과 destination selection을 묶습니다. Shared cache의 성공값·Reset·동시 discovery 정책은 [역할 가이드](../network/network-roles.md)를 따르며 한 작업이 확보한 role snapshot은 뒤 Reset으로 다른 snapshot과 섞이지 않습니다. 다음 작업은 새 snapshot을 읽을 수 있습니다. Server 모델·공개 service/client/source의 동시 쓰기 지원이나 cache 외부 변경 자동 감지를 의미하지 않습니다.

## 조건부 assignment와 owner

Ensure는 skip이면 assignment 없이 반환합니다. 필요한 Neutron branch에서는 기존 서버 ACTIVE와 필수 raw 관측의 Compute client·요청 URL을 mutation 전에 확인한 뒤 준비한 tuple의 port ID·DeviceID·network·fixed IPv4를 다시 GET 검증하고 assignment를 실행합니다. BUILD/ERROR 서버를 이 메서드가 부팅하거나 ACTIVE까지 기다리는 workflow는 아닙니다. Server ACTIVE 비교와 관측은 Go에서 대소문자를 무시하며, Neutron waiter의 최종 실제 ACTIVE 검증은 기존 plan 계약을 따릅니다.

재사용은 true가 기본이며, 실제 owner는 selection 이후 실행 단계에서 명시 `WithEnsureProject` 또는 기록된 Keystone scope로 바인딩합니다. 필요성을 판단하기 위해 project owner를 선조회하지 않습니다. reuse=true인데 owner가 없으면 실행이 `ErrUnsupported`이고 IP mutation을 하지 않습니다. reuse=false는 가용 IP 목록을 생략하며 owner가 없으면 Neutron 인증 scope에 맡깁니다.

`WithAutomaticEnsureOptions`는 port/fixed/NAT·owner·reuse와 waiter 설정을 받습니다. source query와 결과 검증·revision·201/202 accepted allocation partial 결과는 [plan 계약](../network/floating-ip-plan.md)과 같습니다. actual IP ACTIVE는 항상 필수이므로 일반 Ensure 옵션으로 wait=false를 선택해 생략하는 경로는 없습니다. waiter의 timeout/poll/progress 설정은 유지되며 성공 상태 attribute를 바꿔 실제 ACTIVE 검사를 우회하지 못합니다.

## raw Nova 주소 관측과 부분 결과

대상 서버 ID는 시작 시 고정합니다. Neutron assignment/ACTIVE 뒤 그 ID의 raw Nova `GET /servers/{id}`를 반복합니다. 성공은 실제 응답 server ID가 대상과 일치하고 상태가 ACTIVE이며, **모든 network row 중** version=4·type=`floating`·addr가 이번 assigned IPv4와 정확히 같은 행을 찾았을 때입니다. 첫 floating 행만 고르는 비교가 아니며 다른 public 주소·다른 floating 주소·같은 문자열의 fixed 행·IPv6 행은 완료 증거가 아닙니다.

AccessIPv4나 `ExpandServerInterfaces`의 Supplemental 행을 합성해 Observed=true로 만들지 않습니다. 관측은 raw Nova 응답만 사용하며 입력/partial view의 주소를 덮어써 성공으로 표시하지 않습니다. HTTP/decode/identity/ERROR·source·취소는 오류이고 Python처럼 모든 GET 오류를 blanket retry하지 않습니다. 아직 주소가 수렴하지 않은 유효 Nova 응답만 poll/progress 대상입니다.

| 중단 시점 | 보존할 결과 |
|---|---|
| 입력/옵션 준비 실패 | 결과가 없을 수 있음 |
| decision 중 HTTP/주소/역할/선택 실패 | 확인한 Server와 partial Decision; assignment 없음 |
| Needed지만 Nova backend | Needed와 backend Nova Decision; `ErrUnsupported`, assignment 없음 |
| port 재검증/owner/IP 후보 오류 | 알려진 Server와 Decision; mutation 증거가 없으면 assignment 없음 |
| accepted allocation/재사용 PUT/IP wait 실패 | 알려진 Server·Decision과 실제 부분 Assignment |
| Neutron ACTIVE 뒤 Nova 관측 실패/timeout | 실제 Assignment와 마지막 확인한 Nova Server; Observed=false |
| clean policy skip | Server·결정된 reason; assignment 없음·Observed=false |
| raw Nova에서 실제 target 확인 | Server·Decision·Assignment·Observed=true |

`Observed=false`는 무조건 오류가 아닙니다. clean skip이면 관측할 신규 assignment가 없어서 false입니다. true도 주소의 라우팅·SSH 도달성·quota·분산 예약을 보장하지 않습니다. Neutron 성공 뒤 Nova 관측 timeout이 나도 자동 DELETE/새 allocation/다른 backend fallback을 하지 않습니다. 실제 model이나 ID를 해석하지 못한 accepted201/202 allocation은 `Allocated=true`와 nil FloatingIP, actual response error로 남을 수 있습니다.

## 시간 제한·progress·설정

| 옵션 | 기본값과 의미 |
|---|---|
| `WithAutomaticIPEnabled(bool)` | true; 자동 필요성 분기를 켜거나 끔 |
| `WithAutomaticAddressOptions(opts...)` | Connection의 frozen address policy 위에 호출별 private/source/probe/order 등 옵션 적용 |
| `WithAutomaticEnsureOptions(opts...)` | reuse=true; selector·owner·waiter 정책 적용, 실제 Neutron ACTIVE 강제 |
| `WithAutomaticIPTimeout(duration)` | 5분; 판단·selection·assignment·raw Nova 관측 전체, 양수 |
| `WithUnlimitedAutomaticIPTimeout()` | SDK 전체 제한 해제; 부모 context·개별 waiter 제한은 유지 |
| `WithAutomaticIPPollInterval(duration)` | 2초; raw Nova 관측 간격, 양수 |
| `WithAutomaticIPProgress(func(*Server) error)` | 기본 없음; 아직 수렴하지 않은 유효 raw 응답의 복사본으로 실행, 오류/취소/source 변화면 후속 요청 중단 |

부모 context가 더 빠르면 부모 deadline이 적용됩니다. 각 단계를 시작할 때 전체 timeout을 다시 시작하지 않습니다. 예제의 initial Connect/Find는 자동 메서드 밖에서 실행되어 부모 context가 제한하고, 자동 메서드 안은 명시한 3분 timeout이 제한합니다. 개별 IP waiter의 timeout을 더 짧게 지정할 수 있습니다.

Connection은 주소 설정과 Network 역할 cache를 공유하고 필요한 endpoint를 lazy resolve합니다. supplied model만으로 known skip이 나면 metadata helper 자체는 Compute endpoint를 요구하지 않습니다. nil 주소 refresh·Nova source 보충·raw 관측이 필요하면 Compute endpoint가 필요합니다. Network catalog의 정확한 missing endpoint는 Nova backend 판단과 구분하고, 다른 catalog/HTTP/config 오류를 서비스 없음으로 숨기지 않습니다. 이 bounded endpoint 선택을 full Python `has_service`의 설정/version/session 표면 완료로 세지 않습니다.

## 차이·검증·남은 범위

Go는 disabled/private 같은 known skip을 앞에서 확인하고 명시 empty와 nil refresh를 분리하여 불필요한 Network/owner 요청을 피합니다. Python needs의 일부 floating-network SDKException→false 처리를 그대로 숨기지 않고 clean semantic absence와 fatal 오류를 구분합니다. strict tuple ambiguity와 port reGET/revision, 실제 assigned IPv4의 모든 raw floating row 관측, accepted partial 증거 보존과 no-cleanup 정책도 의도된 차이입니다.

[CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)는 생성·ACTIVE/주소 준비부터 이 자동 정책까지 이어갑니다. [GetActiveServer·상위 WaitForServer](server-ready.md)는 기존 서버의 supplied 상태 판정·raw 현재 상태 대기와 조건부 IP 작업을 제공합니다. GetActive의 기본 비동기 접수와 이 문서의 Ensure·생성·상위 Wait가 요구하는 실제 IP ACTIVE·Nova 관측을 구분합니다. 일반 `Servers.Create`, `CreateWithFloatingIP`, collection 대기의 계약은 유지합니다. 서버 부팅의 추가 조합, fault/extra_data·ACTIVE-no-address 삭제 정책, cloud lookup의 exception retry/integer remaining budget, pool>ips>auto 전체 우선순위, Nova-network mutation/fallback, full service/config/Resource/session/normalization·cleanup은 남습니다. `add_auto_ip`의 unconditional/string 결과 계약도 별도입니다.

Python 비교는 고정 source의 정적 검토이며 Python 예제·인증된 OpenStack 실행을 뜻하지 않습니다. 실제 신규 HTTP fixture의 skip/guard/selection/assignment/raw observation/partial/cancellation 근거와 집중·전체 검증 revision, 독립 main 정확한 SHA·컴파일 receipt는 [지원 판정대장](../docs/sdk-support-ledger.md)에 확인한 결과만 기록합니다. 전체 cloud 연산은 해당 remaining이 남으면 unresolved로 유지합니다.
