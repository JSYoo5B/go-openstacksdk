# Neutron 필드 선택 조회와 revision 조건부 삭제

Python `conn.network.get_network(id, fields=[...])`처럼 조회할 필드를 고르면
SDK가 Python Body 속성 이름을 실제 wire 필드로 바꿉니다. Go에서는
`service.API.Networks.GetWithFields(ctx, id, "id", "is_admin_state_up")`를 사용합니다.
기본 `Get(ctx, id)`는 기존 native 호출이며, 새 `GetWithFields`에 필드를 주지 않으면
선택 query를 보내지 않습니다.

| Connection Network의 API | Python 조회 | 조건부 삭제 |
|---|---|---|
| `Networks` | `get_network` | `delete_network` |
| `Subnets` | `get_subnet` | `delete_subnet` |
| `Ports` | `get_port` | `delete_port` |
| `FloatingIPs` | `get_ip` | `delete_ip` |
| `Routers` | `get_router` | `delete_router` |
| `SecurityGroups` | `get_security_group` | `delete_security_group` |
| `SecurityRules` | `get_security_group_rule` | `delete_security_group_rule` |
| `SubnetPools` | `get_subnet_pool` | 기존 `Remove` 사용 |
| `Trunks` | `get_trunk` | 기존 `Remove` 사용 |
| `QoSPolicies` | `get_qos_policy` | 기존 `Remove` 사용 |
| `SecurityAddressGroups` | `get_address_group` | 기존 `Remove` 사용 |

`fields`는 고정 openstacksdk Resource 상속 순서의 **Body descriptor**로 변환합니다.
예를 들어 `is_admin_state_up`은 `admin_state_up`, Network의 `subnet_ids`는 `subnets`,
Router의 `revision_number`는 `revision`이 됩니다. 알려지지 않은 이름은 그대로 보내고
순서와 중복을 보존합니다. `any_tags`처럼 목록 query에서만 쓰는 별칭은 변환하지 않습니다.
서버에 ID 필드를 자동 추가하지 않으므로 `id`를 요청하지 않은 sparse 응답에서는
Go 모델의 ID가 비어 있을 수 있습니다. 인수와 별칭 map은 HTTP callback 전에 복사합니다.
생성기는 기존 AST resolver로 17개 고정 Source 파일과 11종의 유효 Body 필드 매핑 208개을 검증합니다.

Python `delete_network(id, if_revision=0)`는 Go의
`DeleteWithOptions(ctx, id, networks.WithDeleteRevision(0))`에 해당합니다.
revision을 생략하면 SDK가 조건을 추가하지 않고, 명시한 0은
`If-Match: revision_number=0`을 보냅니다. 기존 설정된 header를 덮어쓰더라도
공유 client 설정은 변경하지 않습니다. revision은 Go `int64` 범위입니다.

`DeleteWithOptions`는 Python proxy처럼 **404를 기본으로 무시**합니다.
`WithDeleteIgnoreMissing(false)`를 주면 404도 오류입니다. 412와 transport·취소 오류는
항상 반환합니다. 삭제는 공통 guarded HTTP 엔진을 재사용해 실제 404에 동반된
본문 read/close 실패도 보존하고, 접수된 응답의 처리 실패를 미존재로 무시하지 않습니다.
응답 본문을 JSON으로 decode하지 않습니다. 나중 option이 앞선 option을
덮어쓰고, `WithDeleteOptions(DeleteOpts{...})`는 생성 시 값을 복사합니다.
옵션 적용 결과도 HTTP 전에 별도로 복사하므로 custom option이 보관한 포인터를
이후 바꾸어도 현재 요청의 조건과 missing 정책이 달라지지 않습니다.

새 SDK 조회·삭제 프로필은 API 성공 응답 200..399를 허용합니다.
조회 결과는 기존 Gophercloud typed 모델과 decoder를 사용합니다. SubnetPool 선택 조회에서는
누락된 세 prefix-length 필드만 typed zero로 처리하며, 명시한 null과 잘못된 타입은
native 오류로 유지합니다. 기존 native Get은 그대로입니다. 잘못된 JSON이나
알려진 필드 타입은 오류이고, Python mutable Resource의 descriptor 기본값·임의 확장값
보존을 제공하는 API는 아닙니다. 입력은 명시적 string ID이며 Python Resource 객체는
받지 않습니다. 기존 native `Get/Delete`의 strict status와 결과는 그대로 유지합니다.
에러에는 `resource.OperationError`의 연산·리소스 문맥을 추가합니다.

## 전체 Connection 사용 예제

아래 main은 선택 조회를 실행합니다. 삭제는 `DELETE_NETWORK_ID`가 명시된 경우에만
실행하며 별도 revision 값은 `DELETE_NETWORK_REVISION`으로 설정합니다.

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    selected, err := service.API.Networks.GetWithFields(ctx, os.Getenv("NETWORK_ID"),
        "id", "name", "is_admin_state_up", "revision_number")
    if err != nil { panic(err) }
    fmt.Printf("network: %s %s revision=%d\n", selected.ID, selected.Name, selected.RevisionNumber)

    if id := os.Getenv("DELETE_NETWORK_ID"); id != "" {
        var options []networks.DeleteOption
        if value := os.Getenv("DELETE_NETWORK_REVISION"); value != "" {
            revision, err := strconv.ParseInt(value, 10, 64)
            if err != nil { panic(err) }
            options = append(options, networks.WithDeleteRevision(revision))
        }
        if err := service.API.Networks.DeleteWithOptions(ctx, id, options...); err != nil { panic(err) }
    }
}
```

같은 API는 `network/v2.New(client)` 또는 각 leaf의 `New(client)`에서도 사용합니다.
Python의 대응 호출은 `conn.network.get_network(id, fields=["id", "is_admin_state_up"])`와
`conn.network.delete_network(id, if_revision=0, ignore_missing=True)`입니다.

[기존 native CRUD](native-crud.md)와 [네트워크 SDK 전체 안내](../README.md)를 함께 참고합니다.
검증은 기존 public Gophercloud testhelper를 사용하는 로컬 fixture와 문서 main 빌드이며,
실제 OpenStack 인증·실행은 별도 환경에서 수행해야 합니다.
