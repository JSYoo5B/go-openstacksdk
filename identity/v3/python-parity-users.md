# openstacksdk identity v3 Proxy 대응: domain·group·project·user·role과 role 할당

이 문서는 고정한 openstacksdk 커밋 `ef55d7d`의 [identity v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py) 가운데 domain, group, project, user, role의 생성·조회·수정·삭제·목록과 group membership, 암호 변경, role 할당 메서드가 보내는 요청을 이 SDK의 `identity/v3` 호출과 비교합니다. Keystone 요청 자체의 status, 입력 검사, decode 규칙은 [native 관리 호출 문서](native-admin-a.md)와 [native trust·사용자 본인 호출 문서](native-trusts.md)에 있고, 여기서는 Python 코드를 옮길 때 달라지는 기본값과 반환값을 정리합니다. `user_projects/user_groups`는 [사용자 membership 문서](users/memberships.md), 기존 native `FindIdentity`는 [자동 조회 비교](../../docs/finding-identities.md)를 참고합니다. 새 Project·Group·User·Role·Domain record 조회의 범위와 named find의 미해결 분기는 아래에서 구분합니다.

판정이 `go_mapping`이면 Python이 보내는 요청을 공개 Go API로 재현할 수 있다는 뜻이고, `unresolved`이면 Go에 없는 동작이 남아 있다는 뜻입니다. 기존 요청 판정은 `domains/groups/projects/users/roles/osinherit`의 `python_parity_test.go`에, 새 read 계약은 각 owned tests와 [Project](projects/python-record-reads.md)·[Group](groups/python-records.md)·[User](users/python_records.md)·[Group 사용자](users/python-group-records.md)·[Role](roles/python-records.md)·[Domain 목록](domains/python-records.md)·[Domain 단건 조회](../cloud-domain-get-python.md) 가이드에 있습니다. named find의 unresolved 사유를 concrete 옵션의 존재만으로 지우지 않습니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`create_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L217) | `domains.API.Create` | unresolved |
| [`delete_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L228) | `domains.API.Remove` | go_mapping |
| [`get_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L278) | `domains.API.GetRecord` | go_mapping |
| [`domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L290) | `domains.API.ListRecords/AllRecords/ProxyListRecords` | go_mapping |
| [`update_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L301) | `domains.API.Update`와 `domains.WithUpdateField("id", id)` | unresolved |
| [`create_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L738) | `groups.API.Create` | go_mapping |
| [`delete_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L749) | `groups.API.Remove` | go_mapping |
| [`find_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L782) | `groups.API.FindRecord` (concrete 옵션 부분 구현) | unresolved |
| [`get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L805) | `groups.API.Get` | go_mapping |
| [`groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L818) | `groups.API.ListRecords/AllRecords/ProxyListRecords` | go_mapping |
| [`update_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L829) | `groups.API.Update`와 `groups.WithUpdateField("id", id)` | go_mapping |
| [`add_user_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L843) | `users.API.AddToGroup` | go_mapping |
| [`remove_user_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L860) | `users.API.RemoveFromGroup` | go_mapping |
| [`check_user_in_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L877) | `users.API.IsMemberOfGroup` | go_mapping |
| [`group_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L894) | `users.API.ListUsersInGroupRecords/AllUsersInGroupRecords/ProxyUsersInGroupRecords` | go_mapping |
| [`create_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1019) | `projects.API.Create` | go_mapping |
| [`delete_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1030) | `projects.API.Remove` | go_mapping |
| [`find_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1063) | `projects.API.FindRecord` (concrete 옵션 부분 구현) | unresolved |
| [`get_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1086) | `projects.API.Get` | go_mapping |
| [`projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1098) | `projects.API.ListRecords/AllRecords/ProxyListRecords` | go_mapping |
| [`update_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1150) | `projects.API.Update`와 `projects.WithUpdateField("id", id)` | go_mapping |
| [`create_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1335) | `users.API.Create` | go_mapping |
| [`delete_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1346) | `users.API.Remove` | go_mapping |
| [`get_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1402) | `users.API.Get` | go_mapping |
| [`users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1429) | `users.API.ListRecords/AllRecords/ProxyListRecords` | go_mapping |
| [`update_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1440) | `users.API.Update`와 `users.WithUpdateField("id", id)` | go_mapping |
| [`update_password`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1452) | `users.API.ChangePassword` | go_mapping |
| [`create_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1699) | `roles.API.Create` | go_mapping |
| [`delete_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1710) | `roles.API.Remove` | go_mapping |
| [`get_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1766) | `roles.API.Get` | go_mapping |
| [`roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1778) | `roles.API.ListRecords/AllRecords/ProxyListRecords` | go_mapping |
| [`update_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1788) | `roles.API.Update`와 `roles.WithUpdateField("id", id)` | go_mapping |
| [`role_assignments_filter`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1877) | `roles.API.ListAssignmentsOnResource` | go_mapping |
| [`role_assignments`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1983) | `roles.API.ListAssignments` | unresolved |
| [`assign_domain_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1999) | `roles.API.Assign`, inherited=True는 `osinherit.API.Assign` | go_mapping |
| [`unassign_domain_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2023) | `roles.API.Unassign`, inherited=True는 `osinherit.API.Unassign` | go_mapping |
| [`validate_user_has_domain_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2047) | `roles.API.Validate`, inherited=True는 `osinherit.API.Validate` | go_mapping |
| [`assign_domain_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2070) | `roles.API.Assign`, inherited=True는 `osinherit.API.Assign` | go_mapping |
| [`unassign_domain_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2094) | `roles.API.Unassign`, inherited=True는 `osinherit.API.Unassign` | go_mapping |
| [`validate_group_has_domain_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2118) | `roles.API.Validate`, inherited=True는 `osinherit.API.Validate` | go_mapping |
| [`assign_project_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2141) | `roles.API.Assign`, inherited=True는 `osinherit.API.Assign` | go_mapping |
| [`unassign_project_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2166) | `roles.API.Unassign`, inherited=True는 `osinherit.API.Unassign` | go_mapping |
| [`validate_user_has_project_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2191) | `roles.API.Validate`, inherited=True는 `osinherit.API.Validate` | go_mapping |
| [`assign_project_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2215) | `roles.API.Assign`, inherited=True는 `osinherit.API.Assign` | go_mapping |
| [`unassign_project_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2240) | `roles.API.Unassign`, inherited=True는 `osinherit.API.Unassign` | go_mapping |
| [`validate_group_has_project_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2265) | `roles.API.Validate`, inherited=True는 `osinherit.API.Validate` | go_mapping |
| [`assign_system_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2289) | `roles.API.Assign` (`System: true`) | go_mapping |
| [`unassign_system_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2309) | `roles.API.Unassign` (`System: true`) | go_mapping |
| [`validate_user_has_system_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2329) | `roles.API.Validate` (`System: true`) | go_mapping |
| [`assign_system_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2349) | `roles.API.Assign` (`System: true`) | go_mapping |
| [`unassign_system_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2369) | `roles.API.Unassign` (`System: true`) | go_mapping |
| [`validate_group_has_system_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2389) | `roles.API.Validate` (`System: true`) | go_mapping |

이 표의 52개 선언 중 47개가 `go_mapping`, 5개가 `unresolved`입니다. 이번 cycle4의 새 group_users 한 행은 focused User package gate PASS를 근거로 한 최종 판정입니다. 독립 Python/Go main과 Source oracle의 stub 경계는 가이드에 있습니다. 전체 cycle4 신규14행의 중앙 판정·독립 Go main9개 build·204개 test package 전체 gate는 PASS입니다. 다른 Find·membership mutation·role assignment와 SDK-R1/C1/S1 판정을 유지하며 전체 supported 증가는 0입니다. 최종 SDK 진행량은 [중앙 지원 기록](../../docs/sdk-support-ledger.md)에서 집계합니다.

## 공통 차이

기존 native 호출은 ID 문자열과 Gophercloud 모델을 사용합니다. 새 Project·Group·User·Role·Domain record API는 선언 속성 view와 독립적인 Wire·Envelope·실제 header/status를 제공합니다. 해당 Source 클래스는 `_max_microversion=None`이라 자체 최대 버전 discovery를 요구하지 않지만, 선택한 서비스 기본값과 명시적 per-call version은 새 concrete 옵션으로 처리합니다. Find의 per-call version은 GET에만 적용하고 fallback LIST는 원래 기본값을 사용합니다.

Python은 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)의 `raise_from_response`를 거쳐 400 미만 status를 모두 성공으로 받습니다. Go native 호출은 정해진 status만 받습니다. 새 record read API는 각 가이드의 유효 HTTP 정책을 따르고 accepted 오류는 `ResponseError`, rejected 오류는 native HTTP proof와 실제 원인을 보존합니다. 생성은 201(project만 201, 202), 조회와 수정은 200, 삭제는 202, 204, membership·암호 변경·role 할당은 204입니다. 그래서 Python에서 성공하던 200 응답이 Go에서는 `gophercloud.ErrUnexpectedResponseCode`가 될 수 있습니다.

## Project·Group record 조회와 지원 경계

`projects/groups`는 새 `ListRecords/AllRecords/ProxyListRecords`로 `go_mapping`입니다. query alias·Requests bool/repeated/null 인코딩, 선언 nonquery Body의 ordered local filter, CPython JSON-loaded equality, Keystone dictionary next 무시·root link 우선순위·짧은 페이지의 limit/marker fallback을 SDK가 처리합니다. `AllRecords`는 실패 시 collected slice를 버리고 iterator는 이미 받은 record를 보관할 수 있습니다. Proxy JMESPath는 HTTP 전에 opaque generator를 평가하므로 실제 행을 읽은 expression과 구분하며, Go iterator의 재순회·안전한 continuation 경로·비음수 cap·RFC HTTP Link 호환 처리는 문서화된 Go 정책입니다.

두 `FindRecord`는 GET 우선, clean400/403/404 fallback, 모든 페이지의 exact ID/name·중복·기본 missing과 실제 오류 원인 보존을 제공합니다. 그러나 `find_project/find_group` named 판정은 **unresolved**입니다. 임의 `Resource.fetch` controls(`requires_id/base_path/skip_cache/error_message/resource_response_key`), raw `headers`의 member GET query와 fallback LIST formal header 분기, SDK의 generic synchronized Resource/cache 상태가 현재 concrete 옵션 범위와 같지 않습니다. typed `WithRecordFindHeaders`로 raw Source kwargs까지 재현한다고 주장하지 않습니다. 공통 SDK-R1/C1/S1은 별도로 추적합니다.

[Keystone28.0.0 Project 정책](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py#L157-L174)의 목록은 admin 외에 system reader 또는 대상 domain reader를 허용하고, member GET은 own-project 사용자 경로도 있습니다. Find fallback은 목록 권한을 필요로 합니다. [Group 정책](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/common/policies/group.py)도 admin/system reader/대상 Group domain reader 경로이며 일반 project reader만으로 Group 목록을 허용한다는 뜻은 아닙니다. domain-scoped 목록의 대상·결과 제한은 [서버 controller](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/api/groups.py)가 적용합니다.

## User·Role·Domain record 목록

`users/roles/domains`도 `ListRecords/AllRecords/ProxyListRecords`로 `go_mapping`입니다. [User](users/python_records.md)·[Role](roles/python-records.md)·[Domain](domains/python-records.md) 가이드는 runnable Python/Go와 선언별 defaults/casts/aliases, 실제 Query/Body local 분류, nullable 값, unknown 응답 Wire 유지, raw cap·marker·Source link와 opaque Proxy expression을 제공합니다. Domain과 Role은 options dictionary를 보존하므로 기존 native 모델의 field 누락을 새 목록 경로의 미해결로 보지 않습니다. 기존 native `List/All/Get/FindIdentity`의 모델과 status/pager 정책은 유지합니다.

User·Domain 목록의 system/domain reader, Role의 system reader와 domain-specific 정책은 각 가이드의 권한 감사를 따릅니다. 일반 project reader가 전체 목록을 볼 수 있다고 추정하지 않습니다. User·Role·Domain 목록 계약으로 `find_user/find_role/find_domain`, cloud get/update/create 또는 role_assignments를 추가 승격하지 않습니다. group_users는 아래 별도의 fixed-parent record 목록으로 검증합니다. Generic cache·synchronized Resource lifecycle·자동 Connection bridge는 SDK-R1/C1/S1 잔여이며 supported 증가는 0입니다.

Cloud `conn.get_user/get_role`는 위 Proxy member GET과 별도입니다. 새 `identity.NewUserReadCloud(svc.Users).GetUser`·`NewRoleReadCloud(svc.Roles).GetRole`, 전체 Connection `GetUserRecord/GetRoleRecord` 및 leaf `ResolveRecord`는 named getter에서 도달하는 domain_id와 concrete headers/MV/location만 다루는 닫힌 API입니다. [Cloud User·Role getter 가이드](../cloud-user-role-get-python.md)에 None/falsey filters, User의 search 뒤에도 find하는 고정 Source 동작, Role의 entities[0] 분기와 focused race·독립 main/AST PASS 및 최종204개 test package 전체 gate PASS을 구분했습니다. 일반 `find_user/find_role(**query)`의 raw controls·Resource/cache/session 잔여는 유지하며 이 Cloud 두 선언을 위 52개 Proxy 표의 새 완료 수로 더하지 않습니다.

## Domain 단건 조회

`get_domain`은 [Domain 단건 조회 가이드](../cloud-domain-get-python.md)의 `domains.API.GetRecord`로 `go_mapping`입니다. Domain 목록의 SDK 소유 descriptor를 재사용해 nullable 속성·enabled→is_enabled·options dictionary와 원래 Wire/Envelope·실제 header/status를 보존합니다. ID GET 하나만 보내며 clean403/404에서 목록으로 바꾸지 않습니다. Concrete header/microversion/location 옵션은 복제하고, 기본 location은 null이며 명시한 CloudLocation만 적용합니다. 기존 native `Get`의 반환 모델과 status 정책은 유지합니다.

Cloud `conn.get_domain`은 `identity.NewDomainReadCloud(svc.Domains).GetDomain`을 사용합니다. `WithDomainGetID`가 있으면 이름보다 우선해 direct GET하고, ID가 없고 `WithDomainGetNameOrID`가 있으면 닫힌 `domains.API.ResolveRecord`가 GET 우선·clean400/403/404에서 name-hinted 전체 목록 fallback·중복·기본 missing을 처리합니다. `WithDomainGetFilters`는 deprecated 값의 presence만 기록하고 JSON parse·Marshal·filter·IO를 수행하지 않습니다. 고정 Source는 두 식별자가 모두 None일 때 주석과 달리 실제 InvalidRequest/HTTP0이므로 Go도 입력 오류를 반환합니다. 빈 문자열과 numeric/object ID의 Go 경계, 실제 response proof·retry guard와 실행 가능한 Python/Go main은 가이드에 있습니다.

Member GET은 stock 정책의 admin/system reader 외에 token의 own domain 또는 token project의 domain 사용자 경로도 허용하지만, fallback 목록에는 system/domain reader 권한이 필요합니다. 이 두 named getter로 full `find_domain`의 raw fetch controls를 추가 완료하지 않습니다. Generic Resource/cache·synchronized lifecycle·자동 Connection bridge는 SDK-R1/C1/S1 잔여로 유지하고 fullSDK supported 증가는 0입니다.

## domain·group·project·user·role 생성, 조회, 수정

`create_*`, `get_*`, `update_*`는 resource key로 감싼 본문을 `POST {collection}`, `GET {collection}/{id}`, `PATCH {collection}/{id}`로 보냅니다. Go의 `Create`, `Get`, `Update`가 같은 경로와 envelope를 씁니다.

```go
enabled := false
project, err := projects.New(client).Create(ctx, projects.CreateOpts{Name: "demo", DomainID: "default", Enabled: &enabled})
// Python update_project("p-1", description=None, tags=[])와 같은 본문
empty := []string{}
project, err = projects.New(client).Update(ctx, "p-1",
    projects.UpdateOpts{Tags: &empty, Extra: map[string]any{"description": nil}},
    projects.WithUpdateField("id", "p-1"))
```

Python `Proxy._update`는 ID 문자열로 새 Resource를 만들기 때문에 `id`가 변경된 속성으로 남아 PATCH 본문에 `{"project": {"id": "p-1", ...}}`처럼 들어갑니다. Go `Update`는 경로에만 ID를 넣으므로 같은 본문이 필요하면 `WithUpdateField("id", id)`를 덧붙입니다. Keystone은 이 값을 무시하므로 생략해도 결과는 같습니다.

옮길 때 달라지는 점은 다음과 같습니다.

- 모든 `CreateOpts.Name`이 필수라서 Python처럼 이름 없이 생성 요청을 보내려 하면 Go는 HTTP 전에 거부합니다.
- typed `UpdateOpts`는 빈 문자열과 nil 포인터를 생략합니다. Python에서 `description=None`처럼 null을 보내려면 group·project·user·role은 `Extra` map에 nil 값을 넣습니다. typed 필드와 같은 key를 `WithUpdateField`로 주면 HTTP 전에 오류가 납니다.
- Python Resource에 typed 필드가 없는 속성(user의 `email`, project의 알 수 없는 속성 등)은 `CreateOpts.Extra`, `UpdateOpts.Extra`나 `With...Field`로 envelope 안에 넣습니다. 응답에서도 이런 key는 `Extra`에 모입니다. project의 `links`도 `Extra`에 들어갑니다.
- user의 `password_expires_at`은 Go에서 시간대 없는 Keystone 형식으로 해석한 `time.Time`입니다. Python은 문자열 그대로 두므로 끝에 `Z`가 붙은 값은 Go에서만 decode 오류가 됩니다.

domain의 생성과 수정은 `unresolved`입니다. Native `domains.Domain`에는 Options/Extra가 없어 기존 Get/Create/Update 반환에서 options가 버려집니다. 새 member `GetRecord`는 이 필드를 보존하지만 Create/Update의 반환 계약을 바꾸지는 않습니다. 요청 쪽에서는 `domains.WithCreateField("options", ...)`를 사용할 수 있지만 `domains.UpdateOpts`의 Extra/null 수정 제한은 남습니다.

## 삭제

`delete_*`는 Python 기본값이 `ignore_missing=True`라서 404를 무시합니다. Go에서는 생성된 `Delete` 대신 공통 collection의 `Remove`를 씁니다.

```go
err := users.New(client).Remove(ctx, resource.ID("u-1"))                            // 404면 nil
err = users.New(client).Remove(ctx, resource.ID("u-1"), resource.WithMissingError()) // ignore_missing=False
```

`Remove`는 404만 성공으로 바꾸고 409 같은 다른 오류는 그대로 돌려줍니다. 생성된 `Delete`를 직접 부르면 404도 오류입니다.

## 목록

현재 `projects/groups/domains/users/roles`는 위 record API로 `go_mapping`입니다. 아래는 기존 native `List/All`의 제한이며, 새 record 목록의 구현을 미해결로 세지 않습니다. `group_users`는 별도 fixed-parent record 목록으로 이번 named `go_mapping`이며 `role_assignments`는 여전히 unresolved입니다. 인자 없이 부르면 Python과 native Go 모두 `GET {collection}`을 보내지만 native 경로에는 다음 차이가 남습니다.

- Python `Resource.list`는 `limit`이 있으면 다음 링크가 없어도 마지막 ID를 `marker`로 붙여 빈 페이지가 나올 때까지 다시 요청합니다. Go native pager는 첫 요청만 보냅니다.
- Python은 query 매핑에 없는 Body 속성(예: `users(email=...)`, `domains(description=...)`)을 받은 응답에 지역 filter로 적용합니다. 기존 native collection의 `resource.WithBodyFilter`는 HTTP 전에 `resource.ErrUnsupported`가 됩니다. 새 `WithRecordListQuery`는 선언된 nonquery Body를 로컬에서 처리합니다.

나머지 차이는 판정에 영향을 주지 않지만 알아 둘 만합니다. Python은 bool 값을 `enabled=True`처럼 대문자 텍스트로 보내고 typed `ListOpts`는 `enabled=true`를 보냅니다. Python과 같은 텍스트가 필요하면 `WithListQuery("enabled", "True")`를 씁니다. Python은 Keystone `links` dict에 `rel` key가 없어서 `links.next`를 따라가지 않지만 Go pager는 따라갑니다. Python은 모르는 query 이름을 버리고 Go는 `WithListQuery` key를 모두 서버에 보냅니다.

group·project·user·role의 `ListOpts.Filters`는 `name__contains` 같은 비교 filter를 보내지만, Gophercloud가 `q:"-"` 태그를 건너뛰지 않아 `-=...` 모양의 query가 하나 더 붙습니다. Python은 비교 filter를 아예 보낼 수 없으므로 대응 관계에는 쓰지 않고, 필요하면 `WithListQuery("name__contains", "x")`를 씁니다.

`group_users(group, **attrs)`는 [Group 사용자 record 목록](users/python-group-records.md)의 `Users.ListUsersInGroupRecords/AllUsersInGroupRecords/ProxyUsersInGroupRecords`를 사용합니다. Source는 User subclass를 새로 만들지 않고 fixed groups/{group.id}/users route에 기존 User descriptor·query·local Body filter·short marker/raw cap·opaque JMESPath를 재사용합니다. 전달된 groupID로 route를 고정하고 다른 Group continuation은 actual page proof와 함께 거절합니다. already-bound base_path·formal duplicate는 HTTP 전에 거절하며 ordinary raw group_id는 User의 unknown kwarg로 버립니다. 기존 native `ListInGroup`의 marker/local filter 제한과 User→Group 목록은 유지합니다. Stock 정책은 admin/system reader 또는 대상 Group 도메인의 domain reader 경로이며 일반 project reader만으로 전체 group 사용자 목록을 허용한다고 추정하지 않습니다.

`role_assignments(**query)`는 `role_id`, `user_id`, `group_id`, `scope_project_id`, `scope_domain_id`, `scope_system`, `inherited_to`를 `role.id`, `user.id`, `scope.OS-INHERIT:inherited_to` 같은 점 표기 key로 바꿉니다. `roles.ListAssignmentsOpts`가 앞의 여섯 개를 typed 필드로 받고 `inherited_to`와 Python 텍스트의 bool은 `WithListAssignmentsQuery`로 보냅니다. Keystone은 이 flag가 있으면 값이 `0`일 때만 꺼진 것으로 보므로 Python의 `effective=False`도 Go의 `false`처럼 켜진 것으로 해석됩니다. 다만 native `RoleAssignment`가 `links`와 scope의 `OS-INHERIT:inherited_to`를 버려서 상속 할당과 직접 할당을 결과로 구분할 수 없고, 그래서 `unresolved`입니다.

`role_assignments_filter(domain|project|system, group|user)`는 `go_mapping`입니다. `roles.API.ListAssignmentsOnResource`가 같은 `GET {domains|projects}/{id}/{users|groups}/{id}/roles`나 `GET system/{users|groups}/{id}/roles` 하나를 보냅니다. Python은 domain, project, system을 셋 다 줄 때만 거부하고 둘을 주면 domain, project, system 순서로 하나를 고르는 반면, Go는 정확히 하나를 요구하고 나머지는 HTTP 전에 거부합니다. Python은 system 인자 값을 쓰지 않고 항상 `/system`을 쓰므로 Go에서는 `System: true`만 둡니다.

## group membership과 암호 변경

`add_user_to_group`, `remove_user_from_group`, `check_user_in_group`은 `users.API.AddToGroup`, `RemoveFromGroup`, `IsMemberOfGroup`과 같은 `PUT`, `DELETE`, `HEAD groups/{group}/users/{user}`를 보냅니다. Go 인자 순서는 group ID가 먼저이고 Python은 user가 먼저입니다. `check_user_in_group`과 `IsMemberOfGroup`은 모두 204를 true, 404를 오류 없는 false로 돌려주고 403은 오류입니다. Python은 그 밖의 400 미만 status를 false로 보지만 Go는 오류로 돌려줍니다.

`update_password(user, current_password, password)`는 `users.API.ChangePassword(ctx, userID, users.ChangePasswordOpts{OriginalPassword: ..., Password: ...})`가 같은 `POST users/{id}/password`와 `{"user": {"original_password": ..., "password": ...}}` 본문을 보냅니다. Go는 204만 성공으로 받습니다.

## role 할당

`assign_*`, `unassign_*`, `validate_*` 18개는 [Domain](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/domain.py), [Project](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/project.py), [System](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/system.py)의 helper를 거쳐 `PUT`, `DELETE`, `HEAD {domains|projects}/{id}/{users|groups}/{id}/roles/{role}`나 `system/{users|groups}/{id}/roles/{role}`를 보냅니다. Go에서는 `roles.API.Assign`, `Unassign`, `Validate`에 target과 actor를 하나씩 채운 옵션을 넘깁니다.

```go
api := roles.New(client)
// Python assign_project_role_to_user("p-1", "u-1", "r-1")
err := api.Assign(ctx, "r-1", roles.AssignOpts{ProjectID: "p-1", UserID: "u-1"})
// Python validate_group_has_system_role("g-1", "r-1", "all")
hasRole := api.Validate(ctx, "r-1", roles.ValidateOpts{System: true, GroupID: "g-1"}) == nil
// Python assign_domain_role_to_group("d-1", "g-1", "r-1", inherited=True)
err = osinherit.New(client).Assign(ctx, "r-1", osinherit.AssignOpts{DomainID: "d-1", GroupID: "g-1"})
```

domain·project 할당에 `inherited=True`를 주면 Python은 경로 앞뒤에 `OS-INHERIT`와 `inherited_to_projects`를 붙입니다. Go에서는 같은 경로를 보내는 `osinherit.API.Assign`, `Unassign`, `Validate`를 부릅니다. system 할당에는 `inherited` 인자가 없습니다.

status 처리는 Python과 다릅니다. Python helper는 `Proxy.request`의 `raise_exc=False` 기본값으로 요청하고 결과를 버리기 때문에 `assign_*`와 `unassign_*`는 403이나 404에도 예외 없이 None을 돌려줍니다. `validate_*`는 204일 때만 True이고 404를 포함한 다른 status는 모두 False입니다. Go 호출은 204만 nil로 받고 나머지는 status가 담긴 오류로 돌려줍니다. 그래서 Python과 같은 bool이 필요하면 `Validate`의 오류가 nil인지만 보면 되고, 할당 실패를 Python처럼 무시하려면 오류를 버리면 됩니다. Go 쪽이 실패 원인을 더 많이 알려 주므로 이 차이는 `go_mapping` 판정 안의 차이로 기록합니다.

## unresolved 목록

- `find_project/find_group`: concrete `FindRecord` 부분 구현 외의 raw Resource.fetch control/header kwargs와 generic synchronized Resource/cache 동작이 남습니다. 위 지원 경계를 따릅니다.

- `create_domain`, `update_domain`: native `domains.Domain`이 `options` 응답 필드를 버립니다. `update_domain`은 typed 필드를 null로 보내는 방법도 없습니다.
- `role_assignments`: native `RoleAssignment`가 `links`와 `OS-INHERIT:inherited_to`를 버리고, marker 반복 요청과 `role`·`scope`·`user`·`group` dict 지역 filter가 없습니다.
