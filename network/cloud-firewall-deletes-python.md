# FWaaS 삭제: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `conn.Network(ctx)`로 얻은 `network.Service`가 strict 조회와 삭제, bool 결과, JSON 필터와 실제 응답 증거를 소유합니다. 애플리케이션이 lookup/delete를 연결하거나 builder/interface를 구현할 필요가 없습니다.

| Python | Go |
| --- | --- |
| `delete_firewall_rule(name_or_id, filters=None)` | `DeleteFirewallRuleRecord(ctx, identity, options...)` |
| `delete_firewall_policy(name_or_id, filters=None)` | `DeleteFirewallPolicyRecord(ctx, identity, options...)` |
| `delete_firewall_group(name_or_id, filters=None)` | `DeleteFirewallGroupRecord(ctx, identity, options...)` |
| 반환 bool | `result.Deleted` |
| 조회/삭제 실제 증거 | `result.Lookup`, `result.Delete` |

## 순서와 누락 처리

각 호출은 `find_firewall_*(ignore_missing=False, **filters)` 뒤 찾은 Resource의 **실제 반환 ID**로 `delete_firewall_*(ignore_missing=False)`를 수행합니다. FWaaS extension/has_service gate나 cache를 추가하지 않습니다. 초기 GET의 깨끗한400/403/404만 전체 list fallback을 허용하며, original ID 또는 exact case-sensitive name으로 끝까지 검색하여 후반 duplicate/실패도 관측합니다. Unknown kwargs는 초기 GET query에 남지만 fallback에서는 클래스에 선언된 query/Body만 분류합니다. Name filter는 자동 name hint를 덮지만, SDK는 여전히 원래 identity로 결과를 찾습니다.

Find의 empty 결과 또는 깨끗한 LIST404는 `Deleted=false`, DELETE 없음입니다. 찾은 뒤 **깨끗한 DELETE404도 false**입니다. 두 단계 모두 strict라 QoS의 default ignore-missing DELETE404→true와 다릅니다. Source에서 NotFound를 잡아 False로 만드는 범위를 보존합니다. 정상 DELETE는 본문이 없거나 JSON이 아니어도 status100..399면 true이며 409/503 등은 원래 오류입니다.

깨끗한 부재는 실제 missing evidence만 있는 경우입니다. 404에 Read/Close/cancel/source/callback 실패가 합쳐지거나 callback이 원래503 대신404를 만들어 반환하면 오류를 유지합니다. 물리 실패를 false/성공으로 숨기지 않고 retry/reauth callback도 실행하지 않습니다. Native401의 pointer `ErrUnableToReauthenticate`는 Unwrap을 제공하지 않으므로 `ErrOriginal`의 실제 value-form rejection과 `ErrReauth`의 처리 원인을 별도로 보존·검증합니다.

## 옵션·Source view·증거

`WithFirewallDeleteOpts`, `WithFirewallDeleteFilters(json.RawMessage)`, `WithFirewallDeleteHeaders(map[string]string)`, `WithFirewallDeleteMicroversion(string)`, `WithFirewallDeleteCurrentLocation(resource.CloudLocation)`를 제공합니다. Raw JSON/map/pointer를 option 생성과 적용 시 복제합니다. Bulk 뒤 개별 옵션은 해당 값을 대체합니다. 필터 기본값은 빈 object이고 falsey JSON null/false/zero/empty string·array·object는 {}처럼 처리합니다. Truthy filters는 Source의 **mapping expansion**에 맞춰 object만 허용합니다. Docstring에 JMES가 적혀 있어도 실제 함수는 truthy expression string을 받지 않습니다. Bound name_or_id/ignore_missing 충돌은 null이어도 HTTP 전에 오류입니다.

Headers/Microversion/CurrentLocation는 Go의 구체적인 transport/context extras입니다. Headers는 조회·fallback·삭제에 적용합니다. Explicit Microversion은 초기 GET과 DELETE에 적용하고, SourceFindPolicy fallback LIST는 captured adapter default를 사용합니다. 원래 client facts는 바꾸지 않습니다. Source raw filters의 headers/microversion 의미와 동일하다고 주장하지 않습니다. 그 차이는 아래 named remaining입니다.

