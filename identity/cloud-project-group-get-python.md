# Cloud Project·Group get: openstacksdk 비교

비교 대상은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [`get_project/get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py), 실제 list/search 함수 본문과 [`_utils._filter_list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)입니다. Native 기반은 Gophercloud v2.15.0입니다. 기존 [Project](v3/projects/python-record-reads.md)·[Group](v3/groups/python-records.md) record profile, cloud search와 native API는 유지합니다. 새 `ProjectReadCloud.GetProject`·`GroupReadCloud.GetGroup`만 이번 두 named 선언입니다. Connection의 `conn.GetProjectRecord`·`conn.GetGroupRecord`도 같은 owned getter를 사용하며 location이 명시되지 않으면 기록된 Connection facts를 기본값으로 채웁니다. 아래 Go 예제는 이 공개 bridge를 사용합니다.

## 실제 권한

공식 Keystone **28.0.0** [Project 정책](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py), [Group 정책](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/group.py), [base 규칙](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)을 실제 파일로 확인했습니다. Project member GET은 admin/system reader·대상 Project domain reader·자기 Project에 scope된 사용자 경로이며 Project 목록은 admin/system reader·대상 domain reader를 요구합니다. 따라서 own-project GET 권한이 name fallback이나 deprecated search의 목록 권한까지 보장하지 않습니다. Group GET/LIST는 admin/system reader 또는 대상 Group domain reader 경로입니다. 일반 project reader만으로 전체 목록을 볼 수 있다고 가정하지 않으며 token/scope나 서버 policy override를 SDK가 바꾸지 않습니다.

## 실행 가능한 Python

openstacksdk와 clouds.yaml의 읽기 권한을 가진 cloud를 준비합니다. 아래를 `get-identity.py`로 저장하고 `python get-identity.py --kind project --name PROJECT_UUID --cloud devstack`으로 실행합니다. Group 조회는 `--kind group`, domain 제한은 `--domain-id DOMAIN_UUID`입니다. `--filters '{}'`처럼 명시된 falsey 필터는 deprecated search 분기입니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--kind", choices=["project", "group"], default="project")
    parser.add_argument("--name", required=True)
    parser.add_argument("--domain-id", default="")
    parser.add_argument("--filters", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters) if args.filters else None
    domain = args.domain_id or None
    if args.kind == "project":
        value = conn.get_project(args.name, filters=filters, domain_id=domain)
    else:
        value = conn.get_group(args.name, filters=filters, domain_id=domain)
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go

동일한 인증 환경을 사용합니다. 아래를 Go module의 `main.go`로 저장하고 `go run . -kind project -name PROJECT_UUID -cloud devstack`으로 실행합니다. `-kind group`, `-domain-id DOMAIN_UUID`, `-filters '{}'`가 같은 분기를 선택합니다.

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
	"github.com/JSYoo5B/go-openstacksdk/identity"
)

