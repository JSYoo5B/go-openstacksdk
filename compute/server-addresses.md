# 서버 주소 선택과 interface view

Compute 주소 helper는 조회한 서버에서 사용할 IPv4를 선택하고, `ExpandServerInterfaces`는 public/private/default interface 주소를 계산한 독립 view를 반환합니다. Connection이 네트워크 역할과 Neutron 서비스를 연결하므로 애플리케이션에서 resolver나 interface builder를 만들 필요가 없습니다.

| 호출 | 입력과 결과 |
|---|---|
| `service.GetServerPublicIP(ctx, server, opts...)` | 전달한 Nova 서버 snapshot에서 public IPv4를 선택하여 `(string, error)` 반환 |
| `service.GetServerPrivateIP(ctx, server, opts...)` | 전달한 snapshot에서 internal IPv4를 선택하여 `(string, error)` 반환 |
| `service.ExpandServerInterfaces(ctx, server, opts...)` | 필요한 기존 floating IP 정보를 조회하고 `(*compute.ServerAddressView, error)` 반환 |
| Connection의 같은 이름 3개 method | Compute service와 같은 설정·공유 Network 역할 cache 사용; metadata getter는 Compute endpoint 불필요 |

getter는 서버를 새로 조회하거나 Neutron floating IP를 할당하지 않습니다. `GetServerPublicIP`의 public은 IPv4이며 IPv6는 view의 `PublicIPv6`에서 읽습니다. 주소가 없으면 getter는 빈 문자열과 nil error를 반환합니다. 원본에 빈 주소가 들어 있는 경우도 반환 문자열만으로는 주소 부재와 구분하지 않습니다.

## Python cloud helper와 비교

비교 대상은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. [cloud의 public/private getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L859-L873)는 서버 Resource를 받아 주소 helper로 전달합니다. public getter는 IPv4만 반환합니다.

```python
import openstack
from openstack.cloud import meta

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
print(conn.get_server_public_ip(server))
print(conn.get_server_private_ip(server))

# 고정 소스의 expansion 동작을 비교하기 위한 helper 호출입니다.
# meta는 cloud 내부 helper 모듈이며 별도의 public proxy operation이 아닙니다.
expanded = meta.add_server_interfaces(conn, server)
print(expanded["public_v4"], expanded["public_v6"])
print(expanded["private_v4"], expanded["interface_ip"])
```

Python의 [add_server_interfaces](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L528-L563)는 입력 서버를 수정합니다. Go는 `ServerAddressView`를 새로 만들어 원본 `compute.Server`와 주소 데이터를 보존합니다. raw Get/Find/List와 일반 Create/Wait의 반환 모델에 이 expansion을 자동으로 적용하지 않습니다.

## 독립 Go 예제

아래 main은 clouds.yaml로 연결하고 기존 서버를 조회합니다. Connection에 정책을 설치한 뒤 Compute helper를 호출합니다. `-server-id`를 지정하면 `-server`를 ID로 사용하고, 지정하지 않으면 정확한 이름으로 조회합니다. 주소 계산과 기존 association 조회만 실행합니다.

