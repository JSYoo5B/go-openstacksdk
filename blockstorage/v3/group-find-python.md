# Cinder v3 Group 이름·ID 검색

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [find_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2126-L2155), [Group 모델](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L26-L66), 공통 [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)입니다. 이 검색은 일반 볼륨 그룹을 대상으로 하며 [Group CRUD·목록](groups-python.md)의 같은 응답 모델을 사용합니다.

| Python | Go | 동작 |
|---|---|---|
| `find_group(name_or_id)` | `Groups.FindRecord(ctx, identity)` | literal GET 우선, clean400/403/404이면 목록 검색 |
| `ignore_missing=True` 기본값 | 옵션 생략 | 성공한 전체 목록에 일치 항목이 없으면 `(nil, nil)` |
| `ignore_missing=False` | `WithRecordFindIgnoreMissing(false)` | 같은 경우 `resource.ErrNotFound` |
| `details=True` 기본값 | 옵션 생략 | fallback은 `/groups/detail` |
| `details=False` | `WithRecordFindDetails(false)` | fallback은 `/groups` |
| 선택된 Cinder 기본 microversion | `sdk.WithMicroversion(sdk.BlockStorage, version)` | GET과 fallback 목록의 원래 기본값 |
| 상속된 `Group.find(..., microversion=version)` | `WithRecordFindMicroversion(version)` | Go 확장: GET만 override, 목록은 원래 기본값 |
| session의 일반 추가 헤더 | `WithRecordFindHeaders(headers)` | Go 확장: 두 단계에 복사, 버전·인증·route control은 거부 |

고정된 `find_group` proxy는 `name_or_id`, `ignore_missing`, `details`만 받습니다. Go의 Microversion/Headers는 명시적인 확장입니다. `Group`의 query mapping에는 `name`이 없으므로 fallback LIST에 `?name=...`를 추가하지 않습니다. 모든 페이지를 읽고 실제 string `id` 또는 `name`이 caller identity와 정확히 같은 항목을 비교합니다. 최초 일치 뒤의 페이지도 확인하며, 같은 ID를 가진 두 행도 중복 오류입니다. 성공한 member GET은 서버의 ID/name을 caller 입력과 다시 비교하지 않고 반환합니다.

## 독립 Python main

clouds.yaml의 `dev` cloud에 인증합니다. `--strict`를 지정하지 않으면 고정 SDK의 `ignore_missing=True` 기본 동작을 사용합니다. 이 pin은 기본값 변경 예정 deprecation warning을 발생시킬 수 있습니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("identity", help="group ID or exact name")
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--summary", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    value = conn.block_storage.find_group(
        args.identity,
        ignore_missing=not args.strict,
        details=not args.summary,
    )
    print(json.dumps(None if value is None else value.to_dict(),
                     indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용한 모듈의 `main.go`로 저장합니다. 예: `go run . -cloud dev -identity sample -strict`. context timeout은 여러 페이지와 native retry를 포함한 검색 전체에 적용됩니다.

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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "group ID or exact name")
    version := flag.String("microversion", "3.38", "selected Cinder default")
    strict := flag.Bool("strict", false, "return an error when not found")
    summary := flag.Bool("summary", false, "use summary fallback list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    value, err := service.Groups.FindRecord(ctx, *identity,
        groups.WithRecordFindIgnoreMissing(!*strict),
        groups.WithRecordFindDetails(!*summary))
    if err != nil { log.Fatal(err) }
    if value == nil { fmt.Println("null"); return }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}
```

`groups.New(service.RawClient())`로 같은 API를 만들 수도 있습니다. interface나 사용자 builder 구현은 필요하지 않습니다. `RecordFindOpts`의 nullable bool/string과 `WithRecordFindOpts`·각 옵션 함수는 map/pointer를 복사합니다. 기본값과 명시적인 nil Headers도 빈 소유 map으로 준비하므로 첫 사용자 옵션 callback에서 안전하게 헤더를 추가할 수 있습니다. 옵션 생성 후 caller map을 바꾸거나 옵션 callback이 보관한 pointer를 HTTP 중 바꿔도 검색에 적용한 스냅샷을 바꾸지 않습니다.

## 두 단계의 버전·응답과 오류

GET 전용 override 예시는 다음과 같습니다. selected client의 `Microversion`, `MoreHeaders`, provider 설정은 변경하지 않습니다.

```go
value, err := service.Groups.FindRecord(ctx, "sample",
    groups.WithRecordFindMicroversion("3.90"),
    groups.WithRecordFindHeaders(map[string]string{"X-Trace": "group-find"}))
