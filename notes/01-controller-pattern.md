# 01. Kubernetes Controller 패턴 (Reconciliation Loop)

## 핵심 개념: 선언적(Declarative) 시스템

Kubernetes는 "어떻게 해라(Imperative)"가 아니라 **"이렇게 되어야 한다(Declarative)"를 선언**하면,
Controller가 현재 상태를 원하는 상태로 맞춰가는 구조이다.

## Reconciliation Loop (조정 루프)

Controller의 동작 원리는 에어컨 온도 조절기와 같다:

1. **Observe** - 현재 상태(Current State)를 확인한다
2. **Diff** - 원하는 상태(Desired State)와 비교한다
3. **Act** - 차이가 있으면 조정한다
4. 이 과정을 **끊임없이 반복**한다

```text
         +---> Observe (현재 상태 확인)
         |              |
         |              v
         +--- Diff (원하는 상태와 비교)
         |              |
         |              v
         +--- Act (차이가 있으면 조정)
         |              |
         +------<-------+
              (반복)
```

## 내장 Controller 예시

| Controller            | Desired State (선언) | Act (행동)                           |
| --------------------- | -------------------- | ------------------------------------ |
| ReplicaSet Controller | `replicas: 3`        | Pod가 2개면 1개 생성, 4개면 1개 삭제 |
| Deployment Controller | `image: nginx:1.25`  | 새 ReplicaSet 생성, 이전 것 축소     |
| Service Controller    | `type: LoadBalancer` | 클라우드 LB 프로비저닝               |

## 이 패턴이 강력한 3가지 이유

1. **자가 치유(Self-healing)**: Pod가 죽으면 Controller가 감지하고 다시 만든다
2. **멱등성(Idempotency)**: Reconcile을 몇 번 실행해도 결과가 같다. 이미 원하는 상태면 아무것도 안 한다
3. **장애 내성**: Controller가 잠시 죽었다 살아나도, 다시 Reconcile하면 된다. 중간 상태를 기억할 필요가 없다

## `kubectl apply` 전체 흐름

`kubectl apply -f deployment.yaml` 실행 시 실제 동작 순서:

```text
kubectl apply
    |
    v
API Server (etcd에 Deployment 오브젝트 저장)
    |
    v
Deployment Controller (Watch하다가 감지 -> ReplicaSet 생성)
    |
    v
ReplicaSet Controller (Watch하다가 감지 -> Pod 오브젝트 생성)
    |
    v
Scheduler (Watch하다가 미배정 Pod 감지 -> Node 배정)
    |
    v
kubelet (자기 Node에 배정된 Pod 감지 -> 컨테이너 실행)
```

### 핵심 포인트

- **API Server**는 단순한 저장소 + 이벤트 허브이다. 직접 뭔가를 만들지 않는다
- **각 Controller는 자기 책임 영역만 처리**한다 (Deployment -> ReplicaSet, ReplicaSet -> Pod)
- **kubelet도 사실 Controller**이다. 자기 Node에 배정된 Pod를 Watch하고 컨테이너를 만든다
- 이 체인 구조가 Controller 패턴의 핵심이다
- 우리가 만들 **Custom Controller도 이 체인에 하나 더 끼어드는 것**이다

## Controller는 언제 동작하는가? (Watch vs Polling)

Controller는 타이머로 몇 분마다 확인하는 방식(Polling)이 **아니다.**
**Watch(이벤트 기반)** 로 동작한다.

### Watch 동작 방식

```text
API Server                          Controller
    |                                   |
    |  --- Watch 연결 (HTTP long poll) -->|
    |                                   |  (대기 중... 아무것도 안 함)
    |                                   |
  [누군가 리소스 변경]                     |
    |                                   |
    |  --- 이벤트 전송 (즉시!) ----------->|
    |                                   |  Reconcile 실행!
    |                                   |
    |  (대기 중...)                       |  (대기 중...)
```

- 변경이 생기면 **즉시(수 밀리초 이내)** 알림을 받는다
- 아무 변경이 없으면 **아무것도 하지 않는다**

### Reconcile을 트리거하는 3가지 방식

| 트리거           | 타이밍        | 용도                     |
| ---------------- | ------------- | ------------------------ |
| Watch (이벤트)   | 즉시 (밀리초) | 주요 동작 방식           |
| Resync Period    | 기본 10시간   | 놓친 이벤트 대비 안전망  |
| Requeue (재요청) | 개발자가 지정 | Reconcile 실패 시 재시도 |

### Resync Period (재동기화 주기)

이벤트를 혹시 놓쳤을 경우를 대비한 **안전망**이다.
기본값은 controller-runtime에서 **10시간**이며, 이때 모든 리소스에 대해 Reconcile을 한 번씩 다시 돌린다.

```go
// controller-runtime에서 Resync 설정 예시
mgr, err := ctrl.NewManager(cfg, ctrl.Options{
    Cache: cache.Options{
        SyncPeriod: &syncPeriod, // 기본 10시간
    },
})
```

### Requeue (재요청)

Reconcile 도중 외부 API 호출 실패 등의 상황에서, Controller가 스스로 "N초 후에 다시 시도해줘"라고 요청할 수 있다.

```go
// Reconcile 함수 안에서 30초 후 재시도 요청
return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
```

## 용어 정리

| 용어                     | 설명                                                 |
| ------------------------ | ---------------------------------------------------- |
| Desired State            | 사용자가 선언한 원하는 상태 (YAML의 spec)            |
| Current State            | 클러스터에서 실제 관찰되는 현재 상태                 |
| Reconcile                | Desired State와 Current State의 차이를 조정하는 행위 |
| Watch                    | API Server의 변경 이벤트를 실시간으로 구독하는 것    |
| Idempotency (멱등성)     | 같은 작업을 여러 번 수행해도 결과가 동일한 성질      |
| Self-healing (자가 치유) | 장애 발생 시 자동으로 원하는 상태로 복구하는 능력    |
| Resync Period            | 놓친 이벤트 대비 주기적 전체 재조정 (기본 10시간)    |
| Requeue                  | Reconcile 실패 시 지정 시간 후 재시도 요청           |