예제의 TCP probing은 명시적으로 해제했습니다. 후보의 실제 접속 가능성을 참고하여 고르려면 `-probe`를 지정합니다. `-private`는 서버 접속에 internal IPv4를 우선하는 cloud 정책이며 endpoint interface 선택과 별개입니다.

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server", "web-01", "exact server name, or ID with -server-id")
	serverID := flag.Bool("server-id", false, "treat -server as an ID")
	private := flag.Bool("private", false, "prefer internal IPv4 for server access")
	forceIPv4 := flag.Bool("force-ipv4", false, "omit IPv6 from calculated access fields")
	probe := flag.Bool("probe", false, "try TCP reachability among multiple candidates")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *cloud, *server, *serverID, *private, *forceIPv4, *probe); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, serverNameOrID string, byID, private, forceIPv4, probe bool) error {
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(cloud),
		sdk.WithServerAddressPolicy(
			compute.WithPrivateCloud(private),
			compute.WithForceIPv4(forceIPv4),
			compute.WithFloatingIPSource(compute.FloatingIPNeutron),
			compute.WithAddressReachability(probe),
		),
	)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	ref := resource.Name(serverNameOrID)
	if byID {
		ref = resource.ID(serverNameOrID)
	}
	server, err := service.Servers.Find(ctx, ref)
	if err != nil {
		return err
	}
	view, expandErr := service.ExpandServerInterfaces(ctx, server)
	if view != nil {
		fmt.Printf("server=%s public4=%q public6=%q private4=%q interface=%q access4=%q access6=%q\n",
			view.ServerID, view.PublicIPv4, view.PublicIPv6, view.PrivateIPv4,
			view.InterfaceIP, view.AccessIPv4, view.AccessIPv6)
		if view.SupplementalError != nil {
			log.Printf("supplemental address lookup: %v", view.SupplementalError)
		}
	}
	if expandErr != nil {
		return fmt.Errorf("expand server addresses: %w", expandErr)
	}

	// 두 getter는 원래 Nova snapshot을 사용합니다.
	publicIP, err := service.GetServerPublicIP(ctx, server)
	if err != nil {
		return fmt.Errorf("public address: %w", err)
	}
	privateIP, err := service.GetServerPrivateIP(ctx, server)
	if err != nil {
		return fmt.Errorf("private address: %w", err)
	}
	fmt.Printf("snapshot public4=%q private4=%q\n", publicIP, privateIP)
	return nil
}
```

두 getter는 원래 Nova snapshot을 사용하므로 Neutron의 기존 floating IP를 보충한 view와 값이 다를 수 있습니다. 계산된 최종 결과가 필요하면 `view.PublicIPv4`·`PrivateIPv4`를 사용합니다. 반환한 view가 있는 경우 expand error가 함께 발생해도 이미 읽은 주소가 남아 있으므로, 위 예제는 view와 진단을 출력한 뒤 오류를 반환합니다.

Connection에서 직접 호출할 때는 위 예제의 `service.ExpandServerInterfaces`·`GetServerPublicIP`·`GetServerPrivateIP`를 각각 `conn.ExpandServerInterfaces`·`GetServerPublicIP`·`GetServerPrivateIP`로 바꿉니다. server 조회는 같은 `service.Servers.Find`를 사용합니다. 세 method 모두 `(ctx, server, options ...compute.ServerAddressOption)` 형태입니다.

## 주소와 interface 선택 순서

public IPv4는 [source의 순서](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L215-L269)를 따릅니다. external discovery가 false이면 빈 값을 반환하고, 그렇지 않으면 기존 `AccessIPv4`, external IPv4 역할의 실제 network 이름, floating IPv4, 이름이 `public`인 network, 마지막 IPv4 분류 fallback 순서로 선택합니다. `AccessIPv4`가 있으면 addresses 해석이나 역할 탐색을 생략합니다.

private IPv4는 internal discovery가 false이면 빈 값을 반환합니다. [internal IPv4 역할](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L147-L190)의 fixed 주소를 먼저 찾고, 두 번째 pass에서 tag를 제한하지 않은 이전 Nova 응답도 처리합니다. 이 pass는 같은 MAC의 floating 주소도 선택할 수 있습니다. 첫 IPv4 floating 주소에 MAC이 있으면 같은 MAC의 주소를 고릅니다. 그 뒤 이름이 `private`인 network의 fixed/MAC 선택과 MAC 제한 없는 최종 fallback을 적용합니다. MAC 문자열 비교는 대소문자를 구분하며 누락과 명시 빈 값을 구분합니다. private getter 자체가 Neutron port를 조회하지는 않습니다.

같은 선택 단계 안에서는 floating 주소가 먼저이고 나머지 주소는 목록 순서를 유지합니다. Python dictionary의 network 삽입 순서는 native Go map에 남지 않으므로 Go는 network 이름을 정렬해 안정적인 순서를 사용합니다. 원래 순서를 알고 있다면 `compute.WithAddressNetworkOrder("private", "public")`로 명시할 수 있습니다. 지정하지 않은 network는 정렬 순서로 뒤에 붙습니다. 중복 키는 입력 오류이며, 빈 문자열 network key는 유효한 주소 label로 지정할 수 있습니다.

view의 `InterfaceIP`는 [configured default interface](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L348-L405)의 주소를 먼저 사용합니다. local IPv6가 있고 IPv4 강제가 없으면 default network의 IPv6를 먼저 시도하고 IPv4로 돌아옵니다. 빈 default IPv6가 발견되면 default IPv4 선택으로 넘어가지 않고 최종 interface fallback을 사용합니다. default 주소가 없거나 비었으면 private cloud의 private IPv4, 사용 가능한 local IPv6 환경의 public IPv6, public IPv4 순서입니다. `AccessIPv4`는 private cloud이고 private IPv4가 있으면 이를 사용하며, 그 외에는 public IPv4입니다. `AccessIPv6`는 계산한 public IPv6입니다.

`PublicIPv6`는 원본 `AccessIPv6` 또는 Nova IPv6 주소에서 선택하며 internal/external IPv6 역할 목록으로 다시 분류하지 않습니다. local IPv6와 force 정책은 default/interface 선호에 적용하고, `WithForceIPv4(true)`는 view의 public/access IPv6 계산을 생략합니다. 원본 IPv6 주소 목록은 보존합니다. source docstring에 있는 `private_v6`는 고정 구현에서 계산되지 않으며 Go view도 `PrivateIPv6` 필드를 제공하지 않습니다.

마지막 public IPv4 fallback은 단순한 RFC1918 제외가 아니라 Python의 `ipaddress.is_private` 분류입니다. Python SDK pin은 Python interpreter 버전을 고정하지 않으므로 Go는 [CPython 3.13의 IPv4 분류](https://docs.python.org/3.13/library/ipaddress.html#ipaddress.IPv4Address.is_private)를 명시적으로 사용합니다. shared 주소 공간과 multicast는 이 마지막 fallback에서 제외되지 않습니다. 이 분류에 따른 후보 선택은 실제 라우팅이나 SSH 성공을 보장하지 않습니다.

## 주소 정책과 호출별 옵션

`ServerAddressPolicy`는 `compute.PrepareServerAddressPolicy`로 만드는 concrete 설정입니다. Connection에서 `sdk.WithServerAddressPolicy(opts...)`로 설치하고, 특정 호출에는 같은 Compute 옵션으로 기본 정책을 덮어씁니다. 옵션은 나열한 순서로 적용하며 잘못된 앞선 옵션을 뒤 옵션으로 숨기지 않습니다. reusable 설정과 network 순서는 SDK가 복사합니다.

| Compute 옵션 | 역할과 기본 정책 |
|---|---|
| `WithPrivateCloud(bool)` | cloud 내부 접속을 선호할지 선택; 기본 false |
| `WithForceIPv4(bool)` | 계산한 IPv6/interface 선호 제한; 기본 false |
| `WithLocalIPv6(bool)` | host의 IPv6 감지 결과를 명시값으로 대체 |
| `WithFloatingIPSource(FloatingIPSource)` | Neutron/Nova/none source 지정; 기본 neutron |
| `WithAddressReachability(bool)` | 여러 후보의 best-effort TCP 선택; 기본 true |
| `WithAddressProbeBudget(time.Duration)` | 후보별 probe 예산; 기본 5초, 양수 필수 |
| `WithAddressProbePort(int)` | probe TCP port; 기본 22, 1–65535만 허용 |
| `WithAddressNetworkOrder(names ...string)` | map에서 사라진 알려진 network 순서를 명시 |

local IPv6는 정책을 준비할 때 host interface에서 loopback/link-local이 아닌 IPv6 주소로 감지합니다. ULA도 감지 대상입니다. `WithLocalIPv6(true)`가 있어도 `WithForceIPv4(true)`가 계산 정책에서 우선합니다. source는 `compute.FloatingIPNeutron`·`FloatingIPNova`·`FloatingIPNone` 중 하나를 사용하며 대소문자가 다른 raw 문자열이나 알 수 없는 값은 오류입니다.

TCP 선택은 [고정 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L272-L327)처럼 후보가 여러 개이고 그 주소가 현재 cloud 접근 정책상 도달 대상인 경우에만 실행합니다. 후보 하나는 probe하지 않습니다. 순서대로 연결 가능한 첫 후보를 선택하고, 모두 실패하면 첫 후보를 반환합니다. 후보별 기본 예산은 5초이고 caller context가 전체 작업을 제한합니다. context 취소·기한 초과는 오류로 반환합니다. 이는 추가 네트워크 접속을 실행하는 정책이므로 주소 데이터의 순서만 사용하려면 `WithAddressReachability(false)`를 지정합니다.

네트워크 역할의 internal/external 사용 flag는 [공유 역할 설정](../network/network-roles.md)에서 지정합니다. `WithPrivateCloud`가 internal/external discovery flag를 변경하지는 않습니다. `FloatingIPNone`은 기존 서버에 있는 floating 주소를 getter에서 지우지 않으며 신규 보충 조회를 해제합니다.

## clouds.yaml와 환경 설정

주소 정책은 network 목록과 함께 public profile → cloud → secure cloud의 파일 snapshot을 사용합니다. `private`, `force_ipv4`, `prefer_ipv6`, `floating_ip_source`와 hyphen 별칭을 읽습니다. `floating_ip_source`는 대소문자를 정규화하며 null·빈 값·none은 보충을 해제합니다. 알 수 없는 source와 flag는 bool/string/null을 허용하며 null은 false입니다. 숫자·목록·object flag 값은 입력 오류입니다.

```yaml
clouds:
  dev:
    private: true
    force_ipv4: false
    prefer_ipv6: true
    floating_ip_source: neutron
    networks:
      - name: private
        nat_destination: true
