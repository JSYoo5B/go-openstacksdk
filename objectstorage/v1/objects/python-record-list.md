# Swift object 목록: openstacksdk와 Go

`Objects.ListRecords`는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [`conn.object_store.objects(container, **query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L249-L271)와 이를 감싸는 [`conn.list_objects(container, full_listing=True, prefix=None)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L347-L366)를 제공합니다. 기존 Gophercloud native `List`와 `InContainer` 범위 목록은 그대로 두고, Python 기본값과 [Object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L27-L213) 모양의 행을 소유하는 별도 메서드입니다. 이 문서는 catalog의 `python:object_store/v1/objects`와 `python:cloud/_object_store/ObjectStoreCloudMixin/list_objects` 두 행을 다룹니다.

아래 Python은 같은 이름의 proxy와 cloud 호출을 비교하는 독립 실행 예제입니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--prefix")
parser.add_argument("--mode", choices=("proxy", "cloud"), default="proxy")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "cloud":
        values = conn.list_objects(args.container, prefix=args.prefix)
    else:
        query = {} if args.prefix is None else {"prefix": args.prefix}
        values = conn.object_store.objects(args.container, **query)
    print(json.dumps([value.to_dict() for value in values], indent=2))
```

아래 Go는 독립 `main.go`입니다. `--prefix`를 생략하면 쿼리를 생략하고, `--prefix=`를 명시하면 빈 prefix를 보냅니다. [설치 안내](../../../docs/install.md)의 module 설정과 clouds.yaml을 사용합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, container string, prefix *string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { return err }
    var options []objects.RecordListOption
    if prefix != nil { options = append(options, objects.WithRecordListPrefix(*prefix)) }
    rows := make([]*resource.RawResource, 0)
    for record, err := range swift.Objects.ListRecords(ctx, container, options...) {
        if err != nil {
            if record != nil {
                fmt.Fprintf(os.Stderr, "page status=%d body=%s\n", record.StatusCode, record.Envelope)
            }
            return err
        }
        rows = append(rows, record.Resource)
    }
    output, err := json.MarshalIndent(rows, "", "  ")
    if err != nil { return err }
    fmt.Println(string(output))
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    prefix := flag.String("prefix", "", "optional prefix")
    flag.Parse()
    var selectedPrefix *string
    flag.Visit(func(value *flag.Flag) { if value.Name == "prefix" { selectedPrefix = prefix } })
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *container, selectedPrefix); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## 요청

| Python | Go | 동작 |
|---|---|---|
| `objects("assets")` | `ListRecords(ctx, "assets")` | 이스케이프한 container 경로로 bodyless GET, `format=json`, `Accept: application/json` |
| `list_objects("assets", prefix="logs/")` | `ListRecords(ctx, "assets", WithRecordListPrefix("logs/"))` | `prefix`가 None이면 Python도 쿼리를 생략하므로 옵션을 생략합니다 |
| `prefix`, `limit`, `marker` | `WithRecordListPrefix/Limit/Marker` | 그대로 server query로 보냅니다 |
| `temp_url_sig`, `temp_url_expires`, `filename`, `multipart_manifest`, `symlink` | `WithRecordListQuery(key, value)` | Object `_query_mapping`대로 server query로 보내며 `multipart_manifest`는 `multipart-manifest`로 바꿉니다 |
| `name`, `id`, `_hash`, `_bytes`, `_last_modified`, `_content_type` | `WithRecordListQuery(key, value)` | 선언된 Body attribute이므로 각 행을 만든 뒤 로컬 필터로 비교합니다 |
| `delimiter`, `end_marker`, `path`, `content_type`, `hash` 등 | `WithRecordListQuery(key, value)` | Object query mapping과 Body attribute에 없으므로 Python처럼 무시합니다 |
| `headers={...}` | `WithRecordListHeaders` | 기본 Accept 위에 덮어쓰며 인증, framing, 다른 서비스 version header는 거부합니다 |
| `microversion=...` | `WithRecordListMicroversion` | 해당 호출에서만 `OpenStack-API-Version: object-store <값>`을 보냅니다 |
| `max_items=...` | `WithRecordListMaxItems` | falsey limit이면 첫 요청에 `limit=max_items`를 넣고, 필터로 제외한 행까지 센 raw cap에서 멈춥니다 |
| Go 전용 제어 | `WithRecordListSinglePage`, `WithRecordListMaxPages` | 첫 페이지만 읽거나 페이지 수를 제한합니다. 기본 한도는 10,000페이지입니다 |

Python proxy는 `container`, `paginated=True`, `format='json'`을 직접 넘기므로 `**query`에 같은 이름을 넣으면 Python에서도 중복 인자 오류가 납니다. Go는 `WithRecordListQuery`나 `RecordListOpts.Query`에 `container`, `paginated`, `format`이 있으면 HTTP 전에 `ErrInvalidOption`을 반환합니다. `headers`, `microversion`, `max_items`는 전용 옵션을 쓰도록 같은 방식으로 거부하고, `base_path`와 `jmespath_filters`는 HTTP 전에 명시적으로 거부합니다. `cls`, `self`, `resource_type`, `session`은 Source 호출과 충돌하는 인자이고 `__conflicting_attrs`는 임의 override 제어이므로 역시 HTTP 전에 거부합니다. 값의 JSON marshaler는 실행하지 않습니다. `allow_unknown_params`는 `Resource.list`가 항상 True로 처리하므로 효과가 없어 무시합니다. 서버 이름(`multipart-manifest`)과 client 이름(`multipart_manifest`)을 함께 주면 Python `_transpose`처럼 client 이름의 값이 이깁니다.

로컬 Body 필터는 CPython `json.loads`와 같은 정수/float/bool 비교를 사용합니다. `true == 1`, `1e-400 == 0`이 성립하고, `9007199254740993.0`은 float 반올림 때문에 정수 `9007199254740993`과 다릅니다. 중첩 dict 필터도 같은 비교를 재사용합니다. 빈 dict 필터 `{}`는 모든 truthy 값(string·list·number 포함)과 일치하고 falsey/null은 제외합니다. 비어 있지 않은 dict 필터가 truthy nonmapping 값에 적용되면 실제 page 오류를 보존합니다. 알 수 없는 쿼리는 option 값의 marshaler나 DTO raw JSON 검증 전에 버립니다.

`cloud.list_objects`의 `full_listing`은 Python에서도 무시하는 하위 호환 인자이므로 Go에 대응 옵션이 없습니다. Python은 결과를 `list`로 모으고, Go는 같은 행을 iterator로 돌려주므로 수집은 호출자가 합니다.

## 행 projection

각 행은 독립된 `ObjectRecord`입니다.

- `Resource.Body`는 Python `Object.to_dict()`와 같은 key를 가집니다. Body attribute `name`, `id`, `_hash`, `_bytes`, `_last_modified`, `_content_type`와 28개 Header attribute, `last_modified_at`의 aka인 `updated_at`, `location`이 들어갑니다. 값이 없으면 null입니다.
- `_bytes`와 `content_length`, `delete_after`는 Source int descriptor 변환을 따르고 `is_newest`, `is_static_large_object`, `is_content_type_detected`는 CPython JSON float truthiness로 bool이 됩니다. `1e-400`은 Python처럼 0.0으로 underflow하므로 false이고 `1e400`은 infinity로 읽히므로 true입니다. `if_match`·`if_none_match`는 list, `range`는 dict, `if_modified_since`·`if_unmodified_since`는 str descriptor를 적용합니다.
- Header 값이 행에 없으면 alias가 Body 값을 읽습니다. 그래서 `content_length`는 `bytes`, `etag`는 `hash`, `content_type`은 `content_type`, `last_modified_at`과 `updated_at`은 `last_modified`를 보여 줍니다. `id`는 행에 `id`가 없으면 `name`입니다.
- `Container`는 호출 인자입니다. Python은 URI 값으로 행의 `container`를 덮어쓰고 proxy가 다시 `obj.container`를 설정하므로 Go도 행 값을 쓰지 않습니다. `to_dict()`에는 URI attribute가 없으므로 `Resource.Body`에도 넣지 않습니다.
- `Wire.Body`는 `subdir`, `symlink_path`, 원래 `container` 같은 unknown member까지 포함한 실제 행입니다. `Envelope`, `Header`, `StatusCode`는 그 행을 담은 실제 container page입니다.
- `Metadata`는 행의 `metadata` 생성자 값만 `X-Object-Meta-` prefix를 붙여 `json.RawMessage`로 보관합니다. 숫자, null, 중첩 값도 문자열로 바꾸지 않고 보존합니다. 각 record가 값을 소유하므로 Python BaseResource의 class-level metadata 공유는 재현하지 않습니다.

`subdir` 행도 Python처럼 하나의 Object로 소비합니다. 이름이 없으므로 `name`과 `id`는 null이고 원래 값은 `Wire.Body["subdir"]`에 남습니다. 다만 pinned Object query mapping에 `delimiter`가 없어서 Python과 Go 모두 delimiter를 서버로 보내지 않습니다. 행에 `cls`, `connection`, `microversion`, `_synchronized`가 있으면 Python 생성자 인자 충돌과 같게 실제 page 증거를 담은 오류를 돌려줍니다.

목록 행에는 Swift object HEAD 응답 header가 없습니다. pinned `Resource.list`도 page header를 행 Header attribute로 옮기지 않으므로 Go도 행 안에 선언된 Header 이름이 있을 때만 값을 채웁니다. 실제 object metadata가 필요하면 `HeadMetadata`나 `GetMetadata`를 따로 호출합니다.

## 페이지와 상태

다음 페이지는 singleton object body의 `X-Container-Object-Count` link 목록과 `next`, RFC Link, `X-Container-Object-Count` 응답 header, truthy limit fallback 순서로 찾습니다. count header는 로컬 필터 결과가 아니라 소비한 raw 행 수와 비교하고, 같은 container 경로에 마지막 행의 id(이름)를 `marker`로 붙여 다시 요청합니다. Python은 페이지마다 limit을 지우고 truthy limit만 다시 넣으므로, 0 같은 falsey limit은 첫 요청에만 나갑니다. 빈 배열 페이지에서는 바로 끝납니다.

Go는 advertised link가 같은 origin과 container 경로에 머물고 초기 filter와 limit을 바꾸지 않을 때만 따라갑니다. 생략한 쿼리는 이어받습니다. 반복 page나 marker는 `resource.PaginationCycleError`로 끝납니다. pinned Python은 Requests Link parser 결과에 없는 `uri` key를 읽어 RFC Link가 있으면 KeyError로 실패하고, 반복은 직전 marker와만 비교해 찾습니다.

성공 status는 `raise_from_response`의 유효 범위인 100..399입니다. 204처럼 본문이 JSON이 아니면 실제 page를 담은 decode 오류가 됩니다. 400..599는 404를 포함해 모두 오류이며 ignore-missing을 적용하지 않습니다. 모든 오류는 `resource.OperationError{Operation: "ListRecords", Resource: "objects"}`로 감싸고, page가 있으면 `resource.ResponseError`와 마지막으로 yield한 `ObjectRecord`에 실제 status, header, body를 남깁니다.

## Python과의 차이

- Python은 container 이름이나 Container resource를 받고 `"/%(container)s"`에 이름을 그대로 넣습니다. Go는 문자열만 받으며 `url.PathEscape`로 경로를 만들고, 빈 이름이나 `/`가 들어간 이름은 HTTP 전에 거부합니다. Python은 빈 이름을 `str(None)`으로 바꿔 `/None`을 조회합니다.
- Python 쿼리는 dict 삽입 순서로 인코딩되고 Go는 key 순서로 정렬합니다. 서버 의미는 같습니다.
- Python은 페이지 마지막 행에서 `max_items`에 도달하고 다음 페이지 근거(count header나 truthy limit)가 있으면 다음 페이지를 한 번 더 요청한 뒤 첫 행에서 멈춥니다. `max_items`가 limit을 채운 기본 경우가 여기에 해당합니다. Go는 cap에 도달하면 바로 멈추고 추가 요청을 보내지 않습니다.
- Source `Resource._collect_attrs`는 URI가 있는 목록 행에서 unknown `location`을 버린 뒤 Connection의 location을 계산합니다. Go의 SDK 소유 `Resource.Body["location"]`은 null이며 서버의 같은 이름 값은 `Wire`에 보존합니다. Connection 위치 계산은 공통 SDK-R1 추적 범위입니다.
- Python은 서버 최대치와 협상한 microversion을 쓰고, Go는 설정한 client microversion이나 `WithRecordListMicroversion` 값을 보냅니다.
- Go는 반복 page와 marker, page 한도, 다른 경로로 가는 link를 실제 응답 증거와 함께 거부합니다. Python의 임의 `base_path`, JMESPath, mutable Resource cache와 dirty-state는 공통 SDK-R1/C1/S1 계약에서 계속 추적합니다. 해당 제어는 unsupported 옵션을 성공 호출로 바꾸지 않고 HTTP 전에 거부합니다.

[계약 테스트](python_record_list_test.go)는 기본 요청과 projection, query mapping과 로컬 필터, count와 marker continuation, RFC Link, max_items와 단일 페이지, 상태 코드별 operation context, `list_objects`의 prefix 경로, 옵션 소유권과 nil header reset, CPython float 비교, consumer break·취소·iterator 재사용을 확인합니다. 응답 read/Close/source/취소 실패는 200·203·401·404·429·498·503에서 실제 page와 원인을 보존하고 caller retry·reauth·backoff를 실행하지 않습니다. 깨끗한 native retry·reauth·backoff는 같은 method/path/query/Accept/조건/version과 갱신한 token으로 진행합니다. `ErrUnableToReauthenticate`는 Unwrap하지 않으므로 native `ErrOriginal`과 `ErrReauth`도 따로 검증합니다. 실제 cloud의 ACL·backend·인증 런타임 검증은 이 fixture 테스트와 별개입니다.
