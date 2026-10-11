# RBAC Policy 변경의 Python/Go 사용법

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. leaf는 `network/v2/extensions/rbacpolicies`이고 `network.RBACPolicies`로 접근합니다. 기존 Get/Find/List 지원 수와 중복 집계하지 않는 세 연산입니다.

| Python | Go |
| --- | --- |
| `create_rbac_policy(**attrs)` | `CreateRecord(ctx, options...)` |
| `update_rbac_policy(policy_id, **attrs)` | `UpdateRecord(ctx, policyID, options...)` |
| `delete_rbac_policy(policy_id, ignore_missing=True)` | `DeleteRecord(ctx, policyID, options...)` |

Python의 kwargs를 Go가 소유하는 concrete 옵션으로 표현합니다. 사용자 interface builder나 decoder 구현이 필요하지 않습니다. `WithRecordCreateField`/`WithRecordUpdateField` 또는 `RecordCreateOpts`/`RecordUpdateOpts`의 `Attrs map[string]json.RawMessage`로 unknown extension을 포함한 JSON 속성을 전달합니다. 문자열뿐 아니라 null/false/zero/배열/객체를 그대로 사용할 수 있습니다.

아래 Python은 독립 실행 예제입니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--mode", choices=("create", "update", "delete"), required=True)
parser.add_argument("--policy", default="")
parser.add_argument("--object", default="")
parser.add_argument("--target-project", default="")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        value = conn.network.create_rbac_policy(
            object_type="network", object_id=args.object,
            action="access_as_shared", target_project_id=args.target_project,
        )
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "update":
        value = conn.network.update_rbac_policy(
            args.policy, target_project_id=args.target_project
        )
        print(json.dumps(value.to_dict(), indent=2))
    else:
        conn.network.delete_rbac_policy(args.policy)
        print(json.dumps({"deleted": args.policy}, indent=2))
