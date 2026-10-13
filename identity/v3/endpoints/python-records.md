# Keystone endpoints: openstacksdk 비교

고정 비교 대상은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [`Proxy.endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py), [`Endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/endpoint.py), `Resource.list`와 Gophercloud v2.15.0입니다. 기존 `List`·`All`·`Get`·`Find`와 native 타입 계약은 유지합니다. `ListRecords`·`AllRecords`는 Source 모델과 필터·페이지 동작을 제공하고 `ProxyListRecords`는 Source의 arbitrary JMESPath 반환을 제공합니다.

공식 Keystone **27.0.0** [endpoint 정책](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/endpoint.py)과 [base 규칙](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/base.py)을 실제 파일로 확인했습니다. `identity:list_endpoints`는 `admin_required` 또는 `role:reader and system_scope:all`이며 선언 scope는 system·project입니다. 일반 project reader만으로 허용된다는 뜻은 아닙니다. 아래 예제의 `system-reader` cloud는 system scope reader 인증을 가진 clouds.yaml 항목입니다. 운영 policy override와 실제 cloud acceptance는 서버가 결정합니다. 이 문서는 목록 조회 한 행만 다룹니다.

## 실행 가능한 Python 예제

`openstacksdk`와 clouds.yaml의 `system-reader` 항목을 준비합니다. 아래를 `endpoints.py`로 저장한 뒤 `python endpoints.py --cloud system-reader --service-id SERVICE_UUID --region-id RegionOne`으로 실행합니다. 두 필터 flag는 생략할 수 있습니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="system-reader")
    parser.add_argument("--service-id", default="")
    parser.add_argument("--region-id", default="")
    args = parser.parse_args()
    query = {"is_enabled": True, "max_items": 100}
    if args.service_id:
        query["service_id"] = args.service_id
    if args.region_id:
        query["region_id"] = args.region_id
    conn = openstack.connect(cloud=args.cloud)
    for row in conn.identity.endpoints(**query):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go 예제

동일한 clouds.yaml 항목을 사용합니다. 아래를 Go module의 `main.go`로 저장하고 `go run . -cloud system-reader -service-id SERVICE_UUID -region-id RegionOne`으로 실행합니다. `Resource`의 JSON은 읽기 모델이고 `Wire`·`Envelope`·`Header`·`StatusCode`에 실제 응답 증거가 있습니다.

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
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
)

