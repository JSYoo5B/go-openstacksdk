# Conductor 조회

Ironic conductor의 식별자는 `Conductor.Hostname`입니다. UUID ID 필드가 따로 있는 모델이 아니며, SDK는 같은 hostname을 ID와 이름으로 사용합니다. 직접 조회, 전체 목록, 정확한 이름 조회는 공통 Collection을 사용합니다.

Conductor endpoint는 Ironic microversion **1.49**부터 제공합니다. 이는 [공식 API 참조](https://docs.openstack.org/api-ref/baremetal/)와 [REST API 1.49 변경 기록](https://docs.openstack.org/ironic/latest/contributor/webapi-version-history.html)에 명시되어 있습니다. 고정한 Python [Conductor 모델](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/baremetal/v1/conductor.py)의 `_max_microversion = '1.49'`은 모델이 선택하는 상한이며, endpoint의 최소 버전은 공식 API 근거로 확인했습니다.

연결을 만들 때 `sdk.WithMicroversion(sdk.BareMetal, "1.49")`로 명시하거나 `sdk.WithMicroversionRange(sdk.BareMetal, "1.49", "latest")`로 cloud와 교집합을 선택합니다. 응답 계약을 고정하려면 range의 상한도 지정합니다. SDK는 연결의 버전을 공유하며 conductor 호출을 보고 자동으로 최소 1.49로 올리지는 않습니다. 옵션을 생략하면 discovery 요청과 microversion header가 없으므로 서버의 기본 버전이 1.49보다 낮을 때 호출이 거부될 수 있습니다. [Microversion 선택 정책](../../../docs/microversions.md)을 참고하세요.

| openstacksdk | Go |
|---|---|
| `conn.baremetal.get_conductor("conductor-01")` | `service.Conductors.Resources.Get(ctx, "conductor-01")` |
| `conn.baremetal.conductors()` | `service.Conductors.Resources.List(ctx)` |
| 목록에서 hostname 선택 | `service.Conductors.Find(ctx, resource.Name("conductor-01"))` |

```go
// context.Context ctx를 사용하는 함수 안에서
conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.BareMetal, "1.49"))
if err != nil { return err }
service, err := conn.BareMetalV1(ctx)
if err != nil { return err }
conductor, err := service.Conductors.Find(ctx, resource.ID("conductor-01"))
if err != nil { return err }
fmt.Println(conductor.Hostname, conductor.Alive, conductor.Drivers)
```

`sdk`는 `gophercloudsdk`, `resource`는 `gophercloudsdk/resource`, `fmt`는 표준 라이브러리입니다. `resource.ID(hostname)`은 목록 요청 없이 native Get으로 조회합니다. `resource.Name(hostname)`은 페이지를 순회하며 hostname을 정확히 비교합니다. 접두사만 일치하는 이름은 선택하지 않으며 중복은 `resource.ErrAmbiguous`입니다. hostname이 이름이라는 이유로 ID를 UUID로 변환하거나 이름을 자동 추측하지 않습니다.

Find는 미존재를 `resource.ErrNotFound`로 반환합니다. `resource.WithIgnoreMissing()`을 추가하면 `nil, nil`을 반환합니다. 원래 HTTP 403과 통신 오류는 보존됩니다. `All`은 목록을 수집하고 `List`는 `break`와 context 취소를 적용합니다. `resource.WithPageSize`와 `resource.WithQuery`로 목록 query를 지정할 수 있습니다.

Conductor는 상태 문자열이 없는 조회 리소스입니다. `Alive`는 관찰 값이며 공통 상태 대기의 target status로 변환하지 않습니다. `Resources.Delete`, `Resources.Wait`, 상태 필터는 `resource.ErrUnsupported`입니다. 목록의 필드 선택과 정렬 옵션은 `service.Conductors.List(ctx, conductors.WithListOptions(...))`로 사용할 수 있습니다. `conductors`는 이 패키지입니다. Python `get_conductor(fields=...)`의 개별 조회 필드 선택은 현재 Go Get에 대응 옵션이 없으며 아직 지원하지 않습니다.

구현은 [공통 binding](resources_generated.go), HTTP 검증은 [named resource 계약 테스트](../../../api/baremetal_named_resources_test.go), 전체 서비스 사용법은 [Bare Metal v1](../README.md)에 있습니다.
