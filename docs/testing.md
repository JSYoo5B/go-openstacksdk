# 테스트

테스트는 로컬 HTTP 모의 서버로 SDK에서 Gophercloud를 거쳐 HTTP 요청·응답을 처리하는 경로를 검증합니다. 실클라우드 인증이나 리소스 생성은 수행하지 않습니다.

## 실행

프로젝트 루트에서:

```sh
make check
make smoke
go test -race ./...
go test -coverpkg=./... ./...
go test -run TestCreate ./...
```

Go 1.25 이상, 의존성 다운로드, localhost 포트 바인딩 허용이 필요합니다. `make check`는 vet, 60초 package timeout을 적용한 race test, 지원 판정 근거와 gofmt 상태를 확인합니다. `go test`는 실행 예제를 빌드하지만 예제의 main을 실행하지 않습니다.

`make smoke`는 [개발 preview의 5개 핵심 흐름](release-milestones.md)을 기존 테스트에서 선택해 실행하고 `.reports/core-smoke.json`에 실제 결과를 저장합니다. 새로운 mock이나 같은 계약의 테스트를 따로 복제하지 않습니다.

공개 module namespace 변경은 기존 전체 계약 테스트·smoke·생성기 검증을 재사용합니다. [외부 소비자 예제](install.md)의 정확한 main은 checkout 밖의 별도 module에서 빌드하며, consumer-only local replace와 push된 정확한 커밋의 원격 설치를 구분합니다. HTTP harness를 추가하지 않습니다.

## 공개 테스트 도구 재사용

Glance schema record는 기존 `prepareSchema`의 fixed route/options/source와 common guarded REST 응답을 재사용합니다. 새 class-specific 테스트는 ordinary/metadata의 null·dict/bool/list·ordered alias, bare defaults와 parsed nonobject의 차이, actual Wire/receipt와 Connection location/zone의 연결만 보완합니다. `image/schema_records_test.go`·`connection_image_schema_records_test.go`를 기존 schemas core/HTTP/options/Connection과 공통 REST/source 표에 연결하며 공개 Gophercloud helper·기존 testcloud와 body fault wrapper를 사용합니다. strict16getter의 별도 typed/200 정책을 유지하고 같은 오류 matrix나 HTTP harness를 새로 복제하지 않습니다. 실제 통과 수·정확한 main 빌드·지원 판정은 [판정대장](sdk-support-ledger.md)에 검증 후 기록합니다. [Python/Go 비교](../image/schema-records.md)는 독립 main 하나를 제공합니다.

Nova availability zone 일반 목록과 Cloud 이름은 기존 guarded REST pager·RawResource·cloudread 옵션/source·Python truthiness와 Connection location snapshot을 재사용합니다. 서비스 고유 표는 일반 경로·raw nullable 값·available/unavailable 평가 순서·source HTTP 실패의 빈 목록 처리와 partial Inventory 연결을 확인하며, native SinglePage DTO와 관리자 상세 목록을 구분합니다. 공개 Gophercloud method/header helper·기존 `internal/testcloud.New`와 response fault wrapper를 사용하고 별도 HTTP 서버 구성이나 같은 공통 paging/fault matrix를 복제하지 않습니다. 동일 최종 소스의 집중 race41그룹(새3·기존38재사용), 고유 API48사례와 전체 `make check`43개 실제 test package·vet·parity·progress·gofmt가 PASS했습니다. native List의 정상 envelope 결함은 공통 SinglePageStream으로 수정하고 기존 생성기 그룹에 AZ 사례를 추가했습니다. 의도한 generated native List 변경 이후 반복 생성 drift0·정확한 독립 main 외부 빌드도 PASS했습니다. 새 서버 harness·fault helper는0개이며 소스 SHA·원격 설치 근거는 [판정대장](sdk-support-ledger.md)에 기록합니다. [사용 비교](../compute/availability-zones.md)는 독립 main을 포함합니다.

Cloud Flavor의 All/Search/Get은 기존 owned Flavor ListRecords/FindFlavor·extra specs reader·Service location/guard와 Keypair eager Inventory/Value·cloudfilter.Select 패턴을 재사용합니다. 고유 binding 표에서는 Listfalse/Search·Gettrue 기본값, 전체 inventory/enrichment 이후 검색, Get falsey/object kwargs와 unsupported truthy string, partial Inventory/child receipt만 추가합니다. 기존 public testcloud·flavor fixture·method/header·response fault wrapper를 사용하며 같은 pagination/version/source/fault matrix를 복제하지 않습니다. 동일 최종 소스의 집중 race60그룹(새4·기존56재사용), 고유 API51사례와 전체 `make check`43개 실제 test package·vet·parity·progress·gofmt가 PASS했습니다. 새 HTTP 서버 harness·fault helper는0개이며 공통 옵션 소유권 검증1그룹은 순수 테스트입니다. 생성물의 Go drift0과 정확한 가이드 main의 외부 소비자 빌드도 확인했습니다. 소스 SHA와 원격 설치 결과는 [판정대장](sdk-support-ledger.md)에 기록합니다.

Flavor owned List/Find는 기존 REST ListWithControl·공통 FilterSelection/Body matcher·Collection.FindIdentity·microversions.MemberGet·FetchExtraSpecs reader를 조합합니다. 고유 HTTP binding 표는 `api/compute_flavor_records_test.go`·`api/compute_flavor_find_test.go`에서 공개 Gophercloud method/header helper와 기존 `internal/testcloud`·flavor fixture·response fault wrapper로 검증하고, 공통 float 변환은 jsonfilter 표에서 다룹니다. 새 서버 harness나 서비스별 pagination/fault 엔진을 만들지 않습니다. 기존 Fetch/nativefind/identity/discovery/REST 회귀를 재사용하고 서비스 wrapper에서는 Connection location·binding·partial evidence 연결만 확인합니다. 동일 소스의 집중 race53그룹(신규12·재사용41), 고유 API84사례가 PASS했습니다. HTTP 서버 harness·fault helper를 새로 추가하지 않았습니다. 전체 `make check`의43개 실제 test package·vet·parity·progress·gofmt도 PASS했습니다. 소스 SHA와 외부 예제 빌드 결과는 [판정대장](sdk-support-ledger.md)에 남깁니다.

