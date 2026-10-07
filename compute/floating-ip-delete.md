# Floating IP 삭제와 결과 확인

`Connection.DeleteFloatingIP`과 `compute.Service.DeleteFloatingIP`은 IP ID를 받아 삭제 backend를 선택하고, 성공 뒤 일반 Get으로 결과를 확인합니다. 기본은 첫 DELETE와 최대 한 번의 추가 DELETE입니다. 애플리케이션이 builder나 retry resolver를 구현할 필요 없이 SDK가 옵션, HTTP 요청과 조회 기록을 관리합니다.

```go
result, err := conn.DeleteFloatingIP(ctx,
	compute.DeleteFloatingIPRequest{ID: ipID},
	compute.WithFloatingIPDeleteRetries(1),
	compute.WithFloatingIPDeleteTimeout(time.Minute),
)
```

`Deleted`는 Python cloud의 최종 bool에 대응합니다. `Down`이면 row가 아직 존재해도 성공으로 처리하며, retry0이면 DELETE 접수만으로 true가 됩니다. 실제 부재를 확인했는지는 `Absent`를 별도로 확인합니다. 오류가 있어도 `result`에 앞서 접수한 DELETE와 조회 기록이 남을 수 있습니다.

## 기본 순서와 옵션

1. configured source가 `FloatingIPNeutron`이고 Network 서비스가 있으면 Neutron을 사용합니다. 그 밖에는 Nova를 사용합니다. `FloatingIPNone`도 explicit 삭제에서는 Nova 경로입니다.
2. 정상 DELETE404는 `Deleted=false`, error=nil입니다. 추가 조회·retry를 하지 않으며 **Neutron DELETE404를 Nova DELETE로 fallback하지 않습니다.**
3. `Retries=0`이면 성공 응답을 받은 뒤 확인 조회를 생략합니다. 기본값1은 성공한 DELETE 뒤 일반 Get을 수행합니다.
4. Get에서 IP가 없거나 `status == "DOWN"`이면 `Deleted=true`입니다. 다른 상태이면 다음 DELETE로 이어가며 sleep/backoff는 없습니다.
5. 허용된 횟수 이후에도 row가 있으면 `*compute.FloatingIPDeleteVerificationError`입니다. `ID`, 실제 논리적 `Attempts` 횟수와 마지막 `FloatingIP`을 확인할 수 있습니다.

retry는 초기 요청에 더하는 추가 횟수입니다. 음수는 DELETE를1회로 제한하지만 확인을 수행합니다. 첫 DELETE 접수 → Get ACTIVE → 둘째 DELETE404이면 최종 `Deleted=false`입니다. false를 “어떤 DELETE도 접수되지 않았다”로 읽으면 안 됩니다.

| 공개 옵션 | 동작 |
|---|---|
| `WithFloatingIPDeleteRetries(n)` | 기본1; 0은 검증 생략; 음수는 DELETE1회와 검증 |
| `WithFloatingIPDeleteSource(source)` | cloud 설정 대신 이번 삭제의 source 선택 |
| `WithFloatingIPDeleteLocation(location)` | 검증 조회의 resource location 지정 |
| `WithFloatingIPDeleteStrict(bool)` | Nova 조회 정규화의 strict 모드 |
| `WithFloatingIPDeleteDirectGet(bool)` | UUID형 ID의 검증 조회를 직접 GET으로 선택 |
| `WithFloatingIPDeleteTimeout(duration)` | 양수만 허용; 전체 backend 연결·DELETE attempts·후속 조회에 한 budget |
| `WithUnlimitedFloatingIPDeleteTimeout()` | SDK의 추가 timeout을 제거; caller context는 계속 적용 |
| `WithFloatingIPDeleteOptions(compute.FloatingIPDeleteOpts{...})` | 전체 옵션을 소유한 복사본으로 교체 |

옵션이 없을 때 `Retries=1`이며 추가 SDK deadline은 없습니다. bulk 옵션의 `FloatingIPDeleteOpts{}`는 필드의 zero value를 명시한 것이므로 `Retries=0`입니다. 기본 retry를 유지하려면 bulk 값에도 `Retries: 1`을 지정합니다. 뒤에 오는 setter가 앞선 값을 덮어씁니다. 옵션 slice와 pointer/location 값은 SDK가 복사하고, caller 옵션 함수는 전체 작업에서 각각 한 번만 실행합니다. `PrepareFloatingIPDeleteOptions(ctx, options...)`로 HTTP 없이 옵션을 준비할 수 있습니다.

