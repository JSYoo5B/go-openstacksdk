# 사용자별 프로젝트와 그룹 조회

Keystone v3에서 사용자가 접근할 수 있는 프로젝트와 속한 그룹을 조회합니다. 연결은 `sdk.Connect(ctx)`로 준비하고 `conn.Identity(ctx)`에서 v3 서비스를 가져옵니다. 호출에는 사용자 **ID**를 전달하며, 이름 조회나 현재 사용자의 ID 선택을 자동으로 수행하지 않습니다. 자신 또는 다른 사용자의 목록을 읽을 수 있는지는 클라우드의 권한 정책에 따릅니다.

## Python과 Go의 호출 대응

| openstacksdk | gophercloudsdk | Go 반환형 |
|---|---|---|
| `conn.identity.user_projects(user, **query)` | `identity.Users.ListProjectRecords(ctx, userID, options...)` | `iter.Seq2[*users.UserProjectRecord, error]` |
| 프로젝트 목록의 native 호환 호출 | `identity.Users.ListProjects(ctx, userID)` | `iter.Seq2[*projects.Project, error]` |
| `conn.identity.user_groups(user)` | `identity.Users.ListGroupRecords(ctx, userID)` | `iter.Seq2[*users.UserGroupRecord, error]` |
| 그룹 목록의 native 호환 호출 | `identity.Users.ListGroups(ctx, userID)` | `iter.Seq2[*groups.Group, error]` |

`users`, `projects`, `groups`는 `github.com/JSYoo5B/gophercloudsdk/identity/v3` 아래의 패키지입니다. 새 `UserProjectRecord`·`UserGroupRecord`는 SDK 소유 모델이며 `Resource`와 `Wire`를 각각 독립된 `*resource.RawResource`로 제공합니다. 기존 `Project`·`Group`은 Gophercloud v2.15.0의 동명 모델에 대한 타입 alias입니다. native `ListProjects`·`ListGroups`의 인자·결과 타입·pager는 그대로이고, Python의 mutable Resource 상태나 연결 context를 native alias에 추가하지 않습니다.

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

## 프로젝트 query와 SDK 소유 record

Python의 keyword query는 Go의 concrete `users.ListProjectRecordsOpts`와 `WithProjectList...` 옵션으로 전달합니다. 부모 사용자 ID를 먼저 조회하지 않고 `/users/{userID}/projects`를 읽습니다. 새 API의 userID는 비어 있지 않은 단일 URL path segment여야 하며 `/`, `?`, 이미 escape된 `%` 등은 HTTP 전에 거부합니다. Python의 동적 User/Resource 입력 대신 이 명시 ID 정책을 사용합니다.

```python
import openstack

conn = openstack.connect()
for project in conn.identity.user_projects(
    "user-id", is_enabled=True, limit=20, max_items=50
):
    print(project.id, project.name, project.domain_id)
```

