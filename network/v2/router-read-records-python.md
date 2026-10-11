# Router의 Python/Go Find·List 조회

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud는 `v2.15.0`입니다. Go leaf는 `network/v2/extensions/layer3/routers`이고 `network.Routers`에서 사용합니다. 이 문서는 두 Source 연산에 대응합니다.

| Python | Go |
| --- | --- |
| `find_router(name_or_id, ignore_missing=True, **query)` | `FindRecord(ctx, identity, options...)` |
| `routers(**query)` | `ListRecords(ctx, options...)` / `AllRecords(ctx, options...)` |

기존 `GetWithFields`, 생성·변경·삭제 및 Router 액션/인터페이스 API와 함께 사용합니다. `RouterReadRecord`는 이미 제공하는 `RouterActionRecord`의 alias이고 조회를 위해 별도 응답 구조를 구현할 필요가 없습니다.

아래 Python은 독립 실행 예제입니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--router", default="edge")
parser.add_argument("--mode", choices=("find", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        value = conn.network.find_router(args.router, ignore_missing=False)
        print(json.dumps(value.to_dict(), indent=2))
    else:
        for value in conn.network.routers(name=args.router, max_items=100):
            print(json.dumps(value.to_dict(), indent=2))
```

아래 Go는 독립 `main.go`입니다. [설치 안내](../../docs/install.md)에 따라 module 안에서 실행합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, mode, identity string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "find":
        value, err := network.Routers.FindRecord(
            ctx, identity, routers.WithRecordFindIgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.Routers.AllRecords(
            ctx, routers.WithRecordListQuery("name", identity),
            routers.WithRecordListMaxItems(100),
        )
        if err != nil { return nil, err }
        rows := make([]*resource.RawResource, 0, len(values))
        for _, value := range values { rows = append(rows, value.Resource) }
        return rows, nil
    default:
        return nil, fmt.Errorf("unknown mode %q", mode)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    identity := flag.String("router", "edge", "router name or literal ID")
    mode := flag.String("mode", "list", "find or list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *identity)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## Source view와 실제 응답

`Resource`는 Source `to_dict()`의 canonical Body/Header/default view입니다. `Wire`는 수신한 literal row, `Envelope`은 실제 member body 또는 해당 row의 whole page body이고 `Header`·`StatusCode`도 실제 관찰값입니다. 각각 독립 snapshot이므로 view를 수정해도 Wire/receipt가 바뀌지 않습니다. Header descriptor도 Source 기본 출력처럼 Resource의 JSON view에 포함합니다.

Router의 `revision_number`는 wire `revision`입니다. 상위 NetworkResource의 wire `revision_number` 선언을 Router가 MRO에서 덮습니다. canonical/wire response key가 모두 있으면 원문의 마지막 값이 적용됩니다. `project_id`가 없을 때만 `tenant_id` alias를 읽으며 explicit null은 유지합니다. `is_admin_state_up`, `is_distributed`, `is_ha`는 각각 wire `admin_state_up`, `distributed`, `ha`입니다.

빠진 Body 선언과 `if_match`는 null, 빠진 tags는 빈 목록입니다. Explicit null·false·zero와 unknown extension JSON을 보존합니다. `availability_zone_hints`, `availability_zones`, `routes`, `tags`의 list descriptor는 scalar를 한 원소 목록으로 만들고 dictionary descriptor `external_gateway_info`는 mapping이 아닌 값을 빈 객체로 바꿉니다. Bool은 Python truthiness를 따르므로 `"False"` 문자열은 true이며 `1e-400`은 binary64 underflow로 false입니다. Integer descriptor는 bool을 그대로 유지하고, numeric float를 binary64 후 절단하며, Unicode decimal-digit 문자열을 변환합니다. Signed/space-padded/일반 nonnumeric 문자열은 Source처럼 zero이고 `²`처럼 isdigit이지만 decimal이 아닌 문자열과 infinite float는 실제 응답 증거를 포함한 오류입니다. Unicode 데이터 기준은 공통 descriptor의 Unicode 16.0이며 Python의 설정 가능한 최대 integer-string 길이는 별도 런타임 차이입니다.

`created_at`와 `updated_at`은 고정 Source에서 untyped Body입니다. Go time.Time이나 특정 timestamp 포맷으로 강제 변환하지 않습니다. Connection computed `location`은 immutable record view에서 null이며 서버가 보낸 forged location은 Wire/Envelope에 남습니다. Python Connection의 mutable current_location·Resource/cache/Adapter 생명주기 전체는 별도 잔여 항목입니다.

Member HTTP `If-Match`는 Source Requests처럼 복수 값을 comma-space로 합친 후 list descriptor로 읽습니다. `If-Match: a, b`는 `["a, b"]`입니다. Constructor `if_match`의 canonical underscore 이름은 정확한 대소문자이고 wire `if-match`는 case-insensitive입니다. `IF_MATCH`는 unknown Body extension이며 Header로 바꾸지 않습니다. List는 row의 `if_match`/case-insensitive `if-match` 값을 소비하고 whole-page HTTP `If-Match`를 각 row의 descriptor로 적용하지 않습니다. Whole-page Header는 receipt에 남습니다.

## Find의 기본 동작과 query

Find는 이름/ID를 literal 한 URL segment로 escaping하여 member GET을 먼저 수행합니다. Slash/percent/space는 segment 안에 유지하며 빈 값·dot segment·control·invalid UTF-8은 사전 오류입니다. 원래 kwargs는 alias를 바꾸지 않고 GET query로 전달하고 동시에 Source constructor seed가 됩니다. None은 query에서 생략하지만 seed의 explicit null은 남습니다. Accepted bodyless/JSON parse 실패 member 응답도 seed/default view를 유지하며 Wire는 nil입니다. 유효 JSON의 잘못된 shape·UTF-8·descriptor 오류는 실제 응답을 포함한 실패입니다.

깨끗한 member 400/403/404에서만 같은 collection의 전체 LIST로 fallback합니다. 추가 `name` kwarg가 없으면 자동 `name=identity` hint를 넣고, explicit `name=None`이면 자동 hint를 넣지 않습니다. 완전한 검색의 exact ID 또는 name 한 건을 반환합니다. 후반 페이지 중복은 `resource.ErrAmbiguous`, 성공한 전체 검색에 결과가 없으면 기본 `nil, nil`이고 `WithRecordFindIgnoreMissing(false)`에서는 `resource.ErrNotFound`입니다. LIST forbidden·late failure·physical read/Close·Source 변경·취소는 absence로 숨기지 않습니다.

Find per-call `WithRecordFindMicroversion`은 member GET에만 적용합니다. Fallback의 모든 LIST 페이지는 캡처한 원래 selected/default version을 사용합니다. Source `all_projects`는 Find의 독립 제어 인수이므로 member query/seed에서 소비되고 이 Neutron List의 unknown 값으로 버려집니다. Router 권한을 승격하거나 추가 admin API를 호출하지 않습니다.

## List query·paging·옵션

정확한 server query는 `limit`, `marker`, `description`, `fields`, `flavor_id`, `id`, `name`, `status`, `project_id`, `sort_key`, `sort_dir`, 세 bool alias와 `tags`, `any_tags`, `not_tags`, `not_any_tags`입니다. Tag aliases는 `tags-any`, `not-tags`, `not-tags-any`로 전달합니다. Canonical bool/tag query가 wire alias보다 우선하고 fields 목록은 원래 문자열을 반복 query로 전달합니다. GetWithFields의 canonical-field 변환과 구별합니다. None은 전송하지 않고 query bool은 `True`/`False`로 표현합니다.

Server query로 선언되지 않은 canonical Body kwargs는 local filter입니다. 예를 들어 `revision_number`, `tenant_id`, 날짜·list·dict·EVPN/NDP 값이 해당합니다. Body wire 이름 `revision`은 local attr 이름이 아니므로 unknown List kwargs로 버립니다. Row/header 외의 unknown kwargs는 marshal/JSON 평가 전에 버립니다. Deprecated JMESPath·allow_unknown_params·constructor/session/route controls·`__conflicting_attrs` 복구는 이 JSON record mapping에서 전용 지원하지 않으며 사전 오류로 분리합니다.

Local dictionary filter는 nested subset, falsey 실제 값, truthy scalar에 대한 empty dictionary match와 nonempty mapping.get 실패를 Source처럼 처리합니다. Bool-number equality를 따릅니다. 일반 numeric equality는 공통 exact-decimal JSON 정책이므로 CPython float rounding/nonfinite/custom-object 전체 비교까지 동일하다고 판정하지 않습니다. Integer descriptor cast와 bool/numeric-next truthiness는 binary64를 따릅니다.

List는 lazy iterator이고 `AllRecords`는 late error 때 partial rows를 버립니다. 기본 full paging, `WithRecordListPaginated(false)`는 한 page, max_items=0은 무제한이며 positive cap은 local filter 전 raw row 수를 셉니다. Limit=0에 positive cap이 있으면 cap이 limit hint입니다. Explicit limit이 있으면 short page에서도 raw row ID marker로 계속하며 empty page에서 멈춥니다. 사용자 view/Wire 변경은 다음 marker에 영향을 주지 않습니다. Top-level/plural links와 HTTP Link를 읽되 같은 origin·collection·원래 filter 범위를 유지합니다. Scope/filter rewrite·cycle·truthy malformed next는 실제 whole-page proof를 포함한 오류입니다. Falsey numeric next는 Python binary64 underflow를 따릅니다. Dictionary continuation links와 정확한 `/v2.0/routers` alias는 Go 편의이며 다른 경로를 자유롭게 요청하지 않습니다.

`RecordFindOpts`/`RecordListOpts`와 `WithRecord...` 함수는 값·raw bytes·map·pointer·callback 결과를 snapshot합니다. 같은 옵션을 여러 API instance에서 동시에 재사용할 수 있습니다. 선택된 Source는 옵션 callback 전에 캡처하며 callback/read/Close/retry 동안 route/service/version 변경을 검출합니다. Public 옵션에는 외부 builder/decoder/pager 구현이 필요하지 않습니다.

List 기본 Accept는 application/json입니다. 안전한 Header와 List version 옵션은 concrete policy에 따라 처리하고 원래 selected client를 변경하지 않습니다. Explicit empty version은 version Header를 생략합니다. Retry에서 원래 없던 condition/version/auth/framing/body 또는 protected Header omission을 넣으면 첫 실제 receipt를 보존한 오류입니다. 깨끗한 HTTP retry와 live auth token 갱신은 허용합니다. Physical/Source/cancel 실패에서는 RetryFunc·ReauthFunc를 실행하지 않고 한 번의 실제 HTTP/Close 및 모든 원인을 보존합니다. Native 401 reauth wrapper의 public ErrOriginal/ErrReauth도 각각 receipt와 실패 원인을 유지합니다.

Completed HTTP 200..399를 읽고 모든 실제 400..599를 native status/body/Header 오류로 보존합니다. Source arbitrary Adapter의 final informational 100..199까지 완료 응답으로 지원하는 것은 이 Go profile 밖입니다.

## 권한·검증 범위

[Neutron 27.0.0 Router 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L177)은 collection/member `get_router`를 project ADMIN_OR_PROJECT_READER로 선언합니다. 따라서 이 두 연산은 core user 조회 범위입니다. `get_router:distributed`·`get_router:ha` attribute는 별도 ADMIN branch입니다. User 응답에서 숨겨진 attribute를 true/false로 만들어내지 않고 Source missing null을 표시합니다. 환경별 정책과 extension/backend 지원은 실제 deployment가 결정합니다.

테스트는 기존 Router native fixture, 공통 Neutron Find/List/REST/physical reader 및 public Gophercloud testhelper를 쓰는 internal/testcloud를 재사용합니다. Fixture 검증과 Source/stock policy 비교는 live 인증된 Neutron cloud 성공의 증거와 구별합니다.