FWaaS 세 클래스는 plain Resource를 상속하므로 NetworkResource의 tags/Header if_match defaults를 넣지 않습니다. 대부분 Body는 **untyped**입니다. `shared="false"`, `enabled=0`, `ip_version="4"`를 bool/int로 강제 변환하지 않습니다. Group.ports와 Policy.firewall_rules만 list descriptor로 scalar를 감쌉니다. Unknown Body와 큰 integer 원문을 보존합니다. Rule.summary는 Computed(default="")입니다. Pinned _collect_attrs/_compute_attributes 경로에 따라 getter view는 default를 읽고 server의 forged summary는 실제 Wire에만 남습니다.

Lookup.Resource는 Source getter view, Wire는 실제 row, Envelope는 실제 전체 fetch/page bytes, Header/StatusCode는 실제 receipt입니다. Delete는 bodyless mutation의 metadata-only receipt이며 Resource/Wire를 만들지 않습니다. Physical accepted response 오류에서도 status/body/header만 보존하고 성공 Resource를 만들지 않습니다. Clean DELETE404는 nil error와 false를 반환하더라도 실제 Delete404 증거가 남습니다. 전체 검색이 아무 결과 없이 끝난 경우 Lookup은 nil일 수 있습니다.

`CurrentLocation()` 결과를 전달하면 configured project/location을 복제하며 falsey/matching project는 현재 이름/domain을 유지하고 truthy foreign project는 raw ID만 유지합니다. Unknown availability_zone도 location.zone에 사용합니다. 서버 location은 Wire에 보존하고 view는 연결 사실을 읽습니다. 선택 client/API/endpoint/base/version drift와 custom cancellation cause는 각 callback/physical boundary에서 확인합니다. 조회가204 empty 또는200 malformed여도 Source처럼 seeded 요청 ID를 유지하여 그 ID를 DELETE할 수 있습니다.

## 남은 named 동작

세 named API의 판정은 **unresolved**입니다. 유용한 삭제 workflow를 구현했지만 전체 Source kwargs/Resource SDK parity를 완료로 세지 않습니다.

1. Raw Find filters의 headers/microversion 및 일부 fetch/constructor controls가 common Engine에서 아직 분리되지 않습니다. 실제 Source raw headers는 초기 GET query이고 fallback LIST에서는 formal header입니다. Raw microversion은 초기 GET에만 적용되고 fallback과 DELETE는 adapter default로 돌아갑니다. 현재 Filters에 이 키들을 넣으면 HTTP 전 오류이고 Go transport extra는 별도 동작입니다. Source의 base_path/skip_cache/resource_response_key 등 runtime/fetch controls도 일반 query로 모두 재현하지 않습니다.
2. Common fallback pager는 strict nonnegative Go integer cap/limit와 안전한 marker를 요구합니다. Source가 받는 bool/음수/실수 cap/limit, error phase와 exact cap 다음 continuation GET 동작을 전부 재현하지 않습니다. Source가 다음 페이지의 duplicate/late failure를 관측하는 시점과 DELETE 시작 시점이 달라질 수 있습니다.
3. 반환 ID가 false/0/empty string·array·object면 Source utils.urljoin은 collection DELETE 경로를 만들 수 있습니다. 현재 SDK는 실제 member ID를 요구하여 HTTP 전 오류입니다. 명시 null ID 또는 request seed가 없는 fallback LIST row의 missing ID는 Source InvalidRequest입니다. 반대로 초기 GET 응답에서 id member가 누락되면 요청 ID seed를 유지하여 삭제합니다. 이 구분과 falsey-nonnull collection 동작을 generic Resource 차이로 숨기지 않습니다.

Go는 literal ID를 한 path segment로 escape하고 origin/protocol/header/continuation guard를 유지합니다. Source arbitrary URI percent-format, adapter/class injection, mutable Resource/cache/lifecycle와 일반 JSON 밖 runtime 객체는 별도 SDK 범위입니다. 기본 Source view/actual proof가 있다고 해당 범위를 지원으로 승격하지 않습니다.

