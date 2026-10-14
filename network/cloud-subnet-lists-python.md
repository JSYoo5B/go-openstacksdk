# Cloud 서브넷 목록: Python과 Go

`conn.ListSubnetRecords`와 `conn.SearchSubnetRecords`는 Source cloud 목록·검색의 일반 호출을 제공합니다. 옵션·Source 속성·pagination·현재 프로젝트/zone·실제 HTTP 증거는 SDK가 소유합니다. Proxy의 임의 JMESPath 반환값도 별도로 표현하므로 애플리케이션 builder/interface는 필요 없습니다.

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. 이번 세 inventory 행은 `unresolved`를 유지합니다. 아래 구현이 추가됐지만 Source의 동적 `max_items`와 정확한 cap 경계의 추가 GET, nonfinite microversion까지 완전히 같지는 않습니다. 전체 SDK `supported` 증가량은 0입니다.

| Python | Go | 실행 의미 |
| --- | --- | --- |
| `conn.list_subnets(filters={...})` | `conn.ListSubnetRecords(ctx, network.WithSubnetReadFiltersJSON(raw))` | network 서비스가 없으면 빈 목록 |
| `conn.search_subnets(name, filters={...})` | `conn.SearchSubnetRecords(ctx, name, options...)` | name을 서버 query로 넣은 뒤 filters가 덮음 |
| `conn.network.subnets(**query)` | `service.API.Subnets.ProxyListRecords(ctx, subnets.WithRecordListQuery(key, value))` | opaque Resource generator 또는 임의 JMESPath 값 |
| 일반 `list(conn.network.subnets(...))` | `service.API.Subnets.AllRecords(ctx, options...)` | Resource-only convenience, 실패 시 partial slice 반환 안 함 |
| generator 직접 소비 | `service.API.Subnets.ListRecords(ctx, options...)` | Go iterator; 이미 yield한 행은 보존 |
| `headers`, `microversion`, `base_path` | `WithSubnetReadHeaders`, `WithSubnetReadMicroversion`, `WithSubnetReadBasePath` | concrete transport option 또는 filters의 raw formal |
| `paginated`, `max_items`, `limit`, `marker` | leaf `WithRecordListPaginated/MaxItems/Limit/Marker` | 아래 Source 차이 참조 |

`service, err := conn.Network(ctx)` 뒤 같은 Service 메서드를 직접 사용할 수 있습니다. 이미 선택한 서비스에서는 필요한 network endpoint를 호출자가 공급한 Gophercloud client에서 SDK가 복제합니다. Connection은 current location과 clean network catalog absence를 소유합니다. 기존 native `List`·`FindIdentity`·`Resources.List`·`FindRecord` 모델과 정책은 바꾸지 않습니다.

## 검색과 입력의 정확한 의미

검색은 `_filter_list`를 호출하지 않습니다. UUID, `app*`, `?`도 literal `name` 서버 query이며 local ID/name/glob 선택을 추가하지 않습니다. truthy 이름부터 query를 만들고 truthy filters를 `dict.update`하므로 filters.name이 이름을 덮습니다. 검색 filters는 JSON object뿐 아니라 `[["name","app"],["is_dhcp_enabled",false]]` 같은 key/value 쌍 목록도 받습니다. 목록의 truthy filters는 `**mapping`이므로 같은 쌍 목록을 거부합니다. None/null·false·0·빈 string/list/object는 빈 filters입니다.

`ListSubnetRecords`만 network available=false를 filters/JMESPath 검증보다 먼저 빈 목록으로 처리합니다. `SearchSubnetRecords`는 그 flag를 무시하고 실제 선택한 Proxy가 필요합니다. 취소·nil 옵션·잘못된 서비스 설정은 Go preflight 오류입니다. clean catalog absence 이외 endpoint/provider/인증 오류를 빈 목록으로 숨기지 않습니다.

선언된 Query 이름·wire 별칭을 서버로 보내고, `prefix_length`, `revision_number`, `tenant_id` 같은 non-query Body 속성은 received getter에 local 비교합니다. unknown query kwargs는 marshaling 전에 버립니다. bool/string/list/int descriptor 기본값과 canonical/wire 입력 순서, loaded JSON bool/number equality를 적용하며 임의 unknown 응답 Body와 큰 정수 token은 보존합니다. page `If-Match`는 행의 getter를 덮지 않고, 행의 Header 속성만 constructor에서 반영합니다.

`WithSubnetReadFiltersJSON`·bulk 옵션은 bytes/maps/location을 복제하며 뒤 concrete 옵션이 같은 transport 설정을 덮습니다. `WithSubnetReadHeaders(nil)`은 filters.headers를 지웁니다. bulk nil CurrentLocation은 Connection이 제공한 기본을 유지합니다. raw source JSON object/ordered individual options는 순서를 유지하고 map-only 순서는 lexical입니다.

## Proxy 지연 실행과 JMESPath

