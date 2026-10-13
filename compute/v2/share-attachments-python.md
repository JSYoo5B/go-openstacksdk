# Nova share attachment의 Python/Go 사용법

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Compute Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3422)와 [ShareMapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_share.py)입니다. Gophercloud `v2.15.0`에는 해당 binding이 없어 SDK가 공통 `cloudread`, microversion discovery, REST 응답·paging 계약을 재사용합니다. 사용자 builder나 decoder interface는 필요하지 않습니다.

| Python | Go | 기본 동작 |
| --- | --- | --- |
| `conn.compute.create_share_attachment(server, share, **attrs)` | `compute.ShareAttachments.CreateRecord(ctx, serverID, shareID, options...)` | `POST servers/{serverID}/shares`, `share_id`와 직접 지정한 Body 속성만 전송 |
| `conn.compute.get_share_attachment(server, share)` | `compute.ShareAttachments.GetRecord(ctx, serverID, shareID, options...)` | literal member GET, share ID를 초기 `id`로 유지 |
| `conn.compute.share_attachments(server, **query)` | `compute.ShareAttachments.ListRecords(ctx, serverID, options...)` | 지연 목록, Body 속성은 로컬 필터, limit/marker·페이지 정책은 Source에 대응 |
| `conn.compute.delete_share_attachment(server, share, ignore_missing=True)` | `compute.ShareAttachments.DeleteRecord(ctx, serverID, shareID, options...)` | body 없는 DELETE, 온전한 최종 HTTP 404만 기본 무시 |

서버 UUID와 Manila share UUID를 구분해 명시적으로 전달합니다. 응답 `uuid`는 Nova의 mapping UUID이며 요청의 `{share_id}`가 아닙니다. Python이 Server·Share Resource에서 ID를 읽는 대신 Go는 `server.ID`, `share.ID`를 전달합니다. ID는 이름으로 해석하지 않습니다.

아래 Python은 독립 실행 예제입니다. `--mode create`는 Manila share를 해당 서버에 연결합니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--server", required=True)
parser.add_argument("--share", default="")
parser.add_argument("--mode", choices=("list", "get", "create", "delete"), default="list")
parser.add_argument("--tag", default=None)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud, compute_default_microversion="2.97") as conn:
    if args.mode == "list":
        for value in conn.compute.share_attachments(args.server):
            print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "get":
        value = conn.compute.get_share_attachment(args.server, args.share)
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "create":
        attrs = {} if args.tag is None else {"tag": args.tag}
        value = conn.compute.create_share_attachment(args.server, args.share, **attrs)
        print(json.dumps(value.to_dict(), indent=2))
    else:
        conn.compute.delete_share_attachment(args.server, args.share)
        print(json.dumps({"delete_requested": args.share}, indent=2))
```

다음 코드는 독립 `main.go`입니다. [설치 안내](../../docs/install.md)에 따라 Go module에서 실행합니다. `-mode list`는 조회만 수행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "", "server UUID")
    share := flag.String("share", "", "Manila share UUID")
    mode := flag.String("mode", "list", "list, get, create or delete")
    tag := flag.String("tag", "", "optional guest mount tag for create")
    strict := flag.Bool("strict", false, "return missing attachment errors on delete")
    flag.Parse()
    if *server == "" { log.Fatal("-server is required") }
    if *mode != "list" && *share == "" { log.Fatal("-share is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.Compute, "2.97"))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    switch *mode {
    case "list":
        for value, err := range compute.ShareAttachments.ListRecords(ctx, *server) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    case "get":
        value, err := compute.ShareAttachments.GetRecord(ctx, *server, *share)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
    case "create":
        var options []shareattachments.CreateOption
        if *tag != "" {
            options = append(options, shareattachments.WithCreateAttribute("tag", *tag))
        }
        value, err := compute.ShareAttachments.CreateRecord(ctx, *server, *share, options...)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
        fmt.Printf("HTTP %d\n", value.StatusCode)
    case "delete":
        value, err := compute.ShareAttachments.DeleteRecord(ctx, *server, *share,
            shareattachments.WithDeleteIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        printJSON(value)
    default:
        log.Fatal("unknown -mode")
    }
}
```

