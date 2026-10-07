# 외부 Go 프로젝트에서 사용하기

Go 1.25 이상에서 공개 모듈 `github.com/JSYoo5B/gophercloudsdk`를 사용합니다. 전체 SDK는 개발 중이며, 지원 범위는 [구현 현황](implementation-plan.md)에서 확인합니다. `v0.1.0-alpha.1` tag는 아직 배포하지 않았습니다.

새 프로젝트에서 아래처럼 설치합니다. 기존 Go 프로젝트에서는 `go mod init`을 생략합니다. 이 커밋은 원격 설치·빌드를 확인한 revision입니다.

```sh
go mod init example.com/mycloud
GOWORK=off go get github.com/JSYoo5B/gophercloudsdk@3f0240534253ef06c7c60b708ac2ce8d3c100677
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

main을 `main.go`로 저장한 뒤 빌드합니다.

```sh
GOWORK=off go build -mod=readonly ./...
```

실제 호출 예제와 Python 비교는 [전체 README](../README.md), [Compute](../compute/README.md), [Network](../network/README.md), [Barbican 생성](../keymanager/v1/metadata-create.md)에 있습니다. HTTP 동작은 기존 Gophercloud fixture 기반 계약 테스트와 `make smoke`로 검증합니다.

## 검증 상태

2026-10-08에 checkout 밖의 독립 소비자 2개로 위 main을 빌드했습니다. local-replace 검증과 원격 설치 검증을 구분하며, 원격 소비자는 `GOWORK=off`이고 replace가 없습니다. 실제 설치 버전은 `v0.0.0-20261007215549-3f0240534253`이며 module cache에서 빌드했습니다. source SHA256은 `137a479b9613464c549cfe734a66b91a32a99bcb201ea886d3a068a183128d9d`입니다. 이 설치 단위 검증 당시 API 완료 수는188개였으며, 이후 판정으로 현재196개입니다. 최신 개수는 [구현 현황](implementation-plan.md)에서 확인합니다.

| 검사 | 상태 |
|---|---|
| 고정 소스·기존 판정 보존 | PASS:namespace 단위의3,362개 fingerprint·513개 판정·당시188개 완료 유지 |
| SDK 재생성 일치 | PASS:생성기 race test·재생성 drift0 |
| 기존 전체 계약 검사·핵심 smoke | PASS:40개 test package·5흐름/9그룹; Go source SHA 보존 |
| 외부 module의 local-replace 빌드 | PASS:위 main 그대로 별도 module에서 빌드; consumer에만 replace |
| push된 정확한 커밋의 replace 없는 설치·빌드 | PASS:`3f0240534253`; `go get`·`go build -mod=readonly` exit0, Replace 없음 |
