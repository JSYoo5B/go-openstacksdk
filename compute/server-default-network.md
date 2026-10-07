# 서버 생성의 기본 네트워크

Connection은 서버에 NIC나 network mode를 지정하지 않았을 때 사용할 기본 네트워크를 준비합니다. `clouds.yaml` 설정을 읽거나 `sdk.WithDefaultNetwork`로 직접 선택할 수 있습니다. SDK가 이름 조회와 서비스 연결을 처리하므로 애플리케이션에서 builder나 resolver를 구현할 필요가 없습니다. 이미지 부팅, 기존 볼륨 부팅, 이미지에서 새 볼륨을 만드는 부팅에 같은 정책을 적용합니다.

## 선택 순서

서버 생성 시 다음 순서로 선택합니다.

1. Create의 `compute.WithNetworks`·`WithNetworkInterfaces`·`WithNetworkMode`
2. Connection의 `sdk.WithDefaultNetwork` 또는 `sdk.WithoutDefaultNetwork`
3. 선택한 cloud의 YAML 기본 네트워크
4. 기본값이 없으면 선택된 Compute microversion 2.37 이상에서 `networks: "auto"`, 이전 버전이나 microversion 미선택 상태에서 networks 생략

명시 NIC나 mode가 있으면 기본 네트워크 조회를 실행하지 않습니다. 빈 NIC 객체 한 개를 지정한 경우도 `[{}]`를 그대로 보내며 default나 auto로 바꾸지 않습니다. 빈 NIC **목록**은 오류입니다. 명시한 auto/none mode는 선택된 Compute 2.37 이상이 필요합니다.

같은 Connection default를 설정하는 옵션은 마지막 값이 우선합니다. `WithoutDefaultNetwork()`는 YAML 기본값도 해제하며 4번 fallback을 사용합니다. Create 옵션은 Connection default를 바꾸지 않고 그 요청에만 적용합니다. 잘못된 Ref 등 앞선 옵션의 오류는 뒤 옵션으로 숨기지 않습니다.

| 입력 | 해석 |
|---|---|
| `sdk.WithDefaultNetwork(resource.Name("private"))` | 정확한 이름 조회; 중복이면 `ErrAmbiguous` |
| `sdk.WithDefaultNetwork(resource.ID("network-id"))` | ID를 직접 전송; Neutron 조회와 endpoint 선택 생략 |
| `sdk.WithoutDefaultNetwork()` | Connection/YAML default를 해제하고 버전에 따른 fallback 사용 |
| YAML `networks[].name` | 이름 **또는** ID를 전체 네트워크 목록에서 정확하게 비교 |

Go Ref는 입력 종류를 명시합니다. UUID 형태인 이름도 `resource.Name`이면 이름으로 조회합니다. YAML의 `name`은 Python 설정 형식에 맞춘 이름/ID 입력이므로 ID처럼 보이는 문자열도 목록에서 확인합니다. 조회 없이 ID를 보내려면 `WithDefaultNetwork(resource.ID(...))`를 사용합니다. 이 옵션들은 이미 인증된 provider를 사용하는 `sdk.FromProvider`에서도 지원합니다.

## clouds.yaml과 Python cloud 호출

다음 설정을 사용하고 인증 URL·계정·project·region을 실제 환경에 맞게 바꿉니다.

```yaml
clouds:
  dev:
    region_name: RegionOne
    auth:
      auth_url: https://identity.example/v3
      username: example-user
      password: example-password
      user_domain_name: Default
      project_name: example-project
      project_domain_name: Default
    networks:
      - name: private
        default_interface: true
```

고정 openstacksdk의 cloud helper는 `network`·`nics`를 생략하면 configured default network를 사용합니다. 자동 floating IP와 대기는 별도 기능이므로 아래 예제에서는 해제합니다.

```python
import openstack

conn = openstack.connect(cloud="dev", compute_api_version="2.37")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    auto_ip=False,
    wait=False,
)
print(server.id)
```

Go에서 `sdk.Connect(ctx, sdk.WithCloud("dev"))`도 같은 YAML 기본값을 읽습니다. `OS_CLOUD=dev`를 설정했다면 `sdk.Connect(ctx)`로 cloud를 선택할 수 있습니다. cloud 선택 없이 `WithAuth`나 `OS_*` 인증만 사용하는 경우에는 YAML 기본 네트워크를 읽지 않습니다.

