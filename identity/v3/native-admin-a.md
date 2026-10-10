# Keystone v3 native 관리 호출: domain·group·project·user·role

`service.Domains`, `service.Groups`, `service.Projects`, `service.Users`, `service.Roles`, `service.Osinherit`의 generated 메서드는 Gophercloud `v2.15.0`의 [domains](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/domains/requests.go), [groups](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/groups/requests.go), [projects](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/projects/requests.go), [users](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/users/requests.go), [roles](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/roles/requests.go), [osinherit](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/osinherit/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "domains"|"groups"|"projects"|"users"|"roles"|"osinherit"}` 문맥만 더하고, 허용하지 않는 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID는 escape 없이 경로에 이어 붙입니다.

기본 policy에서 domain·group·project·user·role의 생성·수정·삭제, role 할당과 추론 규칙 관리는 관리자 호출입니다. 자신이 속한 project·domain 조회처럼 일부 조회는 일반 사용자에게도 열려 있으므로 실제 허용 범위는 클라우드의 policy를 따릅니다. 사용자 본인이 쓰는 `Projects.ListAvailable`·`Users.ChangePassword`는 [native trust·사용자 본인 호출](native-trusts.md)에, 사용자별 project·group 목록은 [사용자 membership 문서](users/memberships.md)에 설명합니다.

## 기본 생성·조회·수정·삭제

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST domains`·`groups`·`projects`·`users`·`roles`, `{"domain"\|"group"\|"project"\|"user"\|"role": {...}}` | 201, project만 201, 202 |
| `Get(ctx, id)` | `GET {collection}/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PATCH {collection}/{id}`, 같은 envelope | 200 |
| `Delete(ctx, id)` | `DELETE {collection}/{id}` | 202, 204 |
| `List(ctx, options...)` | `GET {collection}?...` | native pager 200, 204, 300 |
| `Domains.ListAvailable(ctx)` | `GET auth/domains` | native pager 200, 204, 300 |

project `Create`만 status를 지정하지 않아 native POST 기본값인 201, 202를 받고, 나머지 생성은 201만 받습니다. `Domains.ListAvailable`은 현재 token이 접근할 수 있는 domain 목록이며 옵션이 없습니다.

## 입력 규칙

모든 `CreateOpts`의 `Name`은 필수라 비어 있으면 HTTP 전에 `gophercloud.ErrMissingInput`이 됩니다. 나머지 빈 문자열과 nil 값은 생략합니다. `Enabled`·`IsDomain`처럼 pointer인 bool은 false를 명시해 보낼 수 있고, `UpdateOpts.Description`은 빈 문자열을 가리키면 `"description": ""`을 보내 설명을 지웁니다. project `UpdateOpts.Tags`가 빈 slice를 가리키면 `"tags": []`를 보내 tag를 모두 지웁니다. user의 `Password`는 생성·수정 본문에 그대로 들어갑니다. `Options`는 `projects.Immutable`, `roles.Immutable`, `users.IgnorePasswordExpiry` 같은 key를 가진 map으로 `options` 객체가 됩니다.

group·project·user·role 입력의 `Extra` map은 별도 key로 보내지 않고 resource envelope 안에 펼쳐 넣습니다. 값이 nil이면 `null`로 보내므로 Keystone의 추가 속성을 지울 때 씁니다. `With...Field` 확장 필드도 같은 envelope 안에 들어갑니다. 확장 key가 옵션 struct의 JSON key(예: `name`, `options`, `password`)와 같거나 `Extra`로 이미 넣은 key와 겹치면 HTTP 전에 operation 문맥이 붙은 오류입니다. nil 옵션도 HTTP 전에 거부합니다.

## 목록과 filter

각 `ListOpts`의 필드와 `WithListQuery` 확장 query를 정렬된 query string으로 보냅니다. 같은 key를 확장 query로 주면 typed 값 대신 확장 값이 남습니다. domain은 `enabled`·`name`·`limit`, group은 `domain_id`·`name`, role은 `domain_id`·`name`을 받습니다. project는 `domain_id`, `enabled`, `is_domain`, `name`, `parent_id`, `tags`, `tags-any`, `not-tags`, `not-tags-any`, `limit`을, user는 `domain_id`, `enabled`, `idp_id`, `name`, `password_expires_at`, `protocol_id`, `unique_id`를 받습니다. `Users.ListInGroup(ctx, groupID, options...)`은 `GET groups/{group}/users`에 같은 `users.ListOpts`를 씁니다.

