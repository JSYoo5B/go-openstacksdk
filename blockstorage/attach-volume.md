# 데이터 볼륨 연결

`conn.AttachVolume`은 기존 Cinder v3 볼륨을 Nova v2 서버에 연결하고, 기본적으로 볼륨의 `in-use` 상태까지 기다립니다. 라이브러리가 이름 해석, 연결 전 상태 확인, Nova 요청과 Cinder 대기를 처리합니다. 입력은 `resource.ID` 또는 `resource.Name`이며 별도 request builder를 구현하지 않습니다.

ID 입력의 기본 요청 순서는 Cinder `GET /volumes/{id}` 200 → Nova `POST /servers/{id}/os-volume_attachments` 200 → Cinder `GET /volumes/{id}` 200입니다. 대기 중에는 마지막 조회를 반복합니다. 이름 입력은 먼저 기존 Collection으로 서버와 볼륨 이름을 각각 한 번 해석합니다. 명시적인 서버 ID는 추가 서버 GET을 만들지 않습니다.

다음 함수들은 같은 작업을 호출하는 별도 예입니다. `conn`은 [전체 README](../README.md)처럼 인증과 서비스 선택을 준비한 Connection입니다.

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "gophercloudsdk"
    "gophercloudsdk/blockstorage"
    "gophercloudsdk/resource"
)

func attachDefault(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.AttachVolumeResult, error) {
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    })
}

func attachWithoutWaiting(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.AttachVolumeResult, error) {
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    }, blockstorage.WithAttachVolumeDevice("/dev/vdb"),
        blockstorage.WithAttachVolumeWait(false))
}

func attachWithDeadline(ctx context.Context, conn *sdk.Connection) (*blockstorage.AttachVolumeResult, error) {
    timeout, interval := 5*time.Minute, time.Second
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.Name("worker"), Volume: resource.Name("data"),
    }, blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{
        Timeout: &timeout, PollInterval: &interval,
    }))
}