func main() {
	cloud := flag.String("cloud", "system-reader", "clouds.yaml cloud")
	service := flag.String("service-id", "", "optional service UUID")
	region := flag.String("region-id", "", "optional region ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	identity, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []endpoints.RecordListOption{
		endpoints.WithRecordListQuery("is_enabled", true),
		endpoints.WithRecordListMaxItems(100),
	}
	if *service != "" {
		options = append(options, endpoints.WithRecordListQuery("service_id", *service))
	}
	if *region != "" {
		options = append(options, endpoints.WithRecordListQuery("region_id", *region))
	}
	for row, err := range identity.Endpoints.ListRecords(ctx, options...) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(row.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
}
```

## 목록 계약과 Go 차이

`Endpoint`는 Resource를 직접 상속합니다. Body 필드는 `id`, `name`, `interface`, `is_enabled`, `links`, `region_id`, `service_id`, `url`입니다. missing·null은 null이며 `enabled`→`is_enabled`는 Source bool truthiness를 적용하고 canonical/wire alias가 둘 다 있으면 실제 JSON 멤버 순서대로 마지막 값이 이깁니다. 문자열 `"false"`도 nonempty이므로 true입니다. `links`는 untyped이므로 null·dict·list·scalar를 변환하지 않습니다. `region`, `description`, unknown 값과 제거되는 `self`는 실제 Wire에만 남습니다. Source 직접 Connection의 현재 location은 이 ServiceClient 읽기 모델에 없으므로 SDK-owned `location`은 null입니다.

QueryParameters는 `interface`, `region_id`, `service_id`, 기본 `limit`·`marker`를 서버로 보냅니다. `name`, `id`, `is_enabled`, `links`, `url` 같은 다른 선언 Body kwargs는 로컬 필터입니다. `enabled`는 wire alias지만 선언 Body attribute 이름이 아니므로 목록 kwargs에서는 unknown으로 버립니다. unknown 값은 JSON marshaler를 실행하기 전에 무시합니다. dict 필터는 Source의 truthy actual mapping과 재귀 subset 조건을 사용하며 non-dict 필터는 Python-loaded JSON equality를 적용합니다. 따라서 true와 1, CPython float rounding/underflow 비교를 공유합니다. Go JSON options는 임의 Python Resource 객체 operand를 생성하지 않습니다.

Keystone response의 `{ "links": { "next": "..." } }`는 pinned Python `_get_next_link`가 dict의 각 멤버를 별도 dict로 순회하면서 next relation으로 인식하지 않으므로 새 목록도 무시합니다. root `links`의 존재가 `endpoints_links`보다 우선하고, relation 목록의 첫 literal `rel="next"`·`href`, top-level `next`, HTTP Link 순서로 진행합니다. limit이 truthy이면 마지막 raw row ID를 marker로 사용해 빈 페이지까지 반복하며 필터에서 제외된 row도 marker와 raw cap에 포함됩니다. `max_items`는 로컬 필터 전에 raw 수를 제한하고 falsey wire limit이면 limit hint를 보냅니다. `paginated=false`, break, cap, 취소는 사용하지 않는 다음 링크를 요청하지 않습니다. nil/false/빈값/nonstring marker와 origin·route·cursor 변경은 실제 page receipt를 붙인 terminal error로 중단하는 Go guard 차이가 있습니다. Source가 임의 session/origin으로 이동하는 동작은 허용하지 않습니다.

기본 `_max_microversion=None`이므로 별도 discovery를 실행하지 않습니다. 선택된 버전은 그대로 사용하며 명시된 microversion은 모든 페이지에 적용하고 shared client는 수정하지 않습니다. Identity 역할의 `OpenStack-API-Version`을 소유하며 Nova 전용 header는 보내지 않습니다. `WithRecordListHeaders`, `WithRecordListMicroversion`, `WithRecordListBasePath`, `WithRecordListConflictingAttrs`가 Source formal kwargs와 전용 Go 옵션을 처리합니다. runtime receiver·session·auth/header·origin 교체는 owned guard로 제한합니다. callback의 map/RawMessage는 복제하고 nil map에도 안전한 owned map을 제공합니다.

Source Proxy는 JMESPath를 collected 배열이 아닌 실제 lazy generator에 적용합니다. `ProxyListRecords.Value`는 `RecordListGenerator`, arbitrary JSON 또는 같은 generator를 포함한 값이고 `JSON`은 완전히 직렬화할 수 있는 결과에만 존재합니다. projection/index/type는 null, `@ == @`는 같은 identity이며 `length(@)`·`to_string(@)`와 generator JSON marshal 실패도 HTTP를 소비하지 않습니다. `Value.(*RecordListGenerator).Records`를 소비하면 실제 조회가 시작됩니다. 일반 `ListRecords`는 truthy JMESPath expression에 `ProxyListRecords` 사용을 요구합니다. `AllRecords`는 오류 시 partial slice를 버리며 정상 빈 목록은 nonnil empty slice입니다.

공통 `cloudread`, Source descriptor Engine, REST pager, JSON equality, opaque JMESPath, public Gophercloud testhelper와 기존 native endpoint fixture를 재사용합니다. 새 profile만 HTTP100..399 Source status policy를 명시적으로 opt-in하며 기존 native 동작은 유지합니다. physical read/Close/source/cancel 오류는 actual response와 cause를 유지하고 auth/retry/backoff 또는 다음 요청으로 숨기지 않습니다. clean401은 ReauthFunc, clean429는 RetryBackoffFunc, clean503은 RetryFunc를 통해 native 재시도합니다. 선택된 method·URI·query·header는 동일하게 유지합니다. Optional HTTP cache, Python Resource mutation/Connection lifecycle는 기존 SDK-R1/C1 공통 경계이며 이 한 named 목록을 전체 SDK parity로 확대하지 않습니다.
