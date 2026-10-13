# Cinder quota 조회: Python SDK와 Go

고정한 [block storage proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)의 `get_quota_set(project, usage=False, **query)`는 `scope.GetRecord(ctx, options...)`에 대응합니다. `InProject`가 project를 한 번 고정하고 쿼리·예약값·사용량과 descriptor 변환은 SDK가 맡습니다. 기존 `Get`·`Usage`는 Gophercloud 모델을 유지합니다.

| Python | Go |
|---|---|
| `get_quota_set(project)` | `scope.GetRecord(ctx)` |
| `get_quota_set(project, usage=True)` | `scope.GetRecord(ctx, quotasets.WithQuotaUsage(true))` |
| `get_quota_set(project, fields=["volumes", "gigabytes"])` | `WithQuotaReadQuery("fields", []string{"volumes", "gigabytes"})` |
| `get_quota_set(project, base_path="/os-quota-sets/%(project_id)s/defaults")` | `WithQuotaReadBasePath("/os-quota-sets/%(project_id)s/defaults")` |
| `get_quota_set(project, microversion="3.100")` | `WithQuotaReadMicroversion("3.100")` |
| `get_quota_set(project, skip_cache=True)` | `WithQuotaReadSkipCache(true)` |

```python
import openstack

conn = openstack.connect()
quota = conn.block_storage.get_quota_set(
    "project-id", usage=True, fields=["volumes", "gigabytes"]
)
print(quota.volumes, quota.usage, quota.reservation)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    scope, err := conn.BlockStorageProjectQuotas(ctx, resource.ID("project-id"))
    if err != nil { log.Fatal(err) }
    quota, err := scope.GetRecord(ctx,
        quotasets.WithQuotaUsage(true),
        quotasets.WithQuotaReadQuery("fields", []string{"volumes", "gigabytes"}),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(quota.Resource.Body["volumes"]))
    fmt.Println(string(quota.Resource.Body["usage"]), string(quota.Resource.Body["reservation"]))
    fmt.Println(quota.ProjectID, quota.StatusCode)
}
```

## Query와 fetch control

`usage`는 기본 false이며 일반 조회에서도 `usage=False`를 전송합니다. true이면 같은 `/os-quota-sets/{project}` endpoint에 `usage=True`를 붙입니다. 기존 native `Get`의 usage 생략과 `Usage`의 소문자 true와 구별됩니다. Query 순서는 Go `url.Values`의 key 정렬이며, 같은 key 값의 내부 순서는 유지합니다.

`WithQuotaReadQuery(key, value)`는 임의의 JSON-domain fetch query를 지원합니다. 옵션 생성 시 입력을 복사하고 같은 key를 다시 주면 마지막 값을 사용합니다. Requests 인코딩을 재사용하여 list는 반복 key, bool은 `True`/`False`, null은 생략합니다. 추가 query를 SDK가 서비스 policy로 추측해 삭제하지 않으며 Cinder가 지원하지 않는 query를 보낸 결과도 그대로 관찰합니다. Raw `usage`는 Source처럼 named usage 인자의 literal JSON 값을 Requests에 넘깁니다. 문자열 `"False"`는 그대로 `usage=False`, 숫자 0은 `usage=0`, null이나 빈 list는 usage key 생략이 됩니다. Nova가 endpoint를 고르기 위해 usage truthiness를 쓰는 동작과 다릅니다.

`requires_id`·`base_path`·`microversion`·`error_message`·`skip_cache`·`resource_response_key`는 상속된 `fetch` control이므로 URL query에서 소비합니다. 각각 `WithQuotaReadRequiresID`, `WithQuotaReadBasePath`, `WithQuotaReadMicroversion`, `WithQuotaReadErrorMessage`, `WithQuotaReadSkipCache`, `WithQuotaReadResponseKey`의 구체적 옵션을 제공합니다. `requires_id=True`는 fresh QuotaSet에 ID가 없어 HTTP 전에 실패합니다. Raw control의 null은 Python default 동작으로 돌아갑니다. `self`·`session`·`project`는 이미 고정된 인자와 충돌하므로 거부합니다. nil 옵션·invalid JSON·UTF8·control 타입 오류도 HTTP 전에 거부합니다.

`base_path`는 effective Cinder service base 안에서 해석하며 `%(project_id)s`는 고정 project로 바꿉니다. reverse proxy prefix를 유지하고 다른 origin·query·fragment로 route가 넓어지는 override는 거부합니다. Source의 임의 Adapter URL/session 교체를 제공하지 않는 명시 Go 정책입니다. `resource_response_key`는 common QuotaSet override가 무시하는 인자여서 Go도 envelope 선택을 바꾸지 않습니다.

