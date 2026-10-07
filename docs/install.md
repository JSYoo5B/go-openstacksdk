# 외부 Go 프로젝트에서 사용하기

Go 1.25 이상에서 공개 모듈 `github.com/JSYoo5B/gophercloudsdk`를 사용합니다. 전체 SDK는 개발 중이며, 지원 범위는 [구현 현황](implementation-plan.md)에서 확인합니다. `v0.1.0-alpha.1` tag는 아직 배포하지 않았습니다.

별도 Go 프로젝트에서 검증하려는 실제 커밋을 지정합니다. `SDK_COMMIT`은 이 저장소에 push된 revision으로 바꿉니다.

```sh
go mod init example.com/mycloud
SDK_COMMIT=<push된-커밋>
GOWORK=off go get github.com/JSYoo5B/gophercloudsdk@$SDK_COMMIT
GOWORK=off go build ./...
```

외부 소비자 검증에는 아래 main을 그대로 사용합니다. 공개 root·서비스·leaf·generic 옵션을 컴파일하며, 인증이나 HTTP 요청을 실행하지 않습니다. `CreateRecordOpts`의 공개 alias를 통해 concrete 속성과 SDK 소유 옵션을 사용할 수 있습니다. builder interface 구현은 필요하지 않습니다.

```go
package main

import (
    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/compute"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/containers"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/orders"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secrets"
    "github.com/JSYoo5B/gophercloudsdk/network"
    "github.com/JSYoo5B/gophercloudsdk/request"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func main() {
    _ = sdk.Connect
    _ = (*sdk.Connection).AttachVolume
    var _ *compute.Service
    var _ *network.FloatingIPs
    _ = resource.ID("server-id")
    _ = (*containers.API).CreateRecord
    _ = (*orders.API).CreateRecord
    _ = (*secrets.API).CreateRecord
    _ = containers.WithCreateRecordAttributes(map[string]any{"type": "generic"})
    _ = orders.WithCreateRecordAttribute("meta", map[string]any{"algorithm": "aes"})
    _ = secrets.WithCreateRecordAttribute("name", "example")
    var option containers.CreateRecordOption = func(config *request.Config[containers.CreateRecordOpts]) error {
        config.Options.Attributes = map[string]any{"name": "example"}
        return nil
    }
    _ = option
}
```

실제 호출 예제와 Python 비교는 [전체 README](../README.md), [Compute](../compute/README.md), [Network](../network/README.md), [Barbican 생성](../keymanager/v1/metadata-create.md)에 있습니다. HTTP 동작은 기존 Gophercloud fixture 기반 계약 테스트와 `make smoke`로 검증합니다.

## 검증 상태

모듈 namespace 변경과 기존 판정 보존을 확인했습니다. 외부 소비자의 local-replace 빌드와 push된 정확한 커밋의 replace 없는 설치는 별도 증거로 기록합니다. 설치·빌드 결과와 source SHA는 검증 완료 후 아래 표에 갱신합니다.

| 검사 | 상태 |
|---|---|
| 고정 소스·기존 판정 보존 | PASS:3,362개 fingerprint·513개 판정·188개 완료 유지 |
| SDK 재생성 일치 | PASS:생성기 race test·재생성 drift0 |
| 기존 전체 계약 검사·핵심 smoke | PASS:40개 test package·5흐름/9그룹; Go source SHA 보존 |
| 외부 module의 local-replace 빌드 | PASS:위 main 그대로 별도 module에서 빌드; consumer에만 replace |
| push된 정확한 커밋의 replace 없는 설치·빌드 | 대기 |
