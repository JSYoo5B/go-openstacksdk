# 전체 API 파사드

Gophercloud **v2.15.0**에서 서비스 클라이언트를 받는 공개 함수 **1,126개**를 194개 리소스 패키지에 제공합니다. [연산 목록](gophercloud_inventory.json)은 생성기가 같은 소스와 타입 정보로 만듭니다. `make generate`로 재생성하고 `make check`로 검증합니다.

각 패키지의 `New(client)`는 클라이언트를 보관하는 API 객체를 만듭니다. builder 입력은 SDK가 concrete options로 대체하고, 단순 조회의 선택 옵션은 `WithListOptions` 등의 함수로 제공합니다. 목록은 `iter.Seq2`, 일반 응답은 값과 `error`, 다운로드는 닫을 수 있는 스트림을 반환합니다. 여러 입력 builder가 필요한 Nova scheduler hints도 `WithCreateHintOpts`로 전달합니다.

[Keystone v2 인증](../identity/v2/tokens/README.md)은 토큰·사용자·catalog를 함께 반환합니다. [Nova 암호 조회](../compute/v2/servers/README.md)는 암호화된 문자열을 기본값으로 반환하고 복호화를 옵션으로 선택합니다. 이처럼 단일 extractor로 해석할 수 없는 응답도 SDK가 처리합니다.

```go
import (
    "context"
    "gophercloudsdk/network/v2/ports"
)

func example(ctx context.Context, api *ports.API) error {
    port, err := api.Create(ctx,
        ports.CreateOpts{NetworkID: "network-id", Name: "worker"},
        ports.WithCreateField("binding:host_id", "node-1"),
    )
    if err != nil { return err }
    _ = port
    for port, err := range api.List(ctx,
        ports.WithListOptions(ports.ListOpts{Limit: 100}),
        ports.WithListQuery("tags", "worker"),
    ) {
        if err != nil { return err }
        _ = port
    }
    return nil
}
```

`WithCreateField`는 기본 입력 필드 덮어쓰기를 거부하며 JSON 값을 옵션 생성 시 복사합니다. 확장 스키마와 microversion 요구 사항은 OpenStack 서비스가 검증합니다. 기본 필드는 concrete options로 지정합니다. HTTP 응답 오류는 `errors.As`로 원래 Gophercloud 오류까지 접근할 수 있습니다.

이 디렉토리의 완전성은 **고정한 Gophercloud 공개 연산에 대한 API 지원**을 뜻합니다. Python openstacksdk의 리소스 모델, 이름 해석, 복합 작업, 추가 서비스 지원까지 동등하다는 뜻은 아닙니다. API 파사드와 상위 SDK 지원은 별도로 추적합니다. 인증 관련 함수와 URL 도우미도 목록에 포함되므로 연산 수는 HTTP endpoint 수와 같지 않습니다.

계약 테스트는 Neutron 확장 필드와 페이지 순회, Nova의 scheduler hints, Glance의 JSON Patch, Swift의 업로드 및 다운로드 본문을 로컬 HTTP 서버에서 검증합니다. 전체 API 컴파일과 `go vet`도 수행합니다. 모든 연산의 실제 클라우드 동작을 검증한 것은 아닙니다.
