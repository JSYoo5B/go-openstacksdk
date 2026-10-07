# 기존 서버의 ACTIVE 판정과 cloud 대기

`GetActiveServer`는 제공한 서버의 상태를 판정하고 ACTIVE일 때 조건부 IP 작업을 실행합니다. `WaitForServer`는 같은 서버 ID의 현재 raw Nova 모델을 조회하며 ACTIVE와 주소 준비를 기다린 후 같은 IP 정책을 실행합니다. Connection이 인증·cloud 주소 설정·공유 역할·lazy 서비스 연결을 제공하므로 호출자가 builder나 resolver를 구현하지 않습니다.

| API | 입력/완료 의미 | 기본값 |
|---|---|---|
| `service.GetActiveServer(ctx, request, options...)` | supplied 모델 판정; non-ACTIVE는 nil,nil; ACTIVE는 필요할 때 Neutron assignment | auto/reuse true, IP wait false, 전체180초, raw poll5초 |
| `service.WaitForServer(ctx, request, options...)` | supplied ID를 고정하고 raw 현재 상태·주소 준비 후 조건부 assignment와 실제 Nova 관측 | auto/reuse true, wait 필수, 전체180초, server/raw poll5초 |
| `conn.GetActiveServer` / `conn.WaitForServer` | 같은 정책의 Connection facade | Compute endpoint는 필요한 raw 조회 시 발견 |
| `service.Servers.WaitForServer(ctx, ref, options...)` | 기존 collection의 ACTIVE 상태 대기 | 120초·2초; 자동 IP 작업 없음 |
| `service.CreateWithAutomaticFloatingIP` | 새 서버 생성부터 readiness·조건부 IP·Nova 관측 | 기존5분·2초; 이 문서의 독립 기본값으로 바꾸지 않음 |

요청은 기존 `compute.AutomaticFloatingIPRequest{Server: supplied, Network: externalRef}`입니다. `Network`는 floating IP allocation network이며 서버 NIC를 추가하거나 변경하지 않습니다. ID/Name을 `resource.Ref`로 명시하고 zero이면 공유 역할/router 경로를 사용합니다.

## Python과 비교

비교 소스는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [wait_for_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1415)와 [get_active_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1417-L1481)입니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
	raise RuntimeError("server not found")

# Provided status is inspected. Non-ACTIVE returns None; default wait=False.
active = conn.get_active_server(server, auto_ip=True, reuse=True, wait=False, timeout=180)

