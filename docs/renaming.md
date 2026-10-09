# go-openstacksdk로 이름 변경

프로젝트 이름을 `gophercloudsdk`에서 `go-openstacksdk`로 변경했습니다. 여러 OpenStack 서비스를 일관된 Go API로 제공하려는 목적을 이름에 드러내기 위한 변경입니다. OpenStack, Python openstacksdk, Gophercloud의 공식 프로젝트가 아니며 승인이나 후원을 주장하지 않습니다.

## 모듈과 패키지

| 항목 | 변경 전 | 변경 후 |
|---|---|---|
| 프로젝트·GitHub 저장소·로컬 디렉토리 이름 | `gophercloudsdk` | `go-openstacksdk` |
| Go 모듈 | `github.com/JSYoo5B/gophercloudsdk` | `github.com/JSYoo5B/go-openstacksdk` |
| 루트 Go 패키지 | `gophercloudsdk` | `openstack` |
| 루트 외부 테스트 패키지 | `gophercloudsdk_test` | `openstack_test` |

기존 사용자는 `go.mod`의 의존성과 모든 SDK import 경로를 새 모듈로 옮겨야 합니다. 서비스별 하위 패키지 경로와 공개 함수 이름은 유지합니다. 아래처럼 import 별칭을 `sdk`로 지정하면 기존 `sdk.Connection`, `sdk.Connect` 등의 표기를 계속 사용할 수 있습니다.

```go
import (
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)
```

