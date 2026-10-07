# 사용자별 프로젝트와 그룹 조회

Keystone v3에서 사용자가 접근할 수 있는 프로젝트와 속한 그룹을 조회합니다. 연결은 `sdk.Connect(ctx)`로 준비하고 `conn.Identity(ctx)`에서 v3 서비스를 가져옵니다. 호출에는 사용자 **ID**를 전달하며, 이름 조회나 현재 사용자의 ID 선택을 자동으로 수행하지 않습니다. 자신 또는 다른 사용자의 목록을 읽을 수 있는지는 클라우드의 권한 정책에 따릅니다.

## Python과 Go의 호출 대응

| openstacksdk | gophercloudsdk | Go 반환형 |
|---|---|---|
| `conn.identity.user_projects(user)` | `identity.Users.ListProjects(ctx, userID)` | `iter.Seq2[*projects.Project, error]` |
| `conn.identity.user_groups(user)` | `identity.Users.ListGroups(ctx, userID)` | `iter.Seq2[*groups.Group, error]` |

`projects`와 `groups`는 각각 `github.com/JSYoo5B/gophercloudsdk/identity/v3/projects`, `github.com/JSYoo5B/gophercloudsdk/identity/v3/groups` 패키지입니다. `Project`·`Group`은 Gophercloud v2.15.0의 동명 모델에 대한 타입 alias입니다. Python의 결과는 사용자 URI 범위를 가진 `UserProject`·`UserGroup` Resource이며, Go의 결과 모델에 해당 부모 범위나 Python Resource의 상태를 추가하지는 않습니다.

Python:

```python
import openstack

conn = openstack.connect()
user_id = "user-id"

for project in conn.identity.user_projects(user_id):
    print(project.id, project.name, project.domain_id)

for group in conn.identity.user_groups(user_id):
    print(group.id, group.name, group.domain_id)
```

Go의 독립 실행 예제입니다. `OS_CLOUD`/`clouds.yaml` 또는 `OS_*` 인증 설정을 준비한 뒤 사용자 ID를 첫 번째 실행 인자로 전달합니다. 하나의 context 시간 제한을 인증과 두 목록 조회에 함께 적용합니다.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s USER_ID", os.Args[0])
	}
	userID := os.Args[1]
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	identity, err := conn.Identity(ctx)
	if err != nil {
		return err
	}

	for project, err := range identity.Users.ListProjects(ctx, userID) {
		if err != nil {
			return fmt.Errorf("list user projects: %w", err)
		}
		fmt.Printf("project: %s %s domain=%s\n", project.ID, project.Name, project.DomainID)
	}
	for group, err := range identity.Users.ListGroups(ctx, userID) {
		if err != nil {
			return fmt.Errorf("list user groups: %w", err)
		}
		fmt.Printf("group: %s %s domain=%s\n", group.ID, group.Name, group.DomainID)
	}
	return nil
}
```

## Iterator와 부분 결과

Iterator를 순회할 때 HTTP를 요청하고, native Project/Group pager의 응답 `links.next`를 따라 다음 페이지를 읽습니다. `break`하면 더 이상 행을 전달하지 않고 후속 페이지를 요청하지 않습니다. context가 취소되면 취소 오류를 반환하며, 이미 반환한 행은 유지됩니다. 뒤 페이지에서 403이나 JSON decoding 오류가 발생해도 앞서 처리한 행을 되돌리지 않습니다. 예제에서는 이미 출력한 프로젝트가 있는 상태로 그룹 조회 오류가 반환될 수도 있습니다. 전체 성공이 필요한 애플리케이션은 행을 모아 두고 순회가 오류 없이 끝난 뒤 반영하세요.

빈 목록과 native pager가 빈 페이지로 취급하는 204는 행 없이 끝납니다. 각 행을 사용하기 전에 `err`를 확인하세요. 이 API에는 `ignore_missing` 옵션이 없으며 HTTP 실패를 빈 목록으로 바꾸지 않습니다.

## 반환형 변경과 남은 비교 범위

이전 생성 코드의 두 메서드는 모두 잘못된 `iter.Seq2[*users.User, error]`를 반환하도록 선언되고 User 추출기를 사용했습니다. 실제 ProjectPage·GroupPage에 맞는 모델과 추출기로 교정했습니다. 메서드의 이름과 인자는 그대로지만, 명시적인 iterator 변수·함수 인자·결과 slice를 `*users.User`로 선언한 호출자는 각각 `*projects.Project`, `*groups.Group`으로 변경해야 합니다. 일반 `Users.List`와 `Users.ListInGroup`의 User 반환형은 그대로입니다.

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [user_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1112-L1131), [user_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1414-L1429)와 Resource 정의를 소스에서 비교했습니다. 로컬 HTTP 계약 테스트는 올바른 모델과 필드, 두 페이지의 경로·토큰·`links.next`, 빈 목록·204, 오류 전파·이전 행 보존, 취소·조기 종료와 기존 추출기의 회귀를 검증합니다. 실클라우드 권한 정책이나 Python 예제를 실행한 검증은 아닙니다.

Python `user_projects(user, **query)`의 query·별칭 변환·로컬 필터·목록 제어 옵션은 이 Go 메서드에서 제공하지 않습니다. `user_groups(user)`에는 공개 query 인자가 없습니다. 두 연산의 User/Resource 입력, 부모 `user_id`, descriptor 기본값·모델의 확장 속성 정책·Resource 상태·연결 context와 상속된 session/cache/pagination 비교도 남아 있습니다. Go의 `context.Context`는 요청 취소와 시간 제한을 전달하는 별도 개념입니다.

Go는 native pager의 `links.next`를 사용합니다. 고정 Python의 상속된 `_get_next_link`는 `rel`/`href` 링크, 최상위 `next`, Link header와 marker fallback을 처리하므로 두 페이지 Go 테스트를 Python continuation 동등성의 근거로 사용하지 않습니다. 이는 고정 소스의 정적 대조이며 Python 실행 결과가 아닙니다. 이번 수정과 남은 계약은 [SDK 지원 판정대장](../../../docs/sdk-support-ledger.md#identity-v3-사용자별-프로젝트그룹-목록)에서 추적합니다.
