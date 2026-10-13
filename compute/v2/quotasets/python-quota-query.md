# Nova quota 조회: Python SDK와 Go

고정한 [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)의 `get_quota_set(project, usage=False, **query)`는 `scope.GetRecord(ctx, options...)`에 대응합니다. `InProject`가 project를 한 번 고정하고, query·응답 변환·버전 선택은 SDK가 맡습니다. 기존 `Get`·`Detail`은 Gophercloud 모델을 제공하며, 이 문서의 `GetRecord`는 Python의 공통 quota Resource 표현을 제공합니다.

| Python | Go |
|---|---|
| `get_quota_set(project)` | `scope.GetRecord(ctx)` |
| `get_quota_set(project, usage=True)` | `scope.GetRecord(ctx, quotasets.WithQuotaUsage(true))` |
| `get_quota_set(project, user_id=user)` | `userScope.GetRecord(ctx)` 또는 project scope의 `WithQuotaReadQuery("user_id", user)` |
| `get_quota_set(project, fields=["cores", "ram"])` | `WithQuotaReadQuery("fields", []string{"cores", "ram"})` |
| `get_quota_set(project, microversion="2.100")` | `WithQuotaReadMicroversion("2.100")` |
| `get_quota_set(project, skip_cache=True)` | `WithQuotaReadSkipCache(true)` |

```python
import openstack

conn = openstack.connect()
quota = conn.compute.get_quota_set(
    "project-id", usage=True, user_id="user-id", fields=["cores", "ram"]
)
print(quota.cores, quota.usage, quota.reservation)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    scope, err := conn.ProjectQuotas(ctx, resource.ID("project-id"))
    if err != nil { log.Fatal(err) }
    user, err := scope.InUser(ctx, resource.ID("user-id"))
    if err != nil { log.Fatal(err) }
    quota, err := user.GetRecord(ctx,
        quotasets.WithQuotaUsage(true),
        quotasets.WithQuotaReadQuery("fields", []string{"cores", "ram"}),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(quota.Resource.Body["cores"]))
    fmt.Println(string(quota.Resource.Body["usage"]), string(quota.Resource.Body["reservation"]))
    fmt.Println(quota.ProjectID, quota.UserID, quota.StatusCode)
}
```

## Query와 기본값

`WithQuotaReadQuery(key, value)`는 문자열·bool·number·null·list·object 등 JSON 값을 옵션 생성 시 독립적으로 복사합니다. 임의의 fetch query를 지원하며 같은 key를 다시 주면 마지막 값이 적용됩니다. list는 반복 key, bool은 `True`/`False`, null은 생략으로 Requests의 인코딩을 재사용합니다. `userScope`의 `user_id`는 고정되므로 query로 덮어쓸 수 없습니다. project scope에서는 `user_id`가 일반 query이며 사용자 이름 조회를 자동으로 수행하지 않습니다.

`usage`는 기본 false입니다. true이면 `/os-quota-sets/{project}/detail`, false이면 `/os-quota-sets/{project}`를 조회하며 `usage` 자체는 URL에 넣지 않습니다. query의 `usage`를 사용할 때도 Python JSON truthiness에 따라 같은 endpoint를 선택합니다. `WithQuotaReadOpts`는 query와 nullable control 전체를 복사하고, 뒤의 개별 옵션이 값을 바꿀 수 있습니다.

상속한 `fetch`의 `requires_id`·`microversion`·`error_message`·`skip_cache`·`resource_response_key`는 query로 보내지 않습니다. bool과 string을 받는 명시 옵션을 제공하며 raw query에서도 이 이름을 제어 인자로 해석합니다. `requires_id=True`는 생성 직후 ID가 없는 QuotaSet이므로 HTTP 전에 오류입니다. `base_path`는 Nova proxy가 이미 넘기는 인자여서 Python도 중복 kwargs 오류를 냅니다. Go는 null을 포함해 존재 자체를 HTTP 전에 거부합니다. `self`·`session`·`project`도 고정 인자와 충돌합니다.

`resource_response_key`는 common QuotaSet의 override가 실제로 사용하지 않는 인자입니다. Go도 이를 받아 query나 응답 envelope를 바꾸지 않습니다. `skip_cache`는 URL이나 응답에 영향을 주지 않습니다. Go는 quota 응답을 캐시하지 않고 매 호출 새 Resource와 HTTP 요청을 만듭니다. Python Adapter의 선택적 cache 설정을 재현하는 것은 공통 SDK cache 과제로 남습니다.

## 버전과 응답

