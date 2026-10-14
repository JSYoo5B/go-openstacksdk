# Cloud 서브넷 생성: Python과 Go

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `conn.create_subnet`에 대응하는 호출은 `conn.CreateSubnetRecord(ctx, network, options...)`입니다. 네트워크·서브넷풀 이름 조회, 기본값, 입력 충돌과 IP 버전 변환, Source Body/Header 속성 번역, POST와 캐시 갱신을 SDK가 소유합니다. 애플리케이션이 builder/interface나 여러 HTTP 분기를 작성할 필요가 없습니다. Gophercloud 기준은 `v2.15.0`입니다.

| Python | Go | 기본값·의미 |
| --- | --- | --- |
| `create_subnet(network, cidr="10.61.0.0/24")` | `CreateSubnetRecord(ctx, network, WithSubnetCreateCIDR("10.61.0.0/24"))` | `ip_version=4`, `enable_dhcp=False` |
| `ip_version="6"`, `enable_dhcp=True` | `WithSubnetCreateIPVersion("6")`, `WithSubnetCreateDHCP(true)` | 문자열 IP 버전은 Source처럼 정수 변환 |
| `subnet_name`, `tenant_id` | `WithSubnetCreateName`, `WithSubnetCreateTenantID` | 이름과 조회 필터를 SDK가 연결 |
| `gateway_ip`, `disable_gateway_ip=True` | `WithSubnetCreateGatewayIP`, `WithSubnetCreateDisableGateway(true)` | 비활성화는 명시적인 `gateway_ip:null` |
| `use_default_subnetpool=True` | `WithSubnetCreateDefaultSubnetPool(true)` | CIDR 없이 기본 pool 요청 |
| `subnetpool_name_or_id`, `prefixlen` | `WithSubnetCreateSubnetPool`, `WithSubnetCreatePrefixLength` | pool 조회와 ID 번역 |
| 전체 keyword dict | `WithSubnetCreateJSON(json.RawMessage(...))` | 선언된 formal과 임의 extension kwargs를 분리 |
| Source Subnet getter view | `result.Resource` | 기본값·seed와 실제 응답을 병합 |
| 실제 POST | `Wire`, `Envelope`, `Header`, `StatusCode` | 서버가 보내지 않은 seed를 실제 응답으로 만들지 않음 |
| 별도 lookup 단계 | `Network`, `SubnetPool` | 각각의 owned Resource/Wire/응답 증거 |

서비스를 이미 선택했다면 `service, err := conn.Network(ctx)` 뒤 `service.CreateSubnetRecord(...)`를 사용할 수 있습니다. 낮은 단계의 `service.API.Subnets.CreateCloudRecord`는 이미 번역된 Source attrs와 library-owned order 옵션을 받는 helper입니다. 기존 native `Create`, `CreateWithAttributes`, 기본 `FindRecord`의 required 검사·응답 모델·호출 정책은 유지됩니다. 이번 network/pool `FindSubnetSourceRecord`는 이 cloud 호출에 필요한 고정 조회만 제공하며 일반 raw Proxy.Find 전체의 지원 판정을 올리지 않습니다.

## 기본값과 실행 순서

Source처럼 network lookup을 먼저 실행합니다. `tenant_id`가 None/null이 아닌 경우 false·0·빈 문자열도 member GET query에 전달합니다. 완성된 GET을 clean400/403/404로 거절했을 때만 모든 list page를 읽어 exact ID/name을 찾습니다. 반환한 네트워크 ID가 POST 기본 `network_id`입니다. fallback list에서 unknown tenant_id query를 버리는 것도 Source 규칙이며, client가 소유권을 추정해 추가 필터링하지 않습니다. 누락·후반 중복·page 오류는 POST로 진행하지 않습니다.

그 뒤 `disable_gateway_ip`와 `gateway_ip`가 모두 truthy인지, CIDR/pool 선택 조합을 검사합니다. CIDR과 pool을 동시에 쓰거나 default pool과 named pool을 함께 쓰면 오류입니다. CIDR도 pool도 없으면 오류입니다. named pool이 truthy이면 pool lookup을 수행하고, 그 이후 문자열 `ip_version`을 Python `int()`와 같은 부호·공백·Unicode decimal digit·underscore 규칙으로 변환합니다. `"4.0"`·`"²"`는 오류입니다. 단순히 4/6만 허용하는 SDK 검사를 추가하지 않으며 실제 서버가 값 범위와 driver 지원을 검사합니다.