## 옵션과 반환값

`WithCreateAttribute("tag", value)` 또는 `WithCreateOptions(CreateOpts{Attributes: ...})`로 생략과 명시 JSON null을 구분합니다. 실제 Source의 여덟 Body 속성 `export_location`, `share_id`, `share_proto`, `status`, `tag`, `uuid`, `id`, `name`만 Source 속성으로 처리하며, `share_id`는 별도의 share argument가 소유합니다. 일반 unknown 속성은 버리며 불필요한 marshaler를 실행하지 않습니다. 원시 Attributes map의 unknown 값은 유효하지 않은 JSON이어도 버립니다. 직접 builder·decoder를 만들 필요는 없습니다.

`Record.Resource`는 Source의 선언값과 기본값입니다. `Record.Wire`는 반환된 실제 JSON 객체이고 unknown·`self`·정확한 큰 숫자를 유지합니다. `Envelope`, `Header`, `StatusCode`와 목록 행의 `ServerID`는 실제 응답과 URI 증거입니다. Body·헤더 byte는 서로 독립적으로 소유합니다. 목록 Envelope는 해당 전체 페이지이며 Wire는 해당 행입니다. default `export_location`은 `""`이고 명시 null은 유지합니다. primary `id`가 있는 경우 null/false/빈 값도 alternate `share_id`보다 우선합니다. Get에서는 Proxy가 미리 넣은 member ID가 response share_id보다 우선합니다. `uuid`는 이 계산에 참여하지 않습니다.

Source의 fetch/create는 비 JSON 성공 응답에서 Resource seed를 유지합니다. Go도 이를 따르며 실제 receipt를 보존하고 `Wire=nil`로 둡니다. 유효한 JSON이 객체 형식과 맞지 않으면 오류에 actual receipt를 붙입니다. Delete는 응답 JSON을 Resource로 변환하지 않습니다. 2xx는 Source의 성공 응답 범위로 취급하며 native controller의 일반 결과는 Create 201, Get/List 200, Delete 200입니다. 서버의 정상 응답과 클라이언트가 고른 수락 범위를 혼동하지 않습니다.

`WithListFilter`/`WithListFilters`는 선언 Body 조건을 로컬에서 적용하고 `limit`·`marker`를 서버 query로 보냅니다. 로컬 equality는 Source의 Python JSON 해석을 따릅니다. bool은 integer와 비교하고, decimal/exponent는 float underflow·rounding을 반영하며 정수는 임의 정밀도를 유지합니다. dictionary 조건은 actual truthiness를 먼저 확인한 뒤 재귀 subset으로 비교합니다. 실제 Wire/Resource의 JSON bytes는 독립적으로 유지합니다. `WithListMaxItems(n)`는 필터 전에 소비할 raw 행 수를 제한하며, `WithListPaginated(false)`는 첫 페이지만 읽습니다. limit 없는 기본 목록은 wire query를 보내지 않습니다. Source도 상속한 `limit`·`marker`를 보낼 수 있지만 stock Nova 31의 목록 schema는 query를 지원하지 않으므로 해당 옵션·max-items의 limit hint는 stock 서버가 400으로 거절할 수 있습니다. stock 전체 목록에서 결과 수만 줄이려면 iterator를 `break`로 종료하세요. Go의 marker는 nonblank 문자열이어야 합니다. Source는 primary `id`가 null/false/빈 값일 때 그 값을 marker params에 넣을 수 있지만, Go는 alternate share_id로 바꾸지 않고 실제 페이지 receipt와 함께 종료합니다. 동일 origin·collection path·cursor 진행 검사도 Go의 강화 정책입니다. Source kwargs에서 일반 unknown 항목은 무시하고 `jmespath_filters`, routing/session/constructor 제어 항목은 Go에서 명시적으로 거절합니다. arbitrary JMESPath 변환을 완료한 것으로 처리하지 않습니다.

