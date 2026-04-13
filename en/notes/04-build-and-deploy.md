# 04. Build and Deploy

## make Command Workflow

```text
Modify types.go
    |
    v
make generate    --> Auto-generate DeepCopy Go code
    |
    v
make manifests   --> Auto-generate CRD YAML + RBAC YAML
    |
    v
make install     --> Register CRD in the cluster
    |
    v
make run         --> Run the Controller locally
```

## Detailed Explanation of Each Command

### make generate -- Auto-generate DeepCopy Code

```text
Input:  api/v1/simpleapp_types.go (the struct we wrote)
Output: api/v1/zz_generated.deepcopy.go (auto-generated code)
```

This auto-generates the DeepCopy functions that Kubernetes needs internally when copying objects.
Every Operator needs these, but writing them by hand every time would be tedious repetitive work, so it is automated.
The `zz_` prefix is a convention meaning "auto-generated file, do not edit manually."

### make manifests -- Generate CRD YAML + RBAC YAML

```text
Input 1:  api/v1/simpleapp_types.go (struct + kubebuilder markers)
Output 1: config/crd/bases/apps.example.com_simpleapps.yaml (CRD YAML)
          --> The same thing as the crd.yaml we wrote by hand in Chapter 01,
              but auto-generated from Go code

Input 2:  RBAC marker comments in controller.go
Output 2: config/rbac/role.yaml (permission configuration)
```

In Chapter 01, we wrote the Snack crd.yaml directly as YAML.
kubebuilder auto-generates YAML from Go code instead.

### make install -- Register the CRD in the Cluster

```text
What it does internally:
  make manifests (runs automatically) --> kubectl apply -f config/crd/bases/
```

This is the same as running `kubectl apply -f crd.yaml` in Chapter 01.
Without this step, creating a CR would produce an error like "SimpleApp is not recognized."

### make run -- Run the Controller Locally (for development)

```text
What it does internally:
  make manifests + make generate (run automatically) --> go run ./cmd/main.go
```

This starts the Watch + Reconcile loop we learned about in Chapter 02.
It occupies the terminal, watching for CR changes and executing Reconcile when they occur.

**Note: make run and make install internally run generate/manifests automatically.
Therefore, after modifying types.go, simply running make run handles everything automatically.**

### Automatic Dependency Summary

| Command | What It Automatically Includes |
|---------|-------------------------------|
| make install | manifests |
| make run | manifests + generate + fmt + vet |
| make build | manifests + generate + fmt + vet |

## How make run Works

`make run` **does not install anything inside the cluster.**
It runs as a process on your PC, communicating with the API Server through kubeconfig.

```text
My PC (make run)                          k3d Cluster
----------------                          -----------
Controller process                        API Server
  |                                           |
  |-- Watch request (HTTP long-poll) -------->|  "Notify me of SimpleApp changes"
  |                                           |
  |<-- Event received ------------------------|  "my-app was created!"
  |                                           |
  |  Execute Reconcile                        |
  |  (Assemble Deployment object)             |
  |                                           |
  |-- Create request (HTTP POST) ------------>|  "Create this Deployment"
  |                                           |
  |                                        API Server saves Deployment
  |                                        kubelet creates Pod
```

Even if you check with `kubectl get pods`, the Controller Pod will not appear.
It is simply working remotely from your PC.

### What Is the :8081 Port?

If you run `make run` twice, you get a `:8081 port conflict` error.
This port is not for communicating with the k8s API -- it is for **health probes (checking if the process is alive)**.

```text
:8081 port        = "Am I alive?" health check
kubeconfig address = Actual communication with the k8s API Server (Watch, Create, Update)
```

## Production Deployment (make deploy)

`make run` is for development. **In production, you need to build the Controller as a Docker image and deploy it as a Pod inside the cluster.**

### Deployment Steps

```bash
# 1. Build Docker image
make docker-build IMG=myoperator:v0.1.0

# 2. Push image to a registry (or import locally with k3d)
make docker-push IMG=myoperator:v0.1.0
# k3d local: k3d image import myoperator:v0.1.0 -c operator-lab

# 3. Deploy to the cluster
make deploy IMG=myoperator:v0.1.0
```

### What make deploy Does (all at once)

```text
make deploy IMG=myoperator:v0.1.0
    |
    v
1. Install CRD (what make install used to do)
2. Deploy Controller Pod (what make run used to do, but as a Pod)
3. Set up RBAC (ServiceAccount, ClusterRole, etc.)
4. Everything is created in the myoperator-system namespace
```

What took two steps in development mode is reduced to one step in production mode:

```text
Dev:  make install (CRD) + make run (Controller runs locally)   --> 2 steps
Prod: make deploy (CRD + Controller Pod + RBAC all at once)     --> 1 step
```

### make run vs make deploy Comparison

|           | make run (Development)                | make deploy (Production)              |
| --------- | ------------------------------------- | ------------------------------------- |
| Runs on   | My PC (working remotely)              | Pod inside the cluster (on-site)      |
| Terminal  | Occupied                              | Not occupied                          |
| Restart   | Manual (Ctrl+C, run again)            | Automatic (Pod crash -> auto-recovery)|
| Pod visible? | Not visible in kubectl get pods    | Visible in kubectl get pods           |
| Analogy   | A teacher working from home           | A teacher who showed up at school     |

## Manifest File Locations

The manifests that `make deploy` applies are in the `config/` folder:

```text
config/
  crd/bases/
    apps.example.com_simpleapps.yaml    # CRD (resource type registration)
  rbac/
    role.yaml                           # RBAC (Controller permissions)
    role_binding.yaml
    service_account.yaml
  manager/
    manager.yaml                        # Controller Deployment (Pod deployment)
```

`make manifests` auto-generates these YAMLs from Go code,
and `make deploy` merges them with kustomize and applies them all at once with `kubectl apply`.

## Deploying to an Inaccessible Cluster

`make deploy` only works for clusters accessible via kubeconfig on your PC.
For inaccessible clusters, you need to hand off the manifest files:

```bash
# Bundle all deployment YAMLs into one file (includes CRD + Controller + RBAC)
make build-installer IMG=ghcr.io/teamsmiley/myoperator:v0.1.0

# Result: dist/install.yaml is generated
```

Hand this file to someone who has access to the target cluster:

```bash
kubectl apply -f install.yaml   # This single command installs the entire Operator
```

Ultimately, deployment only requires two things: **an image (registry) + manifests (YAML)**.

## Verifying After make deploy

```bash
# Check the Controller Pod
kubectl get pods -n myoperator-system

# View Controller logs (instead of the make run terminal)
kubectl logs -n myoperator-system -l control-plane=controller-manager -f

# Cleaning up
make undeploy
```

## Testing

```bash
# Create a CR
kubectl apply -f config/samples/apps_v1_simpleapp.yaml

# Verify
kubectl get simpleapp
kubectl get deploy
kubectl get pods
```
