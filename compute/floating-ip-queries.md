# Floating IP 목록·검색·단일 조회

`Connection`은 cloud 설정에 맞는 Neutron 또는 legacy Nova endpoint를 연결하고 floating IP 조회 결과와 실제 HTTP 응답을 함께 반환합니다. 애플리케이션에서 builder나 resolver를 구현할 필요가 없습니다. 목록·검색·조회는 IP를 할당하거나 서버에 연결하지 않습니다. free IP를 재사용하거나 새로 할당하려면 [Available](floating-ip-available.md), 서버에 연결하려면 [IP helper](server-ip-helpers.md)를 사용합니다.

같은 메서드를 `*compute.Service`에서도 사용할 수 있습니다. `conn.Compute(ctx)`로 서비스를 먼저 얻으면 Compute endpoint가 필요합니다. Connection의 조회 메서드는 Neutron 경로에서 Compute를 먼저 발견하지 않으므로 Neutron만 제공하는 cloud에도 적합합니다.

## Python과 Go 대응

| openstacksdk cloud | gophercloudsdk Service·Connection |
|---|---|
| `conn.list_floating_ips(filters=None)` | `ListFloatingIPs(ctx, options...)` → `*compute.FloatingIPQueryResult` |
| `conn.search_floating_ips(id=None, filters=None)` | `SearchFloatingIPs(ctx, compute.SearchFloatingIPsRequest{ID: ...}, options...)` → 같은 결과 타입 |
| `conn.get_floating_ip(id, filters=None)` | `GetFloatingIP(ctx, compute.GetFloatingIPRequest{ID: ...}, options...)` → `*compute.GetFloatingIPResult` |
| `conn.get_floating_ip_by_id(id)` | `GetFloatingIPByID(ctx, compute.GetFloatingIPByIDRequest{ID: ...}, options...)` → 같은 단일 결과 타입 |
| `conn.list_floating_ip_pools()` | `ListFloatingIPPools(ctx, options...)` → `*compute.FloatingIPPoolQueryResult` |
| `conn.search_floating_ip_pools(name=None, filters=None)` | `SearchFloatingIPPools(ctx, compute.SearchFloatingIPPoolsRequest{Name: ...}, options...)` → 같은 pool 결과 타입 |

모든 호출은 결과 포인터와 `error`를 함께 반환합니다. 오류가 있어도 이미 받은 page/member 응답이 결과에 남을 수 있으므로 결과부터 확인할 수 있습니다.

## 공통 concrete 옵션

| 옵션 | 의미 |
|---|---|
| `compute.WithFloatingIPQuerySource(source)` | per-call source. `FloatingIPNeutron`, `FloatingIPNova`, `FloatingIPNone` 사용 |
| `compute.WithFloatingIPQueryFilters(json.RawMessage(...))` | dictionary 또는 로컬 검색에서 소비할 JSON filter 값. 생략과 `{}`는 경로가 다를 수 있음 |
| `compute.WithFloatingIPQueryExpression(expression)` | 문자열 JSON filter를 준비하는 편의 함수. Search에서 JMESPath 적용 |
| `compute.WithFloatingIPQueryDirectGet(true)` | 일반 Get의 UUID형 입력에서 직접 GET을 사용. 기본 false |
| `compute.WithFloatingIPQueryStrict(true)` | Nova 정규화의 호환 alias·extra top-level 복원 제외. 기본 false |
| `compute.WithFloatingIPQueryLocation(location)` | 정규화에 쓸 `resource.CloudLocation` 전체 facts override |
| `compute.WithFloatingIPQueryTimeout(time.Minute)` | 옵션 준비 후 endpoint 발견·HTTP·fallback·변환·검색 전체 deadline |
| `compute.WithUnlimitedFloatingIPQueryTimeout()` | SDK query deadline 해제. 부모 context·transport 제한은 유지 |
| `compute.WithFloatingIPQueryOptions(compute.FloatingIPQueryOpts{...})` | 전체 옵션값을 owned snapshot으로 교체 |

