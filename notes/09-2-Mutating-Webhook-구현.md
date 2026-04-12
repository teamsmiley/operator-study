# 09-2. Mutating Webhook 구현 -- Default 함수 작성

커밋: `a76aa7a`

## 변경 파일

`internal/webhook/v1/simpleapp_webhook.go`

## 변경 내용

scaffold가 생성한 빈 Default 함수에 로직을 채웠다.

### Before (scaffold 생성 직후)

```go
func (d *SimpleAppCustomDefaulter) Default(_ context.Context, obj *appsv1.SimpleApp) error {
    simpleapplog.Info("Defaulting for SimpleApp", "name", obj.GetName())

    // TODO(user): fill in your defaulting logic.

    return nil
}
```

### After (로직 추가)

```go
func (d *SimpleAppCustomDefaulter) Default(_ context.Context, obj *appsv1.SimpleApp) error {
    simpleapplog.Info("Defaulting for SimpleApp", "name", obj.GetName())

    // image에 태그가 없으면 :latest 추가
    if !strings.Contains(obj.Spec.Image, ":") {
        obj.Spec.Image = obj.Spec.Image + ":latest"
    }

    return nil
}
```

## 동작 예시

```text
사용자 입력: image: nginx       -> Webhook이 image: nginx:latest 로 변경
사용자 입력: image: nginx:1.25  -> 태그가 있으므로 변경 없음
```

## 구조 이해

scaffold가 생성한 구조를 보면:

```go
// 1. Defaulter 구조체 정의
type SimpleAppCustomDefaulter struct {}

// 2. Default 메서드 구현
func (d *SimpleAppCustomDefaulter) Default(...) error { ... }

// 3. Manager에 등록 (SetupSimpleAppWebhookWithManager 함수)
ctrl.NewWebhookManagedBy(mgr, &appsv1.SimpleApp{}).
    WithDefaulter(&SimpleAppCustomDefaulter{}).
    Complete()
```

Controller의 Reconcile 패턴과 비교:

|           | Controller                     | Mutating Webhook             |
| --------- | ------------------------------ | ---------------------------- |
| 구조체    | SimpleAppReconciler            | SimpleAppCustomDefaulter     |
| 핵심 함수 | Reconcile()                    | Default()                    |
| 호출 시점 | CR 변경 후 (etcd 저장 이후)    | CR 저장 전 (etcd 저장 이전)  |
| 역할      | 현재 상태를 원하는 상태로 맞춤 | 저장 전에 값을 자동으로 수정 |

## 다음 단계

Validating Webhook을 구현한다.