## 독립 Go 예제

아래 프로그램은 기본적으로 YAML 선택을 사용합니다. `-default-network private`는 Connection의 이름 default를 설정하고, `-no-default-network`는 이를 해제합니다. `-network other-private` 또는 `-network-mode none`은 해당 Create의 선택입니다. network와 mode를 모두 지정하면 뒤에서 적용하는 mode가 우선합니다. 선택한 cloud가 Compute 2.37을 지원해야 합니다. 서버 생성 응답을 반환하며 ACTIVE까지 대기하지 않습니다.

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	image := flag.String("image", "ubuntu", "exact image name")
	flavor := flag.String("flavor", "small", "exact flavor name")
	defaultNetwork := flag.String("default-network", "", "exact Connection default network name")
	noDefaultNetwork := flag.Bool("no-default-network", false, "disable Connection and YAML default network")
	network := flag.String("network", "", "exact network name for this server")
	mode := flag.String("network-mode", "", "Nova auto or none mode for this server")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *cloud, *image, *flavor, *defaultNetwork, *noDefaultNetwork, *network, *mode); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, image, flavor, defaultNetwork string, noDefaultNetwork bool, network, mode string) error {
	opts := []sdk.ConnectionOption{
		sdk.WithCloud(cloud),
		sdk.WithMicroversion(sdk.Compute, "2.37"),
	}
	if defaultNetwork != "" {
		opts = append(opts, sdk.WithDefaultNetwork(resource.Name(defaultNetwork)))
	}
	if noDefaultNetwork {
		opts = append(opts, sdk.WithoutDefaultNetwork())
	}
	conn, err := sdk.Connect(ctx, opts...)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	var createOpts []compute.CreateServerOption
	if network != "" {
		createOpts = append(createOpts, compute.WithNetworks(resource.Name(network)))
	}
	if mode != "" {
		createOpts = append(createOpts, compute.WithNetworkMode(mode))
	}
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name: "web-01", Image: resource.Name(image), Flavor: resource.Name(flavor),
	}, createOpts...)
	if err != nil {
		return err
	}
	fmt.Println(server.ID)
	return nil
}
```

NIC의 세부 입력은 [NIC 선택](server-network-interfaces.md)을 참고하세요.

## 설정 읽기와 상속

Connection을 만들 때 clouds/secure/public 내용을 snapshot하며, 인증·endpoint·TLS의 native parser와 SDK의 network 설정은 같은 bytes를 사용합니다. Connection 생성 후 파일 변경은 기존 Connection에 반영하지 않습니다. 서비스 구성과 network lookup은 계속 lazy로 수행합니다.

기존 clouds 검색과 같이 명시한 `WithCloudFiles`, `OS_CLIENT_CONFIG_FILE`, 표준 검색 경로를 사용합니다. 찾은 clouds 파일 옆의 `secure.yaml`을 읽고, public profile이 필요한 경우의 검색은 기존 native parser의 규칙을 유지합니다.

네트워크 설정은 **secure > clouds 본문 > public profile** 순서로 우선합니다. secure가 profile을 바꾸면 network 상속 대상도 바뀌고, 명시한 빈/null profile은 network metadata 상속을 끊습니다. 인증에서는 기존 native parser의 빈 profile fallback을 보존합니다. 상위 설정에 `networks`가 있으면 목록 전체를 교체하며 하위 항목을 추가하지 않습니다. `networks: []`는 상속한 목록을 비우고, `networks: null`은 설정 오류입니다. 확정된 목록의 `default_interface: true`에서 기본 네트워크를 선택합니다. 기본 후보를 여러 개 설정하면 설정 오류입니다. 명시한 Connection default/disable은 이 설정보다 우선합니다. 설정 파일 자체의 잘못된 목록이나 여러 default 지정은 이 옵션으로 숨기지 않으며 인증 전에 오류로 반환합니다. YAML 이름은 비어 있지 않은 문자열이어야 합니다. Python의 truthy 숫자 이름을 문자열로 변환하는 동작과 공백만 있는 이름은 이 Go 입력 정책에서 허용하지 않습니다. `default_interface` 등 network flag는 bool 또는 문자열을 받으며 문자열은 공백 제거 없이 대소문자를 무시한 `"true"`만 true입니다. 기존 `external_network` 문자열도 default로 읽지만 `networks`와 함께 지정할 수 없습니다.

## 조회, 실패, cache

YAML 또는 `WithNetworkRoles`의 configured default는 [공유 역할 snapshot](../network/network-roles.md)의 `DefaultNetwork`에서 가져옵니다. 첫 탐색에서 Neutron의 모든 network 페이지와 필요한 subnet 페이지를 읽고, 설정 문자열을 network의 `name` **또는** `id`와 비교합니다. 일치하지 않으면 `ErrNotFound`, 여러 network와 일치하면 `ErrAmbiguous`입니다. ID가 일치하는 리소스와 같은 문자열의 이름을 가진 다른 리소스가 있는 경우에도 임의로 하나를 선택하지 않습니다.

성공한 역할 탐색은 getter와 후속 Create가 공유합니다. `ResetNetworkRoles()` 또는 `service.Roles.Reset()` 이후에는 새로 탐색합니다. Network/Subnet의 HTTP·decode·취소 오류와 다른 configured role의 검증 오류도 Nova POST 전에 반환하며, 실패한 탐색은 cache하지 않습니다. 두 discovery flag가 모두 false이거나 network catalog endpoint가 없으면 configured default도 없게 처리하여 Nova의 선택 microversion에 따른 auto/생략 동작을 사용합니다.

`WithDefaultNetwork(resource.ID(...))`는 조회 없이 ID를 전달하고, 명시 `resource.Name(...)`은 기존 exact-name 조회를 생성마다 수행합니다. 이 명시 Name 조회는 역할 cache와 별도입니다. `WithoutDefaultNetwork`, 명시 NIC·mode는 configured default 조회를 우회합니다. Configured default selector 자체가 없을 때도 Go는 탐색을 생략합니다. 이는 default가 없어도 공유 탐색을 실행하는 고정 Python 소스와 다른 조회 정책입니다.

이 선택은 기존 boot volume mapping이나 삭제 정책을 바꾸지 않습니다. 생성 후 `compute.WithWait`가 실패하면 생성한 server와 error를 함께 반환하며 server나 volume을 자동 삭제하지 않습니다.

## Python 비교 범위와 검증

비교 대상은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [cloud create default 분기](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1084-L1110), [설정값 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/cloud_region.py#L1475-L1481), [runtime name/ID 비교](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L252-L264)입니다.

이번에 다루는 것은 server에 사용할 configured default와 공유 역할 조회의 연결입니다. Python의 [shared network discovery](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L110-L365)와 비교하여 Go는 성공한 탐색만 cache하고 오류를 보존하며, 호출자에게 모델 복사본을 반환합니다. [상위 network CRUD](../network/network-mutations.md)의 공유 cache 자동 초기화는 구현했으며, 다른 mutation·외부 변경은 명시 Reset을 사용합니다. 전체 `has_service`·session 정책, advertised microversion bounds/default 검사와 cloud create 전체의 자동 IP·주소 처리·cleanup은 남은 범위입니다. 지원 판정은 계속 부분 구현입니다.

[역할 getter와 공유 cache](../network/network-roles.md)는 외부·내부 family 및 NAT/default 역할을 제공합니다. [소비 경로 테스트](../connection_network_role_consumers_test.go)는 getter 결과의 caller 변경에도 기본 NIC가 유지되고, 연속 Create가 cache를 재사용하며 Reset 이후 새 ID를 선택하는지 확인합니다. Subnet 오류와 다른 역할의 누락이 생성 전에 전달되고 실패 후 재시도되는 경계도 포함합니다.

Python의 동작은 고정 소스에서 확인했습니다. Go의 [compute 테스트](default_network_test.go)와 [Connection HTTP 테스트](../connection_default_network_test.go)는 선택·생략·이름 해석·오류·boot mapping·옵션 재사용을 검증하는 local fixture입니다. Python 예제나 인증한 OpenStack 환경의 생성 작업을 실행했다는 근거로 사용하지 않습니다. YAML 파일 상속과 frozen 인증·region·TLS는 [설정 테스트](../connection_cloud_config_test.go), 전체 페이지 매칭·충돌·뒤 페이지 HTTP 오류·취소·명시 선택 우회는 [YAML HTTP 테스트](../connection_cloud_network_test.go)에서 검증합니다. 최종 실행 근거와 예제 컴파일은 [지원 판정대장](../docs/sdk-support-ledger.md#nova-서버-생성의-기본-네트워크)에 기록합니다.