Flavor 단일 property와 Cloud ID 조회는 기존 Fetch의 입력·옵션·버전·응답 reader, strict identity/enrichment를 공유합니다. 고유 API binding은 공개 testhelper·기존 fixture로 검증하고 기존 Fetch/nativefind/identity/guard/version28개 그룹을 그대로 선택했습니다. 최종 집중 race35그룹에서 새7그룹74사례가 함께 PASS했으며, 공통 옵션/physical fault matrix를 다시 작성하지 않았습니다. [Python/Go 가이드](../compute/flavor-property-and-native.md)와 [판정대장](sdk-support-ledger.md)에 native DTO/string-map/linked page와 owned raw property의 서로 다른 계약을 기록합니다.

공통 `internal/testcloud.New`는 Gophercloud v2.15.0의 공개 `testhelper.SetupHTTP()`와 `FakeServer.Teardown()`을 사용합니다. SDK 어댑터는 공유 Provider, 토큰 잠금과 서비스별 endpoint만 구성합니다. 각 fixture는 격리된 mux/server를 사용하며 cleanup을 등록합니다. 공통 REST 목록 테스트의 기존 `listSpec`도 같은 공개 `SetupHTTP`를 사용합니다. 서비스별로 서버 구성 코드를 복제하지 않습니다.

새 테스트는 공개 `testhelper.TestMethod`, `TestHeader`, `TestHeaderUnset`, `TestBody`, `TestJSONRequest`와 `testhelper/fixture.SetupHandler`를 먼저 검토합니다. 단순 요청 검증과 고정 응답에 맞으면 재사용하고, SDK의 단계 순서·부분 결과·정확한 raw JSON·응답 소유권 검증만 추가합니다. Cinder 생성 계약의 공통 wire 검증은 method·source·token에 공개 helper를 사용합니다. `SetupHandler`는 upstream 고정 token을 검사하므로 SDK의 token 교체·공유 검증에는 별도 handler가 필요합니다.

JSON 비교 helper는 float64 기반이므로 큰 정수나 원문 바이트의 정밀도 검증에는 사용하지 않습니다. 고정 v2.15.0의 deep-equality helper는 slice 길이가 같은지도 보장하지 않으므로 정확한 순서·횟수·추가 요청 부재 검증에는 `reflect.DeepEqual` 등의 기존 검사를 유지합니다. 이미 검증한 fixture·공통 엔진·표 기반 테스트를 확장하며 동등한 setup/assertion을 새로 복제하지 않습니다.

공통 알고리즘의 계약은 공통 패키지 테스트에서 검증하고, 서비스 테스트에는 경로·envelope·서비스 고유 분기와 binding 연결만 추가합니다. 기존 테스트 표에 사례를 넣을 수 있으면 별도 테스트 서버나 harness를 만들지 않습니다. 새로운 transport가 필요할 때도 기존 fault/body wrapper를 먼저 재사용합니다. 검증한 동일 소스의 전체 테스트는 문서·지원 판정 갱신 때문에 반복하지 않습니다.

Cloud keypair 조합의 [HTTP 테스트](../api/compute_keypair_cloud_test.go)는 공개 Gophercloud `TestMethod`·`TestHeader`, `internal/testcloud.New`, 기존 `flavorIdentityClient`·`payloadContractTrack`·`secretFetchRoundTripFunc`를 재사용합니다. 새 목록 엔진이나 fault wrapper를 만들지 않고 기존 `ListRecords`·`FindKeypair`·생성/삭제 binding에 eager 수집과 필터 presence·public-key 생략·bool 결과의 assertion을 추가합니다. dictionary의 첫 semantic 필터와 두 번째 Cloud 필터, ordered/glob/JMESPath 선택은 공통 `internal/cloudfilter.Select`·`First`로 조합합니다. `resource.CloudLocation.ForResource`의 소유 snapshot은 Cloud view에만 보충하고, 실제 Wire/Envelope를 바꾸지 않는 연결을 검증합니다.

명시 [Flavor extra-specs HTTP 테스트](../api/compute_flavor_extra_specs_fetch_test.go)는 같은 public fixture와 response fault helper를 사용합니다. 기존 `TestNativeFlavorExtraSpecs...` 계열의 조건부 보충·native map 계약을 회귀 근거로 재사용하면서, 새 `FetchExtraSpecs`의 항상 GET·raw 값·supplied 모델 복사만 별도 binding으로 검증합니다. 공통 `microversions.MemberGet`과 REST의 guarded response·decode·receipt 처리는 그대로 사용하며 서비스마다 Read/Close/source 오류 matrix를 복제하지 않습니다. 이 단위의 실행 결과와 지원 판정은 [판정대장](sdk-support-ledger.md)에 별도로 기록합니다.

Keypair 목록·찾기는 기존 `internal/testcloud.New`·공개 Gophercloud method/header helper·`flavorIdentityClient`·`payloadContractTrack`·`secretFetchRoundTripFunc`를 사용합니다. 고유6그룹97사례는 row projection·owner·version·native ABI의 binding을 검증하고, 공통 reader의 기존 표에10사례를 넣었습니다. 직접 GET/fallback·필터·continuation·passive ID의 기존91그룹을 함께 선택해 집중 race98그룹·전체42 package gate가 PASS했습니다. 같은 Go SHA의 문서·판정 변경에는 전체 검사를 반복하지 않고 parity·progress·gofmt를 확인합니다. 목록용 서버나 fault wrapper를 새로 만들지 않고, 생성의9필드 projector도 공유합니다. [목록·검색 사용법](../compute/keypairs-list-find.md)에 기본값과 source/Go 차이를 기록합니다.

Console auth-token 조회는 같은 공통 discovery와 Cinder에서 추출한 `microversions.MemberGet`의 bodyless·version header·retry 보호를 사용합니다. 새 고유3그룹42사례만 public fixture 표로 검증하고 기존 console composition·pure version·Cinder member GET·RawResource clone 검증9그룹을 함께 선택했습니다. 집중 race12그룹이 PASS했으며 이 중9그룹은 기존 검증의 재사용입니다. nullable passive 필드·token/auth 구분·location-before-discovery·actual Wire만 해당 리소스의 assertion으로 추가합니다. [사용법](../compute/console-auth-token.md)에 반환과 오류 정책을 설명합니다.

