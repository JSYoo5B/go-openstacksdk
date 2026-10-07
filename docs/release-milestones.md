# 단계별 사용·릴리즈 기준

전체 SDK 완성까지 기다리지 않고, 검증된 핵심 user 흐름부터 개발 preview와 설치 가능한 alpha를 구분해 제공합니다. 전체 API 목표와 [서비스 우선순위](implementation-plan.md)는 유지합니다. 이 문서의 버전 이름은 예정된 범위이며 아직 배포한 tag가 아닙니다.

## 지금 실행 가능한 개발 preview

프로젝트 루트에서 `make smoke`를 실행합니다. 기존 계약 테스트를 선택해 한 번의 race test로 실행하며, 테스트 이름이 없어지거나 skip되어도 실패합니다. 결과는 `.reports/core-smoke.json`, 실행 로그는 `.reports/core-smoke.log`에 남습니다. 보고서는 실행 시각, Go 소스 SHA256, 정확한 테스트와 각 흐름의 PASS/FAIL을 포함합니다.

2026-10-08 로컬 검증에서 아래 5개 흐름의 9개 테스트 그룹이 모두 PASS했습니다.

| 흐름 | 재사용한 그룹 | 확인한 범위와 사용법 |
|---|---:|---|
| 서버·네트워크 | 3 | 이름으로 의존 리소스 조회, 서버 생성·IP 연결·대기 순서와 부분 결과, 네트워크 변경 후 역할 cache 갱신. [자동 IP 생성](../compute/create-with-automatic-floating-ip.md), [네트워크 변경](../network/network-mutations.md) |
| 볼륨 | 2 | 생성 후 기본 상태 대기, Nova 연결과 Cinder 완료 대기, 단계별 owned 응답. [생성](../blockstorage/create-volume.md), [연결](../blockstorage/attach-volume.md) |
| 이미지 | 2 | metadata·stage·import 접수, 다운로드 바이트와 checksum. import 접수 테스트는 ACTIVE 완료를 주장하지 않습니다. [생성·import](../image/create-import.md), [다운로드](../image/download.md) |
| Secret | 1 | 공유 client와 최신 토큰을 사용한 metadata→조건부 payload 조회. [사용법](../keymanager/v1/secrets/README.md) |
| Swift object | 1 | 입력 스트림 소유권과 object 요청, checksum에 따른 업로드 생략. [사용법](../objectstorage/v1/objects/create.md) |

검증은 공개 Gophercloud testhelper의 로컬 HTTP 서버를 사용합니다. 실제 OpenStack 인증·배포 정책·리소스 생성은 실행하지 않았습니다. smoke는 빠르게 사용 흐름을 확인하는 명령이며 전체 API 지원 수는 [연산별 자동 집계](implementation-plan.md#현재-집계와-진행-중인-작업)로 따로 확인합니다.

## 다음 배포 단위

| 단계 | 상태 | 완료 조건 |
|---|---|---|
| 로컬 개발 preview | 위 5개 흐름 PASS | `make smoke`가 실제 테스트 실행 결과를 저장 |
| `v0.1.0-alpha.1` 핵심 user preview | 준비 중, 미배포 | 공개 module·generator·문서·판정 참조, 외부 local-replace·정확한 커밋 원격 설치/빌드, smoke·전체 check PASS. [검증한 revision 사용](install.md) 가능; 예정 alpha tag 배포는 남음 |
| 핵심 user 후속 alpha | 진행 중 | 남은 named API를 작은 기능 묶음으로 완성하고, 기능·테스트·Python 비교 문서·지원 판정·변경 기록을 함께 제공 |
| 핵심 admin 후속 배포 | 대기 | user 단계 이후 관리자 계약·옵션·권한 차이 검증 |
| 후속 user 배포 | 대기 | Octavia·Designate → Ironic·Introspection → 나머지 서비스 순서 |
| 후속 admin 배포 | 대기 | 후속 user 이후 같은 내부 서비스 순서 |

각 단위는 실제 입력·기본값·결과·오류 계약을 시작할 때 고정합니다. 검증된 공통 엔진과 테스트 fixture를 재사용하고 변경 부분을 집중 검증합니다. 공유 엔진이나 fixture 변경은 영향받는 전체 회귀 검증을 묶어 실행합니다. 문구만 수정할 때 동일한 전체 Go 검증을 반복하지 않습니다. 기능별 작은 커밋과 push에 맞춰 구현 단계와 지원 개수를 갱신합니다.

실클라우드 acceptance는 실행 환경과 인증이 준비되면 별도 결과로 기록합니다. 로컬 계약 테스트나 alpha 배포를 실클라우드 검증으로 표시하지 않습니다.
