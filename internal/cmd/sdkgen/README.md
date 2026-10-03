# SDK 생성기

`make generate`는 Gophercloud v2.15.0의 compiled metadata로 API·Resources binding·서비스
README·inventory를 생성합니다. Subnet·Secret·Container·Order·AddressGroup·QoSPolicy·SubnetPool·Network·Router·SecurityGroup·Trunk의 semantic list filters는 openstacksdk commit
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
[Network manifest](../../../api/openstacksdk/resources/network/v2/network.json)와
[Router manifest](../../../api/openstacksdk/resources/network/v2/router.json)와
[SecurityGroup manifest](../../../api/openstacksdk/resources/network/v2/security_group.json)와
[Trunk manifest](../../../api/openstacksdk/resources/network/v2/trunk.json)의
identity·pin·envelope를 확인합니다. 앞의 네 리소스와 SubnetPool·Network·Router·SecurityGroup은 소스 7개, AddressGroup은 소스 5개, QoSPolicy는 소스 6개의
SHA256을 독립적으로 감사합니다. Trunk는 소스 6개를 확인합니다.
현재 checkout의 SHA256을 비교하고 `python3 -I`로 [AST extractor](python_filter_manifest.py)를 실행합니다.
C3 MRO·상속 Body/URI·query mapping과 pagination defaults·tag expansion·unknown 및 canonical
우선 정책·list controls를 다시 추출해 manifest 전체와 비교합니다. Subnet AST proof 15개와
Secret·Container의 proof 각 23개, Order의 25개, AddressGroup의 16개와 QoSPolicy의 18개,
SubnetPool의 33개와 Network의 43개·Router의 41개·SecurityGroup의 39개·Trunk의 35개, parser minor version도 포함합니다. Key Manager 리소스는
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
Router는 NetworkResource+TagMixinNetwork의 정확한 MRO와 inherited descriptor override를
검증합니다. query18개·accepted24개와 local Body10개를 분류하며 revision_number가 raw
revision을 선택하는 것을 고정합니다. project_id의 response alias는 query 별칭을 추가하지
않고 deprecated tenant_id는 로컬 조건입니다. effective Body20개 전체의 타입·default·alias를
감사하고 bool1개의 None shortcut/truthiness·정수2개·원문 JSON7개 정책을 구분합니다.
SecurityGroup은 NetworkResource+TagMixinNetwork의 정확한 MRO와 effective Body12개를
감사합니다. query17개·accepted21개와 local Body3개를 분류하며 revision_number·stateful·
is_shared·project_id·tenant_id는 모두 query입니다. project의 response alias가 별도 tenant
query를 합치지 않으며 두 timestamp·security_group_rules만 원문 JSON으로 비교합니다.
이 단위의 local selector에는 int/bool 정규화를 추가하지 않습니다.

Trunk는 Resource+TagMixin의 정확한 MRO와 effective Body10개를 감사합니다.
query14개·accepted18개와 JSON local Body2개인 id·tenant_id를 분류합니다. project response
alias는 별도 tenant Body를 query로 바꾸지 않습니다. sub_ports·status·name·admin state는
query-only이며 native timestamp/revision을 Python descriptor로 가정하지 않습니다.

