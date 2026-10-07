# Driver 조회

Ironic driver의 식별자와 이름은 `Driver.Name`입니다. SDK는 native `GetDriverDetails`와 `ListDrivers`를 공통 Collection에 연결하여 다른 리소스와 같은 조회·목록·이름 검색 정책을 제공합니다.

| openstacksdk | Go |
|---|---|
| `conn.baremetal.get_driver("redfish")` | `service.Drivers.Resources.Get(ctx, "redfish")` |
| `conn.baremetal.drivers()` | `service.Drivers.Resources.List(ctx)` |
| 목록에서 이름 선택 | `service.Drivers.Find(ctx, resource.Name("redfish"))` |

```go
// context.Context ctx, *gophercloudsdk.Connection conn을 사용하는 함수 안에서
service, err := conn.BareMetalV1(ctx)
if err != nil { return err }
driver, err := service.Drivers.Find(ctx, resource.ID("redfish"))
if err != nil { return err }
fmt.Println(driver.Name, driver.Hosts)
```

`resource`는 `github.com/JSYoo5B/gophercloudsdk/resource`, `fmt`는 표준 라이브러리입니다. ID는 목록 조회를 생략하며 Name은 전체 목록에서 정확히 비교합니다. 이름 중복은 `resource.ErrAmbiguous`, 미존재는 `resource.ErrNotFound`입니다. `resource.WithIgnoreMissing()`을 추가한 Find는 미존재에 `nil, nil`을 반환합니다. HTTP 403과 통신 오류는 미존재로 바꾸지 않습니다.

`Resources.List`는 페이지를 순회하고 `break`와 context 취소를 적용합니다. `Resources.All`은 결과를 수집합니다. Driver에는 삭제와 문자열 상태가 없으므로 `Resources.Delete`, `Resources.Wait`, 상태 필터는 `resource.ErrUnsupported`입니다. 상세 목록과 driver type은 `service.Drivers.ListDrivers(ctx, drivers.WithListDriversOptions(drivers.ListDriversOpts{Detail: true, Type: "dynamic"}))`로 지정합니다. `drivers`는 이 패키지입니다.

Driver의 properties, RAID logical disk properties와 vendor passthru 호출은 기존 typed API를 사용합니다. 이 binding은 조회 정책을 추가하며 없는 생성·삭제 기능을 만들지 않습니다.

구현은 [공통 binding](resources_generated.go), HTTP 검증은 [named resource 계약 테스트](../../../api/baremetal_named_resources_test.go), 전체 서비스 사용법은 [Bare Metal v1](../README.md)에 있습니다.