Console 자동 선택은 Cinder의 version 비교·유한 guarded discovery를 `internal/microversions`로 추출해 공유합니다. composition3그룹과 공통 pure1그룹만 새로 추가하고, 기존 direct type gate1·Cinder import5·version snapshot2·Connection discovery4그룹을 함께 선택합니다. 집중16그룹 중12그룹은 기존 검증의 재사용입니다. 서비스별 status/self-link·광고/선택 분기·protocol/location·winning executor의 오류만 기존 public fixture 표에 추가합니다. source나 옵션이 바뀌면 해당 집중 검사와 전체 회귀 검사를 실행하고, 같은 Go source SHA의 문서·판정 변경은 parity/progress/gofmt만 다시 확인합니다. [사용법](../compute/console-selection.md), [검증 기록](sdk-support-ledger.md#compute-console-자동-선택-완료)에 결과를 연결합니다.

Compute 조회5개도 이 방식을 적용했습니다. metadata·keypair의2테이블24사례는 기존 `testcloud.New`와 공개 method/header helper를 사용하고, 콘솔3그룹은 기존 `flavorIdentityClient`·`payloadContractTrack`·`secretFetchRoundTripFunc`를 재사용합니다. flavor의 GET-only/extra-specs는 기존 HTTP 그룹을 지원 근거로 연결했습니다. 집중15그룹과 최종41 test package 전체 gate가 PASS했으며, 문서/JSON 갱신 뒤에는 같은 Go 전체 검사를 반복하지 않습니다. [비교·실행 예제](../compute/user-read-apis.md), [소스 SHA와 결과](sdk-support-ledger.md#compute-user-조회5개-완료)를 확인할 수 있습니다.

Remote console·keypair 삭제는 같은 fixture의6그룹으로 서비스별 기본값·경로·응답과 native ABI를 확인하고 기존 공통 삭제 classifier2그룹을 재사용했습니다. 헤더 생략은 공개 `TestHeaderUnset`으로 검사합니다. native Get은 앞선13사례를 그대로 연결하고, Create/Delete의 owner·extension binding은 기존 호환성 표 안에서 보강했습니다. 집중8그룹·전체41 package gate·외부 main build가 PASS했습니다. [사용법](../compute/user-actions.md), [단위별 근거](sdk-support-ledger.md#compute-console-생성과-keypair-삭제-및-native-3개-완료)에 API 수와 실제 HTTP 작업 수를 구분했습니다.

Keypair 생성·legacy console 조회는 기존 public HTTP fixture의6그룹과 공통 BodyFieldBoolean/RawResource clone2그룹을 재사용합니다. Keypair 고유43사례와 legacy의6개 action·raw scalar/array/null·strict JSON 오류를 검증하고, native Create의 실제 concrete 필드/extension·partial/null·200/201 계약은 같은 표에 넣었습니다. 집중8그룹·전체41 package gate·정확한 독립/원격 main 빌드가 PASS했습니다. 새 harness나 공통 오류 표 복제 없이 검증 근거를 [가이드](../compute/keypairs-console.md)와 [판정대장](sdk-support-ledger.md#keypair-생성과-legacy-console-url-완료)에 연결합니다.

2026-10-08 정적 감사에서 `api/` 패키지의 직접 `httptest.NewServer` 호출은0개였습니다. 다른 패키지에는24개 호출이 남아 있으며 대부분의 서버 setup은 공개 fixture로 재사용할 수 있는 기존 코드입니다. 우선 Swift metadata의 반복 bootstrap과 단순 RoundTrip forwarding alias를 작은 정리 후보로 기록했습니다. TCP 주소 도달성·응답 Read/Close 실패 같은 별도 계약과 서버 setup의 중복을 구별합니다. 이번 새 조회의 실제12그룹 결과와 기존 setup 현황을 [판정대장](sdk-support-ledger.md#compute-console-auth-token-조회-완료)에 연결합니다.

Read/Close 실패, 전송 중 취소, retry·reauth hook, 동적으로 바뀌는 token·source처럼 공개 helper가 표현하지 못하는 경우에는 전용 transport/handler를 유지합니다. upstream의 `internal` helper는 Go 접근 제한을 따르며 복사해서 우회하지 않습니다. 패키지 내부 테스트로만 공개된 fixture도 외부 import 대상이 아닙니다.

## 검증 범위

| 파일 | 검증하는 계약 |
|---|---|
| `connection_test.go` | 서비스 lazy 구성과 concurrent cache, region/interface, microversion, 토큰 공유, 명시 인증·환경변수·clouds.yaml 선택과 region override |
| `microversion_test.go` | 인증된 discovery, project와 reverse-proxy 경로, 범위 교집합, 명시 버전 우선, 헤더·캐시·재시도·취소 |
| `connection_cancellation_test.go` | 캐시된 상위 서비스 접근에서도 취소를 반환 |
| `compute/boot_volume_test.go`, `connection_boot_volume_test.go` | 기존 볼륨 이름/ID 해석, 삭제 기본값, boot device payload, 상충 입력 사전 검증, 대기 실패 시 생성 서버 보존 |
| `compute/new_boot_volume_test.go` | 이미지 기반 새 볼륨 크기·타입·삭제 옵션, microversion 사전 검사, 이름 해석과 실패 시 무삭제 |
| `collections_test.go` | 다섯 collection의 공통 조회, 모든 페이지의 정확한 이름 매칭, 중복 이름, iterator 중단, HTTP 오류 보존, 삭제 기본값, timeout/cancel/실패 상태 |
| `server_create_test.go` | 이미지·flavor·네트워크 이름 해석, POST payload, false 입력, 옵션 snapshot, 확장 필드 충돌, ID 조회 생략, 대기 실패 후 생성 리소스 보존 |
| `compute/compute_test.go` | 상태가 없는 flavor에 대한 오류와 생성 입력/의존성 오류 |
| `network/network_test.go` | 확장 query 인코딩, 정확한 이름으로 삭제 대상 선택 |
| `image/image_test.go` | Glance의 flat response와 Properties, killed 상태 |
| `image/delete_contracts_test.go`, `image/delete_core_test.go`, `image/delete_options_test.go` | 전체/저장소 고정 DELETE, 정확한 Name·실제404 기본값, 사전 검증·snapshot·현재 token, 실제204 원문/헤더와 Read/Close/context 오류, nested404 오류 보존·redirect·prebody retry·native ABI |
| `image/v2/serviceinfo/contracts_test.go`, `connection_image_serviceinfo_test.go` | 저장소 기본/상세·import singleton, canonical optional 값·raw 숫자·원문/헤더/status, preflight·snapshot·lazy paging·cap·source/live auth, 기존 native Get 호환성 |
| `internal/cmd/sdkgen/glance_serviceinfo_test.go`, `internal/rest/response_validation_test.go` | 두 SDK 소유 모델의 단일 서비스 registry·실제 capability·native drift 거부, 전체 응답 검사 hook와 accepted 증거·lazy break |
| `internal/cmd/sdkgen/glance_delete_test.go` | native Delete signature/ErrResult/ID drift 거부, 기존 API/resource 생성물 보존과 수동 저장소 삭제 문서 연결 |
| `blockstorage/blockstorage_test.go` | Cinder microversion 전송과 error 상태 패턴 |
| `network/floating_ip_test.go`, `connection_floating_ip_test.go` | 외부 네트워크·Compute 참조, 포트·IPv4 선택, 모호성, 생성 후 연결·대기 실패 보존 |
| `image/upload_test.go`, `image/upload_retry_test.go` | metadata/PUT/대기, 옵션 snapshot, Reader 소유권, 부분 소비·비동기 Close에서 무재전송, decode 부분 성공 보존 |
| `api/introspection_contracts_test.go` | UUID 조회·목록, Finished/Error 기반 완료 대기, typed 실패·취소·timeout, 고정 조회 대상 |
| `api/introspection_start_contracts_test.go`, `baremetalintrospection/v1/introspection/start_test.go` | ManageBoot nil/false/true와 확장 query 인코딩, ResourceBase, serializer/옵션 오류 preHTTP, POST202, 오류 status·본문·header·URL과 취소 원인 보존 |
| `internal/cmd/sdkgen/audited_requests_test.go` | StartIntrospection만 helper 호출, pinned 함수 본문·signature·builder·입력·결과 drift 거부, 주석·공백 변경 허용 |
| `resource/wait_identity_test.go` | 응답의 ID가 바뀌거나 빠져도 명시 ID의 polling 대상 유지 |
| `resource/wait_policy_test.go`, `resource/wait_attributes_test.go` | 실패 상태 교체·빈 목록·목표 우선, 무제한/유한 대기와 부모 context, 모델 JSON tag·pointer·null·progress 순서, terminal callback 생략·callback 취소·deleted/nil 결과 |
| `api/wait_workflow_preflight_test.go`, `api/introspection_wait_options_test.go` | 잘못된 대기 속성을 생성·업로드 전에 차단, Inspector의 Finished 고정 조건과 서비스 오류 보존 |
| `resource/collection_test.go` | 잘못된 참조와 iterator 옵션을 HTTP 요청 전에 거부 |
| `resource/filters_test.go`, `api/network_subnet_semantic_filters_test.go` | Subnet query24/accepted30·Body9 자동 분류, bulk canonical nil/false/empty·최종 선택값 검증·snapshot/동시 재사용, unknown/reserved·namespace 충돌/clear·server-only name, raw/native isolation·cap/terminal 페이지 오류 |
| `internal/cmd/sdkgen/python_filters_test.go` | 고정 Python SHA7·isolated AST/C3/inherited 선언 재추출·15개 AST proof·전체 manifest drift 거부, native raw Body gate·Subnet-only 생성 및 inventory |
| `api/identity_user_project_records_test.go`, `identity/v3/users/project_records_descriptor_test.go`, `internal/cmd/sdkgen/python_user_project_filters_test.go` | Keystone user_projects는 기존 testcloud/public testhelper·body fault wrapper와 공통 classifier/matcher/REST를 재사용합니다. query15·local4·Resource/Wire·페이지/한도·대표 오류는 HTTP4그룹, 고정 Python MRO/URI/source proof는3그룹으로 확인하고 native membership·foreign extractor 회귀를 유지합니다. |
| `api/identity_user_group_records_test.go` | Keystone user_groups는 Project의 공통 membership reader와 기존 testcloud/public helper·body fault/transport wrapper를 재사용합니다. Group 고유 raw/URI·기본 페이지·대표 오류는 HTTP3그룹으로 확인하고 native/Project 회귀는 기존 테스트를 실행합니다. 첫 server limit 허용·고정은 공통 REST의10사례와 두 서비스의 기존 continuation 표에 연결합니다. |
| `api/keymanager_create_records_test.go` | Barbican 생성3개의 기존 leaf/cached fixture·public testhelper·body fault wrapper 재사용. flat nullable/alias 입력·seed와 실제 응답 분리·옵션 소유권·view-only descriptor·대표 공통 accepted/source/context/native retry 오류·native ABI/status를6그룹으로 검증 |
| `api/keymanager_metadata_delete_test.go`, `resource/delete_errors_test.go`, `internal/rest/collection_source_guard_test.go` | Barbican3개 default binding은 기존 leaf/Connection·public testhelper·body wrapper로 경로·native/owned 차이를 검증하고, missing 분류·terminal 오류·source guard는 공통 테스트를 재사용 |
| `api/keymanager_secretstores_test.go`, `api/keymanager_quotas_test.go` | Barbican 목록·두 고정 selector·quota 네 연산, SecretStore9 query/5 local 분류는 공통 semantic 필터 재사용, full-ref/raw-id marker·canonical JSON·snapshot·page guard, optional 정수·교체 PUT204·raw GET 정밀도·실제 404와 전송/read 원인 구별 |
| `connection_keymanager_sdk_owned_test.go`, `internal/cmd/sdkgen/keymanager_services_test.go` | SDK 소유 API·native API가 같은 cached ServiceClient/provider 사용, 최신 token·ResourceBase·고정 프로젝트·추가 GET/fallback 없음·cached 취소, 실제 capability만 Service/inventory/docs에 연결 |
| `resource/pagination_test.go`, `api/pagination_contracts_test.go` | linked URL·query 순서·Swift marker 순환 중단, 오류 한 번 전달, break 시 후속 링크 검사 생략 |
| `internal/cmd/paritycheck/reviews_test.go` | catalog 추가 시 기존 판정 보존, 원본 입력 drift 거부, API·테스트·문서 근거, 중복 ID/JSON key와 불완전 지원 판정 거부 |
| `api/server_tags_contracts_test.go` | 서버 1회 해석, 2.26 요구, tag escaping, nil/빈 교체·명시 false·404 정책과 원래 HTTP 오류 |
| `api/instance_actions_scope_test.go` | requestID 조회, 목록·상세·event 확장 및 원본 JSON/헤더, pagination·break·cycle·취소 |
| `api/trove_databases_contracts_test.go`, `api/trove_users_contracts_test.go` | instance 1회 해석, 목록 기반 exact 조회, 단일/batch 배열 Create, 이름 인코딩·미존재 삭제·대기 |
| `api/accelerator_read_test.go`, `api/accelerator_actions_test.go`, `api/accelerator_boundaries_test.go` | SDK 소유 Cyborg 연결·microversion·UUID, 응답 JSON/헤더, 목록 중단·cycle·origin, enable/disable·program payload, 빈 페이지 continuation, redirect 경계·최신 token·재인증/취소와 사전 검증 |
| `api/accelerator_profiles_test.go`, `api/accelerator_attributes_test.go`, `api/accelerator_singleton_test.go`, `api/accelerator_status_filter_test.go` | 프로필 exact 이름·UUID 삭제·array-one 생성, attribute key/ID 구별·flat 생성, singleton 응답 cardinality, snapshot·로컬 상태 필터와 명시 query 보존 |
| `api/accelerator_requests_test.go`, `api/accelerator_request_binding_test.go`, `api/accelerator_request_response_boundary_test.go` | ARQ 전체 batch·부분 해석·raw bytes·오류 보존, state 대기, collection PATCH202·service token·2.1 project gate, single/batch/instance DELETE204 selector, accepted201 body 취소·decode 오류에서 무재시도·body close |
| `api/project_quotas_contracts_test.go`, `api/nova_project_quotas_precision_test.go`, `connection_quotas_test.go` | 고정 project quota, 별도 Keystone 이름 조회·auth scope, zero/-1/force false, singleton 결과·오류·Reset |
| `api/cinder_project_quotas_contracts_test.go`, `api/cinder_project_quotas_responses_test.go` | Cinder defaults·usage query·DELETE200, 별도 Keystone·recorded auth·고정 target, typed/Extra deep snapshot·core 충돌·큰 정수·raw 응답·오류 |
| `api/neutron_project_quotas_contracts_test.go`, `api/neutron_project_quotas_projects_test.go`, `connection_project_quotas_test.go` | Neutron details.json·check_limit false·DELETE202/204, 서비스별 Connection 연결·이름/인증/취소, 응답 metadata·snapshot·큰 정수·malformed·권한 오류 |
| `api/nova_quota_defaults_users_test.go` | 별도 project defaults, 고정 project+user·Keystone user 해석, 사용자 query·retry/reauth snapshot·redirect 범위 보호, 취소·malformed·HTTP 원인 |
| `api/neutron_quota_defaults_test.go`, `api/neutron_quotas_list_test.go` | 별도 defaults, 단일 override 목록·검증된 project/tenant identity, lazy break·로컬 filter/한도·옵션 소유권·continuation 거부·raw metadata·취소 |
| `api/octavia_project_quotas_contracts_test.go`, `api/octavia_project_quotas_projects_test.go`, `api/octavia_project_quotas_list_test.go` | 공식 lbaas 경로·전역 defaults 구분·null 상속·native alias, lazy pagination·필터 보존·True/False·cycle/origin·decode 원문, Connection 프로젝트 해석 |
| `api/manila_project_quotas_test.go`, `api/manila_scoped_quotas_test.go` | SDK 소유 quota transport, 2.7 경로·2.25 Detail·2.39 share type, 고정 project/user/type·Keystone 해석, exact int64·snapshot·DELETE202·selector redirect/reauth 보호 |
| `api/manila_quota_class_test.go` | 고정 class 이름·legacy/current 경로·GET/PUT200, 12개 limit exact int64·snapshot, force/selector 거부·지원 method set·원본 HTTP/JSON/context·redirect 보호 |
| `api/manila_quota_versions_test.go` | 숫자 route 선택·latest 사전거부, legacy header/type 검증·generic-only manual client 거부, header 대소문자·충돌·매 scope 요청 재검사 |
| `api/designate_project_quotas_contracts_test.go`, `api/designate_project_quotas_projects_test.go`, `api/designate_project_quotas_transport_test.go` | root quota 객체·PATCH200·DELETE204, sudo-project/all-projects header, accepted 응답 decode 증거, 고정 프로젝트·snapshot·HTTP/취소·redirect/retry/reauth 보호 |
| `api/nova_project_limits_contracts_test.go`, `api/nova_project_limits_projects_test.go`, `api/nova_project_limits_transport_test.go` | 고정 tenant_id·reserved 0/1, native absolute·legacy rate·raw 응답·accepted 오류, Keystone/auth, redirect/retry/reauth/context |
| `api/cinder_limits_test.go`, `connection_limits_test.go` | 숫자3.39 필터·latest/하위버전 거부·매 요청 버전 재검사, header/type 사전검사, native Get 유지, 고정 project_id·Keystone/auth, optional int64·raw 응답·accepted 오류·Connection |
| `api/magnum_quota_create_test.go`, `api/magnum_quota_operations_test.go`, `connection_project_quotas_test.go` | 고정 project+resource, explicit hard limit·exact int·snapshot/reauth, POST201·GET200·PATCH202·DELETE204, default fallback·raw ID/metadata·strict404·accepted decode/HTTP/context/redirect |
| `api/magnum_quota_list_test.go` | concrete 목록 기본값·snapshot, all_tenants·확장 query 보존, exact row ID, 서버 limit 축소·marker·cycle/origin/query 보호, lazy break·페이지 decode/HTTP/context 증거 |
| `api/heat_stacks_contracts_test.go` | name+ID identity, output query, summary/detail 차이, linked/marker pagination, 고정 대상 waiter·실패/삭제 완료 |
| `api/heat_stackresources_contracts_test.go`, `api/heat_stackresources_responses_test.go` | resource_name/논리·물리 ID 구별, nested owner·health false·metadata·대기, raw 필드·큰 숫자·독립 header·malformed 목록 거부 |
| `api/heat_stackevents_scope_test.go`, `api/heat_stackevents_pagination_test.go` | stack/resource 이벤트 경로·typed query·marker·cycle·break·취소, resource-scoped 단건 GET·raw 필드·독립 header |
| `api/share_access_rules_scope_test.go`, `api/share_access_rules_errors_test.go` | Manila2.45/2.82 access rule 조회·action·잠금, share 일치·부모404/403 보존, 취소·waiter |
| `internal/rest/response_policy_test.go`, `internal/rest/response_contracts_test.go` | nil context 사전 검사, 실제200/201/204의 Read·Close·취소 원인과 독립 응답 증거, 재전송 없음, retry JSON snapshot·소유권 보호와 원래 status gate·native 오류 경계 ([공통 응답 계약](../internal/rest/README.md)) |
| `resource/metadata_test.go`, `internal/rest/response_test.go`, `internal/rest/list_test.go`, `internal/rest/collection_validation_test.go` | 추가 JSON·null·생략·정확한 숫자, 응답 header 소유권·accepted 오류 증거, 고정 URL·retry/reauth·origin, lazy break·marker/link 순환·query 보존·명시 paging 정책, 서비스 row invariant 실패의 단건·전체 page 증거 |
| `connection_sdk_owned_test.go`, `connection_sdk_resources_test.go`, `internal/cmd/sdkgen/sdk_owned_services_test.go` | Senlin·Masakari catalog root/version/project/proxy, raw escaped tenant·Masakari discovery, 공유 인증·캐시·취소·microversion, typed 서비스 연결·singleton/list-only 생성 정책 |
| `api/instanceha_segments_test.go`, `api/instanceha_hosts_test.go` | segment UUID·DB ID 구별, enabled1.2·false·snapshot, 고정 segment host·부모 이름 1회 해석, POST202/201·PUT200·DELETE204, exact 이름·페이지·HTTP 오류·취소 |
| `api/instanceha_notifications_test.go`, `api/instanceha_vmoves_test.go` | notification UUID·payload/timestamp, raw workflow 숫자, 고정 notification scope·VM move1.3, 상태 polling·실패·취소, 부모와 server identity 구별·accepted 응답 증거 |
| `api/instanceha_wait_contracts_test.go` | 네 Masakari waiter facade의 UUID 경로·고정 부모·상태 없는 모델, 실제 outgoing deadline과 기본값·override, 버전 재검사·삭제 GET·실패 목록·취소·오류 원문 |
| `api/clustering_discovery_test.go`, `api/clustering_read_test.go` | build-info singleton·type 이름/schema·profile ops1.4, action epoch/target·event level string/숫자/null/생략·큰 정수·반복 filter snapshot, service1.7/list-only, 페이지·strict envelope·HTTP 오류 |
| `internal/rest/list_short_page_test.go`, `api/clustering_read_test.go` | 명시 limit에서 짧은 페이지 뒤에도 wire ID marker 유지, consumer 모델 변경·break·no-limit·빈 페이지·순환 정책 |
| `internal/senlin/mutation_test.go`, `internal/senlin/filter_test.go`, `internal/senlin/query_test.go` | wrapped/flat body·optional JSON 생략/null/빈 객체, snapshot·원문 숫자·SDK 소유 field/header 보호, 로컬 재귀 subset·배열·타입·정확한 decimal/거대 지수 비교, sort grammar와 별도 key allowlist |
| `api/clustering_profiles_test.go`, `api/clustering_policies_test.go`, `connection_sdk_resources_test.go` | profile/policy CRUD·객체 PATCH·Validate1.2, spec/metadata snapshot·null/빈 객체·숫자, 이름/ID·미존재·중복·409, 로컬 Body 필터·페이지·accepted 오류·인증/버전 공유 |
| `api/clustering_mutation_recheck_test.go`, `api/clustering_profiles_update_snapshot_test.go`, `api/clustering_policies_response_test.go` | custom option·이름 lookup 이후 source/version 재검사, 준비한 body/header 소유권, Get/Validate accepted malformed 응답 증거·재전송 없음·미선언 success code 거부 |
| `resource/delete_identity_test.go` | 이름 조회 응답의 빈/공백/다른 종류 ID를 binding 정책으로 검사해 collection 경로 삭제를 막고, 유효한 ID는 한 번 해석 후 삭제 |
| `request/optional_test.go`, `api/clustering_scalar_options_test.go` | typed 값의 생략/null/false/zero·빈 문자열 구분, 실패한 decode에서 이전 값 보존, With 함수 재사용과 caller config 변경 후 독립 요청 |
| `internal/senlin/async_test.go`, `api/clustering_async_location_case_test.go` | 선택한 action collection·origin·reverse prefix, required/empty/중복·대소문자 Location, accepted 원문/헤더/status 보존, 조회·재전송 없음 |
| `api/clustering_clusters_test.go`, `api/clustering_clusters_response_test.go`, `api/clustering_clusters_pagination_test.go` | cluster POST201·PATCH/DELETE202, raw 숫자·생략/null·profile_only1.6, 이름 해석 전 snapshot·source 재검사, 일반/force404 차이, malformed 응답·짧은 페이지·로컬 필터·취소 |
| `api/clustering_nodes_test.go`, `api/clustering_nodes_pagination_test.go` | node POST/PATCH/DELETE202·필수 action 참조, physical ID/index/details·tainted1.13, nullable 입력·snapshot·strict force404, query/로컬 필터·raw marker·다중 페이지 Find 중복/후속 오류 |
| `connection_senlin_async_test.go`, `internal/cmd/sdkgen/sdk_owned_services_test.go` | Clusters/Nodes/Actions의 CRUD·ScaleOut/Check/Cancel client·최신 token·1.13·reverse prefix 공유, 자동 조회 없이 submission 후 명시 action 조회, 비동기 결과를 버리는 Collection.Delete 미지원 정책과 재생성, Adopt/Preview/Recover/PerformOperation의 선택1.7·token·prefix 공유·incidental Location 미해석 |
| `resource/list_options_snapshot_test.go` | native·SDK 소유 lazy iterator의 caller 옵션 slice 교체/nil·반복 iteration·잘못된 옵션 소유권과 HTTP 사전 차단 |
| `resource/list_control_test.go`, `internal/rest/list_control_test.go` | raw cap을 name/status 필터 전에 적용, 0/음수·옵션 우선·재순회·first-page·미소비 continuation/행 생략, opt-in limit hint와 빈 페이지 종료, native whole-page 오류·소비 중 취소와 break |
| `api/clustering_typed_list_controls_test.go`, `api/clustering_catalog_list_controls_test.go` | 11개 Senlin typed 목록의 raw cap·Body 필터 순서·snapshot·explicit limit·단일 페이지·기본값·empty-next 종료·응답 원문·source/version 재검사, binding의 로컬 cap과 초기 limit/marker 미지원 |
| `api/clustering_resource_list_controls_test.go` | 11개 Resources binding의 List/All·로컬 name/status 필터 앞의 cap·미소비 행/link 생략·empty-next·hint/explicit size·기본 foreign guard 원문, canonical parent 1회 조회와 Services1.7 사전검사 |
| `internal/senlin/command_test.go` | 빈 command 객체·typed key 보호·plugin 입력 snapshot, 응답 action과 필수 Location ID 일치·accepted 오류 증거·경로와 header 소유권 |
| `api/clustering_action_update_test.go` | PATCH202·CANCELLED·1.12, force query 생략/false/true, snapshot·이름 lookup 후 source/version 검사, empty/opaque 원문·native HTTP/context 오류 |
| `api/clustering_clusters_commands_test.go` | ScaleIn/Out count:null·정확한 Resize 숫자·optional 값, 노드 배열/map·gate1.3/1.4, parent 1회 해석·snapshot, strict202 action/Location·오류 원문·취소·재전송 없음 |
| `api/clustering_nodes_commands_test.go` | Check/Recover 기본 객체, operation 생략/empty/null·params/null·명시 check1.6·ops1.4, plugin key 범위, 입력/header snapshot·lookup 후 gate, missing/ambiguous·accepted 오류·HTTP 원인 |
| `api/clustering_nodes_adopt_test.go` | flat Adopt/Preview POST200·1.7, physical body identity·nullable 선택 입력과 snapshot, preview 네 필드 기본 null·숫자/string/null version·정밀한 inner/outer 원문, incidental Location 미해석, custom option 이후 source/version/context 재검사·malformed200/native/read 오류 |
| `api/clustering_cluster_health_test.go` | Check/Recover 기본 객체·check-only1.6 성공·capacity1.7·ops1.4, operation/params optional 및 server Filters 구분, lookup 전 body/header/minimum 고정·재검사, typed 충돌·202/Location 증거·native/context 오류 |
| `api/clustering_receivers_test.go` | 동기 POST201·GET/PATCH/List200·DELETE204, webhook/message/vendor type·nullable cluster/action·raw actor/params/channel, user wire1.4·Resources/후속 page gate, lookup 전 snapshot·version 재검사, short page·로컬 filter의 wire marker, 다중 페이지 Find·strict/ignore404·accepted 증거·native/context 오류 |
| `api/clustering_cluster_policy_commands_test.go` | policy_attach/detach/update의 required body identity·GET 생략·버전 gate 없음, enabled 생략/false/true/null·독립 snapshot, parent lookup 전 body/header 고정·source 재검사, typed/header/query 보호, strict202 action/Location·원문/native 오류 |
| `connection_senlin_receivers_policy_test.go` | Receiver와 정책 명령의 공유 client·최신 token·선택1.4·reverse prefix, user/global_project=false, 동기 응답의 incidental Location 미해석·202 이후 추가 조회 없음 |
| `api/clustering_wait_contracts_test.go` | 9개 facade·기본status/unlimited/ERROR와delete120초·실제요청deadline/override/common5분, attr preflight·target우선·4status/5nostatus 종결, callback/cancel·Name1회·고정ID·token/source/header 재검사, missing/duplicate/later403·native/transport/malformed200 증거 |
| `connection_senlin_wait_test.go` | 공유 provider의 polling 중 token교체·선택1.13·reverse prefix, 응답 ID 변경에도 고정 route, receiver404 완료·channel/action/DELETE 미호출 |
| `internal/senlin/tracked_test.go`, `api/clustering_lifecycle_test.go` | Profile/Policy owned cache·exact JSON equality·sticky dirty·same-value/no-op·삭제 null·부분 응답 병합·별도 response·fixed ID·실패의 dirty 보존·Refresh reset·요청 중 새 편집 보존 |
| `api/clustering_cluster_policies_test.go`, `connection_senlin_scopes_test.go` | binding UUID와 policy route 구분·canonical parent·exact raw ID/name·초기 query와 명시 continuation 정책·snapshot/lazy break·statusless scope wait·HTTP 응답 증거·공유 token/source/version |
| `api/clustering_cluster_attributes_test.go`, `connection_senlin_scopes_test.go` | canonical cluster·escaped JSONPath·1.2 재검사·실제200/공식202 동기 배열·raw 값/null/생략/정밀도·explicit links·list-only capability·원문 증거·공유 인증 |
| `api/clustering_cluster_create_status_test.go` | 공식201/실제202 생성·202 필수 action Location·201 선택 참조·잘못된 accepted 증거·native success-code 오류·재전송 없음 |
| `api/clustering_cluster_metadata_test.go`, `connection_senlin_metadata_test.go`, `internal/senlin/json_equal_test.go` | 실제 GET/PATCH·raw presence/독립 snapshot·exact-decimal no-op·nil/empty/batch삭제·고정 route/최신 token·202 action 증거·원문 오류·option/source/context 재검사·RMW 동시 변경/접수와 완료 구분 |

페이지 테스트는 서로 다른 페이지의 같은 이름을 검사합니다. `break` 테스트는 다음 페이지 요청 횟수가 0인지 확인합니다. 시간 관련 테스트는 짧은 SDK timeout을 사용하고 `errors.Is(context.DeadlineExceeded)`를 검사합니다. 특정 실행 시간과 동일하다고 가정하지 않습니다.

## 아직 확인하지 않은 것

실클라우드별 확장 지원, 권한 정책, endpoint discovery 응답의 모든 형태, 대규모 결과의 성능은 현재 테스트의 범위 밖입니다. 실클라우드 acceptance test와 Python 예제의 실행은 수행하지 않았습니다. cloud 설정과 인증 소스의 주요 선택 경로는 모의 서버로 검증하고, cloud 파서 자체는 Gophercloud 구현을 사용합니다.

새 리소스를 추가할 때는 API 고유의 응답 envelope, pagination link, 상태 대기 실패 값, 확장 query/body를 테스트합니다. 공통 알고리즘을 서비스마다 복사하는 대신 해당 서비스 Adapter가 공통 계약을 유지하는지 확인합니다.

Glance [usage core 계약](../image/v2/serviceinfo/usage_core_test.go)과 [외부 HTTP 계약](../image/v2/serviceinfo/usage_contracts_test.go)은 고정된 current-project singleton·exact canonical JSON·int64 precision/presence·source snapshot·실제 응답/오류 소유권을 검증합니다. [Connection 계약](../connection_image_usage_test.go)은 두 facade의 공유 client·live token·독립 결과를 확인합니다.

Glance [단일 태그·활성화 core](../image/mutation_core_test.go), [concrete 옵션](../image/mutation_options_test.go), [외부 HTTP 계약](../image/mutation_contracts_test.go)은 고정 경로·ID/Name·tag escaping·header 소유권·실제204/오류 증거를 검증합니다. [Connection](../connection_image_mutations_test.go)은 원래 client·live token·독립 acknowledgement를 확인합니다.

Glance 전용 location의 [core](../image/locations_core_test.go)·[옵션](../image/locations_options_test.go)·[외부 HTTP 계약](../image/locations_contracts_test.go)은 literal JSON URL·hash pair·finite 배열·canonical raw metadata, preflight/snapshot과 실제202/200 응답 증거를 검증합니다. [Connection](../connection_image_locations_test.go)과 [generator](../internal/cmd/sdkgen/glance_locations_test.go)는 공유 client 및 기존 native location set PATCH를 확인합니다. [사용법](../image/locations.md)은 서버 비동기 작업의 접수와 검증 완료를 구분합니다.

Glance schema의 [core](../image/schemas_core_test.go)·[옵션](../image/schemas_options_test.go)·[외부 HTTP 계약](../image/schemas_contracts_test.go)은 16개 고정 bodyless GET·canonical nullable 필드·raw additionalProperties·독립 원문과 실제200/오류 증거를 검증합니다. [Connection](../connection_image_schemas_test.go)과 [generator](../internal/cmd/sdkgen/glance_schemas_test.go)는 공유 client·live token·passive discovery와 기존 native binding 보존을 확인합니다.

Glance cache의 [core](../image/cache_core_test.go)·[옵션](../image/cache_options_test.go)·[외부 HTTP 계약](../image/cache_contracts_test.go)은 고정 경로·concrete 기본값과 snapshot·owned target header·정확한 숫자·passive 배열·opaque acknowledgement·owned DELETE404 정책을 검증합니다. [Connection](../connection_image_cache_test.go)과 [generator](../internal/cmd/sdkgen/glance_cache_test.go)는 공유 native client와 기존 binding·문서 생성 정책을 확인합니다.

Glance member의 [core](../image/members_core_test.go)·[옵션](../image/members_options_test.go)·[외부 HTTP 계약](../image/members_contracts_test.go)은 필수 부모·직접 ID·payload·nullable canonical 문자열·유한 lazy 목록·snapshot·실제 응답/오류와 좁은404 기본값을 검증합니다. [Connection](../connection_image_members_test.go)과 [generator](../internal/cmd/sdkgen/glance_members_test.go)는 공유 client·native scope 보존을 확인합니다.

Glance namespace의 [core](../image/v2/metadefnamespaces/core_test.go)·[옵션](../image/v2/metadefnamespaces/options_test.go)·[외부 HTTP 계약](../image/v2/metadefnamespaces/contracts_test.go)은 literal 이름·scalar PUT 교체·nullable raw 응답·명시 zero·advertised pagination·응답 소유권을 검증합니다. [Connection](../connection_image_metadef_namespaces_test.go)과 [generator](../internal/cmd/sdkgen/glance_metadef_namespaces_test.go)는 공유 client와 실제 capability 등록을 확인합니다.

Glance object의 [core](../image/v2/metadefobjects/core_test.go)·[옵션](../image/v2/metadefobjects/options_test.go)·[외부 HTTP 계약](../image/v2/metadefobjects/contracts_test.go)은 고정 namespace·nested 입력 소유권·PUT 교체·canonical 배열·유한 목록·개별/일괄 삭제를 검증합니다. [Connection](../connection_image_metadef_objects_test.go)과 [generator](../internal/cmd/sdkgen/glance_metadef_objects_test.go)는 공유 client와 실제 범위 등록을 확인합니다.

Glance property의 [core](../image/v2/metadefproperties/core_test.go)·[옵션](../image/v2/metadefproperties/options_test.go)·[HTTP 계약](../image/v2/metadefproperties/contracts_test.go)은 필수 Type/Title·flat JSON·snapshot·resource_type·dictionary 소비·오류 소유권을 확인합니다. [Connection](../connection_image_metadef_properties_test.go)과 [generator](../internal/cmd/sdkgen/glance_metadef_properties_test.go)는 공유 client와 실제 scoped definition 등록을 검증합니다.

Glance tag의 [core](../image/v2/metadeftags/core_test.go)·[옵션](../image/v2/metadeftags/options_test.go)·[HTTP 계약](../image/v2/metadeftags/contracts_test.go)은 일곱 경로·snapshot·strict404·bulk 응답·raw marker 페이지와 오류 소유권을 확인합니다. [Connection](../connection_image_metadef_tags_test.go)과 [generator](../internal/cmd/sdkgen/glance_metadef_tags_test.go)는 공유 client와 범위 등록을 검증합니다.

Glance resource type의 [core](../image/v2/metadefresourcetypes/core_test.go)·[옵션](../image/v2/metadefresourcetypes/options_test.go)·[HTTP 계약](../image/v2/metadefresourcetypes/contracts_test.go)은 네 실제 경로·유한 목록·nil/empty 입력·응답 증거와404 기본값을 확인합니다. [Connection](../connection_image_metadef_resource_types_test.go)과 [generator](../internal/cmd/sdkgen/glance_metadef_resource_types_test.go)는 공유 client와 실제 capability 등록을 검증합니다.

핵심 user 고정 조회8개의 최종 감사는 기존43개 계약과112개의 고유 test anchor를 재사용합니다. Password의 기존 표에 성공 반환값과 native203 거부를 추가했고, 기존 handler의 method/token/선택된 microversion 검사는 공개 Gophercloud helper를 사용합니다. 기존 whole-project 검사40 package를 실행한 뒤 API/doc JSON 판정만 갱신하며 같은 전체 HTTP 테스트를 다시 복제하거나 반복하지 않습니다. Source importer를 쓰는5개 generator anchor는 pinned Gophercloud metadata 환경에서 검사합니다.