manifest 수정, 현재 소스 변경, 누락 파일, parser 차이, native raw Body gate drift는 생성
전에 오류입니다. source flag를 생략하거나 감사하지 않은 resource로 분류를 확대하면 거부합니다.
[Subnet 검증 테스트](python_filters_test.go), [Secret 검증 테스트](secret_filters_test.go)와
[Container 검증 테스트](container_filters_test.go), [Order 검증 테스트](order_filters_test.go)와
[AddressGroup 검증 테스트](address_group_filters_test.go)와 [QoSPolicy 검증 테스트](qos_policy_filters_test.go)와
[SubnetPool 검증 테스트](subnet_pool_filters_test.go)와 [Network 검증 테스트](network_filters_test.go)와
[Router 검증 테스트](router_filters_test.go)와 [SecurityGroup 검증 테스트](security_group_filters_test.go)와
[Trunk 검증 테스트](trunk_filters_test.go)가
live 소스·상속·분류·drift·출력 범위를 검사합니다. Secret·Container는 native decoder·pager·원문 숫자
보존에 필요한 함수 선언도 검증합니다. Order는 native `Order`·`Meta`의 두 decoder, pager와 원문 숫자
보존을 포함한 함수 선언 14개를 검증합니다. 다른 리소스에 분류를 추가할 때는 그 소스 계약을 먼저 감사하고 증거를
등록해야 합니다. Subnet의 query 24개·accepted 이름 30개·Body 9개와 Secret의 query 12개·
accepted 이름 13개·Body 12개, Container의 query/accepted 이름 2개·Body 10개와
Order의 query/accepted 이름 2개·Body 14개와 AddressGroup의 query/accepted 이름 8개·Body 3개,
QoSPolicy의 query15개·accepted19개·Body2개, SubnetPool의 query16개·accepted20개·Body10개와
Network의 query23개·accepted35개·Body14개와 Router의 query18개·accepted24개·Body10개,
SecurityGroup의 query17개·accepted21개·Body3개와 Trunk의 query14개·accepted18개·Body2개 연결은
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
Router는 native 모델15개·ListOpts17개·GatewayInfo4개·ExternalFixedIP2개·Route2개 필드를
확인합니다. sole pointer decoder·value query builder·정확한 builder interface·RouterPage 자체
method2개·commonResult.Extract 단일 method·GetResult의 embedding을 검증합니다.
native11개·pagination7개·root5개, 총 함수23개와 resourcePath·RFC3339NoZ 상수2개를 고정합니다.
목록의 ExtractRouters→ExtractRoutersInto→Result.ExtractIntoSlicePtr→private extractIntoPtr와
getter의 commonResult.Extract→Result.ExtractInto 경로를 확인합니다. Router getter가 호출하지
않는 ExtractIntoStructPtr를 dependency로 추가하지 않습니다. raw revision과 native
revision_number의 별도 필드·nested unknown/null·전체 페이지 timestamp 디코드·반복/nil query와
wire status를 보존하며 기존8개 manifest/function AST는 유지합니다. Router에는 공개
ResourceAdapter를 추가하지 않으며 캐시된 versioned Resources를 사용합니다.
SecurityGroup은 native SecGroup11개·concrete ListOpts15개·nested SecGroupRule16개 필드,
Group/Rule의 sole pointer timestamp decoder·Page2개·commonResult.Extract1개·GetResult embedding을
확인합니다. ListOpts에는 own method와 builder interface가 없습니다. native leaf9개·nested rule
decoder1개·pagination7개·root4개의 union guard21개와 rootPath·NoZ 상수2개를 고정합니다.
목록과 getter는 직접 Result.ExtractInto를 사용하며 slice/struct pointer helper를 가정하지 않습니다.
BuildQueryString guard는 유지하는 typed native List의 증거이며 두 SDK raw pager는 이를
호출하지 않습니다. 기존 nativefind.IterateSecurityGroups의 AST는 보존하고 별도
IterateSecurityGroupBodies가 같은 source 검사·전체 query·native pager를 BodyRecord로 연결합니다.
rule의 알려진 필드·timestamp 디코드를 cap보다 먼저 적용하고 원문 unknown/null·숫자는 비교에
남깁니다. 기존9개 generated semantic binding/function과 manifest는 유지합니다.
Trunk는 native 모델13개·Subport3개·ListOpts15개 필드와 value query builder를 확인합니다.
모델에 own decoder가 없고 Page는 IsEmpty만 소유하며 LinkedPageBase.NextPageURL을 상속합니다.
native leaf10개·pagination8개·root2개 총 함수20개와 resourcePath 상수1개를 고정합니다.
목록과 getter는 직접 Result.ExtractInto를 사용하며 NoZ decoder·pointer helper를 추가하지 않습니다.
기존 SDK 소유 full query builder와 native Pager에 BodyStreamWithControl을 연결하고, 이전10개
manifest/generated semantic function을 보존합니다. typed List·FindIdentity·subport 변경 API와
ordinary WithName/WithStatus의 기존 조건은 유지합니다.
Container·Order·AddressGroup·QoSPolicy·SubnetPool·Network·Router·SecurityGroup·Trunk 검증에는 실제 compiled native export를 읽는 회귀 테스트도 포함합니다.
Order의 native identity callback은 [별도 타입 검증](order_identity_collections_test.go)으로 자신의
`OrderRef`를 선택합니다. 이름 기반 탐색 지원이나 서버의 변경 API 지원을 추론하지 않습니다.
`GOPHERCLOUD_METADATA=/path/to/packages.json`으로 생성과 같은 metadata를 지정할 수 있으며,
metadata가 없으면 해당 실제 타입 검사만 skip합니다. 실제 생성은 고정 native 타입과 소스
계약을 항상 검증합니다.

