# Cloud 서버 그룹 검색·조회: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud `v2.15.0`입니다. SDK가 서버 그룹 전체 페이지, 선택 버전, 기본 find와 필터 검색의 분기, 응답 소유권과 오류 처리를 제공합니다. 일반 호출에 application builder/interface 구현은 필요하지 않습니다.

**두 항목은 부분 구현이며 완료 집계는 unresolved입니다.** 기본 조회·이름 패턴·순서 있는 dictionary 필터·주요 JSON/JMES 값 처리를 사용할 수 있으나, 아래의 Python Resource 타입/동등성 및 전체 숫자·직렬화·nonfinite JMESPath 차이가 남아 있습니다. 이 차이를 공통 SDK-R1 설명만으로 숨기거나 전체 Source 필터 호환을 주장하지 않습니다.

| Python cloud 함수 | Go service 함수 | 결과 |
| --- | --- | --- |
| `search_server_groups(name_or_id=None, filters=None)` | `SearchServerGroups(ctx, nameOrID, options...)` | Value JSON, Groups, Inventory, Expression |
| `get_server_group(name_or_id, filters=None)` | `GetServerGroup(ctx, nameOrID, options...)` | Value JSON, Group, Observed, Inventory, Filtered |
| `filters={"policy":"affinity"}` | `WithServerGroupCloudFilters(json.RawMessage(...))` | local ordered dictionary 필터 |
| `filters="[].id"` | `WithServerGroupCloudExpression("[].id")` | 임의 JSON 결과; fake Resource/HTTP 증거 없음 |
| 이름 생략/None인 search | search의 빈 name string | identifier 선택을 생략 |
| 명시 빈 string인 get | `GetServerGroup(ctx, "")` | Source의 trailing-slash collection GET, 이름 None과 구별 |
| Source 연결 location | `WithServerGroupCloudCurrentLocation(location)` | POST/query 없이 owned Resource metadata |

옵션은 `ServerGroupCloudReadOpts`, `WithServerGroupCloudReadOpts`, `WithServerGroupCloudFilters`, `WithServerGroupCloudExpression`, `WithServerGroupCloudHeaders`, `WithServerGroupCloudCurrentLocation`입니다. Opts는 전체 옵션을 대체하고 뒤 옵션이 필드 값을 덮습니다. Filters:nil은 옵션을 지우며 raw `null`도 get의 default find를 선택합니다. SDK가 JSON bytes·headers·location을 복제합니다. Header 옵션은 SDK의 안전한 transport 편의이며 Python cloud 함수의 kwargs를 늘리는 의미가 아닙니다.

## 검색과 단일 조회의 다른 순서

Search는 매번 default `compute.server_groups(all_projects=False)` 전체 목록을 수집한 뒤 이름/ID 선택과 필터를 적용합니다. Name, dictionary, JMESPath를 Nova query로 보내지 않습니다. Exact 이름/ID와 case-sensitive fnmatch 패턴을 지원하고 numeric/null/bool 응답 identity의 Python string spelling도 기존 공통 helper를 사용합니다. API에서 제외된 필드나 unknown Body를 임의 extension filter로 승격하지 않습니다.

ServerGroup의 `policies`, `policy`, `member_ids`, `metadata`, `project_id`, `user_id`, `name`, `id`는 untyped Source Body입니다. `rules`만 dictionary getter cast이며 false/nonmapping은 {}로 읽습니다. Missing 필드는 null입니다. `members`는 Wire 이름이고 builtin dict 기반 cloud 필터/JMESPath key는 **canonical `member_ids`**입니다. `members` 필터를 member_ids로 자동 rewrite하지 않습니다. Unknown response fields는 Wire에 보존하고 Resource에는 추가하지 않습니다. 응답에 members/member_ids가 둘 다 있으면 원래 JSON 순서대로 마지막 값이 canonical view를 결정합니다.

Dictionary 필터는 입력 JSON key 순서대로 재귀 평가합니다. 앞 조건이 mismatch이면 뒤 unknown key를 읽지 않지만 앞 unknown key를 읽으면 오류입니다. Missing key를 null로 만들어 unknown 조건을 통과시키지 않습니다. JSON-loaded 숫자와 bool/int 동등성은 opt-in 공통 helper를 재사용합니다.

Get은 **Filters가 생략되거나 null일 때만 default find**를 수행합니다. Literal ID GET 후 clean400/403/404에만 전체 목록에서 exact ID/name을 찾고, 없으면 nil입니다. 패턴을 find의 wildcard로 해석하지 않습니다. 두 번째 exact match를 발견하면 즉시 ErrAmbiguous로 멈추며 그 뒤 페이지를 요청하지 않습니다. 반면 nonnull filters인 {}, false, 0, 빈 문자열/목록도 deprecated search 분기를 선택합니다. 이 분기는 전체 목록을 먼저 읽으므로 중복 뒤 후속 page 오류가 중복 판정보다 먼저 발생할 수 있습니다. `Filtered`가 이 분기를 표시하며 Python warnings.warn side effect를 그대로 emit하지 않습니다.

