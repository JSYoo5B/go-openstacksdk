# Port 목록과 검색: Python과 Go

고정 기준은 openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py), Gophercloud v2.15.0입니다. 기존 native `List`, `All`, `Port`와 별도로 SDK가 descriptor 기본값·확장 속성·local filter·kwargs·페이지를 소유합니다.

| Python | Go | 반환 |
|---|---|---|
| `conn.network.ports(**query)` | `ListRecords(ctx, options...)` | lazy `PortRecord` iterator |
| `list(conn.network.ports(**query))` | `AllRecords(ctx, options...)` | 성공한 빈 결과도 nonnil slice, 오류 시 partial slice 폐기 |
| `conn.network.ports(jmespath_filters=..., **query)` | `ProxyListRecords(ctx, WithRecordListQuery("jmespath_filters", ...), ...)` | Source generator 기반의 값·null·opaque generator |
| `jmespath.search(expr, list(...))` | `ListRecordValues(ctx, expr, options...)` | 실제 목록을 수집한 임의 JSON 값과 Inventory |
| cloud `search_ports(name_or_id, filters)`의 목록·선택 단계 | `SearchRecords(ctx, nameOrID, WithRecordSearchFilters(raw))` | 선택 JSON과 동일 순서의 records, 전체 Inventory |

`PortRecord.Resource.Body`는 Python 속성 view입니다. `Wire.Body`에는 실제 row spelling·unknown extension·수치 원문이 남고, `Envelope`, `Header`, `StatusCode`는 실제 페이지 응답입니다. bindings는 `binding_host_id/profile/vif_details/vif_type/vnic_type`, DNS는 `dns_assignment/domain/name`, 보안은 `is_port_security_enabled/security_group_ids`, QoS는 `qos_policy_id/qos_network_policy_id`로 읽습니다. 누락된 `is_port_security_enabled`는 false, tags는 [], 나머지 declared Body는 null이며 explicit null은 유지합니다. `revision_number`는 Source 정수 변환, list와 dict descriptor는 Source 단일값 wrapping·dict 기본값, boolean은 CPython JSON float truthiness를 따릅니다. 실제 HTTP If-Match는 LIST Resource 속성의 seed가 아니므로 일반 목록 row의 if_match는 null입니다.

선언 query의 client·wire spelling을 받아 Requests 값으로 인코딩합니다. `is_admin_state_up`→`admin_state_up`, `tenant_id`→`project_id`, `security_group_ids`→`security_groups`, tag aliases도 지원합니다. 배열은 반복 query, boolean은 True/False, null은 생략하며 canonical alias와 선언 순서가 충돌 우선순위를 결정합니다. `fields`는 서버 query이며 누락된 응답 필드를 query 값으로 만들어 넣지 않습니다. `binding:host_id`는 server query지만 `binding_host_id`는 declared nonquery Body이므로 local filter입니다. binding_profile·dns_name도 local filter이고 dict는 재귀 부분 일치입니다. server filter를 SDK가 다시 local 검사하지 않습니다. unknown·all_projects kwargs는 값을 marshal하기 전에 버립니다.

`headers`, `microversion`, `base_path`는 실제 Source formal 인자로 분류하며 `WithRecordListHeaders/Microversion/BasePath` concrete factory도 같은 정책을 제공합니다. 순차 옵션은 마지막 값이 이깁니다. base_path는 캡처한 service endpoint 내부 상대 경로이고 외부 host·query·fragment·점 경로 이동은 HTTP 전에 거부합니다. 다음 페이지도 선택한 collection 경로에 고정합니다. 빈 경로·null은 기본 route입니다.

`__conflicting_attrs`는 truthy JSON object를 한 번만 복원하고 기존 key 위치·신규 key 삽입 순서를 유지합니다. 복원한 paginated/base_path는 Source의 formal 인자 중복 오류이고, 복원한 jmespath_filters는 원래 Proxy 표현식을 바꾸지 않습니다. cls/session/self/resource_type 충돌은 HTTP 전에 오류입니다. location·connection·_synchronized·requires_id·error_message·skip_cache·resource_response_key·list_base_path 같은 Source non-Body unknown 인자는 값의 marshaling 전에 버립니다.

