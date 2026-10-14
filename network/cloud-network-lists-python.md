# Network 목록·검색: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud `v2.15.0`입니다. `conn.Network(ctx)`가 반환한 `network.Service`의 두 메서드가 Source query 합성, JSON descriptors, local Body filters, 전체 페이지, 연결 location, 실제 응답 증거를 처리합니다. 개발자가 builder/interface나 페이지·별칭 처리기를 만들 필요가 없습니다.

| Python | Go | 반환 |
| --- | --- | --- |
| `conn.list_networks(filters=...)` | `service.ListNetworkRecords(ctx, options...)` | `NetworkCloudRecords` |
| `conn.search_networks(name_or_id, filters=...)` | `service.SearchNetworkRecords(ctx, name, options...)` | `NetworkCloudRecords` |
| `conn.network.networks(**query)` | `service.API.Networks.ProxyCloudListRecords(ctx, location, options...)` | lazy generator 또는 arbitrary JMES 값 |
| 일반 Network Resource 목록 | `result.Networks`, `result.Values`, `result.JSON` | owned view와 실제 proof 분리 |

## 지원 범위와 남은 named 차이

이번 구현은 두 cloud 메서드와 기존 networks 목록을 보강하는 **완결된 Go 사용 단위**입니다. Source 전체 named parity 판정은 unresolved로 유지합니다. 모든 API/SDK 완료나 generic Resource/cache/lifecycle 전체 지원을 주장하지 않습니다.

현재 inherited Engine/REST는 `max_items`와 `limit`를 nonnegative Go integer로 제한합니다. Source JSON kwargs는 `max_items=true`, 음수, 실수도 받습니다. 예를 들어 -1은 limit=-1로 GET 후 첫 raw row 전에 빈 결과를 반환하고, 0.5는 한 raw row, 1.5는 두 raw row 뒤 계속 페이지를 확인합니다. 문자열 cap은 empty page GET에서는 성공할 수 있고 첫 실제 row에서 비교 TypeError가 납니다. Raw limit false/음수/실수/문자열도 Source와 현재 Go의 검증 시점·허용 범위가 다릅니다. 더구나 **정수 max_items=2에서 페이지가 정확히 두 row이면 Source는 다음 페이지도 GET하지만 Go는 두 번째 raw row 뒤 즉시 중단**합니다. 다음 요청의 실패·취소·source guard 증거가 달라지는 실제 reachable gap이며 named remaining으로 추적합니다.

Go base_path는 같은 service origin의 안전한 collection 경로를 선택합니다. Source의 arbitrary URI `%` mapping-format과 foreign/fragment/query/traversal retargeting은 제공하지 않습니다. 연결이나 임의 adapter/class/runtime 객체 전달도 SDK-owned 원래 source로 제한됩니다. 이미 있는 typed/Collection/`ProxyListRecords`는 기존 계약을 유지하며, 아래 새 Source cloud lane과 혼동하지 않습니다.

## 기본값·옵션·서버 필터

기본 필터는 비어 있고 pagination은 켜져 있습니다. `WithNetworkCloudFilters(json.RawMessage(...))`는 원본 bytes를 복제하며 omission/null/false/zero/empty/큰 integer를 float64로 바꾸지 않습니다. Bulk `WithNetworkCloudListOpts`와 뒤 개별 With 옵션은 순서대로 적용됩니다. Headers·location·version·service-available도 독립 복제됩니다.

List의 falsey filters는 빈 mapping입니다. Truthy List filters는 **mapping expansion**을 위해 JSON object여야 합니다. Search는 truthy name을 먼저 server `name` query에 넣고 truthy filters로 ordered dict.update합니다. 같은 filters.name이 name seed를 덮으며 Search의 JSON list-of-pairs와 two-character string pair도 허용됩니다. Search에 UUID나 `*`를 넣어도 literal **name query**입니다. Client ID/name/glob 또는 `_filter_list` 검색을 추가하지 않습니다. 서버가 반환한 이름이 다르면 결과에서 임의로 지우지 않습니다.

`ListNetworkRecords(..., WithNetworkCloudServiceAvailable(false))`는 nil service에서도 필터·JMES·HTTP보다 먼저 nonnil `[]`를 반환합니다. ServiceAvailable은 SDK의 구체적인 service-discovery 사실을 주입하는 옵션입니다. Search에는 이 absent-service 빈 목록 규칙이 없으며 Neutron service가 필요합니다. Catalog/auth/HTTP/physical 실패를 서비스 없음으로 숨기지 않습니다.

Declared query canonical name은 server wire alias로 옮깁니다. 예: is_shared→shared, is_admin_state_up→admin_state_up, subnet_ids는 server query가 아닌 local Body filter입니다. Unknown kwargs는 Source처럼 무시하고, 조회 가능한 provider/segments Body 필터와 raw integer·nested bool/number equality는 library가 처리합니다. Network.project_id는 tenant_id alias가 **아닙니다**. Tenant-only row에서 tenant_id로 프로젝트를 추론하지 않습니다.

