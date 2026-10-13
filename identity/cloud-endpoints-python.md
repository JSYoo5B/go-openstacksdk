# Cloud endpoint search/get: openstacksdk 비교

고정 비교 대상은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [`IdentityCloudMixin.search_endpoints/get_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py), [`_utils._filter_list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py), `Endpoint`·`Resource.find`와 Gophercloud v2.15.0입니다. `NewEndpointReadCloud(identityV3.Endpoints)`는 이미 선택된 v3 API에 연결하고 constructor에서 HTTP를 실행하지 않습니다. 기존 native Endpoint API와 [Source 목록](v3/endpoints/python-records.md)을 유지합니다. 이 구현 범위는 cloud search/get 두 named operation입니다.

실제 파일로 확인한 Keystone **27.0.0** [endpoint 정책](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/endpoint.py)과 [base 규칙](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/base.py)에서 list/get은 `admin_required` 또는 `role:reader and system_scope:all`입니다. 일반 project reader 허용을 뜻하지 않습니다. 다음 `system-reader` clouds.yaml 항목에는 system scope reader 인증을 준비합니다. 배포 policy override와 실제 cloud acceptance는 서버가 결정합니다.

## 실행 가능한 Python 예제

openstacksdk와 clouds.yaml을 준비하고 `python endpoint-cloud.py --cloud system-reader --id 'public-*'`로 실행합니다. `--get --id ENDPOINT_UUID`는 default member/find, `--get --id 'public-*' --filters '{}'`는 deprecated search 분기를 실행합니다. `--filters`는 JSON dict 또는 JSON으로 인코딩한 JMESPath 문자열입니다.