Body는 `network_id`, `ip_version=4`, `enable_dhcp=false`부터 시작합니다. 임의 kwargs가 기본값을 덮은 뒤, truthy formal optional 값이 CIDR·name·tenant·allocation pools·gateway·DNS·routes·IPv6 mode·prefix를 덮습니다. falsey formal 값은 Body에 추가하지 않습니다. `disable_gateway_ip`가 truthy이면 gateway를 null로 보내고, default pool은 true로, named pool은 조회된 ID로 번역합니다. `gateway_ip=""`는 cloud formal에서 생략되므로 native builder의 빈 문자열→null 정책과 구분됩니다.

반환 descriptor만 bool/int/list로 변환합니다. 예를 들어 `enable_dhcp:"False"`는 원문 string으로 POST되고 getter는 true이며, `allocation_pools:"ab"`는 string으로 POST되고 getter는 `["ab"]`입니다. null은 보존됩니다. canonical/wire 별칭은 입력·응답 순서의 마지막 값이 승리합니다. Header의 가려진 invalid 값도 마지막 winning alias가 올바르면 실패시키지 않습니다.

`WithSubnetCreateAttribute`는 null/false/0/list/object와 임의 extension을 표현합니다. `WithSubnetCreateJSON`은 전체 keyword map을 대체하고 JSON member 순서를 보존합니다. 뒤의 concrete 옵션은 해당 키만 대체합니다. `WithSubnetCreateOpts(SubnetCreateOpts{Fields:...})`는 map과 bytes를 복제하며 map-only 미지정 순서는 Go lexical 순서입니다. nil CurrentLocation은 이미 SDK가 넣은 기본 context를 유지합니다. 추가 concrete 함수는 AllocationPools, DNSNameservers, HostRoutes, IPv6RAMode, IPv6AddressMode입니다. `network_name_or_id`·직접 self 중복, invalid JSON·UTF-8·nil 옵션은 HTTP 전에 거부합니다.

Source의 `__conflicting_attrs`는 truthy object일 때 한 번 복원하고 falsey 값은 ordinary unknown Body/seed에 유지합니다. `base_path`는 복원 전에 Proxy formal로 소비하여 실제 POST 경로를 바꾸며, 뒤에 복원된 같은 키는 Body입니다. 고정 class getter는 `/subnets`입니다. Constructor cls/connection/_synchronized와 Proxy resource_type 충돌은 network 조회 뒤 POST 전에 오류입니다. `microversion` constructor 값은 Body에 보내지 않고 create가 선택한 client version을 사용합니다. `if_match`는 Body가 아닌 Header이며 null은 `None`, string list는 comma-space 문자열입니다. 응답 Header가 있으면 실제 값이 getter를 덮고, 없으면 요청 seed가 유지됩니다.

## 독립 Python main

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--network", required=True)
parser.add_argument("--kwargs", default='{"cidr":"10.61.0.0/24"}')
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    subnet = conn.create_subnet(args.network, **kwargs)
    print(json.dumps(subnet.to_dict(), indent=2))
