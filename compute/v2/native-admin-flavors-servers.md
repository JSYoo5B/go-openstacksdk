# Nova native flavor 관리와 서버 관리자 action

`flavors.New(client)`와 `servers.New(client)`(또는 `service.Flavors`, `service.Servers`)의 아래 generated 메서드는 Gophercloud `v2.15.0`의 [flavors 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/flavors/requests.go)과 [servers 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/servers/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "flavors"}` 또는 `{Resource: "servers"}` 문맥만 더하며, 허용되지 않은 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID와 extra spec key는 escape 없이 경로 segment로 이어 붙입니다. 여기의 호출은 대부분 Nova 기본 정책에서 관리자 전용이고, 예외는 각 절에 적었습니다. flavor 조회와 단일 extra spec 조회는 [flavor property와 native 조회](../flavor-property-and-native.md), 사용자 서버 action은 [Nova native 서버 호출](servers/native.md)을 참고합니다.

## flavor 생성·수정·삭제와 접근 권한

| 메서드 | 요청 | 성공 status | 반환 |
|---|---|---|---|
| `Create(ctx, opts, options...)` | `POST flavors`, `{"flavor": {...}}` | 200, 201 | 응답의 `flavor` |
| `Update(ctx, id, opts, options...)` | `PUT flavors/{id}`, `{"flavor": {"description": ...}}` | 200 | 응답의 `flavor` |
| `Delete(ctx, id)` | `DELETE flavors/{id}` | 202, 204 | error |
| `AddAccess(ctx, id, opts, options...)` | `POST flavors/{id}/action`, `{"addTenantAccess": {"tenant": ...}}` | 200 | 응답의 `flavor_access` 전체 |
| `RemoveAccess(ctx, id, opts, options...)` | `POST flavors/{id}/action`, `{"removeTenantAccess": {"tenant": ...}}` | 200 | 응답의 `flavor_access` 전체 |
| `ListAccesses(ctx, id)` | `GET flavors/{id}/os-flavor-access` | native pager 200, 204, 300 | `FlavorAccess` 행 |

`CreateOpts`에서 `required` 태그가 실제로 막는 값은 빈 `Name`과 nil `Disk`뿐입니다. `RAM`과 `VCPUs`는 0이어도 HTTP 전에 거부하지 않고 `"ram": 0`, `"vcpus": 0`을 그대로 보내므로 판단은 Nova가 합니다. `Disk`는 pointer라 0도 보내고, `Swap`·`Ephemeral`·`IsPublic`도 pointer라 0이나 false를 명시할 수 있습니다. `ID`·`RxTxFactor`·`Description`은 비어 있으면 생략합니다. `Description`과 `Update`는 microversion 2.55부터 Nova가 받으며, 빈 `UpdateOpts`는 `{"flavor": {}}`를 보냅니다.

`AddAccessOpts`와 `RemoveAccessOpts`의 `Tenant`에는 `required` 태그가 없습니다. 그래서 비어 있어도 `{"tenant": ""}`를 보내고 거부는 서버에 맡깁니다. 두 action의 응답은 바뀐 뒤의 접근 목록 전체이고, 빈 배열이면 nil이 아닌 빈 slice를 돌려줍니다. `ListAccesses`는 다음 페이지를 따라가지 않는 단일 페이지 stream이라 `flavor_access_links`가 있어도 무시합니다. 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 `io.EOF` 오류 하나로 끝나고, 목록 오류에는 operation 문맥이 붙지 않습니다.

응답 `Flavor`의 `swap`은 숫자와 문자열을 모두 받습니다. 빈 문자열은 0이 되고 `"512"` 같은 문자열은 숫자로 바뀝니다. `extra_specs`는 2.61 이상에서 정책이 허용할 때만 응답에 들어옵니다.

## flavor extra spec 관리

| 메서드 | 요청 | 성공 status | 반환 |
|---|---|---|---|
| `CreateExtraSpecs(ctx, flavorID, opts, options...)` | `POST flavors/{id}/os-extra_specs`, `{"extra_specs": {...}}` | 200 | 응답의 `extra_specs` map |
| `UpdateExtraSpec(ctx, flavorID, opts, options...)` | `PUT flavors/{id}/os-extra_specs/{key}`, `{key: value}` | 200 | 응답 본문 map |
| `DeleteExtraSpec(ctx, flavorID, key)` | `DELETE flavors/{id}/os-extra_specs/{key}` | 200 | error |

`CreateExtraSpecs`는 기존 값과 병합하는 생성·갱신입니다. `ExtraSpecsOpts`가 일반 object envelope가 아닌 typed map이라서 `WithCreateExtraSpecsField`의 확장 필드는 `extra_specs` 안이 아니라 그 옆 최상위에 붙습니다. 확장 key가 `extra_specs`와 겹치면 HTTP 전에 거부됩니다.

`UpdateExtraSpec`은 정확히 한 쌍만 받고 그 key를 경로로 씁니다. 비어 있거나 두 쌍 이상이면 HTTP 전에 `ErrInvalidInput`이며, 이 메서드는 확장 필드 옵션을 제공하지 않습니다. `DeleteExtraSpec`은 flavor `Delete`의 기본값과 달리 200만 성공으로 보므로 202나 204도 오류입니다.

## 서버 관리자 action

모든 action은 `POST servers/{id}/action`으로 보내며 본문의 최상위 key가 action 이름입니다.

| 메서드 | 본문 | 성공 status | 반환 |
|---|---|---|---|
| `Evacuate(ctx, id, opts, options...)` | `{"evacuate": {"host": ..., "onSharedStorage": ..., "adminPass": ...}}` | 200 | 응답의 `adminPass` |
| `ForceDelete(ctx, id)` | `{"forceDelete": ""}` | 201, 202 | error |
| `InjectNetworkInfo(ctx, id)` | `{"injectNetworkInfo": null}` | 201, 202 | error |
| `LiveMigrate(ctx, id, opts, options...)` | `{"os-migrateLive": {"host": ..., "block_migration": ..., "disk_over_commit": ...}}` | 201, 202 | error |
| `Migrate(ctx, id)` | `{"migrate": null}` | 201, 202 | error |
| `ResetNetwork(ctx, id)` | `{"resetNetwork": null}` | 201, 202 | error |
| `ResetState(ctx, id, state)` | `{"os-resetState": {"state": ...}}` | 201, 202 | error |

`ForceDelete`는 기본 정책에서 서버 소유 프로젝트의 member도 호출할 수 있고, 나머지는 관리자 전용입니다. `ResetNetwork`는 nova-network용 action이라 최근 Nova에서는 410을 돌려주며 이 값도 오류로 남습니다. `ResetState`는 `StateActive`와 `StateError` 상수를 제공하지만 다른 문자열도 검증 없이 그대로 보냅니다.

`Evacuate`의 `onSharedStorage`에는 omitempty가 없어서 빈 옵션도 `{"evacuate": {"onSharedStorage": false}}`를 보냅니다. Nova는 2.14부터 이 key를 받지 않으므로, 이 native 호출은 사실상 2.14 미만 microversion에서만 성공합니다. client에 더 높은 microversion을 지정해도 header만 바뀌고 본문은 그대로입니다. 응답이 200인데 본문이 없으면 오류 없이 빈 문자열을 돌려주고, `{}`여도 빈 문자열입니다. `Evacuate`만 200을 요구하므로 201과 202도 오류입니다.

`LiveMigrate`의 `Host`는 omitempty가 없는 pointer라 nil이면 `"host": null`로 scheduler 선택을 요청합니다. `BlockMigration`과 `DiskOverCommit`은 nil이면 생략합니다. 2.25 이상 Nova는 `block_migration`을 필수로 요구하고 `"auto"`도 받지만, Gophercloud 필드가 `*bool`이고 `block_migration`은 typed 옵션의 key라서 `WithLiveMigrateField("block_migration", "auto")`도 HTTP 전에 거부됩니다. 그래서 2.25 이상에서는 `BlockMigration`에 true나 false를 직접 넣고 `DiskOverCommit`은 nil로 두어야 합니다. 2.30의 `force` 같은 새 key는 `WithLiveMigrateField`로 `os-migrateLive` 객체 안에 넣을 수 있습니다. `Migrate`는 옵션이 없어서 2.56의 대상 `host`를 보낼 수 없습니다.

`WithEvacuateField`와 `WithLiveMigrateField`의 확장 필드는 action 객체 안에 들어갑니다. 생략된 값이라도 옵션 struct가 가진 key(`host`, `onSharedStorage`, `adminPass`, `block_migration`, `disk_over_commit`)와 같으면 HTTP 전에 거부하고, nil 옵션도 같은 방식으로 거부합니다.
