# 공통 리소스 계층

서비스별 호출의 공통 계약을 `Collection[T]`에 둡니다. 애플리케이션은 서비스가 제공한 collection을 사용하고, Adapter 등록은 SDK 서비스 구현이 담당합니다. 애플리케이션의 builder interface 구현은 필요하지 않습니다.

## 참조와 조회

```go
resource.ID("server-id") // ID로만 조회; 실패해도 이름으로 재해석하지 않음
resource.Name("web-01")  // 정확한 이름으로만 조회
resource.ID(server.ID)   // 조회한 응답을 다음 작업에서 참조
```

빈 참조는 오류입니다. ID의 기본 검증은 URL path 한 구간이며 UUID 형식으로 제한하지는 않습니다. 서비스에 따라 식별자 문법이 다르면 SDK의 binding이 검증과 URL 인코딩을 함께 맡습니다. 이름은 query로 인코딩하므로 이름에 포함된 구두점도 그대로 사용할 수 있습니다.

| 메서드 | 동작 |
|---|---|
| `Get(ctx, id)` | 단일 ID 조회, 404면 `ErrNotFound` |
| `Find(ctx, ref, ...LookupOption)` | 명시 ID 조회 또는 정확한 이름 검색, 중복 검사 |
| `FindIdentity(ctx, identity, ...IdentityFindOption)` | SDK가 지원한 binding의 이름·ID 문자열 자동 조회, 기본 미존재 무시 |
| `ResolveID(ctx, ref)` | ID는 요청 없이 검증, 이름은 정확히 찾아 안정적인 ID 반환 |
| `List(ctx, ...ListOption)` | lazy `iter.Seq2[*T, error]`, 모든 페이지 순회 |
| `All(ctx, ...ListOption)` | iterator를 slice로 수집 |
| `Delete(ctx, ref, ...LookupOption)` | 이름이면 ID 해석 후 삭제, 기본 미존재 무시 |
| `Wait(ctx, ref, status, ...WaitOption)` | 참조를 한 번 해석하고 동일 ID의 상태 확인 |
| `WaitDeleted(ctx, ref, ...WaitOption)` | 동일 ID를 조회하다가 404·nil 결과·deleted 상태면 삭제 완료 |

Find의 이름 검색은 현재 클라이언트의 기본 조회 범위 안에서 수행합니다. Find에 별도 tenant/project 필터를 전달하는 기능은 아직 없습니다. 중복 오류의 IDs는 중복을 확인한 첫 두 리소스입니다.

Nova 서버/flavor·Cinder v3 볼륨·Glance v2 이미지·Neutron 네트워크/subnet/포트/router/security group/subnet pool/trunk/QoS policy/address group, Keystone 프로젝트/사용자/그룹/domain/role과
고정 zone의 Designate recordset·고정 pool의 Octavia member는 `FindIdentity`로 문자열을 자동 조회합니다.
기본 GET400·403·404 fallback, 양쪽 HTTP 단계의 query 옵션, unsafe 이름의 목록 경로와 Python 사용법은
[이름·ID 자동 조회](../docs/finding-identities.md)에 설명합니다. 다른 native binding은
이 자동 정책을 아직 지원하지 않습니다.

Nova 서버·Cinder v3 볼륨에서는 `WithIdentityFindDetails(false)`로 fallback summary
목록을, `WithIdentityFindAllProjects(true)`로 목록의 cross-project 검색을 선택합니다.
두 옵션은 GET에 전달하지 않으며 기본값은 details=true, all_projects=false입니다.

Nova flavor는 이름 query를 자동 추가하지 않고 상세 목록을 검색합니다. caller가
지정하지 않은 `is_public`의 목록 기본값은 `None`입니다. `WithIdentityFindExtraSpecs(true)`는
단일 결과의 ExtraSpecs가 비었을 때만 추가 GET을 하고, 기본 false는 추가 호출이 없습니다.
이 Flavor 전용 옵션은 후속 GET 실패를 미존재로 숨기지 않습니다.

Glance 이미지는 정상적인 일반 검색에서 찾지 못하면 원래 query에 `os_hidden=true`를
적용한 목록을 한 번 더 검색합니다. 두 번째 목록에는 자동 이름 hint를 추가하지 않습니다.
오류·중복·취소는 즉시 반환하며 미존재 옵션은 두 검색이 모두 정상적으로 끝난 뒤 적용합니다.

