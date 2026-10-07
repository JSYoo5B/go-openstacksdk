# 데이터 볼륨 분리

`conn.DetachVolume`은 Nova v2 서버에서 기존 Cinder v3 볼륨을 분리하고, 기본적으로 볼륨의 `available` 상태까지 기다립니다. 라이브러리가 이름 해석, Nova 분리 요청과 Cinder 상태 대기를 처리합니다. `resource.ID` 또는 `resource.Name`으로 서버와 볼륨을 선택하며 별도 request builder를 구현하지 않습니다.

ID 입력의 기본 요청 순서는 Nova `DELETE /servers/{server_id}/os-volume_attachments/{volume_id}` 202 또는 204 → Cinder `GET /volumes/{volume_id}` 200입니다. 첫 대기 조회는 즉시 실행하고 필요하면 반복합니다. DELETE의 마지막 식별자는 attachment ID가 아닌 **볼륨 ID**입니다. ID 입력은 사전 서버·attachment·Cinder 조회를 만들지 않습니다.

다음 함수들은 같은 작업을 호출하는 별도 예입니다. `conn`은 [전체 README](../README.md)처럼 준비한 Connection입니다.

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func detachDefault(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.DetachVolumeResult, error) {
    return conn.DetachVolume(ctx, blockstorage.DetachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    })
}

func detachNovaOnly(ctx context.Context, nova *gophercloud.ServiceClient, serverID, volumeID string) (*blockstorage.DetachVolumeResult, error) {
    return blockstorage.DetachVolume(ctx, nova, nil,
        blockstorage.DetachVolumeRequest{
            Server: resource.ID(serverID), Volume: resource.ID(volumeID),
        }, blockstorage.WithDetachVolumeWait(false))
}

func detachWithDeadline(ctx context.Context, conn *sdk.Connection, input blockstorage.DetachVolumeRequest, progress func(int) error) (*blockstorage.DetachVolumeResult, error) {
    timeout, interval := 5*time.Minute, time.Second
    return conn.DetachVolume(ctx, input,
        blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{
            Timeout: &timeout, PollInterval: &interval,
            ProgressCallback: progress,
        }))
}

