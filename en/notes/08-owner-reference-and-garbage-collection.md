# Owner Reference and Garbage Collection

## One-Line Summary

Owner Reference tells Kubernetes "who owns this resource,"
and Garbage Collection is the built-in Kubernetes feature that "automatically cleans up owned resources when the owner is deleted."

---

## Understanding Through Analogy

When a building (SimpleApp) is demolished, the furniture (Deployment) inside is demolished along with it.
However, **the furniture must be registered in the building's deed as "owned by this building"** for automatic demolition to happen.
If you skip the registration? The building disappears, but the furniture is left behind all alone (orphan resource).

| Analogy                  | Kubernetes                              |
|--------------------------|-----------------------------------------|
| Building                 | SimpleApp (parent, Owner)               |
| Furniture                | Deployment (child, Owned)               |
| Deed registration        | Calling `SetControllerReference()`      |
| Automatic demolition     | Garbage Collection                      |
| Unregistered furniture   | Orphan resource                         |

---

## What Is Owner Reference?

It is the act of recording the parent's information in the `metadata.ownerReferences` field of a resource.

### Actual YAML Example

When you inspect the Deployment created by SimpleApp `myapp`, it looks like this:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  namespace: default
  ownerReferences:           # <-- this field!
  - apiVersion: apps.example.com/v1
    kind: SimpleApp
    name: myapp
    uid: abcd-1234-efgh-5678
    controller: true
    blockOwnerDeletion: true
```

### Meaning of Each Field

| Field | Meaning |
|-------|---------|
| `apiVersion`, `kind` | The parent's API group and kind |
| `name` | The parent's name |
| `uid` | The parent's unique ID (to distinguish from other resources with the same name) |
| `controller: true` | Indicates this parent serves as the "controller" (only one is allowed) |
| `blockOwnerDeletion: true` | Blocks the parent's deletion until the child is fully deleted |

---

## How to Set It in Code

### ctrl.SetControllerReference (Our Code)

Lines 99-101 of `simpleapp_controller.go`:

```go
// Set OwnerReference -- when SimpleApp is deleted, the Deployment is deleted too
if err := ctrl.SetControllerReference(&app, deploy, r.Scheme); err != nil {
    return ctrl.Result{}, err
}
```

This single line populates the Deployment's `metadata.ownerReferences` with the SimpleApp's information.

### SetControllerReference vs SetOwnerReference

| Function | controller field | Characteristics |
|----------|-----------------|-----------------|
| `ctrl.SetControllerReference()` | Set to `true` | Only one parent allowed (this is what you typically use) |
| `controllerutil.SetOwnerReference()` | Set to `false` | Multiple parents allowed |

In Operators, you almost always use `SetControllerReference()`.
The principle is that a single resource should be managed by exactly one controller.

---

## How Garbage Collection Works

Kubernetes has a **Garbage Collector** controller that runs continuously.

### Deletion Flow

```
1. kubectl delete simpleapp myapp
2. Kubernetes API server processes the SimpleApp deletion
3. Garbage Collector checks:
   "Are there any resources listing myapp (uid: abcd-1234) as their parent in ownerReferences?"
4. Deployment found -> automatically deleted
5. The ReplicaSet owned by the Deployment is also cascade-deleted
6. The Pods owned by the ReplicaSet are also cascade-deleted
```

This is called **Cascading Deletion**.

### Deletion Policy (Propagation Policy)

You can change the behavior with the `--cascade` option when running `kubectl delete`:

| Policy | Behavior |
|--------|----------|
| `Foreground` (default) | Delete children first, then delete the parent |
| `Background` | Delete the parent immediately; children are cleaned up later by the Garbage Collector |
| `Orphan` | Delete the parent only; children are left as orphans |

```bash
# Orphan policy -- delete only SimpleApp, leaving the Deployment behind
kubectl delete simpleapp myapp --cascade=orphan
```

---

## Finalizer vs Owner Reference Comparison

| | Finalizer | Owner Reference |
|--|-----------|-----------------|
| Who deletes? | **Your code** handles cleanup directly | **Kubernetes** handles cleanup automatically |
| Target | External resources (DB, DNS, S3, etc.) | K8s internal resources (Deployment, Service, etc.) |
| If not configured | Cleanup code does not run | Children are left as orphans |
| Automatic? | No (requires writing code) | Deletion is automatic once the relationship is set |

**Production pattern**: Use both together.
- Owner Reference: automatic cleanup for K8s resources like Deployments, Services, ConfigMaps
- Finalizer: manual cleanup for external resources outside K8s (AWS load balancers, DNS records, etc.)

---

## Important Notes

### 1. Namespace Constraint

Owner and Owned resources must be in the **same Namespace**.
A namespaced resource cannot own a cluster-scoped resource.

```
SimpleApp (namespace: default) -> Deployment (namespace: default)  -- allowed
SimpleApp (namespace: default) -> ClusterRole (cluster-scoped)     -- not allowed
```

If you need to clean up cluster-scoped resources, you must use a Finalizer.

### 2. Watch Registration with Owns()

Calling `Owns()` in `SetupWithManager` causes Reconcile to be triggered when child resources change as well:

```go
func (r *SimpleAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&myappsv1.SimpleApp{}).
        Owns(&appsv1.Deployment{}). // Also detects Deployment changes
        Named("simpleapp").
        Complete(r)
}
```

`Owns()` examines ownerReferences to determine which parent's Reconcile to invoke.
In other words, Owner Reference is used not only for deletion but also for **watch registration**.

### 3. Call SetControllerReference Before Create

Owner Reference must be set **before** creating the resource.
If you want to attach it to an already-created resource, an Update is required.

```go
// Correct order
deploy := r.buildDeployment(&app)
ctrl.SetControllerReference(&app, deploy, r.Scheme)  // 1. Set it first
r.Create(ctx, deploy)                                 // 2. Then create
```

---

## Summary

```
Set Owner Reference (done manually by the developer)
        |
        v
Parent info is recorded in metadata.ownerReferences
        |
        v
When parent is deleted -> Garbage Collector automatically deletes children (Cascading Deletion)
        +
Owns() detects child changes -> triggers parent's Reconcile
```

Key takeaway: "Establishing the relationship" is manual; "performing the deletion" is automatic.
