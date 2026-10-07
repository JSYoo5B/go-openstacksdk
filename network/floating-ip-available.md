# Neutron의 free Floating IP 조회·할당

`service.FloatingIPs.Available`은 선택한 외부 network와 현재 project에서 **첫 free IP를 반환하거나 새 IP를 할당**합니다. Connection이 cloud 설정·네트워크 역할·필요한 서버 이름 조회를 연결하므로 애플리케이션에서 builder나 resolver를 구현할 필요가 없습니다. 이 문서는 Network의 Neutron lower API를 설명합니다. 설정에 따른 Neutron/Nova 선택과 NotFound 전환은 [Connection·Compute Available](../compute/floating-ip-available.md)을 사용합니다.

free IP를 재사용하면 `Server`를 지정했어도 연결하지 않습니다. 새로 할당할 때만 optional server의 fixed IPv4를 선택해 Neutron POST에 association을 넣을 수 있습니다. 반환이 ACTIVE나 서버 주소 관측을 의미하지 않으며, PUT·IP wait·raw Nova wait·cleanup은 수행하지 않습니다. 서버에 확실히 연결하는 작업은 [Ensure](floating-ip-ensure.md), 기존 IP 하나를 연결하는 작업은 [Attach](floating-ip-attach.md)를 사용합니다.

## 요청과 옵션

`service.FloatingIPs.Available(ctx, input, options...)`는 `network.AvailableFloatingIPRequest`와 `network.AvailableFloatingIPOption`을 받고 `(*network.FloatingIPAvailability, error)`를 반환합니다. 요청은 다음 concrete 값으로 구성합니다.

| 입력 | 의미 |
|---|---|
| `Networks []resource.Ref` | 사용자 순서의 외부 floating network 후보. `Name`은 이름, `ID`는 ID만 비교 |
| 빈 `Networks` | [공유 역할](network-roles.md)의 첫 `ExternalIPv4Floating`; 목록이 비면 enabled router의 첫 external gateway |
| `Server resource.Ref` | optional 서버. zero는 unattached allocation; 이름 조회·ports 선택은 새 allocation에만 필요 |
| `network.WithAvailableProject("project-id")` | **free 재사용의 project 필터만** 교체. 새 allocation의 project_id를 지정하거나 인증 scope를 바꾸지 않음 |
| `network.WithAvailableFixedAddress("10.0.0.10")` | optional server의 새 allocation에서 fixed IPv4 선택 |
| `network.WithAvailableNATDestination(resource.Name("private"))` | 여러 서버 port의 새 allocation에서 NAT network 선택 |
| `network.WithAvailableTimeout(time.Minute)` | lookup·역할·ports·allocation을 함께 제한하는 SDK deadline |
| `network.WithUnlimitedAvailableTimeout()` | SDK deadline 해제. caller context와 transport timeout은 유지 |

기본 SDK deadline은 없습니다. `Available`의 시간 옵션은 IP ACTIVE waiter가 아니라 작업 전체의 예산입니다. 같은 종류의 옵션은 마지막 값으로 교체하며 옵션은 한 번 준비합니다. 잘못된 option·nil option·invalid Ref는 HTTP 전에 오류입니다. network 후보 slice는 option 적용 전에 복사합니다. `PrepareAvailableFloatingIPOptions`와 `WithAvailableFloatingIPPolicy`는 SDK workflow가 준비한 값을 재사용하기 위한 선택적인 concrete 기능이며 사용자가 builder를 구현해야 하는 경로가 아닙니다.

## 독립 Go 예제

SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장합니다. `-networks`는 쉼표로 구분한 정확한 network **이름** 후보이며 빈 값이면 기본 역할/router를 사용합니다. `-server-id`가 없으면 unattached allocation입니다. 이 프로그램은 실제 Available 작업을 호출하므로 free 후보가 없으면 새 IP를 생성할 수 있습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "os"
    "strings"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/network"
    "gophercloudsdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    candidates := flag.String("networks", "public", "ordered network names")
    server := flag.String("server-id", "", "optional existing server ID")
    project := flag.String("project", "", "optional reuse filter only")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *candidates, *server, *project); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, candidates, serverID, project string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.Network(ctx)
    if err != nil {
        return err
    }
    input := network.AvailableFloatingIPRequest{}
    if candidates != "" {
        for _, name := range strings.Split(candidates, ",") {
            input.Networks = append(input.Networks, resource.Name(strings.TrimSpace(name)))
        }
    }
    if serverID != "" {
        input.Server = resource.ID(serverID)
    }
    options := []network.AvailableFloatingIPOption{
        network.WithAvailableTimeout(time.Minute),
    }
    if project != "" {
        options = append(options, network.WithAvailableProject(project))
    }
    result, err := service.FloatingIPs.Available(ctx, input, options...)
    if result != nil {
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        if encodeErr := encoder.Encode(result); encodeErr != nil {
            return errors.Join(err, encodeErr)
        }
    }
    return err
}
```

network ID를 알고 있으면 `Networks: []resource.Ref{resource.ID("external-id")}`를 사용합니다. 이 Available 경로는 명시 ID도 공유 external-floating 역할에 포함된 ID인지 확인합니다. 기존 Ensure의 명시 ID 전달 경로와 같은 lookup 정책이라고 가정하지 않습니다. server ID는 Compute 이름 조회가 필요 없고, `resource.Name("web-01")`는 allocation 단계에서 Connection의 Compute 조회를 사용할 수 있습니다. free 재사용에 서버 이름 조회를 선행하지 않습니다.

예제의 Connect·Network endpoint 준비는 Available 호출 전에 실행되므로 부모 context가 제한합니다. Available 내부에는 명시한 1분 SDK 예산이 적용됩니다. `Metadata.Body/Header/StatusCode`는 JSON 출력용 필드가 아니므로 실제 HTTP/원래 필드를 조사하려면 아래 result 값을 직접 읽습니다.

## free 후보와 optional server

Network 후보 순서가 역할 목록 순서보다 우선입니다. 각 `Name`/`ID`는 해당 의미로만 exact 비교하고 첫 matching external-floating network를 선택합니다. 후보가 모두 없으면 `ErrNotFound`입니다. 후보 중 어느 network에서 free IP를 찾을지 먼저 정하므로 첫 matching network에 free 후보가 없다고 두 번째 network로 옮겨 새 후보를 찾는 방식이 아닙니다.

목록 조회는 unfiltered Neutron `GET /floatingips`이고 **모든 페이지를 읽은 뒤** 첫 후보를 반환하거나 새로 할당합니다. free 필터는 선택한 floating network, project, port의 null 여부입니다. `port_id` 누락은 native Resource의 default처럼 null로 취급하며 빈 문자열은 null과 다릅니다. project는 present `project_id`를 우선하고 없으면 `tenant_id`를 사용합니다. canonical null이 present이면 다른 tenant alias로 바꾸지 않으며 null과 빈 문자열도 구분합니다.

기본 project는 ProviderClient에 이미 기록된 Keystone v3 project 또는 v2 tenant를 읽습니다. 별도 인증 HTTP나 token 문자열·endpoint·cloud 입력에서 scope를 추측하지 않습니다. recorded scope가 없으면 null project 필터이며 빈 문자열 project와 같지 않습니다. `WithAvailableProject`의 값은 재사용 필터에만 사용하고, 새 POST에는 `project_id` override를 넣지 않아 Neutron의 인증 scope로 할당합니다.

free 재사용에서는 status·주소 family·fixed 주소를 추가 필터하지 않습니다. 따라서 `ERROR` 상태나 IPv6 주소를 가진 free 모델이 반환될 수도 있습니다. 원래 모델과 `Metadata.Body`를 함께 확인하고, 재사용을 IP 연결이나 ACTIVE 성공으로 해석하지 않습니다. 앞 페이지에 후보가 있어도 뒤 페이지 HTTP/decode 오류가 있으면 반환·allocation을 진행하지 않습니다. 이 경로의 floatingips/ports 성공 목록 코드는200이며204를 빈 목록으로 바꾸지 않습니다.

새 allocation의 optional server는 `device_id`로 모든 port를 조회합니다. port가 없으면 unattached로 생성합니다. fixed override가 없고 port가 여러 개이면 명시/공유 NAT destination으로 좁히고, `created_at` 내림차순의 첫 유효 IPv4를 선택합니다. 동일 timestamp에서는 응답 순서를 유지합니다. NAT가 없거나 해당 network의 port가 없으면 오류입니다. fixed override가 있으면 exact IPv4를 찾고, 없으면 unattached로 생성합니다. fixed/NAT 옵션은 free 재사용이나 server를 생략한 allocation을 연결 작업으로 바꾸지 않습니다.

Available의 최근 port·첫 IPv4 선택은 이 standalone 연산의 source 계약입니다. [Ensure/PrepareEnsure](floating-ip-plan.md)의 strict unique destination 정책과 혼동하지 않습니다. 여기서 반환한 free IP에 client-side 예약이나 lease를 추가하지 않으므로 반환 이후 다른 caller가 연결할 수 있습니다.

## 반환값·null·HTTP 증거

| 값 | 의미 |
|---|---|
| `FloatingIPAvailability.FloatingIP` | 실제 Neutron native alias 모델. 모델을 확인하지 못한 accepted allocation에서는 nil일 수 있음 |
| `Metadata.Body` | 해당 객체의 원래 JSON field. 누락/null·extension·큰 정수를 RawMessage로 보존 |
| `Metadata.Header/StatusCode` | 객체를 얻은 실제 list 또는 allocation 응답 증거 |
| `Reused` | 선택된 free 후보를 반환함. association 접수나 예약의 뜻이 아님 |
| `Allocated` | 새 Neutron POST가201/202로 접수됨. 이후 응답 처리 오류에서도 알려진 사실로 유지 |
| `AllocationResponse.Envelope` | allocation의 실제 전체 JSON envelope 또는 읽은 partial bytes |
| `AllocationResponse.Header/StatusCode` | allocation의 실제 header/status 복사본 |

native alias의 string 필드는 JSON null과 누락을 모두 빈 값으로 보일 수 있습니다. 그 구분이 필요하면 `Metadata.Body`를 읽습니다. Metadata는 Python의 full normalized mutable Resource를 구현한 모델이라는 뜻이 아닙니다.

allocation 응답은 요청한 외부 network, 선택한 port/fixed IPv4, 유효 ID·floating IPv4와 project 일관성을 확인합니다. 인증 scope가 할당한 project와 재사용 filter override를 같아야 한다고 강제하지 않습니다. `DOWN`은 ACTIVE 대기 없이 반환할 수 있습니다. 검증·read/Close·취소·source 변화가 실패해도 decoded 모델이나 실제 allocation receipt가 알려졌으면 result와 error를 함께 보존합니다. `Allocated=true`만으로 모델이 유효하거나 서버에 연결되었다고 판단하지 않습니다.

이 Network 메서드는 Neutron 실패 뒤 Nova로 바꾸거나 자동 DELETE·다른 network 재할당을 하지 않습니다. source/client가 바뀌거나 취소되면 후속 요청을 멈춥니다. IP 후보·서버 port의 성공/실패 결과를 cache하지 않고, 공유 network 역할의 성공 snapshot만 재사용합니다.

## Python 고정 소스와 비교

```python
import openstack

