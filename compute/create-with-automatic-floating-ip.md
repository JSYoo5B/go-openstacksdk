# 서버 생성과 조건부 floating IPv4 연결

`compute.Service.CreateWithAutomaticFloatingIP`는 서버를 생성하고 raw Nova 응답의 실제 ACTIVE와 주소 inventory를 확인한 뒤, [pool → 순차 명시 IPv4 → automatic 선택](server-ip-dispatch.md)에 따라 Neutron 또는 [legacy Nova 연결](server-nova-floating-ip.md)과 raw Nova 주소 관측을 이어갑니다. 기본 automatic 분기에만 필요성 skip 조건을 적용하며 명시 pool/IP 요청은 별도로 실행합니다. Connection이 이름 조회·서비스 연결·cloud 정책을 제공하므로 애플리케이션에서 builder나 resolver interface를 만들지 않습니다. 이미지 부팅, 기존 볼륨 부팅, 이미지에서 새 볼륨을 만드는 부팅에 같은 흐름을 적용합니다.


| 호출 | 기본 동작 |
|---|---|
| `service.Servers.Create(ctx, request, opts...)` | 기존 생성; 선택적 `WithWait`, 기본 비동기 결과 |
| `service.Servers.CreateWithFloatingIP(...)` | 기존 명시 IP 요청; 서버·IP ACTIVE 필수, 자동 needs와 Nova 주소 관측 없음 |
| `service.CreateWithAutomaticFloatingIP(ctx, request, options)` | 서버 실제 ACTIVE·주소 확인 필수; 선택한 IP 정책의 assignment·Neutron IP ACTIVE·raw Nova 관측 |
| `conn.CreateWithAutomaticFloatingIP(ctx, request, options)` | Connection의 같은 생성 작업 delegate |
| `conn.EnsureServerFloatingIP(...)` | 이미 존재하는 서버의 IP 선택·연결 작업; 새 서버를 생성하지 않음 |

요청은 기존 `compute.CreateServerRequest{Name, Image, Flavor}`입니다. `compute.AutomaticServerCreateOptions`가 서버 옵션, 자동 IP 옵션, 외부 allocation network를 구분합니다.

| 옵션 field | 의미 |
|---|---|
| `Server []compute.CreateServerOption` | NIC·부팅·metadata·key/userdata·서버 대기 설정 등 기존 concrete create 옵션 |
| `AutomaticIP []compute.AutomaticFloatingIPOption` | pool·순차 IPv4 선택, automatic needs enabled/private/source, reuse/owner·공통 destination·IP waiter·전체 timeout·raw Nova poll/progress |
| `FloatingIPNetwork resource.Ref` | 기본 automatic 분기의 외부 allocation network; Neutron zero는 공유 역할/router 선택, 명시 name/ID는 같은 plan의 external 선택 경로; Nova는 literal pool 값 또는 첫 pool; pool/명시 IP가 선택되면 무시 |

서버 NIC network는 `Server`의 `WithNetworks`/`WithNetworkInterfaces`에서 지정합니다. `FloatingIPNetwork`를 서버가 연결할 private NIC로 사용하지 않습니다. 옵션은 생성 요청 전에 한 번 준비하며, 중첩 서버·주소·IP·waiter 옵션의 오류도 Nova POST 이전에 처리합니다.

독립 [AddIPsToServer·AddIPList](server-ip-helpers.md)는 별도 기본60초·비동기 entry입니다. 선택적 wait는 서버/IP ACTIVE 없이 raw 목표 주소를 확인하며, 이 문서의 직접 API나 기존 상위 readiness 조건을 바꾸지 않습니다.

