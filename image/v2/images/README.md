# Image v2 이미지 API·대기

`WaitForState(ctx, ref, target, options...)`는 ERROR/2초와 SDK 시간 제한 없는
상태 대기를 제공합니다. `WaitForDelete`는 삭제 요청 없이 실제 미존재를
기본 120초 기다립니다. parent context와 caller 옵션이 적용되며 기존
`WaitFor`/`WaitForDeletion`의 공통 5분 정책은 유지됩니다.

[서비스 대기 가이드](../../../docs/service-waits.md)는 전체 예제, Python
Resource/cache 및 반환값 차이를 설명합니다. Image v1 binding과 Image v2
Task396 재생성은 이 상태 대기 API에 포함되지 않습니다.

`Images.Update(ctx, imageID, images.UpdateOpts{...}, options...)`는 native concrete Patch를 순서대로 전송하고 strict 200 응답을 Extract합니다. [native PATCH의 Python/Go 비교와 독립 main](update.md)은 아홉 Patch 타입, property add/replace/remove, nil·빈 값·옵션, partial Image 오류와 별도 owned dirty-state 수정 범위를 설명합니다.

`Images.Delete(ctx, imageID)`는 native raw ID와 기본202·204 정책을 유지하며 error만 반환합니다. [native 삭제 가이드](delete.md)는404·응답 Read/Close·재인증/재시도와 [owned 레코드 삭제](../../image-record-delete.md)의 Record/ACK 차이를 설명합니다.
