# Cloud keypair: Python과 Go

`conn.Compute(ctx)`가 반환한 `compute.Service`는 keypair의 전체 목록·검색·단일 선택·생성·삭제를 제공합니다. SDK가 기본값과 기존 keypair reader, 필터, find, 응답 projector를 연결하므로 별도 builder interface를 구현할 필요가 없습니다. 버전별 native 호출과 [leaf 목록/find](keypairs-list-find.md)도 계속 사용할 수 있습니다.

| 고정 openstacksdk Cloud 호출 | Go 호출 | 결과 |
|---|---|---|
| `conn.list_keypairs(filters=None)` | `service.ListKeypairs(ctx, options...)` | 전체 목록의 `KeypairQueryResult` |
| `conn.search_keypairs(name_or_id=None, filters=None)` | `service.SearchKeypairs(ctx, nameOrID, options...)` | 선택 JSON과 record 목록 |
| `conn.get_keypair(name_or_id, filters=None, user_id=None)` | `service.GetKeypair(ctx, nameOrID, options...)` | 단일 JSON·선택적 record |
| `conn.create_keypair(name, public_key=None)` | `service.CreateKeypair(ctx, name, options...)` | owned `KeypairRecord` |
| `conn.delete_keypair(name)` | `service.DeleteKeypair(ctx, name)` | 삭제 성공 `true`, clean missing `false` |

```python
import openstack

conn = openstack.connect()
all_keys = conn.list_keypairs()
work_keys = conn.search_keypairs("work-*", filters={"type": "ssh"})
key = conn.get_keypair("workstation", user_id="owner-user-id")
created = conn.create_keypair("workstation", public_key="ssh-ed25519 ...")
deleted = conn.delete_keypair("workstation")
```

## 독립 Go main