Filtered get의 JMES 결과는 Source처럼 truthiness → len → index0 순서로 읽습니다. 전체 false/0/null/빈 array/object/string은 nil, 길이2 이상은 ErrAmbiguous, truthy scalar와 길이1 object의 정수 index0 부재는 오류입니다. 한 요소 array의 false/0/빈 string은 **존재하는 반환 값**이고 [null]만 nil입니다. 임의 expression 결과는 `Value`로 반환하며 실제 입력 행과 관계를 추측해 Group/HTTP receipt를 만들지 않습니다. 일반 dictionary 선택만 Group/Groups에 실제 owned record를 연결합니다.

Get의 Source None identity는 HTTP 없이 InvalidRequest이며 Go의 required string 인수로 None를 표현할 수 없는 입력 차이입니다. Go 빈 문자열을 None에 합치지 않습니다. 명시 빈 string의 default get은 `/os-server-groups/` GET을 실제로 수행하고, clean404일 때 `/os-server-groups` list fallback을 수행합니다. Fallback은 null ID/name를 빈 문자열로 오인하지 않고 literal empty string만 비교합니다. 예를 들어 default get의 literal `db0*`는 Go에서 `db0%2A`로 전송하지만 Python Requests는 `db0*`를 유지합니다. 두 경로 모두 기본 find에서 glob으로 검색하지 않습니다. 일반 ID는 한 segment로 escape하며 dot/traversal, control, whitespace-only ID는 HTTP 전에 거부하는 Go 경로 정책을 명시합니다.

## 버전·응답·오류

선택된 client.Microversion은 우선하며 2.100처럼 resource ceiling보다 높아도 Source session default로 유지합니다. 미지정이면 기존 Nova discovery를 재사용해 server minimum/maximum과 ServerGroup maximum2.64를 선택합니다. 이 helper의 member/list Source100..399와 strict physical discovery hook fences는 새 opt-in 경로이며 기존 FindRecord/native 정책을 바꾸지 않습니다. All-projects admin query를 기본으로 추가하지 않습니다.

Resource는 descriptor/입력 ID seed/location view, Wire는 실제 반환 JSON object, Envelope/Header/StatusCode는 실제 HTTP observation입니다. Source seed/default를 실제 Wire에 넣지 않습니다. Search Inventory에는 끝까지 소비한 실제 record를, Get Observed에는 실제 member GET proof를, default find Inventory에는 실제로 소비한 fallback rows를 보존합니다. 완료되지 않은 Value/Group/Groups는 오류 때 비웁니다. 클라우드 CurrentLocation은 명시 옵션으로 snapshot하며 foreign project 이름은 비우고 해당 raw project ID를 보존합니다. Physical Wire는 location을 덮지 않습니다.

Clean retry와 live token renewal은 고정 route/header/버전으로 유지합니다. Read/Close 오류, 사용자 cancellation cause와 transient source/API 변경은 retry/reauth/backoff를 호출하거나 missing으로 바꾸지 않습니다. Native401은 Unwrap 없는 ErrUnableToReauthenticate의 value ErrOriginal와 ErrReauth를 보존합니다. Callback이 OkCodes를 확대해 원래 rejected status를 수락해도 SDK의 원래 정책이 success/fallback을 허용하지 않습니다. 실제 failed body/header/status와 원인을 그대로 남깁니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--operation", choices=("search", "get"), default="search")
parser.add_argument("--name")
parser.add_argument("--filters", help="JSON dictionary, string expression or other Source value")
args = parser.parse_args()
kwargs = {}
if args.filters is not None:
    kwargs["filters"] = json.loads(args.filters)

def view(value):
    if hasattr(value, "to_dict"):
        return value.to_dict()
    if isinstance(value, list):
        return [view(item) for item in value]
    return value

with openstack.connect(cloud=args.cloud) as conn:
    if args.operation == "search":
        result = conn.search_server_groups(args.name, **kwargs)
    else:
        result = conn.get_server_group(args.name, **kwargs)
    print(json.dumps(view(result), indent=2))
