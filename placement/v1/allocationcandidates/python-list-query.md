# Allocation candidate의 반복 query

openstacksdk의 [`allocation_candidates(**query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/_proxy.py)는 query를 [`AllocationCandidate.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/allocation_candidate.py)를 거쳐 requests의 `params`로 그대로 넘깁니다. 그래서 `member_of1=["agg-4", "agg-5"]`처럼 목록 값은 같은 키의 반복이 됩니다. 접미사가 붙은 `member_of`의 반복은 placement 1.24부터 유효합니다. Gophercloud v2.15.0 `ResourceGroup.MemberOf`는 문자열 하나이고 `WithListQuery`는 키마다 값 하나를 남기기 때문에, `WithListQueryValues`가 반복 키를 보냅니다.

## 사용법

```go
for page, err := range api.List(ctx,
	allocationcandidates.WithListOptions(allocationcandidates.ListOpts{
		ResourceGroups: map[string]allocationcandidates.ResourceGroup{"1": {Resources: "VCPU:1"}},
	}),
	allocationcandidates.WithListQueryValues("member_of1", "agg-4", "agg-5"),
) {
	...
}
```

- `WithListQueryValues(key, values...)`는 키를 값 순서대로 반복해 `GET allocation_candidates`에 붙이고, 같은 키의 `ListOpts` 직렬화를 대신합니다. 같은 키에 나중에 준 `WithListQuery`나 `WithListQueryValues`가 이긴다는 점은 다른 query option과 같습니다. 값 없이 부르면 이 확장만 빠지고 `ListOpts` 값은 남습니다.
- 빈 키는 HTTP 전에 `resource.ErrInvalidOption`을 담은 `resource.OperationError`(Operation `List`)로 실패합니다. 400 응답은 Gophercloud 상태 오류로 돌려줍니다.

## Python과 다른 점

- Python은 min(`_max_microversion` 1.34, 서버 최대값)을 호출마다 협상하고, session 기본값이 있으면 그 값을 씁니다. Go는 service client의 microversion을 그대로 보내므로 1.34를 보내려면 client를 그렇게 만듭니다.
- Python은 allocation request마다 `AllocationCandidate`를 하나씩 내주며 `provider_summaries`를 그 candidate의 provider로 줄입니다. Go는 응답 전체를 한 번 내주므로 `AllocationRequests[i].Allocations`의 키로 `ProviderSummaries`를 고릅니다.
- Python은 400 미만의 모든 상태를 받고 1.10과 1.11의 목록 형태도 읽습니다. Go는 Gophercloud paging이 받는 200, 204, 300만 받고 1.12 이후 형태만 읽습니다.
- Python은 `paginated`, `base_path`, `jmespath_filters`, `allow_unknown_params`, `microversion`을 query가 아닌 호출 제어로 씁니다. Go에는 이에 맞는 option이 없고 microversion은 client에서 정합니다.
