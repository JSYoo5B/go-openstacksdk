# Cloud 라우터 생성: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud `v2.15.0`입니다. `Connection.CreateRouterRecord`가 캐시한 Neutron과 위치 기본값을 제공하고 `network.Service.CreateRouterRecord`가 cloud 기본값, 외부 게이트웨이 구성, 가용 영역 확장 검사와 POST를 소유합니다. 일반 호출에는 별도 builder/interface 구현이 필요하지 않습니다.

| Python | Go | 의미 |
| --- | --- | --- |
| `conn.create_router(name="edge")` | `service.CreateRouterRecord(ctx, network.WithRouterCreateName("edge"))` | admin_state_up 기본 true |
| `admin_state_up=False` | `WithRouterCreateAdminStateUp(false)` | 명시 false 보존 |
| `ext_gateway_net_id="external"` | `WithRouterCreateExternalGatewayNetworkID("external")` | gateway.network_id 구성 |
| `enable_snat=False` | `WithRouterCreateEnableSNAT(false)` | false도 gateway에 포함; 서버 admin 정책 확인 |
| `ext_fixed_ips=[{...}]` | `WithRouterCreateExternalFixedIPs([]map[string]any{{...}})` | gateway.external_fixed_ips 구성 |
| `project_id=""` | `WithRouterCreateProjectID("")` | null과 달리 빈 값도 포함 |
| `availability_zone_hints=[]` | `WithRouterCreateAvailabilityZoneHints()` | 빈 목록도 확장 검사 뒤 body에 포함 |
| Router Resource | `result.Router` | Source descriptor/default/입력 seed view |
| 실제 응답 | `result.Wire`, `Envelope`, `Header`, `StatusCode` | POST의 실제 값과 증거 |
| 확장 단계 | `result.Extensions` | 이번 호출에서 실제로 관찰한 별도 확장 목록 |

선택 kwargs는 정확히 `name`, `admin_state_up`, `ext_gateway_net_id`, `enable_snat`, `ext_fixed_ips`, `project_id`, `availability_zone_hints`입니다. `WithRouterCreateJSON(rawObject)`, `WithRouterCreateAttribute(key, value)`, `WithRouterCreateOpts(value)`로 complete UTF-8 JSON의 null/false/non-string 값도 표현할 수 있습니다. JSON/Opts 옵션은 kwargs 전체를 대체하고 뒤 개별 옵션은 같은 키를 덮습니다. SDK가 map·slice·JSON bytes·location을 복제합니다. `tenant_id`, `is_admin_state_up`와 unknown kwargs는 pinned cloud 함수의 인수 계약에 없으므로 HTTP 전에 오류입니다. 일반 Proxy.create_router raw attrs API 전체의 완료 판정으로 이 좁은 leaf를 승격하지 않습니다.

## 기본값과 요청 순서

`admin_state_up`은 생략하면 true이고 명시 null이면 null, false/0/string/list/object도 Source처럼 원래 값으로 보냅니다. bool descriptor 변환은 반환 Resource view에서 수행합니다. `name`은 Python-loaded truthiness가 true일 때만 포함합니다. `project_id`는 null이 아니면 빈 문자열/false/0/빈 목록도 포함합니다.

게이트웨이 network_id와 external_fixed_ips는 truthy 입력만 포함하지만 enable_snat은 null이 아니면 false/0/빈 문자열/목록도 포함합니다. 이 세 항목 중 하나라도 포함되면 `external_gateway_info`를 구성하며, 전부 제외되면 gateway key를 보내지 않습니다. External network ID가 생략돼도 enable_snat:false만으로 gateway object가 생깁니다. SDK가 추가 ID 조회, 값 보정이나 서버 policy 우회를 수행하지 않습니다.

AZ 힌트가 생략되거나 null이면 확장 HTTP 없이 POST합니다. Non-null 힌트는 먼저 list인지 검증합니다. False, 0, string, object는 HTTP0 오류이며 list 요소를 SDK가 string으로 보정하지 않습니다. List라면 매 호출 fresh 전체 `router_availability_zone` 확장 조회를 완료한 뒤 POST합니다. 빈 list도 동일합니다. 첫 alias가 일치해도 뒤 unhashable alias나 후속 page 오류를 숨기지 않습니다. 확장이 없으면 `network.ErrUnavailableExtension`, 확장 실패면 원래 오류이며 POST를 수행하지 않습니다.

성공하면 `POST /routers`에 `{"router":...}`를 한 번 보냅니다. Source 100..399 acceptance는 이 새 create leaf에만 적용하고 기존 native Create/CreateWithAttributes/read 정책은 보존합니다. Clean HTTP rejection은 원래 native value 오류와 실제 status/body/header를 보존하며 404도 생성 성공이나 missing으로 바꾸지 않습니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kwargs", default="{}")
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    router = conn.create_router(**kwargs)
    print(json.dumps(router.to_dict(), indent=2))
