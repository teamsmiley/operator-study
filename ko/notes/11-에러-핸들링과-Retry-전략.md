# 에러 핸들링과 Retry 전략

## 한 줄 요약

Reconcile 함수가 에러를 반환하면 controller-runtime이 자동으로 재시도하는데,
**어떻게 반환하느냐**에 따라 재시도 방식이 달라진다.

---

## 비유로 이해하기

택배 배달원(Controller)이 배달(Reconcile)을 시도한다.

| 상황                     | 행동                      | Reconcile 반환값                 |
| ------------------------ | ------------------------- | -------------------------------- |
| 배달 완료                | 다음 건으로 넘어감        | `Result{}, nil`                  |
| 집에 아무도 없음         | 점점 간격 늘려가며 재방문 | `Result{}, err`                  |
| "30초 후에 오세요"       | 정확히 30초 후 재방문     | `Result{RequeueAfter: 30s}, nil` |
| 문 앞에 놨는데 확인 필요 | 바로 한 번 더 확인        | `Result{Requeue: true}, nil`     |

---

## Reconcile 반환값 4가지

### 1. 성공: `return ctrl.Result{}, nil`

```go
// 모든 처리가 정상 완료됨. 재시도 없음.
// 다음 이벤트(CR 변경, 자식 리소스 변경)가 올 때까지 대기.
return ctrl.Result{}, nil
```

- Work queue에서 제거된다
- 다음 Watch 이벤트가 올 때까지 Reconcile이 호출되지 않는다

### 2. 에러 + 자동 백오프: `return ctrl.Result{}, err`

```go
// API 호출 실패 등 일시적 에러.
// controller-runtime이 Exponential Backoff로 재시도한다.
if err := r.Create(ctx, deploy); err != nil {
    return ctrl.Result{}, err
}
```

재시도 간격 (Exponential Backoff):

```
1번째 실패 →  ~1초 후
2번째 실패 →  ~2초 후
3번째 실패 →  ~4초 후
4번째 실패 →  ~8초 후
5번째 실패 →  ~16초 후
...
최대 간격  →  ~16분 (1000초)
```

- 성공할 때까지 계속 재시도한다 (포기하지 않는다)
- 간격이 점점 늘어나므로 API 서버에 부하를 주지 않는다
- 영원히 실패하는 에러도 16분마다 재시도한다 (멈추지 않음)

### 3. 지정 시간 후 재시도: `return ctrl.Result{RequeueAfter: duration}, nil`

```go
// 에러는 아니지만, 나중에 다시 확인이 필요한 경우.
// 정확히 지정한 시간 후에 Reconcile이 다시 호출된다.
if externalResource.Status == "Provisioning" {
    log.Info("외부 리소스 준비 중, 30초 후 재확인")
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

- 에러가 아님 (`nil` 반환) -- 로그에 에러로 찍히지 않는다
- 백오프 없이 정확한 간격으로 재시도한다
- 외부 리소스 대기, 주기적 상태 체크에 적합하다

### 4. 즉시 재시도: `return ctrl.Result{Requeue: true}, nil`

```go
// 한 번 더 바로 확인이 필요한 경우.
// 큐에 즉시 다시 넣는다.
return ctrl.Result{Requeue: true}, nil
```

- 거의 사용하지 않는다
- 무한 루프 위험이 있으므로 주의

---

## 반환값 선택 가이드

```
에러가 발생했나?
├── 예 → 일시적 에러인가?
│   ├── 예 → return Result{}, err          (자동 백오프)
│   └── 아니오 → 로그 남기고 return Result{}, nil  (재시도 의미 없음)
└── 아니오 → 나중에 다시 확인 필요한가?
    ├── 예 → return Result{RequeueAfter: N}, nil  (N초 후 재확인)
    └── 아니오 → return Result{}, nil              (완료)
