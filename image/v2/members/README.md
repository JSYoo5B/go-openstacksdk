# Glance native 이미지 멤버

`service.API.Members`(또는 준비한 client의 `members.New(client)`)는 Gophercloud `v2.15.0`의 native member 함수 다섯 개를 그대로 호출하고 native `*members.Member`를 반환합니다. 부모 Ref 해석·missing 기본값·local cap이 필요하면 [멤버 사용법](../../members.md)의 `AddImageMember` 계열을, Python Resource와 같은 레코드는 [멤버 레코드](../../member-records.md)를 사용합니다.

```go
api := service.API.Members
member, err := api.Create(ctx, imageID, projectID)
if err != nil { return err }
member, err = api.Update(ctx, imageID, projectID, members.UpdateOpts{Status: "accepted"})
if err != nil { return err }
for value, err := range api.List(ctx, imageID) {
    if err != nil { return err }
    fmt.Println(value.MemberID, value.Status)
}
if err := api.Delete(ctx, imageID, projectID); err != nil { return err }
```

위 코드에는 Go `fmt`와 `github.com/JSYoo5B/go-openstacksdk/image/v2/members`를 사용합니다. 공유를 받은 프로젝트가 상태를 바꾸고 이미지 소유 프로젝트가 멤버를 추가·삭제합니다. 실제 권한은 [Glance member 정책](../../../docs/glance-policy-priorities.md)과 배포 설정을 서버가 판단합니다.

## 요청과 성공 status

| 메서드 | 요청 | 기본 성공 status |
|---|---|---|
| `Create(ctx, imageID, memberID)` | `POST images/{imageID}/members`, 본문 `{"member": memberID}` | 200 |
| `Get(ctx, imageID, memberID)` | `GET images/{imageID}/members/{memberID}` | 200 |
| `List(ctx, imageID)` | `GET images/{imageID}/members`, 단일 페이지 | native pager 기본 |
| `Update(ctx, imageID, memberID, opts, options...)` | `PUT .../members/{memberID}`, 본문 `{"status": opts.Status}` | 200 |
| `Delete(ctx, imageID, memberID)` | `DELETE .../members/{memberID}` | 204 |

[고정 요청 원문](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/members/requests.go#L28-L86)은 ID를 ServiceURL segment로 그대로 이어 붙이고 escape하지 않습니다. 그래서 `/`나 `?`가 든 ID는 다른 경로나 query가 됩니다. Update는 Status가 비어 있어도 `"status": ""`를 보냅니다. `WithUpdateField(key, value)`는 본문에 JSON 값을 더하는 Go 확장이며 `status`처럼 이미 있는 key와 겹치면 HTTP 전에 `resource.ErrInvalidOption`입니다.

## 응답과 오류

성공 응답은 `Member{CreatedAt, ImageID, MemberID, Schema, Status, UpdatedAt}`로 decode합니다. List는 `members` 배열의 단일 페이지를 lazy 순회하고, 빈 배열이면 아무것도 반환하지 않습니다. Update의 반환값은 서버 응답이며 요청한 status를 합성하지 않습니다.

다른 status는 native `gophercloud.ErrUnexpectedResponseCode`이고 Expected는 위 표의 한 값입니다. SDK는 `resource.OperationError{Operation: "Create" 등, Resource: "members"}` 문맥만 더하고 owned 응답 증거를 만들지 않습니다. List 순회 오류에는 이 문맥을 더하지 않습니다. native 재시도·재인증 정책은 [native 이미지 삭제 가이드](../images/delete.md#오류와-native-재시도)와 같습니다.
