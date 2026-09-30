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

Go 1.25 이상, 의존성 다운로드, localhost 포트 바인딩 허용이 필요합니다. `make check`는 vet, race test, gofmt 상태를 확인합니다. `go test`는 실행 예제를 빌드하지만 예제의 main을 실행하지 않습니다.

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
| `resource/collection_test.go` | 잘못된 참조와 iterator 옵션을 HTTP 요청 전에 거부 |
| `resource/pagination_test.go`, `api/pagination_contracts_test.go` | linked URL·query 순서·Swift marker 순환 중단, 오류 한 번 전달, break 시 후속 링크 검사 생략 |

페이지 테스트는 서로 다른 페이지의 같은 이름을 검사합니다. `break` 테스트는 다음 페이지 요청 횟수가 0인지 확인합니다. 시간 관련 테스트는 짧은 SDK timeout을 사용하고 `errors.Is(context.DeadlineExceeded)`를 검사합니다. 특정 실행 시간과 동일하다고 가정하지 않습니다.

## 아직 확인하지 않은 것

실클라우드별 확장 지원, 권한 정책, endpoint discovery 응답의 모든 형태, 대규모 결과의 성능은 현재 테스트의 범위 밖입니다. 실클라우드 acceptance test와 Python 예제의 실행은 수행하지 않았습니다. cloud 설정과 인증 소스의 주요 선택 경로는 모의 서버로 검증하고, cloud 파서 자체는 Gophercloud 구현을 사용합니다.

새 리소스를 추가할 때는 API 고유의 응답 envelope, pagination link, 상태 대기 실패 값, 확장 query/body를 테스트합니다. 공통 알고리즘을 서비스마다 복사하는 대신 해당 서비스 Adapter가 공통 계약을 유지하는지 확인합니다.
