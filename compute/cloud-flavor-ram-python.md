# RAM으로 Flavor 선택: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `conn.Compute(ctx)`에서 얻은 service의 `GetFlavorByRAM`이 상세 목록, 조건부 extra specs 보충, RAM 정렬과 이름 부분 문자열 선택을 제공합니다. 애플리케이션은 builder/interface를 구현하지 않아도 됩니다.

**기능 구현을 제공하지만 완료 집계는 unresolved입니다.** 기존 Flavor reader가 200~399만 수락하여 Source의 100~199 응답과 차이가 남습니다. Source의 실제 list와 extra-specs 응답 status199가 모두 성공하는 것을 별도 실행해 확인했습니다. 이 도달 가능한 차이를 일반 SDK 차이로 치환하거나 전체 호환으로 집계하지 않습니다.

| Python | Go | 의미 |
| --- | --- | --- |
| `conn.get_flavor_by_ram(4096)` | `service.GetFlavorByRAM(ctx, 4096)` | RAM이 4096 이상인 첫 정렬 항목; 기본 extra=true |
| `include="m1."` | `compute.WithFlavorRAMInclude("m1.")` | 이름의 대소문자 구별 부분 문자열; glob 아님 |
| `get_extra=False` | `compute.WithFlavorRAMGetExtra(false)` | extra-specs 보충 생략 |
| JSON에서 읽은 동적 minimum/include/get_extra | `json.RawMessage`, `WithFlavorRAMInclude`, `WithFlavorRAMGetExtraValue` | SDK가 JSON-domain 값 snapshot·truthiness·비교·membership 소유 |
| 선택 Resource | `result.Flavor.Resource` | descriptor/default/location이 적용된 선택 view |
| 중간 작업 | `result.Inventory` | 소비한 실제 행과 enrichment; 오류 때도 반환 가능 |
| 결과 없음 `SDKException` | `errors.Is(err, compute.ErrNoFlavorByRAM)` | 로컬 선택 실패이며 HTTP404를 만들지 않음 |

## 순서와 기본값

Source 함수는 `_compute.py`의 `get_flavor_by_ram(ram, include=None, get_extra=True)`입니다. `ram` 인수의 int annotation을 런타임 cast로 해석하지 않습니다. SDK는 모든 상세 페이지와 필요한 extra-specs 조회를 먼저 완료하고, **RAM 오름차순으로 안정 정렬한 뒤** `ram >= requested`와 include를 평가합니다. 같은 RAM은 응답과 페이지의 원래 순서를 유지합니다. 포함 여부는 RAM 조건을 통과한 행에서만 검사합니다.

기본 extra는 **true**입니다. 기존 `AllFlavors`의 기본 false와 구별합니다. 일찍 선택할 수 있는 행이 있어도 뒤 페이지/extra 조회가 실패하면 선택 결과를 반환하지 않습니다. 이미 truthy inline extra_specs가 있으면 추가 조회를 생략합니다. Query에 minimum RAM/include를 보내거나 서버에서 먼저 거르지 않습니다.

Source RAM descriptor를 기존 reader에서 재사용합니다. 누락은 0, 명시 null은 null, bool은 bool 그대로이며 수치 비교에서 0/1, JSON float는 실제 binary float 값을 int로 자르고 digit-only string을 int로 읽습니다. signed string `"-1"`은 이 descriptor에서 0입니다. Float `1e23`은 정수 `99999999999999991611392`로 읽혀 JSON integer `100000000000000000000000`보다 작습니다. Threshold는 임의 정밀도 integer와 JSON-loaded binary float를 구별해 비교합니다. Float를 짧은 십진 문자열로 바꿔 인접한 큰 integer와 잘못 같다고 하지 않습니다.

JSON null RAM의 sort/compare는 오류입니다. 정렬은 include보다 먼저 수행됩니다. Null/string/container threshold도 실제 비교에 도달하면 오류이며, 빈 목록에서는 비교 없이 no-match가 됩니다. Include의 null/false/0/빈 string/list/object는 조건을 생략합니다. 일반 string name은 substring, JSON list name은 loaded Python 동등성 기반 membership, JSON object name은 key membership입니다. Truthy wrong-type include와 null/noniterable name 오류는 해당 membership에 도달했을 때 발생하므로 RAM 아래 행의 잘못된 이름이 선택을 막지 않습니다. Python 예외 클래스 대신 Go의 ErrInvalidOption/error chain을 제공합니다.

`FlavorRAMOpts`는 concrete option DTO이고 `WithFlavorRAMOpts`는 전체 설정을 복제해 대체합니다. `WithFlavorRAMInclude`, `WithFlavorRAMGetExtra`, `WithFlavorRAMGetExtraValue`가 개별 필드를 설정합니다. `WithFlavorRAMQueryOptions`는 기존 `FlavorQueryOption` 목록을 복제해 대체하며 header/version 등 기존 read 옵션을 재사용합니다. QueryOptions에 extra 설정이 있으면 기본 true를 덮고, 명시 RAM GetExtra 필드가 있으면 그것을 최종 적용합니다. Source에 없는 filters 인수는 거부합니다. nil option/context, 잘못된 raw JSON과 변경된 선택 API/client는 조회 전에 실패합니다. 이후 callback이 보관한 옵션 포인터를 바꿔도 이미 적용된 SDK snapshot은 바뀌지 않습니다.

