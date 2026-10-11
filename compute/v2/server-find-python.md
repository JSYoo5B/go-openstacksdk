# Nova 서버 Find: openstacksdk와 Go

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 named API `compute.find_server`에 대응합니다. Gophercloud는 `v2.15.0`입니다. 기존 typed `Servers.Get`·`List`·`FindIdentity`는 계속 사용할 수 있으며, Python descriptor의 별칭·기본값·null과 실제 응답을 함께 확인하려면 `Servers.FindRecord`를 사용합니다. 이 항목은 서버 전체 API나 mutable Resource·metadata API 완료를 의미하지 않습니다.

| openstacksdk | go-openstacksdk | 기본값/동작 |
|---|---|---|
| `conn.compute.find_server(name_or_id)` | `service.Servers.FindRecord(ctx, nameOrID)` | literal member GET 우선, clean 400/403/404만 목록 fallback, 상세 목록, 현재 project, missing은 `nil, nil` |
| `ignore_missing=False` | `WithRecordFindIgnoreMissing(false)` | 전체 성공 목록에서 부재가 확정되면 `resource.ErrNotFound` |
| `details=False` | `WithRecordFindDetails(false)` | fallback만 `/servers` 사용; 기본은 `/servers/detail`, GET은 그대로 |
| `all_projects=True` | `WithRecordFindAllProjects(true)` | fallback에만 `all_tenants=True` 전송; 배포의 Nova 정책이 허용해야 함 |

고정 Python proxy는 위 세 옵션 외에 `**query`를 노출하지 않습니다. 이 Go API의 Query·Headers·Microversion·Location·pagination 옵션은 inherited Resource 동작을 참고한 명시적인 Go 편의 기능입니다. Python proxy 호출에 같은 인자가 있다고 가정하면 안 됩니다.

## 실행 가능한 Python 예제

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--server", required=True)
parser.add_argument("--basic", action="store_true")
parser.add_argument("--all-projects", action="store_true")
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()

