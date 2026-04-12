# Owner Reference와 Garbage Collection

## 한 줄 요약

Owner Reference는 "이 리소스의 주인은 누구다"를 쿠버네티스에게 알려주는 것이고,
Garbage Collection은 "주인이 사라지면 소유물도 자동으로 치워주는" 쿠버네티스 내장 기능이다.

---

## 비유로 이해하기

건물(SimpleApp)을 철거하면 안에 있는 가구(Deployment)도 함께 철거된다.
단, **건물 등기부에 "이 가구는 이 건물 소유"라고 적어놔야** 자동 철거가 된다.
등기를 안 하면? 건물은 사라지고 가구만 덩그러니 남는다 (고아 리소스).

| 비유 | 쿠버네티스 |
|------|-----------|
| 건물 | SimpleApp (부모, Owner) |
| 가구 | Deployment (자식, Owned) |
| 등기부 등록 | `SetControllerReference()` 호출 |
| 자동 철거 | Garbage Collection |
| 등기 안 한 가구 | 고아(Orphan) 리소스 |

---

## Owner Reference란?

리소스의 `metadata.ownerReferences` 필드에 부모 정보를 기록하는 것이다.

### 실제 YAML 예시

SimpleApp `myapp`이 만든 Deployment를 조회하면 이렇게 보인다:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  namespace: default
  ownerReferences:           # <-- 이 필드!
  - apiVersion: apps.example.com/v1
    kind: SimpleApp
    name: myapp
    uid: abcd-1234-efgh-5678
    controller: true
    blockOwnerDeletion: true
```

### 각 필드의 의미

| 필드 | 의미 |
|------|------|
| `apiVersion`, `kind` | 부모의 API 그룹과 종류 |
| `name` | 부모의 이름 |
| `uid` | 부모의 고유 ID (같은 이름의 다른 리소스와 구분) |
| `controller: true` | 이 부모가 "컨트롤러 역할"임을 표시 (하나만 가능) |
| `blockOwnerDeletion: true` | 자식이 삭제 완료될 때까지 부모 삭제를 블로킹 |

---

## 코드에서 설정하는 방법

### ctrl.SetControllerReference (우리 코드)

`simpleapp_controller.go` 99~101번째 줄:

```go
// OwnerReference 설정 -- SimpleApp이 삭제되면 Deployment도 같이 삭제된다
if err := ctrl.SetControllerReference(&app, deploy, r.Scheme); err != nil {
    return ctrl.Result{}, err
}
```

이 한 줄이 Deployment의 `metadata.ownerReferences`에 SimpleApp 정보를 채워준다.

### SetControllerReference vs SetOwnerReference

| 함수 | controller 필드 | 특징 |
|------|----------------|------|
| `ctrl.SetControllerReference()` | `true`로 설정 | 부모 하나만 가능 (이걸 주로 씀) |
| `controllerutil.SetOwnerReference()` | `false`로 설정 | 부모 여러 개 가능 |

Operator에서는 거의 항상 `SetControllerReference()`를 사용한다.
하나의 리소스는 하나의 컨트롤러만 관리하는 게 원칙이기 때문이다.

---

## Garbage Collection 동작 방식

쿠버네티스에는 **Garbage Collector**라는 컨트롤러가 항상 돌고 있다.

### 삭제 흐름

```
1. kubectl delete simpleapp myapp
2. 쿠버네티스 API 서버가 SimpleApp 삭제 처리
3. Garbage Collector가 확인:
   "ownerReferences에 myapp(uid: abcd-1234)을 부모로 가진 리소스가 있나?"
4. Deployment 발견 → 자동 삭제
5. Deployment가 소유한 ReplicaSet도 연쇄 삭제
6. ReplicaSet이 소유한 Pod도 연쇄 삭제
```

이것을 **Cascading Deletion (계단식 삭제)** 이라고 한다.

### 삭제 정책 (Propagation Policy)

`kubectl delete` 시 `--cascade` 옵션으로 동작을 바꿀 수 있다:

| 정책 | 동작 |
|------|------|
| `Foreground` (기본) | 자식 먼저 삭제 → 부모 삭제 |
| `Background` | 부모 즉시 삭제 → 자식은 Garbage Collector가 나중에 정리 |
| `Orphan` | 부모만 삭제, 자식은 고아로 남김 |

```bash
# 고아 정책 -- Deployment는 남기고 SimpleApp만 삭제
kubectl delete simpleapp myapp --cascade=orphan
```

---

## Finalizer와 Owner Reference 비교

| | Finalizer | Owner Reference |
|--|-----------|-----------------|
| 누가 삭제하나 | **내 코드**가 직접 정리 | **쿠버네티스**가 자동 정리 |
| 대상 | 외부 리소스 (DB, DNS, S3 등) | K8s 내부 리소스 (Deployment, Service 등) |
| 설정 안 하면 | 정리 코드가 실행 안 됨 | 자식이 고아로 남음 |
| 자동인가 | 아님 (코드 작성 필요) | 관계 설정만 하면 삭제는 자동 |

**실무 패턴**: 둘 다 같이 쓴다.
- Owner Reference → K8s 안의 Deployment, Service, ConfigMap 등 자동 정리
- Finalizer → K8s 밖의 외부 자원 (AWS 로드밸런서, DNS 레코드 등) 직접 정리

---

## 주의사항

### 1. Namespace 제약

Owner와 Owned 리소스는 **같은 Namespace**에 있어야 한다.
Namespace 리소스가 Cluster-scoped 리소스를 소유할 수 없다.

```
SimpleApp (namespace: default) → Deployment (namespace: default)  -- 가능
SimpleApp (namespace: default) → ClusterRole (cluster-scoped)     -- 불가능
```

Cluster-scoped 리소스를 정리해야 한다면 Finalizer를 사용해야 한다.

### 2. Owns()로 Watch 등록

`SetupWithManager`에서 `Owns()`를 호출하면, 자식 리소스가 변경될 때도
Reconcile이 호출된다:

```go
func (r *SimpleAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&myappsv1.SimpleApp{}).
        Owns(&appsv1.Deployment{}). // Deployment 변경도 감지한다
        Named("simpleapp").
        Complete(r)
}
```

`Owns()`는 ownerReferences를 보고 어떤 부모의 Reconcile을 호출할지 결정한다.
즉, Owner Reference는 삭제뿐 아니라 **Watch 연결**에도 사용된다.

### 3. SetControllerReference는 Create 전에 호출

Owner Reference는 리소스를 생성하기 **전에** 설정해야 한다.
이미 생성된 리소스에 나중에 붙이려면 Update가 필요하다.

```go
// 올바른 순서
deploy := r.buildDeployment(&app)
ctrl.SetControllerReference(&app, deploy, r.Scheme)  // 1. 먼저 설정
r.Create(ctx, deploy)                                 // 2. 그 다음 생성
```

---

## 정리

```
Owner Reference 설정 (개발자가 직접)
        ↓
metadata.ownerReferences에 부모 정보 기록
        ↓
부모 삭제 시 → Garbage Collector가 자식 자동 삭제 (Cascading Deletion)
        +
Owns()로 자식 변경 감지 → 부모의 Reconcile 호출
```

핵심: "관계를 맺어주는 것"은 수동, "삭제하는 것"은 자동이다.
