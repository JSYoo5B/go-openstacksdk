# Nova action history의 Source record 목록

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [server_actions](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3221)와 [ServerAction·ServerActionEvent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_action.py)를 concrete Go 옵션으로 제공합니다. 기존 Gophercloud v2.15.0 `List`·`Get`·`InServer`의 signature와 native timestamp decoder를 유지하며, Source의 untyped 값·이벤트 Resource descriptor는 owned record에서 처리합니다.

| Python | Go |
| --- | --- |
| `conn.compute.server_actions(server, **query)` | `compute.InstanceActions.ProxyListRecords(ctx, serverID, options...)` |
| 일반 Resource generator 소비 | `compute.InstanceActions.ListRecords(ctx, serverID, options...)` |
| 전체 목록 수집 | `compute.InstanceActions.AllRecords(ctx, serverID, options...)` |
| 명시적 Go name/ID resolver 재사용 | `InServer(ctx, resource.Name(name))` 뒤 `scope.ListRecords(ctx, options...)` |

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--server", required=True)
parser.add_argument("--changes-since", default=None)
args = parser.parse_args()
query = {} if args.changes_since is None else {"changes_since": args.changes_since}
with openstack.connect(cloud=args.cloud) as conn:
    for action in conn.compute.server_actions(args.server, **query):
        print(json.dumps(action.to_dict(), indent=2))