func attachWithClients(ctx context.Context, nova, cinder *gophercloud.ServiceClient, input blockstorage.AttachVolumeRequest) (*blockstorage.AttachVolumeResult, error) {
    return blockstorage.AttachVolume(ctx, nova, cinder, input)
}
```

Python의 실제 대응 API는 cloud helper인 `conn.attach_volume`입니다. 서버와 볼륨 Resource 또는 dict를 먼저 전달합니다.

```python
server = conn.compute.get_server("server-id")
volume = conn.block_storage.get_volume("volume-id")
attachment = conn.attach_volume(server, volume)  # wait=True, timeout=None
```

대기를 끄려면 같은 호출에 `wait=False`를, 대기 시간을 정하려면 `timeout=300`을 추가합니다. 장치 선택은 `device="/dev/vdb"`이며 Go의 `WithAttachVolumeDevice`에 대응합니다. 이 helper를 `conn.block_storage.attach_volume`이라는 Proxy API로 해석하지 않습니다.

## 기본값과 옵션

| 설정 | 기본값 | Go 선택 |
|---|---|---|
| 장치 | 요청에서 생략 | `WithAttachVolumeDevice("/dev/vdb")` |
| 완료 대기 | 켜짐 | `WithAttachVolumeWait(false)`로 끔 |
| SDK 대기 시간 | 제한 없음 | `AttachVolumeWaitOpts.Timeout`에 양수 duration |
| 조회 간격 | 2초 | `AttachVolumeWaitOpts.PollInterval`에 양수 duration |
| 실패 상태 | 정확한 `error` | `AttachVolumeWaitOpts.FailureStates` |
| 진행 callback | 없음 | `AttachVolumeWaitOpts.ProgressCallback` |

`Wait == nil`은 기본 대기를 사용합니다. Timeout의 nil과 명시적인 zero는 SDK 대기 시간 제한을 두지 않습니다. 양수 timeout은 Nova 접수 응답을 읽고 검증한 뒤 시작하고, 만료되면 진행 중인 대기 HTTP와 timer를 context로 취소합니다. Python의 `timeout=None`도 무제한이지만 숫자 `timeout=0`은 이미 목표 상태인 Resource의 단축 반환을 제외하면 첫 poll 전에 timeout입니다. Python의 음수 timeout도 연결 요청 이후 대기에서 즉시 timeout으로 끝나는 반면, Go는 음수 duration을 연결 전 검사에서 거부합니다. 이 numeric zero·음수 정책은 Go SDK의 명시적인 대응 차이입니다. 전체 이름 조회·상태 확인·연결 요청·대기의 시간 제한은 caller context로 지정합니다. PollInterval의 nil은 2초이며 명시적인 zero와 음수는 오류입니다. 대기를 꺼도 잘못된 대기 설정을 사전에 거부합니다.

FailureStates의 nil은 `error`를 선택하고, nonnil empty slice는 해당 실패 검사를 끕니다. 목표 `in-use`와 실패 상태는 대소문자를 구별하지 않고 정확하게 비교하며, 목표 상태를 먼저 확인합니다. `error_deleting` 같은 접두사 상태를 자동으로 실패로 확대하지 않습니다. 연결 전 상태는 정확한 lowercase `available`이어야 합니다.

ProgressCallback은 대기 중인 응답에서만 호출하고 `0`을 전달합니다. 이 흐름에는 canonical Cinder 진행률 필드가 없으므로 원문 확장 속성을 진행률로 추정하지 않습니다. callback이 반환한 오류나 context 취소는 대기를 끝내며 이미 받은 응답은 유지합니다.

`WithAttachVolumeOptions`는 전체 설정을 교체하고, `WithAttachVolumeWaitPolicy`는 대기 설정 전체를 교체합니다. Device와 Wait helper는 자신의 필드만 바꿉니다. 옵션을 순서대로 한 번 적용한 뒤 pointer와 slice를 복사하여 실행합니다. 재사용한 설정이나 callback에 남은 설정 pointer로 진행 중인 요청을 바꾸지 않습니다. 모든 설정은 이름 해석과 첫 Cinder 상태 요청 전에 검사합니다. Connection의 서비스 초기화·microversion discovery는 이 검사보다 먼저 발생할 수 있습니다.

## 상태 확인과 실제 응답

SDK는 대기를 끈 경우에도 연결 직전에 Cinder를 조회합니다. 실제 volume의 `id`가 선택한 볼륨 ID와 일치하고, `status`가 `available`이며, 명시적인 nonnull `attachments` 배열에 대상 서버의 기록이 없는지 확인합니다. 빈 배열은 허용합니다. 같은 서버 기록의 device가 비어 있거나 생략되어도 다시 연결하지 않습니다. 이 확인과 Nova 요청은 원자적이지 않으므로 동시에 발생한 다른 변경은 서버 오류로 반환될 수 있습니다.

Nova 본문은 `volumeAttachment.volumeId`와 선택한 nonempty device로 구성합니다. 이 helper는 Nova의 서버 연결 API를 사용하며 Cinder attachment 생성 API를 호출하지 않습니다. 응답의 `volumeId`는 선택한 볼륨과 일치해야 하고, nonnull `serverId`가 있으면 선택한 서버와 일치해야 합니다. 응답의 attachment ID나 Location은 이후 조회 경로를 바꾸지 않습니다. tag와 delete-on-termination 요청 옵션은 기존 [Nova VolumeAttachment API](../compute/v2/README.md)에서 사용합니다.

| `AttachVolumeResult` 필드 | 의미 |
|---|---|
| `ServerID`, `VolumeID` | 이름 해석 후 고정한 대상 ID |
| `Checked` | 연결 전 Cinder의 실제 접수 응답 |
| `Created` | Nova 연결 요청의 실제 접수 응답 |
| `LastAccepted` | 대기에서 마지막으로 받아들인 Cinder 응답 |
| `Ready` | `in-use`를 확인한 실제 볼륨 모델 |

Checked·Created·LastAccepted는 실제 Body·Header·StatusCode와 검증한 모델을 각각 보존합니다. 응답 읽기·Close·JSON·필드 검증에 실패하면 원문 증거와 error를 함께 반환하며 typed 모델은 완전히 decode한 뒤에만 채우고, 이후 상태·대상 검사에서도 오류를 반환할 수 있습니다. 단계 간 원문, header, metadata와 모델은 별도로 소유합니다. nullable pointer는 값과 미존재를 구별하고, metadata의 raw JSON은 명시적인 null과 알려지지 않은 필드·큰 숫자를 보존합니다. timestamp는 원문의 문자열입니다.

오류가 발생해도 result를 함께 확인합니다. result가 nonnil인 것만으로 Nova 연결이 접수되었다고 판단하지 않습니다. `Created != nil`이고 StatusCode가 200이면 실제 접수 증거가 있으며, `Created.Attachment != nil`은 모델 검증까지 끝났다는 뜻입니다. `Ready != nil`은 상태 대기가 성공했다는 뜻입니다. `Wait(false)`의 성공 결과에는 Ready와 LastAccepted가 없습니다.

대기 실패·취소·callback 오류가 발생하면 이미 받은 Created와 마지막 접수 관찰을 반환합니다. 거부된 HTTP 응답은 원래 native 오류의 상태·본문·header로 확인하며, LastAccepted를 거부 응답으로 덮어쓰지 않습니다. LastAccepted는 모든 물리적인 전송 시도나 마지막 실패 응답의 기록을 뜻하지 않습니다. 자동 detach·delete·새 연결 요청은 수행하지 않습니다. caller가 보존한 접수 결과를 바탕으로 후속 확인과 정리를 결정합니다.

Nova 원문에 `id`가 없으면 `Created.Attachment.ID`는 nil입니다. 볼륨 대상은 `Created.Attachment.VolumeID`와 result의 `VolumeID`로 확인합니다. Python Resource의 alternate ID fallback과 URI의 server identity를 응답에 합성하지 않으며, 선택한 서버 ID는 result의 `ServerID`에 남습니다.

`in-use`는 Cinder에서 관찰한 상태이며 대상 서버의 연결 완료 여부까지 증명하지 않습니다. 연결 전 기록 검사와 Nova 응답의 대상 검증은 별도이고, 다른 요청과 동시에 상태가 바뀔 수 있습니다.

## openstacksdk와의 범위

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [BlockStorageCloudMixin.attach_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L431-L492)와 [Cinder wait_for_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3815), [공통 Resource waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591-L2667)와 [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L102)입니다. Gophercloud는 v2.15.0에 고정합니다.

Python은 전달한 mutable Resource/dict의 상태와 attachment 기록을 먼저 읽습니다. 같은 서버의 device가 truthy인 경우만 중복 연결로 거부하고, Resource의 캐시된 상태가 이미 목표이면 waiter가 바로 반환할 수 있습니다. Go는 Ref를 해석한 뒤 Cinder에서 새로 확인하고, device 유무와 관계없이 같은 서버 기록을 거부하며, 대기에서도 새 응답을 조회합니다. Python은 대기 실패 때 예외를 던지며 Nova attachment를 반환하지 않습니다. Go는 접수 결과와 오류를 함께 남깁니다.

이 직접 선언된 helper의 server·volume·device·wait·timeout과 연결 전 확인·Nova 요청·선택적 상태 대기는 위의 Go 대응 정책으로 제공합니다. `go_mapping`은 이러한 의도적인 표현·동작 차이를 기록한 판정입니다. Python Resource 기본값·descriptor/coercion·alternate identity·dirty state·캐시·주입 session/adapter를 같은 객체 모델로 재현했다는 뜻은 아닙니다. 전체 Compute attachment 생성·Cinder 조회·공통 waiter와 inherited Resource 동작의 지원 판정은 각각 독립적으로 남습니다. 이름 조회는 기존 Collection의 native 목록·pagination·전체 페이지 decoder를 사용합니다. volume 생성·분리·삭제를 조합하는 별도 작업도 계속 구현할 대상입니다.

Nova와 Cinder client의 endpoint·ResourceBase·service type·microversion·provider 선택은 호출 시작 때 고정하고 다음 단계에서 원래 source의 교체를 검출합니다. 일반 source header는 시작 시 복사하며 인증은 원래 provider의 현재 token을 사용합니다. JSON 요청은 configured provider의 reauthentication·retry 정책을 사용하고, SDK는 선택한 method·URL·JSON 본문과 응답 소유권을 검증합니다. 임의 transport 내부 전송이나 모든 요청의 packet-level exactly-once를 보장하지 않습니다. client/provider의 설정은 동시 사용 전에 준비합니다.

Python의 양수 timeout은 poll 사이에 시간을 검사하며 이미 진행 중인 fetch나 sleep를 그 deadline으로 중단하지 않습니다. Go의 대기 context는 HTTP와 timer에도 적용되므로 같은 숫자 timeout의 정확한 중단 시점은 다를 수 있습니다.

권한·quota·볼륨 backend·서버 상태와 실제 연결 가능 여부는 서버가 판단합니다. 이 helper의 계약 테스트와 대응 판정은 deployment 성공이나 다른 연산을 포함한 전체 SDK 완료의 증거로 확대하지 않습니다.