group·project·user·role의 `Filters` map은 `name__contains`처럼 `NAME__COMPARATOR` 형식의 key만 받습니다. 형식이 틀린 key는 native pager를 만들 때 `InvalidListFilter`가 되어 HTTP 없이 목록의 첫 오류로 나오며 operation 문맥은 붙지 않습니다. 다만 Gophercloud가 `q:"-"` 태그를 건너뛰지 않아 `Filters`를 쓰면 실제 filter 앞에 `-={'name__contains':'x'}` 모양의 query가 하나 더 붙습니다. key가 여럿이면 이 값의 순서는 map 순회 순서를 따릅니다. 이 query 없이 비교 filter만 보내려면 `WithListQuery("name__contains", "x")`처럼 확장 query를 씁니다.

목록은 Keystone `links.next` 문자열을 받은 그대로 따라가고 null이나 빈 값이면 멈춥니다. 빈 목록은 아무 값도 내보내지 않습니다. 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 `io.EOF` 오류 하나로 끝나고, 404는 기대 status가 200, 204, 300인 native 오류입니다. 목록 오류에는 operation 문맥이 붙지 않으며, nil 옵션처럼 SDK가 HTTP 전에 거부하는 경우에만 문맥이 붙습니다.

## 응답 decode

단건 응답은 resource envelope를 읽습니다. envelope가 없거나 null이면 오류 없이 nil을 돌려주고, envelope가 객체가 아니면 operation 문맥이 붙은 decode 오류입니다. group·project·user·role은 응답에 `extra` 객체가 있으면 그것을 `Extra`로 쓰고, 없으면 struct에 없는 나머지 key를 모읍니다. project에는 `Links` 필드가 없어서 `links`도 `Extra`에 들어가고, role은 `description` 필드와 별도로 `Extra["description"]`에도 같은 값을 복사합니다. domain에는 `Extra`가 없습니다.

user의 `enabled`는 bool 외에 `"true"` 같은 문자열도 받고 null이면 false입니다. 해석할 수 없는 문자열이나 숫자는 decode 오류입니다. `password_expires_at`은 Keystone 형식인 `2006-01-02T15:04:05.999999`(시간대 없음)로 읽어 UTC 시각이 되고, null은 zero time입니다. 끝에 `Z`가 붙은 값은 이 형식과 맞지 않아 decode 오류입니다.

## project tag

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Projects.ListTags(ctx, projectID)` | `GET projects/{id}/tags` | 200 |
| `Projects.ModifyTags(ctx, projectID, opts, options...)` | `PUT projects/{id}/tags`, `{"tags": [...]}` | 200 |
| `Projects.DeleteTags(ctx, projectID)` | `DELETE projects/{id}/tags` | 204 |

`ModifyTags` 본문에는 envelope가 없어서 확장 필드는 `tags` 옆 최상위에 붙습니다. `Tags`는 비어 있으면 생략되어 `{}`를 보내므로 이 호출로는 tag를 비울 수 없습니다. 모두 지우려면 `DeleteTags`나 빈 slice를 가리키는 `UpdateOpts.Tags`를 씁니다. `ListTags`는 `{"tags": [...]}`를 `Tags`로 읽습니다. `ModifyTags` 결과 `ProjectTags`는 최상위 `projects`·`links`만 읽고 최상위 `tags`는 무시합니다. 두 호출 모두 오류일 때 nil이 아닌 빈 값을 오류와 함께 돌려줍니다.

## group membership

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Users.AddToGroup(ctx, groupID, userID)` | `PUT groups/{group}/users/{user}`, 본문 없음 | 204 |
| `Users.IsMemberOfGroup(ctx, groupID, userID)` | `HEAD groups/{group}/users/{user}` | 204는 true, 404는 false |
| `Users.RemoveFromGroup(ctx, groupID, userID)` | `DELETE groups/{group}/users/{user}` | 204 |
| `Users.ListInGroup(ctx, groupID, options...)` | `GET groups/{group}/users?...` | native pager 200, 204, 300 |

`IsMemberOfGroup`은 404를 오류가 아닌 "속하지 않음"으로 받습니다. 그래서 group이나 user 자체가 없을 때도 false가 나옵니다. 200을 포함한 다른 status는 기대 status가 204, 404인 native 오류입니다.

