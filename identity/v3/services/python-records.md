# Keystone Service 목록·검색·조회

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/service.py), [Proxy.services/find_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1274), [Cloud search_services/get_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L600)입니다. Native 기반은 Gophercloud v2.15.0입니다. 이번 검토 선언은 Proxy 목록, Cloud 검색, Cloud 조회의 세 개입니다.

[Keystone 2026.1 기본 정책](https://docs.openstack.org/keystone/2026.1/configuration/policy.html)에서 서비스 GET/LIST는 admin 또는 system scope의 reader에게 허용됩니다. [정책 구현](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/service.html)은 scope 종류 system/project와 실제 권한 조건을 구분합니다(2026-10-13 확인). 일반 project reader만으로 가능한 호출이라고 가정하지 않습니다. 이 레이어는 token/scope를 바꾸거나 관리자 client를 자동 선택하지 않고 403과 실제 서버 정책을 보존합니다.

| Python | Go | 결과 |
|---|---|---|
| `conn.identity.services(**query)` | `svc.Services.ListRecords(ctx, options...)` | lazy `ServiceRecord` iterator |
| `list(conn.identity.services(**query))` | `svc.Services.AllRecords(ctx, options...)` | 전체 slice; 실패하면 부분 slice 폐기 |
| `conn.identity.services(jmespath_filters=...)` | `svc.Services.ProxyListRecords(ctx, ...)` | arbitrary JSON 또는 unconsumed generator |
| `conn.search_services(name_or_id, filters)` | `identity.NewServiceReadCloud(svc.Services).SearchServices(ctx, nameOrID, ...)` | `Value`, `Services`, 실제 `Inventory` |
| `conn.get_service(name_or_id)` | `ServiceReadCloud.GetService(ctx, nameOrID)` | GET 우선 조회, 없으면 JSON null |
| `conn.get_service(name_or_id, filters={})` | `GetService(ctx, nameOrID, WithServiceReadFilters(json.RawMessage("{}")))` | 목록 전체 검색 후 하나를 선택 |

기존 native `Service`, `List`, `All`, `Find`, 생성·수정·삭제의 계약은 유지됩니다. 새 API는 descriptor/default/query/pagination과 검색을 SDK가 소유하며 사용자 decoder나 builder interface를 요구하지 않습니다. named `find_service`·`list_services`나 쓰기 선언을 이번 신규 완료 수에 추가하지 않습니다. `ResolveRecord`는 Cloud 기본 조회에 필요한 닫힌 finder의 공개 Go helper입니다.

## 동일 인증으로 실행하는 예제

OS_CLOUD/clouds.yaml 또는 OS_AUTH_URL과 사용자 인증 환경을 설정합니다. 읽을 수 있는 서비스 ID 또는 정확한 이름을 `IDENTITY_SERVICE_ID`에 지정합니다. Python에는 openstacksdk를 설치하고 Go module에는 `github.com/JSYoo5B/go-openstacksdk`를 추가합니다. Go 코드를 `main.go`로 저장하면 `go run .`으로 실행할 수 있습니다. 둘 다 독립 main이며 Live cloud 성공을 검증했다고 주장하지 않습니다.

```python
import json
import os
import openstack


def print_value(value):
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, indent=2, ensure_ascii=False))


def main():
    conn = openstack.connect()
    identifier = os.environ["IDENTITY_SERVICE_ID"]
    for row in conn.identity.services(
        type="compute", is_enabled=True, max_items=5
    ):
        print_value(row)
    print_value(conn.identity.services(jmespath_filters="[*].id"))
    print_value(conn.search_services(name_or_id="n*", filters="length(@)"))
    print_value(conn.get_service(identifier))
    print_value(conn.get_service(identifier, filters={}))


if __name__ == "__main__":
    main()
```

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
)