옵션은 Query·Headers·Microversion·BasePath·callback 결과를 복사합니다. nil Headers도 callback이 직접 key를 쓸 수 있고, callback의 취소와 선택 client 변경은 option error와 함께 보존해 HTTP 전에 종료합니다. 인증·framing·다른 서비스 version 제어와 runtime Resource class 변경은 허용하지 않습니다. 잘못된 JSON·UTF-8·음수 cap·nil option은 오류입니다.

`paginated` 기본값은 true, `max_items=0`은 무제한이며 limit은 기본 생략합니다. cap은 local filter로 제외된 raw row도 세고, truthy cap과 falsey limit은 첫 server limit hint를 제공합니다. body/plural links·top-level next·HTTP Link·limit/marker fallback을 공통 pager로 처리하며 empty page에서 종료합니다. underflow float next는 falsey입니다. consumer break·cap·single page 후에는 다음 row/next를 평가하지 않습니다. continuation은 원래 origin·collection·filter를 유지하고 반복/위험한 route는 실제 page 오류로 반환합니다. Go의 dictionary/RFC Link 처리와 route 검증은 Source와 다른 호환성·보호 정책입니다.

## Generator JMESPath와 eager 결과

[Source `Proxy._list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)는 `Resource.list`의 아직 읽지 않은 generator를 직접 JMESPath에 전달합니다. 비교 runtime은 Python jmespath 1.0.1이며 openstacksdk의 추가 pin은 아닙니다.

| `jmespath_filters` | Source와 `ProxyListRecords` | HTTP |
|---|---|---|
| 생략·falsey·string이 아닌 값 | lazy generator 유지 | 반환 시 0, iteration 시 요청 |
| `[*].id`, `[0]`, `[?status=='ACTIVE']`, `type(@)` | JSON null | 0 |
| `length(@)`, `sort_by(@,&id)`, `contains(@,'x')`, `to_string(@)` | opaque generator 타입/직렬화 오류 | 0 |
| `@`, `not_null(@,\`[]\`)` | 동일 generator 반환 | 반환 시 0 |
| `to_array(@)` | generator를 포함한 배열 | 반환 시 0 |
| `@ == @`, `[ @ ] == [ @ ]` | true | 0 |
| JSON literal expression | 해당 JSON 값 | 0 |

`RecordProxyListResult.Value`는 JSON 값이나 `*RecordListGenerator`, 또는 그 pointer를 포함한 배열·객체입니다. 전체 값이 JSON일 때만 `JSON`을 채웁니다. opaque generator에는 `Records` iterator가 있고, JSON marshaling은 오류입니다. 같은 generator pointer의 identity는 중첩 equality에서도 유지합니다. `ListRecords`는 resource-only 타입이므로 truthy expression은 `ProxyListRecords`로 안내하고, falsey/nonstring 값은 원래 lazy list를 제공합니다.

실제 port 행에서 이름을 추출하거나 count·object·scalar 결과를 얻으려면 `ListRecordValues(ctx, expression, ...)`를 사용합니다. 이 API는 전체 owned view를 먼저 읽어 full JMESPath를 평가하는 명시적 Go 개선입니다. `Value`는 임의 JSON이고 `Inventory`에는 원래 records가 남습니다. expression 오류나 뒤 페이지 오류에도 읽은 Inventory가 보존되며 성공한 값으로 꾸미지 않습니다. cloud `search_ports`도 원본부터 eager 목록에 표현식을 적용합니다.

`SearchRecords`는 dict 필터를 raw insertion 순서대로 list kwargs로 분류하고, 문자열은 query로 내리지 않습니다. 그다음 name/ID의 exact·fnmatch glob과 nested dict 또는 full JMESPath를 Python 속성 view에 적용합니다. unknown dict key는 실제 목록을 읽은 뒤 오류이고, Source Resource의 builtin dict.get/membership를 따릅니다. canonical binding_profile·security_group_ids·is_admin_state_up으로 local 검색해야 하며 wire binding:profile·security_groups·admin_state_up은 server query로 전달된 뒤 cloud filter 단계에서 AttributeError에 대응하는 오류입니다. Source의 직접 resource[wire] fallback과 이 dict 검색을 혼동하지 않습니다. dict/string이 아닌 truthy 값은 Python **kwargs가 불가능하므로 HTTP 전에 오류입니다. Source falsey float underflow도 필터 없음으로 처리합니다. expression 결과에 Port pointer를 합성하지 않으며 `Ports`는 nil입니다. Dict의 Proxy jmespath_filters도 무시하지 않습니다. null/타입 오류는 HTTP0에서 반환하고, @이면 실제 목록을 읽은 뒤 원래 dict 필터를 평가합니다. JSON literal 목록에는 응답 증거나 Port pointer를 합성하지 않습니다.

`WithRecordSearchOpts`는 완전한 concrete 옵션을 복사합니다. Connection adapter가 `WithRecordSearchCloudLocation(currentLocation)`를 주면 실제 행마다 project_id·availability_zone으로 location을 계산해 local 선택 전에 반영합니다. foreign project는 현재 프로젝트의 name/domain을 지우고 실제 ID·zone을 유지하며, Wire에는 계산값을 넣지 않습니다. `WithRecordSearchLocation(raw)`는 명시 raw view override입니다. SDK가 계산을 소유하므로 호출자가 per-record builder를 구현할 필요가 없습니다.

## 실행 예제

다음 Python 예제는 실제 인증 환경에서 실행하는 독립 코드입니다.

```python
import argparse
import json
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--network", default="")
    parser.add_argument("--host", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {}
    if args.network:
        query["network_id"] = args.network
    if args.host:
        query["binding_host_id"] = args.host
    for port in conn.network.ports(**query):
        print(json.dumps(port.to_dict(), ensure_ascii=False))


if __name__ == "__main__":
    main()
```

같은 Go main은 `--mode=list`, `proxy`, `eager`, `search`를 선택할 수 있습니다. `--expression='[*].id'`를 proxy에 주면 HTTP 없이 null이며, eager에 주면 실제 ID 배열입니다. `--filter`는 JSON object나 JSON string입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    networkID := flag.String("network", "", "server network_id filter")
    host := flag.String("host", "", "local binding_host_id filter")
    mode := flag.String("mode", "list", "list, proxy, eager or search")
    expression := flag.String("expression", "[*].id", "JMESPath expression")
    name := flag.String("name", "", "search name, ID or glob")
    filters := flag.String("filter", "", "search raw JSON filters")
    flag.Parse()
    if err := run(*cloud, *networkID, *host, *mode, *expression, *name, *filters); err != nil { log.Fatal(err) }
}

func run(cloud, networkID, host, mode, expression, name, filters string) error {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    api := network.Ports
    var options []ports.RecordListOption
    if networkID != "" { options = append(options, ports.WithRecordListQuery("network_id", networkID)) }
    if host != "" { options = append(options, ports.WithRecordListQuery("binding_host_id", host)) }
    switch mode {
    case "list":
        for row, err := range api.ListRecords(ctx, options...) {
            if err != nil { return err }
            if err := printPort(row); err != nil { return err }
        }
    case "proxy":
        options = append(options, ports.WithRecordListQuery("jmespath_filters", expression))
        result, err := api.ProxyListRecords(ctx, options...)
        if err != nil { return err }
        if generator, ok := result.Value.(*ports.RecordListGenerator); ok {
            for row, err := range generator.Records {
                if err != nil { return err }
                if err := printPort(row); err != nil { return err }
            }
        } else if result.JSON != nil {
            fmt.Println(string(result.JSON))
        } else {
            fmt.Printf("opaque result: %T\n", result.Value)
        }
    case "eager":
        result, err := api.ListRecordValues(ctx, expression, options...)
        if err != nil { return err }
        fmt.Println(string(result.Value))
    case "search":
        var search []ports.RecordSearchOption
        if filters != "" { search = append(search, ports.WithRecordSearchFilters(json.RawMessage(filters))) }
        result, err := api.SearchRecords(ctx, name, search...)
        if err != nil { return err }
        fmt.Println(string(result.Value))
    default:
        return fmt.Errorf("unknown mode %q", mode)
    }
    return nil
}

func printPort(row *ports.PortRecord) error {
    data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}
```

## 오류·권한·검증 범위

shared Engine은 실제 representation200..399를 허용합니다. clean 오류는 native method·URL·status·body·Header를 유지하며, list404를 missing 성공으로 바꾸지 않습니다. malformed row·projection·local filter·unsafe next 오류에도 실제 페이지 증거를 보존합니다. 물리 Read/Close 오류는 body를 한 번 닫고 RetryFunc·reauth 전에 종료합니다. mixed401의 native `*gophercloud.ErrUnableToReauthenticate`는 `ErrOriginal`에 실제401 receipt, `ErrReauth`에 원인을 보존하며 이 두 필드를 별도로 검사해야 합니다. clean retry·token renewal은 유지하되 선택한 인증/route/version/조건 header와 status 정책은 바뀌지 않습니다. 오류의 Operation은 실제 entry API, Resource는 ports입니다.

[Neutron 27.0.0 get_port 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py#L347)은 project scope에서 admin/service·network owner reader·project reader를 허용합니다. binding host/profile/vif type/details는 admin/service, resource_request와 trusted는 admin 기본 분기입니다. SDK가 해당 필드 접근 권한을 추정하지 않고, 서버가 준 row와 누락 상태를 보존합니다. 전체 cloud 접근이나 확장 지원은 fixture의 성공으로 입증되지 않습니다.

Source Port에는 최대 microversion이 없으므로 기본 discovery·ceiling을 추가하지 않습니다. 기존 client 또는 명시 version을 보존합니다. SDK는 변경 가능한 Python Resource·connection location·cache/dirty state·runtime class/session을 완전 구현한 상태가 아니며 shared SDK-R1/C1/S1에서 추적합니다. Resource의 location은 owned null이고 server 값은 Wire에만 남깁니다. cloud list_ports/search_ports의 서비스 없음→빈 결과는 Connection adapter 통합으로 별도로 닫아야 합니다. location은 앞의 concrete 검색 옵션과 공통 CloudLocation을 통해 그 통합에 연결합니다. 목록 local 비교와 cloud 두 번째 dict 비교는 opt-in CPython JSON-loaded equality를 사용해 float rounding·arbitrary integer·bool·중첩 배열을 보존합니다. 기존 shared exact-decimal 비교는 변경하지 않습니다. full JMESPath에는 exact-decimal 수치·lexical object traversal·엄격한 문법이라는 SDK-C1 정책 차이가 남습니다. Python Port는 dict subclass이므로 keys(Port) 같은 함수가 Python의 exact typename 검사에서 오류인데 Go JSON view는 object 함수 결과를 제공할 수 있습니다. 이를 없는 runtime 탓으로 숨기거나 완전 동일한 mutable Resource로 주장하지 않습니다.

현재 leaf 테스트 27그룹은 기존 port recorder·fault body와 public Gophercloud testhelper 기반 internal/testcloud를 재사용합니다. 새 HTTP server framework를 추가하지 않습니다. Source generator 참고값은 Python jmespath1.0.1에서 확인했으며, root의 focused/full race와 독립 Go main 빌드 결과는 별도 통합 evidence입니다. 실제 cloud 호출·Python 인증 실행·릴리즈 완료를 주장하지 않습니다.
