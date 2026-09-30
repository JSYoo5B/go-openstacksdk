# 노드의 virtual media 조회

`GetVirtualMedia`는 Gophercloud Result 대신 `(*VirtualMedia, error)`를 반환합니다. Ironic microversion 1.93 이상이 필요하며 현재는 연결 옵션으로 microversion을 지정합니다.

```go
conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"),
    sdk.WithMicroversion("baremetal", "1.93"))
if err != nil { return err }
service, err := conn.BareMetal(ctx)
if err != nil { return err }
media, err := service.Nodes.GetVirtualMedia(ctx, nodeID)
if err != nil { return err }
fmt.Println(media.Image, media.Inserted, media.MediaTypes)
```

위 코드는 오류 반환 함수 안에서 사용합니다. `sdk`는 `gophercloudsdk`, `fmt`는 표준 라이브러리입니다. `ctx context.Context`와 `nodeID string`을 준비합니다.

이미지가 연결되지 않으면 `Image`는 빈 문자열, `Inserted`는 false입니다. `Header`는 복사된 응답 헤더이며 `Body`는 추가 필드를 포함한 JSON snapshot입니다. HTTP 오류와 JSON 타입 오류는 error로 반환하고 모델은 nil입니다. microversion 자동 협상은 아직 제공하지 않습니다.

고정한 openstacksdk 비교 목록에는 이와 직접 대응하는 Proxy 메서드가 없습니다. 이 API는 Gophercloud v2.15.0의 virtual media 연산을 SDK 응답 모델로 제공하며 Python과의 기능 동등성을 뜻하지 않습니다.
