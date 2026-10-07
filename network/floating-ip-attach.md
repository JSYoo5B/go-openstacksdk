# 기존 Floating IP 연결

`FloatingIPs.Attach`는 지정한 기존 IP를 서버의 fixed IPv4에 연결합니다. `PrepareAttach`와 `AttachPrepared`로 선택과 실행을 나눌 수도 있습니다. SDK가 이름 조회·목적지 선택·조건부 PUT·대기·부분 결과를 담당하므로 애플리케이션이 builder나 resolver interface를 구현할 필요가 없습니다.

| 입력·호출 | 동작 |
|---|---|
| `AttachFloatingIPRequest{Server: resource.ID(id), IP: resource.ID(ipID)}` | 기존 IP ID를 정확히 GET |
| `IP: resource.Name("198.51.100.10")` | floating 주소 query와 모든 페이지의 정확한 주소 비교 |
| `Server: resource.Name(name)` | Connection이 제공하는 Compute resolver로 서버 ID를 한 번 조회 |
| `service.FloatingIPs.Attach(ctx, request, opts...)` | 준비와 실행을 한 호출로 수행 |
| `service.FloatingIPs.PrepareAttach(ctx, request, opts...)` | 기존 IP·목적지·revision을 조회하여 고정; mutation 없음 |
| `plan.Selection()` | IP ID/주소, 외부 network, 서버·port·fixed IPv4 값의 복사본 |
| `service.FloatingIPs.AttachPrepared(ctx, plan)` | 같은 service에서 포트와 IP를 재검증하고 연결·선택적 대기 |
| `planner.PrepareAttach(ctx, request, opts...)` | 같은 planner의 source binding과 lazy NAT 역할 snapshot 사용 |

`Name`에는 유효한 floating **IPv4 주소**를 넣습니다. IP ID와 주소를 문자열 모양으로 추측하지 않습니다. 주소 검색은 `floating_ip_address`만 전송하며 현재 프로젝트·미연결 여부·외부 network로 제한하지 않습니다. Neutron 권한이 허용하면 다른 프로젝트의 IP나 이미 다른 포트에 연결된 IP도 지정할 수 있습니다. 모순된 project/tenant 응답은 오류이고, 조회에서 확인한 owner는 실행 응답의 일관성을 검증하는 데만 사용합니다.

독립 [AddIPsToServer·AddIPList](../compute/server-ip-helpers.md)는 별도 기본60초·비동기 entry입니다. 선택적 wait는 서버/IP ACTIVE 없이 raw 목표 주소를 확인하며, 이 문서의 직접 API나 기존 상위 readiness 조건을 바꾸지 않습니다.

## openstacksdk와 비교

비교 소스는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [add_ip_list와 Neutron 연결](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1342-L1391)입니다. Python은 주소 목록을 순서대로 찾아 연결하고 Server를 반환합니다.

```python
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ip_list(
    server, ips=["198.51.100.10"],
    fixed_address="10.0.0.10", wait=True, timeout=60,
)
```

이 Network API는 **기존 Neutron IP 한 개의 연결 기반**입니다. `FloatingIPAssignment`를 반환하며 기본은 비동기이고 `WithAttachWait`는 실제 Neutron IP ACTIVE를 기다립니다. [Compute의 IP dispatch](../compute/server-ip-dispatch.md)가 이 기반과 Ensure를 pool → 순차 명시 IPv4 → automatic 순서로 소비합니다. 기존 Ensure/Get/Wait/Create의 Neutron 동기 소비자는 실제 IP ACTIVE와 raw Nova 목표 주소를 확인하고, GetActiveServer 기본 async는 접수 결과를 반환합니다. 직접 Network Attach는 Server 반환·raw Nova 관측·여러 주소 dispatch를 수행하지 않습니다. [Compute legacy Nova backend](../compute/server-nova-floating-ip.md)는 별도로 실행하며, standalone cloud `add_ip_list`의 full returned Resource/normalization·전체 Resource/session/normalization 계약은 남아 있으므로 전체 `add_ip_list` 지원 완료로 세지 않습니다.

## 독립 Go 예제

기본 실행은 선택한 대상을 출력합니다. `-execute`를 주면 동일 계획을 연결하고 실제 Neutron ACTIVE까지 기다립니다. `-ip-id`는 선택 사항이며 비어 있으면 `-address`로 조회합니다. `-fixed`를 지정하면 해당 fixed IPv4를 가진 유일한 server-owned 포트를 선택합니다.

