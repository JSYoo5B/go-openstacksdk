# Placement native 호출

`service.ResourceProviders`, `service.ResourceClasses`, `service.Traits`, `service.Allocations`, `service.AllocationCandidates`, `service.Usages`의 generated 메서드는 Gophercloud `v2.15.0`의 [resourceproviders](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/resourceproviders/requests.go), [resourceclasses](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/resourceclasses/requests.go), [traits](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/traits/requests.go), [allocations](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/allocations/requests.go), [allocationcandidates](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/allocationcandidates/requests.go), [usages](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/usages/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "resourceproviders"}`처럼 패키지 이름 문맥만 더하며, 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. 목록 stream의 HTTP 오류에는 operation 문맥이 붙지 않습니다. ID, resource class 이름, trait 이름은 escape 없이 경로 segment로 이어 붙입니다.

Placement 기본 정책에서 이 문서의 호출은 모두 관리자 또는 `service` 역할 호출입니다. 배포가 정책을 바꾸었다면 그 정책을 따릅니다.

## microversion 헤더

Placement는 요청마다 `OpenStack-API-Version: placement X.Y` 헤더로 microversion을 고릅니다. SDK는 값을 정하지 않으며, 호출자가 서비스 client의 `Microversion`에 넣은 문자열을 Gophercloud가 그대로 보냅니다. 값이 비어 있으면 헤더 자체를 보내지 않으므로 서버는 최소 버전 1.0으로 처리합니다. `latest` 같은 문자열도 헤더로는 그대로 전달됩니다.

요청 본문과 응답 모양이 microversion에 따라 달라지는 호출이 많습니다. native 호출이 microversion을 직접 읽는 곳은 `UpdateAggregates` 한 곳뿐이고, 나머지는 호출자가 고른 microversion에 맞는 입력을 넣어야 합니다. 응답 decode도 아래에 적은 하나의 모양만 받습니다.

## resource provider

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST resource_providers`, `{"name", "uuid", "parent_provider_uuid"}` | 200 |
| `Get(ctx, id)` | `GET resource_providers/{id}` | 200 |
| `List(ctx, options...)` | `GET resource_providers` | native pager 200, 204, 300 |
| `Update(ctx, id, opts, options...)` | `PUT resource_providers/{id}` | 200 |
| `Delete(ctx, id)` | `DELETE resource_providers/{id}` | 202, 204 |
| `GetUsages(ctx, id)` | `GET resource_providers/{id}/usages` | 200 |
| `GetAllocations(ctx, id)` | `GET resource_providers/{id}/allocations` | 200 |
| `GetInventories(ctx, id)` | `GET resource_providers/{id}/inventories` | 200 |
| `UpdateInventories(ctx, id, opts, options...)` | `PUT resource_providers/{id}/inventories` | 200 |
| `DeleteInventories(ctx, id)` | `DELETE resource_providers/{id}/inventories` | 204 |
| `GetInventory(ctx, id, class)` | `GET resource_providers/{id}/inventories/{class}` | 200 |
| `UpdateInventory(ctx, id, class, opts, options...)` | `PUT resource_providers/{id}/inventories/{class}` | 200 |
| `DeleteInventory(ctx, id, class)` | `DELETE resource_providers/{id}/inventories/{class}` | 204 |
| `GetTraits(ctx, id)` | `GET resource_providers/{id}/traits` | 200 |
| `UpdateTraits(ctx, id, opts, options...)` | `PUT resource_providers/{id}/traits` | 200 |
| `DeleteTraits(ctx, id)` | `DELETE resource_providers/{id}/traits` | 204 |
| `GetAggregates(ctx, id)` | `GET resource_providers/{id}/aggregates` | 200 |
| `UpdateAggregates(ctx, id, opts, options...)` | `PUT resource_providers/{id}/aggregates` | 200 |

`Create`는 200만 성공으로 봅니다. 서버는 1.20부터 생성한 provider를 200 본문으로 돌려주고, 그보다 낮은 microversion에서는 본문 없는 201을 돌려주므로 이때는 HTTP 요청이 끝난 뒤에도 오류가 됩니다. `CreateOpts.Name`에는 omitempty가 없어 빈 값도 `"name": ""`으로 보내며 `UUID`와 `ParentProviderUUID`는 비면 생략합니다. `UpdateOpts`의 두 필드는 pointer이고, `ParentProviderUUID`가 빈 문자열을 가리키면 `"parent_provider_uuid": null`로 바꾸어 provider를 root로 만듭니다.

