# 08. Finalizer -- Guaranteeing Cleanup When a CR Is Deleted

## What OwnerReference Can Handle

Resources with OwnerReference set are **automatically deleted by Kubernetes GC when the CR is deleted**.

Condition: same namespace + namespaced resource

```text
SimpleApp (owner)
  +-- Deployment    <- OwnerReference possible
  +-- Service       <- OwnerReference possible
  +-- ConfigMap     <- OwnerReference possible
  +-- Secret        <- OwnerReference possible

CR deleted -> all of the above are automatically deleted (Kubernetes GC)
-> Finalizer not needed
```

## What OwnerReference Cannot Handle

```text
SimpleApp (owner)
  +-- Secret in another namespace        <- OwnerReference not possible
  +-- ClusterRole                        <- OwnerReference not possible
  +-- AWS RDS instance                   <- OwnerReference not possible
  +-- Cloudflare DNS record              <- OwnerReference not possible
  +-- Slack channel notification         <- OwnerReference not possible

CR deleted -> the above resources are left behind!
-> Finalizer must handle the cleanup
```

## OwnerReference Limitations = Why Finalizer Is Needed

| OwnerReference Limitation              | Example                                       |
| -------------------------------------- | --------------------------------------------- |
| Cannot cross namespace boundaries      | Secrets or ConfigMaps in other namespaces      |
| Cannot be set on cluster-scoped resources | ClusterRole, ClusterRoleBinding             |
| Does not cover resources outside the cluster | Cloud resources, external APIs, notifications |
| Cannot execute logic before deletion   | Graceful shutdown, logging                     |

In summary:

| Mechanism      | Who Executes             | Cleanup Target                                    |
| -------------- | ------------------------ | ------------------------------------------------- |
| OwnerReference | Kubernetes GC (automatic)| Child resources in the same namespace              |
| Finalizer      | Controller code (manual) | Everything else that OwnerReference cannot handle  |

## What Is a Finalizer?

A mechanism that adds a string to a CR's metadata.finalizers so that
**Kubernetes cannot delete the object as long as this string remains**.

Analogy: When leaving a company, you have a checklist like "return corporate card" and "deactivate accounts."
The offboarding process cannot be completed until every item on the checklist is checked off.

## How It Works

```text
Normal deletion:
  kubectl delete simpleapp my-app -> deleted immediately

Deletion with a Finalizer:
  kubectl delete simpleapp my-app
    -> Kubernetes: "There's a Finalizer? Deletion on hold. I'll just set deletionTimestamp."
    -> Controller: "deletionTimestamp is set? Running cleanup tasks!"
    -> Controller: "Cleanup done. Removing the Finalizer."
    -> Kubernetes: "Finalizer is gone? Now I can actually delete it."
```

## Warning: What If the Controller Is Down?

If a Finalizer remains but the Controller is not running, the CR **stays stuck in Terminating state forever**.

In an emergency, you must manually remove the Finalizer:

```bash
kubectl patch simpleapp my-app -p '{"metadata":{"finalizers":[]}}' --type=merge
```

This forces deletion without running any cleanup tasks.
Resources connected via OwnerReference will still be deleted by GC, but external resources that the Finalizer was supposed to handle must be cleaned up manually.

## Implementation

Commits: `428f54f`, `48de5f5`

### Finalizer Naming Convention

Use a domain/purpose format:

```go
const simpleAppFinalizer = "apps.example.com/finalizer"
```

### Reconcile Flow

```text
1. Fetch the CR
2. Finalizer handling
   +-- If deletionTimestamp is set (being deleted):
   |    -> Execute cleanup tasks
   |    -> Remove the Finalizer
   |    -> return (do not proceed further)
   +-- If deletionTimestamp is not set (normal operation):
        -> Add the Finalizer if it is not present (on first creation)
3. Create or update the Deployment
4. Update Status
```

### Core Code

```go
// Is it being deleted?
if !app.DeletionTimestamp.IsZero() {
    if controllerutil.ContainsFinalizer(&app, simpleAppFinalizer) {
        // Execute cleanup tasks
        log.Info("Running Finalizer cleanup", "name", app.Name)

        // Remove the Finalizer -> Kubernetes proceeds with actual deletion
        controllerutil.RemoveFinalizer(&app, simpleAppFinalizer)
        if err := r.Update(ctx, &app); err != nil {
            return ctrl.Result{}, err
        }
    }
    return ctrl.Result{}, nil
}

// Add the Finalizer if it is not present yet (on first creation)
if !controllerutil.ContainsFinalizer(&app, simpleAppFinalizer) {
    controllerutil.AddFinalizer(&app, simpleAppFinalizer)
    if err := r.Update(ctx, &app); err != nil {
        return ctrl.Result{}, err
    }
}
```

### Functions Used (controllerutil package)

| Function                             | Role                                |
| ------------------------------------ | ----------------------------------- |
| `controllerutil.ContainsFinalizer()` | Check if the Finalizer is present   |
| `controllerutil.AddFinalizer()`      | Add the Finalizer                   |
| `controllerutil.RemoveFinalizer()`   | Remove the Finalizer                |

### Connection to RBAC

This marker provides the permission to modify Finalizers:

```go
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps/finalizers,verbs=update
```

In Note 06, we mentioned "not used yet, but kubebuilder generated it in advance" --
now it is being used in both Note 07 (Status) and Note 08 (Finalizer).

## Does Our Operator Actually Need a Finalizer?

Our SimpleApp Operator only manages a Deployment, and OwnerReference is already set,
so it would work correctly even without a Finalizer. The pattern was added purely for learning purposes.

Operators that commonly use Finalizers in production:

| Operator                             | What the Finalizer Cleans Up                    |
| ------------------------------------ | ----------------------------------------------- |
| AWS Controllers for Kubernetes (ACK) | S3 buckets, RDS instances, SQS queues           |
| ExternalDNS                          | Cloudflare, Route53 DNS records                 |
| Crossplane                           | All cloud resources (AWS, GCP, Azure)            |
| cert-manager                         | ACME certificate order cancellation              |

## Next Steps

We will gain a deeper understanding of Owner Reference and Garbage Collection.