자동 조회가 활성화된 native binding은 `WithIdentityFindQuery("status", ...)`를
모델의 Status 필드 유무와 관계없이 wire query로 보존합니다. 일반 목록의 raw
`WithQuery("status", ...)` 단독도 응답 status의 로컬 필터를 활성화하지 않습니다.
Octavia Member의 `WithStatus`는 wire 대신 로컬 필터를 사용하며, raw status를 함께
지정하면 마지막 값으로 로컬 비교합니다. cap은 로컬 필터 전에 적용합니다.

## 옵션

| 연산 | 옵션 |
|---|---|
| 목록 | `WithName`, `WithStatus`, `WithPageSize`, `WithMaxItems`, `WithPaginated`, `WithFilter`, `WithFilters`, `WithQuery`, `WithBodyFilter`, `WithBodyFilters` |
| 조회/삭제 | `WithIgnoreMissing`, `WithMissingError` |
| 자동 문자열 조회 | `WithIdentityFindOptions`, `WithIdentityFindIgnoreMissing`, `WithIdentityFindFallback`, `WithIdentityFindQuery`, `WithIdentityFindExtraSpecs` |
| 대기 | `WithTimeout`, `WithUnlimitedWait`, `WithPollInterval`, `WithFailureStates`, `WithStatusAttribute`, `WithProgressCallback` |

`WithFilter`/`WithFilters`는 감사된 속성 이름을 서버 query 또는 로컬 Body 조건으로
분류합니다. 현재 Subnet의 query 24개·로컬 Body 9개, Secret의 query 12개·로컬 Body 12개,
Container의 query 2개·로컬 Body 10개, Order의 query 2개·로컬 Body 14개,
AddressGroup의 query 8개·로컬 Body 3개, QoS Policy의 query 15개·로컬 Body 2개,
Subnet Pool의 query 16개·로컬 Body 10개, Network의 query 23개·로컬 Body 14개,
Router의 query 18개·로컬 Body 10개, Security Group의 query 17개·로컬 Body 3개에
연결되어 있고 다른 binding은 clear를 포함해 `ErrUnsupported`입니다. 개별 옵션은 같은 target의 마지막 값이 이기고,
한 bulk map의 query canonical 이름은 wire 별칭보다 우선합니다. bulk 교체/clear는 semantic
조건만 바꾸며 최종 선택값만 검증합니다. 알 수 없는 이름은 버리고 raw query·명시 Body와
같은 target을 지정하면 HTTP 전에 `ErrInvalidOption`입니다. 선언·reserved controls·인코딩은
[Subnet Python/Go 사용법](../network/v2/subnets/README.md)과
[Secret Python/Go 사용법](../keymanager/v1/secrets/listing/README.md)과
[Container Python/Go 사용법](../keymanager/v1/containers/listing/README.md)과
[Order Python/Go 사용법](../keymanager/v1/orders/listing/README.md)과
[AddressGroup Python/Go 사용법](../network/v2/extensions/security/addressgroups/listing/README.md)과
[QoS Policy Python/Go 사용법](../network/v2/extensions/qos/policies/listing/README.md)과
[Subnet Pool Python/Go 사용법](../network/v2/extensions/subnetpools/listing/README.md)과
[Network Python/Go 사용법](../network/v2/networks/listing/README.md)과
[Router Python/Go 사용법](../network/v2/extensions/layer3/routers/listing/README.md)과
[Security Group Python/Go 사용법](../network/v2/extensions/security/groups/listing/README.md)에 설명합니다.

