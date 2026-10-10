# Cinder v3 native transfer·availability zone·버전 호출

이 문서는 Gophercloud `v2.15.0`의 [v3 transfers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/transfers/requests.go), [v3 availability zones](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/availabilityzones/requests.go), [API versions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/apiversions/requests.go) 요청을 호출하는 generated 메서드를 설명합니다. SDK는 오류에 `resource.OperationError` 문맥만 더하며 목록 stream 오류에는 문맥을 더하지 않습니다.

## volume transfer

`service.Transfers`(`blockstorage/v3/transfers`)는 경로 `os-volume-transfer`, envelope `transfer`를 씁니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST os-volume-transfer`, `{"transfer": {...}}` | 202 |
| `Accept(ctx, id, opts, options...)` | `POST os-volume-transfer/{id}/accept`, `{"accept": {"auth_key": ...}}` | 202 |
| `Get(ctx, id)` | `GET os-volume-transfer/{id}` | 200 |
| `List(ctx, options...)` | `GET os-volume-transfer/detail` | native pager 200, 204, 300 |
| `Delete(ctx, id)` | `DELETE os-volume-transfer/{id}` | 202, 204 |

`CreateOpts`의 `VolumeID`와 `AcceptOpts`의 `AuthKey`는 필수라 비어 있으면 HTTP 전에 오류입니다. 생성 응답의 `auth_key`는 받는 쪽 프로젝트에 전달해야 하는 비밀 값이며 `Transfer.AuthKey`로 그대로 decode합니다. 수락 응답도 `transfer` key의 volume ID를 돌려줍니다. 목록은 상세 경로를 쓰고 `transfers_links`의 next href를 따라가며, `AllTenants`는 `all_tenants=true`를 보냅니다.

응답은 `transfer` key를 직접 찾아 `{}`·null을 빈 transfer로 돌려주고, 다른 key만 있으면 오류입니다. 생성 시각은 시간대 없는 형식만 받아서 끝에 `Z`가 붙으면 decode 오류입니다.

## availability zone 목록

`service.AvailabilityZones.List(ctx)`는 `GET os-availability-zone`을 한 번만 보내고 `availabilityZoneInfo` 배열을 이름과 `zoneState.available` 값으로 돌려줍니다. 링크는 따라가지 않습니다. Gophercloud의 page는 객체 본문을 거부하는 배열 전용 `IsEmpty`를 물려받아서, generated facade는 native `AllPages`처럼 한 페이지를 바로 추출합니다. 빈 배열이나 key 없는 객체는 아무 값도 내보내지 않고, 배열이 아닌 `availabilityZoneInfo`나 최상위 배열 본문은 추출 오류입니다.

## API 버전 목록

`apiversions.New(client).List(ctx)`는 `ResourceBase`가 아니라 client `Endpoint`에서 버전 segment와 query를 잘라낸 root(`GET {root}/`)를 한 번 조회합니다. Cinder가 root에 돌려주는 300 Multiple Choices도 native pager가 성공으로 받습니다. 각 버전의 `version`·`min_version`은 문자열이고 `updated`는 RFC3339 시각이라 형식이 어긋나면 decode 오류입니다.