```python
import argparse
import json
import openstack


def jsonable(value):
    if hasattr(value, "to_dict"):
        return value.to_dict()
    if isinstance(value, list):
        return [jsonable(item) for item in value]
    if isinstance(value, dict):
        return {key: jsonable(item) for key, item in value.items()}
    return value


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="system-reader")
    parser.add_argument("--id", default="")
    parser.add_argument("--filters", default="")
    parser.add_argument("--get", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters) if args.filters else None
    if args.get:
        value = conn.get_endpoint(args.id, filters=filters)
    else:
        value = conn.search_endpoints(id=args.id, filters=filters)
    print(json.dumps(jsonable(value), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go 예제

동일한 clouds.yaml을 사용합니다. 아래를 Go module의 `main.go`로 저장하고 `go run . -cloud system-reader -id 'public-*'`로 실행합니다. `-get -id ENDPOINT_UUID`와 `-get -id 'public-*' -filters '{}'`가 같은 분기를 선택합니다.

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
	cloud := flag.String("cloud", "system-reader", "clouds.yaml cloud")
	id := flag.String("id", "", "literal ID or search glob")
	filters := flag.String("filters", "", "JSON dict or JSON encoded JMESPath string")
	get := flag.Bool("get", false, "get one endpoint")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	api, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	reader := identity.NewEndpointReadCloud(api.Endpoints)
	var options []identity.EndpointReadOption
	if *filters != "" {
		options = append(options, identity.WithEndpointReadFilters(json.RawMessage(*filters)))
	}
	var value json.RawMessage
	if *get {
		result, err := reader.GetEndpoint(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	} else {
		result, err := reader.SearchEndpoints(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## 선택·응답 계약

`SearchEndpoints(ctx, id, ...)`는 Source처럼 query hint 없이 전체 Endpoint 목록을 먼저 수집합니다. `id`가 truthy이면 각 row의 id 또는 name에 exact match를 먼저 시도하고 다른 row에는 fnmatch glob을 적용합니다. duplicate를 제거하거나 exact match 때문에 나머지 glob match를 버리지 않습니다. 빈 identifier는 전체 목록을 대상으로 합니다. `filters`는 이후 ordered nested dict subset 또는 materialized-list JMESPath로 적용합니다. Source cloud 함수는 추가 server kwargs를 받지 않으므로 `interface`, `region_id`, `service_id`도 Filters dict에서 로컬로 적용하며 HTTP query에 전달하지 않습니다.

falsey filters(null·false·0·underflow float·빈 문자열·빈 배열·빈 dict)는 필터를 생략합니다. dict 필터는 actual truthiness를 먼저 확인하고 Source insertion order로 short circuit하며, 도달한 unknown key는 오류입니다. 빈 nested dict는 truthy actual dict 또는 scalar도 통과시키고 falsey actual은 제외합니다. bool/int와 CPython-loaded float equality를 공통 `SelectLoaded`로 재사용합니다. 잘못된 filters·expression도 모든 페이지를 수집한 후 판정하므로 늦은 HTTP 오류가 먼저 반환됩니다.

`GetEndpoint(ctx, id, ...)`는 Filters omission 또는 literal null에서 private Source Resource.find를 실행합니다. literal member GET에 clean400/403/404가 돌아오면 name query 없이 목록의 exact id-or-name을 확인하고 duplicate는 `resource.ErrAmbiguous`, missing은 nil Value/Endpoint로 반환합니다. wildcard identifier도 이 분기에서는 exact 비교이며 Resource/dict passthrough는 없습니다. Source get_endpoint는 `_get_entity`를 호출하지 않습니다. Filters가 None 이외이면 `{}`, false, 0, [], 빈 문자열처럼 falsey여도 `DeprecatedFilters=true`로 search 분기를 선택하고 glob을 적용합니다. Go는 Python warning category를 실행하지 않으며 result의 marker로 이 분기를 보여줍니다.

deprecated JMESPath 출력은 arbitrary JSON입니다. Source의 truthiness→len→integer index0 순서를 유지하며 `[false]`, `[0]`, `[""]`는 각각 present false/zero/빈 문자열이고 `[null]`는 missing입니다. 두 개 이상의 배열·문자열·dict 결과는 ambiguous, truthy numeric/bool이나 integer key0가 없는 dict는 shape 오류입니다. ordinary row selection만 `Endpoint`/`Endpoints`에 실제 record를 연결합니다. expression 변형에는 HTTP record를 새로 만들지 않습니다. `Inventory`는 search/deprecated-get에서 실제 수집한 전체 rows를 보존하고 default find는 반환한 record의 실제 receipt를 제공합니다.

`EndpointRecord.Resource`는 Source Body 기본값·enabled bool alias·untyped links와 computed location view이고 `Wire`·`Envelope`·`Header`·`StatusCode`는 실제 응답 증거입니다. known/unknown/self·정밀 JSON 숫자를 Wire에 유지합니다. Direct ServiceClient의 location은 기본 null입니다. `WithEndpointReadLocation`으로 지정한 독립 CloudLocation은 현재 scope를 사용하며 endpoint row의 region_id·spoofed location·unknown project 값으로 auth scope를 바꾸지 않습니다. selected view·Inventory·헤더·원문 bytes는 독립 복제본입니다.

Header·microversion·location 옵션은 Go의 명시적 전송 편의입니다. caller callback 전에 원본 API/client/provider/route/version/header를 snapshot하고 `cloudread` 및 기존 outer REST operation guard로 전체 workflow를 감시합니다. 원본 header map의 이후 수정은 captured HTTP에 반영하지 않으며, source/route 변경·취소·auth/header 교체는 HTTP 또는 후속 callback을 중단합니다. selected microversion 그대로 적용하고 공유 client를 수정하지 않습니다. 모든 요청은 owned bodyless GET입니다. physical read/Close/source/cancel 오류는 actual body/status/header와 cause를 유지하며 fallback/auth/retry/backoff로 숨기지 않습니다. clean native 재시도는 공통 transport 정책을 유지합니다.

공통 `cloudread`, `rest.OperationGuard`, Source descriptor/find/list 엔진, `cloudfilter.SelectLoaded/First`와 public Gophercloud testhelper를 재사용합니다. Source Link/status/marker 규칙은 고정 Endpoint leaf와 동일합니다. Go의 typed v3 factory·safe route/blank identity·JSON operand는 Python의 arbitrary proxy/session/object를 제한합니다. Optional HTTP cache, mutable Resource/Connection lifecycle와 selected runtime-version metadata는 기존 **SDK-R1/C1/S1** 공통 경계로 유지합니다. 이 두 named helper의 구현을 전체 SDK parity로 확대하지 않습니다.
