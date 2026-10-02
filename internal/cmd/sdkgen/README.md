# SDK 생성기

`make generate`는 Gophercloud v2.15.0의 compiled metadata로 API·Resources binding·서비스
README·inventory를 생성합니다. Subnet·Secret·Container·Order·AddressGroup·QoSPolicy·SubnetPool·Network의 semantic list filters는 openstacksdk commit
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
[Container manifest](../../../api/openstacksdk/resources/key_manager/v1/container.json)와
[Order manifest](../../../api/openstacksdk/resources/key_manager/v1/order.json)와
[AddressGroup manifest](../../../api/openstacksdk/resources/network/v2/address_group.json)와
[QoSPolicy manifest](../../../api/openstacksdk/resources/network/v2/qos_policy.json)와
[SubnetPool manifest](../../../api/openstacksdk/resources/network/v2/subnet_pool.json)와
[Network manifest](../../../api/openstacksdk/resources/network/v2/network.json)의
identity·pin·envelope를 확인합니다. 앞의 네 리소스와 SubnetPool·Network는 소스 7개, AddressGroup은 소스 5개, QoSPolicy는 소스 6개의
SHA256을 독립적으로 감사합니다.
현재 checkout의 SHA256을 비교하고 `python3 -I`로 [AST extractor](python_filter_manifest.py)를 실행합니다.
C3 MRO·상속 Body/URI·query mapping과 pagination defaults·tag expansion·unknown 및 canonical
우선 정책·list controls를 다시 추출해 manifest 전체와 비교합니다. Subnet AST proof 15개와
Secret·Container의 proof 각 23개, Order의 25개, AddressGroup의 16개와 QoSPolicy의 18개,
SubnetPool의 33개와 Network의 43개, parser minor version도 포함합니다. Key Manager 리소스는
Resource.id의 literal/alternate 접근과 별도의 HREFToUUID formatter를 구분합니다.
Container는 상속된 `Resource._query_mapping`과 pagination defaults를 함께 검증하며
`name`은 로컬 Body 속성입니다. Order는 별도의 `order_id`·`secret_id` formatter와 `meta`의 dict
descriptor를 검증합니다. literal `id`가 없을 때의 전체 `order_ref`와 suffix formatter를 구분하고,
top-level `name`을 native `Meta.Name`과 구분합니다. AddressGroup은 `project_id`의 descriptor
alias와 별도의 deprecated `tenant_id` 속성을 검증합니다. semantic `project_id`는 query,
`tenant_id`는 로컬 Body이며 descriptor alias로 query 별칭을 추가하지 않습니다.
QoSPolicy는 Resource+TagMixin의 정확한 MRO와 tag query expansion을 검증합니다.
`is_shared`→`shared`·태그 별칭은 query이며 deprecated `tenant_id`·list-typed `rules`는 원문 로컬 필드입니다.
Python 소스 자체는 실행하지 않습니다.
SubnetPool은 Resource와 TagMixinNetwork→TagMixin의 정확한 상속·tag expansion을 검증합니다.
`NetworkResource`를 상속한다고 가정하지 않으며 `project_id`의 descriptor alias로 query 별칭을
추가하지 않습니다. 다섯 int Body descriptor·두 untyped timestamp·list prefixes·literal id를
검증하고 Python 정수 변환과 Go의 정확한 정수 정책을 구분합니다.
Network는 NetworkResource의 inherited revision number와 TagMixinNetwork→TagMixin의 정확한
C3 MRO를 확인합니다. query23개·accepted35개와 non-query Body14개를 분류하며 local bool4개의
default None과 None shortcut·bool 변환을 검증합니다. 응답 bool truthiness·정확한 정수2개·
원문 JSON8개의 정책을 구분하고 caller 값은 변환하지 않습니다. native timestamp·revision
decode와 반환 모델은 유지하며 Python scalar-list wrapping·float underflow 등의 차이를 문서화합니다.