# Only the supplied ID chooses the target; current state is fetched and polled.
ready = conn.wait_for_server(server, auto_ip=True, reuse=True, timeout=180)
print(ready.id, ready.interface_ip)
```

Python의 ERROR 분기는 fault.message가 있으면 이유를 포함한 SDKException과 server extra_data를 만듭니다. non-ACTIVE는 None입니다. ACTIVE에서 falsy addresses는 서버 DELETE를 시도한 뒤 오류이며, 삭제 실패까지 별도 오류로 감쌉니다. Go는 알려진 Server를 보존하는 typed 오류를 반환하고 자동 삭제하지 않습니다. Python의 `{net: []}`는 truthy이지만 Go는 실제 주소 row가 있어야 준비된 것으로 판단합니다.

Python의 [attach helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1109-L1123)는 supplied floating 주소가 없고 선택한 IP의 port_id가 있으면 wait=false에서도 Compute GET을 한 번 하여 이미 연결된 주소인지 확인합니다. Go의 Neutron async branch는 이 refresh를 하지 않고 접수한 Assignment와 분리한 supplied Server를 반환합니다. Observed=false는 raw Nova 수렴을 검사하지 않았다는 의미입니다.

Python Wait는 get_server의 lookup Exception과 None을 재시도하고, `timeout - int(elapsed)`를 후속 IP 작업에 전달합니다. Go의 raw target 조회·응답 소유권·취소·source 오류 정책과 연속 context deadline은 이 동작 전체와 같지 않습니다. 일반 get_server의 Resource/interface expansion, pool > ips > auto dispatch와 Nova-network mutation은 별도로 추적합니다.

## 독립 Go 예제

이 예제는 supplied Nova server JSON을 파일에서 읽습니다. 모델을 이미 받은 애플리케이션은 같은 `AutomaticFloatingIPRequest`를 직접 구성하면 됩니다. GetActive 입력에는 안전한 `id`와 실제 supplied `status`가 필요하며 ACTIVE 판정에는 주소 rows도 필요합니다. `-mode wait`는 안전한 ID만 대상 선택에 사용하고 supplied status·addresses·fault를 현재 상태로 믿지 않습니다. 같은 ID의 raw 조회가 현재 모델을 대체합니다. Nova envelope의 `server` field 안 객체를 파일에 넣습니다.

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
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
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	file := flag.String("server-json", "server.json", "supplied Nova server JSON object")
	mode := flag.String("mode", "active", "active or wait")
	activeWait := flag.Bool("active-wait", false, "wait for GetActiveServer IP readiness and Nova observation")
	external := flag.String("floating-network", "", "exact allocation network name; empty uses roles/router")
	enabled := flag.Bool("auto-ip", true, "enable conditional floating IPv4 assignment")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *file, *mode, *external, *enabled, *activeWait); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, file, mode, external string, enabled, activeWait bool) error {
	if mode != "active" && mode != "wait" {
		return fmt.Errorf("mode must be active or wait")
	}
	wire, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var server compute.Server
	if err := json.Unmarshal(wire, &server); err != nil {
		return fmt.Errorf("supplied server: %w", err)
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	input := compute.AutomaticFloatingIPRequest{Server: &server}
	if external != "" {
		input.Network = resource.Name(external)
	}
	options := []compute.ServerReadyOption{
		compute.WithServerReadyAutomaticIPOptions(
			compute.WithAutomaticIPEnabled(enabled),
			compute.WithAutomaticIPTimeout(180*time.Second),
			compute.WithAutomaticIPPollInterval(time.Second),
			compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
			compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(true)),
		),
		compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Second)),
		compute.WithActiveServerWait(activeWait),
	}
	var result *compute.AutomaticServerIPResult
	if mode == "wait" {
		result, err = conn.WaitForServer(ctx, input, options...)
	} else {
		result, err = conn.GetActiveServer(ctx, input, options...)
	}
	if result == nil && err == nil {
		fmt.Println("supplied server is not ACTIVE")
		return nil
	}
	if result != nil {
		if result.Server != nil {
			fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
		}
		if decision := result.Decision; decision != nil {
			fmt.Printf("needed=%t reason=%s backend=%s\n", decision.Needed, decision.Reason, decision.Backend)
		}
		if assignment := result.Assignment; assignment != nil && assignment.FloatingIP != nil {
			fmt.Printf("floating4=%s status=%s reused=%t allocated=%t\n", assignment.FloatingIP.FloatingIP, assignment.FloatingIP.Status, assignment.Reused, assignment.Allocated)
		}
		fmt.Printf("observed=%t\n", result.Observed)
	}
	if err != nil {
		if errors.Is(err, compute.ErrServerAddressesUnavailable) {
			log.Printf("ACTIVE server has no usable address rows; resources were retained")
		}
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response status=%d", response.StatusCode)
		}
		return fmt.Errorf("server readiness: %w", err)
	}
	return nil
}
```

