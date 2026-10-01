# Heat stack resources

`service, err := conn.OrchestrationV1(ctx)`로 준비한 서비스에서 `service.StackResources.ForStack(identity)` 또는 `InStack(ctx, stackRef)`를 사용합니다. 기존 `service.StackResources.Get(ctx, stackName, stackID, resourceName)` 등의 generated API도 유지합니다.

## openstacksdk 대응

| openstacksdk / Heat 계약 | gophercloudsdk |
|---|---|
| `conn.orchestration.resources(stack)` | `scope.List(ctx)` 또는 `scope.All(ctx)` |
| name/id를 가진 Stack Resource | `ForStack(stacks.StackIdentity{Name: name, ID: id})` |
| 문자열로 부모 stack 지정 | `InStack(ctx, resource.Name(name))` 또는 `resource.ID(id)` |
| 목록에서 정확한 resource name 선택 | `scope.Find(ctx, resource.Name(name))` |
| Heat resource 단건 조회 | `scope.Get(ctx, resourceName)` |
| Heat resource metadata 조회 | `scope.Metadata(ctx, resource.ID(resourceName))` |
| Heat resource health 변경 | `scope.MarkUnhealthy(ctx, ref, unhealthy, ...HealthOption)` |
| 상태 대기 | `scope.Wait(ctx, ref, target, ...resource.WaitOption)` |

Pinned Python Resource/Proxy는 resource 목록을 중심으로 제공하며, 별도 metadata/health proxy는 없습니다. 단건 조회·metadata·health scope는 실제 Heat endpoint와 Gophercloud v2.15.0 계약을 바탕으로 추가한 Go 매핑입니다. Python의 전체 orchestration 기능을 구현했다는 의미는 아닙니다.

## 부모와 resource identity

`ForStack`은 두 부모 값을 검증하고 HTTP 조회를 생략합니다. `InStack`은 기존 [stack resolver](../stacks/README.md)를 재사용하여 이름의 정확한 일치·중복 판정 또는 ID의 canonical redirect를 처리합니다. 부모 identity는 scope에 고정되며 `Identity()`가 반환한 복사본을 변경해도 요청 대상은 바뀌지 않습니다.

아래 Go 조각은 `service`, `ctx`가 준비된 오류 반환 함수 안에서 사용하며 `stackresources`, `stacks`, `resource`, `fmt`를 import합니다.

```go
scope, err := service.StackResources.InStack(ctx, resource.Name("app"))
if err != nil { return err }
item, err := scope.Find(ctx, resource.Name("web"))
if err != nil { return err }
fmt.Println(item.Name, item.LogicalID, item.PhysicalID, item.Owner)

id, err := scope.ResolveID(ctx, resource.Name("web"))
if err != nil { return err }
detail, err := scope.Get(ctx, id)
if err != nil { return err }
fmt.Println(detail.Attributes, detail.Detailed)
```

Scope 모델은 native `Resource`를 embed한 `ResourceView`입니다. `Name`은 응답의 `resource_name`이며 SDK의 routing ID입니다. `LogicalID`와 Nova 서버 등의 `PhysicalID`는 별도 값으로 유지하고 URL 대상으로 대체하지 않습니다. `Find(Name)`는 목록 summary를 반환하며 같은 이름이 둘 이상이면 `resource.ErrAmbiguous`입니다. `Find(ID)`와 `Get(name)`은 고정 부모 안에서 `resource_name` 단건 GET을 수행합니다. 기본 조회 누락은 오류이며 `resource.WithIgnoreMissing()`로 `nil, nil`을 선택할 수 있습니다.

`ResolveID(ID)`는 resource 조회 없이 입력을 검증해 반환합니다. 이름을 ID로 바꿀 때는 응답의 소속 stack을 확인하므로 같은 scope 안에서 Name→ID→Get/Metadata/Wait 흐름이 다른 부모를 선택하지 않습니다. 확인 가능한 owner가 없으면 이름의 ID 변환은 `resource.ErrUnsupported`이며, 다른 owner면 오류입니다.

## nested 목록과 실제 owner

`ResourceView.Owner`와 `OwnerKnown`은 Heat의 `stack`/`self` 링크에서 실제 소속 stack을 보존합니다. 링크의 host로 추가 요청을 보내지 않습니다. `ParentResource`는 소속 stack의 부모 resource 이름이며 stack UUID가 아닙니다. 목록에 nested 항목이 포함되어도 모두 root stack 소속으로 채우지 않습니다.

```go
scope, err := service.StackResources.ForStack(stacks.StackIdentity{
    Name: "app", ID: "stack-uuid",
})
if err != nil { return err }
items, err := scope.All(ctx, stackresources.WithNestedDepth(1),
    stackresources.WithDetails(true))
if err != nil { return err }
for _, item := range items {
    identity, err := item.Identity()
    if err != nil { return err }
    child, err := service.StackResources.ForResource(identity)
    if err != nil { return err }
    metadata, err := child.Metadata(ctx)
    if err != nil { return err }
    fmt.Println(identity.Stack.Name, identity.Stack.ID, identity.Name, metadata)
}
```