`ProxyListRecords`는 페이지를 먼저 수집하지 않습니다. `[0]`, `[*].id`, filter projection, `type(@)`는 실제 generator를 JSON array로 간주하지 않아 null을 반환하며 HTTP 0입니다. `@`는 generator, `to_array(@)`는 같은 generator를 담은 한 원소 목록, `length(@)`는 HTTP 전에 오류입니다. literal JMESPath는 임의 JSON을 반환할 수 있습니다.

Proxy bound self/resource_type/cls/session 및 truthy nonmapping `__conflicting_attrs`는 즉시 오류입니다. Source conflict object를 한 번 복원하고 다시 주입된 conflict object는 재귀 복원하지 않습니다. restored base_path/paginated/cls/session도 binding 오류지만 restored self/resource_type는 Resource.list unknown으로 무시됩니다. restored jmespath_filters는 원래 Proxy expression을 바꾸지 않습니다.

headers/base_path/microversion 등의 Resource controls는 generator 소비시에만 검증합니다. 따라서 `{"headers":"bad","base_path":false,"jmespath_filters":"`[]`"}`는 빈 literal 목록으로 HTTP 0 성공하고, expression을 빼면 소비시 오류입니다. 오직 base_path 빈 문자열만 빈 generator이고 null은 기본 `/subnets`, false/0/list/object는 소비시 오류입니다. 빈 generator도 나중에 소비하는 시점의 context/source guard를 확인합니다.

cloud의 `list()` 변환은 Resource generator를 끝까지 소비하거나, JSON list/string의 Unicode 문자/object의 key를 목록으로 바꿉니다. scalar/null은 오류입니다. Go map key 결과는 lexical이며 Python의 insertion order와 다릅니다. `to_array(@)`의 nested generator는 소비하지 않은 채 `Value`에 남고 `JSON`은 nil입니다. 이후 nested `Records`를 소비할 때도 캡처한 source/location/context와 실제 응답 오류 증거를 사용합니다. HTTP 전 오류에는 응답을 만들지 않습니다.

## 반환값·페이지·현재 위치

`Subnets`는 실제 Resource generator에서 온 record만, `Value`는 실제 cloud `list()` 결과입니다. `Resource`에는 descriptor/default와 SDK computed location, `Wire`·`Envelope`·`Header`·`StatusCode`에는 실제 HTTP 자료가 있습니다. JSON literal을 Subnet으로 만들거나 seed를 Wire로 꾸미지 않습니다. Resource/Wire/Inventory/LastResponse 자료는 서로 독립 복제입니다.

완료된 목록만 Subnets/Value/JSON에 commit합니다. 후반 page/descriptor/physical 오류에서는 이전 실제 행이 `Inventory`에 남으며 `LastResponse`는 마지막 관찰한 resource 또는 오류 receipt입니다. 정상 empty final page나 모든 filter-discarded page의 최종 receipt를 의미하지 않습니다. accepted decode/read/Close/취소/source 오류는 raw body·status·header와 Cause를 `ResponseError`로 유지합니다. clean native 거절과 native401 ErrOriginal도 실제 body/header/status를 유지하며 physical faults를 RetryFunc/backoff/reauth로 replay하지 않습니다. clean retry/token renewal은 유지합니다.

Source `links`가 있으면 plural links보다 우선하며 dict형 links.next는 무시합니다. Source root next·plural/list Link·HTTP Link·limit/marker fallback을 기존 guarded pager로 처리합니다. local match 결과 수로 다음 marker를 추정하지 않고 raw 행을 사용합니다. cycle/외부 origin·route/traversal continuation은 Go가 거부합니다.

Connection은 current cloud/region/project/auth 이름을 computed location으로 공급합니다. falsey 또는 현재 project와 같은 project ID는 current scope, truthy foreign ID는 raw ID와 null project names를 유지합니다. 응답 unknown `availability_zone`의 null/false/0/list/object도 Source처럼 zone에 남습니다. standalone service에서 CurrentLocation을 생략하면 location은 null이며 응답 forged location은 Wire에서만 보존합니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=["list", "search"], default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "list":
        values = conn.list_subnets(filters=filters)
    else:
        values = conn.search_subnets(args.name, filters=filters)
    print(json.dumps([
        value.to_dict() if hasattr(value, "to_dict") else value
        for value in values
    ], indent=2))
```

## 독립 Go main

[설치 안내](../docs/install.md)의 module/clouds.yaml 준비 후 main.go로 저장합니다. 두 main은 목록 GET만 수행합니다. `to_array(@)`처럼 opaque nested generator를 반환하는 옵션은 두 main의 JSON serialization에서 오류가 납니다. 이는 generator 소비나 임의 Subnet 변환을 추가하는 sample이 아닙니다.

```go
package main

import (
 "context"
 "encoding/json"
 "flag"
 "fmt"
 "os"
 "time"

 sdk "github.com/JSYoo5B/go-openstacksdk"
 "github.com/JSYoo5B/go-openstacksdk/network"
)

