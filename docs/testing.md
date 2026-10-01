# 테스트

테스트는 로컬 HTTP 모의 서버로 SDK에서 Gophercloud를 거쳐 HTTP 요청·응답을 처리하는 경로를 검증합니다. 실클라우드 인증이나 리소스 생성은 수행하지 않습니다.

## 실행

프로젝트 루트에서:

```sh
make check
go test -race ./...
go test -coverpkg=./... ./...
go test -run TestCreate ./...
```

Go 1.25 이상, 의존성 다운로드, localhost 포트 바인딩 허용이 필요합니다. `make check`는 vet, 60초 package timeout을 적용한 race test, 지원 판정 근거와 gofmt 상태를 확인합니다. `go test`는 실행 예제를 빌드하지만 예제의 main을 실행하지 않습니다.

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
| `resource/metadata_test.go`, `internal/rest/response_test.go`, `internal/rest/list_test.go` | 추가 JSON·null·생략·정확한 숫자, 응답 header 소유권·accepted 오류 증거, 고정 URL·retry/reauth·origin, lazy break·marker/link 순환·query 보존·명시 paging 정책 |
| `connection_sdk_owned_test.go`, `connection_sdk_resources_test.go`, `internal/cmd/sdkgen/sdk_owned_services_test.go` | Senlin·Masakari catalog root/version/project/proxy, raw escaped tenant·Masakari discovery, 공유 인증·캐시·취소·microversion, typed 서비스 연결·singleton/list-only 생성 정책 |
| `api/instanceha_segments_test.go`, `api/instanceha_hosts_test.go` | segment UUID·DB ID 구별, enabled1.2·false·snapshot, 고정 segment host·부모 이름 1회 해석, POST202/201·PUT200·DELETE204, exact 이름·페이지·HTTP 오류·취소 |
| `api/instanceha_notifications_test.go`, `api/instanceha_vmoves_test.go` | notification UUID·payload/timestamp, raw workflow 숫자, 고정 notification scope·VM move1.3, 상태 polling·실패·취소, 부모와 server identity 구별·accepted 응답 증거 |
| `api/clustering_discovery_test.go`, `api/clustering_read_test.go` | build-info singleton·type 이름/schema·profile ops1.4, action epoch/target·event level string/숫자/null/생략·큰 정수·반복 filter snapshot, service1.7/list-only, 페이지·strict envelope·HTTP 오류 |
| `internal/rest/list_short_page_test.go`, `api/clustering_read_test.go` | 명시 limit에서 짧은 페이지 뒤에도 wire ID marker 유지, consumer 모델 변경·break·no-limit·빈 페이지·순환 정책 |
| `internal/senlin/mutation_test.go`, `internal/senlin/filter_test.go`, `internal/senlin/query_test.go` | optional JSON 생략/null/빈 객체, snapshot·원문 숫자·SDK 소유 field/header 보호, 로컬 재귀 subset·배열·타입·정확한 decimal/거대 지수 비교, sort grammar와 별도 key allowlist |
| `api/clustering_profiles_test.go`, `api/clustering_policies_test.go`, `connection_sdk_resources_test.go` | profile/policy CRUD·객체 PATCH·Validate1.2, spec/metadata snapshot·null/빈 객체·숫자, 이름/ID·미존재·중복·409, 로컬 Body 필터·페이지·accepted 오류·인증/버전 공유 |
| `api/clustering_mutation_recheck_test.go`, `api/clustering_profiles_update_snapshot_test.go`, `api/clustering_policies_response_test.go` | custom option·이름 lookup 이후 source/version 재검사, 준비한 body/header 소유권, Get/Validate accepted malformed 응답 증거·재전송 없음·미선언 success code 거부 |
| `resource/delete_identity_test.go` | 이름 조회 응답의 빈/공백/다른 종류 ID를 binding 정책으로 검사해 collection 경로 삭제를 막고, 유효한 ID는 한 번 해석 후 삭제 |
| `request/optional_test.go`, `api/clustering_scalar_options_test.go` | typed 값의 생략/null/false/zero·빈 문자열 구분, 실패한 decode에서 이전 값 보존, With 함수 재사용과 caller config 변경 후 독립 요청 |
| `internal/senlin/async_test.go`, `api/clustering_async_location_case_test.go` | 선택한 action collection·origin·reverse prefix, required/empty/중복·대소문자 Location, accepted 원문/헤더/status 보존, 조회·재전송 없음 |
| `api/clustering_clusters_test.go`, `api/clustering_clusters_response_test.go`, `api/clustering_clusters_pagination_test.go` | cluster POST201·PATCH/DELETE202, raw 숫자·생략/null·profile_only1.6, 이름 해석 전 snapshot·source 재검사, 일반/force404 차이, malformed 응답·짧은 페이지·로컬 필터·취소 |
| `api/clustering_nodes_test.go`, `api/clustering_nodes_pagination_test.go` | node POST/PATCH/DELETE202·필수 action 참조, physical ID/index/details·tainted1.13, nullable 입력·snapshot·strict force404, query/로컬 필터·raw marker·다중 페이지 Find 중복/후속 오류 |
| `connection_senlin_async_test.go`, `internal/cmd/sdkgen/sdk_owned_services_test.go` | Clusters/Nodes/Actions의 CRUD·ScaleOut/Check/Cancel client·최신 token·1.13·reverse prefix 공유, 자동 조회 없이 submission 후 명시 action 조회, 비동기 결과를 버리는 Collection.Delete 미지원 정책과 재생성 |
| `resource/list_options_snapshot_test.go` | native·SDK 소유 lazy iterator의 caller 옵션 slice 교체/nil·반복 iteration·잘못된 옵션 소유권과 HTTP 사전 차단 |
| `internal/senlin/command_test.go` | 빈 command 객체·typed key 보호·plugin 입력 snapshot, 응답 action과 필수 Location ID 일치·accepted 오류 증거·경로와 header 소유권 |
| `api/clustering_action_update_test.go` | PATCH202·CANCELLED·1.12, force query 생략/false/true, snapshot·이름 lookup 후 source/version 검사, empty/opaque 원문·native HTTP/context 오류 |
| `api/clustering_clusters_commands_test.go` | ScaleIn/Out count:null·정확한 Resize 숫자·optional 값, 노드 배열/map·gate1.3/1.4, parent 1회 해석·snapshot, strict202 action/Location·오류 원문·취소·재전송 없음 |
| `api/clustering_nodes_commands_test.go` | Check/Recover 기본 객체, operation 생략/empty/null·params/null·명시 check1.6·ops1.4, plugin key 범위, 입력/header snapshot·lookup 후 gate, missing/ambiguous·accepted 오류·HTTP 원인 |

페이지 테스트는 서로 다른 페이지의 같은 이름을 검사합니다. `break` 테스트는 다음 페이지 요청 횟수가 0인지 확인합니다. 시간 관련 테스트는 짧은 SDK timeout을 사용하고 `errors.Is(context.DeadlineExceeded)`를 검사합니다. 특정 실행 시간과 동일하다고 가정하지 않습니다.

## 아직 확인하지 않은 것

실클라우드별 확장 지원, 권한 정책, endpoint discovery 응답의 모든 형태, 대규모 결과의 성능은 현재 테스트의 범위 밖입니다. 실클라우드 acceptance test와 Python 예제의 실행은 수행하지 않았습니다. cloud 설정과 인증 소스의 주요 선택 경로는 모의 서버로 검증하고, cloud 파서 자체는 Gophercloud 구현을 사용합니다.

새 리소스를 추가할 때는 API 고유의 응답 envelope, pagination link, 상태 대기 실패 값, 확장 query/body를 테스트합니다. 공통 알고리즘을 서비스마다 복사하는 대신 해당 서비스 Adapter가 공통 계약을 유지하는지 확인합니다.
