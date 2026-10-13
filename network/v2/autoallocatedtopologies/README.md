# Neutron auto allocated topology

`GetRecord`·`ValidateRecord`·`DeleteRecord`는 고정 [network proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)의 세 API에 대응합니다. project를 생략하면 Provider에 기록된 Keystone project를 사용합니다. `WithProjectID`는 literal ID를 고정합니다. Source 문서의 ID/name 설명과 달리 실제 코드에는 이름 lookup이 없으며 Go도 Keystone 이름 검색을 합성하지 않습니다.

| Python | Go | 요청 |
|---|---|---|
| `get_auto_allocated_topology(project)` | `GetRecord(ctx, WithProjectID(project))` | `GET /auto-allocated-topology/{project}` |
| `validate_auto_allocated_topology(project)` | `ValidateRecord(ctx, WithProjectID(project))` | 같은 경로, 고정 `fields=dry-run` |
| `delete_auto_allocated_topology(project)` | `DeleteRecord(ctx, WithProjectID(project))` | DELETE, 기본 strict404 |
| `delete_auto_allocated_topology(project, ignore_missing=True)` | `WithIgnoreMissing(true)` | clean actual404만 무시 |

```python
import openstack

conn = openstack.connect()
validation = conn.network.validate_auto_allocated_topology(project="project-id")
print(validation.id, validation.project_id, validation.dry_run)
# topology가 없으면 Neutron이 이 GET에서 리소스를 생성할 수 있습니다.
topology = conn.network.get_auto_allocated_topology(project="project-id")
print(topology.id)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := autoallocatedtopologies.New(service.RawClient())
    project := autoallocatedtopologies.WithProjectID("project-id")
    validation, err := api.ValidateRecord(ctx, project)
    if err != nil { log.Fatal(err) }
    fmt.Println(string(validation.Resource.Body["id"]), string(validation.Resource.Body["dry_run"]))
    topology, err := api.GetRecord(ctx, project)
    if err != nil { log.Fatal(err) }
    fmt.Println(topology.ProjectID, string(topology.Resource.Body["id"]))
}
```

기본 `ProjectID=nil`은 기록된 auth result를 요청 전에 한 번 해석하며 retry/reauth 때문에 route를 바꾸지 않습니다. project를 알 수 없는 manual/unscoped 인증은 ErrUnsupported이고 token·endpoint에서 추측하거나 추가 인증 조회를 하지 않습니다. 명시 ID는 UTF8/nonblank/control 검증과 URL segment escape를 적용합니다. concrete Options와 header/version pointer는 복사하며 callback 전후 sticky source guard를 적용합니다.

Get의 initial Resource.id는 요청 project입니다. 서버가 network ID를 반환하면 Resource.id는 그 값으로 바뀌지만 TopologyRecord.ProjectID는 요청 대상을 유지합니다. project_id의 Source alias는 tenant_id이며 명시 project_id:null은 fallback을 막습니다. dry_run은 untyped Body 필드이므로 null/string/list를 bool로 바꾸지 않습니다.

Validate는 URI project를 고정하고 Resource.id seed를 넣지 않습니다. bodyless GET의 `fields=dry-run`을 고정하고 별도 조회·생성을 하지 않습니다. [Neutron27 backend](https://github.com/openstack/neutron/blob/27.0.0/neutron/services/auto_allocate/db.py)는 검사 성공 시 id=`dry-run=pass`와 project/tenant ID를 반환합니다. dry_run이 없으면 Source view는 null이며 true를 합성하지 않습니다. 일반 Get은 topology가 없을 때 서버가 network/subnet/router를 provision할 수 있지만 SDK는 지정 GET만 제출합니다.

Resource는 Source projection, Wire는 실제 객체, Envelope/Header/StatusCode는 실제 응답이며 독립 복사입니다. Validate의 empty/non-JSON accepted 응답은 null id의 기본 Resource와 actual bytes를 유지하며 Wire를 만들지 않습니다. 잘못된 shape/decode는 actual ResponseError와 passive receipt를 반환하고 Resource 성공을 합성하지 않습니다. location은 공통 SDK null view이고 실제 확장은 Wire에 보존합니다.

Delete의 pinned `ignore_missing=False`를 유지합니다. true는 clean actual404만 무시하며 IgnoredMissing와 실제 receipt를 제공합니다. Read/Close/cancel/source fault가 섞인 404는 부재를 증명하지 않습니다. 성공 DELETE도 topology 객체·후속 GET을 만들지 않습니다. 공통 요청 계층은 actual status/body/header와 native401 ErrOriginal/ErrReauth를 보존합니다. physical fault의 HTTP1/Close1/hooks0과 clean retry/renewal를 구별합니다.

`WithHeaders`/`WithMicroversion`은 명시 Go transport 제어입니다. route·query·body·버전과 선택하지 않은 conditional/auth header의 부재를 retry에서 고정합니다. shared client/provider/token lock을 바꾸지 않습니다. Source proxy에 없는 list/find/wait·name lookup·response cache를 합성하지 않으며 optional Adapter cache/session/dirty lifecycle은 전체 SDK-R1/C1/S1에 남습니다.

[Neutron27 topology 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/auto_allocated_topology.py)은 GET/validate에 project ADMIN_OR_PROJECT_READER, DELETE에 ADMIN_OR_PROJECT_MEMBER를 적용합니다. backend는 nonadmin의 다른 tenant 접근도 거부합니다. extension·default external network·subnetpool·router 설정과 실제 override는 배포가 제공합니다. core user 범위이며 실클라우드 extension acceptance를 주장하지 않습니다.

[계약 테스트](records_test.go)는 public Gophercloud testhelper/internal/testcloud로 literal/current project, dry-run seed, alias/null, strict/ignored404, source guard, actual 오류 증거, 세 연산 native401 physical fault와 clean retry/renewal를 검증합니다. root가 Go/race·독립 main build 판정을 기록합니다.