```go
package main

import (
	"context"
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
	server := flag.String("server-id", "", "existing Nova server ID")
	ipID := flag.String("ip-id", "", "existing Floating IP ID")
	address := flag.String("address", "", "existing floating IPv4 address")
	fixed := flag.String("fixed", "", "exact destination fixed IPv4")
	execute := flag.Bool("execute", false, "execute the prepared attachment")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *server, *ipID, *address, *fixed, *execute); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, server, ipID, address, fixed string, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	ip := resource.Name(address)
	if ipID != "" {
		ip = resource.ID(ipID)
	}
	options := []network.AttachFloatingIPOption{
		network.WithAttachWait(resource.WithTimeout(time.Minute)),
	}
	if fixed != "" {
		options = append(options, network.WithAttachFixedAddress(fixed))
	}
	plan, err := service.FloatingIPs.PrepareAttach(ctx,
		network.AttachFloatingIPRequest{Server: resource.ID(server), IP: ip}, options...)
	if err != nil {
		return err
	}
	fmt.Printf("selection: %+v\n", plan.Selection())
	if !execute {
		return nil
	}
	result, err := service.FloatingIPs.AttachPrepared(ctx, plan)
	if result != nil && result.FloatingIP != nil {
		fmt.Printf("IP: %s address=%s port=%s reused=%t allocated=%t\n",
			result.FloatingIP.ID, result.FloatingIP.FloatingIP, result.FloatingIP.PortID,
			result.Reused, result.Allocated)
	}
	return err
}
```

## 목적지·대기 옵션

`WithAttachPort`는 정확한 포트 이름/ID를 선택하고 그 서버 소유인지 확인합니다. `WithAttachNATDestination`은 private network 이름/ID, `WithAttachFixedAddress`는 정확한 fixed IPv4를 지정합니다. 여러 후보가 남으면 `ErrAmbiguous`이고, 대상이 없으면 오류입니다. 명시 Port/fixed와 단일 포트는 NAT 역할 탐색을 생략하며, 여러 포트의 자동 NAT 선택만 planner의 역할 snapshot을 사용합니다.

`PrepareAttachFloatingIPOptions`와 `WithAttachFloatingIPPolicy`로 준비한 옵션을 재사용할 수 있습니다. `WithAttachDestinationPolicy(ensurePolicy)`는 [Ensure 정책](floating-ip-ensure.md)의 Port/NAT/fixed/wait만 복사합니다. allocation·reuse·project 정책을 가져오거나 원래 옵션 callback을 다시 실행하지 않습니다. bridge 뒤 Attach 옵션을 주면 뒤의 값이 우선합니다.

`WithAttachWait`는 공통 timeout·poll·progress 옵션을 준비합니다. `WithAttachActive`/`WithAttachNoWait`는 대기 여부만 바꾸므로 앞서 검증한 대기 옵션을 유지합니다. 잘못된 wait 입력은 NoWait로 숨길 수 없습니다. 기본 대기 옵션은 공통 resource waiter의 5분·2초이며 부모 context의 deadline/cancellation도 적용됩니다. custom status attribute가 완료를 주장해도 실제 `Status == "ACTIVE"`를 확인합니다.

## 실행과 실패 결과

주소 목록은 빈 페이지의 JSON/HTTP next도 따라가고 전체 탐색을 마친 뒤 unique/missing/ambiguous를 결정합니다. 후속 HTTP·decode·Close·query/source·취소 오류를 앞선 후보로 숨기지 않습니다. ID 선택은 정확한 singular GET이며, 전체 주소 목록을 대신 조회하지 않습니다.

실행은 선택한 포트의 ID·서버·network·fixed 주소, IP의 ID·floating 주소·외부 network·기록된 owner·association을 재검증합니다. 원래 다른 포트에 안정적으로 연결되어 있으면 이동을 허용하고, 준비 이후 제3의 tuple로 바뀌면 오류입니다. 이미 요청 tuple에 연결되어 있으면 PUT을 생략하고 필요한 ACTIVE 대기를 수행합니다. revision이 있으면 **준비 당시** 값으로 `If-Match: revision_number=N`을 전송하며 0도 보존합니다. absent/null은 header를 생략하고, 재조회 revision으로 조건을 바꾸지 않습니다. revision이 없으면 재조회와 PUT 사이의 원자적 동시성 보호는 보장하지 않습니다.

유효한 계획의 실행이 늦게 실패하면 알려진 IP를 `Reused=true`, `Allocated=false`로 반환합니다. accepted PUT200의 complete 응답은 정확한 ID/주소/owner/network/destination 검증을 통과한 경우에만 read/Close/source/cancel 오류 옆에 보존합니다. 잘못된 응답은 이전 확인 후보를 유지하고, decode·검증·처리 오류와 실제 HTTP 증거를 함께 반환합니다. 모델이 있다는 이유로 성공이나 ACTIVE 완료로 취급하지 않습니다.

계획과 결과는 별도 소유이며 Selection이나 반환 모델의 Tags를 바꿔도 계획을 수정하지 않습니다. zero/실패/다른 service의 계획은 실행할 수 없습니다. source 변경은 sticky 오류이고 복원해도 같은 계획으로 이어가지 않습니다. 412·retry header 변경·그 밖의 실패에 재선택·새 IP allocation·Nova fallback·자동 DELETE가 없습니다.

검증 근거와 전체 남은 범위는 [지원 대장](../docs/sdk-support-ledger.md#기존-floating-ip의-고정-연결)과 [진행표](../docs/implementation-plan.md)를 참고하세요.