`WithBodyFilter`/`WithBodyFilters`는 ordinary `Resources.List/All`에서 감사된 응답 필드를
로컬 비교합니다. 현재 QoS Policy의 `rules`, Address Group의 `addresses`, Subnet Pool의
`prefixes`, Network의 `subnets`를 지원합니다. Network의 Python 이름 `subnet_ids`는 SDK가
`subnets`의 별칭으로 처리하며 canonical/alias의 마지막 옵션이 이깁니다. 같은 bulk map에
두 이름을 넣으면 `ErrInvalidOption`입니다.
QoS Policy는 `rules`·`tenant_id`를 원본 행에서 비교합니다. native `Rules`의 float64 반환값을
유지하면서 로컬 조건은 원래 숫자 정밀도로 비교합니다. `is_shared`·`name`·`id`·`project_id`·태그는
서버 query이며 `tenant_id`가 `project_id`의 query 별칭이 되지는 않습니다.
AddressGroup은 `id`·`tenant_id`·`addresses`를 원본 행에서 비교합니다. `name`·`project_id`는
semantic 서버 query이므로 로컬 Body 조건에 포함하지 않습니다. 기존 `WithName`은 별도의
서버 hint와 로컬 이름 조건을 제공하며 semantic `name`과 동시에 지정하면 충돌 오류입니다.
Subnet의 9개 로컬 필드는 [원본 응답 비교](../network/v2/subnets/README.md)를 제공하며
`prefix_length`를 `prefixlen`의 별칭으로 처리합니다. native Subnet은 그대로 반환합니다.
Subnet Pool은 `id`·`tenant_id`·prefix 배열·두 timestamp·다섯 정수 속성의 10개 로컬 필드를
원문에서 선택합니다. semantic `id`는 로컬 조건이고 raw `WithQuery("id", ...)`는 서버 query입니다.
`default_prefix_length`·`minimum_prefix_length`·`maximum_prefix_length`는 Python 이름이며,
명시 Body 옵션은 각각 `default_prefixlen`·`min_prefixlen`·`max_prefixlen`도 받습니다.
native 전체 페이지 디코드를 먼저 적용하며 prefix 배열의 null 요소는 원문으로 비교하지만
반환 `Prefixes`의 해당 요소는 native 빈 문자열입니다. 정수 변환과 timestamp의 경계는
[Subnet Pool 사용법](../network/v2/extensions/subnetpools/listing/README.md)에 설명합니다.
Network는 query 23개·accepted 이름 35개와 로컬 Body 14개를 분류합니다. `name`·`status`·`id`는
서버 조건이고 `WithName`·`WithStatus`가 요청한 로컬 비교는 별도로 유지합니다. `subnet_ids`를
원문 `subnets`에 연결해 null 요소를 비교하며 typed 반환 배열의 null→빈 문자열 변환은 유지합니다.
네 boolean의 응답 truthiness는 missing/null을 null로 보존하고 caller 조건은 변환하지 않습니다.
`mtu`·`revision_number`에는 정확한 정수 정책을 사용하고 나머지 여덟 필드는 원문 JSON입니다.
같은 정책을 상위 `Network(ctx).Networks`와 versioned `Networks.Resources`에서 사용합니다.
[Network 사용법](../network/v2/networks/listing/README.md)에 Python 변환·native decoder·상태 query의 차이를 설명합니다.
Router는 query 18개·accepted 이름 24개와 로컬 Body 10개를 분류합니다. `name`·`status`·`id`는
서버 조건이며 `WithName`·`WithStatus`의 기존 로컬 비교는 별도로 유지합니다.
semantic `revision_number`와 명시 Body의 `revision`·`revision_number`는 원문 `revision`을
선택합니다. native 모델의 `RevisionNumber`는 별도의 `revision_number` 응답을 읽으며
누락된 `revision`을 대신하지 않습니다. `enable_ndp_proxy`는 null을 보존한 boolean truthiness,
`evpn_vni`·`revision`은 정확한 정수, 나머지 일곱 필드는 원문 JSON으로 비교합니다.
`tenant_id`는 로컬 조건이고 `project_id`는 query입니다. gateway·routes의 unknown 필드와
null 요소는 비교에 보존하며 반환값은 native Router입니다.
[Router 사용법](../network/v2/extensions/layer3/routers/listing/README.md)에 변환·페이지 경계를 설명합니다.
Security Group은 query 17개·accepted 이름 21개와 로컬 Body 3개를 분류합니다.
`revision_number`·`tenant_id`·`project_id`·`stateful`·`is_shared`는 서버 조건이며 timestamp 두 개와
`security_group_rules`만 원문 JSON으로 비교합니다. rule 배열의 추가 필드·정확한 숫자·null 요소를
비교에 보존하고 native `SecGroup`·`SecGroupRule`의 알려진 필드·timestamp 디코드를 먼저 적용합니다.
native 반환 배열의 null 요소는 zero struct입니다. `WithName`의 기존 로컬 비교는 유지하고
`WithStatus`는 모델에 Status가 없어 미지원입니다. semantic `status`는 버리며 raw status는 전달합니다.
[Security Group 사용법](../network/v2/extensions/security/groups/listing/README.md)에 concrete native 목록의
SDK 소유 raw query·입력 복사·페이지 경계를 설명합니다.
Secret은 timestamp의 속성 이름과 raw wire 이름을 구분하고 literal `id`·전체 `secret_ref`·
별도의 `secret_id` formatter 결과를 비교합니다. 로컬 필드 12개를 원문 행에서 선택하고 native Secret을 반환합니다.
Container는 `name`·timestamp·참조·중첩 배열 등 10개 로컬 속성을 비교합니다. `id`는 literal
필드가 없을 때 전체 `container_ref`를 사용하며 `container_id`는 마지막 path 부분입니다.
`secret_refs`·`consumers`는 원문 배열의 순서와 추가 필드까지 비교하고 native Container를 반환합니다.
Order는 최상위 `name`, 원문 `meta`와 timestamp 등 14개 로컬 속성을 비교합니다. 전체
`id`/`order_ref`와 두 formatted 속성 `order_id`·`secret_id`를 각각 선택합니다.
native `Meta.Name`은 중첩 metadata이며 공통 `WithName`이나 이름 조회에 연결하지 않습니다.
SDK가 JSON snapshot·필드 선택·배열 전체 equality를 처리하며 builder/predicate는 필요하지
않습니다. 지원이 없는 리소스·알 수 없는 필드는 HTTP 전에 거부하고 raw query와 typed List·
FindIdentity는 유지합니다. [Python/Go Body 필터 사용법](../docs/listing.md#명시적인-native-body-필터)에
bulk 교체·clear·nil/빈 배열·raw cap·native 숫자 정밀도의 경계를 설명합니다.

페이지 크기와 로컬 행 수 제한을 구분합니다. `WithPageSize(100)`은 페이지마다 서버에 요청하는
크기이며 기본 List/All은 후속 페이지도 읽습니다. `WithMaxItems(250)`은 서버 응답에서 decode한
행을 로컬 name/status 필터 전에 최대 250개 소비합니다. 따라서 필터를 통과해 반환하는 결과는
250개보다 적을 수 있습니다. 0은 무제한이고 음수는 lazy 순회가 시작될 때 HTTP 전에
`ErrInvalidOption`입니다. 같은 설정은 마지막 옵션이 우선합니다.

`WithPaginated(false)`는 첫 페이지만 읽습니다. MaxItems와 함께 사용하면 첫 페이지에서도
cap에서 멈춥니다. 기본값과 `WithPaginated(true)`는 continuation을 허용합니다. cap 또는
`break`에서 중단하면 추가 페이지를 가져오지 않습니다. wire limit hint는 서비스별 opt-in이며,
공통 MaxItems가 모든 서버에 limit을 보내는 것은 아닙니다.

`max_items`와 `paginated`는 로컬 옵션 이름입니다. `WithQuery`에 넣으면 HTTP 전에
`ErrInvalidOption`을 반환하므로 전용 `WithMaxItems`와 `WithPaginated`를 사용합니다.

iterator 사용 예제는 오류를 반환하는 함수 안에서 작성합니다:

```go
for server, err := range service.Servers.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(server.ID)
    if shouldStop { break }
}
```

에러는 `(nil, error)`로 한 번 전달하고 순회를 종료합니다. iterator를 다시 순회하면 새로운 API 요청이 시작됩니다. All 도중 오류가 발생하면 부분 결과 대신 `nil, error`를 반환합니다.
List를 만들 때 전달한 옵션 slice를 보관하므로 caller가 나중에 slice의 옵션을 바꾸어도
기존 iterator의 동작은 바뀌지 않습니다. 옵션 적용과 검증은 각 순회가 시작될 때 수행합니다.

Senlin REST bindings는 실제 소비한 행의 decode·검증 뒤 cap을 적용하고, 사용하지 않을
후속 행이나 next link를 해석하지 않습니다. native Gophercloud pager는 페이지 전체를 먼저
Extract하므로 cap 뒤 malformed 행도 extraction 오류를 낼 수 있습니다. page 경계가 없는
custom Iterate는 cap을 적용하지만 first-page 제어를 지원하지 않으면
`WithPaginated(false)`에 `ErrUnsupported`입니다. 빈 페이지 continuation 정책도 binding에
따릅니다. Senlin의 typed 옵션·limit hint·Python과의 차이는
[목록 제어](../clustering/v1/listing/README.md)에 있습니다.

생성된 native binding 121개(일반 Collection 107개와 부모 범위 14개)는 공통
`WithMaxItems`와 `WithPaginated`를 native pager 소비 지점에 적용합니다. Swift 컨테이너·객체,
Trove 데이터베이스·사용자, Nova action 이력의 수동 binding 5개도 지원합니다.
native typed API의 `WithListOptions` 등 기존 옵션은 그대로 사용하며, 로컬 제어는
`Resources.List/All` 또는 scope의 `List/All`에 전달합니다. native 경로는 MaxItems로
wire limit을 추정하지 않습니다. Trove 사용자 host 필터도 raw cap 뒤에 적용합니다.
상위 Compute 서버/flavor, Image 이미지, Network 네트워크와 Block Storage 볼륨의 native
pager 경로 및 Senlin의 REST 목록 제어도 사용할 수 있습니다.
[서비스별 Python/Go 목록 비교](../docs/listing.md)에 실제 사용 예제가 있습니다.

공통 native stream은 기존 pager의 continuation 의미와 순환 검사를 유지합니다.
origin·path·query 연속성 검사는 별도 REST binding의 정책이며 native stream에 자동으로
적용되지 않습니다. cap·첫 페이지·break로 소비하지 않을 next link는 해석하지 않습니다.

서버의 `next` URL 또는 marker가 이전에 요청한 페이지를 반복하면 추가 요청 전에 `ErrPaginationCycle`로 중단합니다. 같은 URL의 query 순서가 바뀌어도 반복으로 판정합니다. `PaginationCycleError.URL`에는 반복한 링크가 들어 있습니다. 이 정책은 서비스의 typed List, 공통 Collection, page 단위 iterator에 함께 적용됩니다. 소비자가 `break`하면 다음 링크 검사와 후속 요청을 하지 않습니다.

## 오류와 상태 대기

`errors.Is`로 `ErrNotFound`, `ErrAmbiguous`, `ErrUnsupported`, `ErrInvalidOption`, `ErrFailedState`, `ErrPaginationCycle`을 구분합니다. `errors.As`로 `NotFoundError`, `AmbiguousError`, `FailedStateError`, `PaginationCycleError`, `OperationError`의 상세 정보를 확인합니다. HTTP에서 발생한 미존재는 원래 Gophercloud 오류도 보존합니다.

SDK 소유 Cyborg·Senlin·Masakari 모델의 `Metadata.Body`는 원래 JSON 필드를 `json.RawMessage`로 보존합니다. 큰 정수를 `float64`로 바꾸지 않으며 null·빈 값·생략도 구별합니다. `Metadata.Header`와 `StatusCode`는 받아들인 응답의 HTTP 근거입니다. Senlin·Masakari의 받아들인 응답을 읽거나 해석하지 못하면 `ResponseError`가 원문·헤더·상태·cause를 보존합니다. 생성 요청이 이미 성공했을 수 있으므로 이 오류만으로 자동 재전송하지 않습니다.

Wait의 기본 timeout은 5분, 간격은 2초입니다. `WithUnlimitedWait()`는 SDK의 timeout을 없애며 부모 context의 취소와 deadline은 유지합니다. 뒤의 `WithTimeout(...)`으로 다시 제한할 수 있습니다. 대상 상태 비교는 대소문자를 구분하지 않습니다. 실패 상태는 서비스별 Adapter가 선언합니다. `WithFailureStates("ERROR", "BROKEN")`은 이를 정확한 대소문자 무시 비교로 교체하고, 인자 없는 `WithFailureStates()`는 상태에 의한 실패 검사를 끕니다. 대상 상태와 실패 상태가 같다면 대상 도달을 먼저 판정합니다. 서비스 조회 자체가 반환하는 실패 오류는 이 옵션으로 무시하지 않습니다.

Python `resource.wait_for_status(..., failures=[], wait=None)`에 대응하는 Go 옵션은 `WithFailureStates(), WithUnlimitedWait()`입니다. Go는 응답 객체를 받아 캐시된 상태를 검사하는 대신 처음부터 HTTP로 조회하고, timeout을 HTTP 요청에도 context deadline으로 전달합니다. 이미 취소된 context에서는 목표 상태 응답도 성공으로 반환하지 않습니다.

Python의 `attribute="provision_state"`는 `WithStatusAttribute("provision_state")`에 대응합니다. SDK가 모델의 JSON tag 또는 exported Go 필드 이름을 찾으므로 별도 getter나 builder를 구현하지 않습니다. embedded 모델도 지원하며, 없는 필드·문자열이 아닌 필드·모호한 필드는 요청 전에 `ErrUnsupported`입니다. `*string`은 지원하지만 실제 응답이 nil이면 명확한 오류를 반환합니다. `WaitUntilFinished`처럼 완료 조건을 고정한 전용 waiter에서는 속성을 교체할 수 없습니다.

```go
node, err := service.Nodes.Resources.Wait(ctx, resource.ID("node-uuid"), "power on",
    resource.WithStatusAttribute("power_state"),
    resource.WithFailureStates("error"),
    resource.WithUnlimitedWait(),
    resource.WithProgressCallback(func(progress int) { fmt.Println(progress) }))
if err != nil { return err }
_ = node
```

`WithProgressCallback`은 초기 조회와 이후 각 비종료 응답의 `progress` 정수 필드를 읽습니다. 필드가 없거나 nil이면 0이며 값의 범위를 보정하지 않습니다. 완료·실패·HTTP 오류에서는 호출하지 않습니다. callback은 호출한 goroutine에서 동기적으로 실행되며, callback에서 context를 취소하면 추가 조회를 하지 않습니다. 서비스 생성 workflow는 `ValidateWaitOptionsFor[Model]`로 속성의 지원 여부까지 검사한 뒤 생성 요청을 보냅니다.

Wait는 없는 ID나 삭제된 리소스를 계속 기다리지 않고 조회 오류를 반환합니다. 성공 응답에서 리소스 객체가 nil이면 `ErrFailedState`입니다. WaitDeleted는 이미 없는 리소스·nil 결과·대소문자 무시 `deleted` 상태에도 성공하며, 인증 오류나 서버 오류를 삭제 완료로 처리하지 않습니다. 삭제 대기에도 callback과 속성 선택을 사용할 수 있습니다. flavor처럼 기본 상태가 없는 리소스는 Wait/WithStatus를 요청하면 `ErrUnsupported`를 반환하며, Wait에서 실제 문자열 속성을 명시적으로 선택할 수는 있습니다.

## SDK 내부 어댑터

`Adapter[T]`는 getter, pager, extractor, ID/name/status 접근 함수를 등록하는 concrete descriptor입니다. 새 서비스 구현은 원래 Gophercloud의 API 호출과 모델 매핑에 집중하고 이름 조회·중복 검사·대기 알고리즘을 재작성하지 않습니다.

`ValidateID`를 지정한 binding은 모든 조회·해석·삭제·대기에 같은 서비스 식별자 문법을 적용합니다. 생략하면 단일 URL path segment 검증을 사용합니다. `NameQueryKey`는 이름 검색의 서버 query 이름을 지정하며 기본값은 `name`입니다. `prefix` 검색을 쓰더라도 공통 계층은 정확한 이름 비교를 추가합니다. 애플리케이션이 이 설정을 구성하지는 않습니다.

Collection과 서비스 객체의 zero value는 사용하지 않습니다. `Connection`이 구성한 서비스에서 가져오는 것이 일반적인 사용 경로입니다. 응답 객체를 수정해도 자동으로 서버에 반영되지 않습니다. 변경은 서비스 API의 typed Update, 또는 부모 범위 객체의 Update에 전달합니다.

부모가 필요한 리소스는 [범위 객체](../docs/scoped-resources.md)를 사용합니다. `dns.RecordSets.InZone(ctx, resource.Name("example.org."))`처럼 부모를 한 번 해석하고, 반환된 객체의 Find/List/Delete/Wait와 Create/Update가 같은 부모를 사용합니다. 기본 식별자 검증은 빈 ID, `.`, `..`와 URL 경로·query 문자가 들어간 ID를 요청 전에 거부합니다.

테스트는 [collection_test.go](collection_test.go)와 [서비스 공통 통합 테스트](../collections_test.go)에 있습니다.

## 목록 제어 함수 예제

준비된 collection을 받아 첫 페이지에서 최대 5개의 raw 행을 소비하는 함수입니다.

```go
package example

import (
    "context"
    "fmt"

    "gophercloudsdk/resource"
)

func PrintFirstPage[T any](ctx context.Context, collection *resource.Collection[T]) error {
    for value, err := range collection.List(ctx,
        resource.WithMaxItems(5), resource.WithPaginated(false)) {
        if err != nil { return err }
        fmt.Println(value)
    }
    return nil
}
```