func run(ctx context.Context, cloud, mode, name, filters string) error {
 conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
 if err != nil { return err }
 options := []network.SubnetReadOption{network.WithSubnetReadFiltersJSON(json.RawMessage(filters))}
 var result *network.SubnetRecords
 switch mode {
 case "list": result, err = conn.ListSubnetRecords(ctx, options...)
 case "search": result, err = conn.SearchSubnetRecords(ctx, name, options...)
 default: return fmt.Errorf("mode must be list or search")
 }
 if result != nil && result.LastResponse != nil {
  fmt.Fprintf(os.Stderr, "last observed resource/error status=%d request-id=%q partial-resources=%d\n",
   result.LastResponse.StatusCode, result.LastResponse.Header.Get("X-Request-Id"), len(result.Inventory))
 }
 if err != nil { return err }
 encoded, err := json.MarshalIndent(result.Value, "", "  ")
 if err != nil { return err }
 _, err = fmt.Fprintln(os.Stdout, string(encoded))
 return err
}

func main() {
 cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
 mode := flag.String("mode", "list", "list or search")
 name := flag.String("name", "", "literal search name, including UUID or glob characters")
 filters := flag.String("filters", `{}`, "Source cloud filter JSON")
 flag.Parse()
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 if err := run(ctx, *cloud, *mode, *name, *filters); err != nil {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(1)
 }
}
```

`python main.py --cloud dev --mode list --filters '{"network_id":"network-id","is_dhcp_enabled":false}'`

`go run . --cloud dev --mode list --filters '{"network_id":"network-id","is_dhcp_enabled":false}'`

`python main.py --cloud dev --mode search --name app --filters '{"name":"overridden","paginated":false}'`

`go run . --cloud dev --mode search --name app --filters '{"name":"overridden","paginated":false}'`

## 남은 차이와 검증 경계

현재 pager의 `max_items`는 nonnegative Go int입니다. Source의 true/float/negative/string 값과 오류 발생 시점은 남은 named gap입니다. Source는 cap을 다음 raw row 직전에 검사하므로 page raw 수가 cap과 정확히 같으면 다음 GET이 발생하지만 Go pager는 cap에 도달하면 그 page에서 멈춥니다. limit의 동적 numeric/string 비교도 Go controls와 다르며 이 계약이 닫힐 때까지 Proxy/cloud 세 행은 unresolved입니다. 기존 shared pager를 이번 작업에서 확대하지 않습니다.

call microversion은 None/default와 non-None falsey disable을 구분하고, ordinary numeric/string/iterable JSON은 KSA의 version normalization을 재사용한 규칙으로 변환합니다. `true`·1→1.0, [2,4]→2.4, v2.4→2.4, latest→latest, major.latest는 오류입니다. Python float overflow `1e400`·[1e400]가 KSA에서 latest가 되는 nonfinite domain은 Go에서 지원하지 않습니다. 또 Source의 빈 base_path는 HTTP/KSA version 검증을 건너뛰어 잘못된 microversion도 빈 목록이지만, 현재 leaf는 version을 먼저 normalize하여 오류가 되는 timing gap이 남습니다. invalid headers는 Source도 while 전에 검사하므로 이 차이와 구분합니다. Python float repr/runtime objects, arbitrary class/URI `%` formatting·mutable Resource/Container/descriptor dispatch, custom session/cache negotiation 및 SDK-R1·C1·S1 기반은 별도 remaining입니다. Go의 검증된 same-origin service-relative base_path와 immutable source/headers는 명시적인 SDK 정책입니다.

[Neutron 27 subnet 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnet.py#L121-L176)의 `get_subnet`은 project scope의 PROJECT_READER/shared/external_network/ADMIN_OR_NET_OWNER_READER/SERVICE 조합입니다. `segment_id`는 별도 ADMIN 조건이며 tags에도 별도 read 정책이 있습니다. 읽기 권한이 모든 프로젝트 행이나 숨겨진 field 노출을 보장하지 않고 배포 policy가 최종 결과를 결정합니다. SDK는 권한을 확대하지 않습니다.

독립 actual pinned Source에서 cloud list/search·Proxy._list·Resource.list/constructor, location, argument restoration과 controls를 실행한 receipt를 재사용합니다. 모든 Network/Subnet 합계 696 observations(114+72+92+22+48+348), 그 중 Subnet 348 observations이며 서로 다른 fixture 영역입니다. 기본 list/control/limit은 Adapter.get/has_service fixture, location은 config/auth fixture, microversion22는 실제 KSA Adapter/Session과 physical HTTP fixture입니다. overflow edge는 실제 KSA header normalizer만 실행했습니다. 이 자료는 live cloud·실제 인증/ACL·배포 수락 증거가 아닙니다.

회귀는 기존 public Gophercloud testhelper 기반 testcloud/nativeSubnet/body observers를 재사용한 cloud8+leaf11, 총19그룹입니다. Go/race 및 exact main build 결과는 root 실행 ledger로 별도 확인하며 AST/compile을 runtime authorization이나 전체 Source parity로 부르지 않습니다.
