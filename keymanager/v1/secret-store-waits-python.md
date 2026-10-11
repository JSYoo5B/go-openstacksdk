# Barbican SecretStore 대기: Python과 Go

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, stock 서버는 Barbican 21.0.0입니다. `waits.SecretStore(id)`는 기존 Container·Order·Secret 대기에 SecretStore라는 네 번째 concrete 대상을 추가합니다. `wait_for_status`와 `wait_for_delete` 전체 선언은 SecretACL·Quota·ProjectQuota 대상의 실제 fetch 계약이 남아 **unresolved**이며, 이 작업의 완료 API 증가량은 0입니다.

| Python | Go | 경계 |
|---|---|---|
| `conn.key_manager.wait_for_status(SecretStore(id=...), "ACTIVE")` | `api.WaitForStatus(ctx, waits.SecretStore(id), &status)` | 고정 member GET, 기본 status/ERROR/interval2초, timeout 없음 |
| `conn.key_manager.wait_for_delete(resource)` | `api.WaitForDelete(ctx, waits.SecretStore(id))` | GET만 polling, 기본 120초, clean metadata404 또는 fetched deleted |
| 이미 읽은 backend Resource | `waits.ForSecretStore(id, nativeValue)` | 요청 id를 명시하고 native Metadata.Body를 snapshot |

`GET /v1/secret-stores/{id}`의 응답은 singular envelope 없이 flat object입니다. Source SecretStore는 allow_fetch=true, allow_delete=false이며 삭제 대기는 delete 호출이 아닙니다. Source allow_fetch를 가진 finite 대상이므로 polling할 수 있습니다. payload를 추가 조회하지 않습니다. [Source schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/secret_store.py)와 [stock controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/secretstores.py#L102-L156)를 기준으로 구현합니다.

Resource는 id/name/status/created_at/updated_at/secret_store_ref/secret_store_id/global_default/crypto_plugin/secret_store_plugin 및 owned-null location을 제공합니다. missing descriptor는 null이며, global_default만 Python bool 규칙을 적용합니다. null/false/0/빈 container·decimal float underflow는 구분하고 숫자 원문은 Wire에 보존합니다. dates/plugins/status/name/ref는 untyped입니다. secret_store_id는 같은 ref의 passive HREF formatter 결과이며 percent spelling을 유지합니다. HTTP body에 canonical/wire 또는 ref/formatter 별칭이 겹치면 실제 JSON 행의 입력 순서에 따라 마지막 값이 이깁니다. native Metadata.Body map에는 원래 object 입력 순서가 없으므로 cached constructor는 key를 정렬한 순서로 적용하는 명시적인 Go 정책입니다.

Resource의 id는 실제 id presence(null/false/zero 포함)가 우선합니다. 직접 Target의 누락 id는 명시한 request id로 seed하지만 요청 경로는 항상 immutable constructor id입니다. returned id/ref/convenience native ID가 다음 polling 주소를 바꾸지 않습니다. Source의 mutable Resource.id 또는 full alternate ref를 조합해 요청하는 동작은 고정 route 정책과 다릅니다. unknown/self/location과 원문 숫자 및 actual envelope/header/status는 Wire·Envelope에 보존하며 Resource의 known schema에는 unknown을 추가하지 않습니다. computed Connection/location/runtime subclass·mutable Resource/cache는 기존 SDK-R1 경계입니다.

동기화된 view는 실제 response에서 존재한 known 값만 이전 값에 덮어씁니다. omission은 이전 값 유지, null은 실제 clear입니다. generic Source fetch처럼 empty/malformed JSON은 이전 known state를 유지하고 실제 새 response proof를 보존합니다. valid JSON의 null/list/wrong shape·invalid UTF8·HREF formatter 오류·선택한 상태의 nonstring 값은 actual accepted response를 가진 오류입니다. status cached match는 0HTTP이고 native cached seed에 원래 byte envelope가 없으므로 JSON map을 재직렬화해서 Envelope를 만들지 않습니다.

기존 `reader`·captured Source·header/version guard·PreparedWaitPolicy·response proof·public Gophercloud testhelper fixture를 재사용합니다. selected key-manager version 또는 부재, explicit Header, live auth token을 유지하며 임의 version discovery/상한을 추가하지 않습니다. Source generic HTTP100..399 admission과400..599 rejection을 보존합니다. physical Read/Close/source/cancel 오류는 한 HTTP·한 Close로 끝나고 native retry/reauth callback을 호출하지 않습니다. 401의 native `*ErrUnableToReauthenticate`는 Unwrap이 없어 ErrOriginal의 실제 rejection과 ErrReauth의 mixed physical causes를 따로 확인합니다. clean native retry/reauth/token renewal은 유지합니다.

[Barbican 21 stock policy](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/secretstores.py#L126-L138)는 member `secretstore:get`에 new-default project reader를 적용하고 deprecated rule은 all_users입니다. [2025.2 policy configuration](https://docs.openstack.org/barbican/2025.2/configuration/policy.html)은 runtime enforce_new_defaults/enforce_scope가 False라는 점을 명시하므로 실제 배포 설정·override가 권한을 결정합니다. SDK가 권한을 추정하거나 확대하지 않습니다.

stock controller는 multiple backend 기능이 꺼져도 404를 반환합니다. 따라서 Source와 같은 clean404 삭제 대기 종료는 **backend가 실제 삭제되었다는 증거가 아닙니다**. status 대기의404는 오류로 유지합니다. 정상 backend 목록의 status는 보통 ACTIVE이며 SDK는 backend lifecycle 변경 API나 fake 상태 변화를 추가하지 않습니다.

아래 독립 Python 예제는 explicit literal identity로 generic Source Resource를 만듭니다.

```python
import argparse
import json
import os
import openstack
from openstack.key_manager.v1.secret_store import SecretStore


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("id")
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--status", default="ACTIVE")
    parser.add_argument("--delete", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    target = SecretStore(id=args.id)
    if args.delete:
        value = conn.key_manager.wait_for_delete(target)
    else:
        value = conn.key_manager.wait_for_status(target, args.status)
    print(json.dumps(value.to_dict(), ensure_ascii=False))


if __name__ == "__main__":
    main()
```

다음 Go main은 같은 기본값을 사용합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    id := flag.String("id", "", "literal SecretStore identity")
    status := flag.String("status", "ACTIVE", "desired state")
    deleting := flag.Bool("delete", false, "poll GET for absence/deleted")
    flag.Parse()
    if err := run(*cloud, *id, *status, *deleting); err != nil { log.Fatal(err) }
}

func run(cloud, id, status string, deleting bool) error {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { return err }
    api := waits.New(service.RawClient())
    target := waits.SecretStore(id)
    var value *waits.Record
    if deleting { value, err = api.WaitForDelete(ctx, target) } else { value, err = api.WaitForStatus(ctx, target, &status) }
    if err != nil { return err }
    encoded, err := json.MarshalIndent(value.Resource.Body, "", "  ")
    if err != nil { return err }
    fmt.Println(string(encoded))
    return nil
}
```

새 9개 테스트 그룹은 raw descriptor/default, alias 순서/merge, Source bool truthiness, cached snapshot/0HTTP/no fabricated Envelope, shape/formatter/error receipt, clean404/disabled feature, policy/callback/preflight,10 mixedphysical401/404/429/503 lanes와 positive retry/reauth을 검증합니다. root가 3개 private dispatch hook 및 full package race/standalone compile을 통합합니다.
