# Group 사용자 목록: openstacksdk 비교

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 Proxy.group_users, User/Resource.list/Proxy._list입니다. Native 기반은 Gophercloud v2.15.0입니다. 새 `Users.ListUsersInGroupRecords/AllUsersInGroupRecords/ProxyUsersInGroupRecords`는 **Group→User** 목록입니다. 기존 User→Group `ListGroupRecords`와 native ListInGroup, membership mutation은 유지합니다.

[Keystone28.0.0 Group 정책](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/group.py)의 list_users_in_group는 admin/system reader 또는 **대상 Group 도메인의 domain reader**를 허용합니다. [controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/groups.py)는 실제 Group을 정책 대상에 넣습니다. 일반 project reader만으로 이 목록을 허용한다고 가정하지 않으며 SDK는 token/scope를 바꾸지 않습니다.

## 실행 가능한 Python

인증 환경과 실제 group ID를 준비하고 아래를 `group-users.py`로 저장한 뒤 `python group-users.py --group-id GROUP_UUID`로 실행합니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--group-id", required=True)
    args = parser.parse_args()
    conn = openstack.connect()
    for row in conn.identity.group_users(
        args.group_id, email="member@example.com", limit=100, max_items=100
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go

같은 인증 환경에서 Go module의 main.go로 저장하고 `go run . -group-id GROUP_UUID`로 실행합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
)

func main() {
    groupID := flag.String("group-id", "", "required group UUID")
    flag.Parse()
    if *groupID == "" { log.Fatal("-group-id is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Users.ListUsersInGroupRecords(ctx, *groupID,
        users.WithGroupRecordListQuery("email", "member@example.com"),
        users.WithGroupRecordListLimit(100),
        users.WithGroupRecordListMaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s\n", data)
    }
}
```

## User 전체 profile과 고정 Group scope

Source는 별도의 GroupUser subclass를 만들지 않고 **User 그대로** base_path만 groups/{group.id}/users로 바꿉니다. Docstring의 password_expires_at 한 필터 설명보다 실제 QueryParameters가 넓습니다. Domain_id/name/password_expires_at/unique_id/idp_id/protocol_id/is_enabled→enabled·limit/marker를 서버에 보내고 email/description/default_project_id/links/password/federated/options/id 등 nonquery Body를 로컬에서 거릅니다. [User 가이드](python_records.md)의 nullable raw·enabled bool·federated list·options dict(default{})·password_expires_at raw 문자열 선언을 그대로 재사용합니다. Unknown response는 Wire, unknown kwargs는 marshaler 실행 전 discard입니다.

Group identity는 required literal string으로 고정하며 별도의 Group GET이나 version discovery를 하지 않습니다. ID를 URL path segment로 escape하고 blank/control/traversal identity를 거절합니다. Source의 mutable Group 객체/임의 base_path, unescaped slash/percent formatting 결함을 복제하지 않습니다. Group Resource 객체 입력과 dynamic routing은 generic SDK-C1/S1 범위로 남습니다.

User profile을 복사하여 collection path만 바꾸므로 standalone Users.ListRecords가 Group 경로로 바뀌지 않습니다. 요청 group ID는 caller option이나 응답 group_id/id/HREF로 바뀌지 않고 다른 Group continuation은 actual page proof와 함께 거절합니다. Source Proxy가 base_path를 이미 bound하므로 raw base_path **null 포함** 또는 typed BasePath override는 HTTP 전에 오류입니다. Bound group keyword, restored paginated/base_path duplicate도 거절합니다. Group_id는 Source User의 unknown attr라 ordinary unknown discard입니다.

## 페이지·expression·location

Default paginated=true/max_items=0/limit 생략, short nonempty explicit-limit marker continuation, raw cap-before-local-filter, Source root links dict-ignore/presence/null-error와 owned Header/version/route를 재사용합니다. Break 뒤 lookahead가 없고 AllUsersInGroupRecords는 오류에 partial slice를 버립니다. Bool limit은 Source의 True/False wire와 달리 Go decimal integer 정책으로 HTTP0 거절하며 1/0로 변환하지 않습니다. RFC Link·unsafe route/cycle/malformed marker 검증·Go iterator 재순회는 명시적인 Go 정책입니다.

ProxyUsersInGroupRecords는 아직 소비되지 않은 User generator를 JMESPath에 전달합니다. Projection/type은 HTTP0 null, length(@)는 HTTP0 type 오류, @/to_array는 generator/nested generator를 유지합니다. RecordProxyListResult의 전체가 JSON일 때만 JSON을 채우고 literal/opaque 값에 User나 HTTP receipt를 합성하지 않습니다. 기존 RecordListGenerator/UserRecord.Clone을 재사용합니다.

Default location은 null이며 WithGroupRecordListLocation은 recorded CloudLocation을 복제하는 Go 옵션입니다. User에 project_id/zone descriptor가 없어 unknown project_id와 default_project_id가 current configured project를 바꾸지 않습니다. 이 옵션의 explicit zone 편의값은 기존 User record 정책을 유지하며 자동 Connection 계산을 주장하지 않습니다. Resource와 Wire·전체 Envelope·실제 Header/StatusCode는 독립 채널입니다.

## 오류와 지원 경계

선택 Source는 option callbacks 전에 캡처되고 전후 sticky context/selected API/client guard를 적용합니다. Restore callback으로 실패를 없애지 못합니다. Accepted 오류는 actual ResponseError, rejected 오류는 native response proof와 Read/Close/custom cancel/source 원인을 보존합니다. Native401의 Unwrap 없는 pointer wrapper는 ErrOriginal 값형 actual error와 ErrReauth를 따로 검사합니다. Physical fault 이후 retry/backoff/reauth·fallback HTTP를 막고 clean401/503/429/498은 live token과 fixed group route/query/header/version을 유지합니다.

새10그룹은 기존 User native/testcloud/Gophercloud public testhelper fixture를 재사용하여 full profile·group scope·local filters·short marker/raw cap·links/null/cross-group·bound options/boollimit·opaque/standalone isolation·clone/location·sticky Source/cancel·formal restore를 검사합니다. Physical28와 clean retry4 lanes를 포함합니다. Source AST oracle은 실제 Group/Proxy/list/pager 선언과 동작을 session/resource stub으로 관측하며 full SDK integration으로 세지 않습니다. Root의 현재 Go/race/main gate만 검증 완료로 기록합니다. Named go_mapping 후보, supported 증가0이며 SDK-R1/C1/S1 cache·generic lifecycle·Connection/session은 남습니다.

Root의 `/private/tmp/go-openstacksdk-cycle4-identity-repaired.jsonl`에서 Region과 User 전체 leaf package가 PASS했습니다. 첫 build는 기존 역방향 User→Group `ListGroupRecords`와 이름 충돌로 실패했고, 새 Group→User 메서드만 `ListUsersInGroupRecords/AllUsersInGroupRecords/ProxyUsersInGroupRecords`로 바꾼 뒤 재검증했습니다. Source 구현 commit은 `86ff98c2`·`74963af6`이며 root가 push를 확인했습니다. 이 package 검증은 실제 cloud 통합 또는 whole-SDK supported 완료가 아닙니다.
