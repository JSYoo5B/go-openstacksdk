# Compute API version native 호출

`apiversions.New(client)`의 generated `List(ctx)`와 `Get(ctx, version)`은 Gophercloud `v2.15.0`의 [compute apiversions 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/apiversions/requests.go)을 바꾸지 않고 호출합니다. SDK는 `Get` 오류에 `resource.OperationError{Resource: "apiversions"}` 문맥만 더합니다.

두 호출은 ResourceBase가 아니라 client의 `Endpoint`에서 처음 나오는 `v<숫자>` segment 앞까지만 잘라 version root를 만듭니다. 그래서 `https://nova/compute/v2.1/project`의 목록 요청은 `GET https://nova/compute/`이고, `Get(ctx, "v2.1")`과 `Get(ctx, "v2.1/")`는 모두 `GET https://nova/compute/v2.1/`입니다. endpoint의 query와 fragment는 버립니다.

| 메서드 | 성공 status | 반환 |
|---|---|---|
| `List(ctx)` | native pager 200, 204, 300 | `versions` 행 |
| `Get(ctx, version)` | 200 | 응답의 `version` |

`Get` 응답에 `version` 객체가 없으면 `apiversions.ErrVersionNotFound`를 operation 문맥과 함께 돌려줍니다. `List`는 단일 페이지이고 목록 오류는 operation 문맥 없이 전달됩니다. `Updated`는 RFC3339 시각으로 해석합니다. 연결의 microversion 협상은 이 호출과 별개이며 [microversion 문서](../../docs/microversions.md)를 참고합니다.
