# Cinder v3 볼륨 합계

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [summary Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1059-L1080), [BlockStorageSummary](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/block_storage_summary.py), [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)입니다. 집계 경로는 `/volumes/summary`입니다. [ListSummary](volume-summary-python.md)의 `/volumes` 개별 행 목록과 다른 연산입니다.

| Python | Go | 동작 |
|---|---|---|
| `conn.block_storage.summary(all_projects=False)` | `service.Volumes.SummaryRecord(ctx)` | 현재 프로젝트 합계, `all_tenants=False` |
| `summary(all_projects=True)` | `WithSummaryAllProjects(true)` | `all_tenants=True`, 실제 범위는 서버 권한으로 결정 |
| `summary(..., headers={"X-Key": "v"})` | `WithSummaryQuery(map[string]any{"headers": map[string]any{"X-Key": "v"}})` | Requests dictionary query 확장, `?headers=X-Key` |
| session의 실제 추가 HTTP 헤더 | `WithSummaryHeaders(...)` | Go typed 확장, raw `headers` query와 별개 |
| `summary(..., key=value)` | `WithSummaryQuery(map[string]any{"key": value})` | 두 번째 GET query, JSON null 생략·반복값 유지 |
| `summary(..., microversion="3.90")` | `WithSummaryMicroversion("3.90")` | 두 번째 GET만 해당 버전 |
| `summary(..., microversion="")` | `WithSummaryMicroversion("")` | 두 번째 GET의 버전 헤더 생략 |
| 고정 SDK의 `_get` 후 재차 `fetch` | 옵션 생략 | 첫 GET query 없음, 두 번째 GET에 합계 query |
| 한 번만 합계 조회 | `WithSummarySingleFetch(true)` | 명시적인 Go 개선: 불필요한 첫 GET 생략 |

Python의 `all_projects`는 필수 bool이지만 Go는 안전한 현재 프로젝트 기본값을 제공합니다. 고정 Python 구현은 `_get(requires_id=False)`로 먼저 조회한 뒤 `volume-summary` key와 query를 지정하여 다시 조회합니다. 기본 Go 호출도 두 번의 실제 member 요청을 보존합니다. `SingleFetch(true)`는 이 순서를 개선하는 명시적인 옵션이고, 첫 응답의 seed 필드를 합치는 효과도 생략합니다.

## 독립 Python main

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.36")
    parser.add_argument("--all-projects", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    value = conn.block_storage.summary(all_projects=args.all_projects)
    print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용한 별도 모듈에서 `go run . -cloud dev`로 실행합니다. `-all-projects`와 `-single-fetch`를 필요에 따라 지정합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.36", "selected Cinder version")
    all := flag.Bool("all-projects", false, "ask for all projects")
    single := flag.Bool("single-fetch", false, "omit pinned redundant first GET")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    value, err := service.Volumes.SummaryRecord(ctx,
        volumes.WithSummaryAllProjects(*all),
        volumes.WithSummarySingleFetch(*single))
    if err != nil { log.Fatal(err) }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}
```

## 버전·권한·실제 응답

[공식 Cinder API](https://docs.openstack.org/api-ref/block-storage/v3/#get-volumes-summary)는 집계를 **3.12+**, metadata 반환을 **3.36+**로 제공합니다. 고정 Source class의 `_max_microversion`은 **3.36**입니다. selected version이 있으면 이를 유지하고, 없으면 기존 공통 discovery로 `min(server maximum, 3.36)`을 선택합니다. 높은 selected `3.90`을 자동 downgrade하지 않습니다. 낮은 selected `3.11`에서도 Source처럼 요청을 보내 실제 서버의 404/정책 오류를 보존하며, SDK가 존재하지 않은 지원 상태를 만들지 않습니다. fetch override는 두 번째 요청에만 적용되고 빈 문자열은 해당 요청에서 버전 헤더를 생략합니다.

[Cinder volume:get_all 정책](https://github.com/openstack/cinder/blob/master/cinder/policies/volumes.py)은 프로젝트 reader를 포함합니다. [get_volume_summary](https://github.com/openstack/cinder/blob/master/cinder/volume/api.py)의 전역 조회는 `all_tenants`와 `context.is_admin`이 모두 참일 때만 적용됩니다. 일반 사용자가 `all_projects=True`를 보냈다고 전체 프로젝트 접근 권한을 얻지는 않습니다. 배포의 정책 변경은 서버가 판단합니다.

`VolumeSummaryRecord.Resource`는 `id`, `name`, `total_size`, `total_count`, `metadata`의 untyped JSON을 보존하고 생략은 null로 표시합니다. 첫 조회의 declared 필드는 두 번째 응답에서 생략되면 남고, 명시적인 null은 덮어씁니다. `Wire`는 현재 응답의 literal object, `Envelope`·`Header`·`StatusCode`는 마지막 실제 member receipt입니다. `Requests`는 각 Source fetch의 최종 관찰을 순서대로 소유합니다(기본 두 개). native clean retry 중간 응답 전체를 별도 수집하는 목록은 아닙니다. `Discovery`는 실제 버전 응답을 별도로 소유합니다. 두 단계의 응답을 하나의 가짜 HTTP proof로 합치지 않습니다. 첫 조회 또는 버전 확인이 실패하면 이후 요청을 수행하지 않습니다.

고정 `Resource.fetch`처럼 accepted non-JSON/empty body는 기존 declared seed/default를 유지합니다. 잘못된 JSON object 모양·UTF-8 및 read/Close/cancellation/source 오류는 실제 receipt를 보존한 오류입니다. 401의 native `*ErrUnableToReauthenticate`는 `ErrOriginal`·`ErrReauth` public 필드를 유지합니다. 물리 응답 장애에는 retry/backoff/reauth를 호출하지 않으며, 깨끗한 401 재인증과 일반 retry는 유지합니다. SDK-owned route·GET body·version 및 금지 헤더는 재시도에서도 바뀔 수 없습니다.

`SummaryOpts`·옵션 생성·콜백 경계에서 query JSON, map, raw bytes, version pointer를 복사합니다. nil map은 안전한 빈 map으로 준비합니다. 원래 service client는 수정하지 않습니다. `all_tenants`·base_path·session 등 Python 제어 인자의 raw query 주입은 JSON marshaler와 HTTP 전에 거부하고 typed 옵션을 사용합니다. 이 사전 거부는 고정 Python에서 첫 GET 뒤에 발생할 수 있는 kwargs 충돌을 개선한 Go 입력 경계입니다. Headers는 Go 확장이며 인증·버전 제어는 library가 소유합니다.

standalone leaf의 connection-derived `location`은 null인 명시적인 SDK-R1 범위입니다. unknown 응답 필드와 서버의 location은 `Wire`에 그대로 남습니다. Python object의 dirty/cache/session과 고정 Source warning 문자열을 보장하는 것은 별도 계약입니다.

검증 계약은 [python_summary_contracts_test.go](volumes/python_summary_contracts_test.go)에 있으며 기존 public Gophercloud testhelper 기반 `pythonVolumeAPI`·응답 fault 도우미를 재사용합니다. 실제 클라우드 호출과 Python 인증 실행은 이 fixture 검증과 별개입니다.