대응하는 독립 실행 Go 예제입니다. 인증 설정을 준비한 뒤 사용자 ID를 첫 인자로 전달합니다. 프로젝트와 그룹의 원문 행·Resource view를 두 칸 들여쓴 JSON으로 출력합니다. native 모델이 필요한 호출자는 `record.Resource.Decode(&target)`로 별도로 변환하고 그 변환 오류를 처리할 수 있습니다.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/identity/v3/users"
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
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	identity, err := conn.Identity(ctx)
	if err != nil {
		return err
	}
	for record, err := range identity.Users.ListProjectRecords(ctx, os.Args[1],
		users.WithProjectListOptions(users.ListProjectRecordsOpts{Limit: 20, MaxItems: 50}),
		users.WithProjectListFilter("is_enabled", true),
	) {
		if err != nil {
			return fmt.Errorf("list user project records: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{
			"resource": record.Resource,
			"wire": record.Wire,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	for record, err := range identity.Users.ListGroupRecords(ctx, os.Args[1]) {
		if err != nil {
			return fmt.Errorf("list user group records: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{
			"resource": record.Resource,
			"wire": record.Wire,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}
```

| 옵션 | 의미 |
|---|---|
| `WithProjectListOptions(ListProjectRecordsOpts{...})` | typed 옵션 교체; raw query·헤더·semantic 필터는 유지 |
| `WithProjectListMaxItems(n)` | 로컬 필터를 적용하기 전 원문 행 수의 상한, 0은 무제한 |
| `WithProjectListPaginated(false)` | 첫 페이지만 소비, 기본값은 모든 페이지 |
| `WithProjectListMicroversion(version)` | 이 목록 요청의 명시 버전, 빈 값도 명시적 override |
| `WithProjectListHeader(key, value)` | 목록 요청의 추가 헤더 |
| `WithProjectListQuery(key, value)` | 별칭 변환 없이 전달하는 명시 wire query |
| `WithProjectListFilter(name, value)` | source descriptor에 따라 query 또는 로컬 Body 필터로 분류 |
| `WithProjectListFilters(map[string]any{...})` | semantic 필터 집합 교체, nil/빈 map은 해당 집합만 비움 |

semantic query는 `domain_id`, `is_domain`, `name`, `parent_id`, `is_enabled`, `tags`, `any_tags`, `not_tags`, `not_any_tags`, `limit`, `marker`입니다. `is_enabled`는 `enabled`로, 나머지 tag 별칭은 각각 `tags-any`, `not-tags`, `not-tags-any`로 변환합니다. wire 이름도 허용하며 한 bulk map에 canonical 이름과 wire 별칭이 함께 있으면 canonical 값이 우선합니다. 개별 `WithProjectListFilter` 옵션은 같은 wire 키의 마지막 값을 사용합니다. 같은 query를 semantic 옵션과 명시 wire/page 옵션으로 동시에 지정하면 오류입니다.

`Limit=0`, `Marker=""`는 wire 생략이고 음수 Limit·MaxItems는 오류입니다. nil Paginated는 전체 순회, nil Microversion은 선택된 Identity client의 버전 정책 유지입니다. Go는 버전을 자동 discovery하거나 Python의 `_max_microversion=None` 선택 과정을 수행하지 않습니다. 명시 버전·헤더는 이 작업의 소유한 client에 적용하며 기존 서비스 client를 수정하지 않습니다. custom 옵션은 iterator의 각 순회에서 한 번 적용하고, 입력 map·slice·pointer와 헤더·query는 소유한 snapshot으로 사용합니다.

로컬 Body 필터는 `id`, `description`, `options`, `links` 네 가지입니다. 서버 query로 보내지 않으며 nested object는 비어 있지 않은 실제 object에 대한 subset 조건으로 비교합니다. JSON 타입을 구별하므로 Python의 bool과0/1 동등 비교는 복제하지 않습니다. `location`은 computed 속성이므로 Body 필터에 포함되지 않습니다. unknown semantic 필터는 버립니다. `WithProjectListQuery`의 unknown wire key는 서버에 전달하는 Go 확장입니다. query 값의 null은 URL 값으로 보내지 않고, false·빈 문자열·0과 scalar 배열은 명시한 값 또는 반복 query로 전달합니다. JSON object를 query 값으로 지정하면 오류입니다. 이 직렬화는 Python/requests의 동적 값 표현과 다릅니다.

`user_id`는 고정 부모 인자입니다. semantic 또는 raw query로 다시 지정할 수 없습니다. Python public `user_projects`도 `user_id=user.id`와 같은 keyword를 중복 전달하면 TypeError를 냅니다. Go는 HTTP 전에 옵션 오류로 거부하여 부모 경로를 유지합니다. `base_path`, `jmespath_filters`, `allow_unknown_params`, `resource_type`, `session`은 이 API의 field filter가 아니며 임의 경로·Python session·JMESPath 실행 옵션을 노출하지 않습니다. 목록 제어·헤더·버전은 전용 concrete 옵션을 사용하세요.

`Wire`는 받은 행의 JSON 필드와 실제 응답 헤더·상태를 보존합니다. `Resource`는 그 복사본에 선택한 부모 `user_id`를 설정하고, nonnull `options`가 object가 아니면 source의 dict descriptor처럼 `{}`로 변환합니다. `options` 생략/null은 그대로 유지합니다. 둘은 소유한 원문 숫자·null·확장 필드를 제공하고 서로 수정 사항을 공유하지 않습니다.

Go의 RawResource는 전체 Python descriptor 객체가 아닙니다. `enabled`→`is_enabled` 응답 키 변환, `is_domain`·`enabled`의 Python truthiness coercion, 생략된 `tags`의 `[]` 기본값, 생략 속성·computed location·연결 상태를 자동으로 추가하지 않습니다. 받은 필드 이름과 값은 Wire에 남고, Resource에도 위의 options·부모 변환 외에는 그대로 남습니다. 응답 `self`도 수동 데이터이며 GET하거나 Python list처럼 제거하지 않습니다. Python Resource의 mutation/commit·cache·연결 객체 상태를 만들지 않으며, UserProject의 list-only source capability를 별도 Projects CRUD에 적용하지 않습니다.

### 프로젝트 record 목록의 페이지와 오류

새 목록은 실제 GET200의 `projects` 배열을 읽고 204는 빈 목록으로 처리합니다. 행은 nonnull JSON object여야 합니다. source Python은 HTTP adapter의 성공 범위를 사용하고 non-list envelope를 한 행으로 감싸지만, Go는 이 고정 성공 코드와 배열 구조를 검사합니다. 응답 read·Close·JSON decoding 오류는 실제 body·헤더·상태와 원인을 보존하고 SDK가 같은 요청을 자동 재전송하지 않습니다. provider에 설정된 native retry·재인증은 별도 정책입니다.

`rel`/`href` links, 최상위 `next`, HTTP Link header를 읽고 native `links.next` dict도 호환 확장으로 허용합니다. 여러 continuation 표현이 함께 있으면 각각 검사하고 서로 다른 목적지를 광고하면 거부합니다. Python의 표현별 우선순위를 그대로 사용하지 않습니다. 지정한 limit은 양수 한 개이고 marker는 비어 있지 않은 string 한 개여야 합니다. semantic limit/marker의 null·빈 배열도 허용하지 않습니다. 최초 요청에 limit이 있으면 짧은 nonempty 페이지 뒤에도 마지막 원문 string `id`를 marker로 사용하고, 빈 페이지에서는 광고된 next가 있어도 끝납니다. MaxItems만 지정한 경우 wire limit hint를 공급합니다. cap은 로컬 필터 전 원문 행을 세므로 제외된 행도 소비하며, cap 이후의 행을 decode하거나 continuation을 검사하지 않습니다. Paginated=false는 첫 페이지만 소비합니다.

최초 요청에 limit이 없으면 서버의 첫 next가 양수 limit 한 개를 추가할 수 있고, 이후 요청에서는 그 값을 유지합니다. 이후의 limit 변경이나 처음부터 지정한 limit의 교체는 거부합니다. 서버가 limit을 추가하더라도 최초 limit이 없었던 요청에 marker fallback을 새로 활성화하지 않습니다. 고정 Python도 최초 local limit=None을 유지하면서 광고된 next query를 따라갑니다.

다음 URL은 같은 collection 경로와 선택한 source에 머물러야 하며 위의 첫 server limit 외에는 기존 필터를 바꾸거나 새 필터를 추가할 수 없습니다. 반복 marker·URL, 잘못된 continuation, 후속 페이지의 HTTP/JSON/context 실패는 terminal 오류입니다. 앞서 반환한 record는 유지하고 `break`하면 후속 요청을 중단합니다. 이 경로·필터·source 보호는 임의 base_path 및 동적 session을 허용하는 Python과 다른 Go 정책입니다.

## 그룹의 SDK 소유 record

Python `conn.identity.user_groups(user)`에는 query나 다른 공개 선택 인자가 없습니다. Go도 `ListGroupRecords(ctx, userID)`로 호출하며 프로젝트의 `WithProjectList...` 옵션을 그룹에 전달하지 않습니다. 사용자 ID 조회나 group GET을 먼저 수행하지 않고 선택한 Identity client의 `/users/{userID}/groups`를 lazy하게 순회합니다. 고정 사용자 ID 정책은 프로젝트와 같고 source/version·provider·인증 정책은 선택한 client에서 유지합니다.

`UserGroupRecord.Wire`는 받은 행의 원문·실제 응답 헤더·상태를 보존하고, `Resource`는 독립 복사본에 요청 부모 `user_id`만 설정합니다. unknown vendor 필드·큰 정수·known 속성의 null과 생략을 raw JSON으로 남기며 native Group의 string/Links schema를 자동 적용하지 않습니다. Group에는 Project의 options descriptor가 없으므로 vendor `options:false`도 Resource에서 false로 유지합니다. Python Group은 unknown Body retention을 비활성화하고 descriptor 기본값·computed location·연결 Resource 상태를 제공하므로 이 owned raw 모델과 다릅니다. 응답 self 링크는 수동 데이터이고 자동 GET하지 않습니다.

기본 목록은 실제 GET200의 `groups` 배열과 native 호환204 빈 응답을 처리하고, nonnull object 행을 요구합니다. 배열이 아닌 envelope를 단일 행으로 감싸는 Python 정책은 사용하지 않습니다. `rel`/`href` links, `groups_links`, 최상위 next, HTTP Link header와 native dict `links.next` 호환 형식을 지원합니다. 빈 페이지는 광고된 continuation이 있어도 끝나고, `break`는 후속 행·페이지 소비를 중단합니다. 공개 limit 입력이 없어 최초 요청은 query 없이 시작합니다. 서버가 첫 next에 양수 limit을 추가하면 이후 고정된 값을 유지하지만 marker fallback을 활성화하지 않습니다. next가 더 없으면 임의 marker 요청이나 limit을 만들어내지 않습니다. 여러 next 채널의 충돌·외부 source/path·그 밖의 query 변경·반복 marker/URL을 거부합니다.

Project와 Group은 같은 SDK 소유 membership paging·source guard 엔진을 재사용합니다. accepted read·Close·JSON 오류는 원문·헤더·상태와 원인을 보존하고 SDK가 다시 요청하지 않습니다. 뒤 페이지의 HTTP/decode 실패나 취소 전까지 반환한 행은 유지합니다. Python의 mutable Resource/cache/session 상태 또는 UserGroup의 list-only capability를 별도 Group CRUD에 설치하지 않으며 기존 native `ListGroups`는 그대로 사용할 수 있습니다.

## native Iterator와 부분 결과

Iterator를 순회할 때 HTTP를 요청하고, native Project/Group pager의 응답 `links.next`를 따라 다음 페이지를 읽습니다. `break`하면 더 이상 행을 전달하지 않고 후속 페이지를 요청하지 않습니다. context가 취소되면 취소 오류를 반환하며, 이미 반환한 행은 유지됩니다. 뒤 페이지에서 403이나 JSON decoding 오류가 발생해도 앞서 처리한 행을 되돌리지 않습니다. 예제에서는 이미 출력한 프로젝트가 있는 상태로 그룹 조회 오류가 반환될 수도 있습니다. 전체 성공이 필요한 애플리케이션은 행을 모아 두고 순회가 오류 없이 끝난 뒤 반영하세요.

빈 목록과 native pager가 빈 페이지로 취급하는 204는 행 없이 끝납니다. 각 행을 사용하기 전에 `err`를 확인하세요. 이 API에는 `ignore_missing` 옵션이 없으며 HTTP 실패를 빈 목록으로 바꾸지 않습니다.

## 반환형 변경과 남은 비교 범위

이전 생성 코드의 두 메서드는 모두 잘못된 `iter.Seq2[*users.User, error]`를 반환하도록 선언되고 User 추출기를 사용했습니다. 실제 ProjectPage·GroupPage에 맞는 모델과 추출기로 교정했습니다. 메서드의 이름과 인자는 그대로지만, 명시적인 iterator 변수·함수 인자·결과 slice를 `*users.User`로 선언한 호출자는 각각 `*projects.Project`, `*groups.Group`으로 변경해야 합니다. 일반 `Users.List`와 `Users.ListInGroup`의 User 반환형은 그대로입니다.

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [user_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1112-L1131), [user_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1414-L1429)와 Resource 정의를 소스에서 비교했습니다. 로컬 HTTP 계약 테스트는 올바른 모델과 필드, 두 페이지의 경로·토큰·`links.next`, 빈 목록·204, 오류 전파·이전 행 보존, 취소·조기 종료와 기존 추출기의 회귀를 검증합니다. 실클라우드 권한 정책이나 Python 예제를 실행한 검증은 아닙니다.

기존 native `ListProjects`에는 query·로컬 필터·목록 제어 옵션이 없습니다. 별도 `ListProjectRecords`가 이 문서의 concrete 대응을 제공합니다. `user_groups(user)`의 공개 query 인자 부재는 `ListGroupRecords`에도 유지하며 native `ListGroups`도 기존 모델/pager 경계를 유지합니다. Python User/Resource 입력 대신 Go는 명시적인 사용자 ID를 받고, Go의 `context.Context`는 요청 취소와 시간 제한을 전달하는 별도 개념입니다. 상속된 mutable Resource·session/cache 상태는 전체 SDK 목표에서 별도로 추적합니다.

native `ListProjects`·`ListGroups`는 pager의 `links.next`를 사용합니다. 고정 Python의 상속된 `_get_next_link`는 `rel`/`href` 링크, 최상위 `next`, Link header와 marker fallback을 처리하므로 기존 두 페이지 native 테스트를 Python continuation 동등성의 근거로 사용하지 않습니다. source descriptor는 [UserProject manifest](../../../api/openstacksdk/resources/identity/v3/user_project.json)에 기록하고 생성 시 fresh AST·고정 source hash와 대조합니다. source class는 UserProject이며 Users.Resources의 User 또는 native Project collection에 이 semantic descriptor를 설치하지 않습니다. 이 문서의 소스 비교는 Python 실행 결과가 아닙니다. 새 record 목록의 판정은 [Keystone user_projects 완료 기록](../../../docs/sdk-support-ledger.md#keystone-user_projects-목록-완료)에서, 이전 native 반환형 교정은 [사용자별 목록 이력](../../../docs/sdk-support-ledger.md#identity-v3-사용자별-프로젝트그룹-목록)에서 추적합니다. 전체 Resource/session 목표는 계속 구현합니다.
