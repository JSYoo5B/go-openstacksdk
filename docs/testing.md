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
| `resource/collection_test.go` | 잘못된 참조와 iterator 옵션을 HTTP 요청 전에 거부 |
| `resource/pagination_test.go`, `api/pagination_contracts_test.go` | linked URL·query 순서·Swift marker 순환 중단, 오류 한 번 전달, break 시 후속 링크 검사 생략 |
| `internal/cmd/paritycheck/reviews_test.go` | catalog 추가 시 기존 판정 보존, 원본 입력 drift 거부, API·테스트·문서 근거, 중복 ID/JSON key와 불완전 지원 판정 거부 |
| `api/server_tags_contracts_test.go` | 서버 1회 해석, 2.26 요구, tag escaping, nil/빈 교체·명시 false·404 정책과 원래 HTTP 오류 |
| `api/instance_actions_scope_test.go` | requestID 조회, 목록·상세·event 확장 및 원본 JSON/헤더, pagination·break·cycle·취소 |
| `api/trove_databases_contracts_test.go`, `api/trove_users_contracts_test.go` | instance 1회 해석, 목록 기반 exact 조회, 단일/batch 배열 Create, 이름 인코딩·미존재 삭제·대기 |
| `api/accelerator_read_test.go`, `api/accelerator_actions_test.go`, `api/accelerator_boundaries_test.go` | SDK 소유 Cyborg 연결·microversion·UUID, 응답 JSON/헤더, 목록 중단·cycle·origin, enable/disable·program payload, 빈 페이지 continuation, redirect 경계·최신 token·재인증/취소와 사전 검증 |
| `api/accelerator_profiles_test.go`, `api/accelerator_attributes_test.go`, `api/accelerator_singleton_test.go`, `api/accelerator_status_filter_test.go` | 프로필 exact 이름·UUID 삭제·array-one 생성, attribute key/ID 구별·flat 생성, singleton 응답 cardinality, snapshot·로컬 상태 필터와 명시 query 보존 |
| `api/accelerator_requests_test.go`, `api/accelerator_request_binding_test.go`, `api/accelerator_request_response_boundary_test.go` | ARQ 전체 batch·부분 해석·raw bytes·오류 보존, state 대기, collection PATCH202·service token·2.1 project gate, single/batch/instance DELETE204 selector, accepted201 body 취소·decode 오류에서 무재시도·body close |
| `api/project_quotas_contracts_test.go`, `connection_quotas_test.go` | 고정 project quota, 별도 Keystone 이름 조회·auth scope, zero/-1/force false, singleton 결과·오류·Reset |
| `api/heat_stacks_contracts_test.go` | name+ID identity, output query, summary/detail 차이, linked/marker pagination, 고정 대상 waiter·실패/삭제 완료 |
| `api/heat_stackresources_contracts_test.go`, `api/heat_stackresources_responses_test.go` | resource_name/논리·물리 ID 구별, nested owner·health false·metadata·대기, raw 필드·큰 숫자·독립 header·malformed 목록 거부 |
| `api/heat_stackevents_scope_test.go`, `api/heat_stackevents_pagination_test.go` | stack/resource 이벤트 경로·typed query·marker·cycle·break·취소, resource-scoped 단건 GET·raw 필드·독립 header |
| `api/share_access_rules_scope_test.go`, `api/share_access_rules_errors_test.go` | Manila2.45/2.82 access rule 조회·action·잠금, share 일치·부모404/403 보존, 취소·waiter |

페이지 테스트는 서로 다른 페이지의 같은 이름을 검사합니다. `break` 테스트는 다음 페이지 요청 횟수가 0인지 확인합니다. 시간 관련 테스트는 짧은 SDK timeout을 사용하고 `errors.Is(context.DeadlineExceeded)`를 검사합니다. 특정 실행 시간과 동일하다고 가정하지 않습니다.

## 아직 확인하지 않은 것

실클라우드별 확장 지원, 권한 정책, endpoint discovery 응답의 모든 형태, 대규모 결과의 성능은 현재 테스트의 범위 밖입니다. 실클라우드 acceptance test와 Python 예제의 실행은 수행하지 않았습니다. cloud 설정과 인증 소스의 주요 선택 경로는 모의 서버로 검증하고, cloud 파서 자체는 Gophercloud 구현을 사용합니다.

새 리소스를 추가할 때는 API 고유의 응답 envelope, pagination link, 상태 대기 실패 값, 확장 query/body를 테스트합니다. 공통 알고리즘을 서비스마다 복사하는 대신 해당 서비스 Adapter가 공통 계약을 유지하는지 확인합니다.
