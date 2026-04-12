# 08. Finalizer -- CR 삭제 시 정리 작업 보장

## 먼저 OwnerReference가 처리할 수 있는 것

OwnerReference가 설정된 리소스는 **CR 삭제 시 Kubernetes GC가 자동으로 삭제**한다.

조건: 같은 네임스페이스 + 네임스페이스 리소스

```text
SimpleApp (owner)
  ├─ Deployment    ← OwnerReference 가능
  ├─ Service       ← OwnerReference 가능
  ├─ ConfigMap     ← OwnerReference 가능
  └─ Secret        ← OwnerReference 가능

CR 삭제 → 위 리소스 전부 자동 삭제 (Kubernetes GC)
→ Finalizer 필요 없음
```

## OwnerReference가 처리할 수 없는 것

```text
SimpleApp (owner)
  ├─ 다른 네임스페이스의 Secret       ← OwnerReference 불가
  ├─ ClusterRole                      ← OwnerReference 불가
  ├─ AWS RDS 인스턴스                 ← OwnerReference 불가
  ├─ Cloudflare DNS 레코드            ← OwnerReference 불가
  └─ Slack 채널 알림                  ← OwnerReference 불가

CR 삭제 → 위 리소스는 그대로 남아버림!
→ Finalizer가 직접 정리해야 함
```

## OwnerReference의 한계 = Finalizer가 필요한 이유

| OwnerReference 한계               | 예시                                  |
| --------------------------------- | ------------------------------------- |
| 네임스페이스를 넘지 못함          | 다른 네임스페이스의 Secret, ConfigMap |
| 클러스터 범위 리소스에 걸 수 없음 | ClusterRole, ClusterRoleBinding       |
| 클러스터 밖은 대상이 아님         | 클라우드 리소스, 외부 API, 알림       |
| 삭제 전 로직 실행 불가            | graceful shutdown, 로그 기록          |

정리하면:

| 메커니즘       | 누가 실행              | 정리 대상                                |
| -------------- | ---------------------- | ---------------------------------------- |
| OwnerReference | Kubernetes GC (자동)   | 같은 네임스페이스의 자식 리소스          |
| Finalizer      | Controller 코드 (직접) | OwnerReference가 처리 못하는 나머지 전부 |

## Finalizer란?

CR의 metadata.finalizers에 문자열을 추가해서,
**이 문자열이 남아있는 한 Kubernetes가 오브젝트를 삭제하지 못하게** 하는 메커니즘이다.

비유: 퇴사할 때 "법인카드 반납", "계정 비활성화" 같은 체크리스트.
체크리스트가 다 완료되기 전에는 퇴사 처리가 안 된다.

## 동작 원리

```text
일반적인 삭제:
  kubectl delete simpleapp my-app → 바로 삭제됨

Finalizer가 있는 삭제:
  kubectl delete simpleapp my-app
    → Kubernetes: "Finalizer가 있네? 삭제 보류. deletionTimestamp만 찍어둘게"
    → Controller: "deletionTimestamp가 찍혔네? 정리 작업 실행!"
    → Controller: "정리 끝. Finalizer 제거"
    → Kubernetes: "Finalizer가 없어졌네? 이제 진짜 삭제"
```

## 주의: Controller가 죽어있으면?

Finalizer가 남아있는데 Controller가 없으면 CR이 **Terminating 상태로 영원히 멈춘다.**

긴급 상황에서는 수동으로 Finalizer를 제거해야 한다:

```bash
kubectl patch simpleapp my-app -p '{"metadata":{"finalizers":[]}}' --type=merge
```

이러면 정리 작업 없이 강제 삭제된다.
OwnerReference로 연결된 리소스는 GC가 삭제하지만, Finalizer가 처리해야 할 외부 리소스는 수동으로 치워야 한다.

## 구현

커밋: `428f54f`, `48de5f5`

### Finalizer 이름 관례

도메인/용도 형식으로 짓는다:

```go
const simpleAppFinalizer = "apps.example.com/finalizer"
```

### Reconcile 흐름

```text
1. CR 가져오기
2. Finalizer 처리
   ├─ deletionTimestamp가 있으면 (삭제 중):
   │    → 정리 작업 실행
   │    → Finalizer 제거
   │    → return (더 이상 진행하지 않음)
   └─ deletionTimestamp가 없으면 (정상):
        → Finalizer가 없으면 추가 (최초 생성 시)
3. Deployment 생성 또는 업데이트
4. Status 업데이트
```

### 핵심 코드

```go
// 삭제 중인가?
if !app.DeletionTimestamp.IsZero() {
    if controllerutil.ContainsFinalizer(&app, simpleAppFinalizer) {
        // 정리 작업 실행
        log.Info("Finalizer 정리 작업 실행", "name", app.Name)

        // Finalizer 제거 → Kubernetes가 진짜 삭제를 진행한다
        controllerutil.RemoveFinalizer(&app, simpleAppFinalizer)
        if err := r.Update(ctx, &app); err != nil {
            return ctrl.Result{}, err
        }
    }
    return ctrl.Result{}, nil
}

// Finalizer가 아직 없으면 추가한다 (최초 생성 시)
if !controllerutil.ContainsFinalizer(&app, simpleAppFinalizer) {
    controllerutil.AddFinalizer(&app, simpleAppFinalizer)
    if err := r.Update(ctx, &app); err != nil {
        return ctrl.Result{}, err
    }
}
```

### 사용한 함수 (controllerutil 패키지)

| 함수                                 | 역할                    |
| ------------------------------------ | ----------------------- |
| `controllerutil.ContainsFinalizer()` | Finalizer가 있는지 확인 |
| `controllerutil.AddFinalizer()`      | Finalizer 추가          |
| `controllerutil.RemoveFinalizer()`   | Finalizer 제거          |

### RBAC와의 연결

이 marker가 Finalizer 수정 권한을 제공한다:

```go
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps/finalizers,verbs=update
```

06번 노트에서 "아직 안 쓰지만 kubebuilder가 미리 생성"이라고 했던 것이
07번(Status)과 08번(Finalizer)에서 모두 쓰이게 되었다.

## 우리 Operator에 Finalizer가 필요한가?

우리 SimpleApp Operator는 Deployment만 관리하고 OwnerReference도 설정되어 있으므로
사실 Finalizer가 없어도 정상 동작한다. 학습 목적으로 패턴만 넣어둔 것이다.

실무에서 Finalizer를 쓰게 되는 대표적인 Operator:

| Operator                             | Finalizer가 정리하는 것                |
| ------------------------------------ | -------------------------------------- |
| AWS Controllers for Kubernetes (ACK) | S3 버킷, RDS 인스턴스, SQS 큐          |
| ExternalDNS                          | Cloudflare, Route53 DNS 레코드         |
| Crossplane                           | 모든 클라우드 리소스 (AWS, GCP, Azure) |
| cert-manager                         | ACME 인증서 주문 취소                  |

## 다음 단계

Owner Reference와 Garbage Collection을 깊이 이해한다.
