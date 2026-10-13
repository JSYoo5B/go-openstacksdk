# Cloud QoS 정책 생성·수정: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud `v2.15.0`입니다. `conn.Network(ctx)`의 cloud facade가 kwargs 검증, 전체 extension 조회, 기본 정책 옵션 번역, 정책 조회, 변경 감지까지 처리합니다. 애플리케이션에 builder/interface 구현을 요구하지 않습니다. Native API와 기존 typed CRUD는 유지합니다.

| Python | Go |
| --- | --- |
| `conn.create_qos_policy(**kwargs)` | `service.CreateQoSPolicyRecord(ctx, options...)` |
| `conn.update_qos_policy(name_or_id, **kwargs)` | `service.UpdateQoSPolicyRecord(ctx, identity, options...)` |
| `name`, `description`, `shared`, `default`, `project_id` | 같은 cloud 키의 `WithQoSPolicyMutationJSON` 또는 아래 convenience options |
| kwargs 없는 create도 `POST {"policy":{}}` | 동일; 필수 name나 가짜 request default를 넣지 않음 |
| default 처리 뒤 kwargs 없는 update는 `None` | `result.NoOp == true`, `result.Policy == nil`; Find/PUT 없음 |
| 기존 raw 값과 같은 update는 기존 Resource | `result.NoOp == true`, `result.Policy != nil`; PUT 없음 |
| update 대상 없음은 `SDKException` | `errors.Is(err, resource.ErrNotFound)`; PUT 없음 |
| 반환 Resource | `result.Policy.Resource`의 descriptor view와 별도 `Wire/Envelope/Header/StatusCode` |

편의 옵션은 `WithQoSPolicyName(string)`, `WithQoSPolicyDescription(string)`, `WithQoSPolicyShared(bool)`, `WithQoSPolicyDefault(bool)`, `WithQoSPolicyProjectID(string)`입니다. `WithQoSPolicyAttribute(key, value)`와 `WithQoSPolicyMutationOpts`도 사용할 수 있습니다. `WithQoSPolicyMutationJSON`은 전체 kwargs 객체를 대체하고, 뒤에 오는 개별 옵션이 같은 키를 덮습니다. 옵션의 map/JSON bytes는 SDK가 복제합니다. 명시 null과 누락을 구분하려면 JSON 객체를 사용합니다. 예를 들어 `{}`와 `{"name":null}`은 다른 update입니다.

## 권한과 서버 필드 검증

[고정 Neutron 27 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L90-L134)의 create_policy/update_policy는 project scope에서 admin 또는 해당 project manager를 허용합니다. [manager 규칙](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L28-L46)은 `role:manager`와 project 소유 조건을 검사합니다. Deprecated admin-only 규칙이나 Python docstring의 admin-only 표기를 현재 기본 정책으로 바꾸어 말하지 않습니다. 배포 정책, driver와 실제 인증이 최종 수락을 결정합니다.

