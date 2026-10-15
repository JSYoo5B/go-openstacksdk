# Glance task 생성의 null input

openstacksdk의 [`create_task(**attrs)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py)는 받은 [`Task`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/task.py) 속성을 `POST tasks`에 보냅니다. 그래서 `create_task(type="import", input=None)`은 `{"type": "import", "input": null}`을 보냅니다. `Service.CreateTask`는 nil Input을 `{}`로 바꾸어 보내므로 이 요청을 표현할 수 없었고, `WithCreateTaskNullInput`이 그 빈자리를 채웁니다.

## 사용법

```go
task, err := svc.CreateTask(ctx, "import", image.WithCreateTaskNullInput())
```

- `WithCreateTaskNullInput()`은 최상위 `"input": null`을 보냅니다. 앞서 준 input 객체를 지우고, 뒤에 오는 `WithCreateTaskInput`이나 `WithCreateTaskOpts`가 이 설정을 대신합니다.
- `CreateTaskOpts{NullInput: true}`도 같은 요청을 보냅니다. `NullInput`과 nil이 아닌 `Input`을 함께 주면 HTTP 전에 `resource.ErrInvalidOption`을 담은 `resource.OperationError`(Operation `CreateTask`)로 실패합니다.
- option이 없으면 이전과 같이 `{"type": ..., "input": {}}`를 보냅니다. Glance controller가 `task['input'].get(...)`을 쓰기 때문에 null input은 서버가 거부할 수 있고, 400 같은 오류는 Gophercloud 상태 오류로 돌려줍니다.
- 201 응답의 `input`이 null이면 `TaskInfo.Input`은 nil이고 원본 null은 `Metadata.Body`에 남습니다.

그 밖의 요청 규칙, 응답 모델, 상태 코드는 [tasks.md](tasks.md)에 설명한 `CreateTask` 동작과 같습니다.

## Python과 다른 점

- Python은 받은 Task 속성(`status`, `owner`, `message` 등)을 모두 보내며 type이나 input을 빼고 보낼 수도 있습니다. Go는 Glance controller가 task 생성에 넘기는 type과 input 두 키만 항상 보냅니다.
- Python은 목록이나 문자열 같은 어떤 input 값도 보내지만 Go는 JSON 객체(기본 `{}`) 또는 `WithCreateTaskNullInput`의 null만 보냅니다.
- Python은 빈 type 문자열도 서버에 보내고, Go는 빈 type이나 잘못된 UTF-8 type을 HTTP 전에 거부합니다.
- Python은 400 미만의 모든 상태를 받고 JSON이 아닌 본문을 무시합니다. Go는 201만 받고 객체가 아닌 본문은 `resource.ResponseError`로 돌려줍니다.
- Python은 요청 속성과 응답을 합친 Task Resource를 돌려주고 `self` 키를 버립니다. Go는 응답 필드만 담은 `TaskInfo`를 돌려줍니다.
