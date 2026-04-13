# 01. CRD and CR -- Hands-on Without kubebuilder

## Goal

Understand the relationship between CRD and CR using just two YAML files, without kubebuilder.

## What is a CRD?

Kubernetes comes with built-in resource types:

- Pod, Deployment, Service, ConfigMap, etc.

A CRD (Custom Resource Definition) **registers a new kind of resource** in Kubernetes.

Analogy: A school originally has "classrooms", "cafeteria", and "library."
A CRD is like **registering a new category called "Snack Request" in the school system.**

## What is a CR?

After registering a kind with a CRD, creating an **actual instance** of that kind is a CR (Custom Resource).

Analogy: After registering the "Snack Request" category, you **submit an actual request for "30 Choco Pies."**

```text
CRD = "A kind called Snack Request exists" (kind registration)
CR  = "I'm requesting 30 Choco Pies" (actual request)
```

## Lab Files

There are 2 files in the `examples/` folder, written manually without kubebuilder.

### crd.yaml -- Kind Registration (Blueprint)

```yaml
# Register "a resource kind called Snack" with Kubernetes
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: snacks.school.example.com
spec:
  group: school.example.com  # API group
  names:
    kind: Snack       # Name used in YAML
    plural: snacks    # kubectl get snacks
    singular: snack
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                menu:       # Snack menu
                  type: string
                quantity:   # Amount
                  type: integer
```

### cr.yaml -- Actual Request

```yaml
# "I'd like 30 Choco Pies please"
apiVersion: school.example.com/v1
kind: Snack
metadata:
  name: afternoon-snack
  namespace: default
spec:
  menu: chocopie
  quantity: 30
```

## Lab Steps

### Step 1: Apply CR Without CRD (Failure)

```bash
kubectl apply -f examples/cr.yaml
```

Result:

```
error: resource mapping not found for name: "afternoon-snack"
no matches for kind "Snack" in version "school.example.com/v1"
```

Kubernetes says "Snack? I don't know that resource."
This makes sense since we haven't registered the kind (CRD) yet.

### Step 2: Register the CRD

```bash
kubectl apply -f examples/crd.yaml
```

Result:

```
customresourcedefinition.apiextensions.k8s.io/snacks.school.example.com created
```

Now Kubernetes knows "a kind called Snack exists."

### Step 3: Apply CR Again (Success)

```bash
kubectl apply -f examples/cr.yaml
```

Result:

```
snack.school.example.com/afternoon-snack created
```

### Step 4: Verify

```bash
# Check CR list
kubectl get snack

# Check CR details
kubectl get snack afternoon-snack -o yaml
```

The CR is created, but **nothing happens.**
The snack request was accepted, but since there's no handler (Controller), no snacks are delivered.

## Summary

```text
1. Create CR without CRD --> Fails ("Don't know what Snack is")
2. Register CRD          --> "A kind called Snack exists" is registered
3. Create CR             --> "30 Choco Pies request" accepted
4. However               --> Nothing happens (no Controller)
```

| Component  | File     | Analogy                       | What it does                          |
| ---------- | -------- | ----------------------------- | ------------------------------------- |
| CRD        | crd.yaml | Register "Snack Request" kind | Makes Kubernetes recognize Snack      |
| CR         | cr.yaml  | "30 Choco Pies" request       | Stored as data in etcd                |
| Controller | (none)   | No handler                    | Nobody processes the request          |

## Cleanup

```bash
kubectl delete snack afternoon-snack
kubectl delete crd snacks.school.example.com
```

## Next Step

See what changes when we add a Controller.