클라이언트에 선택한 microversion이 있으면 그대로 사용합니다. 없으면 Nova가 광고한 범위를 공통 discovery로 읽고 QuotaSet ceiling `2.56` 안에서 선택합니다. `WithQuotaReadMicroversion`은 명시 값이므로 ceiling을 적용하지 않습니다. 빈 문자열은 버전 헤더와 discovery를 생략하는 명시 Go 옵션입니다. Discovery 응답은 `quota.Discovery`에 별도로 남으며 quota GET의 응답인 것처럼 합치지 않습니다. 공유 client의 Microversion은 바꾸지 않습니다.

`Resource.Body`는 common QuotaSet의 view입니다. 각 quota의 `limit`을 quota key에, `in_use`를 `usage` dictionary에, `reserved`를 `reservation` dictionary에 놓습니다. `allocated`만 있는 object는 limit 속성을 생성하지 않습니다. 미지의 quota에도 같은 변환을 적용하고 JSON 숫자 precision을 유지합니다. 선언된 15개 integer 필드는 Source descriptor의 int 변환을, `force`는 JSON truthiness를 적용합니다. bool은 Python int subclass처럼 integer 필드에서도 bool로 남고, 숫자 float은 정수로 절삭하며, digit-only 문자열은 integer로 바뀝니다. 부호·공백 문자열과 일반 list는 0입니다. 빠진 필드는 null이며 원본은 바꾸지 않습니다.

`Wire.Body`는 실제 quota 객체, `Envelope`는 전체 실제 응답 bytes입니다. `Header`·`StatusCode`와 두 view는 독립적인 값입니다. `self`는 Resource view에서 제거하되 Wire에서 보존합니다. URI `project_id`와 선택한 `microversion`은 서버가 확장 필드로 보낸 값보다 SDK가 고정한 사실을 우선합니다. `location`은 기본 null이며 `WithQuotaReadLocation(resource.CloudLocation{...})`으로 연결 metadata를 제공하면 공통 `ForResource`가 고정 project와 resource zone으로 계산합니다. 응답의 임의 location object는 Wire에 남습니다.

200–399를 accepted response로 처리하고 body 없는 204·non-JSON은 기본 Resource를 반환합니다. 파싱 가능한 bare object도 지원합니다. null·array quota object, Unicode digit 변환 실패, dictionary 순서에 따른 usage/reservation 충돌은 실제 응답 증거를 가진 `resource.ResponseError`로 반환합니다. 이러한 accepted 실패는 `QuotaRecord`의 actual metadata를 반환하면서 Resource는 nil로 둡니다. HTTP 404는 clean native 응답일 때만 `resource.ErrNotFound`가 추가됩니다. physical read/Close·취소·source 변경이 섞인 404는 부재로 분류하지 않습니다.

401의 native `*gophercloud.ErrUnableToReauthenticate`가 가지는 `ErrOriginal`·`ErrReauth`를 보존하고, fault가 있는 응답에 retry/reauth hook을 다시 실행하지 않습니다. clean 401 재인증과 clean 503 retry는 원래 Provider로 위임하되 method·route·query·version은 고정합니다.

## 권한과 남은 프로젝트 범위

Quota 조회 권한과 project/user 대상 검증은 Nova 정책을 따릅니다. 자기 project reader의 기본 quota 조회와 admin override를 구분하며 usage detail은 별도 stock policy를 확인해야 합니다. 실제 배포의 policy override를 SDK가 우회하거나 사용자 membership을 추정하지 않습니다. update/reset은 기존 관리 API이며 이번 조회 구현에서 확장하지 않았습니다.

Go는 JSON-domain Resource와 명시적인 `Decode`를 제공합니다. Python 객체의 임의 속성·메서드 name collision, Adapter cache·session 교체·dirty commit 전체를 재현하지 않습니다. Source가 임의 객체를 받는 control은 Go에서 string/bool control로 검증하며 invalid UTF8는 actual proof와 함께 거부합니다. [공통 Resource/cache/adapter 작업](../../../docs/sdk-support-ledger.md)은 전체 SDK 목표에 남아 있고, 이 조회를 서비스 전체 parity나 실클라우드 검증으로 집계하지 않습니다.

근거: [quota class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/quota_set.py), [공통 translator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py), [서비스 계약 테스트](python_quota_query_test.go), [공통 계약 테스트](../../../internal/quotafetch/read_test.go). 고정 Source 검토와 격리 HTTP 검증 범위를 표시하며 인증된 Nova 결과로 주장하지 않습니다.