GetActive의 known non-ACTIVE·ERROR·주소 오류·automatic skip에는 Compute endpoint가 필요하지 않습니다. 필요한 Neutron selection/assignment는 Network 서비스가 필요할 수 있습니다. GetActive의 기본 Neutron async assignment는 IP ACTIVE 대기와 raw Nova 관측을 하지 않으므로 이 작업을 위해 Compute를 사전에 준비하지 않습니다. 기존 Nova-source floating 주소 보충이 필요한 판단은 별도로 lazy Compute를 사용할 수 있습니다. `WithActiveServerWait(true)`로 실제 IP readiness와 raw 관측을 요청하거나 WaitForServer를 사용하면 필요한 시점에 Compute를 발견합니다. Connect 자체의 인증 요청과 이 endpoint 준비를 구분합니다.

## 상태·주소·부분 결과

| 상태/응답 | GetActiveServer | WaitForServer |
|---|---|---|
| supplied non-ACTIVE | nil,nil, 조회/할당 없음 | supplied ID의 raw 현재 모델부터 조회 |
| supplied ERROR/fault | 알려진 Server와 failed-state 오류; mutation 없음 | supplied 상태 대신 raw 현재 모델을 판정 |
| ACTIVE nil 주소 | typed unavailable·알려진 Server, DELETE 없음 | raw ACTIVE nil이면 metadata 준비를 같은 budget에서 poll |
| ACTIVE map/모든 rows가 명시 비어 있음 | typed unavailable·알려진 Server, DELETE 없음 | typed unavailable·마지막 matching Server, DELETE 없음 |
| ACTIVE 실제 rows와 known skip | 결정된 reason과 Server, assignment 없음 | raw readiness 후 같은 skip, observation 필요 없음 |
| Neutron 필요, GetActive wait=false | 접수한 assignment와 partial Server; Observed=false | 적용되지 않음; wait=true 강제 |
| Neutron 필요, wait=true | actual IP ACTIVE 후 exact raw Nova IPv4 floating row 관측 | 동일 |
| later HTTP/decode/Close/source/cause 오류 | 알려진 Server/Assignment와 실제 error proof | 마지막 matching Server/Assignment와 실제 error proof |

Go는 상태를 대소문자 구분 없이 비교하며, pinned Python helper는 ERROR/ACTIVE literal을 비교합니다. ERROR는 user failure states를 비워도 반드시 실패입니다. Native Fault 모델은 source의 key 미제공/null/빈 message 구분과 mutable Resource/extra_data taxonomy 전부를 보존하지 않습니다. malformed addresses를 clean absence로 바꾸지 않습니다. 접수된 GET200/203 body에서 동일 ID의 유효한 Server를 확인한 뒤 Close/source/취소 오류가 나면 마지막 모델과 응답 증거를 함께 반환합니다. wrong-ID·malformed envelope는 대상 모델로 채택하지 않습니다.