conn = openstack.connect(cloud="dev")  # floating_ip_source: neutron
ip = conn.available_floating_ip(network="public")
print(ip["id"], ip["floating_ip_address"])

server = conn.compute.find_server("web-01", ignore_missing=False)
ip = conn.available_floating_ip(network="public", server=server)
# free 재사용이면 server에 연결하지 않음; 새 Neutron allocation에서만 사용.
```

비교 pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`다. [available public/default/network/free 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L773), [Neutron create·association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L850-L949), [서버 port 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1710)을 대조한다.

Python public Available에는 wait/reuse/timeout 옵션이 없다. Go의 fixed/NAT/project/time 옵션은 concrete 추가 기능이며, default SDK deadline을 추가하지 않는다. Go의 fixed 주소 옵션은 IPv4를 검증하며 빈 값은 override를 해제한다. Python private helper는 literal fixed 주소를 비교한다. Python의 network 문자열은 name OR ID로 비교하고 private helper는 후보 목록도 받는다. Go는 typed Ref로 의미를 고정하고 caller 순서의 후보 목록을 받는다.

Python은 Neutron helper 전 구간의 NotFound를 잡아 Nova pool로 fall through한다. 명시 network 무일치나 default network/router 부재도 이 catch에 들어갈 수 있다. 내부 unfiltered list도 별도의 Neutron404→Nova fallback을 한다. 이 직접 Network API는 그 fallback을 소비하지 않으며 Neutron 오류를 보존한다. [Connection·Compute Available](../compute/floating-ip-available.md)은 pure Neutron NotFound에서 Nova로 전환하며, 이 직접 Network 메서드는 전환하지 않는다. Source 내부 list의 nested fallback과 configured-source normalization 전체는 같은 의미로 구현한 범위가 아니다.

Python Neutron 성공은 network FloatingIP Resource를 그대로 반환하고 Nova path만 [normalizer](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1732-L1830)를 거친다. normalizer의 location/project·strict aliases·properties·합성 Nova ACTIVE, mutable Resource의 inherited session/fetch/commit·field/query semantics는 native alias와 raw Metadata만으로 완료되지 않는다. Python의 configured GET response cache, broad has_service/config/version 정책과 transport/error·cleanup 전체도 별도 비교 범위다.

새 allocation의 실제 HTTP/부분 모델을 error 옆에 보존하는 Go 계약은 Python의 단순 예외 반환과 다르다. 부분 lower 구현만으로 public available/create/list 또는 전체 floating Resource 계약의 지원을 승격하지 않는다. Python 비교는 고정 source 정적 확인이며 위 Python 예제나 인증 cloud 실행의 확인을 뜻하지 않는다. 예제 컴파일·로컬 HTTP fixtures·최종 revision의 gate는 실제 확인된 근거만 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록한다.
