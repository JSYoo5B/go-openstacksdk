# Share access rules

`ShareAccessRules.InShare`는 Manila share를 고정하고 access ID 조회·목록·허용·취소·대기를 제공합니다. `resource.Name` 부모는 공통 share 검색으로 한 번 해석하며, 같은 이름이 여러 개면 `resource.ErrAmbiguous`를 반환합니다. `resource.ID` 부모는 생성 시 조회하지 않습니다. Rule 자체에는 이름이 없으므로 `Find`, `ResolveID`, `Delete`, `Wait`, `WaitDeleted`는 `resource.ID(accessID)`를 사용합니다.

```go
package main

import (
    "context"
    "fmt"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/resource"
    "gophercloudsdk/sharedfilesystems/v2/shareaccessrules"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx,
        sdk.WithMicroversionRange(sdk.SharedFileSystem, "2.45", "2.82"),
    )
    if err != nil { panic(err) }
    service, err := conn.SharedFileSystemV2(ctx)
    if err != nil { panic(err) }
    rules, err := service.ShareAccessRules.InShare(ctx, resource.Name("team-data"))
    if err != nil { panic(err) }

    created, err := rules.Allow(ctx, shareaccessrules.AllowOpts{
        AccessType: "ip",
        AccessTo: "192.0.2.0/24",
    }, shareaccessrules.WithAccessLevel("ro"),
       shareaccessrules.WithMetadata(map[string]string{"owner": "ops"}))
    if err != nil {
        // The action may have succeeded before response decoding failed.
        // A non-nil created value retains any known ID and raw response.
        fmt.Println(created)
        panic(err)
    }
    ready, err := rules.Wait(ctx, resource.ID(created.ID), "active",
        resource.WithTimeout(2*time.Minute),
        resource.WithPollInterval(time.Second))
    if err != nil { panic(err) }
    fmt.Println(ready.ID, ready.State)

    for rule, err := range rules.List(ctx) {
        if err != nil { panic(err) }
        fmt.Println(rule.ID, rule.AccessTo, rule.State)
    }
    if err := rules.Delete(ctx, resource.ID(created.ID)); err != nil { panic(err) }
    if err := rules.WaitDeleted(ctx, resource.ID(created.ID)); err != nil { panic(err) }
}
```

| openstacksdk | Go |
|---|---|
| `conn.shared_file_system.access_rules(share, **query)` | `rules.List(ctx, shareaccessrules.WithListQuery(...))` |
| `access_rules(share, max_items=5, paginated=False)` | `rules.List(ctx, WithListMaxItems(5), WithListPaginated(false))`; 아래 single-response 정책 적용 |
| `conn.shared_file_system.get_access_rule(access_id)` | `rules.Get(ctx, accessID)`; 응답의 `share_id`도 확인 |
| `conn.shared_file_system.create_access_rule(share, **attrs)` | `rules.Allow(ctx, AllowOpts{...}, ...)` 또는 `Create` |
| `conn.shared_file_system.delete_access_rule(access_id, share, ignore_missing=True)` | `rules.Delete(ctx, resource.ID(accessID))` |
| `delete_access_rule(..., unrestrict=True)` | `rules.Deny(ctx, accessID, shareaccessrules.WithUnrestrict(true))` |
| 반환 Resource의 `state` 확인 | `rules.Wait(ctx, resource.ID(accessID), "active", ...)` |

