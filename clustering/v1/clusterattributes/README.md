# Senlin cluster attributes

`ClusterAttributes.InCluster(ctx, ref, path)`는 canonical cluster와 JSONPath를 고정한
scope를 만들고, 각 node에서 일치한 값을 `List` / `All`로 수집합니다. 선택한 numeric
microversion은 1.2 이상이어야 합니다. Connection을 사용하면
`sdk.WithMicroversion(sdk.Clustering, "1.2")` 또는 그 이상의 버전을 설정합니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `collect_cluster_attrs(cluster, path, **query)` | `scope.List(ctx)` / `All(ctx)` | GET `/clusters/{cluster}/attrs/{path}`, `cluster_attributes` 배열 |
| cluster와 path를 URI에 직접 대입 | `InCluster(ctx, ref, path)` | 부모 GET 또는 정확한 Name 검색 후 canonical ID와 path 고정 |
| `node_id` / `attr_value` | `NodeID` / `Value` | node ID 문자열과 임의 raw JSON 값 |

```python
for attr in conn.clustering.collect_cluster_attrs("CLUSTER_ID", "$.status"):
    print(attr.node_id, attr.attr_value)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
scope, err := service.ClusterAttributes.InCluster(ctx,
    resource.ID("CLUSTER_NAME_OR_ID"), "$.status")
if err != nil { return err }
for attr, err := range scope.List(ctx) {
    if err != nil { return err }
    fmt.Println(attr.NodeID, string(attr.Value))
}
```

직접 client를 가진 경우 `clusterattributes.New(client)`로 같은 API를 사용할 수 있습니다.
`resource.ID("cluster-name")`은 이름·short ID·UUID를 controller identity로 직접 GET합니다.
`resource.Name("cluster-name")`은 목록에서 정확히 일치하는 이름을 찾고 중복을 거부합니다.
Python은 이 proxy에서 parent 값을 URI에 직접 넣지만, Go는 scope 생성 시 부모 확인 요청을
추가합니다. Name 조회는 여러 페이지를 검색할 수 있습니다. 부모가 없거나 응답의 canonical
`id`가 누락·null·잘못된 문자열이면 전체 HTTP 응답 증거를 가진 오류를 반환합니다.

Scope 생성 후 부모를 다시 해석하지 않습니다. `ClusterID()`와 `Path()`는 고정 범위를
보여주고, 각 행의 `URIClusterID`와 `URIPath`에도 그 범위를 보존합니다. 반환한 모델의
NodeID나 URI 필드를 수정해도 다음 요청의 parent/path는 바뀌지 않습니다.

## JSONPath와 raw 값

JSONPath의 앞뒤 공백을 제거하고, 빈 문자열과 literal `None`은 HTTP 전에 거부합니다.
슬래시·괄호·물음표·fragment 문자를 포함한 입력도 하나의 escaped URI segment에 넣습니다.
완전한 `.` / `..` segment도 encode하여 router가 경로 이동으로 처리하지 않게 합니다.
JSONPath 문법과 실제 조회 가능 속성은 서버 parser가 판정합니다. 버전 gate와 입력 검사는
부모 요청 전후에 실행하고, 각 수집 페이지에서도 source와 선택 버전을 다시 검사합니다.

16.0.0 conductor는 node 정보에서 JSONPath의 첫 번째 일치 값을 수집하며, 일치하는 값이
없는 node는 결과에 포함하지 않습니다. 응답의 `Value`는 `json.RawMessage`여서 object·array·
string·boolean·number·null을 모두 보존하고 큰 정수나 decimal을 float64로 바꾸지 않습니다.
`Value == nil`은 wire 필드 생략이고 `string(Value) == "null"`은 명시 null입니다. 필요한
경우 caller가 `json.Unmarshal` 또는 `json.Decoder.UseNumber`로 해석합니다.

`Body`에는 unknown 필드와 원문 숫자·null을 보존하고 `Value`와 독립된 byte slice를 사용합니다.
알려진 `id` / `value`와 parent의 `id` / `name`은 정확한 lower-case Body 필드에서 읽으므로
대소문자가 다른 extension 필드가 식별자·이름 매칭·값을 바꾸지 않습니다. 각 행은 유효한
node `id` 문자열을 요구합니다. `Header`와 `StatusCode`도 독립적으로 보존합니다.

## 성공 코드와 목록 범위

공식 API reference는 202를 기재하지만 16.0.0 router에는 성공 코드 override가 없고 WSGI
Resource의 기본 응답은 200입니다. 이 API는 두 코드를 모두 `cluster_attributes` 읽기
응답으로 처리합니다. conductor는 배열을 동기 반환하며 action이나 Location을 생성하지
않습니다. 응답에 incidental Location 또는 action이 있어도 action Submission을 합성하거나
추가 URL을 조회하지 않습니다.

목록 호출은 lazy이고 `All`은 모든 행을 모읍니다. 서버가 제공한 body/HTTP next link만
따라가며 같은 origin·고정 cluster·고정 escaped JSONPath에 머물러야 합니다. node ID를
marker로 추측하거나 limit/marker fallback 요청을 생성하지 않습니다. Stock endpoint는
paging을 약속하지 않습니다. `break`는 후속 페이지를 조회하지 않고, 같은 iterator의
재순회는 독립적인 요청을 시작합니다.

Pinned Python proxy는 `**query`를 받지만 `_list`에 전달하지 않습니다. Go의 `List` / `All`
도 query나 per-request header 옵션을 노출하지 않습니다. `Resources.List`의 초기 query는
`ErrUnsupported`입니다. 인증·endpoint prefix·선택 microversion은 source client를 공유합니다.
Body local filters, Resource cache/lifecycle과 generic query 소비까지의 Python 동일성은 남은
비교 범위입니다. 별도 단건 endpoint가 없는 list-only 리소스이므로 Get/Delete/상태 대기와
삭제 대기는 지원하지 않습니다.

잘못된 성공 envelope·행·node ID·next link는 원래 페이지 전체 body/header/status를 담은
`resource.ResponseError`로 반환합니다. HTTP 실패의 Gophercloud 오류와 context 원인을
유지하며 재전송하지 않습니다.

계약 근거: [공식 attribute collection API](https://docs.openstack.org/api-ref/clustering/#collect-attributes-across-a-cluster),
[16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py),
[cluster controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/clusters.py),
[WSGI Resource](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/common/wsgi.py),
[conductor](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py).
Pinned Python 비교는 `openstack/clustering/v1/cluster_attr.py:16–31`과
`openstack/clustering/v1/_proxy.py:630–648`에 대응합니다.
