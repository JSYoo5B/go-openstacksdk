# 외부 Go 프로젝트에서 사용하기

Go 1.25 이상에서 공개 모듈 `github.com/JSYoo5B/gophercloudsdk`를 사용합니다. 전체 SDK는 개발 중이며, 지원 범위는 [구현 현황](implementation-plan.md)에서 확인합니다. `v0.1.0-alpha.1` tag는 아직 배포하지 않았습니다.

새 프로젝트에서 아래처럼 설치합니다. 기존 Go 프로젝트에서는 `go mod init`을 생략합니다. 이 커밋은 원격 설치·빌드를 확인한 revision입니다.

```sh
go mod init example.com/mycloud
GOWORK=off go get github.com/JSYoo5B/gophercloudsdk@40dbb0eaa8f97a19eb0c7aca417bc4251db955e9
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

2026-10-08에 source revision `2490e4813a564b5d8163f15091eff5448817398f`을 새 외부 module에 replace 없이 설치하고 위 설치 main과 [Keypair 목록·검색 main](../compute/keypairs-list-find.md)을 빌드했습니다. `GOWORK=off`, get/build exit0·실제 버전 `v0.0.0-20261008020804-2490e4813a56`이며 module-cache Go source1,973개 SHA256 `bbb1c68d0a0a67e5a50a531247d85c0f13b71ca2b8ebf3d530ffbb3ffb3ef9b3`가 집중98그룹·전체42 package gate와 같습니다. 새3행12계약으로 현재 완료 수는224/3,362이며 기존528 reviews/catalog/source pins를 보존했습니다. 소비자-only local replace 예제 빌드와 이 원격 설치는 별개로 확인했습니다. 인증·실제 OpenStack/Python 호출·alpha tag 배포는 수행하지 않았습니다.

2026-10-08에 source revision `40dbb0eaa8f97a19eb0c7aca417bc4251db955e9`을 새 외부 module에 replace 없이 설치하고 위 설치 main과 [Console token 조회 main](../compute/console-auth-token.md)을 빌드했습니다. `GOWORK=off`, get/build exit0·실제 버전 `v0.0.0-20261008013006-40dbb0eaa8f9`이며 module-cache Go source1,968개 SHA256 `1b22fdecda0348eb93f1fc99c17b4a6caf4e79b1808eff1406d66dbadc901e31`가 집중12그룹·전체42 package gate와 같습니다. catalog/source pins/기존527 reviews를 보존한 새1행5계약으로 현재 API 완료 수는221/3,362입니다. SDK와 원격 소비자에 replace가 없고 인증·OpenStack/Python 호출은 실행하지 않았습니다.

2026-10-08에 최종 source revision `466f28d7f3f4a6f7f44d75cf9a4e4c1a96bea2f2`를 새 외부 module에 replace 없이 설치하고 위 설치 main과 [Console 자동 선택 main](../compute/console-selection.md)을 빌드했습니다. `GOWORK=off`, get/build exit0·실제 버전 `v0.0.0-20261008010620-466f28d7f3f4`이며 module-cache Go source1,965개 SHA256 `7543a1f9eaea5ab349f95f4bc6a9d3027137111e26ffc70acadc00d78f2bbc99`가 최종 전체42 package gate와 같습니다. 새 composition1개를 검토한 현재 API 완료 수는220/3,362입니다. SDK와 소비자에 replace가 없고 인증·OpenStack/Python 호출은 실행하지 않았습니다.

2026-10-08에 push한 `0851b89cb637`을 새 외부 module에서 replace 없이 설치하고, 위 설치 main과 [Keypair·legacy console main](../compute/keypairs-console.md)을 함께 빌드했습니다. `GOWORK=off`, `go get`·`go build -mod=readonly` exit0이며 실제 버전은 `v0.0.0-20261008003103-0851b89cb637`입니다. module-cache Go source1,960개 SHA256 `65af6c636b84bb427a832758738be65d823a78ac0e00dc4efaa31632b470d1b2`가 로컬 전체41 package gate의 최종 소스와 같습니다. SDK와 소비자에 replace가 없고 인증·OpenStack 호출은 실행하지 않았습니다. 당시 같은 Go 소스에서 user 작업2개와 native Create1개를 개별 판정하여 API 완료 수는219개였습니다.

앞선 `1d159655cb11`에서도 설치 main과 [Compute 작업 main](../compute/user-actions.md)의 원격 빌드가 PASS했습니다. 당시 버전은 `v0.0.0-20261008001048-1d159655cb11`이며 로컬 전체 gate와 원격 Go SHA가 같았습니다.

앞선 `f32680a5cb51`에서도 설치 main과 [Compute 조회 main](../compute/user-read-apis.md)의 원격 빌드가 PASS했습니다. 당시 버전은 `v0.0.0-20261007234227-f32680a5cb51`이며 로컬 전체 gate와 원격 Go SHA가 같았습니다.

앞선 `36e16d08dc5f`에서도 설치 main과 [Keystone native/owned 두 main](../identity/v3/users/memberships.md)의 원격 빌드가 PASS했습니다. 당시 버전은 `v0.0.0-20261007231141-36e16d08dc5f`이며 로컬 전체 gate와 원격 Go SHA가 같았습니다.

아래는 namespace 도입 당시의 검증 이력입니다. checkout 밖의 독립 소비자 2개로 위 main을 빌드했습니다. local-replace 검증과 원격 설치 검증을 구분하며, 원격 소비자는 `GOWORK=off`이고 replace가 없습니다. 실제 설치 버전은 `v0.0.0-20261007215549-3f0240534253`이며 module cache에서 빌드했습니다. source SHA256은 `137a479b9613464c549cfe734a66b91a32a99bcb201ea886d3a068a183128d9d`입니다. 이 설치 단위 검증 당시 API 완료 수는188개였으며, 해당 직전 판정 구간에서196개까지 증가했습니다. 최신 개수는 [구현 현황](implementation-plan.md)에서 확인합니다.

| 검사 | 상태 |
|---|---|
| 고정 소스·기존 판정 보존 | PASS:namespace 단위의3,362개 fingerprint·513개 판정·당시188개 완료 유지 |
| SDK 재생성 일치 | PASS:생성기 race test·재생성 drift0 |
| 기존 전체 계약 검사·핵심 smoke | PASS:40개 test package·5흐름/9그룹; Go source SHA 보존 |
| 외부 module의 local-replace 빌드 | PASS:위 main 그대로 별도 module에서 빌드; consumer에만 replace |
| push된 정확한 커밋의 replace 없는 설치·빌드 | PASS:`3f0240534253`; `go get`·`go build -mod=readonly` exit0, Replace 없음 |
| Keystone 두 owned 목록 추가 후 외부 설치·빌드 | PASS:`36e16d08dc5f`, no replace; 설치 main+native/owned membership main3개와 원격/로컬 Go SHA 일치 |
| Compute 조회 추가 후 외부 설치·빌드 | PASS:`f32680a5cb51`, no replace; 설치 main+조회 main2개와 원격/로컬 Go SHA 일치 |
| Compute user 작업 추가 후 외부 설치·빌드 | PASS:`1d159655cb11`, no replace; 설치 main+action main2개와 원격/로컬 Go SHA 일치 |
| Keypair 생성·legacy console 추가 후 외부 설치·빌드 | PASS:`0851b89cb637`, no replace; 설치 main+keypair/console main2개와 원격/로컬 Go SHA 일치 |
| Console 자동 선택 추가 후 외부 설치·빌드 | PASS:`466f28d7f3f4`, no replace; 설치 main+자동 선택 main2개와 최종 원격/로컬 Go SHA 일치 |
| Console token 조회 추가 후 외부 설치·빌드 | PASS:`40dbb0eaa8f9`, no replace; 설치 main+조회 main2개와 최종 원격/로컬 Go SHA 일치 |
