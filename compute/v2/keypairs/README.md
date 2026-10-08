# 키페어 조회·생성·삭제

`KeyPairs.Get(ctx, name)`은 키페어 이름으로 직접 조회합니다. 다른 owner를 명시하려면 `keypairs.WithGetOptions(keypairs.GetOpts{UserID: ownerID})`를 사용하고 cloud에 맞는 Compute microversion을 선택합니다.

[Compute 조회 가이드](../../user-read-apis.md)에 Python `get_keypair` 대응, 자기 user 기본값·optional owner, native 반환 필드와 빈 값·오류 정책, 독립 실행 main을 설명합니다. 키페어 생성·삭제 또는 Python mutable Resource 전체의 지원 판정은 이 조회와 분리합니다.

`DeleteKeypair(ctx, name, options...)`는 직접 이름 삭제의 optional owner와 기본 미존재 무시를 함께 제공합니다. [Compute 작업 가이드](../../user-actions.md)에 strict 선택·접수 후 오류와 native `Delete`·`Remove`의 차이를 설명합니다.

`CreateKeypair(ctx, options...)`는 nullable 전체 속성·ID/name 별칭·입력/응답 병합과 `ssh` 반환 기본값을 처리합니다. [생성·legacy console 가이드](../../keypairs-console.md)에 public key import·concrete 옵션·Resource/Wire·native Create 비교와 전체 main을 제공합니다.

`ListRecords(ctx, options...)`와 `FindKeypair(ctx, nameOrID, options...)`는 SDK가 필터·페이지네이션·owner를 유지하는 fallback을 처리합니다. [목록·검색 가이드](../../keypairs-list-find.md)에 concrete `With` 옵션, 기본값, nullable Resource/실제 Wire·오류와 native SinglePage 목록 비교, 전체 실행 main을 제공합니다.
