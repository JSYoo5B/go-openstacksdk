# Nova extension 검색의 Python/Go 사용법

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [find_extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L108)과 [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/extension.py)에 대응합니다. `compute.Extensions.FindRecord(ctx, nameOrID, options...)`는 concrete 옵션을 사용하며 기존 native `Get`/`List`를 변경하지 않습니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--extension", required=True)
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    value = conn.compute.find_extension(args.extension, ignore_missing=not args.strict)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

독립 `main.go`이며 [설치 안내](../../../docs/install.md)를 따릅니다.

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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions"
)

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    identity := flag.String("extension", "", "literal alias or exact name")
    strict := flag.Bool("strict", false, "return missing errors")
    flag.Parse()
    if *identity == "" { log.Fatal("-extension is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    value, err := compute.Extensions.FindRecord(ctx, *identity,
        extensions.WithRecordFindIgnoreMissing(!*strict))
    if err != nil { log.Fatal(err) }
    var output any
    if value != nil { output = value.Resource }
    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

GET-first로 `extensions/{literal}`을 읽고 clean400/403/404일 때만 전체 fallback 목록으로 진행합니다. 첫 GET이 성공하면 seed ID를 유지하고 반환된 다른 alias/name으로 재검색하지 않습니다. 목록의 primary id는 null/false/빈 값까지 alternate alias보다 우선하며 exact ID 또는 name을 비교합니다. 둘 이상 일치하면 `resource.ErrAmbiguous`, missing은 기본 nil이고 `WithRecordFindIgnoreMissing(false)`에서는 `resource.ErrNotFound`입니다. 첫 일치 뒤의 다른 페이지에서도 중복을 확인합니다.

Compute Extension은 Resource를 직접 상속하며 Network Extension과 독립적인 모델입니다. unknown Body를 보존하지 않고 `namespace`를 가지며 links에 list_type=dict를 적용합니다. null links는 null, object는 한 요소 목록, nonobject item은 `{}`입니다. updated는 literal `updated_at`이고 inherited id/name과 다른 missing descriptor는 null입니다. 실제 JSON·self·unknown·alias spelling은 독립적인 Wire와 Envelope/header/status에 보존합니다. 직접 ServiceClient의 location은 SDK-owned null이며 공통 SDK-R1의 Connection 계약을 완료로 처리하지 않습니다.

Source max microversion은 상속값 **None**입니다. 기본 Find는 discovery를 추가로 하지 않고 selected client version이 있으면 그대로 사용합니다. `WithRecordFindMicroversion`은 GET만 바꾸고 fallback LIST는 원래 selected version을 사용합니다. `WithRecordFindListBasePath`는 fallback만의 owned service-relative route이며 member route를 바꾸지 않습니다. Header·raw JSON Query kwargs를 callback 전후에 복제하고 source/context를 검사합니다. `WithRecordFindQuery`/headers/base-path/microversion은 Python 직접 find_extension signature보다 넓은 Go convenience이고 대응 Source는 generic Resource.find입니다. URI/session/constructor 충돌을 조용히 덮어쓰지 않습니다.

공식 Nova **31.0.0**의 [extension policy](https://github.com/openstack/nova/blob/31.0.0/nova/policies/extensions.py)는 list/show를 project scope의 RULE_ANY로 허용합니다. admin 전용 연산이 아니며 실제 deployment policy는 해당 클라우드가 결정합니다. member403에서 Source대로 fallback해도 list 결과를 permission 성공이나 다른 project의 availability 증거로 취급하지 않습니다.

Physical read/Close·source mutation·cancel을 동반한400/403/404는 fallback하지 않습니다. native401 pointer의 ErrOriginal method/URL/body/header/code와 ErrReauth faultcause를 유지하며 fault 뒤의 retry/auth/backoff를 멈춥니다. clean401 renewal은 literal member route를 유지합니다. optional cache와 arbitrary Python Resource lifecycle은 공통SDK-C1/R1에서 관리하며 이 검색을 fullSDK parity나 realcloud acceptance로 확대하지 않습니다.