```

---

## 에러의 종류와 대응

### 일시적 에러 (Transient Error)

잠깐 기다리면 해결되는 에러. 자동 백오프로 재시도하면 된다.

```go
// 네트워크 일시 장애, API 서버 과부하 등
if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
    return ctrl.Result{}, err  // 자동 백오프 재시도
}
```

예시: 네트워크 타임아웃, API 서버 503, etcd 일시 장애

### 영구적 에러 (Permanent Error)

재시도해도 절대 해결되지 않는 에러. 재시도하면 안 된다.

```go
// 잘못된 이미지 이름 -- 100번 재시도해도 안 됨
if !isValidImageName(app.Spec.Image) {
    log.Error(nil, "잘못된 이미지 이름", "image", app.Spec.Image)
    // Status에 에러 상태 기록
    meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
        Type:    "Available",
        Status:  metav1.ConditionFalse,
        Reason:  "InvalidImage",
        Message: "잘못된 이미지 이름: " + app.Spec.Image,
    })
    r.Status().Update(ctx, &app)
    return ctrl.Result{}, nil  // err가 아닌 nil 반환 -- 재시도 안 함
}
```

예시: 잘못된 spec 값, 권한 부족 (RBAC 설정 문제), 존재하지 않는 리소스 참조

핵심: **영구적 에러는 err를 반환하지 않고, Status에 에러 상태를 기록한다.**

### 대기 필요 에러

에러는 아니지만 아직 준비되지 않은 상태.

```go
// 외부 로드밸런서가 아직 프로비저닝 중
if lb.Status == "Provisioning" {
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

---

## 우리 코드에서의 에러 핸들링

`simpleapp_controller.go`에서 현재 사용 중인 패턴:

```go
// 1. CR 조회 실패 -- NotFound면 무시, 그 외는 재시도
var app myappsv1.SimpleApp
if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
    if errors.IsNotFound(err) {
        return ctrl.Result{}, nil       // 삭제된 거니까 무시
    }
    return ctrl.Result{}, err           // 그 외 에러는 재시도
}

// 2. Deployment 생성 실패 -- 재시도
if err := r.Create(ctx, deploy); err != nil {
    return ctrl.Result{}, err           // 자동 백오프 재시도
}

// 3. Deployment 업데이트 실패 -- 재시도
if err := r.Update(ctx, &deploy); err != nil {
    return ctrl.Result{}, err           // 자동 백오프 재시도
}

// 4. Status 업데이트 실패 -- 재시도
if err := r.Status().Update(ctx, &app); err != nil {
    return ctrl.Result{}, err           // 자동 백오프 재시도
}
```

현재 코드는 모든 에러를 일시적 에러로 취급하고 있다.
단순한 Operator에서는 이 정도면 충분하다.

---

## Conflict 에러 (중요!)

쿠버네티스 API는 **낙관적 동시성 제어 (Optimistic Concurrency)** 를 사용한다.

```
1. Reconcile이 Deployment를 읽음 (resourceVersion: "100")
2. 그 사이에 다른 누군가가 Deployment를 수정 (resourceVersion: "101")
3. Reconcile이 Update 시도 → Conflict 에러 (409)
```

이때 `return ctrl.Result{}, err`로 반환하면 자동으로 재시도된다.
재시도 시 최신 버전을 다시 읽으므로 자연스럽게 해결된다.

```go
// Conflict는 별도 처리 없이 err 반환만 하면 된다.
// controller-runtime이 알아서 재시도하고, 재시도 시 최신 데이터를 읽는다.
if err := r.Update(ctx, &deploy); err != nil {
    return ctrl.Result{}, err  // Conflict 포함 모든 에러 자동 재시도
}
```

이것이 Reconcile 패턴의 강점이다.
**매번 현재 상태를 새로 읽고 원하는 상태와 비교**하기 때문에,
Conflict가 발생해도 다음 Reconcile에서 자연스럽게 수렴한다.

---

## Rate Limiting (속도 제한)

controller-runtime의 기본 Rate Limiter는 두 가지를 조합한다:

| Rate Limiter                        | 역할                                                                      |
| ----------------------------------- | ------------------------------------------------------------------------- |
| `ItemExponentialFailureRateLimiter` | 같은 키가 반복 실패하면 간격을 늘림 (1초 → 2초 → 4초 → ... → 최대 1000초) |
| `BucketRateLimiter`                 | 전체 큐의 처리 속도를 제한 (초당 10건, 버스트 100건)                      |

커스텀 Rate Limiter를 설정할 수도 있다:

```go
import "sigs.k8s.io/controller-runtime/pkg/controller"
import "golang.org/x/time/rate"
import "k8s.io/client-go/util/workqueue"

func (r *SimpleAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&myappsv1.SimpleApp{}).
        Owns(&appsv1.Deployment{}).
        WithOptions(controller.Options{
            RateLimiter: workqueue.NewTypedMaxOfRateLimiter(
                workqueue.NewTypedItemExponentialFailureRateLimiter[ctrl.Request](
                    5*time.Millisecond,  // 최소 간격
                    30*time.Second,      // 최대 간격 (기본 1000초보다 짧게)
                ),
                &workqueue.TypedBucketRateLimiter[ctrl.Request]{
                    Limiter: rate.NewLimiter(rate.Limit(10), 100),
                },
            ),
        }).
        Named("simpleapp").
        Complete(r)
}
```

대부분의 Operator에서는 기본 Rate Limiter로 충분하다.

---

## 실무 패턴 요약

| 상황                | 반환값                           | 이유                         |
| ------------------- | -------------------------------- | ---------------------------- |
| 처리 완료           | `Result{}, nil`                  | 끝                           |
| API 호출 실패       | `Result{}, err`                  | 일시적 에러, 자동 재시도     |
| CR이 NotFound       | `Result{}, nil`                  | 이미 삭제됨, 할 일 없음      |
| spec 값이 잘못됨    | `Result{}, nil` + Status 기록    | 영구적 에러, 재시도 무의미   |
| 외부 리소스 대기 중 | `Result{RequeueAfter: 30s}, nil` | 일정 시간 후 재확인          |
| Conflict (409)      | `Result{}, err`                  | 다음 Reconcile에서 자동 해결 |

---

## 정리

```
Reconcile 에러 반환
    ↓
controller-runtime Work Queue
    ↓
Exponential Backoff (1초 → 2초 → 4초 → ... → 최대 1000초)
    ↓
성공할 때까지 무한 재시도
```

핵심 원칙:

1. **일시적 에러** → `err` 반환 (자동 백오프)
2. **영구적 에러** → `nil` 반환 + Status에 에러 기록 (재시도 안 함)
3. **대기 필요** → `RequeueAfter` 사용 (정확한 간격)
4. **Conflict는 신경 쓰지 마라** → Reconcile 패턴이 자연스럽게 해결한다