`WithNetworkCloudHeaders(map[string]string)`와 `WithNetworkCloudMicroversion(string)`는 Go transport convenience입니다. Source **raw kwargs**의 `headers`, `microversion`, `base_path`, `paginated`, `max_items`, `__conflicting_attrs`, `jmespath_filters`는 Filters object 안에 전달합니다. 원본 Source kwargs와 Go extra가 동시에 있으면 복제된 개별 transport option이 해당 original raw formal을 대체합니다. Conflicting attrs의 복원 시점과 bound collision은 Source Proxy 순서를 유지합니다.

## 지연된 JMES·검증 시점

Source Proxy는 list가 아닌 opaque **generator**에 JMES를 적용합니다. `@`는 실제 생성기를 반환하며 cloud의 list()가 HTTP와 페이지를 소비합니다. `type(@)`, `[0]`, `[*]`의 Proxy 값은 null이고 cloud list(None)는 오류입니다. `length(@)` 같은 type error는 HTTP 없이 발생합니다.

Literal array/string/object는 HTTP 없이 iterable cloud 결과가 됩니다. Go의 object iterator는 lexical keys를 사용하므로 Source `{z:0,a:1}`의 insertion order `["z","a"]`와 달리 `["a","z"]`입니다. 테스트가 이 Go 차이를 명시적으로 확인합니다. `to_array(@)` 또는 `[@]`는 generator 하나를 **소비하지 않고** 보관합니다. `Values`에 opaque generator가 남으면 `JSON=nil`, `Networks=[]`이며 fabricated HTTP metadata를 넣지 않습니다. 생성기를 실제로 반복할 때 원래 source·context·location guard가 적용됩니다.

새 leaf `WithCloudRecordListQuery(key, raw)`는 Resource formals를 생성기 소비 시 검증합니다. 그러므로 literal JMES `[]`는 invalid raw headers/base_path/microversion를 소비하지 않고 성공할 수 있습니다. Original self/resource_type는 Proxy call에서 binding error이고 restored self/resource_type는 unknown Resource params로 무시합니다. Restored cls/session와 explicit base_path/paginated duplicate는 Proxy call에서 오류입니다. Original JMES는 conflicting attrs로 복원된 JMES spelling에 덮이지 않습니다.

Raw base_path=null은 default, **empty string은 HTTP 없는 empty generator**입니다. False/0/array/object base_path는 generator 소비 시 오류입니다. Falsey raw headers는 무시하고 truthy non-map/잘못된 header는 소비 시 오류입니다. Falsey nonnull raw microversion는 version header를 제거합니다. 실제 Keystoneauth JSON-domain normalization에 맞춰 true/1→network 1.0, [2,4]→network 2.4, null→selected default가 됩니다. Resource의 Python runtime microversion 속성과 mutable object lifecycle은 raw JSON Resource 모델과 별도 SDK parity 범위입니다. JSON exponent overflow의 reachable 차이도 남습니다: CPython이 raw `1e400` 또는 `[1e400]`를 +inf로 읽으면 실제 KSA는 `network latest`로 normalize하지만 현재 Go finite numeric conversion은 오류를 반환합니다. `[2,1e400]`는 KSA에서도 major.latest 오류입니다. 이 raw microversion 경계는 named remaining이며 arbitrary non-JSON object 차이로 숨기지 않습니다. 추가로 Source `base_path=""`는 URI가 비어 HTTP/KSA 호출을 하지 않으므로 `microversion="broken"` 또는 `["wrong"]`도 빈 목록으로 끝나지만, 현재 Go는 empty-path 반환 전에 microversion을 normalize하여 오류를 반환합니다. 이 empty-route/invalid-MV 검증 시점 차이도 같은 named remaining에 포함합니다.

## Resource view·location·증거·권한

29개 declared Body fields의 null/default/list/bool/int getter를 소유하며 tags 기본 [], port-security/router-external 기본 false를 보존합니다. Unknown Body와 큰 integer를 유지합니다. 정수 descriptor는 Source truncate/bool conversion, 목록 descriptor는 scalar wrapping, local equality는 CPython-loaded float/bool/int comparison을 적용합니다. Resource view를 미리 계산하므로 사용하지 않을 malformed getter도 Go에서는 일찍 실패할 수 있습니다.

`WithNetworkCloudCurrentLocation(conn.CurrentLocation())`에 해당하는 사실을 주입하면 Source current-project 판단을 적용합니다. Missing/falsey/matching project는 configured project 이름/domain을 유지하고 truthy foreign project는 raw ID만 남깁니다. Unknown availability_zone도 Source Resource getter가 읽는 값이므로 false/null/0/array/object zone을 보존합니다. 서버가 보낸 forged location은 `Wire`에 남고 view는 연결 사실로 계산됩니다. Leaf에 location을 전달하지 않으면 owned null입니다.

