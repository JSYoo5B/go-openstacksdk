# SDK 지원 판정대장

이 문서는 생성된 API 호출과 SDK 수준의 지원을 구분하기 위한 판정 기준과 확인한 구현 과제를 기록합니다. 고정 기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**입니다. 아래 초기 조사 결과는 SDK 커밋 **6c4ac1e**를 기준으로 합니다. 이후 추가한 구현은 해당 연산의 증거를 갱신해야 합니다.

[Gophercloud 연산 목록](../api/gophercloud_inventory.json), [공통 리소스 목록](../api/resource_inventory.json), [Python 연산 목록](../api/openstacksdk/manifest.json)은 조사 대상을 찾는 자료입니다. 함수가 생성되거나 모델 이름이 일치하는 것만으로 SDK 동등성이 증명되지는 않습니다.

## 초기 조사 이후 검증한 계약

| 구현 단위 | 검증 증거 | 남은 비교 범위 |
|---|---|---|
| 페이지 순환 중단 (`b5af42b`) | [네 가지 공통 iterator 테스트](../resource/pagination_test.go), [Swift marker 테스트](../api/pagination_contracts_test.go): 자기 링크·A→B→A·query 순서·반복 marker, 오류 한 번 전달, break 이후 링크 검사 생략 | 서버가 계속 다른 URL로 중복 데이터를 반환하는 경우를 일반적으로 deduplicate하지 않음 |
| 선택적 microversion 교집합 협상 (`ba63bde`) | [HTTP discovery 테스트](../microversion_test.go): 인증, project/reverse-proxy 경로, 숫자 비교, 명시 버전 우선, 헤더, 동시 캐시, 취소·재시도 | [문서화한 Go 선택 정책](microversions.md)은 Python의 자동 기본값과 다름. 모든 연산의 필드 capability를 자동 판정하지 않음 |
| 기존 볼륨 부팅 (`350e511`) | [Compute 계약](../compute/boot_volume_test.go), [Connection과 Cinder 연결](../connection_boot_volume_test.go): ID 조회 생략, 정확 이름, 모호성, 삭제 기본값, 실패 시 생성 서버 보존 | cloud `create_server` 전체의 floating IP·추가 볼륨·snapshot 부팅은 별도 |
| 이미지에서 새 볼륨 부팅 (`4c92f74`) | [Nova mapping 계약](../compute/new_boot_volume_test.go): 크기·타입, 2.67 요구, 숫자 minor 비교, 실패 시 무삭제 | Python의 기본 50 GiB 대신 Go는 `WithBootVolumeSize`로 양의 용량을 명시. snapshot source는 별도 |
| Ironic conductor·driver 공통 조회 (`52ec267`) | [native 식별자 계약](../api/baremetal_named_resources_test.go): Hostname/Name, 정확한 이름·중복·미존재·페이지·취소·1.49 헤더 | read-only 리소스에 없는 Delete/Status를 만들지 않음. conductor Get의 fields 선택은 별도 미지원 |
| Neutron 새 floating IP 생성·연결 (`8ec5bc3`) | [포트·IPv4·부분 성공 계약](../network/floating_ip_test.go), [Connection의 Compute 해석](../connection_floating_ip_test.go) | 기존 IP 재사용, Nova fallback, clouds.yaml 자동 network 선택과 서버 생성에서 자동 연결은 별도 |
| Inspector 완료 대기 (`26abc05`) | [UUID·Finished/Error 계약](../api/introspection_contracts_test.go): 요청 ID 고정, typed 메시지, 완료·실패·취소·timeout | Python의 무제한 timeout/ignore_error와 Go의 유한 기본값·실패 반환을 구분 |
| Inspector Start query 보정 | [시작 HTTP 계약](../api/introspection_start_contracts_test.go), [serializer·header 계약](../baremetalintrospection/v1/introspection/start_test.go), [감사된 호출 생성](../internal/cmd/sdkgen/audited_requests_test.go): ManageBoot nil/false/true, 확장 query, POST202, 원래 오류, native drift 거부 | Python과 같은 선택 인자 구분을 확인. Node/Resource 입력, Resource 반환, 연산별 microversion 선택 등 전체 Python Resource create 계약은 이 테스트의 검증 범위 밖 |
| Glance metadata·직접 업로드 (`ad9d06e`) | [업로드 계약](../image/upload_test.go), [재전송·Reader 회귀 계약](../image/upload_retry_test.go): 단일 PUT, 실패 시 생성 객체 보존, caller Close 미호출 | 바이너리 자동 재인증·backoff retry, 기존 이미지의 안전한 재업로드, import/task·checksum·중복 제거는 별도 |
| Nova 서버 tag 범위 (`7da0631`) | [문자열 집합 계약](../api/server_tags_contracts_test.go): 이름 1회 해석, 2.26 요구, escaping, nil/빈 교체, 명시 false·404·HTTP 오류 | Python Server의 tag cache/dirty state 대신 scope의 명시적인 요청·결과 사용 |
| Trove database 범위 (`8138e64`) | [목록·생성·삭제 계약](../api/trove_databases_contracts_test.go): pinned SDK의 list 기반 fetch, 정확 이름, charset, 단일/batch 배열, 오류·삭제 대기 | credential/access 변경은 다른 리소스 동작. batch 원자성·비동기 생성 완료를 추정하지 않음 |
| Nova action 이력 범위 (`b59edb4`) | [목록·상세·event 계약](../api/instance_actions_scope_test.go): requestID·고정 parent, 추가 JSON·헤더, pagination·cycle·취소, non-object 응답 거부 | 이름·삭제·상태 대기를 가정하지 않음. event 노출은 실제 microversion과 cloud 권한 정책을 따름 |
| Trove user/host 범위 (`546b679`) | [계정 식별자 계약](../api/trove_users_contracts_test.go): all-host 정확 이름, WithHost와 default % ID 일관성, literal @/%·2단계 decode, 오류·삭제 대기 | password/credential 갱신, access grant/revoke와 root 관리는 별도 |

