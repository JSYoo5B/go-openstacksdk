# Identity User 조회와 Cloud 목록·검색

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [User](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/user.py), [Proxy.users](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1429), [Cloud list_users/search_users](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L290)입니다. Native 기반은 Gophercloud v2.15.0입니다. 이 문서는 이 세 조회 선언만 다룹니다.

[Keystone의 현재 기본 정책](https://docs.openstack.org/keystone/2026.1/configuration/policy)에서 user 목록은 system reader 또는 대상 domain의 reader가 조회할 수 있습니다(2026-10-13 확인). 일반 project reader 권한만으로 가능한 목록이라고 가정하지 않습니다. SDK는 token/scope를 바꾸거나 admin client를 선택하지 않고 서버의 403·정책 판단을 보존합니다.

| Python | Go | 반환 |
|---|---|---|
| `conn.identity.users(**query)` | `svc.Users.ListRecords(ctx, options...)` | lazy `UserRecord` iterator |
| `list(conn.identity.users(**query))` | `svc.Users.AllRecords(ctx, options...)` | 전체 slice; 오류 시 부분 slice 폐기 |
| `conn.identity.users(jmespath_filters=...)` | `svc.Users.ProxyListRecords(ctx, WithRecordListJMESPath(...))` | arbitrary JSON 또는 unconsumed generator |
| `conn.list_users(domain_id=..., name=...)` | `identity.NewUserReadCloud(svc.Users).ListUsers(ctx, ...)` | `UserListResult.Value/Users/Inventory` |
| `conn.search_users(name_or_id=..., domain_id=..., filters=...)` | `UserReadCloud.SearchUsers(ctx, nameOrID, ...)` | `UserSearchResult.Value/Users/Inventory` |

기존 native `User`, `List`, `All`과 membership API는 그대로 사용할 수 있습니다. 새 record API는 Source의 descriptor/default·query/local filtering·pagination을 SDK가 소유하며 사용자 decoder나 builder interface를 요구하지 않습니다. `find_user`·비밀번호 변경·생성·수정·삭제를 이 작업의 완료 항목으로 세지 않습니다.

## 동일 인증으로 실행하는 예제

OS_CLOUD/clouds.yaml 또는 OS_AUTH_URL/사용자 인증 환경을 설정하고, 조회 가능한 domain ID를 `USER_QUERY_DOMAIN_ID`로 지정합니다. Python에는 openstacksdk를 설치하고 Go module에는 `github.com/JSYoo5B/go-openstacksdk`를 추가합니다. Go 예제를 `main.go`로 저장하면 `go run .`으로 실행할 수 있습니다.

```python
import json
import os
import openstack


def main():
    conn = openstack.connect()
    domain = os.environ["USER_QUERY_DOMAIN_ID"]
    for row in conn.identity.users(
        domain_id=domain, is_enabled=True, max_items=10
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    listed = conn.list_users(domain_id=domain)
    print(json.dumps([row.to_dict() for row in listed], indent=2))
    searched = conn.search_users(
        name_or_id="team*", domain_id=domain, filters="length(@)"
    )
    print(json.dumps(searched, indent=2))


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
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
)

func printJSON(value any) {
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
	domain := os.Getenv("USER_QUERY_DOMAIN_ID")
	if domain == "" {
		log.Fatal("set USER_QUERY_DOMAIN_ID to an allowed domain ID")
	}

	rows, err := svc.Users.AllRecords(ctx,
		users.WithRecordListQuery("domain_id", domain),
		users.WithRecordListQuery("is_enabled", true),
		users.WithRecordListMaxItems(10),
		users.WithRecordListLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		printJSON(row.Resource.Body)
	}

	cloud := identity.NewUserReadCloud(svc.Users)
	listed, err := cloud.ListUsers(ctx,
		identity.WithUserListDomainID(domain),
		identity.WithUserListLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(listed.Value)

	searched, err := cloud.SearchUsers(ctx, "team*",
		identity.WithUserSearchDomainID(domain),
		identity.WithUserSearchFilters(json.RawMessage(`"length(@)"`)),
		identity.WithUserSearchLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(searched.Value)
}
```

Go 예제는 선택한 `IdentityV3().Users`와 `identity.NewUserReadCloud` factory를 사용합니다. `CurrentLocation()` 결과를 명시적으로 제공하며 direct native-client API가 Connection의 구성 정보를 자동 추정한다는 의미는 아닙니다. `default_project_id`는 기본 인증 project 편의 정보일 뿐 사용자 리소스의 소유 project를 뜻하지 않으므로 location을 변경하지 않습니다.

## User descriptor와 Query

`UserRecord.Resource.Body`는 User의 12개 선언 Body와 계산된 `location`입니다. `default_project_id`, `description`, `domain_id`, `email`, `links`, `name`, `password`, `password_expires_at`, `id`는 nullable untyped이므로 명시적 false·숫자·문자열·null을 그대로 보존합니다. `password_expires_at`는 native time.Time parsing에 묶이지 않습니다.

`is_enabled`는 wire `enabled`를 Python JSON truthiness로 변환하며, null은 null입니다. 빈 문자열은 false, 문자열 `"false"`는 true, binary64로 underflow하는 float는 false입니다. `federated`는 nullable list이며 scalar를 한 개 배열로 감쌉니다. `options`는 dict, 누락 기본값은 `{}`, 명시한 null은 null, nonmapping 값은 `{}`입니다. Canonical 이름과 wire 이름이 함께 있으면 원래 응답 member 순서의 마지막 값이 선언 view를 결정합니다. 이 클래스는 unknown Body 보존을 선언하지 않으므로 extension은 `Wire`에만 남습니다.

| 분류 | Source 키 |
|---|---|
| server QueryParameters | limit, marker, domain_id, name, password_expires_at, unique_id, idp_id, protocol_id, is_enabled→enabled |
| local Body filter | default_project_id, description, email, links, password, federated, options, id |
| Header descriptor | 없음; page HTTP Header는 실제 evidence로 별도 보관 |
| unknown kwargs | Source가 버림; Go도 JSON marshaler 전에 버림 |

`WithRecordListQuery`는 JSON 값의 null/false/배열/누락을 보존합니다. Request query 배열은 반복 value, null은 생략, bool은 `True`/`False`로 전송합니다. canonical `is_enabled` 값이 wire `enabled` 키보다 우선합니다. Mapped server 조건을 local predicate로 다시 검사하지 않습니다. Local dict filter는 재귀 subset·actual truthiness와 순서에 따른 short circuit을 사용하며, equality는 CPython JSON 로드 결과의 float 반올림과 bool/숫자 동등성을 따릅니다. 누락 `options:{}`와 명시한 options:null의 필터 결과는 다릅니다.

`Wire`는 원래 JSON spelling/unknown 필드, `Envelope`는 전체 page, `Header`·`StatusCode`는 실제 HTTP 응답이며 각각 독립 복사입니다. `Clone()`도 독립적인 view·wire·receipt를 반환합니다. `self`는 Source 생성자 규칙대로 제거하고, cls/connection/microversion/_synchronized row collision 및 malformed plural은 실제 accepted page evidence를 가진 오류로 반환합니다. 로컬 JMESPath 결과나 preflight 오류에 HTTP receipt를 만들지 않습니다.

## 페이지·옵션·Proxy JMESPath

기본 pagination=true, limit 생략, max_items=0은 무제한입니다. `WithRecordListMaxItems`는 local filter 전 raw 행 수를 제한하고, limit 미지정 양수 cap은 wire limit hint가 됩니다. explicit positive limit과 nonempty page에서는 짧은 페이지도 마지막 raw wire ID marker로 재요청하고 빈 page에서 끝납니다. local filter에서 마지막 행을 제외해도 marker는 그 원래 행의 ID입니다. break·cap·single-page 뒤에는 다음 요청을 하지 않습니다.

Keystone root `links:{next:url}` dictionary는 고정 Source처럼 다음 링크로 인식하지 않습니다. root links 키가 있으면 users_links로 대신 찾지 않습니다. Source rel/href·top-level next 우선순위를 사용하고 valid RFC HTTP Link는 Go interoperability로 지원합니다(고정 Requests `uri`/`url` 불일치의 명시적 보완). foreign origin·collection·filter 변경·순환을 actual page receipt와 함께 거절합니다. typed BasePath는 같은 서비스의 안전한 상대 경로만 사용하며 pagination amount는 비음수 Go integer입니다.

헤더·microversion·base_path와 `__conflicting_attrs`의 one-pass restoration은 SDK가 처리합니다. Restored paginated/base_path의 bound-argument 중복과 session/class/route/auth/protocol 조작은 preflight 오류입니다. User에는 `_max_microversion`이 없으므로 class maximum discovery를 추가하지 않습니다. 선택한 native default 또는 `WithRecordListMicroversion`을 모든 page에 적용하며 explicit empty는 version header를 생략합니다. Header role은 identity입니다. 새 profile만 `SourceStatusPolicy`로 valid 100..399를 Source 성공 범위로 처리하고 기존 native/다른 reviewed API를 바꾸지 않습니다.

Source Proxy._list는 JMESPath를 **unconsumed generator**에 적용합니다. `[*].id`·`[0]`·`type(@)`는 HTTP0/null, `length(@)`·`to_string(@)`·sort_by는 HTTP0/type error입니다. `@`, not_null, to_array는 generator 자체/중첩 generator를 보존합니다. `ProxyListRecords.Value`는 이 polymorphic 결과이고 전체 값이 JSON일 때만 `JSON`을 채웁니다. typed `ListRecords`는 truthy expression을 `ProxyListRecords`로 보내라는 오류를 반환합니다. Go iter.Seq2 재순회는 캡처한 요청을 다시 실행하며 Python one-shot generator와 다릅니다.

## Cloud helper의 별도 규칙

`list_users`의 실제 valid_kwargs는 **domain_id/name 두 개뿐**입니다. `WithUserListQuery`도 이 두 키만 받아 임의 JSON 값을 보존합니다. email·enabled·filters·limit·jmespath_filters 등의 cloud kwargs는 Source decorator처럼 HTTP 전에 오류이며 invalid value의 marshaler를 호출하지 않습니다. 이 제한은 `Proxy.users`의 더 넓은 QueryParameters와 구별됩니다. `WithUserListHeaders/Microversion/Location`은 명시적인 Go transport/location 편의 옵션입니다.

`search_users`는 truthy non-UUID name_or_id를 server **name hint**로도 전송합니다. UUID의 uppercase hex·braces·urn/uuid prefix·hyphen normalization은 Source `_is_uuid_like`의 실제 규칙을 따릅니다. glob도 literal server name hint로 전송하므로 서버 필터가 먼저 결과를 제한할 수 있습니다. name/id exact 또는 case-sensitive glob matching은 목록을 다 읽은 뒤 적용합니다. Search filters는 server kwargs로 확장하지 않습니다.

Cloud Search는 materialized Source view에 local dict/JMESPath를 적용합니다. Truthy 문자열 JMESPath를 지원하고 scalar/null/배열/object JSON을 그대로 `Value`에 담으며 expression에는 가짜 User association을 추가하지 않습니다. Ordinary selection만 `Users`를 채우고 `Inventory`는 실제 읽은 rows만 유지합니다. enabled wire 이름은 local Resource membership에 없으므로 `is_enabled`를 사용합니다. Invalid expression/필터도 HTTP 목록 완료 뒤 평가하고, 늦은 페이지 HTTP 오류가 local expression 오류보다 먼저 반환됩니다.

Standalone API 기본 location은 null입니다. 명시적 `CloudLocation`은 snapshot/clone하며 User에 project_id/zone Body descriptor가 없으므로 unknown project_id와 default_project_id가 현재 configured scope를 바꾸지 않습니다. Cloud local location filter는 명시적 위치를 적용한 뒤 실행합니다. Source Connection의 자동 v3 selection, dynamic Resource/session/dirty/cache lifecycle과 모든 cloud configuration은 SDK-R1/C1/S1 계약으로 따로 추적합니다.

JMESPath의 exact-decimal 숫자 처리·lexical object key order·stricter malformed grammar·임의 Python object/function 타입과의 차이는 기존 common engine 정책입니다. Python dictionary insertion order를 포함한 모든 런타임 동작이 같다고 주장하지 않습니다. 옵션은 concrete JSON/context/선택된 client 범위이며 SDK-owned route, immutable copies와 Go error 결과를 사용합니다.

## 검증

`python_records_test.go`의 18 named tests와 physical fault matrix, `identity/user_record_cloud_test.go`의 12 named tests는 기존 Gophercloud/testcloud fixture를 재사용합니다. Descriptor defaults/casts/alias order, 실제 query 반복/null/bool, source page/marker/cap, option copy/formal/opaque generator, Source UUID/name hint와 cloud valid_kwargs, materialize-before-filter, late error/inventory, clone/location/source/cancel을 검증합니다. 200/203/401/404/429/503 Read·Close·mixed faults는 실제 proof/cause와 401 pointer ErrOriginal/ErrReauth를 보존하고 retry/reauth를 추가 실행하지 않습니다. Clean401/503은 캡처한 query/header/version과 live token을 유지합니다.

CPython3.14의 고정 Source AST-extracted query/list/pager/existing/consume_attrs·field converter·User declaration·UUID·Cloud method/valid_kwargs oracle 26개 관측도 비교했습니다. 이 oracle은 external session/constructor/version과 cloud selection collaborators를 stub하며 설치된 전체 SDK/JMESPath 통합 검증이라고 세지 않습니다. Root의 현재 branch Go gate와 독립 main compile 결과만 구현 검증 완료로 기록합니다. 실제 배포 cloud 또는 whole-SDK parity 완료를 뜻하지 않습니다.