func detachWithClients(ctx context.Context, nova, cinder *gophercloud.ServiceClient) (*blockstorage.DetachVolumeResult, error) {
    return blockstorage.DetachVolume(ctx, nova, cinder,
        blockstorage.DetachVolumeRequest{
            Server: resource.Name("worker"), Volume: resource.Name("data"),
        })
}
```

Python의 실제 대응 API는 cloud helper인 `conn.detach_volume`입니다. helper가 서버와 볼륨의 ID를 읽도록 Resource 또는 dict를 전달합니다.

```python
server = conn.compute.get_server("server-id")
volume = conn.block_storage.get_volume("volume-id")
conn.detach_volume(server, volume)  # wait=True; success returns None
```

대기를 끄려면 같은 호출에 `wait=False`를 추가합니다. 이 고정 Python 구현은 선언된 `timeout` 인자를 실제 대기에 전달하지 않습니다. 따라서 `timeout=300`을 넣어도 helper의 SDK 대기는 무제한이며, Go의 양수 Timeout과 같은 효과가 아닙니다. `conn.block_storage.detach_volume`은 Cinder의 별도 detach action으로, 이 Nova 서버 분리 helper와 다른 연산입니다.

## 기본값과 옵션

| 설정 | 기본값 | Go 선택 |
|---|---|---|
| 완료 대기 | 켜짐 | `WithDetachVolumeWait(false)`로 끔 |
| SDK 대기 시간 | 제한 없음 | `DetachVolumeWaitOpts.Timeout`에 양수 duration |
| 조회 간격 | 2초 | `DetachVolumeWaitOpts.PollInterval`에 양수 duration |
| 실패 상태 | 정확한 `error` | `DetachVolumeWaitOpts.FailureStates` |
| 진행 callback | 없음 | `DetachVolumeWaitOpts.ProgressCallback` |

`Wait == nil`은 기본 대기를 사용합니다. Timeout의 nil과 명시적인 zero는 SDK 대기 시간 제한을 두지 않습니다. 양수 timeout은 DELETE의 실제 접수 응답을 읽고 닫은 뒤 source 검사가 끝나면 시작합니다. 만료되면 대기 HTTP와 timer를 context로 취소합니다. 전체 이름 조회·분리 요청·대기의 시간 제한은 caller context로 지정합니다. 음수 timeout, 명시적인 zero·음수 PollInterval과 잘못된 실패 상태는 대기가 꺼져 있어도 요청 전에 거부합니다.

FailureStates의 nil은 `error`를 선택하고, nonnil empty slice는 해당 실패 검사를 끕니다. 목표 `available`과 실패 상태는 대소문자를 구별하지 않고 정확하게 비교하며 목표를 먼저 확인합니다. `error_deleting` 같은 접두사 상태를 자동으로 실패로 확대하지 않습니다. ProgressCallback은 대기 중인 응답에서만 `0`을 받으며, 반환한 오류나 context 취소는 대기를 끝냅니다. 확장 JSON의 임의 진행률 필드를 사용하지 않습니다.

`DetachVolumeWaitOpts`는 연결 작업의 `AttachVolumeWaitOpts`와 같은 concrete 대기 정책을 사용합니다. `WithDetachVolumeOptions`는 전체 설정을 교체하고 `WithDetachVolumeWaitPolicy`는 대기 설정 전체를 교체합니다. `WithDetachVolumeWait`는 Wait만 바꿉니다. 원래 옵션 callback은 순서대로 한 번 적용하고, pointer·slice를 복사한 정책으로 실행합니다. retained 설정 pointer와 재사용한 옵션 값은 진행 중인 작업을 바꾸지 않습니다.

`PrepareDetachVolumeOptions(ctx, options...)`는 HTTP와 서비스 초기화 없이 옵션을 한 번 적용하고 선택한 전체 정책을 소유·검사하여 `DetachVolumeOpts`를 반환합니다. 이를 `WithDetachVolumeOptions`로 재사용해도 원래 callback을 다시 실행하지 않습니다. Connection도 이 preparation을 서비스 선택 전에 수행하므로 잘못된 대기 설정은 endpoint·microversion discovery 전에 실패합니다. 직접 package 호출은 필요한 service client를 먼저 capture한 뒤 원래 옵션을 적용합니다.

Nova는 항상 필요합니다. Cinder는 기본 대기 또는 볼륨 이름 해석에만 필요합니다. `Wait(false)`와 볼륨 ID를 선택하면 package 함수는 nil Cinder로 동작하며 Connection은 Cinder를 초기화하지 않습니다. 서버 이름은 Nova로 해석하므로 그 자체로 Cinder를 요구하지 않습니다. 이름은 기존 Collection에서 각각 한 번 해석하고 해석한 ID를 이후 경로에 고정합니다.

## 접수 결과와 상태 대기

Nova 요청은 bodyless DELETE이며 202와 204만 접수합니다. 404는 없는 attachment를 나타내는 오류이고 성공으로 무시하지 않습니다. JSON 본문이나 force·connector·cascade 옵션, Cinder `os-detach` action, 볼륨 자체 DELETE를 대신 보내지 않습니다. 사전 상태·서버 association 검사도 추가하지 않습니다.

DELETE의 실제 응답 본문은 빈 값, JSON이 아닌 텍스트 또는 binary일 수 있습니다. SDK는 이를 JSON으로 해석하지 않고 `Deleted.Body`에 opaque 원문 bytes로 보존합니다. 원문 Header와 StatusCode도 별도로 복사합니다. 202·204는 서버가 요청을 받아들였다는 증거이며 분리가 완료되었다는 증거가 아닙니다.

| `DetachVolumeResult` 필드 | 의미 |
|---|---|
| `ServerID`, `VolumeID` | 이름 해석 후 고정한 대상 ID |
| `Deleted` | Nova DELETE의 실제 202·204 접수 응답 |
| `LastAccepted` | 대기에서 마지막으로 받아들인 Cinder 응답 |
| `Ready` | `available`을 확인한 실제 볼륨 모델 |

오류가 발생해도 result를 함께 확인합니다. result가 nonnil인 것만으로 DELETE 접수 여부를 판단하지 않습니다. `Deleted != nil`이고 StatusCode가 202 또는 204이면 실제 접수 증거가 있습니다. 응답 읽기·Close·context·source 검사 오류에서도 이 증거와 원래 오류를 함께 반환합니다. `Wait(false)`의 성공 결과에는 LastAccepted와 Ready가 없습니다.

대기는 고정한 볼륨 ID를 매번 Cinder에서 새로 조회합니다. canonical `id`가 선택한 ID와 일치하고 nonnull `status`가 있어야 합니다. 이 관찰에서 `attachments`의 생략·null·빈 배열은 모두 허용하며, 배열이 있으면 원문 JSON과 알려진 필드 형식을 검사합니다. available 성공 모델인 Ready는 LastAccepted와 독립적으로 소유합니다. 큰 숫자·알려지지 않은 속성·명시적인 null은 raw metadata에 보존하고 timestamp는 literal 문자열로 유지합니다.

`available`은 볼륨 전체의 상태입니다. multiattach 볼륨에서 대상 서버의 association이 사라졌어도 다른 연결 때문에 `in-use`이면 계속 기다립니다. Cinder 404나 볼륨 소멸은 성공한 분리로 처리하지 않습니다. guest device·hypervisor 정리까지 Ready가 증명하지는 않습니다.

대기 실패·취소·callback 오류가 발생하면 Deleted와 마지막 접수 관찰을 반환합니다. 거부된 HTTP 응답은 원래 native 오류의 상태·본문·header로 확인하며 LastAccepted를 덮어쓰지 않습니다. LastAccepted는 마지막으로 받아들인 Cinder 응답이며 모든 물리적인 전송 시도의 기록이 아닙니다. SDK는 접수한 분리 요청을 새 논리적 작업으로 재전송하거나, 자동 attach·볼륨 삭제·강제 detach로 보상하지 않습니다. caller가 보존한 결과로 후속 확인과 정리를 결정합니다.

## openstacksdk와의 범위

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [BlockStorageCloudMixin.detach_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L398-L429), [Compute delete_volume_attachment](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2542-L2578), [Cinder waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3815)와 [Resource waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591-L2667)입니다. Gophercloud는 v2.15.0에 고정합니다.

Python helper는 delete proxy에 string ID 두 개를 전달합니다. 그 proxy의 [호환 인자 검사](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2502-L2540)는 `find_server`로 ID를 검사하고 찾지 못하면 server·volume 순서를 바꾸기도 합니다. Go는 명시적인 Ref를 사용하며 인자를 추측해서 뒤집거나, 실패한 ID를 이름으로 바꿔 재조회하지 않습니다.

Python의 대기 전 `self.get_volume(id)`는 [cloud get_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L82-L122)을 통해 `find_volume`을 호출하고, 얻은 Resource가 이미 available이면 waiter가 cache에서 바로 반환할 수 있습니다. Go는 고정 ID로 새 상태를 확인하고 원문 관찰을 보존합니다. Python 성공은 None이며 Go는 접수·관찰·완료 결과를 반환합니다. Python에서 무시한 timeout은 Go의 양수 wait policy에서 실제 시간 제한으로 동작합니다.

이 직접 helper의 server·volume·wait·timeout과 Nova 요청·선택적 available 대기는 위의 Go 대응 정책으로 제공합니다. `go_mapping`은 이러한 의도적인 표현·동작 차이를 기록합니다. 별도의 Compute delete proxy, cloud get_volume, Cinder find·wait와 inherited Resource 기본값·descriptor/coercion·캐시·session/adapter의 전체 지원 판정은 각각 독립적입니다. 이름 해석은 기존 native Collection의 목록·pagination·전체 페이지 decoder 경계를 유지합니다. 이 workflow 하나로 다른 연산이나 전체 SDK 완료를 판정하지 않습니다.

선택한 client의 endpoint·ResourceBase·service type·microversion·provider를 capture하고 이후 source 교체를 검출합니다. 일반 source header는 복사하며 인증은 원래 provider의 현재 token을 사용합니다. configured native reauthentication·retry는 유지하고 SDK는 고정 method·URL·bodyless 요청과 접수 응답 소유권을 검사합니다. 임의 transport 내부 전송이나 packet-level exactly-once를 보장하지 않습니다. 설정은 동시 사용 전에 준비합니다.

[볼륨 연결](attach-volume.md)은 연결 전 fresh available·association 검사를 포함합니다. 분리는 위에서 설명한 DELETE와 available 관찰의 별도 계약입니다. 권한·quota·서버와 backend 상태, 실제 분리 가능 여부는 서버가 판단합니다.