manifest 수정, 현재 소스 변경, 누락 파일, parser 차이, native raw Body gate drift는 생성
전에 오류입니다. source flag를 생략하거나 감사하지 않은 resource로 분류를 확대하면 거부합니다.
[Subnet 검증 테스트](python_filters_test.go), [Secret 검증 테스트](secret_filters_test.go)와
[Container 검증 테스트](container_filters_test.go), [Order 검증 테스트](order_filters_test.go)와
[AddressGroup 검증 테스트](address_group_filters_test.go)와 [QoSPolicy 검증 테스트](qos_policy_filters_test.go)와
[SubnetPool 검증 테스트](subnet_pool_filters_test.go)와 [Network 검증 테스트](network_filters_test.go)가
live 소스·상속·분류·drift·출력 범위를 검사합니다. Secret·Container는 native decoder·pager·원문 숫자
보존에 필요한 함수 선언도 검증합니다. Order는 native `Order`·`Meta`의 두 decoder, pager와 원문 숫자
보존을 포함한 함수 선언 14개를 검증합니다. 다른 리소스에 분류를 추가할 때는 그 소스 계약을 먼저 감사하고 증거를
등록해야 합니다. Subnet의 query 24개·accepted 이름 30개·Body 9개와 Secret의 query 12개·
accepted 이름 13개·Body 12개, Container의 query/accepted 이름 2개·Body 10개와
Order의 query/accepted 이름 2개·Body 14개와 AddressGroup의 query/accepted 이름 8개·Body 3개,
QoSPolicy의 query15개·accepted19개·Body2개, SubnetPool의 query16개·accepted20개·Body10개와
Network의 query23개·accepted35개·Body14개 연결은
전체 Python Resource 또는 Proxy lifecycle 완료 판정과 구분합니다. Container는 중첩
`ConsumerRef`·`SecretRef` 모델도 확인하며 세 Key Manager raw bridge의 반복 query와 기존
Secret 생성 함수의 AST가 유지되는지 검사합니다.
AddressGroup은 native 모델 5개 필드·ListOpts 9개 필드·Link 2개 필드와 pager/extractor를 확인하며,
pagination 및 root `ExtractNextURL`·`Result.ExtractInto`를 포함한 선언 14개를 검증합니다.
원문 `id`·`tenant_id`·`addresses` 필터를 연결하며 기존 identity binding과 native typed query는 유지합니다.
QoSPolicy는 native 모델12개·ListOpts16개 필드·Link2개와 자체 pager·value query builder를 확인하고,
기존 getter를 포함한 native 선언12개·pagination3개·root4개, 총19개를 검증합니다.
`ExtractPolicies`→`ExtractPolicysInto`→`Result.ExtractIntoSlicePtr`→private `extractIntoPtr`의
전체 페이지 decoder와 원문 행의 order/count를 연결하며 `Rules`의 native float64 반환을 유지합니다.
SubnetPool은 native 모델18개·ListOpts22개·Link2개 필드, sole pointer decoder와 value query
builder·정확한 builder interface·page methods·timestamp wrapper를 검증합니다. native12개,
pagination7개, root3개의 함수 선언 22개와 `resourcePath`·`RFC3339NoZ` 상수 2개를 고정합니다.
`ExtractSubnetPools`→`ExtractInto`→`Result.ExtractInto`의 전체 페이지 디코드와 원문 행 순서·개수를
연결하며 필수 prefix length·strict native quota/revision·timestamp decoding 정책을 유지합니다.
동일 raw Neutron bridge의 기존 query snapshot과 getter/identity 기능은 유지합니다.
Network는 native 모델14개·ListOpts17개 필드, sole pointer decoder·value query builder와 정확한
builder interface·NetworkPage 자체 method3개를 확인합니다. native15개·pagination7개·root6개,
총 함수 선언28개와 `RFC3339NoZ` 상수1개를 검증합니다. `ExtractNetworks`→`ExtractNetworksInto`→
`Result.ExtractIntoSlicePtr`→private `extractIntoPtr` 및 getter 경로의 전체 페이지 디코드와
원문 행 순서·개수를 연결합니다. 원문 Body bridge와 ordinary fullquery bridge 모두 반복/nil 값을
복사하고 status를 제거하지 않습니다. Network의 SDK-owned `ResourceAdapter()`는 매 호출마다
독립 metadata를 반환하며 상위 facade가 기존 Kind·Failed·native callback을 유지해 조립합니다.
다른 일곱 resource의 manifest·generated function AST는 이 변경으로 확장하지 않습니다.
Container·Order·AddressGroup·QoSPolicy·SubnetPool·Network 검증에는 실제 compiled native export를 읽는 회귀 테스트도 포함합니다.
Order의 native identity callback은 [별도 타입 검증](order_identity_collections_test.go)으로 자신의
`OrderRef`를 선택합니다. 이름 기반 탐색 지원이나 서버의 변경 API 지원을 추론하지 않습니다.
`GOPHERCLOUD_METADATA=/path/to/packages.json`으로 생성과 같은 metadata를 지정할 수 있으며,
metadata가 없으면 해당 실제 타입 검사만 skip합니다. 실제 생성은 고정 native 타입과 소스
계약을 항상 검증합니다.
