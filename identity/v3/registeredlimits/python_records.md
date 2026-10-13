# Keystone registered_limits: openstacksdk 비교

고정 비교 대상은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `Proxy.registered_limits` / `RegisteredLimit` / `Resource.list`이며, native 기반은 Gophercloud v2.15.0입니다. 기존 `List`·`All`은 native 타입 API로 유지합니다. `ListRecords`·`AllRecords`는 Source의 선언 속성, server/local 필터 분리와 페이지 규칙을 제공합니다. `ProxyListRecords`는 Source의 polymorphic JMESPath 반환을 제공합니다.

두 API의 [Keystone 기본 조회 정책](https://docs.openstack.org/keystone/latest/configuration/policy)은 `identity:list_limits`·`identity:list_registered_limits`가 빈 check string이고 system·domain·project scope를 허용합니다(2026-10-13 확인). 서버가 인증·scope와 운영 정책을 적용하며 이 SDK가 권한을 추정하거나 scope를 바꾸지 않습니다. 이 문서는 사용자 조회만 다룹니다.

## 실행 가능한 Python 예제

`OS_CLOUD`/clouds.yaml 또는 `OS_AUTH_URL`, `OS_USERNAME`, `OS_PASSWORD`, project/domain 인증 환경을 설정하고 openstacksdk를 설치합니다.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    for row in conn.identity.registered_limits(
        service_id="compute-service-id",
        resource_name="cores",
        description="team quota",
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go 예제

동일한 인증 환경을 사용합니다. 아래를 `main.go`로 저장한 Go module에서 `go get github.com/JSYoo5B/go-openstacksdk` 후 `go run .`을 실행합니다.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits"
)

func main() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	identity, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// service_id/resource_name use Keystone query parameters.
	// description is a declared Body filter, evaluated locally.
	rows, err := identity.RegisteredLimits.AllRecords(ctx,
		registeredlimits.WithRecordListQuery("service_id", "compute-service-id"),
		registeredlimits.WithRecordListQuery("resource_name", "cores"),
		registeredlimits.WithRecordListQuery("description", "team quota"),
		registeredlimits.WithRecordListMaxItems(100),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		encoded, err := json.MarshalIndent(row.Resource.Body, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", encoded)
	}
}
```

`AllRecords`는 전체 결과를 모으며 오류가 생기면 부분 slice를 버립니다. `ListRecords`는 호출 시 HTTP를 보내지 않고 순회할 때 요청합니다. iterator에서 이미 받은 record는 다음 페이지가 실패해도 호출자가 보관할 수 있고, `break`는 다음 요청을 중단합니다.

## Query와 Resource 속성

| 항목 | Python Source | Go 대응 |
|---|---|---|
| server query | `limit`, `marker`, `service_id`, `region_id`, `resource_name` | `WithRecordListQuery`; `WithRecordListLimit`/`Marker` |
| local Body filter | `description`, `links`, `default_limit`, `id`, `name` | 선언 속성에만 적용되는 `WithRecordListQuery` |
| unknown kwargs | API 요청과 필터에서 버림 | 알려지지 않은 값은 JSON marshaler 호출 전 버림 |
| 누락·명시적 null | 누락 descriptor는 None; 명시한 null은 그대로 | `Resource.Body`는 누락 null과 실제 JSON 값 보존 |
| default_limit | `Body('default_limit')`, **type=int 없음** | 숫자·문자열·false·null 그대로 보존 |
| Headers | 이 Resource에는 Header descriptor 없음 | page `Header`는 실제 HTTP evidence; `WithRecordListHeaders`는 실제 request 헤더 |
| location | Connection 없는 Resource는 None | 기본 null; `WithRecordListLocation(resource.CloudLocation)`는 명시적 Go 옵션 |

`links` dictionary 필터는 재귀 subset 비교입니다. 빈 actual mapping/falsey 값은 dictionary 필터와 일치하지 않으며, truthy scalar에 비어 있지 않은 dictionary 필터를 적용하면 page evidence와 함께 오류를 반환합니다. 필터는 입력 순서대로 short circuit 합니다. JSON float는 CPython 로드 결과의 비교를 사용하므로 큰 정수와 반올림된 float, boolean/숫자 동등성이 Source와 일치합니다. `description`, `default_limit` 등에 전달한 값을 요청 query로 변환하지 않습니다.

`Resource`는 선언된 Body + 계산된 location만 담습니다. `Wire`는 unknown 응답 필드를 포함한 원래 행, `Envelope`는 원래 전체 page, `Header`·`StatusCode`는 실제 response를 독립적으로 보관합니다. 서버가 보낸 `location`은 `Wire`에만 남습니다. `self`는 Source 생성자 규칙에 따라 declared view에서 제거되고, `cls`·`connection`·`microversion`·`_synchronized`와 충돌하는 row는 receipt를 가진 오류로 끝납니다.

명시적 location은 옵션 생성·적용 시 복제합니다. RegisteredLimit에는 `project_id` 선언이 없으므로 unknown 응답 project_id는 location을 바꾸지 않습니다. 명시한 현재 project 정보를 유지합니다. Python Connection에서 자동 계산하는 cloud/region/configured project 이름을 standalone native client에서 추측하지 않습니다. Connection의 동적 session/Resource 동작은 전체 SDK의 별도 계약입니다.

## Source 페이지와 옵션

기본은 paginated=true, 무제한 max_items=0, limit 생략입니다. `WithRecordListPaginated(false)`는 한 페이지만 읽고 `WithRecordListMaxItems(n)`은 **필터 적용 전 raw row 수**를 제한합니다. limit을 주지 않은 양수 max_items는 서버 limit hint가 됩니다. explicit limit=0은 자체 marker continuation을 켜지 않습니다.

고정 Source는 Keystone `{"links":{"next":"..."}}`를 다음 링크로 인식하지 않습니다. 이 형태를 그대로 무시하고, root `links` 키가 있으면 `registered_limits_links`로 바꾸어 찾지 않습니다. top-level `next`, Source rel/href 배열과 HTTP Link는 Source 우선순위에 따라 사용합니다. explicit 양수 limit과 nonempty page이면 짧은 페이지도 **마지막 raw wire ID**를 marker로 재요청하여 빈 페이지에서 끝냅니다. local filter에서 마지막 행을 제외해도 그 marker를 사용합니다.

Go는 origin과 collection path를 고정하고 foreign URL·순환·잘못된 marker를 실제 page evidence와 함께 거절합니다. `WithRecordListBasePath`는 같은 서비스의 안전한 상대 경로를 명시적으로 선택할 수 있습니다. URL/token/session을 임의로 바꾸는 옵션은 제공하지 않습니다. pagination amount는 비음수 Go integer로 검증하며 Python의 특이한 음수/비정수 인자 동작은 재현하지 않습니다.

이 Source 클래스는 `_max_microversion=None`이고 list에서 discovery를 요구하지 않습니다. 기본 version은 선택한 native client 설정을 사용합니다. `WithRecordListMicroversion`은 모든 페이지에 동일하게 적용하고 빈 문자열은 version header를 생략합니다. 실제 헤더는 `identity <version>`이며 다른 서비스 version header와 인증/route/protocol 헤더 조작을 거절합니다. 일반 `headers`·`microversion`·`base_path` kwargs도 동일한 formal option으로 처리합니다. `__conflicting_attrs`는 Source처럼 한 번 복원하며, 복원된 paginated/base_path가 이미 bound된 인자와 중복하면 HTTP 전에 거절합니다. 알 수 없는 일반 Resource kwargs는 버립니다.

## Source generator 대상 JMESPath

Python `Proxy._list`는 JMESPath를 아직 순회하지 않은 Resource generator에 적용합니다. 따라서 `[*].id`·`[0]`·`type(@)`는 HTTP 없이 null이고, `length(@)`·`to_string(@)`·`sort_by(@,&id)`는 HTTP 없이 type 오류입니다. `@`·`not_null(@,[] literal)`·`to_array(@)`는 lazy generator 자체 또는 generator가 들어 있는 반환값을 유지합니다.

```go
// 표현식은 실제 행 배열이 아닌 Source generator에 적용됩니다.
result, err := identity.RegisteredLimits.ProxyListRecords(ctx,
    registeredlimits.WithRecordListJMESPath("[*].id"),
)
if err != nil { log.Fatal(err) }
fmt.Println(string(result.JSON)) // null; HTTP 0회

result, err = identity.RegisteredLimits.ProxyListRecords(ctx)
if err != nil { log.Fatal(err) }
generator := result.Value.(*registeredlimits.RecordListGenerator)
for row, err := range generator.Records {
    if err != nil { log.Fatal(err) }
    fmt.Println(string(row.Resource.Body["id"]))
}
```

표현식이 없거나 falsey/non-string이면 lazy generator를 유지합니다. `RecordProxyListResult.JSON`은 값 전체가 JSON일 때만 채워집니다. generator 또는 nested generator를 강제로 JSON 직렬화하지 않습니다. Go `iter.Seq2`를 다시 순회하면 캡처한 요청을 다시 실행하며, Python generator의 한 번 소비되는 상태를 복제하지 않습니다. `ListRecords`는 typed 행 반환을 유지하므로 truthy string JMESPath에는 `ProxyListRecords`를 사용하라는 오류를 반환합니다. Go의 error/functional-option/고정 경로와 이 명시적 return-type 구분은 Python 객체 API와의 문서화된 차이입니다.

## 검증과 오류

기존 `internal/testcloud` 및 package native Gophercloud fixture를 재사용한 `python_records_test.go`는 query/default/local predicate, links-dictionary 무시, raw marker/max_items, null/string/false 타입, option snapshot/formal restore, opaque JMESPath, explicit location, nil/context/선택 source guard를 검증합니다. 200·203·401·404·429·503의 Read/Close/mixed 오류 matrix와 clean 401 재인증·503 재시도도 검증합니다.

accepted 오류는 실제 `resource.ResponseError` body/header/status를 보존합니다. rejected 오류는 native `ErrUnexpectedResponseCode` proof와 causes를 유지하고, 401은 `*gophercloud.ErrUnableToReauthenticate`의 `ErrOriginal`·`ErrReauth`도 유지합니다. physical fault나 취소 이후에는 Retry/Reauth callback과 추가 HTTP를 실행하지 않습니다. option 취소·Close 취소는 `context.Canceled`와 caller cause를 모두 유지합니다. 실패 없이 재시도할 수 있는 clean rejection은 고정 query/header/version을 유지해 처리합니다.

이 검증은 mock transport 및 고정 Source의 실제 AST-extracted list/query/pager oracle에 대한 검증입니다. 실제 배포 cloud의 전체 통합 테스트나 Connection-wide 모든 Resource 계약을 완료했다고 의미하지 않습니다.