## 실제 응답과 검증 범위

선택 Flavor와 Inventory는 독립적으로 복제됩니다. Resource는 Source descriptor/default와 기존 Connection location view, Wire는 실제 JSON 행입니다. **FlavorRecord.Envelope는 기존 reader 계약대로 해당 flat 행의 원본 bytes**이며 전체 list page를 발명하지 않습니다. Header/StatusCode는 그 행을 얻은 실제 페이지의 값입니다. Enrichment는 별도의 extra-specs 전체 응답과 실제 status/header를 보존합니다. 로컬 선택 오류의 ResponseError가 포함하는 증거도 소비한 마지막 행에서 온 것이며 추가 HTTP 관측을 주장하지 않습니다.

기존 목록/보충 reader의 microversion·safe path·retry/reauth·cancellation/source guard·오류 chain을 재사용합니다. 새 Source status opt-in을 기존 reader에 추가하지 않았으므로 **100~199 성공 상태가 남은 구체적 차이**입니다. 이 변경은 기존 native API와 공통 reader 정책을 바꾸지 않습니다. 일반 clean retry, native401, Read/Close 실패, 사용자 취소 원인과 잠시 변경되었다 복원된 client source를 기존 testhelper 기반 fixture로 검증합니다. Live backend 권한과 배포 정책 검증은 별도입니다.

공개 `any` 인수는 임의 Python 런타임 객체 모델이 아니라 SDK가 소유하는 JSON-domain 값 입력입니다. `json.RawMessage`로 integer/float token을 보존할 수 있습니다. NaN/Infinity float 값, 사용자 Python class/연산자, adapter/session subclass, full Resource MRO·동적 attribute·cache/session foundation은 Go 입력/기존 SDK-R1·SDK-C1·SDK-S1 범위 차이입니다. 매우 큰 JSON integer는 CPython의 설정 가능한 문자열 변환 길이 제한을 재현하지 않습니다. 임의 사용자 callback 실행 시점과 예외 문구를 Python과 같다고 주장하지 않습니다. 이 행의 supported 증가량은 0입니다.

Actual pinned Python3.14.8의 cloud 함수→Nova Proxy→Flavor/Resource list·descriptor·fetch_extra_specs를 49개 사례로 실행했고, 별도 list/extra status199 2개를 확인했습니다. Adapter HTTP 응답과 connection만 fixture로 대체했으며 실서비스 인증 검증이 아닙니다. 독립 Source 검토의 14개 사례도 정렬·extra 보충·동적 값 순서를 대조했습니다. Go에는 7개 테스트 그룹을 추가했으며 최종 통과 여부는 별도의 Go gate 결과로 판정합니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--ram", default="4096", help="minimum RAM as JSON number")
parser.add_argument("--include", default="null", help="name substring as JSON string, or null")
parser.add_argument("--get-extra", action=argparse.BooleanOptionalAction, default=True)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud, compute_api_version="2.55") as conn:
    result = conn.get_flavor_by_ram(
        json.loads(args.ram),
        include=json.loads(args.include),
        get_extra=args.get_extra,
    )
    print(json.dumps(result.to_dict(), indent=2))
```

`python main.py --cloud dev --ram 4096 --include '"m1."' --no-get-extra`

## 독립 Go main

[설치 안내](../docs/install.md)에 따라 module을 준비한 뒤 아래를 main.go로 저장합니다. 출력의 resource와 실제 응답 증거를 구별합니다. 이 예제는 실제 OpenStack endpoint·clouds.yaml이 있을 때 실행할 수 있으며 문서 예제 검증은 컴파일과 Source/fixture 실행 범위입니다.

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	ram := flag.String("ram", "4096", "minimum RAM as JSON number")
	include := flag.String("include", "null", "name substring as JSON string, or null")
	extra := flag.Bool("get-extra", true, "fetch missing extra specs before selection")
	flag.Parse()
	if !json.Valid([]byte(*ram)) || !json.Valid([]byte(*include)) {
		log.Fatal("ram and include must be complete JSON")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, readErr := service.GetFlavorByRAM(ctx, json.RawMessage(*ram),
		compute.WithFlavorRAMInclude(json.RawMessage(*include)),
		compute.WithFlavorRAMGetExtra(*extra))
	var selected any
	inventoryCount := 0
	if result != nil {
		inventoryCount = len(result.Inventory)
		if row := result.Flavor; row != nil {
			selected = map[string]any{
				"resource": row.Resource, "wire": row.Wire,
				"envelope": row.Envelope, "header": row.Header,
				"status_code": row.StatusCode, "enrichment": row.Enrichment,
			}
		}
	}
	data, printErr := json.MarshalIndent(map[string]any{
		"selected": selected, "inventory_count": inventoryCount,
	}, "", "  ")
	if printErr == nil {
		fmt.Println(string(data))
	}
	if err := errors.Join(readErr, printErr); err != nil {
		log.Fatal(err)
	}
}
```

`go run main.go -cloud dev -ram 4096 -include '"m1."' -get-extra=false`

두 예제는 compute microversion2.55를 선택합니다. 별도의 Connection RAM forwarding 메서드를 요구하지 않으며, 기존 compute service를 그대로 사용합니다. Extra 값을 생략하면 true입니다.
