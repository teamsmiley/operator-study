# 07. Status Subresource -- 현재 상태를 CR에 기록하기

## 왜 필요한가?

Status가 없으면 CR을 조회해도 "지금 잘 돌아가는지" 알 수 없다.

```text
Status 없이:
  kubectl get simpleapp my-app
  → NAME만 보임. 상태 정보 없음.

Status 구현 후:
  kubectl get simpleapp my-app
  → NAME, REPLICAS, AVAILABLE, STATUS가 보임
```

비유: 식당에 주문만 넣고 음식이 나왔는지 알 방법이 없는 상태에서,
주문 현황판에 "준비 중", "완료"가 표시되는 것과 같다.

## 왜 "Subresource"인가?

Kubernetes API는 URL 경로로 리소스에 접근한다:

```text
/apis/apps.example.com/v1/namespaces/default/simpleapps/my-app          ← 리소스
/apis/apps.example.com/v1/namespaces/default/simpleapps/my-app/status   ← 서브리소스
```

리소스 URL 아래에 `/status`가 하위 경로로 붙으니까 sub-resource다.

### 왜 분리했는가?

spec과 status의 업데이트 주체가 다르기 때문이다:

| 구분                | API 경로                       | 누가 쓰나        | 용도             |
| ------------------- | ------------------------------ | ---------------- | ---------------- |
| 리소스 (spec)       | `.../simpleapps/my-app`        | 사용자 (kubectl) | 원하는 상태 선언 |
| 서브리소스 (status) | `.../simpleapps/my-app/status` | Controller       | 현재 상태 보고   |

같은 경로로 업데이트하면 resourceVersion 충돌이 발생한다.
경로가 다르니까 사용자가 spec을 수정하는 동시에 Controller가 status를 수정해도 충돌이 없다.

## 활성화 방법

`api/v1/simpleapp_types.go`의 SimpleApp 구조체 위에 marker가 이미 있다:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status    ← 이 한 줄이 status subresource를 활성화

type SimpleApp struct {
    ...
    Status SimpleAppStatus `json:"status,omitzero"`
}
```

kubebuilder가 scaffold할 때 자동으로 만들어 준다.
이 marker가 있으면 `make manifests` 시 CRD YAML에 `subresources.status: {}`가 추가된다.

## SimpleAppStatus 구조체

```go
type SimpleAppStatus struct {
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

`metav1.Condition`은 Kubernetes 표준 조건 구조체다:

| 필드               | 의미                            | 예시                                   |
| ------------------ | ------------------------------- | -------------------------------------- |
| Type               | 조건 이름                       | "Available", "Progressing", "Degraded" |
| Status             | 참/거짓/알 수 없음              | True, False, Unknown                   |
| Reason             | 기계가 읽는 이유 (CamelCase)    | "DeploymentReady"                      |
| Message            | 사람이 읽는 설명                | "모든 Pod가 정상 동작 중"              |
| ObservedGeneration | 이 조건이 관찰한 CR의 세대 번호 | app.Generation                         |

## 구현: Reconcile에 status 업데이트 추가

커밋: `268328b`

### 변경 파일

`internal/controller/simpleapp_controller.go`

### 추가된 import

```go
"k8s.io/apimachinery/pkg/api/meta"
```

`meta.SetStatusCondition()`, `meta.RemoveStatusCondition()` 함수를 제공한다.

### 추가된 로직 (Reconcile 4단계)

기존 Reconcile 흐름:

```text
1. SimpleApp CR 가져오기
2. Deployment 존재 확인
3. Deployment 생성 또는 업데이트
```

추가된 4단계:

```text
4. Deployment 상태를 읽어서 SimpleApp status에 기록
```

```go
// 4. Status 업데이트 -- Deployment의 실제 상태를 SimpleApp CR에 기록한다
if deploy.Status.AvailableReplicas == replicas {
    // 원하는 수만큼 Pod가 준비됨 → Available
    meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
        Type:               "Available",
        Status:             metav1.ConditionTrue,
        Reason:             "DeploymentReady",
        Message:            "모든 Pod가 정상 동작 중",
        ObservedGeneration: app.Generation,
    })
    meta.RemoveStatusCondition(&app.Status.Conditions, "Progressing")
} else {
    // 아직 Pod가 준비되지 않음 → Progressing
    meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
        Type:               "Progressing",
        Status:             metav1.ConditionTrue,
        Reason:             "DeploymentUpdating",
        Message:            "Pod 배포 진행 중",
        ObservedGeneration: app.Generation,
    })
    meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
        Type:               "Available",
        Status:             metav1.ConditionFalse,
        Reason:             "DeploymentUpdating",
        Message:            "Pod 배포 진행 중",
        ObservedGeneration: app.Generation,
    })
}
```

### 핵심: r.Update() vs r.Status().Update()

```go
r.Update(ctx, &app)            // PUT .../simpleapps/my-app         (spec 경로)
r.Status().Update(ctx, &app)   // PUT .../simpleapps/my-app/status  (status 경로)
```

status를 수정할 때는 반드시 `r.Status().Update()`를 써야 한다.
`r.Update()`를 쓰면 status 변경이 무시된다 (subresource가 활성화된 경우).

### RBAC와의 연결

controller에 이미 이 marker가 있었다:

```go
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps/status,verbs=get;update;patch
```

이 권한이 있어야 `r.Status().Update()`가 동작한다.
06번 노트에서 "아직 안 쓰지만 kubebuilder가 미리 생성"이라고 했던 것이 이제 실제로 쓰이게 되었다.

## 다음 단계

Finalizer 패턴을 배운다. CR 삭제 시 정리 작업을 보장하는 패턴이다.