Neutron allocation의 accepted201/202 응답에서 read/Close/source 오류가 나도 디코드 가능한 body의 IP 모델과 Allocated=true를 보존합니다. malformed·truncated body로 모델을 확인하지 못한 경우 FloatingIP가 nil일 수 있습니다. 실제 ResponseError의 status/header/body·처리 오류·decode/검증 원인을 함께 확인합니다. Get/Wait/Create/Ensure의 [후속 IP 부분 모델 검증](../docs/sdk-support-ledger.md#neutron-접수-응답의-ip-모델-보존)은 matching Nova GET의 모델 보존 검증과 구분합니다.

raw 조회가 성공하기 전 반환 Server는 supplied 모델의 복사본일 수 있으며, 현재 Nova 상태를 확인한 증거가 아닙니다. 오류 여부·응답 증거·Observed를 함께 확인합니다. GetActive의 async 성공은 요청 접수/모델을 얻었다는 의미입니다. IP ACTIVE 또는 Nova가 목표 floating IPv4를 보았다는 뜻은 `Observed=true`의 sync 완료와 구분합니다. GetActive 기본 async에서 이미 할당한 자원을 자동 삭제하지 않고, association 실패를 새 allocation fallback으로 덮지 않습니다.

## 옵션과 전체 budget

`WithServerReadyAutomaticIPOptions`는 기존 concrete automatic 옵션을 owned slice로 받아 준비합니다. 기본 auto/reuse true·전체180초·raw poll5초이며 caller options가 기본값을 바꿉니다. `WithServerReadyWaitOptions`는 server waiter의 interval/progress/failure 정책을 한 번 준비하고 재사용합니다. 기본 server waiter 자체 timeout은 SDK unlimited, interval5초이며 전체180초 context 안에 있습니다. 더 짧은 server timeout과 부모 deadline이 적용될 수 있으며 단계마다 전체 예산을 다시 시작하지 않습니다.

`WithActiveServerWait`의 기본은 GetActive에서 false이고 WaitForServer는 최종 true로 고정합니다. async라도 전체 정책과 waiter 옵션을 preflight합니다. `WithServerReadyWaitOptions` 안의 status override는 준비 단계에서 거부합니다. 실제 모델의 문자열 attribute인 `Status`를 지정해도 `ErrUnsupported`이며, 미존재·비문자열 attribute도 모델 검증에서 `ErrUnsupported`입니다. 빈 값·구분자 등 잘못된 attribute 문법은 옵션 검증의 `ErrInvalidOption`입니다. IP waiter의 별도 custom status 옵션은 기존 계약을 유지하며 synchronous 완료 후 실제 IP Status가 ACTIVE인지도 검증합니다. 잘못된 옵션 때문에 역할/port/할당 HTTP가 먼저 나가지 않도록 합니다. nested application 옵션을 각 단계에서 다시 적용하지 않습니다. network.WithEnsureNoWait는 최종 prepared 정책에서 async를 선택하기 위한 concrete 옵션이며 기존 automatic/create가 요구하던 actual ACTIVE·관측 기본 계약을 바꾸지 않습니다.

supplied Server의 top-level field와 ID는 옵션 callback 전에 캡처합니다. GetActive의 non-ACTIVE/ERROR 분기는 불필요한 주소·중첩 모델 필드를 읽지 않고, 유효한 ACTIVE 모델은 dependency callback이나 association 전에 분리하여 반환합니다. Wait는 raw 모델을 채택하기 전 supplied의 나머지 필드를 readiness 증거로 사용하지 않습니다. source·raw client·의존 서비스는 단계 중 교체를 검출합니다. `WithServerReadyWaitOptions(resource.WithProgressCallback(func(int)))`는 raw server readiness의 정수 progress입니다. `WithAutomaticIPProgress(func(*Server) error)`는 아직 floating 주소가 수렴하지 않은 raw Server의 분리한 복사본입니다. async Get에는 이 observation callback이 호출될 raw convergence loop가 없습니다. 취소·source 변경 후 추가 fetch/mutation을 보내지 않습니다.

## 남은 소스 계약과 검증

공개 get_active_server/wait_for_server의 typed supplied entry를 제공해도 전체 source operation은 unresolved로 추적합니다. pool > explicit ips > automatic dispatch와 Nova mutation/fallback, full has_service/config/network/session/Resource model, public/private/interface 필드 expansion, cloud get_server의 lookup/defaultquery/broad Exception·missing retry, fault/extra_data/cleanup 정책과 integer remaining-time budget은 남습니다. Go의 noDELETE와 owned partial-response 정책은 명시한 차이이며 Python의 cleanup 결과와 같다고 주장하지 않습니다.

신규 17개 테스트 그룹의 입력 상태·비동기 접수·강제 관측·raw metadata·부분 실패·취소·동일 deadline·180초 기본값·옵션 1회 적용·lazy Compute 수명 검증은 [지원대장](../docs/sdk-support-ledger.md#기존-서버의-active-판정과-상위-대기)에 기록합니다. 집중 검증과 전체 gate, 정확한 독립 main의 컴파일·SHA는 실제 결과를 기준으로 구분합니다. Python runtime·인증된 OpenStack 실행은 검증하지 않았습니다.
