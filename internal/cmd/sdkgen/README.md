# SDK 생성기

`make generate`는 Gophercloud v2.15.0의 compiled metadata로 API·Resources binding·서비스
README·inventory를 생성합니다. Subnet·Secret·Container의 semantic list filters는 openstacksdk commit
`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 선언과 상속을 함께 검증합니다.

Go 의존성 외에 Git과 Python 3.14의 표준 라이브러리가 필요합니다. 실행 시 OpenStackSDK를
import하거나 Python 패키지를 설치하지 않습니다. 기본 스크립트는 고정 commit을 임시
checkout으로 가져오며, 이미 준비한 checkout은 다음처럼 지정합니다.

```sh
SDKGEN_OPENSTACKSDK_SOURCE=/path/to/openstacksdk make generate
```

생성기를 직접 실행하면 같은 source 경로를 필수로 넘깁니다. metadata는
`generate.sh`처럼 고정 Go dependency에서 `go list -deps -export -json ./openstack/...`로
얻은 파일입니다.

```sh
go run ./internal/cmd/sdkgen -metadata /path/to/packages.json -output . \
    -openstacksdk-source /path/to/openstacksdk
```

검증은 checked-in [Subnet manifest](../../../api/openstacksdk/resources/network/v2/subnet.json)와
[Secret manifest](../../../api/openstacksdk/resources/key_manager/v1/secret.json)와
[Container manifest](../../../api/openstacksdk/resources/key_manager/v1/container.json)의
identity·pin·envelope와 각 리소스에서 독립적으로 감사한 소스 7개 SHA256을 확인합니다.
현재 checkout의 SHA256을 비교하고 `python3 -I`로 [AST extractor](python_filter_manifest.py)를 실행합니다.
C3 MRO·상속 Body/URI·query mapping과 pagination defaults·tag expansion·unknown 및 canonical
우선 정책·list controls를 다시 추출해 manifest 전체와 비교합니다. Subnet AST proof 15개와
Secret·Container의 proof 각 23개, parser minor version도 포함합니다. 두 Key Manager 리소스는
Resource.id의 literal/alternate 접근과 별도의 HREFToUUID formatter를 구분합니다.
Container는 상속된 `Resource._query_mapping`과 pagination defaults를 함께 검증하며
`name`은 로컬 Body 속성입니다. Python 소스 자체는 실행하지 않습니다.

manifest 수정, 현재 소스 변경, 누락 파일, parser 차이, native raw Body gate drift는 생성
전에 오류입니다. source flag를 생략하거나 감사하지 않은 resource로 분류를 확대하면 거부합니다.
[Subnet 검증 테스트](python_filters_test.go), [Secret 검증 테스트](secret_filters_test.go)와
[Container 검증 테스트](container_filters_test.go)가
live 소스·상속·분류·drift·출력 범위를 검사합니다. Secret·Container는 native decoder·pager·원문 숫자
보존에 필요한 함수 선언도 검증합니다. 다른 리소스에 분류를 추가할 때는 그 소스 계약을 먼저 감사하고 증거를
등록해야 합니다. Subnet의 query 24개·accepted 이름 30개·Body 9개와 Secret의 query 12개·
accepted 이름 13개·Body 12개, Container의 query/accepted 이름 2개·Body 10개 연결은
전체 Python Resource 또는 Proxy lifecycle 완료 판정과 구분합니다. Container는 중첩
`ConsumerRef`·`SecretRef` 모델도 확인하며 두 Key Manager raw bridge의 반복 query와 기존
Secret 생성 함수의 AST가 유지되는지 검사합니다.
Container 검증에는 실제 compiled native export를 읽는 회귀 테스트도 포함합니다.
`GOPHERCLOUD_METADATA=/path/to/packages.json`으로 생성과 같은 metadata를 지정할 수 있으며,
metadata가 없으면 해당 실제 타입 검사만 skip합니다. 실제 생성은 고정 native 타입과 소스
계약을 항상 검증합니다.
