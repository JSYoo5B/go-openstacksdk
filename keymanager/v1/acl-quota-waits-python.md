# Barbican ACL 및 effective Quota 대기

`service.Waits`는 SecretACL과 현재 프로젝트의 effective Quota를 SDK가 선택해 조회합니다. 앱에서 adapter나 builder를 구현하지 않아도 됩니다. 비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud v2.15.0, stock 정책은 Barbican 21.0.0입니다.

| 대상 | Go 생성자 | 고정 GET route | 선택할 수 있는 선언 속성 |
| --- | --- | --- | --- |
| Secret ACL | `waits.SecretACL(secretID)` | `/v1/secrets/{secretID}/acl` | `id`, `name`, `read`, `acl_ref`, URI `secret_id`, computed `location` |
| 기존 ACL 상태 | `waits.ForSecretACL(acl)` | 기존 `acl.SecretID`의 ACL | 위와 동일 |
| 현재 프로젝트 quota | `waits.Quota()` | `/v1/quotas` | `id`, `name`, `secrets`, `orders`, `containers`, `consumers`, `cas`, computed `location` |
| 기존 effective quota | `waits.ForQuota(quota)` | `/v1/quotas` | 위와 동일 |

두 Source 모델은 **status descriptor가 없습니다.** `WaitForStatus`에는 `WithAttribute`를 지정해야 하며 기본 `status`는 HTTP 전에 `ErrUnsupported`입니다. 선택한 값은 Source의 `.lower()` 비교가 가능한 문자열 또는 null이어야 합니다. 일반 quota의 숫자를 상태 문자열로 변환하지 않습니다. 숫자나 ACL의 read dict를 선택하면 `ErrInvalidOption`이고, HTTP 후 확인한 값이면 실제 응답 proof도 반환합니다. null 목표는 Go에서 `nil`이며 캐시의 null과 맞으면 HTTP 없이 끝납니다.

## 고정 Source의 singleton fetch 문제

Python public `get_secret_acl`과 `get_quota`는 `_get(..., requires_id=False)`를 전달하여 정상 route를 조회합니다. 그러나 generic wait는 `res.fetch(..., skip_cache=True)`를 호출하고, 고정 `Resource.fetch`의 매개변수 기본 `requires_id=True`가 적용됩니다. SecretACL 클래스의 `requires_id=False`도 이 매개변수로 덮입니다. 따라서 getter로 얻은 `id=None` 객체는 polling 직전 `InvalidRequest`가 발생하며, id를 넣으면 `/acl/{id}` 또는 `/quotas/{id}`가 됩니다. cached 값이 이미 목표와 일치하는 경우에는 fetch 전에 반환됩니다. [고정 fetch 및 request 준비](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1275-L1336), [wait의 fetch 호출](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715).

Go의 concrete target은 정상 singleton route를 SDK가 소유합니다. 이는 고정 Source의 잘못된 request 준비를 개선한 동작입니다. 응답 `id`·`acl_ref`와 외부 URL은 다음 poll의 route를 바꾸지 않습니다. Quota의 `RequestID == "current-project"`는 내부 wait selector이며 실제 project/member ID가 아닙니다. `/quotas`에 붙이지 않고 `Resource.id`에도 합성하지 않습니다.

## 독립 Python 프로그램

아래 프로그램은 기본으로 고정 Source 자체의 동작을 실행합니다. `--correct-singleton-fetch`는 위 request 준비 문제만 고친 명시적 Python subclass를 선택합니다. Go는 해당 workaround를 앱에 요구하지 않습니다. `python acl_quota_wait.py --kind acl --secret-id UUID --attribute acl_ref --state URL` 또는 `--operation delete`로 실행합니다. 현재 stock effective quota에는 상태 문자열이 없으므로 quota 문자열 대기는 서비스가 해당 선언 값을 실제로 반환할 때에만 유용합니다.

