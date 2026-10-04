# 데이터 볼륨 삭제

`Connection.DeleteVolume`은 볼륨 확인, 선택한 microversion에 맞는 삭제 요청, 선택적 완료 대기를 하나의 작업으로 제공합니다. `blockstorage.DeleteVolume`은 같은 작업을 이미 준비한 Cinder client로 실행합니다. 직접 request builder를 구현할 필요 없이 `WithDeleteVolume…` 옵션으로 정책을 지정합니다.

기본값은 `Wait=true`, `Force=false`, 2초 polling, SDK 대기 시간 제한 없음입니다. 볼륨이 처음부터 없으면 오류 없이 `Found=false`, `Deleted=false`를 반환합니다. 볼륨을 찾았다면 삭제 요청 후 실제 GET의 404 또는 정확한 `deleted` 상태까지 기다립니다.

다음 Go 코드는 서로 다른 사용 방법을 담은 독립 실행 가능한 예제입니다. 원하는 함수를 골라 준비한 provider/client/Connection에 연결하면 됩니다. `forceDelete`에 `"3.23"`을 전달하면 현대 DELETE 분기, `"3.22"`를 전달하면 legacy force action 분기를 선택합니다. SDK helper 내부에서는 해당 버전 지원 여부를 새로 검색하거나 다른 분기로 재시도하지 않습니다.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/resource"
)

func defaultDelete(ctx context.Context, conn *sdk.Connection) (*blockstorage.DeleteVolumeResult, error) {
	return conn.DeleteVolume(ctx, blockstorage.DeleteVolumeRequest{
		Volume: resource.Name("data"),
	})
}

func noWaitDelete(ctx context.Context, cinder *gophercloud.ServiceClient) (*blockstorage.DeleteVolumeResult, error) {
	return blockstorage.DeleteVolume(ctx, cinder, blockstorage.DeleteVolumeRequest{
		Volume: resource.ID("volume-id"),
	}, blockstorage.WithDeleteVolumeWait(false))
}

func forceDelete(ctx context.Context, provider *gophercloud.ProviderClient, selectedMicroversion string) (*blockstorage.DeleteVolumeResult, error) {
	conn, err := sdk.FromProvider(provider,
		sdk.WithMicroversion(sdk.BlockStorage, selectedMicroversion),
	)
	if err != nil {
		return nil, err
	}
	return conn.DeleteVolume(ctx, blockstorage.DeleteVolumeRequest{
		Volume: resource.ID("volume-id"),
	}, blockstorage.WithDeleteVolumeForce(true), blockstorage.WithDeleteVolumeWait(false))
}

func waitDelete(ctx context.Context, conn *sdk.Connection, progress func(int) error) (*blockstorage.DeleteVolumeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	waitTimeout, interval := 90*time.Second, time.Second
	return conn.DeleteVolume(ctx, blockstorage.DeleteVolumeRequest{
		Volume: resource.ID("volume-id"),
	}, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{
		Timeout:          &waitTimeout,
		PollInterval:     &interval,
		ProgressCallback: progress,
	}))
}

func inspectDelete(result *blockstorage.DeleteVolumeResult, err error) {
	if err != nil {
		fmt.Println("삭제 작업 오류:", err)
	}
	if result == nil {
		return
	}
	fmt.Printf("volume=%s found=%t deleted=%t\n", result.VolumeID, result.Found, result.Deleted)
	if result.Located != nil {
		fmt.Println("초기 확인 HTTP:", result.Located.StatusCode)
	}
	if result.Deletion != nil {
		fmt.Println("삭제 요청 HTTP:", result.Deletion.StatusCode)
	}
	if result.LastAccepted != nil {
		fmt.Println("마지막 대기 응답 HTTP:", result.LastAccepted.StatusCode)
	}
	if result.Absent != nil {
		fmt.Println("GET 404 응답 HTTP:", result.Absent.StatusCode)
	}
	if result.Ready != nil && result.Ready.Status != nil {
		fmt.Println("완료 상태:", *result.Ready.Status)
	}
}

func main() {}
```

Python의 실제 cloud helper에서는 `conn.delete_volume("data", wait=True, timeout=None, force=False)`를 호출합니다. Go에서는 모호한 문자열 대신 `resource.Name` 또는 `resource.ID`를 명시하고, Python의 bool 반환과 오류 후 상태를 함께 읽을 수 있는 결과를 받습니다.

```python
import openstack