`project_id`를 전달할 수 있다는 SDK 계약과 서버의 소유자 변경 권한은 별개입니다. [고정 Neutron-lib 3.21.0 필드/권한 규칙](https://github.com/openstack/neutron-lib/blob/3.21.0/neutron_lib/api/attributes.py#L24-L56)은 다른 project 지정에 admin/service 역할을 요구하고 project_id/tenant_id를 맞춥니다. [QoS 필드 정의](https://github.com/openstack/neutron-lib/blob/3.21.0/neutron_lib/api/definitions/qos.py#L69-L76)의 owner 필드는 POST 허용·PUT 불허입니다. 이 버전 고정 자료는 필드 검토 근거이며 사용자의 배포 버전을 확인한 결과가 아닙니다. SDK는 Source처럼 허용 kwargs를 전송하고 서버 오류의 실제 증거를 보존합니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "update"), default="create")
parser.add_argument("--identity", default="")
parser.add_argument("--attrs", default="{}")
args = parser.parse_args()
if args.mode == "update" and not args.identity:
    parser.error("update identity is required")
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        value = conn.create_qos_policy(**attrs)
    else:
        value = conn.update_qos_policy(args.identity, **attrs)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

생성: `python main.py --cloud dev --attrs '{"name":"gold","shared":false,"default":false}'`.
수정: `python main.py --cloud dev --mode update --identity gold --attrs '{"description":"updated"}'`.
생성/수정은 실제 서버를 변경합니다.

## 독립 Go main

[설치 안내](../docs/install.md)에 따라 module을 준비한 뒤 아래를 main.go로 저장합니다.

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

func printJSON(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "create", "create or update")
	identity := flag.String("identity", "", "update: literal ID or exact name")
	attrs := flag.String("attrs", "{}", "JSON cloud mutation kwargs")
	flag.Parse()
	if *mode != "create" && *mode != "update" {
		log.Fatal("mode must be create or update")
	}
	if *mode == "update" && *identity == "" {
		log.Fatal("update identity is required")
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
	var result *network.QoSPolicyMutation
	option := network.WithQoSPolicyMutationJSON(json.RawMessage(*attrs))
	if *mode == "create" {
		result, err = service.CreateQoSPolicyRecord(ctx, option)
	} else {
		result, err = service.UpdateQoSPolicyRecord(ctx, *identity, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.Policy == nil {
		printJSON(nil)
		return
	}
	printJSON(result.Policy.Resource)
}
```

생성: `go run . -cloud dev -attrs '{"name":"gold","shared":false,"default":false}'`.
수정: `go run . -cloud dev -mode update -identity gold -attrs '{"description":"updated"}'`.

## 검증·확장 확인·기본 옵션

허용 cloud kwargs는 name/description/shared/default/project_id의 다섯 키입니다. `is_shared`, `is_default`, `policy_name`은 허용하지 않습니다. 잘못된 키나 JSON은 extension HTTP 전에 오류입니다. Python은 valid_kwargs decorator로, Go는 library-owned 옵션/JSON 검증으로 이 순서를 유지합니다.

각 호출은 cache 없이 extension 목록의 모든 페이지를 읽고 `qos` alias를 확인합니다. 발견 후 첫 page에서 중단하지 않으므로 후반 HTTP 오류·잘못된 alias를 숨기지 않습니다. 없는 extension은 `network.ErrUnavailableExtension`이며 policy HTTP를 수행하지 않습니다.

`default`는 꺼낸 뒤 **None/null이 아니면** 별도의 새 전체 extension 조회에서 `qos-default`를 확인합니다. false/0/빈 array/빈 object도 두 번째 조회를 수행합니다. 확장이 있으면 값을 그대로 `is_default`로 번역하고, 없으면 default를 제거합니다. 두 번째 gate의 HTTP·decode·physical 오류를 unsupported default로 숨기지 않습니다. Go carrier의 `DefaultDropped`는 nonnull default가 제거되었음을 표시하고, `Extensions`에는 두 gate에서 이미 관찰한 extension evidence가 남습니다.

Create는 번역 뒤 빈 attrs도 collection POST합니다. Update는 번역 뒤 빈 attrs이면 lookup을 하지 않고 nil Policy를 반환합니다. 나머지 update는 고정 기본 Find(ignore_missing=True)의 member GET→clean400/403/404 전체 list fallback을 사용합니다. 없으면 SDK 오류, 후반 중복이면 ErrAmbiguous이며 PUT하지 않습니다. PUT은 조회에서 **반환된 ID**를 사용합니다. Generic cloud get의 raw headers/fetch controls remaining을 이 두 method가 닫았다고 주장하지 않습니다. 두 method의 valid_kwargs는 그런 키를 받지 않습니다.

## 변경 감지와 실제 응답

변경 감지는 descriptor getter가 아닌 Source raw Body의 presence/값을 비교합니다. GET 응답에 name이 없을 때 `name:null`을 넣으면 dirty입니다. GET의 tenant_id alias로 project_id getter가 같은 값을 돌려주더라도 실제 project_id wire 키가 없으면 dirty입니다. GET의 shared:"false" getter는 true이지만 shared:true update는 raw 값이 달라 dirty입니다. shared:true와 shared:1은 Python bool/int equality로 같아 PUT하지 않습니다. JSON float는 CPython-loaded 비교를 사용하고 큰 integer는 그대로 보존합니다.

LIST로 찾은 policy의 constructor alias/wire 변환과 GET 응답 갱신을 구분해 private raw ledger를 보관합니다. 조회 record의 공개 Resource/Wire/Header/Envelope를 변경하거나 Clone을 바꾸어도 그 ledger를 바꿀 수 없습니다. 저수준 `policies.API.FindCloudMutationRecord`가 만든 trusted record/Clone만 같은 API의 `UpdateCloudRecord`에 빌려줄 수 있으며 다른 API나 변경된 selected source를 거부합니다. 일반 애플리케이션은 cloud method만 호출하면 됩니다.

PUT body는 실제로 달라진 키만 포함합니다. 응답이 일부 필드만 반환해도 Source처럼 seed의 나머지 필드를 Resource view에서 유지합니다. Wire/Envelope에는 실제 응답만 남으므로 seed를 서버가 반환한 필드처럼 표시하지 않습니다. Equal update는 원래 private GET/LIST receipt를 돌려주고 PUT receipt를 만들지 않습니다. 선택 microversion은 create/dirty update에서 유지하며 equal update는 Source `_update`처럼 None으로 재설정합니다. Unknown response microversion은 Wire에 그대로 남고 Resource view에서 이 선택 규칙을 따릅니다.

Rejected POST/PUT의 실제 status/body/header와 native error를 보존합니다. Native401은 Unwrap이 없는 ErrUnableToReauthenticate의 ErrOriginal과 value ErrUnexpectedResponseCode, ErrReauth 원인을 함께 보존합니다. Read/Close 실패·취소·source 변경은 성공/NoOp/missing으로 숨기지 않고 retry/reauth callback을 중단합니다. Clean retry와 live token renewal은 유지합니다. 원래 selected source와 headers는 사용자 option callback 전에 복제하므로 뒤 MoreHeaders 변경이 진행 중인 요청에 섞이지 않습니다.

## Go 차이와 검증 범위

Go의 결과 carrier는 Policy·Lookup·Extensions와 NoOp/DefaultDropped를 별도로 보존합니다. 오류가 없을 때 NoOp를 읽습니다. SDK의 selected service-relative 경로 정책은 ID를 한 segment로 escape하고 null/false/0/underflow/empty ID의 dirty update를 거부합니다. Source는 null에서 InvalidRequest이며 falsey nonnull ID의 URL join이 collection PUT 경로가 될 수 있습니다. Go가 그런 collection mutation을 재현하지 않는 경로 정책 차이를 명시합니다. Equal update는 ID를 먼저 요구하지 않습니다.

입력은 complete UTF-8 JSON과 concrete Go 옵션입니다. JSON object의 wire 키 순서는 의미를 보장하지 않습니다. Mutable Python Resource/cache/custom class/session/current_location 및 custom runtime 객체 계약은 SDK-R1/SDK-C1/SDK-S1의 공통 기반 과제로 별도 추적합니다. 이 문서의 두 named method는 그 범용 기반이나 다른 get/create/update helper의 완료 개수를 확대하지 않습니다.

실제 고정 Python cloud decorator·Neutron Proxy·Resource 코드를 Python 3.14.8에서 실행한 reference 26 cases와 Gophercloud public testhelper-backed internal/testcloud fixture를 비교했습니다. Source reference는 HTTP adapter 및 extension generator만 대체하며 live OpenStack/Keystone 권한 검증이 아닙니다. Root 집중 검증은 cloud 7그룹/leaf 3그룹, POST/PUT physical·cancel·source 24사례, clean retry/reauth/route/Source199 8사례를 통과했습니다. 정확한 독립 Go main build·Python AST와 최종204개 test package 전체 make check가 PASS입니다(`...cycle5-exact-guides/build-receipt.json`, `...cycle5-check-final.log`). 실제 backend 수락·배포 권한·전체 SDK 동등성은 별도 잔여입니다.

Source: [cloud create/update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1432-L1510), [Neutron Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5080-L5210), [QoSPolicy descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py), [Resource dirty/create/commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py).