```

아래 Go는 독립 `main.go`이며 [설치 안내](../../docs/install.md)에 따라 module 안에서 실행합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
)

func run(ctx context.Context, cloud, mode, policyID, objectID, target string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "create":
        value, err := network.RBACPolicies.CreateRecord(ctx,
            rbacpolicies.WithRecordCreateField("object_type", "network"),
            rbacpolicies.WithRecordCreateField("object_id", objectID),
            rbacpolicies.WithRecordCreateField("action", "access_as_shared"),
            rbacpolicies.WithRecordCreateField("target_project_id", target),
        )
        if err != nil { return nil, err }
        return value.Resource, nil
    case "update":
        value, err := network.RBACPolicies.UpdateRecord(ctx, policyID,
            rbacpolicies.WithRecordUpdateField("target_project_id", target),
        )
        if err != nil { return nil, err }
        return value.Resource, nil
    case "delete":
        return network.RBACPolicies.DeleteRecord(ctx, policyID)
    default:
        return nil, fmt.Errorf("mode must be create, update or delete")
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    mode := flag.String("mode", "", "create, update or delete")
    policyID := flag.String("policy", "", "literal RBAC policy ID")
    objectID := flag.String("object", "", "network ID for create")
    target := flag.String("target-project", "", "concrete target project ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *policyID, *objectID, *target)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## 요청과 기본값

Source `RBACPolicy`는 `Resource`를 직접 상속합니다. Body는 `object_id`, `target_project_id`(wire `target_tenant`), `project_id`(read alias `tenant_id`), deprecated `tenant_id`, `object_type`, `action`, inherited `id`, `name`이고 모두 untyped/default null입니다. TagMixin/NetworkResource/Header descriptor를 추가하지 않습니다.

Create는 제공한 dirty 속성만 `{"rbac_policy": {...}}`로 POST합니다. 누락한 null 기본값을 전송하지 않으며, 빈 Create도 `{"rbac_policy": {}}`를 전송합니다. required/enum/native 정수 validator는 Source에 없으므로 Gophercloud native builder의 required-field 검사를 적용하지 않습니다. 서버 스키마 위반은 실제 HTTP 오류로 반환합니다. Source의 unknown Body 속성도 전송합니다.

Update는 literal ID에 대해 제공한 dirty 속성만 PUT하고 Body에서 identity를 제외합니다. 빈 속성이나 constructor-only `microversion` 속성은 Source `requires_commit=False`처럼 HTTP 없이 seed/default view를 반환합니다. 이 경우 `StatusCode=0`, `Wire=nil`, receipt 없음입니다. 안전한 Go Header/version 옵션만 전달해도 빈 Update를 강제로 전송하지 않습니다. `WithRecordUpdateField("target_project_id", nil)`은 변경으로 취급하여 explicit null을 전송합니다. `Attrs`의 nil RawMessage도 explicit null입니다.

순서가 있는 With 함수에서 canonical `target_project_id`와 wire `target_tenant`는 constructor kwargs 순서대로 소비하여 마지막 alias가 이깁니다. 같은 key의 재대입은 기존 key 위치를 유지합니다. Bulk map은 Go에 삽입 순서가 없으므로 key를 사전순으로 정렬합니다. `project_id`의 `tenant_id`는 읽기 fallback alias이며 요청 key를 임의로 `project_id`로 복제하지 않습니다.

`if_match`, `if_none_match`, `headers` 같은 unknown attrs는 Body 속성입니다. 별도 `WithRecordCreateHeaders`/`WithRecordUpdateHeaders`는 실제 HTTP Header를 설정하는 Go 편의이며 Source의 Body/header 혼합을 만들지 않습니다. null attrs와 HTTP If-None-Match 조건은 서로 독립적입니다.

member ID는 nonblank UTF-8 literal 한 URL segment로 escape합니다. slash/percent/space를 ID의 일부로 취급하고 parent lookup/GET/Find를 추가하지 않습니다. 빈 값/dot/control/invalid UTF-8은 HTTP 전 사전 오류입니다. `id`/`rbac_policy` Update 속성의 positional collision과 `base_path`/`cls`/`connection`/`_synchronized`/`self`/Resource 내부 constructor·route controls는 사전 오류입니다. mutable Resource/session/route 교체, deprecated `__conflicting_attrs` 복구와 class/method shadowing은 이 immutable JSON profile 밖이며 별도로 추적합니다.

## 응답과 오류의 증거

Create/Update는 기존 `RBACPolicyRecord`를 반환합니다. `Resource`는 public aliases/defaults와 constructor seed를 포함한 Source view, `Wire`는 actual row, `Envelope`은 actual whole body입니다. `Header`/`StatusCode`도 실제 관찰값이며 서로 독립 snapshot입니다. sparse response는 seed를 유지하고 실제 필드로 덮어씁니다. accepted bodyless/JSON parse 실패 응답은 Source처럼 seed/default view와 nil Wire를 유지합니다. 유효 JSON의 잘못된 shape 또는 invalid UTF-8은 actual response proof를 가진 오류입니다.

`location`은 SDK 소유 computed 값으로 null이며 서버의 forged location은 Wire/Envelope에 원문으로 남습니다. member unknown `microversion`은 actual operation-selected version/null로 Resource에 표시하고 raw value는 Wire에 남깁니다. Python Connection의 mutable current_location은 별도 잔여 항목입니다.

일반 JSON 숫자는 raw exact-decimal 표현을 유지합니다. CPython JSON float의 rounding/underflow/nonfinite 변환을 Resource Body 전체에 재현했다고 판정하지 않습니다. Body descriptor 자체는 untyped이며 불필요한 int/bool cast를 하지 않습니다.

Delete는 bodyless이며 Source의 None 반환 대신 `RBACPolicyDeleteResult`로 actual Envelope/Header/StatusCode를 노출하는 Go 편의입니다. 기본 ignore-missing true는 깨끗한 actual404만 무시하고 `IgnoredMissing=true`로 receipt를 남깁니다. `WithRecordDeleteIgnoreMissing(false)`는 native404를 반환합니다. Read/Close/Source/cancel/callback 실패가 있는404는 무시하지 않습니다. Delete의 성공 body는 decode하지 않습니다.

HTTP 완료 200..399를 성공으로 다루며 모든400..599의 native method/URL/status/body/header를 보존합니다. Source arbitrary adapter의 final informational100..199는 이 완료 profile 밖입니다. `StopFaultRetryCallbacks`를 활성화하여 물리적 읽기/Close/Source/취소 실패 뒤에 retry/reauth callback을 호출하거나 replay하지 않습니다. 여러 원인을 모두 유지하고 요청과 Close는 한 번입니다. native401은 pointer `ErrUnableToReauthenticate`의 `ErrOriginal`/`ErrReauth` 필드로 원래 응답과 handling 원인을 확인합니다.

깨끗한 native retry/live-token reauthentication은 허용하며 fixed method/route/body/query/status/version/auth/framing과 선택한 Header 정책을 지킵니다. 옵션의 maps/raw/pointers/callback 결과는 snapshot하고 selected Source는 callback 전에 capture합니다. marshal 실패의 원래 cause를 유지합니다.

## 버전과 권한

Source public attrs의 `microversion`은 Resource constructor metadata로 소비되고 HTTP override가 되지 않습니다. 실제 Create/commit/delete는 captured Adapter default를 사용하며 class ceiling/discovery가 없습니다. Go의 `WithRecord*Microversion`은 operation-local 편의입니다. explicit empty override는 Header를 생략하고 원래 ServiceClient/MoreHeaders를 수정하지 않습니다.

Stock Neutron `27.0.0`에서 create/update/delete 기본 연산은 project `ADMIN_OR_PROJECT_MEMBER`입니다. project member는 자신의 정책을 다룹니다. wildcard `*` 공유 대상은 create/update의 별도 target attribute policy로 admin 권한을 요구하므로 일반 user branch와 admin branch를 구분합니다. [공식 RBAC policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/rbac.py#L89)와 [owner base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L32)가 근거입니다.

[Stock RBAC schema](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/rbac.py#L60)는 object_type/object_id/action/tenant_id의 Create와 target_tenant 변경을 구분합니다. 기본 스키마에서 Update writable field는 target_tenant이며 object_type/object_id/action/id는 변경 불가입니다. `access_as_external`은 network/external-net 지원 여부에 영향을 받습니다. Go가 extension/role 지원을 fixture로 추정하거나 Source에 없는 local enum/required 제약을 만들지 않습니다. operator overrides/deprecated defaults/backend authorization은 실제 cloud 응답으로 확인합니다.

## 검증 범위

기존 `nativeRBACAPI`/`nativeRBACWire`/`nativeRBACCall`, internal/testcloud와 Gophercloud 공개 testhelper를 재사용합니다. 공통 `neutronfind.Engine.Mutate/ProjectSeed`는 기존 captured Source/guarded REST/record projector를 사용합니다. 새로운 전송 계층이나 HTTP 서버 구현은 없습니다. Root가 focused/full race와 독립 Go main 빌드를 실행합니다. fixture 통과는 live authenticated Neutron의 권한/extension/backend 검증 또는 다른 Neutron mutation 전체 지원을 뜻하지 않습니다.
