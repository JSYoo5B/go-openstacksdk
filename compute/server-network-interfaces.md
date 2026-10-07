# 서버 생성 시 NIC 선택

`service.Servers.Create`는 네트워크·기존 Neutron 포트·고정 IP·NIC tag를 SDK 소유 입력과 With 옵션으로 받습니다. Connection이 이름 조회를 연결하므로 애플리케이션에서 builder나 resolver를 구현할 필요가 없습니다. 이 기능은 [서버 생성](README.md)의 이미지·기존 볼륨·새 부팅 볼륨 흐름에서 동일하게 사용합니다.

## Python과 Go의 호출 대응

| 고정 openstacksdk | gophercloudsdk |
|---|---|
| cloud `network="private"` | `compute.WithNetworks(resource.Name("private"))` |
| cloud `nics=[{"net-id": id}]` | `ServerNetworkInterface{Network: resource.ID(id)}` |
| cloud `nics=[{"net-name": "private", "fixed_ip": ip}]` | `ServerNetworkInterface{Network: resource.Name("private"), FixedIP: ip}` |
| cloud `nics=[{"port-id": id}]` | `ServerNetworkInterface{Port: resource.ID(id)}` |
| cloud `nics=[{"tag": ""}]` | `ServerNetworkInterface{Tag: request.Present("")}` |
| cloud `nics=[{"tag": None}]` | `ServerNetworkInterface{Tag: request.Null[string]()}` |
| Proxy `conn.compute.create_server(..., networks="auto")` | `compute.WithNetworkMode("auto")` |
| Proxy `conn.compute.create_server(..., networks="none")` | `compute.WithNetworkMode("none")` |

NIC 배열은 `compute.WithNetworkInterfaces(...)`로 지정합니다.


`ServerNetworkInterface`의 `Network`와 `Port`는 각각 `resource.Ref`입니다. 명시한 ID는 조회 없이 보내고 이름은 정확히 일치하는 리소스를 찾습니다. 누락·중복은 각각 `ErrNotFound`·`ErrAmbiguous`이며, 조회 실패는 서버 POST 전에 반환합니다. standalone `compute.New`에서 이름을 쓰려면 해당 `Dependencies.Network`·`Port`를 제공해야 합니다. Connection은 이를 자동으로 준비하고 ID 입력에서는 Neutron endpoint도 선택하지 않습니다.

하나의 NIC에 network와 port를 함께 지정하거나, fixed IP만 있는 항목·tag만 있는 항목·빈 `ServerNetworkInterface{}`를 사용할 수 있습니다. 배열과 항목 순서를 그대로 보냅니다. Nova가 조합·주소·리소스 가용성을 검사합니다. `FixedIP`는 문자열을 그대로 전송하며 IPv4/IPv6 파싱을 추가하지 않습니다. zero Ref와 빈 FixedIP는 해당 키를 생략합니다.

## network와 기존 port를 함께 사용

Python cloud helper는 `port`·`port-id`를 ID로 전달합니다. 이름을 쓰려면 먼저 port를 찾습니다.

```python
import openstack

conn = openstack.connect(compute_api_version="2.42")
port = conn.network.find_port("app-port", ignore_missing=False)
server = conn.create_server(
    name="web-01", image="ubuntu", flavor="small",
    nics=[
        {"net-name": "private", "fixed_ip": "10.0.0.10", "tag": "app"},
        {"port-id": port.id, "tag": "existing-port"},
    ],
    auto_ip=False, wait=False,
)
```