기본 실행은 목록 조회입니다. `-operation search/get/create/delete`를 선택하며 생성·삭제를 실행하면 실제 cloud를 변경합니다. `-filters`를 생략하면 absent, `-filters null`은 None, `-filters '{}'`는 present empty dictionary입니다. `-filters '"length(@)"'`처럼 JSON 문자열을 전달하면 검색의 JMESPath 결과를 그대로 출력합니다. `-owner-user-id`는 filters가 absent/null인 get에만 적용합니다. 예제 검증은 빌드이며 실제 인증·cloud 실행은 별도입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
    operation := flag.String("operation", "list", "list, search, get, create, delete")
    name := flag.String("name", "", "keypair name or search pattern")
    filters := flag.String("filters", "", "optional filters as JSON")
    owner := flag.String("owner-user-id", "", "owner for unfiltered get")
    publicKey := flag.String("public-key", "", "optional public key for create")
    flag.Parse()
    filtersSet := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "filters" { filtersSet = true }
    })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *operation, *name, *filters, filtersSet, *owner, *publicKey); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, operation, name, filters string, filtersSet bool, owner, publicKey string) error {
    conn, err := sdk.Connect(ctx)
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    options := []compute.KeypairQueryOption{compute.WithKeypairQueryUserID(owner)}
    if filtersSet {
        options = append(options, compute.WithKeypairQueryFilters(json.RawMessage(filters)))
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    printResult := func(value any, callErr error) error {
        return errors.Join(callErr, encoder.Encode(value))
    }
    switch operation {
    case "list":
        result, err := service.ListKeypairs(ctx, options...)
        return printResult(result, err)
    case "search":
        result, err := service.SearchKeypairs(ctx, name, options...)
        return printResult(result, err)
    case "get":
        result, err := service.GetKeypair(ctx, name, options...)
        return printResult(result, err)
    case "create":
        result, err := service.CreateKeypair(ctx, name, compute.WithCloudKeypairPublicKey(publicKey))
        return printResult(result, err)
    case "delete":
        deleted, err := service.DeleteKeypair(ctx, name)
        return printResult(map[string]bool{"deleted": deleted}, err)
    default:
        return fmt.Errorf("unknown operation %q", operation)
    }
}
```

## SDK가 처리하는 순서

`ListKeypairs`는 lazy iterator인 leaf `ListRecords`를 끝까지 읽습니다. `Keypairs`와 `Value`는 전체 목록을 성공적으로 읽은 경우에만 채웁니다. `Inventory`는 leaf의 첫 단계 선언 필터를 통과해 실제 yielded된 record이며, 늦은 페이지 오류가 발생하면 그때까지 읽은 부분 결과를 유지합니다. 필터에 빠진 모든 raw 행을 Inventory로 주장하지 않습니다. 배열 순서와 중복은 그대로 유지합니다.

`SearchKeypairs`의 dictionary는 **두 번 적용**합니다. 먼저 같은 dictionary를 leaf에 전달해 `user_id`·`limit`·`marker` query와 선언된 로컬 속성을 분류합니다. 전체 inventory를 받은 뒤 동일한 ordered dictionary를 Cloud Resource에 다시 비교합니다. 따라서 서버가 다른 `user_id`의 행을 반환하면 두 번째 단계에서 제외됩니다. leaf에서 버린 unknown 속성도 두 번째 단계의 일치 후보에 도달하면 오류가 될 수 있고, 앞선 속성이 불일치하면 뒤 속성은 평가하지 않습니다. `limit` 같은 목록 controls를 두 번째 단계에서 자동으로 지우지 않습니다.

이름/ID는 공통 exact/glob matcher를 쓰며 source처럼 raw 숫자·null identity의 문자열 변환도 처리합니다. JSON 문자열 filters는 공통 JMESPath engine으로 평가합니다. expression은 배열뿐 아니라 scalar·object·null을 반환할 수 있으므로 `Value`를 읽으며, 대응이 없는 결과에 가짜 KeypairRecord를 만들지 않습니다. 복잡한 Python 객체·arbitrary session kwargs·CloudRegion loader 전체 지원을 의미하지 않습니다.

`GetKeypair`는 filters absent/null일 때만 leaf `FindKeypair`를 사용하고 `UserID`를 전달합니다. `{}`·`""`를 포함한 **present filters는 search**를 사용하며 별도 `UserID`는 전달하지 않습니다. 검색은 전체 목록을 읽은 뒤 공통 Python truthiness·length·첫 항목 선택을 적용합니다. 중복은 `resource.ErrAmbiguous`, 성공한 빈 조회는 nil Value/record입니다. JMESPath가 선택한 임의 값은 Value에 유지합니다. 직접 find의 fallback·missing·버전 정책은 [leaf 설명](keypairs-list-find.md)을 따릅니다.

## 옵션·Resource·오류

`KeypairQueryOpts`는 `Filters *json.RawMessage`, `UserID`, `Microversion *string`, `MaxItems`, `Paginated *bool`을 갖습니다. `WithKeypairQueryOptions`와 개별 Filters/Expression/UserID/Microversion/MaxItems/Paginated/Header helper를 사용할 수 있습니다. pointer·raw JSON·header는 SDK가 소유하고 사용자 option callback은 한 번 적용합니다. context가 timeout을 소유하며 SDK가 별도 deadline을 넣지 않습니다. dictionary의 `paginated`, `max_items`, `microversion`, `headers`는 기존 leaf의 dedicated controls로 전달합니다. raw query·fields와 invalid UTF-8/JSON·음수 cap은 거부합니다. 임의 `base_path`·deprecated `jmespath_filters`·`allow_unknown_params`의 source 전체 동작은 이 고정 경로에서 지원하지 않습니다.

Cloud Resource는 leaf의9개 속성에 current Connection `location`을 더한 view입니다. 기존 `CloudLocation` adapter를 옵션 전에 한 번 snapshot하며 row별 bytes를 소유합니다. direct client에 adapter가 없으면 location은 null입니다. 이 location은 현재 연결의 기록된 cloud/region/project이며 keypair `user_id`를 project로 바꾸지 않습니다. Wire·Envelope는 실제 서버 응답이며 response에 있는 location도 원문 그대로 유지합니다. 완전한 Python mutable Resource·dirty tracking·session 동작은 별도 범위입니다.

`CreateKeypair`는 name을 항상 보내고 nonempty public key만 포함합니다. 빈 public key는 생략하며 nullable full attrs가 필요하면 versioned `CreateKeypair`를 사용합니다. 조회·대기·cache 작업을 추가하지 않고 기존 생성 projector와 location adapter를 사용합니다. 이미 선택된 client 버전은 보존합니다.

`DeleteKeypair`는 strict leaf DELETE 한 번의 성공에 true를 반환합니다. 같은 실제 DELETE의 clean404만 false이며 조회·find·대기와 location getter를 호출하지 않습니다. 다른 HTTP 오류·accepted response의 read/Close 오류·nested404·source/context 변경은 오류로 반환합니다. Source binding·옵션·응답 검증과 partial receipt는 기존 공통 guard/reader를 재사용하며 body를 실패 후 재조회하지 않습니다.

비교 기준은 고정 Python `_compute.py`의 list/search/get/create/delete와 관련 공통 filter입니다. [검증 전략](../docs/testing.md)과 [지원 검토 기록](../docs/sdk-support-ledger.md)에 source별 named 판정 및 실제 테스트 근거를 남깁니다.