```

`python main.py --cloud dev --network private --kwargs '{"cidr":"10.61.0.0/24","subnet_name":"app","disable_gateway_ip":true}'`

`python main.py --cloud dev --network private --kwargs '{"subnetpool_name_or_id":"ipv6-pool","ip_version":"6","prefixlen":"64"}'`

## 독립 Go main

[설치 안내](../docs/install.md)의 module/clouds.yaml 준비 후 main.go로 저장합니다. 두 main은 실제 생성 POST를 수행합니다. Go는 JSON 숫자를 float64로 먼저 바꾸지 않고 raw keyword bytes를 SDK에 전달합니다.

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
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func run(ctx context.Context, cloud, networkName, keywords string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	result, err := conn.CreateSubnetRecord(ctx, networkName,
		network.WithSubnetCreateJSON(json.RawMessage(keywords)))
	if result != nil {
		if result.Network != nil {
			fmt.Fprintf(os.Stderr, "network lookup status=%d\n", result.Network.StatusCode)
		}
		if result.SubnetPool != nil {
			fmt.Fprintf(os.Stderr, "subnetpool lookup status=%d\n", result.SubnetPool.StatusCode)
		}
		if result.StatusCode != 0 {
			fmt.Fprintf(os.Stderr, "subnet POST status=%d request-id=%q\n",
				result.StatusCode, result.Header.Get("X-Request-Id"))
		}
	}
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result.Resource, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	networkName := flag.String("network", "", "literal network ID or exact name")
	keywords := flag.String("kwargs", `{"cidr":"10.61.0.0/24"}`, "Source cloud keyword JSON object")
	flag.Parse()
	if *networkName == "" {
		fmt.Fprintln(os.Stderr, "network is required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *cloud, *networkName, *keywords); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

`go run . --cloud dev --network private --kwargs '{"cidr":"10.61.0.0/24","subnet_name":"app","disable_gateway_ip":true}'`

`go run . --cloud dev --network private --kwargs '{"subnetpool_name_or_id":"ipv6-pool","ip_version":"6","prefixlen":"64"}'`

## 응답·권한·지원 경계

Resource는 Source 기본값과 seed를 포함하므로 빈/partial 응답에서도 입력의 IP 버전·DHCP·CIDR을 유지합니다. Wire와 Envelope는 실제 응답뿐이며, response id:null은 seed를 덮습니다. ordered response canonical aliases와 실제 If-Match Header도 반영합니다. Connection은 SDK가 가진 current location을 옵션에 공급하고 standalone 서비스의 미지정 location은 null입니다. 실제/foreign project ID로 computed location을 만들지만 응답의 forged location/microversion은 Wire에서만 보존합니다.

100..399를 Source 성공 상태로 받아들이는 새 lane이며 기존 native 상태 정책은 바꾸지 않습니다. clean rejected POST도 실제 status/body/header를 유지합니다. physical read·Close·취소·selected API/provider/route/header source 오류는 fallback·missing·성공으로 숨기거나 retry/reauth에 넘겨 replay하지 않습니다. Native401의 `ErrUnableToReauthenticate`는 Unwrap이 없으므로 `ErrOriginal`의 value `ErrUnexpectedResponseCode`와 `ErrReauth`를 별도로 확인합니다. clean 재시도·token renewal은 유지합니다. 수락된 POST 뒤 실패는 실제 `ResponseError` receipt를 남기고 성공 Resource를 반환하지 않습니다. 실제 수락 status 뒤에는 private 네트워크 역할 캐시를 무효화하므로 decode/read/Close/cancel 오류에도 이전 subnet/gateway topology가 유지되지 않습니다. 거절된 POST는 성공 캐시를 무효화하지 않습니다.

[Neutron 27 subnet 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnet.py#L60-L118)의 create_subnet은 project scope의 ADMIN_OR_NET_OWNER_MEMBER입니다. 읽기 가능한 네트워크라는 이유만으로 생성 권한을 보장하지 않습니다. `segment_id`와 `service_types` 생성은 별도 admin 정책이며 tags에도 별도 정책이 적용됩니다. 다른 프로젝트 tenant, pool 접근, CIDR/IP·gateway·allocation·driver 검증과 배포 policy의 최종 수락은 서버가 결정합니다. SDK가 역할을 확대하지 않습니다.

Go의 network/named-pool 참조는 검증한 문자열입니다. slash/percent는 한 segment로 escape하고 blank/control/traversal 참조를 거부합니다. base_path는 literal service-relative 경로만 받으며 Source의 arbitrary URI `%` formatting·query·외부 경로와 달라집니다. Unicode integer 변환은 arbitrary precision이고 CPython의 설정 가능한 digit-limit 자체는 구현하지 않습니다. ordered JSON/개별 옵션을 제외한 bulk map의 alias 순서는 lexical이며 JSON wire key 순서는 보장하지 않습니다. raw JSON numeric tokens와 Python float serialization/runtime-domain 차이는 명시한 Go 정책입니다.

닫힌 catalog 대상은 cloud create_subnet 한 선언입니다. mutable Resource/Container/class attr dispatch·arbitrary computed members·custom runtime objects·일반 raw Find/Proxy 생성과 SDK-R1·C1·S1 기반은 별도 remaining입니다. 일반 class/member override 전체를 unknown JSON Body 보존과 같다고 주장하지 않습니다. 전체 SDK supported 증가량은 0입니다.

실제 pinned cloud get_network/get_subnetpool/create_subnet·Neutron Proxy.find/create·Resource.find/new/create를 CPython3.14.8에서 실행한 127-case Source receipt가 있습니다. HTTP adapter만 fixture로 바꿨으며 live cloud/인증/ACL/배포 수락 증거가 아닙니다. cloud13+leaf5 테스트 그룹은 기존 public Gophercloud testhelper 기반 testcloud/nativeSubnet fixture와 실제 body observer를 재사용합니다. physical GET/POST 24 subcases와 accepted cache5lanes를 포함하며 Go/race·정확 main build·저장소 전체 gate 결과는 root 실행 ledger로 별도 확인합니다. 이 문서는 미실행 PASS를 주장하지 않습니다.

Source: [cloud create_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2401-L2569), [Subnet descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet.py), [Proxy._create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L771-L800).

Root 최종 집중 race 검증에서 이 cloud13+leaf5 테스트 그룹이 모두 PASS했습니다. 통합 실행은 8개 package의 신규67그룹/249 subcase를 포함하며 기록은 `/private/tmp/go-openstacksdk-cycle7-focused-pass.jsonl`입니다. 정확 예제 main build와 전체 저장소 gate는 별도 root ledger를 따릅니다. 이 결과는 live cloud 인증·정책 수락이나 전체 SDK supported 완료를 뜻하지 않습니다.