빈 ID와 invalid 옵션은 요청 전 오류입니다. 이번 API에는 search filters가 없습니다. 삭제 결과 확인에 필터를 끼워 ID 선택을 바꾸지 않습니다.

## 결과와 실제 응답

`DeleteFloatingIPResult`는 `ID`, 마지막 DELETE의 `Backend`, 최종 `Deleted`·`Absent`·`Down`, 순서대로 누적한 `Attempts`, `LastVerification`, `Failure`를 제공합니다.

| 필드 | 의미 |
|---|---|
| `Attempts[i].Accepted` | 해당 DELETE의 허용된 성공 응답을 확보함. 응답 body 읽기·close 또는 source/context 오류가 함께 있을 수 있음 |
| `Response` | DELETE의 실제 status/header와 passive body인 `Envelope` |
| `Failure` | DELETE 또는 검증 조회에서 확보한 실패 응답 기록; 마지막 결과만으로 과거 attempt의 기록을 대신하지 않음 |
| `NotFound` | 정상 DELETE404의 원래 오류. 이 경우 `Accepted=false`, 검증 없이 최종 false |
| `Verification` | 해당 DELETE 뒤 일반 Get의 결과·Pages·Observed·fallback/실패 기록 |
| `Verified` | 검증 조회가 오류 없이 끝남; 실제 부재 확인과 같지 않음 |
| `Absent` | 성공한 검증 조회의 no-match. DELETE404 자체를 부재 확인으로 기록하지 않음 |
| `Down` | 성공한 검증 조회가 `DOWN`인 row를 반환함 |

`LastVerification`은 가장 최근에 수행한 조회이며, 이후 DELETE가 실패해도 앞선 조회가 남습니다. DELETE의 backend와 검증 조회의 backend가 다를 수 있으므로 `Verification.Backend`도 확인합니다. native transport retry는 새 논리적 attempt로 세지 않습니다.

DELETE 응답은 리소스 JSON으로 해석하지 않습니다. 최초 성공 정책인 HTTP200–399를 고정하고 실제 body/header/status를 보존합니다. Python의 `<400` 비교와 달리 final informational1xx는 허용하지 않습니다. 이는 Go transport의 명시적인 성공 정책 차이입니다. `Envelope`는 `json.RawMessage`이지만 passive body가 유효한 JSON이라는 보장은 없으므로, 전체 result를 `json.Marshal`하면 body 때문에 실패할 수 있습니다. 아래 예제는 scalar 결과와 counts만 출력합니다.

## 검증 조회와 legacy Nova

후속 조회는 [일반 Get](floating-ip-queries.md)의 기본 목록 기반 선택을 따릅니다. direct-ID GET을 임의로 대신 사용하지 않습니다. `DirectGet=true`이고 UUID형 ID일 때 직접 GET을 사용하며, decimal Nova ID는 목록 검색을 계속 사용합니다. opt-in 직접 GET404는 일반 Get 계약에 따라 오류로 전파됩니다. `DOWN` 판정은 `Verification.FloatingIP.Resource`의 cloud 조회 view를 사용합니다. Nova source의 wire `DOWN`이 합성 `ACTIVE`로 정규화되면 삭제 성공 조건을 만족하지 않으며, 실제 `Wire`는 그대로 남습니다.

Neutron 검증의 unfiltered 목록404 → Nova fallback, Nova 목록404 → empty 결과와 duplicate/403/decode 오류도 일반 Get 계약을 따릅니다. DELETE404 처리와 후속 조회의 fallback은 별개의 단계입니다. 조회 오류를 부재로 감추지 않습니다. source가 반환한 direct member의 ID를 별도로 대조하는 정책은 추가하지 않습니다.

Nova DELETE는 legacy `/os-floating-ips/{id}`를 사용합니다. selected Compute microversion이2.36 이상이면 `resource.ErrUnsupported`를 포함한 오류이며, SDK가 버전을 낮추지 않습니다. Nova 또는 검증의 Nova fallback이 필요한 cloud에서는 호환 endpoint/version이 실제 제공되어야 합니다. [legacy Nova 가이드](server-nova-floating-ip.md)를 함께 참고하세요.

