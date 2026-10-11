# Neutron 태그: Python/Go 사용법

`conn.NetworkV2(ctx).AttributeTags`는 Neutron 태그 proxy의 7개 연산을 제공합니다.
`NewTaggedResource(kind, id)`로 리소스를 지정하면 SDK가 경로·빈 배열 기본값·태그 캐시와
HTTP 오류 처리를 맡습니다. 사용자에게 builder나 resource interface 구현을 요구하지 않습니다.

| 고정 openstacksdk | Go | 요청과 기본 동작 |
|---|---|---|
| `get_tags(resource)` | `GetTags(ctx, target)` | 한 번 GET; 응답에 tags가 없으면 기존 캐시/default [] 유지 |
| `set_tags(resource, tags)` | `SetTags(ctx, target, tags)` | 전체 교체 PUT; nil은 JSON null, 빈 nonnil slice는 [] |
| `add_tags(resource, tags)` | `AddTags(ctx, target, tags)` | 추가 POST; nil은 []; Network TagMixin 종류 8개 |
| `add_tag(resource, tag)` | `AddTag(ctx, target, tag)` | body 없는 개별 PUT; 반환 캐시에 중복도 추가 |
| `remove_tag(resource, tag)` | `RemoveTag(ctx, target, tag)` | body 없는 strict DELETE; 캐시의 첫 일치만 제거 |
| `remove_all_tags(resource)` | `RemoveAllTags(ctx, target)` | body 없는 strict DELETE; 반환 캐시 [] |
| `check_tag(resource, tag)` | `CheckTag(ctx, target, tag)` | body 없는 GET; 성공 결과와 HTTP 오류 반환 |

고정 Python에는 `remove_tags` 선언이 없습니다. `check_tag`의 proxy annotation은 bool이지만
실제 TagMixin은 성공 때 같은 Resource를 반환하고 404 때 예외를 냅니다. Go의 `CheckTag`도
404를 오류로 유지합니다. 기존 native `Confirm`의 404→false 정책과 구별됩니다.

```python
import openstack
from openstack.network.v2.network import Network

conn = openstack.connect(cloud="dev")
network = Network(id="NETWORK_ID")  # name lookup이나 parent GET 없음
print(conn.network.get_tags(network))

# 필요할 때 실행하는 변경 예시
conn.network.set_tags(network, ["team-a", "production"])
conn.network.add_tags(network, ["reviewed"])
conn.network.add_tag(network, "managed")
conn.network.check_tag(network, "managed")  # 404는 예외
conn.network.remove_tag(network, "managed")
conn.network.remove_all_tags(network)
```

## 리소스 종류와 경로

| ResourceKind | 태그 parent collection | AddTags |
|---|---|---|
| `NetworkResource` | networks | POST 지원 |
| `PortResource` | ports | POST 지원 |
| `SubnetResource` | subnets | POST 지원 |
| `SubnetPoolResource` | subnetpools | POST 지원 |
| `RouterResource` | routers | POST 지원 |
| `FloatingIPResource` | floatingips | POST 지원 |
| `SecurityGroupResource` | security-groups | POST 지원 |
| `SecurityGroupRuleResource` | security-group-rules | Source compatibility 옵션 필요 |
| `QoSPolicyResource` | policies | 고정 Source에 add_tags 없음 |
| `TrunkResource` | trunks | 고정 Source에 add_tags 없음 |

고정 Python은 SecurityGroupRule에도 TagMixinNetwork를 선언하지만 stock Neutron 27.0.0의
태그 controller/OVO registry에는 security_group_rules가 없습니다. 이 kind의 기본 호출은 HTTP 전에
`resource.ErrUnsupported`입니다. 이 경로를 제공하는 backend에서 고정 Source 동작을 사용하려면
`WithSourceSecurityGroupRuleTags()`를 모든 해당 호출에 명시합니다. 이 옵션의 fixture 검증은
Source route 매핑 근거이며 stock27 API 지원 또는 클라우드 acceptance 근거가 아닙니다.

고정 Source의 QoSPolicy·Trunk는 공통 TagMixin을 사용하며 bulk-add method를 상속하지 않습니다.
이 두 종류의 `AddTags`는 HTTP 전에 `resource.ErrUnsupported`를 반환합니다. stock Neutron이
bulk POST를 제공하는 사실과 고정 Python 구현의 capability를 별도로 기록한 것입니다.