기본 SDK deadline은 없습니다. Connection은 cloud/auth에 기록된 location과 기존 server-address source 설정을 사용합니다. `WithCloudLocation`은 Connection의 기록된 location facts를 지정할 때, `WithFloatingIPQueryLocation`은 한 조회에 적용할 때 사용합니다. 알 수 없는 cloud/project/name/domain 정보는 null로 남습니다.

JSON bytes·source pointer·location 값은 옵션 생성 및 준비 시 복사합니다. 일반 조회의 옵션은 순서대로 한 번 적용하고 마지막 값이 우선합니다. 기존 객체 passthrough Get은 아래 설명처럼 옵션을 소비하지 않습니다.

## 독립 Go 예제

SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장합니다. 기본은 목록 조회이고 `-id`를 지정하면 일반 Get을 사용합니다. `-filters`는 JSON 문자열이므로 shell에서는 작은따옴표로 감쌉니다. 예를 들어 `-filters '{"status":"ACTIVE"}'` 또는 `-id IP_ID -direct`를 사용할 수 있습니다. `-source`를 생략하면 cloud 설정을 사용합니다.

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
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    source := flag.String("source", "", "cloud default, neutron, nova or none")
    id := flag.String("id", "", "floating IP identifier; empty lists IPs")
    filters := flag.String("filters", "", "JSON query or local filter")
    direct := flag.Bool("direct", false, "use direct GET for a UUID-shaped ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *source, *id, *filters, *direct); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, source, id, filters string, direct bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud),
        sdk.WithMicroversion(sdk.Compute, "2.35"))
    if err != nil {
        return err
    }
    options := []compute.FloatingIPQueryOption{
        compute.WithFloatingIPQueryTimeout(time.Minute),
        compute.WithFloatingIPQueryDirectGet(direct),
    }
    if source != "" {
        options = append(options,
            compute.WithFloatingIPQuerySource(compute.FloatingIPSource(source)))
    }
    if filters != "" {
        options = append(options,
            compute.WithFloatingIPQueryFilters(json.RawMessage(filters)))
    }
    var value json.RawMessage
    var pages int
    if id == "" {
        result, queryErr := conn.ListFloatingIPs(ctx, options...)
        err = queryErr
        if result != nil {
            value, pages = result.Value, len(result.Pages)
        }
    } else {
        result, queryErr := conn.GetFloatingIP(ctx,
            compute.GetFloatingIPRequest{ID: id}, options...)
        err = queryErr
        if result != nil {
            value, pages = result.Value, len(result.Pages)
        }
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    encodeErr := encoder.Encode(map[string]any{"value": value, "pages": pages})
    return errors.Join(err, encodeErr)
}
```

Compute microversion2.35는 Nova 조회나 fallback을 위해 지정했습니다. 이 설정 때문에 Neutron 조회에서 Compute endpoint를 미리 발견하지 않습니다. Nova legacy IP/pool API는 selected Compute2.36 이상에서 사용할 수 없으며 SDK가 버전을 자동으로 낮추지 않습니다. Connect는 예제의 부모2분 예산을 사용하고 내부 query는 옵션 준비 후1분 예산을 사용합니다.

이 예제는 실패 시에도 확보한 page 수를 출력합니다. 전체 evidence를 확인하려면 아래의 `Pages`, `Failure`, `Observed`를 읽습니다. JSON `value:null`만으로 실패와 정상 no-match를 구분할 수 없으므로 함께 반환한 오류를 확인합니다.

## List와 Search의 필터 분기

Neutron List의 dictionary에서 알려진 query key는 서버로 전송합니다. 서버 query에 없는 known Body key는 로컬 Resource view에 적용하고, 임의 unknown key는 source처럼 무시합니다. 예를 들어 `status`는 서버 query, `updated_at`·`port_details`는 로컬 Body 필터입니다. 서버 query를 응답에서 다시 검증하지 않습니다. `tenant_id`는 wire `project_id`로 매핑되며 둘 다 주면 source의 tenant alias 우선순위를 사용합니다.

Neutron Search에 dictionary를 주면 **`SearchFloatingIPsRequest.ID`를 무시**하고 같은 Resource 목록 경로를 사용합니다. 빈 `{}`도 dictionary라 ID가 무시됩니다. ID로 로컬 검색하려면 필터를 생략하거나 문자열 표현식을 사용합니다. 일반 Get도 기본적으로 Search를 이용하므로 `GetFloatingIPRequest{ID: ...}`에 `{}`를 추가하면 결과가 여러 개인 오류가 날 수 있습니다.

Search에서 필터를 생략하거나 문자열 표현식을 주면 unfiltered public List 뒤 식별자 exact/glob 검색과 로컬 필터를 적용합니다. 식별자는 각 row의 id/name에 적용하고 순서·중복을 유지합니다. exact match가 존재한다고 다른 glob match를 제외하지 않습니다. Neutron view의 name은 floating_ip_address alias이고 Nova 정규화는 별도 name alias를 만들지 않습니다.

Nova/None의 nonempty List filters는 서버 조회 전에 `ErrInvalidOption`입니다. Nova에서 dictionary 로컬 필터를 사용하려면 Search를 호출합니다. Source=None도 이 explicit 조회에서는 Nova 경로이며 자동 needs의 disabled skip와 다릅니다.

로컬 cloud dictionary는 nested subset과 Python bool/number equality를 사용하며 missing key는 오류입니다. Resource Body dictionary는 missing nested key를 null로 비교하는 별도 정책입니다. JMESPath는 임의 JSON 값을 만들 수 있으므로 expression 사용 후에는 `Value`를 읽습니다. projection/scalar 결과를 실제 floating IP 원본 목록으로 합성하지 않고 `FloatingIPs`는 nil로 둡니다.

## 404와 fallback

| 경로 | 처리 |
|---|---|
| Neutron List, 필터 생략 또는 `{}` | 순수 HTTP404면 Nova로 fallback하고 `FallbackError`에 원인을 보존 |
| Neutron List, nonempty dictionary | 순수 HTTP404면 `Value:[]`, 빈 FloatingIPs, nil error. `SuppressedNotFound`와 응답 evidence 유지 |
| Neutron Search, dictionary (`{}` 포함) | 404를 전파. public List fallback을 우회 |
| Search, 필터 생략 또는 JMES 문자열 | unfiltered List의 fallback을 사용한 뒤 로컬 검색 |
| Nova public List/Search | 순수 HTTP404는 빈 목록. `SuppressedNotFound`에 원인 보존 |
| GetByID 또는 UUID direct Get | 404 전파. 목록 fallback 없음 |
| Pools List/Search | Compute404 전파. Neutron networks로 대체하지 않음 |

403, 응답 decode 실패, context 취소, source 변경과 혼합된 오류를 빈 목록이나 Nova fallback으로 숨기지 않습니다. Neutron 목록의 뒤 페이지404도 public List의 위 필터 정책을 적용하지만 먼저 받은 목록 일부를 성공한 최종 선택으로 반환하지 않습니다. `Pages`는 이미 받은 응답의 증거를 보존합니다.

정확한 Network catalog endpoint 부재는 Nova 경로를 선택할 수 있습니다. 혼합 catalog 오류는 전파합니다. catalog 부재만으로 선택한 Nova는 Neutron HTTP404를 받지 않았으므로 `FallbackError`가 nil일 수 있습니다.

## Get·GetByID·Existing

일반 Get의 기본은 목록 기반 선택입니다. 선택이 없으면 오류 없이 결과의 `Value`와 `FloatingIP`가 nil이며 Python의 `None`에 대응합니다. 결과 객체에는 조회한 Pages 또는 suppressed/fallback 증거가 남을 수 있습니다. 한 row는 반환하고, 복수 match는 `FloatingIPSelectionError`이며 `errors.Is(err, resource.ErrAmbiguous)`로 확인할 수 있습니다.

`WithFloatingIPQueryDirectGet(true)`와 UUID형 ID를 조합하면 직접 GET합니다. 필터를 적용하지 않고 404도 전파합니다. Nova decimal ID처럼 UUID가 아닌 입력은 일반 Get에서 계속 검색합니다. 입력 형식과 관계없이 직접 GET을 원하면 `GetFloatingIPByID`를 사용합니다. GetByID는 typed ID를 검증하고 URL path에 escaping합니다.

`GetFloatingIPRequest{Existing: record}`는 Resource에 id key가 있는 기존 `*compute.FloatingIPRecord`를 같은 포인터로 반환합니다. HTTP 및 옵션 적용을 생략하며 별도 ID 입력·필터·source 설정도 사용하지 않습니다. id key의 값이 null이어도 passthrough 조건은 충족합니다. nil/canceled context는 오류입니다. 이 concrete passthrough는 임의 Python object/Munch가 가진 모든 후속 동작을 제공한다는 뜻은 아닙니다.

Get에서 JMES 결과가 원본 row 목록이 아니면 `Value`에 source 방식으로 선택한 JSON을 반환할 수 있고 `FloatingIP`는 nil입니다. 표현식 전체가 false·0·빈 문자열·null이면 선택 없음이며 Value가 nil입니다. 반면 한 원소 배열 [false]·[0]·[""]는 해당 원소를 반환합니다. scalar/object의 길이·인덱스 선택이 불가능하면 오류입니다. resource가 필요한 코드는 표현식 결과 대신 `FloatingIP` 여부를 확인합니다.

## Resource·Wire·NormalizationSource

`FloatingIPRecord.Backend`는 실제 데이터를 받은 backend입니다. `Resource`는 공개 조회·로컬 검색용 owned view이고 `Wire`는 실제 응답 row입니다. unknown·nullable JSON과 HTTP Header/StatusCode를 별도로 유지하며 한쪽 view 변경으로 원본 증거가 바뀌지 않습니다. `RawResource.Decode`로 native model에 투영할 수 있지만 해당 native decoder의 field type 제한도 적용됩니다.

Neutron Resource view는 known Body의 missing null, name/IP alias, project/tenant alias, missing tags[], revision/port_details/tags descriptor 변환과 location을 제공합니다. 예를 들어 canonical project_id present-null은 tenant_id로 보충하지 않습니다. Neutron 단건 응답에 id가 빠지면 Resource는 요청 ID를 유지하지만 Wire에 ID를 만들지 않습니다. 응답에 id가 present-null 또는 다른 값이면 그 응답 값을 유지합니다. missing key와 present-null은 Wire에서 계속 구별됩니다. native Python mutable Resource의 fetch/commit/context 전체를 이 값 모델이 제공하지는 않습니다.

Neutron은 iterator가 반환한 각 raw 행의 descriptor 정규화와 known Body 로컬 필터를 **다음 페이지 요청 전에** 적용합니다. 따라서 앞 페이지의 변환·필터 오류를 뒤 페이지404나 fallback으로 숨기지 않습니다. 빈 nested dictionary는 실제 값이 truthy이면 shape를 소비하지 않고 일치합니다. 최종 목록은 전체 조회가 성공한 뒤 반환하며, 준비한 행을 재사용해 location과 필터를 두 번 소비하지 않습니다. max_items와 continuation marker는 필터 전 physical 행을 기준으로 합니다.

Nova는 cloud 정규화 view를 만들며 `Normalized=true`입니다. canonical present-null이 legacy alias를 우선하고, instance_id를 attached 계산에 소비하며 location/project와 properties를 구성합니다. `WithFloatingIPQueryStrict(true)`는 Nova 호환 aliases 및 properties의 extra top-level 복원을 제외합니다. properties 자체를 제거하거나 Neutron view를 strict 형태로 바꾸는 옵션은 아닙니다.

Nova 정규화의 `NormalizationSource`는 source의 configured Neutron 판정까지 기록합니다. 예를 들어 Neutron404 후 Nova 응답을 받으면 `Backend=FloatingIPNova`지만 `NormalizationSource=FloatingIPNeutron`일 수 있으며 attached는 port, missing status는 UNKNOWN 규칙을 사용합니다. 직접 Nova/None 정규화는 Python과 같이 canonical status ACTIVE를 합성합니다. 이는 **조회 view의 source 호환 값**이며 실제 IP ACTIVE나 연결 완료 증거가 아닙니다. raw mutation/Available의 [Nova model](server-nova-floating-ip.md)은 이 합성 상태를 사용하지 않습니다.

Nova ID는 조회 view/Wire의 raw JSON 값을 보존합니다. null·숫자·불완전 주소/association row를 mutation 후보 자격으로 선제 거절하지 않습니다. 읽기 결과를 서버 연결 작업에 넘길 때는 해당 mutation API가 필요한 ID/IPv4/pool/association 제약을 다시 확인합니다.

## Pages·Failure와 부분 결과

* `Pages`는 이미 받은 물리적 목록 응답의 Backend, 전체 Envelope, Header, StatusCode입니다. 목록 자체가 실패해도 이미 받은 페이지가 남을 수 있습니다.
* `Failure`는 실패·억제·fallback에서 확인한 물리적 응답 evidence입니다. 성공한 fallback의 최종 Backend와 Failure.Backend가 다를 수 있습니다. 모든 로컬 오류가 HTTP 응답을 가진 것은 아닙니다.
* 단일 직접 GET의 `Observed`는 접수한 member 응답입니다. JSON body의 변환·정규화가 실패해도 원본 Wire/Observed가 남을 수 있고 성공한 Resource를 합성하지 않습니다.
* `FallbackError`와 `SuppressedNotFound`는 nil error 결과에서도 보존할 수 있습니다. 빈 선택과 API404를 뒤늦게 구분하는 데 사용합니다.

Pages는 validation hook에 도달한 페이지를 기록합니다. 접수200 뒤 read/Close가 실패하면 해당 응답은 Pages에 들어가지 않을 수 있고 Failure로 보존합니다. 완료되지 않은 목록/표현식에서 확보한 page를 최종 `FloatingIPs`·`Value` 성공 선택으로 표시하지 않습니다. 전체 목록 수집과 변환이 완료된 뒤 선택을 반환합니다. 요청·client/provider/endpoint/microversion·lazy service의 source가 변경되면 조회를 중단하고 새로운 backend로 재시도하지 않습니다.

## 페이지 제어

Neutron dictionary filters는 Resource kwargs `limit`, `marker`, `max_items`, `paginated`도 소비합니다. 별도 builder를 만들지 않고 `WithFloatingIPQueryFilters(json.RawMessage(...))`에 함께 넣습니다. 예를 들어 `{"status":"ACTIVE","limit":10,"max_items":20,"paginated":true}`는 page limit10과 raw row cap20을 요청합니다.

`limit`은 wire query이고 양의 정수여야 합니다. `max_items`는 로컬 Body 필터를 적용하기 전 성공적으로 decode한 raw row 수를 제한합니다. 필터에서 탈락한 row도 포함하므로 최종 match20개를 보장하지 않습니다. explicit limit이 없으면 max_items를 wire limit hint로 사용합니다. `paginated:false`는 첫 페이지만 소비합니다. cap 이후 row 또는 next link를 불필요하게 소비하지 않습니다.

기본 Neutron pagination은 links/next/HTTP Link와 explicit limit의 marker continuation을 처리하며 source처럼 빈 page에서 종료합니다. 반복 continuation과 다른 collection/filter로 바뀌는 link는 오류입니다. limit·marker의 null 또는 값이 없는 iterable query(예: []·[null])는 HTTP에서 생략합니다. 빈 문자열은 유효한 limit·marker가 아닙니다. limit 없이 서버 link도 없으면 다음 page를 추정하지 않습니다. Go의 strict positive limit·nonnegative integer max_items 및 continuation 검증은 Python의 모든 동적 controls 입력을 그대로 허용한다는 뜻은 아닙니다.

이 controls는 Neutron dictionary Resource 경로에 적용합니다. Nova List는 nonempty dict를 허용하지 않고, Nova Search의 dict는 normalized row 로컬 필터이므로 같은 key를 자동 pagination 옵션으로 바꾸지 않습니다. SDK Nova HTTP Link traversal은 Python cloud helper의 단일 GET보다 넓은 구현 범위입니다.

## Pools의 이름 projection

Pools는 IP source와 관계없이 legacy Compute를 조회합니다. 각 logical pool은 name key만 가진 `RawResource`이며 서버가 반환한 id/extension은 `Pools`·`Value`에서 제외합니다. 전체 원본은 `Pages[].Envelope`에 남습니다. Search는 이 projection에 name exact/glob과 cloud filters/JMES를 적용하므로 제거된 raw extension 필터를 조회 가능한 속성으로 합성하지 않습니다. Pool 목록404는 빈 성공 결과로 바꾸지 않습니다.

## 고정 Python 비교와 남은 범위

```python
import openstack