Go는 port 이름 해석도 함께 처리합니다. `OS_CLOUD`/`clouds.yaml` 또는 `OS_*` 인증 설정을 준비한 뒤 실행할 독립 예제입니다.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.42"))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name: "web-01", Image: resource.Name("ubuntu"), Flavor: resource.Name("small"),
	}, compute.WithNetworkInterfaces(
		compute.ServerNetworkInterface{Network: resource.Name("private"), FixedIP: "10.0.0.10", Tag: request.Present("app")},
		compute.ServerNetworkInterface{Port: resource.Name("app-port"), Tag: request.Present("existing-port")},
	))
	if err != nil {
		return err
	}
	fmt.Println(server.ID)
	return nil
}
```

이 입력은 기존 port를 선택합니다.

선택한 NIC가 새 서버의 부팅 볼륨 매핑을 바꾸지는 않습니다. 생성 후 ACTIVE 대기가 필요하면 `compute.WithWait(...)`를 함께 사용합니다. 대기 실패는 생성 응답과 오류를 함께 반환하며, 생성한 서버나 기존 포트를 자동으로 삭제하지 않습니다.

## 선택 교체와 기본값

`WithNetworks`·`WithNetworkInterfaces`·`WithNetworkMode`는 같은 선택을 설정합니다. **마지막 유효한 옵션이 이전 선택을 교체**합니다. 따라서 교체된 포트 이름의 조회나 tag 버전 검사를 실행하지 않습니다. 빈 목록·잘못된 Ref·잘못된 mode 같은 옵션 자체의 오류는 즉시 실패하며 뒤 옵션으로 숨기지 않습니다. `WithNetworks()`·`WithNetworkInterfaces()`의 빈 선택은 오류입니다.

[Connection·YAML 기본 네트워크](server-default-network.md)도 없고 선택을 생략하면 선택된 Compute microversion **2.37 이상에서 `networks:"auto"`**를 보냅니다. [Nova 2.37 규약](https://docs.openstack.org/nova/2026.1/reference/api-microversion-history.html)은 이 버전부터 networks 입력을 필수로 지정합니다. 이전 버전이나 버전 미설정에서는 networks를 생략합니다. 이는 기존 상위 Create가 모든 버전에서 필드를 생략하던 동작의 교정이며, 이미지와 볼륨 부팅 모두에 적용합니다. `ServerNetworkInterface{}` 하나를 명시한 경우에는 `[{}]`를 유지하고 auto로 바꾸지 않습니다.

명시적인 `auto`·`none` mode는 선택된 2.37 이상이 필요합니다. tag는 값이 빈 문자열이나 null이어도 선택된 **2.42 이상**이 필요합니다. 값이 없는 `request.Optional[string]`은 tag 키를 생략합니다. native Gophercloud의 tag는 2.32–2.36도 표현하지만 이 상위 API는 Python cloud helper의 2.42 기준을 사용합니다.

NIC 옵션은 공유 client 버전을 변경하거나 추가 discovery를 수행하지 않습니다. [WithMicroversionRange/WithLatestMicroversion](../docs/microversions.md)로 먼저 협상하거나 `WithMicroversion`으로 버전을 선택합니다. 부족한 버전은 의존 리소스 조회 전에 `ErrUnsupported`로 반환합니다. 입력 slice와 Optional 값은 옵션 생성 시 복사되고 호출마다 독립적으로 적용하므로 같은 옵션을 재사용할 수 있습니다.

## 고정 소스와 남은 범위

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [cloud NIC 분기](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1074-L1154)와 [capability 검사](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L242-L312)를 비교했습니다. Python은 configured default network를 먼저 찾고 advertised min/max와 default microversion을 함께 검사합니다. 현재 Go의 선택된 버전 기준과 opt-in Connection 협상은 이 전체 정책과 같다고 판정하지 않습니다. configured default-network 선택은 [기본 네트워크 가이드](server-default-network.md)에 구현 범위를 기록했습니다. shared NAT·외부/내부 역할 탐색과 cache·service/flags·capability 정책 비교가 남아 있습니다.

명시적인 none의 비교 대상은 Python **Proxy**입니다. 고정 cloud helper는 NIC가 없고 2.37 capability가 확인되면 kwargs의 networks 값도 auto로 교체합니다. Go는 명시한 mode를 보존합니다. Python의 net-id/net-name 및 port/port-id 중복 별칭은 각각 하나의 typed Ref로 대체하고 caller dict를 변경하지 않습니다. 서버용 wire ID에도 기존 Go Ref의 안전한 ID 검증을 적용합니다.

현재 계약 테스트는 local HTTP 요청·응답, 이름 조회·오류·옵션 소유권·동시 재사용·버전 경계·부팅과 대기를 검증합니다. Go 예제는 컴파일 검증이며 Python 예제나 인증된 OpenStack 생성은 실행하지 않았습니다. 자동 floating IP·재사용, scheduler hints/server group, 추가 volume 조합, 생성 후 fault/GET, Resource 모델·descriptor·session/cache 전체 비교는 서버 생성의 남은 계약입니다. [지원 판정대장](../docs/sdk-support-ledger.md)은 NIC 부분 구현과 전체 연산 지원 판정을 구분합니다.
