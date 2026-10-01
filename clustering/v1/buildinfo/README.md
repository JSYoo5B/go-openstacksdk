# Senlin build information

Python openstacksdk의 `conn.clustering.get_build_info()`에 대응합니다.

```go
info, err := buildinfo.New(client).Get(ctx)
if err != nil {
    return err
}
if info.API != nil && info.API.Revision != nil {
    fmt.Println(*info.API.Revision)
}
```

`client`는 Senlin v1 endpoint를 사용하는 `*gophercloud.ServiceClient`입니다.
`GET build-info`는 ID가 없는 singleton이며 요청 body를 보내지 않습니다.
`API`와 `Engine`은 revision과 추가 JSON 필드를 보존합니다. 각 결과의
`Body`, `Header`, `StatusCode`에서 확장 필드와 HTTP 응답 근거를 확인할 수 있습니다.
숫자 확장 필드를 `float64`로 변환하지 않습니다.

이 패키지는 build information 조회만 구현합니다. Python의 mutable Resource,
dirty-field 관리나 모든 Senlin API 지원을 의미하지 않습니다. 잘못된 envelope는
원본 응답을 담은 `*resource.ResponseError`로 반환합니다.

계약 근거: pinned openstacksdk revision
`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `build_info.py`, `_proxy.py`와
[공식 Clustering API](https://docs.openstack.org/api-ref/clustering/#build-information-build-info).
공식 문서는 GET 성공 코드 200을 선언합니다.
