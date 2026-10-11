# Cinder volume summary 목록

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [volumes Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L894-L920)입니다. 목록은 상세 경로가 기본이고 `details=False`를 지정하면 `/volumes`를 사용합니다.

| Python | Go | 첫 GET 경로 |
|---|---|---|
| `conn.block_storage.volumes()` | `service.Volumes.List(ctx)` | `/volumes/detail` |
| `conn.block_storage.volumes(details=False)` | `service.Volumes.ListSummary(ctx)` | `/volumes` |
| `conn.block_storage.volumes(details=details)` | `service.Volumes.ListWithDetails(ctx, details)` | bool에 따른 경로 |
| `conn.block_storage.volumes(details=False, all_projects=True)` | `ListSummary(ctx, volumes.WithListQuery("all_tenants", "True"))` | `/volumes?all_tenants=True` |

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    for volume, err := range service.Volumes.ListSummaryWithControl(ctx,
        resource.ListControl{MaxItems: 100},
        volumes.WithListOptions(volumes.ListOpts{Limit: 20})) {
        if err != nil { log.Fatal(err) }
        fmt.Println(volume.ID, volume.Name)
    }
}
```

`Connect`는 OS_CLOUD/clouds.yaml 또는 OS_* 환경 인증을 사용합니다. 전체 프로젝트 조회는 서버 권한·정책에 따라 결정되므로 명시한 경우에만 all_tenants query를 넣습니다. Python의 대문자 True 문자열이 필요하면 위 raw query 옵션을 사용합니다. typed `ListOpts.AllTenants`의 기존 문자열 형식도 유지합니다.

summary 응답은 상세 정보가 적은 native `Volume` 모델로 반환됩니다. 모델에 없는 응답 필드는 저장하지 않으며 생략된 detail 필드는 zero value입니다. 기본 `List`와 `Resources.List`는 기존 detail 경로를 유지합니다. 이 목록은 집계용 `/volumes/summary` API와 다른 연산입니다.

`ListSummaryWithControl`은 기존 공통 native pager의 로컬 raw-row cap과 첫 페이지 정책을 재사용합니다. MaxItems0은 무제한, 음수는 lazy 순회 시 HTTP 전에 오류입니다. SinglePage는 다음 페이지를 요청하지 않으며, 소비자가 break해도 요청이 중단됩니다. 로컬 cap은 명시한 wire limit을 덮어쓰지 않습니다. Python의 max_items 초기 limit hint는 이 native profile에 자동 적용하지 않습니다. native extractor는 페이지 전체를 검사하므로 cap 뒤의 malformed 행도 오류가 됩니다.

pagination은 서버가 보낸 `volumes_links`를 native pager로 따라갑니다. 첫 query 옵션과 ResourceBase·인증을 유지하며, 다음 URL의 query는 native advertised URL 계약을 따릅니다. Python의 Body 로컬 필터·동적 Resource descriptor·dirty/cache/session 및 class별 microversion negotiation은 별도 SDK 범위입니다. 취소와 native HTTP/decode 오류는 ListSummary/volumes 문맥과 원래 cause를 보존하고, 뒤 페이지 실패 전에 반환한 행은 유지됩니다.

회귀는 [summary_test.go](volumes/summary_test.go)의 경로/옵션·cap/break·사전 검사/부분 오류3그룹과 기존 detail 기본값 그룹입니다. 기존 Gophercloud public testhelper 기반 Python volume fixture와 resource.StreamWithControl을 재사용합니다. 집중 `go test -race -count=1 -timeout 120s`가 PASS이며 실제 클라우드 호출은 실행하지 않았습니다.
