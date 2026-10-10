# Neutron native 관리자 호출: agent·segment·IP 사용량

`service.Agents`(`extensions/agents`), `service.Segments`(`extensions/segments`), `service.NetworkIPAvailabilities`(`extensions/networkipavailabilities`)와 `service.Routers.ListL3Agents`는 Gophercloud `v2.15.0`의 [agents](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/agents/requests.go), [segments](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/segments/requests.go), [networkipavailabilities](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/networkipavailabilities/requests.go), [routers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/layer3/routers/requests.go) 요청을 바꾸지 않고 호출합니다. Neutron 기본 policy에서 이 문서의 호출은 모두 관리자 전용이라 일반 프로젝트 토큰으로는 403을 받습니다. SDK는 단건 호출 오류에 `resource.OperationError{Resource: "agents"|"segments"|"networkipavailabilities"}` 문맥만 더하고, 받지 않는 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. 목록 오류는 operation 문맥 없이 native 오류 그대로 전달되며, 목록 status는 native pager의 200, 204, 300입니다. ID는 escape 없이 경로에 이어 붙입니다.

## agent

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Get(ctx, id)` | `GET agents/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT agents/{id}`, `{"agent": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE agents/{id}` | 202, 204 |
| `List(ctx, options...)` | `GET agents` | native pager |
| `ListDHCPNetworks(ctx, id)` | `GET agents/{id}/dhcp-networks` | 200 |
| `ScheduleDHCPNetwork(ctx, id, opts, options...)` | `POST agents/{id}/dhcp-networks`, `{"network_id": ...}` | 201 |
| `RemoveDHCPNetwork(ctx, id, networkID)` | `DELETE agents/{id}/dhcp-networks/{networkID}` | 202, 204 |
| `ListL3Routers(ctx, id)` | `GET agents/{id}/l3-routers` | 200 |
| `ScheduleL3Router(ctx, id, opts, options...)` | `POST agents/{id}/l3-routers`, `{"router_id": ...}` | 201 |
| `RemoveL3Router(ctx, id, routerID)` | `DELETE agents/{id}/l3-routers/{routerID}` | 202, 204 |
| `ListBGPSpeakers(ctx, agentID)` | `GET agents/{agentID}/bgp-drinstances` | native pager |
| `ScheduleBGPSpeaker(ctx, agentID, opts, options...)` | `POST agents/{agentID}/bgp-drinstances`, `{"bgp_speaker_id": ...}` | 201 |
| `RemoveBGPSpeaker(ctx, agentID, speakerID)` | `DELETE agents/{agentID}/bgp-drinstances/{speakerID}` | 202, 204 |
| `ListDRAgentHostingBGPSpeakers(ctx, bgpSpeakerID)` | `GET bgp-speakers/{bgpSpeakerID}/bgp-dragents` | native pager |

`UpdateOpts`의 `Description`·`AdminStateUp`은 pointer라 빈 설명과 false를 명시해 보낼 수 있고, 둘 다 nil이면 `{"agent": {}}`를 보냅니다. 세 schedule 호출은 `NetworkID`·`RouterID`·`SpeakerID`가 필수라 비어 있으면 HTTP 전에 오류입니다. schedule 본문에는 envelope가 없어서 `With...Field` 확장 필드는 ID 옆 최상위에 붙습니다. 확장 key가 옵션 struct의 기존 key와 겹치거나 nil 옵션이 섞이면 HTTP 전에 거부합니다. schedule은 Gophercloud가 성공 status를 201 하나로 좁혀서 202나 204도 오류이며, 응답 본문은 읽지 않습니다.

agent 응답의 `created_at`·`started_at`·`heartbeat_timestamp`는 Neutron 형식인 `2006-01-02 15:04:05`만 받습니다. 빈 문자열은 zero 시각이 되고, RFC3339 형식(`2026-10-10T01:02:03Z`)은 decode 오류입니다. `configurations`는 `map[string]any`로 그대로 둡니다. `Get`·`Update`는 `agent` key를 일반 struct로 읽기 때문에 본문이 `{}`이거나 값이 null이거나 다른 key만 있으면 오류 없이 nil agent를 돌려줍니다. 호출자는 nil 여부를 따로 확인해야 합니다. `ListDHCPNetworks`와 `ListL3Routers`도 같은 방식으로 `networks`·`routers` key를 읽어서, key가 없으면 오류 없이 nil slice입니다. 두 호출은 pager가 아니라 단건 GET이라 링크를 따라가지 않습니다.

`List`와 `ListDRAgentHostingBGPSpeakers`는 Gophercloud `AgentPage`가 `agents_links` 배열의 `rel=next` href를 따라가고, `{"links": {"next": "..."}}` 문자열은 무시합니다. `List`는 `ListOpts` 필드(`Alive`는 pointer라 false도 보냄)와 `WithListQuery` 확장 query를 보냅니다. `ListBGPSpeakers`는 한 페이지만 읽고 어떤 링크도 따라가지 않으며, 응답의 `bgp_speakers`를 `speakers.BGPSpeaker`로 decode합니다. 세 목록 모두 빈 목록은 아무 값도 내보내지 않고, 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 `io.EOF` 오류 하나로 끝납니다.

## segment

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST segments`, `{"segment": {...}}` | 201, 202 |
| `Get(ctx, id)` | `GET segments/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT segments/{id}`, `{"segment": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE segments/{id}` | 202, 204 |
| `List(ctx, options...)` | `GET segments` | native pager |

`CreateOpts`의 `NetworkID`와 `NetworkType`은 필수라 비어 있으면 HTTP 전에 오류입니다. 나머지 필드는 비어 있거나 0이면 생략하므로 `SegmentationID` 0은 생성 때 보낼 수 없습니다. `UpdateOpts`의 `Name`·`Description`·`SegmentationID`는 pointer라 빈 문자열과 0을 명시해 보낼 수 있습니다. 확장 필드는 `segment` envelope 안에 들어가고, 기존 key와 겹치거나 nil 옵션이면 HTTP 전에 거부합니다.

응답 decode는 `segment` key를 직접 찾습니다. 본문이 `{}`이거나 값이 null이면 오류 없이 빈 segment를 돌려주고, 다른 key만 있거나 envelope가 객체가 아니면 오류입니다. `created_at`·`updated_at`은 표준 RFC3339만 받아서 시간대 없는 형식은 decode 오류이며, null `segmentation_id`는 0입니다.

segment 목록은 Gophercloud의 `SegmentPage`가 `NextPageURL`을 정의하지 않아 `segments_links` 배열을 읽지 않고 `{"links": {"next": "..."}}` 문자열만 따라갑니다. 그래서 Neutron 응답에서는 첫 페이지만 반환합니다. `ListOpts`에는 `Limit`·`Marker`가 없으니 페이지를 나눠 읽으려면 `WithListQuery("limit", ...)`와 `marker`를 직접 다뤄야 합니다. 정수 필터인 `SegmentationID`·`RevisionNumber`는 0이면 생략합니다.

## IP 사용량

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Get(ctx, networkID)` | `GET network-ip-availabilities/{networkID}` | 200 |
| `List(ctx, options...)` | `GET network-ip-availabilities` | native pager |

`List`는 `ListOpts`의 `network_id`·`network_name`·`ip_version`·`project_id`·`tenant_id` 필터와 확장 query를 보내고, 한 페이지만 읽어 어떤 링크도 따라가지 않습니다. 빈 목록과 본문 없는 204는 agent 목록과 같은 규칙입니다.

`total_ips`와 `used_ips`는 IPv6 범위처럼 `uint64`를 넘는 값을 정확히 보존하려고 10진 정수 문자열로 돌려줍니다. 응답이 `1e3`처럼 지수 표기여도 정수 문자열(`"1000"`)로 바꿉니다. 이 두 값은 network와 모든 `subnet_ip_availability` 항목에서 필수라, 하나라도 없거나 null이면 `Get`은 operation 문맥이 붙은 decode 오류이고 `List`는 그 페이지에서 native decode 오류를 냅니다. `Get`은 agent와 같이 envelope를 일반 struct로 읽어서, 본문이 `{}`이거나 `network_ip_availability`가 null이면 오류 없이 nil을 돌려줍니다.

## router의 L3 agent 목록

`service.Routers.ListL3Agents(ctx, routerID)`는 `GET routers/{routerID}/l3-agents`를 보내고 한 페이지만 읽습니다. `agents_links`와 `links.next` 모두 따라가지 않으며 옵션도 없습니다. 결과는 `routers.L3Agent`이고 시각 필드는 agent와 같은 `2006-01-02 15:04:05` 형식만 받습니다. RFC3339 시각이 있으면 그 페이지는 native decode 오류입니다. 응답에 `agents` key가 없으면 빈 목록으로 보고, 404는 operation 문맥 없는 native status 오류입니다.

## Python SDK와의 차이

이 문서는 Gophercloud native 호출의 계약만 다룹니다. Python openstacksdk의 resource model, finder, agent 아래 router·network 추가·제거 helper의 인자 형태와 비교는 따로 추적합니다. agent와 segment의 `Resources` collection(`Find`, `All`, `WaitFor` 등)은 위 `Get`·`List`·`Delete`를 그대로 사용합니다.