범위 객체는 [Manila 2.45의 현대 access rule 조회·목록 API](https://docs.openstack.org/api-ref/shared-file-system/#share-access-rules)를 사용합니다. 선택된 microversion이 없거나 2.45 미만이면 HTTP 요청 전에 `resource.ErrUnsupported`를 반환하며 클라이언트 버전을 변경하지 않습니다. 명시 버전 또는 위와 같은 [범위 협상](../../../docs/microversions.md)을 사용합니다. 상한은 애플리케이션이 검증한 버전으로 정합니다. 기존 `ShareAccessRules.Get/List`와 `Shares.GrantAccess/RevokeAccess/ListAccessRights` 호출은 유지되며, 이 문서의 검증·부모 정책은 scope에 적용됩니다. Legacy `access_list` action은 2.45부터 사용할 수 없습니다.

`AllowOpts`는 `access_type`과 `access_to`를 요구하며 backend별 형식·허용 여부는 Manila가 검사합니다. `AccessLevel`은 `ro` 또는 `rw`이고 생략하면 해당 필드를 보내지 않습니다. `WithMetadata(nil)`은 명시적인 `{}`를 전송하며, metadata 옵션은 입력 map을 복사합니다. `WithLockVisibility`, `WithLockDeletion`, `WithLockReason`, `WithUnrestrict`는 [restricted access 기능의 2.82](https://docs.openstack.org/api-ref/shared-file-system/#allow-access)에 맞춰 실제 선택된 버전을 확인합니다. `false`와 빈 reason도 생략과 구분합니다. 잠금의 reason은 문자열입니다. Backend의 잠금 조합·권한 정책은 HTTP 오류로 전달됩니다.

`Allow`는 `POST /shares/{shareID}/action`의 `allow_access` 응답을 반환하며 완료 상태를 기다리지 않습니다. `Wait`는 같은 access ID를 조회하고 `state=error`에서 `resource.ErrFailedState`를 반환합니다. 취소·timeout과 HTTP 오류를 보존합니다. 생성 후 응답 해석이 실패했을 때 알려진 ID·body·header를 가진 결과와 오류를 함께 반환하며 자동으로 rule을 지우지 않습니다.

`Get`과 `Deny`는 global rule endpoint에서 응답의 `share_id`를 확인합니다. 다른 share의 rule이나 누락된 `share_id`는 `ErrParentMismatch`입니다. `Deny`는 이 검증 후 `deny_access` action을 보내며 native 성공 코드인 200·202를 처리합니다. `Delete`와 `Deny`는 기본적으로 rule 미존재를 무시합니다. 엄격하게 처리하려면 각각 `resource.WithMissingError()`와 `shareaccessrules.WithDenyIgnoreMissing(false)`를 사용합니다. `Find`는 기본적으로 미존재를 오류로 처리하고 `resource.WithIgnoreMissing()`으로 바꿀 수 있습니다.

Rule 조회·취소가 404이면 고정된 share를 조회해 부모 미존재·권한 오류를 구분합니다. 부모를 확인할 수 없으면 `ErrParentUnavailable`과 `*ParentError`를 반환하며 원본 HTTP 오류도 `errors.As`와 `gophercloud.ResponseCodeIs`로 확인할 수 있습니다. `Find`의 미존재 무시, `Delete`의 기본 정책과 `WaitDeleted` 모두 부모 실패를 숨기지 않습니다. 이 확인에는 share 조회 권한이 필요합니다. 부모가 존재하고 접근 가능한 경우에만 rule의 404를 `resource.ErrNotFound` 또는 무시 정책으로 처리합니다.

`List`는 현대 endpoint의 `access_list` slice를 `iter.Seq2`로 반환하고 `All`은 비nil slice로 수집합니다. `WithListMaxItems(n)`은 decode·부모 검증한 raw 행을 로컬에서 제한합니다. 0은 무제한이고 음수는 lazy 순회 시 HTTP 전에 오류이며 마지막 옵션이 우선합니다. cap 뒤의 행 모델은 decode하지 않고 `break`도 후속 행을 소비하지 않습니다. 재순회마다 독립된 카운터와 요청을 사용하며 취소와 소비한 행의 오류는 페이지 원문·헤더·status와 함께 반환합니다.

`WithListPaginated(false)`는 첫 응답 제어를 명시할 수 있습니다. 이 endpoint는 원래 하나의 전체 collection을 반환하므로 true와 false 모두 한 번만 GET합니다. Python의 inherited limit/marker mapping만으로 서버 페이지 순회를 추측하지 않으며 `access_list_links`나 HTTP Link를 따라가지 않습니다. cap을 wire limit hint로 보내지 않습니다. 현재 API에 없는 이름·상태 필터도 만들지 않습니다. 추가 query는 `WithListQuery`로 그대로 전송하며 서버가 의미를 검사합니다. `share_id`, `max_items`, `paginated`는 scope 또는 로컬 제어가 소유하므로 확장 query로 덮어쓸 수 없습니다. `Get`은 추가 header, `Allow`·`Deny`는 추가 body field와 header를 지원합니다. Typed core field·인증·microversion·기본 header 덮어쓰기와 지원되지 않는 query/body/argument 옵션은 요청 전에 거부합니다.

```go
package example

import (
    "context"

    sdk "gophercloudsdk"
    "gophercloudsdk/resource"
    "gophercloudsdk/sharedfilesystems/v2/shareaccessrules"
)

func FirstAccessRules(ctx context.Context, conn *sdk.Connection,
    share resource.Ref) ([]*shareaccessrules.AccessRule, error) {
    service, err := conn.SharedFileSystemV2(ctx)
    if err != nil { return nil, err }
    scope, err := service.ShareAccessRules.InShare(ctx, share)
    if err != nil { return nil, err }
    return scope.All(ctx,
        shareaccessrules.WithListMaxItems(5),
        shareaccessrules.WithListPaginated(false))
}
```

선택 버전은 source service type과 실제 Manila version header가 일치하는 숫자 `2.N`이어야 합니다. `latest`는 먼저 Connection에서 범위 협상하고 숫자로 선택해야 합니다. 수동 client의 Type이 비어 있다면 matching `X-OpenStack-Manila-API-Version`이 있어야 하며 generic header만으로 조건을 증명하지 않습니다. 옵션 적용 뒤와 부모 이름 해석 뒤에도 버전을 재검사하며 client 설정을 바꾸는 custom option으로 조건을 우회할 수 없습니다.

`AccessRule`은 native `ShareAccess`, 2.82 optional lock 필드, 원본 `Body`·`Header`를 보존합니다. Manila 목록 응답은 `share_id`를 생략할 수 있으므로 `ShareID`를 채워 넣지 않고 scope의 부모를 별도 `ParentShareID`로 제공합니다. 응답 변경 추적·자동 commit·metadata 수정 연산을 이 scope가 지원한다고 가정하지 않습니다. 공유 scope와 클라이언트는 동시 호출 중 설정을 변경하지 않습니다.

응답 식별자는 정확한 lowercase `id`와 `share_id`만 사용합니다. `ID`나 `SHARE_ID` 같은 다른 대소문자의 확장 필드는 `Body`에 보존하지만 원래 ID·부모를 덮어쓰지 못합니다. `id`는 유효한 비어 있지 않은 문자열을 요구하고, 명시한 `share_id`의 null·잘못된 타입·경로 구분자도 거부합니다. 명시한 빈 `share_id`는 `ErrParentMismatch`입니다. List/Allow에서 `share_id`가 생략된 경우에는 scope의 부모를 합성하지 않습니다. 부모가 다른 rule은 Deny action을 전송하기 전에 거부하며, Allow 후 다른 필드의 decode가 실패해도 유효한 canonical ID가 알려졌다면 부분 결과에 유지합니다.

비교 기준은 pinned [Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/_proxy.py)와 [ShareAccessRule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/share_access_rule.py), [Gophercloud v2.15.0 requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/sharedfilesystems/v2/shareaccessrules/requests.go) 및 [share actions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/sharedfilesystems/v2/shares/requests.go)입니다.