func printValue(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", encoded)
}
func main() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	svc, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	identifier := os.Getenv("IDENTITY_SERVICE_ID")
	if identifier == "" {
		log.Fatal("set IDENTITY_SERVICE_ID to a readable service ID or name")
	}
	rows, err := svc.Services.AllRecords(ctx,
		services.WithRecordListQuery("type", "compute"),
		services.WithRecordListQuery("is_enabled", true),
		services.WithRecordListMaxItems(5),
		services.WithRecordListLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		printValue(row.Resource)
	}
	proxy, err := svc.Services.ProxyListRecords(ctx, services.WithRecordListJMESPath("[*].id"))
	if err != nil {
		log.Fatal(err)
	}
	printValue(proxy.JSON)
	cloud := identity.NewServiceReadCloud(svc.Services)
	found, err := cloud.SearchServices(ctx, "n*",
		identity.WithServiceReadFilters(json.RawMessage(`"length(@)"`)),
		identity.WithServiceReadLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	printValue(found.Value)
	direct, err := cloud.GetService(ctx, identifier, identity.WithServiceReadLocation(location))
	if err != nil {
		log.Fatal(err)
	}
	printValue(direct.Value)
	searched, err := cloud.GetService(ctx, identifier,
		identity.WithServiceReadFilters(json.RawMessage(`{}`)),
		identity.WithServiceReadLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	printValue(searched.Value)
}
```

## SDK가 소유하는 목록 계약

`ServiceRecord.Resource`는 Source의 선언된 속성과 computed location을 담습니다. `Wire`, `Envelope`, `Header`, `StatusCode`는 실제 응답 증거이고 서로 독립적으로 복사됩니다. `Clone()`은 JSON bytes와 header까지 복사합니다. Service의 Body는 nullable `id`, `name`, `description`, `type`, `links`, `is_enabled`입니다. `is_enabled`만 Source bool 변환을 적용하므로 문자열 `"false"`는 true입니다. 같은 응답에 `enabled`와 `is_enabled`가 있으면 실제 member 순서의 마지막 값이 이깁니다. 다른 Body는 untyped JSON을 유지하고 서비스 type을 enum으로 제한하지 않습니다. 모르는 Body는 논리 view에서 빠지고 원문 Wire에는 남습니다.

서버 query는 `name`, `type`과 상속된 `limit`, `marker`입니다. `WithRecordListQuery("is_enabled", false)`는 페이지에서 bool 속성을 비교하는 local filter입니다. `enabled`라는 wire spelling은 서버 query로 선언되지 않았고 클래스 속성명도 아니므로 Source처럼 discard합니다. `description`, `links`, `id`도 local filter이며 nested dictionary는 순서대로 subset을 검사합니다. Source의 JSON-loaded float equality/truthiness를 사용합니다. `{}`는 truthy scalar에도 맞고 falsey/missing actual에는 맞지 않습니다. `WithRecordListQuery`는 kwargs 순서를 기록하고 unknown을 marshal 전에 버립니다. `WithRecordListOpts`의 Go map에는 Python 삽입 순서가 없으므로 lexical 순서로 고정합니다.

기본은 lazy, paginated=true, limit 생략, max_items=0(unbounded)입니다. Source는 root `links` key의 존재를 우선하고 dictionary의 next를 무시합니다. 양수 limit이 있으면 짧은 nonempty 페이지에서도 마지막 원문 row ID로 marker를 만들어 빈 페이지까지 진행합니다. local filter 결과 대신 원문 행을 세어 max_items에 적용하고, 초기 limit 생략 시 cap을 서버 hint로 보냅니다. 단일 plural object를 한 행으로 받을 수 있습니다. break는 다음 페이지를 미리 읽지 않습니다. malformed row, constructor collision(cls/connection/microversion/_synchronized), missing plural와 204 빈 representation은 실제 응답의 status/header/body를 가진 오류입니다.

`WithRecordListHeaders`, `Microversion`, `BasePath`, `Location`은 immutable option입니다. raw formal headers/microversion/base_path와 `__conflicting_attrs`의 한 번만 복원하는 Source 분기도 처리합니다. cls/session/resource_type/self는 SDK가 소유하는 runtime control이라 명시적 오류입니다. callback을 직접 작성할 필요 없이 제공된 With 함수만 사용하면 됩니다. 실제 Source의 kwarg 자체에 표현할 수 없는 Go raw JSON 문법 오류와 route/header 보호는 concrete Go 오류로 구분합니다.

Proxy의 truthy string JMES는 HTTP 전에 unconsumed generator에 적용됩니다. `[*].id`, `[0]`, `type(@)`는 일반적으로 null, 상수는 그 JSON, `@`와 `not_null(@, ...)`는 같은 generator, `to_array(@)`는 generator를 포함한 배열이 됩니다. `RecordProxyListResult.JSON`은 전체가 JSON일 때만 있으며 generator와 nested generator에는 nil입니다. `Value`가 `*RecordListGenerator`이면 `Records`를 순회할 때 실제 요청을 합니다. 생략/falsey/nonstring JMES는 lazy generator를 유지합니다. typed `ListRecords`에 truthy string JMES를 주면 반환형이 달라지므로 `ProxyListRecords` 사용 오류를 돌려줍니다. Python opaque generator의 repr, 특히 `to_string(@)`의 Python 문자열화는 Go에서 보존하지 않고 직렬화 오류로 제한합니다. 공용 Go JMES 문법/숫자 정책과 이 opaque 경계는 문서화된 Go 차이입니다.

## Cloud의 필터·조회 분기

`SearchServices`는 queryless `/services` 목록 전체를 먼저 읽은 뒤 name/ID exact 또는 glob와 local filter를 적용합니다. `name_or_id`를 server name query로 넘기지 않습니다. null/missing 값의 Python string 비교(`None`)와 중복 row를 보존합니다. falsey filter는 아무 제한도 추가하지 않고, truthy dictionary는 ordered nested subset, truthy string은 materialized Source views에 대한 JMES입니다. filter key가 선언된 view에 없거나 reached nested actual이 잘못된 모양이면 목록 완료 후 오류입니다. 앞선 mismatch에서 단락된 잘못된 filter는 평가하지 않습니다. 이후 페이지의 HTTP 오류가 local expression 오류보다 먼저입니다.

`ServiceSearchResult.Services`는 일반 선택 결과만 갖고 `Inventory`와 독립적으로 복사됩니다. 표현식은 number/string/object/array/null을 반환할 수 있어 `Expression=true`, `Value`에 JSON을 넣고 `Services=nil`입니다. 표현식 결과를 실제 HTTP 행과 억지로 연결하지 않습니다. 실패하면 `Value/Services`를 만들지 않고 관찰된 이전 페이지 `Inventory`를 유지합니다.

`GetService`는 filters 생략 또는 literal JSON null이면 닫힌 Proxy.find_service 경로입니다. member GET은 모든 이름/ID에 먼저 시도하고 clean 400/403/404만 name query fallback으로 이어집니다. 그 fallback은 ID-or-name exact 비교를 하며 glob를 적용하지 않습니다. 첫 match 후의 늦은 HTTP 오류를 보존하고, 두 번째 match에 도달하면 즉시 ambiguity 오류입니다. missing은 nil record와 JSON null입니다. per-call microversion override는 GET에만 적용되고 fallback LIST는 선택된 client의 버전을 사용합니다. physical 읽기/닫기·guard·취소 오류가 섞인 400/403/404는 fallback하지 않습니다. 공개 helper는 raw **fetch kwargs를 노출하지 않아 named Find의 일반 kwargs gap이 이 Cloud 호출에서 도달하지 않습니다.

filters=`{}`/`[]`/`false`/`0`/`""`도 None과 달라 deprecated search 경로입니다. 전체 목록을 검색한 뒤 결과가 falsey이면 None, len>1이면 ambiguity, 길이 1이면 integer index 0을 읽는 실제 Source 분기를 보존합니다. 표현식의 truthy number/bool은 len 오류, 길이 1 object는 integer-key lookup 오류, 길이 1 string은 그 Unicode 문자가 결과입니다. `ServiceGetResult.Service`는 실제 nonexpression Service 선택에만 있고, 표현식 결과는 `Value`와 실제 `Inventory`를 분리합니다. Python deprecation warning 출력 대신 `DeprecatedFilters`로 이 분기를 표현합니다.

선택된 client와 Source/route/options를 HTTP 전에 캡처하고 caller context, custom cause, owner drift 및 private operation guard를 처리합니다. opts callback은 한 번만 실행하고 source mutation/cancel 뒤의 callback을 중단합니다. Source profile의 유효 HTTP100..399 성공 범위는 새 record helper에만 적용합니다. 실제 read/close fault는 native 오류와 cause, 원문 status/header/body를 보존하며 physical fault에서 Retry/Reauth/Backoff callback을 실행하지 않습니다. clean 재인증은 live token을 반영하면서 선택된 query/header/MV를 유지합니다.

직접 `services.New(client)` 또는 `NewServiceReadCloud(api)`는 Python Connection이 없어 기본 location=null입니다. 위 Go main은 `conn.CurrentLocation()`의 명시적 facts를 immutable option으로 전달합니다. Service에는 project/zone descriptor가 없으므로 unknown remote project/location은 scope를 바꾸지 않습니다. Connection의 `SearchServiceRecords(ctx, nameOrID, options...)`와 `GetServiceRecord(ctx, nameOrID, options...)` bridge는 선택된 v3 Service API와 location을 연결합니다. 직접 factory 예제는 그 location 전달을 명시합니다. 전체 token/Connection lifecycle이나 SDK-R1/C1/S1 전부 완료를 주장하지 않습니다. Go iter.Seq2는 다시 순회하면 같은 캡처 요청을 반복할 수 있습니다. 보호된 route/header, stricter UTF-8/nonnegative pagination, cycle detection 및 HTTP Link 해석은 Go 계약 차이입니다.

## 검증 근거와 범위

`python_records_test.go`의 16 named tests와 `identity/service_record_cloud_test.go`의 14 named tests가 transport/header/query/projection/default/alias/local filter/pagination/opaque result/closed find/deprecated get/late failure/immutable option/context/actual receipt를 검증합니다. physical fault matrix는 8 status×4 mode이며 기존 공개 Gophercloud testhelper를 사용하는 `testcloud`와 같은 패키지 native fixture/transport를 재사용합니다. Root의 services leaf gate는 PASS이며, Cloud/race 및 독립 main compile은 검증 대기 상태입니다. 전체 PASS라고 기재하지 않습니다.

`/private/tmp/go-openstacksdk-cycle4-services-source-oracle.py`는 고정 Source의 AST Resource list/query/pager/existing/consume/closed find/_get_one_match, Service 선언과 primitive converter, Cloud methods/_filter_list를 CPython3.14에서 실행해 41 observations PASS를 기록했습니다. Source 파일 SHA-256과 결과는 같은 basename의 JSON receipt에 있습니다. HTTP/fetch/Resource constructor/version/logger/warnings는 stub이고 JMES literal output을 제공하여 실제 Python truth/len/index 분기를 검증했습니다. 설치된 SDK 전체나 독립 JMES parser/runtime/live cloud 검증으로 확대하지 않습니다. 이 선언의 Go mapping은 concrete Go 반환형과 차이를 갖는 API 대응이며 전체 SDK supported 판정이 아닙니다.