conn = openstack.connect()
deleted = conn.delete_volume("data", wait=True, timeout=None)
request_finished = conn.delete_volume("volume-id", wait=False, force=True)
```

| Python cloud 인자 | Go 사용법 |
| --- | --- |
| `name_or_id` | `DeleteVolumeRequest{Volume: resource.Name(...)}` 또는 `resource.ID(...)` |
| `wait=True` | 기본값. `WithDeleteVolumeWait(false)`로 완료 대기 생략 |
| `timeout=None` | 기본 무제한 대기. `WithDeleteVolumeWaitPolicy`의 `Timeout`으로 대기 구간 제한 |
| `force=False` | 기본값. `WithDeleteVolumeForce(true)`로 force 분기 선택 |
| bool 반환 | `result.Deleted`; 최초 존재 여부는 `result.Found` |

## 최초 존재 확인과 이름 선택

명시적 ID도 삭제 전에 고정 ID로 GET합니다. `Wait(false)`는 이 최초 확인을 생략하지 않습니다. 이름은 기존 Cinder Collection의 exact-name 조회로 ID를 한 번 결정한 뒤, 그 ID로 새 GET을 수행합니다. 이때 동일 이름의 여러 볼륨, 안전하지 않은 ID, 조회의 HTTP/transport/JSON 오류는 삭제 전에 반환합니다. ID와 이름을 추측해서 서로 바꾸거나 실패한 ID 조회를 이름 검색으로 전환하지 않습니다.

이름 조회의 정상적인 논리적 없음은 `Found=false`, `Deleted=false`이며 실제 404 응답이 없으므로 `Absent`를 만들지 않습니다. 최초 GET의 정상 404는 `Located`와 별도로 소유한 `Absent`에 실제 응답을 보존합니다. 200은 정확한 canonical `id`가 선택한 ID와 일치해야 하며, 상태는 없어도 됩니다. `available`, `in-use`, `error` 등 현재 상태를 이유로 삭제를 막지 않습니다. 서버가 요청의 가능 여부를 판단합니다.

Python `find_volume`은 문자열로 member GET을 먼저 시도하고 400/403/404일 때 목록의 ID 또는 이름을 찾을 수 있습니다. 이름 목록에서 얻은 Resource도 곧바로 삭제에 사용합니다. Go의 explicit Ref와 추가 확인 GET은 존재 판정 시점을 바꿉니다. 예를 들어 이름 조회 뒤 볼륨이 사라져 확인 GET이 404라면 Go는 처음부터 없던 작업으로 `Deleted=false`를 반환하고, Python은 삭제의 404를 무시한 뒤 성공 bool을 반환할 수 있습니다. 이 차이는 새 상태를 확인하는 Go 정책입니다.

## force와 microversion

분기는 선택된 Cinder client의 `Microversion`을 기준으로 고정합니다. 빈 값 또는 `3.23` 미만은 legacy, `3.23` 이상은 현대 분기입니다. 직접 package 호출은 별도의 discovery를 하지 않습니다. Connection의 기존 `WithLatestMicroversion`/range 설정은 cached client를 선택하는 과정에서 버전을 결정할 수 있고, helper는 그렇게 정해진 값을 사용합니다.

| 선택한 버전과 Force | 실제 요청 | 처리 가능한 응답 |
| --- | --- | --- |
| 빈 값 또는 `<3.23`, false | body 없는 `DELETE /volumes/{id}?cascade=false` | 202, 204, 정상 race 404 |
| `>=3.23`, false | body 없는 `DELETE /volumes/{id}?cascade=false&force=false` | 202, 204, 정상 race 404 |
| `>=3.23`, true | body 없는 `DELETE /volumes/{id}?cascade=false&force=true` | 202, 204, 정상 race 404 |
| 빈 값 또는 `<3.23`, true | `POST /volumes/{id}/action`, `{"os-force_delete":null}` | 201, 202, 정상 race 404 |

쿼리는 위의 lowercase bool 값만 사용하며 cascade는 항상 false입니다. 전체 정책은 cloud helper의 네 인자에 대응하며 cascade/query/body/header builder 옵션은 추가하지 않습니다. 빈 값 외의 microversion은 leading zero 없는 ASCII `3.<minor>` 형식이어야 합니다. 직접 package 호출에서는 다른 major, `latest`, malformed 값이나 정수 overflow를 원래 callback 또는 HTTP 전에 거절합니다. Connection은 옵션을 먼저 준비하고 client를 선택한 뒤 그 source를 검증합니다. 빈 Type의 직접 client는 복사본에서 `volumev3`로 정규화하여 선택한 microversion 헤더를 일관되게 보냅니다.

Python proxy는 endpoint가 보고하는 min/max와 adapter의 default microversion으로 `3.23` 지원 여부를 판단합니다. Go의 선택 버전 기준 분기와 이 discovery 정책은 서로 다릅니다. 서버가 선택한 요청을 거절하면 원래 오류를 반환하며 legacy/modern/normal/force 분기로 자동 전환하지 않습니다.

legacy body의 null은 pinned Python과 같습니다. 기존 Gophercloud native `ForceDelete`는 `{"os-force_delete":""}`를 보내고 native 일반 `Delete`는 false cascade를 생략할 수 있습니다. 이 cloud workflow는 자체의 null body와 명시적 `cascade=false`를 사용합니다. 기존 native leaf API는 별도 API로 사용할 수 있습니다.

## 대기와 취소

`Wait(true)`이면 삭제 요청을 정상 처리한 뒤 즉시 고정 ID로 GET하며 이후 기본 2초 간격으로 반복합니다. 삭제 단계의 정상 race 404도 대기를 켰다면 새 GET으로 이어집니다. GET의 정상 404 또는 정확한 case-insensitive `deleted` 상태가 완료 조건입니다. `error`, `error_deleting`, `in-use`는 완료 조건이 아니며 기다립니다. `DeleteVolumeWaitOpts`에는 `FailureStates`가 없습니다.

응답의 canonical ID는 항상 확인하지만 상태가 absent/null/빈 문자열이면 계속 기다립니다. 잘못된 known 필드 타입이나 다른 ID는 오류입니다. Python waiter도 실패 상태 predicate는 없지만, Resource에 status가 있고 값이 None이면 `.lower()` 호출에서 예외가 날 수 있습니다. Go는 nullable 모델의 상태를 비종료 관찰로 다룹니다. Python의 falsy fetch 결과나 mutable Resource/cache 상태를 별도의 Go 완료 모델로 만들지는 않습니다.

`Timeout`은 삭제 응답을 읽고 닫고 source 확인을 마친 뒤 시작하는 대기 구간의 제한입니다. 이름 선택·최초 확인 GET·삭제 요청에는 이 제한을 적용하지 않습니다. 호출자 context는 전체 작업을 제한합니다. 예제의 5분 context와 90초 wait policy는 각각 전체 작업과 삭제 후 대기 구간에 적용됩니다. 음수 Timeout 또는 0 이하 PollInterval은 대기를 꺼도 요청 전에 거절합니다. nil/0 Timeout은 무제한입니다.

Python cloud helper도 timeout을 명시적으로 waiter에 전달하여 기본 None은 proxy의 120초 기본값을 덮어씁니다. 다만 Python의 숫자 0/음수 timeout은 삭제 뒤 첫 fetch 전에 즉시 timeout을 발생시킵니다. Go는 숫자 0을 무제한으로 다루고 음수를 사전에 거절합니다. 양수 Go 제한은 HTTP와 timer를 context로 중단하고, Python 반복문은 fetch 사이에서 시간을 검사합니다.

`ProgressCallback`은 비종료 200 관찰에서만 0을 받습니다. 완료한 404/deleted 상태는 callback을 호출하지 않습니다. callback 오류 또는 `context.WithCancelCause` 취소는 원래 오류와 custom cause를 보존하며 다음 polling을 막습니다. callback은 cloud helper에 없는 Go 확장입니다.

## 결과와 오류 후 처리

`Found`는 최초의 정상 200 확인 여부입니다. `Deleted`는 이 workflow의 성공 bool이며, `Wait(false)`에서는 삭제 단계의 요청 처리가 끝났다는 뜻입니다. 실제 소멸 확인이 필요하면 대기를 켜고 `Absent` 또는 `Ready`를 읽습니다. 초기 논리적 없음/정상 404는 두 bool 모두 false입니다. 최초 확인 이후 정상 mutation 404는 이미 사라진 race로 처리하므로 성공 경로에서 `Found=true`, `Deleted=true`가 될 수 있습니다.

| 결과 필드 | 의미 |
| --- | --- |
| `VolumeID` | 한 번 선택하고 고정한 안전한 ID. 이름의 논리적 없음에는 비어 있음 |
| `Located` | 최초 GET의 실제 200/404 raw body, cloned header, status. 정상 200 확인 후 `Volume` 채움 |
| `Deletion` | 실제 DELETE 또는 legacy force action 응답. opaque body를 JSON으로 해석하지 않음 |
| `LastAccepted` | 마지막으로 처리 대상으로 받아들인 polling GET 200/404. 다음 transport/rejected HTTP 오류가 덮어쓰지 않음 |
| `Absent` | 최초/대기 GET에서 실제 받은 404의 독립적인 raw 증거. mutation의 404를 빌리지 않음 |
| `Ready` | 실제 fresh 200의 `deleted` VolumeInfo를 독립적으로 디코드한 모델. 404에서는 만들지 않음 |

raw body/header/status는 응답을 읽거나 닫는 과정, JSON/identity 확인, context/source 확인이 실패해도 가능한 증거를 보존합니다. 따라서 `result != nil`과 `err != nil`이 함께 나올 수 있습니다. admitted 404에서도 read/Close/context/source 오류가 발생하면 정상 없음으로 바꾸지 않고 오류를 반환합니다. 중첩된 transport 오류의 404나 과거 응답의 404도 소멸 증거가 되지 않습니다.

`VolumeInfo`의 nullable 필드와 `Metadata.Body`/`MetadataFields`는 원본 JSON과 숫자 정밀도를 보존합니다. `Ready`, `LastAccepted.Volume`, raw 결과와 header는 독립적으로 소유합니다. polling 모델은 디코드됐지만 ID가 잘못되어 거절된 경우에도 `LastAccepted.Volume`에서 확인할 수 있습니다. 오류 후 SDK가 삭제를 다시 orchestration하거나 detach/cascade/rollback 작업을 추가하지 않습니다. 다음 조치는 보존한 응답과 원래 오류를 확인한 호출자가 결정합니다.

## 옵션, Connection, 원본 범위

`WithDeleteVolumeOptions`는 전체 정책을 교체하고 `WithDeleteVolumeWaitPolicy`는 대기 정책을 교체합니다. 옵션의 포인터와 reusable 정책을 복사하고 callback마다 결과를 다시 소유하여 retained handle이 이후 실행을 바꾸지 못하게 합니다. `PrepareDeleteVolumeOptions`는 HTTP 없이 유효한 기본값을 채우며 원래 callback을 한 번 적용합니다.

직접 package 호출은 필수 Cinder source와 일반 header를 원래 옵션 callback 전에 잡고, provider의 현재 인증을 사용합니다. Provider/Endpoint/ResourceBase/Type/Microversion이 바뀌면 다음 단계로 진행하지 않습니다. Connection은 context/receiver/Ref와 옵션을 먼저 준비한 뒤 cached Cinder v3를 선택하고 준비한 정책으로 위임합니다. NoWait도 Cinder와 최초 GET이 필요합니다. native reauthentication/backoff/retry는 provider에 유지하며, SDK는 고정한 method/URL/body/응답 정책을 보호합니다.

기준 원본은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud `v2.15.0`입니다. [실제 cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L280), [Cinder 삭제 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L933), [Resource 삭제 waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2670), [timeout 반복](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53), [microversion 지원 판단](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L242), [native Delete/ForceDelete](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go)를 비교합니다.

직접 cloud `delete_volume`의 인자와 흐름을 Go 방식으로 제공하는 단위입니다. Cinder find/delete/wait proxy, generic Resource/session/cache/version discovery와 Cinder v2 모델의 전체 의미는 별도 리뷰 대상입니다. 이 단위의 매핑이 해당 의존 선언이나 전체 SDK 완료를 뜻하지는 않습니다. [Block Storage 서비스](README.md), [native v3 API](v3/README.md), [생성](create-volume.md), [서버 연결](attach-volume.md), [연결 해제](detach-volume.md)도 함께 참고할 수 있습니다.