선택 microversion이 있으면 그 값을 그대로 사용합니다. 없으면 공통 discovery로 advertised 최대와 Source ceiling **2.97** 중 작은 값을 선택합니다. `With*Microversion("...")`은 한 호출의 변경이며 shared ServiceClient를 수정하지 않습니다. 낮은 2.96·ceiling보다 높은 명시 버전을 2.97로 올리거나 자르지 않고 서버 판단에 맡깁니다. 명시 빈 버전은 version 헤더를 생략하는 Go 옵션입니다. discovery 증거는 `Discovery`에 별도로 반환하며 실패한 discovery를 member HTTP 성공으로 꾸미지 않습니다.

## 서버 권한과 지원 범위

확인한 stock 서버는 공식 Nova **31.0.0**의 [정책](https://github.com/openstack/nova/blob/31.0.0/nova/policies/server_shares.py), [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/server_shares.py), [schema](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/schemas/server_shares.py), [역할 정의](https://github.com/openstack/nova/blob/31.0.0/nova/policies/base.py)입니다. 서버 소스의 SHA256은 policy `ef40e310182aea3aeccf821d159b030e9a2f13cf530b18373c71b3ba8cce7c06`, controller `068dd5ba7ee69e53269a7252d450ffadc4879101449617dad7caff97740cc518`, schema `a12b3807ef32c6b242eaa5620525e13e2ece94b30b6c030682d10455d6e14cb5`입니다.

네 endpoint는 microversion **2.97**부터 존재합니다. Get/List는 server 소속 project의 reader, Create/Delete는 member 경로이며 controller는 instance mapping의 project_id를 policy target으로 전달합니다. 배포된 policy override는 별개입니다. SDK가 reader에게 create 권한을 제공하는 것이 아닙니다.

Create의 stock schema는 UUID `share_id`와 선택 `tag`만 받습니다. Source가 다른 declared Body 속성을 보내는 기능과 stock 서버가 그 속성을 수락한다는 것은 다릅니다. tag 생략 시 share UUID를 사용하는 주체는 Nova 서버이며 SDK가 request에 기본 tag를 넣지 않습니다. Nova는 share available 상태, export location, 지원 프로토콜, 서버/virt 드라이버 설정과 중복 share/tag를 검사합니다. 반환된 mapping의 비동기 상태나 HTTP receipt를 guest mount 완료로 취급하지 않습니다. 이 패키지는 Manila share 생성·access 관리나 attachment 상태 wait를 수행하지 않습니다.

공통 SDK 계약에 남아 있는 Connection location/optional HTTP cache, Python Resource 객체의 변이·임의 session/base_path, arbitrary JMESPath는 구현 범위와 별도로 남깁니다. 이 binding의 Resource view는 직접 ServiceClient가 제공하지 않는 `location`을 null로 두며 actual response는 Wire에서 확인합니다. 고정 Source의 직접 네 API를 concrete Go 사용법으로 매핑하며, 이를 전체 SDK parity나 실제 cloud 권한·mount acceptance 검증으로 확대하지 않습니다.

테스트는 재사용 가능한 public Gophercloud testhelper 기반 `internal/testcloud`를 사용합니다. 고정 요청 URI·Body·microversion, 실제 응답의 default/seed/null/unknown, 지연 목록·local cap·alternate marker, clean404·clean401 갱신과 physical Read/Close 오류를 별도로 확인합니다. 물리적 오류 후 retry/auth/backoff callback이 동작하거나 다른 요청을 발행하지 않으며 native 401 wrapper의 실제 body/header/code와 각 fault cause를 보존합니다. 실제 클라우드와 Python 패키지 실행은 별도 검증입니다.
