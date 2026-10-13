# Neutron availability zones

`ListRecords`와 `AllRecords`는 고정 [availability_zones proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)의 실제 동작에 대응합니다. 이 proxy는 `**query`를 받지만 `_list(AvailabilityZone)`에 전달하지 않습니다. Go도 이 기본 계약을 유지합니다. `WithProxyQuery`의 name/resource/limit/marker/jmespath_filters/headers 등은 모두 무시하며 호출자 객체를 marshal하거나 URL query·로컬 필터로 바꾸지 않습니다. Resource 단계의 query 개선은 별도 기능이고 현재 proxy 동작으로 집계하지 않습니다.

```python
import openstack

conn = openstack.connect()
# 고정 proxy는 name을 무시하고 보이는 zone 전체를 반환합니다.
zones = list(conn.network.availability_zones(name="az-a"))
for zone in zones:
    print(zone.name, zone.resource, zone.state)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := availabilityzones.New(service.RawClient())
    zones, err := api.AllRecords(ctx, availabilityzones.WithProxyQuery("name", "az-a"))
    if err != nil { log.Fatal(err) }
    for _, zone := range zones {
        fmt.Println(string(zone.Resource.Body["name"]), string(zone.Resource.Body["resource"]), string(zone.Resource.Body["state"]))
    }
}
```

`ListRecords`는 lazy iterator입니다. 반복을 시작할 때 service를 캡처하고 callback을 적용한 뒤 GET합니다. `AllRecords`는 성공 시 nonnil slice를, 오류 시 partial rows를 버리고 nil을 반환합니다. 반복을 일찍 끝내면 다음 페이지를 요청하지 않습니다. 서버 links·cursor는 공통 Neutron Engine/REST의 origin·route 검증을 따르며 name을 ID로 추측하지 않습니다.

`AvailabilityZoneRecord.Resource`에는 name/resource/state/id의 Source nullable view와 미지의 확장이 있습니다. 이 필드들은 임의 string/bool 변환을 하지 않습니다. `Wire`는 실제 row, `Envelope`는 실제 page, Header/StatusCode는 실제 HTTP 증거이며 독립 복사입니다. connection location은 공통 SDK null view이고 서버 확장은 Wire에 남습니다. malformed row·constructor 충돌은 actual page 증거를 가진 오류입니다.

공통 Engine은 HTTP200–399, source/context guard, actual Read/Close 오류, native401·clean renewal/retry와 선택한 service microversion을 처리합니다. kwargs의 microversion은 무시하며 기존 client의 버전은 적용합니다. 응답 cache는 없고 Python의 optional Adapter cache/session/객체 lifecycle은 전체 SDK-R1/C1/S1에 남습니다. Go callback·nil 옵션·strict UTF8·continuation origin 제약은 명시 차이입니다.

[Neutron27 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/availability_zone.py)은 project scope의 `role:reader`입니다. AZ는 project 소유 리소스가 아니므로 project_id 조건이 없는 reader rule을 사용합니다. deprecated rule은 RULE_ANY이고 배포의 role implication·override·visibility는 서버가 결정합니다. 핵심 user 조회이며 실클라우드 인증 검증을 주장하지 않습니다.

[계약 테스트](records_test.go)는 kwargs 무시와 marshal 없음, 실제 row/receipt, 페이지 오류와 eager partial 폐기, 조기 종료, callback source/cancel, constructor 오류와 clean401을 검증합니다. public Gophercloud testhelper의 internal/testcloud를 재사용하며 Go/race·독립 main build는 root gate로 기록합니다.
