# 새 Floating IP 생성과 할당 후 대기

`Connection.CreateFloatingIP`과 `compute.Service.CreateFloatingIP`은 항상 새 Floating IP를 할당합니다. SDK가 cloud 설정에 따라 Neutron 또는 legacy Nova를 선택하고, Neutron에서는 선택한 port로 생성합니다. 기존 free IP를 재사용하려면 [AvailableFloatingIP](floating-ip-available.md), 서버 연결·재사용이 필요하면 [IP helper](server-ip-helpers.md) 또는 [Network Ensure](../network/floating-ip-ensure.md)를 사용합니다.

```go
networkName := "public"
result, err := conn.CreateFloatingIP(ctx,
	compute.CreateFloatingIPRequest{Network: &networkName},
	compute.WithFloatingIPCreateServer(resource.ID(serverID)),
	compute.WithFloatingIPCreateWait(true),
)
```

애플리케이션은 builder나 resolver interface를 구현하지 않습니다. 서버는 `resource.ID` 또는 `resource.Name`으로 지정하고 SDK가 필요한 이름 조회·port·NAT 선택을 처리합니다. `err != nil`이어도 `Allocated=true`이면 실제 POST 응답을 접수했으므로, 부분 결과와 cleanup 기록을 함께 확인합니다.

## 입력·backend·우선순위

`CreateFloatingIPRequest`의 `Network *string`은 absent와 empty를 구분합니다.

| Network 입력 | Neutron | Nova |
|---|---|---|
| nil | configured floating network의 첫 후보, 없으면 router의 external gateway 선택 | 첫 floating IP pool 이름 조회 |
| pointer to "" | 기본 network 선택 | literal empty pool로 POST |
| pointer to name/ID | 일반 network 이름/ID 조회 | literal pool 이름으로 POST |

Neutron network를 명시하면 external 여부를 추가로 강제하지 않으며 실제 POST가 판단합니다. 설정된 source가 `FloatingIPNeutron`이고 Network 서비스를 사용할 수 있으면 Neutron을 선택합니다. 그 밖에는 Nova이며, explicit Create의 `FloatingIPNone`도 Nova 경로입니다. Nova endpoints는 selected Compute microversion2.36 이상에서 사용할 수 없고 SDK가 버전을 낮추지 않습니다.

Neutron의 nonempty `PortID`는 Server·FixedAddress·NATDestination을 우회합니다. 이 경로는 서버·port inventory를 미리 조회하지 않습니다. port가 없고 Server가 있으면 서버의 ports를 조회해 destination을 고릅니다.

- `FixedAddress`를 지정하면 port의 fixed IP 목록에서 literal equality로 찾습니다. 이 분기는 IPv4만으로 제한하지 않습니다. 일치하지 않거나 port가 없으면 연결 없는 새 allocation을 수행합니다.
- fixed를 지정하지 않고 ports가 여러 개이면 explicit 또는 configured NAT destination으로 좁힙니다. 같은 destination의 ports는 created_at 역순으로 보고 첫 parse 가능한 IPv4를 선택합니다.
- Server가 없으면 fixed/NAT를 무시합니다. selected port가 없으면 wait도 소비하지 않습니다.
- Nova는 Server·FixedAddress·NATDestination·wait를 무시하고 새 allocation만 수행합니다. nonempty PortID는 Nova POST 전 unsupported입니다. addFloatingIp action이나 서버 ACTIVE 대기를 대신 수행하지 않습니다.

## concrete 옵션과 기본값

| 옵션 | 의미 |
|---|---|
| `WithFloatingIPCreateServer(ref)` | 서버 ID 또는 이름; explicit port가 있으면 우회 |
| `WithFloatingIPCreatePort(id)` | 직접 지정한 Neutron port ID |
| `WithFloatingIPCreateFixedAddress(value)` | Server destination의 fixed IP literal |
| `WithFloatingIPCreateNATDestination(value)` | 다중 port 선택의 network 이름/ID |
| `WithFloatingIPCreateWait(bool)` | 기본false; Neutron selected port가 있을 때만 사용 |
| `WithFloatingIPCreateWaitTimeout(duration)` | 기본60초; allocation 뒤 시작하는 wait budget; 0/음수는 소비 시 즉시 timeout |
| `WithFloatingIPCreateWaitInterval(duration)` | 양수 interval 지정; 생략하면 min(5초, wait timeout) |
| `WithFloatingIPCreateSource(source)` | 이번 호출의 source override |
| `WithFloatingIPCreateLocation(location)` | 조회 view의 cloud/project location |
| `WithFloatingIPCreateStrict(bool)` | Nova view 정규화의 strict 모드 |
| `WithFloatingIPCreateDirectGet(bool)` | wait/cleanup의 일반 Get에서 UUID direct 조회 선택 |
| `WithFloatingIPCreateTimeout(duration)` | 양수인 전체 작업 cap; lookup·allocation·compat GET·wait·cleanup에 적용 |
| `WithUnlimitedFloatingIPCreateTimeout()` | 추가 전체 SDK cap 제거; caller context는 유지 |
| `WithFloatingIPCreateOptions(compute.FloatingIPCreateOpts{...})` | 전체 옵션의 소유한 복사본으로 교체 |