```

root `client`의 `prefer_ipv6`/`prefer-ipv6`, `force_ipv4`/`broken-ipv6`는 base·secure를 병합하고 underscore 키가 우선합니다. `OS_PREFER_IPV6`·`OS_FORCE_IPV4`는 global 값을 덮어쓰며 명시 빈 환경 값은 false입니다. global prefer=false이면 force=true가 된 뒤, cloud의 명시 force=false가 이를 해제할 수 있습니다. cloud prefer=false는 최종 force=true를 적용합니다. `FromProvider`는 이 환경·파일을 읽지 않고 concrete 정책 또는 기본값을 사용합니다.

Go는 bool과 문자열 true/false를 정규화합니다. 고정 Python 소스는 cloud private/force/prefer에 raw truthiness를 적용하여 문자열 'false'도 참으로 볼 수 있으므로 이 차이를 명시합니다. `sdk.WithServerAddressPolicy(...)`는 파일·환경 정책 전체를 교체하고 호출 옵션은 해당 필드만 다시 덮어씁니다. 이 typed 최종 override는 Python constructor와 cloud overlay의 우선순위와 다릅니다. cloud 설정에서도 underscore 키가 hyphen 별칭보다 우선하므로 두 spelling이 서로 다른 layer에 함께 있으면 단순 layer 우선순위와 다를 수 있습니다. 이 결정적 Go 별칭 정책도 Python normalize-keys 순서와 차이가 있습니다. root 설정의 모든 서비스·session 정책과 env-only Python raw-string 동작이 대응된 것은 아닙니다.

## 기존 floating IP 정보 보충과 오류

[고정 supplemental helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L474-L525)와 비교하는 주소 보충은 상태가 정확히 `ACTIVE`인 서버에 적용합니다. Nova에 기존 non-IPv6 floating row가 있으면 추가 조회를 생략합니다. Neutron에서 해당 서버의 port 목록을 읽고, source가 neutron이면 port별 Neutron floating IP 목록을 읽습니다. source가 nova이면 port별 Compute `/os-floating-ips` 목록을 읽어 정규화한 뒤 `port_id`로 걸러냅니다. source nova도 network 서비스가 있어야 이 보충을 수행합니다. 목록의 다른 서버 port나 다른 port의 IP는 선택하지 않습니다. floating IP의 fixed 주소가 Nova의 non-IPv6 주소와 일치할 때 그 network label에 floating 주소와 port MAC을 추가합니다. Neutron 목록은 후속 페이지까지 처리합니다.

Nova 응답의 `floating_ip_address`·`fixed_ip_address`는 canonical 필드가 없을 때 legacy `ip`·`fixed_ip`로 보완합니다. missing/null fixed 값은 빈 문자열 fixed 주소에 매칭하지 않습니다. 매칭한 항목의 floating 주소가 missing/null이면 Go typed 주소 계약에 따라 원래 응답을 가진 오류와 부분 view를 반환합니다. 명시 canonical null은 legacy 값보다 우선하며 `port_id`가 없는 Nova 항목은 연결할 port가 없으므로 제외합니다. Nova endpoint의 정상 404는 목록이 없는 것으로 처리합니다. 이 주소 조회 지원은 Nova floating IP 할당/연결 workflow 전체의 지원 판정과 별개입니다.

보충 조회의 정상 HTTP 거부와 직접적인 DNS·접속 실패(net.Error)는 view의 `SupplementalError`에 남기고 이미 확보한 주소로 계산을 계속합니다. 앞 페이지의 보충 주소가 있다면 이를 보존합니다. context 취소와 취소 원인, 입력/서비스 source 오류, 성공 응답의 decode/read/Close 실패와 다른 오류는 반환 error입니다. source가 넓은 `SDKException`을 삼키는 동작과 달리 Go는 이 fatal 경계를 드러냅니다. 기존 Nova 입력과 이미 얻은 view를 임의로 변경하거나 삭제하지 않습니다.

`Addresses`는 network별 `ServerAddress` 목록이며 family, type, MAC과 원본 JSON 필드를 보존합니다. `MACPresent`와 `Fields`로 MAC의 누락·null·빈 문자열 및 확장 필드를 확인할 수 있습니다. 주소의 null/map과 network의 null/빈 목록을 보존합니다. Go typed 주소 row에는 version 키와 문자열 addr가 필요하며 null addr는 입력 오류입니다; Python의 동적 null 주소와 차이가 있습니다. `Supplemental`은 Neutron/Nova IP 목록에서 추가된 행인지 나타내며 원래 Nova server 응답에서 본 주소와 구분합니다. 반환 view를 수정해도 입력 서버나 후속 호출의 주소 snapshot은 변하지 않습니다. 일반 native 모델은 원래 network wire 순서와 모든 원본 Resource 상태를 복원하지 못하므로 이 view가 Python Resource 모델 전체를 대체하는 것은 아닙니다.

## 서버 생성과 자동 IP의 범위

이 API는 이미 존재하는 주소를 선택하거나 기존 floating IP association 정보를 보충합니다. [CreateWithFloatingIP](create-with-floating-ip.md)는 명시적으로 서버를 만들고 실제 server/IP ACTIVE까지 기다려 하나의 Neutron floating IPv4를 연결하는 별도 workflow입니다. 두 작업은 독립적이며 주소 helper에 할당·연결·삭제 책임을 넣지 않습니다.

Python cloud create/wait의 `auto_ip` 필요 여부, pool/명시 ips dispatch, private/source/서비스 가용성별 자동 분기, 오류 서버 cleanup과 Nova 주소 갱신 대기는 여전히 별도 범위입니다. `ExpandServerInterfaces`의 현재 association 조회는 새 assignment가 Nova 주소에 나타날 때까지 기다리는 동작과 다릅니다. 일반 Get/Create/Wait 전체나 cloud의 서비스 가용성·session·설정 loader 전체가 완료되었다고 판정하지 않습니다.

## 검증과 지원 판정

이 문서의 Python 비교는 고정 소스 정적 검토 기준입니다. 주소 선택 local fixture는 [server_address_selection_test.go](server_address_selection_test.go)에 있으며 역할/MAC/fixed fallback, public 우선순위, use flag와 AccessIPv4 short circuit, CPython 3.13 fallback 분류, TCP 후보 선택과 context, 입력 오류의 dependency 호출 선행 방지를 검증하도록 작성되어 있습니다. 주소 보충·부분 결과·Nova source는 [expansion 테스트](server_address_expansion_test.go), 설정·공유 cache와 endpoint 생략은 [Connection 테스트](../connection_server_addresses_test.go)와 [설정 테스트](../connection_server_address_config_test.go)에 있습니다. 실제 실행 근거와 최종 문서 예제 컴파일은 [지원 판정대장](../docs/sdk-support-ledger.md)에 별도로 기록합니다.

public cloud getter 두 개에 대한 지원 판정은 기존 native transport 완료나 `meta` 함수 이름으로 자동 승격하지 않습니다. source catalog가 수집하지 않은 module free/private helper는 실제 public getter와 관련 lifecycle의 부분 계약·차이·남은 범위에 기록합니다.

[자동 floating IPv4](server-automatic-ip.md)는 위 주소 기반과 guarded 역할 snapshot을 기존 서버의 조건부 Neutron assignment에 연결합니다. 일반 주소 helper는 allocation을 하지 않으며, 자동 메서드의 최종 완료는 합성 Supplemental 행 대신 raw Nova 응답에서 정확한 tagged IPv4로 확인합니다. 일반 Get/Create/Wait에 자동 통합하는 계약은 계속 남습니다.