## 권한과 배포

[FWaaS 2025.2 공식 sample policy](https://docs.openstack.org/neutron-fwaas/2025.2/configuration/policy-sample.html)는 일반 삭제를 project member 또는 admin으로 제한하고, shared 객체 삭제에는 별도 admin rule을 둡니다. Shared 객체의 read 권한은 delete 권한을 보장하지 않습니다. FWaaS plug-in과 실제 배포 policy가 필요하며 SDK는 권한을 높이거나 rule/policy/group 연관을 자동 해제하지 않습니다. 연결된 객체 삭제가409이면 실제 오류를 반환합니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=("rule", "policy", "group"), default="rule")
parser.add_argument("--name", required=True)
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "rule":
        result = conn.delete_firewall_rule(args.name, filters=filters)
    elif args.kind == "policy":
        result = conn.delete_firewall_policy(args.name, filters=filters)
    else:
        result = conn.delete_firewall_group(args.name, filters=filters)
    print(json.dumps(result))
```

## 독립 Go main

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
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kind := flag.String("kind", "rule", "rule, policy or group")
	name := flag.String("name", "", "literal resource ID or exact name to delete")
	filters := flag.String("filters", "{}", "JSON Source Find kwargs")
	flag.Parse()
	if *name == "" {
		log.Fatal("name is required")
	}
	if *kind != "rule" && *kind != "policy" && *kind != "group" {
		log.Fatal("unknown kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	opts := []network.FirewallDeleteOption{network.WithFirewallDeleteFilters(json.RawMessage(*filters)), network.WithFirewallDeleteCurrentLocation(location)}
	var result *network.FirewallDeletion
	switch *kind {
	case "rule":
		result, err = service.DeleteFirewallRuleRecord(ctx, *name, opts...)
	case "policy":
		result, err = service.DeleteFirewallPolicyRecord(ctx, *name, opts...)
	case "group":
		result, err = service.DeleteFirewallGroupRecord(ctx, *name, opts...)
	}
	if err != nil {
		log.Fatal(err)
	}
	value, err := json.Marshal(result.Deleted)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(value))
}
```

각 main을 firewall-delete.py/main.go에 저장한 뒤 cloud와 삭제 대상을 지정합니다. 예: `python3 firewall-delete.py --cloud dev --kind rule --name obsolete-rule --filters '{"project_id":"owner"}'`, `go run ./main.go -cloud dev -kind rule -name obsolete-rule -filters '{"project_id":"owner"}'`. 설정은 clouds.yaml에서 읽으며 실행하면 지정한 객체를 삭제합니다. RawMessage로 필터의 integer 원문을 보존합니다.

## 검증 근거

소유 새 테스트는 기존 internal/testcloud/public Gophercloud testhelper와 기존 물리 Read/Close fixture를 재사용합니다. Top-level6 groups가 세 family의 요청 순서·actual-ID 삭제·missing/list404/delete404 false·late duplicate/HTTP 실패·falsey mapping/입력 복제·untyped/list/Computed view·204/malformed seed를 검사하도록 준비됐습니다. Physical matrix는 세 family×GET/DELETE×200/401/404/503×Read+Close/cancel/source72 lanes이며 fabricated callback404도 검증합니다. Go 실행과 최종 PASS 판정은 root 통합 검증이 담당합니다.

CPython3.14 actual pinned Source oracle은 **156 cases, family당52**입니다. 실제 cloud→Proxy→Resource find/fetch/list/delete를 import하고 physical Adapter.get/delete와 connection fixture만 바꿨습니다. Code/file hashes와 실행 경계를 별도 receipt로 남겼으며 새 테스트를 Source 실행이나 live cloud 성공으로 바꾸어 표시하지 않습니다. Python main AST, byte-identical guide mains, Go fixture/race/build와 실제 배포 RBAC/backend는 구분합니다.
