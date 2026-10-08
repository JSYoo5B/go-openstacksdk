# 외부 Go 프로젝트에서 사용하기

Go 1.25 이상에서 공개 모듈 `github.com/JSYoo5B/gophercloudsdk`를 사용합니다. 전체 SDK는 개발 중이며, 지원 범위는 [구현 현황](implementation-plan.md)에서 확인합니다. `v0.1.0-alpha.1` tag는 아직 배포하지 않았습니다.

새 프로젝트에서 아래처럼 설치합니다. 기존 Go 프로젝트에서는 `go mod init`을 생략합니다. 이 커밋은 원격 설치·빌드를 확인한 revision입니다.

```sh
go mod init example.com/mycloud
GOWORK=off go get github.com/JSYoo5B/gophercloudsdk@f22a803d095233c151a4d47206b49e8a74d8b92d
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

2026-10-08에 source revision `f22a803d095233c151a4d47206b49e8a74d8b92d`을 별도 외부 module에 replace 없이 설치하고 위 설치 main과 [Cloud Flavor main](../compute/flavor-cloud.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008042836-f22a803d0952`이며 module-cache Go source1,998개 SHA256 `c6b80797734918b28faaca111130f30d9dd2a154a8f16f91d0c4ca3eede1dc08`가 집중60그룹(기존56재사용)·전체43 package gate와 같습니다. 새 Source3행9계약 판정으로 완료241/3,362이며 catalog/source pins와 기존544 reviews를 보존했습니다. 같은 최종 Go의 JSON/prose에는 전체 Go gate를 재실행하지 않습니다. 인증/OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

2026-10-08에 source revision `037bd1535f699c20d09aedef6dea11e963f022a4`을 별도 외부 module에 replace 없이 설치하고 위 설치 main과 [Flavor 목록·검색 main](../compute/flavor-records.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008035547-037bd1535f69`이며 module-cache Go source1,996개 SHA256 `cc3404ffe583ef697d0eb7040d71e3d7bf177b738f150ae529bbbce1493d2350`가 집중53그룹(기존41재사용)·전체43 package gate와 같습니다. Source2개 판정으로 현재 완료 수는238/3,362이고 catalog/source pins·다른542 reviews·기존 Find8계약을 보존했습니다. 같은 최종 Go의 JSON/prose에는 전체 Go gate를 재실행하지 않습니다. 인증/OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

2026-10-08에 source revision `3d446f604fb57fefe5be31595679c24ff9ffd796`을 새 외부 module에 replace 없이 설치하고 위 설치 main과 [Flavor property/native main](../compute/flavor-property-and-native.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008031535-3d446f604fb5`이며 module-cache Go source1,983개 SHA256 `bccbeed5b2b97484735d5d6f5d90532313c2f6d1379c3f4c6d844ed1f7f9ce9d`가 집중35그룹(기존28재사용)·전체42 package gate와 같습니다. 새5행15계약으로 현재 완료 수는236/3,362이며 기존538 reviews/catalog/source pins를 보존했습니다. 같은 최종 Go의 JSON/prose 갱신에는 전체 Go gate를 재실행하지 않습니다. 인증·OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

2026-10-08에 source revision `df3afaea603f3eb627f9948be30948a87ec29991`을 새 외부 module에 replace 없이 설치하고 위 설치 main, [Flavor extra-specs main](../compute/flavor-extra-specs.md), [Cloud keypair main](../compute/keypairs-cloud.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008024734-df3afaea603f`이며 module-cache Go source1,979개 SHA256 `47121553d380a5b4b7ed6e34b1dcbfc1fa5a7144a92910e43773c034959d1774`가 집중46그룹·전체42 package gate와 같습니다. 새7행24계약으로 현재 완료 수는231/3,362이며 기존531 reviews/catalog/source pins를 보존했습니다. 같은 최종 Go의 JSON/prose 갱신에는 전체 Go gate를 재실행하지 않습니다. 인증·OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

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
