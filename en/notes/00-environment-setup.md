# 00. Development Environment Setup

## Required Tools

| Tool        | Purpose                                          | Install Command                |
| ----------- | ------------------------------------------------ | ------------------------------ |
| Go          | Operator development language                    | `brew install go`              |
| Docker      | k3d uses containers to create clusters           | `brew install --cask docker`   |
| kubectl     | CLI for communicating with Kubernetes clusters   | `brew install kubectl`         |
| k3d         | Runs k3s clusters as Docker containers locally   | `brew install k3d`             |
| kubebuilder | Operator project scaffolding tool                | `brew install kubebuilder`     |

## Installation Order

Docker must be running before k3d can work, so the order matters.

```bash
# 1. Install Go
brew install go

# 2. Install and start Docker
brew install --cask docker
# Launch the Docker Desktop app to start the Docker daemon

# 3. Install kubectl
brew install kubectl

# 4. Install k3d
brew install k3d

# 5. Install kubebuilder
brew install kubebuilder
```

## Verify Installation

```bash
go version
docker --version
kubectl version --client
k3d version
kubebuilder version
```

### Installed Versions (verified 2026-04-11)

| Tool        | Version                     |
| ----------- | --------------------------- |
| Go          | 1.26.2                      |
| Docker      | 29.3.1                      |
| kubectl     | v1.33.2 (Kustomize v5.6.0)  |
| k3d         | v5.8.3 (k3s v1.33.6-k3s1)   |
| kubebuilder | v4.13.1 (Kubernetes 1.35.0) |

## Creating a Cluster with k3d

```bash
# Create a cluster (name: operator-lab)
k3d cluster create operator-lab

# If KUBECONFIG has multiple files, the context may not be auto-registered.
# In that case, copy the kubeconfig to ~/.kube/ and add it to KUBECONFIG:
#   k3d kubeconfig merge operator-lab -o ~/.kube/k3d-operator-lab.yaml
#   export KUBECONFIG="$KUBECONFIG:$HOME/.kube/k3d-operator-lab.yaml"

# Switch context and verify:
kubectl config use-context k3d-operator-lab
kubectl get nodes
```

## Basic k3d Commands

```bash
# List clusters
k3d cluster list

# Stop cluster (stops Docker containers, preserves state)
k3d cluster stop operator-lab

# Restart cluster
k3d cluster start operator-lab

# Delete cluster
k3d cluster delete operator-lab
```

## k3d vs Kind vs Minikube

| Category             | k3d              | Kind              | Minikube         |
| -------------------- | ----------------- | ----------------- | ---------------- |
| Based on             | k3s (lightweight) | Standard k8s      | Standard k8s     |
| Execution method     | Docker containers | Docker containers | VM or Docker     |
| Cluster creation     | Fast (seconds)    | Moderate          | Slow             |
| Resource usage       | Low               | Moderate          | High             |
| Multi-node           | Easy              | Possible          | Limited          |

Why k3d: Lightweight, fast, and sufficient for Operator development/testing.