Cinder v2/v3 Snapshot의 `UpdateMetadata`에는 [별도 결과 선택 규칙](snapshot_metadata_extractors.go)을
적용합니다. native 결과가 상속하는 Snapshot용 `Extract` 대신 metadata 객체를 반환하며,
native `ExtractMetadata`의 unchecked assertion은 [안전한 helper](../../snapshotmetadata/extract.go)로
대체합니다. 원래 native 오류를 먼저 반환하고 원본 decoded map과 `json.Number`를 보존합니다.
결과에 보존되지 않은 HTTP 성공 코드나 응답 원문은 만들지 않습니다. [생성기 테스트](snapshot_metadata_extractors_test.go)의
6그룹은 정확한 두 package·operation·result만 허용하고 request/result/method graph, Metadata의
전체 `json:"metadata,omitempty"` tag, 패키지별 native 선언 6개, 누락·중복 소스를 검증합니다.
실제 compiled graph와 다른 metadata 연산의 기존 extractor도 확인합니다. [helper 테스트](../../snapshotmetadata/extract_test.go)의
3그룹과 [HTTP 테스트](../../../api/snapshot_metadata_results_test.go)의 6그룹은 malformed envelope,
숫자 정밀도, 원래 오류, 옵션 소유권과 explicit Get을 검증합니다. [연산 목록](../../../api/gophercloud_inventory.json)의
`result_policy: sdk_snapshot_metadata_object`는 이 두 결과 해석만 표시하며 Snapshot CRUD의 반환형이나
Python의 metadata merge·cache 정책을 확장하지 않습니다.

Cinder Volume·Snapshot v2/v3의 수동 `MetadataIn` 범위는 [등록 규칙](cinder_metadata_scopes.go)이
네 기존 collection의 ID·Name·Get·List binding을 확인해 `metadata_scope`에 기록합니다.
별도 collection·native 연산을 생성하지 않고 다른 서비스나 Backup으로 추론하지 않습니다.
[등록 회귀](cinder_metadata_scopes_test.go)는 실제 compiled native 타입으로 네 binding과
기존 Snapshot 결과 해석·Volume image metadata API·버전별 capability 문서를 확인하며,
부모 binding의 변경·누락과 감사하지 않은 리소스를 거부합니다. 요청 실행과 metadata 응답은
[공통 SDK 구현](../../cindermetadata/scope.go), [네 HTTP binding](../../../api/cinder_metadata_scopes_test.go),
[Python/Go 사용법](../../../blockstorage/metadata/README.md)에서 별도로 검증합니다.

`service_waits.go`는 Compute Server·Cinder v2/v3 Volume/Snapshot·Glance v2 Image 여섯 부모 collection에만 `service_wait` 정책을 기록합니다. ID/name/status와 GET/list/delete binding이 달라지면 생성 오류를 반환하며, 다른 모델의 이름으로 capability를 추론하지 않습니다. 추가 메서드는 수동 `wait.go`에 있으며 기존 native 및 공통 waiter의 생성 코드는 유지합니다. 실제 pin과 scope 제외 회귀는 `service_waits_test.go`로 검증합니다.

[Glance Task 등록 규칙](glance_task_wait.go)은 `image/v2/tasks`의 기존 ID-only Task collection에만 `task_wait: WaitForTask`를 기록합니다. 실제 ID/status/Get/List/Create와 canonical type/message/input/result 모델을 확인하며 JSON map 값·부모 binding이 달라지면 생성 오류를 반환합니다. [두 회귀 그룹](glance_task_wait_test.go)은 실제 compiled native 모델과 무관한 서비스 제외, native Get/Create/List 유지 및 서비스 README를 검증합니다. 상태별 재생성·단일 시간 제한·실제 응답은 수동 [Task 구현과 사용법](../../../image/v2/tasks/README.md)에 있으며 별도 native 연산이나 새 Collection을 생성하지 않습니다.

