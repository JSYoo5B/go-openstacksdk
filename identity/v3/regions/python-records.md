# Keystone Region 목록: openstacksdk 비교

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 Proxy.regions, Region/Resource.list/Proxy._list이며 native 기반은 Gophercloud v2.15.0입니다. 새 `ListRecords/AllRecords/ProxyListRecords`만 다루고 기존 native List/All/Get/Find/Create/Update/Delete는 유지합니다.

[Keystone28.0.0 Region 정책](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/region.py)의 list_regions는 empty check string이며 system/domain/project scope를 선언합니다. Stock 정책은 valid token 조회를 허용하므로 project-scoped token도 이 조회 경로를 사용할 수 있습니다. 서버의 인증·운영 policy override 판단은 유지합니다. SDK가 권한이나 scope를 바꾸지 않습니다.

## 실행 가능한 Python

OS_CLOUD/clouds.yaml 또는 OpenStack 인증 환경을 설정하고 아래를 독립 프로그램으로 실행합니다.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    for row in conn.identity.regions(
        parent_region_id="Root", description="operations", limit=100,
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go

같은 인증 환경에서 아래를 Go module의 main.go로 저장합니다. SDK dependency를 추가한 뒤 실행합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Regions.ListRecords(ctx,
        regions.WithRecordListQuery("parent_region_id", "Root"),
        regions.WithRecordListQuery("description", "operations"),
        regions.WithRecordListLimit(100),
        regions.WithRecordListMaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s\n", data)
    }
}
```

## 선언과 query

Region은 Resource 직접 상속이며 별도의 Header descriptor가 없습니다. Body는 description/links/parent_region_id와 inherited id/name의 **nullable untyped 다섯 필드**입니다. 누락 null·명시 null/false/숫자/객체를 유지합니다. Server query는 parent_region_id와 inherited limit/marker이고 name은 server hint가 아닌 **local Body filter**입니다. Unknown kwargs는 marshaler 실행 전 버리고 unknown response는 Wire에만 보관합니다.

RegionRecord.Resource는 선언 view와 계산된 location, Wire는 원래 행, Envelope는 전체 page, Header/StatusCode는 actual receipt입니다. Clone은 모든 채널을 복제합니다. 응답 id/HREF와 caller-mutated view는 후속 URI나 marker를 바꾸지 않습니다. Default location은 null이며 WithRecordListLocation(CloudLocation)은 recorded explicit facts만 복제합니다. Region에는 project_id/zone Body descriptor가 없으므로 foreign unknown project/zone을 신뢰하지 않고 current project를 유지하며 location zone은 null입니다. 자동 Connection location/cache bridge는 구현했다고 주장하지 않습니다.

## 페이지·옵션·오류

ListRecords는 HTTP 없이 lazy iterator를 만들고 AllRecords는 실패 시 partial slice를 버립니다. Break는 다음 요청을 막습니다. Paginated=true, max_items=0, limit 생략이 기본입니다. 양수 explicit limit이 있으면 짧은 nonempty page도 마지막 raw wire ID로 marker를 보내 empty page에서 끝납니다. Local filter로 버린 마지막 행도 marker/raw max_items에 포함됩니다.

Source root links dictionary next는 무시하고 root links 키가 regions_links보다 우선합니다. Explicit links:null은 iterable 오류이며 actual page proof를 유지합니다. Rel/href 배열과 top-level next는 Source 우선순위입니다. RFC HTTP Link는 Source Requests next[uri]/next[url] 차이를 고치는 명시적 Go 호환 정책입니다. Origin·collection path·query/header/version·cycle은 SDK가 검증합니다.

Source는 limit=True/False를 정수로 변환하지 않고 wire query True/False로 보내며 truthy True일 때 marker fallback을 합니다. **Go paging은 비음수 decimal integer만 허용**하므로 raw boolean limit은 HTTP0 ErrInvalidOption입니다. Bool→1/0로 바꾸어 wire parity를 주장하지 않습니다. Negative/noninteger cap, malformed marker, unsafe route도 명시적인 Go 경계입니다.

Known Body filters는 ordered recursive subset/actual truthiness와 CPython loaded float rounding/underflow·bool/integer equality를 사용합니다. 첫 mismatch는 뒤 bad shape를 건너뛰며 reached error는 실제 receipt를 유지합니다. Headers/microversion/base_path/paginated/max_items/JMESPath와 __conflicting_attrs 한 번 복원은 concrete options와 공통 normalization이 처리합니다. Nil option/ctx·source 변경·custom cancel은 sticky이고 restoring callback도 이를 undo하지 않습니다. Region은 max microversion=None이라 기본 discovery가 없고 새 SourceStatusPolicy로 valid100..399를 받습니다.

Accepted Read/Close/decode/local 오류는 actual ResponseError, rejected 오류는 native actual proof와 원인을 유지합니다. Native *ErrUnableToReauthenticate는 Unwrap가 없어 ErrOriginal 값형 response error와 ErrReauth를 각각 검사합니다. Physical fault/cancel/source 변화 뒤 retry/backoff/reauth·추가 HTTP를 하지 않고 clean401/503/429/498은 live token과 captured route/query/header/version을 유지합니다.

## Proxy expression과 검증 범위

ProxyListRecords의 expression은 아직 읽지 않은 generator에 적용됩니다. Projection/type(@)는 HTTP0 null, length(@)는 HTTP0 type 오류, @/not_null/to_array는 generator 또는 nested generator를 유지합니다. 전체가 JSON일 때만 JSON을 채우고 typed ListRecords의 truthy string expression은 ProxyListRecords를 사용하도록 오류를 냅니다. Falsey/nonstring은 lazy generator입니다. Go iterator 재순회는 캡처된 요청을 다시 실행하며 Python one-shot 상태와 다릅니다.

기존 native Region/testcloud/Gophercloud public testhelper를 재사용한 새10그룹은 query/local/default/clone/location, links/null, short marker/raw cap/break, loaded filters, opaque/formal/options/guard/status, physical28 lanes와 clean retry4 lanes를 검증합니다. 고정 Source AST oracle은 actual 선언/route/pager를 실행하며 외부 session/resource collaborators는 stub합니다. Root Go/race/main gate 결과만 검증 완료로 기록하고 unrun PASS를 주장하지 않습니다. Concrete go_mapping 후보이며 supported 증가는 0, generic Resource/cache/Connection/session SDK-R1/C1/S1은 남습니다.

Root의 `/private/tmp/go-openstacksdk-cycle4-identity-repaired.jsonl`에서 Region과 User 전체 leaf package가 PASS했습니다. 첫 build는 기존 역방향 User→Group `ListGroupRecords`와 이름 충돌로 실패했고, 새 Group→User 메서드만 `ListUsersInGroupRecords/AllUsersInGroupRecords/ProxyUsersInGroupRecords`로 바꾼 뒤 재검증했습니다. Source 구현 commit은 `86ff98c2`·`74963af6`이며 root가 push를 확인했습니다. 이 package 검증은 실제 cloud 통합 또는 whole-SDK supported 완료가 아닙니다.
