# 03. CRD + CR + Controller = Operator

## What We've Learned So Far

```text
01. CRD  = Register a new resource kind (register snack request form)
02. CR   = An actual instance of that kind (request for 30 Choco Pies)
03. Controller = A program that watches and reconciles (the handler)
```

Combining these three makes an **Operator**.

## Goal

Build an Operator where creating a SimpleApp CR automatically creates a Deployment.

```text
What the user does:              What the Controller does:

"Run 2 nginx instances"          "Got it" -> Create Deployment
                                            -> 2 Pods running
kind: SimpleApp
spec:                   --->  kind: Deployment
  image: nginx:1.25           spec:
  replicas: 2                   replicas: 2
                                containers:
                                  - image: nginx:1.25
```

## What is kubebuilder?

A tool that **auto-generates** CRDs, Controllers, build configs, etc.

In Chapter 01, we wrote the CRD as raw YAML, but kubebuilder **generates YAML automatically from Go code**.

```text
Manual approach:
  Write crd.yaml + write controller.go + write build config manually

With kubebuilder:
  kubebuilder auto-generates skeleton code -> We only write the core logic
```

Analogy: Building a house

- Manual: Foundation, plumbing, electrical, walls, roof -- all by yourself
- kubebuilder: Basic structure is ready, you just do the interior design

## Project Creation Process

### Step 1: Initialize the Project

```bash
cd ~/Desktop/Operator/myoperator
kubebuilder init --domain example.com --repo github.com/teamsmiley/myoperator
```

| Option   | Value                            | Meaning                                        |
| -------- | -------------------------------- | ---------------------------------------------- |
| --domain | example.com                      | Domain part of CRD API Group                   |
| --repo   | github.com/teamsmiley/myoperator | Go module name (doesn't need an actual repo)   |

This command only generates build configs (Makefile, Dockerfile, go.mod, etc.). No CRD or Controller yet.

### Step 2: Create CRD + Controller

```bash
kubebuilder create api --group apps --version v1 --kind SimpleApp --resource --controller
```

| Option       | Value     | Meaning                                                       |
| ------------ | --------- | ------------------------------------------------------------- |
| --group      | apps      | API Group name (combined with domain: apps.example.com)       |
| --version    | v1        | Resource version                                              |
| --kind       | SimpleApp | Resource name (used as kind: SimpleApp in YAML)               |
| --resource   |           | Generate CRD type file                                        |
| --controller |           | Generate Controller file                                      |

This command generates **two key files**.

## The 2 Key Files

kubebuilder generates many files, but we only modify **2**.

### File 1: api/v1/simpleapp_types.go -- CRD Definition

This is the same as what we manually wrote as `crd.yaml` in Chapter 01, but **written in Go code**.

```text
Ch 01 (raw YAML):                   kubebuilder (Go code):

schema:                              type SimpleAppSpec struct {
  properties:                            Image    string
    menu:                                Replicas *int32
      type: string           <-->    }
    quantity:
      type: integer
```

kubebuilder reads the Go code and auto-generates CRD YAML (`make manifests`).

Actual code:

```go
// SimpleAppSpec -- The part where users declare the "desired state"
type SimpleAppSpec struct {
    // Container image (e.g., nginx:1.25)
    Image string `json:"image"`

    // Number of Pods to run (default 1, max 10)
    Replicas *int32 `json:"replicas,omitempty"`
}
```

Just like we defined `menu` and `quantity` fields in the Snack CRD in Chapter 01,
here we define `Image` and `Replicas` fields.

**Note: CRD field names are arbitrary.** Using `pizza`, `count` would also work.
However, since the Controller maps these values into a k8s Deployment,
convention is to use **the same names** as k8s built-in resources.

```text
CRD fields (free)        Controller converts          k8s Deployment (fixed)
-----------------        ----------------             --------------------------
spec.image       -->    Controller reads & maps -->   containers[0].image
spec.replicas    -->    Controller reads & maps -->   spec.replicas
```

CRD field names = free (conventionally the same)
k8s Deployment fields = fixed (cannot change)

### File 2: internal/controller/simpleapp_controller.go -- Controller

This is where the Reconcile function from Chapter 02 lives.

kubebuilder-generated initial state (before):

```go
func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    _ = logf.FromContext(ctx)

    // TODO(user): your logic here

    return ctrl.Result{}, nil
}
```

After our modifications (after):

```go
func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := logf.FromContext(ctx)

    // 1. Fetch the SimpleApp CR (check desired state)
    var app myappsv1.SimpleApp
    if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
        if errors.IsNotFound(err) {
            return ctrl.Result{}, nil  // Deleted, ignore
        }
        return ctrl.Result{}, err
    }

    // 2. Check if a Deployment exists
    var deploy appsv1.Deployment
    err := r.Get(ctx, req.NamespacedName, &deploy)

    if errors.IsNotFound(err) {
        // 2a. Doesn't exist -> create it
        log.Info("Creating Deployment", "name", app.Name)
        deploy := r.buildDeployment(&app)
        ctrl.SetControllerReference(&app, deploy, r.Scheme)
        return ctrl.Result{}, r.Create(ctx, deploy)
    }

    // 2b. Exists -> compare and update if changed
    replicas := int32(1)
    if app.Spec.Replicas != nil {
        replicas = *app.Spec.Replicas
    }

    needsUpdate := false
    if *deploy.Spec.Replicas != replicas {
        deploy.Spec.Replicas = &replicas
        needsUpdate = true
    }
    if deploy.Spec.Template.Spec.Containers[0].Image != app.Spec.Image {
        deploy.Spec.Template.Spec.Containers[0].Image = app.Spec.Image
        needsUpdate = true
    }
    if needsUpdate {
        return ctrl.Result{}, r.Update(ctx, &deploy)
    }
    return ctrl.Result{}, nil
}
```

Before vs After comparison:

|                | before (kubebuilder initial)  | after (our modifications)                  |
| -------------- | ----------------------------- | ------------------------------------------ |
| Reconcile body | Empty (TODO)                  | CR query -> Deployment create/update       |
| Lines of code  | 2 lines                       | ~30 lines                                  |
| Behavior       | Does nothing                  | Manages Deployment based on SimpleApp CR   |

## Complete Flow at a Glance

```text
1. kubebuilder create api     --> Generate skeleton code (types.go + controller.go)
2. Modify types.go            --> Define CRD fields (Image, Replicas)
3. Modify controller.go       --> Write Reconcile logic
4. make install               --> Register CRD in cluster (same as crd.yaml apply in Ch 01)
5. make run                   --> Start Controller (Watch standby)
6. kubectl apply CR           --> Create CR (same as cr.yaml apply in Ch 01)
7. Controller detects         --> Reconcile runs -> Deployment created
8. Deployment Controller      --> Creates Pods (handled by k8s built-in Controller)
```

## Summary

| What we did manually in Ch 01 | What kubebuilder does for us                          |
| ----------------------------- | ----------------------------------------------------- |
| Write crd.yaml manually       | Auto-generate from types.go (`make manifests`)        |
| kubectl apply crd.yaml        | `make install`                                        |
| No Controller                 | Auto-generate controller.go skeleton, just implement Reconcile |

**The code we actually wrote: field definitions in types.go + Reconcile logic in controller.go. That's everything.**

Next: [04-Build and Deploy](04-build-and-deploy.md) -- make commands, deployment methods
