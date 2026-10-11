# Cinder Group 복제 action

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [세 proxy 호출](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2275-L2319)과 [Group action 구현](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L101-L140)입니다. 같은 [Group leaf](groups-python.md)를 사용하며 Find, GET 또는 완료 대기를 mutation 앞뒤에 자동으로 추가하지 않습니다.

| 고정 Python proxy | Go | 기본 요청 |
|---|---|---|
| `enable_group_replication(group)` | `Groups.EnableReplicationRecord(ctx, id)` | `{"enable_replication":{}}` |
| `disable_group_replication(group)` | `Groups.DisableReplicationRecord(ctx, id)` | `{"disable_replication":{}}` |
| `failover_group_replication(group)` | `Groups.FailoverReplicationRecord(ctx, id)` | `{"failover_replication":{"allow_attached_volume":false,"secondary_backend_id":null}}` |
| `allowed_attached_volume=True` | `WithRecordFailoverAllowedAttachedVolume(true)` | wire 이름은 `allow_attached_volume` |
| `secondary_backend_id="backend"` | `WithRecordFailoverSecondaryBackendID("backend")` | null 기본값과 empty string을 구분 |

이 API는 project member를 위한 Group 복제입니다. 세 action 모두 [Cinder microversion **3.38**부터](https://docs.openstack.org/api-ref/block-storage/v3/#group-replication-groups-action) 제공되며, stock **27.0.0 / 2025.2**의 [group_actions 정책](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py#L81-L116)은 `group:enable_replication`, `group:disable_replication`, `group:failover_replication`에 project member 규칙을 적용합니다. [2025.2 정책 문서](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)에서도 확인할 수 있습니다. GroupType 관리와 reset-status 관리자 API는 이 묶음에 포함하지 않습니다. 실제 backend와 배포 정책이 작업 수용과 실행을 결정합니다.

## 고정 Python wire와 stock 서버의 차이

이 pin의 Python은 enable/disable에 **null**을 보내고 failover에는 **`modify_body_for_action`** key를 보냅니다. 반면 stock Cinder27의 [요청 schema](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/schemas/groups.py#L132-L173)는 enable/disable의 값을 object로 제한하며, [controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/groups.py#L339-L404)는 failover를 `failover_replication`으로 등록합니다. 따라서 해당 Source wire를 stock 정상 요청과 같다고 설명하지 않습니다.

Go 기본값은 위 표처럼 세 stock schema에 맞게 고칩니다. 고정 Source의 wire를 비교·재현할 때만 `WithRecordReplicationPinnedSourceWire(true)`를 enable/disable에, `WithRecordFailoverPinnedSourceWire(true)`를 failover에 적용합니다. 이 모드는 Source의 null/wrong-key를 그대로 보내며 실제 400을 수용 성공으로 바꾸지 않습니다. 이 named mapping은 Source 입력·목표 동작과 owned 정책을 제공하며, 세 잘못된 Source body의 byte parity를 기본값으로 주장하지 않습니다.

## 독립 Python main

clouds.yaml의 기존 cloud로 인증한 뒤 요청한 mutation 하나를 실행합니다. 이 pin을 그대로 쓰면 위 schema 차이 때문에 stock 서버에서 400이 발생할 수 있습니다. Python 함수의 성공 반환값은 `None`이며 backend 완료 상태를 조회하지 않습니다.

```python
import argparse
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("group", help="literal group ID")
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--action", choices=["enable", "disable", "failover"],
                        default="enable")
    parser.add_argument("--allow-attached-volume", action="store_true")
    parser.add_argument("--secondary-backend-id", default=None)
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    if args.action == "enable":
        conn.block_storage.enable_group_replication(args.group)
    elif args.action == "disable":
        conn.block_storage.disable_group_replication(args.group)
    else:
        conn.block_storage.failover_group_replication(
            args.group,
            allowed_attached_volume=args.allow_attached_volume,
            secondary_backend_id=args.secondary_backend_id,
        )
    print("action call completed without exception")


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용한 모듈의 `main.go`로 저장합니다. `go run . -group ID -action failover -secondary-backend-id backend`는 failover 요청 하나를 보냅니다. 아래 프로그램은 선택한 mutation을 실행하므로 실제 group ID와 목적에 맞는 action을 지정합니다.

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("group", "", "literal group ID")
    action := flag.String("action", "enable", "enable/disable/failover")
    version := flag.String("microversion", "3.38", "selected Cinder version")
    allow := flag.Bool("allow-attached-volume", false, "allow attached volumes")
    backend := flag.String("secondary-backend-id", "", "optional backend ID")
    sourceWire := flag.Bool("pinned-source-wire", false, "replay the fixed Source body")
    flag.Parse()
    backendPresent := false
    flag.Visit(func(f *flag.Flag) { if f.Name == "secondary-backend-id" { backendPresent = true } })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    var value *groups.ReplicationRecordResult
    switch *action {
    case "enable":
        value, err = service.Groups.EnableReplicationRecord(ctx, *id,
            groups.WithRecordReplicationPinnedSourceWire(*sourceWire))
    case "disable":
        value, err = service.Groups.DisableReplicationRecord(ctx, *id,
            groups.WithRecordReplicationPinnedSourceWire(*sourceWire))
    case "failover":
        options := []groups.RecordFailoverOption{
            groups.WithRecordFailoverAllowedAttachedVolume(*allow),
            groups.WithRecordFailoverPinnedSourceWire(*sourceWire),
        }
        if backendPresent { options = append(options, groups.WithRecordFailoverSecondaryBackendID(*backend)) }
        value, err = service.Groups.FailoverReplicationRecord(ctx, *id, options...)
    default:
        log.Fatal("unknown action")
    }
    if value != nil {
        fmt.Printf("reference=%q actual_http=%d actual_body=%q discovery=%d\n",
            value.Reference, value.StatusCode, value.Envelope, len(value.Discovery))
    }
    if err != nil { log.Fatal(err) }
}
```

`groups.New(service.RawClient())`로 같은 API를 만들 수 있습니다. 사용자 interface/builder 구현이 필요하지 않습니다. literal ID는 nonblank UTF-8로 검증하고 한 segment로 escape합니다. 이름 자동 검색과 input Resource의 mutable state는 제공하지 않습니다.

## 기본값·버전·응답 소유권

failover body는 두 parameter를 항상 포함합니다. allowed-attached 기본값은 false이고 secondary backend는 null입니다. explicit false·empty string도 그대로 보내며 생략과 구분합니다. `RecordFailoverOpts`·`WithRecordFailoverOpts`는 bool/string pointers와 Headers를 복사하고, `RecordReplicationOpts`도 version/header 값을 복사합니다. Bulk Opts로 profile을 다시 설정하면 이전 backend 옵션을 null 기본값으로 되돌릴 수 있습니다. Header factory는 생성 시 복사하며 기본/nil header map도 첫 사용자 옵션 callback이 수정할 수 있는 빈 소유 map으로 준비합니다. callback이 보관한 pointer를 나중에 바꿔도 적용한 body/header/version 스냅샷을 바꾸지 않습니다. backend ID의 invalid UTF-8은 encoder 치환 전에 오류로 반환합니다.

선택된 finite Cinder3.x version이 `3.38` 이상이면 그대로 사용합니다. `3.37`, 다른 major 또는 global `latest`는 이 profile의 최소 버전 검증에서 `ErrUnsupported`이며 mutation HTTP를 보내지 않습니다. 이것은 Source가 낮은 version으로 요청해 서버 오류를 받는 동작과 다른 Go 정책입니다. version이 비어 있으면 공통 bounded discovery로 advertised 범위가 `3.38`을 지원하는지 확인하고 Group ceiling **3.38**로 협상합니다. Group의 ceiling은3.38이고 GroupSnapshot의3.29와 구분됩니다. 선택된 명시 version에는 추가 discovery를 하지 않으므로 실제 서버·driver 지원을 별도로 증명하지 않습니다.

`WithRecordReplicationMicroversion`·`WithRecordFailoverMicroversion`은 해당 action의 소유 버전 override입니다. empty override는 기본 negotiation을 요청합니다. typed version이 유효하면 selected source의 동등한 version header aliases를 private copy에서 재구성하므로 원래 client 설정을 바꾸지 않습니다. 세 버전 헤더를 per-call Headers로 직접 덮어쓰는 것은 사전 오류입니다. ordinary Headers는 공통 정책을 따르며 auth/route/framing은 SDK가 소유합니다.

`ReplicationRecordResult`는 caller의 literal `Reference`와 실제 action의 `Envelope`, `Header`, `StatusCode`, 별도의 `Discovery` 복사만 제공합니다. accepted body의 JSON 타입이나 UTF-8은 추가로 해석하지 않는 Source action 동작을 유지합니다. Group Resource나 `status=available` 같은 상태를 만들지 않습니다. HTTP202는 요청 수용 증거이며, 복제·failover 완료는 이후 Group 조회와 backend 정책으로 확인합니다. preparation 오류에서 실제 discovery가 있었다면 `StatusCode=0`, action body/header 없이 `Discovery`만 반환합니다. 이는 POST가 실행됐다는 뜻이 아닙니다.

400·403·404도 action 오류로 반환하고 ignore-missing/Find fallback으로 바꾸지 않습니다. Native 거부의 code/body/header, accepted read/Close 오류의 actual `ResponseError`, 원래 read/Close/context/custom-cancel/source cause를 유지합니다. source가 Close에서 복구돼도 guard 오류는 남습니다. retry callback의 body 변경, owned version/Content-Type 변경·OmitHeaders 제거, reserved auth/Nova header 추가는 원래503 증거와 옵션 오류를 보존하고 다음 HTTP를 차단합니다. 이미 관찰된 물리적 오류는 callback 전에 중단하며, ordinary retry header 추가와 live provider token 갱신은 native 정책을 유지합니다.

Python의 mutable Resource/session/subclass, persistent dirty/cache 상태와 connection overload는 기존 공통 SDK 경계로 남습니다. 여기서 제공한 세 action과 schema 개선을 전체 lifecycle/cache parity나 실제 backend 인수 테스트 완료로 간주하지 않습니다. `python_replication_contracts_test.go`의15개 그룹은 기존 Group/public Gophercloud testhelper fixture와 fault injector를 재사용합니다. 검증 결과와 새 named API3개 집계는 중앙 ledger에서 관리합니다.

물리적 Read/Close 오류가 있는401은 실제401 응답과 원인을 남기고 retry/reauth callback을 호출하지 않습니다. 완전한401은 기존 native token renewal로 한 번 재인증한 뒤 같은 owned POST body/version으로 재시도합니다.

Native `*gophercloud.ErrUnableToReauthenticate`는 ErrOriginal/ErrReauth를 Unwrap하지 않습니다. 실제401 receipt는 ErrOriginal의 native 응답에서만 복사하며 pointer wrapper와 ErrReauth의 물리 원인을 유지합니다. errors.As로 wrapper를 얻어 두 필드를 확인합니다.