```python
import argparse
import openstack
from openstack.key_manager.v1.secret_acl import SecretACL
from openstack.key_manager.v1.quota import Quota

class PollableSecretACL(SecretACL):
    def fetch(self, session, **kwargs):
        return super().fetch(session, requires_id=False, **kwargs)

class PollableQuota(Quota):
    def fetch(self, session, **kwargs):
        return super().fetch(session, requires_id=False, **kwargs)

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=["acl", "quota"], default="acl")
parser.add_argument("--secret-id")
parser.add_argument("--operation", choices=["status", "delete"], default="status")
parser.add_argument("--attribute", default="acl_ref")
parser.add_argument("--state")
parser.add_argument("--null-state", action="store_true")
parser.add_argument("--interval", type=float, default=2)
parser.add_argument("--timeout", type=float)
parser.add_argument("--correct-singleton-fetch", action="store_true")
args = parser.parse_args()
if args.kind == "acl" and not args.secret_id:
    parser.error("--secret-id is required for ACL")
if args.operation == "status" and args.state is None and not args.null_state:
    parser.error("--state or --null-state is required")
conn = openstack.connect(cloud=args.cloud)
if args.kind == "acl":
    cls = PollableSecretACL if args.correct_singleton_fetch else SecretACL
    res = cls.existing(secret_id=args.secret_id)
else:
    cls = PollableQuota if args.correct_singleton_fetch else Quota
    res = cls.existing()
options = {"interval": args.interval, "callback": lambda progress: print("progress", progress)}
if args.timeout is not None:
    options["wait"] = args.timeout
if args.operation == "delete":
    result = conn.key_manager.wait_for_delete(res, **options)
else:
    state = None if args.null_state else args.state
    result = conn.key_manager.wait_for_status(res, state, attribute=args.attribute, **options)
print(result.to_dict())
```

## 독립 Go main

[설치 안내](../../docs/install.md)에 따라 module을 준비합니다. ACL 문자열 대기는 `go run . -kind acl -secret-id UUID -attribute acl_ref -state URL`, ACL 삭제 관찰은 `-operation delete`로 실행합니다. `-kind quota -attribute orders -null-state`는 기본 null 캐시를 비교하고 HTTP를 보내지 않습니다. quota의 현재 숫자를 조회하려면 `service.Quotas.Get(ctx)`를 사용합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    kind := flag.String("kind", "acl", "acl/quota")
    secretID := flag.String("secret-id", "", "literal secret ID for ACL")
    operation := flag.String("operation", "status", "status/delete")
    attribute := flag.String("attribute", "acl_ref", "declared string/null attribute")
    state := flag.String("state", "", "target string; empty string is explicit")
    nullState := flag.Bool("null-state", false, "target is Source None")
    interval := flag.Duration("interval", 2*time.Second, "polling interval")
    timeout := flag.Duration("timeout", 0, "nonzero override; zero keeps Source default")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { log.Fatal(err) }
    var target waits.Target
    switch *kind {
    case "acl": target = waits.SecretACL(*secretID)
    case "quota": target = waits.Quota()
    default: log.Fatal("unknown kind")
    }
    options := []waits.Option{
        waits.WithInterval(*interval),
        waits.WithProgressCallback(func(progress int) error {
            fmt.Println("progress", progress)
            return nil
        }),
    }
    if *timeout != 0 { options = append(options, waits.WithTimeout(*timeout)) }
    var result *waits.Record
    switch *operation {
    case "delete": result, err = service.Waits.WaitForDelete(ctx, target, options...)
    case "status":
        options = append(options, waits.WithAttribute(*attribute))
        selected := state
        if *nullState { selected = nil }
        result, err = service.Waits.WaitForStatus(ctx, target, selected, options...)
    default: log.Fatal("unknown operation")
    }
    if err != nil { log.Fatal(err) }
    encoded, err := json.MarshalIndent(result.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}