QoS policy의 일반 resource 경로는 `/qos/policies`지만, stock Neutron 태그 controller는
`/policies/{id}/tags`에 등록됩니다. Python QoSPolicy의 `set_tags` override는 이를 사용하지만
다른 inherited TagMixin은 `/qos/policies`를 사용합니다. Go는 위 태그 전용 경로를 모든 연산에
일관되게 적용합니다. [Neutron 27.0.0 tagging router](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/tagging.py)와
[extension middleware](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/extensions.py)가 경로 근거입니다.

서버의 `standard-attr-tag` extension과 리소스별 정책은 여전히 필요합니다. stock 27.0.0의
network 태그 쓰기는 project member, QoS policy 태그 쓰기는 project manager 정책을 사용하며
SDK가 권한을 확대하지 않습니다. [Network 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py),
[QoS 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)을 기준으로 분류했습니다.
QoSPolicy.set_tags가 explicit raise_from_response를 생략한 부분도 Go에서는 오류를 검사합니다.

## 독립 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`에 저장합니다.
기본 `-operation get`은 태그만 읽습니다. 다른 연산은 `-operation`으로 명시합니다.

```sh
go run . -cloud dev -kind network -id NETWORK_ID
go run . -cloud dev -kind network -id NETWORK_ID -operation set -tags team-a,production
go run . -cloud dev -kind network -id NETWORK_ID -operation check -tag managed
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    kind := flag.String("kind", "network", "network, port, subnet, subnet-pool, router, floating-ip, security-group, security-group-rule, qos-policy, trunk")
    id := flag.String("id", "", "literal resource ID")
    operation := flag.String("operation", "get", "get, set, add-many, add, remove, clear, check")
    list := flag.String("tags", "", "comma-separated tag list; empty clears with set")
    tag := flag.String("tag", "", "individual tag")
    sourceRuleTags := flag.Bool("source-rule-tags", false, "enable Source SGR tag compatibility on a backend that supports it")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    target := attributestags.NewTaggedResource(attributestags.ResourceKind(*kind), *id)
    tags := []string{}
    if *list != "" { tags = strings.Split(*list, ",") }
    options := []attributestags.TagOption{}
    if *sourceRuleTags { options = append(options, attributestags.WithSourceSecurityGroupRuleTags()) }
    var result *attributestags.TagResult
    switch *operation {
    case "get": result, err = service.AttributeTags.GetTags(ctx, target, options...)
    case "set": result, err = service.AttributeTags.SetTags(ctx, target, tags, options...)
    case "add-many": result, err = service.AttributeTags.AddTags(ctx, target, tags, options...)
    case "add": result, err = service.AttributeTags.AddTag(ctx, target, *tag, options...)
    case "remove": result, err = service.AttributeTags.RemoveTag(ctx, target, *tag, options...)
    case "clear": result, err = service.AttributeTags.RemoveAllTags(ctx, target, options...)
    case "check": result, err = service.AttributeTags.CheckTag(ctx, target, *tag, options...)
    default: log.Fatalf("unknown -operation %q", *operation)
    }
    if err != nil { log.Fatal(err) }
    view := map[string]any{
        "kind": result.Resource.Kind(), "id": result.Resource.ID(),
        "tags": result.Resource.Tags(), "observed_tags": result.ObservedTags,
        "status_code": result.StatusCode,
    }
    data, err := json.MarshalIndent(view, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## 캐시와 실제 응답

`TaggedResource`는 immutable 값입니다. `Tags()`는 독립 slice를 반환하고
`WithCachedTags(tags)`는 이미 아는 cache를 복사합니다. 새 참조의 cache는 []입니다.
결과의 `Resource`를 다음 연산에 넘기면 Python Resource의 cache에 해당하는 상태를 이어갑니다.
입력 객체를 변경하거나 remote parent를 자동 조회하지 않습니다.

이 local tag 상태는 Python Proxy의 HTTP response cache와 별개입니다. 고정 Source는
Connection cache를 켜고 해당 resource expiration을 설정하면 `get_tags`·`check_tag`의
GET을 공유 response cache에서 반환할 수 있고, 변경 요청은 해당 cache prefix를 무효화합니다.
Go 태그 API는 공유 HTTP response cache를 구현하지 않으므로 GET은 항상 transport를 사용하며,
Python cache hit의 zero-HTTP 동작이나 공유 cache 무효화를 제공하지 않습니다.

`GetTags`는 응답의 `tags`가 있으면 cache를 교체합니다. 누락은 기존 cache 유지,
[]는 빈 cache, null은 nil cache입니다. `ObservedTags`는 GET이 실제 tags 필드를 해석한
경우에만 true입니다. unknown 응답 필드는 `Body`에 그대로 남습니다. Go는 응답을 JSON object로,
태그를 문자열 배열 또는 null로 검사합니다. Python의 untyped list/scalar descriptor 변환과 다른
명시적 Go 타입 경계이며 잘못된 응답은 실제 status/header/body를 가진 `resource.ResponseError`입니다.

변경 응답은 태그 목록으로 decode하지 않습니다. `SetTags`는 입력 목록을 반환 cache로,
`AddTags`는 기존 cache와 합집합 대신 입력 목록을 cache로 기록합니다. 이것이 고정 Source의
동작입니다. `TagResult.Body`, `Header`, `StatusCode`는 실제 응답이므로 input-derived cache를
서버가 관측한 목록으로 오해하지 않습니다. 변경 후 전체 실제 목록이 필요하면 `GetTags`를 호출합니다.

`tags:null` cache에 개별 add/remove를 적용하면 Python도 성공한 요청 이후 local list 작업에서
실패합니다. Go도 actual acknowledgement를 결과와 오류에 유지합니다. 실패가 이미 완료된 HTTP
변경을 취소했다는 뜻은 아닙니다. cache를 원하지 않으면 새 `NewTaggedResource`를 사용합니다.

## 옵션과 오류

`WithSourceSecurityGroupRuleTags()`는 SGR kind에만 적용하며 기본 stock27 capability와
Source compatibility를 구별합니다. 다른 kind에 이 옵션을 적용하면 HTTP 전에 거부합니다.

`WithTagHeader`는 ordinary 요청 header를 복사합니다. `SetTags`·`AddTags`에는
`WithTagField`로 JSON extension을 줄 수 있습니다. nil/false/0/빈 배열도 보존합니다.
`tags`는 typed 인수가 소유하고 query·secondary argument·bodyless method의 Body extension은
HTTP 전에 거부합니다. 인증·route·framing header는 SDK 소유입니다.

선택한 generic API version 값(또는 version 미선택 시 header 부재), Neutron에 해당하지 않는
Nova·Cinder version header 부재, 인증·framing header 제약은 native retry의 모든 실제 시도에서
유지합니다. bulk JSON의 `Content-Type`은 service/option에서 선택한 값 또는 기본
`application/json`을 유지합니다. retry가 이 값을 변경·삭제하거나 `OmitHeaders`로 숨기면
첫 실제 rejected response의 status/body/header와 callback 원인을 보존하고 재전송 전에 거부합니다.
ordinary `X-Retry` 같은 header는 native retry 정책을 유지하며, bodyless `Content-Type`도 고정하지
않습니다. `X-Auth-Token`은 shared provider의 live token을 매 시도 반영하므로 frozen header로
취급하거나 부재를 강제하지 않습니다.

ID와 개별 tag는 literal segment로 escaping하므로 `/`, `%`가 경로를 바꾸지 않습니다.
빈 ID/tag·dot segment·잘못된 UTF-8·control 문자는 HTTP 전에 거부하는 Go 개선입니다.
bulk tag의 빈 문자열·중복·길이·개수·authorization 제약은 서버가 검사합니다.

Source처럼 200..399를 accepted response로 처리하며 check/remove/clear의 404는 strict 오류입니다.
GET의 빈 204는 JSON을 해석할 수 없어 actual response 오류이고, 변경·개별 check의 bodyless
204는 유효한 acknowledgement입니다. read/Close·context custom cause·선택 service 변경·retry
ownership 오류를 합쳐서 보존합니다. 실제 rejected status/body/header와 successful response의
receipt를 서로 바꿔 표현하지 않습니다.

이 API는 고정 proxy의 callable 동작을 매핑합니다. 모든 Resource subclass·동적 extension 등록·
캐시 동기화·인증된 클라우드 acceptance를 완료했다는 주장은 하지 않습니다.