각 row의 `Resource`는 Source getter view, `Wire`는 실제 row, `Envelope`는 실제 전체 page bytes, `Header`/`StatusCode`는 실제 page receipt입니다. View에 default/location을 채웠다고 Wire에 추가하지 않습니다. Source cloud lane은 HTTP100..399를 성공으로 해석하며 rejected/physical read·Close/cancel/source-drift는 실제 status/body/header를 오류에 보존합니다. Native401 ErrUnableToReauthenticate의 ErrOriginal은 value-form rejection proof로 유지합니다. Physical fault에서는 retry/reauth callback을 호출하지 않고 clean retry/token renewal은 원래 source를 다시 확인합니다. Cloud는 후반 실패 때 partial public 목록을 버리며 leaf iterator는 먼저 받은 row와 실제 error를 제공할 수 있습니다.

[Neutron 27 network policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py)의 get_network는 project reader/admin, service, shared, external, advsvc branches를 허용합니다. Provider attributes와 segments는 별도 ADMIN rules입니다. 공유 network를 읽을 수 있다고 모든 provider field나 all-project scope가 보장되지 않습니다. SDK는 policy/권한을 승격하지 않으며 fixture는 실제 배포 인증·RBAC·extension/backend 성공 증거가 아닙니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "search"), default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "search":
        result = conn.search_networks(args.name, filters=filters)
    else:
        result = conn.list_networks(filters=filters)
    values = [value.to_dict() if hasattr(value, "to_dict") else value for value in result]
    print(json.dumps(values, indent=2))
```

## 독립 Go main

다음 main은 이미 선택한 Neutron service를 사용하고 CurrentLocation을 명시적으로 복제합니다. Connection의 catalog-discovery wrapper와 달리 `conn.Network(ctx)` 자체가 실패하면 그 service-selection 오류를 반환합니다.

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
	mode := flag.String("mode", "list", "list or search")
	name := flag.String("name", "", "literal server-side network name for search")
	filters := flag.String("filters", "{}", "JSON filters; search also accepts key/value pairs")
	flag.Parse()
	if *mode != "list" && *mode != "search" {
		log.Fatal("mode must be list or search")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	options := []network.NetworkCloudListOption{
		network.WithNetworkCloudFilters(json.RawMessage(*filters)),
		network.WithNetworkCloudCurrentLocation(location),
	}
	var result *network.NetworkCloudRecords
	if *mode == "search" {
		result, err = service.SearchNetworkRecords(ctx, *name, options...)
	} else {
		result, err = service.ListNetworkRecords(ctx, options...)
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON == nil {
		log.Fatal("result contains an opaque lazy generator; inspect result.Values instead")
	}
	var formatted json.RawMessage = result.JSON
	data, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

각 main을 network-lists.py/main.go로 저장한 뒤 `--mode search --name private --filters '{"is_shared":false}'`로 실행합니다. Go는 `go run ./main.go -mode search -name private -filters '{"is_shared":false}'`입니다. Required config/token/endpoint는 clouds.yaml cloud에서 읽습니다. JSON RawMessage를 사용하므로 filters의 큰 정수 원문이 보존됩니다. Nested generator는 일반 JSON print 대상이 아니며 Go main은 Values를 직접 다루도록 오류를 표시합니다.

## 검증 근거

새 테스트는 기존 internal/testcloud가 재사용하는 public Gophercloud testhelper와 기존 native Network fixture를 사용합니다. Cloud 8 + leaf 5 top-level groups는 service absence, falsey/pair filters, server name overwrite, loaded local equality, page receipt, opaque JMES/validation phases, current location, late atomic failure, actual200/401/404/503×read-close/cancel/source12lane, clean retry/token renewal을 준비했습니다. Go 실행은 root 통합 검증 담당이며 이 문서는 준비한 테스트를 실행 PASS로 표시하지 않습니다.

Actual pinned Source 실행 증거는 Network **348 cases**입니다: list/proxy/cloud 57, controls46, connected location36, real KSA microversion11, original/restored binding24, limit/cap174. CPython3.14에서 실제 pinned methods를 import하고 HTTP/auth/config 또는 physical requests만 fixture로 대체했습니다. KSA runtime는 openstacksdk git pin과 별도 파일 SHA를 기록합니다. HTTP 없는 Source oracle/standalone Python AST 확인과 Go fixture/race/build는 서로 다른 증거이며 live deployment를 실행하지 않았습니다. 소유 작업자도 cap boundary/negative/string의 겹치는 24 Network cases를 실제 Source로 재실행하여 확인했습니다. 추가 KSA-only12 boundary cases는 physical HTTP나 pinnedSDK 실행 증거와 구분하여 별도 SHA receipt로 기록했습니다. Limit/cap oracle이 드러낸 남은 Source 동작은 위 named remaining에 보존합니다.