inventory 본문은 `UpdateInventoriesOpts`가 `{"resource_provider_generation": N, "inventories": {class: {...}}}`이고 `UpdateInventoryOpts`는 inventory 필드를 generation 옆에 펼친 객체입니다. inventory의 `allocation_ratio`, `max_unit`, `min_unit`, `reserved`, `step_size`, `total`에는 omitempty가 없어 0도 보냅니다. `allocation_ratio`는 `float32`로 decode합니다. `UpdateTraitsOpts`는 `{"resource_provider_generation": N, "traits": [...]}`이고 nil traits는 빈 목록이 아니라 `null`로 나갑니다. generation이 서버 값과 다르면 서버가 409를 돌려주고, 이 status는 native 오류로 그대로 남습니다.

`UpdateAggregates`는 client microversion을 `major.minor`로 해석해 minor가 19보다 작으면 envelope 없이 aggregate 목록 배열만 보냅니다. 이때 `ResourceProviderGeneration`과 `WithUpdateAggregatesField` 확장 필드는 조용히 버려지고, nil `Aggregates`는 본문 없는 PUT이 됩니다. 1.19 이상이거나 microversion이 비어 있으면 `{"aggregates": [...], "resource_provider_generation": N}` 객체를 보내며 generation은 nil이면 생략합니다. `latest`처럼 숫자 두 개로 나뉘지 않는 microversion은 HTTP 전에 해석 오류가 됩니다. major 값은 확인하지 않습니다.

응답은 envelope 없이 provider나 하위 객체를 직접 decode합니다. 본문이 `{}`이거나 `null`이면 오류 없이 zero 값을 돌려주고, 타입이 맞지 않는 값만 오류입니다. aggregate 응답의 generation은 pointer라 1.19 미만 응답에서는 nil입니다.

목록은 `resource_providers` 배열 하나를 담은 단일 페이지입니다. `ListOpts`의 `Name`, `UUID`, `MemberOf`, `Resources`, `InTree`, `Required`는 모두 문자열 하나라서 `resources=VCPU:1,MEMORY_MB:512`나 `member_of=in:agg-1,agg-2`처럼 쉼표 목록을 한 값으로 보내고, query 전체를 percent-encoding합니다. 1.39의 반복 `member_of`·`required`를 쓰는 native `ListOpts139` 타입은 alias로 노출되지만 `WithListOptions`에 넣을 수 없습니다. `WithListQuery`는 같은 key의 native 값을 대체하며 값 하나만 넣으므로 반복 key도 만들 수 없습니다.

## resource class와 trait

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `ResourceClasses.Create(ctx, opts, options...)` | `POST resource_classes`, `{"name": ...}` | 201 |
| `ResourceClasses.Get(ctx, name)` | `GET resource_classes/{name}` | 200 |
| `ResourceClasses.List(ctx)` | `GET resource_classes` | native pager 200, 204, 300 |
| `ResourceClasses.Update(ctx, name)` | `PUT resource_classes/{name}`, 본문 없음 | 201, 204 |
| `ResourceClasses.Delete(ctx, name)` | `DELETE resource_classes/{name}` | 204 |
| `Traits.Create(ctx, name)` | `PUT traits/{name}`, 본문 없음 | 201, 204 |
| `Traits.Get(ctx, name)` | `GET traits/{name}` | 204 |
| `Traits.List(ctx, options...)` | `GET traits` | native pager 200, 204, 300 |
| `Traits.Delete(ctx, name)` | `DELETE traits/{name}` | 204 |

`ResourceClasses.CreateOpts.Name`은 필수라 비어 있으면 HTTP 전에 오류이고, 생성은 결과 없이 오류만 돌려줍니다. `ResourceClasses.Update`는 custom class가 있음을 보장하는 멱등 PUT이라 새로 만들면 201, 이미 있으면 204를 받습니다. `Traits.Get`은 존재 확인만 하므로 본문 없는 204만 성공이고 값은 돌려주지 않습니다.

trait 목록의 `ListOpts.Name`은 `startswith:CUSTOM`이나 `in:A,B` 같은 문자열을 그대로 보내고, `Associated` pointer는 false도 보냅니다. trait 목록은 trait 이름 문자열을 하나씩 내보냅니다.

