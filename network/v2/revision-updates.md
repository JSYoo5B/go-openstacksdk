# Neutron revision 조건 수정

`service.Networks.Update` 등 아홉 Update API는 typed `UpdateOpts.RevisionNumber` 포인터로 `If-Match: revision_number=N`을 전송합니다. nil은 typed 조건 header를 생략하고 nonnil0은 `revision_number=0`을 만듭니다. revision은 JSON body나 query 필터가 아니며 native가 지원하는 signed int도 decimal 문자열로 전달합니다. 서버가 조건의 유효성과 일치를 판정합니다.

Python의 NetworkResource는 다음처럼0도 명시한 조건으로 사용합니다.

```python
network = conn.network.update_network(network_id, if_revision=0, name="renamed")
```

Go의 `service`는 `conn.NetworkV2(ctx)`가 반환하는 `*networkv2.Service`입니다. 이미 존재하는 network ID, 원하는 이름과 서버에서 관측한 revision을 전달합니다. 아래 `WithUpdateOptions`는 앞서 전달한 옵션 전체를 교체해 최종 body와 revision을 함께 선택합니다.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    networkv2 "github.com/JSYoo5B/gophercloudsdk/network/v2"
    "github.com/JSYoo5B/gophercloudsdk/network/v2/networks"
)

func renameNetwork(ctx context.Context, service *networkv2.Service, id, name string, revision int) (*networks.Network, error) {
    options := networks.UpdateOpts{
        Name: &name,
        RevisionNumber: &revision,
    }
    updated, err := service.Networks.Update(ctx, id, networks.UpdateOpts{},
        networks.WithUpdateOptions(options))
    if err != nil {
        var response gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &response) {
            fmt.Printf("HTTP%d body=%q headers=%v\n",
                response.Actual, response.Body, response.ResponseHeader)
        }
    }
    return updated, err
}

func main() {}
```

아래 API의 UpdateOpts에도 같은 RevisionNumber 포인터가 있습니다. 다른 body 필드는 각 패키지의 typed options를 사용합니다. 기존 `WithUpdateField`로 허용된 body 확장을 추가할 수 있으며 Update signature·옵션 alias·반환 타입은 그대로입니다.

| Go service field / UpdateOpts package | pinned Python update | pinned Python 조건 header |
|---|---|---|
| Networks / [networks](networks/api_generated.go) | update_network | NetworkResource: if_revision 지원 |
| Ports / [ports](ports/api_generated.go) | update_port | NetworkResource: if_revision 지원 |
| Subnets / [subnets](subnets/api_generated.go) | update_subnet | NetworkResource: if_revision 지원 |
| Routers / [routers](extensions/layer3/routers/api_generated.go) | update_router | NetworkResource: if_revision 지원 |
| FloatingIPs / [floatingips](extensions/layer3/floatingips/api_generated.go) | update_ip | NetworkResource: if_revision 지원 |
| SecurityGroups / [groups](extensions/security/groups/api_generated.go) | update_security_group | NetworkResource: if_revision 지원 |
| Trunks / [trunks](extensions/trunks/api_generated.go) | update_trunk | Resource 상속: 자동 revision header 없음 |
| SubnetPools / [subnetpools](extensions/subnetpools/api_generated.go) | update_subnet_pool | Resource 상속: 자동 revision header 없음 |
| QoSPolicies / [policies](extensions/qos/policies/api_generated.go) | update_qos_policy | Resource 상속: 자동 revision header 없음 |

[pinned Network Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L213-L229)는 NetworkResource이고 if_revision이 None이 아닐 때 `if_match=revision_number=N`을 설정하므로0도 조건입니다. [NetworkResource.if_match](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py#L23-L29)는 list Header로 보관하고 generic request 준비가 wire 문자열로 만듭니다. [Trunk](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/trunk.py#L22), [SubnetPool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py#L23), [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py#L21)는 Resource를 직접 상속하므로 forwarded if_revision은 helper에 소비되지만 header를 생성하지 않습니다. 이 세 Go API의 typed revision 전송은 기존 native Gophercloud capability입니다.

raw `ServiceClient.MoreHeaders`, `RequestOpts.OmitHeaders`와 native retry/reauth/redirect/transport callback 정책은 유지합니다. header를 직접 덮어쓰거나 생략하는 확장이 있으면 최종 wire 값은 typed RevisionNumber와 달라질 수 있습니다. nil은 typed header 생성만 생략하며 raw client에 이미 설정된 If-Match를 지우지 않습니다. 이 API는 native hook의 header 변경을 막는 불변 CAS 계약을 추가하지 않습니다. override가 없는 native 재시도는 이미 준비한 조건을 재전송합니다.

조건 불일치의412를 포함한 원래 native 응답 오류는 `OperationError`를 통해 확인합니다. SDK는 이 오류 뒤 무조건 수정으로 전환하거나 사전 GET으로 revision을 추론하지 않습니다. 성공 코드·body envelope·typed response decoding과 native callback 재시도는 각 기존 Update를 따릅니다. Python의 kwargs alias·descriptor coercion·[dirty/cache·no-op commit·session](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L1993) 전체 동작을 완료한 판정은 아닙니다.
