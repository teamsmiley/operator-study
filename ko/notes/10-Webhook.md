# 09. Webhook -- CR 생성/수정 시 검증과 기본값 설정

## 왜 필요한가?

CRD validation marker로 기본적인 검증은 가능하다:

```go
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=10
Replicas *int32 `json:"replicas,omitempty"`
```

하지만 이런 검증은 불가능하다:

- image 이름에 태그가 반드시 포함되어야 한다 (`nginx:1.25` OK, `nginx` NG)
- 특정 registry만 허용한다 (`docker.io/`로 시작해야 함)
- replicas를 줄일 때 한 번에 절반 이상 줄이지 못하게 한다
- 다른 리소스(ConfigMap 등)를 참조해서 검증한다

이런 비즈니스 로직 검증은 Webhook이 담당한다.

## CRD validation vs Webhook

|                  | CRD validation marker                    | Validating Webhook                           |
| ---------------- | ---------------------------------------- | -------------------------------------------- |
| 실행 위치        | API Server 내부 (OpenAPI 스키마)         | 외부 HTTP 서버 (Controller Pod)              |
| 검증 수준        | 단순: 타입, 최소/최대값, 필수 여부, enum | 복잡: 필드 간 비교, 외부 조회, 비즈니스 로직 |
| 코드 필요?       | 불필요 (marker 주석만)                   | 필요 (Go 함수 작성)                          |
| 다른 리소스 참조 | 불가                                     | 가능                                         |

원칙: CRD validation으로 할 수 있으면 그걸 쓰고, 부족할 때 Webhook을 추가한다.
Webhook은 별도 서버가 필요하고 인증서 관리도 해야 해서 복잡도가 올라간다.

## 두 종류

| 종류       | 언제 실행                            | 하는 일            | 비유                                             |
| ---------- | ------------------------------------ | ------------------ | ------------------------------------------------ |
| Mutating   | CR 생성/수정 요청 시 (먼저 실행)     | 값을 자동으로 수정 | 안내데스크: "이름표가 없네요, 제가 붙여드릴게요" |
| Validating | CR 생성/수정 요청 시 (Mutating 다음) | 검증해서 허용/거부 | 경비원: "출입증 없으면 못 들어갑니다"            |

실행 순서:

```text
사용자가 CR 생성/수정 요청
  -> Mutating Webhook: 빠진 값 자동 채움 (기본값 설정 등)
  -> Validating Webhook: 규칙에 맞는지 검증
  -> 통과하면 etcd에 저장
  -> 거부되면 에러 반환
```

## 구현

(다음 단계에서 추가)
