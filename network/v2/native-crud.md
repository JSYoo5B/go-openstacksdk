# Neutron native 생성·조회·목록·삭제

`service.Networks`, `service.Subnets`, `service.Ports`의 generated `Create/Get/List/Delete`는 Gophercloud `v2.15.0`의 [networks](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/networks/requests.go), [subnets](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/subnets/requests.go), [ports](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/ports/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "networks"|"subnets"|"ports"}` 문맥만 더하며, 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID는 escape 없이 경로에 이어 붙입니다. 수정 호출의 revision 조건은 [revision 조건 수정](revision-updates.md)에 따로 설명합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST networks`·`subnets`·`ports`, `{"network"\|"subnet"\|"port": {...}}` | 201, 202 |
| `Get(ctx, id)` | `GET {collection}/{id}` | 200 |
| `List(ctx, options...)` | `GET {collection}` | native pager 200, 204, 300 |
| `Delete(ctx, id)` | `DELETE {collection}/{id}` | 202, 204 |

`With...Field` 확장 필드는 단일 resource envelope 안에 들어가며 옵션 struct의 기존 key와 겹치거나 nil 옵션이면 HTTP 전에 거부됩니다. 생성 응답은 서버가 돌려준 resource이고 SDK가 요청 값을 합성하지 않습니다. 생성·수정 시각은 `2006-01-02T15:04:05`와 RFC3339 두 형식을 모두 받습니다.

## 생성 본문

`networks.CreateOpts`에는 필수 필드가 없고 빈 값은 생략합니다. `subnets.CreateOpts`와 `ports.CreateOpts`는 `NetworkID`가 비어 있으면 HTTP 전에 필수 입력 오류입니다.

subnet의 `GatewayIP`는 nil이면 생략하고, 빈 문자열을 가리키면 `"gateway_ip": null`로 보내 gateway를 끕니다. Python의 `gateway_ip=None`과 같은 의미입니다.

port의 `ValueSpecs` map은 별도 key로 보내지 않고 `port` 객체에 펼쳐 넣습니다. `shared`와 `tenant_id`는 금지 key이고, 이미 있는 필드(예: `name`)를 덮어쓰려 하면 HTTP 전에 오류입니다. `FixedIPs`는 `[]ports.IP`처럼 JSON으로 직렬화되는 값을 그대로 보냅니다.

## 목록

각 `ListOpts`의 필드와 `WithListQuery` 확장 query를 보냅니다. ports의 `FixedIPs []ports.FixedIPOpts`는 조건마다 `fixed_ips=ip_address=...`, `fixed_ips=subnet_id=...`처럼 반복 query가 되고 `SecurityGroups`도 반복 key로 보냅니다. `*_links`의 `rel=next` href를 받은 그대로 따라가며, 호출자가 순회를 멈추면 다음 페이지를 요청하지 않습니다. 빈 목록은 아무 값도 내보내지 않고, 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 `io.EOF` 오류 하나로 끝납니다. 목록 오류는 operation 문맥 없이 native 오류 그대로 전달됩니다.
