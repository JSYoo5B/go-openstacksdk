# Cinder connector dict: 초기화와 종료

`Volumes.InitializeWithConnector`와 `Volumes.TerminateWithConnector`는 openstacksdk의 `init_volume_attachment`·`terminate_volume_attachment`에 대응합니다. native `InitializeConnection`·`TerminateConnection`의 여덟 개 typed connector 입력을 유지하면서, 이 두 호출은 nested connector JSON을 직접 전달합니다. `wwnns` 배열, `nqn`, `uuid`, backend extension, null·false·0·빈 배열을 생략하거나 문자열로 변환하지 않습니다.

```python
import openstack

conn = openstack.connect()
connector = {
    "host": "compute-1",
    "wwnns": ["1000000000000001", "1000000000000002"],
    "nqn": "nqn.2014-08.org.example:compute-1",
    "do_local_attach": False,
    "extensions": {"optional": None, "number": 0, "devices": []},
}
connection_info = conn.block_storage.init_volume_attachment("volume-id", connector)
conn.block_storage.terminate_volume_attachment("volume-id", connector)
```

동일한 Go 예제는 standalone 프로그램으로 빌드할 수 있습니다. 실행에는 인증 환경과 연결 작업이 가능한 기존 volume이 필요합니다.

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    connector := map[string]any{
        "host": "compute-1",
        "wwnns": []string{"1000000000000001", "1000000000000002"},
        "nqn": "nqn.2014-08.org.example:compute-1",
        "do_local_attach": false,
        "extensions": map[string]any{"optional": nil, "number": 0, "devices": []any{}},
    }
    initialized, err := cinder.Volumes.InitializeWithConnector(ctx, "volume-id", connector,
        volumes.WithConnectorHeader("X-Client-Request-Id", "connector-init"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("connection_info=%s action_status=%d microversion=%s discovery=%d\n",
        initialized.ConnectionInfo, initialized.Response.StatusCode,
        initialized.Microversion, len(initialized.Discovery))

    terminated, err := cinder.Volumes.TerminateWithConnector(ctx, "volume-id", connector)
    if err != nil { log.Fatal(err) }
    fmt.Printf("terminate action_status=%d body=%q\n",
        terminated.Response.StatusCode, terminated.Response.Body)
}
```

| Python | Go | 응답 |
|---|---|---|
| `init_volume_attachment(id, connector)` | `InitializeWithConnector(ctx, id, connector, options...)` | raw `ConnectionInfo`와 실제 action 응답 |
| `terminate_volume_attachment(id, connector)` | `TerminateWithConnector(ctx, id, connector, options...)` | 실제 action 응답, `ConnectionInfo`는 nil |
| native typed connector 호출 | 기존 `InitializeConnection`·`TerminateConnection` | 기존 typed 모델과 Gophercloud 계약 유지 |

두 호출은 Resource를 GET하지 않고 `POST /volumes/{id}/action`을 보냅니다. 초기화 본문은 `{"os-initialize_connection":{"connector":...}}`, 종료는 `{"os-terminate_connection":{"connector":...}}`입니다. nil map은 connector `null`, 빈 map은 `{}`이고 map 내부의 값 역시 그대로 JSON에 남습니다. 객체의 ID는 하나의 unescaped URL segment이어야 하며 이름 검색은 수행하지 않습니다.

입력 JSON은 서비스 source를 먼저 캡처한 후 options를 호출하기 전에 snapshot합니다. custom `MarshalJSON`은 이 시점에 호출되고, 이후 option이 caller의 nested map이나 slice를 바꾸어도 전송 본문은 바뀌지 않습니다. JSON으로 표현할 수 없는 입력은 HTTP 전에 오류입니다. `json.RawMessage`를 nested 값으로 사용하면 큰 정수나 구조를 Go floating-point decode 없이 그대로 전달할 수 있습니다. headers는 `WithConnectorHeader`·`WithConnectorHeaders`로 지정하며 인증·framing·선택한 microversion의 충돌은 사전 오류입니다.

이미 선택한 client microversion이 있으면 추가 discovery 없이 POST합니다. 비어 있으면 기존 Cinder discovery 규칙으로 Resource ceiling `3.71`과 서버 광고 범위에서 version을 선택합니다. 실제 discovery 응답은 `Discovery`, POST 응답은 `Response`에 따로 보관합니다. 원래 shared client의 microversion이나 headers를 변경하지 않습니다. protocol version header와 JSON 본문은 retry callback 이후에도 유지되어야 합니다. native clean retry는 허용하지만 callback이 해당 본문이나 version을 바꾸면 원래 HTTP cause를 보존하고 다음 전송을 막습니다.

pinned `Volume.init_attachment`의 `cast(dict, resp['connection_info'])`는 runtime 타입 검사가 아닙니다. Go의 `ConnectionInfo`도 object뿐 아니라 JSON null·scalar·배열을 raw bytes로 유지합니다. nonnil bytes `null`은 성공 응답에서 명시한 null이고, nil은 미관측·누락·실패를 뜻합니다. field가 없거나 root 응답이 object가 아니면 실제 POST 증거를 가진 decode 오류입니다. 반환된 `ConnectionInfo`와 `Response.Body`는 별도의 bytes입니다.

성공 범위는 pinned `_action`처럼 200–399입니다. 초기화는 성공 body의 JSON을 읽고, 종료는 opaque body를 유지합니다. 이 응답은 connector 작업 ACK이며 volume 상태·Nova attachment 완료·로컬 장치 연결을 증명하지 않습니다. Source가 종료에서 `None`을 반환하는 것과 달리 Go는 문제 조사에 사용할 실제 HTTP receipt를 돌려줍니다. accepted 응답의 read/Close/cancel/source 변경은 `resource.ResponseError`에 status·body·headers와 cause가 남고, rejected 응답도 native HTTP 증거와 물리 원인을 함께 보존합니다.

계약은 [connector 테스트](volumes/python_connector_test.go)의 nested snapshot, null/empty/scalar, Source 성공 범위, 실패 body, Marshal/options 경계, 실제 Read·Close·cancellation·source 변경, retry ownership, discovery receipt로 검증합니다. 테스트는 기존 `pythonVolumeAPI`와 Gophercloud `testhelper` 기반 fixture를 재사용합니다. 클라우드 장치 연결 acceptance는 fixture 검증과 별도의 확인 단계입니다.
