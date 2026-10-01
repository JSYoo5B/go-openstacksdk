# Cyborg device profiles

`deviceprofiles.API`는 profile 생성, UUID 조회, 목록, 정확한 이름 조회, 삭제와 삭제 완료 대기를 제공합니다. `New`에는 accelerator v2 `*gophercloud.ServiceClient`를 전달합니다. Gophercloud에 없는 Cyborg 모델과 HTTP 계약은 SDK가 관리하므로 호출자가 builder interface를 구현할 필요가 없습니다.

| openstacksdk | Go SDK |
| --- | --- |
| `create_device_profile(name=..., groups=...)` | `Create(ctx, CreateOpts{...}, WithCreateField(...))` |
| `get_device_profile(uuid)` | `Get(ctx, uuid)` 또는 `Find(ctx, resource.ID(uuid))` |
| `device_profiles(**query)` | `List(ctx, resource.WithQuery(...))` / `All(ctx, ...)` |
| 명시적인 이름 구분 없이 문자열 전달 | `Find(ctx, resource.Name(name))` / `ResolveID(ctx, resource.Name(name))` |
| `delete_device_profile(uuid, ignore_missing=True)` | `Delete(ctx, resource.ID(uuid))` |
| `ignore_missing=False` | `resource.WithMissingError()` |
| 일반 `resource.wait_for_delete(...)` | `WaitDeleted(ctx, ref, resource.WithTimeout(...))` |

```python
profile = conn.accelerator.create_device_profile(
    name="fpga-profile",
    groups=[{"resources:CUSTOM_FPGA": "1", "trait:CUSTOM_ACCEL": "required"}],
)
conn.accelerator.delete_device_profile(profile.uuid, ignore_missing=False)
```

```go
func createAndDelete(ctx context.Context, client *gophercloud.ServiceClient) error {
    profiles := deviceprofiles.New(client)
    profile, err := profiles.Create(ctx, deviceprofiles.CreateOpts{
        Name: "fpga-profile",
        Groups: []deviceprofiles.Group{{
            "resources:CUSTOM_FPGA": "1",
            "trait:CUSTOM_ACCEL": "required",
        }},
    })
    if err != nil { return err }
    if err := profiles.Delete(ctx, resource.ID(profile.UUID), resource.WithMissingError()); err != nil {
        return err
    }
    return profiles.WaitDeleted(ctx, resource.ID(profile.UUID),
        resource.WithTimeout(2*time.Minute), resource.WithPollInterval(time.Second))
}
```

Go 예제의 import는 `context`, `time`, `github.com/gophercloud/gophercloud/v2`, `gophercloudsdk/accelerator/v2/deviceprofiles`, `gophercloudsdk/resource`입니다. 기존 Connection에서는 `conn.Accelerator(ctx)`로 서비스 client를 선택한 후 `service.DeviceProfiles`를 사용해 같은 API에 접근할 수 있습니다.

`Create`는 **profile 하나가 든 배열**을 POST하며 201 응답을 요구합니다. pinned Python `DeviceProfile._prepare_request_body`와 Cyborg controller 모두 이 형태를 사용하고, controller는 여러 profile을 담은 배열을 거절합니다. `CreateOpts`의 `Name`은 영문자·숫자·`-`·`_`를 허용하고 `Groups`는 하나 이상의 JSON object를 요구합니다. group의 resource class, trait와 accelerator 값 검증 및 키 정규화는 서버가 담당합니다. `UUID`·`Description`은 pointer로 생략과 명시적인 값을 구분하며, 빈 description은 그대로 전송합니다.

`WithCreateOptions`는 option 생성 시 slice, group map과 pointer 내용을 JSON으로 깊게 복사합니다. 이후 원본 입력을 바꾸어도 재사용하는 option의 내용은 바뀌지 않습니다. `WithCreateField`도 확장 값을 복사하며 `name`, `groups`, `uuid`, `description`을 덮어쓰면 HTTP 전송 전에 `resource.ErrInvalidOption`을 반환합니다. `WithCreateHeader`는 추가 헤더를 지원하고 인증·microversion·Host·Content-Type 등 공통 헤더 충돌을 거절합니다. 호출자가 typed options를 대체하는 경우 뒤의 option이 우선합니다.

`Get`과 `resource.ID`는 하이픈을 포함하는 36자리 UUID만 받습니다. `resource.Name`은 UUID처럼 생긴 이름도 이름으로 처리하고, 모든 목록 페이지에서 정확한 이름을 찾습니다. 일치 항목이 둘 이상이면 `resource.ErrAmbiguous`, 없으면 `resource.ErrNotFound`입니다. `Find`의 `WithIgnoreMissing`은 없는 항목을 `nil, nil`로 반환합니다. pinned controller의 직접 이름 GET은 microversion 2.2부터 제공되지만 SDK 이름 조회는 목록에서 UUID를 해석하므로 해당 endpoint를 사용하지 않습니다.

삭제에서도 이름은 먼저 UUID로 고정합니다. Cyborg controller는 UUID가 아닌 DELETE 입력을 쉼표로 나누어 여러 이름을 삭제할 수 있으므로 SDK는 이름이나 쉼표 목록을 ID로 보내지 않습니다. 목록 응답의 잘못된 UUID도 decode 오류로 거절합니다. `Delete`는 기본적으로 404를 무시하고 403과 다른 HTTP 오류를 보존합니다. `WaitDeleted`는 이름을 한 번 해석한 후 동일한 UUID의 404를 기다리며 기본 timeout은 5분, poll 간격은 2초입니다. 부모 context 취소와 timeout은 `errors.Is`로 판별할 수 있습니다. profile에는 상태 필드가 없으므로 `Wait(..., status)`와 상태 필터는 `resource.ErrUnsupported`입니다.

응답의 `Body`는 profile object 전체를 `map[string]json.RawMessage`로 보존하며 알 수 없는 필드, `null`, 빈 값과 누락을 구분합니다. `Groups`의 숫자는 `json.Number`로 디코딩해 큰 정수를 반올림하지 않습니다. `Header`·`StatusCode`와 원본 timestamp 문자열도 보존합니다. UUID가 요청 값과 다른 응답도 원본 그대로 반환하며 waiter의 조회 대상은 최초 UUID를 유지합니다. 성공한 singleton 응답은 flat object, singular envelope 또는 항목이 정확히 하나인 plural envelope를 허용합니다. 비 object, 잘못된 group 타입, 빈/복수 결과는 오류입니다. HTTP 오류의 body·header·status는 원본 Gophercloud 오류를 통해 `errors.As`로 확인할 수 있습니다.

근거는 [openstacksdk pinned proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/_proxy.py), [pinned DeviceProfile resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/device_profile.py), [pinned Cyborg controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/device_profiles.py)와 [공식 accelerator API](https://docs.openstack.org/api-ref/accelerator/)입니다. 계약은 `api/accelerator_profiles_test.go`의 실제 HTTP fixture로 검증합니다.
