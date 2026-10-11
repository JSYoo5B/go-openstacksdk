# Cinder v3 사용자 메시지

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [Message 모델](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/message.py)과 [세 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3029-L3064)입니다. Cinder 비동기 작업 실패의 사용자 메시지를 조회·목록·삭제합니다.

| openstacksdk | Go | 기본값 |
|---|---|---|
| `get_message(id)` | `Messages.GetRecord(ctx, id)` | bodyless member GET, 누락은 오류, fallback 없음 |
| `messages(**query)` | `Messages.ListRecords(ctx, options...)` | lazy 전체 페이지, server query와 local Body filter 분류 |
| `delete_message(id)` | `Messages.DeleteRecord(ctx, id)` | bodyless DELETE, clean404만 무시 |
| `delete_message(id, ignore_missing=False)` | `WithRecordDeleteIgnoreMissing(false)` | clean404도 오류 |

새 leaf는 `messages.New(service.RawClient())`로 기존 인증된 Cinder client에 붙입니다. 생성된 transport API나 다른 메시징 서비스(Zaqar)의 API를 Cinder 기능으로 계산하지 않습니다.

## 독립 Python 예제

`main.py`로 저장합니다. 기본은 조회만 하며 `--delete`를 지정한 경우 해당 메시지를 삭제합니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.37")
    parser.add_argument("--message", default="")
    parser.add_argument("--project", default="")
    parser.add_argument("--delete", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    if args.message:
        if args.delete:
            conn.block_storage.delete_message(args.message, ignore_missing=False)
        else:
            value = conn.block_storage.get_message(args.message)
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
    else:
        query = {"project_id": args.project} if args.project else {}
        for value in conn.block_storage.messages(**query):
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용해 `main.go`로 저장합니다. `go run . -cloud dev`는 목록, `-message ID`는 단일 조회입니다. `-delete -message ID`는 strict 삭제를 실행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printMessage(value *resource.RawResource) {
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.37", "Cinder selected version")
    id := flag.String("message", "", "optional message ID")
    project := flag.String("project", "", "optional server project filter")
    remove := flag.Bool("delete", false, "strictly delete the selected message")
    flag.Parse()
    if *remove && *id == "" { log.Fatal("-delete requires -message") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := messages.New(service.RawClient())
    if *id != "" {
        if *remove {
            if err := api.DeleteRecord(ctx, *id,
                messages.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
            return
        }
        value, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printMessage(value.Resource)
        return
    }
    options := []messages.RecordListOption{}
    if *project != "" { options = append(options, messages.WithRecordListQuery("project_id", *project)) }
    for value, err := range api.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        printMessage(value.Resource)
    }
}
```

## Source 모델과 stock 응답 차이

고정 Source `Message.resource_key`는 **messages**인데 실제 Cinder [단일 조회](https://docs.openstack.org/api-ref/block-storage/v3/#show-message-details)의 envelope는 **message**입니다. 기본 `GetRecord`는 두 키와 flat object를 읽습니다. 두 키가 함께 있으면 Source messages 키의 존재가 우선합니다. Python의 오류를 그대로 재현하려면 `WithRecordGetSourceEnvelope(true)`를 사용합니다. 이 옵션에서는 stock message wrapper를 일반 unknown object로 Wire에 보존하고, Source처럼 지정한 id seed만 Resource에 남깁니다. 위 Python get 예제의 고정 Source에서는 이 불일치 때문에 메시지 속성이 비어 있을 수 있습니다.

Resource는 inherited id/name, created_at/event_id/guaranteed_until/message_level/request_id/resource_type/resource_uuid/user_message와 nullable location을 제공합니다. 누락은 null이고, explicit null도 유지합니다. 8개 string descriptor는 공통 PythonString 규칙을 적용합니다. false는 문자열 `"False"`, 큰 정수는 정확한 문자열, nonempty container는 Python repr 형식이며 날짜도 type=str 문자열로 유지합니다. id/name에는 문자열 강제 변환을 적용하지 않습니다.

Wire에는 원래 타입·unknown 키·큰 정수·links가 그대로 있습니다. Resource·Wire·전체 Envelope·각 Header는 독립 복사하고 StatusCode는 실제 응답값입니다. member의 생략된 id는 요청 literal identity를 Resource에만 seed합니다. null/number id는 그대로 보존합니다. 빈 body·malformed JSON syntax는 Source처럼 seed를 유지하고, 유효한 nonobject/null 또는 invalidUTF8는 actual ACK를 가진 오류입니다. accepted Read/Close 실패에는 부분 MessageRecord와 resource.ResponseError를 반환하며 decoded fields를 만들지 않습니다.

## 목록 옵션과 페이지

`WithRecordListQuery`의 project_id/sort/limit/offset/marker는 Source server mapping입니다. JSON bool·iterable·null은 Requests query 규칙으로 처리합니다. 나머지 declared Body attribute는 local filter이고 unknown query는 무시합니다. name/id도 server mapping에 없으므로 local filter입니다. project_id는 이 Source 모델의 Body 속성이 아니며 Wire에만 나타날 수 있습니다.

local filter는 descriptor 변환 **이후** 비교합니다. 예를 들어 message_level=false 응답은 Resource에서 `"False"`가 되어 string filter와 일치합니다. 공통 JSON-domain Python equality는 bool-number 동등성을 유지합니다. dict subset은 재귀 비교하며 truthy scalar와 빈 dict는 일치하고, nonempty dict가 scalar의 mapping.get을 요구하면 실제 page 증거를 가진 오류입니다. Go map 순서 차이 때문에 local filter를 key 순서로 평가하고 첫 불일치에서 멈춥니다.

`WithRecordListLimit/Marker/MaxItems/Paginated/Headers/Microversion/Opts`는 SDK가 소유하는 옵션입니다. MaxItems는 local filter 이전 raw row 수를 세고, 생략한 limit 또는 0에 최초 limit hint를 줍니다. cap/single-page/consumer break는 unconsumed row나 next link를 해석하기 전에 멈춥니다. 기본은 전체 페이지입니다. messages_links·next·HTTP Link·nonempty short-page marker fallback·empty-page 종료와 singleton plural object를 지원합니다. marker는 원래 Wire ID에서 계산합니다.

공통 REST 보호 규칙은 origin·collection·initial filter 변경, cursor loop·상충 continuation을 차단하고 실제 오류 page를 남깁니다. dictionary links는 Go 호환 확장입니다. offset만 지정한 경우 증가 offset continuation을 적용하며 Source의 자동 marker fallback과 다른 명시된 Go pager 선택입니다. list null row·잘못된 envelope·constructor와 충돌하는 connection/microversion/_synchronized는 오류입니다.

## 버전·권한·물리 오류

Stock User Messages는 [microversion3.3부터](https://docs.openstack.org/cinder/2026.1/configuration/block-storage/policy-personas.html#user-messages-microversion-3-3) 제공됩니다. Source Message에는 _max_microversion이 없으므로 default가 없으면 versionless, 있으면 그 값을 사용합니다. Go도 임의3.3 upgrade·capability discovery·상한을 만들지 않습니다. 예제는 pagination 사용을 위해3.37을 명시합니다. 낮은 버전의 실제 HTTP 오류를 보존하며, DELETE clean404의 기본 무시는 낮은 버전이나 없는 ID를 구분해 주는 기능이 아닙니다.

Cinder2026.1 기본 [정책](https://docs.openstack.org/cinder/2026.1/configuration/block-storage/policy.html)은 messages 목록·단일 조회를 project reader, 삭제를 project member에 허용합니다. SDK는 기존 token/context와 선택 ResourceBase를 사용하며 project_id filter를 지정한다고 관리자 범위를 취득하지 않습니다. 실제 거부는 서버 policy가 결정합니다.

Source의 공통 `raise_from_response`는 status<400을 허용합니다. Go는 유효한 HTTP100..399와 모든400..599 거부에 대해 공통 body observer를 사용합니다. custom transport가 직접 반환한199도 원래 status/body/header를 보존합니다. 일반 net/http client는 interim1xx를 자체 처리합니다. DELETE는 성공 body를 decode하지 않고 clean404만 무시합니다. Read/Close/source/cancel/callback 원인이 섞인404는 억제하지 않습니다. accepted 실제 body/header/status와 rejected native status/method/URL/body/header 및 원래 causes를 보존하고, 응답 Close는 한 번입니다. 선택한 version headers 또는 absence를 retry/physical attempts에서도 고정합니다. 원래 shared client version/headers는 변경하지 않고 live token·reauth·transport를 재사용합니다.

Go의 요청 소유 정책은 최초 cloudread에서 금지한 Authorization·X-Service-Token·Host·Cookie·Content-Length·Transfer-Encoding·Connection·Trailer·TE·Upgrade와 다른 서비스의 Nova version header를 retry hook에서도 다시 넣지 못하게 합니다. 선택한 Cinder version header를 OmitHeaders로 제거하는 것도 차단합니다. 첫 native HTTP 오류의 실제 status/body/header와 callback 원인을 함께 남기고 두 번째 전송을 하지 않습니다. 일반 retry header는 기존 native 정책을 유지하고 X-Auth-Token 값은 provider의 live token·reauth에 맡기며 고정하지 않습니다. 이 보호는 Python custom session/adapter 전체 동작과의 동등성 주장에 포함되지 않는 명시된 Go 정책입니다.

옵션은 raw JSON·map·nullable pointer를 복사합니다. nil callback·음수 typed controls·invalid known JSON·보호 auth/protocol header·invalid identity는 HTTP 전에 오류입니다. Source가 Read에서 바뀌고 Close에서 원복돼도 관측한 실패를 유지합니다. Python mutable Resource/subclass/cache/session/location은 별도 SDK-R1/C1/S1 범위이며, 공통 PythonString의 float repr·Unicode/runtime 경계도 유지합니다.

[회귀 테스트](messages/python_records_contracts_test.go)는 existing internal/testcloud와 공개 Gophercloud testhelper fixture를 재사용합니다. Source/stock envelope, descriptor/null/partial receipt, lazy query/filter/raw cap/linked pages, constructor/continuation errors, 실제 read/close·retry·cancellation 및 옵션/재인증 소유권을 검증합니다. 실제 인증된 Cinder 배포 integration 통과를 주장하지 않습니다.