func main() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	kind := flag.String("kind", "project", "project or group")
	name := flag.String("name", "", "literal name or ID")
	domain := flag.String("domain-id", "", "optional domain UUID")
	filters := flag.String("filters", "", "optional JSON filters")
	flag.Parse()
	if *name == "" {
		log.Fatal("-name is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	var value json.RawMessage
	switch *kind {
	case "project":
		var options []identity.ProjectGetOption
		if *domain != "" {
			options = append(options, identity.WithProjectGetDomainID(*domain))
		}
		if *filters != "" {
			options = append(options, identity.WithProjectGetFilters(json.RawMessage(*filters)))
		}
		result, err := conn.GetProjectRecord(ctx, *name, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	case "group":
		var options []identity.GroupGetOption
		if *domain != "" {
			options = append(options, identity.WithGroupGetDomainID(*domain))
		}
		if *filters != "" {
			options = append(options, identity.WithGroupGetFilters(json.RawMessage(*filters)))
		}
		result, err := conn.GetGroupRecord(ctx, *name, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	default:
		log.Fatal("-kind must be project or group")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## default/null closed lookup

Filters omission 또는 literal JSON null이면 `ResolveRecord`를 사용합니다. 이 새 leaf helper는 identity와 `domain_id`만 Source kwargs로 고정하고 Header·microversion·location은 명시적인 Go 옵션입니다. Request domain은 원문 JSON로 소유하며 omitted domain은 Source의 `domain_id=None`과 같습니다. null은 query에서 빠지지만 bool·number·empty string·반복 list 값은 공통 Requests 인코딩을 사용하며 member Resource domain seed에도 원래 값을 유지합니다. 응답에 실제 domain_id가 있으면 Source constructor처럼 응답 값이 우선합니다.

literal member GET에 clean400/403/404가 오면 원래 domain과 automatic name query를 가진 목록을 순회해 exact ID 또는 name을 비교합니다. glob은 default finder에서 적용하지 않습니다. missing은 nil Value/record, duplicate는 `resource.ErrAmbiguous`입니다. physical read/Close·Source/route/context/cancel 오류가 섞인 HTTP 실패는 fallback하지 않습니다. per-call version은 GET에만 적용하고 fallback LIST는 선택 client의 버전을 유지하는 기존 Source find 정책입니다. location 기본은 null이고 명시된 CloudLocation은 독립 복제한 SDK facts를 사용합니다. Project가 보존하는 unknown project_id/availability_zone은 기존 Project location 정책을 따르고 Group은 선언되지 않은 응답 값으로 current scope를 바꾸지 않습니다.

full Proxy `find_project/find_group`는 여전히 named **unresolved**입니다. 임의 raw Resource.fetch controls·header kwargs·synchronized Resource/cache 동작을 새 closed helper의 concrete 옵션으로 대체한다고 주장하지 않습니다. 이 두 cloud 메서드는 domain_id만 넘기므로 그 일반 kwargs 경로에 도달하지 않습니다. 기존 named Find·mutation 판정과 **SDK-R1/C1/S1** 공통 잔여는 유지합니다.

## filters presence와 실제 search 분기

None 이외의 filters는 `{}`, [], false, 0, underflow float, 빈 문자열도 `DeprecatedFilters=true`로 search를 선택합니다. default의 exact finder와 달리 이 branch는 각 row의 ID/name exact 또는 fnmatch glob을 사용하고 duplicate를 제거하지 않습니다. Python warning category 대신 Go result marker가 해당 분기를 표시합니다. 실제 rows는 `Inventory`, ordinary single match는 `Project`/`Group`, 임의 local 결과는 `Value`입니다. expression output에 HTTP metadata를 새로 만들지 않으며 selected record와 Inventory의 bytes/view/header는 독립 복제합니다.

Project의 실제 `search_projects`는 먼저 `list_projects`를 실행합니다. falsey filters는 empty mapping으로, truthy filters는 `dict(**filters)`로 확장하므로 문서에 JMESPath가 언급되어 있어도 truthy string·bool·number·nonempty array는 HTTP 전에 오류입니다. dict kwargs의 server/local 분류·aliases·formal Proxy controls·unknown discard는 기존 Project 목록이 처리합니다. truthy domain 인자는 server `domain_id`를 덮어쓰지만 `_filter_list`의 original filters는 바꾸지 않습니다. falsey domain은 원래 filter의 domain_id를 덮어쓰지 않습니다.

Project kwargs가 `jmespath_filters`로 literal list를 반환하면 실제 Source `_dict_filter`의 actual truthiness가 먼저입니다. identifier 없는 nonempty dict filter에서 falsey scalar row는 shape 오류 전에 제외됩니다. Go는 private `{value: actual}`/`{value: filter}` wrapper로 공통 ordered mapping/loaded equality를 재사용하고 원래 row indices/JSON만 반환합니다. identifier가 있으면 scalar의 e.get 오류를 보존합니다. 이 wrapper는 Resource나 HTTP receipt가 아닙니다. opaque Python object는 JSON bounded SDK-R1 차이로 오류를 보존하고 eager HTTP/가짜 record로 바꾸지 않습니다.

Group search는 domain null까지 list_groups에 그대로 전달하고 나머지 filters는 모든 목록 페이지 수집 뒤 `_filter_list`에 적용합니다. ordered nested dict·Python-loaded bool/number/float equality와 reached unknown-key 오류, materialized-list JMESPath를 공유합니다. invalid local filter보다 늦은 HTTP 오류가 먼저이며 실제 이전 Inventory를 유지합니다. Get의 최종 선택은 loaded truthiness→len→integer index0입니다. `[false]`, `[0]`, `[""]`는 present 값, `[null]`는 missing이고 한 문자 Unicode string도 반환할 수 있습니다. 길이 2 이상은 ambiguity, truthy scalar len과 dict integer-key lookup은 shape 오류입니다.

## 소유권과 검증 경계

기존 cloud capture·`rest.OperationGuard`·Source descriptor/find/list·`SelectLoaded/First`와 public Gophercloud testhelper/native fixture를 재사용합니다. caller options 전에 API/client/provider/route/version/header를 캡처하고 sticky outer guard와 custom cancel cause를 유지합니다. restored callback으로 이미 관찰한 source/cancel 실패를 제거하지 못하며 fault 이후 retry/auth/backoff를 실행하지 않습니다. 실제 rejected401의 Unwrap 없는 pointer wrapper는 ErrOriginal의 native 값형 response와 ErrReauth cause를 별도로 보존합니다. 정상 native retry는 기존 transport와 live token 정책을 따릅니다.

새 코드의 계약 함수는 21개(cloud Project7·Group6, closed leaf 각4)이며 domain seed·None/presentfalsey·kwargs/literal/JMES·version phase·late failure·source/options·physical actual receipt를 검사합니다. 최종 identity cloud race와 closed leaf race가 PASS입니다(`/private/tmp/go-openstacksdk-cycle5-identity-final.jsonl` 및 `...cycle5-project-group-leafs-first.jsonl`). 최초 Group raw domain query가 byte slice로 base64 인코딩되던 오류를 json.RawMessage carrier 한 줄로 수리하고 기대값을 보존했습니다. 독립 Python main은 AST로, pinned cloud 함수 본문은 CPython3.14의 제한된 oracle로 확인했습니다. Oracle의 proxy/Resource rows·version/logger/warning·JMES literal dispatch는 stub이며 전체 Python SDK/JMES runtime/auth/live cloud 검증이 아닙니다. 정확한 가이드의 독립 Go main build와 Python AST가 PASS이며(`...cycle5-exact-guides/build-receipt.json`), 중앙 두 연산 판정을 반영했습니다. 최종204개 test package 전체 make check도 PASS입니다. 이 두 named 구현을 전체 SDK parity로 확대하지 않으며 supported 증가는 0입니다.