뒤에 오는 setter가 앞선 값을 덮어씁니다. bulk `FloatingIPCreateOpts{}`는 wait timeout도0으로 교체합니다. bulk를 쓰면서 대기할 때는 `WaitTimeout: time.Minute`을 지정하거나 뒤에 timeout setter를 추가합니다. `PrepareFloatingIPCreateOptions(ctx, options...)`로 HTTP 없이 공통 옵션을 준비할 수 있습니다.

request의 Network pointer, 옵션 slice·Source pointer·Location은 SDK가 복사하며 caller 옵션 함수는 전체 작업에서 각각 한 번 실행합니다. nil 옵션, invalid source, 음수 전체 cap 또는 invalid interval 같은 공통 옵션 오류는 요청 전 전파합니다. consumed Server/Port/network의 validation은 해당 경로에서 적용되므로 explicit port 또는 Nova가 우회하는 destination을 앱이 별도로 해석할 필요가 없습니다.

## 대기와 cleanup

selected Neutron port + wait=true이면 allocation 뒤 기본60초 동안 [일반 Get](floating-ip-queries.md)을 수행합니다. 초기 POST가 ACTIVE여도 조회를 실행하며, 기본은 목록 검색입니다. UUID direct 조회는 옵션으로 선택합니다. public Get의 resource view에서 exact string `ACTIVE`만 완료이며 no-match·DOWN·ERROR·UNKNOWN 등은 계속 대기합니다. ACTIVE를 관측한 뒤 요청 port와 returned `port_id`가 같은지 확인합니다.

`Waited=true`는 이 대기와 port 검증이 성공했음을 뜻합니다. 대기를 시작했다는 표시가 아닙니다. 실제 조회는 `Observations`에 남고, 실패 시 앞서 접수한 allocation을 지우지 않습니다. Nova view의 합성 ACTIVE는 이 wait 성공으로 기록하지 않습니다.

0 또는 음수 wait timeout은 port+wait 경로에서 첫 조회 전에 timeout입니다. no port 또는 Nova에서는 이 wait 값이 사용되지 않습니다. 기본 interval은 min(5초, timeout)이고, 전체 cap과 caller context는 별도의 제한입니다. wait timer는 lookup/POST 이후에 시작하지만 optional 전체 cap은 작업 준비 뒤 backend 연결부터 적용됩니다.

이 SDK가 만든 wait timer가 만료되면 `*compute.FloatingIPCreateTimeoutError`를 반환하고, 원래 작업 context가 유효할 때 같은 source로 [DeleteFloatingIP](floating-ip-delete.md)의 기본 retry1·일반 Get 검증을 수행합니다. `errors.As`로 timeout의 ID·Timeout·Last를 확인하고 `errors.Is(err, context.DeadlineExceeded)`로 deadline 원인을 확인할 수 있습니다.

cleanup은 만료된 wait context를 쓰지 않습니다. caller 취소·부모 deadline·전체 cap·source 변경으로 원래 작업을 계속할 수 없으면 삭제를 강행하지 않습니다. SDK 전용 wait timer가 만료되지 않은 HTTP/decode/ambiguous 오류 또는 port mismatch도 cleanup 이유가 아닙니다. timer 만료는 child context의 원인으로 판별하며, 다른 오류와 동시에 만료되면 timeout과 가능한 cleanup 기록을 함께 보존합니다. cleanup 오류는 `CleanupError`와 반환 error에 원래 timeout과 함께 남습니다. `Cleanup.Deleted=true`라도 DOWN 정책의 성공일 수 있으므로 실제 부재는 `Cleanup.Absent`로 구분합니다.