별칭을 생략하면 루트 패키지의 식별자는 `openstack`입니다. 설치 명령과 외부 소비자 예제는 [설치 안내](install.md)를 참고하세요. GitHub 저장소는 [JSYoo5B/go-openstacksdk](https://github.com/JSYoo5B/go-openstacksdk)이며 실제 로컬 checkout은 `/Users/jsyoo5b/Workspace/OpenStack/3rdParty/go-openstacksdk`입니다. Codex에 등록된 프로젝트 이름과 경로도 `go-openstacksdk`와 새 로컬 checkout으로 일치함을 확인했습니다.

## 현재 확인한 상태 (2026-10-09)

모듈·package·import·생성기·서비스 문서·설치 예제·Swift 생성 prefix에서
운영용 옛 이름 참조는 **0개**입니다. Git origin과 GitHub 저장소는
`JSYoo5B/go-openstacksdk`이며, GitHub 라이선스 표시는 Apache-2.0입니다.
옛 이름이 남은 여섯 파일은 이름 변경 안내와 변경 전 검증·Source 출처 기록입니다.

라이선스 원문 **12개**의 SHA-256과 JMESPath 원본 고지를 확인했습니다.
현재 Go 의존성 **네 모듈**은 `licenses/dependencies.json`에 고정되어 있습니다.
Darwin·Linux의 amd64/arm64 및 Windows amd64에서 실제 SDK·테스트 패키지의
의존성을 조회해 모두 이 고지 범위에 들어오는 것을 확인했습니다.
`x/sys`는 CPU 지원 코드가 필요한 플랫폼에서 사용되는 간접 의존성입니다.
Gophercloud 원본 발췌를 담은 테스트 파일의 출처·변경 고지를 보존하고,
이를 조합하는 테스트에 대한 설명도 실제 fixture 정의 범위에 맞게 정정했습니다.

전체 `make check`의 **45개 테스트 package**·race·vet·고정 API 판정·진행표·
포맷·라이선스 검사가 통과했습니다. 라이선스 원문 변경, 미등록 의존성,
버전 변경, fork 고지 변경, NOTICE 누락을 넣은 임시 사본 다섯 개는
`make license-check`의 검사기로 모두 거부했습니다.

고지 보완 revision `831bfb1542e1263d0c1604e4ffc269946d85bde5`을 새 외부 Go 프로젝트에서
`GOWORK=off`·replace 없이 설치하고 문서의 main 두 개를 빌드했습니다.
실제 버전은 `v0.0.0-20261009051450-831bfb1542e1`이며 get/build exit0입니다.
Go 소스 **2,184개** SHA-256 `d8e5d6529981f7e9874087b56fc60c400bce477c3e617fcb2cdb3bcf1def044a`,
라이선스·고지·대응표 **17개 파일**, 라이선스 안내와 API 판정·catalog가
로컬 검증본과 byte-identical입니다. [기본 설치 명령](install.md)도 이 revision을 사용합니다.

이 절은 현재 검증 결과입니다. 아래의 API 집계·의존성 수·고지 파일 수·
revision은 각 변경 당시 기록으로 보존합니다. 이름·라이선스 정리는 API 지원
판정을 올리지 않으며 현재 완료 집계는 **292 / 3,362**입니다.

근거는 `/private/tmp/go-openstacksdk-rename-license-final-consumer.json`,
`go-openstacksdk-rename-license-current-dependencies.json`,
`go-openstacksdk-rename-license-current-check.log`,
`go-openstacksdk-rename-license-negative-probes.json`에 기록했습니다.

## 이름 변경 당시 구현과 검증 기록

이름 변경은 지원 API를 추가하거나 지원 판정을 올리지 않습니다. 변경 기준의 완료 집계는 Go 매핑 **255 / 3,362**, 검토 기록 **550개**, 계약 **3,418개**이며, 이후의 현재 수치는 [구현 계획](implementation-plan.md)과 [지원 판정대장](sdk-support-ledger.md)에서 확인합니다.

Go 지원 버전과 의존성 고정도 유지합니다. Go 모듈은 `go 1.25.0`, Gophercloud는 `v2.15.0`, `gopkg.in/yaml.v2`는 `v2.4.0`입니다. 비교 기준 Python openstacksdk revision은 `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. 원본 선언 ID·source pin·3,362개 source fingerprint와 기존 판정 상태·계약을 보존하며, 판정의 Go API 참조와 inventory의 SDK 패키지·후보 경로만 새 모듈로 변경합니다.

생성기의 import·루트 package 선언·문서 템플릿, 생성된 Go 코드와 서비스 문서, Python manifest 생성 스크립트 및 JSON manifest도 새 모듈 경로를 사용합니다. upstream Gophercloud 및 Python openstacksdk 이름과 원본 출처는 유지합니다.

Go 코드의 import·package 선언이 바뀌므로 전체 Go source SHA는 변경됩니다. 과거 커밋의 검증 결과와 SHA는 당시 소스의 증거로 보존하며, 변경 후 트리의 검증과 구분합니다. 이름 변경 이전 커밋의 `go.mod`는 이전 모듈 경로를 선언합니다. 따라서 새 모듈 경로에 이름 변경 전 SHA를 붙인 `go get github.com/JSYoo5B/go-openstacksdk@<변경 전 SHA>`를 설치 명령으로 사용하면 안 됩니다. 새 모듈 경로를 선언하는 변경 후 revision을 사용해야 합니다.

## Swift 업로드 이름

SDK가 새 segmented upload용으로 만드는 prefix는 `.gophercloudsdk-upload-<generation>/`에서 `.go-openstacksdk-upload-<generation>/`로 변경합니다. 새 임시 파일의 prefix도 `gophercloudsdk-swift-`에서 `go-openstacksdk-swift-`로 바꿉니다. 생성한 segment prefix는 기존과 같이 결과와 새 manifest에 반영됩니다.

기존 object 경로와 저장된 manifest를 일괄 변경하거나 이전 prefix의 object를 이동하지 않습니다. 사용자가 명시한 object 경로도 그대로 사용합니다.

## 라이선스와 출처

프로젝트의 배포 조건은 [LICENSE](../LICENSE), 포함하거나 의존하는 제3자 자료의 출처와 고지는 [THIRD_PARTY_NOTICES](../THIRD_PARTY_NOTICES.md), 적용 범위와 이름 사용 설명은 [라이선스 안내](licensing.md)에 기록합니다. 기존 저작권과 upstream 라이선스 고지는 이름 변경 후에도 보존합니다.

이름 변경과 라이선스 문서 추가는 상표 사용 허가나 상표 중복 검토 완료를 의미하지 않습니다. Python openstacksdk와의 모든 동작 동등성 또는 전체 API 구현 완료를 주장하지 않으며, 실제 지원 범위와 남은 작업은 연산별 판정을 따릅니다.

## 이름 변경 직후 재검증 기록

2026-10-09에 실제 디렉토리, Git remote, GitHub 저장소 이름·설명, GoLand 설정, Go import·패키지 선언, 생성기·manifest·문서·예제를 재점검했습니다. 사용 중인 경로는 모두 새 이름입니다. 옛 이름은 변경 전후 비교, 과거 커밋의 설치 검증과 테스트 reference 출처에만 보존합니다.

라이선스 재검토에서 yaml.v2에 포함된 libyaml 포팅 코드 여덟 파일의 MIT 고지를 보완했습니다. upstream `LICENSE.libyaml` 원문과 Kirill Simonov의 저작권을 보존하며, 프로젝트의 Apache-2.0 조건과 적용 범위를 구분합니다. 라이선스 표의 SHA-256 10개를 확인했고, 실제 SDK와 테스트의 외부 Go 의존성은 고정된 Gophercloud·yaml.v2 두 모듈입니다.

전체 `make check`는 43개 테스트 package·vet·API 판정·포맷 검사 모두 통과했습니다. 푸시한 `478f9b4feff6ea5b71a036a5db0c83aae45e8cbc`을 별도 Go 프로젝트에서 `GOWORK=off`·replace 없이 설치하고 문서의 설치·목록 main 2개를 빌드했습니다. 원격 Go source 2,044개가 검증한 로컬 소스와 같으며, 라이선스·고지 14개 파일도 byte-identical입니다. 원격 설치 버전은 `v0.0.0-20261008183000-478f9b4feff6`입니다.

추가 재점검에서 기본 설치 명령이 libyaml MIT 고지 보완 전 revision을 가리키는 것을 확인해 `95a7c6de03b24b76ffbe49fa6f17efdad553abc6`으로 갱신했습니다. 새 외부 프로젝트에서 [설치 안내](install.md)의 `go mod init`, `GOWORK=off go get`, `GOWORK=off go build -mod=readonly ./...` 명령과 정확한 main을 그대로 사용해 세 명령 모두 exit0을 확인했습니다. 소비자와 SDK에 replace가 없으며 라이선스·고지 14개 파일이 현재 트리와 byte-identical입니다. 원격 Go source 2,062개 SHA256 `5edf83e69d402ecd8b22677afb70d3d7ca5f8af692534fdd8b6273ea46fb022c`도 기존 전체 43개 테스트 package 검사에 통과한 소스와 같습니다. 근거는 `/private/tmp/go-openstacksdk-naming-install-final-consumer.json`입니다. 인증·OpenStack 호출은 실행하지 않았습니다.

## 최종 고지와 배포 검증 (2026-10-09)

Gophercloud generator-test에 포함한 고정 source 발췌·변형과 Unicode full lowercase/context 데이터의 고지 범위를 `fea9a547`에서 보완했습니다. 모듈·import·생성기·Git remote의 실제 참조는 `go-openstacksdk`이며, 이전 이름은 변경 이력과 reference 출처에서만 보존합니다. 원본 라이선스 표10개의 hash와 internal JMESPath LICENSE 사본도 확인했습니다.

정확한 문서·판정 revision `72cf240b6a9508cbbbf4404ae882ef9576a306ce`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008214624-72cf240b6a95`·get/build exit0이며 태그/설치 main **2개**를 빌드했습니다. 원격 Go source2,087개 SHA256 `b449631fcefd58d9a3f240ddce214040a3a60bf190d5d20c9a8c497c1166eba4`가 최종 집중/전체 gate·반복 생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-tags-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다.

## 파일별 변경 고지와 최종 배포 확인 (2026-10-09)

tracked 파일을 전수 검색해 실제 module·import·package·생성기·manifest·Git remote에서 옛 이름 참조가 없음을 확인했습니다. GitHub 저장소도 `JSYoo5B/go-openstacksdk`이며 Apache-2.0으로 표시됩니다. 옛 이름은 변경 이력과 과거 reference 출처에서만 보존합니다.

Gophercloud source fixture를 포함하거나 결합·변형하는 테스트 파일 **18개**에 원저작권·Apache-2.0·원본 범위·로컬 변경 고지를 추가했습니다. fixture 문자열과 실행 코드는 변경하지 않았습니다. 보존한 라이선스 표의 SHA-256 **10개**와 내부 JMESPath 원문 사본도 일치합니다. 최종 `make check`의 **44개 테스트 package**·race·vet·API 판정·포맷 검사가 통과했습니다.

보완 커밋 `1ec3b909bd3588df74f2e84ed38baa96d99cc577`을 원격 main에 반영하고 별도 외부 Go module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008222501-1ec3b909bd35`에서 문서 예제 main **2개**가 빌드되며, 배포된 Go source2,089개 SHA256 `fdd94cb1618b66c86fd59ee79bde0c27120f68d61c0a0634f47f643431db5659`가 최종 검증 소스와 같습니다. 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. 당시 [기본 설치 명령](install.md)은 이 revision을 사용했습니다. 실제 인증·OpenStack 호출 및 별도 상표 사용 허가는 이 검증에 포함하지 않습니다.

당시 [기본 설치 명령](install.md)은 owned 이미지 수정까지 포함한 검증 revision `c068cacecf76dc4ab40254b4324233aa4cfcc86e`을 사용했습니다. 원격 get/build·문서 main2개·최종 Go source2,095개 일치와 기존 라이선스·고지14개 파일의 배포를 다시 확인했습니다. 이름·라이선스 적용 범위는 위와 같으며 새 자동 JSON Patch engine은 독립 작성한 프로젝트 코드입니다.


## MemberRecord 완료 당시 배포 확인

정확한 문서·판정 revision `4aa85ff43194756fcbe31c6beffe5d1c6707cc2a`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008232408-4aa85ff43194`·get/build exit0이며 멤버 CRUD/설치 main **2개**를 빌드했습니다. 원격 Go source2,101개 SHA256 `0f37f5bd403d6f253c7088d9b339d239d9962045303a06acafff41be973c55d3`가 최종 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-member-record-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 모듈·root package·서비스·문서 예제의 현재 이름은 `go-openstacksdk`/`openstack`이며 라이선스·고지 원문은 그대로 보존합니다. 현재 기본 설치 revision은 [설치 안내](install.md)를 따릅니다.


## 이미지 속성 helper 완료 후 배포 확인

정확한 문서·판정 revision `48d6ef15374db885ca1d02bf2e4bb7b052e8135e`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261009010239-48d6ef15374d`·get/build exit0이며 속성 helper/설치 main **2개**를 빌드했습니다. 원격 Go source2,123개 SHA256 `dc7351eaadb7b5a1402fabf4dc3bf15636747813a20cc716bbf7514344f91d50`가 최종 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-properties-remote-consumer.json`에 근거가 있습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 이름 변경·라이선스 재감사에서 현재 module·import·package·생성기·manifest·Swift 생성 prefix의 옛 이름 참조가 없고, 보존한 라이선스 원문10개와 원본 저작권·변경 고지도 일치함을 확인했습니다. 이름 변경과 라이선스 적용 범위는 [라이선스 안내](licensing.md)를 따릅니다.


## 이미지 staging 완료 후 배포 확인

정확한 문서·판정 revision `046b489d4cf328e8be77e67b36dbe501b6727c61`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261009013644-046b489d4cf3`·get/build exit0이며 staging/설치 main **2개**를 빌드했습니다. 원격 Go source2,132개 SHA256 `845e89a3170ae14989cb3279403749d618be928802179d7ccc2429c491bc1bdb`가 최종 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-stage-remote-consumer.json`에 근거가 있습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 공개 module·root package·import 경로와 라이선스 배포 범위는 [이름 변경 안내](#모듈과-패키지)와 [라이선스 안내](licensing.md)를 따릅니다.


## 이미지 import 완료 후 배포 확인

정확한 문서·판정 revision `8c616a274fb357161d717193902018194886e367`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261009021503-8c616a274fb3`·get/build exit0이며 import/설치 main **2개**를 빌드했습니다. 원격 Go source2,141개 SHA256 `4c9a95cd0621ed629ff6e4daacc0f82189594d8fbaaf391edd3b1ee4ec099e0f`가 최종 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-import-remote-consumer.json`에 근거가 있습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 공개 module·root package·import 경로와 라이선스 배포 범위는 [이름 변경 안내](#모듈과-패키지)와 [라이선스 안내](licensing.md)를 따릅니다.


## 이미지 upload 완료 후 배포 확인

정확한 문서·판정 revision `a20f06ca2f5b6572de79e9c251f643fc1e1a980d`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261009024601-a20f06ca2f5b`·get/build exit0이며 upload/설치 main **2개**를 빌드했습니다. 원격 Go source2,148개 SHA256 `c36b09fce4c592dbc54b9ef68d2c62bb1182a9790e2236611816e5617c3c84f2`가 최종 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-upload-remote-consumer.json`에 근거가 있습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 공개 module·root package·import 경로와 라이선스 배포 범위는 [이름 변경 안내](#모듈과-패키지)와 [라이선스 안내](licensing.md)를 따릅니다.


## 이미지 download 완료 후 배포 확인

정확한 문서·판정 revision `a4688f1b4cf4ae8aa26896629a0321965054840d`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261009033426-a4688f1b4cf4`·get/build exit0이며 download/설치 main **2개**를 빌드했습니다. 원격 Go source2,157개 SHA256 `cdc70487ebaa12a8b466146917ddfae15da7c64abbff133f8f882cbcf89ce83f`가 최종 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **16개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-download-remote-consumer.json`에 근거가 있습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 공개 module·root package·import 경로와 현재 라이선스 배포16파일 범위는 [이름 변경 안내](#모듈과-패키지)와 [라이선스 안내](licensing.md)를 따릅니다. 앞선14파일 기록은 당시 revision의 검증 이력입니다.

## 라이선스 자동 점검

2026-10-09에 `make license-check`를 전체 `make check`에 추가했습니다.
보관 원문 12개의 해시, 현재 Go 의존성 네 개의 고지 대응, JMESPath fork 고지와
배포 고지를 검사합니다. 임시 사본의 원문 변경·미등록 의존성·버전 변경·fork 고지 변경·
NOTICE 누락을 모두 거부하는 것을 확인했습니다. 전체 vet·race 45개 테스트 package,
pinned parity·진행표·gofmt 검사도 통과했습니다. 운영 경로·import·설치 예제는
`go-openstacksdk`이며 이전 이름은 변경 이력과 참조 출처에만 보존합니다.
GitHub 저장소 `JSYoo5B/go-openstacksdk`와 Apache-2.0 표시도 확인했습니다.

당시 원격 설치 확인: revision `a6b944ae6fdc6f98a90942345ca5e2cd519ba2b3`, 실제 버전 `v0.0.0-20261009041345-a6b944ae6fdc`을 외부 프로젝트에서 replace 없이 설치하고 문서 main 두 개를 빌드했습니다. get/build exit0이며 Go 소스 2,173개와 라이선스·고지·대응표 17개 파일이 로컬과 동일합니다. 기본 설치 명령도 이 검증한 revision으로 갱신했습니다.
