# Remote console 생성

`RemoteConsoles.CreateConsole(ctx, serverID, options...)`는 type에서 protocol을 선택하고 지정한 microversion의 type 지원 조건을 검사합니다. SDK 소유 결과는 요청 속성을 유지한 Resource view와 실제 Wire 응답을 구분합니다.

[Compute 작업 가이드](../../user-actions.md)에 Python `create_server_remote_console` 대응, 생략·null·빈 값과 버전 조건, 독립 실행 main을 설명합니다. 기존 native `Create(ctx, serverID, CreateOpts, options...)`와 RemoteConsole alias는 그대로 사용할 수 있습니다.