## 부분 결과와 실제 응답

`CreateFloatingIPResult`는 다음을 나눕니다.

| 필드 | 의미 |
|---|---|
| `Backend` | 실제 최종 allocation backend |
| `Allocated` | POST의 허용된 실제 응답을 접수함; 모델 decode·read/close·guard 오류가 함께 있을 수 있음 |
| `AllocationResponse` | 최초 POST의 status/header와 실제 envelope bytes |
| `Allocation` | 최초 POST 모델. Nova에서는 raw Wire이며 Resource 정규화는 compatibility GET 이후 |
| `FloatingIP` | 성공한 최종 view 또는 알려진 allocation. timeout의 마지막 조회는 Observations/timeout.Last도 확인 |
| `Selection` | Neutron의 network/server/port/fixed 선택. 기존 `FixedIPv4` 필드는 explicit fixed literal 분기에서 IPv6 literal도 담을 수 있음 |
| `PoolQuery` | default Nova pool을 선택한 조회·Pages·실패 기록 |
| `Compatibility` | Nova POST 뒤 필수 raw GET의 모델과 실제 응답·오류 |
| `Observations` | 순서대로 수행한 wait의 일반 Get 결과·Pages·fallback·실패 기록 |
| `FallbackError` | accepted allocation 전에 Nova로 전환한 pure Neutron NotFound |
| `Failure` | 확보한 응답 처리/HTTP 실패 증거; 이전 allocation·조회 기록을 대신하지 않음 |
| `Cleanup`, `CleanupError` | SDK wait timeout 뒤 삭제 작업의 부분 결과 또는 실행/처리 오류 |

HTTP200–399 성공 정책은 native callback이 변경해도 최초 값으로 고정합니다. accepted POST 뒤 처리 오류를 다른 backend의 추가 allocation으로 감추지 않습니다. accepted GET/read/close 오류도 실제 응답과 가능한 모델을 보존합니다.

Neutron은 응답에서 **누락된** network/port/fixed 값을 요청 seed로 조회 Resource view에 보충합니다. present null 또는 다른 값은 response가 우선이며 `Wire`에는 seed를 넣지 않습니다. async Neutron response에 ID가 없으면 view에는 null, Wire에는 실제 missing이 유지됩니다. wait나 Nova compatibility GET에 사용할 실행 가능한 ID를 얻지 못하면 응답 기록을 남긴 오류입니다.

Nova는 POST `/os-floating-ips` 뒤 allocation ID로 raw compatibility GET을 수행합니다. 일반 Get의 목록 검색으로 바꾸지 않으며 반환 model의 pool/free association/canonical IPv4를 추가로 강제하지 않습니다. 조회 결과는 `FloatingIPRecord.Resource`와 actual `Wire`로 구분합니다. configured Neutron의 fallback Nova는 Neutron mode로, direct Nova/None는 합성 ACTIVE로 정규화될 수 있습니다.

`AllocationResponse.Envelope`는 passive bytes라 유효한 JSON을 보장하지 않습니다. 부분 결과 전체를 json.Marshal하면 invalid body 때문에 실패할 수 있어 아래 예제는 scalar summary만 출력합니다.

## 독립 Go 예제