conn = openstack.connect(cloud=args.cloud)
server = conn.compute.find_server(
    args.server,
    ignore_missing=not args.strict,
    details=not args.basic,
    all_projects=args.all_projects,
)
print(json.dumps(None if server is None else server.to_dict(), indent=2))
```

## 실행 가능한 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`에 저장합니다. `go run . -cloud dev -server VM_NAME`은 조회만 실행합니다. `-basic`은 기본 목록으로 fallback하고, `-strict`는 부재를 오류로 처리합니다. `-all-projects`는 project 범위를 넓혀 요청할 뿐 인증 권한을 바꾸지 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("server", "", "exact server ID or name")
    basic := flag.Bool("basic", false, "use basic fallback list")
    allProjects := flag.Bool("all-projects", false, "request cross-project lookup")
    strict := flag.Bool("strict", false, "return an error for missing server")
    version := flag.String("microversion", "", "optional selected service microversion")
    flag.Parse()
    if *identity == "" { log.Fatal("-server is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    connectionOptions := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" {
        connectionOptions = append(connectionOptions,
            sdk.WithMicroversion(sdk.Compute, *version))
    }
    conn, err := sdk.Connect(ctx, connectionOptions...)
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    record, err := service.Servers.FindRecord(ctx, *identity,
        servers.WithRecordFindIgnoreMissing(!*strict),
        servers.WithRecordFindDetails(!*basic),
        servers.WithRecordFindAllProjects(*allProjects))
    if errors.Is(err, resource.ErrNotFound) { log.Fatal("server not found") }
    if errors.Is(err, resource.ErrAmbiguous) { log.Fatal("server name is ambiguous") }
    if err != nil { log.Fatal(err) }
    if record == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(record.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
    fmt.Printf("HTTP %d microversion=%s request-id=%s\n",
        record.StatusCode, record.Microversion,
        record.Header.Get("X-Openstack-Request-Id"))
}
```

## 검색·권한·microversion

identity는 공백이 아닌 literal UTF-8 string입니다. `/`·`%`는 outgoing path에서 한 번 escape하며 응답의 `id=null`·숫자·누락 때문에 요청 ID를 교체하지 않습니다. GET 성공은 즉시 반환합니다. fallback은 정확한 문자열 ID **또는** 이름을 비교하며 하나를 찾은 뒤에도 나머지 관찰 가능한 page에서 중복과 오류를 검사합니다. 중복은 `resource.ErrAmbiguous`, 성공한 전체 목록의 부재만 기본 `nil, nil` 또는 strict `resource.ErrNotFound`입니다. 첫 GET의 억제된 400/403/404를 논리적 missing 오류에 다시 붙이지 않습니다. 목록 403·다른 HTTP 오류·GET의 read/Close/cancel/source/retry callback 오류는 missing으로 억제하지 않습니다.

`all_projects`는 Source처럼 GET query에 들어가지 않습니다. 고정 proxy docstring은 cross-project 검색을 기본 admin-only라고 설명합니다. Go는 별도 관리자 token 획득·정책 우회·권한 추정 없이 선택된 인증과 route로 요청하고, Nova의 실제 403을 보존합니다. default false는 자동 `all_tenants=False`도 전송하지 않습니다.

선택된 service microversion은 `latest`와 ceiling보다 높은 값도 변경하지 않습니다. 미선택이면 공통 Nova discovery가 원래 endpoint에서 읽은 실제 응답을 검증하고 advertised maximum과 Server ceiling **2.100** 중 작은 값을 선택합니다. 최소 버전이 ceiling보다 높거나 maximum이 없으면 version header 없이 요청합니다. discovery의 advertised self link는 eligibility 검증에만 쓰며 새로운 origin/route를 선택하지 않습니다. 동일 Find에서 기본 협상은 한 번 수행합니다. discovery 오류는 GET 누락으로 해석하지 않습니다.

Go 편의 `WithRecordFindMicroversion("2.50")`는 pinned `Resource.find`처럼 **member GET에만** 적용하고, fallback은 캡처한 service default 또는 별도 협상 결과를 사용합니다. `WithRecordFindMicroversion("")`는 member에 version header를 생략하며 LIST default를 지우지 않습니다. `record.Microversion`은 그 record를 수신한 phase의 실제 선택값입니다. `Discovery`는 실제 발견 응답들만 별도 snapshot으로 보관합니다. 원본 service client와 provider를 변경하지 않습니다.

## Resource 뷰와 원본 응답

`Resource.Body`는 정확한 Source Server Body descriptor와 inherited `id`·`name`·metadata·tags, computed IP 기본값과 location을 투영합니다. 예를 들어 `tenant_id` → `project_id`, `created` → `created_at`, `locked` → `is_locked`, `OS-EXT-SRV-ATTR:host` → `compute_host`입니다. 같은 wire field를 공유하는 `config_drive`·`has_config_drive`는 같은 값을 가집니다. dictionary/kwargs 입력 순서를 보존하는 경우 같은 field의 마지막 별칭 값이 적용됩니다. `volumes`는 `attached_volumes`의 Source `aka` 출력 별칭입니다.

미수신 일반 descriptor는 null이며 `tags`와 `attached_volumes`는 `[]`, computed IP 다섯 필드는 `""`입니다. 명시 null은 null로 남습니다. 정수 descriptor는 공통 stricter BodyRecordField integer 변환 대신 audited `jsonfilter.DescriptorIntegerJSON`의 Source profile을 재사용합니다. Python bool은 int subclass이므로 int descriptor의 true/false는 그대로 유지하며, Unicode decimal digit 문자열과 큰 정수는 정확히 보존합니다. JSON float는 binary64 값으로 잘라 정수로 만들고 signed/space-padded/nonnumeric 문자열과 일반 container는 0이 됩니다. isdigit이지만 int로 변환할 수 없는 `"²"` 및 nonfinite float 변환은 실제 응답 proof를 보존한 오류입니다. bool descriptor는 Python truthiness를 따릅니다. 문자열 `"false"`도 bool descriptor에서는 true입니다. untyped 값과 timestamp를 native 시간 타입으로 바꾸지 않습니다. dict descriptor의 nonnull 비객체는 `{}`, list descriptor의 nonnull scalar는 한 원소 list입니다. 이 Source 규칙은 malformed native struct decoding과 다를 수 있습니다.

중첩 `flavor`·`image`·`attached_volumes`는 실제 Source `_convert_type`의 Resource subtype 조건과 기본값을 따릅니다. Flavor RAM·disk·vcpus 등은 기본 0, public은 true, extra_specs는 `{}`이며 ID는 truthy id/name/original_name 순서입니다. Image unknown 값은 `properties`에 들어가고 owner/owner_id는 공유 field입니다. Image `min_disk`·`min_ram`은 이 pin에서 **untyped**라 문자열 원본을 보존합니다. Attachment ID는 `id`가 없을 때만 `volumeId`를 사용합니다. `image=""` 같은 비객체 Resource 값과 typed list의 null/nonobject Resource 원소는 Source처럼 `{}`입니다. `security_groups`의 typed dict 원소도 비객체이면 `{}`입니다. 중첩 Resource에는 JSON runtime connection을 만들지 않으며 location은 null입니다. Python class/subclass·arbitrary Connection·nonfinite/custom runtime object는 이 JSON profile에 포함하지 않습니다. `self` initializer 충돌과 nonnull 중첩 connection은 오류로 처리합니다.

Server는 unknown Body attrs를 유지하는 Resource가 아닙니다. unknown 응답 field와 원본 integer/decimal/timestamp는 `Wire.Body`에 남고 임의 Body Header descriptor를 만들어 주지 않습니다. `If-Match`는 실제 `Header` 증거이며 이 Server class에 없는 `if_match` 속성으로 합성하지 않습니다. `Wire`는 실제 row, `Envelope`는 실제 전체 HTTP body, `Header`·`StatusCode`는 실제 수신 receipt입니다. LIST record의 Envelope은 row를 받은 전체 page입니다. 각각 독립 snapshot이라 한 뷰를 수정해 다른 뷰를 변경하지 않습니다. 빈/문법 오류 GET 성공 body는 Source처럼 constructor seed/default Resource를 유지하고 실제 row가 없으므로 Wire는 nil입니다. 유효 JSON wrong shape·잘못된 UTF-8·descriptor 오류는 실제 accepted receipt를 `resource.ResponseError`로 보존합니다. 고정 Source `raise_from_response`는 `<400`에서 반환합니다. 이 Go layer는 final representation의 200..399만 성공으로 처리하고 informational 100..199는 의도적으로 거부하며 실제 native status/body/header를 보존합니다. 이 차이를 Source의 동일 status 범위라고 집계하지 않습니다.

outer location은 이미 기록된 Keystone v2/v3 인증 project ID에서 계산하며 별도 인증 요청을 보내지 않습니다. manual token이나 scope 없는 결과로 project 이름을 추정하지 않습니다. `WithRecordFindLocation`으로 cloud/region/current-project names를 제공할 수 있습니다. foreign project는 raw ID를 유지하고 현재 project 이름들을 지우며 zone은 resource의 availability_zone을 사용합니다. 이 옵션은 native connection 설정으로부터 전체 Python current_location을 자동 복제한다는 뜻이 아닙니다.

## Go query·pagination·요청 소유권

`WithRecordFindQuery`는 JSON bytes·null·false·0·empty를 snapshot으로 저장합니다. 원래 kwargs는 GET query에 그대로 전달하고 알려진 Body field는 constructor seed로도 사용합니다. fallback은 Source QueryParameters로 분류한 server query만 전송합니다. `changes_since` → `changes-since`, `any_tags` → `tags-any` 같은 client query 이름이 wire alias보다 우선하며 명시 null도 자동 hint/alias를 막습니다. name이 아예 없는 경우만 literal identity를 name hint로 넣습니다. 알려진 Body canonical 이름 중 server query가 아닌 값은 local filter이며 unknown/wire-only query는 LIST filter를 만들지 않습니다. sequential options는 순서를 유지하고 bulk map이나 callback의 새 map keys는 Go map 순서가 없어 sorted keys를 사용합니다.

local dict filter는 Python의 falsey/empty/nested subset 및 truthy scalar empty-dict match와 mapping.get 오류를 보존합니다. bool-number equality(`True == 1`)를 지원하며 일반 JSON equality는 shared exact-decimal profile입니다. scalar decimal truthiness는 binary64 underflow/overflow를 따르지만 모든 Python float rounding/nonfinite/custom object 비교를 재현하지는 않습니다. float descriptor는 finite JSON 표현만 지원합니다.

`WithRecordFindLimit`, `Marker`, `MaxItems`, `Paginated`는 typed Go LIST controls입니다. 기본은 전체 lazy paging이며 MaxItems=0은 무제한입니다. 양수 raw-row cap은 local filter **전**에 적용합니다. 명시 limit=0과 양수 cap이 함께 있으면 Source처럼 cap이 limit hint가 됩니다. explicit limit의 짧은 page도 wire id marker로 이어가며 빈 page에서 멈춥니다. plural links·top-level next·HTTP Link와 singleton row를 처리하고 dictionary links는 Go 호환 확장입니다. top-level next의 falsey 판정은 library-owned Python binary64 truthiness를 사용합니다. 따라서 `next=1e-400`은 0.0처럼 continuation 없이 끝나며, truthy numeric next는 문자열 URL이 아니므로 actual whole-page proof를 보존한 오류입니다. falsey next로 physical read/Close 오류를 숨기지 않습니다. 실제 collection path 및 정확한 relative `/v2.1/servers[/detail]` alias만 사용하며 foreign origin·collection/filter 변경·cursor loop는 오류로 처리합니다. 이 고정 route 정책은 다른 Source callback/continuation retargeting을 허용하지 않습니다. cap 또는 single page로 반환한 결과는 **관찰한 범위**에서만 유일하며 보지 않은 page의 중복을 보장하지 않습니다.

positional/constructor/control query `ignore_missing`, `name_or_id`, `session`, `cls`, `id`, `connection`, `self`, route/cache/response-key/headers/microversion controls 등은 HTTP 전에 거부합니다. 지원되는 값은 구체적 named options를 사용합니다. LIST row의 `cls`/connection/microversion/_synchronized constructor 충돌은 실제 whole-page proof를 유지한 오류입니다. typed all-projects와 raw all_tenants를 동시에 주는 것도 사전 오류입니다.

Headers는 양 phase에 적용하는 명시적 Go 옵션입니다. LIST는 Source 기본 `Accept: application/json`이 configured service Accept보다 우선하며 safe explicit Headers Accept가 있으면 그 값을 사용합니다. 인증·framing·외부 service version은 user Header로 변경하지 않습니다. native retry에서도 route/query/body/status-policy/fixed headers 및 required-absent Authorization/X-Service-Token/framing/foreign-version과 selected-version OmitHeaders를 보호합니다. 깨끗한 retry와 token reauthentication은 허용하고, 요청을 변경하는 callback은 실제 원래 rejection과 callback 원인을 함께 보존한 오류로 종료합니다. read·Close·custom cancel·일시 변경 후 복원한 captured Source 오류도 실제 accepted/rejected receipt와 한 번의 physical Close를 보존합니다.

검증은 Gophercloud 공개 `testhelper.SetupHTTP` 기반 공통 `internal/testcloud`와 기존 Nova fixtures를 재사용합니다. 신규 계약 테스트는 lookup/defaults/aliases/nested projections/discovery/paging/권한 응답/snapshot/race/native retry/physical proof를 검증합니다. fixture test, race test, 외부 main 빌드와 실제 Nova 배포의 인증·권한·endpoint 동작 검증은 별개입니다.