매 호출은 새 quota record와 HTTP 조회를 만들고, 응답 캐시를 사용하지 않습니다. `skip_cache`는 query로 보내거나 결과 view에 추가하지 않는 control입니다. Python의 선택적 Adapter cache를 재현하는 공통 SDK cache 기능은 별도 작업으로 남습니다.

## 버전과 두 응답 view

Cinder QuotaSet에는 `_max_microversion`이 없으므로 기본 조회에서 version discovery를 수행하지 않습니다. 클라이언트 microversion이 있으면 그대로 사용하며, `WithQuotaReadMicroversion`의 명시 값이 이를 대체합니다. 빈 문자열은 version 헤더를 생략하는 명시 Go 옵션입니다. 기존 공유 client의 endpoint·microversion을 변경하지 않습니다.

`Resource.Body`는 Python common QuotaSet view입니다. quota dictionary에서 `limit`은 quota 이름의 속성, `in_use`는 `usage`, `reserved`는 `reservation` dictionary로 옮깁니다. `allocated`는 literal `Wire.Body`에 보존하며, Source처럼 allocated만 있는 dictionary는 limit 속성을 만들지 않습니다. 미지의 volume-type quota 이름에도 같은 normalization을 적용합니다.

일곱 known integer descriptor는 숫자·digit-only 문자열을 변환합니다. bool은 Python int subclass처럼 bool로 남고 float은 정수로 절삭하며 부호/공백 문자열·일반 list는 0입니다. null과 누락을 구분해 처리하고 빠진 known field는 view에서 null입니다. 큰 정수는 exact JSON bytes를 유지합니다. nested usage와 reservation 자체의 값은 추가 int 변환 없이 원본 JSON 값을 유지합니다.

`Wire.Body`는 실제 quota object, `Envelope`는 전체 응답 bytes, `Header`와 `StatusCode`는 실제 응답 증거입니다. 각 view와 metadata는 독립적으로 복사됩니다. `self`는 normalized view에서 지우되 Wire에서 보존합니다. 응답 body의 `project_id`가 있으면 view는 URI의 고정 project를 보여주고 Wire는 서버 값을 보존합니다. `location`은 기본 null이며 `WithQuotaReadLocation(resource.CloudLocation{...})`에 제공한 connection metadata로 공통 `ForResource`를 사용해 project/zone을 계산합니다. 서버가 임의로 보내는 location object는 Wire에만 남습니다.

## 오류·권한과 검증 범위

200–399를 accepted response로 처리합니다. body 없는 204와 non-JSON 응답은 기본 Resource를 반환하고 actual bytes는 Envelope에 남깁니다. quota_set envelope가 없는 bare object도 Source처럼 처리합니다. 파싱 가능한 null·array quota object, integer descriptor 변환 실패, ordered usage/reservation 충돌은 actual status/body/header를 가진 `resource.ResponseError`입니다. accepted read/Close 실패도 actual metadata만 있는 QuotaRecord와 오류를 돌려주며 Resource 성공을 합성하지 않습니다.

clean native 404에만 `resource.ErrNotFound`를 더합니다. read/Close·source 변경·취소가 섞인 404는 부재로 분류하지 않습니다. native401 `*gophercloud.ErrUnableToReauthenticate`의 `ErrOriginal`과 `ErrReauth`를 보존하고 fault가 있는 응답에 retry/reauth hook을 실행하지 않습니다. clean401/503은 원래 Provider 정책으로 재시도하면서 route·query·version을 유지합니다.

Stock Cinder quota show policy의 project reader/admin 분기와 실제 deployment override를 서버가 검사합니다. GetRecord는 조회이며 quota update/reset의 admin 권한을 합성하지 않습니다. project 이름 lookup·current-project binding은 기존 Connection scope를 사용하고 토큰 문자열이나 endpoint에서 project를 추측하지 않습니다. 임의 user_id query를 보낼 수 있어도 Cinder가 사용자 quota를 지원한다고 주장하지 않습니다.

Go는 JSON-domain Resource와 명시 `Decode`를 제공하며 Python의 임의 객체 name collision·Adapter cache/session 교체·dirty commit 전체를 재현하지 않습니다. string/bool control과 strict UTF8는 명시 Go 차이입니다. 이러한 공통 SDK-R1/C1/S1 작업과 실클라우드 acceptance는 전체 목표에 남으며 이번 named quota 조회 구현을 서비스 전체 parity로 집계하지 않습니다.

근거: [quota class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/quota_set.py), [공통 translator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py), [상속 fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py), [서비스 계약 테스트](python_query_test.go), [공통 계약 테스트](../../../internal/quotafetch/read_test.go). 비교 예제 빌드와 격리 HTTP 검증은 root의 최종 검증으로 보고하며 이 문서는 인증된 Cinder 테스트 결과를 주장하지 않습니다.
