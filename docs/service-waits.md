# 서비스별 상태·삭제 대기

서비스 대기 API는 openstacksdk의 서비스별 기본 상태, 실패 상태, 시간 제한을 SDK가 적용합니다. 호출자는 `resource.Ref`와 구체적인 `resource.WaitOption`만 전달합니다. 별도의 상태 adapter나 polling loop를 구현할 필요가 없습니다. 대기는 조회만 수행하며 생성·삭제 요청을 보내지 않습니다.

| API | 기본 대상 | 기본 실패 상태 | SDK 시간 제한 | 조회 간격 |
|---|---|---|---|---|
| `compute.WaitForState`, Server `WaitForState` | 호출자가 지정 | `ERROR` | 없음 | 2초 |
| Server `WaitForServerState` | 호출자가 지정 | `ERROR` | 120초 | 2초 |
| Server `WaitForServer` | `ACTIVE` | `ERROR` | 120초 | 2초 |
| `blockstorage.WaitForState`, Volume/Snapshot `WaitForState` | 호출자가 지정 | `error` | 없음 | 2초 |
| `blockstorage.WaitForAvailable`, Volume/Snapshot `WaitForAvailable` | `available` | `error` | 없음 | 2초 |
| `image.WaitForState`, Image v2 `WaitForState` | 호출자가 지정 | `ERROR` | 없음 | 2초 |
| 위 서비스의 `WaitForDelete` | 실제 미존재 또는 성공한 nil/`deleted` 응답 | 상태 실패 목록 없음 | 120초 | 2초 |

시간 제한이 없다는 것은 SDK가 deadline을 추가하지 않는다는 뜻입니다. 전달한 context의 취소·deadline은 항상 적용됩니다. 성공 상태와 실패 상태는 대소문자를 구분하지 않고 비교하며, 성공 대상과 실패 상태가 같으면 성공을 먼저 판정합니다.

Compute Server, Block Storage v2/v3 Volume·Snapshot, Image v2 Image leaf에서 위 메서드를 사용할 수 있습니다. 패키지 함수 `compute.WaitForState[T]` / `WaitForDelete[T]`, `blockstorage.WaitForState[T]` / `WaitForAvailable[T]` / `WaitForDelete[T]`, `image.WaitForState[T]` / `WaitForDelete[T]`는 해당 서비스의 기존 `*resource.Collection[T]`를 받습니다. Connection의 수동 Server collection에도 `WaitForState`, `WaitForServerState`, `WaitForServer`, `WaitForDelete`가 있습니다.

## Server 대기

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/compute/v2/servers"
    "gophercloudsdk/resource"
)

func waitForServer(ctx context.Context, client *gophercloud.ServiceClient, id string) (*servers.Server, error) {
    return servers.New(client).WaitForServer(ctx, resource.ID(id),
        resource.WithTimeout(time.Minute))
}
```

`WaitForServer`는 기본적으로 ACTIVE를 최대 120초 기다립니다. 위 코드는 호출별 제한을 1분으로 바꿉니다. 다른 상태가 필요하면 시간 제한 없는 `WaitForState(ctx, ref, target, options...)` 또는 120초 제한의 `WaitForServerState`를 사용합니다. 기본 `ERROR` 외의 상태를 서비스 전체의 실패 상태로 추정하지 않습니다.

고정 Python의 대응 호출은 다음과 같습니다. Python은 Resource 객체를 받으며, Go는 명시적인 ID/Name 참조를 받습니다.

```python
server = conn.compute.wait_for_server(server, wait=60)
# 다른 상태: conn.compute.wait_for_status(server, status="SHUTOFF")
```

## Volume 준비·삭제 대기

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/blockstorage/v3/volumes"
    "gophercloudsdk/resource"
)

func waitForVolume(ctx context.Context, client *gophercloud.ServiceClient, id string) (*volumes.Volume, error) {
    bounded, cancel := context.WithTimeout(ctx, 10*time.Minute)
    defer cancel()
    return volumes.New(client).WaitForAvailable(bounded, resource.ID(id))
}

func waitForVolumeDeletion(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
    return volumes.New(client).WaitForDelete(ctx, resource.ID(id),
        resource.WithUnlimitedWait())
}
```

`WaitForAvailable` 자체는 시간 제한이 없지만 첫 함수의 parent context가 최대 10분으로 제한합니다. 둘째 함수는 삭제 대기의 기본 120초 제한을 제거합니다. `WaitForDelete`는 삭제 요청을 보내지 않으므로 실제 삭제는 별도 호출로 시작합니다. v2/v3 Volume·Snapshot은 같은 정책을 사용합니다.

