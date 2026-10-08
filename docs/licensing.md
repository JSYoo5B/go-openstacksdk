# 라이선스와 출처

`go-openstacksdk`의 프로젝트 코드는 [Apache License 2.0](../LICENSE)을 적용합니다.
제3자 코드·포팅된 부분·데이터에는 각각의 원래 조건도 적용합니다.
전체 프로젝트의 라이선스를 이유로 기존 저작권이나 제3자 조건을 제거하지 않습니다.
배포할 때 [NOTICE](../NOTICE), [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md),
[라이선스 원문](../licenses/README.md)과 구성요소별 기존 고지를 함께 보존합니다.

## 포함·참조 범위

- **실제 포함한 변경 소스:** `internal/jmespath`는 go-jmespath v0.4.0의 fork입니다.
  James Saryerwinnie의 기존 저작권과 Apache 적용 안내문을 보존하며,
  [해당 README](../internal/jmespath/README.md)에 변경사항을 기록합니다.
  그 디렉터리의 짧은 `LICENSE`와 루트의 Apache 조항 전문을 함께 읽습니다.
- **포팅한 부분:** `internal/cloudfilter/glob.go`의 bracket 변환은
  CPython 3.14.8의 `Lib/fnmatch.py`를 바탕으로 합니다.
  Python 라이선스 원문과 Go matcher로 변경한 내용을 별도로 고지합니다.
- **데이터:** `internal/jsonfilter/descriptor_integer.go`의 숫자 표는
  Unicode 16.0.0 속성을 고정 Python runtime으로 재현할 수 있습니다.
  `internal/cloudfilter/python_lower_data.go`의 full lowercase와 문맥 속성 표도
  공식 Unicode16 원문에서 생성하며 파일에 source SHA·Unicode-3.0 출처를 보존합니다.
  해당 데이터의 Unicode-3.0 조건과 저작권을 보존합니다.
- **Go 의존성:** Gophercloud v2.15.0과 yaml.v2 v2.4.0은 모듈 의존성입니다.
  Gophercloud의 generator 테스트용 source 발췌·변형도 고지 범위에 포함합니다.
  원본 라이선스를 보관하며 YAML의 실제 upstream `NOTICE`도 그대로 포함합니다.
  YAML의 Go 구현에는 Apache-2.0, libyaml에서 포팅한 여덟 파일에는 원래 MIT 조건이
  적용되므로 Kirill Simonov의 저작권과 별도 `LICENSE.libyaml` 원문도 보존합니다.
- **비교·추출 기준:** 고정 OpenStackSDK 소스에서 공개 선언과 리소스 metadata를
  추출합니다. Python과 jmespath.py는 동작 비교용이며 SDK runtime에 포함하지 않습니다.
  고정 버전·출처·참조 결과의 범위는 통합 고지 문서에 구분해 기록합니다.

원본 조건과 파일별 출처의 구체적인 내용은
[THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md)를 기준으로 확인합니다.

## 이름과 공식 관계

이 SDK는 JaeSang Yoo와 기여자들이 개발하는 독립 프로젝트입니다.
OpenStack, OpenInfra Foundation 및 Gophercloud의 공식 프로젝트나 승인·후원 제품임을
주장하지 않습니다. 프로젝트의 코드 라이선스는 상표 사용 권한을 부여하지 않습니다.
이 작업에서 OpenStack 이름의 별도 사용 허가를 받은 것은 아니며,
[OpenInfra 상표 정책](https://www.openinfra.dev/legal/trademark-policy/)은 별도로 적용됩니다.

## 기여

새로 작성한 기여 코드는 프로젝트의 Apache-2.0 조건을 따릅니다.
외부 코드나 데이터를 포함하는 변경은 원래 저작권·라이선스·버전·출처 및 변경사항을
고지 문서에 함께 기록합니다. 기존 upstream 고지를 프로젝트 저작권으로 덮어쓰지 않습니다.
