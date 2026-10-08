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

별칭을 생략하면 루트 패키지의 식별자는 `openstack`입니다. 설치 명령과 외부 소비자 예제는 [설치 안내](install.md)를 참고하세요. GitHub 저장소와 로컬 checkout의 최종 이름도 `go-openstacksdk`로 맞춥니다.

## 구현과 검증 기록

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