## 독립 Go 예제

다음 프로그램은 cloud `dev`의 인증을 사용해 지정한 IP를 실제 삭제합니다. `-retry 0`은 검증을 생략하며, `-source nova`는 legacy backend를 선택합니다. Compute2.35를 명시해 필요한 Nova fallback의 호환 버전을 선택합니다. Neutron 경로에서는 Compute client를 필요할 때 연결합니다.

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

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/compute"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	id := flag.String("id", "", "floating IP ID to delete")
	retries := flag.Int("retry", 1, "additional deletes; zero skips verification")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	direct := flag.Bool("direct", false, "use direct GET for a UUID verification ID")
	flag.Parse()
	if *id == "" {
		return errors.New("-id is required")
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
	opts := []compute.FloatingIPDeleteOption{
		compute.WithFloatingIPDeleteRetries(*retries),
		compute.WithFloatingIPDeleteDirectGet(*direct),
		compute.WithFloatingIPDeleteTimeout(time.Minute),
	}
	if *source != "" {
		opts = append(opts, compute.WithFloatingIPDeleteSource(compute.FloatingIPSource(*source)))
	}
	result, deleteErr := conn.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: *id}, opts...)
	var summary map[string]any
	if result != nil {
		accepted := 0
		for _, attempt := range result.Attempts {
			if attempt.Accepted {
				accepted++
			}
		}
		summary = map[string]any{
			"id": result.ID, "backend": result.Backend,
			"deleted": result.Deleted, "absent": result.Absent, "down": result.Down,
			"attempts": len(result.Attempts), "accepted_attempts": accepted,
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(deleteErr, encoder.Encode(summary))
}
```

## openstacksdk 비교와 남은 범위

```python
import openstack

conn = openstack.connect(cloud="dev")
deleted = conn.delete_floating_ip("IP_ID")            # 최대 DELETE2회 + 성공마다 Get
accepted = conn.delete_floating_ip("IP_ID", retry=0)  # 확인 Get 없음
```

비교 pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. [public retry/verification](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L977-L1017), [Neutron/Nova delete helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1019-L1046), [Get 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L148-L201)을 기준으로 합니다. Python은 최종 bool 또는 exception을 반환하며, Go는 bool에 대응하는 `Deleted`와 실제 응답·부분 결과를 함께 제공합니다. 음수 retry에서 Go exhaustion 오류는 source의 `retry+1` 표현 대신 실제 attempts 횟수를 기록합니다.

SDK의 추가 timeout은 Python Delete에는 없는 옵션입니다. 옵션 준비 후 하나의 budget을 backend 연결·전체 retry·검증에 적용하며 caller context의 취소 원인을 보존합니다. 예제는 Connect를 포함한 부모2분 예산과 Delete의1분 예산을 사용합니다. mutable Resource/Munch와 전체 inherited session/cache/discovery/context 동작이 같은 것은 아닙니다. 여기서는 owned 조회 모델과 source 선택·retry·검증 흐름을 다룹니다.

기존 generic `Network.FloatingIPs.Delete`의 ignoreMissing/error-only 정책, [Available](floating-ip-available.md)의 free IP 조회·할당, [IP helper](server-ip-helpers.md)의 서버 연결과 이 standalone 삭제를 구분합니다. 이 API는 서버·port detach, `delete_unattached_floating_ips`, Create의 timeout cleanup, 직접 Network Proxy의 revision-aware Delete 전체를 수행하지 않습니다. 전체 Floating IP CRUD와 Python Resource parity는 해당 후속 범위를 함께 검토해야 합니다.

새 IP 할당·선택적 대기·timeout 정리는 [CreateFloatingIP](floating-ip-create.md)에서 제공합니다.

[미연결 Floating IP 일괄 정리](floating-ip-unattached-delete.md)는 Neutron 전체 목록을 확보한 뒤 port가 비어 있는 항목을 순차 삭제합니다. 개별 false는 계속 처리하고 오류는 중단하며 SDK 소유 옵션·한 deadline·항목별 부분 결과를 제공합니다.