이 프로그램은 IP를 실제로 새로 생성합니다. Network flag를 생략하면 기본 선택이며, `-network ""`를 명시하면 Nova의 literal empty pool을 표현합니다. `-server`는 ID, `-port`는 직접 port입니다. Connect를 포함한 부모2분 예산 안에서 수행하며 `-timeout`을 지정하면 별도 전체 SDK cap도 적용합니다. Nova/fallback이 필요한 경우를 위해 selected Compute2.35를 명시합니다.

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
	"gophercloudsdk/resource"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	networkName := flag.String("network", "", "network name/ID or literal Nova pool")
	serverID := flag.String("server", "", "optional server ID")
	portID := flag.String("port", "", "optional explicit Neutron port ID")
	fixed := flag.String("fixed", "", "optional fixed IP literal")
	nat := flag.String("nat", "", "optional NAT destination name/ID")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	wait := flag.Bool("wait", false, "wait for selected Neutron port allocation")
	waitTimeout := flag.Duration("wait-timeout", time.Minute, "post-allocation wait timeout")
	timeout := flag.Duration("timeout", 0, "optional whole SDK operation cap")
	flag.Parse()

	input := compute.CreateFloatingIPRequest{}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "network" {
			input.Network = networkName
		}
	})
	opts := []compute.FloatingIPCreateOption{
		compute.WithFloatingIPCreatePort(*portID),
		compute.WithFloatingIPCreateFixedAddress(*fixed),
		compute.WithFloatingIPCreateNATDestination(*nat),
		compute.WithFloatingIPCreateWait(*wait),
		compute.WithFloatingIPCreateWaitTimeout(*waitTimeout),
	}
	if *serverID != "" {
		opts = append(opts, compute.WithFloatingIPCreateServer(resource.ID(*serverID)))
	}
	if *source != "" {
		opts = append(opts, compute.WithFloatingIPCreateSource(compute.FloatingIPSource(*source)))
	}
	if *timeout != 0 {
		opts = append(opts, compute.WithFloatingIPCreateTimeout(*timeout))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.Compute, "2.35"),
	)
	if err != nil {
		return err
	}
	result, createErr := conn.CreateFloatingIP(ctx, input, opts...)
	var summary map[string]any
	if result != nil {
		summary = map[string]any{
			"backend": result.Backend, "allocated": result.Allocated,
			"waited": result.Waited, "observations": len(result.Observations),
			"compatibility_attempted": result.Compatibility != nil,
		}
		if result.AllocationResponse != nil {
			summary["allocation_status"] = result.AllocationResponse.StatusCode
		}
		if result.Cleanup != nil {
			summary["cleanup_deleted"] = result.Cleanup.Deleted
			summary["cleanup_absent"] = result.Cleanup.Absent
			summary["cleanup_down"] = result.Cleanup.Down
			summary["cleanup_attempts"] = len(result.Cleanup.Attempts)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(createErr, encoder.Encode(summary))
}
```

## openstacksdk 비교와 남은 범위

```python
import openstack

conn = openstack.connect(cloud="dev")
ip = conn.create_floating_ip(network="public")
attached = conn.create_floating_ip(
    network="public", server={"id": "SERVER_ID"}, wait=True, timeout=60
)
# explicit port wins over server/fixed/NAT in the pinned implementation
on_port = conn.create_floating_ip(network="public", port="PORT_ID", wait=True)
```

비교 pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. [public/backend selection](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L775-L842), [Neutron create/wait/cleanup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L850-L949), [Nova POST/compat GET](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975), [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)을 기준으로 합니다.

Python은 최종 Resource/Munch 또는 exception을 반환합니다. Go는 concrete 옵션과 owned Wire/Resource, accepted POST·compatibility·wait·cleanup 부분 결과를 반환합니다. source가 accepted Neutron 생성 뒤 wait NotFound까지 Nova로 전환할 수 있는 반면, Go는 접수 뒤 processing 오류에서 두 번째 allocation을 차단합니다. HTTP 오류에 source/context/decode 같은 terminal 원인이 섞이면 fallback하지 않습니다.

Python의 timeout loop는 allocation 뒤 wall-clock을 확인하며 in-flight HTTP를 중단하지 않습니다. Go의 wait child context는 HTTP와 마지막 완료 guard에도 deadline을 적용하고, original guarded context가 살아 있을 때 library-owned wait timeout만 cleanup합니다. caller 취소를 분리해 삭제하지 않습니다. source cleanup은 오류를 로그만 남기지만 Go는 별도 CleanupError와 원래 timeout을 함께 보존합니다.

이 named Cloud 생성은 concrete 입력·SDK 기본값·fresh allocation·선택적 대기·timeout 정리를 제공하는 Go 매핑입니다. Python의 임의 object/Munch·JSON 바깥 동적 입력·null/unsafe executable ID·timeout=None은 typed Go 입력과 context 정책으로 대응하며, 전체 mutable Resource·inherited fetch/commit/session/cache/discovery·직접 Network Proxy/native 선언·DeleteUnattached는 별도 SDK 구현 범위로 추적합니다. async 결과는 접수한 리소스의 현재 view입니다. 해당 범위나 인증된 실제 클라우드 검증을 이 생성의 완료 근거로 대신하지 않습니다.


[미연결 Floating IP 일괄 정리](floating-ip-unattached-delete.md)는 Neutron 전체 목록을 확보한 뒤 port가 비어 있는 항목을 순차 삭제합니다. 개별 false는 계속 처리하고 오류는 중단하며 SDK 소유 옵션·한 deadline·항목별 부분 결과를 제공합니다.
