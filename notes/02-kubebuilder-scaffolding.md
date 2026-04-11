# 02. kubebuilder 프로젝트 스캐폴딩

## 프로젝트 초기화

```bash
mkdir -p ~/Desktop/Operator/myoperator
cd ~/Desktop/Operator/myoperator

# 프로젝트 초기화
# --domain: CRD API Group의 도메인 (실제 소유 도메인이 아니어도 됨)
# --repo: Go module 경로 (실제 GitHub repo가 없어도 됨)
kubebuilder init --domain example.com --repo github.com/teamsmiley/myoperator

# CRD + Controller 생성
# --group: API Group 이름 (domain과 합쳐져서 apps.example.com이 됨)
# --version: API 버전
# --kind: 리소스 이름 (PascalCase)
# --resource: CRD 타입 파일 생성
# --controller: Controller 파일 생성
kubebuilder create api --group apps --version v1 --kind SimpleApp --resource --controller
```

## API Group 구성 원리

```
--domain example.com  +  --group apps  +  --version v1
                            |
                  API Group: apps.example.com
                            |
           apiVersion: apps.example.com/v1
```

## 생성된 프로젝트 구조 (핵심 파일만)

```
myoperator/
  api/v1/
    simpleapp_types.go      # CRD 필드 정의 (spec, status)
    groupversion_info.go    # API Group/Version 등록
  internal/controller/
    simpleapp_controller.go # Reconcile 로직
  cmd/
    main.go                 # 엔트리포인트 (Manager 생성 및 실행)
  config/
    crd/                    # 생성된 CRD YAML (make manifests로 갱신)
    rbac/                   # RBAC 관련 설정
    manager/                # Controller Manager 배포 설정
  Makefile                  # 빌드/배포 명령 모음
  go.mod                    # Go 의존성
```

## 핵심 파일 3개의 역할

### 1. api/v1/simpleapp_types.go -- "리소스의 모양"

사용자가 YAML로 작성할 수 있는 필드를 Go struct로 정의한다.

```go
// 사용자가 "원하는 상태"를 쓰는 부분 (Desired State)
type SimpleAppSpec struct { ... }

// Controller가 "현재 상태"를 기록하는 부분 (Current State)
type SimpleAppStatus struct { ... }

// 전체 리소스 구조
type SimpleApp struct {
    metav1.TypeMeta   // apiVersion, kind
    metav1.ObjectMeta // metadata (name, namespace, labels 등)
    Spec   SimpleAppSpec
    Status SimpleAppStatus
}
```

### 2. internal/controller/simpleapp_controller.go -- "동작 로직"

SimpleApp CR이 생성/수정/삭제될 때마다 호출되는 Reconcile 함수가 여기에 있다.

```go
func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 여기에 로직을 넣는다
}
```

### 3. cmd/main.go -- "실행 진입점"

Manager를 생성하고, Controller를 등록하고, 실행한다.
일반적으로 직접 수정할 일은 거의 없다.

## 주요 Makefile 명령어

### 개발 시 사용하는 명령어

| 명령             | 하는 일                                        | 의존성 (자동 실행)            |
| ---------------- | ---------------------------------------------- | ----------------------------- |
| `make generate`  | types.go의 struct에서 DeepCopy 코드 자동 생성  | controller-gen                |
| `make manifests` | types.go의 marker에서 CRD YAML, RBAC YAML 생성 | controller-gen                |
| `make build`     | Go 바이너리 빌드 (`bin/manager`)               | manifests, generate, fmt, vet |
| `make run`       | Controller를 로컬 PC에서 실행 (개발용)         | manifests, generate, fmt, vet |
| `make install`   | CRD를 클러스터에 설치 (`kubectl apply`)        | manifests                     |
| `make uninstall` | CRD를 클러스터에서 제거                        | manifests                     |

### 배포 시 사용하는 명령어

| 명령                                | 하는 일                                                      |
| ----------------------------------- | ------------------------------------------------------------ |
| `make docker-build IMG=이미지:태그` | Controller Docker 이미지 빌드                                |
| `make docker-push IMG=이미지:태그`  | 이미지를 레지스트리에 push                                   |
| `make deploy IMG=이미지:태그`       | Controller를 클러스터에 Pod로 배포 (CRD + RBAC + Deployment) |
| `make undeploy`                     | 배포된 Controller를 클러스터에서 제거                        |

### 의존성 참고

`make build`와 `make run`은 내부적으로 `manifests`, `generate`, `fmt`, `vet`를 먼저 실행한다.
따라서 types.go를 수정한 후 `make run`만 하면 코드 생성과 CRD 갱신이 자동으로 된다.

## kubebuilder marker (주석 태그)

코드에 있는 `// +kubebuilder:...` 주석은 단순 주석이 아니라, 코드 생성기가 읽는 지시어이다.

```go
// +kubebuilder:validation:Required       -- 이 필드는 필수
// +kubebuilder:validation:Minimum=1      -- 최솟값 1
// +kubebuilder:validation:Maximum=10     -- 최댓값 10
// +kubebuilder:default=1                 -- 기본값 1
// +kubebuilder:object:root=true          -- 이 타입이 API 루트 오브젝트
// +kubebuilder:subresource:status        -- status subresource 활성화
// +kubebuilder:rbac:groups=...           -- RBAC 권한 선언
```

`make manifests`를 실행하면 이 marker들을 읽어서 CRD YAML과 RBAC YAML을 자동 생성한다.