```

## 캐시·descriptor·실제 응답

`ForSecretACL`은 native `SecretID`를 복사하여 parent route를 고정합니다. `Metadata.Body`가 있으면 이 원본의 선언 속성만 projection하고 typed convenience 필드는 대신 사용하지 않습니다. ACL commit의 HTTP 없는 결과나 malformed 응답처럼 Body가 없으면 실제 `Read`·`ACLRef` 필드로 캐시를 만들며 `Wire`·`Envelope`를 합성하지 않습니다. `ForQuota`는 실제 `Metadata.Body`를 요구합니다. 두 생성자는 원본을 변경하지 않고 독립적으로 snapshot합니다. native 타입에 바이트 Envelope가 없으므로 캐시 성공의 `Envelope`도 nil입니다.

ACL의 `read`는 고정 Source의 dict descriptor입니다. object와 null을 유지하며 false·0·문자열·list는 Source처럼 `{}`로 읽습니다. `Wire`는 원래 raw 값으로 보존합니다. 기존 native `secretacls.Get/Set/Update`의 stock object 검증은 별도 계약이며 이 wait projector는 Source getter coercion을 따릅니다. ACL의 flat object와 quota의 `quotas` envelope를 처리하고, quota에 envelope가 없으면 Source처럼 root object를 사용합니다. 실제 응답에 존재한 null·false·0·큰 숫자만 이전 선언 값을 갱신합니다. unknown 확장은 `Wire`에만 남습니다. Source generic Resource와 같이 malformed/empty JSON은 이전 synchronized 필드를 유지하지만 valid JSON의 null/nonobject 또는 nonobject quota envelope는 실제 응답 오류입니다. Go는 UTF-8 및 완전한 JSON을 검증하고 computed location을 null로 소유합니다.

`Resource`는 누적된 선언 값이고 `Wire`는 마지막 응답의 object, `Envelope`는 마지막 실제 바이트입니다. 각각 독립 복사하며 Header·StatusCode도 실제 관찰만 기록합니다. 오류 반환 Record의 `Resource`는 nil입니다. Read·Close·source 변경·취소의 복합 오류가 401/404/429/503과 같이 발생해도 retry/reauth를 호출하거나 삭제 성공으로 바꾸지 않습니다. native 401의 `ErrUnableToReauthenticate.ErrOriginal`에 실제 method/URL/status/body/header를 유지하고 `ErrReauth`에 fault 원인을 유지합니다. clean 401/503의 정상 reauth/retry는 기존 reader 정책을 유지합니다.

## 삭제 관찰과 권한

삭제 대기는 DELETE를 보내지 않습니다. status descriptor가 없으므로 성공한 GET에 `status:"deleted"`라는 unknown 확장이 있어도 완료로 판단하지 않습니다. clean GET404가 완료를 뜻하며 원래 snapshot Resource와 실제404 증거를 반환합니다. 성공한 GET이 계속되면 callback(0)과 다음 poll을 반복합니다. 기본 delete timeout120초·status 무제한·interval2초, nil/zero/음수·실패 상태와 callback 정책은 [공통 wait 비교](waits-python.md)를 따릅니다.

stock ACL을 삭제하면 GET은 implicit default ACL을 반환합니다. 따라서 ACL 삭제 요청 성공만으로 이 waiter가 끝날 것이라고 기대하면 안 됩니다. 현재 프로젝트 quota도 project override 삭제 뒤 effective defaults를 계속 반환합니다. 이 waiter는 GET 대상의 부재를 관찰하며 관리 설정 삭제의 확인을 대신하지 않습니다. [Barbican ACL controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/acls.py#L153-L207), [effective quota controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/quotas.py#L47-L59).

ACL GET은 `secret_acls:get`, effective quota GET은 `quotas:get`를 따릅니다. 새 secure RBAC의 ACL은 같은 프로젝트 admin 또는 owner/nonprivate 조건의 member, quota는 project reader를 허용합니다. deprecated ACL 조건은 `all_but_audit`와 secret project match, quota는 `all_users`입니다. Barbican은 runtime에서 `enforce_new_defaults`와 `enforce_scope`를 false로 override하므로 실제 배포 정책이 최종 권한을 결정합니다. [Barbican 2025.2 정책](https://docs.openstack.org/barbican/2025.2/configuration/policy.html), [고정 ACL 정책](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/acls.py#L20-L88), [고정 quota 정책](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/quotas.py#L18-L74).

`ProjectQuota`는 `project_quotas:get` admin 정책의 별도 `/project-quotas/{projectID}` 모델입니다. 다음 핵심 admin 단계에서 concrete target을 추가하며 이번 user 묶음에 구현하지 않았습니다. 추가 시 고정 projectID·`project_quotas` envelope·같은 7개 nullable Body·status 없는 삭제·실제404/role거부·캐시 proof를 검증해야 합니다. 임의 Resource subclass와 mutable identity/location/session/cache는 SDK-R1/C1/S1에서 계속 추적합니다. **두 named generic wait API는 아직 unresolved**이며 이번 concrete 대상 두 개를 API 완료 개수로 올리지 않습니다. 실클라우드와 Python runtime 실행은 별도 검증입니다.