```

`python main.py --cloud dev --operation search --name 'db0*' --filters '{"policy":"anti-affinity"}'`

`python main.py --cloud dev --operation get --name g1`

`python main.py --cloud dev --operation get --name g1 --filters '{}'`

## 독립 Go main

[설치 안내](../docs/install.md)에 따라 module을 준비하고 main.go로 저장합니다.

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
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	operation := flag.String("operation", "search", "search or get")
	name := flag.String("name", "", "name, literal ID or search glob")
	filters := flag.String("filters", "", "complete JSON filter; omitted means None")
	flag.Parse()
	if *operation != "search" && *operation != "get" {
		log.Fatal("unknown operation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	options := []compute.ServerGroupCloudReadOption{compute.WithServerGroupCloudCurrentLocation(location)}
	if *filters != "" {
		options = append(options, compute.WithServerGroupCloudFilters(json.RawMessage(*filters)))
	}
	var value json.RawMessage
	if *operation == "search" {
		result, cause := service.SearchServerGroups(ctx, *name, options...)
		err = cause
		if result != nil {
			value = result.Value
		}
	} else {
		result, cause := service.GetServerGroup(ctx, *name, options...)
		err = cause
		if result != nil {
			value = result.Value
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	if value == nil {
		value = json.RawMessage("null")
	}
	data, err := json.MarshalIndent(json.RawMessage(value), "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

`go run . -cloud dev -operation search -name 'db0*' -filters '{"policy":"anti-affinity"}'`

`go run . -cloud dev -operation get -name g1`

`go run . -cloud dev -operation get -name g1 -filters '{}'`

`-filters '"[].id"'`처럼 JSON string으로 expression도 지정할 수 있습니다. 이 예제는 실제 read/discovery를 수행합니다. 이미 구성된 client에서는 `compute.New(client, compute.Dependencies{})`를 사용하고 standalone location 기본은 null입니다. `servergroups.API.ListCloudRecords`와 `FindCloudRecord`는 두 cloud helper를 위한 좁은 SDK-owned leaf이며 general proxy/list/find 전체의 완료로 승격하지 않습니다.

## 정확히 남은 Source 차이

1. **JMESPath가 Python ServerGroup class를 보는 경계.** Source `type([0])`는 "object"이지만 `keys([0])`, `values([0])`, `length([0])`는 JMESPathTypeError(expected object, received unknown/ServerGroup)입니다. Go의 JSON map에는 이 subclass validator가 없으므로 해당 함수가 성공할 수 있습니다. Source Resource.__eq__는 plain object에 false이고, 동일 canonical10field object와 비교한 `[?@ == literal].id`는 Source[]/getNone인데 Go structural map equality는 같다고 볼 수 있습니다. Nested rules/metadata는 일반 dict라 별개입니다. 일반 resource marker/equality/function namespace를 아직 구현하지 않았습니다.
2. **전체 Python numeric/arithmetic/serialization/nonfinite namespace.** 새 SelectSource는 finite float loading과 exact binary operands로 큰 integer/float 비교를 보완합니다. 예를 들어 9007199254740992.0과 float literal9007199254740993.0은 Source처럼 같고, float1e23과 integer100000000000000000000000은 다릅니다. 그러나 Source ``sum(`[9007199254740992.0,1]`)``는9007199254740992.0이고 ``to_string(`1e23`)``는"1e+23"이며 기존 exact-number arithmetic/serialization 전체가 같다고 주장하지 않습니다. Nonfinite literal1e400은 Source search에서 inf를 반환하고 get에서는 TypeError지만 Go Source-loading JSON 경로는 unsupported 오류입니다. 이 reachable named 경계를 SDK-R1 일반 입력 차이로 감추지 않습니다.

위 두 축 때문에 **search_server_groups/get_server_group 두 operation의 review status는 unresolved, full supported 증가분은0**입니다. 기본 흐름은 구현됐지만 known JMES namespace 차이가 해결될 때까지 Source 전체 동등성을 완료로 표시하지 않습니다. 별도로 mutable Resource/class/adapter/cache/session/current_location/custom Python runtime 객체는 공통 SDK-R1/SDK-C1/SDK-S1 차이로 추적하며 이번 helper가 cache 전체를 완성했다고 주장하지 않습니다.

[고정 Nova32 서버 그룹 정책](https://github.com/openstack/nova/blob/32.0.0/nova/policies/server_groups.py#L49-L85)은 index/show에 project scope PROJECT_READER_OR_ADMIN을, index:all_projects에 ADMIN을 요구합니다. 이 cloud helper는 all_projects=False 기본을 유지합니다. 배포 정책·소유권·역할·서버 실제 수락을 mock/runtime 증거로 보장하지 않습니다.

실제 pinned cloud/proxy/resource/filter/JMESPath를 Python3.14.8에서 core114 + boundary25 + arithmetic8 =147회 실행했습니다. Adapter HTTP/endpoint-data/connection만 fixture로 대체했으며 live cloud 증거는 아닙니다. Go cloud10 + leaf5 =15그룹은 Gophercloud public testhelper-backed internal/testcloud와 기존 Nova transport/body observers를 재사용합니다. Source receipt는 `/private/tmp/go-openstacksdk-cycle8-server-group-reads-source.json`, `...-boundaries.json`, `...-arithmetic.json`입니다. 최종 Go·race·standalone·전체 gate는 root의 실제 실행 결과로 기록하고 미실행 PASS를 주장하지 않습니다.

Source: [search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L216-L242), [get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L728-L772), [filter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145), [ServerGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_group.py), [Resource.find/equality](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py).