```

원래 selected version이 `3.13`이면 GET은 `3.90`, fallback 목록은 `3.13`입니다. selected version이 비어 있으면 기본 discovery는 Group의 cap `3.38`까지 협상합니다. 기본 GET이 이미 협상한 경우 그 실제 discovery 응답을 같은 검색의 목록 단계에 재사용합니다. 명시적인 GET override 뒤 fallback이 필요하면 목록 단계에서 원래 기본값을 협상합니다. Discovery가 거부되면 member 실패로 취급하지 않고 중단합니다. `WithRecordFindMicroversion("")`는 원래 기본값을 유지하는 Go 정책입니다. 상속된 Python `Resource.fetch`의 명시적인 빈 microversion 문자열 처리와는 다릅니다. 명시적으로 선택한 유효 version은 `3.38`보다 높아도 client 정책대로 유지됩니다.

GET은 source의 일반 `Accept` 헤더를 유지합니다. LIST는 고정 Source대로 `application/json`을 사용하며, per-call `Headers`에 case-insensitive `Accept`를 명시했을 때만 그 값을 사용합니다. 버전 세 헤더와 reserved 인증/프레이밍 헤더는 SDK가 소유합니다. Native retry callback의 버전 변경·별도 Nova 헤더 추가·Authorization 추가·owned header의 OmitHeaders 억제는 기존 실제 거부 응답과 옵션 오류를 보존하고 다음 HTTP를 차단합니다. 일반 `X-Retry` 변경과 provider의 live token 갱신은 허용됩니다. 이미 관찰된 물리적 read/Close·취소·source 변경 오류는 retry callback 전에 중단합니다.

`GroupRecord.Resource`는 선언된 Body attr의 소유 스냅샷이고 `Wire`, `Envelope`, `Header`, `StatusCode`는 실제 응답 증거입니다. 누락·null, 큰 숫자, scalar/list descriptor 변환은 [기존 Group 응답 계약](groups-python.md#소유-옵션과-source-응답)을 유지합니다. 성공한 GET의 null/numeric ID는 그대로 반환하며, fallback에서는 string identity와 맞지 않습니다. LIST row의 `cls` 같은 constructor control 충돌은 actual accepted page 오류입니다. direct GET body의 unknown `cls`는 `Wire`에 남습니다. 결과의 `Discovery`는 실제 협상 응답의 별도 복사입니다.

`errors.Is(err, resource.ErrNotFound)`는 strict 모드에서 성공한 전체 목록의 논리적 미존재를 판별합니다. LIST의 403/404/503, late-page 오류, accepted decode/constructor 오류, read/Close·취소·source 변경은 missing으로 숨기지 않습니다. `errors.As`로 `*resource.AmbiguousError`와 실제 duplicate page의 `*resource.ResponseError`를 함께 확인할 수 있습니다. accepted 오류는 실제 status/body/header를 가진 `*resource.ResponseError`를 유지하고, rejected 오류는 native `gophercloud.ErrUnexpectedResponseCode`의 status/body/header와 원래 cause를 유지합니다. 앞서 찾은 항목이 있어도 뒤 페이지가 실패하면 결과를 반환하지 않습니다.

## 권한과 범위

stock Cinder **27.0.0 / 2025.2**의 [group 정책](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)은 `group:get`·`group:get_all`을 프로젝트 reader 조회로 분류합니다. 이 profile은 caller의 project 조회를 제공하며 GroupType 관리자 CRUD나 cross-project API를 추가하지 않습니다. 배포의 정책 변경은 인증된 client와 서버가 결정합니다.

Python의 mutable Resource/Munch/subclass, Connection에서 계산한 `location`, dirty state와 session response cache는 standalone Go record와 다릅니다. `location`은 null이며 반복 Find는 매번 HTTP로 확인합니다. Python의 deprecation warning도 Go에서는 발생시키지 않습니다. 문자열 입력은 nonblank UTF-8로 검증하고 dot segment·Unicode control을 거부한 뒤 path segment로 escape하며, route/session controls와 임의 query를 받지 않습니다. 공통 pager의 origin/project/filter lock, forward cursor 검증과 dictionary-link 확장을 사용합니다. 이 named `find_group` mapping 하나가 전체 Resource lifecycle·cache parity를 완료했다는 의미는 아닙니다.

## 회귀 검증 범위

`blockstorage/v3/groups/python_find_test.go`의 17개 그룹은 기존 `groupAPI`·`groupFaultBody` fixture와 public Gophercloud testhelper를 사용하는 `internal/testcloud`를 재사용합니다. literal GET, clean fallback, missing, 모든 페이지·중복·late failure, accepted/rejected 물리적 cause, owned 옵션, 한 captured source의 두 버전 단계·discovery, native retry 정책을 검사합니다. 기존 Group GET/LIST 테스트도 helper 재사용 변경 때문에 함께 검증해야 합니다. 실제 검증 결과와 진행량은 중앙 API ledger에서 관리합니다.