## allocation

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Get(ctx, consumer)` | `GET allocations/{consumer}` | 200 |
| `Update(ctx, consumer, opts, options...)` | `PUT allocations/{consumer}` | 204 |
| `Manage(ctx, opts, options...)` | `POST allocations`, consumer UUID를 key로 둔 객체 | 204 |
| `Delete(ctx, consumer)` | `DELETE allocations/{consumer}` | 202, 204 |

`UpdateOpts`는 envelope 없이 `allocations`, `project_id`, `user_id`, `consumer_generation`을 항상 보내고 `consumer_type`만 비면 생략합니다. `ConsumerGeneration`이 nil이면 `null`을 보내 새 consumer라는 뜻이 되고, 기존 consumer는 직전 `Get`의 generation을 넣어야 합니다. generation이 맞지 않으면 서버가 409를 돌려주며, 이 경우와 다른 status는 모두 native 오류입니다. 서버 쪽 필수 여부는 microversion에 따라 다르므로(1.8부터 project·user, 1.28부터 generation, 1.38부터 consumer type) native 검사는 하지 않습니다.

`ManageOpts`는 consumer UUID에서 `UpdateOpts`로 가는 map이고 본문도 그 map을 그대로 직렬화합니다. 확장 필드 위치는 consumer 수에 따라 달라집니다. consumer가 정확히 하나면 공통 merge 규칙 때문에 확장 필드가 그 consumer 객체 안에 들어가고 그 객체의 key와 겹치면 HTTP 전에 거부됩니다. consumer가 없거나 둘 이상이면 확장 필드가 최상위 key가 됩니다. nil `ManageOpts`는 `null` 본문으로 보냅니다.

`Get`은 envelope 없는 객체를 decode합니다. allocation이 없는 consumer에는 `project_id`, `user_id`, `consumer_generation`, `consumer_type`이 없으므로 해당 pointer가 nil입니다.

## allocation candidate와 usage

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `AllocationCandidates.List(ctx, options...)` | `GET allocation_candidates` | native pager 200, 204, 300 |
| `Usages.Get(ctx, options...)` | `GET usages` | 200 |

allocation candidate의 `ListOpts`는 `Required`, `MemberOf`, `SameSubtree` 목록을 반복 key로 보내고 `Resources`, `InTree`, `GroupPolicy`, `Limit`, `RootRequired`는 값 하나로 보냅니다. `ResourceGroups` map의 key는 suffix가 되어 `resources_NIC`, `required_NIC`(반복), `member_of_NIC`, `in_tree_NIC`처럼 붙고, 그룹의 빈 required 항목은 건너뜁니다. query는 key 순으로 정렬해 percent-encoding합니다.

이 목록은 단일 페이지 하나를 값 하나로 내보냅니다. `allocation_requests`가 비거나 없으면 native page가 비어 있다고 판단하므로 빈 결과 값 없이 끝납니다. generated `List`는 native `ExtractAllocationCandidates`로 decode하므로 `allocations`가 provider UUID를 key로 둔 객체인 1.12 이상 응답을 읽습니다. 1.34의 `mappings`, 1.17의 `traits`, 1.29의 parent·root provider UUID는 pointer라 해당 microversion 아래에서는 nil입니다. `allocations`가 배열인 1.10, 1.11 응답은 decode 오류 하나로 끝나므로, 이 범위에서는 `RawClient()`로 native `List`와 `ExtractAllocationCandidates110`을 직접 호출합니다.

`Usages.Get`의 `GetOpts`는 `project_id`, `user_id`, `consumer_type` query를 보냅니다. `ProjectID`는 서버에서 필수지만 native 필수 tag가 없어 비어 있으면 query 없이 요청합니다. 응답은 native `Extract`로 읽어 1.38 이상의 consumer type별 map만 받습니다. 1.9부터 1.37까지의 평평한 `{"usages": {"VCPU": 2}}` 응답은 decode 오류가 되므로, 이 범위에서는 `RawClient()`로 native `Get`과 `ExtractPre138`을 호출합니다.

## 공통 입력 규칙과 목록

`With...Field` 확장 필드는 본문 최상위에 들어가며, 옵션 struct가 선언한 key와 겹치면 omitempty로 생략된 key여도 HTTP 전에 거부됩니다. 펼쳐진 inventory 필드도 선언된 key로 봅니다. nil 옵션, 빈 확장 key, 빈 query key도 HTTP 전에 오류입니다.

목록 네 개는 모두 Gophercloud `SinglePageBase`라 다음 페이지를 따라가지 않습니다. 404 같은 status는 native pager 오류로 끝나고, 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 `io.EOF` 오류 하나로 끝납니다.

## Python openstacksdk와의 차이

이 문서는 Gophercloud native 호출만 다룹니다. openstacksdk의 `find_resource_provider`, 상태 대기, Resource 객체 동작과 microversion 자동 협상은 별도로 추적합니다. SDK가 microversion을 고르지 않으므로 같은 요청이라도 호출자가 지정한 microversion에 따라 서버 결과가 달라집니다.
