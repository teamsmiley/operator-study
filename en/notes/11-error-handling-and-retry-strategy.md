# Error Handling and Retry Strategy

## One-Line Summary

When the Reconcile function returns an error, controller-runtime automatically retries it.
**How you return** determines how the retry behaves.

---

## Understanding Through Analogy

A delivery person (Controller) attempts a delivery (Reconcile).

| Situation                 | Action                                   | Reconcile Return Value             |
| ------------------------- | ---------------------------------------- | ---------------------------------- |
| Delivery completed        | Move on to the next one                  | `Result{}, nil`                    |
| Nobody home               | Revisit with increasing intervals        | `Result{}, err`                    |
| "Come back in 30 seconds" | Revisit in exactly 30 seconds            | `Result{RequeueAfter: 30s}, nil`   |
| Left at door, need check  | Check once more right away               | `Result{Requeue: true}, nil`       |

---

## 4 Types of Reconcile Return Values

### 1. Success: `return ctrl.Result{}, nil`

```go
// All processing completed successfully. No retry.
// Wait until the next event (CR change, child resource change).
return ctrl.Result{}, nil
```

- Removed from the work queue
- Reconcile is not called again until the next Watch event arrives

### 2. Error + Automatic Backoff: `return ctrl.Result{}, err`

```go
// Transient error such as API call failure.
// controller-runtime retries with Exponential Backoff.
if err := r.Create(ctx, deploy); err != nil {
    return ctrl.Result{}, err
}
```

Retry interval (Exponential Backoff):

```
1st failure  -->  ~1 second later
2nd failure  -->  ~2 seconds later
3rd failure  -->  ~4 seconds later
4th failure  -->  ~8 seconds later
5th failure  -->  ~16 seconds later
...
Max interval -->  ~16 minutes (1000 seconds)
```

- Keeps retrying until success (never gives up)
- The increasing interval prevents overloading the API server
- Even permanently failing errors are retried every 16 minutes (it never stops)

### 3. Retry After Specified Duration: `return ctrl.Result{RequeueAfter: duration}, nil`

```go
// Not an error, but needs to be checked again later.
// Reconcile is called again after exactly the specified duration.
if externalResource.Status == "Provisioning" {
    log.Info("External resource provisioning, rechecking in 30 seconds")
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

- Not an error (returns `nil`) -- does not appear as an error in logs
- Retries at the exact interval without backoff
- Suitable for waiting on external resources or periodic status checks

### 4. Immediate Retry: `return ctrl.Result{Requeue: true}, nil`

```go
// Needs one more immediate check.
// Re-enqueued to the queue immediately.
return ctrl.Result{Requeue: true}, nil
```

- Rarely used
- Be cautious of infinite loop risk

---

## Return Value Selection Guide

```
Did an error occur?
+-- Yes --> Is it a transient error?
|   +-- Yes --> return Result{}, err          (automatic backoff)
|   +-- No  --> Log and return Result{}, nil  (retrying is pointless)
+-- No  --> Does it need to be checked again later?
    +-- Yes --> return Result{RequeueAfter: N}, nil  (recheck after N seconds)
    +-- No  --> return Result{}, nil                  (done)
