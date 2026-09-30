# openstacksdk 비교 기준

이 목록은 openstacksdk의 [고정 소스](https://github.com/openstack/openstacksdk/tree/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe)를 AST로 읽어 생성합니다. 서비스 버전별 `Proxy`에 직접 선언된 공개 메서드 1,834개, cloud 모듈의 공개 메서드 397개, `Connection`에 직접 선언된 공개 메서드 5개를 기록합니다. 상속으로 제공되는 메서드나 런타임에 추가되는 동작은 이 집계에 포함되지 않습니다.

각 항목에는 입력 인자와 기본값, 참조한 Python 리소스, 소스 위치를 보관합니다. `candidates`는 서비스·버전·모델 이름이 일치하는 Go 리소스 패키지 후보이며, 기능 동등성을 판정한 결과가 아닙니다. 생성 목록의 모든 `review`는 `pending`이며, 실제 SDK 판정은 재생성으로 덮어쓰지 않는 [별도 대장](../sdk_reviews.json)에 보관합니다. [검증기](../../internal/cmd/paritycheck/README.md)가 고정 소스와 API·테스트·문서 근거를 확인합니다. API 파사드 생성 완료와 openstacksdk 수준의 SDK 완성 여부를 별도로 확인하기 위한 비교 자료입니다.

SDK가 Gophercloud 모델에 metadata 등을 추가한 경우 공통 정책 목록의 `upstream_model`도 후보 검색에 사용합니다. `model`은 실제 Go Collection의 응답 타입을 기록합니다.

재생성하려면 해당 커밋의 openstacksdk 체크아웃 경로를 지정합니다. 도구는 소스 커밋을 확인하며 다른 버전은 거부합니다.

```sh
python3 internal/cmd/parity/inventory.py --source /path/to/openstacksdk
```

서비스별 목록은 [manifest.json](manifest.json), 공통 작업은 [cloud.json](cloud.json), 연결 객체는 [connection.json](connection.json)에 있습니다. Go의 공통 리소스 기능 적용 범위는 [resource_inventory.json](../resource_inventory.json)에서 확인합니다.