이 표는 특정 계약의 검증 기록이며 전체 Python 연산을 `supported`로 판정한 목록이 아닙니다. Inspector Start query 보정의 이번 검증은 `go test -race ./api ./baremetalintrospection/... ./internal/cmd/sdkgen` 범위입니다. 그 이전 구현을 함께 포함한 전체 `go test -race -timeout 60s ./...`와 `go vet ./...`도 통과했습니다.

고정 Gophercloud `baremetalintrospection/v1/introspection.StartIntrospection`은 `ToStartIntrospectionQuery()`의 오류만 검사하고 반환 query를 버립니다. SDK는 [검토한 호출 규칙](../internal/cmd/sdkgen/audited_requests.go)으로 이 연산만 [query를 보존하는 helper](../baremetalintrospection/v1/introspection/start.go)에 연결했습니다. [연산 inventory](../api/gophercloud_inventory.json)의 `request_policy: sdk_query_preserving_start`는 이 보정을 기록하며, 다른 연산의 요청 실행은 바꾸지 않습니다. pinned native 선언의 hash·signature와 입력/결과 shape가 달라지면 생성이 실패하므로 upstream의 실제 수정도 재검토 대상입니다. Python [Proxy.start_introspection](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/baremetal_introspection/v1/_proxy.py#L77)의 `manage_boot=None` 생략과 False/True 전달은 [사용 문서](../baremetalintrospection/v1/introspection/README.md)에 매핑했습니다.

## 판정 상태

| 상태 | 의미 | 완료로 인정할 증거 |
|---|---|---|
| `supported` | 고정 소스의 동작을 Go API로 제공 | 입력·기본값·결과·오류·해당 리소스의 공통 정책을 구현하고 동작을 검증한 테스트와 사용 문서 |
| `go_mapping` | Go의 타입·context·오류·옵션에 맞게 형태나 기본값을 의도적으로 변경 | 실제 사용 가능한 대체 API, Python과 달라지는 의미의 명시, 동일 기능을 검증한 테스트와 사용 문서 |
| `unsupported` | 필요한 기능이 구현되지 않았음을 확인 | 빠진 동작과 이유, 구현할 대상 또는 의존 API. 지원 완료 상태가 아님 |
| `unresolved` | 동작 또는 근거를 아직 판정하지 못함 | 비교할 소스와 다음 확인 작업. 이름 기반 후보 매칭도 이 상태에 해당 |

`unsupported`는 검토를 마쳤다는 뜻이지 프로젝트 목표를 달성했다는 뜻이 아닙니다. 사용자에게 필요한 기능을 제외하는 결정이나 Go에서 제공하기 어려운 구현을 회피하는 데 `go_mapping`을 사용하지 않습니다. 일부 기능만 지원한 연산은 전체 연산을 `supported`로 표시하지 않고 세부 동작을 나눕니다.

## 생성 목록이 포함하지 않는 표면

현재 [inventory.py](../internal/cmd/parity/inventory.py)는 클래스 본문에 직접 선언된 공개 함수만 읽습니다. `methods_in`은 부모 클래스나 descriptor를 탐색하지 않습니다. 직접 선언 집계는 Proxy 1,834개, cloud 397개, Connection 5개이며 전체 공개 API 수를 의미하지 않습니다.

| 종류 | 현재 집계 | 추가로 추적할 표면과 소스 근거 |
|---|---|---|
| 서비스 Proxy 직접 선언 | 서비스 버전별 JSON에 포함 | [compute/v2/_proxy.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)의 공개 연산 등 |
| Proxy 상속 | 제외 | [proxy.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)의 `request`, `should_skip_resource_cleanup`과 외부 `keystoneauth1.adapter.Adapter`의 공개 기능. 외부 의존성 버전까지 확인해야 전체 표면을 확정할 수 있음 |
| Connection 상속 | 직접 선언 5개와 분리 | [connection.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/connection.py)의 cloud mixin 상속. cloud 목록은 모듈별 클래스의 함수 목록이므로 실제 Connection MRO에서 노출되는 연산과 중복·override 관계를 추가로 판정해야 함 |
| 서비스 descriptor와 별칭 | 제외 | [_services_mixin.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/_services_mixin.py)의 `compute`, `block_storage` 등의 서비스 속성과 `volume`, `block_store` 등의 별칭 |
| 런타임 서비스 등록 | `add_service` 함수만 포함 | [Connection.add_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/connection.py#L524)의 `setattr`와 `service.all_types`로 추가되는 서비스 속성. field/query/header 확장과 서비스 등록은 다른 기능임 |
| Resource 기본 동작 | 제외 | [resource.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)의 `create`, `fetch`, `commit`, `delete`, `list`, `find`, 변경 추적, microversion 선택과 capability 검사 |
| Resource mixin과 action | 제외 | [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py#L41)의 `MetadataMixin`, `TagMixin`; [common/metadata.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py), [common/tag.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py)의 공개 동작 |

상속과 descriptor는 소스 AST의 import·부모 클래스·alias를 따라 별도로 조사해야 합니다. SDK 초기화에 실제 인증이나 서비스 호출이 필요할 수 있으므로 Python 모듈 import만으로 표면을 확인하는 방식에 의존하지 않습니다. 외부 의존 클래스나 런타임에 사용자가 추가한 서비스처럼 고정 소스만으로 확정할 수 없는 표면은 조사 범위와 미확정 이유를 기록합니다.

## 대장의 보존과 검증

[판정 JSON](../api/sdk_reviews.json)은 생성 목록과 별도로 보관합니다. [전체 catalog](../api/sdk_support_catalog.json)는 두 고정 소스의 직접 선언 연산 ID와 fingerprint를 기록하며, 판정 JSON에 없는 연산은 `unresolved`입니다. `gophercloud:`와 `python:` 접두사로 같은 이름의 연산도 구분합니다. catalog에는 직접 선언 연산 3,362개가 포함됩니다. 이는 상속·descriptor·Resource 표면까지 포함한 전체 API 수나 SDK 완성도를 의미하지 않습니다.

[paritycheck](../internal/cmd/paritycheck/README.md)는 다음 계약을 검사합니다.

- catalog의 현재 연산 누락·추가·fingerprint 불일치와 source pin 불일치
- 중복 연산 ID·판정·JSON key, 모르는 판정 필드, 존재하지 않는 연산
- 공개 Go 함수·명시적 receiver method, 실제 `Test` 함수와 사용 문서의 존재
- `supported`/`go_mapping`의 API·계약별 테스트·문서 근거, `go_mapping`의 관찰 가능한 차이
- 남은 기능이 있는 연산을 전체 지원으로 판정하는 경우

fingerprint에는 고정 source pin과 생성 목록의 소스 위치·선언 입력 등 metadata를 포함합니다. Go 후보 패키지나 transport 반환 정책만 바뀌면 원본 판정은 유지합니다. 테스트 이름의 존재를 검사하는 것으로 HTTP 의미까지 증명하지는 않으므로 판정자는 원본 계약·테스트 내용을 확인하고 관련 검증을 실행해야 합니다. 상속 구현과 런타임 surface는 별도 조사 범위로 남습니다.

```sh
go run ./internal/cmd/paritycheck
# 두 소스 inventory를 재생성한 뒤 새 연산을 unresolved로 catalog에 추가:
go run ./internal/cmd/paritycheck -sync
```

`-sync`는 수작업 판정을 다시 쓰지 않습니다. 검토한 원본의 fingerprint가 달라지거나 기존 연산이 없어지면 catalog 저장 전에 실패하여 기존 근거를 보존합니다. 미검토 연산도 삭제하지 않으며 저장 중 write 실패에도 기존 catalog가 남습니다. 새 소스를 다시 검토하고 판정 상태와 근거를 함께 갱신해야 합니다. `make check`도 이 검증을 실행합니다.

현재 durable 판정은 native 암호 조회와 보정한 Inspector 시작의 Go 매핑 두 항목부터 연결했습니다. 대응 Python 연산은 확인한 세부 계약을 기록하고 전체 상속·리소스 의미 비교가 남아 `unresolved`로 유지합니다. 다른 구현의 증거도 같은 방식으로 점진적으로 연결하며, 생성된 함수 수를 지원 판정으로 대체하지 않습니다.

## 증거를 연결할 수 있는 기존 구현

아래 기능은 구현과 의미 있는 HTTP 계약 테스트가 있어 첫 판정 후보로 사용할 수 있습니다. 표는 테스트 소스를 확인한 결과이며, 최종 지원 판정 전에는 해당 테스트 실행 결과와 Python의 전체 연산 계약을 함께 확인해야 합니다.

| 대상 | Go 구현과 테스트 | 판정에서 구분할 의미 |
|---|---|---|
| Nova 관리자 암호 읽기 | [servers/password.go](../compute/v2/servers/password.go), [password 계약 테스트](../api/password_contracts_test.go) | Python은 암호 누락 시 `None`, Go는 빈 string. RSA 복호화는 선택 기능이며 기본값은 암호화된 값 |
| QoS bandwidth limit·DSCP marking·minimum bandwidth의 Get/Create/Update/Delete/List | [rules API](../network/v2/extensions/qos/rules/api_generated.go), [부모 scope](../network/v2/extensions/qos/rules/scopes_generated.go), [QoS 계약 테스트](../api/qos_contracts_test.go), [공통 scope 테스트](../api/scoped_contracts_test.go) | policy를 먼저 고정하는 Go API, concrete opts, 공통 Delete의 미존재 허용. Python Find의 기본 미존재 허용과 추가 query 인자는 별도 비교 필요 |
| Swift 이름·opaque object key·metadata HEAD | [container Collection](../objectstorage/v1/containers/resources.go), [object scope](../objectstorage/v1/objects/resources.go), [Swift 리소스 계약 테스트](../api/swift_resources_contracts_test.go), [Swift 목록 계약 테스트](../api/swift_listing_contracts_test.go) | slash를 포함한 object key와 URL escape, HEAD metadata와 헤더, 명시 Name/ID 정책. Python metadata 갱신의 기본 refresh까지 지원한 것으로 확대하지 않음 |
| DNS optional headers와 Swift versioned COPY query | [optional builder 테스트](../api/optional_builders_test.go), [optional trait 생성기](../internal/cmd/sdkgen/optional.go) | native 함수가 선택 interface로 읽는 header/query를 SDK builder가 보존. 이것만으로 모든 확장 및 microversion 지원이 증명되지는 않음 |

Identity v2 인증 응답의 token·catalog·user·metadata 보존과 Ironic virtual media의 typed body·추가 JSON·header 보존도 [인증 테스트](../api/authentication_contracts_test.go)와 [virtual media 테스트](../api/virtual_media_contracts_test.go)에서 확인할 수 있습니다. Python Proxy 목록에 직접 대응하지 않는 Gophercloud 연산도 원본 API ID를 가진 별도 항목으로 추적해야 합니다.

## 우선 구현할 차이

초기 조사 당시 공통 리소스 목록의 미결 항목은 79개입니다. 그 안에는 CRUD 리소스뿐 아니라 인증, URL 도우미, list-only 자료, project별 singleton도 있으므로 전부 같은 Collection으로 만들지 않습니다.

현재 binding은 일반 Collection 108개, 부모 Collection 범위 18개와 별도 tag set 1개입니다. 공통 binding이 없는 72개 패키지는 계속 조사 대상이며, binding의 추가만으로 대응 Python 연산 전체를 지원 완료로 판정하지 않습니다. 아래 표는 초기 조사 우선순위이며 완료한 세부 계약과 현재 남은 범위는 위 증거 표에 기록합니다.

| 우선 과제 | 확인한 코드 근거 | 필요한 구현과 검증 |
|---|---|---|
| 공통 페이지네이션의 cycle 중단 | [Stream](../resource/stream.go)과 [Collection.List](../resource/collection.go)가 `pagination.Pager.EachPage`에 위임. Gophercloud v2.15.0 `pagination/pager.go`는 `NextPageURL`을 다음 요청 URL로 대입하며 반복 URL을 검사하지 않음 | nonempty page가 자기 URL 또는 A→B→A를 반환할 때 `All`/`Find`가 중복 데이터를 계속 읽지 않도록 공유 guard 적용. linked URL·marker 페이지, break, 취소, 빈 페이지, 원래 오류 보존 테스트 필요 |
| 이름이 다른 Ironic 조회/목록의 공통 정책 | [conductors API](../baremetal/v1/conductors/api_generated.go)의 `Get/List`와 `Conductor.Hostname`; [drivers API](../baremetal/v1/drivers/api_generated.go)의 `GetDriverDetails/ListDrivers`, `Driver.Name`; [introspection API](../baremetalintrospection/v1/introspection/api_generated.go)의 `GetIntrospectionStatus/ListIntrospections` | SDK 소유의 검토된 binding으로 alternate getter/lister와 Hostname 식별자를 연결. introspection은 `Finished bool`/`Error string`을 이용하므로 문자열 Status 기반 Wait와 별도 waiter가 필요. 없는 Delete를 만들지 않음 |
| Heat stack과 복합 부모 식별자 | [stacks API](../orchestration/v1/stacks/api_generated.go): Get/Delete에 `stackName, stackID`, 목록 `ListedStack`과 조회 `RetrievedStack`이 다름. [stackresources](../orchestration/v1/stackresources/api_generated.go), [stackevents](../orchestration/v1/stackevents/api_generated.go)에 추가 부모·자식 인자 | 공통 stack 모델과 library-owned identity/name 해석, 이름 중복, canonical ID 보존, 부모 범위. `*_FAILED` 상태와 deletion waiter를 검증. Python `get_stack`의 `resolve_outputs=True`도 별도 계약으로 추적 |
| Trove database/user의 instance 범위 | [databases API](../db/v1/databases/api_generated.go), [users API](../db/v1/users/api_generated.go)는 parent+name List/Delete와 batch Create만 제공. Python [Database](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/database/v1/database.py)는 name이 alternate ID이며 `allow_fetch`를 켜지 않음 | instance를 고정하는 scope, exact name lookup, batch/single 생성의 명확한 Go 매핑, missing Delete 정책. Python Proxy에 `get_database`가 있어도 실제 Resource capability가 없으므로 존재하지 않는 GET endpoint를 추정해 만들지 않음 |
| Nova instance action/tag의 서버 범위 | [instanceactions API](../compute/v2/instanceactions/api_generated.go)는 `Get(serverID, requestID)`의 `InstanceActionDetail`과 List의 `InstanceAction`이 다름. [tags API](../compute/v2/tags/api_generated.go)는 모든 호출에서 serverID를 요구 | 서버를 먼저 해석하는 scope, requestID 식별자, detail/list 변환의 정보 보존, tags의 Add/Check/Replace/Remove 정책. tag나 requestID를 일반 리소스 이름으로 잘못 추측하지 않음 |
| Manila access rule의 조회 범위 | [shareaccessrules API](../sharedfilesystems/v2/shareaccessrules/api_generated.go)는 `Get(accessID)`와 `List(shareID)`의 스코프가 다르고 List가 slice를 반환. 생성·해제는 share action과 연결해야 함 | share parent scope와 typed iterator, 전역 accessID 조회의 부모 확인 정책, share의 allow/deny action 연결, access 상태 대기. 잘못된 부모에 대한 오류를 숨기지 않음 |
| quotas·limits 등 singleton/read-only 자료 | [compute quotasets](../compute/v2/quotasets/api_generated.go)는 tenant ID별 Get/Detail/Update/Delete이며 List가 없음. Cinder·Neutron·Manila quota와 각 서비스 limits도 일반 CRUD 목록과 다름 | project Ref를 해석하는 typed singleton API, Get/Reset/Update와 기본 current-project 정책. 조회가 없는 리소스에 Find/List/Wait를 추가하지 않고 실제 capability를 문서화 |
| Resource 상태와 응답 확장 정책 | 대다수 [생성 API](../image/v2/images/api_generated.go)는 Gophercloud 모델 alias와 Extract 결과만 반환. Python [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)은 dirty 필드와 patch 정책을 처리 | 변경 추적·commit 또는 동일 작업을 제공하는 명시적 Go update API의 계약을 결정하고 검증. header/추가 JSON 보존 정책도 서비스별 확인. Glance `Image.Properties`는 Gophercloud에서도 추가 필드를 보존하므로 모든 alias가 확장 응답을 버린다고 가정하지 않음 |
| Python에만 존재하는 서비스/버전 | Python manifest에는 accelerator, clustering, instance_ha, image v1 등이 있으나 [Connection 서비스 registry](../connection_services_generated.go)와 Gophercloud v2.15.0 호출 목록에 대응 구현이 없음 | 고정 Python Resource/endpoint 계약에서 typed API와 서비스 연결을 구현. 다른 서비스의 RawClient나 generic HTTP 요청을 해당 서비스 SDK 지원으로 판정하지 않음 |

복합 작업에서는 floating IP 재사용·서버 생성 시 자동 연결, 추가 볼륨·snapshot 부팅, 이미지 import·checksum 및 안전한 바이너리 자동 재시도가 별도 남은 계약입니다. Python [cloud create_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L915)는 `auto_ip=True`, `reuse_ips=True`, boot volume·추가 volume 등을 처리합니다. Go의 기존 이미지 기반 생성과 선택적 ACTIVE 대기, 또는 기존 boot volume 한 가지를 지원하는 것으로 전체 `create_server`를 완료 처리하지 않습니다.

## 완료 판정

프로젝트 완료를 주장하려면 다음 근거가 함께 필요합니다.

1. 두 고정 소스의 연산뿐 아니라 확인한 상속·descriptor·Resource 표면을 빠짐없이 추적하고, 미확정 범위를 공개합니다.
2. 각 API의 필수 입력, 생략/false/빈 값/zero 구분, library-owned builder, 기본값, query/header/body 확장, 결과와 오류를 실제 계약 테스트로 확인합니다.
3. 이름 조회, 중복, 미존재, pagination, cancellation, 삭제·상태 대기는 해당 리소스가 지원하는 동작에 맞게 검증합니다. 표면이 비슷한 단일 서비스의 테스트를 전체 서비스의 증거로 사용하지 않습니다.
4. 서비스 간 작업은 정상 흐름과 각 단계의 실패·부분 성공·재시도·대기·정리 정책을 검증합니다. 자동 삭제나 rollback은 문서화한 정책을 따라야 합니다.
5. 모든 서비스와 전체 사용 문서가 구현된 공개 API를 사용하고, 예제 빌드와 문서 링크를 확인합니다.
6. `supported`/`go_mapping` 항목마다 실제 증거가 있고, 요청 범위의 `unsupported`/`unresolved` 항목이 남아 있으면 전체 목표를 완료로 표시하지 않습니다.

microversion 자동 선택·협상과 요청 필드 capability 검증 역시 공통 SDK 계약입니다. [WithMicroversion](../connection_options.go)의 명시 선택 기능만으로 자동 협상까지 지원한다고 판정하지 않습니다. 이 대장의 각 조사 항목은 구현과 검증이 추가될 때 갱신해야 하며, 생성된 transport 함수 수는 완료율의 대체 지표로 사용하지 않습니다.