## Python cloud와 비교

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [create_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L915-L1245)를 비교합니다. 자동 IP 연결을 포함하는 작업 의도는 `wait=True`로 요청합니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    network="private",
    auto_ip=True,
    reuse_ips=True,
    wait=True,
    timeout=180,
)
print(server.id, server.interface_ip)
```

Python의 기본값은 auto_ip=true·reuse_ips=true·wait=false·timeout180입니다. wait=false 분기는 POST 뒤 GET·ERROR 검사·주소 expansion을 하고 자동 IP dispatch를 수행하지 않습니다. wait=true가 [wait_for_server/get_active_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1481)를 거쳐 pool → 명시 ips → auto+needs 작업을 이어갑니다. Go compound는 항상 서버를 기다리는 별도 entry이며 일반 `Servers.Create`의 동작을 바꾸지 않습니다.

Go는 Neutron/Nova pool·순차 명시 IPv4·automatic 소비자를 제공합니다. Nova의 별도 공개 CRUD·detach/cleanup·함수별 fallback, standalone cloud IP helper의 전체 Resource 반환·정규화, mutable Resource·fault/cleanup·session 계약은 남으며 이 API만으로 전체 Python create/get_active/wait 연산을 완료 처리하지 않습니다.

## 독립 Go 예제

clouds.yaml의 `dev`, 이미지 `ubuntu`, flavor `small`, private NIC 이름을 실제 환경에 맞게 바꿉니다. 예제는 microversion2.37을 명시하고 자동 외부 network를 사용합니다. `-floating-network`를 지정하면 정확한 외부 network 이름을 사용합니다. reachability probe는 해제했으며 기존 cloud private/source 설정은 유지합니다.

`-boot-volume`는 기존 볼륨 ID이고 Image를 비웁니다. `-boot-size`는 이미지에서 만들 새 볼륨의 GiB이며 두 옵션을 함께 지정하지 않습니다. `-auto-ip=false`도 필수 서버 ACTIVE·주소 inventory 대기를 생략하지 않습니다.


```go
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	name := flag.String("name", "web-01", "new server name")
	image := flag.String("image", "ubuntu", "exact image name")
	flavor := flag.String("flavor", "small", "exact flavor name")
	nic := flag.String("network", "private", "exact NIC network name")
	external := flag.String("floating-network", "", "exact external network name; empty uses roles/router")
	volume := flag.String("boot-volume", "", "existing boot volume ID")
	size := flag.Int("boot-size", 0, "new image-backed boot volume GiB")
	enabled := flag.Bool("auto-ip", true, "enable conditional floating IPv4 assignment")
	reuse := flag.Bool("reuse", true, "reuse an attached/free floating IP")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *name, *image, *flavor, *nic, *external, *volume, *size, *enabled, *reuse); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, name, image, flavor, nic, external, volume string, size int, enabled, reuse bool) error {
	if volume != "" && size != 0 {
		return fmt.Errorf("boot-volume and boot-size are mutually exclusive")
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.37"))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	request := compute.CreateServerRequest{
		Name: name, Image: resource.Name(image), Flavor: resource.Name(flavor),
	}
	serverOptions := []compute.CreateServerOption{
		compute.WithNetworks(resource.Name(nic)),
		compute.WithWait(resource.WithTimeout(2*time.Minute), resource.WithPollInterval(time.Second)),
	}
	if volume != "" {
		request.Image = resource.Ref{}
		serverOptions = append(serverOptions, compute.WithBootVolume(resource.ID(volume)))
	}
	if size != 0 {
		serverOptions = append(serverOptions, compute.WithBootVolumeSize(size))
	}
	options := compute.AutomaticServerCreateOptions{
		Server: serverOptions,
		AutomaticIP: []compute.AutomaticFloatingIPOption{
			compute.WithAutomaticIPEnabled(enabled),
			compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
			compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(reuse)),
			compute.WithAutomaticIPTimeout(3*time.Minute),
			compute.WithAutomaticIPPollInterval(time.Second),
		},
	}
	if external != "" {
		options.FloatingIPNetwork = resource.Name(external)
	}
	result, err := service.CreateWithAutomaticFloatingIP(ctx, request, options)
	if result != nil {
		if result.Creation != nil {
			fmt.Printf("creation-id=%s\n", result.Creation.ID)
		}
		if result.Server != nil {
			fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
		}
		if automatic := result.Automatic; automatic != nil {
			fmt.Printf("observed=%t\n", automatic.Observed)
			if decision := automatic.Decision; decision != nil {
				fmt.Printf("needed=%t reason=%s backend=%s\n", decision.Needed, decision.Reason, decision.Backend)
			}
			if assignment := automatic.Assignment; assignment != nil {
				fmt.Printf("reused=%t allocated=%t\n", assignment.Reused, assignment.Allocated)
				if ip := assignment.FloatingIP; ip != nil {
					fmt.Printf("ip-id=%s floating4=%s status=%s\n", ip.ID, ip.FloatingIP, ip.Status)
				}
			}
		}
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("the shared deadline expired; inspect the returned resources")
		}
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response evidence: status=%d", response.StatusCode)
		}
		return fmt.Errorf("create with automatic floating IPv4: %w", err)
	}
	return nil
}
```

Connection delegate도 같은 request/options를 받습니다. 예제는 Service entry를 보여주며, Connect·최초 Compute 서비스 준비는 메서드 밖의 부모 context가 제한합니다. 생성 메서드 안의 image/flavor/volume/NIC 해석부터 후속 작업까지 명시한 전체3분 예산을 사용합니다.

Connection delegate의 `Compute(ctx)` endpoint 준비는 Service가 automatic timeout을 시작하기 전입니다. 이 구간은 caller context가 제한하고, Service 안의 생성·주소 작업부터 automatic 전체 timeout을 적용합니다.

## 부팅과 NIC

| 부팅 방식 | request/Server 옵션 | Nova body와 보존 |
|---|---|---|
| 이미지 | Image·Flavor 지정 | imageRef/flavorRef, 기존 concrete create body |
| 기존 볼륨 | Image zero, `WithBootVolume(ref)` | BDM source=volume/destination=volume/bootIndex0, 기본 delete_on_termination=false |
| 이미지에서 새 볼륨 | Image, `WithBootVolumeSize(size)` | BDM source=image/destination=volume/size, native imageRef는 빈 문자열로 전송; 별도 Cinder POST 없음 |

기존 볼륨 이름은 Cinder에서 해석하고 ID는 조회를 생략합니다. 새 볼륨 type은 `WithBootVolumeType`와 microversion2.67 조건을 따릅니다. `WithDeleteBootVolumeOnTermination(true)`는 후속 Nova 서버 삭제 시의 정책이며 SDK가 생성 실패에 Cinder cleanup을 수행한다는 뜻이 아닙니다. [부팅·생성 옵션](README.md)과 [NIC/defaultnetwork](server-network-interfaces.md), [기본 network 설정](server-default-network.md)을 참조하세요.

known automatic skip이라도 이름/port NIC 또는 configured defaultnetwork를 해석하는 Network 조회가 필요할 수 있습니다. 이는 서버 생성 의존 조회입니다. skip의 불필요한 IP 역할·candidate·owner·mutation을 준비하지 않는 정책과 구분합니다. 명시 NIC ID/none 등으로 NIC 조회가 필요 없는 경우에는 자동 skip 때문에 Neutron endpoint/owner를 선조회하지 않습니다.

configured default와 automatic classification/source/NAT가 공유하는 성공 role cache는 [역할 정책](../network/network-roles.md)을 따릅니다. 한 automatic planner가 확보한 snapshot을 후속 selection까지 사용하며, 환경 외부 변경을 감지하거나 생성 전의 모든 topology를 예약하는 기능은 아닙니다.

## 필수 raw ACTIVE와 주소 inventory

서버 `WithWait` 옵션을 넣지 않아도 raw Nova ACTIVE 대기는 필수이며 기본 timeout5분·interval2초입니다. `WithWait`를 지정하면 준비한 timeout·interval·progress·failure-state 정책을 한 번 적용합니다. `resource.WithStatusAttribute`를 지정한 옵션은 모든 값에 대해 Nova POST 이전에 `resource.ErrUnsupported`로 거부합니다. 다른 문자열 field의 ACTIVE를 실제 server Status로 취급하지 않습니다.

실제 raw 응답의 server ID가 생성 ID와 일치해야 하며, ERROR는 사용자 `WithFailureStates()`로 failure-state 판정을 비워도 실패입니다. configured 다른 failure states는 준비한 정책을 따릅니다. nonterminal 상태와 아래 nil 주소는 같은 서버 wait 시간 안에서 poll/progress 대상입니다.

- ACTIVE에서 `addresses:null` 또는 미제공은 주소 metadata가 아직 없으므로 대기합니다.
- ACTIVE에서 명시 빈 map, 모든 network의 null/빈 row 목록은 `*compute.ServerAddressesUnavailableError`이며 확인한 서버를 반환합니다. `errors.Is(err, compute.ErrServerAddressesUnavailable)`로 분류합니다.
- 실제 주소 inventory를 확인한 뒤에만 선택한 pool·순차 명시 IPv4·automatic assignment 단계로 넘어갑니다.

이 규칙은 `WithAutomaticIPEnabled(false)`, private/source disabled인 경우에도 적용됩니다. 별도 existing-server 자동 helper가 empty evidence를 no-fixed skip으로 반환하는 계약과 생성 compound의 준비 조건은 다릅니다. malformed raw 주소·accepted read/Close/decode·identity/source·취소 오류를 clean empty로 바꾸지 않습니다.

Python get_active_server는 ACTIVE인데 주소 map이 falsy이면 서버 DELETE를 시도합니다. Go는 서버를 자동 삭제하지 않고 typed 오류/알려진 부분 결과를 반환합니다. Python의 nonempty map `{net: []}`는 truthy여서 통과할 수 있지만 Go는 실제 주소 row가 하나 이상 있어야 준비되었다고 판단하므로 더 엄격합니다.


## 조건부 IP와 실제 Nova 관측

서버 준비가 끝나면 [IP dispatch](server-ip-dispatch.md)를 적용합니다. automatic 분기에는 disabled/private·이미 public/floating 같은 skip 조건을 적용합니다. Neutron pool은 Ensure 경로에서 owner를 바인딩하고 기존 IP 재사용/새 allocation을 실행합니다. Nova pool은 literal 이름·current project 범위의 free IP 재사용 또는 POST200 allocation을 사용합니다. Neutron 명시 IPv4 목록은 각 기존 IP의 network와 목적지를 고정하고, Nova는 ID·주소·pool·instance association을 재검증하여 owner 조회나 새 allocation 없이 순서대로 연결합니다. 각 항목의 Neutron IP ACTIVE와 raw Nova의 정확한 assigned IPv4 floating row를 확인한 뒤 다음 항목을 시작합니다.

`Creation`은 최초 Nova 생성 응답, `Server`는 마지막 확인한 Nova 모델, `Automatic`은 `Mode`·`Decision`·최신 nonnil `Assignment` 또는 `NovaAssignment`·`Attempts`·`Observed`를 갖는 IP 단계 결과입니다. 목록의 늦은 실패에도 앞선 완료와 현재 항목의 알려진 부분 결과를 보존합니다. 최초 생성 응답의 AdminPass는 `Creation`에 유지합니다. 후속 GET이 값을 생략했다고 최신 `Server`에 비밀번호를 합성하거나 Python의 admin_password 복원 정책을 완료로 주장하지 않습니다.

raw 관측은 생성 시 고정한 server ID와 실제 ACTIVE, 모든 network row 중 version4/type=floating/addr=assignedIPv4를 확인합니다. Supplemental metadata·AccessIPv4·fixed/IPv6·다른 floating 주소와 Neutron ACTIVE만으로 Observed=true를 만들지 않습니다. 호환 Nova backend는 allocation/add action을 실행하고 `Automatic.NovaAssignment`에 실제 모델·접수 증거를 보존합니다. selected2.36 이상·Neutron 전용 selector는 미지원이며 실패 뒤 다른 backend로 전환하지 않습니다.

| 중단/완료 시점 | 반환하는 알려진 결과 |
|---|---|
| 준비·의존 조회 또는 POST가 모델 확인 전에 실패 | 결과/Creation이 없을 수 있음; 생성 부재를 단정하지 않음 |
| 생성 모델/ID 확인 뒤 server wait·주소 오류 | Creation과 마지막 확인 Server; Automatic 없음 |
| 실제 server 준비 뒤 automatic decision/owner/backend 오류 | Creation·Server, 가능하면 partial Automatic/Decision |
| association/allocation/IP wait·Nova 관측 오류 | Creation·마지막 Server와 실제 partial Assignment 또는 NovaAssignment/response evidence |
| 순차 IP 목록의 뒤 항목 실패 | Creation·마지막 확인 Server·앞선 완료 Attempts·현재 항목의 partial Assignment 또는 NovaAssignment/Error; 뒤 항목은 시작하지 않음 |
| clean automatic skip | Creation·준비된 Server·결정된 reason; 양쪽 assignment nil/Observed false |
| 선택한 모든 항목의 floating4 순차 raw 관측 | Creation·최신 Server·Automatic와 Observed true |

각 Observed는 해당 항목을 처리한 시점의 관측이며 마지막 Server에서 모든 이전 IP의 동시 잔존을 검증한 뜻은 아닙니다.

서버/IP/부팅 볼륨 자동 DELETE나 association 실패 후 새 allocation fallback을 하지 않습니다. 부분 결과는 조사·후속 명시 작업의 근거이며 전체 작업의 성공 flag가 아닙니다. 생성 요청은 SDK-owned guarded Nova POST200/202이고 raw GET은200/203입니다. 접수 후 read/Close·source·취소 오류가 있어도 보존한 body를 유효한 Server로 디코드할 수 있으면 Creation 또는 마지막 Server와 실제 response error를 함께 유지합니다. 잘못된 ID는 대상 서버로 채택하지 않고 proof와 오류를 보존합니다. malformed body 때문에 모델/ID를 확인하지 못했다면 Creation이 nil일 수 있으며 실제 ResponseError를 이용해 접수 여부를 확인합니다.


## 전체 deadline과 옵션 재사용

`AutomaticIP`의 `WithAutomaticIPTimeout`은 이 compound에서는 생성 의존 해석·POST·server wait·선택 정책 판정·순차 목록의 모든 assignment·raw 관측 전체의 예산입니다. 기본 전체5분이며 단계마다 다시 시작하지 않습니다. existing-server Ensure의 같은 옵션은 그 호출 시작부터 적용됩니다. 부모 context가 더 빠르면 부모 deadline이 적용됩니다.

server `WithWait(resource.WithTimeout(...))`와 Neutron IP waiter timeout은 전체 예산 안에서 더 짧은 제한을 줄 수 있습니다. 각 단계의 default5분이 늦게 시작해도 전체 제한을 늘리지 않습니다. `WithUnlimitedAutomaticIPTimeout`은 SDK 전체 제한만 없애며 부모 context와 개별 waiter의 timeout은 유지합니다.

Server·AutomaticIP 옵션과 concrete body 입력을 준비해 재사용하며 closure를 각 단계에서 다시 적용하지 않습니다. 서버 waiter의 `WithProgressCallback(func(int))`는 서버 준비의 progress이고 automatic `WithAutomaticIPProgress(func(*Server) error)`는 아직 Nova floating 주소가 수렴하지 않은 응답의 복사본을 제공합니다. 두 callback의 시점·인자를 구분합니다.


## 전체 소스 대비 남은 범위와 검증

새 compound는 실제 서버 생성과 Neutron/Nova pool·순차 명시 IPv4·automatic 소비자를 연결합니다. [기존 서버의 GetActiveServer·상위 WaitForServer](server-ready.md)는 별도로 supplied 상태 판정과 raw 현재 상태 대기를 제공합니다. 일반 Servers.Create·기존 명시 CreateWithFloatingIP와 각 source 연산 전체를 완료로 승격하지 않습니다. Nova의 별도 공개 CRUD·detach/cleanup·함수별 fallback, standalone cloud IP helper의 전체 Resource 반환·정규화, boot/data volume의 모든 조합·root_volume alias, security/count/group/userdata 등 추가 create 입력, returned mutable Resource/location/session, fault/extra_data·lookup exception retry·integer budget·cleanup 정책은 남습니다.

Python 예제와 비교는 고정 source 정적 검토 기준이며 인증된 cloud/Python 실행을 뜻하지 않습니다. 새 HTTP fixture의 이미지/볼륨/NIC·shared cache·known skip·later owner·실제 ACTIVE/metadata·부분 결과·whole deadline 근거와 정확한 main의 최종code 컴파일을 [지원 판정대장](../docs/sdk-support-ledger.md)에 실제 결과만 기록합니다. 해당 source 공개 연산의 remaining이 남으면 unresolved입니다.


새 생성 HTTP 테스트, Connection 이름 조회·기본 NIC, resolver 페이지·source 유지 및 공통 대기 정책의 검증 근거는 [지원대장](../docs/sdk-support-ledger.md#서버-생성과-자동-floating-ip-통합)에 기록합니다.

생성 전에 알려진 legacy 버전·selector 미지원은 서버 POST 전에 거부합니다. 생성 이후 Nova API 실패에서는 알려진 Creation·Server·NovaAssignment와 원래 HTTP 오류를 유지합니다. 동기 관측은 실제 서버 ACTIVE를 요구하며 BUILD·ERROR에 목표 주소가 있어도 완료되지 않습니다. Nova assignment는 pre-action raw 모델을 보존하고 합성 IP ACTIVE를 제공하지 않습니다.