`ResourceIdentity`는 실제 `StackIdentity`와 resource 이름을 함께 유지합니다. `ForResource(identity)`는 알려진 세 값을 고정하고 조회를 생략합니다. `scope.InResource(ctx, ref)`는 root scope의 reference를 한 번 해석해 고정합니다. nested 항목은 `item.Identity()`를 통해 실제 owner에 별도로 바인딩합니다. root scope의 Get/Find 응답이 다른 owner를 가리키면 오류이며 이를 root 변경으로 재해석하지 않습니다. 링크가 생략된 읽기 응답은 `OwnerKnown=false`로 보존하지만 `Identity()` 변환과 health 변경은 거부합니다.

목록은 native summary 필드와 시간을 유지합니다. `Body`는 원본 JSON 필드를 보존하여 omission·null·unknown 필드와 큰 숫자가 사라지지 않으며, `Header`는 항목별 독립된 복사본이고 `StatusCode`도 유지합니다. 기본 `Detailed=false`는 attributes 등의 빈 값이 상세 조회 결과라는 의미가 아닙니다. `WithDetails(true)`로 `with_detail=true`를 요청하거나 단건 Get을 수행하면 `Detailed=true`입니다. 누락되거나 null인 `resources`는 빈 목록으로 해석하지 않고 오류로 처리합니다.

## Metadata와 health 대기

Scope의 Metadata는 `map[string]any`를 반환하여 객체·배열·숫자·bool·null을 보존합니다. JSON 숫자는 기본 decoder의 `float64`입니다. 기존 generated Metadata의 `map[string]string` 반환 타입은 유지됩니다.

```go
node, err := service.StackResources.ForResource(stackresources.ResourceIdentity{
    Stack: stacks.StackIdentity{Name: "app", ID: "stack-uuid"},
    Name: "web",
})
if err != nil { return err }
if err := node.MarkUnhealthy(ctx, false,
    stackresources.WithHealthReason("recovered")); err != nil { return err }
if _, err := node.Wait(ctx, "CHECK_COMPLETE"); err != nil { return err }
```

Health의 bool은 필수 인자로 받으며 false도 `mark_unhealthy:false`로 전송합니다. SDK가 builder를 소유하고 `WithHealthReason`으로 이유를 추가합니다. PATCH 전에는 명시적인 ID도 단건 GET으로 이름과 실제 owner를 확인합니다. Heat health API가 이름을 찾지 못했을 때 physical ID를 대신 해석하는 동작에 의존하지 않습니다. owner를 확인할 수 없거나 다른 stack 소속이면 PATCH를 보내지 않습니다. 확인 조회와 변경 요청 사이의 서버 변경을 원자적으로 막는 계약은 아닙니다.

상태 대기는 Name을 한 번 확인한 뒤 고정 부모/resource_name 단건 GET을 반복합니다. 기본 5분/2초 polling은 공통 `resource.WithTimeout`/`WithPollInterval`로 변경합니다. `*_FAILED`와 `ERROR`는 `resource.ErrFailedState`이며, 명시적인 목표 상태가 `CHECK_FAILED`라면 그 상태 도달을 성공으로 처리할 수 있습니다. Context·HTTP·decode 오류는 원래 cause를 보존하며 404는 `resource.ErrNotFound`입니다.

## 목록 계약과 지원 범위

이 Heat resource endpoint는 단일 페이지 계약입니다. top-level next 링크가 들어 있어도 따라가지 않습니다. iterator break는 추가 항목 처리를 중단하며 `All`의 빈 결과는 non-nil slice입니다. `WithPageSize`/`marker`와 지원되지 않는 query는 HTTP 전에 `resource.ErrUnsupported`입니다. `WithNestedDepth`의 음수 및 잘못된 `with_detail` 값도 요청 전에 검증합니다. `resource.WithName`은 literal exact 비교를 하고 `WithStatus`는 대소문자를 구분하지 않는 로컬 비교를 유지합니다. Heat에 보내는 status는 uppercase로 맞춥니다. 나머지 서버 필터는 `resource.WithQuery`로 `type`, `name`, `action`, `id`, `physical_resource_id`를 전달할 수 있습니다.

이번 scope는 부모 고정·resource 조회·metadata·health·상태 대기를 제공합니다. Resource 생성/삭제 endpoint는 제공하지 않으며 삭제 대기나 resource signal, resource type/schema/template와 stack events는 별도 범위입니다. [서비스 README](../README.md), [HTTP 계약 테스트](../../../api/heat_stackresources_contracts_test.go)와 [SDK 지원 판정대장](../../../docs/sdk-support-ledger.md)을 참고하세요.

계약 근거: [Heat 2026.1 ResourceController](https://docs.openstack.org/heat/2026.1/_modules/heat/api/openstack/v1/resources.html), [Heat resource formatter](https://docs.openstack.org/heat/2026.1/_modules/heat/engine/api.html#format_stack_resource), [Heat resource API reference](https://docs.openstack.org/api-ref/orchestration/v1/#stack-resources), pinned [Python Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/resource.py)와 [Proxy.resources](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/_proxy.py#L370-L403).