```

---

## Types of Errors and How to Handle Them

### Transient Error

An error that resolves itself after a short wait. Let the automatic backoff handle the retry.

```go
// Temporary network failure, API server overload, etc.
if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
    return ctrl.Result{}, err  // automatic backoff retry
}
```

Examples: network timeout, API server 503, temporary etcd failure

### Permanent Error

An error that will never resolve no matter how many times you retry. Do not retry.

```go
// Invalid image name -- retrying 100 times won't help
if !isValidImageName(app.Spec.Image) {
    log.Error(nil, "Invalid image name", "image", app.Spec.Image)
    // Record error state in Status
    meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
        Type:    "Available",
        Status:  metav1.ConditionFalse,
        Reason:  "InvalidImage",
        Message: "Invalid image name: " + app.Spec.Image,
    })
    r.Status().Update(ctx, &app)
    return ctrl.Result{}, nil  // Return nil, not err -- no retry
}
```

Examples: invalid spec values, insufficient permissions (RBAC configuration issue), referencing a non-existent resource

Key point: **For permanent errors, do not return err. Instead, record the error state in Status.**

### Waiting Required

Not an error, but the state is not ready yet.

```go
// External load balancer is still provisioning
if lb.Status == "Provisioning" {
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

---

## Error Handling in Our Code

Patterns currently used in `simpleapp_controller.go`:

```go
// 1. CR lookup failure -- ignore if NotFound, retry otherwise
var app myappsv1.SimpleApp
if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
    if errors.IsNotFound(err) {
        return ctrl.Result{}, nil       // Already deleted, nothing to do
    }
    return ctrl.Result{}, err           // Retry for other errors
}

// 2. Deployment creation failure -- retry
if err := r.Create(ctx, deploy); err != nil {
    return ctrl.Result{}, err           // Automatic backoff retry
}

// 3. Deployment update failure -- retry
if err := r.Update(ctx, &deploy); err != nil {
    return ctrl.Result{}, err           // Automatic backoff retry
}

// 4. Status update failure -- retry
if err := r.Status().Update(ctx, &app); err != nil {
    return ctrl.Result{}, err           // Automatic backoff retry
}
```

The current code treats all errors as transient errors.
For a simple Operator, this is sufficient.

---

## Conflict Error (Important!)

The Kubernetes API uses **Optimistic Concurrency Control**.

```
1. Reconcile reads a Deployment (resourceVersion: "100")
2. Meanwhile, someone else modifies the Deployment (resourceVersion: "101")
3. Reconcile attempts an Update --> Conflict error (409)
```

When you return `return ctrl.Result{}, err`, it is automatically retried.
On retry, it reads the latest version, so the conflict resolves naturally.

```go
// No special handling needed for Conflict.
// controller-runtime retries automatically, and the retry reads the latest data.
if err := r.Update(ctx, &deploy); err != nil {
    return ctrl.Result{}, err  // Automatic retry for all errors including Conflict
}
```

This is the strength of the Reconcile pattern.
Because it **reads the current state fresh each time and compares it with the desired state**,
even when a Conflict occurs, it naturally converges in the next Reconcile.

---

## Rate Limiting

controller-runtime's default Rate Limiter combines two strategies:

| Rate Limiter                        | Role                                                                                 |
| ----------------------------------- | ------------------------------------------------------------------------------------ |
| `ItemExponentialFailureRateLimiter` | Increases interval when the same key fails repeatedly (1s -> 2s -> 4s -> ... -> max 1000s) |
| `BucketRateLimiter`                 | Limits overall queue processing rate (10 items/sec, burst of 100)                    |

You can also configure a custom Rate Limiter:

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
                    5*time.Millisecond,  // minimum interval
                    30*time.Second,      // maximum interval (shorter than default 1000s)
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

For most Operators, the default Rate Limiter is sufficient.

---

## Practical Pattern Summary

| Situation                 | Return Value                     | Reason                                |
| ------------------------- | -------------------------------- | ------------------------------------- |
| Processing complete       | `Result{}, nil`                  | Done                                  |
| API call failure          | `Result{}, err`                  | Transient error, automatic retry      |
| CR is NotFound            | `Result{}, nil`                  | Already deleted, nothing to do        |
| Invalid spec value        | `Result{}, nil` + Status update  | Permanent error, retry is pointless   |
| Waiting for external resource | `Result{RequeueAfter: 30s}, nil` | Recheck after a set interval      |
| Conflict (409)            | `Result{}, err`                  | Automatically resolved in next Reconcile |

---

## Summary

```
Reconcile error return
    |
controller-runtime Work Queue
    |
Exponential Backoff (1s -> 2s -> 4s -> ... -> max 1000s)
    |
Infinite retry until success
```

Core principles:

1. **Transient error** --> return `err` (automatic backoff)
2. **Permanent error** --> return `nil` + record error in Status (no retry)
3. **Waiting required** --> use `RequeueAfter` (exact interval)
4. **Don't worry about Conflict** --> the Reconcile pattern resolves it naturally