[Glance import 검증](glance_import.go)은 실제 native Create/Get·기본 method 상수와 Image의 canonical ID/container_format/disk_format string 모델을 확인합니다. [두 회귀 그룹](glance_import_test.go)은 compiled 모델의 누락·타입·JSON tag·상수 변경을 거부하고 기존 native API 생성물이 동일하며 import의 미적용 inventory 행을 유지하는지 확인합니다. 수동 SDK 제출의 구체적인 옵션과 실제 202 응답은 [사용 문서](../../../image/v2/imageimport/README.md)에 연결하며 새 native 연산·Collection이나 지원 완료 판정을 합성하지 않습니다.

[Glance staging 검증](glance_stage.go)은 실제 native Stage의 context/client/ID/io.Reader→StageResult signature와 Image의 canonical ID/status string 필드를 확인합니다. [두 회귀 그룹](glance_stage_test.go)은 입력·결과 signature와 모델 누락·타입·JSON tag 변경을 거부하고 native Stage/Upload/Download 생성물과 기존 미적용 inventory 행을 보존합니다. [Staging 사용 문서](../../../image/v2/imagedata/README.md)에 단일 전송·실제 PUT204/GET200 결과·부분 실패를 연결하며 새 native 연산이나 Collection을 합성하지 않습니다.

[Glance 생성·import 검증](glance_create_import.go)은 native Image의 canonical ID/status/format과 CreateOpts의 metadata JSON key·타입을 확인합니다. [두 회귀 그룹](glance_create_import_test.go)은 필드 누락·타입·JSON tag 변경을 거부하고 native 이미지 API를 그대로 보존하며 상위 서비스 문서를 생성합니다. 수동 [CreateAndImport 사용법](../../../image/create-import.md)은 첫 요청 전 concrete 정책 준비, 실제 단계별 증거와 선택적 active 대기를 설명합니다. native 연산·resource inventory나 완전한 Python create_image 지원을 합성하지 않습니다.

[Glance 다운로드 검증](glance_download.go)은 native Download의 context/client/ID→DownloadResult와 Image의 canonical ID/checksum string 모델을 확인합니다. [두 회귀 그룹](glance_download_test.go)은 signature·필드·JSON tag 변경을 거부하고 native raw stream API와 미적용 resource inventory를 보존하며 상위 writer 사용법을 생성합니다. [수동 DownloadTo 사용법](../../../image/download.md)은 metadata-first·checksum 우선순위·full200/no-data204·부분 전송 오류를 설명하며 새 native 연산이나 전체 Python 지원을 합성하지 않습니다.

[Glance 삭제 검증](glance_delete.go)은 native Delete의 context/client/ID→DeleteResult, ErrResult embedding과 Image의 canonical ID를 확인합니다. [두 회귀 그룹](glance_delete_test.go)은 signature·결과·identity drift를 거부하고 native API/resource 생성물이 동일하며 상위 서비스의 [삭제 사용법](../../../image/delete.md)이 연결되는지 확인합니다. 저장소 복사본 삭제는 SDK 상위 메서드이며 새 native 연산이나 resource inventory 행을 합성하지 않습니다.

[Glance 서비스 정보 검증](glance_serviceinfo.go)은 기존 native `ImageImport.Get`의 signature와 canonical ImportInfo/ImportMethods 모델을 보호합니다. [회귀 테스트](glance_serviceinfo_test.go)는 세 SDK 소유 capability가 하나의 `ServiceInfo` 필드로 연결되고 native API는 동일하며 문서에 실제 `ListStores/AllStores/GetImportInfo/GetUsageInfo`를 표시되는지 확인합니다. 저장소 목록과 import singleton은 새 native 연산을 합성하지 않습니다. [ServiceInfo 비교](../../../image/v2/serviceinfo/README.md)에 전용 옵션·엄격한200·raw 응답과 상속된 Python Resource 계약의 차이를 설명합니다.

`UsageInfo`는 [API 전용 사용량 singleton](../../../image/v2/serviceinfo/usage.md)입니다. 생성기는 해당 capability를 기존 `ServiceInfo`에 추가하고 실제 [usage 구현](../../../image/v2/serviceinfo/usage.go)을 문서에 연결합니다. 기존 native 연산·생성 Go·Python catalog는 그대로 보존합니다.

[Glance mutation 생성 테스트](glance_mutations_test.go)는 native Image API·Resources 생성물이 동일하며 상위 단일 태그·활성화 메서드의 strict204·미존재 오류 정책과 [Python/Go 사용법](../../../image/mutations.md)이 문서에 연결되는지 확인합니다. 새로운 native 선언이나 Collection capability를 합성하지 않습니다.
