# SDK 생성기

`make generate`는 Gophercloud v2.15.0의 compiled metadata로 API·Resources binding·서비스
README·inventory를 생성합니다. Subnet의 semantic list filters는 openstacksdk commit
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

검증은 checked-in [Subnet manifest](../../../api/openstacksdk/resources/network/v2/subnet.json)의
identity·pin·envelope와 독립적으로 감사한 소스 7개 SHA256을 확인합니다. 이어 현재 checkout의
7개 SHA256을 비교하고 `python3 -I`로 [AST extractor](python_filter_manifest.py)를 실행합니다.
C3 MRO·상속 Body/URI·query mapping과 pagination defaults·tag expansion·unknown 및 canonical
우선 정책·list controls를 다시 추출해 manifest 전체와 비교합니다. AST proof 15개와 parser
minor version도 포함합니다. Python 소스 자체는 실행하지 않습니다.

manifest 수정, 현재 소스 변경, 누락 파일, parser 차이, native raw Body gate drift는 생성
전에 오류입니다. source flag를 생략하거나 다른 resource로 분류를 확대해도 거부합니다.
[검증 테스트](python_filters_test.go)의 5개 그룹이 live 소스·상속·분류·drift·출력 범위를
검사합니다. 다른 리소스에 분류를 추가할 때는 그 소스 계약을 먼저 감사하고 증거를
등록해야 합니다. Subnet의 query 24개·accepted 이름 30개·Body 9개 연결은 전체 Python
Resource 또는 Proxy lifecycle 완료 판정과 구분합니다.
