# Senlin engine services

```python
services = conn.clustering.services()
```

```go
// Configure before sharing the client between goroutines.
client.Type = "clustering"
client.Microversion = "1.7"
api := services.New(client)

for service, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(service.ID, service.Host, service.Status, service.State)
}
```

List는 선택된 숫자 microversion 1.7 이상을 요구합니다. 미선택/낮은 버전/`latest`는 HTTP
전에 거부하고 client를 자동 변경하지 않습니다. ID, binary, host, topic, status/state,
nullable disabled reason, 원래 updated_at 문자열을 제공합니다. `Body`, `Header`,
`StatusCode`에서 확장 JSON과 HTTP 근거를 확인하며 숫자 정밀도를 유지합니다.

List는 lazy입니다. body next/links와 HTTP Link는 동일 origin/collection 경로에 제한하고
기존 query를 유지합니다. 공식 API는 pagination query를 선언하지 않으므로
`WithListOptions(ListOpts{Limit: 20, Marker: "marker"})` 및 `WithListQuery`는 deployment가
지원하는 경우에만 사용합니다. 기본값은 query 생략이며 full page만 보고 다음 marker를
생성하지 않습니다. 공유 `Resources.List(..., resource.WithStatus("enabled"))`는 응답을
로컬 필터링하며 status query를 보내지 않습니다.

서비스 API에는 단건 Get/변경/삭제가 없습니다. Status 필드만으로 poll 가능한 API를
추가하지 않으며 `Resources.Get`과 ID status Wait도 지원하지 않습니다. Python mutable
Resource 및 Senlin 전체 parity는 별도 범위입니다.

근거: pinned openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와
[공식 Services API](https://docs.openstack.org/api-ref/clustering/#services-services). List 성공은 200입니다.