## role 할당

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Roles.Assign(ctx, roleID, opts, options...)` | `PUT {projects\|domains}/{id}/{users\|groups}/{id}/roles/{role}`, system은 `PUT system/{users\|groups}/{id}/roles/{role}` | 204 |
| `Roles.Validate(ctx, roleID, opts, options...)` | 같은 경로에 `HEAD` | 204 |
| `Roles.Unassign(ctx, roleID, opts, options...)` | 같은 경로에 `DELETE` | 204 |
| `Roles.ListAssignmentsOnResource(ctx, options...)` | `GET {projects\|domains}/{id}/{users\|groups}/{id}/roles`, system은 `GET system/{users\|groups}/{id}/roles` | native pager 200, 204, 300 |
| `Roles.ListAssignments(ctx, options...)` | `GET role_assignments?...` | native pager 200, 204, 300 |

할당 옵션은 `UserID`와 `GroupID` 중 정확히 하나, `ProjectID`·`DomainID`·`System` 중 정확히 하나를 요구합니다. 어긋나면 HTTP 전에 `gophercloud.ErrMissingInput`이고 `Assign`·`Validate`·`Unassign`에서는 operation 문맥이 붙습니다. `ListAssignmentsOnResource`는 옵션 없이 부르면 같은 검사가 native pager 안에서 실패해 문맥 없는 오류 하나를 내보냅니다. `With...Options`는 위치 인자로 준 옵션 전체를 교체하며 확장 필드나 query는 없습니다.

`ListAssignments`는 `group.id`, `role.id`, `scope.domain.id`, `scope.project.id`, `scope.system`, `user.id`, `effective`, `include_names`, `include_subtree` query를 보냅니다. pointer bool은 `effective=false`처럼 값 그대로 보냅니다. Keystone은 이 flag가 query에 있으면 값이 `0`일 때만 꺼진 것으로 보므로 `false`를 보내도 켜진 것으로 해석합니다. flag를 끄려면 nil로 둡니다. 결과 `RoleAssignment`의 `Scope.System`은 system 범위일 때만 nil이 아니고, user·group·project 중 응답에 없는 쪽은 빈 값입니다.

## role 추론 규칙

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Roles.CreateRoleInferenceRule(ctx, priorRoleID, impliedRoleID)` | `PUT roles/{prior}/implies/{implied}`, 본문 없음 | 201 |
| `Roles.GetRoleInferenceRule(ctx, priorRoleID, impliedRoleID)` | `GET roles/{prior}/implies/{implied}` | 200 |
| `Roles.DeleteRoleInferenceRule(ctx, priorRoleID, impliedRoleID)` | `DELETE roles/{prior}/implies/{implied}` | 204 |
| `Roles.ListRoleInferenceRules(ctx)` | `GET role_inferences` | 200 |

생성·조회 결과는 최상위 `role_inference`의 `prior_role`·`implies`와 `links`를 읽습니다. `ListRoleInferenceRules`는 pager가 아닌 단일 응답이라 `links.next`가 있어도 따라가지 않고 `Links`에 그대로 남깁니다. 세 결과 모두 오류일 때 nil이 아닌 빈 값을 오류와 함께 돌려줍니다.

## OS-INHERIT 상속 할당

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Osinherit.Assign(ctx, roleID, opts, options...)` | `PUT OS-INHERIT/{projects\|domains}/{id}/{users\|groups}/{id}/roles/{role}/inherited_to_projects` | 204 |
| `Osinherit.Validate(ctx, roleID, opts, options...)` | 같은 경로에 `HEAD` | 204 |
| `Osinherit.Unassign(ctx, roleID, opts, options...)` | 같은 경로에 `DELETE` | 204 |

`UserID`와 `GroupID`, `ProjectID`와 `DomainID` 중 각각 정확히 하나가 필요하고 system 범위는 없습니다. 어긋나면 HTTP 전에 operation 문맥이 붙은 `gophercloud.ErrMissingInput`입니다. native `roles.Scope`에는 Keystone이 상속 할당에 붙이는 `OS-INHERIT:inherited_to` 필드가 없어서 `Roles.ListAssignments` 결과만으로는 상속 할당과 직접 할당을 구분할 수 없습니다.
