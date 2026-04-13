# 05. RBAC -- Rules for Who Can Do What

## What Is RBAC?

It stands for **Role-Based Access Control**.
RBAC is Kubernetes' permission system that defines "who (Subject) can do what (Verb) to which resources (Resource)."

## Analogy: Office Building Access Control

```text
Employee badge (ServiceAccount)       -> "I am Kim, a developer"
Access permission sheet (Role)        -> "Server room on 3rd floor: enter/view OK, modify NO"
Granting access (RoleBinding)         -> "Give Kim this permission sheet"
```

A badge alone grants no access.
A permission sheet alone applies to no one.
**You must link the two (Binding)** for permissions to take effect.

## The 4 Core Resources

### What Are API Groups?

Kubernetes resources are categorized into **API Groups**.
When specifying `apiGroups` in rules, you use these group names.

| API Group | Example Resources | apiGroups Value |
| --------- | ----------------- | --------------- |
| core (fundamental) | Pod, Service, ConfigMap, Secret, Node | `""` (empty string) |
| apps | Deployment, StatefulSet, DaemonSet | `"apps"` |
| batch | Job, CronJob | `"batch"` |
| rbac.authorization.k8s.io | Role, ClusterRole | `"rbac.authorization.k8s.io"` |
| custom (our Operator) | SimpleApp | `"apps.example.com"` |

Key point: **The core API group has no name, so it is represented as an empty string `""`.**
The most fundamental resources like Pod, Service, and ConfigMap belong to this group.

```yaml
# Accessing core resources (Pod, ConfigMap, etc.)
apiGroups: [""]           # <- empty string = core API group

# Accessing apps group (Deployment, etc.)
apiGroups: ["apps"]

# Accessing custom resources
apiGroups: ["apps.example.com"]
```

### 1. Role (Namespace-Scoped)

A set of permissions valid only within a specific namespace.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: default
  name: pod-reader
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
```

### 2. ClusterRole (Cluster-Wide Scope)

Valid across all namespaces, or used for resources that have no namespace (such as Nodes).

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: deployment-manager
rules:
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "create", "update", "delete"]
```

### 3. RoleBinding (Links a Role to a Subject)

The connector that says "grant this person these permissions."

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: default
subjects:
  - kind: ServiceAccount
    name: my-operator
    namespace: default
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

### 4. ClusterRoleBinding (Links a ClusterRole to a Subject)

Grants a ClusterRole across the entire cluster.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: deployment-manager-binding
subjects:
  - kind: ServiceAccount
    name: my-operator
    namespace: myoperator-system
roleRef:
  kind: ClusterRole
  name: deployment-manager
  apiGroup: rbac.authorization.k8s.io
```

## Pairing: Permission Definition and Permission Granting Always Come in Pairs

| Permission Definition (What is allowed?) | Permission Granting (Who gets it?) | Scope |
| ---------------------------------------- | ---------------------------------- | ----- |
| Role | RoleBinding | Within a single specific namespace only |
| ClusterRole | ClusterRoleBinding | Entire cluster (all namespaces) |

Summary:

```text
Role  ---------> RoleBinding           (namespace-scoped)
  "what to allow"     "who to link"

ClusterRole  --> ClusterRoleBinding    (cluster-wide)
  "what to allow"     "who to link"
```

Note: You can also link a ClusterRole with a RoleBinding.
In that case, cluster-scoped permissions are restricted to a specific namespace.

| Combination | Meaning |
| ----------- | ------- |
| Role + RoleBinding | The most basic pairing. Valid in one namespace only |
| ClusterRole + ClusterRoleBinding | Valid across the entire cluster |
| ClusterRole + RoleBinding | Grants ClusterRole permissions restricted to a specific namespace |
| Role + ClusterRoleBinding | Not possible (Role is a namespace resource and cannot be bound cluster-wide) |

## Subject: The 3 Types of Permission Recipients

| Subject Type | Description | Example |
| ------------ | ----------- | ------- |
| ServiceAccount | An account assigned to a Pod. Operators use this | system:serviceaccount:default:my-operator |
| User | A human user (kubeconfig certificate) | A developer using kubectl |
| Group | A group of users | system:masters (administrator group) |

### What Is a ServiceAccount?

When a person accesses the cluster via kubectl, they use the kubeconfig certificate (User).
But **programs running inside a Pod** (including Operators) are not people.
A ServiceAccount is like giving an ID badge to such programs.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-operator
  namespace: myoperator-system
```

By itself, this grants no permissions. It is just a blank badge.
It only becomes meaningful when linked to permissions via a RoleBinding or ClusterRoleBinding.

To use a ServiceAccount in a Pod, specify it in the spec:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-operator-pod
spec:
  serviceAccountName: my-operator
  containers:
    - name: operator
      image: my-operator:v0.1.0
```

The complete flow:

```text
Create ServiceAccount          "Issue a blank badge"
       |
Create ClusterRole             "Write the permission sheet"
       |
Create ClusterRoleBinding      "Link the permission sheet to the badge"
       |
Set serviceAccountName in Pod  "Show up to work with the badge"
       |
Program inside Pod makes API call -> kubelet auto-injects ServiceAccount token
                                  -> API Server verifies token -> checks permissions -> allow/deny
```

## Common Verbs

| Verb | Meaning | HTTP Equivalent |
| ---- | ------- | --------------- |
| get | Retrieve a single resource | GET /pods/nginx |
| list | Retrieve a list of resources | GET /pods |
| watch | Watch for changes | GET /pods?watch=true |
| create | Create a resource | POST /pods |
| update | Full update | PUT /pods/nginx |
| patch | Partial update | PATCH /pods/nginx |
| delete | Delete a resource | DELETE /pods/nginx |

## How RBAC Relates to Operators

Key point: RBAC behaves differently between `make run` (local execution) and `make deploy` (cluster deployment).

```text
Local execution (make run)
  -> Uses admin permissions from kubeconfig
  -> Everything works
  -> You may not notice RBAC issues

Cluster deployment (make deploy)
  -> Uses only ServiceAccount permissions
  -> Only what is defined in Role/ClusterRole is allowed
  -> Missing permissions result in "forbidden" errors!
```

This is why RBAC must be configured correctly when actually deploying an Operator.

## Next Step

We will examine how RBAC is applied to our SimpleApp Operator by looking at the actual code.
