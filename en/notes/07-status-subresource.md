# 07. Status Subresource -- Recording Current State in the CR

## Why Is It Needed?

Without Status, querying a CR tells you nothing about whether it is actually running properly.

```text
Without Status:
  kubectl get simpleapp my-app
  -> Only NAME is shown. No state information.

After implementing Status:
  kubectl get simpleapp my-app
  -> NAME, REPLICAS, AVAILABLE, and STATUS are shown
```

Analogy: Imagine placing an order at a restaurant with no way to know if your food is ready.
It is like adding an order status board that shows "Preparing" or "Complete."

## Why "Subresource"?

The Kubernetes API accesses resources via URL paths:

```text
/apis/apps.example.com/v1/namespaces/default/simpleapps/my-app          <- resource
/apis/apps.example.com/v1/namespaces/default/simpleapps/my-app/status   <- subresource
```

`/status` is appended as a sub-path under the resource URL, hence "sub-resource."

### Why Are They Separated?

Because spec and status have different update owners:

| Category            | API Path                       | Who Writes It    | Purpose                    |
| ------------------- | ------------------------------ | ---------------- | -------------------------- |
| Resource (spec)     | `.../simpleapps/my-app`        | User (kubectl)   | Declare desired state      |
| Subresource (status)| `.../simpleapps/my-app/status` | Controller       | Report current state       |

If both were updated through the same path, resourceVersion conflicts would occur.
Because the paths are different, a user can modify spec at the same time the Controller modifies status without any conflict.

## How to Enable It

A marker already exists above the SimpleApp struct in `api/v1/simpleapp_types.go`:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status    <- this single line enables the status subresource

type SimpleApp struct {
    ...
    Status SimpleAppStatus `json:"status,omitzero"`
}
```

kubebuilder generates this automatically during scaffolding.
When this marker is present, running `make manifests` adds `subresources.status: {}` to the CRD YAML.

## The SimpleAppStatus Struct

```go
type SimpleAppStatus struct {
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

`metav1.Condition` is the standard Kubernetes condition struct:

| Field              | Meaning                                        | Example                                |
| ------------------ | ---------------------------------------------- | -------------------------------------- |
| Type               | Condition name                                 | "Available", "Progressing", "Degraded" |
| Status             | True / False / Unknown                         | True, False, Unknown                   |
| Reason             | Machine-readable reason (CamelCase)            | "DeploymentReady"                      |
| Message            | Human-readable description                     | "All Pods are running normally"        |
| ObservedGeneration | The CR generation this condition was observed at | app.Generation                        |

## Implementation: Adding Status Updates to Reconcile

Commit: `268328b`

### Changed File

`internal/controller/simpleapp_controller.go`

### Added Import

```go
"k8s.io/apimachinery/pkg/api/meta"
```

This provides the `meta.SetStatusCondition()` and `meta.RemoveStatusCondition()` functions.

### Added Logic (Reconcile Step 4)

Existing Reconcile flow:

```text
1. Fetch the SimpleApp CR
2. Check if the Deployment exists
3. Create or update the Deployment
```

Added step 4:

```text
4. Read the Deployment's state and record it in the SimpleApp status
```

```go
// 4. Status update -- Record the Deployment's actual state in the SimpleApp CR
if deploy.Status.AvailableReplicas == replicas {
    // The desired number of Pods are ready -> Available
    meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
        Type:               "Available",
        Status:             metav1.ConditionTrue,
        Reason:             "DeploymentReady",
        Message:            "All Pods are running normally",
        ObservedGeneration: app.Generation,
    })
    meta.RemoveStatusCondition(&app.Status.Conditions, "Progressing")
} else {
    // Pods are not ready yet -> Progressing
    meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
        Type:               "Progressing",
        Status:             metav1.ConditionTrue,
        Reason:             "DeploymentUpdating",
        Message:            "Pod deployment in progress",
        ObservedGeneration: app.Generation,
    })
    meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
        Type:               "Available",
        Status:             metav1.ConditionFalse,
        Reason:             "DeploymentUpdating",
        Message:            "Pod deployment in progress",
        ObservedGeneration: app.Generation,
    })
}
```

### Key Point: r.Update() vs r.Status().Update()

```go
r.Update(ctx, &app)            // PUT .../simpleapps/my-app         (spec path)
r.Status().Update(ctx, &app)   // PUT .../simpleapps/my-app/status  (status path)
```

When modifying status, you must use `r.Status().Update()`.
If you use `r.Update()`, status changes are silently ignored (when the subresource is enabled).

### Connection to RBAC

The controller already had this marker:

```go
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps/status,verbs=get;update;patch
```

This permission is required for `r.Status().Update()` to work.
In Note 06, we mentioned "not used yet, but kubebuilder generated it in advance" -- now it is actually being used.

## Next Steps

We will learn the Finalizer pattern -- a pattern that guarantees cleanup work when a CR is deleted.
