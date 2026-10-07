# Cyborg attributes

`attributes.API`는 deployable attribute 생성, UUID 조회, 목록, 삭제와 삭제 완료 대기를 제공합니다. `New`에는 accelerator v2 `*gophercloud.ServiceClient`를 전달합니다. attribute의 `Key`는 여러 deployable과 항목에서 반복될 수 있어서 이름 식별자로 사용하지 않습니다.

| openstacksdk | Go SDK |
| --- | --- |
| `create_attribute(deployable_id=..., key=..., value=...)` | `Create(ctx, CreateOpts{...}, WithCreateField(...))` |
| `get_attribute(uuid)` | `Get(ctx, uuid)` / `Find(ctx, resource.ID(uuid))` |
| `attributes(deployable_id=..., key=...)` | `List(ctx, resource.WithQuery(...))` / `All(ctx, ...)` |
| `delete_attribute(uuid, ignore_missing=True)` | `Delete(ctx, resource.ID(uuid))` |
| `ignore_missing=False` | `resource.WithMissingError()` |
| 일반 `resource.wait_for_delete(...)` | `WaitDeleted(ctx, resource.ID(uuid), ...)` |

```python
attribute = conn.accelerator.create_attribute(
    deployable_id=17, key="trait:CUSTOM_ACCEL", value="required",
)
for item in conn.accelerator.attributes(deployable_id=17, key="trait:CUSTOM_ACCEL"):
    print(item.uuid, item.value)
```

```go
func createAndList(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := attributes.New(client)
    _, err := api.Create(ctx, attributes.CreateOpts{
        DeployableID: 17, Key: "trait:CUSTOM_ACCEL", Value: "required",
    })
    if err != nil { return err }
    for item, err := range api.List(ctx,
        resource.WithQuery("deployable_id", "17"),
        resource.WithQuery("key", "trait:CUSTOM_ACCEL")) {
        if err != nil { return err }
        fmt.Println(item.UUID, item.Value)
    }
    return nil
}
```

Go 예제의 import는 `context`, `fmt`, `github.com/gophercloud/gophercloud/v2`, `github.com/JSYoo5B/gophercloudsdk/accelerator/v2/attributes`, `github.com/JSYoo5B/gophercloudsdk/resource`입니다. Connection을 사용하는 코드에서는 `conn.Accelerator(ctx)`가 반환한 `service.Attributes`로 같은 API에 접근합니다.

`DeployableID`는 deployable UUID가 아니라 **숫자 database ID**이며 request JSON에도 정수로 전달합니다. `Create`는 `{"deployable_id":17,"key":"...","value":"..."}` 형태의 flat object를 POST하고 201 응답을 요구합니다. pinned controller는 JSON object를 받아 attribute 하나를 만들고 flat object를 반환합니다. 공식 API 문서의 POST 응답 예제에는 `{"attributes":[...]}` envelope가 있으므로 SDK는 flat object, singular `attribute` object, 항목이 정확히 하나인 `attributes` 배열을 모두 허용합니다. 빈 배열, 복수 항목, `null` 또는 잘못된 타입은 오류로 처리하며 결과를 임의로 하나만 선택하지 않습니다. 다른 schema의 서버가 생성 요청을 이미 수락한 뒤 decode 오류를 반환할 수 있으므로 오류가 났다는 이유만으로 생성 요청을 자동 재시도하지 마세요.

`CreateOpts`의 정수 `0`, 빈 `Key`·`Value`도 생략하지 않고 그대로 전송합니다. pinned controller에 없는 값 검증을 SDK가 추가하지 않으며 서버가 도메인 규칙을 판단합니다. `WithCreateOptions`는 scalar typed 값을 복사하고 뒤에 적용한 option이 우선합니다. `WithCreateField`는 확장 JSON을 깊게 복사하지만 `deployable_id`, `key`, `value`를 덮어쓰는 경우 HTTP 전에 `resource.ErrInvalidOption`을 반환합니다. `WithCreateHeader`는 추가 헤더를 지원하고 인증·microversion·Host·Content-Type 등 공통 헤더 충돌을 거절합니다. 사용자 정의 builder는 필요하지 않습니다.

`Get`과 `resource.ID`는 하이픈이 포함된 36자리 UUID를 받습니다. `Find`·`ResolveID`의 ID 경로는 사용할 수 있지만 `resource.Name`, `WithName`과 상태 기반 `Wait`·필터는 `resource.ErrUnsupported`입니다. key로 목록을 좁히는 경우에는 `WithQuery("key", ...)`를 사용하고 반환한 모든 UUID를 확인하세요. iterator는 전체 페이지를 따라가고 `break`하면 다음 페이지를 요청하지 않습니다.

`Delete`는 기본적으로 없는 항목의 404를 무시합니다. `WithMissingError`는 404를 `resource.ErrNotFound`로 반환하고, 403·다른 HTTP 오류는 body/header/status를 가진 원본 Gophercloud 오류를 유지합니다. `Find`의 `WithIgnoreMissing`은 없는 항목을 `nil, nil`로 반환합니다. `WaitDeleted`는 최초 UUID를 고정해 404까지 조회하며 기본 timeout은 5분, poll 간격은 2초입니다. `WithTimeout`·`WithPollInterval`로 변경하고 부모 context로 요청과 대기를 취소할 수 있습니다. 응답 UUID가 요청 UUID와 달라도 반환 데이터는 그대로 유지하며 다음 poll 대상을 바꾸지 않습니다.

응답에는 `UUID`, `DeployableID`, `Key`, `Value`와 공통 timestamp/link 필드가 있습니다. 추가로 서버가 반환하는 numeric `id`는 `json.Number`로 보존하며 SDK 리소스 식별에는 사용하지 않습니다. `Body`는 attribute object 전체를 `map[string]json.RawMessage`로 보존해 큰 정수, 알 수 없는 필드, `null`·빈 값·누락을 구분합니다. `Header`, `StatusCode`와 timestamp 문자열도 보존합니다. 403/404 등의 원본 HTTP 오류는 `errors.As`·`gophercloud.ResponseCodeIs`로 확인할 수 있습니다.

근거는 [pinned openstacksdk proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/_proxy.py), [pinned Attribute resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/attribute.py), [pinned Cyborg controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/attributes.py)와 [공식 accelerator API](https://docs.openstack.org/api-ref/accelerator/)입니다. 요청 형태, singleton 응답 cardinality, 중복 key, 정보 보존과 timeout·취소 계약은 `api/accelerator_attributes_test.go`의 HTTP fixture로 검증합니다.