```

`python main.py --cloud dev --kwargs '{"name":"edge","availability_zone_hints":[]}'`

## 독립 Go main

[설치 안내](../docs/install.md)에 따라 module을 준비하고 다음을 main.go로 저장합니다.

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kwargs := flag.String("kwargs", "{}", "JSON create_router kwargs")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	result, err := conn.CreateRouterRecord(ctx,
		network.WithRouterCreateJSON(json.RawMessage(*kwargs)),
	)
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(result.Router, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

`go run . -cloud dev -kwargs '{"name":"edge","availability_zone_hints":[]}'`

이 main은 실제 라우터 POST를 수행합니다. 이미 구성된 native client를 쓰려면 `network.New(client).CreateRouterRecord(ctx, options...)`를 사용합니다. Standalone service의 location 기본은 null이며 예제의 Connection helper가 owned 위치를 기본으로 전달합니다. 명시 `WithRouterCreateCurrentLocation` 옵션으로 이를 바꿀 수 있습니다. Location은 POST body에 들어가지 않습니다. `routers.API.CreateCloudRecord`는 이미 번역된 body를 위한 좁은 leaf이므로 일반 애플리케이션은 기본값과 gate를 수행하는 service 함수를 사용합니다.

## 응답과 실패 증거

Source Resource는 입력 seed를 부분 응답과 병합합니다. Sparse response가 admin_state_up/name/project_id를 생략하면 해당 입력을 유지하지만 Wire에 입력값을 추가하지 않습니다. Explicit 응답 null은 seed를 덮습니다. `admin_state_up`은 Resource의 `is_admin_state_up`, `revision`은 `revision_number`, tenant_id는 project_id alias로 읽으며 tags 기본은 []입니다. Project seed가 명시 빈 문자열이면 tenant_id 응답만으로 이를 덮지 않습니다. List/dict/bool/int descriptor casts와 Source MRO의 revision shadow를 기존 프로필에서 재사용합니다.

Selected microversion과 SDK location은 같은 이름의 응답 확장값을 Resource view에서 대체하되 실제 Wire 값은 보존합니다. 설정한 location의 foreign project에는 응답 project ID를 유지하고 현재 project 이름을 지웁니다. 반환된 Resource/Header/Envelope를 바꿔도 Wire와 이전 확장 증거는 바뀌지 않습니다.

Failed read/Close, custom context cancellation, selected source/API/route 변경은 재시도/reauth hook을 호출하거나 success로 숨기지 않습니다. Native401은 Unwrap 없는 ErrUnableToReauthenticate의 value ErrOriginal 및 ErrReauth를 보존합니다. Clean retry와 live token renewal은 고정 body/route/header로 계속 지원합니다. Accepted 응답의 read/Close/decode 실패는 실제 POST receipt와 오류를 반환하며 성공 Resource를 지어내지 않습니다. 확장 단계 실패의 실제 GET proof는 오류 chain/관찰한 Extensions에 있고 POST StatusCode/Envelope는 비어 있습니다.

## 서버 권한과 지원 범위

[Neutron 27 create_router 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L57-L145)은 기본 생성 및 gateway/network_id에 project scope `ADMIN_OR_PROJECT_MEMBER`를 요구합니다. **enable_snat과 external_fixed_ips를 명시하는 field 정책은 `ADMIN`**입니다. False 값도 명시한 enable_snat이며 권한 검사를 피해 생략하지 않습니다. Project/외부 네트워크 소유권, 배포 policy override, 드라이버·입력 유효성은 서버가 판단합니다. Deprecated historic 정책과 현행 기본 정책을 구분하며 확장 discovery 성공이 생성 권한을 보장하지 않습니다.

이번 완료 대상은 cloud create_router 한 method입니다. Existing native/shared mutation, network/subnet 역할 cache 및 notifications를 변경하지 않았습니다. Router 생성은 현재 역할 cache inventory를 변경하지 않으므로 Reset하지 않습니다. Python Proxy의 일반 URL API cache invalidation이나 모든 network cache를 이 helper가 완성했다고 주장하지 않습니다.

JSON object wire key order, complete UTF-8 JSON domain, owned immutable values와 고정 service/header protocol은 Go 정책입니다. Python float 입력의 JSON serialization과 custom runtime objects, mutable Resource/class/adapter/session/cache/current_location 및 동적 class dispatch는 SDK-R1/SDK-C1/SDK-S1의 공통 기반 차이로 남깁니다. 일반 raw unknown Body 보존을 custom class 동작과 같다고 주장하지 않으며 이 mapping의 full SDK supported 증가분은 0입니다.

실제 pinned cloud method/helper→Neutron Proxy._create→Resource.new/create/translate/descriptors를 Python3.14.8에서 90 cases로 실행했습니다. HTTP Adapter.post와 extension iterator만 대체했고 live cloud/Keystone authorization 증거는 아닙니다. Go 10 cloud 그룹과 3 leaf 그룹은 기존 Gophercloud public testhelper-backed internal/testcloud 및 기존 router/QoS physical body observer를 재사용합니다. 최종 Go·race·독립 main·전체 repository gate의 PASS는 root가 실제 실행한 결과로 기록합니다.

Source: [cloud create_router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2250-L2307), [gateway helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2882-L2901), [Router descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py), [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py).
