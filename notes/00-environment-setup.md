# 00. 개발 환경 구성

## 필요한 도구

| 도구        | 용도                                           | 설치 명령                    |
| ----------- | ---------------------------------------------- | ---------------------------- |
| Go          | Operator 개발 언어                             | `brew install go`            |
| Docker      | k3d가 컨테이너로 클러스터를 만들 때 사용       | `brew install --cask docker` |
| kubectl     | Kubernetes 클러스터와 통신하는 CLI             | `brew install kubectl`       |
| k3d         | 로컬에서 k3s 클러스터를 Docker 컨테이너로 실행 | `brew install k3d`           |
| kubebuilder | Operator 프로젝트 스캐폴딩 도구                | `brew install kubebuilder`   |

## 설치 순서

Docker가 먼저 실행되고 있어야 k3d가 동작하므로, 순서가 중요하다.

```bash
# 1. Go 설치
brew install go

# 2. Docker 설치 및 실행
brew install --cask docker
# Docker Desktop 앱을 실행하여 Docker daemon을 띄워야 한다

# 3. kubectl 설치
brew install kubectl

# 4. k3d 설치
brew install k3d

# 5. kubebuilder 설치
brew install kubebuilder
```

## 설치 확인

```bash
go version
docker --version
kubectl version --client
k3d version
kubebuilder version
```

### 현재 설치된 버전 (2026-04-11 확인)

| 도구        | 버전                        |
| ----------- | --------------------------- |
| Go          | 1.26.2                      |
| Docker      | 29.3.1                      |
| kubectl     | v1.33.2 (Kustomize v5.6.0)  |
| k3d         | v5.8.3 (k3s v1.33.6-k3s1)   |
| kubebuilder | v4.13.1 (Kubernetes 1.35.0) |

## k3d로 클러스터 생성

```bash
# 클러스터 생성 (이름: operator-lab)
k3d cluster create operator-lab

# kubectl context가 자동으로 설정된다. 확인:
kubectl cluster-info
kubectl get nodes
```

## k3d 기본 명령어

```bash
# 클러스터 목록
k3d cluster list

# 클러스터 중지 (Docker 컨테이너 중지, 상태 유지)
k3d cluster stop operator-lab

# 클러스터 재시작
k3d cluster start operator-lab

# 클러스터 삭제
k3d cluster delete operator-lab
```

## k3d vs Kind vs Minikube

| 항목               | k3d             | Kind            | Minikube       |
| ------------------ | --------------- | --------------- | -------------- |
| 기반               | k3s (경량 k8s)  | 표준 k8s        | 표준 k8s       |
| 실행 방식          | Docker 컨테이너 | Docker 컨테이너 | VM 또는 Docker |
| 클러스터 생성 속도 | 빠름 (수 초)    | 보통            | 느림           |
| 리소스 사용량      | 적음            | 보통            | 많음           |
| 멀티 노드          | 쉬움            | 가능            | 제한적         |

k3d를 선택한 이유: 가볍고 빠르며, Operator 개발/테스트 용도로 충분하다.
