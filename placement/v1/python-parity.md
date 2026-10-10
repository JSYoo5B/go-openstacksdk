# Python openstacksdk placement 대응

이 문서는 openstacksdk [`placement/v1/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/_proxy.py)의 Proxy 메서드를 이 SDK의 Go 호출과 맞춰 봅니다. 요청 경로, query, 본문, microversion 헤더는 각 패키지의 `python_parity_test.go`가 고정합니다. native 호출 자체의 규칙은 [native-calls.md](native-calls.md)에 있습니다.

판정은 두 가지입니다. `대응`은 Python 인자의 효과를 공개 Go API로 모두 재현할 수 있다는 뜻이고, 아래 절에 적은 차이는 남습니다. `미해결`은 Go에 해당 요청이나 기본 동작이 없다는 뜻입니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `allocation_candidates(**query)` | `AllocationCandidates.List(ctx, WithListOptions(ListOpts{...}), WithListQuery(...))` | 미해결 |
| `get_allocation(consumer)` | `Allocations.Get(ctx, consumer)` | 대응 |
| `update_allocation(consumer, **attrs)` | `Allocations.Update(ctx, consumer, UpdateOpts{...})` | 대응 |
| `create_allocations(allocations)` | `Allocations.Manage(ctx, ManageOpts{...})` | 미해결 |
| `delete_allocation(consumer, ignore_missing=True)` | `Allocations.Delete(ctx, consumer)` | 대응 |
| `create_resource_class(**attrs)` | `ResourceClasses.Create(ctx, CreateOpts{Name})` | 대응 |
| `delete_resource_class(rc, ignore_missing=True)` | `ResourceClasses.Remove(ctx, resource.ID(name))` | 대응 |
| `update_resource_class(rc, **attrs)` | 없음 (`ResourceClasses.Update`는 1.7 이상의 존재 보장 PUT) | 미해결 |
| `get_resource_class(rc)` | `ResourceClasses.Get(ctx, name)` | 대응 |
| `resource_classes(**query)` | `ResourceClasses.List(ctx)`, `ResourceClasses.All(ctx, resource.WithName(...))` | 대응 |
| `create_resource_provider(**attrs)` | `ResourceProviders.Create(ctx, CreateOpts{...})` | 대응 |
| `delete_resource_provider(rp, ignore_missing=True)` | `ResourceProviders.Remove(ctx, resource.ID(id))` | 대응 |
| `update_resource_provider(rp, **attrs)` | `ResourceProviders.Update(ctx, id, UpdateOpts{...})` | 대응 |
| `get_resource_provider(rp)` | `ResourceProviders.Get(ctx, id)` | 대응 |
| `find_resource_provider(name_or_id, ignore_missing=True)` | `ResourceProviders.Find(ctx, resource.ID(v), ...)` 다음 `Find(ctx, resource.Name(v), ...)` | 대응 |
| `resource_providers(**query)` | `ResourceProviders.List(ctx, WithListOptions(ListOpts{...}))` | 대응 |
| `fetch_resource_provider_aggregates(rp)` | `ResourceProviders.GetAggregates(ctx, id)` | 대응 |
| `get_resource_provider_aggregates(rp)` | `ResourceProviders.GetAggregates(ctx, id)` | 대응 |
| `set_resource_provider_aggregates(rp, *aggregates)` | `ResourceProviders.UpdateAggregates(ctx, id, UpdateAggregatesOpts{...})` | 대응 |
| `create_resource_provider_inventory(rp, rc, *, total, **attrs)` | 없음 | 미해결 |
| `delete_resource_provider_inventory(inv, rp, ignore_missing=True)` | `ResourceProviders.DeleteInventory(ctx, id, class)` | 대응 |
| `update_resource_provider_inventory(inv, rp, *, resource_provider_generation, **attrs)` | `ResourceProviders.UpdateInventory(ctx, id, class, UpdateInventoryOpts{...})` | 대응 |
| `get_resource_provider_inventory(inv, rp)` | `ResourceProviders.GetInventory(ctx, id, class)` | 대응 |
| `resource_provider_inventories(rp, **query)` | `ResourceProviders.GetInventories(ctx, id)` | 대응 |
| `set_resource_provider_inventories(rp, inventories, generation)` | `ResourceProviders.UpdateInventories(ctx, id, UpdateInventoriesOpts{...})` | 대응 |
| `delete_resource_provider_inventories(rp)` | `ResourceProviders.DeleteInventories(ctx, id)` | 대응 |
| `fetch_resource_provider_usages(rp)` | `ResourceProviders.GetUsages(ctx, id)` | 대응 |
| `resource_provider_allocations(rp, **query)` | `ResourceProviders.GetAllocations(ctx, id)` | 대응 |
| `create_trait(name)` | `Traits.Create(ctx, name)` | 대응 |
| `delete_trait(trait, ignore_missing=True)` | `Traits.Delete(ctx, name)` | 대응 |
| `get_trait(trait)` | `Traits.Get(ctx, name)` | 대응 |
| `traits(**query)` | `Traits.List(ctx, WithListOptions(ListOpts{...}))` | 대응 |
| `get_resource_provider_trait(rp)` | `ResourceProviders.GetTraits(ctx, id)` | 대응 |
| `set_resource_provider_trait(rp_trait, **attrs)` | `ResourceProviders.UpdateTraits(ctx, id, UpdateTraitsOpts{...})` | 대응 |
| `delete_resource_provider_trait(rp, ignore_missing=True)` | `ResourceProviders.DeleteTraits(ctx, id)` | 대응 |
| `usages(project_id, user_id=None, consumer_type=None)` | `Usages.Get(ctx, WithGetOptions(GetOpts{...}))` | 대응 |
| `wait_for_status(res, status, ...)` | `ResourceProviders.WaitFor`, `ResourceClasses.WaitFor` | 미해결 |
| `wait_for_delete(res, ...)` | `ResourceProviders.WaitForDeletion`, `ResourceClasses.WaitForDeletion` | 미해결 |

표의 `ResourceProviders`, `ResourceClasses` 같은 이름은 `placement/v1.Service`의 필드이고, 옵션 함수는 각 패키지에 있습니다.

## microversion

Python은 Resource 클래스마다 `_max_microversion`을 두고 서버 최대값과 비교해 더 낮은 값을 요청마다 보냅니다. 세션에 기본 microversion이 있으면 그 값을 씁니다. 그래서 같은 Proxy 안에서도 호출마다 헤더가 다릅니다.

| Python Resource | 보내는 microversion |
|---|---|
| [`ResourceProvider`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_provider.py) (aggregate, usage, 전체 inventory 교체·삭제 포함) | 1.20 |
| [`ResourceClass`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_class.py) | 1.2 |
| [`Trait`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/trait.py), [`ResourceProviderTrait`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_provider_trait.py) | 1.6 |
| [`AllocationCandidate`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/allocation_candidate.py) | 1.34 |
| [`Allocation`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/allocation.py), [`Usage`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/usage.py) | 1.38 |
| [`ResourceProviderInventory`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_provider_inventory.py), [`ResourceProviderAllocation`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_provider_allocation.py) | 헤더 없음 (서버 1.0) |

Go는 서비스 client의 `Microversion` 문자열 하나를 모든 호출에 그대로 보냅니다. Python과 같은 헤더를 원하면 위 값으로 client를 따로 만들어 해당 패키지의 `New`에 넘기면 됩니다. 테스트도 이렇게 패키지마다 microversion을 고정합니다. 서버 최대값이 위 값보다 낮을 때 Python은 더 낮은 버전으로 내려가지만 Go는 협상하지 않습니다.

## allocation과 allocation candidate

`get_allocation`과 `Allocations.Get`은 같은 GET을 보내고 provider별 generation과 resources, `consumer_generation`, `consumer_type`, `project_id`, `user_id`를 모두 돌려줍니다. Go는 없는 값을 nil pointer로 둡니다.

`update_allocation`은 넘긴 Body 속성만 PUT 본문에 넣습니다. Go `UpdateOpts`는 `allocations`, `project_id`, `user_id`, `consumer_generation`을 항상 보내고 `consumer_type`만 비면 생략합니다. 1.38에서는 서버가 다섯 key를 모두 요구하므로 올바른 Python 호출과 같은 본문이 됩니다. `consumer_generation=None`은 Go에서 nil pointer이고 JSON `null`로 나갑니다. Python은 갱신한 Resource를 돌려주지만 응답이 본문 없는 204라서 Go의 오류 하나와 담긴 정보가 같습니다.

`create_allocations`의 consumer별 값은 Go `ManageOpts`의 `UpdateOpts`로 옮깁니다. consumer의 allocation을 지우려면 nil이 아니라 빈 map을 넣어야 `{}`가 나갑니다. nil이면 `null`이 되어 서버가 거부합니다.

`allocation_candidates`는 응답의 `allocation_requests` 하나마다 candidate 하나를 내보내고, 각 candidate의 `provider_summaries`를 그 candidate가 쓰는 provider로 줄입니다. Go `List`는 응답 전체를 값 하나로 내보내므로 `AllocationRequests[i]`의 `Allocations` key로 `ProviderSummaries`를 골라야 같은 결과가 됩니다. 반복 `required`와 `member_of`, 숫자 suffix 그룹(`resources1`, `required1`, `member_of1`, `in_tree1`)은 `ListOpts`와 `ResourceGroups`로 만들 수 있습니다. Go가 받는 응답은 1.12 이상의 모양입니다.

## resource class와 trait

`create_resource_class`는 `{"name": ...}`을 POST하고 넘긴 속성으로 만든 Resource를 돌려줍니다. 응답 201에는 본문이 없으므로 Go `Create`의 오류 하나와 정보가 같습니다. `get_resource_class`와 `Get`은 같은 GET이고 Go는 `links`도 담습니다.

`resource_classes(**query)`에서 Python은 `limit`, `marker` 외의 query를 버리고 `name` 같은 Body 속성은 받은 목록에서 걸러 냅니다. Go `List`는 query를 받지 않으며, 이름 거르기는 `ResourceClasses.All(ctx, resource.WithName(...))`가 같은 요청 뒤에 로컬에서 합니다. Placement는 이 목록에서 `limit`, `marker`를 쓰지 않습니다.

`delete_resource_class`의 기본 `ignore_missing=True`는 `Remove`의 기본 동작과 같아 404를 성공으로 봅니다. `ignore_missing=False`는 `resource.WithMissingError()`로 바꾸며 이때 `resource.ErrNotFound`가 나옵니다. `Delete`를 직접 부르면 404가 native 오류로 남습니다.

`create_trait`은 PUT에 빈 JSON 객체 `{}`를 실어 보내고 Go `Traits.Create`는 본문 없이 보냅니다. Placement는 이 본문을 읽지 않습니다. `get_trait`은 204 응답이라 넘긴 이름만 담은 Resource를 돌려주며 Go `Get`은 오류만 돌려줍니다. `traits(name=..., associated=...)`는 두 key만 서버에 보내고 Go `ListOpts`도 같은 두 key를 만듭니다. Go는 `associated`를 `true`, `false`로 쓰지만 Python은 받은 값을 그대로 query 문자열로 바꾸므로 Python bool은 `True`로 나갑니다. Go 목록은 trait 이름 문자열을 하나씩 내보냅니다.

## resource provider

`create_resource_provider`의 `id`, `parent_provider_id`는 각각 `uuid`, `parent_provider_uuid`로 나가며 Go `CreateOpts`의 `UUID`, `ParentProviderUUID`에 해당합니다. Go는 `name`을 비어 있어도 항상 보내고, 다른 Body 속성은 `WithCreateField`로 넣습니다. 1.20 응답의 provider 본문을 두 쪽 모두 돌려줍니다.

`update_resource_provider`는 변경한 속성만 PUT하고 Resource 생성 때 표시된 `uuid`는 본문에서 뺍니다. Go `UpdateOpts`는 `Name`, `ParentProviderUUID` pointer로 같은 본문을 만들고, `parent_provider_id=None`은 빈 문자열 pointer로 표현해 `null`이 나갑니다. Python은 바꿀 속성이 없으면 요청을 보내지 않지만 Go는 `{}`를 보냅니다.

`find_resource_provider`는 `GET resource_providers/{v}`를 먼저 보내고 404, 400, 403이면 `GET resource_providers?name={v}`로 다시 찾습니다. Go에는 이 둘을 묶은 호출이 없어서 `Find(ctx, resource.ID(v), resource.WithIgnoreMissing())`가 nil이거나 400·403 오류를 돌려줄 때 `Find(ctx, resource.Name(v), resource.WithIgnoreMissing())`를 이어 부릅니다. Python 기본값 `ignore_missing=True`에 맞추려면 두 호출 모두에 `WithIgnoreMissing()`이 필요하고, 옵션이 없으면 `resource.ErrNotFound`가 나옵니다. Go `Ref` ID는 `/`, `?`, `%`, 공백 같은 문자를 HTTP 전에 거부하므로 그런 값은 바로 `resource.Name`으로 찾습니다.

`resource_providers(**query)`는 `name`, `member_of`, `resources`, `in_tree`, `required`, `id`(`uuid`로 바뀜)를 서버에 보내고 나머지 Body 속성(`generation`, `parent_provider_id` 등)은 받은 목록에서 거릅니다. Go `ListOpts`는 앞의 여섯 key를 만들고, 다른 key는 `WithListQuery`로 서버에 보냅니다. Body 속성 거르기는 호출자가 결과에서 직접 해야 하며 이름만 `resource.WithName`으로 맡길 수 있습니다. Python 기본 1.20에서는 `member_of`, `required`를 반복할 수 없으므로 문자열 하나인 Go 필드로 충분합니다. 세션 microversion을 1.24 이상으로 올려 반복 key를 쓰는 경우는 Go가 만들 수 없습니다.

`delete_resource_provider`는 resource class와 같이 `Remove`가 기본으로 404를 무시합니다.

aggregate 조회는 Python이 provider Resource에 `aggregates`와 `generation`만 채워 돌려주고 Go는 `ResourceProviderAggregates`를 돌려줍니다. 1.20에서 `set_resource_provider_aggregates`는 `{"aggregates": [...], "resource_provider_generation": rp.generation}`을 보내므로, ID 문자열만 넘긴 Python 호출은 generation이 `null`이라 서버가 거부합니다. Go에서는 직전 조회의 generation을 `ResourceProviderGeneration`에 넣습니다. aggregate를 모두 지우려면 Go에서 빈 slice를 넣어야 `[]`가 나갑니다. 1.19보다 낮은 microversion에서는 두 쪽 모두 목록 배열만 보냅니다. Python은 PUT 응답의 generation을 `resource_provider_generation`이라는 별도 key에 저장하므로 반환된 `generation`은 바뀌지 않습니다.

`fetch_resource_provider_usages`와 `resource_provider_allocations`는 각각 `GetUsages`, `GetAllocations`와 같은 GET입니다. Python은 allocation을 consumer마다 Resource 하나로 펼치고 Body 속성 인자로 거르지만 Go는 consumer UUID를 key로 둔 map 하나를 돌려줍니다.

## inventory

inventory 호출은 Python이 microversion 헤더 없이 보내므로 Go도 `Microversion`이 빈 client로 맞춥니다. 다만 `set_resource_provider_inventories`와 `delete_resource_provider_inventories`는 `ResourceProvider` 메서드라 1.20으로 나갑니다.

`update_resource_provider_inventory`는 넘긴 속성과 `resource_provider_generation`만 PUT하며 서버는 빠진 필드를 기본값으로 채웁니다. Go `Inventory`는 여섯 필드를 0까지 항상 보내므로 Python이 생략한 필드에는 서버 기본값(`min_unit` 1, `step_size` 1, `allocation_ratio` 1.0, `reserved` 0, `max_unit`은 큰 정수)을 직접 넣어야 같은 결과가 됩니다. `set_resource_provider_inventories`의 class별 dict도 같으며, 모든 inventory를 지우려면 빈 map을 넣습니다. Go `allocation_ratio`는 `float32`입니다.

`resource_provider_inventories`는 resource class마다 Resource 하나를 내보내고 Body 속성 인자로 거르며, Go `GetInventories`는 class 이름을 key로 둔 map 하나를 돌려줍니다. `get_resource_provider_inventory` 결과의 class 이름은 Python에서 `id`에 있고 Go에서는 호출 인자로만 남습니다.

## 상태 대기

Placement Resource에는 `status` 속성이 없으므로 Python 호출자는 `attribute=`로 다른 속성을 고릅니다. Go에서는 `resource.WithStatusAttribute("name")`처럼 모델의 문자열 필드를 고르고, Python 기본 `failures=['ERROR']`는 `resource.WithFailureStates("ERROR")`, `interval`은 `resource.WithPollInterval`, `wait`은 `resource.WithTimeout`이나 `resource.WithUnlimitedWait`로 옮깁니다. Python `wait_for_status`의 `wait=None`은 끝없이 기다리고 `wait_for_delete`의 기본은 120초인데, Go는 둘 다 기본 5분입니다. Python은 넘긴 Resource가 이미 목표 상태면 요청 없이 돌려주지만 Go는 항상 첫 GET을 보냅니다. Go 대기는 ID를 받아 오류와 최종 모델만 돌려줍니다.

allocation·inventory·trait·provider trait 삭제는 `Remove`가 없으므로 Python 기본값 `ignore_missing=True`를 `resource.IgnoreMissing(api.Delete(...))`처럼 직접 호출을 감싸 고릅니다. 이 helper는 순수 404만 성공으로 바꾸고 409 같은 다른 실패와 전송·취소 오류는 그대로 돌려주며, 감싸지 않은 호출은 `ignore_missing=False`처럼 404를 돌려줍니다.

## 미해결 메서드

- `allocation_candidates`: Python은 `member_of1=["agg-4", "agg-5"]`처럼 suffix 그룹의 `member_of`를 반복 key로 보낼 수 있습니다(1.24 이상, 기본 1.34). Go `ResourceGroup.MemberOf`는 값 하나이고 `WithListQuery`는 같은 key를 대체하므로 이 요청을 만들 수 없습니다.
- `create_allocations`: Python은 consumer 값 dict를 그대로 보내므로 1.34의 `mappings` 같은 key를 consumer마다 넣을 수 있습니다. Go `UpdateOpts`에는 그런 필드가 없고, consumer가 둘 이상이면 `WithManageField`가 본문 최상위에 들어갑니다.
- `update_resource_class`: Python은 1.2에서 JSON 본문을 담은 PUT으로 class 이름을 바꾸고 200 응답을 받습니다. Go `Update`는 1.7 이상의 본문 없는 PUT이라 본문을 보낼 수 없고 200을 오류로 봅니다.
- `create_resource_provider_inventory`: Go에는 `POST resource_providers/{id}/inventories` 호출이 없습니다. `UpdateInventory`는 이미 있는 inventory만 바꾸고 `UpdateInventories`는 모든 inventory를 바꾸는 다른 요청입니다.
- `wait_for_status`, `wait_for_delete`: Python은 fetch가 가능한 모든 placement Resource(inventory, trait, provider trait, allocation 포함)를 기다릴 수 있습니다. Go 대기는 `ResourceProviders`와 `ResourceClasses` collection에만 있습니다.
