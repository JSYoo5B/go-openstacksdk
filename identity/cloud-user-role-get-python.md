# Cloud User·Role getter: Python과 Go

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [get_user](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L344), [get_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1422), [Proxy find_user/find_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)입니다. Native 기반은 Gophercloud v2.15.0이며 이번 선언은 Cloud getter 두 개로 한정합니다.

**고정 Source의 get_user는 filtered search가 한 개여도 그 항목을 반환하지 않고 다시 find_user를 호출합니다.** Go도 그 순서를 보존합니다. get_role는 같은 filtered branch에서 실제 entities[0]을 반환하므로 둘의 결과가 다를 수 있습니다. SDK가 Library-owned descriptor/options/default/lookup/필터·오류 처리 순서를 제공하며 사용자 builder/normalizer interface를 요구하지 않습니다.

| Python | 전체 Connection | 이미 선택한 Identity 서비스 |
|---|---|---|
| `conn.get_user(name_or_id, domain_id=...)` | `conn.GetUserRecord(ctx, nameOrID, options...)` | `identity.NewUserReadCloud(svc.Users).GetUser(...)` |
| `conn.get_user(..., filters={})` | `WithUserGetFilters(json.RawMessage("{}"))` | search gate 후에도 closed find 수행 |
| `conn.get_role(name_or_id, domain_id=...)` | `conn.GetRoleRecord(ctx, nameOrID, options...)` | `identity.NewRoleReadCloud(svc.Roles).GetRole(...)` |
| `conn.get_role(..., filters={})` | `WithRoleGetFilters(json.RawMessage("{}"))` | search의 첫 결과 반환 |
| dictionary 또는 JMES filters | `WithUserGetFilters` / `WithRoleGetFilters`, `With*GetExpression` | 동일 concrete option |
| domain/transport/current location | `With*GetDomainID/DomainIDJSON`, `Headers/Microversion/Location/Opts` | 직접 factory는 location을 명시 전달 |

기존 native User/Role Find/Get, list/search cloud API와 private Source profiles는 유지합니다. 새 leaf `users.API.ResolveRecord`·`roles.API.ResolveRecord`의 `ClosedLookupOpts`는 domain_id와 explicit Go headers/MV/location만 제공합니다. 일반 named `find_user(**query)`·`find_role(**query)`에 존재하는 raw fetch/constructor/control 계약을 닫았다고 세지 않습니다.

## 권한과 인증

[Keystone 2026.1 User 정책 구현](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/user.html)은 user GET을 admin/system reader/대상 domain reader/본인으로 구분하며 LIST는 system 또는 domain reader 조건이 필요합니다. 그래서 본인을 member GET으로 읽을 수 있어도 deprecated search branch는 목록에서 403으로 끝날 수 있습니다. [Role 정책 구현](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/role.html)은 system reader 및 허용된 domain manager persona의 role 조회를 구분합니다(2026-10-13 확인). 대상 role 이름/도메인 조건, token scope와 배포 override는 서버가 판단합니다. SDK가 role을 부여하거나 client를 관리자 권한으로 바꾸지 않고 실제 403 및 후속 LIST 오류를 보존합니다.