```python
volume = conn.block_storage.wait_for_status(volume)  # available, wait=None
conn.block_storage.wait_for_delete(volume, wait=None)
```

## 호출별 옵션과 오류

SDK 기본 옵션 뒤에 caller 옵션을 순서대로 적용합니다. `WithTimeout`과 `WithUnlimitedWait`를 함께 전달하면 마지막 옵션이 유효합니다. `WithFailureStates("error", "aborted")`는 기본 실패 목록을 교체하고, 인자 없는 `WithFailureStates()`는 상태 실패 판정을 끕니다. 삭제 대기는 상태 실패 목록을 사용하지 않습니다.

`WithPollInterval`은 조회 간격을, `WithStatusAttribute`는 모델의 단일 exported string 상태 필드를 선택합니다. `WithProgressCallback`은 첫 조회를 포함한 각 비종료 응답에서 실행하며 성공·실패 응답에는 실행하지 않습니다. 선택한 progress가 없거나 nil이면 0입니다. callback은 동기적으로 실행됩니다. callback, 간격, 시간 제한, 상태 필드에 대한 Go 검증은 [공통 Resource 문서](../resource/README.md)를 따릅니다.

네이티브 Cinder v3 Snapshot의 `Progress`는 string이므로 `WithProgressCallback`을 지정하면 HTTP 전에 `ErrUnsupported`입니다. progress가 없는 v2 Snapshot·Volume·Image 모델은 0을 보고하며, Server의 정수 progress는 실제 값을 보고합니다. Python의 임의 progress 값에 대한 callback을 그대로 재현하지 않습니다.

명시적인 ID는 첫 GET부터 고정합니다. Name은 기존 collection의 조회로 한 번 해결한 뒤 그 ID를 고정합니다. Cinder v2 Snapshot의 기존 Name resolver는 네이티브 단일 페이지 정책을 유지합니다. 응답의 ID가 바뀌어도 다음 조회 대상은 변경하지 않습니다. 이미 대상 상태인 응답은 즉시 반환하고 추가 조회나 callback을 하지 않습니다.

삭제 대기는 실제 404, 성공한 nil, `deleted` 상태에서 끝납니다. 인증·서버·전송·받아들인 응답의 해석/읽기 실패와 context 취소는 삭제 완료로 처리하지 않습니다. 내부 원인에 404 또는 `ErrNotFound`가 들어 있어도 terminal 오류는 반환합니다. Go 삭제 대기의 결과는 error뿐이며 삭제 전 모델을 합성해 반환하지 않습니다.

## 기존 API와 Python의 남은 차이

기존 `Collection.Wait` / `WaitDeleted`와 leaf `WaitFor` / `WaitForDeletion`은 공통 5분·2초 정책을 유지합니다. 별도의 네이티브 `API.WaitForStatus`는 원래 Gophercloud의 error-only 반환, context 기반 시간 제한, 1초 간격, 대소문자를 구분하는 상태 비교를 유지합니다. 이 문서의 서비스 대기 API는 그 메서드의 대체 구현이나 ABI 변경이 아닙니다.

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Compute 대기](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3244-L3349), [Cinder v2 대기](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2373-L2435), [Cinder v3 대기](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3837), [Image v2 상태·삭제 대기](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2299-L2358)입니다. 이는 typed Go collection의 정책 대응이며 Python Resource 전체를 재현하지 않습니다.

Python은 supplied Resource의 cached 현재 상태가 대상이면 HTTP 없이 같은 객체를 반환할 수 있습니다. Go는 ID의 fresh GET 또는 Name의 기존 native List/Find 결과로 시작하며 mutable Resource cache를 받지 않습니다. Python의 nullable 상태, 임의 descriptor 속성·progress 값, `skip_cache=True`와 injected session은 Go의 typed 모델·필드 접근과 다릅니다. Python 삭제 대기의 Resource 반환과 Go의 error-only 반환도 다릅니다.

Python `interval=0`의 내부 보정과 0/음수 timeout을 그대로 재현하지 않습니다. Go는 양수 `WithPollInterval` / `WithTimeout`을 요구하며 무제한은 `WithUnlimitedWait`로 명시합니다. 문자열 상태의 missing/null이 네이티브 모델에서 합쳐질 수 있다는 한계도 유지됩니다.

Image v1은 현재 SDK의 typed binding이 없습니다. Image v2 `wait_for_task`의 특정 396 오류 후 task 재생성, 새 ID로 이어가는 공통 시간 예산은 별도 기능입니다. 이 상태 대기 API는 그 작업을 구현하지 않습니다. Cloud의 `wait_for_server` 180초·IP 할당, `wait_for_image` 3600초, 생성·업로드 workflow의 기존 정책도 별도 계약으로 남습니다.