conn = openstack.connect(cloud="dev")
all_ips = conn.list_floating_ips()
selected = conn.search_floating_ips(id="IP_ID")
ip = conn.get_floating_ip("IP_ID")       # 기본 목록 검색, 없으면 None
direct = conn.get_floating_ip_by_id("IP_ID")  # 직접 GET, 404 오류
pools = conn.search_floating_ip_pools(name="public*")
# Neutron에서 filters={}를 주는 Search는 id 인자를 무시한다.
```

비교 pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`다. [공개 list/search/get/pools](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L435-L606), [cloud 로컬 filter와 Get 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L201), [Neutron Resource query/body 정책](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2217-L2358), [Nova 정규화](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1766-L1830)를 기준으로 한다.

이 named Cloud 조회들은 concrete 입력과 SDK 소유 필터·기본값·backend 선택·owned 결과를 제공하는 Go 매핑입니다. Python 임의 object/Munch·JSON 바깥 동적 filter/control·특수 UUID 표기는 문서화한 typed 입력 경계로 대응합니다. 전체 mutable Resource·inherited fetch/commit/session/cache/discovery·다른 Resource/Proxy 선언은 별도 SDK 작업으로 추적합니다. 응답200·canonical UTF-8·안전한 member/continuation·source/context guard·기록된 location은 명시적인 Go 정책입니다. 아래 예제의 컴파일과 HTTP 계약 테스트는 인증된 실제 OpenStack 배포 실행과 구분합니다.

[독립 Floating IP 삭제](floating-ip-delete.md)는 이 공개 Get 정책으로 접수 후 결과를 확인합니다. 기본값은 추가 DELETE1회이며, 목록의 no-match와 present DOWN을 별도로 반환합니다.

새 IP 할당·선택적 대기·timeout 정리는 [CreateFloatingIP](floating-ip-create.md)에서 제공합니다.

[미연결 Floating IP 일괄 정리](floating-ip-unattached-delete.md)는 Neutron 전체 목록을 확보한 뒤 port가 비어 있는 항목을 순차 삭제합니다. 개별 false는 계속 처리하고 오류는 중단하며 SDK 소유 옵션·한 deadline·항목별 부분 결과를 제공합니다.