OS_CLOUD/clouds.yaml 또는 OS_AUTH_URL과 사용자 인증 환경을 설정하고 조회 가능한 ID 또는 정확한 이름을 `IDENTITY_USER_REF`, `IDENTITY_ROLE_REF`에 지정합니다. `IDENTITY_DOMAIN_ID`는 선택이며 생략과 명시적 빈 문자열은 다릅니다. 아래 프로그램은 독립 main입니다. Python에는 openstacksdk, Go module에는 `github.com/JSYoo5B/go-openstacksdk`가 필요합니다. Go 예제를 main.go로 저장해 `go run .`으로 실행합니다. fixture/compile 성공을 실제 인증·live cloud 호출 성공으로 세지 않습니다.

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
    user = os.environ["IDENTITY_USER_REF"]
    role = os.environ["IDENTITY_ROLE_REF"]
    domain = os.getenv("IDENTITY_DOMAIN_ID")
    print_value(conn.get_user(user, domain_id=domain))
    print_value(conn.get_user(user, filters={}, domain_id=domain))
    # Pinned Source checks len([None]) then still calls find_user.
    print_value(conn.get_user(user, filters="`[null]`", domain_id=domain))
    print_value(conn.get_role(role, domain_id=domain))
    # Role returns entities[0], so this expression returns None.
    print_value(conn.get_role(role, filters="`[null]`", domain_id=domain))


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

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
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
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	user, role := os.Getenv("IDENTITY_USER_REF"), os.Getenv("IDENTITY_ROLE_REF")
	if user == "" || role == "" {
		log.Fatal("set IDENTITY_USER_REF and IDENTITY_ROLE_REF to readable IDs or names")
	}
	userOpts := []identity.UserGetOption{}
	roleOpts := []identity.RoleGetOption{}
	if domain, ok := os.LookupEnv("IDENTITY_DOMAIN_ID"); ok {
		userOpts = append(userOpts, identity.WithUserGetDomainID(domain))
		roleOpts = append(roleOpts, identity.WithRoleGetDomainID(domain))
	}
	directUser, err := conn.GetUserRecord(ctx, user, userOpts...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(directUser.Value)
	filteredUser, err := conn.GetUserRecord(ctx, user, append(userOpts,
		identity.WithUserGetFilters(json.RawMessage(`{}`)))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(filteredUser.Value)
	pinnedUser, err := conn.GetUserRecord(ctx, user, append(userOpts,
		identity.WithUserGetExpression("`[null]`"))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(pinnedUser.Value)
	directRole, err := conn.GetRoleRecord(ctx, role, roleOpts...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(directRole.Value)
	filteredRole, err := conn.GetRoleRecord(ctx, role, append(roleOpts,
		identity.WithRoleGetExpression("`[null]`"))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(filteredRole.Value)
}
```

직접 서비스 API를 사용하는 경우 `svc, err := conn.IdentityV3(ctx)` 뒤 `identity.NewUserReadCloud(svc.Users)`/`NewRoleReadCloud(svc.Roles)`를 생성합니다. 이 factory에는 Python Connection이 없어 기본 location=null입니다. `conn.CurrentLocation()`을 `WithUserGetLocation`/`WithRoleGetLocation`에 명시 전달할 수 있습니다. 위 Connection bridge는 selected v3 API와 immutable current location을 제공합니다. Root의 전체 Connection lifecycle을 완료했다고 주장하지 않습니다.

## None, falsey, raw domain

filters option 생략 또는 literal JSON null은 Source None입니다. `{}`·`[]`·`false`·`0`·빈 JSON string은 falsey여도 None이 아니므로 deprecated search branch에 들어갑니다. `DeprecatedFilters`는 그 분기를 나타내며 process-global Python warning을 대신 출력하지 않습니다. `With*GetExpression(expression)`은 JSON string encoding을 SDK에서 수행합니다.

`DomainID`는 RawMessage로 JSON null/false/0/empty string/array/object 같은 Source dynamic 값과 요청·constructor seed를 보존합니다. typed `With*GetDomainID`는 일반 string이고 raw `With*GetDomainIDJSON`은 copied JSON입니다. default nil domain은 두 closed find와 Role search에서 None으로 전달됩니다. Requests의 None은 query에서 빠지고 false/0/빈 string/repeated 값은 원래 인코딩 정책을 따릅니다.

User search만 domain의 Python-loaded truthiness를 확인해 falsey domain을 list kwargs에서 생략합니다. 그 뒤 find에는 같은 domain을 항상 전달하므로 filters={}와 domain=false는 `GET users?name=...` 다음 `GET users/{identity}?domain_id=False`입니다. Role search는 falsey domain도 넘기므로 domain=false의 목록에도 `domain_id=False`가 있습니다. User search는 nonempty non-UUID nameOrID를 server name으로 힌트하고 UUID-like ID면 생략합니다. Role search의 nameOrID는 server query가 아닌 local exact/glob 조건입니다.

filters는 search 결과가 모두 materialize된 뒤 ordered `_filter_list` 정책으로 평가합니다. falsey filter는 제한을 추가하지 않고 dictionary는 nested subset과 Source loaded equality, truthy string은 materialized row JMES입니다. unknown key/reached bad nested shape/invalid expression은 실제 목록 후 오류이고 앞선 mismatch에서 단락된 key는 방문하지 않습니다. 늦은 HTTP 오류가 local 필터 오류보다 먼저입니다. 중복 ID를 deduplicate하지 않습니다.

## get_user의 search→find 분기

검색 결과가 falsey이면 None, len>1이면 ambiguity, 길이 1이면 **index 0을 읽지 않고 원래 nameOrID/domain으로 다시 find_user**입니다. 검색에서 email 등으로 제한한 행과 최종 finder 행이 달라도 Source처럼 최종 행을 반환합니다. `[null]`, `[false]`, `[0]`, 길이 1 object 또는 Unicode string 표현식도 truthy/len=1이면 최종 find를 호출합니다. 따라서 `{a:1}` 결과가 Role처럼 integer-key error를 내지 않으며 `[null]`도 즉시 None이 되지 않습니다. truthy scalar number/bool은 len error이고 falsey 표현식은 finder를 실행하지 않습니다.

`UserGetResult.Value/User`는 최종 반환값, `Inventory`는 deprecated search에서 실제 읽은 행, `SearchValue/Expression`은 독립된 검색 결과입니다. `Lookup`은 최종 closed finder가 실제 성공한 record일 때만 존재합니다. final finder가 missing이면 search에서 한 항목이 있어도 Value=null입니다. final HTTP/physical/guard failure면 Value/User는 nil이며 관찰한 search Inventory를 유지합니다. search result를 lookup의 response 증거로 재사용하거나 합성하지 않습니다.

## get_role의 search→index 분기

filtered branch는 모든 목록을 읽고 truthiness→len→integer index 0 순서로 결과를 선택합니다. falsey 결과와 `[null]`은 None, `[false]`/`[0]`은 각각 false/0, 길이 1 string은 그 문자입니다. truthy scalar의 len, 길이 1 object의 integer key 0은 오류이며 len>1은 ambiguity입니다. 표현식 JSON에 RoleRecord나 response proof를 합성하지 않습니다. 일반 선택에서만 `RoleGetResult.Role`이 실제 Inventory의 detached clone입니다. filtered branch에는 마지막 finder를 추가하지 않습니다.

None branch는 domain-aware closed finder입니다. `RoleGetResult.Lookup`은 이 경로에서만 존재하며 실제 GET 또는 fallback에서 선택한 행의 receipt입니다. Value=null은 성공한 부재이고, 실패 결과 Value/Role는 nil입니다.

## 닫힌 lookup·응답·취소

두 `ResolveRecord`는 기존 library-owned User/Role descriptor/default/Source status/link/loaded equality/physical-stop profile을 복사해 새 engine에만 name query hint를 켭니다. member GET을 먼저 시도하고 clean400/403/404에서만 name/domain LIST fallback입니다. GET의 domain은 constructor seed와 일반 query이며 실제 response Body가 seed를 덮습니다. fallback은 ID-or-name exact match, 첫 match 후 late error 보존, 두 번째 match 즉시 ambiguity, missing은 nil입니다. global named Find **query 범위는 이 helper의 public opts에 없습니다.

Get options의 microversion override는 member GET만 바꾸고 fallback LIST는 원래 selected version입니다. search branch의 explicit MV는 모든 search 페이지에 적용됩니다. nullable descriptor/default/Boolean/dict/list coercion과 canonical/wire order는 기존 [User](v3/users/python_records.md)·[Role](v3/roles/python-records.md) 패턴을 재사용합니다. `Resource/Wire/Envelope/Header/StatusCode`를 구분하고 Clone은 원문 bytes/header까지 복사합니다. invalid/malformed accepted representation은 실제 receipt가 있는 Go 오류입니다.

client/source/header snapshot은 option callback 전에 캡처합니다. SDK With 함수만 사용하면 되며 opts callback을 직접 쓸 때도 단 한 번만 호출하고 mutation/cancel/option error 뒤 다음 callback을 중단합니다. caller context와 custom cancellation cause, private operation guard와 selected API/client 변경을 유지합니다. physical read/close/Content-Length fault가 섞인400/403/404는 name fallback하지 않고 actual accepted/native rejection·native401 ErrOriginal/ErrReauth·모든 cause를 보존하며 Retry/Reauth/Backoff callback을 실행하지 않습니다. clean 재인증/재시도는 live token과 frozen query/header/version을 유지합니다.

이 API는 concrete JSON/context/options와 typed carrier를 갖는 Go mapping입니다. Source의 mutable Resource identity/dirty state, optional cache, arbitrary Python class/session/Connection subclass, runtime object repr와 JMES Resource-type function 차이, process-global warnings, Python malformed-body no-op와 Go strict JSON response/UTF-8/header/route/pagination bounds는 명시적 차이입니다. shared SDK-R1/C1/S1 전체 과제와 일반 named find_user/find_role의 raw kwargs는 별도로 유지합니다. 이를 이 두 closed named getter의 unresolved 계약으로 재포장하거나 full supported parity로 확대하지 않습니다.

## 검증 근거

신규 테스트는 Cloud User9/Role8, leaf User7/Role7의 **31 named groups**입니다. leaf 각각6 status×3 physical mode, clean retry/renewal, nullable/raw domain seed/query, private guard/context, complete fallback/missing/duplicate/late failure, GET-only MV, descriptor/evidence clone, falsey/JMES·Source bug가 포함됩니다. 기존 공개 Gophercloud testhelper 기반 testcloud와 native transport/physical body observers를 재사용합니다. worker는 Go/Git을 실행하지 않으며 root focused/race/main/whole gate 결과는 아직 대기 상태입니다. 미실행 검사를 PASS로 기록하지 않습니다.

`/private/tmp/go-openstacksdk-cycle5-user-role-get-source-oracle.py`는 pinned Source AST Resource/list/query/pager/existing/consume/find/_get_one_match/field primitive converter/UserRole 선언, Cloud Get/Search/List/_filter_list/UUID를 CPython3.14에서 실행해 **79 observations PASS**를 기록했습니다. Source 파일 SHA-256과 결과는 같은 basename JSON receipt에 있습니다. HTTP/fetch/constructor/version/logger/warnings는 명시 stub이고 JMES literal output을 제공하여 실제 Source truth/len/index/search→find control을 검증합니다. fetch stub의 seed merge는 inspected _translate_response.attributes.update 동작의 모형이며 그 전체 method 실행 검증으로 주장하지 않습니다. list_users decorator는 제거했지만 exact getter가 전달하는 kwargs는 name/domain 두 선언뿐입니다. 설치 SDK/JMES parser/full live cloud 검증과 구분합니다.