```

독립 `main.go`이며 [설치 안내](../../../docs/install.md)를 따릅니다. 생성·삭제 작업을 수행하지 않습니다.

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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions"
)

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "", "literal server UUID")
    since := flag.String("changes-since", "", "optional ISO timestamp")
    flag.Parse()
    if *server == "" { log.Fatal("-server is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    var options []instanceactions.RecordListOption
    if *since != "" {
        options = append(options, instanceactions.WithRecordListQuery("changes_since", *since))
    }
    for value, err := range compute.InstanceActions.ListRecords(ctx, *server, options...) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

`RecordListOpts`와 `WithRecordListQuery`는 known query·Body attrs와 Source formal controls를 소유합니다. Query `limit`, `marker`, `changes_since`/`changes-since`, `changes_before`/`changes-before`를 서버에 전달합니다. 다른 declared Body 속성은 로컬 필터이고 unknown kwargs는 JSON marshaler를 실행하지 않고 버립니다. Header·microversion·service-relative base path는 전용 옵션을 사용합니다. URI parent는 server argument가 소유하므로 직접 server_id 중복이나 route/session constructor 교체는 오류로 처리합니다. literal ID는 이름으로 해석하지 않으며 `InServer` resolver 사용은 별도의 Go 편의 기능입니다.

`WithRecordListMaxItems`는 local filter 전에 raw row 수를 제한하며 falsey wire limit이면 Source의 limit hint를 보냅니다. `WithRecordListPaginated(false)`·consumer break·cap은 사용하지 않는 다음 링크를 읽지 않습니다. 공통 pager가 Source link 우선순위, limit/marker fallback과 falsey next를 처리하며 origin·route·cursor guard를 유지합니다. Source가 허용하는 임의 origin/session 이동까지 허용한 것은 아닙니다.

선택된 microversion이 있으면 그대로 사용합니다. 없으면 실제 generator를 소비할 때 공통 Nova discovery로 server advertised 최대와 Source ceiling **2.84** 중 작은 값을 고릅니다. 각 페이지의 version과 fixed parent는 동일하며 shared client를 수정하지 않습니다. 명시된 버전은 올리거나 자르지 않습니다. Discovery는 원래 page receipt와 분리해 반환합니다. Go `ListRecords`는 lazy이고 `ProxyListRecords`의 JMESPath 평가만으로 discovery·list HTTP를 실행하지 않습니다.

Source `Proxy._list`는 collected 배열에 JMESPath를 실행하지 않고 실제 generator 객체를 대상으로 삼습니다. `ProxyListRecords`의 `Value`는 opaque `RecordListGenerator`, arbitrary JSON, 또는 그 generator가 포함된 값입니다. `[].action`/`[0]`/`type(@)`가 null이거나 `length(@)`가 실패해도 HTTP는 없습니다. `@`는 같은 generator, `@ == @`와 nested identity는 같은 object identity를 유지합니다. generator JSON marshal·`to_string(@)`는 소비 없이 실패합니다. `JSON`은 완전히 직렬화할 수 있는 결과에만 존재합니다. 실제 Record가 필요하면 `Value.(*RecordListGenerator).Records`를 소비합니다. 일반 `ListRecords`는 arbitrary JMESPath 결과를 받을 수 없으므로 truthy expression이면 `ProxyListRecords` 사용을 요구합니다.

`ServerActionRecord.Resource`는 아홉 Source Body 필드와 location view이고 `Wire`는 실제 원문 행입니다. `Envelope`는 actual 전체 페이지, `Header`·`StatusCode`·`ServerID`는 실제 response와 URI 정보입니다. Source가 모델링하지 않는 `instance_uuid`, `updated_at`, unknown 값은 Resource에 넣지 않고 Wire에 보존합니다. 날짜는 Source untyped descriptor대로 literal 값을 유지합니다. request_id는 alternate ID이고 actual primary id의 존재가 null/false/빈 값까지 우선합니다.

`events`의 null·missing은 null, scalar는 한 개 리스트로 감싸며 Source의 `list_type=ServerActionEvent`를 적용합니다. object element는 열 개 Body 기본값과 hostId→host_id 별칭을 가진 nested Resource view이고 nonobject element는 plain `{}`입니다. location은 현재 직접 ServiceClient가 제공하지 않는 SDK-owned null이며 arbitrary Connection/current_location은 공통 SDK-R1 범위입니다. Wire의 nested unknown·original aliases는 그대로 남습니다. Source Resource 객체는 plain dict와 같지 않으므로 JSON array filter의 full dict를 nested Resource와 잘못 같다고 보지 않습니다. plain `{}`·empty list·null 비교는 기존 Python-loaded equality를 사용합니다. Go JSON filters는 Python Resource 객체 operand를 생성하지 않습니다.

확인한 공식 Nova **31.0.0** [instance action policy](https://github.com/openstack/nova/blob/31.0.0/nova/policies/instance_actions.py), [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/instance_actions.py), [schema](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/schemas/instance_actions.py)에서 list는 instance project의 reader-or-admin 경로입니다. limit/marker/changes-since는 2.58부터, changes-before는 2.66부터입니다. 기본 목록은 action summary이고 events를 제공하지 않습니다. Detail Get의 events는 2.51부터 reader/owner에게 보이지만 traceback·host는 별도 admin policy, hostId는 2.62부터, details는 2.84와 별도 events:details policy로 제어합니다. 이 목록 API가 admin 정보를 추가 조회하거나 생성하지 않습니다. 서버가 반환한 nested 값은 보존하며 배포된 policy override는 별개입니다.

공통 `cloudread`, Source descriptor Engine, microversions, REST paging, JSON equality, opaque JMESPath와 public Gophercloud testhelper를 재사용합니다. Physical read/Close/source/cancel 오류는 actual response 및 모든 cause를 유지하며 retry/auth/backoff나 다음 페이지로 숨기지 않습니다. clean401/503는 고정 method·URI·query·header·version으로 native retry를 유지합니다. Gophercloud v2.15에서 401은 ReauthFunc, 503은 RetryFunc를 사용하며 RetryBackoffFunc는 429/498 경로에만 적용됩니다. optional HTTP cache, Python Resource mutation/Connection lifecycle와 실제 cloud acceptance는 별도 공통 계약이며 이 두 named read의 구현을 전체 SDK parity로 확대하지 않습니다.
