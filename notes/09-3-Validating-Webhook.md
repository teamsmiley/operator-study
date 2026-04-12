# 09-3. Validating Webhook -- CR 생성/수정 시 규칙 검증

## Validating Webhook이란?

CR이 생성/수정될 때 규칙에 맞는지 검증해서 **허용하거나 거부**하는 HTTP 엔드포인트다.
Mutating Webhook 이후에 실행된다.

비유: 경비원. 출입증이 없으면 못 들어간다.

```text
사용자가 CR 생성 요청
  -> Mutating Webhook: 빠진 값 자동 채움 (이미 구현)
  -> Validating Webhook: 규칙에 맞는지 검증
     -> 통과: etcd에 저장
     -> 거부: 에러 반환, 저장 안 됨
```

## 언제 쓰나?

| 용도 | 예시 |
| ---- | ---- |
| 필드 값 검증 | image에 태그가 반드시 포함되어야 한다 |
| 필드 간 비교 | replicas를 한 번에 50% 이상 줄이지 못한다 |
| 삭제 방지 | 특정 라벨이 있는 CR은 삭제할 수 없다 |
| 외부 참조 검증 | 지정된 registry 허용 목록에 있는지 확인 |

## scaffold 생성

기존 Mutating Webhook이 있는 상태에서 Validating을 추가한다:

```bash
kubebuilder create webhook --group apps --version v1 --kind SimpleApp --programmatic-validation
```

기존 `simpleapp_webhook.go`에 Validating 관련 코드가 추가된다.

## 생성되는 함수 3개

| 함수 | 호출 시점 | 역할 |
| ---- | --------- | ---- |
| `ValidateCreate()` | CR 생성 시 | 생성 요청 검증 |
| `ValidateUpdate()` | CR 수정 시 | 수정 요청 검증 (이전 값과 새 값 비교 가능) |
| `ValidateDelete()` | CR 삭제 시 | 삭제 요청 검증 |

ValidateUpdate는 `oldObj`와 `newObj` 두 개를 받아서 이전 값과 새 값을 비교할 수 있다.
이것이 CRD validation marker로는 불가능하고 Webhook만 할 수 있는 핵심 기능이다.

## 구현 예정 로직

ValidateCreate, ValidateUpdate:
- image에 태그가 포함되어야 한다 (`nginx` NG, `nginx:1.25` OK)

ValidateDelete:
- 별도 검증 없음 (삭제 허용)

## 다음 단계

scaffold 생성 후 ValidateCreate, ValidateUpdate 함수에 로직을 채운다.
