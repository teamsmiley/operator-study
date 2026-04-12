# 09-4. Validating Webhook 구현 -- Validate 함수 작성

커밋: `0077e73`

## 변경 파일

`internal/webhook/v1/simpleapp_webhook.go`

## 변경 내용

scaffold가 생성한 빈 ValidateCreate, ValidateUpdate 함수에 로직을 채웠다.

### Before (scaffold 생성 직후)

```go
func (v *SimpleAppCustomValidator) ValidateCreate(_ context.Context, obj *appsv1.SimpleApp) (admission.Warnings, error) {
    simpleapplog.Info("Validation for SimpleApp upon creation", "name", obj.GetName())

    // TODO(user): fill in your validation logic upon object creation.

    return nil, nil
}
```

### After (로직 추가)

```go
func (v *SimpleAppCustomValidator) ValidateCreate(_ context.Context, obj *appsv1.SimpleApp) (admission.Warnings, error) {
    simpleapplog.Info("Validation for SimpleApp upon creation", "name", obj.GetName())

    // image에 태그가 포함되어야 한다
    if !strings.Contains(obj.Spec.Image, ":") {
        return nil, fmt.Errorf("image에 태그가 필요합니다 (예: nginx:1.25), 입력값: %s", obj.Spec.Image)
    }

    return nil, nil
}
```

ValidateUpdate도 동일한 검증 로직이다.
단, `newObj`를 검증한다 (`oldObj`는 이전 값).

## 동작 예시

```text
사용자 입력: image: nginx        -> Mutating이 nginx:latest로 변환 -> Validating 통과
사용자 입력: image: nginx:1.25   -> Mutating 변경 없음 -> Validating 통과
```

Mutating이 먼저 실행되므로, 태그 없는 image는 Mutating에서 :latest가 붙고
Validating에 도달할 때는 이미 태그가 있다. 이중 안전장치 역할이다.

## 반환값

| 반환값 | 의미 |
| ------ | ---- |
| `return nil, nil` | 검증 통과 (CR 저장 허용) |
| `return nil, fmt.Errorf(...)` | 검증 실패 (CR 저장 거부, 에러 메시지 반환) |
| `return warnings, nil` | 검증 통과하지만 경고 메시지 표시 |

## Mutating과 Validating 전체 흐름

```text
사용자가 CR 생성/수정 요청
  -> Mutating (Default 함수): 빠진 값 자동 채움
  -> Validating (ValidateCreate/Update 함수): 규칙 검증
     -> 통과: etcd에 저장 -> Controller의 Reconcile 호출
     -> 거부: 에러 반환, 저장 안 됨
```

## 구조 비교

| | Mutating | Validating |
| --- | --- | --- |
| 구조체 | SimpleAppCustomDefaulter | SimpleAppCustomValidator |
| 핵심 함수 | Default() | ValidateCreate(), ValidateUpdate(), ValidateDelete() |
| 반환값 | error (수정 성공/실패) | (Warnings, error) (경고+허용/거부) |
| 할 수 있는 것 | 값 수정 | 허용/거부 판단 |
| 할 수 없는 것 | 거부 (에러 반환 시 수정도 안 됨) | 값 수정 |

## 다음 단계

에러 핸들링 및 Retry 전략을 배운다.
